# HRMS 构建进度报告

> 指挥官：Hermes Agent（架构设计 + 提示词编写 + 代码审查）
> 实现：Claude Code (deepseek-flash, 通过 claude CLI)
> 项目路径：C:\Users\Phunzage\Documents\Code\HRMS
> 开始时间：2026-08-04 14:17

---

## 里程碑状态

| # | 里程碑 | 提示词 | 状态 | 完成时间 |
|---|--------|--------|------|----------|
| M1 | 后端脚手架 | prompts/batch1_backend_scaffold.md | ✅ 完成 | 14:20 |
| M2 | 模型+RBAC+认证 | prompts/batch2_models_rbac.md | ✅ 完成 | 14:26 |
| M3 | 员工/部门/岗位 API | prompts/batch3_org_employee_api.md | ✅ 完成 | 14:47 |
| M4 | 考勤+请假审批流 | prompts/batch4_attendance_leave.md | ✅ 完成 | 15:03 |
| M5 | 薪酬+招聘+统计 | prompts/batch5_salary_recruitment.md | ✅ 完成 | 15:11 |
| M6 | 前端脚手架 | prompts/batch6_frontend_scaffold.md | ✅ 完成 | 15:20 |
| M7 | 前端业务页面 | prompts/batch7_frontend_pages.md | ✅ 完成 | 15:05 |
| M8 | 联调+README+提交 | prompts/batch8_integration_finalize.md | ✅ 完成 | 15:37 |
| M9 | 用户/角色管理+审计日志 | prompts/batch10_user_role_audit.md | ✅ 完成 | 16:47 |
| M10 | 请假考勤联动+Excel导出 | prompts/batch11_leave_attendance_export.md | ✅ 完成 | 16:55 |
| M11 | 前端管理页面(用户/角色/审计) | prompts/batch12_frontend_admin_pages.md | ✅ 完成 | 17:05 |
| M12 | 自动化测试17例 | prompts/batch13_backend_tests.md | ✅ 完成 | 17:20 |
| M13 | 登录安全加固+文件上传 | prompts/batch14_security_upload.md | ✅ 完成 | 18:05 |
| M14 | 站内通知+补卡审批 | prompts/batch15_notification_correction.md | ✅ 完成 | 18:10 |
| M15 | 薪酬批量生成+考勤月报+数据范围 | prompts/batch16_batch_month_scope.md | ✅ 完成 | 18:12 |
| M16 | 公告管理+绩效管理 | prompts/batch17_announcement_performance.md | ✅ 完成 | 18:16 |
| M17 | 前端新功能(通知/补卡/公告/绩效/月报/暗色) | prompts/batch18_frontend_new_features.md | ✅ 完成 | 18:25 |
| M18 | 修复误跳登录+乱码数据 | prompts/batch19_fix_bugs.md | ✅ 完成 | 19:35 |
| M19 | 修复审批后黑屏(空列表null) | prompts/batch20_fix_black_screen.md | ✅ 完成 | 19:56 |
| M20 | 仪表盘刷新+通用审批流引擎 | prompts/batch21_approval_engine.md | ✅ 完成 | 21:20 |
| M21 | 补卡迁移+admin/hr职责分离 | prompts/batch22_correction_rbac.md | ✅ 完成 | 21:40 |
| M22 | 统一待办中心+审批进度时间线 | prompts/batch23_approval_frontend.md | ✅ 完成 | 21:55 |
| M23 | 仪表盘数据范围+HR池通知+角标接口 | prompts/batch24_dashboard_scope_notify_badges.md | ✅ 完成 | 23:05 |
| M24 | 前端导航栏角标+公告已读联动 | prompts/batch25_badges_frontend.md | ✅ 完成 | 23:12 |
| M25 | 权限重构(admin专注系统管理+hr全公司) | prompts/batch26_rbac_rework.md | ✅ 完成 | 次日 11:30 |
| M26 | 前端权限联动(admin菜单收敛+公告admin) | prompts/batch27_frontend_rbac.md | ✅ 完成 | 次日 11:45 |
| M27 | 可配置审批流程(多级+岗位池) | prompts/batch28_approval_config.md | ✅ 完成 | 次日 16:10 |
| M28 | 审批流程配置页(admin前端) | prompts/batch29_approval_flow_page.md | ✅ 完成 | 次日 16:25 |
| M29 | 审批改权限制+考勤员工仅本人(后端) | prompts/batch30_approval_permission.md | ✅ 完成 | 次日 17:35 |
| M30 | 角色审批权限勾选+流程页联动(前端) | prompts/batch31_permission_frontend.md | ✅ 完成 | 次日 17:50 |
| M31 | 审批链修复(递归经理+角色池) | prompts/batch32_approval_fix.md | ✅ 完成 | 次日 19:20 |
| M32 | UI权限与权限点同步(usePerm) | prompts/batch33_perm_frontend.md | ✅ 完成 | 次日 19:35 |
| M33 | VP菜单权限点驱动+审批提示动态化 | prompts/batch34_vp_menu_dynamic_toast.md | ✅ 完成 | 次日 20:30 |
| M34 | 服务器部署+更新检查API | prompts/batch35_server_deploy.md | ✅ 完成 | 次日 21:00 |
| M35 | Electron Windows客户端+随机更新 | prompts/batch36_electron_desktop.md | ✅ 完成 | 次日 22:00 |
| M36 | Capacitor Android APK+随机更新 | prompts/batch37_android_apk.md | ✅ 完成 | 次日 22:30 |
| M37 | 客户端白屏/验证码修复+重打包1.0.1 | prompts/batch38_rebuild_fix.md | ✅ 完成 | 次日 22:45 |
| M38 | 移动端适配-批次A 抽屉侧边栏+登录页 | prompts/batch38M_A_sidebar_login.md | ✅ 完成 | 8-10 09:30 |
| M39 | 移动端适配-批次B 考勤/请假/审批 | prompts/batch38M_B_attendance_leave.md | ✅ 完成 | 8-10 09:55 |
| M40 | 移动端适配-批次C 仪表盘/薪酬/绩效/招聘 | prompts/batch38M_C_dashboard_salary.md | ✅ 完成 | 8-10 10:10 |
| M41 | 移动端适配-批次D 9个管理页 | prompts/batch38M_D_admin_pages.md | ✅ 完成 | 8-10 10:35 |
| M42 | v1.0.2发布+Android改远程加载v1.0.3 | prompts/batch38M_E_release_102.md | ✅ 完成 | 8-10 11:00 |
| M43 | 系统通知-后端recent+Windows端 | prompts/batch39A_system_notify_win.md | ✅ 完成 | 8-10 11:50 |
| M44 | 系统通知-Android端+后台轮询v1.0.4 | prompts/batch39B_android_notify.md | ✅ 完成 | 8-10 12:20 |
| M45 | 修复权限重启重置+Win重打包1.0.5 | prompts/batch39C_fix_seed_rebuild_win.md | ✅ 完成 | 8-10 13:15 |

