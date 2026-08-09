# 任务批次 31：前端 — 角色审批权限勾选 + 流程配置页权限联动 + 考勤员工本人视图

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 30 已完成后端：审批引擎支持 permission 池（leave:approve / attendance_correction:approve）、GET /approval-flows 返回 approval_permissions（含拥有角色）、考勤/补卡对 employee 强制本人可见。本批次实现**前端**。中文界面，daisyUI + Tailwind + lucide-react（禁止 emoji）。

## 一、角色管理页：审批权限勾选（Roles.tsx）

1. 权限分组新增「审批」组：`module === 'approval'` 的权限点（请假审批 leave:approve / 补卡审批 attendance_correction:approve）显示为独立分组卡片（或并入现有模块分组，若权限点 Module 统一为 approval 则自动归组——检查现有分组逻辑，若有 module 映射则加 approval 映射为「审批」）
2. 现有勾选/保存逻辑复用（role_permissions 全量保存）——无需新接口，仅确认权限列表已含新权限点且分组正确
3. 编辑角色时可见「请假审批」「补卡审批」复选框；保存后该角色用户即拥有审批权

## 二、审批流程配置页权限联动（ApprovalFlows.tsx）

### 1. 类型选项更新
编辑 Modal 每行「审批人类型」下拉选项改为：
- 部门经理（manager）—— 申请人直属上级
- 按权限审批（permission）—— 拥有指定审批权限的角色成员

（旧 hr/position 类型不再提供新选项；已存在的旧类型步骤仍显示但只读提示「旧类型，建议迁移」或正常显示均可——选择：正常显示名称，类型 label 用映射兼容）

### 2. 选择「按权限审批」时
- 显示**权限点下拉**（请假审批 / 补卡审批，来自 listApprovalFlows 返回的 approval_permissions 或本地常量）
- 权限点下方显示**当前拥有该权限的角色列表**（badge 组，如「人事」「经理」），数据来自 GET /approval-flows 的 `approval_permissions[].roles`
- 步骤名自动建议：选权限「请假审批」→ 步骤名自动填「请假审批」；「补卡审批」→「补卡审批」

### 3. 卡片步骤展示
- 每步显示审批人类型 badge：manager=「部门经理」badge-info；permission=「按权限·{权限名}」badge-primary；旧类型兼容显示（hr=「HR终审」badge-primary、position=「指定岗位」badge-warning）
- 步骤行下方小字显示「{角色名} 可审批」（permission 类型，如「人事、经理 可审批」）

### 4. 保存
PUT body 中 permission 步骤带 `approver_permission` 字段（如 "leave:approve"），manager 步骤为空。校验：permission 类型必须已选权限点。

## 三、考勤页员工本人视图（AttendancePage.tsx + 补卡 Tab）

批次 30 后端已强制 employee 仅本人——前端确认：
1. employee 角色的考勤页**不再显示部门筛选**（或禁用），列表只显示本人记录（后端已过滤，前端无需改数据逻辑）
2. 检查 AttendancePage 是否有「部门」「员工」筛选下拉：employee 角色隐藏这两个下拉（manager/hr/admin 保留）
3. 补卡申请 Tab：employee 只显示自己的申请（后端已过滤）
4. 「打卡上班/下班」按钮保持可见（employee 有 attendance:write 即可打卡）

## 四、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（JWT 注入法）：
   - admin 角色管理 → 编辑 hr 角色 → 可见「审批」分组含「请假审批」「补卡审批」勾选（默认已勾）
   - admin 审批流程页 → 请假流程编辑 → 把第 2 级改为「按权限审批·请假审批」→ 保存；卡片显示「人事、经理 可审批」（leave:approve 拥有者）
   - liuyang（employee）考勤页 → 无部门筛选下拉、列表仅本人记录；打卡正常
   - hr 考勤页 → 部门筛选保留、全量记录
3. 停掉两个进程，端口释放

## 汇报格式
新增/修改文件、build 结果、角色审批权限勾选实测、流程配置页权限联动实测、考勤员工本人视图实测。
