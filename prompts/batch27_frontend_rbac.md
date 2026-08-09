# 任务批次 27：前端权限联动 — admin 菜单收敛、hr 全量、部门下拉放开

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 26 已完成后端权限重构（admin=仪表盘+公告+系统管理，业务全移除；hr=全公司无数据范围；HR 池终审仅 hr）。本批次同步**前端**。中文界面，daisyUI + Tailwind + lucide-react（禁止 emoji）。

## 一、MainLayout 菜单按角色收敛

### admin 可见菜单（最终）
| 菜单 | 路由 | 图标 |
|------|------|------|
| 仪表盘 | / | LayoutDashboard |
| 公告管理 | /announcements | Megaphone |
| 用户管理 | /users | UserCog |
| 角色管理 | /roles | ShieldCheck |
| 审计日志 | /audit-logs | ScrollText |

**admin 不再看到**：员工管理、部门管理、岗位管理、考勤管理、请假审批、**我的审批**、薪酬管理、招聘管理、绩效管理、个人中心。

### hr 可见菜单
- 现有全部业务菜单 + 我的审批 + 公告管理 + 仪表盘（**不显示**：用户管理/角色管理/审计日志——批次 22 已处理，确认保留）

### manager / employee 不变

具体改动：`MainLayout.tsx` 的 menuItems roles 数组调整（如员工管理 roles 去掉 'admin'；「我的审批」roles 去掉 'admin'；个人中心 roles 是否保留 admin？admin 无员工档案，可去掉）。逐项核对，确保 admin 只看到上表 5 项。

## 二、Users.tsx「负责部门」字段处理

批次 26 已清空所有 hr 的 dept_scope_id，数据范围机制保留但当前不用。前端：
- 编辑 Modal 中「负责部门」下拉**移除**（或注释保留但默认不显示——选择：直接移除，避免困惑）
- `dept_scope_name` 列不展示（或展示为「全公司」）
- 简化：Users 页面不再涉及部门范围概念

## 三、招聘/员工等页面部门下拉放开

hr 全量后，新增岗位/员工时的部门下拉应为**全公司部门树**（当前若是按 hr 范围过滤则放开）：
- `RecruitmentPage`：新增职位部门下拉 → 全部门
- `EmployeeList`：新增员工部门下拉 → 全部门
- 检查是否有 `?department_id=` 默认限定逻辑，放开
- 若后端接口本身已全量（批次 26 清空 scope 后天然全量），前端无需改，只需确认

## 四、公告页 admin 可用

- 批次 26 已给 admin announcement:read/write → `Announcements.tsx` 的 canManage 判断：当前是 `role==='hr'`，改为 `role==='hr' || role==='admin'`（admin 要能发维护公告）
- 确认 admin 公告页正常显示「发布公告」按钮

## 五、其他角色权限相关按钮核对

- `EmployeeList`/`DepartmentTree`/`PositionPage`/`AttendancePage`/`SalaryPage`/`PerformancePage`/`RecruitmentPage` 的写按钮：当前 `canManage = role==='hr'`——admin 已无这些菜单，无需改动，但确认无 admin 残留入口
- `LeavePage`：admin 无菜单；「待我审批」Tab 的 showTodo 判断含 admin——admin 已无入口，可保留逻辑（无害）或同步去掉 admin
- `ApprovalsPage`：菜单已对 admin 隐藏；页面本身若 admin 直接访问 URL，后端已 403（批次 26）——前端无需额外处理

## 六、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（JWT 注入法，见批次 20 附录）：
   - admin 登录：侧边栏仅 5 项（仪表盘/公告/用户/角色/审计）；访问 /employees 路由 → 页面调接口 403（菜单虽隐藏但 URL 直达也应被后端拦截）；公告页可发布
   - hr 登录：业务菜单齐全 + 我的审批；新增招聘岗位部门下拉含全公司部门（含人事部）
   - manager/employee 菜单回归不变
3. 停掉两个进程，端口释放

## 汇报格式
菜单调整前后对比、Users 字段处理、部门下拉验证、admin 公告发布实测、三角色菜单快照、build 结果。
