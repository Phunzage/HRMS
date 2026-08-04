package models

import (
	"time"

	"gorm.io/gorm"
)

// Employee 员工档案模型
type Employee struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	EmpNo        string         `gorm:"size:32;uniqueIndex;not null" json:"emp_no"`     // 工号，如 EMP0001
	UserID       *uint          `gorm:"index" json:"user_id"`                           // 关联系统账号，可空
	Name         string         `gorm:"size:64;not null" json:"name"`                   // 姓名
	Gender       int            `gorm:"type:smallint;default:1;not null" json:"gender"` // 1男 / 2女 / 3其他
	BirthDate    *time.Time     `json:"birth_date"`                                     // 出生日期
	IDCard       string         `gorm:"size:32" json:"id_card"`                         // 身份证号
	Phone        string         `gorm:"size:32" json:"phone"`                           // 手机号
	Email        string         `gorm:"size:128" json:"email"`                          // 邮箱
	DepartmentID uint           `gorm:"index;not null" json:"department_id"`            // 所属部门
	PositionID   uint           `gorm:"index;not null" json:"position_id"`              // 所属岗位
	HireDate     *time.Time     `json:"hire_date"`                                      // 入职日期
	Status       int            `gorm:"type:smallint;default:1;not null" json:"status"` // 1在职 / 2离职 / 3试用
	Address      string         `gorm:"size:255" json:"address"`                        // 家庭住址
	Avatar       string         `gorm:"size:255" json:"avatar"`                         // 头像
	Remark       string         `gorm:"size:255" json:"remark"`                         // 备注
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
