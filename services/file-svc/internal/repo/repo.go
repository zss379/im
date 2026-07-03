package repo

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/file-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(&model.FileResource{})
}

func (r *MySQLRepo) CreateFile(ctx context.Context, f *model.FileResource) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *MySQLRepo) GetFile(ctx context.Context, fileID int64) (*model.FileResource, error) {
	var f model.FileResource
	err := r.db.WithContext(ctx).First(&f, fileID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &f, err
}

func (r *MySQLRepo) SoftDelete(ctx context.Context, fileID int64) error {
	return r.db.WithContext(ctx).Model(&model.FileResource{}).Where("file_id = ?", fileID).Update("status", model.FileStatusDeleted).Error
}

func (r *MySQLRepo) ListFiles(ctx context.Context, tenantID int64, fileType string, page, pageSize int) ([]model.FileResource, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.FileResource{}).Where("tenant_id = ? AND status = ?", tenantID, model.FileStatusActive)
	if fileType != "" {
		query = query.Where("file_type = ?", fileType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.FileResource
	q := r.db.WithContext(ctx).Where("tenant_id = ? AND status = ?", tenantID, model.FileStatusActive)
	if fileType != "" {
		q = q.Where("file_type = ?", fileType)
	}
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error
	return files, total, err
}

// ---- S3/MinIO Storage ----

type Storage struct {
	mc     *minio.Client
	bucket string
}

func NewStorage(mc *minio.Client, bucket string) *Storage {
	return &Storage{mc: mc, bucket: bucket}
}

func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.mc.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.mc.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *Storage) PutFile(ctx context.Context, objectKey string, reader io.Reader, fileSize int64, contentType string) (*minio.UploadInfo, error) {
	info, err := s.mc.PutObject(ctx, s.bucket, objectKey, reader, fileSize,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, fmt.Errorf("s3 put object: %w", err)
	}
	return &info, nil
}

func (s *Storage) PresignedGetURL(ctx context.Context, objectKey string, expirySeconds int) (string, error) {
	url, err := s.mc.PresignedGetObject(ctx, s.bucket, objectKey, expirySeconds, nil)
	if err != nil {
		return "", fmt.Errorf("s3 presigned url: %w", err)
	}
	return url.String(), nil
}

func (s *Storage) DeleteObject(ctx context.Context, objectKey string) error {
	return s.mc.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *Storage) InitMultipartUpload(ctx context.Context, objectKey, contentType string) (string, error) {
	upload, err := s.mc.NewMultipartUpload(ctx, s.bucket, objectKey, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("s3 init multipart: %w", err)
	}
	return upload.UploadID, nil
}

func (s *Storage) UploadPart(ctx context.Context, objectKey, uploadID string, partNumber int, reader io.Reader, partSize int64) (*minio.PartInfo, error) {
	info, err := s.mc.PutObjectPart(ctx, s.bucket, objectKey, uploadID, partNumber, reader, partSize, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("s3 upload part: %w", err)
	}
	return &info, nil
}

func (s *Storage) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []minio.CompletePart) (*minio.UploadInfo, error) {
	info, err := s.mc.CompleteMultipartUpload(ctx, s.bucket, objectKey, uploadID, parts, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 complete multipart: %w", err)
	}
	return &info, nil
}

func (s *Storage) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return s.mc.AbortMultipartUpload(ctx, s.bucket, objectKey, uploadID)
}

func (s *Storage) Bucket() string {
	return s.bucket
}
