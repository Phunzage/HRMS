package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// positionPoolFixture 岗位池审批测试环境：部门（含负责人）+ 经理/申请人/两个同岗位员工/其它岗位员工/HR
type positionPoolFixture struct {
	Dept          *models.Department
	PosP          *models.Position // 岗位池指定岗位
	PosQ          *models.Position // 其它岗位（越权样本）
	ManagerUserID uint
	EmpUserID     uint
	HRUserID      uint
	Wang1UserID   uint // 岗位 P 在职员工甲
	Wang2UserID   uint // 岗位 P 在职员工乙
	OtherUserID   uint // 岗位 Q 在职员工
}

// setupPositionPoolFixture 构造岗位池测试环境并把 leave 流程改为三级（manager → position(P) → hr），
// 所有数据落在传入事务内由调用方回滚
func setupPositionPoolFixture(t *testing.T, tx *gorm.DB) *positionPoolFixture {
	t.Helper()
	dept, posP, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	posQ := &models.Position{Name: uniqueStr("其它岗位"), Code: uniqueStr("posq_"), DepartmentID: dept.ID}
	if err := tx.Create(posQ).Error; err != nil {
		t.Fatalf("创建其它岗位失败: %v", err)
	}
	// 经理（同部门负责人）
	managerEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("经理"), dept.ID, posP.ID)
	if err != nil {
		t.Fatalf("创建经理员工失败: %v", err)
	}
	if err := tx.Model(&models.Department{}).Where("id = ?", dept.ID).Update("manager_id", managerEmp.ID).Error; err != nil {
		t.Fatalf("设置部门负责人失败: %v", err)
	}
	// 申请人
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("员工"), dept.ID, posP.ID)
	if err != nil {
		t.Fatalf("创建申请人员工失败: %v", err)
	}
	// 岗位 P 员工甲/乙、岗位 Q 员工
	w1Emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("王五甲"), dept.ID, posP.ID)
	if err != nil {
		t.Fatalf("创建岗位P员工甲失败: %v", err)
	}
	w2Emp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("王五乙"), dept.ID, posP.ID)
	if err != nil {
		t.Fatalf("创建岗位P员工乙失败: %v", err)
	}
	oEmp, err := createTestEmployee(tx, uniqueStr("EMP"), uniqueStr("其它员工"), dept.ID, posQ.ID)
	if err != nil {
		t.Fatalf("创建岗位Q员工失败: %v", err)
	}

	managerUser, err := createTestUser(tx, uniqueStr("mgr"), "pass123456", "manager", &managerEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建经理账号失败: %v", err)
	}
	empUser, err := createTestUser(tx, uniqueStr("emp"), "pass123456", "employee", &emp.ID, 1)
	if err != nil {
		t.Fatalf("创建申请人账号失败: %v", err)
	}
	w1User, err := createTestUser(tx, uniqueStr("wang1"), "pass123456", "employee", &w1Emp.ID, 1)
	if err != nil {
		t.Fatalf("创建岗位P员工甲账号失败: %v", err)
	}
	w2User, err := createTestUser(tx, uniqueStr("wang2"), "pass123456", "employee", &w2Emp.ID, 1)
	if err != nil {
		t.Fatalf("创建岗位P员工乙账号失败: %v", err)
	}
	oUser, err := createTestUser(tx, uniqueStr("other"), "pass123456", "employee", &oEmp.ID, 1)
	if err != nil {
		t.Fatalf("创建岗位Q员工账号失败: %v", err)
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
		{w1Emp, w1User.ID},
		{w2Emp, w2User.ID},
		{oEmp, oUser.ID},
	} {
		if err := tx.Model(&models.Employee{}).Where("id = ?", e.emp.ID).Update("user_id", e.uid).Error; err != nil {
			t.Fatalf("回写员工账号关联失败: %v", err)
		}
	}

	// 把 leave 流程改为三级：manager → position(P) → hr
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "position", ApproverRef: posP.ID, Name: "人事专员复核"},
			{StepOrder: 3, ApproverType: "hr", ApproverRef: 0, Name: "HR终审"},
		},
	}); err != nil {
		t.Fatalf("更新请假流程失败: %v", err)
	}

	return &positionPoolFixture{
		Dept:          dept,
		PosP:          posP,
		PosQ:          posQ,
		ManagerUserID: managerUser.ID,
		EmpUserID:     empUser.ID,
		HRUserID:      hrUser.ID,
		Wang1UserID:   w1User.ID,
		Wang2UserID:   w2User.ID,
		OtherUserID:   oUser.ID,
	}
}

