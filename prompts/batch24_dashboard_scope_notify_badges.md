# 任务批次 24：仪表盘数据范围 + HR池通知修复 + 角标接口

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。三个问题根因已由架构师定位，按方案修复。严格分层，统一响应。

## 问题 1：仪表盘统计未应用数据范围（hr 与 admin 口径不一致）

**根因**：`internal/service/dashboard.go` 的 `Stats()` 无用户参数，纯全局统计；而 hr 账号绑定了 dept_scope_id（技术部），招聘/员工列表按范围过滤（hr 看到 3 个岗位），仪表盘却是全局数（4 个）——用户删除/新增岗位后对比两个不同口径的数字，误以为不更新。

### 修复
1. `Stats()` 签名改为 `Stats(roleCode string, userID uint)`：
   - 调 `ResolveScope(db, userID, roleCode)`（已存在，internal/service/scope.go）得到 deptIDs（hr 有范围时非 nil；admin/manager/employee 为 nil=全量）
   - **需要 db 依赖**：DashboardService 目前只有 repo，构造时传入 *gorm.DB（NewDashboardService 签名变更，router 相应更新）
2. **按范围过滤的统计项**（deptIDs 非 nil 时 WHERE department_id IN deptIDs）：
   - `OpenJobCount(deptIDs)`：job_openings.department_id IN deptIDs
   - `CandidateCount(deptIDs)`：JOIN job_openings 按部门过滤
   - `TotalEmployees / ActiveEmployees / MonthNewEmployees / MonthResigned`：employees.department_id IN deptIDs（MonthResigned 注意 status=2 AND updated_at 条件叠加）
   - `DepartmentDistribution(deptIDs)`：仅统计范围内部门（保持树形展开——直接按 employees.department_id 分组即可）
   - `MonthAttendanceRate(deptIDs)`：JOIN employees 按部门过滤
   - `TodayLeaveCount(deptIDs)`：JOIN employees 按部门过滤
   - `PendingLeaveCount(deptIDs)`：JOIN employees 按部门过滤
   - `SalaryTrend(deptIDs)`：JOIN employees 按部门过滤
   - `RecruitTrend(deptIDs)`：JOIN job_openings 按部门过滤
   - `PendingSalaryCount(deptIDs)`：JOIN employees 按部门过滤
3. admin/manager/employee 传入 nil → 保持全量（现状不变）；hr 有范围 → 过滤
4. `handler/dashboard.go`：`Stats(c)` 里传 `middleware.GetRoleCode(c)`、`middleware.GetUserID(c)`

### 验证
- hr（绑定技术部）GET /dashboard/stats → open_job_count 应与 hr 招聘列表范围内数量一致（当前应=3：技术部 id3/5 + 测试组 id10）；admin 仍=4
- hr 删除技术部一个 open 岗位 → open_job_count 2；admin 仍 3（若被删是技术部岗位）；新增技术部岗位 → 各自 +1
- 员工/出勤/工资/趋势等 hr 口径均≤admin 口径

## 问题 2：HR 池步骤不发通知（hr 收不到终审通知）

**根因**：`internal/service/approval.go` 的 `resolveStepApproverUserID` 对 hr 步骤返回 `(nil, nil)`（HR 池，ApproverID 留空由任一 hr/admin 审批）；`approvalNotify` 里 `if approverUserID == nil { return }` **直接跳过**——所以创建 HR 终审任务时不发任何通知，hr 永远收不到。

### 修复
1. 新增 `FindAllHRUserIDs(db) ([]uint, error)`：查 role=hr 且 status=1 的所有用户 ID（找不到则查 admin）
2. `approvalNotify` 改为 `approvalNotify(tx, inst, approverUserID *uint, step *models.ApprovalStep)`：
   - approverUserID 非 nil → 发给该用户（现状）
   - approverUserID 为 nil 且 step.ApproverType == "hr" → **发给所有 HR 用户**（FindAllHRUserIDs 逐个 Send；无 HR 则发 admin）
   - 其他 nil 情况保持跳过
3. 调用点（StartApproval 建步骤1任务、ApproveTask 推进建步骤2任务）把当前 step 传入
4. 通知内容沿用现有格式（「新的请假申请待审批」/「新的补卡申请待审批」，link /leaves 或 /attendance）

### 验证
- liuyang 提交请假 → zhangwei 通过 → hr 登录 **GET /notifications 应收到「新的请假申请待审批」（HR终审）通知**，未读数 +1
- 补卡同样验证

## 问题 3：导航栏角标接口 + 公告已读机制（后端部分）

### 1. 公告已读模型
`internal/models/announcement_read.go`：
```
AnnouncementRead: ID, UserID(uint, 索引), AnnouncementID(uint, 索引), CreatedAt
// 联合唯一 (user_id, announcement_id)
```
AutoMigrate 注册。

### 2. 公告 API 扩展（handler/announcement.go + service/repository）
- `PUT /announcements/:id/read`（JWT，登录即可）：标记当前用户已读该公告（幂等，upsert）
- `PUT /announcements/read-all`（JWT，登录即可）：标记当前用户已读全部 published 公告
- `GET /announcements` 列表每条带 `is_read bool`（JOIN announcement_reads 判断当前用户）
- 注意路由顺序：`/announcements/read-all` 静态段须在 `/:id` 之前

### 3. 角标汇总接口（新增 handler/badge.go + service/badge.go，或并入 notification handler）
`GET /api/v1/badges`（JWT，登录即可，无权限点）返回：
```json
{ "pending_approvals": N, "unread_announcements": N, "unread_notifications": N }
```
- pending_approvals：当前用户作为审批人（ApproverID 匹配，或 HR 池步骤）的 pending 任务数——复用 approval 服务的 ListMyTasks 逻辑或新增 CountMyPendingTasks
- unread_announcements：published 公告总数 - 当前用户已读数
- unread_notifications：复用现有未读数逻辑
- router 注册 `GET /badges`

### 完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...`（不回归，审批测试可能需适配新签名）
2. 启动验证（JWT 注入法，见批次 20 附录）：
   - 问题1：hr/admin 的 open_job_count 口径对比（hr=3, admin=4）；删/增岗位各自变化
   - 问题2：hr 收到终审通知（未读数+1）
   - 问题3：GET /badges 返回三项；标记已读后 unread_announcements 减少；GET /announcements 带 is_read
3. 停掉进程，端口释放

## 汇报格式
各问题修复说明、验证结果矩阵（hr/admin 口径对比、hr 通知、badges 接口响应示例）、测试结果。
