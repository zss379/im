package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/group-svc/internal/model"
	"github.com/shulian-paas/im/group-svc/internal/repo"
)

var (
	ErrGroupNotFound    = errors.New("group not found")
	ErrMemberNotFound   = errors.New("member not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrCapacityExceeded = errors.New("capacity limit exceeded")
	ErrAlreadyMember    = errors.New("already a member")
	ErrDuplicateRequest = errors.New("duplicate join request")
	ErrOwnerTransferReq = errors.New("transfer ownership before exiting")
)

type GroupService struct {
	mysqlRepo *repo.MySQLRepo
	cache     *repo.Cache
}

func NewGroupService(mysqlRepo *repo.MySQLRepo, cache *repo.Cache) *GroupService {
	return &GroupService{mysqlRepo: mysqlRepo, cache: cache}
}

// ---- Group CRUD ----

func (s *GroupService) CreateGroup(ctx context.Context, ownerID int64, req *model.CreateGroupReq) (*model.CreateGroupResp, error) {
	// Check capacity: user created groups
	cnt, err := s.mysqlRepo.CountCreatedGroups(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if cnt >= int64(model.MaxCreatedGroupsPerUser) {
		return nil, fmt.Errorf("%w: max %d created groups", ErrCapacityExceeded, model.MaxCreatedGroupsPerUser)
	}

	group := &model.Group{
		TenantID:   req.TenantID,
		Name:       req.Name,
		OwnerID:    ownerID,
		VerifyMode: model.VerifyOpen,
		Status:     model.GroupStatusActive,
	}
	if err := s.mysqlRepo.CreateGroup(ctx, group); err != nil {
		return nil, err
	}

	// Add owner as member with role=owner
	members := []model.GroupMember{
		{GroupID: group.GroupID, UserID: ownerID, Role: model.RoleOwner},
	}
	for _, uid := range req.MemberIDs {
		members = append(members, model.GroupMember{GroupID: group.GroupID, UserID: uid, Role: model.RoleMember})
	}
	if err := s.mysqlRepo.BatchAddMembers(ctx, members); err != nil {
		return nil, err
	}

	return &model.CreateGroupResp{GroupID: group.GroupID, Name: group.Name}, nil
}

func (s *GroupService) DismissGroup(ctx context.Context, groupID, opUserID int64, confirm bool) error {
	if !confirm {
		return errors.New("dismiss requires confirmation")
	}
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.OwnerID != opUserID {
		return ErrPermissionDenied
	}
	return s.mysqlRepo.DeleteGroup(ctx, groupID)
}

func (s *GroupService) TransferOwner(ctx context.Context, groupID, opUserID, newOwnerID int64) error {
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.OwnerID != opUserID {
		return ErrPermissionDenied
	}
	// Demote current owner to member, promote new owner
	if err := s.mysqlRepo.UpdateMemberRole(ctx, groupID, opUserID, model.RoleMember); err != nil {
		return err
	}
	if err := s.mysqlRepo.UpdateMemberRole(ctx, groupID, newOwnerID, model.RoleOwner); err != nil {
		return err
	}
	// Update group owner_id
	return s.mysqlRepo.UpdateGroup(ctx, groupID, map[string]interface{}{"owner_id": newOwnerID})
}

func (s *GroupService) ExitGroup(ctx context.Context, groupID, userID int64) error {
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.OwnerID == userID {
		return ErrOwnerTransferReq
	}
	return s.mysqlRepo.BatchRemoveMembers(ctx, groupID, []int64{userID})
}

func (s *GroupService) UpdateGroup(ctx context.Context, groupID, opUserID int64, req *model.UpdateGroupReq) error {
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	// Check permission: owner or admin
	member, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if member == nil || member.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Notice != nil {
		if len(*req.Notice) > model.MaxNoticeLength {
			return fmt.Errorf("%w: notice exceeds %d chars", ErrCapacityExceeded, model.MaxNoticeLength)
		}
		updates["notice"] = *req.Notice
	}
	if len(updates) == 0 {
		return nil
	}
	return s.mysqlRepo.UpdateGroup(ctx, groupID, updates)
}

func (s *GroupService) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	return s.mysqlRepo.GetGroup(ctx, groupID)
}

func (s *GroupService) ListUserGroups(ctx context.Context, tenantID, userID int64, page, pageSize int) (*model.GroupListResp, error) {
	list, total, err := s.mysqlRepo.ListGroupsByUser(ctx, tenantID, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &model.GroupListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *GroupService) SearchGroups(ctx context.Context, tenantID int64, keyword string, page, pageSize int) ([]model.Group, int64, error) {
	return s.mysqlRepo.SearchGroups(ctx, tenantID, keyword, page, pageSize)
}

// ---- Member Management ----

func (s *GroupService) ListMembers(ctx context.Context, groupID int64, page, pageSize int, role *int8) (*model.MemberListResp, error) {
	list, total, err := s.mysqlRepo.ListMembers(ctx, groupID, page, pageSize, role)
	if err != nil {
		return nil, err
	}
	return &model.MemberListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *GroupService) BatchAddMembers(ctx context.Context, groupID, opUserID int64, req *model.BatchMemberReq) error {
	// Check permission: owner or admin only
	member, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if member == nil || member.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}

	if len(req.UserIDs) > model.MaxBatchMembers {
		return fmt.Errorf("%w: max %d per batch", ErrCapacityExceeded, model.MaxBatchMembers)
	}

	// Check member capacity
	cnt, err := s.mysqlRepo.CountMembers(ctx, groupID)
	if err != nil {
		return err
	}
	if cnt+int64(len(req.UserIDs)) > int64(model.MaxMembersPerGroup) {
		return fmt.Errorf("%w: max %d members per group", ErrCapacityExceeded, model.MaxMembersPerGroup)
	}

	// Check user's total groups
	for _, uid := range req.UserIDs {
		userCnt, err := s.mysqlRepo.CountUserGroups(ctx, uid)
		if err != nil {
			return err
		}
		if userCnt >= int64(model.MaxGroupsPerUser) {
			return fmt.Errorf("%w: user %d has max %d groups", ErrCapacityExceeded, uid, model.MaxGroupsPerUser)
		}
	}

	members := make([]model.GroupMember, 0, len(req.UserIDs))
	for _, uid := range req.UserIDs {
		existing, _ := s.mysqlRepo.GetMember(ctx, groupID, uid)
		if existing != nil {
			continue // skip duplicates
		}
		members = append(members, model.GroupMember{
			GroupID: groupID,
			UserID:  uid,
			Role:    model.RoleMember,
		})
	}
	return s.mysqlRepo.BatchAddMembers(ctx, members)
}

func (s *GroupService) BatchRemoveMembers(ctx context.Context, groupID, opUserID int64, req *model.BatchMemberReq) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}

	for _, uid := range req.UserIDs {
		m, err := s.mysqlRepo.GetMember(ctx, groupID, uid)
		if err != nil {
			return err
		}
		if m == nil {
			continue
		}
		// Admin cannot remove owner or other admins
		if opMember.Role == model.RoleAdmin && m.Role >= model.RoleAdmin {
			return ErrPermissionDenied
		}
	}
	return s.mysqlRepo.BatchRemoveMembers(ctx, groupID, req.UserIDs)
}

