package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rs/xid"

	"github.com/shulian-paas/im/file-svc/internal/metrics"
	"github.com/shulian-paas/im/file-svc/internal/model"
	"github.com/shulian-paas/im/file-svc/internal/repo"
)

type FileService struct {
	repo    *repo.MySQLRepo
	storage *repo.Storage
}

func NewFileService(repo *repo.MySQLRepo, storage *repo.Storage) *FileService {
	return &FileService{repo: repo, storage: storage}
}

const presignedURLExpiry = 3600 // 1 hour

func classifyFileType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return model.FileTypeImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return model.FileTypeVideo
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return model.FileTypeAudio
	}
	return model.FileTypeFile
}

func checkFileSize(fileType string, size int64) error {
	switch fileType {
	case model.FileTypeImage:
		if size > model.MaxImageSize {
			return fmt.Errorf("image exceeds max size %dMB", model.MaxImageSize/(1024*1024))
		}
	case model.FileTypeVideo:
		if size > model.MaxVideoSize {
			return fmt.Errorf("video exceeds max size %dMB", model.MaxVideoSize/(1024*1024))
		}
	default:
		if size > model.MaxFileSize {
			return fmt.Errorf("file exceeds max size %dMB", model.MaxFileSize/(1024*1024))
		}
	}
	return nil
}

// ---- Upload ----

func (s *FileService) UploadFile(ctx context.Context, tenantID, userID int64, fileHeader *multipart.FileHeader) (*model.FileUploadResp, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	fileType := classifyFileType(mimeType)
	fileSize := fileHeader.Size

	if err := checkFileSize(fileType, fileSize); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := filepath.Ext(fileHeader.Filename)
	objectKey := fmt.Sprintf("%d/%s/%s%s", tenantID, time.Now().Format("2006/01/02"), xid.New().String(), ext)

	info, err := s.storage.PutFile(ctx, objectKey, bytes.NewReader(data), fileSize, mimeType)
	if err != nil {
		return nil, err
	}

	resource := &model.FileResource{
		TenantID:  tenantID,
		UserID:    userID,
		FileName:  fileHeader.Filename,
		FileSize:  fileSize,
		MimeType:  mimeType,
		FileType:  fileType,
		ObjectKey: objectKey,
		Bucket:    s.storage.Bucket(),
		Md5:       info.ETag,
	}
	if err := s.repo.CreateFile(ctx, resource); err != nil {
		return nil, err
	}

	presignedURL, _ := s.storage.PresignedGetURL(ctx, objectKey, presignedURLExpiry)

	metrics.FileUploadTotal.WithLabelValues(fileType).Inc()
	metrics.UploadBytesTotal.Add(float64(fileSize))

	return &model.FileUploadResp{
		FileID:   resource.FileID,
		FileName: resource.FileName,
		FileSize: resource.FileSize,
		MimeType: resource.MimeType,
		FileType: resource.FileType,
		URL:      presignedURL,
	}, nil
}

// ---- Multipart Upload ----

func (s *FileService) InitMultipartUpload(ctx context.Context, req *model.InitMultipartUploadReq) (*model.InitMultipartUploadResp, error) {
	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	fileType := classifyFileType(mimeType)

	if err := checkFileSize(fileType, req.FileSize); err != nil {
		return nil, err
	}

	ext := filepath.Ext(req.FileName)
	objectKey := fmt.Sprintf("uploads/%s/%s", time.Now().Format("2006/01/02"), xid.New().String()+ext)

	uploadID, err := s.storage.InitMultipartUpload(ctx, objectKey, mimeType)
	if err != nil {
		return nil, err
	}

	metrics.MultipartUploadTotal.Inc()
	return &model.InitMultipartUploadResp{
		UploadID:  uploadID,
		ObjectKey: objectKey,
	}, nil
}

