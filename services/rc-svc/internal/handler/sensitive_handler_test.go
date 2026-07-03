package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/rc-svc/internal/model"
)

var errTest = errors.New("test error")

type mockSensitiveService struct {
	listWordsFn    func(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error)
	createWordFn   func(ctx context.Context, req *model.SensitiveWordCreateReq) error
	updateWordFn   func(ctx context.Context, wordID int64, updates map[string]interface{}, tenantID int64) error
	deleteWordFn   func(ctx context.Context, wordID int64, tenantID int64) error
	batchImportFn  func(ctx context.Context, req *model.SensitiveWordBatchReq) (int, error)
	checkSensitiveFn func(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error)
	checkChainFn   func(ctx context.Context, tenantID int64, content string, senderID int64, senderType int8, fileType string, fileSize int) *model.CheckChainResp
}

func (m *mockSensitiveService) ListWords(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error) {
	return m.listWordsFn(ctx, tenantID, page, pageSize)
}
func (m *mockSensitiveService) CreateWord(ctx context.Context, req *model.SensitiveWordCreateReq) error {
	return m.createWordFn(ctx, req)
}
func (m *mockSensitiveService) UpdateWord(ctx context.Context, wordID int64, updates map[string]interface{}, tenantID int64) error {
	return m.updateWordFn(ctx, wordID, updates, tenantID)
}
func (m *mockSensitiveService) DeleteWord(ctx context.Context, wordID int64, tenantID int64) error {
	return m.deleteWordFn(ctx, wordID, tenantID)
}
func (m *mockSensitiveService) BatchImportWords(ctx context.Context, req *model.SensitiveWordBatchReq) (int, error) {
	return m.batchImportFn(ctx, req)
}
func (m *mockSensitiveService) CheckSensitive(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error) {
	return m.checkSensitiveFn(ctx, tenantID, content)
}
func (m *mockSensitiveService) CheckChain(ctx context.Context, tenantID int64, content string, senderID int64, senderType int8, fileType string, fileSize int) *model.CheckChainResp {
	return m.checkChainFn(ctx, tenantID, content, senderID, senderType, fileType, fileSize)
}

func setupSensitiveTest(t *testing.T) (*gin.Engine, *mockSensitiveService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mock := &mockSensitiveService{
		listWordsFn: func(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error) {
			return nil, 0, nil
		},
		createWordFn:   func(ctx context.Context, req *model.SensitiveWordCreateReq) error { return nil },
		updateWordFn:   func(ctx context.Context, wordID int64, updates map[string]interface{}, tenantID int64) error { return nil },
		deleteWordFn:   func(ctx context.Context, wordID int64, tenantID int64) error { return nil },
		batchImportFn:  func(ctx context.Context, req *model.SensitiveWordBatchReq) (int, error) { return 0, nil },
		checkSensitiveFn: func(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error) {
			return &model.SensitiveCheckResp{Passed: true}, nil
		},
		checkChainFn: func(ctx context.Context, tenantID int64, content string, senderID int64, senderType int8, fileType string, fileSize int) *model.CheckChainResp {
			return &model.CheckChainResp{Passed: true}
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
	})
	api := r.Group("/api/v1")
	h := NewSensitiveHandler(mock)
	h.RegisterRoutes(api)
	return r, mock
}

func TestSensitiveListWords(t *testing.T) {
	r, mock := setupSensitiveTest(t)
	mock.listWordsFn = func(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error) {
		return []model.SensitiveWord{{WordID: 1, Word: "bad", Strategy: 1}}, 1, nil
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sensitive/words", nil)
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

func TestSensitiveCreateWord(t *testing.T) {
	r, _ := setupSensitiveTest(t)
	body := `{"word":"bad","strategy":1}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/words", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSensitiveCreateWord_BadRequest(t *testing.T) {
	r, _ := setupSensitiveTest(t)
	body := `{"strategy":1}` // missing word

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/words", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSensitiveUpdateWord(t *testing.T) {
	r, _ := setupSensitiveTest(t)
	body := `{"word":"updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/sensitive/words/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSensitiveDeleteWord(t *testing.T) {
	r, _ := setupSensitiveTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/sensitive/words/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSensitiveBatchImport(t *testing.T) {
	r, mock := setupSensitiveTest(t)
	mock.batchImportFn = func(ctx context.Context, req *model.SensitiveWordBatchReq) (int, error) {
		return len(req.Words), nil
	}
	body := `{"words":[{"word":"bad"},{"word":"evil"}]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/words/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	imported := resp.Data["imported"].(float64)
	if imported != 2 {
		t.Errorf("expected 2 imported, got %v", imported)
	}
}

func TestSensitiveCheckText(t *testing.T) {
	r, mock := setupSensitiveTest(t)
	mock.checkSensitiveFn = func(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error) {
		return &model.SensitiveCheckResp{Passed: false, HitWords: []string{"bad"}, Blocked: true}, nil
	}
	body := `{"content":"bad word"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/check/text", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSensitiveCheckChain(t *testing.T) {
	r, mock := setupSensitiveTest(t)
	mock.checkChainFn = func(ctx context.Context, tenantID int64, content string, senderID int64, senderType int8, fileType string, fileSize int) *model.CheckChainResp {
		return &model.CheckChainResp{Passed: false, SensitiveCheck: &model.SensitiveCheckResp{Passed: false}}
	}
	body := `{"content":"bad","sender_id":1,"sender_type":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSensitiveCheckText_Error(t *testing.T) {
	r, mock := setupSensitiveTest(t)
	mock.checkSensitiveFn = func(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error) {
		return nil, errTest
	}
	body := `{"content":"test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sensitive/check/text", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
