package model

import "time"

// User status
const (
	UserStatusActive   = 1
	UserStatusDisabled = 0
	UserStatusLocked   = 2
)

// Online status
const (
	OnlineStatusOffline = 0
	OnlineStatusOnline  = 1
	OnlineStatusBusy    = 2
	OnlineStatusDND     = 3
)

// Login attempt limits
const (
	MaxLoginAttempts = 5
	LoginLockMinutes = 15
)

// User 用户账号
type User struct {
	UserID    int64     `gorm:"primaryKey;autoIncrement" json:"user_id"`
	TenantID  int64     `gorm:"column:tenant_id;not null;uniqueIndex:idx_account" json:"tenant_id"`
	Account   string    `gorm:"column:account;type:varchar(100);not null;uniqueIndex:idx_account" json:"account"`
	Password  string    `gorm:"column:password;type:varchar(200);not null" json:"-"`
	Name      string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Avatar    string    `gorm:"column:avatar;type:varchar(500)" json:"avatar"`
	Phone     string    `gorm:"column:phone;type:varchar(20)" json:"phone"`
	Email     string    `gorm:"column:email;type:varchar(200)" json:"email"`
	Status    int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "auth_user" }

// UserStatus 用户在线状态
type UserStatus struct {
	UserID    int64     `gorm:"primaryKey" json:"user_id"`
	TenantID  int64     `gorm:"column:tenant_id;not null;index" json:"tenant_id"`
	Status    int8      `gorm:"column:status;not null;default:1" json:"status"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (UserStatus) TableName() string { return "auth_user_status" }

// --- Request / Response types ---

type LoginReq struct {
	TenantID int64  `json:"tenant_id"`
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResp struct {
	Token   string    `json:"token"`
	User    UserBrief `json:"user"`
}

type UserBrief struct {
	UserID  int64  `json:"user_id"`
	Account string `json:"account"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	Status  int8   `json:"status"`
}

type RefreshReq struct {
	Token string `json:"token" binding:"required"`
}

type CreateUserReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

type UpdateUserReq struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
	Phone  *string `json:"phone"`
	Email  *string `json:"email"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type SetStatusReq struct {
	Status int8 `json:"status" binding:"required"` // 1=online, 2=busy, 3=DND
}

type UserDetailResp struct {
	UserID  int64  `json:"user_id"`
	TenantID int64 `json:"tenant_id"`
	Account string `json:"account"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Status  int8   `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type BatchUserReq struct {
	UserIDs []int64 `json:"user_ids" binding:"required"`
}

type BatchStatusResp struct {
	Statuses []UserStatusBrief `json:"statuses"`
}

type UserStatusBrief struct {
	UserID int64 `json:"user_id"`
	Status int8  `json:"status"`
}
