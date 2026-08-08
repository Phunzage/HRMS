package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// AttendanceItem 考勤列表项（附带员工姓名、部门名称）
type AttendanceItem struct {
	models.Attendance
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// AttendanceReportItem 考勤月报汇总项（按员工聚合某月各考勤状态计数）
type AttendanceReportItem struct {
	EmployeeID     uint   `json:"employee_id"`
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
	WorkDays       int    `json:"work_days"`        // 当月考勤总条数
	NormalDays     int    `json:"normal_days"`      // normal 条数
	LateCount      int    `json:"late_count"`       // late 条数
	EarlyLeaveCount int   `json:"early_leave_count"` // early_leave 条数
	AbsentDays     int    `json:"absent_days"`      // absent 条数
	LeaveDays      int    `json:"leave_days"`       // leave 条数
}

// AttendanceRepo 考勤数据仓库
type AttendanceRepo struct {
	db *gorm.DB
}

// NewAttendanceRepo 创建考勤仓库
func NewAttendanceRepo(db *gorm.DB) *AttendanceRepo {
	return &AttendanceRepo{db: db}
}

// attendanceQuery 考勤列表基础查询（带员工/部门 JOIN，软删除自动过滤）
// workDate 为 "YYYY-MM-DD" 精确匹配；month 为 "2026-07" 用 LIKE 匹配前缀
// deptIDs 为数据范围部门 ID 列表（HR 负责部门及子部门），nil/空则不过滤
func (r *AttendanceRepo) attendanceQuery(employeeID uint, workDate, month, status string, deptIDs []uint) *gorm.DB {
	q := r.db.Model(&models.Attendance{}).
		Joins("LEFT JOIN employees e ON e.id = attendances.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id")
	if employeeID > 0 {
		q = q.Where("attendances.employee_id = ?", employeeID)
	}
	if workDate != "" {
		q = q.Where("attendances.work_date = ?", workDate)
	}
	if month != "" {
		q = q.Where("CAST(attendances.work_date AS TEXT) LIKE ?", month+"%")
	}
	if status != "" {
		q = q.Where("attendances.status = ?", status)
	}
	if len(deptIDs) > 0 {
		q = q.Where("e.department_id IN ?", deptIDs)
	}
	return q
}

// List 考勤分页列表，支持 employee_id / work_date / month / status 过滤，deptIDs 为数据范围（nil 不过滤）
func (r *AttendanceRepo) List(employeeID uint, workDate, month, status string, deptIDs []uint, page, pageSize int) ([]AttendanceItem, int64, error) {
	var total int64
	if err := r.attendanceQuery(employeeID, workDate, month, status, deptIDs).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AttendanceItem
	err := r.attendanceQuery(employeeID, workDate, month, status, deptIDs).
		Select("attendances.*, e.name AS employee_name, d.name AS department_name").
		Order("attendances.work_date DESC, attendances.employee_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Report 考勤月报：按员工聚合某月的各考勤状态计数，返回当月有考勤记录的员工汇总
// month 为 "2026-08"；departmentID 为界面筛选（0 不过滤）；deptIDs 为数据范围（nil 不过滤）；employeeID 指定员工（0 不过滤，employee 角色仅看自己）
// 排序按 department_id, employee_id
func (r *AttendanceRepo) Report(month string, departmentID uint, deptIDs []uint, employeeID uint) ([]AttendanceReportItem, error) {
	q := r.db.Model(&models.Attendance{}).
		Select(`attendances.employee_id, e.name AS employee_name, d.name AS department_name,
			COUNT(*) AS work_days,
			COUNT(*) FILTER (WHERE attendances.status = 'normal') AS normal_days,
			COUNT(*) FILTER (WHERE attendances.status = 'late') AS late_count,
			COUNT(*) FILTER (WHERE attendances.status = 'early_leave') AS early_leave_count,
			COUNT(*) FILTER (WHERE attendances.status = 'absent') AS absent_days,
			COUNT(*) FILTER (WHERE attendances.status = 'leave') AS leave_days`).
		Joins("JOIN employees e ON e.id = attendances.employee_id").
		Joins("LEFT JOIN departments d ON d.id = e.department_id").
		Where("CAST(attendances.work_date AS TEXT) LIKE ?", month+"%").
		Group("attendances.employee_id, e.department_id, e.name, d.name").
		Order("e.department_id ASC, attendances.employee_id ASC")
	if departmentID > 0 {
		q = q.Where("e.department_id = ?", departmentID)
	}
	if len(deptIDs) > 0 {
		q = q.Where("e.department_id IN ?", deptIDs)
	}
	if employeeID > 0 {
		q = q.Where("attendances.employee_id = ?", employeeID)
	}
	// 初始化为空切片，无匹配时 JSON 返回 [] 而非 null
	items := []AttendanceReportItem{}
	if err := q.Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID 根据 ID 查询考勤记录
func (r *AttendanceRepo) FindByID(id uint) (*models.Attendance, error) {
	var att models.Attendance
	if err := r.db.First(&att, id).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

// FindByEmployeeAndDate 查询某员工指定工作日的考勤记录（可能不存在）
func (r *AttendanceRepo) FindByEmployeeAndDate(employeeID uint, workDate string) (*models.Attendance, error) {
	var att models.Attendance
	if err := r.db.Where("employee_id = ? AND work_date = ?", employeeID, workDate).First(&att).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

// Create 创建考勤记录
func (r *AttendanceRepo) Create(att *models.Attendance) error {
	return r.db.Create(att).Error
}

// Update 按字段更新考勤记录
func (r *AttendanceRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Attendance{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 软删除考勤记录
func (r *AttendanceRepo) Delete(id uint) error {
	return r.db.Delete(&models.Attendance{}, id).Error
}

// MarkLeaveRange 事务内将某员工 [start, end] 区间内每天的考勤标记为 leave：
// 已有记录仅更新 status（保留打卡时间），无记录则新建 leave 记录，已是 leave 则跳过。
// 由请假审批通过时联动调用（同一事务），保证状态更新/余额扣减/考勤标记整体回滚。
func (r *AttendanceRepo) MarkLeaveRange(employeeID uint, start, end time.Time) error {
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if err := r.markLeaveDay(employeeID, d); err != nil {
			return err
		}
	}
	return nil
}

// markLeaveDay 将某员工某天的考勤标记为 leave
func (r *AttendanceRepo) markLeaveDay(employeeID uint, workDate time.Time) error {
	var att models.Attendance
	err := r.db.Where("employee_id = ? AND work_date = ?", employeeID, workDate).First(&att).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 无考勤记录：创建一条 leave 记录（无打卡时间）
		return r.db.Create(&models.Attendance{
			EmployeeID: employeeID,
			WorkDate:   workDate,
			Status:     "leave",
		}).Error
	case err != nil:
		return err
	default:
		// 已有记录：已是 leave 跳过，否则统一覆盖为 leave（保留 check_in/check_out）
		if att.Status == "leave" {
			return nil
		}
		return r.db.Model(&models.Attendance{}).Where("id = ?", att.ID).
			Update("status", "leave").Error
	}
}
