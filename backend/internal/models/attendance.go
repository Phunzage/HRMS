package models

import (
	"time"

	"gorm.io/gorm"
)

// Attendance 考勤记录模型（每人每天一条，联合唯一索引保证）
type Attendance struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EmployeeID uint           `gorm:"index:idx_emp_work_date,unique;not null" json:"employee_id"`         // 员工ID
	WorkDate   time.Time      `gorm:"type:date;index:idx_emp_work_date,unique;not null" json:"work_date"` // 工作日（只存日期）
	CheckIn    *time.Time     `json:"check_in"`                                                           // 上班打卡时间
	CheckOut   *time.Time     `json:"check_out"`                                                          // 下班打卡时间
	Status     string         `gorm:"size:32;default:normal;not null" json:"status"`                      // normal/late/early_leave/absent
	Remark     string         `gorm:"size:255" json:"remark"`                                             // 备注
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