func (s *GroupService) SetMemberRole(ctx context.Context, groupID, opUserID, targetUserID int64, req *model.SetRoleReq) error {
	// Only owner can change roles
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.OwnerID != opUserID {
		return ErrPermissionDenied
	}

	if req.Role == model.RoleAdmin {
		// Check admin ratio: max 10% of members can be admins
		adminCnt, err := s.mysqlRepo.CountAdmins(ctx, groupID)
		if err != nil {
			return err
		}
		total, err := s.mysqlRepo.CountMembers(ctx, groupID)
		if err != nil {
			return err
		}
		if (adminCnt+1)*model.AdminRatioLimit > total {
			return fmt.Errorf("%w: max %d%% admins", ErrCapacityExceeded, model.AdminRatioLimit)
		}
	}
	return s.mysqlRepo.UpdateMemberRole(ctx, groupID, targetUserID, req.Role)
}

func (s *GroupService) SearchMembers(ctx context.Context, groupID int64, keyword string, page, pageSize int) (*model.MemberListResp, error) {
	list, total, err := s.mysqlRepo.SearchMembers(ctx, groupID, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &model.MemberListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

// ---- Mute ----

func (s *GroupService) MuteMember(ctx context.Context, groupID, opUserID int64, req *model.MuteMemberReq) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	// Admin cannot mute owner
	target, err := s.mysqlRepo.GetMember(ctx, groupID, req.UserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrMemberNotFound
	}
	if opMember.Role == model.RoleAdmin && target.Role >= model.RoleAdmin {
		return ErrPermissionDenied
	}

	mutedUntil := time.Now().Add(time.Duration(req.Duration) * time.Second)
	// Set in Redis for fast check
	s.cache.SetMemberMute(ctx, groupID, req.UserID, req.Duration)
	// Persist in DB
	return s.mysqlRepo.UpdateMemberMute(ctx, groupID, req.UserID, &mutedUntil)
}

func (s *GroupService) UnmuteMember(ctx context.Context, groupID, opUserID, targetUserID int64) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	s.cache.RemoveMemberMute(ctx, groupID, targetUserID)
	return s.mysqlRepo.UpdateMemberMute(ctx, groupID, targetUserID, nil)
}