func (s *FileService) UploadPart(ctx context.Context, objectKey, uploadID string, partNumber int, reader io.Reader, partSize int64) (*minio.PartInfo, error) {
	return s.storage.UploadPart(ctx, objectKey, uploadID, partNumber, reader, partSize)
}

func (s *FileService) CompleteMultipartUpload(ctx context.Context, tenantID, userID int64, req *model.CompleteMultipartReq) (*model.FileUploadResp, error) {
	parts := make([]minio.CompletePart, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = minio.CompletePart{PartNumber: p.PartNumber, ETag: p.ETag}
	}

	info, err := s.storage.CompleteMultipartUpload(ctx, req.ObjectKey, req.UploadID, parts)
	if err != nil {
		return nil, err
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	fileType := classifyFileType(mimeType)

	resource := &model.FileResource{
		TenantID:  tenantID,
		UserID:    userID,
		FileName:  req.FileName,
		FileSize:  req.FileSize,
		MimeType:  mimeType,
		FileType:  fileType,
		ObjectKey: req.ObjectKey,
		Bucket:    s.storage.Bucket(),
		Md5:       info.ETag,
	}
	if err := s.repo.CreateFile(ctx, resource); err != nil {
		return nil, err
	}

	presignedURL, _ := s.storage.PresignedGetURL(ctx, req.ObjectKey, presignedURLExpiry)

	metrics.FileUploadTotal.WithLabelValues(fileType).Inc()
	metrics.UploadBytesTotal.Add(float64(req.FileSize))

	return &model.FileUploadResp{
		FileID:   resource.FileID,
		FileName: resource.FileName,
		FileSize: resource.FileSize,
		MimeType: resource.MimeType,
		FileType: resource.FileType,
		URL:      presignedURL,
	}, nil
}

// ---- Download / Access ----

func (s *FileService) GetFile(ctx context.Context, fileID, tenantID int64) (*model.FileDetailResp, error) {
	f, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if f == nil || f.TenantID != tenantID || f.Status != model.FileStatusActive {
		return nil, nil
	}
	return &model.FileDetailResp{
		FileID:    f.FileID,
		FileName:  f.FileName,
		FileSize:  f.FileSize,
		MimeType:  f.MimeType,
		FileType:  f.FileType,
		Width:     f.Width,
		Height:    f.Height,
		Duration:  f.Duration,
		Md5:       f.Md5,
		CreatedAt: f.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *FileService) GetDownloadURL(ctx context.Context, fileID, tenantID int64) (string, error) {
	f, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return "", err
	}
	if f == nil || f.TenantID != tenantID || f.Status != model.FileStatusActive {
		return "", fmt.Errorf("file not found")
	}

	url, err := s.storage.PresignedGetURL(ctx, f.ObjectKey, presignedURLExpiry)
	if err != nil {
		return "", err
	}

	metrics.FileDownloadTotal.Inc()
	return url, nil
}

func (s *FileService) DeleteFile(ctx context.Context, fileID, tenantID int64) error {
	f, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return err
	}
	if f == nil || f.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	if err := s.repo.SoftDelete(ctx, fileID); err != nil {
		return err
	}
	_ = s.storage.DeleteObject(ctx, f.ObjectKey)
	return nil
}

func (s *FileService) ListFiles(ctx context.Context, tenantID int64, req *model.FileListReq) ([]model.FileDetailResp, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 50
	}

	files, total, err := s.repo.ListFiles(ctx, tenantID, req.FileType, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]model.FileDetailResp, len(files))
	for i, f := range files {
		resp[i] = model.FileDetailResp{
			FileID:    f.FileID,
			FileName:  f.FileName,
			FileSize:  f.FileSize,
			MimeType:  f.MimeType,
			FileType:  f.FileType,
			Width:     f.Width,
			Height:    f.Height,
			Duration:  f.Duration,
			Md5:       f.Md5,
			CreatedAt: f.CreatedAt.Format(time.RFC3339),
		}
	}
	return resp, total, nil
}