## 已确认的环境

- Go 1.25.7 / Node v24.13.0 / PostgreSQL 18.3（服务运行中）
- 数据库 `hrms` 已创建（postgres/3143@localhost:5432）
- Claude Code 2.1.221，模型 deepseek-v4-flash（haiku 别名）
- git 仓库已初始化，首个 commit：docs 架构设计

## 架构决策摘要

- 前后端分离单体：React SPA + Gin REST API + PostgreSQL
- 分层：handler → service → repository → GORM
- RBAC：admin/hr/manager/employee 四角色 + 权限点控制
- JWT 认证，统一响应 {code, message, data}，分页规范
- 模块：组织架构 / 员工 / 考勤 / 请假审批流 / 薪酬 / 招聘 / 统计仪表盘

## 审查记录

- **批次1** ✅ 通过：分层清晰（handler→service→repository）、幂等 seed、统一响应规范、bcrypt 存密。
- **批次2** ✅ 通过：JWT 限定 HS256 算法、RBAC 权限点查询正确、13 张表 AutoMigrate、4 账号+部门+岗位+员工 seed 完整。
- **批次3** ✅ 通过：部门树/岗位/员工 CRUD 全接口验证 200；emp_no 用 pg_advisory_xact_lock 防并发重复；employee 角色移除了 employee:read（仅 /me 自助）；修复岗位 code 软删除唯一索引 bug；权限控制（401/403）验证通过。
- **批次4** ✅ 通过：考勤打卡（late/early_leave 自动判定）+ 请假审批流全链路验证；审批人自动匹配部门经理；approve 事务内原子扣减余额；余额不足 400 中文提示；越权审批/补录 403；无审批人自动 approved 分支验证通过。
- **批次5** ✅ 通过：薪酬净额自动计算 + draft→confirmed→paid 状态机 + 同月重复拦截；招聘职位/候选人状态流转；仪表盘 10 项指标 + 部门分布 + 6 个月趋势（缺失月份补 0）；权限矩阵（employee 仅看自己/403）全部验证通过。
- **批次6** ✅ 通过：Vite+React18+TS+Tailwind4+daisyUI 脚手架；axios 拦截器（token 注入/code 解包/401 跳登录）；zustand persist；构建 0 错误；vite proxy 联调后端 login/health 正常。
- **批次7** ✅ 通过：全部业务页面（登录/仪表盘/员工/部门/岗位/考勤/请假/薪酬/招聘/个人中心）完成，tsc + vite build 0 错误 0 警告。
- **批次8** ✅ 通过：端到端联调全链路验证（登录→仪表盘→各模块列表→自助接口→请假审批流→考勤打卡→薪酬状态机→招聘→CRUD→RBAC 越权），修复 1 处问题（`GET /salaries/mine` 未关联员工时状态码 400 与其他模块 404 不一致，统一为 404）；`gofmt -l`、`go vet`、`npm run build` 全绿；编写根目录 README.md；git 提交。
- **批次9** ✅ 通过：lucide-react 替换全部 emoji 图标（侧边栏/仪表盘/部门树/Modal/Toast 共 22 处），浏览器 DOM 实测 19+15 个 lucide SVG 渲染、零 emoji 残留。
- **批次10** ✅ 通过：用户管理（CRUD/重置密码/防自删自禁）、角色管理（权限勾选/全量替换/防删 admin 与占用角色）、审计日志（15 个权限点、登录成功/失败均留痕、多条件过滤）；权限矩阵 admin/hr/employee 全验证。
- **批次11** ✅ 通过：请假审批通过后自动联动考勤（已打卡记录保留打卡时间、状态变 leave；无记录自动新建），同事务原子完成；Excel 导出（员工花名册/工资表，excelize，过滤+权限控制+附件下载）。
- **批次12** ✅ 通过：前端用户管理/角色管理/审计日志三页面 + 「系统管理」菜单分组；菜单按角色可见（admin 全见/hr 无角色管理/employee 全隐藏）；构建 0 错误。
- **批次13** ✅ 通过：自动化测试 17/17 全绿（请假审批流/余额/考勤联动/薪酬状态机/工号并发唯一/认证/Excel），独立复跑确认；顺带修复安全缺口——禁用账号（status=0）拒绝登录。
- **批次14** ✅ 通过：登录安全加固（图形验证码 base64Captcha、连续 5 次失败锁定 15 分钟、access/refresh 双 token 滚动续期、JWT secret 随机化 + 默认值告警）；文件上传（≤5MB 图片、UUID 命名、/uploads 静态服务、员工头像）；测试增至 23 例全绿。
- **批次15** ✅ 通过：站内通知系统（请假提交→通知审批人、审批/驳回→通知申请人；未读数/已读/全部已读）；补卡申请审批流（提交→经理审批→事务内更新考勤打卡时间并重算状态）；修复 PG date 列 UTC 时区偏移 bug。
- **批次16** ✅ 通过（Claude 达 max-turns 后由架构师补完验证）：薪酬批量生成（复制最近工资单金额、返回 generated/skipped）；考勤月报（按人汇总出勤/迟到/早退/旷工/请假）；数据范围权限（hr 绑定负责部门后列表仅见该部门及子孙部门，实测 hr 见 6 人/admin 见 8 人）。
- **批次17** ✅ 通过：公告管理（置顶/搜索/发布/草稿）；绩效管理（评分→S/A/B/C/D 自动映射、draft→submitted→approved 状态流、唯一约束、经理仅本部门）；权限矩阵全验证。
- **批次18** ✅ 通过：前端全部新功能（登录验证码、顶栏通知铃铛+30s 轮询、补卡 Tab、公告页、绩效页、批量生成按钮、考勤月报、个人中心头像上传、暗色模式持久化）；浏览器实测：仪表盘新菜单/铃铛下拉/暗色切换（data-theme=dark 持久化）/公告页/绩效页评级与状态流全部正常；架构师修复「员工自助换头像 403」——新增 `PUT /employees/me/avatar` 自助接口（无需 employee:write），前端改调新接口。
- **批次19** ✅ 通过：修复「点击功能误跳登录」（根因：axios 拦截器对一切 HTTP 错误无条件登出，改为仅 401 登出）+ manager 补 position:read + 8 页面下拉容错 + 清理全库乱码数据（员工改名/公告删除，U+FFFD 扫描归零）。
- **批次20** ✅ 通过：修复「批准最后一条请假后黑屏」（根因：后端空列表返回 list:null + 前端 setTodo(null) + LeaveTable list.map 崩溃 + 无 ErrorBoundary）。三层修复：14 个 repository var→make 初始化返回 []、前端列表 `?? []` 兜底、全局 ErrorBoundary；浏览器实测批准后零错误可导航。
- **批次21** ✅ 通过：通用审批流引擎（ApprovalFlow/Step/Instance/Task 四模型 + StartApproval/ApproveTask/RejectTask/CancelApproval + 业务回调事务分发）；请假迁移为固定两级「部门经理→HR终审」；完整链路验证（提交→经理→hr→approved 联动余额/考勤/通知；驳回/取消/越权 403/边界 400 全通过）。
- **批次22** ✅ 通过：补卡迁移审批引擎；**admin/hr 职责分离**（admin=系统管理+业务只读，hr=业务读写、移除用户/角色/审计权限）；中间件移除 admin 无条件放行；前端各页按钮按角色显隐；权限矩阵实测（hr 访问 /users 403、admin 写业务 403）。
- **批次23** ✅ 通过：前端统一待办中心 /approvals（待我审批/我处理的）+ ApprovalTimeline 审批进度时间线 + 请假/补卡页行内进度展开；全链路 16 项实测 PASS；仪表盘增加手动刷新按钮+窗口聚焦自动刷新（修复 hr 删岗位后仪表盘数字不变）。

