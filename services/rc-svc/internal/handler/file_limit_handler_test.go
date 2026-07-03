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

type mockFileLimitService struct {
	listConfigsFn  func(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error)
	createConfigFn func(ctx context.Context, req *model.FileLimitCreateReq) error
	updateConfigFn func(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error
	deleteConfigFn func(ctx context.Context, configID int64, tenantID int64) error
	checkFileFn    func(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error)
}

func (m *mockFileLimitService) ListFileLimits(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error) {
	return m.listConfigsFn(ctx, tenantID)
}

func (m *mockFileLimitService) CreateFileLimit(ctx context.Context, req *model.FileLimitCreateReq) error {
	return m.createConfigFn(ctx, req)
}

func (m *mockFileLimitService) UpdateFileLimit(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error {
	return m.updateConfigFn(ctx, configID, updates, tenantID)
}

func (m *mockFileLimitService) DeleteFileLimit(ctx context.Context, configID int64, tenantID int64) error {
	return m.deleteConfigFn(ctx, configID, tenantID)
}

func (m *mockFileLimitService) CheckFileLimit(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error) {
	return m.checkFileFn(ctx, tenantID, fileType, fileSizeBytes)
}

func setupFileLimitTest(t *testing.T) (*gin.Engine, *mockFileLimitService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mock := &mockFileLimitService{
		listConfigsFn:  func(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error) { return nil, nil },
		createConfigFn: func(ctx context.Context, req *model.FileLimitCreateReq) error { return nil },
		updateConfigFn: func(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error { return nil },
		deleteConfigFn: func(ctx context.Context, configID int64, tenantID int64) error { return nil },
		checkFileFn: func(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error) {
			return &model.FileLimitCheckResp{Passed: true, MaxSizeMB: 10}, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
	})
	api := r.Group("/api/v1")
	h := NewFileLimitHandler(mock)
	h.RegisterRoutes(api)
	return r, mock
}

func TestFileLimitListConfigs(t *testing.T) {
	r, mock := setupFileLimitTest(t)
	mock.listConfigsFn = func(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error) {
		return []model.FileLimitConfig{{ConfigID: 1, FileType: "image", MaxSizeMB: 10}}, nil
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/file-limit/configs", nil)
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

func TestFileLimitCreateConfig(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	body := `{"file_type":"image","max_size_mb":10}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/file-limit/configs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileLimitCreateConfig_BadRequest(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	body := `{"max_size_mb":10}` // missing file_type
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/file-limit/configs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFileLimitUpdateConfig(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	body := `{"max_size_mb":20}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/file-limit/configs/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileLimitDeleteConfig(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/file-limit/configs/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileLimitCheck(t *testing.T) {
	r, mock := setupFileLimitTest(t)
	mock.checkFileFn = func(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error) {
		return &model.FileLimitCheckResp{Passed: false, MaxSizeMB: 10}, nil
	}
	body := `{"file_type":"video","file_size":104857600}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/file-limit/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileLimitCheck_Passed(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	body := `{"file_type":"image","file_size":1024}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/file-limit/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileLimitUpdateConfig_InvalidID(t *testing.T) {
	r, _ := setupFileLimitTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/file-limit/configs/abc", strings.NewReader(`{"max_size_mb":10}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
