package service

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 员工业务错误
var (
	ErrEmpNotFound     = errors.New("员工不存在")
	ErrEmpNameRequired = errors.New("员工姓名不能为空")
	ErrEmpDeptNotFound = errors.New("所属部门不存在")
	ErrEmpPosNotFound  = errors.New("所属岗位不存在")
	ErrEmpNoUser       = errors.New("当前账号未关联员工档案")
	ErrEmpDateBad      = errors.New("日期格式错误，应为 YYYY-MM-DD")
)

// DefaultPassword 创建系统账号时的默认密码
const DefaultPassword = "123456"

// CreateEmployeeRequest 创建员工请求（含可选创建系统账号）
type CreateEmployeeRequest struct {
	Name         string  `json:"name"`
	Gender       int     `json:"gender"`
	BirthDate    *string `json:"birth_date"`
	IDCard       string  `json:"id_card"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	DepartmentID uint    `json:"department_id"`
	PositionID   uint    `json:"position_id"`
	HireDate     *string `json:"hire_date"`
	Status       int     `json:"status"`
	Address      string  `json:"address"`
	Avatar       string  `json:"avatar"`
	Remark       string  `json:"remark"`
	CreateUser   bool    `json:"create_user"`
	Username     string  `json:"username"`
}

// UpdateEmployeeRequest 更新员工基本信息请求（指针字段区分「未传」与「置空」）
type UpdateEmployeeRequest struct {
	Name         *string `json:"name"`
	Gender       *int    `json:"gender"`
	BirthDate    *string `json:"birth_date"`
	IDCard       *string `json:"id_card"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	DepartmentID *uint   `json:"department_id"`
	PositionID   *uint   `json:"position_id"`
	HireDate     *string `json:"hire_date"`
	Status       *int    `json:"status"`
	Address      *string `json:"address"`
	Avatar       *string `json:"avatar"`
	Remark       *string `json:"remark"`
}

// CreateEmployeeResult 创建员工返回体（含可选生成的账号信息）
type CreateEmployeeResult struct {
	Employee        *models.Employee `json:"employee"`
	Username        string           `json:"username,omitempty"`
	DefaultPassword string           `json:"default_password,omitempty"`
}

// EmployeeService 员工业务服务
type EmployeeService struct {
	db       *gorm.DB // 数据范围解析用（ResolveScope）
	empRepo  *repository.EmployeeRepo
	deptRepo *repository.DepartmentRepo
	posRepo  *repository.PositionRepo
}

// NewEmployeeService 创建员工服务
func NewEmployeeService(db *gorm.DB, empRepo *repository.EmployeeRepo, deptRepo *repository.DepartmentRepo, posRepo *repository.PositionRepo) *EmployeeService {
	return &EmployeeService{db: db, empRepo: empRepo, deptRepo: deptRepo, posRepo: posRepo}
}

// List 员工分页列表，支持 keyword / department_id / status / position_id 过滤，按角色数据范围过滤（hr 仅负责部门及子部门）
func (s *EmployeeService) List(roleCode string, userID uint, keyword string, departmentID uint, status int, positionID uint, page, pageSize int) ([]repository.EmployeeItem, int64, error) {
	scopeDeptIDs, err := ResolveScope(s.db, userID, roleCode)
	if err != nil {
		return nil, 0, err
	}
	return s.empRepo.List(keyword, departmentID, status, positionID, scopeDeptIDs, page, pageSize)
}

// ExportRows 导出员工花名册数据行（含表头与性别/状态中文映射），导出全部匹配不分页，与列表范围一致
func (s *EmployeeService) ExportRows(roleCode string, userID uint, keyword string, departmentID uint, status int, positionID uint) ([]string, [][]interface{}, error) {
	headers := []string{"工号", "姓名", "性别", "部门", "岗位", "手机", "邮箱", "入职日期", "状态", "备注"}
	scopeDeptIDs, err := ResolveScope(s.db, userID, roleCode)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.empRepo.ExportAll(keyword, departmentID, status, positionID, scopeDeptIDs)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]interface{}, 0, len(items))
	for _, it := range items {
		hireDate := ""
		if it.HireDate != nil {
			hireDate = it.HireDate.Format("2006-01-02")
		}
		rows = append(rows, []interface{}{
			it.EmpNo,
			it.Name,
			genderName(it.Gender),
			it.DepartmentName,
			it.PositionName,
			it.Phone,
			it.Email,
			hireDate,
			empStatusName(it.Status),
			it.Remark,
		})
	}
	return headers, rows, nil
}

// genderName 性别枚举中文映射：1男 / 2女 / 3其他
func genderName(gender int) string {
	switch gender {
	case 1:
		return "男"
	case 2:
		return "女"
	default:
		return "其他"
	}
}

