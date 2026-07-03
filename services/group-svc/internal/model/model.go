package model

import "time"

// Group roles
const (
	RoleOwner  = 3
	RoleAdmin  = 2
	RoleMember = 1
)

// Group status
const (
	GroupStatusActive   = 1
	GroupStatusDismissed = 0
)

// Join request status
const (
	JoinPending  = 0
	JoinApproved = 1
	JoinRejected = 2
)

// Verification modes
const (
	VerifyOpen            = "open"
	VerifyApprovalRequired = "approval_required"
)

// Capacity limits
const (
	MaxMembersPerGroup     = 2000
	MaxGroupsPerUser       = 500
	MaxCreatedGroupsPerUser = 200
	MaxNoticeLength        = 5000
	MaxBatchMembers        = 10
	AdminRatioLimit        = 10 // max 10% of members can be admins
)

// Group 群组
type Group struct {
	GroupID    int64     `gorm:"primaryKey;autoIncrement" json:"group_id"`
	TenantID   int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	Name       string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Avatar     string    `gorm:"column:avatar;type:varchar(500)" json:"avatar"`
	Notice     string    `gorm:"column:notice;type:varchar(5000)" json:"notice"`
	OwnerID    int64     `gorm:"column:owner_id;not null;index" json:"owner_id"`
	VerifyMode string    `gorm:"column:verify_mode;type:varchar(20);default:'open'" json:"verify_mode"`
	Status     int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Group) TableName() string { return "group_info" }

// GroupMember 群成员
type GroupMember struct {
	MemberID  int64     `gorm:"primaryKey;autoIncrement" json:"member_id"`
	GroupID   int64     `gorm:"column:group_id;not null;uniqueIndex:idx_group_user" json:"group_id"`
	UserID    int64     `gorm:"column:user_id;not null;uniqueIndex:idx_group_user" json:"user_id"`
	Role      int8      `gorm:"column:role;not null;default:1" json:"role"` // 3=owner, 2=admin, 1=member
	MutedUntil *time.Time `gorm:"column:muted_until" json:"muted_until"`
	JoinedAt  time.Time `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
}

func (GroupMember) TableName() string { return "group_member" }

// JoinRequest 入群申请
type JoinRequest struct {
	RequestID int64     `gorm:"primaryKey;autoIncrement" json:"request_id"`
	GroupID   int64     `gorm:"column:group_id;not null;index" json:"group_id"`
	UserID    int64     `gorm:"column:user_id;not null;index" json:"user_id"`
	Status    int8      `gorm:"column:status;not null;default:0" json:"status"` // 0=pending, 1=approved, 2=rejected
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (JoinRequest) TableName() string { return "group_join_request" }

// --- Request / Response types ---

type CreateGroupReq struct {
	TenantID  int64   `json:"tenant_id"`
	Name      string  `json:"name" binding:"required"`
	MemberIDs []int64 `json:"member_ids"`
}

type CreateGroupResp struct {
	GroupID int64  `json:"group_id"`
	Name    string `json:"name"`
}

type UpdateGroupReq struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
	Notice *string `json:"notice"`
}

type GroupListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type GroupListResp struct {
	List     []GroupSummary `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type GroupSummary struct {
	GroupID   int64  `json:"group_id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	OwnerID   int64  `json:"owner_id"`
	MemberCnt int    `json:"member_count"`
	Role      int8   `json:"role"`
}

type TransferOwnerReq struct {
	NewOwnerID int64 `json:"new_owner_id" binding:"required"`
}

type MemberListReq struct {
	GroupID  int64 `form:"-"`
	Page     int   `form:"page"`
	PageSize int   `form:"page_size"`
	Role     *int8 `form:"role"`
}

type MemberListResp struct {
	List     []MemberSummary `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type MemberSummary struct {
	UserID     int64      `json:"user_id"`
	Role       int8       `json:"role"`
	MutedUntil *time.Time `json:"-"`
	Muted      bool       `json:"muted"`
	JoinedAt   string     `json:"joined_at"`
}

type BatchMemberReq struct {
	UserIDs []int64 `json:"user_ids" binding:"required"`
}

type SetRoleReq struct {
	Role int8 `json:"role" binding:"required"` // 2=admin, 1=member
}

type MuteMemberReq struct {
	UserID   int64 `json:"user_id" binding:"required"`
	Duration int   `json:"duration" binding:"required"` // seconds
}

type GlobalMuteReq struct {
	Duration int `json:"duration" binding:"required"` // seconds
}

type MuteCheckResp struct {
	Muted        bool  `json:"muted"`
	RemainingSec int   `json:"remaining_sec,omitempty"`
}

type JoinRequestCreateReq struct {
	GroupID int64 `json:"-"`
	UserID  int64 `json:"-"`
}

type JoinConfigReq struct {
	VerifyMode string `json:"verify_mode" binding:"required"` // open / approval_required
}

type JoinRequestListResp struct {
	List     []JoinRequestSummary `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type JoinRequestSummary struct {
	RequestID int64  `json:"request_id"`
	UserID    int64  `json:"user_id"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}
