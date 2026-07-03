package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageJSONSerialization(t *testing.T) {
	sendTime := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	msg := &Message{
		MsgID:          "msg_test_001",
		TenantID:       1001,
		ConversationID: 50001,
		ConvType:       2,
		SenderID:       20001,
		SenderName:     "测试用户",
		SenderAvatar:   "https://example.com/avatar.png",
		MsgType:        1,
		Content:        MsgContent{"text": "hello"},
		Status:         MsgStatusSent,
		ClientMsgID:    "client-abc-123",
		SendTime:       sendTime,
		AtUserList:     []int64{30001, 30002},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if decoded.MsgID != msg.MsgID {
		t.Errorf("MsgID: got %s, want %s", decoded.MsgID, msg.MsgID)
	}
	if decoded.TenantID != msg.TenantID {
		t.Errorf("TenantID: got %d, want %d", decoded.TenantID, msg.TenantID)
	}
	if decoded.Content["text"] != "hello" {
		t.Errorf("Content text: got %v, want hello", decoded.Content["text"])
	}
	if len(decoded.AtUserList) != 2 || decoded.AtUserList[0] != 30001 {
		t.Errorf("AtUserList: got %v, want [30001 30002]", decoded.AtUserList)
	}
}

func TestMsgContentVariants(t *testing.T) {
	tests := []struct {
		name    string
		msgType int8
		content MsgContent
	}{
		{"text", MsgTypeText, MsgContent{"text": "hello"}},
		{"image", MsgTypeImage, MsgContent{"url": "https://example.com/img.png", "width": 800, "height": 600}},
		{"video", MsgTypeVideo, MsgContent{"url": "https://example.com/vid.mp4", "duration": 30}},
		{"file", MsgTypeFile, MsgContent{"name": "doc.pdf", "size": 1024000}},
		{"voice", MsgTypeVoice, MsgContent{"url": "https://example.com/voice.amr", "duration": 10}},
		{"card", MsgTypeCard, MsgContent{"title": "系统通知", "desc": "欢迎使用"}},
		{"merge_forward", MsgTypeMergeForward, MsgContent{"title": "聊天记录", "msg_ids": []string{"m1", "m2"}}},
		{"sse", MsgTypeSSE, MsgContent{"text": "streaming response", "stream_id": "sse_001"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{MsgType: tt.msgType, Content: tt.content, Status: MsgStatusSent, SendTime: time.Now()}
			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			var decoded Message
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if decoded.MsgType != tt.msgType {
				t.Errorf("MsgType: got %d, want %d", decoded.MsgType, tt.msgType)
			}
		})
	}
}

func TestSendMessageReqSerialization(t *testing.T) {
	req := &SendMessageReq{
		ClientMsgID:    "client-001",
		MsgType:        1,
		Content:        MsgContent{"text": "hello"},
		ConversationID: 50001,
		ConvType:       2,
		SenderID:       20001,
		SenderName:     "测试用户",
		AtUserList:     []int64{30001},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SendMessageReq
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ConversationID != req.ConversationID {
		t.Errorf("ConversationID: got %d, want %d", decoded.ConversationID, req.ConversationID)
	}
}

func TestSearchReqDefaults(t *testing.T) {
	req := &SearchReq{}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.Page != 1 || req.PageSize != 20 {
		t.Errorf("defaults: got page=%d pageSize=%d, want 1, 20", req.Page, req.PageSize)
	}
}

func TestStatusConstants(t *testing.T) {
	if MsgStatusSending != 1 || MsgStatusSent != 2 || MsgStatusFailed != 3 || MsgStatusRecalled != 4 {
		t.Error("status constants mismatch")
	}
}

func TestMsgTypeConstants(t *testing.T) {
	if MsgTypeText != 1 || MsgTypeImage != 2 || MsgTypeMergeForward != 10 {
		t.Error("msg_type constants mismatch")
	}
}

func TestSSETokenReq(t *testing.T) {
	req := SSETokenReq{BotID: 4001, MsgID: "msg_001", Token: "tok_abc"}
	data, _ := json.Marshal(req)
	var decoded SSETokenReq
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.BotID != 4001 || decoded.MsgID != "msg_001" || decoded.Token != "tok_abc" {
		t.Error("SSETokenReq fields mismatch")
	}
}

func TestForwardReq(t *testing.T) {
	req := ForwardReq{
		MsgIDs:     []string{"m1", "m2"},
		TargetType: 2,
		TargetID:   50001,
		ForwardType: 1,
		SenderID:   20001,
		SenderName: "测试用户",
	}
	data, _ := json.Marshal(req)
	var decoded ForwardReq
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal forward req: %v", err)
	}
	if len(decoded.MsgIDs) != 2 || decoded.TargetID != 50001 {
		t.Error("ForwardReq fields mismatch")
	}
}
