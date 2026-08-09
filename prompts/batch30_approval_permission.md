# 任务批次 30：审批改权限制 + 考勤员工仅本人可见（后端）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户两项改进（架构师已确认可行）：① 请假/补卡审批从角色制改为**权限制**（admin 给角色勾审批权限点，谁有权限谁审批，流程配置页同步显示）；② 普通员工考勤**仅本人可见**（保留打卡权限，读接口按人过滤）。严格分层，统一响应。

## 一、新权限点（internal/database/seed.go）

permissions 表新增（module=approval）：
```go
{Name: "请假审批", Code: "leave:approve", Module: "approval"}
{Name: "补卡审批", Code: "attendance_correction:approve", Module: "approval"}
```
- **默认授予**（rolePerms）：hr 角色增加这两项（hr 继续承担终审职能，但改由权限控制）；manager 角色增加 `leave:approve`（经理继续一审请假）——保持开箱即用行为不变，但机制变为权限
- admin 角色**不默认授予**（admin 专注系统管理，审批业务归 hr/manager；admin 如需审批可自行勾选）

## 二、审批引擎支持 permission 类型（internal/models/approval.go + internal/service/approval.go）

### 模型
`ApprovalStep` 与 `ApprovalTask` 各增加：
```go
ApproverPermission string `gorm:"size:64" json:"approver_permission"` // 审批权限点 code，approver_type=permission 时使用
```
- ApproverType 合法值：`manager`（申请人部门经理，现状）/ `permission`（拥有指定权限点的所有启用用户为审批池）
- 旧的 `hr` / `position` 类型**兼容保留**（已存在流程实例仍按旧逻辑工作），但新配置不再使用；ApproverRef 保留供 position 兼容

### 引擎解析（三处口径一致）
1. `resolveStepApproverUserID`：permission 类型返回 (nil, nil)——任务按池创建（approver_id=NULL, approver_permission=权限code）
2. 创建任务时写入 `ApproverPermission: step.ApproverPermission`
3. **归属/校验**（checkTaskApprover + ListMyTasks + CountMyPendingTasks + approvalNotify 四处同步扩展）：
   - permission 池任务匹配：**当前用户拥有该权限点**（查 users.role_id → role_permissions JOIN permissions WHERE code=step.ApproverPermission）
   - manager 按 approver_id 匹配（现状）
   - hr 池仅 hr 角色（现状兼容）；position 按岗位（现状兼容）
   - admin 不出现业务审批（现状保持：admin 无审批权限点即不匹配）
4. 新增 helper：`UserHasPermission(db, userID uint, permCode string) (bool, error)`；`FindUsersByPermission(db, permCode string) ([]uint, error)`（JOIN users→role_permissions→permissions WHERE code=? AND users.status=1，去重）

### approvalNotify 扩展
- approverUserID 非 nil → 发该用户
- nil 且 type=permission → **广播 FindUsersByPermission 返回的所有用户**
- nil 且 type=hr → 广播所有 HR（现状）
- nil 且 type=position → 广播岗位员工（现状）

## 三、审批流程配置 API 扩展（internal/service/approval_admin.go + handler）

- `PUT /approval-flows/:code` 校验：approver_type ∈ {manager, permission, hr, position}（hr/position 兼容旧配置，新配置建议 manager/permission）；permission 时 `approver_permission` 必须 ∈ {leave:approve, attendance_correction:approve}；manager/hr/position 时 approver_permission 必须为空
- `GET /approval-flows` 响应 steps 增加 `approver_permission` 字段；同时顶层增加：
```json
"approval_permissions": [
  {"code":"leave:approve","name":"请假审批","roles":[{"id":2,"name":"人事"}]},
  {"code":"attendance_correction:approve","name":"补卡审批","roles":[...]}
]
```
（role 列表 = 拥有该权限点的角色名，供前端显示）

## 四、考勤员工仅本人可见（internal/service/attendance.go + repository）

- `AttendanceService.List`（含月报 List/统计相关接口）：当 `roleCode == "employee"` 时**强制** `Where("employee_id = ?", 当前用户 employee_id)`（无视传入的 department_id/employee_id 过滤，仅按本人）
- manager 维持现状（看全公司或本部门，按现有逻辑）；hr/admin 维持全量
- 打卡接口（clock-in/out）权限要求不变（attendance:write）——员工有该权限即可打卡，但列表只看到自己
- 补卡申请接口（correction）同理：employee 强制本人可见
- 边界：employee 无 employee_id（未关联档案）→ 列表返回空，打卡返回 403/400（现状一致）

## 五、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（新增 permission 池测试：manager 角色有 leave:approve 可审 task、无权限角色不可见/403、通知广播）
2. 启动验证（JWT 注入法）：
   - seed 后：hr 拥有 leave:approve + attendance_correction:approve；manager 拥有 leave:approve
   - GET /approval-flows 返回 approval_permissions 且 roles 正确
   - admin 把请假流程改为 2 级：manager → permission(leave:approve)；liuyang 提交请假 → task1 张伟、task2 permission 池(leave:approve 用户：hr、张伟？张伟也是 manager 有 leave:approve——**注意**：池含所有有权限用户；张伟一审后二审也出现在他待办？——若觉得不合理可改为「任务创建时排除已审批过的人」或保持池语义；实现为：permission 池任务所有人可见，审批后任务关闭即可，测试记录该行为）
   - hr 通过 task2 → 请假 approved；通知链路正常
   - **考勤**：liuyang（employee）GET /attendance → 仅本人记录；打卡 200；改看别的 employee_id → 仍返回本人（被强制过滤）；hr GET /attendance → 全量
3. 停掉进程，端口释放

## 汇报格式
权限点与种子、permission 池引擎改动、API 契约变化、审批权限制 E2E 实测、考勤本人可见实测、测试结果。
