package repo

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupCacheTest(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewCache(rdb), mr
}

func TestCheckRateLimit_UnderLimit(t *testing.T) {
	cache, mr := setupCacheTest(t)
	ctx := context.Background()
	key := "rate:test:1"

	for i := 0; i < 5; i++ {
		allowed, remaining, err := cache.CheckRateLimit(ctx, key, 10, 60)
		if err != nil {
			t.Fatal(err)
		}
		if !allowed {
			t.Errorf("iteration %d: expected allowed", i)
		}
		if remaining < 0 {
			t.Errorf("iteration %d: expected non-negative remaining", i)
		}
	}
}

func TestCheckRateLimit_OverLimit(t *testing.T) {
	cache, mr := setupCacheTest(t)
	ctx := context.Background()
	key := "rate:test:2"

	// Fill up to limit (max 5)
	for i := 0; i < 5; i++ {
		allowed, _, err := cache.CheckRateLimit(ctx, key, 5, 60)
		if err != nil {
			t.Fatal(err)
		}
		if !allowed {
			t.Errorf("iteration %d: expected allowed", i)
		}
	}

	// Should be over limit now
	allowed, remaining, err := cache.CheckRateLimit(ctx, key, 5, 60)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("expected blocked when over limit")
	}
	if remaining != 0 {
		t.Errorf("expected 0 remaining, got %d", remaining)
	}
}

func TestCheckRateLimit_SlidingWindow(t *testing.T) {
	cache, mr := setupCacheTest(t)
	ctx := context.Background()
	key := "rate:test:3"

	// Add 3 entries with old timestamps (outside window)
	mr.SetTime(time.Now().Add(-120 * time.Second))
	for i := 0; i < 3; i++ {
		cache.CheckRateLimit(ctx, key, 5, 60)
	}

	// Move clock forward — old entries should be cleaned up
	mr.SetTime(time.Now())

	// Should have 0 active entries, allow up to 5
	for i := 0; i < 5; i++ {
		allowed, _, err := cache.CheckRateLimit(ctx, key, 5, 60)
		if err != nil {
			t.Fatal(err)
		}
		if !allowed {
			t.Errorf("iteration %d: expected allowed after window reset", i)
		}
	}

	// Now over limit again
	allowed, _, _ := cache.CheckRateLimit(ctx, key, 5, 60)
	if allowed {
		t.Error("expected blocked after refilling")
	}
}

func TestCheckRateLimit_KeyExpiry(t *testing.T) {
	cache, mr := setupCacheTest(t)
	ctx := context.Background()
	key := "rate:test:expiry"

	cache.CheckRateLimit(ctx, key, 10, 1) // 1 second window
	if !mr.Exists(key) {
		t.Error("expected key to exist")
	}

	mr.FastForward(2 * time.Second)
	if mr.Exists(key) {
		t.Error("expected key to expire after window + TTL")
	}
}

func TestRateLimitKeyUser(t *testing.T) {
	key := RateLimitKeyUser(123)
	if key != "rate:user:123" {
		t.Errorf("unexpected key: %s", key)
	}
}

func TestRateLimitKeyBot(t *testing.T) {
	key := RateLimitKeyBot(456)
	if key != "rate:bot:456" {
		t.Errorf("unexpected key: %s", key)
	}
}

func TestRateLimitKey(t *testing.T) {
	tests := []struct {
		targetType int8
		targetID   int64
		expected   string
	}{
		{1, 100, "rate:user:100"},
		{2, 200, "rate:bot:200"},
		{0, 300, "rate:user:300"},
	}
	for _, tt := range tests {
		key := RateLimitKey(tt.targetType, tt.targetID)
		if key != tt.expected {
			t.Errorf("RateLimitKey(%d, %d) = %s, want %s", tt.targetType, tt.targetID, key, tt.expected)
		}
	}
}

func TestCheckRateLimit_Concurrent(t *testing.T) {
	cache, _ := setupCacheTest(t)
	ctx := context.Background()
	key := "rate:test:concurrent"

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _, err := cache.CheckRateLimit(ctx, key, 100, 60)
			if err != nil {
				t.Log(err)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
