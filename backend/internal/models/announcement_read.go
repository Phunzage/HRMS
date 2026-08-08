package models

import "time"

// AnnouncementRead 公告已读记录（每个用户对每条公告一条）
// (user_id, announcement_id) 联合唯一，标记已读幂等
type AnnouncementRead struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;uniqueIndex:idx_user_announcement;not null" json:"user_id"`         // 已读用户（系统账号ID）
	AnnouncementID uint      `gorm:"index;uniqueIndex:idx_user_announcement;not null" json:"announcement_id"` // 已读公告ID
	CreatedAt      time.Time `json:"created_at"`
}
