package types

import "time"

type Role struct {
	ID          int32      `json:"id" example:"1"`
	Name        string     `json:"name" binding:"required" example:"Admin"`
	Description *string    `json:"description,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
}

type Permission struct {
	ID           int32      `json:"id" example:"10"`
	Name         string     `json:"name" binding:"required" example:"Can view reporters"`
	Code         string     `json:"code" binding:"required,alphanum,lowercase" example:"can_view_reporters"`
	SystemModule string     `json:"system_module" binding:"required" example:"Reporters"`
	Created      *time.Time `json:"created,omitempty"`
	Updated      *time.Time `json:"updated,omitempty"`
}

type User struct {
	ID               int32      `json:"id" example:"100"`
	UserRole         *int32     `json:"user_role,omitempty" example:"1"`
	Firstname        string     `json:"firstname" binding:"required" example:"Jane"`
	Lastname         string     `json:"lastname" binding:"required" example:"Doe"`
	Username         string     `json:"username" binding:"required,alphanum" example:"jane"`
	Telephone        *string    `json:"telephone,omitempty" example:"+256700000000"`
	Password         *string    `json:"password,omitempty" swaggerignore:"true"` // write-only
	Email            *string    `json:"email,omitempty" binding:"omitempty,email" example:"jane@example.org"`
	AllowedIPs       []string   `json:"allowed_ips,omitempty" example:"[\"10.0.0.0/8\",\"192.168.0.0/16\"]"`
	DeniedIPs        []string   `json:"denied_ips,omitempty"`
	FailedAttempts   *int32     `json:"failed_attempts,omitempty" example:"0"`
	TransactionLimit *float64   `json:"transaction_limit,omitempty" example:"1000"`
	IsActive         *bool      `json:"is_active,omitempty" example:"true"`
	IsSystemUser     *bool      `json:"is_system_user,omitempty" example:"false"`
	LastLogin        *time.Time `json:"last_login,omitempty"`
	LastFailedAt     *time.Time `json:"last_failed_at,omitempty"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	LastPasswdUpdate *time.Time `json:"last_passwd_update,omitempty"`
	Created          *time.Time `json:"created,omitempty"`
	Updated          *time.Time `json:"updated,omitempty"`
}

type GrantUserPermission struct {
	PermissionID int32 `json:"permission_id" binding:"required" example:"10"`
}

type GrantRolePermission struct {
	PermissionID int32 `json:"permission_id" binding:"required" example:"10"`
}

type SetUserRole struct {
	RoleID int32 `json:"role_id" binding:"required" example:"1"`
}
