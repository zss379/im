package model

import "time"

// Contact profile status
const (
	ContactStatusActive   = 1
	ContactStatusResigned = 0
)

// Department 部门
type Department struct {
	DeptID      int64     `gorm:"primaryKey;autoIncrement" json:"dept_id"`
	TenantID    int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	Name        string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	ParentID    int64     `gorm:"column:parent_id;default:0;index" json:"parent_id"`
	SortOrder   int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	MemberCount int       `gorm:"column:member_count;default:0" json:"member_count"`
	Status      int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Department) TableName() string { return "contact_department" }

// ContactProfile 联系人档案（HR 同步缓存）
type ContactProfile struct {
	UserID   int64     `gorm:"primaryKey" json:"user_id"`
	TenantID int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	Name     string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	NamePy   string    `gorm:"column:name_py;type:varchar(200)" json:"name_py"`
	Avatar   string    `gorm:"column:avatar;type:varchar(500)" json:"avatar"`
	Phone    string    `gorm:"column:phone;type:varchar(20)" json:"phone"`
	Position string    `gorm:"column:position;type:varchar(100)" json:"position"`
	Status   int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ContactProfile) TableName() string { return "contact_profile" }

// UserDept 用户部门关联
type UserDept struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  int64     `gorm:"column:tenant_id;not null;index:idx_tenant" json:"tenant_id"`
	UserID    int64     `gorm:"column:user_id;not null;uniqueIndex:idx_user_dept" json:"user_id"`
	DeptID    int64     `gorm:"column:dept_id;not null;index" json:"dept_id"`
	IsPrimary bool      `gorm:"column:is_primary;default:false" json:"is_primary"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (UserDept) TableName() string { return "contact_user_dept" }

// --- Request / Response types ---

type DeptTreeResp struct {
	DeptID    int64           `json:"dept_id"`
	Name      string          `json:"name"`
	ParentID  int64           `json:"parent_id"`
	SortOrder int             `json:"sort_order"`
	MemberCnt int             `json:"member_count"`
	Children  []*DeptTreeResp `json:"children,omitempty"`
}

type CreateDeptReq struct {
	Name      string `json:"name" binding:"required"`
	ParentID  int64  `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
}

type UpdateDeptReq struct {
	Name      *string `json:"name"`
	SortOrder *int    `json:"sort_order"`
}

type DeptDetailResp struct {
	DeptID      int64  `json:"dept_id"`
	Name        string `json:"name"`
	ParentID    int64  `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	MemberCount int    `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type MemberListReq struct {
	DeptID   int64 `form:"dept_id"`
	Page     int   `form:"page"`
	PageSize int   `form:"page_size"`
}

type MemberListResp struct {
	List     []MemberSummary `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type MemberSearchReq struct {
	Keyword  string `form:"keyword"`
	DeptID   int64  `form:"dept_id"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type MemberSummary struct {
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Position   string `json:"position,omitempty"`
	DeptName   string `json:"dept_name,omitempty"`
	OnlineStatus int8 `json:"online_status,omitempty"` // 0=offline, 1=online
}

type MemberDetailResp struct {
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Phone      string `json:"phone,omitempty"`  // masked
	Position   string `json:"position,omitempty"`
	Depts      []DeptBrief `json:"departments,omitempty"`
	OnlineStatus int8  `json:"online_status"`
}

type DeptBrief struct {
	DeptID int64  `json:"dept_id"`
	Name   string `json:"name"`
}

type UpdateMemberReq struct {
	Name     *string `json:"name"`
	NamePy   *string `json:"name_py"`
	Avatar   *string `json:"avatar"`
	Phone    *string `json:"phone"`
	Position *string `json:"position"`
	DeptIDs  []int64 `json:"dept_ids"`
}

type UserDeptResp struct {
	UserID int64        `json:"user_id"`
	Depts  []DeptBrief  `json:"departments"`
}

type SyncReq struct {
	Departments []SyncDept   `json:"departments"`
	Members     []SyncMember `json:"members"`
}

type SyncDept struct {
	DeptID    int64  `json:"dept_id"`
	Name      string `json:"name"`
	ParentID  int64  `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
}

type SyncMember struct {
	UserID   int64   `json:"user_id"`
	Name     string  `json:"name"`
	NamePy   string  `json:"name_py"`
	Avatar   string  `json:"avatar"`
	Phone    string  `json:"phone"`
	Position string  `json:"position"`
	DeptIDs  []int64 `json:"dept_ids"`
	Status   int8    `json:"status"`
}
