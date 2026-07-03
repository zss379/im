package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/message-svc/internal/metrics"
	"github.com/shulian-paas/im/message-svc/internal/model"
	"github.com/shulian-paas/im/message-svc/internal/repo"
	"github.com/shulian-paas/im/message-svc/internal/service"
)

type MessageHandler struct {
	svc   *service.MessageService
	cache *repo.MessageCache
}

func NewMessageHandler(svc *service.MessageService, cache *repo.MessageCache) *MessageHandler {
	return &MessageHandler{svc: svc, cache: cache}
}

func (h *MessageHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// 静态路由（非 :param）需注册在参数路由之前
	rg.POST("/messages/sse", h.SendSSEToken)
	rg.POST("/messages/forward", h.ForwardMessages)
	rg.GET("/messages/search", h.SearchMessages)

	// 参数路由
	rg.POST("/messages", h.SendMessage)
	rg.GET("/messages", h.PullMessages)
	rg.POST("/messages/:msg_id/recall", h.RecallMessage)
	rg.GET("/messages/:msg_id/read-receipt", h.GetReadReceipt)
	rg.GET("/messages/:msg_id/read-status", h.GetReadStatus)

	// 会话级
	rg.PUT("/conversations/:conversation_id/read", h.MarkRead)
}

// SendMessage POST /api/v1/messages
func (h *MessageHandler) SendMessage(c *gin.Context) {
	start := time.Now()
	tenantID := c.GetInt64("tenant_id")

	var req model.SendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 从 token 填充发送者信息（如果请求未显式提供）
	senderID, _ := c.Get("user_id")
	if req.SenderID == 0 {
		req.SenderID, _ = senderID.(int64)
	}

	resp, err := h.svc.SendMessage(c.Request.Context(), &req, tenantID)
	if err != nil {
		var blocked *service.ErrBlocked
		var rateLimited *service.ErrRateLimited
		switch {
		case errors.As(err, &blocked):
			Forbidden(c, blocked.Reason)
			return
		case errors.As(err, &rateLimited):
			c.Header("Retry-After", strconv.Itoa(rateLimited.RetryAfter))
			TooManyRequests(c, "rate limit exceeded")
			return
		default:
			log.Err(err).Str("client_msg_id", req.ClientMsgID).Msg("send message failed")
			metrics.IncMessageSend(strconv.Itoa(int(req.MsgType)), "fail")
			InternalError(c, "send message failed")
			return
		}
	}

	metrics.IncMessageSend(strconv.Itoa(int(req.MsgType)), "success")
	metrics.ObserveHTTP(c.Request.Method, "/messages", 200, time.Since(start).Seconds())
	Success(c, resp)
}

// PullMessages GET /api/v1/messages?conversation_id=&cursor=&limit=
func (h *MessageHandler) PullMessages(c *gin.Context) {
	start := time.Now()

	conversationID, err := strconv.ParseInt(c.Query("conversation_id"), 10, 64)
	if err != nil || conversationID <= 0 {
		BadRequest(c, "invalid conversation_id")
		return
	}

	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.Query("limit"))

	resp, err := h.svc.PullMessages(c.Request.Context(), conversationID, cursor, limit)
	if err != nil {
		log.Err(err).Int64("conversation_id", conversationID).Msg("pull messages failed")
		InternalError(c, "pull messages failed")
		return
	}

	metrics.ObserveHTTP(c.Request.Method, "/messages", 200, time.Since(start).Seconds())
	Success(c, resp)
}

// RecallMessage POST /api/v1/messages/:msg_id/recall
func (h *MessageHandler) RecallMessage(c *gin.Context) {
	start := time.Now()
	tenantID := c.GetInt64("tenant_id")

	msgID := c.Param("msg_id")
	senderID, _ := c.Get("user_id")

	var req model.RecallReq
	if err := c.ShouldBindJSON(&req); err == nil && req.SenderID > 0 {
		senderID = req.SenderID
	}

	err := h.svc.RecallMessage(c.Request.Context(), msgID, senderID.(int64), tenantID)
	if err != nil {
		log.Err(err).Str("msg_id", msgID).Msg("recall message failed")
		switch {
		case err.Error() == "message not found":
			NotFound(c, err.Error())
		case err.Error() == "no permission: not the sender" || err.Error() == "recall timeout: exceeds 2 minutes":
			BadRequest(c, err.Error())
		default:
			InternalError(c, err.Error())
		}
		return
	}

	metrics.IncMessageRecall()
	metrics.ObserveHTTP(c.Request.Method, "/messages/:msg_id/recall", 200, time.Since(start).Seconds())
	Success(c, gin.H{"msg_id": msgID, "status": model.MsgStatusRecalled})
}

