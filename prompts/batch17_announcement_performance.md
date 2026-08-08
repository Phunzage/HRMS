# 任务批次 17：公告管理 + 绩效管理

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 16 已完成批量生成/月报/数据范围。本批次实现**公告管理**与**绩效管理**。严格分层，统一响应，JWT + 权限点。

## 一、公告管理（permission: announcement:read / announcement:write，新增权限点）

### 1. 模型 `internal/models/announcement.go`
```
Announcement: ID, Title(string, not null), Content(text, not null), PublisherID(uint, 发文人用户ID), PublisherName(string), IsPinned(bool, 默认 false, 置顶), Status(string: published/draft, 默认 published), PublishedAt(*time.Time), Timestamps, 软删除
```

### 2. 权限点与 seed
- permissions 追加：`announcement:read` 公告查看、`announcement:write` 公告编辑
- seedRolePermissions：hr 增加 announcement:read + announcement:write；employee/manager 增加 announcement:read（普通员工能看公告）；admin 放行
- 前端菜单 roles 后续批次配（本批次只做后端）

### 3. 公告 API
`/api/v1/announcements`
- `GET /announcements` — 分页列表（published 优先置顶+时间倒序），支持 ?keyword= 搜索标题。权限 announcement:read
- `POST /announcements` — 发布 {title, content, is_pinned}（默认 published，PublishedAt=now）。权限 announcement:write
- `PUT /announcements/:id` — 编辑（标题/内容/置顶/状态）。权限 announcement:write
- `DELETE /announcements/:id` — 删除。权限 announcement:write
- 可选：发布时给所有**启用且有员工账号**的用户发站内通知 type=announcement（员工太多可只发一条系统公告——实现为：不发站内通知，公告天然全员可见；简单优先，前端首页展示公告列表即可，不做通知）

## 二、绩效管理（permission: performance:read / performance:write，新增权限点）

### 1. 模型 `internal/models/performance.go`
```
PerformanceReview: ID, EmployeeID(uint, 索引, 被考核人), ReviewerID(uint, 考核人=部门经理), Period(string, "2026-H1"), Score(float64, 0-100, decimal(5,1)), Rating(string: S/A/B/C/D, 由分数映射), Comments(string, 评语), Status(string: draft/submitted/approved), SubmittedAt(*time.Time), ApprovedAt(*time.Time), Timestamps, 软删除
```
- Rating 映射：90+ → S，80-89 → A，70-79 → B，60-69 → C，<60 → D（service 层计算，不存原始输入）
- 联合唯一：employee_id + period（同一周期一人一条绩效）

### 2. 权限点与 seed
- permissions 追加：`performance:read` 绩效查看、`performance:write` 绩效编辑
- seedRolePermissions：hr 增加 performance:read + performance:write（HR 主导绩效）；manager 增加 performance:read + performance:write（经理给自己部门员工打分）；employee 增加 performance:read（看自己的绩效结果）；admin 放行

### 3. 绩效 API
`/api/v1/performance`
- `GET /performance` — 分页列表，支持 ?period=&status=&employee_id=&department_id= 过滤。权限 performance:read；manager 仅本部门；employee 仅自己
- `GET /performance/mine` — 当前用户自己的绩效记录（登录即可，含 rating 中文）
- `POST /performance` — 创建 {employee_id, period, score, comments}。校验：周期格式 YYYY-H1/H2；分数 0-100；同一员工同一周期唯一（400）；ReviewerID 自动=当前用户（若为 manager 或 hr/admin）。权限 performance:write
- `PUT /performance/:id` — 更新（仅 draft 可改）。权限 performance:write
- `PUT /performance/:id/submit` — 提交（draft → submitted）。权限 performance:write
- `PUT /performance/:id/approve` — 批准（submitted → approved，hr/admin 可操作）。权限 performance:write
- `DELETE /performance/:id` — 删除（仅 draft）。权限 performance:write
- 创建/提交/批准均写审计日志 module=performance

## 三、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`
2. 启动验证（admin）：
   - 发布公告 ×2（一条置顶）→ 列表置顶在前；liuyang 登录 GET /announcements 返回 200（有 announcement:read）
   - 创建绩效（liuyang, 2026-H1, score=92）→ rating 自动 S；重复创建 → 400
   - update → submit → approve 状态流；approved 后再改 → 400
   - zhangwei（manager）创建本部门员工绩效成功；创建人事部员工绩效 → 403 或按实现返回「非本部门员工」400
   - liuyang GET /performance/mine 看到自己的记录
   - hr 登录看到全部绩效
3. 停掉进程，端口释放

## 汇报格式
新增/修改文件清单、公告验证结果、绩效完整状态流验证结果、评分映射验证、权限矩阵。
