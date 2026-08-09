# 任务批次 36：Electron Windows 桌面客户端 + 随机更新拉取

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 35 已完成服务器部署与更新检查 API（GET /api/v1/update/check?platform=win&version=&build=）。本批次构建 **Windows 桌面客户端**（Electron）。中文界面。

## 一、项目结构

新建 `desktop/` 目录（与 backend/frontend 平级）：
```
desktop/
  package.json
  electron/
    main.js          # 主进程
    preload.js       # 预加载（暴露更新信息给渲染进程，contextBridge）
    updater.js       # 更新检查与下载逻辑（随机时间拉取）
  src/               # 渲染进程 = 直接引用 frontend 构建产物
  build/             # 图标等打包资源
  resources/
```

## 二、Electron 配置

### package.json 关键字段
```json
{
  "name": "hrms-desktop",
  "productName": "HRMS人力资源管理系统",
  "version": "1.0.0",
  "main": "electron/main.js",
  "scripts": {
    "dev": "electron .",
    "dist": "electron-builder --win"
  },
  "devDependencies": {
    "electron": "^33",
    "electron-builder": "^25"
  }
}
```

### main.js 要点
1. `BrowserWindow` 创建：宽 1400 高 900，`webPreferences: { preload, contextIsolation: true, nodeIntegration: false }`
2. 加载：生产模式 `loadFile(frontend/dist/index.html)`；开发模式加载 `http://localhost:5173`（env 判断）
3. 菜单：中文菜单（文件/视图/帮助），去掉默认英文菜单
4. `app.setAppUserModelId('com.hrms.desktop')`
5. 单实例锁（`app.requestSingleInstanceLock`），二次启动聚焦已有窗口
6. 窗口关闭行为：主窗口关闭即退出（`app.quit()`）

### preload.js（contextBridge）
```js
contextBridge.exposeInMainWorld('hrmsUpdater', {
  getUpdateInfo: () => ipcRenderer.invoke('update:get-info'),
  checkNow: () => ipcRenderer.invoke('update:check-now'),
  downloadUpdate: () => ipcRenderer.invoke('update:download'),
  onUpdateStatus: (cb) => ipcRenderer.on('update:status', (_e, data) => cb(data)),
})
```

## 三、更新机制（核心：随机时间拉取）

### updater.js 设计
1. **配置**：`API_BASE = 'http://115.29.197.130'`；当前版本从 package.json 读
2. **首次检查**：应用启动后延迟 5 秒执行首次检查
3. **随机间隔轮询**：每次检查完成后，生成 **30~60 分钟随机数**（`30000 + Math.floor(Math.random() * 1800000)` 毫秒 = 30~60 分钟），用 setTimeout 排下次；**下次时间也随机化**（避免所有客户端同时拉取——这正是「随机时间拉取更新机制」）
4. **检查逻辑**：`fetch(${API_BASE}/api/v1/update/check?platform=win&version=${ver}&build=${build})` → `has_update` 时：
   - 发 `update:status` 事件给渲染进程（"发现新版本 vX.Y.Z"）
   - 弹系统通知 + 托盘气泡（可选）
5. **下载安装**（用户确认后）：
   - 方案：下载 exe 到 `app.getPath('downloads')` 或临时目录 → `shell.openPath()` 让用户手动安装（简单可靠）；或集成 electron-updater（需 NSIS 配置，较重）
   - **选择简单方案**：下载 → 提示「安装包已下载到 xxx，请运行安装」→ shell.openPath 打开所在目录。首次安装也是这个 exe（完整安装包，非增量）
6. **状态事件**：idle / checking / update-available / downloading / downloaded / error，渲染进程可展示

### 渲染进程提示
- frontend 是 SPA，在 `frontend/index.html` 加一小段脚本（或单独 overlay 组件）监听 `hrmsUpdater.onUpdateStatus`，右下角显示更新提示条（「发现新版本 v1.1.0，点击下载」）
- 最小侵入：在 `frontend/index.html` 的 body 末尾加 `<script>` 内联（仅桌面端可用，浏览器环境自动忽略 `window.hrmsUpdater` 不存在的情况）

## 四、前端 API 地址配置

frontend 的 API 基址需要支持生产环境：`frontend/src/api/client.ts` 中 axios baseURL：
- 现状：开发用 vite proxy（/api → localhost:8080）
- **桌面端**：打包后没有 proxy → 需要 baseURL 指向 `http://115.29.197.130`
- 方案：`import.meta.env.VITE_API_BASE || ''`（构建时传 `VITE_API_BASE=http://115.29.197.130`）；生产构建 npm script：`"build:prod": "vite build --mode production"`，在 `.env.production` 设 `VITE_API_BASE=http://115.29.197.130`
- **注意**：vite 生产构建时若 baseURL 是绝对地址，则请求直接发往服务器；浏览器跨域由 Nginx CORS 或同源解决——桌面端 Electron 无跨域问题（file:// 加载），直接 fetch 绝对地址即可；浏览器访问 115.29.197.130 时同源也没问题

## 五、打包

1. `cd desktop && npm install electron electron-builder --save-dev`
2. 构建前端生产产物：`cd frontend && npm run build`（含 VITE_API_BASE）
3. 把 frontend/dist 拷入 desktop 目录（或 electron-builder extraResources 指向外部路径——**选择拷入** desktop/build-web/ 简单直接）
4. 图标：生成 256x256 png/ico（可临时用占位，后续美化）
5. `npx electron-builder --win nsis` 产出 `dist/hrms-desktop Setup 1.0.0.exe`
6. **上传到服务器**：`scp` 到 `/opt/hrms/updates/hrms-win-1.0.0.exe`，并用批次 35 的更新管理接口登记版本（若接口未做管理端，直接 psql 插入 app_versions 表：platform=win, version=1.0.0, build=1, url=http://115.29.197.130/updates/hrms-win-1.0.0.exe）

## 六、验证
1. 本机 `npm run dist` 成功产出 exe
2. 手动运行 exe：窗口打开、加载前端、能登录（http://115.29.197.130 后端）、左侧菜单正常
3. 更新链路：把登记版本 build 改为 1（当前 1.0.0/build=1）→ 改本机 package.json version 1.0.0 build 0 模拟旧版 → 启动后 5 秒检查 → 检测到更新 → 下载提示正常
4. 随机间隔：确认 setTimeout 间隔在 30-60 分钟区间（打印日志验证）

## 汇报格式
目录结构、Electron 配置要点、更新机制实现细节（随机间隔代码）、打包产物路径与大小、上传登记结果、本机运行验证、更新链路实测。
