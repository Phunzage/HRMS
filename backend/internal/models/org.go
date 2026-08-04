package models

import (
	"time"

	"gorm.io/gorm"
)

// Department 部门模型（树形结构，ParentID 指向上级）
type Department struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`             // 部门名称
	ParentID  uint           `gorm:"index;default:0;not null" json:"parent_id"` // 上级部门ID，0 表示顶级
	ManagerID *uint          `gorm:"index" json:"manager_id"`                   // 部门负责人（员工ID），可空
	SortOrder int            `gorm:"default:0;not null" json:"sort_order"`      // 同级排序
	Remark    string         `gorm:"size:255" json:"remark"`                    // 备注
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// Position 岗位模型
type Position struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"size:128;not null" json:"name"`            // 岗位名称
	Code         string         `gorm:"size:64;uniqueIndex;not null" json:"code"` // 岗位编码
	DepartmentID uint           `gorm:"index;not null" json:"department_id"`      // 所属部门
	Level        int            `gorm:"default:0;not null" json:"level"`          // 岗位级别
	Remark       string         `gorm:"size:255" json:"remark"`                   // 备注
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}
