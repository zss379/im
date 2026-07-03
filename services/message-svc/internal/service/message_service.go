package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/message-svc/internal/model"
	"github.com/shulian-paas/im/message-svc/internal/mq"
)

// MessageService 消息业务逻辑
type MessageService struct {
	msgRepo    messageRepository
	cache      messageCache
	producer   messageProducer
	botMsgRate int // bot 每秒最大消息数
}

func NewMessageService(msgRepo messageRepository, cache messageCache, producer messageProducer, botMsgRate int) *MessageService {
	return &MessageService{
		msgRepo:    msgRepo,
		cache:      cache,
		producer:   producer,
		botMsgRate: botMsgRate,
	}
}

// SendMessage 发送消息完整链路
func (s *MessageService) SendMessage(ctx context.Context, req *model.SendMessageReq, tenantID int64) (*model.SendMessageResp, error) {
	// 1. 幂等去重
	if req.ClientMsgID != "" {
		dup, err := s.cache.TryDedup(ctx, req.ClientMsgID)
		if err != nil {
			log.Warn().Err(err).Msg("dedup check failed, continuing")
		} else if dup {
			// 重复请求，返回已有消息 ID
			existing, err := s.msgRepo.FindByClientMsgID(ctx, req.ConversationID, req.ClientMsgID)
			if err == nil && existing != nil {
				return &model.SendMessageResp{
					MsgID:    existing.MsgID,
					SendTime: existing.SendTime.Format(time.RFC3339),
					Status:   existing.Status,
				}, nil
			}
		}
	}

	// 2. 生成 msg_id
	msgID := "msg_" + uuid.New().String()

	// 3. 构建消息文档
	msg := &model.Message{
		MsgID:          msgID,
		TenantID:       tenantID,
		ConversationID: req.ConversationID,
		ConvType:       req.ConvType,
		SenderID:       req.SenderID,
		SenderName:     req.SenderName,
		SenderAvatar:   req.SenderAvatar,
		SenderBotID:    req.SenderBotID,
		MsgType:        req.MsgType,
		Content:        req.Content,
		ClientMsgID:    req.ClientMsgID,
		AtUserList:     req.AtUserList,
		SendTime:       time.Now(),
		Status:         model.MsgStatusSent,
	}

	// 4. MongoDB 持久化
	if err := s.msgRepo.Insert(ctx, msg); err != nil {
		// 回滚幂等 key
		if req.ClientMsgID != "" {
			_ = s.cache.ReleaseDedup(ctx, req.ClientMsgID)
		}
		return nil, fmt.Errorf("persist message: %w", err)
	}

	// 5. 发送 Kafka 推送事件
	pushEvent := &mq.MessagePushEvent{
		MsgID:          msgID,
		ConversationID: req.ConversationID,
		SenderID:       req.SenderID,
		MsgType:        req.MsgType,
		SendTime:       msg.SendTime.UnixMilli(),
		TenantID:       tenantID,
	}
	contentBytes, _ := json.Marshal(req.Content)
	pushEvent.Content = string(contentBytes)

	if err := s.producer.PublishMessageNew(ctx, pushEvent); err != nil {
		log.Warn().Err(err).Str("msg_id", msgID).Msg("publish message:new failed")
	}

	// 6. @机器人检测并发布 bot_trigger 事件
	if len(req.AtUserList) > 0 && req.SenderBotID == nil {
		if err := s.producer.PublishBotTrigger(ctx, msgID, tenantID,
			fmt.Sprintf("%d", req.ConversationID), req.ConvType, req.GroupID,
			req.SenderID, req.SenderName, getTextContent(req.Content), req.MsgType, req.AtUserList); err != nil {
			log.Warn().Err(err).Str("msg_id", msgID).Msg("publish bot_trigger failed")
		}
	}

	return &model.SendMessageResp{
		MsgID:    msgID,
		SendTime: msg.SendTime.Format(time.RFC3339),
		Status:   model.MsgStatusSent,
	}, nil
}

// PullMessages 拉取历史消息（游标翻页）
func (s *MessageService) PullMessages(ctx context.Context, conversationID int64, cursor string, limit int) (*model.PullMessagesResp, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	messages, nextCursor, err := s.msgRepo.FindByConversation(ctx, conversationID, cursor, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("pull messages: %w", err)
	}

	resp := &model.PullMessagesResp{
		List:    make([]model.MessageResp, 0, len(messages)),
		Cursor:  nextCursor,
		HasMore: nextCursor != "",
	}

	for _, m := range messages {
		resp.List = append(resp.List, toMessageResp(m))
	}

	return resp, nil
}

