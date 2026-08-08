package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// 绩效状态守卫错误
var (
	ErrPerformanceAlreadyExists = errors.New("该员工本周期已有绩效记录")
	ErrPerformanceNotDraft      = errors.New("仅 draft 状态的绩效可操作")
	ErrPerformanceNotSubmitted  = errors.New("仅 submitted 状态的绩效可批准")
)

// PerformanceItem 绩效列表项（附带被考核人姓名/部门/工号与考核人姓名）
type PerformanceItem struct {
	models.PerformanceReview
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
	EmpNo          string `json:"emp_no"`
	ReviewerName   string `json:"reviewer_name"`
	RatingName     string `json:"rating_name"` // 评级中文名，service 层填充
}

// PerformanceScope 绩效列表访问范围（按角色解析后传给仓库）
// 范围互斥：EmployeeID（员工只看自己）/ DeptIDs（经理本部门及子部门、HR 负责部门及子部门）
type PerformanceScope struct {
	EmployeeID uint   // 员工只看自己
	DeptIDs    []uint // 部门数据范围（经理本部门树 / HR 负责部门树），nil 不过滤
	Empty      bool   // 无任何可见范围，返回空
}

// PerformanceRepo 绩效数据仓库
type PerformanceRepo struct {
	db *gorm.DB
}

// NewPerformanceRepo 创建绩效仓库
func NewPerformanceRepo(db *gorm.DB) *PerformanceRepo {
	return &PerformanceRepo{db: db}
}

// performanceQuery 绩效列表基础查询（JOIN 被考核人/部门/考核人），按角色范围过滤
func (r *PerformanceRepo) performanceQuery(scope PerformanceScope, period, status string, employeeID, departmentID uint) *gorm.DB {
	q := r.db.Model(&models.PerformanceReview{}).
		Joins("JOIN employees e ON e.id = performance_reviews.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Joins("LEFT JOIN employees rv ON rv.id = performance_reviews.reviewer_id")
	if scope.Empty {
		return q.Where("1 = 0")
	}
	switch {
	case scope.EmployeeID > 0:
		q = q.Where("performance_reviews.employee_id = ?", scope.EmployeeID)
	case len(scope.DeptIDs) > 0:
		q = q.Where("e.department_id IN ?", scope.DeptIDs)
	}
	if period != "" {
		q = q.Where("performance_reviews.period = ?", period)
	}
	if status != "" {
		q = q.Where("performance_reviews.status = ?", status)
	}
	if employeeID > 0 {
		q = q.Where("performance_reviews.employee_id = ?", employeeID)
	}
	if departmentID > 0 {
		q = q.Where("e.department_id = ?", departmentID)
	}
	return q
}

// List 绩效分页列表，按角色范围过滤，支持 period/status/employee_id/department_id 过滤
func (r *PerformanceRepo) List(scope PerformanceScope, period, status string, employeeID, departmentID uint, page, pageSize int) ([]PerformanceItem, int64, error) {
	var total int64
	if err := r.performanceQuery(scope, period, status, employeeID, departmentID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]PerformanceItem, 0)
	err := r.performanceQuery(scope, period, status, employeeID, departmentID).
		Select("performance_reviews.*, e.name AS employee_name, d.name AS department_name, e.emp_no AS emp_no, rv.name AS reviewer_name").
		Order("performance_reviews.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListByEmployee 某员工自己的绩效列表（/mine 使用），按周期倒序
func (r *PerformanceRepo) ListByEmployee(employeeID uint, page, pageSize int) ([]PerformanceItem, int64, error) {
	q := r.db.Model(&models.PerformanceReview{}).
		Joins("JOIN employees e ON e.id = performance_reviews.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Joins("LEFT JOIN employees rv ON rv.id = performance_reviews.reviewer_id").
		Where("performance_reviews.employee_id = ?", employeeID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]PerformanceItem, 0)
	err := q.Select("performance_reviews.*, e.name AS employee_name, d.name AS department_name, e.emp_no AS emp_no, rv.name AS reviewer_name").
		Order("performance_reviews.period DESC, performance_reviews.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID 根据 ID 查询绩效记录
func (r *PerformanceRepo) FindByID(id uint) (*models.PerformanceReview, error) {
	var p models.PerformanceReview
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create 创建绩效记录（事务内校验同员工同周期唯一，防止并发重复）
func (r *PerformanceRepo) Create(record *models.PerformanceReview) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.PerformanceReview{}).
			Where("employee_id = ? AND period = ?", record.EmployeeID, record.Period).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrPerformanceAlreadyExists
		}
		return tx.Create(record).Error
	})
}

// UpdateDraft 仅 draft 状态可更新，否则返回 ErrPerformanceNotDraft
func (r *PerformanceRepo) UpdateDraft(id uint, fields map[string]interface{}) error {
	res := r.db.Model(&models.PerformanceReview{}).
		Where("id = ? AND status = ?", id, "draft").
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPerformanceNotDraft
	}
	return nil
}

// Submit 提交绩效：draft → submitted，记录提交时间
func (r *PerformanceRepo) Submit(id uint) error {
	res := r.db.Model(&models.PerformanceReview{}).
		Where("id = ? AND status = ?", id, "draft").
		Updates(map[string]interface{}{"status": "submitted", "submitted_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPerformanceNotDraft
	}
	return nil
}

// Approve 批准绩效：submitted → approved，记录批准时间
func (r *PerformanceRepo) Approve(id uint) error {
	res := r.db.Model(&models.PerformanceReview{}).
		Where("id = ? AND status = ?", id, "submitted").
		Updates(map[string]interface{}{"status": "approved", "approved_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPerformanceNotSubmitted
	}
	return nil
}

// DeleteDraft 删除绩效：仅 draft 状态可删除
func (r *PerformanceRepo) DeleteDraft(id uint) error {
	res := r.db.Model(&models.PerformanceReview{}).
		Where("id = ? AND status = ?", id, "draft").
		Delete(&models.PerformanceReview{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPerformanceNotDraft
	}
	return nil
}
