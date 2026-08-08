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
func (r *AttendanceRepo) attendanceQuery(employeeID uint, workDate, month, status string) *gorm.DB {
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
	return q
}

// List 考勤分页列表，支持 employee_id / work_date / month / status 过滤
func (r *AttendanceRepo) List(employeeID uint, workDate, month, status string, page, pageSize int) ([]AttendanceItem, int64, error) {
	var total int64
	if err := r.attendanceQuery(employeeID, workDate, month, status).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AttendanceItem
	err := r.attendanceQuery(employeeID, workDate, month, status).
		Select("attendances.*, e.name AS employee_name, d.name AS department_name").
		Order("attendances.work_date DESC, attendances.employee_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
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
