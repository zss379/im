package repo

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/session-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(&model.Session{})
}

func (r *MySQLRepo) CreateSession(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *MySQLRepo) GetSession(ctx context.Context, sessionID, userID int64) (*model.Session, error) {
	var s model.Session
	err := r.db.WithContext(ctx).Where("session_id = ? AND user_id = ?", sessionID, userID).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}

func (r *MySQLRepo) GetSessionByConv(ctx context.Context, userID int64, conversationID string) (*model.Session, error) {
	var s model.Session
	err := r.db.WithContext(ctx).Where("user_id = ? AND conversation_id = ?", userID, conversationID).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}

func (r *MySQLRepo) FindByConversation(ctx context.Context, conversationID string) ([]model.Session, error) {
	var sessions []model.Session
	err := r.db.WithContext(ctx).Where("conversation_id = ? AND is_deleted = ?", conversationID, false).Find(&sessions).Error
	return sessions, err
}

func (r *MySQLRepo) UpdateSession(ctx context.Context, sessionID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).Where("session_id = ?", sessionID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteSession(ctx context.Context, sessionID, userID int64) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Update("is_deleted", true).Error
}

func (r *MySQLRepo) ListSessions(ctx context.Context, tenantID, userID int64, filterType *int8, unreadOnly *bool, page, pageSize int) ([]model.Session, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("tenant_id = ? AND user_id = ? AND is_deleted = ?", tenantID, userID, false)
	if filterType != nil {
		query = query.Where("type = ?", *filterType)
	}
	if unreadOnly != nil && *unreadOnly {
		query = query.Where("unread_count > 0")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var sessions []model.Session
	q := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND is_deleted = ?", tenantID, userID, false)
	if filterType != nil {
		q = q.Where("type = ?", *filterType)
	}
	if unreadOnly != nil && *unreadOnly {
		q = q.Where("unread_count > 0")
	}
	err := q.Order("is_pinned DESC, pinned_at DESC, last_message_at DESC, created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&sessions).Error
	return sessions, total, err
}

func (r *MySQLRepo) CountPinned(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND is_pinned = ? AND is_deleted = ?", userID, true, false).
		Count(&count).Error
	return count, err
}

func (r *MySQLRepo) CountUserSessions(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *MySQLRepo) BatchUpdateUnread(ctx context.Context, userID int64, sessionIDs []int64, unreadCount int) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND session_id IN ?", userID, sessionIDs).
		Update("unread_count", unreadCount).Error
}

// ---- Cache (Redis) ----

type Cache struct {
	rdb redis.UniversalClient
}

func NewCache(rdb redis.UniversalClient) *Cache {
	return &Cache{rdb: rdb}
}

const unreadPrefix = "session:unread:"

func (c *Cache) SetUnreadCount(ctx context.Context, userID, sessionID int64, count int) error {
	key := unreadKey(userID, sessionID)
	return c.rdb.Set(ctx, key, count, time.Hour*24).Err()
}

func (c *Cache) GetUnreadCounts(ctx context.Context, userID int64, sessionIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	for _, sid := range sessionIDs {
		key := unreadKey(userID, sid)
		val, err := c.rdb.Get(ctx, key).Int()
		if err == nil {
			result[sid] = val
		}
	}
	return result, nil
}

func unreadKey(userID, sessionID int64) string {
	return unreadPrefix + itoa64(userID) + ":" + itoa64(sessionID)
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
