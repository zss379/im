package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/session-svc/internal/model"
	"github.com/shulian-paas/im/session-svc/internal/service"
)

type SessionHandler struct {
	svc *service.SessionService
}

func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/sessions")
	{
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.DELETE("/:id", h.Delete)
		g.PUT("/:id/pin", h.Pin)
		g.PUT("/:id/mute", h.Mute)
		g.PUT("/:id/read", h.MarkRead)
		g.PUT("/:id/unread", h.MarkUnread)
		g.PUT("/unread/batch", h.BatchUpdateUnread)
	}
}

func (h *SessionHandler) Create(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)

	var req model.CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	sess, err := h.svc.CreateSession(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, sess)
}

func (h *SessionHandler) List(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)

	var req model.SessionListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	sessions, total, err := h.svc.ListSessions(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, model.SessionListResp{
		List:     sessions,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *SessionHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	sess, err := h.svc.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if sess == nil {
		NotFound(c, "session not found")
		return
	}
	Success(c, sess)
}

func (h *SessionHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	if err := h.svc.DeleteSession(c.Request.Context(), sessionID, userID); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *SessionHandler) Pin(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	var req model.PinSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.svc.PinSession(c.Request.Context(), sessionID, userID, req.Pinned); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *SessionHandler) Mute(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	var req model.MuteSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.svc.MuteSession(c.Request.Context(), sessionID, userID, req.Muted); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *SessionHandler) MarkRead(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), sessionID, userID); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *SessionHandler) MarkUnread(c *gin.Context) {
	userID := getUserID(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid session id")
		return
	}

	if err := h.svc.MarkUnread(c.Request.Context(), sessionID, userID); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *SessionHandler) BatchUpdateUnread(c *gin.Context) {
	userID := getUserID(c)

	var req model.BatchUnreadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.svc.BatchUpdateUnread(c.Request.Context(), userID, &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}
