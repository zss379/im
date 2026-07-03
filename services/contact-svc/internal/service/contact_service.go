package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shulian-paas/im/contact-svc/internal/metrics"
	"github.com/shulian-paas/im/contact-svc/internal/model"
	"github.com/shulian-paas/im/contact-svc/internal/repo"
)

type ContactService struct {
	repo  *repo.MySQLRepo
	cache *repo.Cache
}

func NewContactService(repo *repo.MySQLRepo, cache *repo.Cache) *ContactService {
	return &ContactService{repo: repo, cache: cache}
}

// ---- Department ----

func buildTree(depts []model.Department, parentID int64) []*model.DeptTreeResp {
	var nodes []*model.DeptTreeResp
	for _, d := range depts {
		if d.ParentID == parentID && d.Status == model.ContactStatusActive {
			node := &model.DeptTreeResp{
				DeptID:    d.DeptID,
				Name:      d.Name,
				ParentID:  d.ParentID,
				SortOrder: d.SortOrder,
				MemberCnt: d.MemberCount,
				Children:  buildTree(depts, d.DeptID),
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (s *ContactService) GetDeptTree(ctx context.Context, tenantID int64) ([]*model.DeptTreeResp, error) {
	depts, err := s.repo.ListDepartments(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	tree := buildTree(depts, 0)
	if tree == nil {
		tree = []*model.DeptTreeResp{}
	}
	return tree, nil
}

func (s *ContactService) GetDeptDetail(ctx context.Context, deptID int64) (*model.DeptDetailResp, error) {
	d, err := s.repo.GetDepartment(ctx, deptID)
	if err != nil {
		return nil, err
	}
	if d == nil || d.Status != model.ContactStatusActive {
		return nil, nil
	}
	return &model.DeptDetailResp{
		DeptID:      d.DeptID,
		Name:        d.Name,
		ParentID:    d.ParentID,
		SortOrder:   d.SortOrder,
		MemberCount: d.MemberCount,
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *ContactService) CreateDept(ctx context.Context, tenantID int64, req *model.CreateDeptReq) (*model.Department, error) {
	if req.ParentID > 0 {
		parent, err := s.repo.GetDepartment(ctx, req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, fmt.Errorf("parent department not found")
		}
	}

	dept := &model.Department{
		TenantID:  tenantID,
		Name:      req.Name,
		ParentID:  req.ParentID,
		SortOrder: req.SortOrder,
	}
	if err := s.repo.CreateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	metrics.DeptCreateTotal.Inc()
	return dept, nil
}

func (s *ContactService) UpdateDept(ctx context.Context, deptID int64, req *model.UpdateDeptReq) error {
	d, err := s.repo.GetDepartment(ctx, deptID)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("department not found")
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateDepartment(ctx, deptID, updates)
}

func (s *ContactService) DeleteDept(ctx context.Context, deptID int64) error {
	d, err := s.repo.GetDepartment(ctx, deptID)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("department not found")
	}

	count, err := s.repo.CountChildren(ctx, deptID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete department with children")
	}

	if err := s.repo.DeleteDepartment(ctx, deptID); err != nil {
		return err
	}
	metrics.DeptDeleteTotal.Inc()
	return nil
}

// ---- Members ----

func (s *ContactService) SearchMembers(ctx context.Context, tenantID int64, req *model.MemberSearchReq) ([]model.MemberSummary, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 50
	}
	return s.repo.SearchProfiles(ctx, tenantID, req.Keyword, req.DeptID, req.Page, req.PageSize)
}

func (s *ContactService) GetDeptMembers(ctx context.Context, deptID int64, req *model.MemberListReq) ([]model.MemberSummary, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 50
	}
	return s.repo.ListMembersByDept(ctx, deptID, req.Page, req.PageSize)
}

func (s *ContactService) GetMemberDetail(ctx context.Context, userID int64, tenantID int64) (*model.MemberDetailResp, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.TenantID != tenantID {
		return nil, nil
	}

	depts, err := s.repo.GetUserDepts(ctx, userID)
	if err != nil {
		return nil, err
	}

	phone := p.Phone
	if len(phone) > 7 {
		phone = phone[:3] + "****" + phone[len(phone)-4:]
	}

	return &model.MemberDetailResp{
		UserID:   p.UserID,
		Name:     p.Name,
		Avatar:   p.Avatar,
		Phone:    phone,
		Position: p.Position,
		Depts:    depts,
	}, nil
}

func (s *ContactService) GetUserDepts(ctx context.Context, userID int64, tenantID int64) (*model.UserDeptResp, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.TenantID != tenantID {
		return nil, fmt.Errorf("member not found")
	}

	depts, err := s.repo.GetUserDepts(ctx, userID)
	if err != nil {
		return nil, err
	}
	if depts == nil {
		depts = []model.DeptBrief{}
	}
	return &model.UserDeptResp{
		UserID: userID,
		Depts:  depts,
	}, nil
}

func (s *ContactService) UpdateMember(ctx context.Context, userID int64, tenantID int64, req *model.UpdateMemberReq) error {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return err
	}
	if p == nil {
		p = &model.ContactProfile{UserID: userID, TenantID: tenantID}
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.NamePy != nil {
		p.NamePy = *req.NamePy
	}
	if req.Avatar != nil {
		p.Avatar = *req.Avatar
	}
	if req.Phone != nil {
		p.Phone = *req.Phone
	}
	if req.Position != nil {
		p.Position = *req.Position
	}

	if err := s.repo.UpsertProfile(ctx, p); err != nil {
		return err
	}
	if req.DeptIDs != nil {
		if err := s.repo.SetUserDepts(ctx, tenantID, userID, req.DeptIDs); err != nil {
			return err
		}
	}
	return nil
}

// ---- Sync ----

func (s *ContactService) Sync(ctx context.Context, tenantID int64, req *model.SyncReq) error {
	depts := make([]model.Department, len(req.Departments))
	for i, sd := range req.Departments {
		depts[i] = model.Department{
			DeptID:    sd.DeptID,
			TenantID:  tenantID,
			Name:      sd.Name,
			ParentID:  sd.ParentID,
			SortOrder: sd.SortOrder,
			Status:    model.ContactStatusActive,
		}
	}
	if err := s.repo.BatchUpsertDepartments(ctx, depts); err != nil {
		return err
	}

	profiles := make([]model.ContactProfile, len(req.Members))
	for i, sm := range req.Members {
		profiles[i] = model.ContactProfile{
			UserID:   sm.UserID,
			TenantID: tenantID,
			Name:     sm.Name,
			NamePy:   sm.NamePy,
			Avatar:   sm.Avatar,
			Phone:    sm.Phone,
			Position: sm.Position,
			Status:   sm.Status,
		}
	}
	if err := s.repo.BatchUpsertProfiles(ctx, profiles); err != nil {
		return err
	}

	for _, sm := range req.Members {
		if len(sm.DeptIDs) > 0 {
			if err := s.repo.SetUserDepts(ctx, tenantID, sm.UserID, sm.DeptIDs); err != nil {
				return err
			}
		}
	}

	metrics.SyncTotal.Inc()
	return nil
}
