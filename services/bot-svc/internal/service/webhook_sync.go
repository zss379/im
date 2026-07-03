package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
)

// SyncWebhookService 处理同步 Webhook 模式
// 时序: bot-svc → POST → 外部系统 → 等待响应 → 解析 reply → 发回会话
type SyncWebhookService struct {
	cfg        *config.WebhookConfig
	httpClient *http.Client
	retryDelay []time.Duration
}

func NewSyncWebhookService(cfg *config.WebhookConfig) *SyncWebhookService {
	timeout, _ := time.ParseDuration(cfg.SyncTimeout)
	backoffs := make([]time.Duration, len(cfg.RetryBackoff))
	for i, s := range cfg.RetryBackoff {
		d, _ := time.ParseDuration(s)
		backoffs[i] = d
	}

	return &SyncWebhookService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		retryDelay: backoffs,
	}
}

// Invoke 调用外部系统 Webhook，等待同步响应
// 返回的 WebhookResponse 包含 reply 字段，由调用方发回会话
func (s *SyncWebhookService) Invoke(ctx context.Context, webhookURL string, payload *model.WebhookPayload) (*model.WebhookResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= s.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避等待
			delay := s.backoff(attempt - 1)
			log.Debug().Int("attempt", attempt).Dur("delay", delay).Msg("sync webhook retry")
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := s.doPost(ctx, webhookURL, payload)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt+1).Str("url", webhookURL).
			Msg("sync webhook attempt failed")
	}

	return nil, fmt.Errorf("sync webhook failed after %d retries: %w", s.cfg.MaxRetries, lastErr)
}

func (s *SyncWebhookService) doPost(ctx context.Context, url string, payload *model.WebhookPayload) (*model.WebhookResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", "message.mention")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, int64(s.cfg.MaxBodySizeMB)*1024*1024))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d body=%s", resp.StatusCode, string(respBody))
	}

	var whResp model.WebhookResponse
	if err := json.Unmarshal(respBody, &whResp); err != nil {
		return nil, fmt.Errorf("parse response: %w body=%s", err, string(respBody))
	}

	if whResp.Reply == "" {
		return nil, fmt.Errorf("empty reply in webhook response")
	}

	return &whResp, nil
}

func (s *SyncWebhookService) backoff(attempt int) time.Duration {
	if attempt >= len(s.retryDelay) {
		return s.retryDelay[len(s.retryDelay)-1]
	}
	return s.retryDelay[attempt]
}
