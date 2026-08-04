# 任务批次 2：数据模型 + RBAC 完整实现

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 1 已建好后端脚手架（config/database/response/router），请在其基础上**扩展模型并实现 RBAC 中间件与认证模块**。严格遵循既有架构（handler → service → repository 分层），不改变现有文件风格。

## 本批次目标
1. 补齐全部业务模型（部门/岗位/员工/考勤/请假/假期余额/工资/招聘/候选人）
2. 实现 JWT 认证中间件 + RBAC 权限中间件
3. 实现认证接口：登录、当前用户信息、修改密码
4. AutoMigrate 全模型 + Seed 增强（HR 账号、示例部门/岗位/员工）

## 一、新增模型（internal/models/，每个文件一个模型或关联集合，GORM 标签齐全）

### org.go — 组织架构
- `Department`: ID, Name(not null), ParentID(默认0，树形), ManagerID(员工ID，可空), SortOrder(默认0), Remark, Timestamps
- `Position`: ID, Name(not null), Code(unique), DepartmentID, Level(默认0), Remark, Timestamps

### employee.go
- `Employee`: ID, EmpNo(unique, not null), UserID(可空), Name(not null), Gender(1男/2女/3其他), BirthDate(*time.Time), IDCard, Phone, Email, DepartmentID, PositionID, HireDate(*time.Time), Status(1在职/2离职/3试用), Address, Avatar, Remark, Timestamps（全部用软删除）

### attendance.go
- `Attendance`: ID, EmployeeID, WorkDate(time.Time, 只存日期), CheckIn(*time.Time), CheckOut(*time.Time), Status(string: normal/late/early_leave/absent), Remark, Timestamps。联合唯一索引 (employee_id, work_date)。

### leave.go
- `LeaveRequest`: ID, EmployeeID, LeaveType(string: annual/sick/personal/marriage/maternity/other), StartDate, EndDate, Days(float64), Reason, Status(string: pending/approved/rejected/cancelled), ApproverID(可空), ApprovedAt(*time.Time), RejectReason, Timestamps
- `LeaveBalance`: ID, EmployeeID, LeaveType, Year(int), TotalDays, UsedDays(float64)。联合唯一 (employee_id, leave_type, year)

### salary.go
- `SalaryRecord`: ID, EmployeeID, Month(string "2026-07"), BaseSalary, Allowance, Bonus, Deduction, NetSalary, Status(string: draft/confirmed/paid), Remark, Timestamps

### recruitment.go
- `JobOpening`: ID, Title(not null), DepartmentID, PositionID, Headcount(int), Requirement(text), Status(string: open/closed), Timestamps
- `Candidate`: ID, JobOpeningID, Name(not null), Phone, Email, Status(string: applied/interview/offer/hired/rejected), InterviewDate(*time.Time), Remark, Timestamps

### 已有模型补充
- `User` 增加字段：EmployeeID(*uint, 可空)

## 二、RBAC 与认证

### middleware/auth.go
- `JWTAuth()`：从 `Authorization: Bearer <token>` 解析 JWT（HS256，secret 来自 config），claims 含 user_id、username、role_code。失败返回 401 统一错误。成功把 user_id/username/role_code 写入 gin.Context。
- `RequirePermission(code string)`：读取当前用户 role_code → 查该角色是否拥有该权限点。admin 角色直接放行。无权限返回 403。

### 权限种子（Seed 中写入 role_permissions）
- admin：全部（可用通配判断，直接放行即可，无需建行）
- hr：employee:read/write、department:read/write、position:read/write、attendance:read/write、leave:read、salary:read/write、recruitment:read/write
- manager：employee:read、attendance:read、leave:read/write（审批）、department:read
- employee：employee:read（仅自己）、attendance:read（仅自己）、leave:read/write（仅自己）、salary:read（仅自己）

### handler/auth.go + service/auth.go + repository/user_repo.go
- `POST /api/v1/auth/login`：body {username, password}。校验 bcrypt，成功返回 {token, user:{id, username, full_name, role_code, employee_id}}。失败统一 401「用户名或密码错误」。
- `GET /api/v1/auth/me`：JWT 保护，返回当前用户信息 + 角色 + 关联员工。
- `PUT /api/v1/auth/password`：body {old_password, new_password}，校验旧密码后更新，bcrypt 存储。

## 三、Seed 增强（database.go 或独立 seed.go）
- 角色：admin/hr/manager/employee（已有）
- 账号：admin/admin123(admin)、hr/hr123(hr)、zhangwei/123456(manager)、liuyang/123456(employee)——全部 bcrypt，幂等写入
- 部门：技术部、人事部、财务部、市场部（层级：技术部下设前端组/后端组）
- 岗位：后端开发工程师、前端开发工程师、人事专员、财务专员、部门经理
- 员工：给 zhangwei 挂到技术部部门经理岗位、liuyang 挂到后端组后端开发工程师，emp_no 自动生成（EMP0001 起）
- 权限点与 role_permissions 关联

## 四、完成后必须执行
1. `cd backend && go build ./...` 编译通过
2. `go run ./cmd/server` 启动，验证：
   - `curl -s -X POST localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}'` 返回 token
   - 用 token 调 `curl -s localhost:8080/api/v1/auth/me` 返回用户信息
   - 验证无 token 访问 `/api/v1/auth/me` 返回 401
3. 停掉进程

## 汇报格式
列出新增文件、编译结果、login/me 接口实际返回内容（脱敏 token）、seed 的账号与部门岗位员工清单。
