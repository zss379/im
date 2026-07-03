package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
)

// AsyncWebhookService 处理异步 Webhook 模式
// 时序: bot-svc → POST → 外部系统 → 202 Accepted → 外部系统回调 callback_url → bot-svc 发送回复
type AsyncWebhookService struct {
	cfg        *config.WebhookConfig
	httpClient *http.Client
	cache      *repo.BotCache
	pendingTTL time.Duration
}

func NewAsyncWebhookService(cfg *config.WebhookConfig, cache *repo.BotCache) *AsyncWebhookService {
	ttl, _ := time.ParseDuration(cfg.PendingTTL)
	return &AsyncWebhookService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:      cache,
		pendingTTL: ttl,
	}
}

// Invoke 调用外部系统 Webhook，返回 event_id 用于跟踪
func (s *AsyncWebhookService) Invoke(ctx context.Context, webhookURL string, payload *model.WebhookPayload) (string, error) {
	eventID := "evt_" + uuid.New().String()
	payload.Response.CallbackURL = fmt.Sprintf("%s?event_id=%s", payload.Response.CallbackURL, eventID)

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", "message.mention")
	req.Header.Set("X-Event-ID", eventID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, int64(s.cfg.MaxBodySizeMB)*1024*1024))

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("unexpected status: %d body=%s", resp.StatusCode, string(respBody))
	}

	// 存储 pending 状态到 Redis（含会话上下文，供回调时使用）
	state := &repo.PendingState{
		BotID:    payload.BotID,
		EventID:  eventID,
		MsgID:    payload.Trigger.MsgID,
		ConvID:   payload.Trigger.Conversation.ConvID,
		ConvType: payload.Trigger.Conversation.ConvType,
		GroupID:  payload.Trigger.Conversation.GroupID,
		ExpireAt: time.Now().Add(s.pendingTTL).Unix(),
	}
	if err := s.cache.SetPending(ctx, state, s.pendingTTL); err != nil {
		log.Warn().Err(err).Str("event_id", eventID).Msg("failed to store pending state")
	}

	return eventID, nil
}

// HandleCallback 处理外部系统的异步回调
func (s *AsyncWebhookService) HandleCallback(ctx context.Context, req *model.CallbackRequest) (*repo.PendingState, error) {
	state, err := s.cache.GetPending(ctx, req.EventID)
	if err != nil {
		return nil, fmt.Errorf("pending not found or expired: %w", err)
	}

	// 校验通过后删除 pending 记录（防止重复回调）
	if err := s.cache.DeletePending(ctx, req.EventID); err != nil {
		log.Warn().Err(err).Str("event_id", req.EventID).Msg("failed to delete pending state")
	}

	return state, nil
}

// CleanupExpired 定期清理过期的 pending 记录
func (s *AsyncWebhookService) CleanupExpired(ctx context.Context) {
	keys, err := s.cache.ScanExpiredPending(ctx, repo.BotPendingKey, 100)
	if err != nil {
		log.Err(err).Msg("scan expired pending")
		return
	}

	now := time.Now()
	for _, key := range keys {
		ttl, err := s.cache.GetTTL(ctx, key)
		if err != nil {
			continue
		}
		if ttl <= 0 {
			log.Warn().Str("key", key).Msg("expired pending cleaned up")
			_ = s.cache.DeletePending(ctx, key)
		}
	}
}
