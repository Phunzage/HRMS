package repository

import (
	"errors"
	"math"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// 工资单状态守卫错误
var (
	ErrSalaryAlreadyExists = errors.New("该员工本月工资单已存在")
	ErrSalaryNotDraft      = errors.New("仅 draft 状态的工资单可修改")
	ErrSalaryNotConfirmed  = errors.New("仅 confirmed 状态的工资单可发放")
)

// SalaryItem 工资单列表项（附带员工姓名、部门名称、工号）
type SalaryItem struct {
	models.SalaryRecord
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
	EmpNo          string `json:"emp_no"`
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
// deptIDs 为数据范围部门 ID 列表（HR 负责部门及子部门），nil/空则不过滤
func (r *SalaryRepo) salaryQuery(employeeID uint, month, status string, departmentID uint, deptIDs []uint) *gorm.DB {
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
	if len(deptIDs) > 0 {
		q = q.Where("e.department_id IN ?", deptIDs)
	}
	return q
}

// List 工资单分页列表，支持 employee_id / month / status / department_id 过滤，deptIDs 为数据范围（nil 不过滤）
func (r *SalaryRepo) List(employeeID uint, month, status string, departmentID uint, deptIDs []uint, page, pageSize int) ([]SalaryItem, int64, error) {
	var total int64
	if err := r.salaryQuery(employeeID, month, status, departmentID, deptIDs).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]SalaryItem, 0)
	err := r.salaryQuery(employeeID, month, status, departmentID, deptIDs).
		Select("salary_records.*, e.name AS employee_name, d.name AS department_name, e.emp_no AS emp_no").
		Order("salary_records.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ExportAll 导出全部匹配工资单（不分页），支持 month / status / department_id / employee_id 过滤，deptIDs 为数据范围（nil 不过滤）
func (r *SalaryRepo) ExportAll(employeeID uint, month, status string, departmentID uint, deptIDs []uint) ([]SalaryItem, error) {
	items := make([]SalaryItem, 0)
	err := r.salaryQuery(employeeID, month, status, departmentID, deptIDs).
		Select("salary_records.*, e.name AS employee_name, d.name AS department_name, e.emp_no AS emp_no").
		Order("salary_records.month DESC, salary_records.employee_id ASC").
		Scan(&items).Error
	return items, err
}

// BatchGenerateResult 批量生成结果
type BatchGenerateResult struct {
	Generated int `json:"generated"` // 本次为无工资单在职员工生成的草稿数
	Skipped   int `json:"skipped"`   // 已有工资单或非在职（= 员工总数 - generated）
}

// BatchGenerate 批量生成某月工资单：事务内为「当月尚无工资单的在职员工」创建 draft 工资单。
// 基础金额取该员工最近一张历史工资单，无历史则全部为 0；实发自动计算。
func (r *SalaryRepo) BatchGenerate(month string) (*BatchGenerateResult, error) {
	result := &BatchGenerateResult{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 员工总数（非软删除），用于计算 skipped
		var totalEmployees int64
		if err := tx.Model(&models.Employee{}).Count(&totalEmployees).Error; err != nil {
			return err
		}
		// 当月尚无工资单的在职员工（status=1）
		emps := make([]models.Employee, 0)
		if err := tx.Model(&models.Employee{}).
			Where("status = ?", 1).
			Where("NOT EXISTS (SELECT 1 FROM salary_records sr WHERE sr.employee_id = employees.id AND sr.month = ? AND sr.deleted_at IS NULL)", month).
			Find(&emps).Error; err != nil {
			return err
		}
		for _, emp := range emps {
			base, allowance, bonus, deduction, err := r.latestAmounts(tx, emp.ID, month)
			if err != nil {
				return err
			}
			record := &models.SalaryRecord{
				EmployeeID: emp.ID,
				Month:      month,
				BaseSalary: base,
				Allowance:  allowance,
				Bonus:      bonus,
				Deduction:  deduction,
				NetSalary:  math.Round((base+allowance+bonus-deduction)*100) / 100,
				Status:     "draft",
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
			result.Generated++
		}
		result.Skipped = int(totalEmployees) - result.Generated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// latestAmounts 查询员工某月之前的最近一张历史工资单的金额构成；无历史记录返回全 0
func (r *SalaryRepo) latestAmounts(tx *gorm.DB, employeeID uint, month string) (base, allowance, bonus, deduction float64, err error) {
	var rec models.SalaryRecord
	err = tx.Where("employee_id = ? AND month < ?", employeeID, month).
		Order("month DESC, id DESC").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, 0, 0, nil
	}
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return rec.BaseSalary, rec.Allowance, rec.Bonus, rec.Deduction, nil
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
	items := make([]SalaryItem, 0)
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
