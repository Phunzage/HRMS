# 任务批次 38M-A：移动端适配 — 核心骨架（抽屉侧边栏 + 顶栏汉堡 + 登录页）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。Android 端是 Capacitor WebView 包装的前端，当前**无移动端适配**（固定 w-60 侧边栏在手机上看桌面布局）。本批次做**核心骨架**。中文界面，daisyUI 5 + Tailwind + lucide-react（禁 emoji）。

## 一、MainLayout.tsx 侧边栏改响应式抽屉（核心）

### 现状
- `aside` 固定 `w-60 shrink-0`（约 208-210 行），始终显示
- 顶栏无汉堡按钮

### 改造要求
1. **状态**：`const [sidebarOpen, setSidebarOpen] = useState(false)`
2. **桌面（lg 及以上）**：保持现有固定 `w-60` 侧边栏常驻（`hidden lg:flex` 或等价——侧栏在 lg 显示）
3. **移动端（lg 以下）**：
   - 侧边栏变为**抽屉**：`fixed inset-y-0 left-0 z-40 w-60 transform transition-transform`，开=translate-x-0、关=`-translate-x-full`；外层包 `<div className={...lg:hidden...}>`
   - **遮罩层**：抽屉打开时显示半透明遮罩 `<div className="fixed inset-0 z-30 bg-black/50" onClick={() => setSidebarOpen(false)} />`（lg 隐藏）
   - 选中菜单项后自动关闭抽屉（NavLink onClick={() => setSidebarOpen(false)}）
   - 用 `lg:hidden` / `lg:flex` 控制两套侧栏（移动抽屉 + 桌面常驻），保证桌面体验不变
4. **顶栏汉堡按钮**：顶栏左侧加 `button`（lucide `Menu` 图标，`lg:hidden` 才显示）→ `setSidebarOpen(true)`
5. **顶栏其余**：标题、通知铃铛、暗色切换、用户信息保持不变
6. **通知铃铛下拉**（约 273 行 `w-80`）：改 `w-80 max-w-[90vw]` 防窄屏溢出

### 抽屉内容
- 抽屉内复用现有侧边栏结构（logo + 菜单 + 系统管理分组 + 角标逻辑不变），只是容器从 `aside` 换到抽屉
- 建议：把现有 aside 内容抽成组件/变量 `SidebarContent`（含 logo、menuItems 渲染、systemMenuItems 渲染、角标），桌面 aside 和移动抽屉都引用它——避免两处重复代码
- 注意 visibleMenu / visibleSystem / badgeMap 逻辑保持不变

## 二、登录页适配（src/pages/Login.tsx 或类似）

1. 表单容器宽度：桌面保持（如 max-w-md），移动端 `max-w-[92vw]`（或 w-full max-w-md 已够则确认）
2. 验证码图片：`max-w-full h-auto` 防溢出
3. 卡片/容器：`mx-4` 或自适应边距，竖屏不溢出
4. 输入框/按钮：全宽（w-full）已是常态则确认

## 三、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动 `npm run dev`，用浏览器工具（375x667 视口）验证：
   - 移动端：侧边栏默认隐藏 → 点汉堡 → 抽屉滑出 + 遮罩出现 → 点菜单项 → 抽屉关闭并导航 → 通知铃铛下拉不溢出
   - 桌面（1440 视口）：侧边栏常驻、无汉堡按钮、布局与原来一致
3. 停掉进程，端口释放

## 汇报格式
MainLayout 改造说明（结构/类名）、登录页改动、build 结果、移动端/桌面端验证结果。
