package model

import "time"

// SensitiveWord 敏感词
type SensitiveWord struct {
	WordID      int64     `gorm:"primaryKey;autoIncrement" json:"word_id"`
	TenantID    int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	Word        string    `gorm:"column:word;type:varchar(200);not null;index:idx_word(20)" json:"word"`
	Strategy    int8      `gorm:"column:strategy;not null;default:1" json:"strategy"` // 1=replace, 2=block, 3=log
	Replacement string    `gorm:"column:replacement;type:varchar(10);default:'***'" json:"replacement"`
	Category    string    `gorm:"column:category;type:varchar(50)" json:"category"`
	Status      int8      `gorm:"column:status;not null;default:1" json:"status"` // 1=enabled, 0=disabled
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SensitiveWord) TableName() string { return "sensitive_word" }

// SensitiveWord Strategies
const (
	SensitiveStrategyReplace = 1 // replace with ***
	SensitiveStrategyBlock   = 2 // block the message
	SensitiveStrategyLog     = 3 // log only
)

// FrequencyControlRule 频控规则
type FrequencyControlRule struct {
	RuleID            int64     `gorm:"primaryKey;autoIncrement" json:"rule_id"`
	TenantID          int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	TargetType        int8      `gorm:"column:target_type;not null" json:"target_type"` // 1=user, 2=bot
	MaxCount          int       `gorm:"column:max_count;not null" json:"max_count"`
	TimeWindowSeconds int       `gorm:"column:time_window_seconds;not null" json:"time_window_seconds"`
	Action            int8      `gorm:"column:action;not null;default:1" json:"action"` // 1=rate_limit 429
	Status            int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (FrequencyControlRule) TableName() string { return "frequency_control_rule" }

// TargetType constants
const (
	TargetTypeUser = 1
	TargetTypeBot  = 2
)

// FileLimitConfig 文件上传限制
type FileLimitConfig struct {
	ConfigID           int64     `gorm:"primaryKey;autoIncrement" json:"config_id"`
	TenantID           int64     `gorm:"column:tenant_id;not null;uniqueIndex:idx_tenant_type" json:"tenant_id"`
	FileType           string    `gorm:"column:file_type;type:varchar(20);not null;uniqueIndex:idx_tenant_type" json:"file_type"`
	MaxSizeMB          int       `gorm:"column:max_size_mb;not null" json:"max_size_mb"`
	AllowedExtensions  string    `gorm:"column:allowed_extensions;type:varchar(500)" json:"allowed_extensions"`
	Status             int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (FileLimitConfig) TableName() string { return "file_limit_config" }

// --- Request / Response types ---

// SensitiveCheckReq 敏感词检测请求
type SensitiveCheckReq struct {
	TenantID int64     `json:"tenant_id"`
	Content  string    `json:"content" binding:"required"`
	MsgType  int8      `json:"msg_type"`
}

// SensitiveCheckResp 敏感词检测响应
type SensitiveCheckResp struct {
	Passed    bool     `json:"passed"`
	HitWords  []string `json:"hit_words,omitempty"`
	Cleaned   string   `json:"cleaned,omitempty"`   // replacement result
	Blocked   bool     `json:"blocked,omitempty"`   // blocked by strategy=2
}

// RateLimitCheckReq 频控检测请求
type RateLimitCheckReq struct {
	TenantID   int64 `json:"tenant_id"`
	TargetID   int64 `json:"target_id" binding:"required"`
	TargetType int8  `json:"target_type" binding:"required"` // 1=user, 2=bot
}

// RateLimitCheckResp 频控检测响应
type RateLimitCheckResp struct {
	Passed  bool `json:"passed"`
	Remaining int `json:"remaining"`
}

// FileLimitCheckReq 文件限制检测请求
type FileLimitCheckReq struct {
	TenantID int64  `json:"tenant_id"`
	FileType string `json:"file_type" binding:"required"` // image/video/document
	FileSize int    `json:"file_size" binding:"required"` // bytes
}

// FileLimitCheckResp 文件限制检测响应
type FileLimitCheckResp struct {
	Passed     bool   `json:"passed"`
	MaxSizeMB  int    `json:"max_size_mb,omitempty"`
	Extensions string `json:"extensions,omitempty"`
}

// CheckChainResp 全链路校验结果
type CheckChainResp struct {
	Passed        bool                 `json:"passed"`
	SensitiveCheck *SensitiveCheckResp `json:"sensitive_check,omitempty"`
	RateLimitCheck *RateLimitCheckResp `json:"rate_limit_check,omitempty"`
	FileLimitCheck *FileLimitCheckResp `json:"file_limit_check,omitempty"`
}

// --- CRUD req/resp ---

type SensitiveWordCreateReq struct {
	TenantID    int64  `json:"tenant_id"`
	Word        string `json:"word" binding:"required"`
	Strategy    int8   `json:"strategy"`
	Replacement string `json:"replacement"`
	Category    string `json:"category"`
}

type SensitiveWordUpdateReq struct {
	Word        string `json:"word"`
	Strategy    int8   `json:"strategy"`
	Replacement string `json:"replacement"`
	Category    string `json:"category"`
	Status      *int8  `json:"status"`
}

type SensitiveWordBatchReq struct {
	TenantID int64                `json:"tenant_id"`
	Words    []SensitiveWordInput `json:"words" binding:"required"`
}

type SensitiveWordInput struct {
	Word        string `json:"word" binding:"required"`
	Strategy    int8   `json:"strategy"`
	Replacement string `json:"replacement"`
	Category    string `json:"category"`
}

type RateLimitRuleCreateReq struct {
	TenantID          int64 `json:"tenant_id"`
	TargetType        int8  `json:"target_type" binding:"required"`
	MaxCount          int   `json:"max_count" binding:"required"`
	TimeWindowSeconds int   `json:"time_window_seconds" binding:"required"`
}

type FileLimitCreateReq struct {
	TenantID          int64  `json:"tenant_id"`
	FileType          string `json:"file_type" binding:"required"`
	MaxSizeMB         int    `json:"max_size_mb" binding:"required"`
	AllowedExtensions string `json:"allowed_extensions"`
}
