package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 审批引擎业务错误
var (
	ErrApprovalFlowNotFound  = errors.New("审批流程未配置")
	ErrApprovalFlowNoStep    = errors.New("审批流程未配置审批步骤")
	ErrApprovalInstanceNotFound = errors.New("审批实例不存在")
	ErrApprovalTaskNotFound  = errors.New("审批任务不存在")
	ErrApprovalTaskNotPending = errors.New("审批任务已处理")
	ErrApprovalNotPending    = errors.New("审批实例当前状态不可操作")
	ErrApprovalNotAllowed    = errors.New("无权审批该任务")
)

// ApprovalBizCallback 审批终审回调接口：业务模块实现，handler 在审批流 final 后同一事务内调用
type ApprovalBizCallback interface {
	// ApplyApproved 终审通过回调（事务内）：更新业务状态、扣余额、联动考勤、通知等
	ApplyApproved(tx *gorm.DB, bizID uint) error
	// ApplyRejected 驳回回调（事务内）：业务状态置为 rejected、通知申请人
	ApplyRejected(tx *gorm.DB, bizID uint, reason string) error
}

// TaskItem 我的待审批/历史任务项（JOIN 业务表带出申请人、摘要、跳转链接）
type TaskItem struct {
	ID            uint      `json:"id"`
	InstanceID    uint      `json:"instance_id"`
	StepOrder     int       `json:"step_order"`
	StepName      string    `json:"step_name"`
	ApproverType  string    `json:"approver_type"`
	BizType       string    `json:"biz_type"`
	BizID         uint      `json:"biz_id"`
	ApplicantName string    `json:"applicant_name"`
	Summary       string    `json:"summary"` // 业务摘要（申请人 + 类型 + 时间范围）
	Status        string    `json:"status"`
	Link          string    `json:"link"`
	CreatedAt     time.Time `json:"created_at"`
}

// TaskDetail 审批进度时间线项（含步骤名、审批人姓名）
type TaskDetail struct {
	models.ApprovalTask
	StepName     string `json:"step_name"`
	ApproverName string `json:"approver_name"`
}

// BizDetail 按业务查询审批实例+进度（前端详情页展示）
type BizDetail struct {
	Instance *models.ApprovalInstance `json:"instance"`
	Tasks    []TaskDetail             `json:"tasks"`
}

// StartApproval 启动审批流程（事务内调用）：创建审批实例 + 步骤1任务 + 通知步骤1审批人
// bizType 为流程编码（leave / attendance_correction），bizID 为业务记录 ID，applicantUserID 为提交人用户 ID
func StartApproval(tx *gorm.DB, bizType string, bizID uint, applicantUserID uint) (*models.ApprovalInstance, error) {
	// 1. 查询该业务类型的 active 流程及步骤
	var flow models.ApprovalFlow
	if err := tx.Where("code = ? AND status = ?", bizType, "active").First(&flow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalFlowNotFound
		}
		return nil, err
	}
	var steps []models.ApprovalStep
	if err := tx.Where("flow_id = ?", flow.ID).Order("step_order ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrApprovalFlowNoStep
	}

	// 2. 创建审批实例（pending，从步骤1开始）
	inst := &models.ApprovalInstance{
		FlowID:      flow.ID,
		BizType:     bizType,
		BizID:       bizID,
		Status:      "pending",
		CurrentStep: steps[0].StepOrder,
		CreatedBy:   applicantUserID,
	}
	if err := tx.Create(inst).Error; err != nil {
		return nil, err
	}

	// 3. 解析步骤1审批人并创建任务
	approverID, err := resolveStepApproverUserID(tx, inst, &steps[0])
	if err != nil {
		return nil, err
	}
	task := &models.ApprovalTask{
		InstanceID:   inst.ID,
		StepOrder:    steps[0].StepOrder,
		ApproverType: steps[0].ApproverType,
		ApproverID:   approverID,
		Status:       "pending",
	}
	if err := tx.Create(task).Error; err != nil {
		return nil, err
	}

	// 4. 通知步骤1审批人（无具体审批人则跳过，由审批人从待办列表获取）
	approvalNotify(tx, inst, approverID)
	return inst, nil
}

