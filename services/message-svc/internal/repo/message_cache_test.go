package repo

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupTestCache(t *testing.T) (*MessageCache, *miniredis.Miniredis, func()) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewMessageCache(rdb)
	return cache, mr, func() {
		rdb.Close()
		mr.Close()
	}
}

func TestTryDedup(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	clientMsgID := "client-msg-001"

	// First call: should return false (not a duplicate)
	dup, err := cache.TryDedup(ctx, clientMsgID)
	if err != nil {
		t.Fatalf("TryDedup failed: %v", err)
	}
	if dup {
		t.Error("first TryDedup should return false")
	}

	// Second call: should return true (duplicate)
	dup, err = cache.TryDedup(ctx, clientMsgID)
	if err != nil {
		t.Fatalf("TryDedup failed: %v", err)
	}
	if !dup {
		t.Error("second TryDedup should return true")
	}
}

func TestReleaseDedup(t *testing.T) {
	cache, mr, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	clientMsgID := "client-to-release"

	// Set the dedup key
	dup, _ := cache.TryDedup(ctx, clientMsgID)
	if dup {
		t.Fatal("unexpected duplicate on first call")
	}

	// Release it
	if err := cache.ReleaseDedup(ctx, clientMsgID); err != nil {
		t.Fatalf("ReleaseDedup failed: %v", err)
	}

	// The key should be gone, TryDedup returns false again
	dup, err := cache.TryDedup(ctx, clientMsgID)
	if err != nil {
		t.Fatalf("TryDedup after release failed: %v", err)
	}
	if dup {
		t.Error("after release, TryDedup should return false")
	}

	// Verify Redis key is gone
	_, err = mr.Get("idempotent:msg:client-to-release")
	if err == nil {
		t.Error("key should be deleted after ReleaseDedup")
	}
}

func TestDedupExpiry(t *testing.T) {
	cache, mr, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	clientMsgID := "expiring-msg"

	cache.TryDedup(ctx, clientMsgID)

	// Fast-forward TTL
	mr.FastForward(2 * time.Hour)

	// After expiry, TryDedup should return false again
	dup, err := cache.TryDedup(ctx, clientMsgID)
	if err != nil {
		t.Fatalf("TryDedup after expiry: %v", err)
	}
	if dup {
		t.Error("after expiry, TryDedup should return false")
	}
}

func TestMarkReadAndIsRead(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	conversationID := int64(50001)
	msgID := "msg_001"

	// Not read yet
	isRead, _, err := cache.IsRead(ctx, conversationID, msgID)
	if err != nil {
		t.Fatalf("IsRead failed: %v", err)
	}
	if isRead {
		t.Error("new message should not be read")
	}

	// Mark as read
	if err := cache.MarkRead(ctx, conversationID, msgID); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	// Now it should be read
	isRead, readAt, err := cache.IsRead(ctx, conversationID, msgID)
	if err != nil {
		t.Fatalf("IsRead after mark failed: %v", err)
	}
	if !isRead {
		t.Error("message should be read after MarkRead")
	}
	if readAt <= 0 {
		t.Error("readAt should be a positive timestamp")
	}
}

func TestIsolatedReadPerConversation(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	msgID := "msg_shared"

	// Mark as read in conversation 1
	cache.MarkRead(ctx, 100, msgID)

	// Should NOT be read in conversation 2
	isRead, _, _ := cache.IsRead(ctx, 200, msgID)
	if isRead {
		t.Error("message read status should be isolated per conversation")
	}

	// Should be read in conversation 1
	isRead, _, _ = cache.IsRead(ctx, 100, msgID)
	if !isRead {
		t.Error("message should be read in conversation 1")
	}
}

func TestGetReadStatuses(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	conversationID := int64(50001)
	msgIDs := []string{"msg_001", "msg_002", "msg_003"}

	// Mark two as read
	cache.MarkRead(ctx, conversationID, "msg_001")
	cache.MarkRead(ctx, conversationID, "msg_003")

	statuses, err := cache.GetReadStatuses(ctx, conversationID, msgIDs)
	if err != nil {
		t.Fatalf("GetReadStatuses failed: %v", err)
	}

	if len(statuses) != 2 {
		t.Errorf("expected 2 read statuses, got %d", len(statuses))
	}

	if _, ok := statuses["msg_001"]; !ok {
		t.Error("msg_001 should be in statuses")
	}
	if _, ok := statuses["msg_003"]; !ok {
		t.Error("msg_003 should be in statuses")
	}
	if _, ok := statuses["msg_002"]; ok {
		t.Error("msg_002 should NOT be in statuses")
	}
}

func TestStoreAndGetSSEToken(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Store token
	if err := cache.StoreSSEToken(ctx, "msg_sse_001", "tok_secret_abc"); err != nil {
		t.Fatalf("StoreSSEToken failed: %v", err)
	}

	// Get token
	token, err := cache.GetSSEToken(ctx, "msg_sse_001")
	if err != nil {
		t.Fatalf("GetSSEToken failed: %v", err)
	}
	if token != "tok_secret_abc" {
		t.Errorf("token: got %s, want tok_secret_abc", token)
	}

	// Non-existent token
	_, err = cache.GetSSEToken(ctx, "msg_nonexistent")
	if err != redis.Nil {
		t.Errorf("expected redis.Nil, got %v", err)
	}
}

func TestGetReadStatusesEmptyInput(t *testing.T) {
	cache, _, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	statuses, err := cache.GetReadStatuses(ctx, 50001, []string{})
	if err != nil {
		t.Fatalf("GetReadStatuses empty input: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected empty result, got %d", len(statuses))
	}
}
