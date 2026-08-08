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
	// 登录安全加固：连续登录失败计数（达阈值触发锁定）与锁定截止时间（可空表示未锁定）
	FailedLoginCount int        `gorm:"default:0;not null" json:"-"` // 连续登录失败次数
	LockedUntil      *time.Time `json:"-"`                           // 账号锁定截止时间，超过该时间后可重新登录
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