// RecallMessage 撤回消息
func (s *MessageService) RecallMessage(ctx context.Context, msgID string, senderID int64, tenantID int64) error {
	msg, err := s.msgRepo.FindByMsgID(ctx, msgID)
	if err != nil {
		return fmt.Errorf("find message: %w", err)
	}
	if msg == nil {
		return fmt.Errorf("message not found")
	}

	// 仅发送者可撤回
	if msg.SenderID != senderID {
		return fmt.Errorf("no permission: not the sender")
	}

	// 2 分钟超时检查
	if time.Since(msg.SendTime) > 2*time.Minute {
		return fmt.Errorf("recall timeout: exceeds 2 minutes")
	}

	now := time.Now()
	if err := s.msgRepo.UpdateStatus(ctx, msgID, model.MsgStatusRecalled, &now); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// 发布撤回事件
	s.producer.PublishMessageRecalled(ctx, &mq.MessagePushEvent{
		MsgID:          msgID,
		ConversationID: msg.ConversationID,
		SenderID:       senderID,
		SendTime:       now.UnixMilli(),
		TenantID:       tenantID,
	})

	return nil
}

// ForwardMessages 转发消息
func (s *MessageService) ForwardMessages(ctx context.Context, req *model.ForwardReq, tenantID int64) ([]string, error) {
	var msgIDs []string

	if req.ForwardType == 1 {
		// 逐条转发
		for _, mid := range req.MsgIDs {
			original, err := s.msgRepo.FindByMsgID(ctx, mid)
			if err != nil || original == nil {
				continue
			}
			sendReq := &model.SendMessageReq{
				ConversationID: req.TargetID,
				ConvType:       req.TargetType,
				MsgType:        original.MsgType,
				Content:        original.Content,
				SenderID:       req.SenderID,
				SenderName:     req.SenderName,
			}
			resp, err := s.SendMessage(ctx, sendReq, tenantID)
			if err != nil {
				continue
			}
			msgIDs = append(msgIDs, resp.MsgID)
		}
	} else {
		// 合并转发：创建一条 msg_type=10 的消息
		content := model.MsgContent{
			"title":      "聊天记录",
			"msg_ids":    req.MsgIDs,
			"sender_name": req.SenderName,
		}
		sendReq := &model.SendMessageReq{
			ConversationID: req.TargetID,
			ConvType:       req.TargetType,
			MsgType:        model.MsgTypeMergeForward,
			Content:        content,
			SenderID:       req.SenderID,
			SenderName:     req.SenderName,
		}
		resp, err := s.SendMessage(ctx, sendReq, tenantID)
		if err != nil {
			return nil, fmt.Errorf("forward merge: %w", err)
		}
		msgIDs = append(msgIDs, resp.MsgID)
	}

	return msgIDs, nil
}

// SearchMessages 搜索消息
func (s *MessageService) SearchMessages(ctx context.Context, req *model.SearchReq) ([]model.MessageResp, int64, error) {
	messages, total, err := s.msgRepo.Search(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]model.MessageResp, 0, len(messages))
	for _, m := range messages {
		resp = append(resp, toMessageResp(m))
	}

	return resp, total, nil
}

// MarkRead 标记会话消息已读
func (s *MessageService) MarkRead(ctx context.Context, conversationID int64, msgID string) error {
	return s.cache.MarkRead(ctx, conversationID, msgID)
}

// GetReadReceipt 获取单条消息已读回执
func (s *MessageService) GetReadReceipt(ctx context.Context, conversationID int64, msgID string) (*model.ReadReceiptResp, error) {
	isRead, readAt, err := s.cache.IsRead(ctx, conversationID, msgID)
	if err != nil {
		return nil, err
	}
	resp := &model.ReadReceiptResp{IsRead: isRead}
	if isRead {
		resp.ReadAt = time.UnixMilli(readAt).Format(time.RFC3339)
	}
	return resp, nil
}

// GetReadStatus 获取消息已读详情
func (s *MessageService) GetReadStatus(ctx context.Context, msgID string, conversationID int64) (*model.ReadStatusResp, error) {
	isRead, readAt, err := s.cache.IsRead(ctx, conversationID, msgID)
	if err != nil {
		return nil, err
	}
	resp := &model.ReadStatusResp{
		MsgID:  msgID,
		IsRead: isRead,
	}
	if isRead {
		resp.ReadAt = time.UnixMilli(readAt).Format(time.RFC3339)
	}
	return resp, nil
}

// toMessageResp model.Message → model.MessageResp
func toMessageResp(m model.Message) model.MessageResp {
	return model.MessageResp{
		MsgID:        m.MsgID,
		SenderID:     m.SenderID,
		SenderName:   m.SenderName,
		SenderAvatar: m.SenderAvatar,
		SenderBotID:  m.SenderBotID,
		MsgType:      m.MsgType,
		Content:      m.Content,
		Status:       m.Status,
		SendTime:     m.SendTime,
		RecallTime:   m.RecallTime,
		AtUserList:   m.AtUserList,
	}
}

// getTextContent 从 MsgContent 中提取文本内容
func getTextContent(content model.MsgContent) string {
	if content == nil {
		return ""
	}
	if text, ok := content["text"]; ok {
		if s, ok := text.(string); ok {
			return s
		}
	}
	return ""
}
