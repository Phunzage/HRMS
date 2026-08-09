# 任务批次 26：权限体系重构 — admin 专注系统管理、hr 全公司口径

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。架构师已确定新的权限模型，按方案实施。严格分层，统一响应。

## 背景与目标

用户澄清：hr 管理**整个公司**人力资源（非固定部门）；admin 专注系统管理。新权限模型：

| 能力 | admin | hr |
|------|:---:|:---:|
| 用户/角色/审计（系统管理） | ✅ | ❌ |
| 仪表盘总览 | ✅（监督） | ✅ |
| **公告**（发布/管理，用于系统维护通知） | ✅ | ✅ |
| 业务模块（员工/考勤/请假/薪酬/招聘/绩效/部门/岗位） | ❌（全部移除，连只读也无） | ✅ 全量 |
| 「我的审批」/ HR 池终审 | ❌ | ✅ |

## 一、hr 数据范围清空（恢复全公司）

1. `database/seed.go` 的 `seedUsers`：所有 hr 角色账号**不设置** dept_scope_id（保持 NULL）
2. **现有 hr 账号清理**：用 psql 执行 `UPDATE users SET dept_scope_id = NULL WHERE username='hr';`（以及任何 role=hr 的账号，如 testbatch12）
3. `ResolveScope` 逻辑保留（dept_scope_id 为 NULL 时返回 nil=全量）——数据范围机制保留但当前所有 hr 账号全量；`Users.tsx` 前端「负责部门」字段后续批次处理（批次 27）
4. 验证：hr GET /dashboard/stats → open_job_count 与 admin 一致（当前=4）；hr 招聘列表可见全公司岗位；hr 可新增其他部门（如人事部）岗位

## 二、admin 权限收敛（seed.go 的 rolePerms 调整）

### admin 权限点（最终）
```
user:read, user:write, role:read, role:write, audit:read,
dashboard:read, announcement:read, announcement:write
```
**移除**：employee:read/write, department:read/write, position:read/write, attendance:read/write, leave:read/write, salary:read/write, recruitment:read/write, performance:read/write（admin 连业务只读都没有）

### 注意
- 中间件已移除「admin 无条件放行」（批次 22），admin 按 role_permissions 校验——本批次只改 seed 权限配置，重启后 seed 自愈逻辑自动同步（删除 admin 不在配置内的权限关联）
- **迁移提醒**：若 seed 自愈逻辑只处理 rolePerms 里列出的角色，需确认 admin 也在其中（检查 seedRolePermissions 是否包含 admin 的清理逻辑；批次 22 起 admin 应已有权限行配置，若没有则补充 admin 键）

### 验证
- admin：GET /dashboard/stats 200、GET /announcements 200、POST /announcements 200（可发维护公告）、GET /users 200、PUT /roles 200
- admin：GET /employees → **403**、GET /attendance → **403**、GET /salaries → **403**、GET /recruitment/jobs → **403**（业务全不可见）

## 三、admin 移除「我的审批」/ HR 池终审能力

当前审批引擎 HR 池步骤（approver_id=NULL）允许「任何 hr/admin」审批——需收紧为**仅 hr 角色**。

### 修改 `internal/service/approval.go`
1. `ApproveTask`/`RejectTask` 的归属校验：HR 池任务（ApproverID 为 NULL 且步骤 ApproverType=="hr"）时，**仅允许 role_code=="hr" 的用户审批**；admin 不再放行
2. `ListMyTasks` 同理：HR 池任务对 admin 不可见（admin 的待办列表不应出现业务审批）
3. 兜底 `FindHRUser`/`FindAllHRUserIDs` 中若有「找不到 hr 则找 admin」的 fallback：保留（防流程卡死），但一旦有 hr 账号存在则优先 hr——说明：当前系统有 hr 账号，admin 不会成为审批人
4. 验证：
   - liuyang 提交请假 → zhangwei 通过 → **admin** GET /approvals/mine → **空**（不再看到 HR 池任务）；admin 尝试审批 HR 池任务 → **403**
   - **hr** GET /approvals/mine → 看到 HR 池任务；审批 → 成功

## 四、完成后必须执行
1. 先跑数据清理：`UPDATE users SET dept_scope_id=NULL WHERE dept_scope_id IS NOT NULL;`（全部 hr 账号）
2. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（审批测试如有 admin 放行用例需同步更新）
3. 启动验证（JWT 注入法，见批次 20 附录）：
   - hr 仪表盘 open_job_count = admin 口径（全公司）
   - admin 权限矩阵：仪表盘/公告/系统 200；业务 403；我的审批空/HR 池 403
   - hr 可新增人事部岗位（招聘接口不限制部门）
   - 请假全链路回归：提交→经理→hr 终审正常
4. 停掉进程，端口释放

## 汇报格式
权限矩阵调整前后对比、数据清理结果、admin/hr 权限实测矩阵、审批引擎 HR 池收紧验证、测试结果。
