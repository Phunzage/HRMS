package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// permissionPoolFixture 权限池审批测试环境：部门（含负责人）+ 经理(角色有 leave:approve)/申请人/普通员工(无权限)/HR(有 leave:approve)
type permissionPoolFixture struct {
	Dept          *models.Department
	ManagerUserID uint // manager 角色，拥有 leave:approve
	EmpUserID     uint // 申请人（employee 角色，无 leave:approve）
	OtherUserID   uint // 普通员工（employee 角色，无 leave:approve）
	HRUserID      uint // hr 角色，拥有 leave:approve
}

// grantRolePermission 在 tx 内为角色授予权限点（测试库不跑 seed，权限点/关联需自建），事务回滚即撤销
func grantRolePermission(tx *gorm.DB, roleCode, permCode string) error {
	var perm models.Permission
	err := tx.Where("code = ?", permCode).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		perm = models.Permission{Name: permCode, Code: permCode, Module: "approval"}
		if err := tx.Create(&perm).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var role models.Role
	if err := tx.Where("code = ?", roleCode).First(&role).Error; err != nil {
		return err
	}
	var count int64
	if err := tx.Model(&models.RolePermission{}).
		Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return tx.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error
	}
	return nil
}

// setupPermissionPoolFixture 构造权限池测试环境并把 leave 流程改为两级：manager → permission(leave:approve)，
// 所有数据落在传入事务内由调用方回滚
func setupPermissionPoolFixture(t *testing.T, tx *gorm.DB) *permissionPoolFixture {
	t.Helper()
	dept, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	managerEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("经理"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建经理员工失败: %v", err)
	}
	if err := tx.Model(&models.Department{}).Where("id = ?", dept.ID).Update("manager_id", managerEmp.ID).Error; err != nil {
		t.Fatalf("设置部门负责人失败: %v", err)
	}
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("申请人"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建申请人员工失败: %v", err)
	}
	otherEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("普通员工"), dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建普通员工失败: %v", err)
	}

	managerUser, err := createTestUser(tx, uniqueStr("mgr"), "pass123456", "manager", &managerEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建经理账号失败: %v", err)
	}
	empUser, err := createTestUser(tx, uniqueStr("emp"), "pass123456", "employee", &emp.ID, 1)
	if err != nil {
		t.Fatalf("创建申请人账号失败: %v", err)
	}
	otherUser, err := createTestUser(tx, uniqueStr("other"), "pass123456", "employee", &otherEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建普通员工账号失败: %v", err)
	}
	hrUser, err := createTestUser(tx, uniqueStr("hr"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建 HR 账号失败: %v", err)
	}
	// 回写员工→账号关联（审批引擎按 employee.user_id 解析审批人账号）
	for _, e := range []struct {
		emp *models.Employee
		uid uint
	}{
		{managerEmp, managerUser.ID},
		{emp, empUser.ID},
		{otherEmp, otherUser.ID},
	} {
		if err := tx.Model(&models.Employee{}).Where("id = ?", e.emp.ID).Update("user_id", e.uid).Error; err != nil {
			t.Fatalf("回写员工账号关联失败: %v", err)
		}
	}

	// 授予 manager / hr 角色 leave:approve（权限池匹配与通知广播用）
	if err := grantRolePermission(tx, "manager", "leave:approve"); err != nil {
		t.Fatalf("授予 manager leave:approve 失败: %v", err)
	}
	if err := grantRolePermission(tx, "hr", "leave:approve"); err != nil {
		t.Fatalf("授予 hr leave:approve 失败: %v", err)
	}

	// 把 leave 流程改为两级：manager → permission(leave:approve)
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "permission", ApproverRef: 0, ApproverPermission: "leave:approve", Name: "审批权终审"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	return &permissionPoolFixture{
		Dept:          dept,
		ManagerUserID: managerUser.ID,
		EmpUserID:     empUser.ID,
		OtherUserID:   otherUser.ID,
		HRUserID:      hrUser.ID,
	}
}

