package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/rc-svc/internal/model"
	"github.com/shulian-paas/im/rc-svc/internal/service"
)

type SensitiveHandler struct {
	svc *service.RCService
}

func NewSensitiveHandler(svc *service.RCService) *SensitiveHandler {
	return &SensitiveHandler{svc: svc}
}

func (h *SensitiveHandler) RegisterRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/sensitive")
	{
		r.GET("/words", h.ListWords)
		r.POST("/words", h.CreateWord)
		r.PUT("/words/:word_id", h.UpdateWord)
		r.DELETE("/words/:word_id", h.DeleteWord)
		r.POST("/words/batch", h.BatchImport)
		r.POST("/check", h.Check)
		r.POST("/check/text", h.CheckText) // internal check endpoint
	}
}

func getTenantID(c *gin.Context) int64 {
	tid, _ := c.Get("tenant_id")
	if tid, ok := tid.(int64); ok {
		return tid
	}
	return 0
}

func (h *SensitiveHandler) ListWords(c *gin.Context) {
	tenantID := getTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	words, total, err := h.svc.ListWords(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		log.Err(err).Msg("list sensitive words failed")
		InternalError(c, "list words failed")
		return
	}
	Success(c, gin.H{"list": words, "total": total, "page": page, "page_size": pageSize})
}

func (h *SensitiveHandler) CreateWord(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.SensitiveWordCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	if err := h.svc.CreateWord(c.Request.Context(), &req); err != nil {
		log.Err(err).Msg("create sensitive word failed")
		InternalError(c, "create word failed")
		return
	}
	Success(c, gin.H{"word": req.Word})
}

func (h *SensitiveHandler) UpdateWord(c *gin.Context) {
	wordID, err := strconv.ParseInt(c.Param("word_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid word_id")
		return
	}

	var req model.SensitiveWordUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Word != "" {
		updates["word"] = req.Word
	}
	if req.Strategy != 0 {
		updates["strategy"] = req.Strategy
	}
	if req.Replacement != "" {
		updates["replacement"] = req.Replacement
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		BadRequest(c, "no fields to update")
		return
	}

	if err := h.svc.UpdateWord(c.Request.Context(), wordID, updates, getTenantID(c)); err != nil {
		log.Err(err).Msg("update sensitive word failed")
		InternalError(c, "update word failed")
		return
	}
	Success(c, gin.H{"word_id": wordID})
}

func (h *SensitiveHandler) DeleteWord(c *gin.Context) {
	wordID, err := strconv.ParseInt(c.Param("word_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid word_id")
		return
	}

	if err := h.svc.DeleteWord(c.Request.Context(), wordID, getTenantID(c)); err != nil {
		log.Err(err).Msg("delete sensitive word failed")
		InternalError(c, "delete word failed")
		return
	}
	Success(c, gin.H{"word_id": wordID})
}

func (h *SensitiveHandler) BatchImport(c *gin.Context) {
	tenantID := getTenantID(c)

	var req model.SensitiveWordBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	count, err := h.svc.BatchImportWords(c.Request.Context(), &req)
	if err != nil {
		log.Err(err).Msg("batch import failed")
		InternalError(c, "batch import failed")
		return
	}
	Success(c, gin.H{"imported": count})
}

// CheckText POST /api/v1/sensitive/check/text — internal endpoint for message-svc to call
func (h *SensitiveHandler) CheckText(c *gin.Context) {
	start := time.Now()
	tenantID := getTenantID(c)

	var req model.SensitiveCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID

	resp, err := h.svc.CheckSensitive(c.Request.Context(), tenantID, req.Content)
	if err != nil {
		log.Err(err).Msg("sensitive check failed")
		InternalError(c, "check failed")
		return
	}

	log.Debug().Bool("passed", resp.Passed).Str("content_len", strconv.Itoa(len(req.Content))).
		Dur("elapsed", time.Since(start)).Msg("sensitive check")
	Success(c, resp)
}

// Check runs the full validation chain (for internal use)
func (h *SensitiveHandler) Check(c *gin.Context) {
	tenantID := getTenantID(c)

	var req struct {
		Content    string `json:"content"`
		SenderID   int64  `json:"sender_id"`
		SenderType int8   `json:"sender_type"` // 1=user, 2=bot
		FileType   string `json:"file_type"`
		FileSize   int    `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	resp := h.svc.CheckChain(c.Request.Context(), tenantID, req.Content, req.SenderID, req.SenderType, req.FileType, req.FileSize)
	Success(c, resp)
}