// findTaskByStep 在任务列表中查找指定实例+步骤的待办项（nil 表示不存在）
func findTaskByStep(items []TaskItem, instID uint, step int) *TaskItem {
	for i := range items {
		if items[i].InstanceID == instID && items[i].StepOrder == step {
			return &items[i]
		}
	}
	return nil
}

// TestUpdateApprovalFlowValidation 审批流程配置校验：合法两级/三级更新成功，各类非法输入返回对应错误
func TestUpdateApprovalFlowValidation(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	_, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}

	// 合法：两级（manager→hr）
	detail, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "hr", ApproverRef: 0, Name: "HR终审"},
		},
	})
	if err != nil {
		t.Fatalf("合法两级更新失败: %v", err)
	}
	if len(detail.Steps) != 2 || detail.Steps[0].ApproverType != "manager" || detail.Steps[1].ApproverType != "hr" {
		t.Fatalf("更新后的流程步骤错误: %+v", detail.Steps)
	}
	if detail.Steps[0].ApproverRef != 0 || detail.Steps[1].ApproverRef != 0 {
		t.Fatalf("manager/hr 步骤 ApproverRef 应为 0: %+v", detail.Steps)
	}

	// 合法：三级（manager→position→hr），position 引用真实岗位
	detail, err = UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Name: "请假审批",
		Steps: []ApprovalFlowStepInput{
			{StepOrder: 1, ApproverType: "manager", ApproverRef: 0, Name: "部门经理审批"},
			{StepOrder: 2, ApproverType: "position", ApproverRef: pos.ID, Name: "人事专员复核"},
			{StepOrder: 3, ApproverType: "hr", ApproverRef: 0, Name: "HR终审"},
		},
	})
	if err != nil {
		t.Fatalf("合法三级更新失败: %v", err)
	}
	if len(detail.Steps) != 3 || detail.Steps[1].ApproverType != "position" || detail.Steps[1].ApproverRef != pos.ID {
		t.Fatalf("三级流程步骤错误: %+v", detail.Steps)
	}

	// 非法编码
	if _, err := UpdateApprovalFlow(tx, "not_exist", UpdateApprovalFlowRequest{
		Steps: []ApprovalFlowStepInput{{StepOrder: 1, ApproverType: "hr"}},
	}); !errors.Is(err, ErrApprovalFlowCodeInvalid) {
		t.Fatalf("非法编码应返回 ErrApprovalFlowCodeInvalid，实际: %v", err)
	}

	// 空步骤 / 超 5 级
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{}); !errors.Is(err, ErrApprovalFlowStepCount) {
		t.Fatalf("空步骤应返回 ErrApprovalFlowStepCount，实际: %v", err)
	}
	sixSteps := make([]ApprovalFlowStepInput, 6)
	for i := range sixSteps {
		sixSteps[i] = ApprovalFlowStepInput{StepOrder: i + 1, ApproverType: "hr"}
	}
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: sixSteps}); !errors.Is(err, ErrApprovalFlowStepCount) {
		t.Fatalf("6 级步骤应返回 ErrApprovalFlowStepCount，实际: %v", err)
	}

	// step_order 不连续
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager"},
		{StepOrder: 3, ApproverType: "hr"},
	}}); !errors.Is(err, ErrApprovalFlowStepOrder) {
		t.Fatalf("不连续步骤应返回 ErrApprovalFlowStepOrder，实际: %v", err)
	}

	// 非法审批人类型
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "boss"},
	}}); !errors.Is(err, ErrApprovalFlowTypeInvalid) {
		t.Fatalf("非法类型应返回 ErrApprovalFlowTypeInvalid，实际: %v", err)
	}

	// position 无 ref
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager"},
		{StepOrder: 2, ApproverType: "position"},
	}}); !errors.Is(err, ErrApprovalFlowPosRefBad) {
		t.Fatalf("position 无 ref 应返回 ErrApprovalFlowPosRefBad，实际: %v", err)
	}

	// position 岗位不存在
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager"},
		{StepOrder: 2, ApproverType: "position", ApproverRef: 999999},
	}}); !errors.Is(err, ErrApprovalFlowPosRefBad) {
		t.Fatalf("position 岗位不存在应返回 ErrApprovalFlowPosRefBad，实际: %v", err)
	}

	// manager/hr 带 ref
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{Steps: []ApprovalFlowStepInput{
		{StepOrder: 1, ApproverType: "manager", ApproverRef: pos.ID},
	}}); !errors.Is(err, ErrApprovalFlowRefConflict) {
		t.Fatalf("manager 带 ref 应返回 ErrApprovalFlowRefConflict，实际: %v", err)
	}

	// 流程记录不存在
	var lf models.ApprovalFlow
	if err := tx.Where("code = ?", "leave").First(&lf).Error; err != nil {
		t.Fatalf("查询 leave 流程失败: %v", err)
	}
	if err := tx.Delete(&models.ApprovalFlow{}, lf.ID).Error; err != nil {
		t.Fatalf("删除 leave 流程失败: %v", err)
	}
	if _, err := UpdateApprovalFlow(tx, "leave", UpdateApprovalFlowRequest{
		Steps: []ApprovalFlowStepInput{{StepOrder: 1, ApproverType: "hr"}},
	}); !errors.Is(err, ErrApprovalFlowNotFound) {
		t.Fatalf("流程不存在应返回 ErrApprovalFlowNotFound，实际: %v", err)
	}
}

