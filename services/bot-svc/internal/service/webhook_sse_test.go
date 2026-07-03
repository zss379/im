package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/sse"
)

func sseConfig() *config.SSEConfig {
	return &config.SSEConfig{
		MaxConnections:    100,
		IdleTimeout:       "1s",
		MaxStreamDuration: "5s",
	}
}

func ssePayload() *model.WebhookPayload {
	return &model.WebhookPayload{
		Event: "message.mention",
		BotID: 4001,
		Trigger: model.TriggerContext{
			MsgID: "msg_sse_001",
			Text:  "讲个故事",
			Sender: model.SenderInfo{UserID: 10001, UserName: "张三"},
		},
		Response: model.ResponseModeDef{Type: "sse"},
	}
}

// sseStreamServer creates an HTTP server that serves SSE events.
// Each string in events is written as a single SSE line (e.g. "data: hello", "event: done").
func sseStreamServer(t *testing.T, events ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for _, e := range events {
			_, _ = w.Write([]byte(e + "\n"))
			flusher.Flush()
			time.Sleep(2 * time.Millisecond)
		}
		// Keep connection open until client disconnects
		<-r.Context().Done()
	}))
}

// TestSSEWebhook_InvokeStreamSuccess 完整端到端流式触发流程
// 外部系统返回 SSE URL → bot-svc 连接 → 逐 token 接收 → event:done 结束
func TestSSEWebhook_InvokeStreamSuccess(t *testing.T) {
	sseStream := sseStreamServer(t,
		"data: 从前有座山",
		"data: 山里有座庙",
		"data: 庙里有个老和尚",
		"event: done",
		"data: 故事结束",
	)
	defer sseStream.Close()

	extSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		_ = json.NewEncoder(w).Encode(model.WebhookResponse{
			SSEURL: sseStream.URL, SessionID: "sess_001", Status: "streaming",
		})
	}))
	defer extSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.Invoke(context.Background(), extSvr.URL, ssePayload())
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if pool.ActiveCount() != 0 {
		t.Errorf("expected 0 active connections after stream ends, got %d", pool.ActiveCount())
	}
}

// TestSSEWebhook_NoSSEURL 外部系统响应缺少 SSE URL
func TestSSEWebhook_NoSSEURL(t *testing.T) {
	extSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(model.WebhookResponse{
			Status: "error", Reply: "not available",
		})
	}))
	defer extSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.Invoke(context.Background(), extSvr.URL, ssePayload())
	if err == nil || !strings.Contains(err.Error(), "sse_url") {
		t.Fatalf("expected error about missing sse_url, got: %v", err)
	}

	if pool.ActiveCount() != 0 {
		t.Errorf("expected 0 active connections on error, got %d", pool.ActiveCount())
	}
}

// TestSSEWebhook_PoolFull SSE 连接池满拒绝新连接
func TestSSEWebhook_PoolFull(t *testing.T) {
	pool := sse.NewPool(2)
	pool.TryAcquire()
	pool.TryAcquire() // pool full

	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.streamConnect(context.Background(), "http://example.com/sse", "sess_001", ssePayload())
	if err == nil || !strings.Contains(err.Error(), "pool full") {
		t.Fatalf("expected pool full error, got: %v", err)
	}
}

// TestSSEWebhook_MaxDurationExceeded SSE 流超过最大持续时间
func TestSSEWebhook_MaxDurationExceeded(t *testing.T) {
	sseStream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 1000; i++ {
			_, _ = w.Write([]byte("data: keepalive\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer sseStream.Close()

	cfg := &config.SSEConfig{
		MaxConnections:    100,
		IdleTimeout:       "10s",
		MaxStreamDuration: "50ms",
	}
	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(cfg, pool, nil)

	err := svc.streamConnect(context.Background(), sseStream.URL, "", ssePayload())
	if err == nil {
		t.Fatal("expected error for max duration exceeded, got nil")
	}

	if pool.ActiveCount() != 0 {
		t.Errorf("expected 0 active connections after stream end, got %d", pool.ActiveCount())
	}
}

// TestSSEWebhook_EventDone event:done 正常结束流
func TestSSEWebhook_EventDone(t *testing.T) {
	// Server that sends data then done, then closes
	doneSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: partial content\n"))
		flusher.Flush()
		time.Sleep(5 * time.Millisecond)
		_, _ = w.Write([]byte("event: done\ndata: \n"))
		flusher.Flush()
	}))
	defer doneSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.streamConnect(context.Background(), doneSvr.URL, "", ssePayload())
	if err != nil {
		t.Fatalf("expected no error for event:done, got: %v", err)
	}
}

