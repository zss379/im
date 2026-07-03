package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/sse"
)

// SSEWebhookService 处理 SSE 流式 Webhook 模式
// 时序: bot-svc → POST → 外部系统 → 返回 SSE URL → bot-svc 连接 → 逐 token 转发到 message-svc
type SSEWebhookService struct {
	cfg           *config.SSEConfig
	httpClient    *http.Client
	pool          *sse.Pool
	idleTimeout   time.Duration
	maxStreamDur  time.Duration
	msgClient     *MessageClient
}

func NewSSEWebhookService(cfg *config.SSEConfig, pool *sse.Pool, msgClient *MessageClient) *SSEWebhookService {
	idleTO, _ := time.ParseDuration(cfg.IdleTimeout)
	maxDur, _ := time.ParseDuration(cfg.MaxStreamDuration)

	return &SSEWebhookService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		pool:         pool,
		idleTimeout:  idleTO,
		maxStreamDur: maxDur,
		msgClient:    msgClient,
	}
}

// Invoke 调用外部系统 Webhook，获取 SSE URL 后连接流式接收
func (s *SSEWebhookService) Invoke(ctx context.Context, webhookURL string, payload *model.WebhookPayload) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", "message.mention")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	var initResp model.WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return fmt.Errorf("parse initial response: %w", err)
	}

	if initResp.SSEURL == "" {
		return fmt.Errorf("external system did not return sse_url")
	}

	log.Info().Str("sse_url", initResp.SSEURL).Int64("bot_id", payload.BotID).
		Msg("connecting to SSE stream")

	return s.streamConnect(ctx, initResp.SSEURL, initResp.SessionID, payload)
}

func (s *SSEWebhookService) streamConnect(ctx context.Context, sseURL, sessionID string, payload *model.WebhookPayload) error {
	// 检查连接池是否已达上限
	if !s.pool.TryAcquire() {
		return fmt.Errorf("SSE connection pool full (max=%d)", s.cfg.MaxConnections)
	}
	defer s.pool.Release()

	streamCtx, cancel := context.WithTimeout(ctx, s.maxStreamDur)
	defer cancel()

	req, err := http.NewRequestWithContext(streamCtx, http.MethodGet, sseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	return s.readStream(streamCtx, resp.Body, payload)
}

func (s *SSEWebhookService) readStream(ctx context.Context, reader io.Reader, payload *model.WebhookPayload) error {
	scanner := bufio.NewScanner(reader)
	// SSE events may be large — increase scanner buffer
	scanner.Buffer(make([]byte, 64*1024), 256*1024)

	var (
		lastDataTime  = time.Now()
		partialText strings.Builder
	)

	idleTimer := time.AfterFunc(s.idleTimeout, func() {
		log.Warn().Int64("bot_id", payload.BotID).Msg("SSE idle timeout, closing stream")
	})

	for scanner.Scan() {
		idleTimer.Reset(s.idleTimeout)

		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			partialText.WriteString(data)
			lastDataTime = time.Now()

			// 转发 token 到 message-svc
			if s.msgClient != nil {
				s.msgClient.SendToken(payload.BotID, payload.Trigger.MsgID, data)
			}
		} else if strings.HasPrefix(line, "event: ") {
			event := strings.TrimPrefix(line, "event: ")
			if event == "done" || event == "error" {
				idleTimer.Stop()
				log.Info().Int64("bot_id", payload.BotID).Str("event", event).
					Msg("SSE stream ended")
				return nil
			}
		}

		select {
		case <-ctx.Done():
			idleTimer.Stop()
			return ctx.Err()
		default:
		}
	}

	idleTimer.Stop()
	return scanner.Err()
}
