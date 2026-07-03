package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// MessagePushEvent Kafka message_push 事件
type MessagePushEvent struct {
	EventType      string `json:"event_type"`      // message:new, message:recalled, message:status
	MsgID          string `json:"msg_id"`
	ConversationID int64  `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	MsgType        int8   `json:"msg_type"`
	Content        string `json:"content,omitempty"` // JSON 摘要
	SendTime       int64  `json:"send_time"`
	TenantID       int64  `json:"tenant_id"`
}

// BotTriggerEvent Kafka bot_trigger 事件
type BotTriggerEvent struct {
	EventID      string             `json:"event_id"`
	EventType    string             `json:"event_type"` // "message.mention"
	Timestamp    int64              `json:"timestamp"`
	Message      BotTriggerMessage  `json:"message"`
	Sender       BotTriggerSender   `json:"sender"`
	Conversation BotTriggerConv     `json:"conversation"`
	BotIDs       []int64            `json:"bot_ids"`
}

type BotTriggerMessage struct {
	MsgID       string  `json:"msg_id"`
	ClientMsgID string  `json:"client_msg_id"`
	Text        string  `json:"text"`
	MsgType     int8    `json:"msg_type"`
	AtUserIDs   []int64 `json:"at_user_ids"`
}

type BotTriggerSender struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}

type BotTriggerConv struct {
	ConvID   string `json:"conv_id"`
	ConvType int8   `json:"conv_type"`
	GroupID  *int64 `json:"group_id,omitempty"`
}

// Producer Kafka 消息发布器
type Producer struct {
	pushWriter    *kafka.Writer
	triggerWriter *kafka.Writer
}

func NewProducer(brokers []string, pushTopic, triggerTopic string) *Producer {
	return &Producer{
		pushWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    pushTopic,
			Balancer: &kafka.Hash{},
		},
		triggerWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    triggerTopic,
			Balancer: &kafka.Hash{},
		},
	}
}

// PublishMessageNew 发布新消息事件
func (p *Producer) PublishMessageNew(ctx context.Context, event *MessagePushEvent) error {
	event.EventType = "message:new"
	return p.publish(ctx, p.pushWriter, event)
}

// PublishMessageRecalled 发布消息撤回事件
func (p *Producer) PublishMessageRecalled(ctx context.Context, event *MessagePushEvent) error {
	event.EventType = "message:recalled"
	return p.publish(ctx, p.pushWriter, event)
}

// PublishBotTrigger 发布 @机器人触发事件
func (p *Producer) PublishBotTrigger(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error {
	event := BotTriggerEvent{
		EventID:   "evt_" + uuid.New().String(),
		EventType: "message.mention",
		Timestamp: time.Now().UnixMilli(),
		Message: BotTriggerMessage{
			MsgID:     msgID,
			Text:       content,
			MsgType:   msgType,
			AtUserIDs: atUserIDs,
		},
		Sender: BotTriggerSender{
			UserID:   senderID,
			UserName: senderName,
		},
		Conversation: BotTriggerConv{
			ConvID:   fmt.Sprintf("%d", convID),
			ConvType: convType,
			GroupID:  groupID,
		},
		BotIDs: atUserIDs,
	}
	return p.publish(ctx, p.triggerWriter, event)
}

// Close 关闭 Kafka Writer
func (p *Producer) Close() {
	if err := p.pushWriter.Close(); err != nil {
		log.Err(err).Msg("close push writer")
	}
	if err := p.triggerWriter.Close(); err != nil {
		log.Err(err).Msg("close trigger writer")
	}
}

func (p *Producer) publish(ctx context.Context, writer *kafka.Writer, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return writer.WriteMessages(ctx, kafka.Message{Value: data})
}
