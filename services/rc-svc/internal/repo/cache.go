package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	keyRateLimitUser = "rate:user:%d"
	keyRateLimitBot  = "rate:bot:%d"
)

type Cache struct {
	rdb redis.UniversalClient
}

func NewCache(rdb redis.UniversalClient) *Cache {
	return &Cache{rdb: rdb}
}

// CheckRateLimit checks if an action is within rate limits using a sliding window.
// Returns (allowed, remaining_count, error).
func (c *Cache) CheckRateLimit(ctx context.Context, key string, maxCount int, windowSecs int) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowStart := now - int64(windowSecs)*1000

	pipe := c.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	count := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, time.Duration(windowSecs)*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	currentCount, err := count.Result()
	if err != nil {
		return false, 0, err
	}

	remaining := maxCount - int(currentCount)
	if remaining < 0 {
		remaining = 0
	}
	return int(currentCount) <= maxCount, remaining, nil
}

// RateLimitKeyUser returns the Redis key for user rate limiting.
func RateLimitKeyUser(userID int64) string {
	return fmt.Sprintf(keyRateLimitUser, userID)
}

// RateLimitKeyBot returns the Redis key for bot rate limiting.
func RateLimitKeyBot(botID int64) string {
	return fmt.Sprintf(keyRateLimitBot, botID)
}

// RateLimitKey returns the appropriate Redis key based on target type.
func RateLimitKey(targetType int8, targetID int64) string {
	if targetType == 2 {
		return RateLimitKeyBot(targetID)
	}
	return RateLimitKeyUser(targetID)
}
