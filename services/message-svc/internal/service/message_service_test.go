package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"github.com/shulian-paas/im/message-svc/internal/mq"
)

// mock implementations

type mockRepo struct {
	insertFunc           func(ctx context.Context, msg *model.Message) error
	findByMsgIDFunc      func(ctx context.Context, msgID string) (*model.Message, error)
	findByClientMsgIDFunc func(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error)
	findByConversationFunc func(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error)
	updateStatusFunc     func(ctx context.Context, msgID string, status int8, recallTime *time.Time) error
	searchFunc           func(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error)
}

func (m *mockRepo) Insert(ctx context.Context, msg *model.Message) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, msg)
	}
	return nil
}

func (m *mockRepo) FindByMsgID(ctx context.Context, msgID string) (*model.Message, error) {
	if m.findByMsgIDFunc != nil {
		return m.findByMsgIDFunc(ctx, msgID)
	}
	return nil, nil
}

func (m *mockRepo) FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error) {
	if m.findByClientMsgIDFunc != nil {
		return m.findByClientMsgIDFunc(ctx, conversationID, clientMsgID)
	}
	return nil, nil
}

func (m *mockRepo) FindByConversation(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error) {
	if m.findByConversationFunc != nil {
		return m.findByConversationFunc(ctx, conversationID, cursor, limit, direction)
	}
	return nil, "", nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, msgID string, status int8, recallTime *time.Time) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, msgID, status, recallTime)
	}
	return nil
}

func (m *mockRepo) Search(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	return nil, 0, nil
}

type mockCache struct {
	tryDedupFunc    func(ctx context.Context, clientMsgID string) (bool, error)
	releaseDedupFunc func(ctx context.Context, clientMsgID string) error
	markReadFunc    func(ctx context.Context, conversationID int64, msgID string) error
	isReadFunc      func(ctx context.Context, conversationID int64, msgID string) (bool, int64, error)
}

func (m *mockCache) TryDedup(ctx context.Context, clientMsgID string) (bool, error) {
	if m.tryDedupFunc != nil {
		return m.tryDedupFunc(ctx, clientMsgID)
	}
	return false, nil
}

func (m *mockCache) ReleaseDedup(ctx context.Context, clientMsgID string) error {
	if m.releaseDedupFunc != nil {
		return m.releaseDedupFunc(ctx, clientMsgID)
	}
	return nil
}

func (m *mockCache) MarkRead(ctx context.Context, conversationID int64, msgID string) error {
	if m.markReadFunc != nil {
		return m.markReadFunc(ctx, conversationID, msgID)
	}
	return nil
}

func (m *mockCache) IsRead(ctx context.Context, conversationID int64, msgID string) (bool, int64, error) {
	if m.isReadFunc != nil {
		return m.isReadFunc(ctx, conversationID, msgID)
	}
	return false, 0, nil
}

