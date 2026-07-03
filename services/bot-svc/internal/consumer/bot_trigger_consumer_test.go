package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/segmentio/kafka-go"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
)

// TestBotTriggerEvent_JSONSerde tests that BotTriggerEvent serializes/deserializes correctly
func TestBotTriggerEvent_JSONSerde(t *testing.T) {
	event := model.BotTriggerEvent{
		EventID:   "evt_001",
		EventType: "message.mention",
		Timestamp: 1720080000000,
		Message: model.MessageContext{
			MsgID:     "msg_001",
			Text:      "查服务器",
			AtUserIDs: []int64{10001, 4001},
			MsgType:   1,
		},
		Sender: model.SenderInfo{
			UserID:   10001,
			UserName: "张三",
		},
		Conversation: model.ConversationContext{
			ConvID:   "sg_20001",
			ConvType: 2,
			GroupID:  int64Ptr(20001),
			GroupName: "运维群",
		},
		BotIDs: []int64{4001},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded model.BotTriggerEvent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.EventID != "evt_001" {
		t.Errorf("event_id mismatch: %s", decoded.EventID)
	}
	if decoded.Message.Text != "查服务器" {
		t.Errorf("text mismatch: %s", decoded.Message.Text)
	}
	if len(decoded.BotIDs) != 1 || decoded.BotIDs[0] != 4001 {
		t.Errorf("bot_ids mismatch: %v", decoded.BotIDs)
	}
}

// TestConsumer_ProcessMessage tests internal message processing without Kafka
func TestConsumer_ProcessMessage(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis failed: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := repo.NewBotCache(rdb)

	// Add bots to cache
	_ = cache.AddUserID(context.Background(), 4001)
	_ = cache.AddUserID(context.Background(), 4002)

	// Create event with @bots
	event := model.BotTriggerEvent{
		EventID:   "evt_test_001",
		EventType: "message.mention",
		Message:   model.MessageContext{Text: "@bot1 @bot2 查状态", AtUserIDs: []int64{10001, 4001, 4002}},
		BotIDs:    []int64{4001, 4002},
	}

	// Test IntersectUserIDs
	matched, err := cache.IntersectUserIDs(context.Background(), event.Message.AtUserIDs)
	if err != nil {
		t.Fatalf("intersect failed: %v", err)
	}

	if len(matched) != 2 {
		t.Errorf("expected 2 bots matched, got %d: %v", len(matched), matched)
	}
}

// TestConsumer_NoMatch tests that a message without @bot has no intersection
func TestConsumer_NoMatch(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis failed: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := repo.NewBotCache(rdb)

	_ = cache.AddUserID(context.Background(), 4001)

	matched, err := cache.IntersectUserIDs(context.Background(), []int64{10001, 10002})
	if err != nil {
		t.Fatalf("intersect failed: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("expected no match, got %v", matched)
	}
}

// Test End-to-End: mock Kafka message -> consumer process
func TestConsumer_KafkaMessageProcessing(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis failed: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := repo.NewBotCache(rdb)

	// Setup: bot 4001 exists
	_ = cache.AddUserID(context.Background(), 4001)

	// Simulate a Kafka message arriving
	event := model.BotTriggerEvent{
		EventID:   "evt_kafka_001",
		EventType: "message.mention",
		Message:   model.MessageContext{MsgID: "msg_k001", Text: "@bot 查状态", AtUserIDs: []int64{4001}},
		BotIDs:    []int64{4001},
	}

	data, _ := json.Marshal(event)
	msg := kafka.Message{
		Topic:     "bot_trigger",
		Partition: 0,
		Offset:    100,
		Key:       []byte("evt_kafka_001"),
		Value:     data,
		Time:      time.Now(),
	}

	// Verify the message can be parsed and the bot_ids extracted
	var parsed model.BotTriggerEvent
	if err := json.Unmarshal(msg.Value, &parsed); err != nil {
		t.Fatalf("unmarshal kafka message failed: %v", err)
	}

	if parsed.EventID != "evt_kafka_001" {
		t.Errorf("event_id mismatch: %s", parsed.EventID)
	}

	// Verify bot intersection
	matched, err := cache.IntersectUserIDs(context.Background(), parsed.Message.AtUserIDs)
	if err != nil {
		t.Fatalf("intersect failed: %v", err)
	}
	if len(matched) != 1 || matched[0] != 4001 {
		t.Errorf("expected [4001], got %v", matched)
	}
}

func int64Ptr(i int64) *int64 { return &i }
