package models

import (
	"time"

	"gorm.io/gorm"
)

// JobOpening 招聘职位模型
type JobOpening struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"size:128;not null" json:"title"`          // 招聘职位名称
	DepartmentID uint           `gorm:"index" json:"department_id"`              // 需求部门
	PositionID   uint           `gorm:"index" json:"position_id"`                // 对应岗位
	Headcount    int            `gorm:"default:1;not null" json:"headcount"`     // 招聘人数
	Requirement  string         `gorm:"type:text" json:"requirement"`            // 任职要求
	Status       string         `gorm:"size:32;default:open;not null" json:"status"` // open/closed
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// Candidate 候选人模型
type Candidate struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	JobOpeningID  uint           `gorm:"index;not null" json:"job_opening_id"` // 应聘职位
	Name          string         `gorm:"size:64;not null" json:"name"`         // 候选人姓名
	Phone         string         `gorm:"size:32" json:"phone"`                 // 联系电话
	Email         string         `gorm:"size:128" json:"email"`                // 邮箱
	Status        string         `gorm:"size:32;default:applied;not null" json:"status"` // applied/interview/offer/hired/rejected
	InterviewDate *time.Time     `json:"interview_date"`                        // 面试时间
	Remark        string         `gorm:"size:255" json:"remark"`                // 备注
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
