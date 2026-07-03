package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis Key 前缀
const (
	KeyDedup    = "idempotent:msg:%s"        // 消息幂等去重 key
	KeyReadHash = "read:%d"                  // 已读回执 Hash key，field=msg_id
	KeySSEToken = "sse:token:%s"             // SSE token key，field=msg_id
)

const defaultSSETTL = 5 * time.Minute

// MessageCache Redis 缓存层
type MessageCache struct {
	rdb redis.UniversalClient
}

func NewMessageCache(rdb redis.UniversalClient) *MessageCache {
	return &MessageCache{rdb: rdb}
}

// TryDedup 尝试幂等去重，返回是否已存在
func (c *MessageCache) TryDedup(ctx context.Context, clientMsgID string) (bool, error) {
	key := fmt.Sprintf(KeyDedup, clientMsgID)
	ok, err := c.rdb.SetNX(ctx, key, "1", time.Hour).Result()
	if err != nil {
		return false, err
	}
	return !ok, nil // true = 已存在（重复）
}

// ReleaseDedup 释放幂等锁（异常回滚时）
func (c *MessageCache) ReleaseDedup(ctx context.Context, clientMsgID string) error {
	return c.rdb.Del(ctx, fmt.Sprintf(KeyDedup, clientMsgID)).Err()
}

// MarkRead 标记会话中某条消息为已读
func (c *MessageCache) MarkRead(ctx context.Context, conversationID int64, msgID string) error {
	key := fmt.Sprintf(KeyReadHash, conversationID)
	return c.rdb.HSet(ctx, key, msgID, time.Now().UnixMilli()).Err()
}

// IsRead 查询某条消息是否已读
func (c *MessageCache) IsRead(ctx context.Context, conversationID int64, msgID string) (bool, int64, error) {
	key := fmt.Sprintf(KeyReadHash, conversationID)
	val, err := c.rdb.HGet(ctx, key, msgID).Int64()
	if err != nil {
		if err == redis.Nil {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, val, nil
}

// StoreSSEToken 存储 SSE token
func (c *MessageCache) StoreSSEToken(ctx context.Context, msgID, token string) error {
	key := fmt.Sprintf(KeySSEToken, msgID)
	return c.rdb.Set(ctx, key, token, defaultSSETTL).Err()
}

// GetSSEToken 获取 SSE token
func (c *MessageCache) GetSSEToken(ctx context.Context, msgID string) (string, error) {
	key := fmt.Sprintf(KeySSEToken, msgID)
	return c.rdb.Get(ctx, key).Result()
}

// GetReadStatuses 批量查询已读状态
func (c *MessageCache) GetReadStatuses(ctx context.Context, conversationID int64, msgIDs []string) (map[string]int64, error) {
	key := fmt.Sprintf(KeyReadHash, conversationID)
	vals, err := c.rdb.HMGet(ctx, key, msgIDs...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(msgIDs))
	for i, val := range vals {
		if val == nil {
			continue
		}
		if t, ok := val.(int64); ok {
			result[msgIDs[i]] = t
		}
	}
	return result, nil
}
