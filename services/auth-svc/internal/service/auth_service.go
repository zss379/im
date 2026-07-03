package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/shulian-paas/im/auth-svc/internal/metrics"
	"github.com/shulian-paas/im/auth-svc/internal/model"
	"github.com/shulian-paas/im/auth-svc/internal/repo"
)

type AuthService struct {
	repo        *repo.MySQLRepo
	cache       *repo.Cache
	jwtSecret   string
	tokenExpiry time.Duration
}

func NewAuthService(repo *repo.MySQLRepo, cache *repo.Cache, jwtSecret string, tokenExpiryHours int) *AuthService {
	return &AuthService{
		repo:        repo,
		cache:       cache,
		jwtSecret:   jwtSecret,
		tokenExpiry: time.Duration(tokenExpiryHours) * time.Hour,
	}
}

// ---- Auth ----

func (s *AuthService) Login(ctx context.Context, req *model.LoginReq) (*model.LoginResp, error) {
	u, err := s.repo.GetUserByAccount(ctx, req.TenantID, req.Account)
	if err != nil {
		return nil, err
	}
	if u == nil {
		metrics.LoginTotal.WithLabelValues("failure").Inc()
		return nil, fmt.Errorf("invalid account or password")
	}

	if u.Status == model.UserStatusDisabled {
		metrics.LoginTotal.WithLabelValues("failure").Inc()
		return nil, fmt.Errorf("account is disabled")
	}

	if u.Status == model.UserStatusLocked {
		metrics.LoginTotal.WithLabelValues("failure").Inc()
		return nil, fmt.Errorf("account is locked, please contact admin")
	}

	attemptKey := fmt.Sprintf("%d:%s", req.TenantID, req.Account)
	attempts, _ := s.cache.IncrementLoginAttempt(ctx, attemptKey)
	if attempts > model.MaxLoginAttempts {
		s.repo.UpdateUser(ctx, u.UserID, map[string]interface{}{"status": model.UserStatusLocked})
		metrics.LoginTotal.WithLabelValues("failure").Inc()
		return nil, fmt.Errorf("too many login attempts, account locked for %d minutes", model.LoginLockMinutes)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		metrics.LoginTotal.WithLabelValues("failure").Inc()
		return nil, fmt.Errorf("invalid account or password")
	}

	s.cache.ResetLoginAttempts(ctx, attemptKey)

	token, err := s.generateToken(u.UserID, u.TenantID)
	if err != nil {
		return nil, err
	}

	if u.Status == model.UserStatusLocked {
		s.repo.UpdateUser(ctx, u.UserID, map[string]interface{}{"status": model.UserStatusActive})
	}

	metrics.LoginTotal.WithLabelValues("success").Inc()
	return &model.LoginResp{
		Token: token,
		User: model.UserBrief{
			UserID:  u.UserID,
			Account: u.Account,
			Name:    u.Name,
			Avatar:  u.Avatar,
			Status:  u.Status,
		},
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, tokenStr string) (*model.LoginResp, error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	userID := int64(claims["user_id"].(float64))
	tenantID := int64(claims["tenant_id"].(float64))

	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || u.Status == model.UserStatusDisabled {
		return nil, fmt.Errorf("user not found or disabled")
	}

	newToken, err := s.generateToken(userID, tenantID)
	if err != nil {
		return nil, err
	}

	metrics.TokenRefreshTotal.Inc()
	return &model.LoginResp{
		Token: newToken,
		User: model.UserBrief{
			UserID: u.UserID,
			Account: u.Account,
			Name:   u.Name,
			Avatar: u.Avatar,
			Status: u.Status,
		},
	}, nil
}

// ---- User Management ----

func (s *AuthService) CreateUser(ctx context.Context, tenantID int64, req *model.CreateUserReq) (*model.User, error) {
	existing, err := s.repo.GetUserByAccount(ctx, tenantID, req.Account)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("account already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		TenantID: tenantID,
		Account:  req.Account,
		Password: string(hash),
		Name:     req.Name,
		Avatar:   req.Avatar,
		Phone:    req.Phone,
		Email:    req.Email,
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	metrics.UserCreateTotal.Inc()
	return u, nil
}

func (s *AuthService) GetUser(ctx context.Context, userID int64, tenantID int64) (*model.UserDetailResp, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || u.TenantID != tenantID {
		return nil, nil
	}
	return &model.UserDetailResp{
		UserID:    u.UserID,
		TenantID:  u.TenantID,
		Account:   u.Account,
		Name:      u.Name,
		Avatar:    u.Avatar,
		Phone:     u.Phone,
		Email:     u.Email,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, userID int64, req *model.UpdateUserReq) error {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateUser(ctx, userID, updates)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int64, req *model.ChangePasswordReq) error {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
		return fmt.Errorf("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdateUser(ctx, userID, map[string]interface{}{"password": string(hash)})
}

func (s *AuthService) BatchGetUsers(ctx context.Context, userIDs []int64, tenantID int64) ([]model.UserBrief, error) {
	if len(userIDs) == 0 {
		return []model.UserBrief{}, nil
	}

	users, err := s.repo.BatchGetUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	briefs := make([]model.UserBrief, 0, len(users))
	for _, u := range users {
		if u.TenantID == tenantID {
			briefs = append(briefs, model.UserBrief{
				UserID:  u.UserID,
				Account: u.Account,
				Name:    u.Name,
				Avatar:  u.Avatar,
				Status:  u.Status,
			})
		}
	}
	return briefs, nil
}

// ---- Status ----

func (s *AuthService) SetStatus(ctx context.Context, userID, tenantID int64, status int8) error {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil || u.TenantID != tenantID {
		return fmt.Errorf("user not found")
	}

	if err := s.repo.UpsertStatus(ctx, &model.UserStatus{
		UserID:   userID,
		TenantID: tenantID,
		Status:   status,
	}); err != nil {
		return err
	}

	s.cache.SetUserStatus(ctx, userID, status)
	metrics.StatusChangeTotal.Inc()
	return nil
}

func (s *AuthService) GetStatus(ctx context.Context, userID int64) (*model.UserStatusBrief, error) {
	cached, err := s.cache.GetUserStatus(ctx, userID)
	if err == nil && cached != model.OnlineStatusOffline {
		return &model.UserStatusBrief{UserID: userID, Status: cached}, nil
	}

	st, err := s.repo.GetStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &model.UserStatusBrief{UserID: userID, Status: st.Status}, nil
}

func (s *AuthService) BatchGetStatus(ctx context.Context, userIDs []int64) (*model.BatchStatusResp, error) {
	cached, err := s.cache.BatchGetStatus(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	statuses := make([]model.UserStatusBrief, len(userIDs))
	for i, uid := range userIDs {
		status := model.OnlineStatusOffline
		if s, ok := cached[uid]; ok {
			status = s
		}
		statuses[i] = model.UserStatusBrief{UserID: uid, Status: status}
	}
	return &model.BatchStatusResp{Statuses: statuses}, nil
}

// ---- Token ----

func (s *AuthService) generateToken(userID, tenantID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"exp":       time.Now().Add(s.tokenExpiry).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
