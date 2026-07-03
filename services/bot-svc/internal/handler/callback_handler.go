package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
	"github.com/shulian-paas/im/bot-svc/internal/service"
)

type CallbackHandler struct {
	asyncSvc *service.AsyncWebhookService
	msgClnt  *service.MessageClient
	cache    *repo.BotCache
}

func NewCallbackHandler(asyncSvc *service.AsyncWebhookService, msgClnt *service.MessageClient, cache *repo.BotCache) *CallbackHandler {
	return &CallbackHandler{asyncSvc: asyncSvc, msgClnt: msgClnt, cache: cache}
}

// HandleCallback POST /internal/bot/callback
// 外部系统在异步模式处理完成后回调此接口
func (h *CallbackHandler) HandleCallback(c *gin.Context) {
	var req model.CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": err.Error()})
		return
	}

	state, err := h.asyncSvc.HandleCallback(c.Request.Context(), &req)
	if err != nil {
		log.Warn().Err(err).Str("event_id", req.EventID).Msg("callback validation failed")
		c.JSON(http.StatusNotFound, gin.H{"code": 40004, "msg": "event not found or expired"})
		return
	}

	// 获取机器人配置
	bot, err := h.cache.GetConfig(c.Request.Context(), state.BotID)
	if err != nil {
		log.Err(err).Int64("bot_id", state.BotID).Msg("get bot config for callback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": "bot not found"})
		return
	}
	_ = bot // bot config retained for future use (IP whitelist check, etc.)

	// 发回回复消息（使用 pending 中保存的会话上下文）
	conv := model.ConversationContext{
		ConvID:   state.ConvID,
		ConvType: state.ConvType,
		GroupID:  state.GroupID,
	}
	if err := h.msgClnt.SendReply(state.BotID, req.Reply, req.ReplyType, conv); err != nil {
		log.Err(err).Str("event_id", req.EventID).Msg("send callback reply failed")
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
