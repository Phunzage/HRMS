package models

import (
	"time"

	"gorm.io/gorm"
)

// User 系统账号模型
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:64;uniqueIndex;not null" json:"username"` // 登录名
	PasswordHash string         `gorm:"size:255;not null" json:"-"`                   // bcrypt 哈希，不返回给前端
	Email        string         `gorm:"size:128" json:"email"`
	FullName     string         `gorm:"size:128" json:"full_name"`
	RoleID       uint           `gorm:"index;not null" json:"role_id"` // 所属角色
	Role         *Role          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"role,omitempty"`
	EmployeeID   *uint          `gorm:"index" json:"employee_id,omitempty"`             // 关联员工档案，可空
	Status       int            `gorm:"type:smallint;default:1;not null" json:"status"` // 1 启用 / 0 禁用
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
