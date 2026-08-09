package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// findHrRoleID 查询 hr 角色 ID（角色池测试用）
func findHrRoleID(t *testing.T, tx *gorm.DB) uint {
	t.Helper()
	var hrRole models.Role
	if err := tx.Where("code = ?", "hr").First(&hrRole).Error; err != nil {
		t.Fatalf("查询 hr 角色失败: %v", err)
	}
	return hrRole.ID
}

// createNoPermUser 创建无任何审批权限的自定义角色用户（模拟 hr 取消 leave:approve 后的无权限账号）
func createNoPermUser(t *testing.T, tx *gorm.DB) *models.User {
	t.Helper()
	role := &models.Role{Name: "无审批权限", Code: uniqueStr("norole"), Description: "测试无审批权限角色"}
	if err := tx.Create(role).Error; err != nil {
		t.Fatalf("创建无权限角色失败: %v", err)
	}
	u, err := createTestUser(tx, uniqueStr("noperm"), "pass123456", role.Code, nil, 1)
	if err != nil {
		t.Fatalf("创建无权限账号失败: %v", err)
	}
	return u
}

// TestResolveManagerUserIDRecursive 递归经理解析：员工在无经理的子部门 → 沿 parent 链向上找到父部门经理
func TestResolveManagerUserIDRecursive(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	// 父部门（有经理）
	parentDept := &models.Department{Name: uniqueStr("父部门"), ParentID: 0, SortOrder: 1}
	if err := tx.Create(parentDept).Error; err != nil {
		t.Fatalf("创建父部门失败: %v", err)
	}
	managerEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("经理"), parentDept.ID, 0)
	if err != nil {
		t.Fatalf("创建经理员工失败: %v", err)
	}
	if err := tx.Model(&models.Department{}).Where("id = ?", parentDept.ID).Update("manager_id", managerEmp.ID).Error; err != nil {
		t.Fatalf("设置父部门负责人失败: %v", err)
	}
	managerUser, err := createTestUser(tx, uniqueStr("mgr"), "pass123456", "manager", &managerEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建经理账号失败: %v", err)
	}
	if err := tx.Model(&models.Employee{}).Where("id = ?", managerEmp.ID).Update("user_id", managerUser.ID).Error; err != nil {
		t.Fatalf("回写经理员工账号关联失败: %v", err)
	}

	// 中间部门（无经理）→ 孙部门（员工所在，无经理）
	midDept := &models.Department{Name: uniqueStr("中间部"), ParentID: parentDept.ID, SortOrder: 1}
	if err := tx.Create(midDept).Error; err != nil {
		t.Fatalf("创建中间部门失败: %v", err)
	}
	grandDept := &models.Department{Name: uniqueStr("孙部门"), ParentID: midDept.ID, SortOrder: 1}
	if err := tx.Create(grandDept).Error; err != nil {
		t.Fatalf("创建孙部门失败: %v", err)
	}
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("员工"), grandDept.ID, 0)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	mgrUserID, err := ResolveManagerUserID(tx, emp.ID)
	if err != nil {
		t.Fatalf("递归解析经理失败: %v", err)
	}
	if mgrUserID == nil || *mgrUserID != managerUser.ID {
		t.Fatalf("期望递归解析到父部门经理 %d，实际 %v", managerUser.ID, mgrUserID)
	}
}

// TestResolveManagerUserIDNoManager 全链无经理返回 nil（不硬指派）
func TestResolveManagerUserIDNoManager(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	topDept := &models.Department{Name: uniqueStr("顶层部"), ParentID: 0, SortOrder: 1}
	if err := tx.Create(topDept).Error; err != nil {
		t.Fatalf("创建顶层部门失败: %v", err)
	}
	childDept := &models.Department{Name: uniqueStr("子部门"), ParentID: topDept.ID, SortOrder: 1}
	if err := tx.Create(childDept).Error; err != nil {
		t.Fatalf("创建子部门失败: %v", err)
	}
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("员工"), childDept.ID, 0)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	mgrUserID, err := ResolveManagerUserID(tx, emp.ID)
	if err != nil {
		t.Fatalf("解析经理失败: %v", err)
	}
	if mgrUserID != nil {
		t.Fatalf("全链无经理应返回 nil，实际 %v", mgrUserID)
	}
}

