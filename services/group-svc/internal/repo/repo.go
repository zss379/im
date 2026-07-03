package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/group-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.Group{},
		&model.GroupMember{},
		&model.JoinRequest{},
	)
}

// ---- Group ----

func (r *MySQLRepo) CreateGroup(ctx context.Context, group *model.Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *MySQLRepo) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	var g model.Group
	err := r.db.WithContext(ctx).Where("group_id = ? AND status = ?", groupID, model.GroupStatusActive).First(&g).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &g, err
}

func (r *MySQLRepo) UpdateGroup(ctx context.Context, groupID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Group{}).Where("group_id = ?", groupID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteGroup(ctx context.Context, groupID int64) error {
	return r.db.WithContext(ctx).Model(&model.Group{}).Where("group_id = ?", groupID).Update("status", model.GroupStatusDismissed).Error
}

func (r *MySQLRepo) ListGroupsByUser(ctx context.Context, tenantID, userID int64, page, pageSize int) ([]model.GroupSummary, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.GroupMember{}).
		Joins("JOIN group_info ON group_member.group_id = group_info.group_id").
		Where("group_member.user_id = ? AND group_info.tenant_id = ? AND group_info.status = ?", userID, tenantID, model.GroupStatusActive).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var summaries []model.GroupSummary
	err := r.db.WithContext(ctx).
		Table("group_member").
		Select("group_info.group_id, group_info.name, group_info.avatar, group_info.owner_id, group_member.role, (SELECT COUNT(*) FROM group_member gm2 WHERE gm2.group_id = group_info.group_id) as member_count").
		Joins("JOIN group_info ON group_member.group_id = group_info.group_id").
		Where("group_member.user_id = ? AND group_info.tenant_id = ? AND group_info.status = ?", userID, tenantID, model.GroupStatusActive).
		Order("group_member.joined_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&summaries).Error
	return summaries, total, err
}

func (r *MySQLRepo) SearchGroups(ctx context.Context, tenantID int64, keyword string, page, pageSize int) ([]model.Group, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Group{}).Where("tenant_id = ? AND status = ? AND name LIKE ?", tenantID, model.GroupStatusActive, "%"+keyword+"%")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var groups []model.Group
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

func (r *MySQLRepo) CountUserGroups(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *MySQLRepo) CountCreatedGroups(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Group{}).Where("owner_id = ? AND status = ?", userID, model.GroupStatusActive).Count(&count).Error
	return count, err
}

// ---- Member ----

func (r *MySQLRepo) BatchAddMembers(ctx context.Context, members []model.GroupMember) error {
	if len(members) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(members, 100).Error
}

func (r *MySQLRepo) BatchRemoveMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND user_id IN ?", groupID, userIDs).Delete(&model.GroupMember{}).Error
}

func (r *MySQLRepo) ListMembers(ctx context.Context, groupID int64, page, pageSize int, role *int8) ([]model.MemberSummary, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ?", groupID)
	if role != nil {
		query = query.Where("role = ?", *role)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var members []model.MemberSummary
	q := r.db.WithContext(ctx).Model(&model.GroupMember{}).
		Select("user_id", "role", "muted_until", "joined_at").
		Where("group_id = ?", groupID)
	if role != nil {
		q = q.Where("role = ?", *role)
	}
	err := q.Order("role DESC, joined_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&members).Error
	for i := range members {
		members[i].Muted = members[i].MutedUntil != nil && members[i].MutedUntil.After(time.Now())
	}
	return members, total, err
}

func (r *MySQLRepo) GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error) {
	var m model.GroupMember
	err := r.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &m, err
}

func (r *MySQLRepo) CountMembers(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (r *MySQLRepo) CountAdmins(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ? AND role IN ?", groupID, []int8{model.RoleAdmin, model.RoleOwner}).Count(&count).Error
	return count, err
}

func (r *MySQLRepo) UpdateMemberRole(ctx context.Context, groupID, userID int64, role int8) error {
	return r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Update("role", role).Error
}

func (r *MySQLRepo) UpdateMemberMute(ctx context.Context, groupID, userID int64, mutedUntil *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Update("muted_until", mutedUntil).Error
}

func (r *MySQLRepo) SearchMembers(ctx context.Context, groupID int64, keyword string, page, pageSize int) ([]model.MemberSummary, int64, error) {
	return r.ListMembers(ctx, groupID, page, pageSize, nil)
}

// ---- Join Request ----

func (r *MySQLRepo) CreateJoinRequest(ctx context.Context, req *model.JoinRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *MySQLRepo) GetJoinRequest(ctx context.Context, requestID int64) (*model.JoinRequest, error) {
	var jr model.JoinRequest
	err := r.db.WithContext(ctx).First(&jr, requestID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &jr, err
}

func (r *MySQLRepo) GetPendingJoinRequest(ctx context.Context, groupID, userID int64) (*model.JoinRequest, error) {
	var jr model.JoinRequest
	err := r.db.WithContext(ctx).Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, model.JoinPending).First(&jr).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &jr, err
}

func (r *MySQLRepo) UpdateJoinRequest(ctx context.Context, requestID int64, status int8) error {
	return r.db.WithContext(ctx).Model(&model.JoinRequest{}).Where("request_id = ?", requestID).Update("status", status).Error
}

func (r *MySQLRepo) ListJoinRequests(ctx context.Context, groupID int64, page, pageSize int) ([]model.JoinRequestSummary, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.JoinRequest{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var requests []model.JoinRequestSummary
	err := r.db.WithContext(ctx).Model(&model.JoinRequest{}).
		Select("request_id, user_id, status, created_at").
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&requests).Error
	return requests, total, err
}

// ---- Cache (Redis) ----

type Cache struct {
	rdb redis.UniversalClient
}

func NewCache(rdb redis.UniversalClient) *Cache {
	return &Cache{rdb: rdb}
}

func groupMuteKey(groupID int64) string {
	return fmt.Sprintf("group:mute:%d", groupID)
}

func memberMuteKey(groupID, userID int64) string {
	return fmt.Sprintf("group:mute:%d:user:%d", groupID, userID)
}

func (c *Cache) SetMemberMute(ctx context.Context, groupID, userID int64, duration int) error {
	key := memberMuteKey(groupID, userID)
	return c.rdb.Set(ctx, key, 1, time.Duration(duration)*time.Second).Err()
}

func (c *Cache) RemoveMemberMute(ctx context.Context, groupID, userID int64) error {
	key := memberMuteKey(groupID, userID)
	return c.rdb.Del(ctx, key).Err()
}

func (c *Cache) IsMemberMuted(ctx context.Context, groupID, userID int64) (bool, int, error) {
	key := memberMuteKey(groupID, userID)
	ttl, err := c.rdb.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		return false, 0, nil
	}
	return true, int(ttl.Seconds()), nil
}

func (c *Cache) SetGlobalMute(ctx context.Context, groupID int64, duration int) error {
	key := groupMuteKey(groupID)
	return c.rdb.Set(ctx, key, 1, time.Duration(duration)*time.Second).Err()
}

func (c *Cache) RemoveGlobalMute(ctx context.Context, groupID int64) error {
	key := groupMuteKey(groupID)
	return c.rdb.Del(ctx, key).Err()
}

func (c *Cache) IsGloballyMuted(ctx context.Context, groupID int64) (bool, int, error) {
	key := groupMuteKey(groupID)
	ttl, err := c.rdb.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		return false, 0, nil
	}
	return true, int(ttl.Seconds()), nil
}
