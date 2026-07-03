package repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupTestRedis(t *testing.T) (*BotCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewBotCache(rdb)
	return cache, mr
}

func TestBotCache_AddUserID(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	err := cache.AddUserID(ctx, 4001)
	if err != nil {
		t.Fatalf("AddUserID failed: %v", err)
	}

	members := mr.SMembers(BotUserIDsKey)
	if len(members) != 1 || members[0] != "4001" {
		t.Errorf("expected [4001], got %v", members)
	}
}

func TestBotCache_RemoveUserID(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	_ = cache.AddUserID(ctx, 4001)
	_ = cache.AddUserID(ctx, 4002)

	err := cache.RemoveUserID(ctx, 4001)
	if err != nil {
		t.Fatalf("RemoveUserID failed: %v", err)
	}

	members := mr.SMembers(BotUserIDsKey)
	if len(members) != 1 || members[0] != "4002" {
		t.Errorf("expected [4002], got %v", members)
	}
}

func TestBotCache_GetAllUserIDs(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	_ = cache.AddUserID(ctx, 4001)
	_ = cache.AddUserID(ctx, 4002)
	_ = cache.AddUserID(ctx, 4003)

	ids, err := cache.GetAllUserIDs(ctx)
	if err != nil {
		t.Fatalf("GetAllUserIDs failed: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d: %v", len(ids), ids)
	}
}

func TestBotCache_IsBotUser(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	_ = cache.AddUserID(ctx, 4001)

	yes, err := cache.IsBotUser(ctx, 4001)
	if err != nil || !yes {
		t.Errorf("expected true for bot user 4001, got %v err=%v", yes, err)
	}

	no, err := cache.IsBotUser(ctx, 9999)
	if err != nil || no {
		t.Errorf("expected false for non-bot 9999, got %v err=%v", no, err)
	}
}

func TestBotCache_IntersectUserIDs(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	_ = cache.AddUserID(ctx, 4001)
	_ = cache.AddUserID(ctx, 4002)

	tests := []struct {
		name     string
		input    []int64
		expected []int64
	}{
		{"match one bot", []int64{10001, 4001}, []int64{4001}},
		{"match multiple bots", []int64{4001, 4002, 10001}, []int64{4001, 4002}},
		{"no bot", []int64{10001, 10002}, nil},
		{"empty input", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.IntersectUserIDs(ctx, tt.input)
			if err != nil {
				t.Fatalf("IntersectUserIDs failed: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBotCache_SetAndGetConfig(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	bot := &Bot{
		BotID:        4001,
		TenantID:     1,
		BotType:      BotTypeCustom,
		BotName:      "告警机器人",
		WebhookURL:   strPtr("https://hooks.example.com/alert"),
		APIKey:       strPtr("sk_test_key"),
		ResponseMode: ResponseModeSync,
		Status:       1,
	}

	err := cache.SetConfig(ctx, bot)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	got, err := cache.GetConfig(ctx, 4001)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if got.BotID != 4001 || got.BotName != "告警机器人" {
		t.Errorf("GetConfig returned wrong data: %+v", got)
	}
	if got.WebhookURL == nil || *got.WebhookURL != "https://hooks.example.com/alert" {
		t.Errorf("webhook_url mismatch: %v", got.WebhookURL)
	}
}

func TestBotCache_DeleteConfig(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	bot := &Bot{BotID: 4001, BotName: "test", Status: 1}
	_ = cache.SetConfig(ctx, bot)

	err := cache.DeleteConfig(ctx, 4001)
	if err != nil {
		t.Fatalf("DeleteConfig failed: %v", err)
	}

	_, err = cache.GetConfig(ctx, 4001)
	if err != redis.Nil {
		t.Errorf("expected redis.Nil after delete, got %v", err)
	}
}

func TestBotCache_PendingState(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	state := &PendingState{
		BotID:   4001,
		EventID: "evt_test_001",
		MsgID:   "msg_test_001",
		ExpireAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	ttl := 30 * time.Minute

	err := cache.SetPending(ctx, state, ttl)
	if err != nil {
		t.Fatalf("SetPending failed: %v", err)
	}

	got, err := cache.GetPending(ctx, "evt_test_001")
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}

	if got.BotID != 4001 || got.EventID != "evt_test_001" {
		t.Errorf("pending state mismatch: %+v", got)
	}

	err = cache.DeletePending(ctx, "evt_test_001")
	if err != nil {
		t.Fatalf("DeletePending failed: %v", err)
	}

	_, err = cache.GetPending(ctx, "evt_test_001")
	if err != redis.Nil {
		t.Errorf("expected redis.Nil after delete, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
