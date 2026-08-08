package service

import (
	"errors"
	"fmt"
	"math"
	"regexp"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 工资业务错误
var (
	ErrSalaryNotFound      = errors.New("工资单不存在")
	ErrSalaryEmpNotFound   = errors.New("员工不存在")
	ErrSalaryEmpNoUser     = errors.New("当前账号未关联员工档案")
	ErrSalaryMonthInvalid  = errors.New("月份格式错误，应为 YYYY-MM")
	ErrSalaryAlreadyExists = repository.ErrSalaryAlreadyExists
	ErrSalaryNotDraft      = repository.ErrSalaryNotDraft
	ErrSalaryNotConfirmed  = repository.ErrSalaryNotConfirmed
)

// monthRegex 工资月份格式校验 YYYY-MM
var monthRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// CreateSalaryRequest 创建工资单请求
type CreateSalaryRequest struct {
	EmployeeID uint    `json:"employee_id"`
	Month      string  `json:"month"`
	BaseSalary float64 `json:"base_salary"`
	Allowance  float64 `json:"allowance"`
	Bonus      float64 `json:"bonus"`
	Deduction  float64 `json:"deduction"`
	Remark     string  `json:"remark"`
}

// UpdateSalaryRequest 更新工资单请求（指针字段区分「未传」与「置零」）
type UpdateSalaryRequest struct {
	BaseSalary *float64 `json:"base_salary"`
	Allowance  *float64 `json:"allowance"`
	Bonus      *float64 `json:"bonus"`
	Deduction  *float64 `json:"deduction"`
	Remark     *string  `json:"remark"`
}

// SalaryService 工资业务服务
type SalaryService struct {
	salaryRepo *repository.SalaryRepo
	empRepo    *repository.EmployeeRepo
}

// NewSalaryService 创建工资服务
func NewSalaryService(salaryRepo *repository.SalaryRepo, empRepo *repository.EmployeeRepo) *SalaryService {
	return &SalaryService{salaryRepo: salaryRepo, empRepo: empRepo}
}

// List 工资单分页列表，按角色范围过滤（employee 角色仅看自己）
func (s *SalaryService) List(roleCode string, userID uint, month string, employeeID uint, status string, departmentID uint, page, pageSize int) ([]repository.SalaryItem, int64, error) {
	if roleCode == "employee" {
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return nil, 0, err
		}
		employeeID = empID
	}
	return s.salaryRepo.List(employeeID, month, status, departmentID, page, pageSize)
}

// ExportRows 导出工资表数据行（含表头与状态中文映射），导出全部匹配不分页；
// employee 角色仅导出本人工资单，与列表接口范围一致
func (s *SalaryService) ExportRows(roleCode string, userID uint, month, status string, departmentID uint) ([]string, [][]interface{}, error) {
	headers := []string{"月份", "工号", "姓名", "部门", "基本工资", "津贴", "奖金", "扣款", "实发", "状态"}
	employeeID := uint(0)
	if roleCode == "employee" {
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return nil, nil, err
		}
		employeeID = empID
		departmentID = 0
	}
	items, err := s.salaryRepo.ExportAll(employeeID, month, status, departmentID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]interface{}, 0, len(items))
	for _, it := range items {
		rows = append(rows, []interface{}{
			it.Month,
			it.EmpNo,
			it.EmployeeName,
			it.DepartmentName,
			fmt.Sprintf("%.2f", it.BaseSalary),
			fmt.Sprintf("%.2f", it.Allowance),
			fmt.Sprintf("%.2f", it.Bonus),
			fmt.Sprintf("%.2f", it.Deduction),
			fmt.Sprintf("%.2f", it.NetSalary),
			salaryStatusName(it.Status),
		})
	}
	return headers, rows, nil
}

// salaryStatusName 工资单状态中文映射：draft草稿 / confirmed已确认 / paid已发放
func salaryStatusName(status string) string {
	switch status {
	case "draft":
		return "草稿"
	case "confirmed":
		return "已确认"
	case "paid":
		return "已发放"
	default:
		return status
	}
}

// Mine 当前用户自己的工资单列表（登录即可）
func (s *SalaryService) Mine(userID uint, month string, page, pageSize int) ([]repository.SalaryItem, int64, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, 0, err
	}
	return s.salaryRepo.ListByEmployee(empID, month, page, pageSize)
}

// Create 创建工资单：校验月份格式与员工存在，自动计算实发工资，同员工同月唯一
func (s *SalaryService) Create(req CreateSalaryRequest) (*models.SalaryRecord, error) {
	if !monthRegex.MatchString(req.Month) {
		return nil, ErrSalaryMonthInvalid
	}
	if _, err := s.empRepo.FindByID(req.EmployeeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSalaryEmpNotFound
		}
		return nil, err
	}
	record := &models.SalaryRecord{
		EmployeeID: req.EmployeeID,
		Month:      req.Month,
		BaseSalary: req.BaseSalary,
		Allowance:  req.Allowance,
		Bonus:      req.Bonus,
		Deduction:  req.Deduction,
		NetSalary:  round2(req.BaseSalary + req.Allowance + req.Bonus - req.Deduction),
		Status:     "draft",
		Remark:     req.Remark,
	}
	if err := s.salaryRepo.Create(record); err != nil {
		return nil, err
	}
	return record, nil
}

// Update 更新工资单：仅 draft 状态可改，改后重算实发工资
func (s *SalaryService) Update(id uint, req UpdateSalaryRequest) (*models.SalaryRecord, error) {
	record, err := s.salaryRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSalaryNotFound
		}
		return nil, err
	}
	base := record.BaseSalary
	allowance := record.Allowance
	bonus := record.Bonus
	deduction := record.Deduction
	if req.BaseSalary != nil {
		base = *req.BaseSalary
	}
	if req.Allowance != nil {
		allowance = *req.Allowance
	}
	if req.Bonus != nil {
		bonus = *req.Bonus
	}
	if req.Deduction != nil {
		deduction = *req.Deduction
	}
	fields := map[string]interface{}{
		"base_salary": base,
		"allowance":   allowance,
		"bonus":       bonus,
		"deduction":   deduction,
		"net_salary":  round2(base + allowance + bonus - deduction),
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if err := s.salaryRepo.UpdateDraft(id, fields); err != nil {
		return nil, err
	}
	return s.salaryRepo.FindByID(id)
}

// Confirm 确认工资单（draft → confirmed）
func (s *SalaryService) Confirm(id uint) error {
	if _, err := s.salaryRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSalaryNotFound
		}
		return err
	}
	return s.salaryRepo.Confirm(id)
}

// Pay 发放工资（confirmed → paid）
func (s *SalaryService) Pay(id uint) error {
	if _, err := s.salaryRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSalaryNotFound
		}
		return err
	}
	return s.salaryRepo.Pay(id)
}

// requireEmployee 根据登录用户解析关联的员工 ID
func (s *SalaryService) requireEmployee(userID uint) (uint, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, ErrSalaryEmpNoUser
	}
	return *empID, nil
}

// round2 金额四舍五入保留两位小数，避免浮点运算误差
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
