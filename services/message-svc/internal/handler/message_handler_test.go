package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"github.com/shulian-paas/im/message-svc/internal/mq"
	"github.com/shulian-paas/im/message-svc/internal/repo"
	"github.com/shulian-paas/im/message-svc/internal/service"
)

// mockRepo implements the service.messageRepository interface structurally.
type mockRepo struct {
	insertFunc              func(ctx context.Context, msg *model.Message) error
	findByMsgIDFunc         func(ctx context.Context, msgID string) (*model.Message, error)
	findByClientMsgIDFunc   func(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error)
	findByConversationFunc  func(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error)
	updateStatusFunc        func(ctx context.Context, msgID string, status int8, recallTime *time.Time) error
	searchFunc              func(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error)
}

func (m *mockRepo) Insert(ctx context.Context, msg *model.Message) error {
	if m.insertFunc != nil { return m.insertFunc(ctx, msg) }
	return nil
}
func (m *mockRepo) FindByMsgID(ctx context.Context, msgID string) (*model.Message, error) {
	if m.findByMsgIDFunc != nil { return m.findByMsgIDFunc(ctx, msgID) }
	return nil, nil
}
func (m *mockRepo) FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error) {
	if m.findByClientMsgIDFunc != nil { return m.findByClientMsgIDFunc(ctx, conversationID, clientMsgID) }
	return nil, nil
}
func (m *mockRepo) FindByConversation(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error) {
	if m.findByConversationFunc != nil { return m.findByConversationFunc(ctx, conversationID, cursor, limit, direction) }
	return nil, "", nil
}
func (m *mockRepo) UpdateStatus(ctx context.Context, msgID string, status int8, recallTime *time.Time) error {
	if m.updateStatusFunc != nil { return m.updateStatusFunc(ctx, msgID, status, recallTime) }
	return nil
}
func (m *mockRepo) Search(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error) {
	if m.searchFunc != nil { return m.searchFunc(ctx, req) }
	return nil, 0, nil
}

// mockCache implements the service.messageCache interface structurally.
type mockCache struct {
	tryDedupFunc    func(ctx context.Context, clientMsgID string) (bool, error)
	releaseDedupFunc func(ctx context.Context, clientMsgID string) error
	markReadFunc    func(ctx context.Context, conversationID int64, msgID string) error
	isReadFunc      func(ctx context.Context, conversationID int64, msgID string) (bool, int64, error)
}

func (m *mockCache) TryDedup(ctx context.Context, clientMsgID string) (bool, error) {
	if m.tryDedupFunc != nil { return m.tryDedupFunc(ctx, clientMsgID) }
	return false, nil
}
func (m *mockCache) ReleaseDedup(ctx context.Context, clientMsgID string) error {
	if m.releaseDedupFunc != nil { return m.releaseDedupFunc(ctx, clientMsgID) }
	return nil
}
func (m *mockCache) MarkRead(ctx context.Context, conversationID int64, msgID string) error {
	if m.markReadFunc != nil { return m.markReadFunc(ctx, conversationID, msgID) }
	return nil
}
func (m *mockCache) IsRead(ctx context.Context, conversationID int64, msgID string) (bool, int64, error) {
	if m.isReadFunc != nil { return m.isReadFunc(ctx, conversationID, msgID) }
	return false, 0, nil
}

