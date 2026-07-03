package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// MessageHandler 处理消息发送
// 在消息持久化后，检查 @user_list 中是否有机器人，如有则发布 bot_trigger 事件
type MessageHandler struct {
	botTriggerWriter *kafka.Writer
}

func NewMessageHandler(brokers []string, botTriggerTopic string) *MessageHandler {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    botTriggerTopic,
		Balancer: &kafka.Hash{},
	}

	return &MessageHandler{
		botTriggerWriter: writer,
	}
}

// SendMessage POST /messages
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req struct {
		ClientMsgID   string   `json:"client_msg_id" binding:"required"`
		MsgType       int8     `json:"msg_type"`
		Content       string   `json:"content" binding:"required"`
		ConversationID string  `json:"conversation_id"`
		ConvType      int8     `json:"conv_type"`   // 1=单聊, 2=群聊
		GroupID       *int64   `json:"group_id"`
		AtUserList    []int64  `json:"at_user_list"`
		SenderID      int64    `json:"sender_id"`
		SenderName    string   `json:"sender_name"`
		SenderBotID   *int64   `json:"sender_bot_id"` // 机器人身份发送
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": err.Error()})
		return
	}

	// 此处简化：实际实现中会进行 敏感词校验 → 频控校验 → 权限校验 → MongoDB 持久化
	// 持久化后获取 msgID
	msgID := "msg_" + uuid.New().String()

	// 检查是否需要发布 bot_trigger 事件
	// 条件：at_user_list 非空 且 发送者不是机器人
	if len(req.AtUserList) > 0 && req.SenderBotID == nil {
		h.publishBotTrigger(c, msgID, req)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{"msg_id": msgID},
	})
}

// SendSSEToken POST /messages/sse (供 bot-svc SSE 流式转发)
func (h *MessageHandler) SendSSEToken(c *gin.Context) {
	var req struct {
		BotID int64  `json:"bot_id" binding:"required"`
		MsgID string `json:"msg_id" binding:"required"`
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": err.Error()})
		return
	}

	// 将 SSE token 通过 WebSocket 推送给客户端
	// 简化实现：直接返回成功
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func (h *MessageHandler) publishBotTrigger(c *gin.Context, msgID string, req struct {
	ClientMsgID   string   `json:"client_msg_id"`
	MsgType       int8     `json:"msg_type"`
	Content       string   `json:"content"`
	ConversationID string  `json:"conversation_id"`
	ConvType      int8     `json:"conv_type"`
	GroupID       *int64   `json:"group_id"`
	AtUserList    []int64  `json:"at_user_list"`
	SenderID      int64    `json:"sender_id"`
	SenderName    string   `json:"sender_name"`
	SenderBotID   *int64   `json:"sender_bot_id"`
}) {
	event := map[string]any{
		"event_id":   "evt_" + uuid.New().String(),
		"event_type": "message.mention",
		"timestamp":  strconv.FormatInt(c.GetInt64("timestamp"), 10),
		"message": map[string]any{
			"msg_id":        msgID,
			"client_msg_id": req.ClientMsgID,
			"text":          req.Content,
			"msg_type":      req.MsgType,
			"at_user_ids":   req.AtUserList,
		},
		"sender": map[string]any{
			"user_id":   req.SenderID,
			"user_name": req.SenderName,
		},
		"conversation": map[string]any{
			"conv_id":   req.ConversationID,
			"conv_type": req.ConvType,
			"group_id":  req.GroupID,
		},
		"bot_ids": req.AtUserList, // bot-svc 会做 SINTER 过滤
	}

	data, _ := json.Marshal(event)
	if err := h.botTriggerWriter.WriteMessages(c, kafka.Message{Value: data}); err != nil {
		// 仅记录日志，不阻塞主消息流程
		// 任务说明：bot_trigger 发布是异步非阻塞的
	}
}
