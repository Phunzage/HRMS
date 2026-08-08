# 任务批次 19：修复两个 bug — ①误跳登录 ②数据库乱码数据

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户实测发现两个问题，根因已由架构师定位，请按方案修复。

## Bug 1：点击功能误跳转登录页（严重）

**根因**：`frontend/src/api/client.ts` 响应拦截器的错误分支中，`useAuthStore.getState().logout()` 和跳转 `/login` 的代码**在 if(status===401) 块之外无条件执行**——任何 HTTP 错误（400/403/404/500）都会清空登录态并跳登录页。

**触发链路**（已实测确认）：
- hr 点击「个人中心」→ 页面调 `GET /employees/me` → hr 未关联员工档案 → 404 → 误登出
- hr 点击「请假审批」→ 调 `GET /leaves/balance` → 404 → 误登出
- zhangwei 点击「员工管理」→ 页面加载岗位下拉调 `GET /positions` → manager 角色**无 position:read 权限** → 403 → 误登出

### 修复方案

**① 前端拦截器（核心修复）`frontend/src/api/client.ts`**
将错误分支重构为：
- `status === 401` 且非 refresh 端点：先尝试 refresh_token 重放（现有逻辑保留）；**刷新失败或无 refresh_token 时**才执行 logout + 跳 /login
- `status` 为 403/404/400/500 等其他错误：**不登出不跳转**，仅 `Promise.reject(new Error(msg))` 把错误抛给调用方（页面已有 catch/toast 处理）
- 注意保留 `isRefreshEndpoint` 防递归逻辑

**② 后端权限（配合修复）`backend/internal/database/seed.go`**
- manager 角色增加 `position:read`（经理查看本部门岗位、员工管理页岗位筛选需要；员工管理页的岗位下拉不应 403）
- 保持 manager **不加** `salary:read` / `recruitment:read`（工资与招聘仍是 hr/admin 专属）
- seed 有幂等自愈逻辑，重启服务即生效；验证 `GET /positions` 用 manager 登录返回 200

**③ 前端页面容错（防再次踩雷）`frontend/src/pages/EmployeeList.tsx` 等**
- 岗位下拉加载失败时静默降级：`.catch(() => setPositions([]))`（若当前是直接 await 无 catch，补上）
- 检查所有页面：凡是在 mount 时加载的**非关键下拉**（部门/岗位/员工筛选），catch 后应降级为空数组而非抛错白屏

### 验证
1. `cd frontend && npm run build` 通过
2. 启动前后端：
   - hr 登录 → 点击「个人中心」→ **不再跳登录**，页面显示「当前账号未关联员工档案」提示
   - hr 登录 → 点击「请假审批」→ 正常进入，余额区为空但不跳登录
   - zhangwei 登录 → 点击「员工管理」→ 正常进入（岗位下拉可见），**不再跳登录**
   - 未登录访问受保护页 → 仍正常跳登录（回归确认）
3. 停掉进程

## Bug 2：数据库乱码数据清理

**根因**：早期 Windows curl 终端编码问题写入的脏数据（非代码 bug）：
- `employees` 表：id=15 (EMP0009 `���Ʋ���Ա��`)、id=16 (EMP0010 `����`)、id=17 (EMP0011 `����`) —— 姓名全是乱码
- `announcements` 表：id=4 标题乱码

### 修复方案（用 psql 清理，注意外键）
1. 先检查这三名员工是否被业务表引用（attendance/leave_requests/salary_records/leave_balances/performance_reviews）：
   ```sql
   SELECT 'attendance' t, count(*) FROM attendance WHERE employee_id IN (15,16,17)
   UNION ALL SELECT 'leave_requests', count(*) FROM leave_requests WHERE employee_id IN (15,16,17)
   UNION ALL SELECT 'salary_records', count(*) FROM salary_records WHERE employee_id IN (15,16,17)
   UNION ALL SELECT 'performance_reviews', count(*) FROM performance_reviews WHERE employee_id IN (15,16,17);
   ```
2. 若有关联数据：将这些员工姓名**修正**为正常中文（如 EMP0009→测试员工A、EMP0010→测试员工B、EMP0011→测试员工C），不要删除
3. 若无关联数据：直接删除这三名员工（软删或硬删均可，选硬删 DELETE 以免脏数据残留）
4. 公告 id=4：直接 DELETE（或改为正常标题「测试公告」）
5. 清理后验证：`SELECT id, emp_no, name FROM employees ORDER BY id;` 无乱码；公告列表无乱码

### 汇报格式
- Bug1：拦截器修改 diff 要点、manager 权限变更、三个场景实测结果（hr 个人中心/hr 请假/zhangwei 员工管理均不再跳登录 + 未登录仍跳登录的回归）
- Bug2：乱码记录处理方式（删/改）、清理后验证结果