// ApproveTask 审批通过（事务内调用）：任务 → approved；有下一步则推进并创建下一条任务+通知，无下一步则实例 → approved
// 返回实例，调用方根据 Status == "approved" 决定是否执行终审业务回调
func ApproveTask(tx *gorm.DB, taskID uint, approverUserID uint, comment string) (*models.ApprovalInstance, error) {
	task, err := loadPendingTask(tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := checkTaskApprover(tx, task, approverUserID); err != nil {
		return nil, err
	}
	inst, err := loadPendingInstance(tx, task.InstanceID)
	if err != nil {
		return nil, err
	}

	// 标记任务通过；HR 池任务（ApproverID 为空）回填实际审批人，保证历史可追溯
	updates := map[string]interface{}{"status": "approved", "comment": comment, "acted_at": time.Now()}
	if task.ApproverID == nil {
		updates["approver_id"] = approverUserID
	}
	if err := tx.Model(&models.ApprovalTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 查找下一步骤
	var nextStep models.ApprovalStep
	err = tx.Where("flow_id = ? AND step_order = ?", inst.FlowID, task.StepOrder+1).First(&nextStep).Error
	switch {
	case err == nil:
		// 推进实例到下一步
		if err := tx.Model(&models.ApprovalInstance{}).Where("id = ?", inst.ID).
			Update("current_step", nextStep.StepOrder).Error; err != nil {
			return nil, err
		}
		inst.CurrentStep = nextStep.StepOrder
		// 解析下一步审批人并创建任务
		approverID, err := resolveStepApproverUserID(tx, inst, &nextStep)
		if err != nil {
			return nil, err
		}
		newTask := &models.ApprovalTask{
			InstanceID:   inst.ID,
			StepOrder:    nextStep.StepOrder,
			ApproverType: nextStep.ApproverType,
			ApproverID:   approverID,
			Status:       "pending",
		}
		if err := tx.Create(newTask).Error; err != nil {
			return nil, err
		}
		approvalNotify(tx, inst, approverID)
		return inst, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 无下一步：实例终审通过
		if err := tx.Model(&models.ApprovalInstance{}).Where("id = ?", inst.ID).
			Update("status", "approved").Error; err != nil {
			return nil, err
		}
		inst.Status = "approved"
		return inst, nil
	default:
		return nil, err
	}
}

// RejectTask 驳回（事务内调用）：任务 → rejected；实例 → rejected
func RejectTask(tx *gorm.DB, taskID uint, approverUserID uint, comment string) error {
	task, err := loadPendingTask(tx, taskID)
	if err != nil {
		return err
	}
	if err := checkTaskApprover(tx, task, approverUserID); err != nil {
		return err
	}
	if _, err := loadPendingInstance(tx, task.InstanceID); err != nil {
		return err
	}

	updates := map[string]interface{}{"status": "rejected", "comment": comment, "acted_at": time.Now()}
	if task.ApproverID == nil {
		updates["approver_id"] = approverUserID
	}
	if err := tx.Model(&models.ApprovalTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		return err
	}
	return tx.Model(&models.ApprovalInstance{}).Where("id = ?", task.InstanceID).
		Update("status", "rejected").Error
}

// CancelApproval 取消审批（事务内调用）：仅 pending 且当前步骤任务仍 pending 时，申请人可取消
func CancelApproval(tx *gorm.DB, instanceID uint, applicantUserID uint) error {
	var inst models.ApprovalInstance
	if err := tx.First(&inst, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrApprovalInstanceNotFound
		}
		return err
	}
	if inst.CreatedBy != applicantUserID {
		return ErrApprovalNotAllowed
	}
	if inst.Status != "pending" {
		return ErrApprovalNotPending
	}
	// 当前步骤任务必须仍 pending（防止审批处理中取消）
	var task models.ApprovalTask
	if err := tx.Where("instance_id = ? AND step_order = ?", inst.ID, inst.CurrentStep).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrApprovalNotPending
		}
		return err
	}
	if task.Status != "pending" {
		return ErrApprovalNotPending
	}
	return tx.Model(&models.ApprovalInstance{}).Where("id = ?", inst.ID).
		Update("status", "cancelled").Error
}