type mockProducer struct {
	publishMessageNewFunc      func(ctx context.Context, event *mq.MessagePushEvent) error
	publishMessageRecalledFunc func(ctx context.Context, event *mq.MessagePushEvent) error
	publishBotTriggerFunc      func(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error
}

func (m *mockProducer) PublishMessageNew(ctx context.Context, event *mq.MessagePushEvent) error {
	if m.publishMessageNewFunc != nil {
		return m.publishMessageNewFunc(ctx, event)
	}
	return nil
}

func (m *mockProducer) PublishMessageRecalled(ctx context.Context, event *mq.MessagePushEvent) error {
	if m.publishMessageRecalledFunc != nil {
		return m.publishMessageRecalledFunc(ctx, event)
	}
	return nil
}

func (m *mockProducer) PublishBotTrigger(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error {
	if m.publishBotTriggerFunc != nil {
		return m.publishBotTriggerFunc(ctx, msgID, tenantID, convID, convType, groupID, senderID, senderName, content, msgType, atUserIDs)
	}
	return nil
}

func newTestService(repo messageRepository, cache messageCache, producer messageProducer) *MessageService {
	return &MessageService{
		msgRepo:    repo,
		cache:      cache,
		producer:   producer,
		botMsgRate: 10,
	}
}

// ---- Tests ----

func TestSendMessage_Success(t *testing.T) {
	var capturedMsg *model.Message

	repo := &mockRepo{
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			capturedMsg = msg
			return nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, clientMsgID string) (bool, error) {
			return false, nil
		},
	}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.SendMessageReq{
		ClientMsgID:    "client-001",
		ConversationID: 50001,
		ConvType:       2,
		MsgType:        1,
		Content:        model.MsgContent{"text": "hello"},
		SenderID:       20001,
		SenderName:     "测试用户",
	}

	resp, err := svc.SendMessage(context.Background(), req, 1001)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.MsgID == "" {
		t.Error("expected non-empty MsgID")
	}
	if resp.Status != model.MsgStatusSent {
		t.Errorf("expected status %d, got %d", model.MsgStatusSent, resp.Status)
	}

	if capturedMsg == nil {
		t.Fatal("message was not inserted")
	}
	if capturedMsg.TenantID != 1001 {
		t.Errorf("TenantID: got %d, want 1001", capturedMsg.TenantID)
	}
	if capturedMsg.SenderID != 20001 {
		t.Errorf("SenderID: got %d, want 20001", capturedMsg.SenderID)
	}
}

func TestSendMessage_Dedup(t *testing.T) {
	repo := &mockRepo{
		findByClientMsgIDFunc: func(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error) {
			return &model.Message{
				MsgID:    "existing_msg",
				SendTime: time.Now(),
				Status:   model.MsgStatusSent,
			}, nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, clientMsgID string) (bool, error) {
			return true, nil // duplicate
		},
	}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.SendMessageReq{
		ClientMsgID:    "dup-client-id",
		ConversationID: 50001,
		MsgType:        1,
		Content:        model.MsgContent{"text": "hello"},
		SenderID:       20001,
	}

	resp, err := svc.SendMessage(context.Background(), req, 1001)
	if err != nil {
		t.Fatalf("SendMessage dedup failed: %v", err)
	}

	if resp.MsgID != "existing_msg" {
		t.Errorf("expected existing_msg, got %s", resp.MsgID)
	}
}

func TestSendMessage_BotTriggerPublished(t *testing.T) {
	var triggered bool

	repo := &mockRepo{
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			return nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	producer := &mockProducer{
		publishBotTriggerFunc: func(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error {
			triggered = true
			return nil
		},
	}

	svc := newTestService(repo, cache, producer)
	req := &model.SendMessageReq{
		ConversationID: 50001,
		MsgType:        1,
		Content:        model.MsgContent{"text": "@bot hello"},
		SenderID:       20001,
		SenderName:     "测试用户",
		AtUserList:     []int64{4001}, // has at-user = bot
	}

	svc.SendMessage(context.Background(), req, 1001)
	if !triggered {
		t.Error("bot_trigger should be published when AtUserList is non-empty")
	}
}

func TestSendMessage_BotTriggerSkippedForBotSender(t *testing.T) {
	var triggered bool

	repo := &mockRepo{
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			return nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	producer := &mockProducer{
		publishBotTriggerFunc: func(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error {
			triggered = true
			return nil
		},
	}

	botID := int64(4001)
	svc := newTestService(repo, cache, producer)
	req := &model.SendMessageReq{
		ConversationID: 50001,
		MsgType:        1,
		Content:        model.MsgContent{"text": "hello"},
		SenderID:       20001,
		SenderBotID:    &botID,
		AtUserList:     []int64{4002},
	}

	svc.SendMessage(context.Background(), req, 1001)
	if triggered {
		t.Error("bot_trigger should NOT be published for bot senders")
	}
}

func TestSendMessage_DBErrorReleasesDedup(t *testing.T) {
	var dedupReleased bool

	repo := &mockRepo{
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			return errors.New("db error")
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
		releaseDedupFunc: func(ctx context.Context, clientMsgID string) error {
			dedupReleased = true
			return nil
		},
	}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.SendMessageReq{
		ClientMsgID:    "client-rollback",
		ConversationID: 50001,
		MsgType:        1,
		Content:        model.MsgContent{"text": "hello"},
		SenderID:       20001,
	}

	_, err := svc.SendMessage(context.Background(), req, 1001)
	if err == nil {
		t.Fatal("expected error from DB insert")
	}
	if !dedupReleased {
		t.Error("dedup key should be released on DB error")
	}
}

func TestPullMessages_Success(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		findByConversationFunc: func(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error) {
			return []model.Message{
				{MsgID: "msg_003", SendTime: now, Status: model.MsgStatusSent, Content: model.MsgContent{"text": "third"}},
				{MsgID: "msg_002", SendTime: now.Add(-time.Minute), Status: model.MsgStatusSent, Content: model.MsgContent{"text": "second"}},
			}, "next_cursor", nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	resp, err := svc.PullMessages(context.Background(), 50001, "cursor_abc", 20)
	if err != nil {
		t.Fatalf("PullMessages failed: %v", err)
	}
	if len(resp.List) != 2 {
		t.Errorf("expected 2 messages, got %d", len(resp.List))
	}
	if resp.Cursor != "next_cursor" {
		t.Errorf("cursor: got %s, want next_cursor", resp.Cursor)
	}
	if !resp.HasMore {
		t.Error("hasMore should be true")
	}
}

// RecallMessage tests

func TestRecallMessage_Success(t *testing.T) {
	var updatedStatus int8
	var updatedRecallTime *time.Time
	now := time.Now()

	repo := &mockRepo{
		findByMsgIDFunc: func(ctx context.Context, msgID string) (*model.Message, error) {
			return &model.Message{
				MsgID:    "msg_001",
				SenderID: 20001,
				SendTime: now,
				Status:   model.MsgStatusSent,
			}, nil
		},
		updateStatusFunc: func(ctx context.Context, msgID string, status int8, recallTime *time.Time) error {
			updatedStatus = status
			updatedRecallTime = recallTime
			return nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{
		publishMessageRecalledFunc: func(ctx context.Context, event *mq.MessagePushEvent) error {
			return nil
		},
	}

	svc := newTestService(repo, cache, producer)
	err := svc.RecallMessage(context.Background(), "msg_001", 20001, 1001)
	if err != nil {
		t.Fatalf("RecallMessage failed: %v", err)
	}

	if updatedStatus != model.MsgStatusRecalled {
		t.Errorf("status: got %d, want %d", updatedStatus, model.MsgStatusRecalled)
	}
	if updatedRecallTime == nil {
		t.Error("recallTime should not be nil")
	}
}

func TestRecallMessage_NotSender(t *testing.T) {
	repo := &mockRepo{
		findByMsgIDFunc: func(ctx context.Context, msgID string) (*model.Message, error) {
			return &model.Message{
				MsgID:    "msg_001",
				SenderID: 20001,
				SendTime: time.Now(),
			}, nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	err := svc.RecallMessage(context.Background(), "msg_001", 99999, 1001)
	if err == nil {
		t.Fatal("expected error for wrong sender")
	}
	if err.Error() != "no permission: not the sender" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRecallMessage_Timeout(t *testing.T) {
	repo := &mockRepo{
		findByMsgIDFunc: func(ctx context.Context, msgID string) (*model.Message, error) {
			return &model.Message{
				MsgID:    "msg_001",
				SenderID: 20001,
				SendTime: time.Now().Add(-3 * time.Minute), // beyond 2-min window
			}, nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	err := svc.RecallMessage(context.Background(), "msg_001", 20001, 1001)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err.Error() != "recall timeout: exceeds 2 minutes" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRecallMessage_NotFound(t *testing.T) {
	repo := &mockRepo{
		findByMsgIDFunc: func(ctx context.Context, msgID string) (*model.Message, error) {
			return nil, nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	err := svc.RecallMessage(context.Background(), "nonexistent", 20001, 1001)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// ForwardMessages tests

func TestForwardMessages_Individual(t *testing.T) {
	var inserted int

	repo := &mockRepo{
		findByMsgIDFunc: func(ctx context.Context, msgID string) (*model.Message, error) {
			return &model.Message{
				MsgID:  msgID,
				MsgType: 1,
				Content: model.MsgContent{"text": "hello"},
			}, nil
		},
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			inserted++
			return nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.ForwardReq{
		MsgIDs:      []string{"m1", "m2", "m3"},
		TargetID:    60001,
		TargetType:  2,
		ForwardType: 1,
		SenderID:    20001,
		SenderName:  "测试用户",
	}

	msgIDs, err := svc.ForwardMessages(context.Background(), req, 1001)
	if err != nil {
		t.Fatalf("ForwardMessages failed: %v", err)
	}
	if len(msgIDs) != 3 {
		t.Errorf("expected 3 forwarded msg IDs, got %d", len(msgIDs))
	}
	if inserted != 3 {
		t.Errorf("expected 3 inserts, got %d", inserted)
	}
}

func TestForwardMessages_Merge(t *testing.T) {
	var capturedMsg *model.Message

	repo := &mockRepo{
		insertFunc: func(ctx context.Context, msg *model.Message) error {
			capturedMsg = msg
			return nil
		},
	}
	cache := &mockCache{
		tryDedupFunc: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.ForwardReq{
		MsgIDs:      []string{"m1", "m2"},
		TargetID:    60001,
		TargetType:  2,
		ForwardType: 2,
		SenderID:    20001,
		SenderName:  "测试用户",
	}

	msgIDs, err := svc.ForwardMessages(context.Background(), req, 1001)
	if err != nil {
		t.Fatalf("ForwardMessages merge failed: %v", err)
	}
	if len(msgIDs) != 1 {
		t.Errorf("expected 1 merged msg ID, got %d", len(msgIDs))
	}
	if capturedMsg == nil {
		t.Fatal("merge message not inserted")
	}
	if capturedMsg.MsgType != model.MsgTypeMergeForward {
		t.Errorf("expected MsgType %d, got %d", model.MsgTypeMergeForward, capturedMsg.MsgType)
	}
}

// MarkRead tests

func TestMarkRead(t *testing.T) {
	var readConversationID int64
	var readMsgID string

	cache := &mockCache{
		markReadFunc: func(ctx context.Context, conversationID int64, msgID string) error {
			readConversationID = conversationID
			readMsgID = msgID
			return nil
		},
	}
	repo := &mockRepo{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	err := svc.MarkRead(context.Background(), 50001, "msg_001")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}
	if readConversationID != 50001 {
		t.Errorf("conversationID: got %d, want 50001", readConversationID)
	}
	if readMsgID != "msg_001" {
		t.Errorf("msgID: got %s, want msg_001", readMsgID)
	}
}

func TestGetReadReceipt(t *testing.T) {
	cache := &mockCache{
		isReadFunc: func(ctx context.Context, conversationID int64, msgID string) (bool, int64, error) {
			if msgID == "msg_read" {
				return true, 1720000000000, nil
			}
			return false, 0, nil
		},
	}
	repo := &mockRepo{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)

	// Read message
	resp, err := svc.GetReadReceipt(context.Background(), 50001, "msg_read")
	if err != nil {
		t.Fatalf("GetReadReceipt failed: %v", err)
	}
	if !resp.IsRead {
		t.Error("expected is_read=true")
	}
	if resp.ReadAt == "" {
		t.Error("expected non-empty read_at")
	}

	// Unread message
	resp, err = svc.GetReadReceipt(context.Background(), 50001, "msg_unread")
	if err != nil {
		t.Fatalf("GetReadReceipt failed: %v", err)
	}
	if resp.IsRead {
		t.Error("expected is_read=false")
	}
	if resp.ReadAt != "" {
		t.Error("expected empty read_at for unread message")
	}
}

func TestSearchMessages(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		searchFunc: func(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error) {
			return []model.Message{
				{MsgID: "msg_001", SendTime: now, Content: model.MsgContent{"text": "found it"}},
			}, 1, nil
		},
	}
	cache := &mockCache{}
	producer := &mockProducer{}

	svc := newTestService(repo, cache, producer)
	req := &model.SearchReq{Q: "found", ConversationID: 50001, Page: 1, PageSize: 20}

	messages, total, err := svc.SearchMessages(context.Background(), req)
	if err != nil {
		t.Fatalf("SearchMessages failed: %v", err)
	}
	if total != 1 {
		t.Errorf("total: got %d, want 1", total)
	}
	if len(messages) != 1 {
		t.Errorf("messages: got %d, want 1", len(messages))
	}
}
