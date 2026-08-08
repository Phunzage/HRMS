package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// ErrCorrectionNotPending 补卡申请当前状态不可操作（事务内状态守卫）
var ErrCorrectionNotPending = errors.New("补卡申请当前状态不可操作")

// CorrectionItem 补卡申请列表项（附带申请人姓名、部门名称）
type CorrectionItem struct {
	models.AttendanceCorrection
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// CorrectionScope 补卡申请列表访问范围（按角色解析后传给仓库）
// 三种范围互斥：EmployeeID（员工只看自己）/ ApproverID（经理看本部门员工+待自己审批）/ DepartmentID（仅本部门）/ Empty（无可见范围）
type CorrectionScope struct {
	EmployeeID   uint // 仅看自己的申请
	ApproverID   uint // 经理：本部门员工 + 待自己审批的申请
	DepartmentID uint // 经理所属部门
	Empty        bool // 无任何可见范围，返回空
}

// CorrectionAttendance 审批通过时写入考勤记录的目标打卡时间与状态
type CorrectionAttendance struct {
	CheckIn  *time.Time
	CheckOut *time.Time
	Status   string
}

// AttendanceCorrectionRepo 补卡申请数据仓库
type AttendanceCorrectionRepo struct {
	db *gorm.DB
}

// NewAttendanceCorrectionRepo 创建补卡申请仓库
func NewAttendanceCorrectionRepo(db *gorm.DB) *AttendanceCorrectionRepo {
	return &AttendanceCorrectionRepo{db: db}
}

// Create 创建补卡申请
func (r *AttendanceCorrectionRepo) Create(cor *models.AttendanceCorrection) error {
	return r.db.Create(cor).Error
}

// FindManagerEmployeeID 查找员工所属部门负责人（部门 ManagerID 对应的员工ID），无负责人返回 nil
func (r *AttendanceCorrectionRepo) FindManagerEmployeeID(employeeID uint) (*uint, error) {
	rows := make([]struct {
		ManagerID *uint
	}, 0)
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

// FindByID 根据 ID 查询补卡申请
func (r *AttendanceCorrectionRepo) FindByID(id uint) (*models.AttendanceCorrection, error) {
	var cor models.AttendanceCorrection
	if err := r.db.First(&cor, id).Error; err != nil {
		return nil, err
	}
	return &cor, nil
}

// correctionQuery 补卡申请列表基础查询（带员工/部门 JOIN），按角色范围过滤
func (r *AttendanceCorrectionRepo) correctionQuery(scope CorrectionScope, status string, employeeID uint) *gorm.DB {
	q := r.db.Model(&models.AttendanceCorrection{}).
		Joins("JOIN employees e ON e.id = attendance_corrections.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id")
	if scope.Empty {
		return q.Where("1 = 0")
	}
	switch {
	case scope.EmployeeID > 0:
		q = q.Where("attendance_corrections.employee_id = ?", scope.EmployeeID)
	case scope.ApproverID > 0:
		// 经理：本部门员工 + 待自己审批的申请
		q = q.Where("e.department_id = ? OR (attendance_corrections.approver_id = ? AND attendance_corrections.status = ?)",
			scope.DepartmentID, scope.ApproverID, "pending")
	case scope.DepartmentID > 0:
		q = q.Where("e.department_id = ?", scope.DepartmentID)
	}
	if status != "" {
		q = q.Where("attendance_corrections.status = ?", status)
	}
	if employeeID > 0 {
		q = q.Where("attendance_corrections.employee_id = ?", employeeID)
	}
	return q
}

// List 补卡申请分页列表，按角色范围过滤
func (r *AttendanceCorrectionRepo) List(scope CorrectionScope, status string, employeeID uint, page, pageSize int) ([]CorrectionItem, int64, error) {
	var total int64
	if err := r.correctionQuery(scope, status, employeeID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]CorrectionItem, 0)
	err := r.correctionQuery(scope, status, employeeID).
		Select("attendance_corrections.*, e.name AS employee_name, d.name AS department_name").
		Order("attendance_corrections.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListByEmployee 某员工的补卡申请列表（/mine 使用）
func (r *AttendanceCorrectionRepo) ListByEmployee(employeeID uint, status string, page, pageSize int) ([]CorrectionItem, int64, error) {
	q := r.db.Model(&models.AttendanceCorrection{}).
		Joins("JOIN employees e ON e.id = attendance_corrections.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Where("attendance_corrections.employee_id = ?", employeeID)
	if status != "" {
		q = q.Where("attendance_corrections.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]CorrectionItem, 0)
	err := q.Select("attendance_corrections.*, e.name AS employee_name, d.name AS department_name").
		Order("attendance_corrections.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Reject 驳回补卡申请：仅 pending 状态可驳回
func (r *AttendanceCorrectionRepo) Reject(id uint, reason string) error {
	res := r.db.Model(&models.AttendanceCorrection{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{"status": "rejected", "reject_reason": reason})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCorrectionNotPending
	}
	return nil
}

// Cancel 取消补卡申请：仅 pending 状态可取消
func (r *AttendanceCorrectionRepo) Cancel(id uint) error {
	res := r.db.Model(&models.AttendanceCorrection{}).
		Where("id = ? AND status = ?", id, "pending").
		Update("status", "cancelled")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCorrectionNotPending
	}
	return nil
}

// Approve 审批通过：事务内更新该员工 work_date 的考勤记录并置申请为 approved，任一环节失败整体回滚。
// 仅 pending 状态可审批；att 为目标打卡时间与重算状态（仅使用与 correction_type 匹配的字段，避免误清空另一侧）。
func (r *AttendanceCorrectionRepo) Approve(id uint, att CorrectionAttendance) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var cor models.AttendanceCorrection
		if err := tx.First(&cor, id).Error; err != nil {
			return err
		}
		res := tx.Model(&models.AttendanceCorrection{}).
			Where("id = ? AND status = ?", id, "pending").
			Updates(map[string]interface{}{"status": "approved", "approved_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrCorrectionNotPending
		}
		return applyCorrectionToAttendance(tx, &cor, att)
	})
}

// applyCorrectionToAttendance 事务内将补卡结果写入该员工 work_date 的考勤记录：
// 无考勤行则创建一条；已有则仅更新补卡对应的打卡字段（避免覆盖另一侧），状态按时间规则重算。
func applyCorrectionToAttendance(tx *gorm.DB, cor *models.AttendanceCorrection, att CorrectionAttendance) error {
	attRepo := NewAttendanceRepo(tx)
	workDate := cor.WorkDate.Format("2006-01-02")
	existing, err := attRepo.FindByEmployeeAndDate(cor.EmployeeID, workDate)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 无考勤记录：创建一条（带打卡时间与重算状态）
		return attRepo.Create(&models.Attendance{
			EmployeeID: cor.EmployeeID,
			WorkDate:   cor.WorkDate,
			CheckIn:    att.CheckIn,
			CheckOut:   att.CheckOut,
			Status:     att.Status,
		})
	case err != nil:
		return err
	}
	// 已有考勤记录：仅更新补卡对应的打卡字段，状态按时间规则重算
	fields := map[string]interface{}{"status": att.Status}
	if cor.CorrectionType == "check_in" {
		fields["check_in"] = att.CheckIn
	} else {
		fields["check_out"] = att.CheckOut
	}
	return attRepo.Update(existing.ID, fields)
}
