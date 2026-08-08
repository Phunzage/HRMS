# 任务批次 23：审批流前端改造 — 通用待办中心 + 审批进度展示

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 21/22 已完成：通用审批流引擎（固定两级：部门经理→HR终审）、请假与补卡流程迁移、admin/hr 职责分离。本批次改造**前端**。中文界面，daisyUI + Tailwind + lucide-react 图标（禁止 emoji）。

## 一、API 封装（frontend/src/api/approval.ts 新建）

- `fetchMyApprovals(params)` GET /approvals/mine?status=
- `fetchApprovalProgress(bizType, bizId)` GET /approvals/biz?biz_type=&biz_id=
- `approveTask(taskId, comment?)` PUT /approvals/tasks/:taskId/approve
- `rejectTask(taskId, comment)` PUT /approvals/tasks/:taskId/reject

类型：`ApprovalTaskItem`（task_id, biz_type, biz_title, applicant_name, leave_type/days 或补卡摘要, created_at, comment…）、`ApprovalProgress`（instance status, current_step, steps: [{step_order, name, approver_name, status, comment, acted_at}]）

## 二、LeavePage.tsx 改造（请假页）

1. **「待我审批」Tab 重做**：不再调 `/leaves?status=pending`，改调 `fetchMyApprovals({status:'pending'})`，只显示**属于我的审批任务**（经理看到第1步，hr 看到第2步终审）：
   - 表格列：申请人/部门/类型/天数/事由/提交时间/操作（通过、驳回）
   - 通过：调 `approveTask` → 若返回的实例仍 pending（还有第2步）toast「已通过，等待 HR 终审」；若 approved toast「审批已通过」
   - 驳回：弹窗填原因 → `rejectTask`
   - 空态「暂无待审批任务」
2. **「我的申请」Tab**：每条申请增加**审批进度**展示——调 `fetchApprovalProgress('leave', id)`，用简单时间线（步骤名 ✓/○ + 审批人 + 时间 + 意见）。可折叠或直接显示在行下（选择：行内展开小时间线，点击「进度」按钮展开）
3. 余额/提交/取消逻辑不变；提交后 toast「已提交，等待部门经理审批」

## 三、AttendancePage.tsx 改造（补卡 Tab）

同上：
1. 补卡列表的待审批项显示审批按钮（仅任务归属人）：通过/驳回 → 通用接口
2. 我的补卡记录显示进度时间线（fetchApprovalProgress('attendance_correction', id)）
3. 提交补卡 toast「已提交，等待部门经理审批」

## 四、统一待办中心（新页面 /approvals，侧边栏「我的审批」）

考虑到经理和 hr 都要处理请假+补卡两类待办，做一个**统一待办页**更清晰：
- 路由 `/approvals`，MainLayout 菜单「我的审批」（icon: Inbox，roles: manager/hr/admin）
- Tab：待我审批（默认）/ 我处理的
- 待我审批：调 fetchMyApprovals({status:'pending'})，卡片式列表（daisyUI card）：申请人/类型 badge（请假/补卡）/摘要/提交时间 → 「审批」按钮打开 Modal（显示进度时间线 + 通过/驳回按钮 + 驳回原因输入）
- 我处理的：fetchMyApprovals({status:'done'}) 历史记录（任务状态 badge：已通过/已驳回）
- 审批操作后列表自动刷新 + toast

## 五、MainLayout 菜单与守卫

- 新增菜单项「我的审批」→ /approvals（icon: Inbox，roles: admin/hr/manager）
- LeavePage/AttendancePage 的审批入口保留（两处都能操作），但**核心入口引导到 /approvals**
- admin 角色：业务写按钮隐藏（批次 22 已改），但「我的审批」可见（admin 可终审 hr 请假? 不——admin 无 leave:write。**注意**：审批任务归属校验基于 ApproverID，HR 步骤 ApproverID 为 NULL 时允许 hr/admin 审批。保持批次 21 的引擎逻辑，前端 admin 也能看到 HR 步骤任务）

## 六、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（JWT 注入法，见批次 20 附录）：
   - liuyang 提交请假 → zhangwei 登录 /approvals 看到待办 → 通过 → hr 登录 /approvals 看到终审待办 → 通过 → liuyang 我的申请里进度时间线显示两步完成
   - 补卡同样走一遍
   - 驳回场景：经理驳回 → 进度时间线显示驳回
3. 停掉两个进程，端口释放

## 汇报格式
新增/修改文件清单、build 结果、统一待办页与进度时间线实测结果、与后端 /approvals 接口联调确认。