## 三轮增强完成情况

- 审批流：单级 → **通用审批流引擎**（固定两级：部门经理→HR终审），请假/补卡统一走引擎，通知与审批人一致，进度可追踪
- 权限：admin/hr 职责分离（admin=系统配置+业务只读，hr=人事业务读写）
- 修复：误跳登录、黑屏、乱码、仪表盘不刷新共 4 个体验 bug

## 四轮增强完成情况

- **仪表盘数据范围**：hr 仪表盘统计接入 ResolveScope（在招职位/员工/考勤/工资/趋势按负责部门过滤），与列表口径一致（hr 见 3 个岗位、admin 见全局 4 个）
- **HR 池通知修复**：审批引擎 HR 终审步骤原来不发通知（approver_id=nil 直接跳过）→ 改为广播所有 HR 用户，hr 账号现在能收到请假/补卡终审待办通知
- **导航栏角标**：GET /badges 汇总接口（pending_approvals / unread_announcements / unread_notifications），前端 30s+聚焦统一轮询；「我的审批」显示待办数角标、「公告管理」显示未读公告数角标、顶栏铃铛显示通知未读数；公告已读机制（单条/全部已读，列表带 is_read），进入公告页自动全部已读并清零角标

## 五轮增强完成情况（权限模型定稿）

