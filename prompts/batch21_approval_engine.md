# 任务批次 21：通用审批流引擎（模型 + 服务 + API）+ 请假流程迁移

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。架构师已确定方案：将请假/补卡的单级审批重构为**通用审批流引擎**，采用**固定两级审批：部门经理 → HR 终审**。严格分层，统一响应，JWT + 权限点。本批次先实现引擎 + 迁移请假，补卡下一批次迁移。

## 一、设计总览

```
提交申请 ──▶ 创建 ApprovalInstance(流程实例) + 第1步任务(部门经理)
             │
             经理审批 ──通过──▶ 创建第2步任务(HR终审)
             │                    │
             │                    HR审批 ──通过──▶ 实例approved → 回调业务(更新状态/扣余额/联动考勤/通知)
             │                    │
             │                    HR审批 ──驳回──▶ 实例rejected → 回调业务(状态rejected/通知)
             │
             经理驳回 ──▶ 实例rejected → 回调业务
提交后申请人可取消(pending时) ──▶ 实例cancelled
```

**固定两级**（所有请假/补卡都走）：步骤1 approver_type=manager（员工所属部门经理）；步骤2 approver_type=hr（HR 终审）。

## 二、新模型（internal/models/approval.go）

```go
// ApprovalFlow 流程定义（静态配置）
ApprovalFlow: ID, Code(string, unique, "leave"/"attendance_correction"), Name(string), Status(string: active/inactive), Timestamps

// ApprovalStep 流程步骤定义
ApprovalStep: ID, FlowID(uint), StepOrder(int), ApproverType(string: manager/hr), Name(string, "部门经理审批"/"HR终审"), Timestamps
// 唯一约束 (flow_id, step_order)

// ApprovalInstance 审批实例（一次申请一个实例）
ApprovalInstance: ID, FlowID(uint), BizType(string: leave/attendance_correction), BizID(uint), Status(string: pending/approved/rejected/cancelled), CurrentStep(int, 当前进行到第几步), CreatedBy(uint, 提交人用户ID), CreatedAt, UpdatedAt
// 索引: (biz_type, biz_id) 唯一（一个业务记录只对应一个实例）

// ApprovalTask 审批任务（每个审批人一条）
ApprovalTask: ID, InstanceID(uint, 索引), StepOrder(int), ApproverType(string), ApproverID(*uint, 审批人用户ID，HR步骤可为空=取任意hr), Status(string: pending/approved/rejected), Comment(string), ActedAt(*time.Time), CreatedAt
```

AutoMigrate 注册这 4 个模型。

## 三、审批引擎服务（internal/service/approval.go）

### 核心函数（全部用 *gorm.DB 依赖注入，事务内调用）

1. `StartApproval(tx, bizType string, bizID uint, applicantUserID uint) (*models.ApprovalInstance, error)`
   - 查该 bizType 的 active ApprovalFlow 及 steps（按 step_order 排序）
   - 创建 ApprovalInstance（status=pending, current_step=1）
   - 解析**步骤1审批人**（manager）：根据业务查到申请人员工 → 员工部门 → department.manager_id → 该员工的关联用户；无经理则找任意 hr 用户（fallback，防止流程卡死）
   - 创建第1条 ApprovalTask（pending）
   - **发通知**给步骤1审批人（type=leave_approval 或 attendance_correction，title「新的请假申请待审批」/「新的补卡申请待审批」，link=/leaves 或 /attendance）
   - 返回实例

2. `ApproveTask(tx, taskID uint, approverUserID uint, comment string) (*models.ApprovalInstance, error)`
   - 校验任务存在、status=pending、ApproverID==approverUserID（或步骤是 hr 类型且当前用户角色为 hr/admin → 兼容，但**优先匹配 ApproverID**；实现：ApproverID 匹配优先，若步骤为 hr 类型且 ApproverID 为 NULL 则允许任何 hr/admin 用户审批）
   - 任务 → approved，记录 comment/acted_at
   - 若还有下一步骤（step_order+1）：
     - 实例 current_step 前进
     - 解析下一步审批人（hr 步骤：ApproverID 留空，由任一 hr/admin 审批；manager 步骤同理解析部门经理）
     - 创建下一条任务 + **发通知**
   - 若无下一步：实例 → approved
   - 返回实例（调用方根据 status 决定是否执行业务回调）

3. `RejectTask(tx, taskID uint, approverUserID uint, comment string) error`
   - 校验同上；任务 → rejected；实例 → rejected

4. `CancelApproval(tx, instanceID uint, applicantUserID uint) error`
   - 仅 pending 且当前步骤任务状态为 pending 时允许申请人取消；实例 → cancelled