// TestUnassignedManagerTaskOwnership 未指派 manager 任务归属：
// 部门无经理时 manager 步骤 approver_id=NULL，拥有 leave:approve 的用户可见可审，无权限用户（含 hr 取消权限）不可见不可审
func TestUnassignedManagerTaskOwnership(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	// 无经理部门 + 岗位
	dept := &models.Department{Name: uniqueStr("无经理部"), ParentID: 0, SortOrder: 1}
	if err := tx.Create(dept).Error; err != nil {
		t.Fatalf("创建无经理部门失败: %v", err)
	}
	pos := &models.Position{Name: uniqueStr("测试岗"), Code: uniqueStr("pos_"), DepartmentID: dept.ID}
	if err := tx.Create(pos).Error; err != nil {
		t.Fatalf("创建岗位失败: %v", err)
	}
	// 申请人（employee 角色）
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("申请人"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建申请人员工失败: %v", err)
	}
	empUser, err := createTestUser(tx, uniqueStr("emp"), "pass123456", "employee", &emp.ID, 1)
	if err != nil {
		t.Fatalf("创建申请人账号失败: %v", err)
	}
	if err := tx.Model(&models.Employee{}).Where("id = ?", emp.ID).Update("user_id", empUser.ID).Error; err != nil {
		t.Fatalf("回写员工账号关联失败: %v", err)
	}
	// 有审批权限的 hr；无审批权限的自定义角色账号（模拟 hr 取消 leave:approve）
	if err := grantRolePermission(tx, "hr", "leave:approve"); err != nil {
		t.Fatalf("授予 hr leave:approve 失败: %v", err)
	}
	hrUser, err := createTestUser(tx, uniqueStr("hr"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建 hr 账号失败: %v", err)
	}
	noPermUser := createNoPermUser(t, tx)

	// 流程改为单级 manager（无经理 → 未指派）
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", Name: "部门经理审批"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	svc := newLeaveSvc(tx)
	start, end := dateRange(1)
	leave, err := svc.Create(empUser.ID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "未指派经理",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	task1 := stepTask(t, tx, inst, 1)
	if task1.ApproverType != "manager" || task1.ApproverID != nil {
		t.Fatalf("期望未指派 manager 任务（approver_id=NULL），实际 type=%s approver=%v", task1.ApproverType, task1.ApproverID)
	}

	// 可见性：有 leave:approve 的 hr 可见；无权限用户/申请人不可见
	items, err := ListMyTasks(tx, hrUser.ID, "pending")
	if err != nil {
		t.Fatalf("查询 hr 待办失败: %v", err)
	}
	if findTaskByStep(items, inst.ID, 1) == nil {
		t.Error("拥有 leave:approve 的 hr 应看到未指派 manager 任务")
	}
	items, err = ListMyTasks(tx, noPermUser.ID, "pending")
	if err != nil {
		t.Fatalf("查询无权限用户待办失败: %v", err)
	}
	if findTaskByStep(items, inst.ID, 1) != nil {
		t.Error("无 leave:approve 的用户不应看到未指派 manager 任务")
	}
	items, err = ListMyTasks(tx, empUser.ID, "pending")
	if err != nil {
		t.Fatalf("查询申请人待办失败: %v", err)
	}
	if findTaskByStep(items, inst.ID, 1) != nil {
		t.Error("申请人（无 leave:approve）不应看到未指派 manager 任务")
	}

	// 角标：hr=1，无权限用户=0
	cnt, err := CountMyPendingTasks(tx, hrUser.ID)
	if err != nil {
		t.Fatalf("查询 hr 待办计数失败: %v", err)
	}
	if cnt != 1 {
		t.Errorf("hr 期望待办计数 1，实际 %d", cnt)
	}
	cnt, err = CountMyPendingTasks(tx, noPermUser.ID)
	if err != nil {
		t.Fatalf("查询无权限用户待办计数失败: %v", err)
	}
	if cnt != 0 {
		t.Errorf("无权限用户期望待办计数 0，实际 %d", cnt)
	}

	// 越权：无权限用户审批被拒
	if _, err := ApproveTask(tx, task1.ID, noPermUser.ID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("无权限用户审批应返回 ErrApprovalNotAllowed，实际: %v", err)
	}

	// hr 审批通过 → 实例终审通过
	inst = approveTask(t, tx, svc, task1.ID, hrUser.ID)
	if inst.Status != "approved" {
		t.Fatalf("期望实例 approved，实际 %s", inst.Status)
	}

	// 通知：hr（有权限）收到；无权限用户未收到
	var n1, n2 int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", hrUser.ID, NotifTypeLeaveApproval).Count(&n1).Error; err != nil {
		t.Fatalf("查询 hr 通知失败: %v", err)
	}
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", noPermUser.ID, NotifTypeLeaveApproval).Count(&n2).Error; err != nil {
		t.Fatalf("查询无权限用户通知失败: %v", err)
	}
	if n1 == 0 {
		t.Error("hr（有 leave:approve）应收到未指派经理审批通知")
	}
	if n2 != 0 {
		t.Error("无 leave:approve 的用户不应收到未指派经理审批通知")
	}
}

