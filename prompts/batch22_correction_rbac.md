# 任务批次 22：补卡流程迁移审批引擎 + admin/hr 职责分离

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 21 已完成通用审批流引擎并迁移请假流程（固定两级：部门经理 → HR终审）。本批次：① 将**补卡流程**迁移到审批引擎 ② 按架构师方案调整 **admin/hr 角色职责分离**。严格分层，统一响应。

## 一、补卡流程迁移（internal/service/attendance_correction.go 重构）

参照批次 21 的请假迁移模式（StartApproval / ApproveTask / RejectTask / CancelApproval + 业务回调）：

1. **提交** `POST /attendance/corrections`：
   - 校验逻辑保留（日期范围/类型/时间格式）
   - 创建 AttendanceCorrection（status 保持 pending 兼容，approver_id 保留但不再作为审批依据）
   - 事务内 `StartApproval(tx, "attendance_correction", correctionID, userID)` → 创建实例 + 经理任务 + 通知经理
2. **终审通过回调** `ApplyCorrectionApproved(tx, correctionID)`：原 approve 逻辑移入（更新对应考勤打卡时间并重算状态、通知申请人）
3. **驳回回调** `ApplyCorrectionRejected(tx, correctionID, reason)`：状态→rejected、通知申请人
4. **取消** `PUT /corrections/:id/cancel`：仅 pending → 同时 `CancelApproval`
5. 删除旧的 `PUT /corrections/:id/approve`、`/reject`（由通用 /approvals 接口替代）
6. **列表** `GET /corrections` 保留（含 status 过滤），`GET /corrections/mine` 保留

## 二、admin / hr 角色职责分离（database/seed.go 权限矩阵调整）

**目标**：
- **admin = 系统管理员**：只管系统配置（用户管理、角色管理、审计日志、仪表盘只读、全部业务**只读**——可监督不可操作）
- **hr = 业务管理员**：负责全部人事业务**读写**（员工/部门/岗位/考勤/请假/薪酬/招聘/绩效/公告），**不再有**用户/角色/审计管理权限

### 具体权限矩阵（seedRolePermissions 调整）

**admin**（中间件目前直接放行，改为按配置走；若中间件保留 admin 放行逻辑，则在此给 admin 建全量只读+系统权限行，二选一——**推荐改中间件**：admin 不再无条件放行，改为与 hr 一样查 role_permissions，保证"业务只读"生效）：
- `employee:read, department:read, position:read, attendance:read, leave:read, salary:read, recruitment:read, performance:read, announcement:read, dashboard:read`
- `user:read, user:write, role:read, role:write, audit:read`
- **不含任何 :write 业务权限**（employee:write 等均无）

**hr**：
- 全部业务读写：`employee:read/write, department:read/write, position:read/write, attendance:read/write, leave:read/write, salary:read/write, recruitment:read/write, performance:read/write, announcement:read/write, dashboard:read`
- **移除**：`user:read, user:write, role:read, audit:read`

**manager / employee 不变**（manager 已有 position:read 等；employee 自助权限不动）

### 中间件调整（middleware/auth.go）
- `RequirePermission` 移除「admin 直接放行」逻辑，改为统一查 role_permissions（与普通角色一致）
- 这样 admin 的业务只读真正生效（调用 employee:write 会 403）

### 前端菜单联动（本批次顺手改，避免 admin 点业务页全 403 的体验问题）
- `MainLayout.tsx`：admin 的菜单**隐藏**业务编辑入口？——**不改菜单结构**（admin 仍可浏览只读列表），但以下按钮按 role 隐藏：
  - EmployeeList「新增员工」按钮：仅 hr/admin 显示？→ 改为**仅 hr 显示**（admin 只读）
  - 各页面的新增/编辑/删除按钮：仅 hr 显示（admin 隐藏）——检查 EmployeeList/DepartmentTree/PositionPage/AttendancePage(补录)/SalaryPage/RecruitmentPage/PerformancePage/Announcements/LeavePage(审批 Tab 保留? admin 无 leave:write 会 403——admin 的「待我审批/全部申请」Tab 保留查看，审批按钮隐藏)
- 简单起见：前端判断 `role === 'hr'` 才显示业务写操作按钮；admin 只读浏览

## 三、数据范围逻辑适配
- 现有 `ResolveScope`：hr 按 dept_scope_id 过滤，admin 返回 nil（全量）。保持：**admin 全量只读**、hr 有范围时按范围。无需改动，确认即可。

## 四、完成后必须执行
1. `cd backend && go build ./...`、`go vet ./...`、`go test ./...` 全绿
2. 启动验证（JWT 注入法）：
   - 补卡链路：liuyang 提交补卡 → zhangwei 待办看到 → 通过 → hr 待办看到 → 通过 → 考勤更新生效
   - admin 权限：admin 登录 GET /employees 200（只读）；POST /employees → **403**；GET /users 200；PUT /roles 200
   - hr 权限：POST /employees 200（业务可写）；GET /users → **403**；GET /audit-logs → **403**
   - 审批引擎兼容：请假/补卡完整两级流程再走一遍确认无回归
3. `cd frontend && npm run build` 通过（按钮权限调整后无 TS 错误）
4. 停掉进程，端口释放

## 汇报格式
补卡迁移说明、权限矩阵调整前后对比、中间件改动、前端按钮权限改动清单、验证结果矩阵（admin/hr 各接口 200/403）。
