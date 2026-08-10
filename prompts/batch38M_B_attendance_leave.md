# 任务批次 38M-B：移动端适配 — 考勤/请假/审批页面 + 表格横滑

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 38M-A 已完成抽屉侧边栏。本批次适配**考勤/请假/审批**页面。中文界面，daisyUI 5 + Tailwind + lucide-react（禁 emoji）。

## 通用原则
- 所有 `<table>` 外包 `<div className="overflow-x-auto">`（或确认已有），保证手机横滑
- 筛选栏/搜索栏/按钮组：`flex flex-wrap gap-*` 允许换行，不横向溢出
- 分页控件：`flex flex-wrap` 或 `join` 自适应
- 弹窗 modal-box：`max-w-[95vw]`（若已有 max-w-lg 等则确认不超宽）
- 保持桌面（lg）布局不变

## 一、AttendancePage.tsx（考勤管理）
1. 表格加 `overflow-x-auto` 容器
2. 顶部筛选（月份/状态/部门/员工下拉 + 打卡按钮）：`flex flex-wrap gap-2`，打卡按钮在移动端靠右或换行
3. 月报 Tab（AttendanceReport 组件如独立文件则一并处理）：表格横滑
4. 打卡按钮组：移动端 `btn-block`（全宽）或保留紧凑

## 二、AttendanceCorrections.tsx（补卡申请）
1. 表格横滑容器
2. 补卡申请表单（若在页面内）：日期/原因输入全宽
3. 审批操作按钮（通过/驳回）：移动端换行

## 三、LeavePage.tsx（请假审批）
1. 表格横滑容器
2. 余额卡片区：`grid` 断点（移动 1 列 → 桌面多列）
3. 请假表单弹窗：`max-w-[95vw]`，表单元素全宽
4. 类型/日期选择：不超宽
5. Tab 切换（我的申请/待我审批等）：`tabs` 在窄屏可横向滚动（`overflow-x-auto`）

## 四、ApprovalsPage.tsx（我的审批）
1. 表格/列表横滑
2. Tab（待我审批/我处理的）：窄屏横向滚动
3. 审批弹窗（意见输入 + 通过/驳回按钮）：`max-w-[95vw]`，按钮组移动端 `btn-block` 纵向堆叠

## 五、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端 dev，浏览器 375x667 视口验证：考勤页表格可横滑、筛选换行不溢出；请假页余额卡片单列、表单弹窗可操作；审批页弹窗按钮可用
3. 桌面 1440 视口回归：布局与原来一致
4. 停掉进程，端口释放

## 汇报格式
每页面改动点、build 结果、移动端/桌面验证结果。
