package service

import (
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// uniqSeq 全局唯一序号，配合纳秒时间戳生成跨运行不重复的标识
var uniqSeq atomic.Uint64

// uniqueStr 生成唯一字符串：前缀 + 纳秒时间戳片段 + 自增序号，避免跨测试残留冲突
func uniqueStr(prefix string) string {
	n := uniqSeq.Add(1)
	return fmt.Sprintf("%s%d-%d", prefix, time.Now().UnixNano()%100000000, n)
}

// createTestDeptPos 在 tx 内创建部门与岗位，返回二者
func createTestDeptPos(tx *gorm.DB) (*models.Department, *models.Position, error) {
	dept := &models.Department{Name: uniqueStr("测试部"), SortOrder: 1}
	if err := tx.Create(dept).Error; err != nil {
		return nil, nil, fmt.Errorf("创建部门失败: %w", err)
	}
	pos := &models.Position{
		Name:         uniqueStr("测试岗"),
		Code:         uniqueStr("pos_"),
		DepartmentID: dept.ID,
	}
	if err := tx.Create(pos).Error; err != nil {
		return nil, nil, fmt.Errorf("创建岗位失败: %w", err)
	}
	return dept, pos, nil
}

// createTestEmployee 直接创建员工（手动指定工号，不经过自动生成逻辑）
func createTestEmployee(tx *gorm.DB, empNo, name string, deptID, posID uint) (*models.Employee, error) {
	emp := &models.Employee{
		EmpNo:        empNo,
		Name:         name,
		Gender:       1,
		DepartmentID: deptID,
		PositionID:   posID,
		Status:       1,
	}
	if err := tx.Create(emp).Error; err != nil {
		return nil, fmt.Errorf("创建员工失败: %w", err)
	}
	return emp, nil
}

// createTestUser 创建用户（bcrypt 密码），可关联员工
func createTestUser(tx *gorm.DB, username, password, roleCode string, employeeID *uint, status int) (*models.User, error) {
	var role models.Role
	if err := tx.Where("code = ?", roleCode).First(&role).Error; err != nil {
		return nil, fmt.Errorf("查询角色 %s 失败: %w", roleCode, err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密码哈希失败: %w", err)
	}
	u := &models.User{
		Username:     username,
		PasswordHash: string(hash),
		FullName:     username,
		RoleID:       role.ID,
		EmployeeID:   employeeID,
		Status:       status,
	}
	if err := tx.Create(u).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	// Status 有 default:1 标签，值为 0 时会被 GORM 替换为默认值 1，
	// 需要禁用账号（status=0）时插入后显式回写
	if status == 0 {
		if err := tx.Model(&models.User{}).Where("id = ?", u.ID).Update("status", 0).Error; err != nil {
			return nil, fmt.Errorf("设置禁用状态失败: %w", err)
		}
	}
	return u, nil
}
