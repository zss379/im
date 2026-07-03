package model

import "time"

// BotType 机器人类型
type BotType int8

const (
	BotTypeSystem   BotType = 1 // 系统机器人
	BotTypeCustom   BotType = 2 // 用户自定义
)

// ResponseMode Webhook 响应模式
type ResponseMode int8

const (
	ResponseModeSync  ResponseMode = 1 // 同步
	ResponseModeAsync ResponseMode = 2 // 异步
	ResponseModeSSE   ResponseMode = 3 // 流式
)

// Bot 机器人配置
type Bot struct {
	BotID        int64        `db:"bot_id"         json:"bot_id"`
	TenantID     int64        `db:"tenant_id"      json:"tenant_id"`
	BotType      BotType      `db:"bot_type"       json:"bot_type"`
	BotName      string       `db:"bot_name"       json:"bot_name"`
	AvatarURL    *string      `db:"avatar_url"     json:"avatar_url"`
	Description  *string      `db:"description"    json:"description"`
	WebhookURL   *string      `db:"webhook_url"    json:"webhook_url"`
	APIKey       *string      `db:"api_key"        json:"api_key,omitempty"`
	OpenIMUserID *int64       `db:"openim_user_id" json:"openim_user_id"`
	ResponseMode ResponseMode `db:"response_mode"  json:"response_mode"`
	CallbackURL  *string      `db:"callback_url"   json:"callback_url"`
	IPWhitelist  *string      `db:"ip_whitelist"   json:"ip_whitelist,omitempty"`
	Status       int8         `db:"status"         json:"status"`
	CreatorID    *int64       `db:"creator_id"     json:"creator_id"`
	CreatedAt    time.Time    `db:"created_at"     json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"     json:"updated_at"`
}

// GroupBot 群机器人关联
type GroupBot struct {
	ID           int64        `db:"id"             json:"id"`
	GroupID      int64        `db:"group_id"       json:"group_id"`
	BotID        int64        `db:"bot_id"         json:"bot_id"`
	WebhookURL   *string      `db:"webhook_url"    json:"webhook_url"`
	Token        *string      `db:"token"          json:"token,omitempty"`
	ResponseMode ResponseMode `db:"response_mode"  json:"response_mode"`
	CallbackURL  *string      `db:"callback_url"   json:"callback_url"`
	Status       int8         `db:"status"         json:"status"`
	CreatedAt    time.Time    `db:"created_at"     json:"created_at"`
}
