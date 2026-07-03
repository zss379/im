package repo

import (
	"context"
	"testing"
	"time"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestInsert(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	msg := &model.Message{
		MsgID:          "msg_001",
		TenantID:       1001,
		ConversationID: 50001,
		ConvType:       2,
		SenderID:       20001,
		SenderName:     "test",
		MsgType:        1,
		Content:        model.MsgContent{"text": "hello"},
		Status:         model.MsgStatusSent,
		SendTime:       time.Now(),
	}

	mt.AddMockResponses(mtest.CreateSuccessResponse()...)
	err := repo.Insert(mt.DB, msg)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
}

func TestFindByMsgID_Found(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	now := time.Now()
	expectedDoc := bson.D{
		{"_id", primitive.NewObjectID()},
		{"msg_id", "msg_001"},
		{"tenant_id", int64(1001)},
		{"conversation_id", int64(50001)},
		{"sender_id", int64(20001)},
		{"sender_name", "test"},
		{"msg_type", int8(1)},
		{"status", int8(model.MsgStatusSent)},
		{"send_time", now},
	}

	mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.messages", mtest.FirstBatch, expectedDoc))
	msg, err := repo.FindByMsgID(mt.DB, "msg_001")
	if err != nil {
		t.Fatalf("FindByMsgID failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.MsgID != "msg_001" {
		t.Errorf("MsgID: got %s, want msg_001", msg.MsgID)
	}
}

func TestFindByMsgID_NotFound(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	// Return no documents (empty cursor)
	mt.AddMockResponses(mtest.CreateCursorResponse(0, "db.messages", mtest.FirstBatch))
	msg, err := repo.FindByMsgID(mt.DB, "nonexistent")
	if err != nil {
		t.Fatalf("FindByMsgID failed: %v", err)
	}
	if msg != nil {
		t.Error("expected nil for not found")
	}
}

func TestFindByClientMsgID(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	expectedDoc := bson.D{
		{"_id", primitive.NewObjectID()},
		{"msg_id", "msg_001"},
		{"client_msg_id", "client-abc"},
		{"conversation_id", int64(50001)},
		{"sender_id", int64(20001)},
		{"status", int8(model.MsgStatusSent)},
	}

	mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.messages", mtest.FirstBatch, expectedDoc))
	msg, err := repo.FindByClientMsgID(mt.DB, 50001, "client-abc")
	if err != nil {
		t.Fatalf("FindByClientMsgID failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.ClientMsgID != "client-abc" {
		t.Errorf("ClientMsgID: got %s, want client-abc", msg.ClientMsgID)
	}
}

func TestUpdateStatus(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	now := time.Now()
	mt.AddMockResponses(mtest.CreateSuccessResponse()...)
	err := repo.UpdateStatus(mt.DB, "msg_001", model.MsgStatusRecalled, &now)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
}

func TestUpdateStatus_NilRecallTime(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	mt.AddMockResponses(mtest.CreateSuccessResponse()...)
	err := repo.UpdateStatus(mt.DB, "msg_001", model.MsgStatusSent, nil)
	if err != nil {
		t.Fatalf("UpdateStatus with nil time: %v", err)
	}
}

func TestFindByConversation_NoCursor(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	now := time.Now()
	doc1 := bson.D{
		{"_id", primitive.NewObjectID()},
		{"msg_id", "msg_002"},
		{"conversation_id", int64(50001)},
		{"send_time", now},
		{"status", int8(model.MsgStatusSent)},
		{"is_deleted", false},
	}
	doc2 := bson.D{
		{"_id", primitive.NewObjectID()},
		{"msg_id", "msg_001"},
		{"conversation_id", int64(50001)},
		{"send_time", now.Add(-time.Minute)},
		{"status", int8(model.MsgStatusSent)},
		{"is_deleted", false},
	}

	mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.messages", mtest.FirstBatch, doc1, doc2))
	msgs, cursor, err := repo.FindByConversation(mt.DB, 50001, "", 20, 0)
	if err != nil {
		t.Fatalf("FindByConversation failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if cursor == "" {
		t.Error("expected non-empty cursor when hasMore is false but messages exist")
	}
}

func TestSearch(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	repo := NewMessageRepo(mt.DB)

	now := time.Now()
	doc := bson.D{
		{"_id", primitive.NewObjectID()},
		{"msg_id", "msg_001"},
		{"conversation_id", int64(50001)},
		{"sender_id", int64(20001)},
		{"send_time", now},
		{"status", int8(model.MsgStatusSent)},
		{"is_deleted", false},
	}

	// CountDocuments response
	mt.AddMockResponses(mtest.CreateSuccessResponse(bson.D{{"n", int64(1)}})...)
	// Find response
	mt.AddMockResponses(mtest.CreateCursorResponse(1, "db.messages", mtest.FirstBatch, doc))

	req := &model.SearchReq{
		Q:              "hello",
		ConversationID: 50001,
		SenderID:       20001,
		Page:           1,
		PageSize:       20,
	}

	msgs, total, err := repo.Search(mt.DB, req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}
