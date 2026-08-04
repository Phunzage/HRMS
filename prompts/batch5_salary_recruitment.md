# 任务批次 5：薪酬 + 招聘 + 统计仪表盘 API

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 4 已完成考勤与请假。本批次实现**薪酬、招聘与统计仪表盘**。严格分层 handler → service → repository，统一响应，JWT + 权限点。

## 一、薪酬管理（permission: salary:read / salary:write）

`/api/v1/salaries`
- `GET /api/v1/salaries` — 分页列表，支持 ?month=&employee_id=&status=&department_id= 过滤。每条带 employee_name、department_name。权限 salary:read（employee 角色仅看自己）
- `POST /api/v1/salaries` — 创建工资单 {employee_id, month, base_salary, allowance, bonus, deduction}。**NetSalary 自动计算** = base + allowance + bonus - deduction。同一员工同一月份唯一（重复返回 400）。权限 salary:write
- `PUT /api/v1/salaries/:id` — 更新（仅 draft 状态可改，改后重算 net）。权限 salary:write
- `PUT /api/v1/salaries/:id/confirm` — 确认（draft → confirmed）。权限 salary:write
- `PUT /api/v1/salaries/:id/pay` — 发放（confirmed → paid）。权限 salary:write
- `GET /api/v1/salaries/mine?month=` — 员工查看自己的工资单（仅自己，登录即可）

## 二、招聘管理（permission: recruitment:read / recruitment:write）

`/api/v1/recruitment/jobs`
- `GET /api/v1/recruitment/jobs` — 职位列表（支持 ?status=&keyword= 过滤），带 department_name。权限 recruitment:read
- `POST /api/v1/recruitment/jobs` — 创建职位 {title, department_id, position_id, headcount, requirement, status}。权限 recruitment:write
- `PUT /api/v1/recruitment/jobs/:id` — 更新。权限 recruitment:write
- `DELETE /api/v1/recruitment/jobs/:id` — 删除。权限 recruitment:write

`/api/v1/recruitment/candidates`
- `GET /api/v1/recruitment/candidates` — 候选人列表（?job_id=&status= 过滤），带 job_title。权限 recruitment:read
- `POST /api/v1/recruitment/candidates` — 添加候选人 {job_opening_id, name, phone, email, remark}，status 默认 applied。权限 recruitment:write
- `PUT /api/v1/recruitment/candidates/:id` — 更新（含状态流转）。权限 recruitment:write
- `DELETE /api/v1/recruitment/candidates/:id` — 删除。权限 recruitment:write

## 三、统计仪表盘（权限：admin/hr 可访问）

`GET /api/v1/dashboard/stats`
- 返回：员工总数、在职员工数、本月新增员工数、本月离职数
- 本月出勤率（= 正常出勤人次 / 总考勤人次）、今日请假人数
- 待审批请假数、待确认工资单数、在招职位数、候选人数
- 部门人数分布 [{department_name, count}]
- 近 6 个月招聘趋势 [{month, count}]（按候选人 created_at）
- 近 6 个月工资总额 [{month, total_net}]（按 confirmed/paid 工资单）

## 四、通用要求
- 金额字段用 float64，展示保留两位小数
- 月份格式 "YYYY-MM" 字符串
- 所有列表 JOIN 返回名称字段；错误中文具体

## 五、完成后必须执行
1. `cd backend && go build ./...`
2. 启动验证：
   - admin 创建 3 张工资单（不同员工同月 1 张、再验证同月重复被拒）
   - confirm → pay 状态流转
   - 创建 2 个招聘职位 + 3 个候选人，状态流转 applied → interview → offer
   - liuyang 登录 GET /salaries/mine 只能看到自己的
   - GET /dashboard/stats 返回完整统计
3. 停掉进程

## 汇报格式
新增文件清单、各接口验证结果、统计接口返回的完整 JSON、权限验证结果。
