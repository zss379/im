package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/rc-svc/internal/model"
)

type RateLimitHandler struct {
	svc interface {
		CheckRateLimit(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error)
		CreateRateLimitRule(ctx context.Context, req *model.RateLimitRuleCreateReq) error
		ListRateLimitRules(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error)
		UpdateRateLimitRule(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error
		DeleteRateLimitRule(ctx context.Context, ruleID int64, tenantID int64) error
	}
}

func NewRateLimitHandler(svc interface {
	CheckRateLimit(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error)
	CreateRateLimitRule(ctx context.Context, req *model.RateLimitRuleCreateReq) error
	ListRateLimitRules(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error)
	UpdateRateLimitRule(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error
	DeleteRateLimitRule(ctx context.Context, ruleID int64, tenantID int64) error
}) *RateLimitHandler {
	return &RateLimitHandler{svc: svc}
}

func (h *RateLimitHandler) RegisterRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/rate-limit")
	{
		r.GET("/rules", h.ListRules)
		r.POST("/rules", h.CreateRule)
		r.PUT("/rules/:rule_id", h.UpdateRule)
		r.DELETE("/rules/:rule_id", h.DeleteRule)
		r.POST("/check", h.Check)
	}
}

func (h *RateLimitHandler) ListRules(c *gin.Context) {
	tenantID := getTenantID(c)
	rules, err := h.svc.ListRateLimitRules(c.Request.Context(), tenantID)
	if err != nil {
		log.Err(err).Msg("list rate limit rules failed")
		InternalError(c, "list rules failed")
		return
	}
	Success(c, gin.H{"list": rules})
}

func (h *RateLimitHandler) CreateRule(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.RateLimitRuleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	if err := h.svc.CreateRateLimitRule(c.Request.Context(), &req); err != nil {
		log.Err(err).Msg("create rate limit rule failed")
		InternalError(c, "create rule failed")
		return
	}
	Success(c, gin.H{"target_type": req.TargetType})
}

func (h *RateLimitHandler) UpdateRule(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("rule_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid rule_id")
		return
	}

	var body struct {
		MaxCount          *int  `json:"max_count"`
		TimeWindowSeconds *int  `json:"time_window_seconds"`
		Action            *int8 `json:"action"`
		Status            *int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if body.MaxCount != nil {
		updates["max_count"] = *body.MaxCount
	}
	if body.TimeWindowSeconds != nil {
		updates["time_window_seconds"] = *body.TimeWindowSeconds
	}
	if body.Action != nil {
		updates["action"] = *body.Action
	}
	if body.Status != nil {
		updates["status"] = *body.Status
	}

	if err := h.svc.UpdateRateLimitRule(c.Request.Context(), ruleID, updates, getTenantID(c)); err != nil {
		log.Err(err).Msg("update rate limit rule failed")
		InternalError(c, "update rule failed")
		return
	}
	Success(c, gin.H{"rule_id": ruleID})
}

func (h *RateLimitHandler) DeleteRule(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("rule_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid rule_id")
		return
	}
	if err := h.svc.DeleteRateLimitRule(c.Request.Context(), ruleID, getTenantID(c)); err != nil {
		log.Err(err).Msg("delete rate limit rule failed")
		InternalError(c, "delete rule failed")
		return
	}
	Success(c, gin.H{"rule_id": ruleID})
}

func (h *RateLimitHandler) Check(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.RateLimitCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	resp, err := h.svc.CheckRateLimit(c.Request.Context(), tenantID, req.TargetID, req.TargetType)
	if err != nil {
		log.Err(err).Msg("rate limit check failed")
		InternalError(c, "check failed")
		return
	}
	Success(c, resp)
}
