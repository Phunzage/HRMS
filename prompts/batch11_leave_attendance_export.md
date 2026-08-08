# 任务批次 11：请假考勤联动 + Excel 导出

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 10 已完成用户/角色管理与审计日志。本批次实现**请假与考勤联动**和 **Excel 导出**。严格分层，统一响应，JWT + 权限点。

## 一、请假与考勤联动

**业务规则**：请假申请审批通过后，请假区间内的考勤记录应标记为 `leave`（请假中），避免与 absent（旷工）矛盾；驳回/取消不影响（状态机保证只有 pending 才能被 reject/cancel，approved 不可逆，故只在 approve 时联动一次）。

### 实现（修改 `internal/service/leave.go` 的 Approve 方法）
1. 在现有「更新状态 + 扣减余额」事务内追加：
   - 遍历请假区间 [StartDate, EndDate] 的每一天（用 `date` 循环）：
     - 若当天已有考勤记录（employee_id + work_date 存在）→ 更新其 status 为 `leave`（保留 check_in/check_out 原值）
     - 若当天无考勤记录 → 创建一条 `Attendance{EmployeeID, WorkDate, Status: "leave"}`（CheckIn/CheckOut 为 nil）
   - 注意：若当天考勤已是 leave 则跳过；已打卡记录（check_in 非空）也统一覆盖为 leave（审批通过代表公司认可该日请假）
2. **联动须在同一事务**：任一环节失败整体回滚（状态更新、余额扣减、考勤标记）。
3. 考勤状态枚举增加 `leave`：`normal/late/early_leave/absent/leave`。考勤列表与统计中 leave 视为「正常出勤」之外的请假日（仪表盘出勤率计算按现有逻辑即可，不必改统计口径，但注释说明）。

### 验证
- liuyang 提交 2026-08-10~11 年假 2 天 → zhangwei 审批通过 → 查询 GET /attendance?month=2026-08 应有 2 条 status=leave 记录（8/10、8/11），且不影响 8/12 之后
- 若 8/10 已打卡（先打卡再审批）→ 审批后该条 status 变 leave，check_in 保留
- 审批通过后余额已扣减（沿用现逻辑）

## 二、Excel 导出（用 excelize 库）

### 安装
```
cd backend && go get github.com/xuri/excelize/v2
```

### 1. 员工花名册导出（permission: employee:read）
`GET /api/v1/employees/export`（注意：必须放在 `/:id` 路由**之前**注册，否则被吞）
- 支持与列表相同的过滤参数（keyword/department_id/status/position_id），导出**全部匹配**（不分页）
- 生成 Excel：工作表「员工花名册」，列：工号/姓名/性别/部门/岗位/手机/邮箱/入职日期/状态/备注
- 性别映射 1男/2女/3其他；状态映射 1在职/2离职/3试用；部门/岗位显示名称
- 表头加粗、列宽自适应、冻结首行（excelize 样式）
- 响应：`Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`，`Content-Disposition: attachment; filename="employees_YYYYMMDD_HHMM.xlsx"`，直接写二进制流（不走统一 JSON 响应）

### 2. 工资表导出（permission: salary:read）
`GET /api/v1/salaries/export`
- 支持 ?month=&status=&department_id= 过滤，导出全部匹配
- 列：月份/工号/姓名/部门/基本工资/津贴/奖金/扣款/实发/状态
- 状态映射 draft草稿/confirmed已确认/paid已发放；金额保留两位小数
- 同样附件下载

### 3. 通用导出辅助
新建 `internal/service/export.go`（或 util）：封装 `WriteExcel(filename, sheetName, headers []string, rows [][]interface{}) ([]byte, error)`，两个导出复用。

### 完成后必须执行
1. `cd backend && go mod tidy && go build ./...`、`go vet ./...`
2. 启动验证（admin）：
   - `curl -s -o /tmp/emp.xlsx -w "%{http_code}" "localhost:8080/api/v1/employees/export"` 返回 200
   - 用 Python 验证文件是合法 xlsx（`python -c "from openpyxl import load_workbook; wb=load_workbook('/tmp/emp.xlsx'); print(wb.sheetnames, wb.active.max_row)"`；若 openpyxl 未装则 pip install openpyxl 到 python3.13）
   - 同样验证 salaries/export
   - 验证 `GET /employees/export` 与 `GET /employees/:id` 路由不冲突（export 请求应走导出而非详情）
3. 停掉进程

## 汇报格式
新增/修改文件清单、联动验证结果（审批前打卡→审批后 status=leave）、两个导出的验证结果（HTTP 状态 + openpyxl 读到的行列数）。
