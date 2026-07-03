package consumer

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/service"
)

// BotTriggerConsumer 消费 Kafka bot_trigger topic
// 从 message-svc 接收 @触发 事件，交由 BotService 处理
type BotTriggerConsumer struct {
	reader *kafka.Reader
	svc    *service.BotService
}

func NewBotTriggerConsumer(brokers []string, topic string, groupID string, svc *service.BotService) *BotTriggerConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10,
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.LastOffset,
	})

	return &BotTriggerConsumer{
		reader: reader,
		svc:    svc,
	}
}

// Start 开始消费 bot_trigger 事件（阻塞）
func (c *BotTriggerConsumer) Start(ctx context.Context) error {
	log.Info().Str("topic", c.reader.Config().Topic).Msg("starting bot_trigger consumer")

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // context cancelled
			}
			log.Err(err).Msg("read bot_trigger message failed")
			continue
		}

		c.processMessage(ctx, msg)
	}
}

func (c *BotTriggerConsumer) processMessage(ctx context.Context, msg kafka.Message) {
	var event model.BotTriggerEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Err(err).Str("topic", msg.Topic).Int("partition", msg.Partition).
			Int64("offset", msg.Offset).Msg("parse bot_trigger event failed")
		return
	}

	log.Info().Str("event_id", event.EventID).Str("text", event.Message.Text).
		Int64s("bot_ids", event.BotIDs).Msg("received bot_trigger event")

	if err := c.svc.HandleTrigger(ctx, &event); err != nil {
		log.Err(err).Str("event_id", event.EventID).Msg("handle bot_trigger failed")
	}
}

func (c *BotTriggerConsumer) Close() error {
	return c.reader.Close()
}
