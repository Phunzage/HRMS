# 任务批次 18：前端新功能 — 通知/补卡/公告/绩效/月报/批量生成/暗色模式/头像

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。后端已完成：安全加固（验证码/锁定/refresh）、文件上传、通知系统、补卡、薪酬批量生成、考勤月报、数据范围、公告、绩效。本批次实现**全部对应前端功能**。中文界面，daisyUI + Tailwind + lucide-react 图标（**禁止 emoji**）。

## 一、API 封装（`frontend/src/api/`）

1. `auth.ts` 更新：login 增加 captcha_id/captcha_code 参数；新增 `getCaptcha()` GET /auth/captcha、`refreshToken(refreshToken)` POST /auth/refresh
2. 新建 `notification.ts`：list / unreadCount / markRead / markAllRead
3. 新建 `upload.ts`：`uploadFile(file)` POST /upload（FormData）
4. 新建 `correction.ts`：corrections 列表/提交/审批/驳回/取消/mine
5. 新建 `announcement.ts`：列表/创建/更新/删除
6. 新建 `performance.ts`：列表/创建/更新/提交/批准/删除/mine
7. `salary.ts` 增加：`batchGenerate(month)`、`monthlyReport(month)`（考勤月报放 attendance.ts）
8. `attendance.ts` 增加：`getReport(month)`
9. `user.ts` 更新：updateUser 支持 dept_scope_id

## 二、登录页改造（Login.tsx）
- 增加验证码输入框 + 验证码图片（点击刷新，`<img src={captchaImage}>`，captchaImage 是 base64 data URI 直接可用）
- 登录成功后存储 `refresh_token`（zustand store 增加字段）
- axios 响应拦截器：401 时若存在 refresh_token → 调 /auth/refresh 重放原请求（**最多重试 1 次**，避免死循环）；失败则清空跳 /login

## 三、顶栏通知铃铛（MainLayout.tsx）
- 顶栏右侧加铃铛按钮（lucide Bell 图标 + 未读红色 badge 数字）
- 下拉面板：最新 10 条通知（标题/内容/时间/未读高亮），点击「全部已读」按钮，点击单条 → 标记已读并跳转 Link（如 /leaves）
- 每 30 秒轮询 unread-count（简单 setInterval，组件卸载清理）；收到通知时铃铛 badge 更新
- 通知下拉用 daisyUI dropdown

## 四、新页面

### 1. `AttendanceCorrections.tsx`（考勤补卡，路由 /attendance/corrections 或作为考勤页 Tab）
- 员工：提交补卡 Modal（日期/类型 check_in|check_out/时间/原因），列表显示自己记录与状态 badge
- manager/admin/hr：待审批列表，approve/reject（reject 弹窗填原因）
- 权限：员工隐藏审批操作

### 2. `Announcements.tsx`（公告，路由 /announcements）
- 公告列表（置顶 badge、发布人/时间/内容全文），admin/hr 显示新增/编辑/删除
- 新增/编辑 Modal：标题、内容 textarea、置顶 checkbox
- employee/manager 只读

### 3. `PerformancePage.tsx`（绩效，路由 /performance）
- 工具栏：周期筛选、状态筛选、员工筛选（hr/admin）
- 表格：员工/部门/周期/分数/评级 badge(S绿/A蓝/B黄/C橙/D红)/状态/评语/操作
- 新增/编辑 Modal：员工、周期(YYYY-H1/H2 select)、分数 input(0-100)、评语
- 操作按钮：编辑(draft)、提交(draft)、批准(submitted, hr/admin)、删除(draft)
- employee 只显示自己（后端已过滤），隐藏操作按钮

### 4. `SalaryBatchModal`（嵌入 SalaryPage）
- 「批量生成」按钮（hr/admin）→ Modal 选月份 → 调 batchGenerate → 显示 generated/skipped 结果 toast
- 「考勤月报」入口：可在 SalaryPage 或 AttendancePage 加「月报」按钮 → 打开 Modal 显示表格（员工/出勤天数/迟到/早退/旷工/请假）或独立路由 /attendance/report

### 5. 仪表盘增强（Dashboard.tsx）
- 增加公告区：最新 3 条公告（标题+时间）→ 点击跳 /announcements

### 6. 个人中心增强（ProfilePage.tsx）
- 头像显示（avatar 字段有值则显示图片，无则显示默认占位）+ 「更换头像」按钮 → 文件选择 → uploadFile → 更新员工 avatar 字段 → 刷新显示

### 7. 用户管理增强（Users.tsx）
- 编辑 Modal 增加「负责部门」下拉（仅当角色为 hr 时显示，选项来自 /departments 树扁平化，可清空）

## 五、暗色模式
- 新建 `store/theme.ts`：zustand + persist，state {theme: 'light'|'dark'}，action toggle
- MainLayout 顶栏加切换按钮（lucide Sun/Moon 图标）：点击 toggle 并设置 `<html data-theme=...>`（daisyUI 主题机制，dark 主题 daisyUI 内置）
- 初始化：main.tsx 或 App 挂载时读取 store 设置 data-theme（默认 light）
- 全局样式无需大改，daisyUI 组件自动适配

## 六、路由与菜单（App.tsx + MainLayout.tsx）
- 新增路由：/attendance/corrections（或并入考勤页 Tab，选 Tab 方案更省）、/announcements、/performance
- 菜单新增（lucide 图标）：
  - 公告管理 → /announcements（icon: Megaphone，roles: 全部）
  - 绩效管理 → /performance（icon: Target，roles: admin/hr/manager/employee）
  - 考勤补卡 → 并入考勤管理页 Tab（roles 同考勤）
  - 考勤月报 → 并入考勤管理页 Tab（roles: admin/hr/manager）
- 菜单 roles 与后端权限一致

## 七、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测：
   - 登录页有验证码、输错验证码提示、输入正确后登录成功
   - 顶栏铃铛有未读 badge，点开看到通知，全部已读后 badge 消失
   - 补卡流程：liuyang 提交 → zhangwei 审批 → 成功提示
   - 公告：admin 发布 → liuyang 看到；liuyang 无编辑按钮
   - 绩效：admin 创建 liuyang 绩效 score=92 → 显示评级 S
   - 批量生成工资单 → toast 显示 generated 数；月报 Modal 显示数据
   - 暗色模式切换 → 页面变暗且刷新后保持
   - 个人中心上传头像 → 显示新头像
3. 停掉两个进程，端口释放

## 汇报格式
新增/修改文件清单、build 结果、各功能实测结果、暗色模式持久化验证、任何遗留问题。
