package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shulian-paas/im/session-svc/internal/metrics"
	"github.com/shulian-paas/im/session-svc/internal/model"
	"github.com/shulian-paas/im/session-svc/internal/repo"
)

type SessionService struct {
	repo  *repo.MySQLRepo
	cache *repo.Cache
}

func NewSessionService(repo *repo.MySQLRepo, cache *repo.Cache) *SessionService {
	return &SessionService{repo: repo, cache: cache}
}

func conversationID(sessionType int8, targetID int64, userID int64) string {
	switch sessionType {
	case model.SessionTypeSingle:
		if userID < targetID {
			return fmt.Sprintf("s_%d_%d", userID, targetID)
		}
		return fmt.Sprintf("s_%d_%d", targetID, userID)
	case model.SessionTypeGroup:
		return fmt.Sprintf("g_%d", targetID)
	case model.SessionTypeBot:
		return fmt.Sprintf("b_%d", targetID)
	default:
		return fmt.Sprintf("t_%d_%d", sessionType, targetID)
	}
}

func (s *SessionService) CreateSession(ctx context.Context, tenantID, userID int64, req *model.CreateSessionReq) (*model.Session, error) {
	convID := conversationID(req.Type, req.TargetID, userID)

	existing, err := s.repo.GetSessionByConv(ctx, userID, convID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.IsDeleted {
			existing.IsDeleted = false
			if err := s.repo.UpdateSession(ctx, existing.SessionID, map[string]interface{}{
				"is_deleted": false, "last_message_at": time.Now(),
			}); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	count, err := s.repo.CountUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= model.MaxSessionsPerUser {
		return nil, fmt.Errorf("max sessions per user reached: %d", model.MaxSessionsPerUser)
	}

	now := time.Now()
	session := &model.Session{
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: convID,
		Type:           req.Type,
		TargetID:       req.TargetID,
		LastMessageAt:  &now,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	metrics.SessionCreateTotal.Inc()
	return session, nil
}

func (s *SessionService) ListSessions(ctx context.Context, tenantID, userID int64, req *model.SessionListReq) ([]model.SessionSummary, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 50
	}

	sessions, total, err := s.repo.ListSessions(ctx, tenantID, userID, req.Type, req.Unread, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}

	summaries := make([]model.SessionSummary, len(sessions))
	for i, sess := range sessions {
		summaries[i] = toSummary(sess)
	}
	return summaries, total, nil
}

func (s *SessionService) GetSession(ctx context.Context, sessionID, userID int64) (*model.SessionDetailResp, error) {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if sess == nil || sess.IsDeleted {
		return nil, nil
	}
	return &model.SessionDetailResp{
		SessionID:      sess.SessionID,
		ConversationID: sess.ConversationID,
		Type:           sess.Type,
		TargetID:       sess.TargetID,
		IsPinned:       sess.IsPinned,
		IsMuted:        sess.IsMuted,
		UnreadCount:    sess.UnreadCount,
		LastMessage:    sess.LastMessage,
		LastMsgType:    sess.LastMsgType,
		LastSenderID:   sess.LastSenderID,
		LastMessageAt:  formatTime(sess.LastMessageAt),
		CreatedAt:      sess.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      sess.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, sessionID, userID int64) error {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if sess == nil || sess.IsDeleted {
		return fmt.Errorf("session not found")
	}

	if err := s.repo.DeleteSession(ctx, sessionID, userID); err != nil {
		return err
	}
	metrics.SessionDeleteTotal.Inc()
	return nil
}

func (s *SessionService) PinSession(ctx context.Context, sessionID, userID int64, pinned bool) error {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if sess == nil || sess.IsDeleted {
		return fmt.Errorf("session not found")
	}

	updates := map[string]interface{}{"is_pinned": pinned}
	if pinned {
		count, err := s.repo.CountPinned(ctx, userID)
		if err != nil {
			return err
		}
		if count >= model.MaxPinnedSessions {
			return fmt.Errorf("max pinned sessions reached: %d", model.MaxPinnedSessions)
		}
		now := time.Now()
		updates["pinned_at"] = &now
	} else {
		updates["pinned_at"] = nil
	}

	if err := s.repo.UpdateSession(ctx, sessionID, updates); err != nil {
		return err
	}

	action := "pin"
	if !pinned {
		action = "unpin"
	}
	metrics.PinOpsTotal.WithLabelValues(action).Inc()
	return nil
}

func (s *SessionService) MuteSession(ctx context.Context, sessionID, userID int64, muted bool) error {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if sess == nil || sess.IsDeleted {
		return fmt.Errorf("session not found")
	}

	if err := s.repo.UpdateSession(ctx, sessionID, map[string]interface{}{"is_muted": muted}); err != nil {
		return err
	}

	action := "mute"
	if !muted {
		action = "unmute"
	}
	metrics.MuteOpsTotal.WithLabelValues(action).Inc()
	return nil
}

func (s *SessionService) MarkRead(ctx context.Context, sessionID, userID int64) error {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if sess == nil || sess.IsDeleted {
		return fmt.Errorf("session not found")
	}

	if sess.UnreadCount == 0 {
		return nil
	}

	if err := s.repo.UpdateSession(ctx, sessionID, map[string]interface{}{"unread_count": 0}); err != nil {
		return err
	}
	_ = s.cache.SetUnreadCount(ctx, userID, sessionID, 0)
	metrics.UnreadOpsTotal.Inc()
	return nil
}

func (s *SessionService) MarkUnread(ctx context.Context, sessionID, userID int64) error {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if sess == nil || sess.IsDeleted {
		return fmt.Errorf("session not found")
	}

	if err := s.repo.UpdateSession(ctx, sessionID, map[string]interface{}{"unread_count": 1}); err != nil {
		return err
	}
	_ = s.cache.SetUnreadCount(ctx, userID, sessionID, 1)
	metrics.UnreadOpsTotal.Inc()
	return nil
}

func (s *SessionService) BatchUpdateUnread(ctx context.Context, userID int64, req *model.BatchUnreadReq) error {
	if len(req.SessionIDs) == 0 {
		return nil
	}

	if err := s.repo.BatchUpdateUnread(ctx, userID, req.SessionIDs, req.UnreadCount); err != nil {
		return err
	}
	for _, sid := range req.SessionIDs {
		_ = s.cache.SetUnreadCount(ctx, userID, sid, req.UnreadCount)
	}
	metrics.UnreadOpsTotal.Inc()
	return nil
}

func toSummary(sess model.Session) model.SessionSummary {
	return model.SessionSummary{
		SessionID:      sess.SessionID,
		ConversationID: sess.ConversationID,
		Type:           sess.Type,
		TargetID:       sess.TargetID,
		IsPinned:       sess.IsPinned,
		IsMuted:        sess.IsMuted,
		UnreadCount:    sess.UnreadCount,
		LastMessage:    sess.LastMessage,
		LastMsgType:    sess.LastMsgType,
		LastSenderID:   sess.LastSenderID,
		LastMessageAt:  formatTime(sess.LastMessageAt),
		CreatedAt:      sess.CreatedAt.Format(time.RFC3339),
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
