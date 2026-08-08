package models

import "time"

// Notification 站内通知模型（无软删除）
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`                       // 接收人（系统账号ID）
	Type      string    `gorm:"size:32;index;not null" json:"type"`                  // leave_approval/leave_result/attendance_correction/announcement/system
	Title     string    `gorm:"size:128;not null" json:"title"`                      // 标题
	Content   string    `gorm:"size:500" json:"content"`                             // 内容
	Link      string    `gorm:"size:255" json:"link"`                                // 前端跳转路由，可空
	IsRead    bool      `gorm:"default:false;index;not null" json:"is_read"`         // 是否已读
	CreatedAt time.Time `json:"created_at"`                                          // 创建时间
}
