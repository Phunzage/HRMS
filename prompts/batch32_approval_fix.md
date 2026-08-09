# 任务批次 32：修复审批链 + 部门经理递归解析 + 审批角色制重构（后端）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户实测发现 4 个后端相关问题，根因已由架构师定位，按方案修复。严格分层，统一响应。

## 问题 1 + 2：部门经理递归解析 + 审批兜底硬指派 hr 的 bug

**现状（bug 确认）**：
- `ResolveManagerUserID` 只查**直属部门** manager_id。张佳在「算法组」（parent=研发部），算法组 manager_id 为空 → 找不到经理 → 兜底 `FindHRUser` **硬指派给 hr 用户**（approver_id=2）→ 刘尧（研发部经理）收不到通知/角标、无法办理；hr 反而收到（**即使取消 hr 的 leave:approve 权限也会收到**——因为是 approver_id 硬指派，与权限无关）

### 修复 A：ResolveManagerUserID 递归向上解析
`internal/service/approval.go` 的 `ResolveManagerUserID`：从员工所在部门开始，**循环向上（parent_id）找第一个有 manager_id 的部门**，返回其经理的用户 ID：
```go
func ResolveManagerUserID(db *gorm.DB, employeeID uint) (*uint, error) {
    // 1. 查员工部门 deptID
    // 2. for deptID != 0:
    //      查 departments.manager_id、parent_id
    //      若 manager_id != nil → 查其用户 ID，有则返回
    //      deptID = parent_id（继续向上）
    // 3. 返回 nil（全链无经理）
}
```
- 算法组 → 研发部(manager=23 刘尧) → 返回刘尧 user ID ✅
- 保持幂等，不破坏现有部门（刘尧在研发部、技术部等已配经理的场景）

### 修复 B：manager 步骤无经理时不再硬指派 hr
`resolveStepApproverUserID` 的 manager 分支：递归解析后仍无经理时，**返回 (nil, nil)**（不再 FindHRUser 兜底）。此时任务创建为 `approver_type='manager', approver_id=NULL, approver_permission=''`——成为「未指派任务」。

**未指派 manager 任务的归属规则**（ListMyTasks / CountMyPendingTasks / checkTaskApprover 三处同步）：approver_id 为 NULL 的 manager 任务 → 该业务类型的审批权限用户可审（leave→leave:approve，attendance_correction→attendance_correction:approve，通过 inst.BizType 推断）。**hr 取消 leave:approve 后即不再可见/可审** ✅

### 修复 C：未指派 manager 任务的通知
`approvalNotify`：approverUserID 为 nil 且 step 类型为 manager 时 → 按 BizType 广播给拥有对应审批权限的用户（复用 FindUsersByPermission）。**hr 有权限才收到，取消权限即不再收到** ✅

## 问题 4：审批流程配置从「按权限审批」改为「按角色审批」

用户需求：admin 配置审批步骤时选择**具体角色**（如部门经理、人事专员、副总经理）并排序，而不是「按权限审批」。

### 模型
`ApprovalStep` 增加：
```go
ApproverRoleID uint `gorm:"default:0" json:"approver_role_id"` // 审批角色 ID，approver_type=role 时使用
```
`ApprovalTask` 同样增加 `ApproverRoleID`（用于 role 池任务匹配）。
- ApproverType 合法值扩展：`manager` / `role`（指定角色池：该角色的所有用户可审）/ 兼容保留 hr / position / permission（旧数据不破坏，新配置不再用 permission）
- 新配置使用：manager（部门经理动态解析）+ role（指定角色）

### 引擎
- `resolveStepApproverUserID`：role 类型返回 (nil, nil)——按池创建（approver_id=NULL, approver_role_id=角色ID）
- 创建任务写入 `ApproverRoleID: step.ApproverRoleID`
- **role 池任务匹配**（ListMyTasks / CountMyPendingTasks / checkTaskApprover 三处）：当前用户的 role_id == task.ApproverRoleID 且 status=1 → 可见/可审
- `approvalNotify`：nil + role → 广播该角色的所有启用用户（FindUsersByRoleID：users WHERE role_id=? AND status=1）
- **优先级=步骤顺序**：step_order 1→N 即审批优先级，前端排序保存，引擎按序推进（现状已支持）

### API
- `PUT /approval-flows/:code` 校验：approver_type ∈ {manager, role, hr, position, permission}；**role 时 approver_role_id 必须 >0 且角色存在**，approver_permission 必须为空；manager/hr/position 时 approver_role_id 必须为 0；permission 时 approver_permission 合法（兼容旧数据）
- `GET /approval-flows`：steps 增加 `approver_role_id` + `approver_role_name`（JOIN roles 名称）；顶层 `approval_permissions` 保留（前端展示权限-角色映射）

### 默认种子（seed.go）
- leave：step1 manager（部门经理）→ step2 role（hr 角色 ID）「HR终审」
- attendance_correction：step1 manager → step2 role（hr 角色 ID）
- 行为与现状一致（hr 终审），但机制变为「角色池」

## 问题 3：通知逻辑全面检查

1. 确认审批推进全链路通知：
   - StartApproval：步骤1有指派→发审批人；未指派→按权限/角色池广播
   - ApproveTask 推进：下一步骤同样逻辑
   - 终审通过→申请人收到结果通知（leave_result）；驳回→申请人收到结果通知
2. **排除重复通知**：张佳场景（有经理时）只发刘尧，hr 不应收到（修复 A/B/C 后自然满足）；角色池广播按角色去重
3. 检查 `bizSummary` 中 LeaveRequest/AttendanceCorrection 的申请人名与摘要正常
4. 边界：manager 任务已指派但审批人账号禁用（status=0）→ 通知跳过该用户（Send 内部已有？确认）

## 完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（新增：递归经理测试、role 池测试、未指派 manager 任务归属测试）
2. 启动验证（JWT 注入法）：
   - **张佳场景**：张佳(zhangjia, 算法组) 提交病假 → task1 **approver_id=刘尧(3)**（递归解析到研发部经理）→ 刘尧待办/角标/通知全部出现；hr 无通知（未指派给 hr）✅
   - **hr 取消权限场景**：admin 取消 hr 的 leave:approve → 新请假（算法组无经理的测试员工）→ 任务未指派 → hr 待办/角标无、通知无；恢复权限 → 重新可见 ✅
   - **role 池**：admin 把请假流程 step2 改为 role(hr) → 新请假 → task2 approver_role_id=hr 角色、approver_id=NULL → hr 用户可见可审 ✅
   - **递归边界**：顶级部门（parent=0）无经理 → 未指派 → 权限用户可审
3. 停掉进程，端口释放

## 汇报格式
修复 A/B/C 前后对比（张佳场景、hr 取消权限场景）、role 池实测、通知链路检查结果、测试结果。
