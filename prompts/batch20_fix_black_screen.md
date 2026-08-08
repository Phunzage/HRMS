# 任务批次 20：修复「批准请假后黑屏」bug（空列表 null → 前端崩溃）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户实测：hr 批准最后一条待审批请假后页面黑屏、卡在 /leaves 无法操作。根因已由架构师定位，按方案修复。

## 根因（已复现确认）

1. **后端**：repository 里 `var items []LeaveItem`（Go nil 切片）经 GORM `Scan` 空结果后返回 nil，JSON 序列化为 `"list": null`。实测：`GET /leaves?status=pending`（无数据）返回 `{"list":null,"total":0,...}`。
2. **前端**：`LeavePage.loadTodo` 中 `setTodo(res.list)` 把 null 存入 state → `<LeaveTable list={null}>` → `list.map()` 抛 `TypeError: Cannot read properties of null` → React 无 ErrorBoundary，**整棵树卸载** → 黑屏、所有交互失效、无法返回。
3. 触发条件：待审批列表从「有数据」变「空」（批准最后一条）时必现。任何列表页都有同样隐患。

## 修复方案

### ① 后端根治：空列表返回 [] 而非 null（所有 repository）

将以下文件中所有 `var items []Xxx`（以及 `var list []Xxx`、`var rows []Xxx`）改为 `items := make([]Xxx, 0)`（或 `make([]Xxx, 0, 预估值)`），确保 GORM Scan/Find 空结果时序列化为 `[]`：

- `internal/repository/leave_repo.go`（List / ListByEmployee 的 `var items []LeaveItem`、`GetBalanceList` 的 `var list []models.LeaveBalance`、第 220 行 `var rows []struct{...}`）
- `internal/repository/employee_repo.go`（2 处 `var items []EmployeeItem`）
- `internal/repository/attendance_repo.go`（`var items []AttendanceItem`）
- `internal/repository/attendance_correction_repo.go`（`var items []CorrectionItem` ×2、`var rows []struct`）
- `internal/repository/salary_repo.go`、`internal/repository/recruitment_repo.go`（JobItem/CandidateItem）、`internal/repository/position_repo.go`、`internal/repository/performance_repo.go`（×2）、`internal/repository/announcement_repo.go`、`internal/repository/audit_repo.go`、`internal/repository/notification_repo.go`、`internal/repository/dashboard_repo.go`（3 处 rows）、`internal/repository/user_repo.go`（若有用 `var []` 的列表）

**注意**：`grep -rn "var items \[\]\|var list \[\]\|var rows \[\]" internal/repository/` 全量排查，全部改为 make 初始化。凡是返回给 handler 组装进 PageData.List 的切片都必须非 nil。改完后：
```
cd backend && go build ./... && go vet ./...
go test ./... -count=1  # 不回归
```

### ② 前端防御：列表渲染 null 兜底

- `frontend/src/pages/LeavePage.tsx`：
  - `loadTodo` 中 `setTodo(res.list)` 改为 `setTodo(res.list ?? [])`
  - LeaveTable 组件签名处 `list.map` 前统一兜底：函数开头 `const rows = list ?? []`，用 `rows.map`（或改 props 默认值 `list: LeaveRequest[] = []`，二选一，选更简洁的）
  - 同理检查 `mine?.list ?? []`、`allData?.list ?? []` 已有 `?? []` 则保留
- 全局排查：`grep -rn "\.map(" frontend/src/pages/` 检查所有对后端返回数组的 `.map()` 是否都有 `?? []` 兜底（重点：EmployeeList、AttendancePage、SalaryPage、RecruitmentPage、PerformancePage、Users、Roles、AuditLogs、Announcements、Dashboard 的趋势/分布数组）。凡是从 `res.xxx` 直接取数组渲染的，统一加 `?? []`。

### ③ 全局 ErrorBoundary（防止未来任何崩溃变黑屏）

- 新建 `frontend/src/components/ErrorBoundary.tsx`：
  - class 组件，`componentDidCatch` 捕获错误，state.hasError 时渲染友好错误页（daisyUI：大标题「页面出错了」+ 错误信息 + 「返回首页」按钮 `<a href="/">` 或 location.reload 重试按钮 + 返回上一页按钮）
  - 底部小字「错误已记录，如反复出现请联系管理员」
- `main.tsx` 中把 `<App />` 包进 `<ErrorBoundary>`（App 外层）
- 这样即使未来再有任何渲染异常，用户看到的是可操作的错误页而不是黑屏

## 验证（必须全部通过）

1. `cd frontend && npm run build` 0 错误
2. 启动前后端：
   - 复现场景回归：构造一条 pending 请假 → hr 登录（用签名 JWT 注入 localStorage 即可，见下方）→ 待我审批 Tab → 批准 → **页面正常刷新、列表显示「暂无数据」、不黑屏、可自由点击其他菜单**
   - `curl` 验证：`GET /leaves?status=pending`（无数据）返回 `"list":[]` 而非 null；其他列表接口抽查 2-3 个（如 employees、audit-logs 空过滤）同样返回 []
   - 手动触发一个渲染错误验证 ErrorBoundary 生效（可选：临时在某个组件 throw 或跳过）
3. 停掉进程，端口释放

## JWT 注入法（供验证用，登录有图形验证码无法脚本化）
用 Python 生成 hr 的签名 token（secret 从 backend/.env 的 JWT_SECRET 读）：
```python
import hmac, hashlib, base64, json, time
secret = open('../backend/.env').read().split('JWT_SECRET=')[1].splitlines()[0]
def b64(d): return base64.urlsafe_b64encode(d).rstrip(b'=')
h = b64(json.dumps({'alg':'HS256','typ':'JWT'}).encode())
now = int(time.time())
p = b64(json.dumps({'user_id':2,'username':'hr','role_code':'hr','type':'access','exp':now+86400,'iat':now}).encode())
s = b64(hmac.new(secret.encode(), h+b'.'+p, hashlib.sha256).digest())
print((h+b'.'+p+b'.'+s).decode())
```
然后浏览器 console 注入 localStorage（key `hrms_token`，格式 `{"state":{"token":"...","refresh_token":null,"user":{"id":2,"username":"hr","full_name":"人事专员","role_code":"hr","employee_id":null}},"version":0}`），刷新页面即可。

## 汇报格式
后端修改的 repository 文件清单（每处 var→make）、前端兜底改动清单、ErrorBoundary 实现说明、复现场景回归结果、空列表接口抽查结果。
