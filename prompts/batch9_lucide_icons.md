# 任务：用 lucide-react 图标库替换全部 emoji 图标

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户不喜欢用 emoji 做图标，需要把前端所有 emoji/符号图标替换为 **lucide-react** 图标组件（React 生态主流图标库，与 Tailwind/daisyUI 完美配合，tree-shaking 按需打包）。

## 第一步：安装依赖
```
cd frontend && npm install lucide-react
```

## 第二步：替换所有 emoji 图标（共 5 个文件）

### 1. `frontend/src/layouts/MainLayout.tsx`（侧边栏 9 个）
当前 `menuItems` 用字符串 emoji 做 icon（`'📊'` 等）。改为：
- 引入 `import { LayoutDashboard, Users, Building2, FolderKanban, Clock, ClipboardList, Wallet, Megaphone, UserCircle } from 'lucide-react'`
- `menuItems` 的 `icon` 字段改为存储 **React 组件**（`icon: LayoutDashboard` 等，不写 JSX、不写字符串）
- 渲染处改为 `<item.icon className="h-5 w-5" />`（注意：组件名必须大写，`item.icon` 在 JSX 中用小写属性访问组件时，用 `<item.icon />` 这种带点属性访问是合法的 JSX 写法，保持风格统一即可）

图标映射建议：
- 仪表盘 → LayoutDashboard
- 员工管理 → Users
- 部门管理 → Building2
- 岗位管理 → FolderKanban
- 考勤管理 → Clock
- 请假审批 → ClipboardList
- 薪酬管理 → Wallet
- 招聘管理 → Megaphone
- 个人中心 → UserCircle

### 2. `frontend/src/pages/Dashboard.tsx`（统计卡片 10 个）
`statsCards` 数组中的 `icon: '👥'` 等字符串全部替换为 lucide 组件，渲染处 `<card.icon className="h-6 w-6" />`（若原来是 `{card.icon}` 则改为组件渲染）。
映射建议：
- 员工总数 → Users
- 在职员工 → BadgeCheck
- 本月新增 → UserPlus
- 本月离职 → UserMinus
- 本月出勤率 → CalendarCheck
- 今日请假 → CalendarDays
- 待审批请假 → Inbox（或 ClipboardCheck）
- 待确认工资单 → FileText
- 在招职位 → Briefcase
- 候选人 → UserRound（或 Contact）

### 3. `frontend/src/pages/DepartmentTree.tsx`（文件夹图标）
`{node.children?.length ? '📂' : '📁'}` 改为 `{node.children?.length ? <FolderOpen className="h-4 w-4 text-base-content/50" /> : <Folder className="h-4 w-4 text-base-content/50" />}`，import 对应组件。

### 4. `frontend/src/components/Modal.tsx` 与 `frontend/src/components/Toast.tsx`（关闭按钮）
`✕`（U+2715 符号）替换为 `<X className="h-4 w-4" />`，import `X` 组件。按钮保持原有 hover 样式类。

## 第三步：全局排查
用 `grep -rnP "[\x{1F000}-\x{1FFFF}\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{FE0F}]" src --include="*.tsx" --include="*.ts"` 确认没有遗漏的 emoji/符号图标（注释里的箭头 → 不算，不用改）。

## 第四步：验证
1. `cd frontend && npm run build` 构建通过，无 TS 错误
2. 检查 dist 产物：`grep -c "emoji" dist/assets/*.js` 无需关心，重点是确认 lucide 图标已打包（产物中应包含图标 SVG path，体积略增属正常）
3. 确认修改的文件只有上述 5 个（+ package.json / package-lock.json）

## 汇报格式
列出每个文件修改的图标映射、npm run build 结果、grep 复查结果（确认 0 个 emoji 残留）。