// TestPositionPoolFullFlow 三级流程端到端：manager → position(岗位池) → hr，
// 覆盖岗位池任务创建（ApproverRef 写入）、可见性、角标、越权拒绝、回填审批人、终审通过
func TestPositionPoolFullFlow(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupPositionPoolFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(2)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "annual", StartDate: start, EndDate: end, Reason: "岗位池三级流程",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1：经理通过 → 步骤2 岗位池任务产生
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	if inst.CurrentStep != 2 {
		t.Fatalf("期望当前步骤 2，实际 %d", inst.CurrentStep)
	}
	task2 := stepTask(t, tx, inst, 2)
	if task2.ApproverType != "position" {
		t.Fatalf("期望步骤2类型 position，实际 %s", task2.ApproverType)
	}
	if task2.ApproverID != nil {
		t.Fatalf("岗位池任务期望未指派审批人，实际 %v", task2.ApproverID)
	}
	if task2.ApproverRef != fx.PosP.ID {
		t.Fatalf("期望岗位池引用 %d，实际 %d", fx.PosP.ID, task2.ApproverRef)
	}

	// 可见性：岗位P两个在职员工看到 task2；岗位Q员工看不到
	for _, uid := range []uint{fx.Wang1UserID, fx.Wang2UserID} {
		items, err := ListMyTasks(tx, uid, "pending")
		if err != nil {
			t.Fatalf("查询岗位P用户待办失败: %v", err)
		}
		if findTaskByStep(items, inst.ID, 2) == nil {
			t.Errorf("岗位P用户 %d 应看到 task2", uid)
		}
	}
	otherItems, err := ListMyTasks(tx, fx.OtherUserID, "pending")
	if err != nil {
		t.Fatalf("查询岗位Q用户待办失败: %v", err)
	}
	if findTaskByStep(otherItems, inst.ID, 2) != nil {
		t.Error("岗位Q用户不应看到 task2")
	}

	// 角标：岗位P用户 pending=1；岗位Q用户=0
	for _, uid := range []uint{fx.Wang1UserID, fx.Wang2UserID} {
		cnt, err := CountMyPendingTasks(tx, uid)
		if err != nil {
			t.Fatalf("查询岗位P用户待办计数失败: %v", err)
		}
		if cnt != 1 {
			t.Errorf("岗位P用户 %d 期望待办计数 1，实际 %d", uid, cnt)
		}
	}
	cnt, err := CountMyPendingTasks(tx, fx.OtherUserID)
	if err != nil {
		t.Fatalf("查询岗位Q用户待办计数失败: %v", err)
	}
	if cnt != 0 {
		t.Errorf("岗位Q用户期望待办计数 0，实际 %d", cnt)
	}

	// 越权：岗位Q用户审批 task2 → 拒绝
	if _, err := ApproveTask(tx, task2.ID, fx.OtherUserID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("岗位Q用户审批应返回 ErrApprovalNotAllowed，实际: %v", err)
	}
	if err := RejectTask(tx, task2.ID, fx.OtherUserID, "越权"); !errors.Is(err, ErrApprovalNotAllowed) {
		t.Fatalf("岗位Q用户驳回应返回 ErrApprovalNotAllowed，实际: %v", err)
	}

	// 岗位P员工甲通过 → 步骤3 HR 池任务；岗位池任务回填实际审批人
	inst = approveTask(t, tx, svc, task2.ID, fx.Wang1UserID)
	task2Done := stepTask(t, tx, inst, 2)
	if task2Done.ApproverID == nil || *task2Done.ApproverID != fx.Wang1UserID {
		t.Fatalf("期望岗位池任务回填审批人 %d，实际 %v", fx.Wang1UserID, task2Done.ApproverID)
	}
	if inst.CurrentStep != 3 {
		t.Fatalf("期望当前步骤 3，实际 %d", inst.CurrentStep)
	}
	task3 := stepTask(t, tx, inst, 3)
	if task3.ApproverType != "hr" || task3.ApproverID != nil {
		t.Fatalf("期望步骤3为 HR 池任务，实际 type=%s approver=%v", task3.ApproverType, task3.ApproverID)
	}

	// HR 终审通过 → 实例 approved + 业务回调生效
	inst = approveTask(t, tx, svc, task3.ID, fx.HRUserID)
	if inst.Status != "approved" {
		t.Fatalf("期望实例 approved，实际 %s", inst.Status)
	}
	lr, err := repository.NewLeaveRepo(tx).FindByID(leave.ID)
	if err != nil {
		t.Fatalf("查询请假失败: %v", err)
	}
	if lr.Status != "approved" {
		t.Errorf("期望请假单 approved，实际 %s", lr.Status)
	}
}

