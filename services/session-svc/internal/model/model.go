package model

import "time"

// Session types
const (
	SessionTypeSingle = 1
	SessionTypeGroup  = 2
	SessionTypeBot    = 3
)

// Capacity limits
const (
	MaxPinnedSessions  = 5
	MaxSessionsPerUser = 2000
)

// Session 会话
type Session struct {
	SessionID      int64      `gorm:"primaryKey;autoIncrement" json:"session_id"`
	TenantID       int64      `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	UserID         int64      `gorm:"column:user_id;not null;uniqueIndex:idx_user_conv" json:"user_id"`
	ConversationID string     `gorm:"column:conversation_id;type:varchar(64);not null;uniqueIndex:idx_user_conv" json:"conversation_id"`
	Type           int8       `gorm:"column:type;not null;comment:'1=single,2=group,3=bot'" json:"type"`
	TargetID       int64      `gorm:"column:target_id;not null;comment:'other user ID or group ID or bot ID'" json:"target_id"`
	IsPinned       bool       `gorm:"column:is_pinned;default:false" json:"is_pinned"`
	PinnedAt       *time.Time `gorm:"column:pinned_at" json:"pinned_at"`
	IsMuted        bool       `gorm:"column:is_muted;default:false" json:"is_muted"`
	UnreadCount    int        `gorm:"column:unread_count;default:0" json:"unread_count"`
	LastMessage    string     `gorm:"column:last_message;type:varchar(500)" json:"last_message"`
	LastMsgType    int8       `gorm:"column:last_msg_type;default:0" json:"last_msg_type"`
	LastSenderID   int64      `gorm:"column:last_sender_id" json:"last_sender_id"`
	LastMessageAt  *time.Time `gorm:"column:last_message_at;index" json:"last_message_at"`
	IsDeleted      bool       `gorm:"column:is_deleted;default:false" json:"is_deleted"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Session) TableName() string { return "session" }

// --- Request / Response types ---

type CreateSessionReq struct {
	Type     int8  `json:"type" binding:"required"`
	TargetID int64 `json:"target_id" binding:"required"`
}

type SessionListReq struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Type     *int8  `form:"type"`
	Unread   *bool  `form:"unread"`
	Keyword  string `form:"keyword"`
}

type SessionListResp struct {
	List     []SessionSummary `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type SessionSummary struct {
	SessionID     int64  `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	Type          int8   `json:"type"`
	TargetID      int64  `json:"target_id"`
	IsPinned      bool   `json:"is_pinned"`
	IsMuted       bool   `json:"is_muted"`
	UnreadCount   int    `json:"unread_count"`
	LastMessage   string `json:"last_message,omitempty"`
	LastMsgType   int8   `json:"last_msg_type"`
	LastSenderID  int64  `json:"last_sender_id"`
	LastMessageAt string `json:"last_message_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type PinSessionReq struct {
	Pinned bool `json:"pinned" binding:"required"`
}

type MuteSessionReq struct {
	Muted bool `json:"muted" binding:"required"`
}

type BatchUnreadReq struct {
	SessionIDs []int64 `json:"session_ids" binding:"required"`
	UnreadCount int    `json:"unread_count" binding:"required"`
}

type SessionDetailResp struct {
	SessionID      int64  `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	Type           int8   `json:"type"`
	TargetID       int64  `json:"target_id"`
	IsPinned       bool   `json:"is_pinned"`
	IsMuted        bool   `json:"is_muted"`
	UnreadCount    int    `json:"unread_count"`
	LastMessage    string `json:"last_message,omitempty"`
	LastMsgType    int8   `json:"last_msg_type"`
	LastSenderID   int64  `json:"last_sender_id"`
	LastMessageAt  string `json:"last_message_at,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
