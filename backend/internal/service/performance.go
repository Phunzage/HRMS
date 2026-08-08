package service

import (
	"errors"
	"regexp"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 绩效业务错误
var (
	ErrPerformanceNotFound      = errors.New("绩效记录不存在")
	ErrPerformanceEmpNotFound   = errors.New("员工不存在")
	ErrPerformanceEmpNoUser     = errors.New("当前账号未关联员工档案")
	ErrPerformancePeriodInvalid = errors.New("考核周期格式错误，应为 YYYY-H1 或 YYYY-H2")
	ErrPerformanceScoreInvalid  = errors.New("考核分数应在 0-100 之间")
	ErrPerformanceNotInDept     = errors.New("非本部门员工，不能操作")
	ErrPerformanceNotApprover   = errors.New("无权限批准绩效")

	ErrPerformanceAlreadyExists = repository.ErrPerformanceAlreadyExists
	ErrPerformanceNotDraft      = repository.ErrPerformanceNotDraft
	ErrPerformanceNotSubmitted  = repository.ErrPerformanceNotSubmitted
)

// periodRegex 考核周期格式校验 YYYY-H1 / YYYY-H2
var periodRegex = regexp.MustCompile(`^\d{4}-(H1|H2)$`)

// ratingNames 评级中文名
var ratingNames = map[string]string{
	"S": "卓越",
	"A": "优秀",
	"B": "良好",
	"C": "合格",
	"D": "待改进",
}

// CreatePerformanceRequest 创建绩效请求
type CreatePerformanceRequest struct {
	EmployeeID uint    `json:"employee_id"`
	Period     string  `json:"period"`
	Score      float64 `json:"score"`
	Comments   string  `json:"comments"`
}

// UpdatePerformanceRequest 更新绩效请求（指针字段区分「未传」与「置零」）
type UpdatePerformanceRequest struct {
	Score    *float64 `json:"score"`
	Comments *string  `json:"comments"`
}

// PerformanceService 绩效业务服务
type PerformanceService struct {
	db       *gorm.DB // 数据范围解析用（ResolveScope / CollectDeptSubtree）
	perfRepo *repository.PerformanceRepo
	empRepo  *repository.EmployeeRepo
}

// NewPerformanceService 创建绩效服务
func NewPerformanceService(db *gorm.DB, perfRepo *repository.PerformanceRepo, empRepo *repository.EmployeeRepo) *PerformanceService {
	return &PerformanceService{db: db, perfRepo: perfRepo, empRepo: empRepo}
}

// List 绩效分页列表，按角色范围过滤（manager 仅本部门及子部门；employee 仅自己）
func (s *PerformanceService) List(roleCode string, userID uint, period, status string, employeeID, departmentID uint, page, pageSize int) ([]repository.PerformanceItem, int64, error) {
	scope, err := s.buildScope(roleCode, userID)
	if err != nil {
		return nil, 0, err
	}
	items, total, err := s.perfRepo.List(scope, period, status, employeeID, departmentID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	s.enrichRating(items)
	return items, total, nil
}

// Mine 当前用户自己的绩效记录（登录即可，含评级中文名）
func (s *PerformanceService) Mine(userID uint, page, pageSize int) ([]repository.PerformanceItem, int64, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, 0, err
	}
	items, total, err := s.perfRepo.ListByEmployee(empID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	s.enrichRating(items)
	return items, total, nil
}

// Create 创建绩效：校验周期/分数/员工存在/同一员工同周期唯一，Rating 由分数自动映射；
// ReviewerID 自动取当前用户的员工ID（HR/系统无员工档案时为 0）；manager 仅限本部门员工
func (s *PerformanceService) Create(roleCode string, userID uint, req CreatePerformanceRequest) (*models.PerformanceReview, error) {
	if !periodRegex.MatchString(req.Period) {
		return nil, ErrPerformancePeriodInvalid
	}
	if req.Score < 0 || req.Score > 100 {
		return nil, ErrPerformanceScoreInvalid
	}
	emp, err := s.empRepo.FindByID(req.EmployeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPerformanceEmpNotFound
		}
		return nil, err
	}
	// manager 仅可给本部门（含子部门）员工打分
	if roleCode == "manager" {
		inDept, err := s.employeeInMyDept(userID, emp.DepartmentID)
		if err != nil {
			return nil, err
		}
		if !inDept {
			return nil, ErrPerformanceNotInDept
		}
	}
	// 考核人 = 当前用户的员工ID（HR/系统无员工档案时为 0，表示由 HR 统筹考核）
	reviewerID := uint(0)
	myEmpID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return nil, err
	}
	if myEmpID != nil {
		reviewerID = *myEmpID
	}
	record := &models.PerformanceReview{
		EmployeeID: req.EmployeeID,
		ReviewerID: reviewerID,
		Period:     req.Period,
		Score:      req.Score,
		Rating:     calcRating(req.Score),
		Comments:   req.Comments,
		Status:     "draft",
	}
	if err := s.perfRepo.Create(record); err != nil {
		return nil, err
	}
	return record, nil
}

