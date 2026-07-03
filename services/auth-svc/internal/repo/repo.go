package repo

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/auth-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.User{},
		&model.UserStatus{},
	)
}

// ---- User ----

func (r *MySQLRepo) CreateUser(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *MySQLRepo) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).First(&u, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &u, err
}

func (r *MySQLRepo) GetUserByAccount(ctx context.Context, tenantID int64, account string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND account = ?", tenantID, account).First(&u).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &u, err
}

func (r *MySQLRepo) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *MySQLRepo) BatchGetUsers(ctx context.Context, userIDs []int64) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&users).Error
	return users, err
}

// ---- UserStatus ----

func (r *MySQLRepo) UpsertStatus(ctx context.Context, s *model.UserStatus) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *MySQLRepo) GetStatus(ctx context.Context, userID int64) (*model.UserStatus, error) {
	var s model.UserStatus
	err := r.db.WithContext(ctx).First(&s, userID).Error
	if err == gorm.ErrRecordNotFound {
		return &model.UserStatus{UserID: userID, Status: model.OnlineStatusOffline}, nil
	}
	return &s, err
}

// ---- Cache (Redis) ----

const (
	statusPrefix    = "user:status:"
	loginAttemptKey = "login:attempt:"
)

type Cache struct {
	rdb redis.UniversalClient
}

func NewCache(rdb redis.UniversalClient) *Cache {
	return &Cache{rdb: rdb}
}

func (c *Cache) SetUserStatus(ctx context.Context, userID int64, status int8) error {
	key := statusPrefix + itoa64(userID)
	return c.rdb.Set(ctx, key, status, time.Hour*2).Err()
}

func (c *Cache) GetUserStatus(ctx context.Context, userID int64) (int8, error) {
	key := statusPrefix + itoa64(userID)
	val, err := c.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return model.OnlineStatusOffline, nil
	}
	if err != nil {
		return model.OnlineStatusOffline, err
	}
	return int8(val), nil
}

func (c *Cache) BatchGetStatus(ctx context.Context, userIDs []int64) (map[int64]int8, error) {
	result := make(map[int64]int8)
	for _, uid := range userIDs {
		key := statusPrefix + itoa64(uid)
		val, err := c.rdb.Get(ctx, key).Int()
		if err == nil {
			result[uid] = int8(val)
		} else {
			result[uid] = model.OnlineStatusOffline
		}
	}
	return result, nil
}

// Login attempt tracking
func (c *Cache) IncrementLoginAttempt(ctx context.Context, key string) (int, error) {
	redisKey := loginAttemptKey + key
	val, err := c.rdb.Incr(ctx, redisKey).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		c.rdb.Expire(ctx, redisKey, time.Duration(model.LoginLockMinutes)*time.Minute)
	}
	return int(val), nil
}

func (c *Cache) ResetLoginAttempts(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, loginAttemptKey+key).Err()
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string('0'+byte(n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