// TestRolePoolFullFlow 角色池端到端：manager → role(hr)，
// 覆盖 role 池任务创建（ApproverRoleID 写入、未指派）、可见性、角标、越权拒绝、回填审批人、终审通过
func TestRolePoolFullFlow(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	hrRoleID := findHrRoleID(t, tx)
	svc := newLeaveSvc(tx)

	// 流程：manager → role(hr)
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "role", ApproverRoleID: hrRoleID, Name: "HR终审"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "sick", StartDate: start, EndDate: end, Reason: "角色池流程",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 经理通过 → 步骤2 role 池任务
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	if inst.CurrentStep != 2 {
		t.Fatalf("期望当前步骤 2，实际 %d", inst.CurrentStep)
	}
	task2 := stepTask(t, tx, inst, 2)
	if task2.ApproverType != "role" {
		t.Fatalf("期望步骤2类型 role，实际 %s", task2.ApproverType)
	}
	if task2.ApproverID != nil {
		t.Fatalf("role 池任务期望未指派审批人，实际 %v", task2.ApproverID)
	}
	if task2.ApproverRoleID != hrRoleID {
		t.Fatalf("期望 role 池角色 %d，实际 %d", hrRoleID, task2.ApproverRoleID)
	}

	// 可见性：hr 用户（hr 角色）看到 task2；申请人（employee 角色）看不到
	hrItems, err := ListMyTasks(tx, fx.HRUserID, "pending")
	if err != nil {
		t.Fatalf("查询 hr 待办失败: %v", err)
	}
	if findTaskByStep(hrItems, inst.ID, 2) == nil {
		t.Error("hr 用户应看到 role 池任务")
	}
	empItems, err := ListMyTasks(tx, fx.EmpUserID, "pending")
	if err != nil {
		t.Fatalf("查询申请人待办失败: %v", err)
	}
	if findTaskByStep(empItems, inst.ID, 2) != nil {
		t.Error("employee 用户不应看到 role 池任务")
	}

	// 角标
	cnt, err := CountMyPendingTasks(tx, fx.HRUserID)
	if err != nil {
		t.Fatalf("查询 hr 待办计数失败: %v", err)
	}
	if cnt != 1 {
		t.Errorf("hr 期望待办计数 1，实际 %d", cnt)
	}
	cnt, err = CountMyPendingTasks(tx, fx.EmpUserID)
	if err != nil {
		t.Fatalf("查询申请人待办计数失败: %v", err)
	}
	if cnt != 0 {
		t.Errorf("employee 期望待办计数 0，实际 %d", cnt)
	}

	// 越权：employee 审批 task2 → 拒绝
	if _, err := ApproveTask(tx, task2.ID, fx.EmpUserID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("employee 审批应返回 ErrApprovalNotAllowed，实际: %v", err)
	}

	// hr 审批通过 → 实例终审通过；role 池任务回填实际审批人
	inst = approveTask(t, tx, svc, task2.ID, fx.HRUserID)
	if inst.Status != "approved" {
		t.Fatalf("期望实例 approved，实际 %s", inst.Status)
	}
	task2Done := stepTask(t, tx, inst, 2)
	if task2Done.ApproverID == nil || *task2Done.ApproverID != fx.HRUserID {
		t.Fatalf("期望 role 池任务回填审批人 %d，实际 %v", fx.HRUserID, task2Done.ApproverID)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "approved" {
		t.Errorf("期望请假单 approved，实际 %s", lr.Status)
	}
}

// TestRolePoolNotify role 池任务创建时向该角色所有启用用户广播待审批通知；
// 步骤1（经理）创建时不应通知 hr，employee 用户不应收到 role 池通知
func TestRolePoolNotify(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	hrRoleID := findHrRoleID(t, tx)
	hr2, err := createTestUser(tx, uniqueStr("hr2"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建第二个 hr 账号失败: %v", err)
	}
	svc := newLeaveSvc(tx)

	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "role", ApproverRoleID: hrRoleID, Name: "HR终审"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "personal", StartDate: start, EndDate: end, Reason: "角色池通知",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1创建时：经理收到；hr（role 池用户）不应收到
	var mgrNotif, hrNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.ManagerUserID, NotifTypeLeaveApproval).Count(&mgrNotif).Error; err != nil {
		t.Fatalf("查询经理通知失败: %v", err)
	}
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.HRUserID, NotifTypeLeaveApproval).Count(&hrNotif).Error; err != nil {
		t.Fatalf("查询 hr 通知失败: %v", err)
	}
	if mgrNotif == 0 {
		t.Error("步骤1创建应通知经理")
	}
	if hrNotif != 0 {
		t.Error("步骤1创建不应通知 hr（role 池用户）")
	}

	// 经理通过 → 步骤2 role 池任务创建 → 广播所有 hr 角色用户，employee 不收
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	for _, uid := range []uint{fx.HRUserID, hr2.ID} {
		var n int64
		if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", uid, NotifTypeLeaveApproval).Count(&n).Error; err != nil {
			t.Fatalf("查询 hr 用户通知失败: %v", err)
		}
		if n == 0 {
			t.Errorf("hr 用户 %d 应收到 role 池待审批通知", uid)
		}
	}
	var empNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.EmpUserID, NotifTypeLeaveApproval).Count(&empNotif).Error; err != nil {
		t.Fatalf("查询申请人员工通知失败: %v", err)
	}
	if empNotif != 0 {
		t.Error("employee 用户不应收到 role 池通知")
	}
}

