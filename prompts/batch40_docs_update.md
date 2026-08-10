# 任务批次 40：文档全面更新 — 补全缺失、同步最新功能

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。项目已进入**使用、维护阶段**，需将全部文档更新到与当前代码/功能一致。**只改文档，不动代码**。

## 背景：项目现状（文档需反映的最新事实）

1. **权限模型定稿**：admin=系统管理（仪表盘监督+公告+用户/角色/审计+审批流程配置，**无业务权限**）；hr=全公司人事业务（含公告、审批终审）；manager=部门审批；employee=自助。权限点 23+ 个（含 leave:approve / attendance_correction:approve / approval_flow:read/write / update:read/write）
2. **可配置审批流**：通用审批流引擎（approval_flows/steps/instances/tasks 四表），admin 在「审批流程」页配置请假/补卡的级别（1~5 级）与每级审批人类型（部门经理 manager / 指定角色 role / 兼容旧 hr/position/permission）；审批人解析：部门经理递归向上查父部门；角色池；未指派任务按审批权限点匹配
3. **系统通知**：后端 GET /notifications/recent（增量检测）；Windows Electron 系统通知（notify IPC）；Android 前台 LocalNotifications + 后台 WorkManager 轮询 + TokenStore 桥接
4. **三端形态**：网页版（http://115.29.197.130）+ Windows 桌面端（Electron，loadURL 线上前端）+ Android APK（Capacitor，**远程加载模式** server.url 指向线上）；**移动端全面适配**（抽屉侧边栏、表格横滑、响应式断点）
5. **部署上线**：阿里云 ECS 115.29.197.130（Ubuntu 24.04，PostgreSQL + Go systemd + Nginx，UFW 加固）；更新机制（GET /api/v1/update/check?platform=win|android&build=N，更新包 /opt/hrms/updates/，app_versions 表）；当前版本 win 1.0.5 / android 1.0.4
6. **前端细节**：移动端适配完成（lg 断点抽屉、overflow-x-auto 表格、max-w-[95vw] Modal）；导航菜单按权限点驱动（usePerm）；导航高亮；暗色模式；角标（pending_approvals/unread_announcements/unread_notifications）
7. **数据范围**：ResolveScope 机制保留，但当前所有 hr 账号 dept_scope_id=NULL（全公司）
8. **seed 行为变更**：角色权限只在角色首次配置时初始化，**重启不还原 admin 的增删**

## 任务：更新以下文档

### 1. README.md（大改）
- 头部简介更新：覆盖三端（网页/Windows/Android）与核心功能（可配置审批流、系统通知、移动端适配）
- 功能清单：补充「可配置审批流（admin 配置级别与角色）」「系统级推送通知」「移动端全面适配」「角标提醒」「Windows/Android 客户端」「自动更新机制」；更新「站内通知」为含系统推送；「数据范围权限」注明当前全公司
- 技术栈表：加 Electron / Capacitor / WorkManager / LocalNotifications
- 目录结构：加 desktop/、frontend/android/、prompts/
- 内置账号表：admin 描述改「系统管理员（用户/角色/审计/公告/审批流程配置）」；hr 改「全公司人事业务」；zhangwei 改「部门经理（请假/补卡一级审批）」；liuyang 改「普通员工（自助）」
- API 概览：补充认证（captcha、refresh）、审批流（approvals 系列）、公告、绩效、通知（recent/badges）、审批流程配置（approval-flows）、更新（update/check）、员工头像；请假接口改为通用审批流说明
- 新增「部署与分发」章节：服务器地址、网页版/Windows/Android 下载方式、更新机制说明
- 验证与检查：测试数量更新（当前 go test 覆盖含 approval/notification 等）
- 快速开始保留本地开发部分，但注明生产走服务器

### 2. DESIGN.md（中改）
- 表清单：18 张 → **25 张**（新增 approval_flows、approval_steps、approval_instances、approval_tasks、app_versions、announcement_reads、audit_logs、notifications、attendance_corrections、announcements、performance_reviews——核对 models/ 实际文件数）
- 状态机：请假/补卡改为「通用审批流引擎」说明（多级、按配置推进）；补充工资/绩效/招聘状态机不变
- 权限矩阵：更新为定稿版（admin 无业务、hr 全公司、manager 审批、employee 自助）；权限点数更新（23+）
- 业务规则：审批人解析（递归经理/角色池/权限池）、seed 权限初始化行为、通知增量检测
- 部署结构：加 desktop/、frontend/android/、服务器部署信息
- 更新日期改为 2026-08-10

### 3. ARCHITECTURE.md（中改）
- 架构图：加三端形态（浏览器 / Electron / Capacitor WebView → 同一服务器）
- 技术栈表：加 Electron、Capacitor、WorkManager
- 核心业务规则：审批流改通用引擎描述
- 里程碑表：M8 之后补 M9~M45 汇总（可用「十轮增强」概括：P0-P2 增强、审批引擎、权限重构、可配置审批流、移动端适配、部署上线、系统通知），标注日期 2026-08-04~08-10
- 日期更新

### 4. TESTING.md（中改）
- 登录说明：**现在需要图形验证码**（GET /auth/captcha）——curl 示例更新为「先取验证码再登录」或注明开发环境可用注入方式
- 内置账号表更新（同 README）
- 新增测试路径：
  - **admin**：审批流程配置（改请假为 3 级 → 提交请假 → 按级流转）、角色权限勾选（取消 hr 审批权限 → 重启不还原）
  - **经理**：请假/补卡一级审批、系统通知接收（Windows 通知中心）
  - **员工**：移动端抽屉导航、自助功能
- 新增「三端测试」小节：网页版 / Windows 桌面端 / Android 端各自的验证点
- 新增「系统通知测试」：提交请假 → 上级 30s 内收到系统通知
- 验证码说明：测试时如何绕过（批次 20 附录的 JWT 注入法简述）

### 5. PROGRESS.md（核对）
- 确认 M1-M45 齐全、章节顺序合理；如有章节错位（如「二轮增强」排在十二轮之后）调整为时间顺序
- 补充「最终状态」段：项目进入使用维护阶段，版本 win 1.0.5/android 1.0.4，遗留事项清单

### 6. CLAUDE.md（小改）
- 目录结构加 desktop/、frontend/android/；运行方式注明生产部署在服务器

## 完成后必须执行
1. 各文档改完通读一遍，确认无残留过时内容（如「13 张表」「固定两级审批」「admin 全部权限」等旧描述）
2. `grep -rn "13 张表\|固定两级\|admin.*全部权限" *.md` 确认无过时描述（README 的 admin 行除外——需按新权限改）
3. 不提交 git（等架构师审查后统一提交）

## 汇报格式
每份文档的改动要点、更新后行数、确认无过时描述的结果、发现的代码-文档不一致点。
