package handler

import (
	"bytes"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/file-svc/internal/model"
	"github.com/shulian-paas/im/file-svc/internal/service"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	f := rg.Group("/files")
	{
		f.POST("/upload", h.Upload)
		f.POST("/uploads", h.InitMultipart)
		f.PUT("/uploads/:id/parts/:part", h.UploadPart)
		f.POST("/uploads/:id/complete", h.CompleteMultipart)
		f.GET("", h.List)
		f.GET("/:id", h.Get)
		f.GET("/:id/download", h.Download)
		f.DELETE("/:id", h.Delete)
	}
}

func (h *FileHandler) Upload(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	resp, err := h.svc.UploadFile(c.Request.Context(), tenantID, userID, header)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, resp)
}

func (h *FileHandler) InitMultipart(c *gin.Context) {
	var req model.InitMultipartUploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	resp, err := h.svc.InitMultipartUpload(c.Request.Context(), &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, resp)
}

func (h *FileHandler) UploadPart(c *gin.Context) {
	objectKey := c.Param("id")
	uploadID := c.Query("upload_id")
	partNumberStr := c.Param("part")

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		BadRequest(c, "invalid part number")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "file part is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		BadRequest(c, "read part failed")
		return
	}

	info, err := h.svc.UploadPart(c.Request.Context(), objectKey, uploadID, partNumber, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, gin.H{"part_number": partNumber, "etag": info.ETag})
}

func (h *FileHandler) CompleteMultipart(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)

	var req model.CompleteMultipartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	resp, err := h.svc.CompleteMultipartUpload(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, resp)
}

func (h *FileHandler) Get(c *gin.Context) {
	tenantID := getTenantID(c)
	fileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid file id")
		return
	}
	f, err := h.svc.GetFile(c.Request.Context(), fileID, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if f == nil {
		NotFound(c, "file not found")
		return
	}
	Success(c, f)
}

func (h *FileHandler) Download(c *gin.Context) {
	tenantID := getTenantID(c)
	fileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid file id")
		return
	}
	url, err := h.svc.GetDownloadURL(c.Request.Context(), fileID, tenantID)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}
	c.Redirect(302, url)
}

func (h *FileHandler) Delete(c *gin.Context) {
	tenantID := getTenantID(c)
	fileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid file id")
		return
	}
	if err := h.svc.DeleteFile(c.Request.Context(), fileID, tenantID); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *FileHandler) List(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.FileListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	files, total, err := h.svc.ListFiles(c.Request.Context(), tenantID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, model.FileListResp{
		List:     files,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}