// TestPositionPoolNotify 岗位池任务创建时向该岗位所有在职员工广播待审批通知；
// 步骤1（经理）创建时不应通知岗位池用户，岗位Q用户不应收到岗位池通知
func TestPositionPoolNotify(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupPositionPoolFixture(t, tx)
	svc := newLeaveSvc(tx)

	start, end := dateRange(1)
	leave, err := svc.Create(fx.EmpUserID, SubmitLeaveRequest{
		LeaveType: "personal", StartDate: start, EndDate: end, Reason: "岗位池通知",
	})
	if err != nil {
		t.Fatalf("提交请假失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "leave", leave.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}

	// 步骤1创建时：经理收到，岗位P用户不应收到
	var mgrNotif, wangNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.ManagerUserID, NotifTypeLeaveApproval).Count(&mgrNotif).Error; err != nil {
		t.Fatalf("查询经理通知失败: %v", err)
	}
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.Wang1UserID, NotifTypeLeaveApproval).Count(&wangNotif).Error; err != nil {
		t.Fatalf("查询岗位P用户通知失败: %v", err)
	}
	if mgrNotif == 0 {
		t.Error("步骤1创建应通知经理")
	}
	if wangNotif != 0 {
		t.Error("步骤1创建不应通知岗位P用户")
	}

	// 经理通过 → 步骤2 岗位池任务创建 → 广播岗位P在职员工，岗位Q员工不收
	inst = approveTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID)
	for _, uid := range []uint{fx.Wang1UserID, fx.Wang2UserID} {
		var n int64
		if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", uid, NotifTypeLeaveApproval).Count(&n).Error; err != nil {
			t.Fatalf("查询岗位P用户通知失败: %v", err)
		}
		if n == 0 {
			t.Errorf("岗位P用户 %d 应收到待审批通知", uid)
		}
	}
	var otherNotif int64
	if err := tx.Model(&models.Notification{}).Where("user_id = ? AND type = ?", fx.OtherUserID, NotifTypeLeaveApproval).Count(&otherNotif).Error; err != nil {
		t.Fatalf("查询岗位Q用户通知失败: %v", err)
	}
	if otherNotif != 0 {
		t.Error("岗位Q用户不应收到岗位池通知")
	}
}

// TestApprovalFlowList 查询所有流程及步骤（流程配置管理 GET 用）
func TestApprovalFlowList(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	details, err := ListApprovalFlows(tx)
	if err != nil {
		t.Fatalf("查询审批流程列表失败: %v", err)
	}
	if len(details) == 0 {
		t.Fatal("期望至少存在审批流程定义")
	}
	found := false
	for _, d := range details {
		if d.Code == "leave" && len(d.Steps) == 2 {
			found = true
			if d.Steps[0].ApproverType != "manager" || d.Steps[1].ApproverType != "hr" {
				t.Errorf("默认 leave 流程步骤应为 manager→hr: %+v", d.Steps)
			}
		}
	}
	if !found {
		t.Error("未找到默认两级 leave 流程")
	}
}
