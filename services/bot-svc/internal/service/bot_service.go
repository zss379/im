package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
)

// BotService 是 @触发 的核心编排器
// 职责:
//   - 消费 bot_trigger Kafka 事件
//   - 检测 @ 命中哪些机器人
//   - 按 response_mode 分发到对应的 webhook service
//   - 将回复发回 message-svc
type BotService struct {
	cache     *repo.BotCache
	botRepo   *repo.BotRepo
	msgClient *MessageClient

	syncSvc  *SyncWebhookService
	asyncSvc *AsyncWebhookService
	sseSvc   *SSEWebhookService
}

func NewBotService(
	cache *repo.BotCache,
	botRepo *repo.BotRepo,
	msgClient *MessageClient,
	syncSvc *SyncWebhookService,
	asyncSvc *AsyncWebhookService,
	sseSvc *SSEWebhookService,
) *BotService {
	return &BotService{
		cache:     cache,
		botRepo:   botRepo,
		msgClient: msgClient,
		syncSvc:   syncSvc,
		asyncSvc:  asyncSvc,
		sseSvc:    sseSvc,
	}
}

// HandleTrigger 处理 bot_trigger 事件
// 这是 @触发 的核心入口
func (s *BotService) HandleTrigger(ctx context.Context, event *model.BotTriggerEvent) error {
	log.Info().Str("event_id", event.EventID).Int64s("bot_ids", event.BotIDs).
		Msg("handling bot trigger")

	for _, botID := range event.BotIDs {
		if err := s.triggerBot(ctx, botID, event); err != nil {
			log.Err(err).Int64("bot_id", botID).Msg("trigger bot failed")
		}
	}
	return nil
}

func (s *BotService) triggerBot(ctx context.Context, botID int64, event *model.BotTriggerEvent) error {
	// 1. 从 Redis 获取机器人配置
	bot, err := s.cache.GetConfig(ctx, botID)
	if err != nil {
		log.Warn().Err(err).Int64("bot_id", botID).Msg("bot config not in cache, fallback to DB")
		// fallback to DB
		dbBot, err := s.botRepo.GetByID(botID)
		if err != nil {
			return fmt.Errorf("bot not found: %w", err)
		}
		// populate cache
		_ = s.cache.SetConfig(ctx, dbBot)
		bot = dbBot
	}

	// 2. 检查机器人状态
	if bot.Status != 1 {
		log.Debug().Int64("bot_id", botID).Msg("bot is disabled, skip")
		return nil
	}

	if bot.WebhookURL == nil || *bot.WebhookURL == "" {
		log.Warn().Int64("bot_id", botID).Msg("bot has no webhook URL, skip")
		return nil
	}

	// 3. 构建 Webhook payload
	payload := s.buildPayload(bot, event)

	// 4. 按响应模式分发
	switch bot.ResponseMode {
	case model.ResponseModeSync:
		return s.handleSync(ctx, bot, payload, event)
	case model.ResponseModeAsync:
		return s.handleAsync(ctx, bot, payload, event)
	case model.ResponseModeSSE:
		return s.handleSSE(ctx, bot, payload, event)
	default:
		return s.handleSync(ctx, bot, payload, event)
	}
}

func (s *BotService) handleSync(ctx context.Context, bot *model.Bot, payload *model.WebhookPayload, event *model.BotTriggerEvent) error {
	resp, err := s.syncSvc.Invoke(ctx, *bot.WebhookURL, payload)
	if err != nil {
		log.Err(err).Int64("bot_id", bot.BotID).Msg("sync webhook failed")
		return err
	}

	return s.msgClient.SendReply(bot.BotID, resp.Reply, resp.ReplyType, event.Conversation)
}

func (s *BotService) handleAsync(ctx context.Context, bot *model.Bot, payload *model.WebhookPayload, event *model.BotTriggerEvent) error {
	eventID, err := s.asyncSvc.Invoke(ctx, *bot.WebhookURL, payload)
	if err != nil {
		log.Err(err).Int64("bot_id", bot.BotID).Msg("async webhook invoke failed")
		return err
	}
	log.Info().Int64("bot_id", bot.BotID).Str("event_id", eventID).
		Msg("async webhook accepted, waiting for callback")
	return nil
}

func (s *BotService) handleSSE(ctx context.Context, bot *model.Bot, payload *model.WebhookPayload, event *model.BotTriggerEvent) error {
	return s.sseSvc.Invoke(ctx, *bot.WebhookURL, payload)
}

func (s *BotService) buildPayload(bot *model.Bot, event *model.BotTriggerEvent) *model.WebhookPayload {
	return &model.WebhookPayload{
		Event: "message.mention",
		BotID: bot.BotID,
		Trigger: model.TriggerContext{
			MsgID: event.Message.MsgID,
			Text:  event.Message.Text,
			Sender: model.SenderInfo{
				UserID:   event.Sender.UserID,
				UserName: event.Sender.UserName,
			},
			Conversation: model.ConvBrief{
				ConvID:    event.Conversation.ConvID,
				ConvType:  event.Conversation.ConvType,
				GroupName: event.Conversation.GroupName,
				GroupID:   event.Conversation.GroupID,
			},
			Mentions: []model.MentionBot{
				{UserID: bot.BotID, UserName: bot.BotName},
			},
		},
		Response: model.ResponseModeDef{
			Type:      modeToString(bot.ResponseMode),
			TimeoutMs: 3000,
		},
	}
}

func modeToString(m model.ResponseMode) string {
	switch m {
	case model.ResponseModeSync:
		return "sync"
	case model.ResponseModeAsync:
		return "async"
	case model.ResponseModeSSE:
		return "sse"
	default:
		return "sync"
	}
}
