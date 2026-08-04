# 任务批次 4：考勤 + 请假审批流 API

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 3 已完成组织人事 API。本批次实现**考勤与请假审批流**。严格分层 handler → service → repository，统一响应，JWT + 权限点。

## 一、考勤管理（permission: attendance:read / attendance:write）

`/api/v1/attendance`
- `GET /api/v1/attendance` — 分页列表，支持 ?employee_id=&work_date=&month=2026-07&status= 过滤（month 用 LIKE '2026-07%' 匹配 work_date）。每条带 employee_name、department_name。权限 attendance:read
- `POST /api/v1/attendance/check-in` — 打卡上班。按当前登录用户 → employee_id，今日已打卡则返回 400「今日已打卡」。记录 CheckIn=now，Status 由系统按规则算：>9:00 记为 late。权限 attendance:write（员工自助也能打）
- `POST /api/v1/attendance/check-out` — 打卡下班。今日无 CheckIn 返回 400；CheckOut=now；若 <18:00 记为 early_leave；已有 CheckOut 返回 400。权限 attendance:write
- `POST /api/v1/attendance/manual` — HR 手工补录 {employee_id, work_date, check_in, check_out, status}（HR 专用）。权限 attendance:write
- `PUT /api/v1/attendance/:id` — 修改记录。权限 attendance:write
- `DELETE /api/v1/attendance/:id` — 删除。权限 attendance:write

## 二、请假管理（permission: leave:read / leave:write）

`/api/v1/leaves`
- `POST /api/v1/leaves` — 员工提交请假申请 {leave_type, start_date, end_date, reason}。校验：start <= end、天数>0；按自然日算天数（end-start+1）；**校验假期余额**（annual 年假年度总额 10 天、sick 病假 5 天、personal 事假 3 天、maternity 产假 98 天、marriage 婚假 3 天、other 其他 5 天；若已用+本次 > 总额则返回 400「余额不足」）。自动找审批人：员工所属部门 manager（Employee 表中 department 的 ManagerID 对应的员工）；找不到则状态直接 approved。权限 leave:write（仅自己）
- `GET /api/v1/leaves` — 分页列表。admin/hr 看全部；manager 看本部门员工 + 待自己审批的；employee 只看自己的。支持 ?status=&employee_id= 过滤。权限 leave:read
- `GET /api/v1/leaves/mine` — 当前用户自己的申请列表（含余额信息）。登录即可
- `GET /api/v1/leaves/balance` — 当前用户各类型假期余额 {leave_type, total, used, remaining}。登录即可
- `PUT /api/v1/leaves/:id/approve` — 审批通过（manager/admin/hr）。校验自己是审批人或 admin/hr；状态必须 pending；通过后**扣减余额**（LeaveBalance 当年该类型 used += days，无记录则创建）。权限 leave:write
- `PUT /api/v1/leaves/:id/reject` — 驳回 {reject_reason}。权限 leave:write
- `PUT /api/v1/leaves/:id/cancel` — 申请人取消 pending 的申请。登录即可（仅本人）

## 三、通用要求
- 日期用字符串 "YYYY-MM-DD" 传输；LeaveRequest.StartDate/EndDate 存 time.Time（只取日期部分）
- 所有列表 JOIN 员工表返回 employee_name、department_name
- 审批动作在 service 层用事务（approve 时更新状态+扣余额原子完成）
- 状态机严格：pending → approved/rejected；pending → cancelled
- 错误消息中文、具体（如「今日已打卡」「年假余额不足：剩余 5 天，本次需 6 天」）

## 四、完成后必须执行
1. `cd backend && go build ./...`
2. 启动服务验证：
   - admin 登录 → 查看 zhangwei（经理）的 employee_id
   - 用 liuyang（后端组员工）登录 → check-in 打卡 → check-out 打卡 → GET /attendance 确认有记录
   - liuyang 提交年假申请 3 天 → 查询状态 pending
   - 用 zhangwei（manager 角色，liuyang 的部门经理）登录 → GET /leaves 看到该申请 → approve
   - liuyang 再 GET /leaves/balance 确认 used=3、remaining=7
   - 重复打卡/重复审批等边界返回 400
3. 停掉进程

## 汇报格式
新增文件清单、每个接口的验证结果（curl 与响应）、审批流完整链路结果、余额校验结果。
