package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/group-svc/internal/model"
	"github.com/shulian-paas/im/group-svc/internal/service"
)

type GroupHandler struct {
	svc *service.GroupService
}

func NewGroupHandler(svc *service.GroupService) *GroupHandler {
	return &GroupHandler{svc: svc}
}

func (h *GroupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/groups")
	{
		r.POST("", h.Create)
		r.GET("", h.List)
		r.GET("/search", h.Search)
		r.GET("/:group_id", h.Get)
		r.PUT("/:group_id", h.Update)
		r.DELETE("/:group_id", h.Dismiss)
		r.POST("/:group_id/exit", h.Exit)
		r.PUT("/:group_id/transfer", h.Transfer)

		// Members
		r.GET("/:group_id/members", h.ListMembers)
		r.POST("/:group_id/members", h.AddMembers)
		r.DELETE("/:group_id/members", h.RemoveMembers)
		r.PUT("/:group_id/members/:user_id/role", h.SetRole)
		r.GET("/:group_id/members/search", h.SearchMembers)

		// Mute
		r.POST("/:group_id/mute/members", h.MuteMember)
		r.DELETE("/:group_id/mute/members/:user_id", h.UnmuteMember)
		r.POST("/:group_id/mute/global", h.GlobalMute)
		r.DELETE("/:group_id/mute/global", h.RemoveGlobalMute)
		r.GET("/:group_id/mute/check", h.CheckMute)

		// Join requests
		r.GET("/:group_id/join-requests", h.ListJoinRequests)
		r.POST("/:group_id/join-requests", h.RequestJoin)
		r.POST("/:group_id/join-requests/:request_id/approve", h.ApproveJoin)
		r.POST("/:group_id/join-requests/:request_id/reject", h.RejectJoin)
		r.PUT("/:group_id/join-config", h.SetJoinConfig)
	}
}

func (h *GroupHandler) Create(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)
	var req model.CreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.TenantID = tenantID
	resp, err := h.svc.CreateGroup(c.Request.Context(), userID, &req)
	if err != nil {
		log.Err(err).Msg("create group failed")
		InternalError(c, "create group failed")
		return
	}
	Success(c, resp)
}

func (h *GroupHandler) Get(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	group, err := h.svc.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		log.Err(err).Msg("get group failed")
		InternalError(c, "get group failed")
		return
	}
	if group == nil {
		NotFound(c, "group not found")
		return
	}
	Success(c, group)
}

func (h *GroupHandler) List(c *gin.Context) {
	tenantID := getTenantID(c)
	userID := getUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.svc.ListUserGroups(c.Request.Context(), tenantID, userID, page, pageSize)
	if err != nil {
		log.Err(err).Msg("list groups failed")
		InternalError(c, "list groups failed")
		return
	}
	Success(c, resp)
}

func (h *GroupHandler) Search(c *gin.Context) {
	tenantID := getTenantID(c)
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	groups, total, err := h.svc.SearchGroups(c.Request.Context(), tenantID, keyword, page, pageSize)
	if err != nil {
		log.Err(err).Msg("search groups failed")
		InternalError(c, "search failed")
		return
	}
	Success(c, gin.H{"list": groups, "total": total, "page": page, "page_size": pageSize})
}

