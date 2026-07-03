package repo

import (
	"context"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/contact-svc/internal/model"
)

type MySQLRepo struct {
	db *gorm.DB
}

func NewMySQLRepo(db *gorm.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.Department{},
		&model.ContactProfile{},
		&model.UserDept{},
	)
}

// ---- Department ----

func (r *MySQLRepo) CreateDepartment(ctx context.Context, d *model.Department) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *MySQLRepo) UpdateDepartment(ctx context.Context, deptID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Department{}).Where("dept_id = ?", deptID).Updates(updates).Error
}

func (r *MySQLRepo) DeleteDepartment(ctx context.Context, deptID int64) error {
	return r.db.WithContext(ctx).Model(&model.Department{}).Where("dept_id = ?", deptID).Update("status", 0).Error
}

func (r *MySQLRepo) GetDepartment(ctx context.Context, deptID int64) (*model.Department, error) {
	var d model.Department
	err := r.db.WithContext(ctx).First(&d, deptID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &d, err
}

func (r *MySQLRepo) ListDepartments(ctx context.Context, tenantID int64) ([]model.Department, error) {
	var depts []model.Department
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND status = ?", tenantID, model.ContactStatusActive).
		Order("sort_order ASC, dept_id ASC").Find(&depts).Error
	return depts, err
}

func (r *MySQLRepo) CountChildren(ctx context.Context, parentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Department{}).Where("parent_id = ? AND status = ?", parentID, model.ContactStatusActive).Count(&count).Error
	return count, err
}

// ---- Contact Profile ----

func (r *MySQLRepo) UpsertProfile(ctx context.Context, p *model.ContactProfile) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *MySQLRepo) GetProfile(ctx context.Context, userID int64) (*model.ContactProfile, error) {
	var p model.ContactProfile
	err := r.db.WithContext(ctx).First(&p, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &p, err
}

func (r *MySQLRepo) UpdateProfile(ctx context.Context, userID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.ContactProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *MySQLRepo) SearchProfiles(ctx context.Context, tenantID int64, keyword string, deptID int64, page, pageSize int) ([]model.MemberSummary, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Table("contact_profile").
		Joins("LEFT JOIN contact_user_dept ON contact_profile.user_id = contact_user_dept.user_id AND contact_user_dept.is_primary = ?", true).
		Joins("LEFT JOIN contact_department ON contact_user_dept.dept_id = contact_department.dept_id").
		Where("contact_profile.tenant_id = ? AND contact_profile.status = ?", tenantID, model.ContactStatusActive)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(contact_profile.name LIKE ? OR contact_profile.name_py LIKE ?)", like, like)
	}
	if deptID > 0 {
		query = query.Where("contact_user_dept.dept_id = ?", deptID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []model.MemberSummary
	q := r.db.WithContext(ctx).Table("contact_profile").
		Select("contact_profile.user_id, contact_profile.name, contact_profile.avatar, contact_profile.position, contact_department.name as dept_name").
		Joins("LEFT JOIN contact_user_dept ON contact_profile.user_id = contact_user_dept.user_id AND contact_user_dept.is_primary = ?", true).
		Joins("LEFT JOIN contact_department ON contact_user_dept.dept_id = contact_department.dept_id").
		Where("contact_profile.tenant_id = ? AND contact_profile.status = ?", tenantID, model.ContactStatusActive)

	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(contact_profile.name LIKE ? OR contact_profile.name_py LIKE ?)", like, like)
	}
	if deptID > 0 {
		q = q.Where("contact_user_dept.dept_id = ?", deptID)
	}

	err := q.Order("contact_profile.user_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&results).Error
	if err != nil {
		return nil, 0, err
	}
	if len(results) == 0 {
		return []model.MemberSummary{}, total, nil
	}
	return results, total, nil
}

func (r *MySQLRepo) ListMembersByDept(ctx context.Context, deptID int64, page, pageSize int) ([]model.MemberSummary, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.UserDept{}).Where("dept_id = ?", deptID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []model.MemberSummary
	err := r.db.WithContext(ctx).Table("contact_user_dept").
		Select("contact_profile.user_id, contact_profile.name, contact_profile.avatar, contact_profile.position, contact_department.name as dept_name").
		Joins("JOIN contact_profile ON contact_user_dept.user_id = contact_profile.user_id").
		Joins("JOIN contact_department ON contact_user_dept.dept_id = contact_department.dept_id").
		Where("contact_user_dept.dept_id = ? AND contact_profile.status = ?", deptID, model.ContactStatusActive).
		Order("contact_profile.user_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&results).Error
	if err != nil {
		return nil, 0, err
	}
	if len(results) == 0 {
		return []model.MemberSummary{}, total, nil
	}
	return results, total, nil
}

// ---- UserDept ----

func (r *MySQLRepo) SetUserDepts(ctx context.Context, tenantID, userID int64, deptIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserDept{}).Error; err != nil {
			return err
		}
		for i, did := range deptIDs {
			ud := &model.UserDept{
				TenantID:  tenantID,
				UserID:    userID,
				DeptID:    did,
				IsPrimary: i == 0,
			}
			if err := tx.Create(ud).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLRepo) GetUserDepts(ctx context.Context, userID int64) ([]model.DeptBrief, error) {
	var depts []model.DeptBrief
	err := r.db.WithContext(ctx).Table("contact_user_dept").
		Select("contact_department.dept_id, contact_department.name").
		Joins("JOIN contact_department ON contact_user_dept.dept_id = contact_department.dept_id").
		Where("contact_user_dept.user_id = ?", userID).
		Order("contact_user_dept.is_primary DESC").
		Scan(&depts).Error
	return depts, err
}

// ---- Sync ----

func (r *MySQLRepo) BatchUpsertProfiles(ctx context.Context, profiles []model.ContactProfile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range profiles {
			if err := tx.Save(&profiles[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLRepo) BatchUpsertDepartments(ctx context.Context, depts []model.Department) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range depts {
			if err := tx.Save(&depts[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ---- Cache (Redis) ----

type Cache struct {
	rdb redis.UniversalClient
}

func NewCache(rdb redis.UniversalClient) *Cache {
	return &Cache{rdb: rdb}
}
