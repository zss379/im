package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
)

func newSyncConfig() *config.WebhookConfig {
	return &config.WebhookConfig{
		SyncTimeout:   "2s",
		MaxRetries:    2,
		RetryBackoff:  []string{"10ms", "50ms"},
		MaxBodySizeMB: 1,
	}
}

func TestSyncWebhook_Success(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"reply": "CPU 45%", "reply_type": "text"}`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{
		Event: "message.mention",
		BotID: 4001,
		Trigger: model.TriggerContext{
			Text: "查服务器状态",
			Sender: model.SenderInfo{UserID: 10001, UserName: "张三"},
		},
		Response: model.ResponseModeDef{Type: "sync", TimeoutMs: 2000},
	}

	resp, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if resp.Reply != "CPU 45%" {
		t.Errorf("expected 'CPU 45%', got '%s'", resp.Reply)
	}
	if resp.ReplyType != "text" {
		t.Errorf("expected 'text', got '%s'", resp.ReplyType)
	}
}

func TestSyncWebhook_Timeout(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // exceed 2s timeout
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"reply": "too late"}`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	_, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSyncWebhook_RetryThenFail(t *testing.T) {
	attempts := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	_, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}

	expected := newSyncConfig().MaxRetries + 1 // initial + retries
	if attempts != expected {
		t.Errorf("expected %d attempts, got %d", expected, attempts)
	}
}

func TestSyncWebhook_RetryThenSuccess(t *testing.T) {
	attempts := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"reply": "success after retry"}`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	resp, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if resp.Reply != "success after retry" {
		t.Errorf("unexpected reply: %s", resp.Reply)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestSyncWebhook_EmptyReply(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	_, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err == nil {
		t.Fatal("expected error for empty reply")
	}
}

func TestSyncWebhook_MalformedResponse(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer svr.Close()

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	_, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestSyncWebhook_ContextCancelled(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	svc := NewSyncWebhookService(newSyncConfig())
	payload := &model.WebhookPayload{Event: "message.mention", BotID: 4001}

	_, err := svc.Invoke(ctx, svr.URL, payload)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
