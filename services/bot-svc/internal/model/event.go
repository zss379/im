package model

// BotTriggerEvent Kafka bot_trigger 事件结构
type BotTriggerEvent struct {
	EventID      string              `json:"event_id"`
	EventType    string              `json:"event_type"` // "message.mention"
	Timestamp    int64               `json:"timestamp"`
	Message      MessageContext      `json:"message"`
	Sender       SenderInfo          `json:"sender"`
	Conversation ConversationContext `json:"conversation"`
	BotIDs       []int64             `json:"bot_ids"`
}

// MessageContext 触发消息的上下文
type MessageContext struct {
	MsgID       string  `json:"msg_id"`
	ClientMsgID string  `json:"client_msg_id"`
	SendTime    int64   `json:"send_time"`
	Text        string  `json:"text"`
	AtUserIDs   []int64 `json:"at_user_ids"`
	MsgType     int8    `json:"msg_type"`
}

// SenderInfo 发送者信息
type SenderInfo struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}

// ConversationContext 会话上下文
type ConversationContext struct {
	ConvID      string `json:"conv_id"`
	ConvType    int8   `json:"conv_type"` // 1=单聊, 2=群聊
	GroupID     *int64 `json:"group_id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	ReceiverID  *int64 `json:"receiver_id,omitempty"` // 单聊时对方用户ID
}

// WebhookPayload 发送给外部系统的 Webhook 请求体
type WebhookPayload struct {
	Event    string          `json:"event"`    // "message.mention"
	BotID    int64           `json:"bot_id"`
	Trigger  TriggerContext  `json:"trigger"`
	Response ResponseModeDef `json:"response_mode"`
}

// TriggerContext 触发上下文
type TriggerContext struct {
	MsgID        string       `json:"msg_id"`
	Text         string       `json:"text"`
	Sender       SenderInfo   `json:"sender"`
	Conversation ConvBrief    `json:"conversation"`
	Mentions     []MentionBot `json:"mentions"`
}

// ConvBrief 会话摘要
type ConvBrief struct {
	ConvID    string `json:"conv_id"`
	ConvType  int8   `json:"conv_type"`
	GroupName string `json:"group_name,omitempty"`
	GroupID   *int64 `json:"group_id,omitempty"`
}

// MentionBot 被 @ 的机器人信息
type MentionBot struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}

// ResponseModeDef 响应模式定义
type ResponseModeDef struct {
	Type        string `json:"type"`        // "sync" | "async" | "sse"
	TimeoutMs   int    `json:"timeout_ms"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// WebhookResponse 外部系统响应
type WebhookResponse struct {
	Reply     string `json:"reply"`
	ReplyType string `json:"reply_type"` // "text"
	ReplyRaw  any    `json:"reply_raw,omitempty"`

	// Async 模式
	Status      string `json:"status,omitempty"` // "accepted"
	EventID     string `json:"event_id,omitempty"`

	// SSE 模式
	SSEURL      string `json:"sse_url,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

// CallbackRequest 异步回调请求
type CallbackRequest struct {
	EventID   string `json:"event_id" binding:"required"`
	Reply     string `json:"reply" binding:"required"`
	ReplyType string `json:"reply_type" default:"text"`
}
