package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"

	"github.com/shulian-paas/im/session-svc/internal/model"
	"github.com/shulian-paas/im/session-svc/internal/repo"
)

// MessagePushEvent 消息推送事件（来自 message-svc）
type MessagePushEvent struct {
	EventType      string `json:"event_type"`
	MsgID          string `json:"msg_id"`
	ConversationID int64  `json:"conversation_id"`
	ConvType       int8   `json:"conv_type"` // 1=single, 2=group
	SenderID       int64  `json:"sender_id"`
	MsgType        int8   `json:"msg_type"`
	Content        string `json:"content,omitempty"`
	SendTime       int64  `json:"send_time"`
	TenantID       int64  `json:"tenant_id"`
}

// Consumer 消费 message_push topic 的消息事件
type Consumer struct {
	reader *kafka.Reader
	repo   *repo.MySQLRepo
	cache  *repo.Cache
}

func NewConsumer(brokers []string, topic string, groupID string, mysqlRepo *repo.MySQLRepo, redisCache *repo.Cache) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafka.LastOffset,
			MinBytes:    1,
			MaxBytes:    10e6,
			MaxWait:     1 * time.Second,
		}),
		repo:  mysqlRepo,
		cache: redisCache,
	}
}

// Start 启动消费者 goroutine
func (c *Consumer) Start(ctx context.Context) {
	go func() {
		log.Info().Str("topic", c.reader.Config().Topic).Msg("session consumer started")
		defer log.Info().Str("topic", c.reader.Config().Topic).Msg("session consumer stopped")

		for {
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Err(err).Msg("read message_push failed")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			c.processMessage(msg.Value)
		}
	}()
}

func (c *Consumer) processMessage(data []byte) {
	var event MessagePushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Err(err).Msg("parse message_push event failed")
		return
	}

	switch event.EventType {
	case "message:new":
		c.handleMessageNew(event)
	default:
		log.Debug().Str("event_type", event.EventType).Msg("ignoring unhandled event type")
	}
}

func (c *Consumer) handleMessageNew(event MessagePushEvent) {
	convID := resolveConvID(event.ConvType, event.ConversationID, event.SenderID)
	if convID == "" {
		log.Warn().Int8("conv_type", event.ConvType).
			Int64("conversation_id", event.ConversationID).Msg("unknown conv_type, skipping")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessions, err := c.repo.FindByConversation(ctx, convID)
	if err != nil {
		log.Err(err).Str("conv_id", convID).Msg("find sessions by conversation failed")
		return
	}
	if len(sessions) == 0 {
		log.Debug().Str("conv_id", convID).Msg("no sessions found for conversation")
		return
	}

	now := time.Now()
	summary := messageSummary(event.MsgType, event.Content)

	for i := range sessions {
		sess := &sessions[i]
		updates := map[string]interface{}{
			"last_message":    summary,
			"last_msg_type":   event.MsgType,
			"last_sender_id":  event.SenderID,
			"last_message_at": now,
		}

		// 非发送者：增加未读计数
		if sess.UserID != event.SenderID {
			updates["unread_count"] = sess.UnreadCount + 1
		}

		if err := c.repo.UpdateSession(ctx, sess.SessionID, updates); err != nil {
			log.Err(err).Int64("session_id", sess.SessionID).
				Int64("user_id", sess.UserID).Msg("update session failed")
			continue
		}

		// 同步 Redis 未读缓存
		if newCount, ok := updates["unread_count"].(int); ok {
			_ = c.cache.SetUnreadCount(ctx, sess.UserID, sess.SessionID, newCount)
		}
	}
}

// resolveConvID 根据会话类型构建 session 表的 conversation_id
func resolveConvID(convType int8, conversationID int64, senderID int64) string {
	switch convType {
	case 1: // single: event.ConversationID = other user's ID
		minID, maxID := senderID, conversationID
		if minID > maxID {
			minID, maxID = maxID, minID
		}
		return fmt.Sprintf("s_%d_%d", minID, maxID)
	case 2: // group
		return fmt.Sprintf("g_%d", conversationID)
	case 3: // bot
		return fmt.Sprintf("b_%d", conversationID)
	default:
		return ""
	}
}

// messageSummary 从消息类型和内容中提取会话列表展示摘要
func messageSummary(msgType int8, content string) string {
	switch msgType {
	case 1: // text
		var m map[string]interface{}
		if json.Unmarshal([]byte(content), &m) == nil {
			if text, ok := m["text"].(string); ok {
				runes := []rune(text)
				if len(runes) > 100 {
					return string(runes[:100])
				}
				return text
			}
		}
		return content
	case 2:
		return "[图片]"
	case 3:
		return "[视频]"
	case 4:
		return "[文件]"
	case 5:
		return "[语音]"
	case 6:
		return "[名片]"
	case 7:
		return "[商务名片]"
	case 9:
		return "[流式消息]"
	case 10:
		return "[合并转发]"
	default:
		runes := []rune(content)
		if len(runes) > 100 {
			return string(runes[:100])
		}
		return content
	}
}

// Stop 关闭消费者
func (c *Consumer) Stop() {
	if err := c.reader.Close(); err != nil {
		log.Err(err).Msg("close session consumer")
	}
}
