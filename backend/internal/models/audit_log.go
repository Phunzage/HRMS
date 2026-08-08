package models

import (
	"time"
)

// AuditLog 操作审计日志模型（审计日志永不删除，故无软删除字段）
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`              // 操作人用户ID，无登录用户为 0
	Username   string    `gorm:"size:64;index;not null" json:"username"`     // 操作人用户名
	Module     string    `gorm:"size:32;index;not null" json:"module"`       // 所属模块
	Action     string    `gorm:"size:32;not null" json:"action"`             // 操作动作
	ResourceID string    `gorm:"size:64" json:"resource_id,omitempty"`       // 资源标识，如 employee:5，可空
	Detail     string    `gorm:"size:512" json:"detail"`                     // 操作描述
	IP         string    `gorm:"size:64" json:"ip"`                          // 操作来源 IP
	CreatedAt  time.Time `gorm:"index" json:"created_at"`                    // 操作时间
}