// ForwardMessages POST /api/v1/messages/forward
func (h *MessageHandler) ForwardMessages(c *gin.Context) {
	start := time.Now()
	tenantID := c.GetInt64("tenant_id")

	var req model.ForwardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	senderID, _ := c.Get("user_id")
	if req.SenderID == 0 {
		req.SenderID, _ = senderID.(int64)
	}

	msgIDs, err := h.svc.ForwardMessages(c.Request.Context(), &req, tenantID)
	if err != nil {
		log.Err(err).Msg("forward messages failed")
		InternalError(c, "forward messages failed")
		return
	}

	metrics.IncMessageForward()
	metrics.ObserveHTTP(c.Request.Method, "/messages/forward", 200, time.Since(start).Seconds())
	Success(c, gin.H{"msg_ids": msgIDs})
}

// SearchMessages GET /api/v1/messages/search?q=&conversation_id=&sender_id=&msg_type=&start_time=&end_time=&page=&page_size=
func (h *MessageHandler) SearchMessages(c *gin.Context) {
	start := time.Now()
	tenantID := c.GetInt64("tenant_id")

	var req model.SearchReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 若未指定 conversation_id，限制在当前租户范围（由 service 层处理）
	_ = tenantID

	messages, total, err := h.svc.SearchMessages(c.Request.Context(), &req)
	if err != nil {
		log.Err(err).Msg("search messages failed")
		InternalError(c, "search messages failed")
		return
	}

	metrics.IncMessageSearch()
	metrics.ObserveHTTP(c.Request.Method, "/messages/search", 200, time.Since(start).Seconds())
	Success(c, gin.H{
		"list":  messages,
		"total": total,
	})
}

// MarkRead PUT /api/v1/conversations/:conversation_id/read
func (h *MessageHandler) MarkRead(c *gin.Context) {
	start := time.Now()

	conversationID, err := strconv.ParseInt(c.Param("conversation_id"), 10, 64)
	if err != nil || conversationID <= 0 {
		BadRequest(c, "invalid conversation_id")
		return
	}

	var body struct {
		MsgID string `json:"msg_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), conversationID, body.MsgID); err != nil {
		log.Err(err).Int64("conversation_id", conversationID).Str("msg_id", body.MsgID).Msg("mark read failed")
		InternalError(c, "mark read failed")
		return
	}

	metrics.IncReadOp("mark_read")
	metrics.ObserveHTTP(c.Request.Method, "/conversations/:conversation_id/read", 200, time.Since(start).Seconds())
	Success(c, gin.H{"conversation_id": conversationID, "msg_id": body.MsgID})
}

// GetReadReceipt GET /api/v1/messages/:msg_id/read-receipt?conversation_id=
func (h *MessageHandler) GetReadReceipt(c *gin.Context) {
	start := time.Now()

	msgID := c.Param("msg_id")
	conversationID, err := strconv.ParseInt(c.Query("conversation_id"), 10, 64)
	if err != nil || conversationID <= 0 {
		BadRequest(c, "invalid conversation_id")
		return
	}

	resp, err := h.svc.GetReadReceipt(c.Request.Context(), conversationID, msgID)
	if err != nil {
		log.Err(err).Str("msg_id", msgID).Msg("get read receipt failed")
		InternalError(c, "get read receipt failed")
		return
	}

	metrics.IncReadOp("get_receipt")
	metrics.ObserveHTTP(c.Request.Method, "/messages/:msg_id/read-receipt", 200, time.Since(start).Seconds())
	Success(c, resp)
}

// GetReadStatus GET /api/v1/messages/:msg_id/read-status?conversation_id=
func (h *MessageHandler) GetReadStatus(c *gin.Context) {
	start := time.Now()

	msgID := c.Param("msg_id")
	conversationID, err := strconv.ParseInt(c.Query("conversation_id"), 10, 64)
	if err != nil || conversationID <= 0 {
		BadRequest(c, "invalid conversation_id")
		return
	}

	resp, err := h.svc.GetReadStatus(c.Request.Context(), msgID, conversationID)
	if err != nil {
		log.Err(err).Str("msg_id", msgID).Msg("get read status failed")
		InternalError(c, "get read status failed")
		return
	}

	metrics.IncReadOp("get_status")
	metrics.ObserveHTTP(c.Request.Method, "/messages/:msg_id/read-status", 200, time.Since(start).Seconds())
	Success(c, resp)
}

// SendSSEToken POST /api/v1/messages/sse
func (h *MessageHandler) SendSSEToken(c *gin.Context) {
	start := time.Now()

	var req model.SSETokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.cache.StoreSSEToken(c.Request.Context(), req.MsgID, req.Token); err != nil {
		log.Err(err).Str("msg_id", req.MsgID).Int64("bot_id", req.BotID).Msg("store SSE token failed")
		InternalError(c, "store SSE token failed")
		return
	}

	metrics.ObserveHTTP(c.Request.Method, "/messages/sse", 200, time.Since(start).Seconds())
	Success(c, gin.H{"msg_id": req.MsgID})
}
