package model

import "time"

// Message 消息文档结构（MongoDB messages 集合）
type Message struct {
	ID             interface{} `bson:"_id,omitempty" json:"-"`
	MsgID          string      `bson:"msg_id" json:"msg_id"`
	TenantID       int64       `bson:"tenant_id" json:"tenant_id"`
	ConversationID int64       `bson:"conversation_id" json:"conversation_id"`
	ConvType       int8        `bson:"conv_type" json:"conv_type"`
	SenderID       int64       `bson:"sender_id" json:"sender_id"`
	SenderName     string      `bson:"sender_name" json:"sender_name"`
	SenderAvatar   string      `bson:"sender_avatar,omitempty" json:"sender_avatar,omitempty"`
	SenderBotID    *int64      `bson:"sender_bot_id,omitempty" json:"sender_bot_id,omitempty"`
	MsgType        int8        `bson:"msg_type" json:"msg_type"`
	Content        MsgContent  `bson:"content" json:"content"`
	Status         int8        `bson:"status" json:"status"` // 2=sent, 4=recalled
	ClientMsgID    string      `bson:"client_msg_id,omitempty" json:"client_msg_id,omitempty"`
	SendTime       time.Time   `bson:"send_time" json:"send_time"`
	RecallTime     *time.Time  `bson:"recall_time,omitempty" json:"recall_time,omitempty"`
	EditTime       *time.Time  `bson:"edit_time,omitempty" json:"edit_time,omitempty"`
	IsDeleted      bool        `bson:"is_deleted" json:"-"`
	AtUserList     []int64     `bson:"at_user_list,omitempty" json:"at_user_list,omitempty"`
	ExtFields      interface{} `bson:"ext_fields,omitempty" json:"ext_fields,omitempty"`
}

// MsgContent 消息内容体（按 msg_type 不同结构）
type MsgContent map[string]interface{}

// MessageStatus 消息状态
const (
	MsgStatusSending = 1
	MsgStatusSent    = 2
	MsgStatusFailed  = 3
	MsgStatusRecalled = 4
)

// MsgType 消息类型
const (
	MsgTypeText       = 1
	MsgTypeImage      = 2
	MsgTypeVideo      = 3
	MsgTypeFile       = 4
	MsgTypeVoice      = 5
	MsgTypeCard       = 6
	MsgTypeBusinessCard = 7
	MsgTypeSystem     = 8
	MsgTypeSSE        = 9
	MsgTypeMergeForward = 10
)

// SendMessageReq POST /messages 请求
type SendMessageReq struct {
	ClientMsgID    string        `json:"client_msg_id"`
	MsgType        int8          `json:"msg_type"`
	Content        MsgContent    `json:"content" binding:"required"`
	ConversationID int64         `json:"conversation_id" binding:"required"`
	ConvType       int8          `json:"conv_type"`
	GroupID        *int64        `json:"group_id"`
	AtUserList     []int64       `json:"at_user_list"`
	SenderID       int64         `json:"sender_id"`
	SenderName     string        `json:"sender_name"`
	SenderAvatar   string        `json:"sender_avatar"`
	SenderBotID    *int64        `json:"sender_bot_id"`
}

// SendMessageResp POST /messages 响应 data
type SendMessageResp struct {
	MsgID    string `json:"msg_id"`
	SendTime string `json:"send_time"`
	Status   int8   `json:"status"`
}

// MessageResp 消息查询响应体
type MessageResp struct {
	MsgID          string     `json:"msg_id"`
	SenderID       int64      `json:"sender_id"`
	SenderName     string     `json:"sender_name"`
	SenderAvatar   string     `json:"sender_avatar,omitempty"`
	SenderBotID    *int64     `json:"sender_bot_id,omitempty"`
	MsgType        int8       `json:"msg_type"`
	Content        MsgContent `json:"content"`
	Status         int8       `json:"status"`
	SendTime       time.Time  `json:"send_time"`
	RecallTime     *time.Time `json:"recall_time,omitempty"`
	AtUserList     []int64    `json:"at_user_list,omitempty"`
}

// PullMessagesResp GET /messages 响应
type PullMessagesResp struct {
	List     []MessageResp `json:"list"`
	Cursor   string        `json:"cursor"`
	HasMore  bool          `json:"has_more"`
}

// SearchReq GET /messages/search 查询参数
type SearchReq struct {
	Q              string `form:"q"`
	ConversationID int64  `form:"conversation_id"`
	SenderID       int64  `form:"sender_id"`
	MsgType        int8   `form:"msg_type"`
	StartTime      string `form:"start_time"`
	EndTime        string `form:"end_time"`
	Page           int    `form:"page" binding:"min=1"`
	PageSize       int    `form:"page_size" binding:"min=1,max=100"`
}

// RecallReq POST /messages/{msg_id}/recall
type RecallReq struct {
	SenderID int64 `json:"sender_id" binding:"required"`
}

// ForwardReq POST /messages/forward
type ForwardReq struct {
	MsgIDs     []string `json:"msg_ids" binding:"required"`
	TargetType int8     `json:"target_type"` // 1=单聊, 2=群聊
	TargetID   int64    `json:"target_id" binding:"required"`
	ForwardType int8    `json:"forward_type"` // 1=逐条, 2=合并
	SenderID   int64    `json:"sender_id" binding:"required"`
	SenderName string   `json:"sender_name"`
}

// ReadReceiptResp POST /messages/{msg_id}/read-receipt 响应
type ReadReceiptResp struct {
	IsRead bool   `json:"is_read"`
	ReadAt string `json:"read_at,omitempty"`
}

// ReadStatusResp GET /messages/{msg_id}/read-status 响应
type ReadStatusResp struct {
	MsgID  string `json:"msg_id"`
	IsRead bool   `json:"is_read"`
	ReadAt string `json:"read_at,omitempty"`
}

// SSETokenReq POST /messages/sse 请求
type SSETokenReq struct {
	BotID int64  `json:"bot_id" binding:"required"`
	MsgID string `json:"msg_id" binding:"required"`
	Token string `json:"token" binding:"required"`
}

// 游标翻页 — 编码/解码
type Cursor struct {
	LastSendTime int64 `json:"t"`
	LastID       string `json:"i"`
}