用户澄清角色定位后重构权限体系：
- **admin = 系统管理员**（最终版）：仅仪表盘总览（监督）+ 公告管理（发维护/升级通知）+ 用户管理/角色管理/审计日志；**全部业务菜单移除**（员工/考勤/请假/薪酬/招聘/绩效/部门/岗位，连只读也无）；「我的审批」移除；HR 池终审仅 hr 角色可审（admin 审批 HR 池任务 → 403，待办列表/角标不再出现）
- **hr = 全公司人事管理**：清空 dept_scope_id（所有 hr 账号恢复全量），仪表盘/招聘/员工等恢复全公司口径（实测 hr 与 admin 口径一致：在招职位 7、部门 4 个、岗位覆盖全部部门）；hr 可新增任意部门岗位；hr 账号仅 admin 可创建（hr 无 user 权限），新 hr 账号自动继承角色全部权限
- 数据范围机制（ResolveScope）保留但当前所有 hr 全量，未来可复用

## 六轮增强完成情况（可配置审批流程）

- **需求**：每个公司职能划分不同（批假/补卡可能一级/两级/三级），admin 可配置审批级别与每级审批职位
- **模型**：ApprovalStep/ApprovalTask 增加 ApproverRef（岗位 ID）；审批人类型扩展为 manager（部门经理）/ hr（HR 池）/ position（指定岗位池）
- **引擎**：岗位池任务按「当前用户岗位==ApproverRef」匹配可见性与审批权；岗位池通知广播该岗位所有在职员工；多级步骤自动推进
- **API**：GET /api/v1/approval-flows（admin 查看）、PUT /api/v1/approval-flows/:code（全量保存步骤，1~5 级校验，position 必须带有效岗位）
- **前端**：admin「系统管理」新增「审批流程」页——两张流程卡片（请假/补卡）展示当前级别；编辑 Modal 动态增删步骤行、类型下拉（部门经理/HR终审/指定岗位）、岗位选择、步骤名自动建议
- **验证**：批次 28 后端 34/34 E2E（含三级链路、岗位池越权 403、角标、通知）；批次 29 前端 13/13（含配置持久化、hr 无权限）；架构师浏览器实测三级配置 API+页面渲染+恢复两级
- **兼容**：默认两级（经理→HR）为种子数据；配置变更仅影响新申请，进行中审批不受影响

