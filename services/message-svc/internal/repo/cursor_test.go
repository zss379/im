package repo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEncodeDecodeCursor(t *testing.T) {
	sendTime := time.Date(2026, 7, 3, 10, 0, 0, 123456789, time.UTC)
	oid := primitive.NewObjectID()

	cursor := encodeCursor(sendTime, oid)
	if cursor == "" {
		t.Fatal("encodeCursor returned empty string")
	}

	decoded, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decodeCursor failed: %v", err)
	}

	if decoded.LastSendTime != sendTime.UnixMilli() {
		t.Errorf("LastSendTime: got %d, want %d", decoded.LastSendTime, sendTime.UnixMilli())
	}
	if decoded.LastID != oid.Hex() {
		t.Errorf("LastID: got %s, want %s", decoded.LastID, oid.Hex())
	}
}

func TestCursorRoundTripJSON(t *testing.T) {
	original := model.Cursor{
		LastSendTime: 1720000000000,
		LastID:       "507f1f77bcf86cd799439011",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal cursor: %v", err)
	}

	var decoded model.Cursor
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal cursor: %v", err)
	}

	if decoded.LastSendTime != original.LastSendTime {
		t.Errorf("LastSendTime: got %d, want %d", decoded.LastSendTime, original.LastSendTime)
	}
	if decoded.LastID != original.LastID {
		t.Errorf("LastID: got %s, want %s", decoded.LastID, original.LastID)
	}
}

func TestDecodeInvalidCursor(t *testing.T) {
	tests := []string{
		"",
		"not-base64",
		"@@@invalid@@@",
		"aGVsbG8=", // valid base64 but not valid JSON
	}

	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			decoded, err := decodeCursor(tc)
			if err == nil && decoded != nil {
				t.Error("expected error for invalid cursor")
			}
		})
	}
}

func TestEncodeCursorWithNonObjectID(t *testing.T) {
	cursor := encodeCursor(time.Now(), "not-an-oid")
	if cursor != "" {
		t.Error("expected empty cursor for non-OID id")
	}
}

func TestCursorEdgeCases(t *testing.T) {
	// zero time
	oid := primitive.NewObjectID()
	zeroTime := time.Time{}
	cursor := encodeCursor(zeroTime, oid)
	if cursor == "" {
		t.Fatal("encodeCursor with zero time returned empty")
	}
	decoded, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode zero time cursor: %v", err)
	}
	if decoded.LastSendTime != 0 {
		t.Errorf("expected 0, got %d", decoded.LastSendTime)
	}
	if decoded.LastID != oid.Hex() {
		t.Errorf("LastID mismatch")
	}
}

func TestCursorUniquePerMessage(t *testing.T) {
	// Different send times should produce different cursors
	oid1 := primitive.NewObjectID()
	oid2 := primitive.NewObjectID()
	t1 := time.Now()
	t2 := t1.Add(time.Second)

	c1 := encodeCursor(t1, oid1)
	c2 := encodeCursor(t1, oid2) // same time, different OID
	c3 := encodeCursor(t2, oid1) // different time, same OID

	if c1 == c2 {
		t.Error("same time different OID should produce different cursors")
	}
	if c1 == c3 {
		t.Error("different time same OID should produce different cursors")
	}
}
