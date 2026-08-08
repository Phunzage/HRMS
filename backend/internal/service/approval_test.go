package service

import (
	"errors"
	"testing"
	"time"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// TestStartApprovalCreatesInstanceAndTask 提交请假后审批实例与步骤1任务建立，并通知部门经理
func TestStartApprovalCreatesInstanceAndTask(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "启动流程",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if inst.Status != "pending" {
		t.Errorf("期望实例 pending，实际 %s", inst.Status)
	}
	if inst.CurrentStep != 1 {
		t.Errorf("期望当前步骤 1，实际 %d", inst.CurrentStep)
	}
	if inst.CreatedBy != fx.EmpUserID {
		t.Errorf("期望提交人为员工账号，实际 %d", inst.CreatedBy)
	}
	// 步骤1任务指派给部门经理账号
	task := stepTask(t, tx, inst, 1)
	if task.ApproverType != "manager" {
		t.Errorf("期望步骤类型 manager，实际 %s", task.ApproverType)
	}
	if task.ApproverID == nil || *task.ApproverID != fx.ManagerUserID {
		t.Errorf("期望步骤1审批人为经理账号，实际 %v", task.ApproverID)
	}
	// 已通知经理
	var n int64
	if err := tx.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", fx.ManagerUserID, NotifTypeLeaveApproval).
		Count(&n).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if n == 0 {
		t.Error("应已发送待审批通知给经理")
	}
}

// TestApprovalHRPoolStepNotifiesAllHR HR 池步骤（HR终审）创建时向所有启用 HR 用户广播待审批通知；
// 步骤1（经理）创建时不应通知 HR
func TestApprovalHRPoolStepNotifiesAllHR(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "HR池通知",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	// 步骤1（经理审批）创建时不应通知 HR
	var hrNotif int64
	if err := tx.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", fx.HRUserID, NotifTypeLeaveApproval).
		Count(&hrNotif).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if hrNotif != 0 {
		t.Errorf("步骤1创建时不应通知 HR，实际通知数 %d", hrNotif)
	}

	// 经理通过 → 推进至步骤2（HR 池）→ 应通知所有启用 HR 用户
	approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	if err := tx.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", fx.HRUserID, NotifTypeLeaveApproval).
		Count(&hrNotif).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if hrNotif == 0 {
		t.Error("HR 池步骤创建后应广播待审批通知给 HR 用户")
	}
}

// TestApprovalManagerToHRFullFlow 完整两级链路：经理通过→实例推进至步骤2→HR 终审通过→业务回调生效
func TestApprovalManagerToHRFullFlow(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "完整链路",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1：经理通过，实例推进至步骤2
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	if inst.Status != "pending" {
		t.Errorf("经理通过后实例应仍 pending，实际 %s", inst.Status)
	}
	if inst.CurrentStep != 2 {
		t.Errorf("期望当前步骤 2，实际 %d", inst.CurrentStep)
	}

	// 步骤2：HR 池任务（未指派具体审批人）
	task2 := stepTask(t, tx, inst, 2)
	if task2.ApproverType != "hr" {
		t.Errorf("期望步骤类型 hr，实际 %s", task2.ApproverType)
	}
	if task2.ApproverID != nil {
		t.Errorf("HR 池任务期望未指派审批人，实际 %v", task2.ApproverID)
	}

	// 步骤2：HR 终审通过 → 实例 approved + 业务回调生效
	inst = approveTask(t, tx, svc, task2.ID, fx.HRUserID)
	if inst.Status != "approved" {
		t.Errorf("期望实例 approved，实际 %s", inst.Status)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "approved" {
		t.Errorf("期望请假单 approved，实际 %s", lr.Status)
	}
	if lr.ApprovedAt == nil {
		t.Error("期望记录审批时间")
	}
	bal, err := repository.NewLeaveRepo(tx).GetBalance(fx.Emp.ID, "annual", time.Now().Year())
	if err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	if bal == nil || bal.UsedDays != 2 {
		t.Errorf("期望已用 2 天，实际 %v", bal)
	}
}

// TestApprovalRejectByHR HR 终审驳回：请假单 rejected，不扣余额
func TestApprovalRejectByHR(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "sick", StartDate: start, EndDate: end, Reason: "HR驳回",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	// 经理通过
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	// HR 驳回
	if err := RejectTask(tx, stepTask(t, tx, inst, 2).ID, fx.HRUserID, "排期冲突"); err != nil {
		t.Fatalf("HR 驳回失败: %v", err)
	}
	if err := svc.ApplyRejected(tx, inst.BizID, "排期冲突"); err != nil {
		t.Fatalf("驳回回调失败: %v", err)
	}

	inst, err = GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if inst.Status != "rejected" {
		t.Errorf("期望实例 rejected，实际 %s", inst.Status)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "rejected" || lr.RejectReason != "排期冲突" {
		t.Errorf("期望请假单 rejected(排期冲突)，实际 %s/%s", lr.Status, lr.RejectReason)
	}
	bal, err := repository.NewLeaveRepo(tx).GetBalance(fx.Emp.ID, "sick", time.Now().Year())
	if err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	if bal != nil && bal.UsedDays != 0 {
		t.Errorf("驳回后不应扣余额，实际已用 %v", bal.UsedDays)
	}
}

// TestApprovalTaskPermissionDenied 越权审批被拒绝：员工试图审批经理的任务 → ErrApprovalNotAllowed
func TestApprovalTaskPermissionDenied(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "personal", StartDate: start, EndDate: end, Reason: "越权测试",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	// 申请人（员工）试图审批经理的任务
	_, err = ApproveTask(tx, stepTask(t, tx, inst, 1).ID, fx.EmpUserID, "越权")
	if !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("期望 ErrApprovalNotAllowed，实际: %v", err)
	}
}

// TestApprovalCancelPending 申请人可取消 pending 实例；非申请人/已处理实例不可取消
func TestApprovalCancelPending(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "取消测试",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	// 非申请人取消被拒绝
	if err := CancelApproval(tx, inst.ID, fx.ManagerUserID); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("非申请人取消应返回 ErrApprovalNotAllowed，实际: %v", err)
	}
	// 申请人取消成功
	if err := CancelApproval(tx, inst.ID, fx.EmpUserID); err != nil {
		t.Fatalf("申请人取消失败: %v", err)
	}
	inst, err = GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if inst.Status != "cancelled" {
		t.Errorf("期望实例 cancelled，实际 %s", inst.Status)
	}
	// 已取消实例再次取消被拒绝
	if err := CancelApproval(tx, inst.ID, fx.EmpUserID); !errors.Is(err, ErrApprovalNotPending) {
		t.Fatalf("已取消实例取消应返回 ErrApprovalNotPending，实际: %v", err)
	}
}

// TestApprovalCancelViaLeaveService 通过请假服务取消：请假单与审批实例同步 cancelled
func TestApprovalCancelViaLeaveService(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "服务取消",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	if err := svc.Cancel(fx.EmpUserID, leave.ID); err != nil {
		t.Fatalf("取消请假失败: %v", err)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "cancelled" {
		t.Errorf("期望请假单 cancelled，实际 %s", lr.Status)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if inst.Status != "cancelled" {
		t.Errorf("期望实例 cancelled，实际 %s", inst.Status)
	}
}

// TestListMyTasks 待办/历史列表：经理看到步骤1，HR 在步骤2 创建后看到，历史随处理更新
func TestListMyTasks(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "待办列表",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 经理待办 1 条
	items, err := ListMyTasks(tx, fx.ManagerUserID, "pending")
	if err != nil {
		t.Fatalf("查询经理待办失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("期望经理 1 条待办，实际 %d", len(items))
	}
	if items[0].BizType != "leave" || items[0].BizID != leave.ID {
		t.Errorf("待办业务信息错误: %+v", items[0])
	}
	if items[0].ApplicantName == "" {
		t.Error("期望带出申请人姓名")
	}

	// HR 此时无待办（步骤2 未创建）
	hrItems, err := ListMyTasks(tx, fx.HRUserID, "pending")
	if err != nil {
		t.Fatalf("查询 HR 待办失败: %v", err)
	}
	if len(hrItems) != 0 {
		t.Fatalf("期望 HR 0 条待办，实际 %d", len(hrItems))
	}

	// 经理通过后：经理待办清空，HR 出现待办
	approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	items, err = ListMyTasks(tx, fx.ManagerUserID, "pending")
	if err != nil {
		t.Fatalf("查询经理待办失败: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("期望经理待办清空，实际 %d", len(items))
	}
	hrItems, err = ListMyTasks(tx, fx.HRUserID, "pending")
	if err != nil {
		t.Fatalf("查询 HR 待办失败: %v", err)
	}
	if len(hrItems) != 1 {
		t.Fatalf("期望 HR 1 条待办，实际 %d", len(hrItems))
	}

	// HR 终审通过：HR 待办清空、历史 +1
	approveTask(t, tx, svc, stepTask(t, tx, inst, 2).ID, fx.HRUserID)
	hrItems, err = ListMyTasks(tx, fx.HRUserID, "pending")
	if err != nil {
		t.Fatalf("查询 HR 待办失败: %v", err)
	}
	if len(hrItems) != 0 {
		t.Fatalf("期望 HR 待办清空，实际 %d", len(hrItems))
	}
	hrItems, err = ListMyTasks(tx, fx.HRUserID, "approved")
	if err != nil {
		t.Fatalf("查询 HR 历史失败: %v", err)
	}
	if len(hrItems) != 1 {
		t.Fatalf("期望 HR 1 条已通过历史，实际 %d", len(hrItems))
	}
}
