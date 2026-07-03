package repo

import (
	"context"
	"fmt"

	"github.com/shulian-paas/im/rc-svc/internal/model"
	"gorm.io/gorm"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

// ---- Sensitive Words ----

func (r *MySQLRepo) ListSensitiveWords(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if err := query.Model(&model.SensitiveWord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var words []model.SensitiveWord
	if err := query.Order("word_id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&words).Error; err != nil {
		return nil, 0, err
	}
	return words, total, nil
}

func (r *MySQLRepo) GetAllActiveWords(ctx context.Context, tenantID int64) ([]model.SensitiveWord, error) {
	var words []model.SensitiveWord
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND status = 1", tenantID).Find(&words).Error
	return words, err
}

func (r *MySQLRepo) CreateSensitiveWord(ctx context.Context, word *model.SensitiveWord) error {
	return r.db.WithContext(ctx).Create(word).Error
}

func (r *MySQLRepo) BatchCreateSensitiveWords(ctx context.Context, words []model.SensitiveWord) error {
	if len(words) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(words, 100).Error
}

func (r *MySQLRepo) UpdateSensitiveWord(ctx context.Context, wordID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.SensitiveWord{}).Where("word_id = ?", wordID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteSensitiveWord(ctx context.Context, wordID int64) error {
	return r.db.WithContext(ctx).Delete(&model.SensitiveWord{}, wordID).Error
}

func (r *MySQLRepo) BatchDeleteSensitiveWords(ctx context.Context, wordIDs []int64) error {
	return r.db.WithContext(ctx).Where("word_id IN ?", wordIDs).Delete(&model.SensitiveWord{}).Error
}

// ---- Frequency Control Rules ----

func (r *MySQLRepo) ListRateLimitRules(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error) {
	var rules []model.FrequencyControlRule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND status = 1", tenantID).Find(&rules).Error
	return rules, err
}

func (r *MySQLRepo) GetRateLimitRule(ctx context.Context, tenantID int64, targetType int8) (*model.FrequencyControlRule, error) {
	var rule model.FrequencyControlRule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND target_type = ? AND status = 1", tenantID, targetType).First(&rule).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &rule, err
}

func (r *MySQLRepo) CreateRateLimitRule(ctx context.Context, rule *model.FrequencyControlRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *MySQLRepo) UpdateRateLimitRule(ctx context.Context, ruleID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.FrequencyControlRule{}).Where("rule_id = ?", ruleID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteRateLimitRule(ctx context.Context, ruleID int64) error {
	return r.db.WithContext(ctx).Delete(&model.FrequencyControlRule{}, ruleID).Error
}

// ---- File Upload Limits ----

func (r *MySQLRepo) ListFileLimits(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error) {
	var limits []model.FileLimitConfig
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND status = 1", tenantID).Find(&limits).Error
	return limits, err
}

func (r *MySQLRepo) GetFileLimit(ctx context.Context, tenantID int64, fileType string) (*model.FileLimitConfig, error) {
	var limit model.FileLimitConfig
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND file_type = ? AND status = 1", tenantID, fileType).First(&limit).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &limit, err
}

func (r *MySQLRepo) CreateFileLimit(ctx context.Context, limit *model.FileLimitConfig) error {
	return r.db.WithContext(ctx).Create(limit).Error
}

func (r *MySQLRepo) UpdateFileLimit(ctx context.Context, configID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.FileLimitConfig{}).Where("config_id = ?", configID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteFileLimit(ctx context.Context, configID int64) error {
	return r.db.WithContext(ctx).Delete(&model.FileLimitConfig{}, configID).Error
}

// ---- AutoMigrate ----

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.SensitiveWord{},
		&model.FrequencyControlRule{},
		&model.FileLimitConfig{},
	)
}

// ---- Default seed data ----

func (r *MySQLRepo) SeedDefaults(ctx context.Context) error {
	var count int64
	r.db.WithContext(ctx).Model(&model.FrequencyControlRule{}).Count(&count)
	if count > 0 {
		return nil // already seeded
	}

	defaults := []model.FrequencyControlRule{
		{TenantID: 0, TargetType: model.TargetTypeUser, MaxCount: 5, TimeWindowSeconds: 1, Action: 1, Status: 1},
		{TenantID: 0, TargetType: model.TargetTypeBot, MaxCount: 10, TimeWindowSeconds: 1, Action: 1, Status: 1},
	}
	for i := range defaults {
		if err := r.db.WithContext(ctx).Create(&defaults[i]).Error; err != nil {
			return fmt.Errorf("seed rule: %w", err)
		}
	}

	fileDefaults := []model.FileLimitConfig{
		{TenantID: 0, FileType: "image", MaxSizeMB: 10, AllowedExtensions: "jpg,jpeg,png,gif,webp", Status: 1},
		{TenantID: 0, FileType: "video", MaxSizeMB: 100, AllowedExtensions: "mp4,mov,avi,mkv", Status: 1},
		{TenantID: 0, FileType: "document", MaxSizeMB: 50, AllowedExtensions: "pdf,doc,docx,xls,xlsx,ppt,pptx,txt,zip", Status: 1},
	}
	for i := range fileDefaults {
		if err := r.db.WithContext(ctx).Create(&fileDefaults[i]).Error; err != nil {
			return fmt.Errorf("seed file limit: %w", err)
		}
	}
	return nil
}