// empStatusName 员工状态中文映射：1在职 / 2离职 / 3试用
func empStatusName(status int) string {
	switch status {
	case 1:
		return "在职"
	case 2:
		return "离职"
	case 3:
		return "试用"
	default:
		return "未知"
	}
}

// GetByID 员工详情（含部门/岗位名称）
func (s *EmployeeService) GetByID(id uint) (*repository.EmployeeItem, error) {
	item, err := s.empRepo.GetByIDWithNames(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmpNotFound
		}
		return nil, err
	}
	return item, nil
}

// Create 创建员工：校验必填与部门/岗位存在，自动生成工号，可选创建系统账号
func (s *EmployeeService) Create(req CreateEmployeeRequest) (*CreateEmployeeResult, error) {
	if req.Name == "" {
		return nil, ErrEmpNameRequired
	}
	ok, err := s.deptRepo.Exists(req.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrEmpDeptNotFound
	}
	ok, err = s.posRepo.Exists(req.PositionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrEmpPosNotFound
	}

	birthDate, err := parseDatePtr(req.BirthDate)
	if err != nil {
		return nil, err
	}
	hireDate, err := parseDatePtr(req.HireDate)
	if err != nil {
		return nil, err
	}

	emp := &models.Employee{
		Name:         req.Name,
		Gender:       req.Gender,
		BirthDate:    birthDate,
		IDCard:       req.IDCard,
		Phone:        req.Phone,
		Email:        req.Email,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		HireDate:     hireDate,
		Status:       req.Status,
		Address:      req.Address,
		Avatar:       req.Avatar,
		Remark:       req.Remark,
	}
	if emp.Status == 0 {
		emp.Status = 1
	}
	if emp.Gender == 0 {
		emp.Gender = 1
	}

	// 可选创建系统账号
	var user *models.User
	if req.CreateUser {
		roleID, err := s.empRepo.FindRoleIDByCode("employee")
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		username := req.Username
		if username == "" {
			username = toPinyin(req.Name)
		}
		user = &models.User{
			Username:     username,
			PasswordHash: string(hash),
			Email:        req.Email,
			FullName:     req.Name,
			RoleID:       roleID,
			Status:       1,
		}
	}

	created, err := s.empRepo.CreateEmployee(emp, user)
	if err != nil {
		return nil, err
	}

	result := &CreateEmployeeResult{Employee: created}
	if req.CreateUser {
		// user 指针在仓库事务内已被回填最终用户名
		result.Username = user.Username
		result.DefaultPassword = DefaultPassword
	}
	return result, nil
}

// Update 更新员工基本信息
func (s *EmployeeService) Update(id uint, req UpdateEmployeeRequest) error {
	if _, err := s.empRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmpNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	if req.Name != nil && *req.Name != "" {
		fields["name"] = *req.Name
	}
	if req.Gender != nil {
		fields["gender"] = *req.Gender
	}
	if req.BirthDate != nil {
		t, err := parseDatePtr(req.BirthDate)
		if err != nil {
			return err
		}
		fields["birth_date"] = t
	}
	if req.IDCard != nil {
		fields["id_card"] = *req.IDCard
	}
	if req.Phone != nil {
		fields["phone"] = *req.Phone
	}
	if req.Email != nil {
		fields["email"] = *req.Email
	}
	if req.DepartmentID != nil {
		ok, err := s.deptRepo.Exists(*req.DepartmentID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrEmpDeptNotFound
		}
		fields["department_id"] = *req.DepartmentID
	}
	if req.PositionID != nil {
		ok, err := s.posRepo.Exists(*req.PositionID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrEmpPosNotFound
		}
		fields["position_id"] = *req.PositionID
	}
	if req.HireDate != nil {
		t, err := parseDatePtr(req.HireDate)
		if err != nil {
			return err
		}
		fields["hire_date"] = t
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.Address != nil {
		fields["address"] = *req.Address
	}
	if req.Avatar != nil {
		fields["avatar"] = *req.Avatar
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if len(fields) == 0 {
		return nil
	}
	return s.empRepo.Update(id, fields)
}

// Delete 软删除员工，若有系统账号一并禁用
func (s *EmployeeService) Delete(id uint) error {
	if _, err := s.empRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmpNotFound
		}
		return err
	}
	return s.empRepo.SoftDeleteAndDisableUser(id)
}

// GetMyProfile 当前登录用户关联的员工档案（仅自己）
func (s *EmployeeService) GetMyProfile(userID uint) (*repository.EmployeeItem, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return nil, err
	}
	if empID == nil {
		return nil, ErrEmpNoUser
	}
	return s.GetByID(*empID)
}

// parseDatePtr 解析 YYYY-MM-DD 日期；空串表示清空（返回 nil）
func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	if *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, ErrEmpDateBad
	}
	return &t, nil
}
