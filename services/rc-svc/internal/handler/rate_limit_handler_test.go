package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/rc-svc/internal/model"
)

type mockRateLimitService struct {
	listRulesFn    func(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error)
	createRuleFn   func(ctx context.Context, req *model.RateLimitRuleCreateReq) error
	updateRuleFn   func(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error
	deleteRuleFn   func(ctx context.Context, ruleID int64, tenantID int64) error
	checkRateFn    func(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error)
}

func (m *mockRateLimitService) ListRateLimitRules(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error) {
	return m.listRulesFn(ctx, tenantID)
}

func (m *mockRateLimitService) CreateRateLimitRule(ctx context.Context, req *model.RateLimitRuleCreateReq) error {
	return m.createRuleFn(ctx, req)
}

func (m *mockRateLimitService) UpdateRateLimitRule(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error {
	return m.updateRuleFn(ctx, ruleID, updates, tenantID)
}

func (m *mockRateLimitService) DeleteRateLimitRule(ctx context.Context, ruleID int64, tenantID int64) error {
	return m.deleteRuleFn(ctx, ruleID, tenantID)
}

func (m *mockRateLimitService) CheckRateLimit(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error) {
	return m.checkRateFn(ctx, tenantID, targetID, targetType)
}

func setupRateLimitTest(t *testing.T) (*gin.Engine, *mockRateLimitService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mock := &mockRateLimitService{
		listRulesFn:  func(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error) { return nil, nil },
		createRuleFn: func(ctx context.Context, req *model.RateLimitRuleCreateReq) error { return nil },
		updateRuleFn: func(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error { return nil },
		deleteRuleFn: func(ctx context.Context, ruleID int64, tenantID int64) error { return nil },
		checkRateFn: func(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error) {
			return &model.RateLimitCheckResp{Passed: true, Remaining: 10}, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
	})
	api := r.Group("/api/v1")
	h := NewRateLimitHandler(mock)
	h.RegisterRoutes(api)
	return r, mock
}

func TestRateLimitListRules(t *testing.T) {
	r, mock := setupRateLimitTest(t)
	mock.listRulesFn = func(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error) {
		return []model.FrequencyControlRule{{RuleID: 1, TargetType: 1, MaxCount: 5}}, nil
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/rate-limit/rules", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestRateLimitCreateRule(t *testing.T) {
	r, _ := setupRateLimitTest(t)
	body := `{"target_type":1,"max_count":5,"time_window_seconds":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rate-limit/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRateLimitCreateRule_BadRequest(t *testing.T) {
	r, _ := setupRateLimitTest(t)
	body := `{"max_count":5}` // missing target_type
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rate-limit/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRateLimitUpdateRule(t *testing.T) {
	r, _ := setupRateLimitTest(t)
	body := `{"max_count":10}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/rate-limit/rules/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRateLimitDeleteRule(t *testing.T) {
	r, _ := setupRateLimitTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/rate-limit/rules/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRateLimitCheck(t *testing.T) {
	r, mock := setupRateLimitTest(t)
	mock.checkRateFn = func(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error) {
		return &model.RateLimitCheckResp{Passed: false, Remaining: 0}, nil
	}
	body := `{"target_id":1,"target_type":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rate-limit/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int                      `json:"code"`
		Data *model.RateLimitCheckResp `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Passed {
		t.Error("expected not passed")
	}
}

func TestRateLimitCheck_InvalidRuleID(t *testing.T) {
	r, _ := setupRateLimitTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/rate-limit/rules/abc", strings.NewReader(`{"max_count":5}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
