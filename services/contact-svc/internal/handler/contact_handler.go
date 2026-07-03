package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/shulian-paas/im/contact-svc/internal/model"
	"github.com/shulian-paas/im/contact-svc/internal/service"
)

type ContactHandler struct {
	svc *service.ContactService
}

func NewContactHandler(svc *service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

func (h *ContactHandler) RegisterRoutes(rg *gin.RouterGroup) {
	d := rg.Group("/contacts")
	{
		d.GET("/departments/tree", h.DeptTree)
		d.GET("/departments/:id", h.DeptDetail)
		d.GET("/departments/:id/members", h.DeptMembers)
		d.POST("/departments", h.CreateDept)
		d.PUT("/departments/:id", h.UpdateDept)
		d.DELETE("/departments/:id", h.DeleteDept)

		d.GET("/members/search", h.SearchMembers)
		d.GET("/members/:id", h.MemberDetail)
		d.GET("/members/:id/departments", h.MemberDepts)
		d.PUT("/members/:id", h.UpdateMember)

		d.POST("/sync", h.Sync)
	}
}

func (h *ContactHandler) DeptTree(c *gin.Context) {
	tenantID := getTenantID(c)
	tree, err := h.svc.GetDeptTree(c.Request.Context(), tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, tree)
}

func (h *ContactHandler) DeptDetail(c *gin.Context) {
	deptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid department id")
		return
	}
	d, err := h.svc.GetDeptDetail(c.Request.Context(), deptID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if d == nil {
		NotFound(c, "department not found")
		return
	}
	Success(c, d)
}

func (h *ContactHandler) DeptMembers(c *gin.Context) {
	deptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid department id")
		return
	}
	var req model.MemberListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	req.DeptID = deptID

	members, total, err := h.svc.GetDeptMembers(c.Request.Context(), deptID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, model.MemberListResp{
		List:     members,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *ContactHandler) CreateDept(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.CreateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	dept, err := h.svc.CreateDept(c.Request.Context(), tenantID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, dept)
}

func (h *ContactHandler) UpdateDept(c *gin.Context) {
	deptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid department id")
		return
	}
	var req model.UpdateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.UpdateDept(c.Request.Context(), deptID, &req); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *ContactHandler) DeleteDept(c *gin.Context) {
	deptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid department id")
		return
	}
	if err := h.svc.DeleteDept(c.Request.Context(), deptID); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *ContactHandler) SearchMembers(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.MemberSearchReq
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	members, total, err := h.svc.SearchMembers(c.Request.Context(), tenantID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, model.MemberListResp{
		List:     members,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *ContactHandler) MemberDetail(c *gin.Context) {
	tenantID := getTenantID(c)
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	m, err := h.svc.GetMemberDetail(c.Request.Context(), userID, tenantID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if m == nil {
		NotFound(c, "member not found")
		return
	}
	Success(c, m)
}

func (h *ContactHandler) MemberDepts(c *gin.Context) {
	tenantID := getTenantID(c)
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	resp, err := h.svc.GetUserDepts(c.Request.Context(), userID, tenantID)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, resp)
}

func (h *ContactHandler) UpdateMember(c *gin.Context) {
	tenantID := getTenantID(c)
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user id")
		return
	}
	var req model.UpdateMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.UpdateMember(c.Request.Context(), userID, tenantID, &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

func (h *ContactHandler) Sync(c *gin.Context) {
	tenantID := getTenantID(c)
	var req model.SyncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.Sync(c.Request.Context(), tenantID, &req); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}
