# 任务批次 38M-C：移动端适配 — 仪表盘/薪酬/绩效/招聘

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 A/B 已完成骨架与考勤请假。本批次适配**仪表盘/薪酬/绩效/招聘**。中文界面，daisyUI 5 + Tailwind + lucide-react（禁 emoji）。

## 通用原则
- 表格 `overflow-x-auto` 容器；筛选/按钮 `flex flex-wrap`；弹窗 `max-w-[95vw]`；桌面 lg 布局不变

## 一、Dashboard.tsx（仪表盘）
1. 统计卡片栅格：`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`（或现有断点，确保移动 1 列）
2. 图表（工资趋势/招聘趋势）：容器 `w-full`、图表组件自适应（若图表库有响应式则确认；echarts/recharts 需 `width` 用容器百分比或窗口监听——检查现有实现）
3. 部门人数分布/最新公告列表：移动端单列不溢出
4. 工具栏（标题+刷新按钮）：`flex flex-wrap justify-between` 防挤压

## 二、SalaryPage.tsx（薪酬管理）
1. 表格横滑
2. 批量生成/确认/发放按钮组：`flex flex-wrap`
3. 筛选（月份/部门/员工）：换行
4. 工资单详情弹窗：`max-w-[95vw]`

## 三、PerformancePage.tsx（绩效管理）
1. 表格横滑
2. 评级 badge/状态流：窄屏可换行
3. 评分弹窗：`max-w-[95vw]`，表单全宽
4. 筛选：换行

## 四、RecruitmentPage.tsx（招聘管理）
1. 岗位/候选人表格横滑
2. 新增岗位/候选人弹窗：`max-w-[95vw]`，表单全宽
3. 状态筛选/搜索：换行
4. 职位卡片（若有卡片视图）：移动单列

## 五、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动 dev，375x667 视口验证：仪表盘卡片单列、图表不溢出；薪酬/绩效/招聘表格横滑、弹窗可操作
3. 桌面 1440 回归
4. 停掉进程，端口释放

## 汇报格式
每页面改动点、build 结果、移动端/桌面验证结果。