// TestSSEWebhook_EventError event:error 正常结束流
func TestSSEWebhook_EventError(t *testing.T) {
	errSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: processing started\n"))
		flusher.Flush()
		time.Sleep(5 * time.Millisecond)
		_, _ = w.Write([]byte("event: error\ndata: upstream timeout\n"))
		flusher.Flush()
	}))
	defer errSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.streamConnect(context.Background(), errSvr.URL, "", ssePayload())
	if err != nil {
		t.Fatalf("expected no error for event:error, got: %v", err)
	}
}

// TestSSEWebhook_ExternalSystemFailure 外部系统返回 500
func TestSSEWebhook_ExternalSystemFailure(t *testing.T) {
	extSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer extSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.Invoke(context.Background(), extSvr.URL, ssePayload())
	if err == nil {
		t.Fatal("expected error for non-200 external system, got nil")
	}
}

// TestSSEWebhook_MalformedExternalResponse 外部系统返回非法 JSON
func TestSSEWebhook_MalformedExternalResponse(t *testing.T) {
	extSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer extSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.Invoke(context.Background(), extSvr.URL, ssePayload())
	if err == nil || !strings.Contains(err.Error(), "parse initial response") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

// TestSSEWebhook_SSEConnectFail SSE 连接不可达
func TestSSEWebhook_SSEConnectFail(t *testing.T) {
	// Start a server that immediately closes (no SSE URL reachable)
	closedSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.WebhookResponse{
			SSEURL: "http://127.0.0.1:1/invalid-sse", // unreachable
		})
	}))
	defer closedSvr.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	err := svc.Invoke(context.Background(), closedSvr.URL, ssePayload())
	if err == nil {
		t.Fatal("expected error for unreachable SSE URL, got nil")
	}
}

// TestSSEWebhook_TokenForwarding 验证 token 被转发到 message-svc
func TestSSEWebhook_TokenForwarding(t *testing.T) {
	var (
		mu       sync.Mutex
		tokens   []string
		endpoint string
	)

	// Mock message-svc SSE endpoint
	msgSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body struct {
			BotID int64  `json:"bot_id"`
			MsgID string `json:"msg_id"`
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		tokens = append(tokens, body.Token)
		w.WriteHeader(http.StatusOK)
	}))
	defer msgSvc.Close()
	endpoint = msgSvc.URL

	sseStream := sseStreamServer(t,
		"data: token-A",
		"data: token-B",
		"data: token-C",
		"event: done",
		"data: done",
	)
	defer sseStream.Close()

	extSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(model.WebhookResponse{
			SSEURL: sseStream.URL, SessionID: "sess_001", Status: "streaming",
		})
	}))
	defer extSvr.Close()

	msgClient := NewMessageClient(endpoint)
	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, msgClient)

	err := svc.Invoke(context.Background(), extSvr.URL, ssePayload())
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens forwarded, got %d: %v", len(tokens), tokens)
	}
	expected := []string{"token-A", "token-B", "token-C"}
	for i, tok := range expected {
		if tokens[i] != tok {
			t.Errorf("token[%d] = %s, want %s", i, tokens[i], tok)
		}
	}
}

// TestSSEWebhook_ContextCancelled 父 context 取消时流终止
func TestSSEWebhook_ContextCancelled(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	sseStream := sseStreamServer(t, "data: hello", "data: world")
	defer sseStream.Close()

	pool := sse.NewPool(100)
	svc := NewSSEWebhookService(sseConfig(), pool, nil)

	// streamConnect creates a new context.WithTimeout(ctx, ...) from the cancelled parent,
	// so the derived context should also be cancelled immediately.
	err := svc.streamConnect(cancelledCtx, sseStream.URL, "", ssePayload())
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
