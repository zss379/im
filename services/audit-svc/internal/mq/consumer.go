package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"

	"github.com/shulian-paas/im/audit-svc/internal/model"
	"github.com/shulian-paas/im/audit-svc/internal/repo"
)

// BlockedMessageEvent 拦截消息事件（来自 message-svc）
type BlockedMessageEvent struct {
	EventType      string `json:"event_type"`
	TenantID       int64  `json:"tenant_id"`
	SenderID       int64  `json:"sender_id"`
	ConversationID int64  `json:"conversation_id"`
	ConvType       int8   `json:"conv_type"`
	MsgType        int8   `json:"msg_type"`
	Content        string `json:"content,omitempty"`
	BlockedReason  string `json:"blocked_reason"`
	BlockedDetail  string `json:"blocked_detail,omitempty"`
	BlockedAt      int64  `json:"blocked_at"`
}

// Consumer 消费拦截消息并写入审计日志
type Consumer struct {
	reader *kafka.Reader
	repo   *repo.MySQLRepo
}

func NewConsumer(brokers []string, topic string, groupID string, repo *repo.MySQLRepo) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafka.LastOffset,
			MinBytes:    1,
			MaxBytes:    10e6, // 10MB
			MaxWait:     1 * time.Second,
		}),
		repo: repo,
	}
}

// Start 启动消费者 goroutine
func (c *Consumer) Start(ctx context.Context) {
	go func() {
		log.Info().Msg("blocked-message consumer started")
		defer log.Info().Msg("blocked-message consumer stopped")

		for {
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Err(err).Msg("read blocked-message failed")
				time.Sleep(100 * time.Millisecond)
				continue
			}

			c.processMessage(msg.Value)
		}
	}()
}

func (c *Consumer) processMessage(data []byte) {
	var event BlockedMessageEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Err(err).Msg("parse blocked-message event failed")
		return
	}

	logEntry := &model.MsgAuditLog{
		TenantID:     event.TenantID,
		MsgID:        "",
		SenderID:     event.SenderID,
		SessionID:    fmt.Sprintf("%d", event.ConversationID),
		SessionType:  event.ConvType,
		MsgType:      event.MsgType,
		Content:      event.Content,
		HasSensitive: event.BlockedReason == "sensitive_word",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.repo.CreateMsgAuditLog(ctx, logEntry); err != nil {
		log.Err(err).Str("reason", event.BlockedReason).
			Int64("sender_id", event.SenderID).Msg("store blocked-message failed")
	}
}

// Stop 关闭消费者
func (c *Consumer) Stop() {
	if err := c.reader.Close(); err != nil {
		log.Err(err).Msg("close blocked-message consumer")
	}
}
