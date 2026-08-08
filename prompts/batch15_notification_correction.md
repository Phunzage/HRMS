# 任务批次 15：站内通知系统 + 补卡申请审批

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 14 已完成后端安全加固与上传。本批次实现**站内通知**与**补卡申请**。严格分层，统一响应，JWT + 权限点。

## 一、站内通知系统

### 1. 模型 `internal/models/notification.go`
```
Notification: ID, UserID(uint, 索引), Type(string: leave_approval/leave_result/attendance_correction/announcement/system), Title(string), Content(string), Link(string, 可空，前端跳转路由如 /leaves), IsRead(bool, 默认 false, 索引), CreatedAt(time)
```
无软删除。

### 2. 通知服务 `internal/service/notification.go`
- `Send(db, userID uint, ntype, title, content, link string) error` — 创建通知
- 供其他 service 调用（请假审批、补卡审批、公告发布时发通知）

### 3. 通知 API（登录即可，无权限点）
`/api/v1/notifications`
- `GET /notifications` — 分页列表（仅当前用户），支持 ?is_read= 过滤，每条带相对时间（created_at 即可）
- `GET /notifications/unread-count` — 返回 `{count: N}` 未读数（顶栏铃铛用）
- `PUT /notifications/:id/read` — 标记已读（校验归属当前用户，否则 403）
- `PUT /notifications/read-all` — 全部标记已读（当前用户）

### 4. 业务集成（修改现有 service）
- **请假审批**（service/leave.go）：
  - 提交申请时 → 给审批人（approver_id 对应员工关联的用户）发通知：type=leave_approval, title「新的请假申请待审批」, content「{申请人名} 提交了 {类型} {天数} 天（{起}~{止}）」, link=/leaves
  - 审批通过时 → 给申请人发通知：type=leave_result, title「请假审批通过」, content=「{类型} {天数} 天（{起}~{止}）已通过」, link=/leaves
  - 驳回时 → 给申请人发通知：type=leave_result, title「请假审批驳回」, content=「原因：{reject_reason}」, link=/leaves
  - 注意：审批人可能是员工无账号（approver_id 对应员工没有 user 记录）→ 跳过发送不报错
- **考勤补卡**（本批次新增，见下）：提交/审批通过/驳回均发通知

## 二、补卡申请（考勤修正）

### 1. 模型 `internal/models/attendance_correction.go`
```
AttendanceCorrection: ID, EmployeeID(uint, 索引), WorkDate(time.Time, type:date), Reason(string), CorrectionType(string: check_in/check_out), CorrectionTime(string, "HH:MM"), Status(string: pending/approved/rejected), ApproverID(*uint), ApprovedAt(*time.Time), RejectReason(string), Timestamps, 软删除
```

### 2. 补卡 API（permission: attendance:write）
`/api/v1/attendance/corrections`
- `POST /corrections` — 员工提交补卡 {work_date, correction_type(check_in/check_out), correction_time, reason}。校验：work_date 不得晚于今天、不得早于 30 天前；审批人=部门经理（同请假逻辑，无经理则直接 approved）。发通知给审批人。权限 attendance:write
- `GET /corrections` — 列表。admin/hr 看全部；manager 看本部门员工；employee 只看自己。支持 ?status=&employee_id= 过滤。权限 attendance:read
- `GET /corrections/mine` — 当前用户自己的补卡记录（登录即可）
- `PUT /corrections/:id/approve` — 审批通过：校验自己是审批人或 admin/hr，状态必须 pending；**同事务内**：① 更新该员工 work_date 的考勤记录（找到 attendance 行：check_in 类型则设置 CheckIn=该日期+CorrectionTime；check_out 类似；无考勤行则创建一条，status 按时间规则重算）② 状态置 approved ③ 发通知给申请人。权限 attendance:write
- `PUT /corrections/:id/reject` — 驳回 {reject_reason}，发通知。权限 attendance:write
- `PUT /corrections/:id/cancel` — 申请人取消 pending。登录即可

### 3. 路由注册注意
`/attendance/corrections` 必须注册在 `/attendance/:id` **之前**（否则 corrections 被当 id 捕获）。

## 三、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`
2. 启动验证（admin 登录拿 token）：
   - liuyang 提交请假 → zhangwei 登录 GET /notifications/unread-count 应 ≥1，列表可见该通知，标记已读后未读数减一
   - 张伟审批通过 → liuyang 的通知列表出现「请假审批通过」
   - liuyang 提交补卡（昨天 check_in 09:10）→ zhangwei 审批通过 → 查 GET /attendance 该日 check_in=09:10 且 status 按规则（9:10 > 9:00 → late）
   - 补卡重复审批 → 400
3. 停掉进程，端口释放

## 汇报格式
新增/修改文件清单、通知链路验证结果（提交→通知审批人→审批→通知申请人）、补卡完整链路结果、边界校验结果。