5. `ResolveManagerUserID(db, employeeID uint) (*uint, error)` — 员工→部门→manager_id→关联用户ID（无则 nil）
6. `FindHRUser(db) (*uint, error)` — 找第一个 role=hr 且 status=1 的用户 ID（无则找 admin）
7. `ListMyTasks(db, userID uint, status string) ([]TaskItem, error)` — 我作为审批人的 pending/历史任务，JOIN 业务表带出申请人/类型/时间/链接
8. `GetInstanceByBiz(db, bizType string, bizID uint) (*models.ApprovalInstance, error)`
9. `ListInstanceTasks(db, instanceID uint) ([]ApprovalTask, error)` — 审批进度（前端时间线用）

### 回调机制（业务状态在**最终审批通过**时更新）

审批引擎**不直接**改业务表，通过 handler 层在 `ApproveTask` 返回后判断 `instance.Status == "approved"` 再调用业务 service 的「终审通过」方法。为保持事务一致性，handler 里用 `db.Transaction` 包住「ApproveTask + 业务回调」。

## 四、请假流程迁移（internal/service/leave.go 重构）

### 现状
`POST /leaves` 提交 → 直接找部门经理存 approver_id，status=pending；`PUT /leaves/:id/approve|reject` 直接改状态+扣余额+联动考勤+通知。

### 改为
1. **提交** `POST /leaves`：
   - 校验逻辑保留（余额/日期/天数）
   - 创建 LeaveRequest（status 仍存 pending 保持兼容，approver_id 字段保留但不再作为审批依据）
   - 事务内调用 `StartApproval(tx, "leave", leaveID, userID)` → 创建审批实例 + 经理任务 + 通知经理
2. **审批**：删除旧的 `PUT /leaves/:id/approve` 和 `/reject` 逻辑，由**通用审批接口**替代（见下）。
3. **终审通过回调** `ApplyLeaveApproved(tx, leaveID)`：原 approve 里的业务逻辑（状态→approved、扣减余额、联动考勤标记 leave、通知申请人）全部移入此函数，由审批引擎 final 后在同一事务调用。
4. **驳回回调** `ApplyLeaveRejected(tx, leaveID, reason)`：状态→rejected、通知申请人（含驳回原因）。
5. **取消** `PUT /leaves/:id/cancel`：仅 pending 可取消 → 同时 `CancelApproval`。

### 通用审批 API（新增 handler/approval.go，router 注册）

`/api/v1/approvals`（JWT 保护，无新增权限点，靠任务归属校验）
- `GET /approvals/mine?status=` — 我的待审批/历史任务列表（含 biz_type、申请人名、业务摘要、时间）。manager 与 hr 都从这里看待办。
- `GET /approvals/:instanceId/tasks` — 某实例的审批进度（时间线：步骤名/审批人/状态/时间/意见）
- `PUT /approvals/tasks/:taskId/approve` — body {comment?}，审批通过（引擎推进，final 时同事务执行业务回调）
- `PUT /approvals/tasks/:taskId/reject` — body {comment(必填)}，驳回（实例→rejected+业务回调）
- `GET /approvals/biz?biz_type=&biz_id=` — 按业务查实例+进度（前端详情页展示）

权限：taskId 归属校验（ApproverID==当前用户，或 hr 类型步骤由 hr/admin 审批）；越权 403「无权审批该任务」。

## 五、Seed 流程定义

`database/seed.go` 增加 `seedApprovalFlows(db)`：
- ApprovalFlow{code:"leave", name:"请假审批", status:"active"} + 2 steps（1:manager 部门经理审批 / 2:hr HR终审）
- ApprovalFlow{code:"attendance_correction", name:"补卡审批", status:"active"} + 2 steps（同上）
- 幂等：code 已存在则跳过

## 六、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（不回归）
2. 启动验证（JWT 注入法，见批次20提示词附录）：
   - liuyang 提交请假 2 天 → GET /approvals/mine（zhangwei 视角）看到 1 条待办；leave_requests 状态 pending
   - zhangwei 审批通过 → GET /approvals/mine（zhangwei）该条消失；GET /approvals/biz?biz_type=leave&biz_id=X 显示 current_step=2、task1 approved
   - hr 登录 GET /approvals/mine 看到该任务 → 审批通过 → leave 状态 approved、余额扣减、考勤联动（验证原有业务回调生效）
   - 再测一条：经理驳回 → leave 状态 rejected、申请人收到通知
   - 越权：liuyang 尝试审批 zhangwei 的任务 → 403
3. 停掉进程，端口释放

## 汇报格式
新增/修改文件清单、审批引擎设计落地说明、完整两级审批链路验证结果（含通知/余额/考勤联动）、越权验证、测试结果。
