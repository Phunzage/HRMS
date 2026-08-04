package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// PositionItem 岗位列表项（附带所属部门名称）
type PositionItem struct {
	models.Position
	DepartmentName string `json:"department_name"`
}

// PositionRepo 岗位数据仓库
type PositionRepo struct {
	db *gorm.DB
}

// NewPositionRepo 创建岗位仓库
func NewPositionRepo(db *gorm.DB) *PositionRepo {
	return &PositionRepo{db: db}
}

// positionQuery 岗位列表基础查询（带部门 JOIN，软删除自动过滤）
func (r *PositionRepo) positionQuery(keyword string, departmentID uint) *gorm.DB {
	q := r.db.Model(&models.Position{}).
		Joins("LEFT JOIN departments d ON d.id = positions.department_id")
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("positions.name LIKE ? OR positions.code LIKE ?", kw, kw)
	}
	if departmentID > 0 {
		q = q.Where("positions.department_id = ?", departmentID)
	}
	return q
}

// List 岗位分页列表，支持 keyword / department_id 过滤
func (r *PositionRepo) List(keyword string, departmentID uint, page, pageSize int) ([]PositionItem, int64, error) {
	var total int64
	if err := r.positionQuery(keyword, departmentID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PositionItem
	err := r.positionQuery(keyword, departmentID).
		Select("positions.*, d.name AS department_name").
		Order("positions.id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID 根据 ID 查询岗位
func (r *PositionRepo) FindByID(id uint) (*models.Position, error) {
	var pos models.Position
	if err := r.db.First(&pos, id).Error; err != nil {
		return nil, err
	}
	return &pos, nil
}

// Exists 判断岗位是否存在
func (r *PositionRepo) Exists(id uint) (bool, error) {
	if id == 0 {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.Position{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeptExists 判断所属部门是否存在（跨表校验）
func (r *PositionRepo) DeptExists(id uint) (bool, error) {
	if id == 0 {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.Department{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CodeExists 判断岗位编码是否已存在（含软删除记录，与 emp_no 的保留策略一致），
// excludeID 用于更新时排除自身
func (r *PositionRepo) CodeExists(code string, excludeID uint) (bool, error) {
	var count int64
	q := r.db.Unscoped().Model(&models.Position{}).Where("code = ?", code)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create 创建岗位
func (r *PositionRepo) Create(pos *models.Position) error {
	return r.db.Create(pos).Error
}

// Update 按字段更新岗位
func (r *PositionRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Position{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 软删除岗位
func (r *PositionRepo) Delete(id uint) error {
	return r.db.Delete(&models.Position{}, id).Error
}

// CountByEmployee 统计岗位下占用的员工数（跨表校验）
func (r *PositionRepo) CountByEmployee(id uint) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Employee{}).Where("position_id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