// TestRoleStepValidation 角色池步骤配置校验：合法 role 通过；
// role 无/无效角色 ID、role 带权限点、manager 带角色 ID、role 带岗位引用均拒绝
func TestRoleStepValidation(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	hrRoleID := findHrRoleID(t, tx)

	// 合法：role(hr)
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "role", ApproverRoleID: hrRoleID, Name: "HR终审"},
		},
	}); err != nil {
		t.Fatalf("合法 role 步骤更新失败: %v", err)
	}

	// role 无角色 ID
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "role"},
	}}); !errors.Is(err, ErrApprovalFlowRoleBad) {
		t.Fatalf("role 无角色 ID 应返回 ErrApprovalFlowRoleBad，实际: %v", err)
	}

	// role 角色不存在
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "role", ApproverRoleID: 999999},
	}}); !errors.Is(err, ErrApprovalFlowRoleBad) {
		t.Fatalf("role 角色不存在应返回 ErrApprovalFlowRoleBad，实际: %v", err)
	}

	// role 带权限点
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "role", ApproverRoleID: hrRoleID, ApproverPermission: "leave:approve"},
	}}); !errors.Is(err, ErrApprovalFlowPermBad) {
		t.Fatalf("role 带权限点应返回 ErrApprovalFlowPermBad，实际: %v", err)
	}

	// role 带岗位引用
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "role", ApproverRoleID: hrRoleID, ApproverRef: 1},
	}}); !errors.Is(err, ErrApprovalFlowRefConflict) {
		t.Fatalf("role 带岗位引用应返回 ErrApprovalFlowRefConflict，实际: %v", err)
	}

	// manager 带角色 ID
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager", ApproverRoleID: hrRoleID},
	}}); !errors.Is(err, ErrApprovalFlowRoleBad) {
		t.Fatalf("manager 带角色 ID 应返回 ErrApprovalFlowRoleBad，实际: %v", err)
	}

	// permission 带角色 ID（permission 兼容旧配置但不得混用角色）
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "permission", ApproverPermission: "leave:approve", ApproverRoleID: hrRoleID},
	}}); !errors.Is(err, ErrApprovalFlowRoleBad) {
		t.Fatalf("permission 带角色 ID 应返回 ErrApprovalFlowRoleBad，实际: %v", err)
	}
}

// TestApprovalFlowListRoleName 流程详情步骤带出审批角色名称（approver_role_name）
func TestApprovalFlowListRoleName(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	hrRoleID := findHrRoleID(t, tx)

	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "role", ApproverRoleID: hrRoleID, Name: "HR终审"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	details, err := ListApprovalFlows(tx)
	if err != nil {
		t.Fatalf("查询审批流程列表失败: %v", err)
	}
	found := false
	for _, d := range details {
		if d.Code != "leave" {
			continue
		}
		found = true
		if len(d.Steps) != 2 {
			t.Fatalf("期望 leave 流程 2 步，实际 %d", len(d.Steps))
		}
		s2 := d.Steps[1]
		if s2.ApproverType != "role" || s2.ApproverRoleID != hrRoleID {
			t.Fatalf("期望步骤2 role(%d)，实际 type=%s role_id=%d", hrRoleID, s2.ApproverType, s2.ApproverRoleID)
		}
		if s2.ApproverRoleName != "人事" {
			t.Errorf("期望审批角色名称 人事，实际 %q", s2.ApproverRoleName)
		}
	}
	if !found {
		t.Fatal("未找到 leave 流程")
	}
}

// TestNotificationSendSkipsDisabledUser 禁用账号不推送站内通知
func TestNotificationSendSkipsDisabledUser(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	disabledUser, err := createTestUser(tx, uniqueStr("dis"), "pass123456", "employee", nil, 0)
	if err != nil {
		t.Fatalf("创建禁用账号失败: %v", err)
	}
	notif := NewNotificationService(repository.NewNotificationRepo(tx), repository.NewEmployeeRepo(tx))
	if err := notif.Send(disabledUser.ID, NotifTypeSystem, "标题", "内容", "/"); err != nil {
		t.Fatalf("发送通知失败: %v", err)
	}
	var n int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ?", disabledUser.ID).Count(&n).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if n != 0 {
		t.Error("禁用账号不应收到站内通知")
	}
}
