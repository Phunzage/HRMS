package models

import (
	"time"

	"gorm.io/gorm"
)

// LeaveRequest 请假申请模型
type LeaveRequest struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	EmployeeID   uint           `gorm:"index;not null" json:"employee_id"`   // 请假人（员工ID）
	LeaveType    string         `gorm:"size:32;not null" json:"leave_type"`  // annual/sick/personal/marriage/maternity/other
	StartDate    time.Time      `gorm:"type:date;not null" json:"start_date"` // 开始日期
	EndDate      time.Time      `gorm:"type:date;not null" json:"end_date"`   // 结束日期
	Days         float64        `gorm:"type:decimal(5,1);not null" json:"days"` // 请假天数
	Reason       string         `gorm:"size:500" json:"reason"`               // 请假事由
	Status       string         `gorm:"size:32;default:pending;not null" json:"status"` // pending/approved/rejected/cancelled
	ApproverID   *uint          `gorm:"index" json:"approver_id"`             // 审批人（员工ID），可空
	ApprovedAt   *time.Time     `json:"approved_at"`                          // 审批时间
	RejectReason string         `gorm:"size:255" json:"reject_reason"`        // 驳回原因
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// LeaveBalance 假期余额模型
type LeaveBalance struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	EmployeeID uint    `gorm:"index:idx_balance_emp_type_year,unique;not null" json:"employee_id"` // 员工ID
	LeaveType  string  `gorm:"size:32;index:idx_balance_emp_type_year,unique;not null" json:"leave_type"` // 假期类型
	Year       int     `gorm:"index:idx_balance_emp_type_year,unique;not null" json:"year"`        // 年份
	TotalDays  float64 `gorm:"type:decimal(5,1);default:0;not null" json:"total_days"`              // 年度总天数
	UsedDays   float64 `gorm:"type:decimal(5,1);default:0;not null" json:"used_days"`               // 已用天数
}
