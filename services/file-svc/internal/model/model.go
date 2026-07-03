package model

import "time"

// File status
const (
	FileStatusActive  = 1
	FileStatusDeleted = 0
)

// File type categories (from PRD §4.9.5)
const (
	FileTypeImage = "image"
	FileTypeVideo = "video"
	FileTypeAudio = "audio"
	FileTypeFile  = "file"
)

// File size limits (bytes, from PRD §4.9.5)
const (
	MaxImageSize = 10 * 1024 * 1024  // 10MB
	MaxVideoSize = 100 * 1024 * 1024 // 100MB
	MaxFileSize  = 50 * 1024 * 1024  // 50MB
)

// FileResource 文件资源
type FileResource struct {
	FileID    int64     `gorm:"primaryKey;autoIncrement" json:"file_id"`
	TenantID  int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	UserID    int64     `gorm:"column:user_id;not null;index" json:"user_id"`
	FileName  string    `gorm:"column:file_name;type:varchar(500);not null" json:"file_name"`
	FileSize  int64     `gorm:"column:file_size;not null" json:"file_size"`
	MimeType  string    `gorm:"column:mime_type;type:varchar(100)" json:"mime_type"`
	FileType  string    `gorm:"column:file_type;type:varchar(20)" json:"file_type"`
	ObjectKey string    `gorm:"column:object_key;type:varchar(500);not null;uniqueIndex" json:"object_key"`
	Bucket    string    `gorm:"column:bucket;type:varchar(100);not null" json:"bucket"`
	Width     int       `gorm:"column:width;default:0" json:"width,omitempty"`
	Height    int       `gorm:"column:height;default:0" json:"height,omitempty"`
	Duration  int       `gorm:"column:duration;default:0" json:"duration,omitempty"`
	Md5       string    `gorm:"column:md5;type:varchar(64)" json:"md5"`
	Status    int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (FileResource) TableName() string { return "file_resource" }

// --- Request / Response types ---

type FileUploadResp struct {
	FileID   int64  `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
	FileType string `json:"file_type"`
	URL      string `json:"url"`
}

type FileDetailResp struct {
	FileID    int64  `json:"file_id"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
	FileType  string `json:"file_type"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Md5       string `json:"md5"`
	CreatedAt string `json:"created_at"`
}

type InitMultipartUploadReq struct {
	FileName string `json:"file_name" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	MimeType string `json:"mime_type"`
}

type InitMultipartUploadResp struct {
	UploadID string `json:"upload_id"`
	ObjectKey string `json:"object_key"`
}

type UploadPartResp struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type CompleteMultipartReq struct {
	UploadID   string         `json:"upload_id" binding:"required"`
	ObjectKey  string         `json:"object_key" binding:"required"`
	Parts      []UploadedPart `json:"parts" binding:"required"`
	FileName   string         `json:"file_name"`
	FileSize   int64          `json:"file_size"`
	MimeType   string         `json:"mime_type"`
}

type UploadedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type FileListReq struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	FileType string `form:"file_type"`
}

type FileListResp struct {
	List     []FileDetailResp `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
