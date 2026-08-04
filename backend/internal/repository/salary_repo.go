package repository

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// 工资单状态守卫错误
var (
	ErrSalaryAlreadyExists = errors.New("该员工本月工资单已存在")
	ErrSalaryNotDraft      = errors.New("仅 draft 状态的工资单可修改")
	ErrSalaryNotConfirmed  = errors.New("仅 confirmed 状态的工资单可发放")
)

// SalaryItem 工资单列表项（附带员工姓名、部门名称）
type SalaryItem struct {
	models.SalaryRecord
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// SalaryRepo 工资数据仓库
type SalaryRepo struct {
	db *gorm.DB
}

// NewSalaryRepo 创建工资仓库
func NewSalaryRepo(db *gorm.DB) *SalaryRepo {
	return &SalaryRepo{db: db}
}

// salaryQuery 工资单基础查询（JOIN 员工/部门，软删除自动过滤）
func (r *SalaryRepo) salaryQuery(employeeID uint, month, status string, departmentID uint) *gorm.DB {
	q := r.db.Model(&models.SalaryRecord{}).
		Joins("JOIN employees e ON e.id = salary_records.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id")
	if employeeID > 0 {
		q = q.Where("salary_records.employee_id = ?", employeeID)
	}
	if month != "" {
		q = q.Where("salary_records.month = ?", month)
	}
	if status != "" {
		q = q.Where("salary_records.status = ?", status)
	}
	if departmentID > 0 {
		q = q.Where("e.department_id = ?", departmentID)
	}
	return q
}

// List 工资单分页列表，支持 employee_id / month / status / department_id 过滤
func (r *SalaryRepo) List(employeeID uint, month, status string, departmentID uint, page, pageSize int) ([]SalaryItem, int64, error) {
	var total int64
	if err := r.salaryQuery(employeeID, month, status, departmentID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []SalaryItem
	err := r.salaryQuery(employeeID, month, status, departmentID).
		Select("salary_records.*, e.name AS employee_name, d.name AS department_name").
		Order("salary_records.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListByEmployee 某员工的工资单列表（/mine 使用）
func (r *SalaryRepo) ListByEmployee(employeeID uint, month string, page, pageSize int) ([]SalaryItem, int64, error) {
	q := r.db.Model(&models.SalaryRecord{}).
		Joins("JOIN employees e ON e.id = salary_records.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Where("salary_records.employee_id = ?", employeeID)
	if month != "" {
		q = q.Where("salary_records.month = ?", month)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []SalaryItem
	err := q.Select("salary_records.*, e.name AS employee_name, d.name AS department_name").
		Order("salary_records.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID 根据 ID 查询工资单
func (r *SalaryRepo) FindByID(id uint) (*models.SalaryRecord, error) {
	var record models.SalaryRecord
	if err := r.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// Create 创建工资单（事务内校验同员工同月唯一，防止并发重复）
func (r *SalaryRepo) Create(record *models.SalaryRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.SalaryRecord{}).
			Where("employee_id = ? AND month = ?", record.EmployeeID, record.Month).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrSalaryAlreadyExists
		}
		return tx.Create(record).Error
	})
}

// UpdateDraft 仅 draft 状态可更新，否则返回 ErrSalaryNotDraft
func (r *SalaryRepo) UpdateDraft(id uint, fields map[string]interface{}) error {
	res := r.db.Model(&models.SalaryRecord{}).
		Where("id = ? AND status = ?", id, "draft").
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSalaryNotDraft
	}
	return nil
}

// Confirm 确认工资单：draft → confirmed
func (r *SalaryRepo) Confirm(id uint) error {
	res := r.db.Model(&models.SalaryRecord{}).
		Where("id = ? AND status = ?", id, "draft").
		Update("status", "confirmed")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSalaryNotDraft
	}
	return nil
}

// Pay 发放工资：confirmed → paid
func (r *SalaryRepo) Pay(id uint) error {
	res := r.db.Model(&models.SalaryRecord{}).
		Where("id = ? AND status = ?", id, "confirmed").
		Update("status", "paid")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSalaryNotConfirmed
	}
	return nil
}
