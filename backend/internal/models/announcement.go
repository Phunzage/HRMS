package models

import (
	"time"

	"gorm.io/gorm"
)

// Announcement 公告模型
type Announcement struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"size:128;not null" json:"title"`                   // 标题
	Content       string         `gorm:"type:text;not null" json:"content"`                // 正文
	PublisherID   uint           `gorm:"index;not null" json:"publisher_id"`               // 发文人用户ID
	PublisherName string         `gorm:"size:64;not null" json:"publisher_name"`           // 发文人姓名
	IsPinned      bool           `gorm:"default:false;index;not null" json:"is_pinned"`    // 置顶
	Status        string         `gorm:"size:32;default:published;not null" json:"status"` // published/draft
	PublishedAt   *time.Time     `json:"published_at"`                                     // 发布时间，draft 时为 nil
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
