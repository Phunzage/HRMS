package models

import (
	"time"

	"gorm.io/gorm"
)

// Permission 权限点模型
type Permission struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:64;not null" json:"name"`             // 权限名称，如 员工查看
	Code      string         `gorm:"size:64;uniqueIndex;not null" json:"code"` // 权限编码，如 employee:read
	Module    string         `gorm:"size:64" json:"module"`                    // 所属模块，如 employee
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
