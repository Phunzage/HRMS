# 任务批次 10：用户/角色管理 API + 操作审计日志

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。项目已有完整 HRMS（认证/组织/考勤/请假/薪酬/招聘/仪表盘）。本批次新增**用户与角色管理**和**操作审计日志**。严格分层 handler → service → repository，统一响应，JWT + 权限点。

## 一、新增模型

### 1. `internal/models/audit_log.go`
```
AuditLog: ID, UserID(uint, 索引), Username(string, 索引), Module(string: user/role/employee/department/position/attendance/leave/salary/recruitment/auth), Action(string: create/update/delete/login/logout/approve/reject/pay/confirm/reset_password), ResourceID(string, 可空，如 "employee:5"), Detail(string, 描述), IP(string), CreatedAt(time)
```
无软删除（审计日志永不删除）。

### 2. 权限点新增（seed.go 的 permissions 数组追加）
- `user:read` 用户查看 / `user:write` 用户编辑
- `role:read` 角色查看 / `role:write` 角色编辑
- `audit:read` 审计日志查看

### 3. seed 角色权限更新（seedRolePermissions）
- hr 增加：`user:read`、`user:write`、`role:read`、`audit:read`（角色**编辑** role:write 仅 admin）
- admin 依旧全放行（中间件直接放行，无需建行）
- 清理逻辑已存在，会自动同步

## 二、审计日志机制

新建 `internal/middleware/audit.go`：
- `Record(db *gorm.DB, c *gin.Context, module, action, resourceID, detail string)` 辅助函数：从上下文取当前用户（无登录用户时 UserID=0/Username="system"），记录 IP（`c.ClientIP()`），写入 AuditLog。内部用 `go func` 异步写或直接写均可（直接写更可靠，量小可接受）。
- 在以下位置调用 Record（各 handler 或 service 的关键操作处）：
  - auth：登录成功、登录失败（UserID=0）、修改密码
  - user：创建/更新/重置密码/删除用户
  - role：创建/更新/删除角色
  - employee：创建/更新/删除员工
  - department：创建/更新/删除部门
  - position：创建/更新/删除岗位
  - attendance：手工补录/修改/删除考勤
  - leave：提交/审批通过/驳回/取消请假
  - salary：创建/更新/确认/发放工资
  - recruitment：职位与候选人创建/更新/删除

**实现建议**：在各 handler 成功操作后调用 `middleware.Record(...)`，detail 用简短中文描述（如「创建员工 王强 EMP0003」）。

## 三、用户管理 API（permission: user:read / user:write）

`/api/v1/users`
- `GET /users` — 分页列表，支持 ?keyword=&role_id=&status= 过滤（keyword 匹配 username/full_name），每条含 role_name。权限 user:read
- `POST /users` — 创建 {username, password(min 6), full_name, email, role_id}。username 唯一校验（含软删除检查 Unscoped），bcrypt 存密。权限 user:write
- `PUT /users/:id` — 更新 {full_name, email, role_id, status(0/1)}。不可修改自己角色为禁用（若目标是 admin 自己，禁止 status=0）。权限 user:write
- `PUT /users/:id/password` — 重置密码 {password}。权限 user:write
- `DELETE /users/:id` — 删除（软删除）；admin 自己不可删；被删除用户关联的员工不受影响。权限 user:write
- `GET /users/roles` — 全部角色列表（供前端下拉，不要求 role:read，登录即可，简化前端）

## 四、角色管理 API（permission: role:read / role:write）

`/api/v1/roles`
- `GET /roles` — 列表，每条含 permissions（该角色权限点 code 数组，前端用于勾选）。权限 role:read
- `GET /permissions` — 全部权限点（分组返回：模块 → 权限点数组）。权限 role:read
- `POST /roles` — 创建 {name, code, description, permissions: [code...]}。code 唯一校验；permissions 关联写入 role_permissions。权限 role:write
- `PUT /roles/:id` — 更新名称/描述/权限（permissions 全量替换：先删后插，事务）。权限 role:write
- `DELETE /roles/:id` — 删除；角色下有用户时拒绝（400「该角色下存在用户，无法删除」）；admin 角色不可删。权限 role:write

## 五、审计日志查询 API（permission: audit:read）

`/api/v1/audit-logs`
- `GET /audit-logs` — 分页列表，支持 ?module=&action=&username=&start_date=&end_date= 过滤（start_date/end_date 格式 YYYY-MM-DD，过滤 created_at 日期）。权限 audit:read

## 六、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...` 全绿
2. 启动服务验证（admin 登录）：
   - 创建用户 tester/123456（角色 employee）→ 列表可见 → 更新角色 → 重置密码 → 用新密码登录成功
   - 创建角色「临时角色」+ 勾选 2 个权限 → 列表可见权限 → 编辑权限 → 删除（无用户）→ 再建一个挂到用户上验证删除被拒
   - 做一次员工创建 + 工资确认 → GET /audit-logs 能看到对应记录
   - hr 登录验证可访问 /users、/audit-logs，但 POST /roles 返回 403
3. 停掉进程

## 汇报格式
新增文件清单、接口验证结果、审计日志落库内容示例、权限矩阵验证结果。