## 七轮增强完成情况（审批权限制 + 考勤行级隔离）

用户理解权限制的重要性后提出两项改进：
- **审批从角色制改为权限制**：新增权限点 `leave:approve`（请假审批）/ `attendance_correction:approve`（补卡审批），admin 在角色管理中给任意角色勾选即拥有审批权（默认 hr 两个都勾、manager 勾请假）；审批流程配置页步骤类型新增「按权限审批」（permission 池——拥有该权限点的所有用户可审），页面同步显示各权限点的拥有角色（如 leave:approve→人事,经理）；通知/待办/角标按权限池匹配与广播；旧 hr/position 类型兼容保留
- **考勤员工仅本人可见**：attendance:read/write 权限点保留（员工可打卡），但 employee 角色的考勤/补卡列表接口**强制按本人过滤**（传 employee_id 参数也被忽略），hr/manager/admin 维持全量/部门视图
- **验证**：批次 30 后端 E2E（permission 池全流程、无权限 403、通知广播）+ 4 新测试；批次 31 前端实测（角色弹窗审批分组勾选、流程页「按权限·请假审批」+「人事、经理 可审批」、liuyang 仅本人 11 条/传参仍本人）；架构师浏览器复核全部通过
- **效果**：审批配置彻底解耦角色——「谁当第几级」由 admin 自由勾选权限决定，适应任意公司职能划分

## 八轮增强完成情况（审批链修复 + UI 权限同步）

用户深度测试发现 5 个问题，全部修复：
1. **部门经理递归解析**：张佳（算法组，直属无经理）提交病假，刘尧（研发部经理）收不到——`ResolveManagerUserID` 原来只查直属部门，改为**递归向上**（算法组→研发部→刘尧）；实测请假39 任务 approver_id=liu(20) ✅
2. **hr 取消权限仍收到审批**：manager 步骤无经理时兜底 `FindHRUser` **硬指派** hr（与权限无关）——改为返回未指派任务（approver_id=NULL），归属按 BizType 推断审批权限点（leave:approve），hr 无权限即不可见/不可审/无通知 ✅
3. **通知链路检查**：StartApproval/ApproveTask 推进按新逻辑（经理指派→仅经理；未指派→权限池广播；role 池→角色广播）；终审/驳回→申请人 leave_result；禁用账号跳过通知 ✅
4. **审批流程页改角色制**：步骤类型改为 manager（部门经理）+ role（**指定角色**，如人事专员/副总经理），ApprovalStep/Task 增加 approver_role_id；优先级=步骤顺序；默认请假=经理→角色(人事) ✅
5. **UI 与权限点同步**：/auth/me 返回 permissions 列表 + 前端 usePerm hook（has/hasAny），全部页面 canManage 从 `role==='hr'` 硬编码改为权限点判断（Announcements 用 announcement:write 等 11 处）；manager 勾 announcement:write 后「发布公告」按钮即出现 ✅

