package models

import (
	"time"

	"gorm.io/gorm"
)

// PerformanceReview 绩效考评模型
type PerformanceReview struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	EmployeeID  uint           `gorm:"index:idx_perf_emp_period,unique;not null" json:"employee_id"`    // 被考核人（员工ID）
	ReviewerID  uint           `gorm:"index;not null" json:"reviewer_id"`                               // 考核人（员工ID），HR/系统操作时为 0
	Period      string         `gorm:"size:16;index:idx_perf_emp_period,unique;not null" json:"period"` // 考核周期，如 2026-H1
	Score       float64        `gorm:"type:decimal(5,1);not null" json:"score"`                         // 分数 0-100
	Rating      string         `gorm:"size:8;not null" json:"rating"`                                   // 评级 S/A/B/C/D（service 层由分数映射）
	Comments    string         `gorm:"size:1000" json:"comments"`                                       // 评语
	Status      string         `gorm:"size:32;default:draft;not null" json:"status"`                    // draft/submitted/approved
	SubmittedAt *time.Time     `json:"submitted_at"`                                                    // 提交时间
	ApprovedAt  *time.Time     `json:"approved_at"`                                                     // 批准时间
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