func (h *GroupHandler) Update(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.UpdateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.UpdateGroup(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("update group failed")
		InternalError(c, "update group failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

func (h *GroupHandler) Dismiss(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var body struct {
		Confirm bool `json:"confirm"`
	}
	c.ShouldBindJSON(&body)
	if err := h.svc.DismissGroup(c.Request.Context(), groupID, userID, body.Confirm); err != nil {
		log.Err(err).Msg("dismiss group failed")
		InternalError(c, "dismiss failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

func (h *GroupHandler) Exit(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.ExitGroup(c.Request.Context(), groupID, userID); err != nil {
		log.Err(err).Msg("exit group failed")
		InternalError(c, "exit failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

func (h *GroupHandler) Transfer(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.TransferOwnerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.TransferOwner(c.Request.Context(), groupID, userID, req.NewOwnerID); err != nil {
		log.Err(err).Msg("transfer owner failed")
		InternalError(c, "transfer failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

// ---- Members ----

func (h *GroupHandler) ListMembers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	var role *int8
	if r := c.Query("role"); r != "" {
		v, _ := strconv.ParseInt(r, 10, 8)
		v8 := int8(v)
		role = &v8
	}
	resp, err := h.svc.ListMembers(c.Request.Context(), groupID, page, pageSize, role)
	if err != nil {
		log.Err(err).Msg("list members failed")
		InternalError(c, "list members failed")
		return
	}
	Success(c, resp)
}

func (h *GroupHandler) AddMembers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.BatchMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.BatchAddMembers(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("add members failed")
		InternalError(c, "add members failed")
		return
	}
	Success(c, gin.H{"added": len(req.UserIDs)})
}

func (h *GroupHandler) RemoveMembers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.BatchMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.BatchRemoveMembers(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("remove members failed")
		InternalError(c, "remove members failed")
		return
	}
	Success(c, gin.H{"removed": len(req.UserIDs)})
}

func (h *GroupHandler) SetRole(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user_id")
		return
	}
	userID := getUserID(c)
	var req model.SetRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.SetMemberRole(c.Request.Context(), groupID, userID, targetUserID, &req); err != nil {
		log.Err(err).Msg("set role failed")
		InternalError(c, "set role failed")
		return
	}
	Success(c, gin.H{"user_id": targetUserID, "role": req.Role})
}

func (h *GroupHandler) SearchMembers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.svc.SearchMembers(c.Request.Context(), groupID, keyword, page, pageSize)
	if err != nil {
		log.Err(err).Msg("search members failed")
		InternalError(c, "search failed")
		return
	}
	Success(c, resp)
}

// ---- Mute ----

func (h *GroupHandler) MuteMember(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.MuteMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.MuteMember(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("mute member failed")
		InternalError(c, "mute failed")
		return
	}
	Success(c, gin.H{"user_id": req.UserID})
}

func (h *GroupHandler) UnmuteMember(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	targetID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid user_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.UnmuteMember(c.Request.Context(), groupID, userID, targetID); err != nil {
		log.Err(err).Msg("unmute member failed")
		InternalError(c, "unmute failed")
		return
	}
	Success(c, gin.H{"user_id": targetID})
}

func (h *GroupHandler) GlobalMute(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.GlobalMuteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.GlobalMute(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("global mute failed")
		InternalError(c, "global mute failed")
		return
	}
	Success(c, gin.H{"duration": req.Duration})
}

func (h *GroupHandler) RemoveGlobalMute(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.RemoveGlobalMute(c.Request.Context(), groupID, userID); err != nil {
		log.Err(err).Msg("remove global mute failed")
		InternalError(c, "remove global mute failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

func (h *GroupHandler) CheckMute(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		userID = getUserID(c)
	}
	resp, err := h.svc.CheckMute(c.Request.Context(), groupID, userID)
	if err != nil {
		log.Err(err).Msg("check mute failed")
		InternalError(c, "check mute failed")
		return
	}
	Success(c, resp)
}

// ---- Join Requests ----

func (h *GroupHandler) RequestJoin(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.RequestJoin(c.Request.Context(), groupID, userID); err != nil {
		log.Err(err).Msg("join request failed")
		InternalError(c, "join request failed")
		return
	}
	Success(c, gin.H{"group_id": groupID})
}

func (h *GroupHandler) ApproveJoin(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	requestID, err := strconv.ParseInt(c.Param("request_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid request_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.ApproveJoinRequest(c.Request.Context(), groupID, requestID, userID); err != nil {
		log.Err(err).Msg("approve join failed")
		InternalError(c, "approve failed")
		return
	}
	Success(c, gin.H{"request_id": requestID})
}

func (h *GroupHandler) RejectJoin(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	requestID, err := strconv.ParseInt(c.Param("request_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid request_id")
		return
	}
	userID := getUserID(c)
	if err := h.svc.RejectJoinRequest(c.Request.Context(), groupID, requestID, userID); err != nil {
		log.Err(err).Msg("reject join failed")
		InternalError(c, "reject failed")
		return
	}
	Success(c, gin.H{"request_id": requestID})
}

func (h *GroupHandler) ListJoinRequests(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.svc.ListJoinRequests(c.Request.Context(), groupID, page, pageSize)
	if err != nil {
		log.Err(err).Msg("list join requests failed")
		InternalError(c, "list join requests failed")
		return
	}
	Success(c, resp)
}

func (h *GroupHandler) SetJoinConfig(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid group_id")
		return
	}
	userID := getUserID(c)
	var req model.JoinConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.svc.SetJoinConfig(c.Request.Context(), groupID, userID, &req); err != nil {
		log.Err(err).Msg("set join config failed")
		InternalError(c, "set join config failed")
		return
	}
	Success(c, gin.H{"verify_mode": req.VerifyMode})
}
