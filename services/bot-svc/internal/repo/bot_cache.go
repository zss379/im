package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/shulian-paas/im/bot-svc/internal/model"
)

const (
	BotUserIDsKey = "bot:user_ids"           // Set of all bot OpenIM user IDs
	BotConfigKey  = "bot:%d"                 // Hash of bot configuration
	BotPendingKey = "bot:pending:%s"         // String for async callback pending state
)

type BotCache struct {
	rdb redis.UniversalClient
}

func NewBotCache(rdb redis.UniversalClient) *BotCache {
	return &BotCache{rdb: rdb}
}

// ---- bot:user_ids Set (task 1.3/2.2) ----

func (c *BotCache) AddUserID(ctx context.Context, botID int64) error {
	return c.rdb.SAdd(ctx, BotUserIDsKey, botID).Err()
}

func (c *BotCache) RemoveUserID(ctx context.Context, botID int64) error {
	return c.rdb.SRem(ctx, BotUserIDsKey, botID).Err()
}

func (c *BotCache) GetAllUserIDs(ctx context.Context) ([]int64, error) {
	return c.rdb.SMembers(ctx, BotUserIDsKey).Int64Slice()
}

func (c *BotCache) IsBotUser(ctx context.Context, userID int64) (bool, error) {
	return c.rdb.SIsMember(ctx, BotUserIDsKey, userID).Err()
}

// IntersectUserIDs 返回 atUserIDs 中是机器人的用户 ID
func (c *BotCache) IntersectUserIDs(ctx context.Context, atUserIDs []int64) ([]int64, error) {
	if len(atUserIDs) == 0 {
		return nil, nil
	}
	args := make([]any, len(atUserIDs))
	for i, id := range atUserIDs {
		args[i] = id
	}
	return c.rdb.SInter(ctx, BotUserIDsKey, args...).Int64Slice()
}

// ---- bot:{bot_id} Hash (task 1.3/2.3) ----

func (c *BotCache) SetConfig(ctx context.Context, bot *model.Bot) error {
	key := fmt.Sprintf(BotConfigKey, bot.BotID)
	vals := map[string]any{
		"bot_id":         bot.BotID,
		"tenant_id":      bot.TenantID,
		"bot_type":       bot.BotType,
		"bot_name":       bot.BotName,
		"webhook_url":    nullToEmpty(bot.WebhookURL),
		"api_key":        nullToEmpty(bot.APIKey),
		"openim_user_id": nullInt64ToZero(bot.OpenIMUserID),
		"response_mode":  bot.ResponseMode,
		"callback_url":   nullToEmpty(bot.CallbackURL),
		"ip_whitelist":   nullToEmpty(bot.IPWhitelist),
		"status":         bot.Status,
	}
	return c.rdb.HSet(ctx, key, vals).Err()
}

func (c *BotCache) GetConfig(ctx context.Context, botID int64) (*model.Bot, error) {
	key := fmt.Sprintf(BotConfigKey, botID)
	data, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, redis.Nil
	}
	return hashToBot(data), nil
}

func (c *BotCache) DeleteConfig(ctx context.Context, botID int64) error {
	key := fmt.Sprintf(BotConfigKey, botID)
	return c.rdb.Del(ctx, key).Err()
}

// ---- bot:pending:{event_id} (task 5.2) ----

type PendingState struct {
	BotID     int64  `json:"bot_id"`
	EventID   string `json:"event_id"`
	MsgID     string `json:"msg_id"`
	ExpireAt  int64  `json:"expire_at"`
}

func (c *BotCache) SetPending(ctx context.Context, state *PendingState, ttl time.Duration) error {
	key := fmt.Sprintf(BotPendingKey, state.EventID)
	data, _ := json.Marshal(state)
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *BotCache) GetPending(ctx context.Context, eventID string) (*PendingState, error) {
	key := fmt.Sprintf(BotPendingKey, eventID)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var state PendingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *BotCache) DeletePending(ctx context.Context, eventID string) error {
	key := fmt.Sprintf(BotPendingKey, eventID)
	return c.rdb.Del(ctx, key).Err()
}

// ScanExpiredPending 扫描过期 pending 记录
func (c *BotCache) ScanExpiredPending(ctx context.Context, match string, count int) ([]string, error) {
	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error
		batch, cursor, err = c.rdb.Scan(ctx, cursor, match, int64(count)).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (c *BotCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

// ---- helpers ----

func nullToEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullInt64ToZero(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func hashToBot(data map[string]string) *model.Bot {
	bot := &model.Bot{}
	fmt.Sscanf(data["bot_id"], "%d", &bot.BotID)
	fmt.Sscanf(data["tenant_id"], "%d", &bot.TenantID)
	var bt int8
	fmt.Sscanf(data["response_mode"], "%d", &bt)
	bot.ResponseMode = model.ResponseMode(bt)
	fmt.Sscanf(data["status"], "%d", &bot.Status)
	bot.BotName = data["bot_name"]
	s := data["webhook_url"]
	if s != "" {
		bot.WebhookURL = &s
	}
	s = data["api_key"]
	if s != "" {
		bot.APIKey = &s
	}
	s = data["callback_url"]
	if s != "" {
		bot.CallbackURL = &s
	}
	return bot
}
