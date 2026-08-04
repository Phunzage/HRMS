package models

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色模型
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:64;not null" json:"name"`             // 角色名称，如 管理员
	Code        string         `gorm:"size:64;uniqueIndex;not null" json:"code"` // 角色编码，如 admin
	Description string         `gorm:"size:255" json:"description"`              // 角色描述
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
