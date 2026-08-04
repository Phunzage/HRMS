package models

import (
	"time"

	"gorm.io/gorm"
)

// SalaryRecord 工资记录模型（每人每月一条）
type SalaryRecord struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EmployeeID uint           `gorm:"index;not null" json:"employee_id"`            // 员工ID
	Month      string         `gorm:"size:7;index;not null" json:"month"`           // 工资月份，如 2026-07
	BaseSalary float64        `gorm:"type:decimal(10,2);default:0;not null" json:"base_salary"` // 基本工资
	Allowance  float64        `gorm:"type:decimal(10,2);default:0;not null" json:"allowance"`   // 津贴
	Bonus      float64        `gorm:"type:decimal(10,2);default:0;not null" json:"bonus"`       // 奖金
	Deduction  float64        `gorm:"type:decimal(10,2);default:0;not null" json:"deduction"`   // 扣款
	NetSalary  float64        `gorm:"type:decimal(10,2);default:0;not null" json:"net_salary"`  // 实发工资
	Status     string         `gorm:"size:32;default:draft;not null" json:"status"`             // draft/confirmed/paid
	Remark     string         `gorm:"size:255" json:"remark"`                       // 备注
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