// TestPermissionPoolFullFlow 权限池端到端：manager → permission(leave:approve)，
// 覆盖权限池任务创建（ApproverPermission 写入、未指派）、可见性、角标、越权拒绝、回填审批人、终审通过
func TestPermissionPoolFullFlow(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupPermissionPoolFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "权限池两级流程",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1：经理通过 → 步骤2 权限池任务产生
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	if inst.CurrentStep != 2 {
		t.Fatalf("期望当前步骤 2，实际 %d", inst.CurrentStep)
	}
	task2 := stepTask(t, tx, inst, 2)
	if task2.ApproverType != "permission" {
		t.Fatalf("期望步骤2类型 permission，实际 %s", task2.ApproverType)
	}
	if task2.ApproverID != nil {
		t.Fatalf("权限池任务期望未指派审批人，实际 %v", task2.ApproverID)
	}
	if task2.ApproverPermission != "leave:approve" {
		t.Fatalf("期望权限点 leave:approve，实际 %s", task2.ApproverPermission)
	}

	// 可见性：拥有 leave:approve 的用户（经理、HR）看到 task2；无权限员工看不到
	for _, uid := range []uint{fx.ManagerUserID, fx.HRUserID} {
		items, err := ListMyTasks(tx, uid, "pending")
		if err != nil {
			t.Fatalf("查询有权限用户待办失败: %v", err)
		}
		if findTaskByStep(items, inst.ID, 2) == nil {
			t.Errorf("拥有 leave:approve 的用户 %d 应看到 task2", uid)
		}
	}
	otherItems, err := ListMyTasks(tx, fx.OtherUserID, "pending")
	if err != nil {
		t.Fatalf("查询无权限用户待办失败: %v", err)
	}
	if findTaskByStep(otherItems, inst.ID, 2) != nil {
		t.Error("无 leave:approve 的员工不应看到 task2")
	}

	// 角标：经理/HR pending 含 task2；无权限员工计数不含
	for _, uid := range []uint{fx.ManagerUserID, fx.HRUserID} {
		cnt, err := CountMyPendingTasks(tx, uid)
		if err != nil {
			t.Fatalf("查询有权限用户待办计数失败: %v", err)
		}
		if cnt != 1 {
			t.Errorf("有权限用户 %d 期望待办计数 1，实际 %d", uid, cnt)
		}
	}
	cnt, err := CountMyPendingTasks(tx, fx.OtherUserID)
	if err != nil {
		t.Fatalf("查询无权限用户待办计数失败: %v", err)
	}
	if cnt != 0 {
		t.Errorf("无权限员工期望待办计数 0，实际 %d", cnt)
	}

	// 越权：无权限员工审批 task2 → 拒绝
	if _, err := ApproveTask(tx, task2.ID, fx.OtherUserID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("无权限员工审批应返回 ErrApprovalNotAllowed，实际: %v", err)
	}
	if err := RejectTask(tx, task2.ID, fx.OtherUserID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("无权限员工驳回应返回 ErrApprovalNotAllowed，实际: %v", err)
	}

	// 经理（有权限）审批 task2 → 终审通过；权限池任务回填实际审批人
	inst = approveTask(t, tx, svc, task2.ID, fx.ManagerUserID)
	if inst.Status != "approved" {
		t.Fatalf("期望实例 approved，实际 %s", inst.Status)
	}
	task2Done := stepTask(t, tx, inst, 2)
	if task2Done.ApproverID == nil || *task2Done.ApproverID != fx.ManagerUserID {
		t.Fatalf("期望权限池任务回填审批人 %d，实际 %v", fx.ManagerUserID, task2Done.ApproverID)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "approved" {
		t.Errorf("期望请假单 approved，实际 %s", lr.Status)
	}
}

// TestPermissionPoolNotify 权限池任务创建时向拥有该权限点的所有启用用户广播待审批通知；
// 步骤1（经理）创建时不应通知权限池用户，无权限员工不应收到权限池通知
func TestPermissionPoolNotify(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupPermissionPoolFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "personal", StartDate: start, EndDate: end, Reason: "权限池通知",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1创建时：经理收到，HR（权限池用户）不应收到
	var mgrNotif, hrNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.ManagerUserID, NotifTypeLeaveApproval).Count(&mgrNotif).Error; err != nil {
		t.Fatalf("查询经理通知失败: %v", err)
	}
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.HRUserID, NotifTypeLeaveApproval).Count(&hrNotif).Error; err != nil {
		t.Fatalf("查询 HR 通知失败: %v", err)
	}
	if mgrNotif == 0 {
		t.Error("步骤1创建应通知经理")
	}
	if hrNotif != 0 {
		t.Error("步骤1创建不应通知 HR（权限池用户）")
	}

	// 经理通过 → 步骤2 权限池任务创建 → 广播所有拥有 leave:approve 的用户（经理+HR），无权限员工不收
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	for _, uid := range []uint{fx.ManagerUserID, fx.HRUserID} {
		var n int64
		if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", uid, NotifTypeLeaveApproval).Count(&n).Error; err != nil {
			t.Fatalf("查询有权限用户通知失败: %v", err)
		}
		if n == 0 {
			t.Errorf("拥有 leave:approve 的用户 %d 应收到待审批通知", uid)
		}
	}
	var otherNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.OtherUserID, NotifTypeLeaveApproval).Count(&otherNotif).Error; err != nil {
		t.Fatalf("查询无权限员工通知失败: %v", err)
	}
	if otherNotif != 0 {
		t.Error("无权限员工不应收到权限池通知")
	}
}

