package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// ErrLeaveNotPending 请假单当前状态不可操作（事务内状态守卫）
var ErrLeaveNotPending = errors.New("请假单当前状态不可操作")

// LeaveItem 请假列表项（附带请假人姓名、部门名称）
type LeaveItem struct {
	models.LeaveRequest
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// LeaveScope 请假列表访问范围（按角色解析后传给仓库）
// 范围互斥：EmployeeID（员工只看自己）/ ApproverID（经理看本部门+待自己审批）/ DepartmentID（仅本部门）/ DeptIDs（HR 负责部门及子部门）
type LeaveScope struct {
	EmployeeID   uint   // 仅看自己的申请
	ApproverID   uint   // 经理：本部门员工 + 待自己审批的申请
	DepartmentID uint   // 经理所属部门
	DeptIDs      []uint // HR 数据范围部门 ID 列表（nil/空则不过滤）
	Empty        bool   // 无任何可见范围，返回空
}

// LeaveRepo 请假数据仓库
type LeaveRepo struct {
	db *gorm.DB
}

// NewLeaveRepo 创建请假仓库
func NewLeaveRepo(db *gorm.DB) *LeaveRepo {
	return &LeaveRepo{db: db}
}

// leaveQuery 请假列表基础查询（带员工/部门 JOIN），按角色范围过滤
func (r *LeaveRepo) leaveQuery(scope LeaveScope, status string, employeeID uint) *gorm.DB {
	q := r.db.Model(&models.LeaveRequest{}).
		Joins("JOIN employees e ON e.id = leave_requests.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id")
	if scope.Empty {
		return q.Where("1 = 0")
	}
	switch {
	case scope.EmployeeID > 0:
		q = q.Where("leave_requests.employee_id = ?", scope.EmployeeID)
	case scope.ApproverID > 0:
		// 经理：本部门员工 + 待自己审批的
		q = q.Where("e.department_id = ? OR (leave_requests.approver_id = ? AND leave_requests.status = ?)",
			scope.DepartmentID, scope.ApproverID, "pending")
	case scope.DepartmentID > 0:
		q = q.Where("e.department_id = ?", scope.DepartmentID)
	case len(scope.DeptIDs) > 0:
		// HR 数据范围：负责部门及子部门员工
		q = q.Where("e.department_id IN ?", scope.DeptIDs)
	}
	if status != "" {
		q = q.Where("leave_requests.status = ?", status)
	}
	if employeeID > 0 {
		q = q.Where("leave_requests.employee_id = ?", employeeID)
	}
	return q
}

// List 请假分页列表，按角色范围过滤
func (r *LeaveRepo) List(scope LeaveScope, status string, employeeID uint, page, pageSize int) ([]LeaveItem, int64, error) {
	var total int64
	if err := r.leaveQuery(scope, status, employeeID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []LeaveItem
	err := r.leaveQuery(scope, status, employeeID).
		Select("leave_requests.*, e.name AS employee_name, d.name AS department_name").
		Order("leave_requests.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListByEmployee 某员工的请假列表（/mine 使用）
func (r *LeaveRepo) ListByEmployee(employeeID uint, status string, page, pageSize int) ([]LeaveItem, int64, error) {
	q := r.db.Model(&models.LeaveRequest{}).
		Joins("JOIN employees e ON e.id = leave_requests.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Where("leave_requests.employee_id = ?", employeeID)
	if status != "" {
		q = q.Where("leave_requests.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []LeaveItem
	err := q.Select("leave_requests.*, e.name AS employee_name, d.name AS department_name").
		Order("leave_requests.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID 根据 ID 查询请假申请
func (r *LeaveRepo) FindByID(id uint) (*models.LeaveRequest, error) {
	var lr models.LeaveRequest
	if err := r.db.First(&lr, id).Error; err != nil {
		return nil, err
	}
	return &lr, nil
}

// Create 创建请假申请
func (r *LeaveRepo) Create(leave *models.LeaveRequest) error {
	return r.db.Create(leave).Error
}

// CreateAutoApproved 无审批人时直接审批通过：创建申请 + 扣减余额 + 请假区间考勤联动（事务原子）
// 同样视为一次 approve，需一并标记请假区间考勤为 leave
func (r *LeaveRepo) CreateAutoApproved(leave *models.LeaveRequest, year int, totalDays float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(leave).Error; err != nil {
			return err
		}
		if err := r.addUsedDays(tx, leave.EmployeeID, leave.LeaveType, year, leave.Days, totalDays); err != nil {
			return err
		}
		attRepo := NewAttendanceRepo(tx)
		return attRepo.MarkLeaveRange(leave.EmployeeID, leave.StartDate, leave.EndDate)
	})
}

// ApproveAndDeduct 审批通过：更新状态 + 扣减余额 + 请假区间考勤联动标记 leave（事务原子，任一环节失败整体回滚）
func (r *LeaveRepo) ApproveAndDeduct(leaveID uint, year int, totalDays float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var lr models.LeaveRequest
		if err := tx.Select("id", "employee_id", "leave_type", "days", "start_date", "end_date").
			First(&lr, leaveID).Error; err != nil {
			return err
		}
		res := tx.Model(&models.LeaveRequest{}).
			Where("id = ? AND status = ?", leaveID, "pending").
			Updates(map[string]interface{}{"status": "approved", "approved_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrLeaveNotPending
		}
		if err := r.addUsedDays(tx, lr.EmployeeID, lr.LeaveType, year, lr.Days, totalDays); err != nil {
			return err
		}
		// 请假区间内的每一天标记为 leave（无记录则新建，已有则覆盖状态保留打卡）
		attRepo := NewAttendanceRepo(tx)
		return attRepo.MarkLeaveRange(lr.EmployeeID, lr.StartDate, lr.EndDate)
	})
}

// Reject 驳回申请：仅 pending 状态可驳回
func (r *LeaveRepo) Reject(id uint, reason string) error {
	res := r.db.Model(&models.LeaveRequest{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{"status": "rejected", "reject_reason": reason})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrLeaveNotPending
	}
	return nil
}

// Cancel 取消申请：仅 pending 状态可取消
func (r *LeaveRepo) Cancel(id uint) error {
	res := r.db.Model(&models.LeaveRequest{}).
		Where("id = ? AND status = ?", id, "pending").
		Update("status", "cancelled")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrLeaveNotPending
	}
	return nil
}

// GetBalance 查询某员工某类型某年余额记录，不存在返回 nil
func (r *LeaveRepo) GetBalance(employeeID uint, leaveType string, year int) (*models.LeaveBalance, error) {
	var bal models.LeaveBalance
	err := r.db.Where("employee_id = ? AND leave_type = ? AND year = ?", employeeID, leaveType, year).
		First(&bal).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &bal, nil
}

// GetBalanceList 查询某员工某年的全部余额记录
func (r *LeaveRepo) GetBalanceList(employeeID uint, year int) ([]models.LeaveBalance, error) {
	var list []models.LeaveBalance
	err := r.db.Where("employee_id = ? AND year = ?", employeeID, year).Find(&list).Error
	return list, err
}

// FindManagerEmployeeID 查找员工所属部门负责人（部门 ManagerID 对应的员工ID），无负责人返回 nil
func (r *LeaveRepo) FindManagerEmployeeID(employeeID uint) (*uint, error) {
	var rows []struct {
		ManagerID *uint
	}
	err := r.db.Model(&models.Employee{}).
		Select("d.manager_id").
		Joins("JOIN departments d ON d.id = employees.department_id").
		Where("employees.id = ?", employeeID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return rows[0].ManagerID, nil
}

// addUsedDays 事务内扣减假期余额：无记录则创建（总额用默认额度），有记录则原子累加已用天数
func (r *LeaveRepo) addUsedDays(tx *gorm.DB, employeeID uint, leaveType string, year int, days, totalDays float64) error {
	var bal models.LeaveBalance
	err := tx.Where("employee_id = ? AND leave_type = ? AND year = ?", employeeID, leaveType, year).
		First(&bal).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&models.LeaveBalance{
			EmployeeID: employeeID,
			LeaveType:  leaveType,
			Year:       year,
			TotalDays:  totalDays,
			UsedDays:   days,
		}).Error
	case err != nil:
		return err
	default:
		return tx.Model(&models.LeaveBalance{}).Where("id = ?", bal.ID).
			Update("used_days", gorm.Expr("used_days + ?", days)).Error
	}
}
