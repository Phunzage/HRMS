# 任务批次 25：前端导航栏角标 + 公告已读联动 + 通知完善

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 24 已完成后端：仪表盘数据范围、HR池通知修复、角标接口 GET /badges、公告已读（PUT /announcements/:id/read、read-all、列表 is_read）。本批次实现**前端**。中文界面，daisyUI + Tailwind + lucide-react（禁止 emoji）。

## 一、API 封装

### 1. `frontend/src/api/badge.ts` 新建
- `fetchBadges()` GET /badges → `{pending_approvals, unread_announcements, unread_notifications}`
- 类型 `Badges` 导出

### 2. `frontend/src/api/announcement.ts` 扩展
- `markAnnouncementRead(id)` PUT /announcements/:id/read
- `markAllAnnouncementsRead()` PUT /announcements/read-all
- `Announcement` 类型增加 `is_read: boolean`

## 二、导航栏角标（MainLayout.tsx 核心改造）

### 1. 角标状态与轮询
- 新建 `store/badges.ts`（zustand）：state {pending_approvals, unread_announcements, unread_notifications, lastUpdated}，action `setBadges`、`decrementAnnouncement`、`decrementApproval`、`decrementNotification`
- MainLayout 挂载时启动轮询：**每 30 秒** + **窗口聚焦时** 调 `fetchBadges()` 更新 store（与通知铃铛现有轮询合并或复用——检查现有铃铛轮询实现，统一成一个 30s 轮询同时更新 badges 和通知列表，避免重复请求）
- 侧边栏菜单项渲染角标：daisyUI `<span className="badge badge-primary badge-sm">N</span>` 显示在 label 右侧

### 2. 角标映射
| 菜单项 | 路由 | 角标来源 | 显示条件 |
|--------|------|---------|---------|
| 我的审批 | /approvals | pending_approvals | >0 显示（manager/hr/admin） |
| 公告管理 | /announcements | unread_announcements | >0 显示（全部角色） |
| （顶栏铃铛已有未读角标，保留） | — | unread_notifications | 已有 |

- 菜单项数据结构扩展：`badge?: keyof Badges`（如 `badge: 'pending_approvals'`），渲染时从 store 取值
- 仅 `>0` 时渲染 badge，否则不渲染（保持干净）

### 3. 角标联动消减（关键：用户处理完要即时减）
- 进入 /approvals 页并成功加载待办列表后 → `setBadges({...pending_approvals: 实际待办数})` 或 decrementApproval；简单做法：审批操作成功后调用 fetchBadges() 刷新
- 进入 /announcements 页后 → 调 `markAllAnnouncementsRead()` → 成功后 store 清零 unread_announcements（**注意**：只有用户实际浏览公告页才标记已读，符合直觉）
- 打开通知下拉/全部已读 → 已有逻辑刷新铃铛

## 三、公告页已读联动（Announcements.tsx）

1. 页面挂载时：加载列表 → 若列表有未读（`list.some(a => !a.is_read)`）→ 调 `markAllAnnouncementsRead()` → store 清零 → 本地把列表 is_read 全置 true（避免刷新 UI 闪烁）
2. 单条公告仍提供「标记已读」无需单独按钮——阅读即已读（页面打开即全部已读）
3. 列表渲染 is_read：未读条目加粗或 dot 标记（如标题前 `<span className="badge badge-primary badge-xs">新</span>`），已读正常样式

## 四、通知中心完善（MainLayout 铃铛 + 可选 /notifications 页）

1. 铃铛下拉现有功能保留（未读数、列表、全部已读）；确认 hr 现在能收到终审通知（批次 24 后端修复）
2. 通知列表每条显示 type 对应中文 badge（请假审批/请假结果/补卡审批/公告/系统），已有则确认；通知项点击 markRead 后 decrementNotification

## 五、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（JWT 注入法，见批次 20 附录）：
   - liuyang 提交请假 → zhangwei 登录：侧边栏「我的审批」显示角标 1 → 进入 /approvals 审批通过 → 角标消失（轮询或操作后刷新）
   - zhangwei 通过后 → hr 登录：铃铛收到「新的请假申请待审批」通知 + 「我的审批」角标 1 → 终审 → 角标消失
   - admin 发布新公告 → liuyang 登录：侧边栏「公告管理」角标 1 → 进入公告页 → 角标清零（已读）
   - 等待 30s 或切换窗口，badges 轮询正常更新
3. 停掉两个进程，端口释放

## 汇报格式
新增/修改文件清单、build 结果、三角色角标实测（zhangwei 审批角标/hr 通知+角标/liuyang 公告角标）、轮询与联动消减验证。
