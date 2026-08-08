# 任务批次 13：后端自动化测试（service 层核心逻辑）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。项目功能已完备。本批次为后端补充**单元/集成测试**，覆盖核心业务逻辑。测试用**本机 PostgreSQL 独立测试库**（避免污染开发库）。

## 一、测试库方案

- 测试库名：`hrms_test`
- 先建库：`export PGPASSWORD=3143; "/c/Program Files/PostgreSQL/18/bin/psql.exe" -U postgres -h localhost -c "CREATE DATABASE hrms_test;"`（已存在则跳过）
- 测试连接 DSN 从环境变量 `TEST_DB_NAME` 读取（默认 hrms_test），其余 host/user/password 与开发库一致（复用 config 的默认值）
- 每个测试文件用 `TestMain` 或 `sync.Once` 初始化：连接 + AutoMigrate 全部模型（不跑 Seed，测试内自建数据，保证隔离）
- 跑完**不删库**（表会残留，但每个测试用唯一的数据或事务回滚，互不干扰；简单起见测试内创建的记录用唯一标识，如 username 加随机后缀）

## 二、测试文件与用例

### 1. `internal/service/leave_test.go` — 请假审批流（核心）
- 准备：建部门（含 ManagerID）、建经理员工、建员工（设 DepartmentID）、建用户（role employee/manager，bcrypt）
- 用例：
  - `TestLeaveBalanceCheck`：提交超余额的请假 → 返回 ErrInsufficientBalance（或对应错误）；余额刚好够 → 通过
  - `TestLeaveApproveDeductsBalance`：提交 3 天年假 → approve → balance.UsedDays == 3
  - `TestLeaveApproveCreatesAttendance`：approve 后请假区间 2 天都有 attendance 记录且 status == "leave"（**联动验证**）；区间外无记录
  - `TestLeaveApproveExistingAttendanceUpdated`：先给某天建 normal 考勤 → 请假含该天 → approve → 该条 status 变 leave
  - `TestLeaveApproveIdempotent`：重复 approve → 返回错误（状态机拒绝）
  - `TestLeaveRejectNoDeduction`：reject → 余额未扣
- 事务回滚：每个测试用 `db.Begin()` + `defer tx.Rollback()`，service 传 tx 调用，测完自动回滚不残留

### 2. `internal/service/salary_test.go` — 薪酬
- `TestNetSalaryCalculation`：base 10000 + allowance 1500 + bonus 2000 - deduction 500 = 13000
- `TestSalaryDuplicateMonth`：同员工同月创建第二张 → 返回错误
- `TestSalaryStatusFlow`：draft 可更新；confirm 后可 pay；已 pay 再 confirm → 错误；draft 直接 pay → 错误（若实现如此）
- `TestSalaryUpdateRecalculatesNet`：改 base → net 重算

### 3. `internal/service/employee_test.go` — 员工
- `TestGenerateEmpNo`：库中最大 EMP0005 → 生成 EMP0006（可 mock 或直接查库后调用）
- `TestEmpNoConcurrentUnique`：并发 10 个创建 → 全部成功且 emp_no 不重复（用 goroutine + WaitGroup，串行提交到独立事务）

### 4. `internal/service/auth_test.go` — 认证
- `TestLoginSuccess`：正确密码 → 返回 token + user
- `TestLoginWrongPassword`：错误密码 → ErrInvalidCredentials
- `TestLoginDisabledUser`：status=0 用户 → 拒绝登录
- `TestChangePassword`：旧密码对 → 新密码可登录

### 5. `internal/service/export_test.go` — Excel 导出（可选，若批次11已完成）
- 调 WriteExcel 生成字节 → 用 excelize 重新打开（`excelize.OpenReader(bytes.NewReader(data))`）→ 验证 sheet 名、表头、行数

## 三、注意事项

- service 结构体若依赖 repository，直接构造（`service.NewXxxService(db)` 形式应已存在）；若没有构造函数，测试直接 `&service.XxxService{DB: tx}` 或按现有代码风格构造
- 错误用 errors.Is 比较（service 包内应已有 sentinel error 定义；没有的用例按实际返回值断言）
- 测试命名 `TestXxx`，一个文件内多个子用例用 `t.Run`
- 若测试需要登录 token 直接调 service（不经 HTTP），避免依赖路由
- `go test ./internal/service/ -v -count=1` 必须全绿

## 四、完成后必须执行
1. `cd backend && go test ./... -count=1` 全部通过（报告各包通过数）
2. `go build ./...`、`go vet ./...` 依旧全绿
3. 汇报每个测试文件的用例清单与通过情况

## 汇报格式
测试文件清单、每个用例名称与结果、go test 汇总输出（PASS/FAIL 统计）。
