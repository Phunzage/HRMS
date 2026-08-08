package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// leaveFixture 请假测试环境：部门（含负责人）+ 经理/员工/HR + 对应账号
type leaveFixture struct {
	Dept          *models.Department
	ManagerEmp    *models.Employee
	Emp           *models.Employee
	HRUserID      uint
	ManagerUserID uint
	EmpUserID     uint
}

// setupLeaveFixture 构造请假测试环境：建部门（含 ManagerID）、建经理/员工、建账号（manager/employee/hr）、
// 所有数据落在传入的事务内由调用方回滚
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
	// 回写部门负责人，保证审批引擎自动匹配部门经理逻辑命中
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
	hrUser, err := createTestUser(tx, uniqueStr("hr"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建 HR 账号失败: %v", err)
	}
	// 回写员工 → 账号关联（审批引擎按 employee.user_id 解析审批人账号）
	if err := tx.Model(&models.Employee{}).Where("id = ?", managerEmp.ID).Update("user_id", managerUser.ID).Error; err != nil {
		t.Fatalf("回写经理员工账号关联失败: %v", err)
	}
	if err := tx.Model(&models.Employee{}).Where("id = ?", emp.ID).Update("user_id", empUser.ID).Error; err != nil {
		t.Fatalf("回写员工账号关联失败: %v", err)
	}
	return &leaveFixture{
		Dept:          dept,
		ManagerEmp:    managerEmp,
		Emp:           emp,
		HRUserID:      hrUser.ID,
		ManagerUserID: managerUser.ID,
		EmpUserID:     empUser.ID,
	}
}

// newLeaveSvc 构造绑定测试事务的请假服务（审批引擎与请假共用同一事务）
func newLeaveSvc(tx *gorm.DB) *LeaveService {
	return NewLeaveService(tx, repository.NewLeaveRepo(tx), repository.NewEmployeeRepo(tx))
}

// dateRange 生成 [今天, 今天+n-1] 的日期范围字符串（YYYY-MM-DD），共 n 天
func dateRange(n int) (string, string) {
	start := time.Now()
	return start.Format("2006-01-02"), start.AddDate(0, 0, n-1).Format("2006-01-02")
}

// stepTask 查询实例指定步骤的审批任务
func stepTask(t *testing.T, tx *gorm.DB, inst *models.ApprovalInstance, step int) *models.ApprovalTask {
	t.Helper()
	var task models.ApprovalTask
	if err := tx.Where("instance_id = ? AND step_order = ?", inst.ID, step).First(&task).Error; err != nil {
		t.Fatalf("查询步骤 %d 审批任务失败: %v", step, err)
	}
	return &task
}

// approveTask 模拟 handler 审批链路：事务内审批任务，终审通过时调用业务回调
func approveTask(t *testing.T, tx *gorm.DB, svc *LeaveService, taskID uint, approverUserID uint) *models.ApprovalInstance {
	t.Helper()
	inst, err := ApproveTask(tx, taskID, approverUserID, "同意")
	if err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}
	if inst.Status == "approved" {
		if err := svc.ApplyApproved(tx, inst.BizID); err != nil {
			t.Fatalf("终审通过回调失败: %v", err)
		}
	}
	return inst
}

// TestLeaveBalanceCheck 余额校验：超余额拒绝；刚好够通过且进入审批流保持 pending
func TestLeaveBalanceCheck(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	// 超余额：年假默认额度 10，提交 11 天
	start, end := dateRange(11)
	_, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "超余额",
	})
	var balErr *BalanceInsufficientError
	if !errors.As(err, &balErr) {
		t.Fatalf("期望余额不足错误，实际: %v", err)
	}

	// 刚好够：提交 10 天，走审批流保持 pending 不扣余额
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
	// 审批实例已创建且步骤1任务指派给部门经理
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("审批实例应已创建: %v", err)
	}
	task := stepTask(t, tx, inst, 1)
	if task.ApproverID == nil || *task.ApproverID != fx.ManagerUserID {
		t.Errorf("步骤1审批人应为部门经理，实际 %v", task.ApproverID)
	}
}

// TestLeaveApproveDeductsBalance 审批通过后扣减余额：提交 3 天年假 → 经理+HR 两级通过 → UsedDays == 3
func TestLeaveApproveDeductsBalance(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(3)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "年假",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID) // 经理审批
	approveTask(t, tx, svc, stepTask(t, tx, inst, 2).ID, fx.HRUserID)      // HR 终审

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

// TestLeaveApproveCreatesAttendance 联动验证：终审通过后请假区间每天都有 attendance 且 status==leave；区间外无记录
func TestLeaveApproveCreatesAttendance(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "sick", StartDate: start, EndDate: end, Reason: "病假",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	approveTask(t, tx, svc, stepTask(t, tx, inst, 2).ID, fx.HRUserID)

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

// TestLeaveApproveExistingAttendanceUpdated 已有考勤联动更新：先建 normal → 请假含该天 → 终审通过 → 状态变 leave
func TestLeaveApproveExistingAttendanceUpdated(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

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
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	approveTask(t, tx, svc, stepTask(t, tx, inst, 2).ID, fx.HRUserID)

	att, err := attRepo.FindByEmployeeAndDate(fx.Emp.ID, day)
	if err != nil {
		t.Fatalf("查询考勤失败: %v", err)
	}
	if att.Status != "leave" {
		t.Errorf("期望考勤状态变为 leave，实际 %s", att.Status)
	}
}

// TestLeaveApproveIdempotent 同一任务重复审批被状态机拒绝
func TestLeaveApproveIdempotent(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "幂等",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	task1 := stepTask(t, tx, inst, 1)
	if _, err := ApproveTask(tx, task1.ID, fx.ManagerUserID, "同意"); err != nil {
		t.Fatalf("首次审批应成功: %v", err)
	}
	if _, err := ApproveTask(tx, task1.ID, fx.ManagerUserID, "重复"); !errors.Is(err, ErrApprovalTaskNotPending) {
		t.Fatalf("重复审批应返回 ErrApprovalTaskNotPending，实际: %v", err)
	}
	// 实例未终审，业务单仍 pending
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "pending" {
		t.Errorf("未终审前请假单应保持 pending，实际 %s", lr.Status)
	}
}

// TestLeaveRejectNoDeduction 经理驳回不扣余额，状态变为 rejected
func TestLeaveRejectNoDeduction(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(3)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "驳回测试",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if err := RejectTask(tx, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID, "不需要"); err != nil {
		t.Fatalf("驳回失败: %v", err)
	}
	if err := svc.ApplyRejected(tx, inst.BizID, "不需要"); err != nil {
		t.Fatalf("驳回回调失败: %v", err)
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
	if lr.RejectReason != "不需要" {
		t.Errorf("期望驳回原因记录，实际 %s", lr.RejectReason)
	}
}
