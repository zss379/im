package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// SessionSyncEvent 会话变更同步事件
type SessionSyncEvent struct {
	EventType  string `json:"event_type"`             // session:created, session:pin, session:mute, session:read, session:deleted
	SessionID  int64  `json:"session_id"`
	UserID     int64  `json:"user_id"`
	TenantID   int64  `json:"tenant_id"`
	ChangeType string `json:"change_type"`            // pin, mute, read, delete
	Timestamp  int64  `json:"timestamp"`
}

// Producer 发布会话变更事件
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		},
	}
}

// PublishSessionEvent 发布会话变更事件
func (p *Producer) PublishSessionEvent(ctx context.Context, event *SessionSyncEvent) error {
	return p.publish(ctx, event)
}

// Close 关闭 producer
func (p *Producer) Close() {
	if err := p.writer.Close(); err != nil {
		log.Err(err).Msg("close session producer")
	}
}

func (p *Producer) publish(ctx context.Context, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal session event: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: data})
}
