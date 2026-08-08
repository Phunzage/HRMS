package repository

import (
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// DeptCountRow 部门人数分布行
type DeptCountRow struct {
	DepartmentName string `gorm:"column:department_name" json:"department_name"`
	Count          int64  `gorm:"column:count" json:"count"`
}

// MonthCountRow 按月计数行（招聘趋势）
type MonthCountRow struct {
	Month string `gorm:"column:month" json:"month"`
	Count int64  `gorm:"column:count" json:"count"`
}

// MonthAmountRow 按月金额行（工资总额趋势）
type MonthAmountRow struct {
	Month    string  `gorm:"column:month" json:"month"`
	TotalNet float64 `gorm:"column:total_net" json:"total_net"`
}

// DashboardRepo 仪表盘数据仓库
type DashboardRepo struct {
	db *gorm.DB
}

// NewDashboardRepo 创建仪表盘仓库
func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

// CountEmployees 员工总数
func (r *DashboardRepo) CountEmployees() (int64, error) {
	var n int64
	err := r.db.Model(&models.Employee{}).Count(&n).Error
	return n, err
}

// CountActiveEmployees 在职员工数（在职 + 试用）
func (r *DashboardRepo) CountActiveEmployees() (int64, error) {
	var n int64
	err := r.db.Model(&models.Employee{}).Where("status IN ?", []int{1, 3}).Count(&n).Error
	return n, err
}

// CountMonthNewEmployees 本月新增员工数（按 created_at）
func (r *DashboardRepo) CountMonthNewEmployees(monthStart time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&models.Employee{}).Where("created_at >= ?", monthStart).Count(&n).Error
	return n, err
}

// CountMonthResigned 本月离职数（status=离职且本月内变更）
func (r *DashboardRepo) CountMonthResigned(monthStart time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&models.Employee{}).
		Where("status = ? AND updated_at >= ?", 2, monthStart).Count(&n).Error
	return n, err
}

// MonthAttendanceRate 本月出勤率 = 正常出勤人次 / 总考勤人次
// 注意：批次 11 起考勤新增 leave 状态（请假联动标记），此处统计口径维持原逻辑不变，
// leave 不计入「正常出勤」，作为请假日参与总数分母即可，无需单独剔除。
func (r *DashboardRepo) MonthAttendanceRate(monthStart, nextMonthStart time.Time) (float64, error) {
	base := r.db.Model(&models.Attendance{}).
		Where("work_date >= ? AND work_date < ?", monthStart, nextMonthStart)
	var total, normal int64
	if err := base.Count(&total).Error; err != nil {
		return 0, err
	}
	if err := base.Where("status = ?", "normal").Count(&normal).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	return float64(normal) / float64(total), nil
}

// TodayLeaveCount 今日请假人数（审批通过且覆盖今日的去重员工数）
func (r *DashboardRepo) TodayLeaveCount(today time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&models.LeaveRequest{}).
		Where("status = ? AND start_date <= ? AND end_date >= ?", "approved", today, today).
		Distinct("employee_id").
		Count(&n).Error
	return n, err
}

// PendingLeaveCount 待审批请假数
func (r *DashboardRepo) PendingLeaveCount() (int64, error) {
	var n int64
	err := r.db.Model(&models.LeaveRequest{}).Where("status = ?", "pending").Count(&n).Error
	return n, err
}

// PendingSalaryCount 待确认工资单数（draft）
func (r *DashboardRepo) PendingSalaryCount() (int64, error) {
	var n int64
	err := r.db.Model(&models.SalaryRecord{}).Where("status = ?", "draft").Count(&n).Error
	return n, err
}

// OpenJobCount 在招职位数（open）
func (r *DashboardRepo) OpenJobCount() (int64, error) {
	var n int64
	err := r.db.Model(&models.JobOpening{}).Where("status = ?", "open").Count(&n).Error
	return n, err
}

// CandidateCount 候选人数
func (r *DashboardRepo) CandidateCount() (int64, error) {
	var n int64
	err := r.db.Model(&models.Candidate{}).Count(&n).Error
	return n, err
}

// DepartmentDistribution 部门人数分布（按员工归属部门分组）
func (r *DashboardRepo) DepartmentDistribution() ([]DeptCountRow, error) {
	var rows []DeptCountRow
	err := r.db.Model(&models.Employee{}).
		Select("d.name AS department_name, COUNT(*) AS count").
		Joins("LEFT JOIN departments d ON d.id = employees.department_id").
		Group("d.name").
		Order("count DESC").
		Scan(&rows).Error
	return rows, err
}

// RecruitTrend 招聘趋势（按候选人 created_at 月份分组，近 start 之后）
func (r *DashboardRepo) RecruitTrend(start time.Time) ([]MonthCountRow, error) {
	var rows []MonthCountRow
	err := r.db.Model(&models.Candidate{}).
		Select("to_char(created_at, 'YYYY-MM') AS month, COUNT(*) AS count").
		Where("created_at >= ?", start).
		Group("to_char(created_at, 'YYYY-MM')").
		Order("month ASC").
		Scan(&rows).Error
	return rows, err
}

// SalaryTrend 工资总额趋势（confirmed/paid 工资单按月分组，近 startMonth 之后）
func (r *DashboardRepo) SalaryTrend(startMonth string) ([]MonthAmountRow, error) {
	var rows []MonthAmountRow
	err := r.db.Model(&models.SalaryRecord{}).
		Select("month, SUM(net_salary) AS total_net").
		Where("status IN ? AND month >= ?", []string{"confirmed", "paid"}, startMonth).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	return rows, err
}
