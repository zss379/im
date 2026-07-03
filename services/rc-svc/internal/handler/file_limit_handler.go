package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/rc-svc/internal/model"
)

type FileLimitHandler struct {
	svc interface {
		CheckFileLimit(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error)
		CreateFileLimit(ctx context.Context, req *model.FileLimitCreateReq) error
		ListFileLimits(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error)
		UpdateFileLimit(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error
		DeleteFileLimit(ctx context.Context, configID int64, tenantID int64) error
	}
}

func NewFileLimitHandler(svc interface {
	CheckFileLimit(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error)
	CreateFileLimit(ctx context.Context, req *model.FileLimitCreateReq) error
	ListFileLimits(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error)
	UpdateFileLimit(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error
	DeleteFileLimit(ctx context.Context, configID int64, tenantID int64) error
}) *FileLimitHandler {
	return &FileLimitHandler{svc: svc}
}

func (h *FileLimitHandler) RegisterRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/file-limit")
	{
		r.GET("/configs", h.ListConfigs)
		r.POST("/configs", h.CreateConfig)
		r.PUT("/configs/:config_id", h.UpdateConfig)
		r.DELETE("/configs/:config_id", h.DeleteConfig)
		r.POST("/check", h.Check)
	}
}

func (h *FileLimitHandler) ListConfigs(c *gin.Context) {
	tenantID := getTenantID(c)
	configs, err := h.svc.ListFileLimits(c.Request.Context(), tenantID)
	if err != nil {
		log.Err(err).Msg("list file limits failed")
		InternalError(c, "list configs failed")
		return
	}
	Success(c, gin.H{"list": configs})
}

func (h *FileLimitHandler) CreateConfig(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.FileLimitCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	if err := h.svc.CreateFileLimit(c.Request.Context(), &req); err != nil {
		log.Err(err).Msg("create file limit failed")
		InternalError(c, "create config failed")
		return
	}
	Success(c, gin.H{"file_type": req.FileType})
}

func (h *FileLimitHandler) UpdateConfig(c *gin.Context) {
	configID, err := strconv.ParseInt(c.Param("config_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid config_id")
		return
	}

	var body struct {
		MaxSizeMB         *int    `json:"max_size_mb"`
		AllowedExtensions *string `json:"allowed_extensions"`
		Status            *int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if body.MaxSizeMB != nil {
		updates["max_size_mb"] = *body.MaxSizeMB
	}
	if body.AllowedExtensions != nil {
		updates["allowed_extensions"] = *body.AllowedExtensions
	}
	if body.Status != nil {
		updates["status"] = *body.Status
	}

	if err := h.svc.UpdateFileLimit(c.Request.Context(), configID, updates, getTenantID(c)); err != nil {
		log.Err(err).Msg("update file limit failed")
		InternalError(c, "update config failed")
		return
	}
	Success(c, gin.H{"config_id": configID})
}

func (h *FileLimitHandler) DeleteConfig(c *gin.Context) {
	configID, err := strconv.ParseInt(c.Param("config_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid config_id")
		return
	}
	if err := h.svc.DeleteFileLimit(c.Request.Context(), configID, getTenantID(c)); err != nil {
		log.Err(err).Msg("delete file limit failed")
		InternalError(c, "delete config failed")
		return
	}
	Success(c, gin.H{"config_id": configID})
}

func (h *FileLimitHandler) Check(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.FileLimitCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	resp, err := h.svc.CheckFileLimit(c.Request.Context(), tenantID, req.FileType, req.FileSize)
	if err != nil {
		log.Err(err).Msg("file limit check failed")
		InternalError(c, "check failed")
		return
	}
	Success(c, resp)
}
