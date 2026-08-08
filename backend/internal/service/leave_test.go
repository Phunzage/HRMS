package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// leaveFixture 请假测试环境：部门（含负责人）+ 经理/员工 + 对应账号
type leaveFixture struct {
	Dept          *models.Department
	ManagerEmp    *models.Employee
	Emp           *models.Employee
	ManagerUserID uint
	EmpUserID     uint
}

// setupLeaveFixture 构造请假测试环境：建部门（含 ManagerID）、建经理员工、建员工（设 DepartmentID）、
// 建用户（role employee/manager，bcrypt），所有数据落在传入的事务内由调用方回滚
func setupLeaveFixture(t *testing.T, tx *gorm.DB) *leaveFixture {
	t.Helper()
	dept, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	managerEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("经理"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建经理员工失败: %v", err)
	}
	// 回写部门负责人，保证自动匹配审批人逻辑命中
	if err := tx.Model(&models.Department{}).Where("id = ?", dept.ID).Update("manager_id", managerEmp.ID).Error; err != nil {
		t.Fatalf("设置部门负责人失败: %v", err)
	}
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("员工"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}
	managerUser, err := createTestUser(tx, uniqueStr("mgr"), "pass123456", "manager", &managerEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建经理账号失败: %v", err)
	}
	empUser, err := createTestUser(tx, uniqueStr("emp"), "pass123456", "employee", &emp.ID, 1)
	if err != nil {
		t.Fatalf("创建员工账号失败: %v", err)
	}
	return &leaveFixture{
		Dept:          dept,
		ManagerEmp:    managerEmp,
		Emp:           emp,
		ManagerUserID: managerUser.ID,
		EmpUserID:     empUser.ID,
	}
}

// dateRange 生成 [今天, 今天+n-1] 的日期范围字符串（YYYY-MM-DD），共 n 天
func dateRange(n int) (string, string) {
	start := time.Now()
	return start.Format("2006-01-02"), start.AddDate(0, 0, n-1).Format("2006-01-02")
}

// TestLeaveBalanceCheck 余额校验：超余额拒绝；刚好够通过
func TestLeaveBalanceCheck(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	// 超余额：年假默认额度 10，提交 11 天
	start, end := dateRange(11)
	_, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "超余额",
	})
	var balErr *BalanceInsufficientError
	if !errors.As(err, &balErr) {
		t.Fatalf("期望余额不足错误，实际: %v", err)
	}

	// 刚好够：提交 10 天，有审批人保持 pending 不扣余额
	start, end = dateRange(10)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "刚好够",
	})
	if err != nil {
		t.Fatalf("余额刚好够应通过，实际: %v", err)
	}
	if leave.Status != "pending" {
		t.Errorf("期望 pending，实际 %s", leave.Status)
	}
}

// TestLeaveApproveDeductsBalance 审批通过后扣减余额：提交 3 天年假 → approve → UsedDays == 3
func TestLeaveApproveDeductsBalance(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	start, end := dateRange(3)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "年假",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Approve("manager", fx.ManagerUserID, leave.ID); err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}

	bal, err := repository.NewLeaveRepo(tx).GetBalance(fx.Emp.ID, "annual", time.Now().Year())
	if err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	if bal == nil {
		t.Fatal("余额记录不存在")
	}
	if bal.UsedDays != 3 {
		t.Errorf("期望已用 3 天，实际 %v", bal.UsedDays)
	}
}

// TestLeaveApproveCreatesAttendance 联动验证：approve 后请假区间每天都有 attendance 且 status==leave；区间外无记录
func TestLeaveApproveCreatesAttendance(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "sick", StartDate: start, EndDate: end, Reason: "病假",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Approve("manager", fx.ManagerUserID, leave.ID); err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}

	attRepo := repository.NewAttendanceRepo(tx)
	for _, day := range []string{start, end} {
		att, err := attRepo.FindByEmployeeAndDate(fx.Emp.ID, day)
		if err != nil {
			t.Fatalf("查询 %s 考勤失败: %v", day, err)
		}
		if att.Status != "leave" {
			t.Errorf("%s 状态期望 leave，实际 %s", day, att.Status)
		}
	}
	// 区间外（第 4 天）无记录
	outside := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	if _, err := attRepo.FindByEmployeeAndDate(fx.Emp.ID, outside); err == nil {
		t.Errorf("区间外 %s 不应有考勤记录", outside)
	}
}

// TestLeaveApproveExistingAttendanceUpdated 已有考勤联动更新：先建 normal → 请假含该天 → approve → 状态变 leave
func TestLeaveApproveExistingAttendanceUpdated(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	workDate := time.Now()
	attRepo := repository.NewAttendanceRepo(tx)
	if err := attRepo.Create(&models.Attendance{
		EmployeeID: fx.Emp.ID, WorkDate: workDate, Status: "normal",
	}); err != nil {
		t.Fatalf("创建 normal 考勤失败: %v", err)
	}

	day := workDate.Format("2006-01-02")
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "personal", StartDate: day, EndDate: day, Reason: "事假",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Approve("manager", fx.ManagerUserID, leave.ID); err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}

	att, err := attRepo.FindByEmployeeAndDate(fx.Emp.ID, day)
	if err != nil {
		t.Fatalf("查询考勤失败: %v", err)
	}
	if att.Status != "leave" {
		t.Errorf("期望考勤状态变为 leave，实际 %s", att.Status)
	}
}

// TestLeaveApproveIdempotent 重复审批被状态机拒绝
func TestLeaveApproveIdempotent(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "幂等",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Approve("manager", fx.ManagerUserID, leave.ID); err != nil {
		t.Fatalf("首次审批应成功: %v", err)
	}
	if err := svc.Approve("manager", fx.ManagerUserID, leave.ID); !errors.Is(err, ErrLeaveNotPending) {
		t.Fatalf("重复审批应返回 ErrLeaveNotPending，实际: %v", err)
	}
}

// TestLeaveRejectNoDeduction 驳回不扣余额，状态变为 rejected
func TestLeaveRejectNoDeduction(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := NewLeaveService(repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))

	start, end := dateRange(3)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "驳回测试",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Reject("manager", fx.ManagerUserID, leave.ID, "不需要"); err != nil {
		t.Fatalf("驳回失败: %v", err)
	}

	bal, err := repository.NewLeaveRepo(tx).GetBalance(fx.Emp.ID, "annual", time.Now().Year())
	if err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	if bal != nil && bal.UsedDays != 0 {
		t.Errorf("驳回后余额不应扣减，实际已用 %v", bal.UsedDays)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "rejected" {
		t.Errorf("期望状态 rejected，实际 %s", lr.Status)
	}
}