## 九轮增强完成情况（VP 菜单权限化 + 审批提示动态化）

用户实测两个问题，全部修复：
1. **VP（副总经理）登录侧边栏空白**：菜单项 roles 数组全小写，自定义角色 role_code（如 VP 大写）大小写不匹配被全过滤。改为**权限点优先 + roles 兜底**：每个业务菜单项补充 perms（有权限点按 hasAny 判定、perms=[] 登录可见、无 perms 按 roles 大小写不敏感兜底）；系统管理保持 admin 专属。VP（14 权限）实测侧边栏 11 项：员工/部门/岗位/考勤/请假/我的审批/薪酬/公告/绩效/招聘/个人中心，无系统管理、无仪表盘（VP 无 dashboard:read，默认按权限点不给）✅
2. **审批通过 toast 硬编码「等待 HR 终审」**：审批流程可配置（本库请假二级实为副总经理/VP 角色）。后端 `ApproveTask` 响应增加 `next_step_name`/`next_step_type`（按步骤类型计算显示名：manager→部门经理审批、role→{角色名}审批、hr→HR终审、position→岗位名、permission→权限点名，终审=空），前端 ApprovalsPage/LeavePage/AttendanceCorrections 三处据 next_step_name 动态提示。实测：刘尧过请假一级→「已通过，等待副总经理审批」，VP 过二级终审→「审批已通过」；补卡单级终审→「补卡审批已通过」（不再误报等待 HR 终审）✅
3. **顺带修复 usePerm 无限重渲染崩溃**：`useAuthStore(s => s.user?.permissions ?? [])` 在 permissions 缺失时每次返回新数组引用，触发 React 18 useSyncExternalStore「Maximum update depth exceeded」崩溃（旧 localStorage 无 permissions 字段即触发，guard 的 /auth/me 补全来不及执行）。改为模块级 `EMPTY_PERMS` 稳定空引用，实测 VP 登录渲染正常 ✅

## 十轮（部署上线）完成情况

- **服务器部署**：Ubuntu 24.04 @ 115.29.197.130，PostgreSQL（低内存调优）+ Go 后端 systemd + Nginx 反代（80 端口，SPA 回退）+ UFW 加固（仅 22/80/443）；公网 health/登录/静态页全通
- **更新检查 API**：GET /api/v1/update/check?platform=win|android&build=N → has_update + 最新版本信息；admin 管理接口 POST/GET /api/v1/update/versions；权限点 update:read/write
- **Windows 桌面端**：Electron + electron-builder → `hrms-desktop Setup 1.0.0.exe`（81.8MB）；**随机更新拉取**（启动 5s 首次检查 + 30~60min 随机间隔轮询）；下载完整安装包到下载目录引导安装；前端更新提示条（hrmsUpdater bridge）
- **Android 端**：Capacitor 包装（3.2MB APK，com.hrms.app）；原生 Updater（随机间隔 + 通知 + DownloadManager 下载 + 安装引导）；targetSdk 36 / minSdk 24
- **更新包分发**：/opt/hrms/updates/（win exe + android apk），Nginx /updates/ 静态下载，双平台 build=0 均返回 has_update:true、下载 200 OK
- **遗留**：Android 用 debug 签名（上架需正式 keystore）、默认图标、http 明文（绑域名后上 https）、gradle 需镜像下载

## 十一轮（移动端全面适配）完成情况

