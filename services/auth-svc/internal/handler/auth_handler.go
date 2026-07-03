package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/auth-svc/internal/model"
	"github.com/shulian-paas/im/auth-svc/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/refresh", h.Refresh)

	u := rg.Group("/users")
	{
		u.POST("", h.Create)
		u.GET("/:id", h.Get)
		u.PUT("/:id", h.Update)
		u.PUT("/:id/password", h.ChangePassword)
		u.PUT("/:id/status", h.SetStatus)
		u.GET("/:id/status", h.GetStatus)
		u.POST("/batch", h.BatchGet)
		u.POST("/status/batch", h.BatchStatus)
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}
	Success(c, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	resp, err := h.svc.RefreshToken(c.Request.Context(), req.Token)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}
	Success(c, resp)
}

func (h *AuthHandler) Create(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	u, err := h.svc.CreateUser(c.Request.Context(), tenantID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, u)
}

func (h *AuthHandler) Get(c *gin.Context) {
	tenantID := getTenantID(c)
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	u, err := h.svc.GetUser(c.Request.Context(), userID, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if u == nil {
		NotFound(c, "user not found")
		return
	}
	Success(c, u)
}

func (h *AuthHandler) Update(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.UpdateUser(c.Request.Context(), userID, &req); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	var req model.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuthHandler) SetStatus(c *gin.Context) {
	tenantID := getTenantID(c)
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	var req model.SetStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), userID, tenantID, req.Status); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *AuthHandler) GetStatus(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	st, err := h.svc.GetStatus(c.Request.Context(), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, st)
}

func (h *AuthHandler) BatchGet(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.BatchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	users, err := h.svc.BatchGetUsers(c.Request.Context(), req.UserIDs, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, users)
}

func (h *AuthHandler) BatchStatus(c *gin.Context) {
	var req model.BatchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	resp, err := h.svc.BatchGetStatus(c.Request.Context(), req.UserIDs)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, resp)
}