// mockProducer implements the service.messageProducer interface structurally.
type mockProducer struct {
	publishMessageNewFunc      func(ctx context.Context, event *mq.MessagePushEvent) error
	publishMessageRecalledFunc func(ctx context.Context, event *mq.MessagePushEvent) error
	publishBotTriggerFunc      func(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error
}

func (m *mockProducer) PublishMessageNew(ctx context.Context, event *mq.MessagePushEvent) error {
	if m.publishMessageNewFunc != nil { return m.publishMessageNewFunc(ctx, event) }
	return nil
}
func (m *mockProducer) PublishMessageRecalled(ctx context.Context, event *mq.MessagePushEvent) error {
	if m.publishMessageRecalledFunc != nil { return m.publishMessageRecalledFunc(ctx, event) }
	return nil
}
func (m *mockProducer) PublishBotTrigger(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error {
	if m.publishBotTriggerFunc != nil { return m.publishBotTriggerFunc(ctx, msgID, tenantID, convID, convType, groupID, senderID, senderName, content, msgType, atUserIDs) }
	return nil
}

// setupHandler creates a test Gin engine with real Redis (miniredis) + mock repo/producer.
func setupHandler(t *testing.T) (*gin.Engine, *mockRepo, *repo.MessageCache, *mockProducer) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	realCache := repo.NewMessageCache(rdb)

	mrRepo := &mockRepo{}
	mrProd := &mockProducer{}

	svc := service.NewMessageService(mrRepo, realCache, mrProd, 10)
	h := NewMessageHandler(svc, realCache)

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)

	return r, mrRepo, realCache, mrProd
}

// request helper: performs an HTTP request and returns the response
func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		buf = *bytes.NewBuffer(data)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	// Auth context would normally be set by middleware; manually set for tests
	// We just test the handler routes directly
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- Tests ----