用户发现 Android 端（Capacitor WebView）无移动端适配，桌面布局在手机上不可用。分 4 批次完成全部 19 页适配：
- **批次 A（骨架）**：MainLayout 抽屉侧边栏（桌面 lg 常驻 / 移动 lg:hidden 抽屉 + 汉堡按钮 + 遮罩 + 选中自动关闭）；SidebarContent 抽共享组件避免重复；通知下拉移动端固定视口面板；登录页窄屏适配（min-w-0/验证码 max-w-full）
- **批次 B（考勤/请假/审批）**：Tab 栏 overflow-x-auto + w-max；余额卡片 grid 断点（移动1列→桌面4列）；弹窗表单单列化；审批弹窗按钮移动端纵向堆叠；**共享 Modal 组件改 `max-w-[min(95vw,42rem)]`** 全局覆盖
- **批次 C（仪表盘/薪酬/绩效/招聘）**：统计卡片 1→5 列断点；工具栏/按钮 flex-wrap；弹窗表单单列化；招聘 Tab 横滑；**修复顶栏窄屏溢出 6px**（隐藏标题/姓名文本）
- **批次 D（9 个管理页）**：员工/用户/岗位/角色/审计表格横滑真实可滑；部门树缩进响应式；角色权限网格移动单列；审批流程步骤行移动端堆叠；个人中心网格单列；公告 textarea 全宽（顺带修复 daisyUI textarea 不随 Field 拉伸的历史问题）
- **验证**：每批 375×667 视口实测（累计 84+ 项全过，含无横向溢出、抽屉交互、弹窗可操作）+ 桌面 1440 回归；npm run build 全绿
- **待办**：重新打包 Android APK v1.0.2（含移动端适配）与 Windows 包；LeavePage 请假事由 textarea 补 w-full（批次 D 遗留小项）

## 十二轮（系统级推送通知）完成情况

用户需求：审批/请假等业务通知要像微信 QQ 一样弹出**系统通知**，而非仅应用内可见。
- **后端**：新增 `GET /api/v1/notifications/recent?limit=N`（未读通知按 id 倒序，供客户端增量检测）；已部署服务器（旧后端 404 → 新后端 401/正常返回）
- **Windows（Electron）**：`notify.js` IPC 模块（`registerNotifyIpc`，isSupported 检查、标题/内容截断）+ preload 暴露 `notify`；前端 `useSystemNotify` hook（localStorage `hrms_last_notify_id_<username>` 增量检测、登录防重复、**网页版自动跳过**）；30s 轮询内顺带检测 → 新通知弹 Windows 通知中心；headless 打桩测试 10/10
- **Android**：前台 `@capacitor/local-notifications`（JS 轮询检测 → LocalNotifications.schedule + Android 13 动态权限）；后台 `NotifyWorker`（WorkManager 15 分钟周期 + onResume 即时检查，前台只记基线不弹避免双弹）；`TokenStore` Capacitor 插件桥接 WebView token → SharedPreferences 供原生轮询；v1.0.4 已打包上传登记
- **验证**：后端/前端 build+test 全绿；recent 接口公网 401→正常；Android 更新链路 1.0.4 可用；无真机（代码走查）
- **遗留**：Android 13+ 需用户授权通知权限；通知点击仅打开 App（未跳具体页）；WorkManager 最小 15 分钟系统可延迟；App 被强停需手动打开恢复

## 二轮增强完成情况

P0 全部完成（8/8）：用户角色管理、审计日志、请假考勤联动、Excel 导出、自动化测试、登录安全加固、审批通知、文件上传。
P1 完成（7/7）：薪酬批量生成、考勤月报、补卡申请、公告管理、数据范围权限、通知系统、工资导出。
P2 完成（2/2）：绩效管理、暗色模式。
剩余可做：通用审批流引擎、合同管理、培训管理、Docker 化、CI/CD、i18n。

---

## 最终结论

HRMS 人力资源管理系统已完整交付并通过联调验收：

- **后端**：Go + Gin + GORM 分层实现，AutoMigrate 建表 + seed 内置数据，JWT + RBAC 权限控制，7 大模块（组织人事/考勤/请假审批流/薪酬/招聘/统计仪表盘）接口全部验证通过。
- **前端**：React 18 + TypeScript + Vite + Tailwind + daisyUI，axios 统一拦截器，zustand 状态管理，全部业务页面构建通过。
- **质量**：`go build`、`go vet`、`gofmt`、`npm run build` 全绿；端到端联调覆盖登录、CRUD、审批流、状态机、RBAC 越权等场景。
- **文档**：根目录 README.md（含快速开始、内置账号、API 概览），ARCHITECTURE.md 里程碑全部完成。
