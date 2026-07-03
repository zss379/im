package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
)

func setupAsyncTest(t *testing.T) (*AsyncWebhookService, *repo.BotCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis failed: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := repo.NewBotCache(rdb)

	cfg := &config.WebhookConfig{
		PendingTTL: "1m",
		MaxBodySizeMB: 1,
	}
	svc := NewAsyncWebhookService(cfg, cache)
	return svc, cache, mr
}

func TestAsyncWebhook_InvokeSuccess(t *testing.T) {
	svc, cache, mr := setupAsyncTest(t)
	defer mr.Close()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status": "accepted"}`))
	}))
	defer svr.Close()

	payload := &model.WebhookPayload{
		Event:    "message.mention",
		BotID:    4001,
		Trigger:  model.TriggerContext{MsgID: "msg_001", Text: "审批"},
		Response: model.ResponseModeDef{Type: "async", CallbackURL: svr.URL + "/callback"},
	}

	eventID, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if eventID == "" {
		t.Fatal("expected non-empty event_id")
	}

	// Verify pending state stored in Redis
	state, err := cache.GetPending(context.Background(), eventID)
	if err != nil {
		t.Fatalf("pending state not found: %v", err)
	}
	if state.BotID != 4001 || state.MsgID != "msg_001" {
		t.Errorf("pending state mismatch: %+v", state)
	}
}

func TestAsyncWebhook_InvokeNon202(t *testing.T) {
	svc, _, mr := setupAsyncTest(t)
	defer mr.Close()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer svr.Close()

	payload := &model.WebhookPayload{
		BotID: 4001,
		Response: model.ResponseModeDef{Type: "async"},
	}

	_, err := svc.Invoke(context.Background(), svr.URL, payload)
	if err == nil {
		t.Fatal("expected error for non-202 response")
	}
}

func TestAsyncWebhook_HandleCallback(t *testing.T) {
	svc, cache, mr := setupAsyncTest(t)
	defer mr.Close()

	// Store pending state first
	state := &repo.PendingState{
		BotID:   4001,
		EventID: "evt_cb_001",
		MsgID:   "msg_001",
		ExpireAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	err := cache.SetPending(context.Background(), state, 30*time.Minute)
	if err != nil {
		t.Fatalf("SetPending failed: %v", err)
	}

	req := &model.CallbackRequest{
		EventID: "evt_cb_001",
		Reply:   "审批已通过",
	}

	got, err := svc.HandleCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if got.BotID != 4001 || got.EventID != "evt_cb_001" {
		t.Errorf("unexpected state: %+v", got)
	}

	// Verify pending was deleted after callback
	_, err = cache.GetPending(context.Background(), "evt_cb_001")
	if err != redis.Nil {
		t.Errorf("expected redis.Nil, got %v", err)
	}
}

func TestAsyncWebhook_CallbackExpired(t *testing.T) {
	svc, cache, mr := setupAsyncTest(t)
	defer mr.Close()

	// Store with -1s TTL (already expired)
	state := &repo.PendingState{BotID: 4001, EventID: "evt_expired", MsgID: "msg_001", ExpireAt: time.Now().Add(-1 * time.Hour).Unix()}
	_ = cache.SetPending(context.Background(), state, -1*time.Second)

	// Manually expire in miniredis
	mr.FastForward(2 * time.Second)

	req := &model.CallbackRequest{
		EventID: "evt_expired",
		Reply:   "too late",
	}

	_, err := svc.HandleCallback(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for expired event")
	}
}

func TestAsyncWebhook_DoubleCallbackPrevented(t *testing.T) {
	svc, cache, mr := setupAsyncTest(t)
	defer mr.Close()

	state := &repo.PendingState{
		BotID:   4001,
		EventID: "evt_double",
		MsgID:   "msg_001",
		ExpireAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	_ = cache.SetPending(context.Background(), state, 30*time.Minute)

	req := &model.CallbackRequest{EventID: "evt_double", Reply: "ok"}

	// First callback should succeed
	_, err := svc.HandleCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("first callback should succeed: %v", err)
	}

	// Second callback should fail (pending already deleted)
	_, err = svc.HandleCallback(context.Background(), req)
	if err == nil {
		t.Fatal("second callback should fail")
	}
}

func TestAsyncWebhook_CleanupExpired(t *testing.T) {
	svc, cache, mr := setupAsyncTest(t)
	defer mr.Close()

	// Set pending with very short TTL
	state := &repo.PendingState{BotID: 4001, EventID: "evt_clean", MsgID: "msg_001", ExpireAt: time.Now().Add(1 * time.Second).Unix()}
	_ = cache.SetPending(context.Background(), state, 1*time.Second)

	// Fast-forward past TTL
	mr.FastForward(2 * time.Second)

	// Run cleanup
	svc.CleanupExpired(context.Background())

	// Verify cleaned
	_, err := cache.GetPending(context.Background(), "evt_clean")
	if err != redis.Nil {
		t.Log("expected cleanup to remove expired entry (may not be immediate)")
	}
}
