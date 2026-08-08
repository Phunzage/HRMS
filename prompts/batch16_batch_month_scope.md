# 任务批次 16：薪酬批量生成 + 考勤月报 + 数据范围权限

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 15 已完成通知与补卡。本批次实现**薪酬批量生成**、**考勤月报**与**数据范围权限**。严格分层，统一响应，JWT + 权限点。

## 一、薪酬批量生成（permission: salary:write）

`POST /api/v1/salaries/batch-generate`（必须注册在 `/salaries/:id` 之前）
- body `{month: "2026-08"}`（格式校验 YYYY-MM）
- 逻辑：查当月尚未有工资单的**在职员工**（employee.status=1），为每人创建一张 draft 工资单：
  - 基础金额来源：取该员工**最近一张**历史工资单的 base_salary/allowance/bonus/deduction 复制；无历史记录则全部 0
  - NetSalary = base + allowance + bonus - deduction 自动计算
- 返回 `{generated: N, skipped: M}`（skipped = 已有工资单或非在职）
- 审计日志：module=salary, action=batch_generate, detail「批量生成 2026-08 工资单 N 张」

## 二、考勤月报（permission: attendance:read）

`GET /api/v1/attendance/report?month=2026-08`
- 返回当月**每个有考勤记录的员工**（含 0 记录员工可选）的汇总：
  ```
  { list: [{ employee_id, employee_name, department_name, work_days, late_count, early_leave_count, absent_days, leave_days, normal_days }], month: "2026-08" }
  ```
- 统计口径：按 attendance.status 计数（normal_days=status normal 条数，late_count=late，early_leave_count=early_leave，absent_days=absent，leave_days=leave；work_days = 该员工当月考勤总条数）
- 支持 ?department_id= 过滤；employee 角色只返回自己（service 层过滤）
- 排序按 department_id, employee_id

## 三、数据范围权限（HR 负责部门）

**目标**：hr 账号可绑定「负责部门」，则员工/考勤/请假/薪酬/招聘列表只显示该部门及其子部门的数据；未绑定则全量（保持现状）。

### 1. 模型
- `users` 表新增 `DeptScopeID *uint`（可空，HR 负责的部门 ID）
- seed 不需要改（默认空=全量）

### 2. 范围工具 `internal/service/scope.go`
- `ResolveScope(db, userID, roleCode) (scopeDeptIDs []uint, err error)`：
  - admin / manager / employee 角色 → 返回 nil（不过滤）
  - hr 角色：查 user.DeptScopeID；为空 → nil；非空 → 收集该部门及**所有子孙部门** ID（递归查 departments.parent_id）返回
- 列表过滤：employee_repo/attendance_repo/leave_repo/salary_repo/recruitment_repo 的列表查询增加**可选** deptIDs 参数（nil 不过滤），service 层在列表方法开头调 ResolveScope 后传入
  - employees：WHERE department_id IN (...)
  - attendance：JOIN employees，WHERE employees.department_id IN (...)
  - leaves：JOIN employees，同
  - salaries：JOIN employees，同
  - recruitment jobs：WHERE department_id IN (...)
- manager 角色**已有**本部门过滤逻辑的保持原样，本批次只做 hr 的

### 3. 用户管理接口支持设置
- `PUT /users/:id` 请求体增加可选字段 `dept_scope_id`（可传 null 清空）
- `GET /users` 列表每条附带 dept_scope_name（部门名，空则 null）
- 前端（后续批次）在用户编辑 Modal 中给 hr 角色显示「负责部门」下拉

## 四、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`
2. 启动验证（admin）：
   - 创建 2 个员工分布在技术部（EMP 已有）与人事部 → 给 hr 账号设 dept_scope_id=技术部
   - 用 hr 登录 GET /employees → 只看到技术部及子部门（前端组/后端组）员工；GET /salaries、/attendance 同理
   - 用 admin 登录 → 全量不变
   - POST /salaries/batch-generate {"month":"2026-08"} → generated>0；重复执行 → generated=0, skipped>0
   - GET /attendance/report?month=2026-08 → 汇总正确（含 leave 统计）
   - GET /salaries/export 与 /employees/export 也受数据范围影响（若实现简单则保持一致）
3. 停掉进程，端口释放

## 汇报格式
新增/修改文件清单、数据范围权限验证矩阵（hr 有范围/无范围、admin、employee）、批量生成结果、月报 JSON 示例。
