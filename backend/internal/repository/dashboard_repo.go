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

// applyDeptScope 按数据范围过滤：deptIDs 为空表示全量不过滤；col 为部门列名
// 多表查询时 col 需限定表名（如 e.department_id）避免列名歧义
func applyDeptScope(q *gorm.DB, deptIDs []uint, col string) *gorm.DB {
	if len(deptIDs) > 0 {
		q = q.Where(col+" IN ?", deptIDs)
	}
	return q
}

// CountEmployees 员工总数（按部门范围过滤）
func (r *DashboardRepo) CountEmployees(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.Employee{})
	q = applyDeptScope(q, deptIDs, "department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// CountActiveEmployees 在职员工数（在职 + 试用，按部门范围过滤）
func (r *DashboardRepo) CountActiveEmployees(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.Employee{}).Where("status IN ?", []int{1, 3})
	q = applyDeptScope(q, deptIDs, "department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// CountMonthNewEmployees 本月新增员工数（按 created_at，按部门范围过滤）
func (r *DashboardRepo) CountMonthNewEmployees(monthStart time.Time, deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.Employee{}).Where("created_at >= ?", monthStart)
	q = applyDeptScope(q, deptIDs, "department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// CountMonthResigned 本月离职数（status=离职且本月内变更，按部门范围过滤）
func (r *DashboardRepo) CountMonthResigned(monthStart time.Time, deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.Employee{}).Where("status = ? AND updated_at >= ?", 2, monthStart)
	q = applyDeptScope(q, deptIDs, "department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// MonthAttendanceRate 本月出勤率 = 正常出勤人次 / 总考勤人次
// 注意：批次 11 起考勤新增 leave 状态（请假联动标记），此处统计口径维持原逻辑不变，
// leave 不计入「正常出勤」，作为请假日参与总数分母即可，无需单独剔除。
// JOIN employees 按部门范围过滤（status 列需限定表名避免歧义）
func (r *DashboardRepo) MonthAttendanceRate(monthStart, nextMonthStart time.Time, deptIDs []uint) (float64, error) {
	base := r.db.Model(&models.Attendance{}).
		Joins("JOIN employees e ON e.id = attendances.employee_id").
		Where("attendances.work_date >= ? AND attendances.work_date < ?", monthStart, nextMonthStart)
	base = applyDeptScope(base, deptIDs, "e.department_id")
	var total, normal int64
	if err := base.Count(&total).Error; err != nil {
		return 0, err
	}
	if err := base.Where("attendances.status = ?", "normal").Count(&normal).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	return float64(normal) / float64(total), nil
}

// TodayLeaveCount 今日请假人数（审批通过且覆盖今日的去重员工数，JOIN employees 按部门范围过滤）
func (r *DashboardRepo) TodayLeaveCount(today time.Time, deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.LeaveRequest{}).
		Joins("JOIN employees e ON e.id = leave_requests.employee_id").
		Where("leave_requests.status = ? AND leave_requests.start_date <= ? AND leave_requests.end_date >= ?", "approved", today, today).
		Distinct("leave_requests.employee_id")
	q = applyDeptScope(q, deptIDs, "e.department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// PendingLeaveCount 待审批请假数（JOIN employees 按部门范围过滤）
func (r *DashboardRepo) PendingLeaveCount(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.LeaveRequest{}).
		Joins("JOIN employees e ON e.id = leave_requests.employee_id").
		Where("leave_requests.status = ?", "pending")
	q = applyDeptScope(q, deptIDs, "e.department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// PendingSalaryCount 待确认工资单数（draft，JOIN employees 按部门范围过滤）
func (r *DashboardRepo) PendingSalaryCount(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.SalaryRecord{}).
		Joins("JOIN employees e ON e.id = salary_records.employee_id").
		Where("salary_records.status = ?", "draft")
	q = applyDeptScope(q, deptIDs, "e.department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// OpenJobCount 在招职位数（open，按部门范围过滤）
func (r *DashboardRepo) OpenJobCount(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.JobOpening{}).Where("status = ?", "open")
	q = applyDeptScope(q, deptIDs, "department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// CandidateCount 候选人数（JOIN job_openings 按部门范围过滤）
func (r *DashboardRepo) CandidateCount(deptIDs []uint) (int64, error) {
	q := r.db.Model(&models.Candidate{}).
		Joins("JOIN job_openings jo ON jo.id = candidates.job_opening_id")
	q = applyDeptScope(q, deptIDs, "jo.department_id")
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// DepartmentDistribution 部门人数分布（按员工归属部门分组，仅统计范围内部门）
func (r *DashboardRepo) DepartmentDistribution(deptIDs []uint) ([]DeptCountRow, error) {
	q := r.db.Model(&models.Employee{}).
		Select("d.name AS department_name, COUNT(*) AS count").
		Joins("LEFT JOIN departments d ON d.id = employees.department_id")
	q = applyDeptScope(q, deptIDs, "employees.department_id")
	rows := make([]DeptCountRow, 0)
	err := q.Group("d.name").
		Order("count DESC").
		Scan(&rows).Error
	return rows, err
}

// RecruitTrend 招聘趋势（按候选人 created_at 月份分组，近 start 之后，JOIN job_openings 按部门范围过滤）
func (r *DashboardRepo) RecruitTrend(start time.Time, deptIDs []uint) ([]MonthCountRow, error) {
	q := r.db.Model(&models.Candidate{}).
		Select("to_char(candidates.created_at, 'YYYY-MM') AS month, COUNT(*) AS count").
		Joins("JOIN job_openings jo ON jo.id = candidates.job_opening_id").
		Where("candidates.created_at >= ?", start)
	q = applyDeptScope(q, deptIDs, "jo.department_id")
	rows := make([]MonthCountRow, 0)
	err := q.Group("to_char(candidates.created_at, 'YYYY-MM')").
		Order("month ASC").
		Scan(&rows).Error
	return rows, err
}

// SalaryTrend 工资总额趋势（confirmed/paid 工资单按月分组，近 startMonth 之后，JOIN employees 按部门范围过滤）
func (r *DashboardRepo) SalaryTrend(startMonth string, deptIDs []uint) ([]MonthAmountRow, error) {
	q := r.db.Model(&models.SalaryRecord{}).
		Select("salary_records.month, SUM(salary_records.net_salary) AS total_net").
		Joins("JOIN employees e ON e.id = salary_records.employee_id").
		Where("salary_records.status IN ? AND salary_records.month >= ?", []string{"confirmed", "paid"}, startMonth)
	q = applyDeptScope(q, deptIDs, "e.department_id")
	rows := make([]MonthAmountRow, 0)
	err := q.Group("salary_records.month").
		Order("month ASC").
		Scan(&rows).Error
	return rows, err
}
