package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/shulian-paas/im/audit-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.AdminOpLog{},
		&model.MsgAuditLog{},
	)
}

// ---- Admin Op Log ----

func (r *MySQLRepo) CreateAdminOpLog(ctx context.Context, log *model.AdminOpLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *MySQLRepo) GetAdminOpLog(ctx context.Context, logID int64) (*model.AdminOpLog, error) {
	var l model.AdminOpLog
	err := r.db.WithContext(ctx).First(&l, logID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &l, err
}

func (r *MySQLRepo) ListAdminOpLogs(ctx context.Context, tenantID int64, operatorID int64, opType string, startTime, endTime string, page, pageSize int) ([]model.AdminOpLog, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AdminOpLog{}).Where("tenant_id = ?", tenantID)
	if operatorID > 0 {
		query = query.Where("operator_id = ?", operatorID)
	}
	if opType != "" {
		query = query.Where("op_type = ?", opType)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.AdminOpLog
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if operatorID > 0 {
		q = q.Where("operator_id = ?", operatorID)
	}
	if opType != "" {
		q = q.Where("op_type = ?", opType)
	}
	if startTime != "" {
		q = q.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		q = q.Where("created_at <= ?", endTime)
	}
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// ---- Msg Audit Log ----

func (r *MySQLRepo) CreateMsgAuditLog(ctx context.Context, log *model.MsgAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *MySQLRepo) BatchCreateMsgAuditLogs(ctx context.Context, logs []model.MsgAuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

func (r *MySQLRepo) GetMsgAuditLog(ctx context.Context, logID int64) (*model.MsgAuditLog, error) {
	var l model.MsgAuditLog
	err := r.db.WithContext(ctx).First(&l, logID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &l, err
}

func (r *MySQLRepo) ListMsgAuditLogs(ctx context.Context, tenantID int64, req *model.MsgAuditLogListReq, page, pageSize int) ([]model.MsgAuditLog, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.MsgAuditLog{}).Where("tenant_id = ?", tenantID)
	query = applyMsgLogFilters(query, req)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.MsgAuditLog
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	q = applyMsgLogFilters(q, req)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func applyMsgLogFilters(q *gorm.DB, req *model.MsgAuditLogListReq) *gorm.DB {
	if req.SenderID > 0 {
		q = q.Where("sender_id = ?", req.SenderID)
	}
	if req.SessionID != "" {
		q = q.Where("session_id = ?", req.SessionID)
	}
	if req.MsgType != nil {
		q = q.Where("msg_type = ?", *req.MsgType)
	}
	if req.HasSensitive != nil {
		q = q.Where("has_sensitive = ?", *req.HasSensitive)
	}
	if req.Keyword != "" {
		q = q.Where("content LIKE ?", "%"+req.Keyword+"%")
	}
	if req.StartTime != "" {
		q = q.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		q = q.Where("created_at <= ?", req.EndTime)
	}
	return q
}

// ---- Retention cleanup ----

func (r *MySQLRepo) DeleteOldAdminLogs(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&model.AdminOpLog{}).Error
}

func (r *MySQLRepo) DeleteOldMsgAuditLogs(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&model.MsgAuditLog{}).Error
}

func (r *MySQLRepo) CountAdminLogs(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AdminOpLog{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

func (r *MySQLRepo) CountMsgLogs(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.MsgAuditLog{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

// ---- Health / Stats ----

type Stats struct {
	AdminLogCount  int64 `json:"admin_log_count"`
	MsgLogCount    int64 `json:"msg_log_count"`
}

func (r *MySQLRepo) GetStats(ctx context.Context) (*Stats, error) {
	var s Stats
	r.db.WithContext(ctx).Model(&model.AdminOpLog{}).Select("COUNT(*)").Scan(&s.AdminLogCount)
	r.db.WithContext(ctx).Model(&model.MsgAuditLog{}).Select("COUNT(*)").Scan(&s.MsgLogCount)
	return &s, nil
}

// ---- page helper ----

func ParsePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return page, pageSize
}

// ensure fmt is used
var _ = fmt.Sprintf