func (s *GroupService) GlobalMute(ctx context.Context, groupID, opUserID int64, req *model.GlobalMuteReq) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	return s.cache.SetGlobalMute(ctx, groupID, req.Duration)
}

func (s *GroupService) RemoveGlobalMute(ctx context.Context, groupID, opUserID int64) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	return s.cache.RemoveGlobalMute(ctx, groupID)
}

func (s *GroupService) CheckMute(ctx context.Context, groupID, userID int64) (*model.MuteCheckResp, error) {
	// Check global mute first
	globalMuted, remaining, err := s.cache.IsGloballyMuted(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if globalMuted {
		// Admins bypass global mute
		member, err := s.mysqlRepo.GetMember(ctx, groupID, userID)
		if err != nil {
			return nil, err
		}
		if member != nil && member.Role >= model.RoleAdmin {
			return &model.MuteCheckResp{Muted: false}, nil
		}
		return &model.MuteCheckResp{Muted: true, RemainingSec: remaining}, nil
	}

	// Check individual mute
	muted, remaining, err := s.cache.IsMemberMuted(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if muted {
		return &model.MuteCheckResp{Muted: true, RemainingSec: remaining}, nil
	}
	return &model.MuteCheckResp{Muted: false}, nil
}

// ---- Join Request ----

func (s *GroupService) SetJoinConfig(ctx context.Context, groupID, opUserID int64, req *model.JoinConfigReq) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	return s.mysqlRepo.UpdateGroup(ctx, groupID, map[string]interface{}{"verify_mode": req.VerifyMode})
}

func (s *GroupService) RequestJoin(ctx context.Context, groupID, userID int64) error {
	group, err := s.mysqlRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}

	// Check if already a member
	existing, _ := s.mysqlRepo.GetMember(ctx, groupID, userID)
	if existing != nil {
		return ErrAlreadyMember
	}

	// Check if open mode
	if group.VerifyMode == model.VerifyOpen {
		members := []model.GroupMember{
			{GroupID: groupID, UserID: userID, Role: model.RoleMember},
		}
		return s.mysqlRepo.BatchAddMembers(ctx, members)
	}

	// Check for duplicate request
	pending, _ := s.mysqlRepo.GetPendingJoinRequest(ctx, groupID, userID)
	if pending != nil {
		return ErrDuplicateRequest
	}

	return s.mysqlRepo.CreateJoinRequest(ctx, &model.JoinRequest{
		GroupID: groupID,
		UserID:  userID,
		Status:  model.JoinPending,
	})
}

func (s *GroupService) ApproveJoinRequest(ctx context.Context, groupID, requestID, opUserID int64) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}

	req, err := s.mysqlRepo.GetJoinRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil || req.Status != model.JoinPending {
		return errors.New("invalid or already processed request")
	}

	// Check capacity
	cnt, err := s.mysqlRepo.CountMembers(ctx, groupID)
	if err != nil {
		return err
	}
	if cnt >= int64(model.MaxMembersPerGroup) {
		return ErrCapacityExceeded
	}

	if err := s.mysqlRepo.UpdateJoinRequest(ctx, requestID, model.JoinApproved); err != nil {
		return err
	}
	return s.mysqlRepo.BatchAddMembers(ctx, []model.GroupMember{
		{GroupID: groupID, UserID: req.UserID, Role: model.RoleMember},
	})
}

func (s *GroupService) RejectJoinRequest(ctx context.Context, groupID, requestID, opUserID int64) error {
	opMember, err := s.mysqlRepo.GetMember(ctx, groupID, opUserID)
	if err != nil {
		return err
	}
	if opMember == nil || opMember.Role < model.RoleAdmin {
		return ErrPermissionDenied
	}
	return s.mysqlRepo.UpdateJoinRequest(ctx, requestID, model.JoinRejected)
}

func (s *GroupService) ListJoinRequests(ctx context.Context, groupID int64, page, pageSize int) (*model.JoinRequestListResp, error) {
	list, total, err := s.mysqlRepo.ListJoinRequests(ctx, groupID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &model.JoinRequestListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}
