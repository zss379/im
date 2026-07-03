package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shulian-paas/im/bot-svc/internal/model"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
)

type BotHandler struct {
	botRepo *repo.BotRepo
	cache   *repo.BotCache
}

func NewBotHandler(botRepo *repo.BotRepo, cache *repo.BotCache) *BotHandler {
	return &BotHandler{botRepo: botRepo, cache: cache}
}

// ListBots GET /bots
func (h *BotHandler) ListBots(c *gin.Context) {
	tenantID, _ := strconv.ParseInt(c.GetHeader("X-Tenant-ID"), 10, 64)
	botTypeStr := c.Query("bot_type")
	var botType *int8
	if botTypeStr != "" {
		bt, _ := strconv.ParseInt(botTypeStr, 10, 64)
		t := int8(bt)
		botType = &t
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	bots, total, err := h.botRepo.ListByTenant(tenantID, botType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"list":      bots,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CreateBot POST /bots
func (h *BotHandler) CreateBot(c *gin.Context) {
	tenantID, _ := strconv.ParseInt(c.GetHeader("X-Tenant-ID"), 10, 64)
	userID, _ := strconv.ParseInt(c.GetHeader("X-User-ID"), 10, 64)

	var req struct {
		BotName      string              `json:"bot_name" binding:"required"`
		AvatarURL    *string             `json:"avatar_url"`
		Description  *string             `json:"description"`
		WebhookURL   *string             `json:"webhook_url"`
		ResponseMode model.ResponseMode  `json:"response_mode"`
		CallbackURL  *string             `json:"callback_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": err.Error()})
		return
	}

	bot := &model.Bot{
		TenantID:     tenantID,
		BotType:      model.BotTypeCustom,
		BotName:      req.BotName,
		AvatarURL:    req.AvatarURL,
		Description:  req.Description,
		WebhookURL:   req.WebhookURL,
		ResponseMode: req.ResponseMode,
		CallbackURL:  req.CallbackURL,
		Status:       1,
		CreatorID:    &userID,
	}

	if err := h.botRepo.Create(bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": err.Error()})
		return
	}

	// 同步到 Redis 缓存 (task 2.2/2.3)
	_ = h.cache.AddUserID(c, bot.BotID)
	_ = h.cache.SetConfig(c, bot)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": bot,
	})
}

// ToggleBot POST /bots/:bot_id/toggle
func (h *BotHandler) ToggleBot(c *gin.Context) {
	botID, _ := strconv.ParseInt(c.Param("bot_id"), 10, 64)

	var req struct {
		Status int8 `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": err.Error()})
		return
	}

	if err := h.botRepo.ToggleStatus(botID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": err.Error()})
		return
	}

	// 更新缓存
	bot, _ := h.botRepo.GetByID(botID)
	if bot != nil {
		_ = h.cache.SetConfig(c, bot)
	}
	if req.Status == 0 {
		_ = h.cache.RemoveUserID(c, botID)
	} else {
		_ = h.cache.AddUserID(c, botID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

// DeleteBot DELETE /bots/:bot_id
func (h *BotHandler) DeleteBot(c *gin.Context) {
	botID, _ := strconv.ParseInt(c.Param("bot_id"), 10, 64)

	if err := h.botRepo.Delete(botID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": err.Error()})
		return
	}

	// 清理缓存
	_ = h.cache.RemoveUserID(c, botID)
	_ = h.cache.DeleteConfig(c, botID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
