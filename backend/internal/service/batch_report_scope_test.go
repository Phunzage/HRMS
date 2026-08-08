package service

import (
	"encoding/json"
	"testing"
	"time"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// TestResolveScopeNonHR 非 hr 角色一律不过滤（返回 nil）
func TestResolveScopeNonHR(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	for _, role := range []string{"admin", "manager", "employee"} {
		ids, err := ResolveScope(tx, 0, role)
		if err != nil {
			t.Fatalf("角色 %s 解析失败: %v", role, err)
		}
		if ids != nil {
			t.Errorf("角色 %s 应返回 nil 不过滤，实际 %v", role, ids)
		}
	}
}

// TestResolveScopeHR  hr 未绑定负责部门 → nil；绑定部门 → 返回该部门及其子孙部门
func TestResolveScopeHR(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	// 未绑定：hr 用户 DeptScopeID 为空 → nil
	hrNoScope, err := createTestUser(tx, uniqueStr("hr"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建 hr 账号失败: %v", err)
	}
	ids, err := ResolveScope(tx, hrNoScope.ID, "hr")
	if err != nil {
		t.Fatalf("hr 未绑定解析失败: %v", err)
	}
	if ids != nil {
		t.Errorf("hr 未绑定应返回 nil，实际 %v", ids)
	}

	// 绑定技术部：部门树 技术部 → 前端组/后端组
	root := &models.Department{Name: uniqueStr("技术部"), SortOrder: 1}
	if err := tx.Create(root).Error; err != nil {
		t.Fatalf("创建根部门失败: %v", err)
	}
	child1 := &models.Department{Name: uniqueStr("前端组"), ParentID: root.ID, SortOrder: 1}
	child2 := &models.Department{Name: uniqueStr("后端组"), ParentID: root.ID, SortOrder: 2}
	if err := tx.Create(child1).Error; err != nil {
		t.Fatalf("创建子部门失败: %v", err)
	}
	if err := tx.Create(child2).Error; err != nil {
		t.Fatalf("创建子部门失败: %v", err)
	}
	other := &models.Department{Name: uniqueStr("人事部"), SortOrder: 2}
	if err := tx.Create(other).Error; err != nil {
		t.Fatalf("创建无关部门失败: %v", err)
	}
	if err := tx.Model(&models.User{}).Where("id = ?", hrNoScope.ID).Update("dept_scope_id", root.ID).Error; err != nil {
		t.Fatalf("绑定负责部门失败: %v", err)
	}

	ids, err = ResolveScope(tx, hrNoScope.ID, "hr")
	if err != nil {
		t.Fatalf("hr 绑定解析失败: %v", err)
	}
	want := map[uint]bool{root.ID: true, child1.ID: true, child2.ID: true}
	if len(ids) != len(want) {
		t.Fatalf("期望 %d 个部门，实际 %v", len(want), ids)
	}
	for _, id := range ids {
		if !want[id] {
			t.Errorf("范围不应包含部门 %d", id)
		}
	}
	if !want[other.ID] {
		t.Log("无关部门不应在范围内（校验通过）")
	}
}

// TestEmployeeListScopeHR 员工列表按 hr 数据范围过滤：绑定技术部 → 只见技术部及子部门员工
func TestEmployeeListScopeHR(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	root := &models.Department{Name: uniqueStr("技术部"), SortOrder: 1}
	if err := tx.Create(root).Error; err != nil {
		t.Fatalf("创建根部门失败: %v", err)
	}
	child := &models.Department{Name: uniqueStr("前端组"), ParentID: root.ID, SortOrder: 1}
	if err := tx.Create(child).Error; err != nil {
		t.Fatalf("创建子部门失败: %v", err)
	}
	other := &models.Department{Name: uniqueStr("人事部"), SortOrder: 2}
	if err := tx.Create(other).Error; err != nil {
		t.Fatalf("创建无关部门失败: %v", err)
	}
	pos := &models.Position{Name: uniqueStr("测试岗"), Code: uniqueStr("pos_"), DepartmentID: root.ID}
	if err := tx.Create(pos).Error; err != nil {
		t.Fatalf("创建岗位失败: %v", err)
	}

	empRoot, err := createTestEmployee(tx, uniqueStr("EMP"), "根部门员工", root.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建根部门员工失败: %v", err)
	}
	empChild, err := createTestEmployee(tx, uniqueStr("EMP"), "子部门员工", child.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建子部门员工失败: %v", err)
	}
	empOther, err := createTestEmployee(tx, uniqueStr("EMP"), "其他部门员工", other.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建无关部门员工失败: %v", err)
	}

	hrUser, err := createTestUser(tx, uniqueStr("hr"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建 hr 账号失败: %v", err)
	}
	if err := tx.Model(&models.User{}).Where("id = ?", hrUser.ID).Update("dept_scope_id", root.ID).Error; err != nil {
		t.Fatalf("绑定负责部门失败: %v", err)
	}

	svc := NewEmployeeService(tx, repository.NewEmployeeRepo(tx), repository.NewDepartmentRepo(tx), repository.NewPositionRepo(tx))
	items, total, err := svc.List("hr", hrUser.ID, "", 0, 0, 0, 1, 50)
	if err != nil {
		t.Fatalf("hr 员工列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("hr 范围期望 2 名员工，实际 %d", total)
	}
	ids := map[uint]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}
	if !ids[empRoot.ID] || !ids[empChild.ID] {
		t.Errorf("应包含技术部及子部门员工")
	}
	if ids[empOther.ID] {
		t.Errorf("不应包含其他部门员工")
	}

	// admin 不受范围限制，全量可见（库中可能还有其他已提交的员工，因此只校验本次创建的 3 名都在）
	adminItems, adminTotal, err := svc.List("admin", 0, "", 0, 0, 0, 1, 1000)
	if err != nil {
		t.Fatalf("admin 员工列表失败: %v", err)
	}
	if adminTotal < 3 {
		t.Errorf("admin 期望至少 3 名员工全量，实际 %d", adminTotal)
	}
	adminIDs := map[uint]bool{}
	for _, it := range adminItems {
		adminIDs[it.ID] = true
	}
	if !adminIDs[empRoot.ID] || !adminIDs[empChild.ID] || !adminIDs[empOther.ID] {
		t.Errorf("admin 应可见全部 3 名本次创建员工")
	}
}

// TestBatchGenerate 批量生成：金额取最近历史工资单；无历史全 0；重复执行跳过
func TestBatchGenerate(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	// 事务内清空员工表，保证人数确定性
	if err := tx.Unscoped().Where("1 = 1").Delete(&models.Employee{}).Error; err != nil {
		t.Fatalf("清理员工表失败: %v", err)
	}
	emp1, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工1失败: %v", err)
	}
	emp2, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工2失败: %v", err)
	}
	// 员工1 有历史工资单（2026-06），员工2 无历史
	salaryRepo := repository.NewSalaryRepo(tx)
	if err := salaryRepo.Create(&models.SalaryRecord{
		EmployeeID: emp1.ID, Month: "2026-06",
		BaseSalary: 10000, Allowance: 500, Bonus: 1000, Deduction: 200,
		Status: "draft",
	}); err != nil {
		t.Fatalf("创建历史工资单失败: %v", err)
	}

	svc := newSalarySvc(tx)
	result, err := svc.BatchGenerate("2026-07")
	if err != nil {
		t.Fatalf("批量生成失败: %v", err)
	}
	if result.Generated != 2 {
		t.Errorf("期望生成 2 张，实际 %d", result.Generated)
	}
	if result.Skipped != 0 {
		t.Errorf("首次期望跳过 0，实际 %d", result.Skipped)
	}

	// 员工1 金额复制自历史；员工2 全 0
	var rec1, rec2 models.SalaryRecord
	if err := tx.Where("employee_id = ? AND month = ?", emp1.ID, "2026-07").First(&rec1).Error; err != nil {
		t.Fatalf("查询员工1工资单失败: %v", err)
	}
	if rec1.BaseSalary != 10000 || rec1.Allowance != 500 || rec1.Bonus != 1000 || rec1.Deduction != 200 {
		t.Errorf("员工1金额应复制历史，实际 %+v", rec1)
	}
	if rec1.NetSalary != 11300 {
		t.Errorf("员工1实发期望 11300，实际 %v", rec1.NetSalary)
	}
	if err := tx.Where("employee_id = ? AND month = ?", emp2.ID, "2026-07").First(&rec2).Error; err != nil {
		t.Fatalf("查询员工2工资单失败: %v", err)
	}
	if rec2.BaseSalary != 0 || rec2.Allowance != 0 || rec2.Bonus != 0 || rec2.Deduction != 0 {
		t.Errorf("员工2无历史应全 0，实际 %+v", rec2)
	}

	// 重复执行：已有工资单 → generated=0, skipped=2
	result2, err := svc.BatchGenerate("2026-07")
	if err != nil {
		t.Fatalf("二次批量生成失败: %v", err)
	}
	if result2.Generated != 0 {
		t.Errorf("二次期望生成 0，实际 %d", result2.Generated)
	}
	if result2.Skipped != 2 {
		t.Errorf("二次期望跳过 2，实际 %d", result2.Skipped)
	}
}

// TestBatchGenerateMonthInvalid 月份格式非法应报错
func TestBatchGenerateMonthInvalid(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	svc := newSalarySvc(tx)
	if _, err := svc.BatchGenerate("2026-13"); err == nil {
		t.Fatal("非法月份应报错")
	}
}

// TestAttendanceReport 考勤月报：按员工聚合各状态计数
func TestAttendanceReport(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	dept, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	emp, err := createTestEmployee(tx, uniqueStr("EMP"), "考勤员工", dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	attRepo := repository.NewAttendanceRepo(tx)
	statuses := []struct {
		day    int
		status string
	}{
		{1, "normal"}, {2, "normal"}, {3, "late"}, {4, "early_leave"}, {5, "absent"}, {6, "leave"},
	}
	for _, s := range statuses {
		if err := attRepo.Create(&models.Attendance{
			EmployeeID: emp.ID,
			WorkDate:   time.Date(2026, 7, s.day, 0, 0, 0, 0, time.Local),
			Status:     s.status,
		}); err != nil {
			t.Fatalf("创建考勤记录失败: %v", err)
		}
	}

	svc := NewAttendanceService(tx, attRepo, repository.NewEmployeeRepo(tx))
	report, err := svc.Report("admin", 0, "2026-07", 0)
	if err != nil {
		t.Fatalf("考勤月报失败: %v", err)
	}
	if report.Month != "2026-07" {
		t.Errorf("月报月份期望 2026-07，实际 %s", report.Month)
	}
	if len(report.List) != 1 {
		t.Fatalf("期望 1 名员工汇总，实际 %d", len(report.List))
	}
	item := report.List[0]
	if item.WorkDays != 6 || item.NormalDays != 2 || item.LateCount != 1 ||
		item.EarlyLeaveCount != 1 || item.AbsentDays != 1 || item.LeaveDays != 1 {
		t.Errorf("月报计数不正确: %+v", item)
	}
	if item.EmployeeName != "考勤员工" {
		t.Errorf("月报姓名不正确: %s", item.EmployeeName)
	}
}

// TestAttendanceReportEmployeeSelf employee 角色月报仅返回自己
func TestAttendanceReportEmployeeSelf(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	dept, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	emp1, err := createTestEmployee(tx, uniqueStr("EMP"), "员工一", dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建员工1失败: %v", err)
	}
	emp2, err := createTestEmployee(tx, uniqueStr("EMP"), "员工二", dept.ID, pos.ID)
	if err != nil {
		t.Fatalf("创建员工2失败: %v", err)
	}
	emp1User, err := createTestUser(tx, uniqueStr("u1"), "pass123456", "employee", &emp1.ID, 1)
	if err != nil {
		t.Fatalf("创建员工1账号失败: %v", err)
	}

	attRepo := repository.NewAttendanceRepo(tx)
	for _, e := range []*models.Employee{emp1, emp2} {
		if err := attRepo.Create(&models.Attendance{
			EmployeeID: e.ID,
			WorkDate:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local),
			Status:     "normal",
		}); err != nil {
			t.Fatalf("创建考勤失败: %v", err)
		}
	}

	svc := NewAttendanceService(tx, attRepo, repository.NewEmployeeRepo(tx))
	report, err := svc.Report("employee", emp1User.ID, "2026-07", 0)
	if err != nil {
		t.Fatalf("employee 月报失败: %v", err)
	}
	if len(report.List) != 1 || report.List[0].EmployeeID != emp1.ID {
		t.Fatalf("employee 角色应仅返回自己，实际 %+v", report.List)
	}
}

// TestOptionalUintJSON OptionalUint 反序列化：区分「未传」「显式 null」「指定值」
func TestOptionalUintJSON(t *testing.T) {
	// 未传：Set=false
	var omitted UpdateUserRequest
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatalf("解析空请求失败: %v", err)
	}
	if omitted.DeptScopeID.Set {
		t.Error("未传 dept_scope_id 时 Set 应为 false")
	}
	// 显式 null：Set=true 且 Value=nil
	var cleared UpdateUserRequest
	if err := json.Unmarshal([]byte(`{"dept_scope_id":null}`), &cleared); err != nil {
		t.Fatalf("解析 null 请求失败: %v", err)
	}
	if !cleared.DeptScopeID.Set || cleared.DeptScopeID.Value != nil {
		t.Errorf("显式 null 应 Set=true 且 Value=nil，实际 Set=%v Value=%v", cleared.DeptScopeID.Set, cleared.DeptScopeID.Value)
	}
	// 指定值：Set=true 且 Value 指向该值
	var bound UpdateUserRequest
	if err := json.Unmarshal([]byte(`{"dept_scope_id":5}`), &bound); err != nil {
		t.Fatalf("解析值请求失败: %v", err)
	}
	if !bound.DeptScopeID.Set || bound.DeptScopeID.Value == nil || *bound.DeptScopeID.Value != 5 {
		t.Errorf("指定值应 Set=true 且 Value=5，实际 Set=%v Value=%v", bound.DeptScopeID.Set, bound.DeptScopeID.Value)
	}
}

