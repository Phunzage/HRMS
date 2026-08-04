package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// DepartmentRepo 部门数据仓库
type DepartmentRepo struct {
	db *gorm.DB
}

// NewDepartmentRepo 创建部门仓库
func NewDepartmentRepo(db *gorm.DB) *DepartmentRepo {
	return &DepartmentRepo{db: db}
}

// FindAll 查询全部部门（软删除自动过滤，按 sort_order、id 排序）
func (r *DepartmentRepo) FindAll() ([]models.Department, error) {
	var depts []models.Department
	if err := r.db.Order("sort_order ASC, id ASC").Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

// FindByID 根据 ID 查询部门
func (r *DepartmentRepo) FindByID(id uint) (*models.Department, error) {
	var dept models.Department
	if err := r.db.First(&dept, id).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

// Exists 判断部门是否存在（id 为 0 视为不存在）
func (r *DepartmentRepo) Exists(id uint) (bool, error) {
	if id == 0 {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.Department{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create 创建部门
func (r *DepartmentRepo) Create(dept *models.Department) error {
	return r.db.Create(dept).Error
}

// Update 按字段更新部门（map 中存在的键一律更新，支持置 NULL）
func (r *DepartmentRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Department{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 软删除部门
func (r *DepartmentRepo) Delete(id uint) error {
	return r.db.Delete(&models.Department{}, id).Error
}

// HasChildren 判断部门下是否有（未删除的）子部门
func (r *DepartmentRepo) HasChildren(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Department{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasEmployees 判断部门下是否有在职（未删除的）员工
func (r *DepartmentRepo) HasEmployees(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Employee{}).Where("department_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
