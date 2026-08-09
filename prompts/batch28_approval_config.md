# 任务批次 28：可配置审批流程 — 模型扩展 + 引擎增强 + admin 管理 API

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户新需求：**admin 可配置审批流程**（每个公司职能划分不同，批假/补卡可能一级/两级/三级，每级审批人可为部门经理/HR/指定岗位）。现有通用审批流引擎架构支持此扩展，按方案实施。严格分层，统一响应。

## 一、模型扩展（internal/models/approval.go）

### ApprovalStep 增加字段
```go
ApproverRef uint `gorm:"default:0" json:"approver_ref"` // 审批人引用：approver_type=position 时为岗位 ID；manager/hr 为 0
```
- ApproverType 合法值扩展为：`manager`（申请人部门经理）/ `hr`（HR 池，任意 hr）/ `position`（指定岗位池，该岗位所有在职员工可审）
- Name 为步骤显示名（如「部门经理审批」「人事专员审批」「总经理终审」）

### ApprovalTask 增加字段（用于池任务匹配）
```go
ApproverRef uint `gorm:"default:0" json:"approver_ref"` // position 池任务存岗位 ID；manager/hr 为 0
```

AutoMigrate 会自动加列（已有表）。数据库迁移后 seed 的默认两步流程保持：step1 manager、step2 hr。

## 二、引擎增强（internal/service/approval.go）

### 1. resolveStepApproverUserID 扩展
```go
switch step.ApproverType {
case "hr":       return nil, nil                    // HR 池，approver_id=NULL
case "position": return nil, nil                    // 岗位池，approver_id=NULL + ApproverRef=岗位ID
case "manager":  // 现状：申请人部门经理，无经理兜底 hr
}
```

### 2. 创建任务时写入 ApproverRef
- StartApproval / ApproveTask 推进时：`ApproverRef: step.ApproverRef`（manager/hr 为 0）

### 3. 归属/权限校验扩展（checkTaskApprover + ListMyTasks + CountMyPendingTasks 三处口径一致）
当前口径：`(approver_id = ? OR (approver_type='hr' AND approver_id IS NULL))`
新口径增加 position 池：
```go
// position 池任务：当前用户岗位 == task.ApproverRef 才可见/可审
// 判断方法：查当前用户 employee_id → employees.position_id == task.ApproverRef
// SQL 或 Go 内联过滤均可；建议复用已加载的 user.employee_id 后查岗位
```
- checkTaskApprover：HR 池仅 hr 角色（现状）；**position 池仅该岗位的在职员工用户**（查 employee.position_id 匹配）；manager 按 approver_id
- ListMyTasks / CountMyPendingTasks：`(approver_id=? OR (type='hr' AND NULL) OR (type='position' AND approver_ref=?))`，position 的 approver_ref 需查当前用户岗位——用子查询或先查岗位再拼条件
- admin 仍不出现任何业务审批任务（现状保持）

### 4. approvalNotify 扩展
- approverUserID 非 nil → 发该用户（现状）
- nil 且 type=hr → 广播所有 HR（现状）
- **nil 且 type=position → 广播该岗位所有在职员工的用户**（新增 `FindUsersByPositionID(tx, ref) ([]uint, error)`：JOIN employees ON users.employee_id=employees.id WHERE employees.position_id=ref AND employees.status=1 AND users.status=1；无则跳过）

### 5. bizSummary / 其余逻辑不变

## 三、流程配置管理（新增 service/approval_admin.go + handler/approval_admin.go）

### 模型：新增权限点 `approval_flow:read` / `approval_flow:write`（seed.go rolePerms 中 admin 增加这两项；permissions 表 seed 同步加）

### API（路由前缀 /api/v1/approval-flows）
1. `GET /approval-flows`（approval_flow:read，admin）→ 返回所有流程及步骤：
```json
{ "code": 0, "data": { "list": [
  { "id":1, "code":"leave", "name":"请假审批流程", "status":"active",
    "steps": [ {"id":11, "step_order":1, "approver_type":"manager", "approver_ref":0, "name":"部门经理审批"},
               {"id":12, "step_order":2, "approver_type":"hr", "approver_ref":0, "name":"HR终审"} ] }
] } }
```
2. `PUT /approval-flows/:code`（approval_flow:write，admin）请求体：
```json
{ "name": "请假审批流程", "steps": [
  {"step_order":1, "approver_type":"manager", "approver_ref":0, "name":"部门经理审批"},
  {"step_order":2, "approver_type":"position", "approver_ref":4, "name":"人事专员复核"},
  {"step_order":3, "approver_type":"hr", "approver_ref":0, "name":"HR终审"}
] }
```
- 校验：code 存在（leave/attendance_correction）；steps 1~5 级；step_order 必须 1..N 连续；approver_type ∈ {manager, hr, position}；position 时 approver_ref 必须 >0 且岗位存在；hr/manager 时 approver_ref 必须为 0
- 实现：事务内删除该流程旧步骤 + 批量插入新步骤；已存在的 ApprovalInstance/ApprovalTask 不受影响（只影响新申请）
- 返回更新后的流程

3. router.go 注册：`GET /approval-flows`、`PUT /approval-flows/:code`（admin 专属权限点）

## 四、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（新增审批配置测试 + position 池测试）
2. 启动验证（JWT 注入法）：
   - 默认流程 GET /approval-flows：leave/correction 各两级（manager→hr）
   - admin PUT 把 leave 改为三级（manager→position 人事专员→hr），200；非法（step_order 不连续 / position 无 ref / 4 级）→ 400
   - liuyang 提交请假 → 新实例按三级流程：task1 manager(张伟)、task2 position 池(岗位4)、task3 hr 池
   - 张伟通过 → 人事专员（建一个岗位4员工账号如 hr2 或王五？王五岗位正是4）→ 王五(wang, employee角色, 岗位4) 的待办可见 task2 并可通过 → hr 终审通过 → 请假 approved
   - 越权：岗位不是4的普通员工看不到/不能审 task2（403）
   - 通知：task2 创建时王五收到「新的请假申请待审批」；task3 创建时 hr 收到
   - 角标：王五 pending_approvals 在 task2 期间 =1
   - 改回两级（manager→hr）后再提一条 → 恢复原行为
3. 停掉进程，端口释放

## 汇报格式
模型变更、引擎增强点、API 契约、三级流程端到端实测矩阵（含岗位池越权）、测试结果、改回两级回归。
