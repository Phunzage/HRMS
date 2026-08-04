# 任务批次 3：员工 / 部门 / 岗位 CRUD API

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 2 已完成全部模型 + RBAC 认证。本批次实现**组织人事模块 API**。严格分层：handler → service → repository。所有接口统一响应、JWT 保护 + 权限点控制。

## 一、部门管理（permission: department:read / department:write）

`/api/v1/departments`
- `GET /api/v1/departments` — 树形列表（一次性返回全部，前端组树），每项含 children。权限 department:read
- `POST /api/v1/departments` — 创建 {name, parent_id, manager_id, sort_order, remark}。校验名称非空、父部门存在。权限 department:write
- `PUT /api/v1/departments/:id` — 更新。权限 department:write
- `DELETE /api/v1/departments/:id` — 删除（有子部门或员工时拒绝，返回 400 提示）。权限 department:write

## 二、岗位管理（permission: position:read / position:write）

`/api/v1/positions`
- `GET /api/v1/positions` — 分页列表，支持 ?keyword=&department_id= 过滤。权限 position:read
- `POST /api/v1/positions` — 创建 {name, code, department_id, level, remark}。code 唯一校验。权限 position:write
- `PUT /api/v1/positions/:id` — 更新。权限 position:write
- `DELETE /api/v1/positions/:id` — 删除（岗位被员工占用时拒绝）。权限 position:write

## 三、员工管理（permission: employee:read / employee:write）

`/api/v1/employees`
- `GET /api/v1/employees` — 分页列表，支持 ?keyword=&department_id=&status=&position_id= 过滤，keyword 匹配姓名/工号/手机号。每条附带 department_name、position_name。权限 employee:read
- `GET /api/v1/employees/:id` — 详情（含部门/岗位名称）。权限 employee:read
- `POST /api/v1/employees` — 创建。body 全字段；**EmpNo 自动生成**（格式 EMP + 4位序号，如 EMP0001，查当前最大序号+1，事务内防止并发重复）；创建员工的同时可选创建系统账号（body 传 create_user: true 时用 name 拼音或 username 字段 + 默认密码 123456，bcrypt）。权限 employee:write
- `PUT /api/v1/employees/:id` — 更新基本信息。权限 employee:write
- `DELETE /api/v1/employees/:id` — 软删除（若有用户账号一并禁用）。权限 employee:write

## 四、employee 自助接口（权限 employee:read 的"仅自己"逻辑）
- `GET /api/v1/employees/me` — 当前登录用户关联的员工档案（通过 user.employee_id）。JWT 保护，无需额外权限（登录即可）。

## 五、通用要求
- 所有列表响应走 response.PageOK；详情/创建/更新走 response.OK
- 创建/更新时校验必填字段，错误信息中文且具体
- repository 层用 GORM 查询，service 层做业务校验（如删除前检查）
- 员工列表 JOIN 部门/岗位返回名称；软删除默认过滤（GORM 自动）
- 分页参数 page/page_size 默认 1/20 上限 100

## 六、完成后必须执行
1. `cd backend && go build ./...`
2. 启动服务，用 admin 登录拿 token，依次验证：
   - 创建 2 个部门（技术部/人事部已有则跳过）
   - 创建岗位 ×3
   - 创建员工 ×3（检查 emp_no 自动生成 EMP0001/2/3）
   - 员工列表分页与 keyword 过滤
   - 用 hr 账号（hr/hr123）登录验证员工接口可访问
   - 用 employee 账号（liuyang/123456）调 GET /api/v1/employees/me 返回本人档案；调 GET /api/v1/employees 应返回 403
3. 停掉进程

## 汇报格式
新增文件清单、接口验证结果（每个接口的 curl 与响应摘要）、权限控制验证结果。