// TestPermissionStepValidation 权限池步骤配置校验：合法 permission 通过；非法权限点 / manager 带权限点 / permission 带 ref 均拒绝
func TestPermissionStepValidation(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	// 合法：permission(leave:approve)
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "permission", ApproverRef: 0, ApproverPermission: "leave:approve", Name: "审批权终审"},
		},
	}); err != nil {
		t.Fatalf("合法 permission 步骤更新失败: %v", err)
	}

	// permission 非法权限点
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "permission", ApproverPermission: "not_exist:approve"},
	}}); !errors.Is(err, ErrApprovalFlowPermBad) {
		t.Fatalf("非法权限点应返回 ErrApprovalFlowPermBad，实际: %v", err)
	}

	// manager 带权限点
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager", ApproverPermission: "leave:approve"},
	}}); !errors.Is(err, ErrApprovalFlowPermBad) {
		t.Fatalf("manager 带权限点应返回 ErrApprovalFlowPermBad，实际: %v", err)
	}

	// hr 带权限点
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "hr", ApproverPermission: "leave:approve"},
	}}); !errors.Is(err, ErrApprovalFlowPermBad) {
		t.Fatalf("hr 带权限点应返回 ErrApprovalFlowPermBad，实际: %v", err)
	}

	// permission 带岗位引用
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "permission", ApproverRef: 1, ApproverPermission: "leave:approve"},
	}}); !errors.Is(err, ErrApprovalFlowRefConflict) {
		t.Fatalf("permission 带 ref 应返回 ErrApprovalFlowRefConflict，实际: %v", err)
	}
}

// TestListApprovalPermissions 审批权限点角色列表：授予 manager/hr leave:approve 后返回对应角色
func TestListApprovalPermissions(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	if err := grantRolePermission(tx, "manager", "leave:approve"); err != nil {
		t.Fatalf("授予 manager leave:approve 失败: %v", err)
	}
	if err := grantRolePermission(tx, "hr", "leave:approve"); err != nil {
		t.Fatalf("授予 hr leave:approve 失败: %v", err)
	}
	perms, err := ListApprovalPermissions(tx)
	if err != nil {
		t.Fatalf("查询审批权限点失败: %v", err)
	}
	if len(perms) != 2 {
		t.Fatalf("期望 2 个审批权限点，实际 %d", len(perms))
	}
	found := false
	for _, p := range perms {
		if p.Code == "leave:approve" {
			found = true
			if len(p.Roles) != 2 {
				t.Errorf("期望 leave:approve 有 2 个角色，实际 %d", len(p.Roles))
			}
			names := map[string]bool{}
			for _, r := range p.Roles {
				names[r.Name] = true
			}
			if !names["经理"] || !names["人事"] {
				t.Errorf("期望角色含 经理/人事，实际 %v", names)
			}
		}
	}
	if !found {
		t.Error("未找到 leave:approve 权限点")
	}
}
