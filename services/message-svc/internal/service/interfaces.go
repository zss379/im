package service

import (
	"context"
	"time"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"github.com/shulian-paas/im/message-svc/internal/mq"
)

// messageRepository defines the interface for message storage operations.
type messageRepository interface {
	Insert(ctx context.Context, msg *model.Message) error
	FindByMsgID(ctx context.Context, msgID string) (*model.Message, error)
	FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error)
	FindByConversation(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error)
	UpdateStatus(ctx context.Context, msgID string, status int8, recallTime *time.Time) error
	Search(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error)
}

// messageCache defines the interface for message cache operations.
type messageCache interface {
	TryDedup(ctx context.Context, clientMsgID string) (bool, error)
	ReleaseDedup(ctx context.Context, clientMsgID string) error
	MarkRead(ctx context.Context, conversationID int64, msgID string) error
	IsRead(ctx context.Context, conversationID int64, msgID string) (bool, int64, error)
}

// rcChecker defines the interface for risk-control preflight check.
type rcChecker interface {
	CheckChain(ctx context.Context, req *CheckChainReq) (*PreflightResult, error)
}

// messageProducer defines the interface for Kafka message publishing.
type messageProducer interface {
	PublishMessageNew(ctx context.Context, event *mq.MessagePushEvent) error
	PublishMessageRecalled(ctx context.Context, event *mq.MessagePushEvent) error
	PublishBotTrigger(ctx context.Context, msgID string, tenantID int64, convID string, convType int8, groupID *int64, senderID int64, senderName string, content string, msgType int8, atUserIDs []int64) error
	PublishBlockedMessage(ctx context.Context, event *mq.BlockedMessageEvent) error
}