// TestUserDeptScopeUpdate 用户更新 dept_scope_id：传值绑定、传 null 清空
func TestUserDeptScopeUpdate(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "pass123456", "hr", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	dept := &models.Department{Name: uniqueStr("负责部门"), SortOrder: 1}
	if err := tx.Create(dept).Error; err != nil {
		t.Fatalf("创建部门失败: %v", err)
	}

	svc := NewUserService(repository.NewUserRepo(tx), repository.NewRoleRepo(tx))
	// 绑定
	deptID := dept.ID
	if err := svc.Update(u.ID, u.ID, UpdateUserRequest{DeptScopeID: OptionalUint{Set: true, Value: &deptID}}); err != nil {
		t.Fatalf("绑定 dept_scope_id 失败: %v", err)
	}
	got, err := repository.NewUserRepo(tx).FindByID(u.ID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if got.DeptScopeID == nil || *got.DeptScopeID != dept.ID {
		t.Errorf("期望绑定部门 %d，实际 %v", dept.ID, got.DeptScopeID)
	}
	// 显式 null 清空
	if err := svc.Update(u.ID, u.ID, UpdateUserRequest{DeptScopeID: OptionalUint{Set: true}}); err != nil {
		t.Fatalf("清空 dept_scope_id 失败: %v", err)
	}
	got2, err := repository.NewUserRepo(tx).FindByID(u.ID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if got2.DeptScopeID != nil {
		t.Errorf("期望清空为 null，实际 %v", got2.DeptScopeID)
	}
}