func TestSendMessage_Success(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	mockRepo.insertFunc = func(ctx context.Context, msg *model.Message) error { return nil }
	mockRepo.findByClientMsgIDFunc = func(ctx context.Context, cid int64, id string) (*model.Message, error) { return nil, nil }

	body := map[string]interface{}{
		"client_msg_id":   "client-001",
		"conversation_id": 50001,
		"conv_type":       2,
		"msg_type":        1,
		"content":         map[string]interface{}{"text": "hello"},
		"sender_id":       20001,
		"sender_name":     "测试用户",
	}

	w := doRequest(r, "POST", "/api/v1/messages", body)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestSendMessage_BadRequest(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	// Empty body: should fail binding
	w := doRequest(r, "POST", "/api/v1/messages", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	req := httptest.NewRequest("POST", "/api/v1/messages",
		bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestPullMessages_Success(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	now := time.Now()
	mockRepo.findByConversationFunc = func(ctx context.Context, cid int64, cursor string, limit int, dir int) ([]model.Message, string, error) {
		return []model.Message{
			{MsgID: "msg_002", SendTime: now, Status: model.MsgStatusSent, Content: model.MsgContent{"text": "two"}},
			{MsgID: "msg_001", SendTime: now.Add(-time.Minute), Status: model.MsgStatusSent, Content: model.MsgContent{"text": "one"}},
		}, "next_cursor", nil
	}

	w := doRequest(r, "GET", "/api/v1/messages?conversation_id=50001&cursor=abc&limit=20", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("expected data")
	}
	list, ok := resp.Data["list"].([]interface{})
	if !ok || len(list) != 2 {
		t.Errorf("expected 2 messages, got %v", resp.Data["list"])
	}
}

func TestPullMessages_NoConversationID(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	w := doRequest(r, "GET", "/api/v1/messages?cursor=abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRecallMessage_Success(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	mockRepo.findByMsgIDFunc = func(ctx context.Context, msgID string) (*model.Message, error) {
		return &model.Message{
			MsgID: "msg_001", SenderID: 20001, SendTime: time.Now(), Status: model.MsgStatusSent,
		}, nil
	}
	mockRepo.updateStatusFunc = func(ctx context.Context, msgID string, status int8, recallTime *time.Time) error {
		return nil
	}

	w := doRequest(r, "POST", "/api/v1/messages/msg_001/recall", map[string]interface{}{
		"sender_id": 20001,
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestRecallMessage_NotFound(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	mockRepo.findByMsgIDFunc = func(ctx context.Context, msgID string) (*model.Message, error) {
		return nil, nil
	}

	w := doRequest(r, "POST", "/api/v1/messages/notexist/recall", map[string]interface{}{
		"sender_id": 20001,
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestForwardMessages_Success(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	mockRepo.findByMsgIDFunc = func(ctx context.Context, msgID string) (*model.Message, error) {
		return &model.Message{MsgID: msgID, MsgType: 1, Content: model.MsgContent{"text": "fwd"}}, nil
	}
	mockRepo.insertFunc = func(ctx context.Context, msg *model.Message) error { return nil }

	body := map[string]interface{}{
		"msg_ids":      []string{"m1", "m2"},
		"target_type":  2,
		"target_id":    60001,
		"forward_type": 2,
		"sender_id":    20001,
		"sender_name":  "测试用户",
	}

	w := doRequest(r, "POST", "/api/v1/messages/forward", body)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestForwardMessages_BadRequest(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	w := doRequest(r, "POST", "/api/v1/messages/forward", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearchMessages_Success(t *testing.T) {
	r, mockRepo, _, _ := setupHandler(t)
	now := time.Now()
	mockRepo.searchFunc = func(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error) {
		return []model.Message{
			{MsgID: "msg_001", SendTime: now, Content: model.MsgContent{"text": "found it"}},
		}, 1, nil
	}

	w := doRequest(r, "GET", "/api/v1/messages/search?q=hello&conversation_id=50001&page=1&page_size=20", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarkRead_Success(t *testing.T) {
	r, _, realCache, _ := setupHandler(t)

	body := map[string]string{"msg_id": "msg_001"}
	w := doRequest(r, "PUT", "/api/v1/conversations/50001/read", body)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was actually marked
	isRead, _, err := realCache.IsRead(context.Background(), 50001, "msg_001")
	if err != nil {
		t.Fatalf("IsRead: %v", err)
	}
	if !isRead {
		t.Error("message should be marked as read")
	}
}

func TestMarkRead_InvalidConversationID(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	w := doRequest(r, "PUT", "/api/v1/conversations/0/read", map[string]string{"msg_id": "msg_001"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetReadReceipt(t *testing.T) {
	r, _, realCache, _ := setupHandler(t)

	// First mark as read
	ctx := context.Background()
	realCache.MarkRead(ctx, 50001, "msg_read")

	// Then get receipt
	w := doRequest(r, "GET", "/api/v1/messages/msg_read/read-receipt?conversation_id=50001", nil)
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
	if resp.Data == nil {
		t.Fatal("expected data")
	}
	isRead, ok := resp.Data["is_read"].(bool)
	if !ok || !isRead {
		t.Error("expected is_read=true")
	}
}

func TestGetReadStatus(t *testing.T) {
	r, _, realCache, _ := setupHandler(t)

	ctx := context.Background()
	realCache.MarkRead(ctx, 50001, "msg_001")

	w := doRequest(r, "GET", "/api/v1/messages/msg_001/read-status?conversation_id=50001", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendSSEToken_Success(t *testing.T) {
	r, _, realCache, _ := setupHandler(t)

	body := map[string]interface{}{
		"bot_id": 4001,
		"msg_id": "msg_sse_001",
		"token":  "tok_secret_abc",
	}
	w := doRequest(r, "POST", "/api/v1/messages/sse", body)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify stored in Redis
	token, err := realCache.GetSSEToken(context.Background(), "msg_sse_001")
	if err != nil {
		t.Fatalf("GetSSEToken: %v", err)
	}
	if token != "tok_secret_abc" {
		t.Errorf("token: got %s, want tok_secret_abc", token)
	}
}

func TestSendSSEToken_BadRequest(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	// Missing required fields
	w := doRequest(r, "POST", "/api/v1/messages/sse", map[string]interface{}{"bot_id": 4001})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRouteOrder_StaticBeforeParam(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	// POST /api/v1/messages/sse should NOT be caught by /api/v1/messages/:msg_id/recall
	w := doRequest(r, "POST", "/api/v1/messages/sse", map[string]interface{}{
		"bot_id": 4001,
		"msg_id": "msg_001",
		"token":  "tok_abc",
	})
	if w.Code == http.StatusNotFound {
		t.Error("sse route should not return 404")
	}
}

func TestGetReadReceipt_MissingConversationID(t *testing.T) {
	r, _, _, _ := setupHandler(t)
	w := doRequest(r, "GET", "/api/v1/messages/msg_001/read-receipt", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
