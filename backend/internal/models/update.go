package models

import "time"

// AppVersion 客户端版本记录（win/android 客户端升级检查用）
type AppVersion struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Platform     string    `gorm:"size:16;uniqueIndex:idx_plat_ver;not null" json:"platform"` // win / android
	Version      string    `gorm:"size:32;uniqueIndex:idx_plat_ver;not null" json:"version"`  // 如 1.0.0
	BuildNumber  int       `gorm:"not null" json:"build_number"`                              // 递增构建号
	URL          string    `gorm:"size:512;not null" json:"url"`                              // 下载地址（绝对 URL）
	ReleaseNotes string    `gorm:"size:1000" json:"release_notes"`                            // 更新说明
	CreatedAt    time.Time `json:"created_at"`
}
