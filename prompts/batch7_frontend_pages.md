# 任务批次 7：前端业务页面完整实现

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 6 已建前端骨架（登录/布局/占位页）。本批次**实现全部业务页面**，对接后端 API。中文界面，daisyUI + Tailwind。

## 页面清单（替换批次 6 的占位页）

### 1. Dashboard.tsx（完整实现）
- 调 `GET /dashboard/stats` 渲染：统计卡片（员工总数/在职/本月新增/本月离职/出勤率/今日请假/待审批请假/待确认工资/在招职位/候选人）
- 部门人数分布：横向条形图（div 宽度百分比）
- 近 6 个月工资总额与招聘趋势：简单柱状图（div 高度百分比，纯 CSS）

### 2. EmployeeList.tsx（员工管理）
- 顶部工具栏：搜索框（姓名/工号/手机）、部门下拉、状态下拉、新增按钮
- daisyUI table：工号/姓名/性别/部门/岗位/手机/入职日期/状态 badge/操作（编辑、删除）
- 分页：上一页/下一页 + 总数显示
- 新增/编辑：Modal 表单（姓名、性别、手机、邮箱、身份证、部门下拉、岗位下拉、入职日期、地址、备注、是否创建账号）
- 删除：confirm 提示后调 DELETE
- 权限控制：admin/hr 显示新增编辑删除按钮

### 3. DepartmentTree.tsx（部门管理）
- 树形展示（递归组件渲染 children），显示部门名/负责人/排序
- 新增子部门、编辑、删除按钮（admin/hr）
- Modal 表单：名称、上级部门（树选择或下拉）、负责人（员工下拉）、排序、备注

### 4. PositionPage.tsx（岗位管理）
- 表格：岗位名/编码/所属部门/级别/备注
- 增删改 Modal（admin/hr）

### 5. AttendancePage.tsx（考勤）
- 工具栏：月份选择（type=month，默认当月）、员工筛选、状态筛选
- 表格：日期/员工/部门/上班时间/下班时间/状态 badge（正常绿/late 黄/early_leave 橙/absent 红）
- 「打卡上班」「打卡下班」按钮（当前登录员工自助打卡，调 check-in/check-out）
- HR 补录 Modal（员工、日期、上下班时间、状态）
- 编辑/删除（admin/hr）

### 6. LeavePage.tsx（请假审批）
- Tab 切换：我的申请 / 待我审批 / 全部申请（admin/hr 可见全部）
- 我的申请：提交申请按钮（类型下拉：年假/病假/事假/婚假/产假/其他、开始日期、结束日期、原因），列表显示余额 badge；pending 可取消
- 待我审批：approve/reject 按钮（reject 弹窗填原因）
- 全部申请：状态筛选
- 顶部显示我的假期余额（各类型 remaining）

### 7. SalaryPage.tsx（薪酬）
- 工具栏：月份选择、员工筛选、状态筛选
- 表格：月份/员工/部门/基本工资/津贴/奖金/扣款/实发/状态 badge（draft 灰/confirmed 蓝/paid 绿）/操作
- 新增/编辑 Modal（admin/hr，仅 draft 可编辑）
- confirm/pay 操作按钮
- employee 角色只显示自己的（后端已限制，前端隐藏操作按钮）

### 8. RecruitmentPage.tsx（招聘）
- Tab：职位 / 候选人
- 职位：表格（标题/部门/岗位/编制/状态 badge/操作），增删改 Modal
- 候选人：表格（姓名/电话/邮箱/应聘职位/状态 badge/面试日期/操作），增删改 Modal，状态流转下拉

### 9. ProfilePage.tsx（个人中心，新增）
- 当前用户信息 + 关联员工档案展示（只读）
- 修改密码表单（旧密码/新密码/确认新密码）

## 通用要求
- 每个页面用 hooks 封装数据请求（useEffect + loading + error），loading 用 daisyUI loading spinner，错误用 alert
- 复用 axios api 模块（src/api/ 下对应文件补全函数）
- 日期 input 用 `<input type="date">`，值格式 YYYY-MM-DD
- 状态 badge 颜色映射统一放 `src/utils/status.ts`（员工状态/考勤状态/请假状态/工资状态/候选人状态）
- 下拉选项（部门/岗位/员工）从对应列表接口加载
- 权限判断：`useAuth()` 拿 user.role_code，admin/hr 显示管理按钮
- 所有交互有成功提示（daisyUI toast 或 alert，可自写简单 toast）

## 完成后必须执行
1. `cd frontend && npm run build` 构建通过，无 TS 错误
2. `npm run dev` 启动 + 后端 `go run ./cmd/server` 同时启动，用 curl 验证页面可访问（HTML 200）
3. 停掉两个进程

## 汇报格式
实现页面清单、build 结果、各页面与后端接口的对接情况说明。
