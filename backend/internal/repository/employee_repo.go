package repository

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// EmployeeItem 员工列表/详情项（附带部门、岗位名称）
type EmployeeItem struct {
	models.Employee
	DepartmentName string `json:"department_name"`
	PositionName   string `json:"position_name"`
}

// empNoLockKey pg_advisory_xact_lock 的键值，用于在事务内串行化工号生成
const empNoLockKey = 10001

// EmployeeRepo 员工数据仓库
type EmployeeRepo struct {
	db *gorm.DB
}

// NewEmployeeRepo 创建员工仓库
func NewEmployeeRepo(db *gorm.DB) *EmployeeRepo {
	return &EmployeeRepo{db: db}
}

// employeeQuery 员工列表基础查询（带部门/岗位 JOIN，软删除自动过滤）
func (r *EmployeeRepo) employeeQuery(keyword string, departmentID uint, status int, positionID uint) *gorm.DB {
	q := r.db.Model(&models.Employee{}).
		Joins("LEFT JOIN departments d ON d.id = employees.department_id").
		Joins("LEFT JOIN positions p ON p.id = employees.position_id")
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("employees.name LIKE ? OR employees.emp_no LIKE ? OR employees.phone LIKE ?", kw, kw, kw)
	}
	if departmentID > 0 {
		q = q.Where("employees.department_id = ?", departmentID)
	}
	if status > 0 {
		q = q.Where("employees.status = ?", status)
	}
	if positionID > 0 {
		q = q.Where("employees.position_id = ?", positionID)
	}
	return q
}

// List 员工分页列表，支持 keyword / department_id / status / position_id 过滤
func (r *EmployeeRepo) List(keyword string, departmentID uint, status int, positionID uint, page, pageSize int) ([]EmployeeItem, int64, error) {
	var total int64
	if err := r.employeeQuery(keyword, departmentID, status, positionID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []EmployeeItem
	err := r.employeeQuery(keyword, departmentID, status, positionID).
		Select("employees.*, d.name AS department_name, p.name AS position_name").
		Order("employees.id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByIDWithNames 查询员工详情（含部门/岗位名称）
func (r *EmployeeRepo) GetByIDWithNames(id uint) (*EmployeeItem, error) {
	var item EmployeeItem
	err := r.db.Model(&models.Employee{}).
		Select("employees.*, d.name AS department_name, p.name AS position_name").
		Joins("LEFT JOIN departments d ON d.id = employees.department_id").
		Joins("LEFT JOIN positions p ON p.id = employees.position_id").
		Where("employees.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

// FindByID 根据 ID 查询员工（不含名称）
func (r *EmployeeRepo) FindByID(id uint) (*models.Employee, error) {
	var emp models.Employee
	if err := r.db.First(&emp, id).Error; err != nil {
		return nil, err
	}
	return &emp, nil
}

// CreateEmployee 事务内创建员工：串行化生成工号 + 可选创建系统账号，防止并发下工号重复
// user 为 nil 时不创建账号；创建成功后会回填 user 的最终用户名
func (r *EmployeeRepo) CreateEmployee(emp *models.Employee, user *models.User) (*models.Employee, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 事务内加咨询锁，串行化工号生成
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", empNoLockKey).Error; err != nil {
			return err
		}
		// 2. 查当前最大序号 +1
		empNo, err := r.nextEmpNo(tx)
		if err != nil {
			return err
		}
		emp.EmpNo = empNo
		// 3. 创建员工
		if err := tx.Create(emp).Error; err != nil {
			return err
		}
		// 4. 可选创建系统账号
		if user != nil {
			user.EmployeeID = &emp.ID
			if err := r.ensureUniqueUsername(tx, user, empNo); err != nil {
				return err
			}
			if err := tx.Create(user).Error; err != nil {
				return err
			}
			// 回写员工表 user_id
			if err := tx.Model(&models.Employee{}).Where("id = ?", emp.ID).Update("user_id", user.ID).Error; err != nil {
				return err
			}
			emp.UserID = &user.ID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return emp, nil
}

// Update 按字段更新员工基本信息
func (r *EmployeeRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Employee{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDeleteAndDisableUser 软删除员工，若关联系统账号则一并禁用
func (r *EmployeeRepo) SoftDeleteAndDisableUser(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var emp models.Employee
		if err := tx.First(&emp, id).Error; err != nil {
			return err
		}
		if emp.UserID != nil {
			if err := tx.Model(&models.User{}).Where("id = ?", *emp.UserID).Update("status", 0).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&models.Employee{}, id).Error
	})
}

// FindEmployeeIDByUserID 查询用户关联的员工档案 ID（可能为空）
func (r *EmployeeRepo) FindEmployeeIDByUserID(userID uint) (*uint, error) {
	var user models.User
	if err := r.db.Select("employee_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return user.EmployeeID, nil
}

// FindRoleIDByCode 根据角色编码查询角色 ID
func (r *EmployeeRepo) FindRoleIDByCode(code string) (uint, error) {
	var role models.Role
	if err := r.db.Where("code = ?", code).First(&role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

// nextEmpNo 生成下一个工号（EMP0001 起，取当前最大序号 +1）
func (r *EmployeeRepo) nextEmpNo(tx *gorm.DB) (string, error) {
	var maxNo string
	if err := tx.Unscoped().Model(&models.Employee{}).Order("emp_no DESC").Limit(1).Pluck("emp_no", &maxNo).Error; err != nil {
		return "", err
	}
	if maxNo == "" {
		return "EMP0001", nil
	}
	seq := 0
	if len(maxNo) > 3 {
		if v, err := strconv.Atoi(maxNo[3:]); err == nil {
			seq = v
		}
	}
	return fmt.Sprintf("EMP%04d", seq+1), nil
}

// ensureUniqueUsername 确保用户名唯一：用户名冲突时追加序号；为空时用 emp+工号序号 兜底
func (r *EmployeeRepo) ensureUniqueUsername(tx *gorm.DB, user *models.User, empNo string) error {
	base := user.Username
	if base == "" {
		if len(empNo) > 3 {
			base = "emp" + empNo[3:]
		} else {
			base = "emp"
		}
	}
	candidate := base
	for i := 1; ; i++ {
		var count int64
		if err := tx.Unscoped().Model(&models.User{}).Where("username = ?", candidate).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			user.Username = candidate
			return nil
		}
		candidate = fmt.Sprintf("%s%d", base, i)
	}
}