// ResolveManagerUserID 员工 → 所属部门 → 部门负责人（manager_id）→ 负责人关联的系统账号用户 ID
// 无负责人或负责人无账号时返回 nil
func ResolveManagerUserID(db *gorm.DB, employeeID uint) (*uint, error) {
	var rows []struct {
		ManagerID *uint
	}
	err := db.Model(&models.Employee{}).
		Select("d.manager_id").
		Joins("JOIN departments d ON d.id = employees.department_id").
		Where("employees.id = ?", employeeID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 || rows[0].ManagerID == nil {
		return nil, nil
	}
	return repository.NewEmployeeRepo(db).FindUserIDByEmployeeID(*rows[0].ManagerID)
}

// FindHRUser 查找第一个启用（status=1）的 hr 角色用户 ID；无 hr 则回退找 admin；仍无返回 nil
func FindHRUser(db *gorm.DB) (*uint, error) {
	for _, code := range []string{"hr", "admin"} {
		var user models.User
		err := db.Joins("JOIN roles ON roles.id = users.role_id").
			Where("roles.code = ? AND users.status = 1", code).
			Order("users.id ASC").
			First(&user).Error
		if err == nil {
			return &user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

// ListMyTasks 我作为审批人的任务列表（pending/历史），JOIN 业务表带出申请人、摘要、跳转链接
// status：空=全部；pending=待审批；approved/rejected=对应历史
func ListMyTasks(db *gorm.DB, userID uint, status string) ([]TaskItem, error) {
	roleCode, err := getUserRoleCode(db, userID)
	if err != nil {
		return nil, err
	}
	isHR := roleCode == "hr" || roleCode == "admin"

	q := db.Table("approval_tasks").
		Select(`approval_tasks.*, ai.biz_type, ai.biz_id, ai.status AS instance_status,
			COALESCE(ast.name, '') AS step_name, COALESCE(e.name, '') AS applicant_name,
			lr.leave_type, lr.days, lr.start_date, lr.end_date,
			ac.work_date, ac.correction_type, ac.correction_time`).
		Joins("JOIN approval_instances ai ON ai.id = approval_tasks.instance_id").
		Joins("LEFT JOIN approval_steps ast ON ast.flow_id = ai.flow_id AND ast.step_order = approval_tasks.step_order").
		Joins("LEFT JOIN leave_requests lr ON ai.biz_type = 'leave' AND lr.id = ai.biz_id").
		Joins("LEFT JOIN attendance_corrections ac ON ai.biz_type = 'attendance_correction' AND ac.id = ai.biz_id").
		Joins("LEFT JOIN employees e ON e.id = COALESCE(lr.employee_id, ac.employee_id)")

	if isHR {
		// hr/admin：自己名下的任务 + HR 池任务（approver_type=hr 且未指派）
		q = q.Where("(approval_tasks.approver_id = ? OR (approval_tasks.approver_type = 'hr' AND approval_tasks.approver_id IS NULL))", userID)
	} else {
		q = q.Where("approval_tasks.approver_id = ?", userID)
	}

	switch status {
	case "pending":
		// 仅展示进行中实例的待办，排除已取消实例遗留的 pending 任务
		q = q.Where("approval_tasks.status = ? AND ai.status = ?", "pending", "pending")
	case "approved", "rejected":
		q = q.Where("approval_tasks.status = ?", status)
	}

	rows := make([]taskRow, 0)
	if err := q.Order("approval_tasks.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]TaskItem, 0, len(rows))
	for i := range rows {
		items = append(items, buildTaskItem(&rows[i]))
	}
	return items, nil
}

// GetInstanceByID 根据实例 ID 查询审批实例
func GetInstanceByID(db *gorm.DB, instanceID uint) (*models.ApprovalInstance, error) {
	var inst models.ApprovalInstance
	if err := db.First(&inst, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalInstanceNotFound
		}
		return nil, err
	}
	return &inst, nil
}

// GetInstanceByBiz 按业务类型+业务 ID 查询审批实例
func GetInstanceByBiz(db *gorm.DB, bizType string, bizID uint) (*models.ApprovalInstance, error) {
	var inst models.ApprovalInstance
	if err := db.Where("biz_type = ? AND biz_id = ?", bizType, bizID).First(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetInstanceByTask 根据任务 ID 查询所属审批实例
func GetInstanceByTask(db *gorm.DB, taskID uint) (*models.ApprovalInstance, error) {
	var task models.ApprovalTask
	if err := db.Select("instance_id").First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalTaskNotFound
		}
		return nil, err
	}
	return GetInstanceByID(db, task.InstanceID)
}

// ListInstanceTasks 某实例的审批进度时间线（步骤名/审批人/状态/时间/意见），按步骤顺序
func ListInstanceTasks(db *gorm.DB, instanceID uint) ([]TaskDetail, error) {
	rows := make([]TaskDetail, 0)
	err := db.Table("approval_tasks").
		Select(`approval_tasks.*, COALESCE(ast.name, '') AS step_name,
			COALESCE(e.name, u.full_name, '') AS approver_name`).
		Joins("JOIN approval_instances ai ON ai.id = approval_tasks.instance_id").
		Joins("LEFT JOIN approval_steps ast ON ast.flow_id = ai.flow_id AND ast.step_order = approval_tasks.step_order").
		Joins("LEFT JOIN users u ON u.id = approval_tasks.approver_id").
		Joins("LEFT JOIN employees e ON e.id = u.employee_id").
		Where("approval_tasks.instance_id = ?", instanceID).
		Order("approval_tasks.step_order ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// taskRow ListMyTasks 扫描行：审批任务 + 实例业务信息（申请人/摘要字段）
type taskRow struct {
	models.ApprovalTask
	BizType        string     `gorm:"column:biz_type"`
	BizID          uint       `gorm:"column:biz_id"`
	InstanceStatus string     `gorm:"column:instance_status"`
	StepName       string     `gorm:"column:step_name"`
	ApplicantName  string     `gorm:"column:applicant_name"`
	LeaveType      string     `gorm:"column:leave_type"`
	Days           float64    `gorm:"column:days"`
	StartDate      *time.Time `gorm:"column:start_date"`
	EndDate        *time.Time `gorm:"column:end_date"`
	WorkDate       *time.Time `gorm:"column:work_date"`
	CorrectionType string     `gorm:"column:correction_type"`
	CorrectionTime string     `gorm:"column:correction_time"`
}

// buildTaskItem 组装待办/历史任务项：按业务类型拼装摘要与跳转链接
func buildTaskItem(r *taskRow) TaskItem {
	item := TaskItem{
		ID:            r.ID,
		InstanceID:    r.InstanceID,
		StepOrder:     r.StepOrder,
		StepName:      r.StepName,
		ApproverType:  r.ApproverType,
		BizType:       r.BizType,
		BizID:         r.BizID,
		ApplicantName: r.ApplicantName,
		Status:        r.Status,
		CreatedAt:     r.CreatedAt,
	}
	switch r.BizType {
	case "leave":
		item.Summary = fmt.Sprintf("%s 提交 %s %s 天（%s~%s）", r.ApplicantName,
			leaveTypeNames[r.LeaveType], formatDays(r.Days),
			formatDatePtr(r.StartDate), formatDatePtr(r.EndDate))
		item.Link = "/leaves"
	case "attendance_correction":
		item.Summary = fmt.Sprintf("%s 提交%s补卡申请（%s %s）", r.ApplicantName,
			correctionTypeNames[r.CorrectionType], formatDatePtr(r.WorkDate), r.CorrectionTime)
		item.Link = "/attendance"
	}
	return item
}

// loadPendingTask 加载任务并校验其仍处于 pending 状态
func loadPendingTask(tx *gorm.DB, taskID uint) (*models.ApprovalTask, error) {
	var task models.ApprovalTask
	if err := tx.First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalTaskNotFound
		}
		return nil, err
	}
	if task.Status != "pending" {
		return nil, ErrApprovalTaskNotPending
	}
	return &task, nil
}

// loadPendingInstance 加载实例并校验其仍处于 pending 状态
func loadPendingInstance(tx *gorm.DB, instanceID uint) (*models.ApprovalInstance, error) {
	var inst models.ApprovalInstance
	if err := tx.First(&inst, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalInstanceNotFound
		}
		return nil, err
	}
	if inst.Status != "pending" {
		return nil, ErrApprovalNotPending
	}
	return &inst, nil
}

// checkTaskApprover 审批权限校验：ApproverID 匹配优先；
// HR 池任务（approver_type=hr 且 ApproverID 为空）允许任意 hr/admin 审批
func checkTaskApprover(tx *gorm.DB, task *models.ApprovalTask, approverUserID uint) error {
	if task.ApproverID != nil && *task.ApproverID == approverUserID {
		return nil
	}
	if task.ApproverType == "hr" && task.ApproverID == nil {
		roleCode, err := getUserRoleCode(tx, approverUserID)
		if err != nil {
			return err
		}
		if roleCode == "hr" || roleCode == "admin" {
			return nil
		}
	}
	return ErrApprovalNotAllowed
}

// getUserRoleCode 查询用户角色编码
func getUserRoleCode(db *gorm.DB, userID uint) (string, error) {
	var user models.User
	if err := db.Select("role_id").First(&user, userID).Error; err != nil {
		return "", err
	}
	var role models.Role
	if err := db.Select("code").First(&role, user.RoleID).Error; err != nil {
		return "", err
	}
	return role.Code, nil
}

// resolveStepApproverUserID 解析某步骤的审批人用户 ID
// manager：申请人的部门经理（无则兜底任意 hr/admin）；hr：留空由任一 hr/admin 审批
func resolveStepApproverUserID(tx *gorm.DB, inst *models.ApprovalInstance, step *models.ApprovalStep) (*uint, error) {
	if step.ApproverType == "hr" {
		return nil, nil
	}
	// manager：申请人的部门经理
	empID, err := repository.NewEmployeeRepo(tx).FindEmployeeIDByUserID(inst.CreatedBy)
	if err != nil {
		return nil, err
	}
	if empID == nil {
		return FindHRUser(tx)
	}
	mgrUserID, err := ResolveManagerUserID(tx, *empID)
	if err != nil {
		return nil, err
	}
	if mgrUserID != nil {
		return mgrUserID, nil
	}
	// 无部门经理（或无账号）：兜底给任意 hr，防止流程卡死
	return FindHRUser(tx)
}

// approvalNotify 向指定审批人发送待审批通知（无具体审批人则跳过）
func approvalNotify(tx *gorm.DB, inst *models.ApprovalInstance, approverUserID *uint) {
	if approverUserID == nil {
		return
	}
	var ntype, title, link string
	switch inst.BizType {
	case "leave":
		ntype, title, link = NotifTypeLeaveApproval, "新的请假申请待审批", "/leaves"
	case "attendance_correction":
		ntype, title, link = NotifTypeAttCorrection, "新的补卡申请待审批", "/attendance"
	default:
		return
	}
	applicantName, detail := bizSummary(tx, inst)
	notif := NewNotificationService(repository.NewNotificationRepo(tx), repository.NewEmployeeRepo(tx))
	_ = notif.Send(*approverUserID, ntype, title, fmt.Sprintf("%s %s", applicantName, detail), link)
}

// bizSummary 返回业务申请人姓名与摘要（供通知与展示）
func bizSummary(tx *gorm.DB, inst *models.ApprovalInstance) (applicantName, detail string) {
	switch inst.BizType {
	case "leave":
		var lr models.LeaveRequest
		if err := tx.First(&lr, inst.BizID).Error; err != nil {
			return "", ""
		}
		var emp models.Employee
		if err := tx.Select("name").First(&emp, lr.EmployeeID).Error; err != nil {
			return "", ""
		}
		return emp.Name, fmt.Sprintf("提交了 %s %s 天（%s~%s）",
			leaveTypeNames[lr.LeaveType], formatDays(lr.Days),
			lr.StartDate.Format("2006-01-02"), lr.EndDate.Format("2006-01-02"))
	case "attendance_correction":
		var cor models.AttendanceCorrection
		if err := tx.First(&cor, inst.BizID).Error; err != nil {
			return "", ""
		}
		var emp models.Employee
		if err := tx.Select("name").First(&emp, cor.EmployeeID).Error; err != nil {
			return "", ""
		}
		return emp.Name, fmt.Sprintf("提交了%s补卡申请（%s %s）",
			correctionTypeNames[cor.CorrectionType],
			cor.WorkDate.Format("2006-01-02"), cor.CorrectionTime)
	}
	return "", ""
}

// formatDatePtr 时间指针格式化（空指针返回空串）
func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
