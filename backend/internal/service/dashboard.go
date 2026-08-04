package service

import (
	"time"

	"hrms/backend/internal/repository"
)

// DeptCount 部门人数分布项
type DeptCount struct {
	DepartmentName string `json:"department_name"`
	Count          int64  `json:"count"`
}

// MonthCount 按月计数项（招聘趋势）
type MonthCount struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

// MonthAmount 按月金额项（工资总额趋势）
type MonthAmount struct {
	Month    string  `json:"month"`
	TotalNet float64 `json:"total_net"`
}

// DashboardStats 仪表盘统计结果
type DashboardStats struct {
	TotalEmployees         int64         `json:"total_employees"`           // 员工总数
	ActiveEmployees        int64         `json:"active_employees"`          // 在职员工数
	MonthNewEmployees      int64         `json:"month_new_employees"`       // 本月新增员工数
	MonthResigned          int64         `json:"month_resigned"`            // 本月离职数
	MonthAttendanceRate    float64       `json:"month_attendance_rate"`     // 本月出勤率
	TodayLeaveCount        int64         `json:"today_leave_count"`         // 今日请假人数
	PendingLeaveCount      int64         `json:"pending_leave_count"`       // 待审批请假数
	PendingSalaryCount     int64         `json:"pending_salary_count"`      // 待确认工资单数
	OpenJobCount           int64         `json:"open_job_count"`            // 在招职位数
	CandidateCount         int64         `json:"candidate_count"`           // 候选人数
	DepartmentDistribution []DeptCount   `json:"department_distribution"`   // 部门人数分布
	RecruitTrend           []MonthCount  `json:"recruit_trend"`             // 近6个月招聘趋势
	SalaryTrend            []MonthAmount `json:"salary_trend"`              // 近6个月工资总额
}

// DashboardService 仪表盘业务服务
type DashboardService struct {
	repo *repository.DashboardRepo
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(repo *repository.DashboardRepo) *DashboardService {
	return &DashboardService{repo: repo}
}

// Stats 汇总仪表盘统计
func (s *DashboardService) Stats() (*DashboardStats, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	trendStart := monthStart.AddDate(0, -5, 0) // 近 6 个月（含当月）
	startMonth := trendStart.Format("2006-01")

	stats := &DashboardStats{}

	var err error
	if stats.TotalEmployees, err = s.repo.CountEmployees(); err != nil {
		return nil, err
	}
	if stats.ActiveEmployees, err = s.repo.CountActiveEmployees(); err != nil {
		return nil, err
	}
	if stats.MonthNewEmployees, err = s.repo.CountMonthNewEmployees(monthStart); err != nil {
		return nil, err
	}
	if stats.MonthResigned, err = s.repo.CountMonthResigned(monthStart); err != nil {
		return nil, err
	}
	if stats.MonthAttendanceRate, err = s.repo.MonthAttendanceRate(monthStart, nextMonthStart); err != nil {
		return nil, err
	}
	if stats.TodayLeaveCount, err = s.repo.TodayLeaveCount(today); err != nil {
		return nil, err
	}
	if stats.PendingLeaveCount, err = s.repo.PendingLeaveCount(); err != nil {
		return nil, err
	}
	if stats.PendingSalaryCount, err = s.repo.PendingSalaryCount(); err != nil {
		return nil, err
	}
	if stats.OpenJobCount, err = s.repo.OpenJobCount(); err != nil {
		return nil, err
	}
	if stats.CandidateCount, err = s.repo.CandidateCount(); err != nil {
		return nil, err
	}

	deptRows, err := s.repo.DepartmentDistribution()
	if err != nil {
		return nil, err
	}
	stats.DepartmentDistribution = make([]DeptCount, 0, len(deptRows))
	for _, r := range deptRows {
		stats.DepartmentDistribution = append(stats.DepartmentDistribution, DeptCount{
			DepartmentName: r.DepartmentName,
			Count:          r.Count,
		})
	}

	recruitRows, err := s.repo.RecruitTrend(trendStart)
	if err != nil {
		return nil, err
	}
	stats.RecruitTrend = fillMonthCounts(lastMonths(6), recruitRows)

	salaryRows, err := s.repo.SalaryTrend(startMonth)
	if err != nil {
		return nil, err
	}
	stats.SalaryTrend = fillMonthAmounts(lastMonths(6), salaryRows)

	return stats, nil
}

// lastMonths 返回含当月在内的近 n 个月字符串列表（升序）
func lastMonths(n int) []string {
	months := make([]string, 0, n)
	now := time.Now()
	for i := n - 1; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		months = append(months, t.Format("2006-01"))
	}
	return months
}

// fillMonthCounts 补齐近 6 个月缺失月份（计数补 0）
func fillMonthCounts(months []string, rows []repository.MonthCountRow) []MonthCount {
	byMonth := make(map[string]int64, len(rows))
	for _, r := range rows {
		byMonth[r.Month] = r.Count
	}
	result := make([]MonthCount, 0, len(months))
	for _, m := range months {
		result = append(result, MonthCount{Month: m, Count: byMonth[m]})
	}
	return result
}

// fillMonthAmounts 补齐近 6 个月缺失月份（金额补 0）
func fillMonthAmounts(months []string, rows []repository.MonthAmountRow) []MonthAmount {
	byMonth := make(map[string]float64, len(rows))
	for _, r := range rows {
		byMonth[r.Month] = r.TotalNet
	}
	result := make([]MonthAmount, 0, len(months))
	for _, m := range months {
		result = append(result, MonthAmount{Month: m, TotalNet: byMonth[m]})
	}
	return result
}
