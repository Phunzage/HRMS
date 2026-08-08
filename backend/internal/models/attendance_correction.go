package models

import (
	"time"

	"gorm.io/gorm"
)

// AttendanceCorrection 补卡申请模型（考勤修正）
type AttendanceCorrection struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	EmployeeID     uint           `gorm:"index;not null" json:"employee_id"`             // 申请人（员工ID）
	WorkDate       time.Time      `gorm:"type:date;index;not null" json:"work_date"`     // 补卡日期
	Reason         string         `gorm:"size:500" json:"reason"`                        // 补卡原因
	CorrectionType string         `gorm:"size:32;not null" json:"correction_type"`       // check_in/check_out
	CorrectionTime string         `gorm:"size:8;not null" json:"correction_time"`        // 打卡时间 HH:MM
	Status         string         `gorm:"size:32;default:pending;not null" json:"status"` // pending/approved/rejected
	ApproverID     *uint          `gorm:"index" json:"approver_id"`                      // 审批人（员工ID），可空
	ApprovedAt     *time.Time     `json:"approved_at"`                                   // 审批时间
	RejectReason   string         `gorm:"size:255" json:"reject_reason"`                 // 驳回原因
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
