# 任务批次 39A：系统级推送通知 — 后端辅助接口 + Windows Electron 系统通知

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户需求：审批/请假等业务通知要像微信 QQ 一样**弹出系统通知**（Windows 通知中心），而不是仅应用内可见。本批次做**后端辅助接口 + Windows 端**。Android 端下一批次。

## 一、后端：通知增量查询接口（小改）

现状：前端已有 `GET /api/v1/badges`（含 unread_notifications 数量）。为支持增量检测（客户端记住上次数量，仅当**增加**时弹通知），后端补充：

### `GET /api/v1/notifications/recent?limit=5`（JWT，登录即可）
返回最近 N 条未读通知（id、type、title、content、created_at），前端对比**最新一条 id**：
- 上次已知 latest_id 与新返回的 latest id 不同 → 有新通知 → 弹系统通知（显示最新一条 title）
- 避免每次轮询重复弹

实现：`internal/handler/notification.go` + service/repository 加 `RecentUnread(userID, limit)`（未读 + 按 id desc 排序，取前 N）。router 注册 `GET /notifications/recent`。

### 验证
- hr 登录 GET /notifications/recent → 返回未读列表，id 递减
- 无未读 → 空数组（不是 null）

## 二、Windows 系统通知（Electron 主进程）

### 1. 主进程（desktop/electron/main.js 或新增 notify.js）
- 复用已 import 的 `Notification`（updater.js 已在用，main.js 需补 import 或用 require）
- 新增 IPC handler：
```js
ipcMain.on('notify', (_e, { title, body }) => {
  if (Notification.isSupported()) {
    new Notification({ title, body, silent: false }).show()
  }
})
```
- 注意 `app.setAppUserModelId('com.hrms.desktop')` 已设置（Windows 通知必需，确认存在）

### 2. 预加载（desktop/electron/preload.js）
contextBridge 暴露：
```js
notify: (payload) => ipcRenderer.send('notify', payload)
```

### 3. 渲染进程（frontend/src/layouts/MainLayout.tsx 或独立 hook）
- 新建 `frontend/src/hooks/useSystemNotify.ts`：
  - localStorage 持久化 `hrms_last_notify_id`（每用户维度可加 username 后缀避免串号）
  - 现有 30s `fetchBadges()` 轮询处（约 177-184 行）增强：轮询时额外调 `GET /notifications/recent?limit=1`（或合并到 badges 请求后并行 fetch）
  - 逻辑：
    ```ts
    const recent = await fetchRecentNotifications(1)
    const latestId = recent[0]?.id
    const lastId = Number(localStorage.getItem('hrms_last_notify_id') || 0)
    if (latestId && latestId !== lastId) {
      localStorage.setItem('hrms_last_notify_id', String(latestId))
      // 仅在桌面端弹（window.hrmsUpdater 存在 = Electron 环境）
      if (window.hrmsUpdater?.notify) {
        window.hrmsUpdater.notify({ title: 'HRMS 新通知', body: recent[0].title })
      }
    }
    ```
  - **防重复**：登录时重置 last_id 为当前最新（或首次轮询只记录不弹）
  - 说明：Electron 环境下才有 window.hrmsUpdater；纯浏览器（网页版）自动跳过（浏览器不能随便弹系统通知，符合预期——网页版保持应用内铃铛）
- `frontend/src/api/notification.ts`：加 `fetchRecentNotifications(limit)`（GET /notifications/recent）

### 4. 测试通知内容
- 请假待审批：「新的请假申请待审批」
- 补卡待审批：「新的补卡申请待审批」
- 审批结果：「请假审批通过/驳回」
- 公告：「{标题}」等（title 即通知 title，body 用 content 摘要）

## 三、完成后必须执行
1. `cd backend && go build ./...`、`go test ./...`（新增 RecentUnread 测试）
2. `cd frontend && npm run build` 0 错误 0 警告
3. 桌面端本地验证（Electron 开发模式连本地或直接测线上）：
   - 方法：启动 dev，注入用户 A token 提交请假 → 用户 B（上级）桌面端轮询触发 → Windows 通知中心出现「新的请假申请待审批」系统通知
   - 或 headless 模拟：断言 `ipcMain.on('notify')` 被触发 + `new Notification` 调用（可用 electron 的 Notification 打桩）
4. 网页版回归：浏览器打开时无系统通知弹窗（仅应用内铃铛），代码路径正确
5. 停掉进程，端口释放

## 汇报格式
后端 recent 接口、Electron notify IPC、useSystemNotify hook 增量逻辑、Windows 通知实测、网页版跳过确认、build/测试结果。
