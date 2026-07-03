package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shulian-paas/im/audit-svc/internal/metrics"
	"github.com/shulian-paas/im/audit-svc/internal/model"
	"github.com/shulian-paas/im/audit-svc/internal/repo"
)

type AuditService struct {
	repo *repo.MySQLRepo
}

func NewAuditService(repo *repo.MySQLRepo) *AuditService {
	return &AuditService{repo: repo}
}

const (
	adminOpRetentionDays = 365 * 2 // 2 years
	msgLogRetentionDays  = 365 / 2 // 6 months
)

// ---- Admin Op Log ----

func (s *AuditService) CreateAdminOpLog(ctx context.Context, req *model.CreateAdminOpLogReq) error {
	log := &model.AdminOpLog{
		TenantID:        req.TenantID,
		OperatorID:      req.OperatorID,
		OperatorAccount: req.OperatorAccount,
		OpType:          req.OpType,
		TargetID:        req.TargetID,
		TargetDesc:      req.TargetDesc,
		Detail:          req.Detail,
		Result:          req.Result,
		IP:              req.IP,
	}
	if err := s.repo.CreateAdminOpLog(ctx, log); err != nil {
		return err
	}
	metrics.AdminLogWriteTotal.Inc()
	return nil
}

func (s *AuditService) ListAdminOpLogs(ctx context.Context, tenantID int64, req *model.AdminOpLogListReq) ([]model.AdminOpLog, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		req.PageSize = 50
	}

	logs, total, err := s.repo.ListAdminOpLogs(ctx, tenantID, req.OperatorID, req.OpType, req.StartTime, req.EndTime, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	if logs == nil {
		logs = []model.AdminOpLog{}
	}

	metrics.LogQueryTotal.Inc()
	return logs, total, nil
}

func (s *AuditService) GetAdminOpLog(ctx context.Context, logID int64, tenantID int64) (*model.AdminOpLog, error) {
	l, err := s.repo.GetAdminOpLog(ctx, logID)
	if err != nil {
		return nil, err
	}
	if l == nil || l.TenantID != tenantID {
		return nil, nil
	}
	return l, nil
}

// ---- Msg Audit Log ----

func (s *AuditService) CreateMsgAuditLog(ctx context.Context, req *model.CreateMsgAuditLogReq) error {
	log := &model.MsgAuditLog{
		TenantID:     req.TenantID,
		MsgID:        req.MsgID,
		SenderID:     req.SenderID,
		SessionID:    req.SessionID,
		SessionType:  req.SessionType,
		MsgType:      req.MsgType,
		Content:      req.Content,
		HasSensitive: req.HasSensitive,
		IP:           req.IP,
	}
	if err := s.repo.CreateMsgAuditLog(ctx, log); err != nil {
		return err
	}
	metrics.MsgLogWriteTotal.Inc()
	return nil
}

func (s *AuditService) BatchCreateMsgAuditLogs(ctx context.Context, req *model.BatchCreateMsgLogReq) error {
	if len(req.Logs) == 0 {
		return nil
	}
	logs := make([]model.MsgAuditLog, len(req.Logs))
	for i, l := range req.Logs {
		logs[i] = model.MsgAuditLog{
			TenantID:     l.TenantID,
			MsgID:        l.MsgID,
			SenderID:     l.SenderID,
			SessionID:    l.SessionID,
			SessionType:  l.SessionType,
			MsgType:      l.MsgType,
			Content:      l.Content,
			HasSensitive: l.HasSensitive,
			IP:           l.IP,
		}
	}
	if err := s.repo.BatchCreateMsgAuditLogs(ctx, logs); err != nil {
		return err
	}
	metrics.MsgLogBatchWriteTotal.Inc()
	return nil
}

func (s *AuditService) ListMsgAuditLogs(ctx context.Context, tenantID int64, req *model.MsgAuditLogListReq) ([]model.MsgAuditLog, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 200 {
		req.PageSize = 50
	}

	logs, total, err := s.repo.ListMsgAuditLogs(ctx, tenantID, req, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	if logs == nil {
		logs = []model.MsgAuditLog{}
	}

	metrics.LogQueryTotal.Inc()
	return logs, total, nil
}

func (s *AuditService) GetMsgAuditLog(ctx context.Context, logID int64, tenantID int64) (*model.MsgAuditLog, error) {
	l, err := s.repo.GetMsgAuditLog(ctx, logID)
	if err != nil {
		return nil, err
	}
	if l == nil || l.TenantID != tenantID {
		return nil, nil
	}
	return l, nil
}

// ---- Stats ----

func (s *AuditService) GetStats(ctx context.Context) (*model.Stats, error) {
	return s.repo.GetStats(ctx)
}

// ---- Cleanup ----

func (s *AuditService) Cleanup(ctx context.Context) error {
	adminBefore := time.Now().AddDate(0, 0, -adminOpRetentionDays)
	if err := s.repo.DeleteOldAdminLogs(ctx, adminBefore); err != nil {
		return fmt.Errorf("clean admin logs: %w", err)
	}
	msgBefore := time.Now().AddDate(0, 0, -msgLogRetentionDays)
	if err := s.repo.DeleteOldMsgAuditLogs(ctx, msgBefore); err != nil {
		return fmt.Errorf("clean msg logs: %w", err)
	}
	return nil
}
