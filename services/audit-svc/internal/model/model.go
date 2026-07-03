package model

import "time"

// AdminOpLog 管理员操作日志
type AdminOpLog struct {
	LogID           int64     `gorm:"primaryKey;autoIncrement" json:"log_id"`
	TenantID        int64     `gorm:"column:tenant_id;not null;index:idx_tenant_time" json:"tenant_id"`
	OperatorID      int64     `gorm:"column:operator_id;not null;index" json:"operator_id"`
	OperatorAccount string    `gorm:"column:operator_account;type:varchar(100)" json:"operator_account"`
	OpType          string    `gorm:"column:op_type;type:varchar(50);not null;index" json:"op_type"`
	TargetID        string    `gorm:"column:target_id;type:varchar(100)" json:"target_id"`
	TargetDesc      string    `gorm:"column:target_desc;type:varchar(500)" json:"target_desc"`
	Detail          string    `gorm:"column:detail;type:text" json:"detail,omitempty"`
	Result          bool      `gorm:"column:result;not null" json:"result"`
	IP              string    `gorm:"column:ip;type:varchar(45)" json:"ip"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;index:idx_tenant_time" json:"created_at"`
}

func (AdminOpLog) TableName() string { return "audit_admin_op_log" }

// MsgAuditLog 消息审计日志
type MsgAuditLog struct {
	LogID        int64     `gorm:"primaryKey;autoIncrement" json:"log_id"`
	TenantID     int64     `gorm:"column:tenant_id;not null;index:idx_tenant_time" json:"tenant_id"`
	MsgID        string    `gorm:"column:msg_id;type:varchar(64);index" json:"msg_id"`
	SenderID     int64     `gorm:"column:sender_id;not null;index" json:"sender_id"`
	SessionID    string    `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	SessionType  int8      `gorm:"column:session_type;default:1;comment:'1=single,2=group'" json:"session_type"`
	MsgType      int8      `gorm:"column:msg_type;not null" json:"msg_type"`
	Content      string    `gorm:"column:content;type:text" json:"content,omitempty"`
	HasSensitive bool      `gorm:"column:has_sensitive;default:false" json:"has_sensitive"`
	IP           string    `gorm:"column:ip;type:varchar(45)" json:"ip"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime;index:idx_tenant_time" json:"created_at"`
}

func (MsgAuditLog) TableName() string { return "audit_msg_log" }

// --- Request / Response types ---

type CreateAdminOpLogReq struct {
	TenantID        int64  `json:"tenant_id"`
	OperatorID      int64  `json:"operator_id"`
	OperatorAccount string `json:"operator_account"`
	OpType          string `json:"op_type" binding:"required"`
	TargetID        string `json:"target_id"`
	TargetDesc      string `json:"target_desc"`
	Detail          string `json:"detail"`
	Result          bool   `json:"result"`
	IP              string `json:"ip"`
}

type AdminOpLogListReq struct {
	TenantID   int64  `form:"-"`
	OperatorID int64  `form:"operator_id"`
	OpType     string `form:"op_type"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type AdminOpLogListResp struct {
	List     []AdminOpLog `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type CreateMsgAuditLogReq struct {
	TenantID    int64  `json:"tenant_id"`
	MsgID       string `json:"msg_id"`
	SenderID    int64  `json:"sender_id" binding:"required"`
	SessionID   string `json:"session_id" binding:"required"`
	SessionType int8   `json:"session_type"`
	MsgType     int8   `json:"msg_type" binding:"required"`
	Content     string `json:"content"`
	HasSensitive bool  `json:"has_sensitive"`
	IP          string `json:"ip"`
}

type MsgAuditLogListReq struct {
	TenantID    int64  `form:"-"`
	SenderID    int64  `form:"sender_id"`
	SessionID   string `form:"session_id"`
	MsgType     *int8  `form:"msg_type"`
	HasSensitive *bool `form:"has_sensitive"`
	Keyword     string `form:"keyword"`
	StartTime   string `form:"start_time"`
	EndTime     string `form:"end_time"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type MsgAuditLogListResp struct {
	List     []MsgAuditLog `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// Batch create
type BatchCreateMsgLogReq struct {
	Logs []CreateMsgAuditLogReq `json:"logs" binding:"required"`
}
