package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/audit-svc/internal/model"
	"github.com/shulian-paas/im/audit-svc/internal/service"
)

type AuditHandler struct {
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) RegisterRoutes(rg *gin.RouterGroup) {
	a := rg.Group("/audit")
	{
		a.POST("/admin-logs", h.CreateAdminOpLog)
		a.GET("/admin-logs", h.ListAdminOpLogs)
		a.GET("/admin-logs/:id", h.GetAdminOpLog)

		a.POST("/msg-logs", h.CreateMsgAuditLog)
		a.POST("/msg-logs/batch", h.BatchCreateMsgLogs)
		a.GET("/msg-logs", h.ListMsgAuditLogs)
		a.GET("/msg-logs/:id", h.GetMsgAuditLog)

		a.GET("/stats", h.Stats)
		a.POST("/cleanup", h.Cleanup)
	}
}

func (h *AuditHandler) CreateAdminOpLog(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.CreateAdminOpLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	req.TenantID = tenantID
	// Allow operator to be inferred from token if not set
	if req.OperatorID == 0 {
		req.OperatorID = getUserID(c)
	}
	if err := h.svc.CreateAdminOpLog(c.Request.Context(), &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuditHandler) ListAdminOpLogs(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.AdminOpLogListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	req.TenantID = tenantID

	logs, total, err := h.svc.ListAdminOpLogs(c.Request.Context(), tenantID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, model.AdminOpLogListResp{
		List:     logs,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *AuditHandler) GetAdminOpLog(c *gin.Context) {
	tenantID := getTenantID(c)
	logID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid log id")
		return
	}
	l, err := h.svc.GetAdminOpLog(c.Request.Context(), logID, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if l == nil {
		NotFound(c, "log not found")
		return
	}
	Success(c, l)
}

func (h *AuditHandler) CreateMsgAuditLog(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.CreateMsgAuditLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	req.TenantID = tenantID
	if err := h.svc.CreateMsgAuditLog(c.Request.Context(), &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuditHandler) BatchCreateMsgLogs(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.BatchCreateMsgLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	for i := range req.Logs {
		req.Logs[i].TenantID = tenantID
	}
	if err := h.svc.BatchCreateMsgAuditLogs(c.Request.Context(), &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuditHandler) ListMsgAuditLogs(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.MsgAuditLogListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	logs, total, err := h.svc.ListMsgAuditLogs(c.Request.Context(), tenantID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, model.MsgAuditLogListResp{
		List:     logs,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *AuditHandler) GetMsgAuditLog(c *gin.Context) {
	tenantID := getTenantID(c)
	logID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid log id")
		return
	}
	l, err := h.svc.GetMsgAuditLog(c.Request.Context(), logID, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if l == nil {
		NotFound(c, "log not found")
		return
	}
	Success(c, l)
}

func (h *AuditHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, stats)
}

func (h *AuditHandler) Cleanup(c *gin.Context) {
	if err := h.svc.Cleanup(c.Request.Context()); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}