// Update 更新绩效（仅 draft 可改），改分数时同步重算评级
func (s *PerformanceService) Update(id uint, req UpdatePerformanceRequest) (*models.PerformanceReview, error) {
	record, err := s.perfRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPerformanceNotFound
		}
		return nil, err
	}
	fields := map[string]interface{}{}
	if req.Score != nil {
		if *req.Score < 0 || *req.Score > 100 {
			return nil, ErrPerformanceScoreInvalid
		}
		fields["score"] = *req.Score
		fields["rating"] = calcRating(*req.Score)
	}
	if req.Comments != nil {
		fields["comments"] = *req.Comments
	}
	if len(fields) == 0 {
		return record, nil
	}
	if err := s.perfRepo.UpdateDraft(id, fields); err != nil {
		return nil, err
	}
	return s.perfRepo.FindByID(id)
}

// Submit 提交绩效（draft → submitted）
func (s *PerformanceService) Submit(id uint) error {
	if _, err := s.perfRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPerformanceNotFound
		}
		return err
	}
	return s.perfRepo.Submit(id)
}

// Approve 批准绩效（submitted → approved），仅 hr/admin 可操作
func (s *PerformanceService) Approve(roleCode string, id uint) error {
	if roleCode != "admin" && roleCode != "hr" {
		return ErrPerformanceNotApprover
	}
	if _, err := s.perfRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPerformanceNotFound
		}
		return err
	}
	return s.perfRepo.Approve(id)
}

// Delete 删除绩效（仅 draft 可删）
func (s *PerformanceService) Delete(id uint) error {
	if _, err := s.perfRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPerformanceNotFound
		}
		return err
	}
	return s.perfRepo.DeleteDraft(id)
}

// buildScope 按角色构建绩效列表访问范围
func (s *PerformanceService) buildScope(roleCode string, userID uint) (repository.PerformanceScope, error) {
	switch roleCode {
	case "admin":
		return repository.PerformanceScope{}, nil
	case "hr":
		// HR 数据范围：绑定负责部门则仅可见该部门及子部门，未绑定则全量
		deptIDs, err := ResolveScope(s.db, userID, roleCode)
		if err != nil {
			return repository.PerformanceScope{}, err
		}
		return repository.PerformanceScope{DeptIDs: deptIDs}, nil
	case "manager":
		// 经理：本部门及子部门员工
		myEmpID, err := s.empRepo.FindEmployeeIDByUserID(userID)
		if err != nil {
			return repository.PerformanceScope{}, err
		}
		if myEmpID == nil {
			return repository.PerformanceScope{Empty: true}, nil
		}
		myEmp, err := s.empRepo.FindByID(*myEmpID)
		if err != nil {
			return repository.PerformanceScope{Empty: true}, nil
		}
		deptIDs, err := CollectDeptSubtree(s.db, myEmp.DepartmentID)
		if err != nil {
			return repository.PerformanceScope{}, err
		}
		return repository.PerformanceScope{DeptIDs: deptIDs}, nil
	default: // employee
		// 员工：仅自己
		myEmpID, err := s.empRepo.FindEmployeeIDByUserID(userID)
		if err != nil {
			return repository.PerformanceScope{}, err
		}
		if myEmpID == nil {
			return repository.PerformanceScope{Empty: true}, nil
		}
		return repository.PerformanceScope{EmployeeID: *myEmpID}, nil
	}
}

// employeeInMyDept 判断目标员工是否属于当前 manager 的部门树（本部门及子部门）
func (s *PerformanceService) employeeInMyDept(userID uint, empDeptID uint) (bool, error) {
	myEmpID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return false, err
	}
	if myEmpID == nil {
		return false, ErrPerformanceEmpNoUser
	}
	myEmp, err := s.empRepo.FindByID(*myEmpID)
	if err != nil {
		return false, err
	}
	deptIDs, err := CollectDeptSubtree(s.db, myEmp.DepartmentID)
	if err != nil {
		return false, err
	}
	for _, d := range deptIDs {
		if d == empDeptID {
			return true, nil
		}
	}
	return false, nil
}

// enrichRating 填充评级中文名
func (s *PerformanceService) enrichRating(items []repository.PerformanceItem) {
	for i := range items {
		items[i].RatingName = ratingNames[items[i].Rating]
	}
}

// requireEmployee 根据登录用户解析关联的员工 ID
func (s *PerformanceService) requireEmployee(userID uint) (uint, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, ErrPerformanceEmpNoUser
	}
	return *empID, nil
}

// calcRating 分数映射评级：90+ → S，80-89 → A，70-79 → B，60-69 → C，<60 → D
func calcRating(score float64) string {
	switch {
	case score >= 90:
		return "S"
	case score >= 80:
		return "A"
	case score >= 70:
		return "B"
	case score >= 60:
		return "C"
	default:
		return "D"
	}
}
