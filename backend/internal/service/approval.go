package service

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// bizApprovePermissions 业务类型 → 审批权限点 code：
// 未指派 manager 任务（部门全链无经理）的归属匹配与通知广播均按此推断可审批用户
var bizApprovePermissions = map[string]string{
	"leave":                 "leave:approve",
	"attendance_correction": "attendance_correction:approve",
}

// 审批引擎业务错误
var (
	ErrApprovalFlowNotFound     = errors.New("审批流程未配置")
	ErrApprovalFlowNoStep       = errors.New("审批流程未配置审批步骤")
	ErrApprovalInstanceNotFound = errors.New("审批实例不存在")
	ErrApprovalTaskNotFound     = errors.New("审批任务不存在")
	ErrApprovalTaskNotPending   = errors.New("审批任务已处理")
	ErrApprovalNotPending       = errors.New("审批实例当前状态不可操作")
	ErrApprovalNotAllowed       = errors.New("无权审批该任务")
)

// ApprovalBizCallback 审批终审回调接口：业务模块实现，handler 在审批流 final 后同一事务内调用
type ApprovalBizCallback interface {
	// ApplyApproved 终审通过回调（事务内）：更新业务状态、扣余额、联动考勤、通知等
	ApplyApproved(tx *gorm.DB, bizID uint) error
	// ApplyRejected 驳回回调（事务内）：业务状态置为 rejected、通知申请人
	ApplyRejected(tx *gorm.DB, bizID uint, reason string) error
}

// TaskItem 我的待审批/历史任务项（JOIN 业务表带出申请人、摘要、跳转链接）
// 业务明细字段（leave_type/days/reason/work_date 等）仅供前端列表精确展示，可选返回
type TaskItem struct {
	ID             uint       `json:"id"`
	InstanceID     uint       `json:"instance_id"`
	StepOrder      int        `json:"step_order"`
	StepName       string     `json:"step_name"`
	ApproverType   string     `json:"approver_type"`
	BizType        string     `json:"biz_type"`
	BizID          uint       `json:"biz_id"`
	ApplicantName  string     `json:"applicant_name"`
	DepartmentName string     `json:"department_name"`
	Summary        string     `json:"summary"` // 业务摘要（申请人 + 类型 + 时间范围）
	Status         string     `json:"status"`
	Link           string     `json:"link"`
	LeaveType      string     `json:"leave_type"`
	Days           float64    `json:"days"`
	Reason         string     `json:"reason"`
	WorkDate       *time.Time `json:"work_date"`
	CorrectionType string     `json:"correction_type"`
	CorrectionTime string     `json:"correction_time"`
	CreatedAt      time.Time  `json:"created_at"`
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
		InstanceID:         inst.ID,
		StepOrder:          steps[0].StepOrder,
		ApproverType:       steps[0].ApproverType,
		ApproverID:         approverID,
		ApproverRef:        steps[0].ApproverRef,
		ApproverPermission: steps[0].ApproverPermission,
		ApproverRoleID:     steps[0].ApproverRoleID,
		Status:             "pending",
	}
	if err := tx.Create(task).Error; err != nil {
		return nil, err
	}

	// 4. 通知步骤1审批人（有具体审批人 → 单独通知；HR/角色/权限/岗位池及未指派 manager 步骤 → 按对应池广播，无目标则跳过）
	approvalNotify(tx, inst, approverID, &steps[0])
	return inst, nil
}

// ApproveTask 审批通过（事务内调用）：任务 → approved；有下一步则推进并创建下一条任务+通知，无下一步则实例 → approved
// 返回实例，调用方根据 Status == "approved" 决定是否执行终审业务回调
func ApproveTask(tx *gorm.DB, taskID uint, approverUserID uint, comment string) (*models.ApprovalInstance, error) {
	task, err := loadPendingTask(tx, taskID)
	if err != nil {
		return nil, err
	}
	// 先加载实例（校验 pending + 取 BizType 供未指派 manager 任务归属校验）
	inst, err := loadPendingInstance(tx, task.InstanceID)
	if err != nil {
		return nil, err
	}
	if err := checkTaskApprover(tx, task, inst.BizType, approverUserID); err != nil {
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
			InstanceID:         inst.ID,
			StepOrder:          nextStep.StepOrder,
			ApproverType:       nextStep.ApproverType,
			ApproverID:         approverID,
			ApproverRef:        nextStep.ApproverRef,
			ApproverPermission: nextStep.ApproverPermission,
			ApproverRoleID:     nextStep.ApproverRoleID,
			Status:             "pending",
		}
		if err := tx.Create(newTask).Error; err != nil {
			return nil, err
		}
		approvalNotify(tx, inst, approverID, &nextStep)
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
	// 先加载实例（校验 pending + 取 BizType 供未指派 manager 任务归属校验）
	inst, err := loadPendingInstance(tx, task.InstanceID)
	if err != nil {
		return err
	}
	if err := checkTaskApprover(tx, task, inst.BizType, approverUserID); err != nil {
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

// ResolveManagerUserID 员工 → 所属部门 → 沿 parent_id 递归向上找第一个配置了 manager_id 的部门，
// 返回该部门负责人关联的系统账号用户 ID。
// 直属部门无经理时向上层部门解析（如 算法组 → 研发部 → 技术部），全链均无经理或无账号时返回 nil。
func ResolveManagerUserID(db *gorm.DB, employeeID uint) (*uint, error) {
	// 1. 查员工所在部门
	var emp struct {
		DepartmentID uint
	}
	if err := db.Model(&models.Employee{}).
		Select("department_id").
		Where("id = ?", employeeID).
		Scan(&emp).Error; err != nil {
		return nil, err
	}
	// 员工不存在（Scan 不报 RecordNotFound）或未配置部门 → 返回 nil
	if emp.DepartmentID == 0 {
		return nil, nil
	}
	// 2. 从所在部门开始循环向上（parent_id），返回第一个有 manager_id 的部门经理账号
	deptID := emp.DepartmentID
	for deptID != 0 {
		var dept struct {
			ManagerID *uint
			ParentID  uint
		}
		if err := db.Model(&models.Department{}).
			Select("manager_id, parent_id").
			Where("id = ?", deptID).
			Scan(&dept).Error; err != nil {
			return nil, err
		}
		if dept.ManagerID != nil {
			return repository.NewEmployeeRepo(db).FindUserIDByEmployeeID(*dept.ManagerID)
		}
		deptID = dept.ParentID
	}
	return nil, nil
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

// FindAllHRUserIDs 查找所有启用（status=1）的 hr 角色用户 ID（HR 池步骤广播通知用）；
// 一个 hr 都没有则回退查 admin；仍无返回空切片
func FindAllHRUserIDs(db *gorm.DB) ([]uint, error) {
	ids := make([]uint, 0)
	for _, code := range []string{"hr", "admin"} {
		var tmp []uint
		if err := db.Model(&models.User{}).
			Joins("JOIN roles ON roles.id = users.role_id").
			Where("roles.code = ? AND users.status = 1", code).
			Order("users.id ASC").
			Pluck("users.id", &tmp).Error; err != nil {
			return nil, err
		}
		if len(tmp) > 0 {
			return tmp, nil
		}
	}
	return ids, nil
}

// FindUsersByPositionID 查询指定岗位所有在职员工的系统账号用户 ID（position 池广播通知用）；
// JOIN employees ON users.employee_id=employees.id，须员工在职（status=1）且账号启用（users.status=1）；无则返回空切片
func FindUsersByPositionID(db *gorm.DB, positionID uint) ([]uint, error) {
	ids := make([]uint, 0)
	if positionID == 0 {
		return ids, nil
	}
	if err := db.Model(&models.User{}).
		Joins("JOIN employees ON employees.id = users.employee_id").
		Where("employees.position_id = ? AND employees.status = 1 AND users.status = 1", positionID).
		Order("users.id ASC").
		Pluck("users.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// UserHasPermission 判断当前用户是否拥有指定权限点（查 users.role_id → role_permissions → permissions.code）
// permission 池任务归属/审批校验用；permCode 为空返回 false
func UserHasPermission(db *gorm.DB, userID uint, permCode string) (bool, error) {
	if permCode == "" {
		return false, nil
	}
	var user models.User
	if err := db.Select("role_id").First(&user, userID).Error; err != nil {
		return false, err
	}
	var count int64
	err := db.Model(&models.RolePermission{}).
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ? AND permissions.code = ?", user.RoleID, permCode).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// getUserPermissionCodes 查询当前用户所属角色拥有的全部权限点 code（ListMyTasks/CountMyPendingTasks 权限池匹配用）
func getUserPermissionCodes(db *gorm.DB, userID uint) ([]string, error) {
	var user models.User
	if err := db.Select("role_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	codes := make([]string, 0)
	err := db.Model(&models.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", user.RoleID).
		Pluck("permissions.code", &codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// FindUsersByPermission 查询拥有指定权限点的所有启用用户 ID（permission 池广播通知用）；
// 经 users.role_id → role_permissions → permissions.code 关联，须账号启用（users.status=1）；无则返回空切片
func FindUsersByPermission(db *gorm.DB, permCode string) ([]uint, error) {
	ids := make([]uint, 0)
	if permCode == "" {
		return ids, nil
	}
	if err := db.Model(&models.User{}).
		Joins("JOIN roles ON roles.id = users.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("permissions.code = ? AND users.status = 1", permCode).
		Order("users.id ASC").
		Pluck("users.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// FindUsersByRoleID 查询指定角色的所有启用用户 ID（role 池广播通知用）；
// 须账号启用（users.status=1）；无则返回空切片
func FindUsersByRoleID(db *gorm.DB, roleID uint) ([]uint, error) {
	ids := make([]uint, 0)
	if roleID == 0 {
		return ids, nil
	}
	if err := db.Model(&models.User{}).
		Where("role_id = ? AND status = 1", roleID).
		Order("id ASC").
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// getUserPositionID 查询当前用户关联员工档案的岗位 ID（position 池任务归属匹配用）；
// 用户无员工档案或非在职返回 0
func getUserPositionID(db *gorm.DB, userID uint) (uint, error) {
	empID, err := repository.NewEmployeeRepo(db).FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, nil
	}
	var emp models.Employee
	if err := db.Select("position_id, status").First(&emp, *empID).Error; err != nil {
		return 0, err
	}
	if emp.Status != 1 {
		return 0, nil
	}
	return emp.PositionID, nil
}

// ListMyTasks 我作为审批人的任务列表（pending/历史），JOIN 业务表带出申请人、摘要、跳转链接
// status：空=全部；pending=待审批；approved/rejected=对应历史
func ListMyTasks(db *gorm.DB, userID uint, status string) ([]TaskItem, error) {
	roleCode, err := getUserRoleCode(db, userID)
	if err != nil {
		return nil, err
	}
	isHR := roleCode == "hr"
	roleID, err := getUserRoleID(db, userID)
	if err != nil {
		return nil, err
	}
	positionID, err := getUserPositionID(db, userID)
	if err != nil {
		return nil, err
	}
	permCodes, err := getUserPermissionCodes(db, userID)
	if err != nil {
		return nil, err
	}

	q := db.Table("approval_tasks").
		Select(`approval_tasks.*, ai.biz_type, ai.biz_id, ai.status AS instance_status,
			COALESCE(ast.name, '') AS step_name, COALESCE(e.name, '') AS applicant_name,
			COALESCE(d.name, '') AS department_name,
			lr.leave_type, lr.days, lr.start_date, lr.end_date, lr.reason,
			ac.work_date, ac.correction_type, ac.correction_time`).
		Joins("JOIN approval_instances ai ON ai.id = approval_tasks.instance_id").
		Joins("LEFT JOIN approval_steps ast ON ast.flow_id = ai.flow_id AND ast.step_order = approval_tasks.step_order").
		Joins("LEFT JOIN leave_requests lr ON ai.biz_type = 'leave' AND lr.id = ai.biz_id").
		Joins("LEFT JOIN attendance_corrections ac ON ai.biz_type = 'attendance_correction' AND ac.id = ai.biz_id").
		Joins("LEFT JOIN employees e ON e.id = COALESCE(lr.employee_id, ac.employee_id)").
		Joins("LEFT JOIN departments d ON d.id = e.department_id")

	// 归属口径（与 checkTaskApprover / CountMyPendingTasks 一致）：
	// approver_id 指派匹配；HR 池任务（approver_type=hr 且未指派）仅 hr 角色可见；
	// position 池任务（approver_type=position 且未指派）仅该岗位在职员工可见；
	// role 池任务（approver_type=role 且未指派）仅该角色用户可见；
	// permission 池任务（approver_type=permission 且未指派）仅拥有对应权限点的用户可见；
	// 未指派 manager 任务（approver_type=manager 且未指派）→ 拥有该业务类型审批权限点的用户可审（ai.biz_type 推断）；admin 不出现任何业务审批任务
	matchParts := []string{"approval_tasks.approver_id = ?"}
	matchArgs := []interface{}{userID}
	if isHR {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'hr' AND approval_tasks.approver_id IS NULL)")
	}
	matchParts = append(matchParts, "(approval_tasks.approver_type = 'position' AND approval_tasks.approver_ref = ?)")
	matchArgs = append(matchArgs, positionID)
	if roleID != 0 {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'role' AND approval_tasks.approver_id IS NULL AND approval_tasks.approver_role_id = ?)")
		matchArgs = append(matchArgs, roleID)
	}
	if len(permCodes) > 0 {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'permission' AND approval_tasks.approver_id IS NULL AND approval_tasks.approver_permission IN ?)")
		matchArgs = append(matchArgs, permCodes)
	}
	if slices.Contains(permCodes, "leave:approve") {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'manager' AND approval_tasks.approver_id IS NULL AND ai.biz_type = 'leave')")
	}
	if slices.Contains(permCodes, "attendance_correction:approve") {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'manager' AND approval_tasks.approver_id IS NULL AND ai.biz_type = 'attendance_correction')")
	}
	q = q.Where("("+strings.Join(matchParts, " OR ")+")", matchArgs...)

	switch status {
	case "pending":
		// 仅展示进行中实例的待办，排除已取消实例遗留的 pending 任务
		q = q.Where("approval_tasks.status = ? AND ai.status = ?", "pending", "pending")
	case "approved", "rejected":
		q = q.Where("approval_tasks.status = ?", status)
	case "done":
		// 我处理的全部历史（通过+驳回）
		q = q.Where("approval_tasks.status IN (?, ?)", "approved", "rejected")
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

// CountMyPendingTasks 当前用户待审批任务数（导航栏角标用）：
// 与 ListMyTasks / checkTaskApprover 的归属口径一致——ApproverID 匹配，或 hr 可认领的 HR 池任务（approver_type=hr 且未指派），
// 或 position 池任务（approver_type=position 且 approver_ref=当前用户岗位），
// 或 permission 池任务（approver_type=permission 且未指派且用户拥有对应权限点）；admin 不计入业务审批
func CountMyPendingTasks(db *gorm.DB, userID uint) (int64, error) {
	roleCode, err := getUserRoleCode(db, userID)
	if err != nil {
		return 0, err
	}
	isHR := roleCode == "hr"
	roleID, err := getUserRoleID(db, userID)
	if err != nil {
		return 0, err
	}
	positionID, err := getUserPositionID(db, userID)
	if err != nil {
		return 0, err
	}
	permCodes, err := getUserPermissionCodes(db, userID)
	if err != nil {
		return 0, err
	}

	q := db.Model(&models.ApprovalTask{}).
		Joins("JOIN approval_instances ai ON ai.id = approval_tasks.instance_id").
		Where("approval_tasks.status = ? AND ai.status = ?", "pending", "pending")
	// 归属口径与 ListMyTasks / checkTaskApprover 一致（见 ListMyTasks 注释）
	matchParts := []string{"approval_tasks.approver_id = ?"}
	matchArgs := []interface{}{userID}
	if isHR {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'hr' AND approval_tasks.approver_id IS NULL)")
	}
	matchParts = append(matchParts, "(approval_tasks.approver_type = 'position' AND approval_tasks.approver_ref = ?)")
	matchArgs = append(matchArgs, positionID)
	if roleID != 0 {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'role' AND approval_tasks.approver_id IS NULL AND approval_tasks.approver_role_id = ?)")
		matchArgs = append(matchArgs, roleID)
	}
	if len(permCodes) > 0 {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'permission' AND approval_tasks.approver_id IS NULL AND approval_tasks.approver_permission IN ?)")
		matchArgs = append(matchArgs, permCodes)
	}
	if slices.Contains(permCodes, "leave:approve") {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'manager' AND approval_tasks.approver_id IS NULL AND ai.biz_type = 'leave')")
	}
	if slices.Contains(permCodes, "attendance_correction:approve") {
		matchParts = append(matchParts, "(approval_tasks.approver_type = 'manager' AND approval_tasks.approver_id IS NULL AND ai.biz_type = 'attendance_correction')")
	}
	q = q.Where("("+strings.Join(matchParts, " OR ")+")", matchArgs...)
	var n int64
	err = q.Count(&n).Error
	return n, err
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
	DepartmentName string     `gorm:"column:department_name"`
	LeaveType      string     `gorm:"column:leave_type"`
	Days           float64    `gorm:"column:days"`
	StartDate      *time.Time `gorm:"column:start_date"`
	EndDate        *time.Time `gorm:"column:end_date"`
	Reason         string     `gorm:"column:reason"`
	WorkDate       *time.Time `gorm:"column:work_date"`
	CorrectionType string     `gorm:"column:correction_type"`
	CorrectionTime string     `gorm:"column:correction_time"`
}

// buildTaskItem 组装待办/历史任务项：按业务类型拼装摘要与跳转链接
func buildTaskItem(r *taskRow) TaskItem {
	item := TaskItem{
		ID:             r.ID,
		InstanceID:     r.InstanceID,
		StepOrder:      r.StepOrder,
		StepName:       r.StepName,
		ApproverType:   r.ApproverType,
		BizType:        r.BizType,
		BizID:          r.BizID,
		ApplicantName:  r.ApplicantName,
		DepartmentName: r.DepartmentName,
		Status:         r.Status,
		LeaveType:      r.LeaveType,
		Days:           r.Days,
		Reason:         r.Reason,
		WorkDate:       r.WorkDate,
		CorrectionType: r.CorrectionType,
		CorrectionTime: r.CorrectionTime,
		CreatedAt:      r.CreatedAt,
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
// HR 池任务（approver_type=hr 且 ApproverID 为空）仅允许 hr 角色审批，admin 不再放行（admin 专注系统管理）；
// position 池任务（approver_type=position 且 ApproverID 为空）仅允许该岗位的在职员工用户审批；
// permission 池任务（approver_type=permission 且 ApproverID 为空）仅允许拥有该审批权限点的用户审批；
// role 池任务（approver_type=role 且 ApproverID 为空）仅允许 ApproverRoleID 对应角色的用户审批；
// 未指派 manager 任务（approver_type=manager 且 ApproverID 为空）→ 拥有该业务类型审批权限点的用户可审批（bizType 推断）
func checkTaskApprover(tx *gorm.DB, task *models.ApprovalTask, bizType string, approverUserID uint) error {
	if task.ApproverID != nil && *task.ApproverID == approverUserID {
		return nil
	}
	switch task.ApproverType {
	case "hr":
		if task.ApproverID == nil {
			roleCode, err := getUserRoleCode(tx, approverUserID)
			if err != nil {
				return err
			}
			if roleCode == "hr" {
				return nil
			}
		}
	case "position":
		if task.ApproverID == nil {
			positionID, err := getUserPositionID(tx, approverUserID)
			if err != nil {
				return err
			}
			if positionID != 0 && positionID == task.ApproverRef {
				return nil
			}
		}
	case "permission":
		if task.ApproverID == nil {
			ok, err := UserHasPermission(tx, approverUserID, task.ApproverPermission)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	case "role":
		if task.ApproverID == nil {
			roleID, err := getUserRoleID(tx, approverUserID)
			if err != nil {
				return err
			}
			if roleID != 0 && roleID == task.ApproverRoleID {
				return nil
			}
		}
	case "manager":
		if task.ApproverID == nil {
			// 未指派 manager 任务：按业务类型推断审批权限点
			permCode := bizApprovePermissions[bizType]
			if permCode == "" {
				return ErrApprovalNotAllowed
			}
			ok, err := UserHasPermission(tx, approverUserID, permCode)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
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

// getUserRoleID 查询当前用户的角色 ID（role 池任务归属匹配用）
func getUserRoleID(db *gorm.DB, userID uint) (uint, error) {
	var user models.User
	if err := db.Select("role_id").First(&user, userID).Error; err != nil {
		return 0, err
	}
	return user.RoleID, nil
}

// resolveStepApproverUserID 解析某步骤的审批人用户 ID
// hr / position / permission / role：池步骤留空（approver_id=NULL），由池内任一审批人认领
// （HR 池=hr 角色；岗位池=ApproverRef 指定岗位的在职员工；权限池=拥有 ApproverPermission 权限点的所有启用用户；
//  角色池=ApproverRoleID 指定角色的所有启用用户）
// manager：申请人的部门经理递归解析；全链无经理则留空（approver_id=NULL），成为未指派任务
// （归属按业务类型审批权限点推断，见 ListMyTasks / checkTaskApprover），不再硬指派 hr
func resolveStepApproverUserID(tx *gorm.DB, inst *models.ApprovalInstance, step *models.ApprovalStep) (*uint, error) {
	switch step.ApproverType {
	case "hr", "position", "permission", "role":
		return nil, nil
	case "manager":
		// 申请人的部门经理（递归向上解析）
		empID, err := repository.NewEmployeeRepo(tx).FindEmployeeIDByUserID(inst.CreatedBy)
		if err != nil {
			return nil, err
		}
		if empID == nil {
			return nil, nil
		}
		mgrUserID, err := ResolveManagerUserID(tx, *empID)
		if err != nil {
			return nil, err
		}
		return mgrUserID, nil
	}
	// 未知类型返回 nil（不指派具体审批人，避免流程中断）
	return nil, nil
}

// approvalNotify 向审批人发送待审批通知：
// 有具体审批人 → 发给该用户；HR 池步骤（approverUserID 为 nil 且 ApproverType=hr）→ 广播给所有启用 HR 用户（无 HR 回退 admin）；
// position 池步骤（approverUserID 为 nil 且 ApproverType=position）→ 广播给该岗位所有在职员工的用户；
// permission 池步骤（approverUserID 为 nil 且 ApproverType=permission）→ 广播给拥有该审批权限点的所有启用用户；
// role 池步骤（approverUserID 为 nil 且 ApproverType=role）→ 广播给该角色的所有启用用户；
// 未指派 manager 步骤（approverUserID 为 nil 且 ApproverType=manager）→ 按业务类型广播给拥有对应审批权限点的用户；其余无审批人情况保持跳过
func approvalNotify(tx *gorm.DB, inst *models.ApprovalInstance, approverUserID *uint, step *models.ApprovalStep) {
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
	content := fmt.Sprintf("%s %s", applicantName, detail)

	if approverUserID != nil {
		_ = notif.Send(*approverUserID, ntype, title, content, link)
		return
	}
	if step == nil {
		return
	}
	switch step.ApproverType {
	case "hr":
		// HR 池步骤：广播通知所有 HR（无 HR 回退 admin），保证 hr 能收到终审通知
		ids, err := FindAllHRUserIDs(tx)
		if err != nil {
			slog.Warn("查询 HR 用户发送审批通知失败", "error", err, "instance_id", inst.ID)
			return
		}
		for _, id := range ids {
			_ = notif.Send(id, ntype, title, content, link)
		}
	case "position":
		// position 池步骤：广播通知该岗位所有在职员工的用户
		ids, err := FindUsersByPositionID(tx, step.ApproverRef)
		if err != nil {
			slog.Warn("查询岗位用户发送审批通知失败", "error", err, "instance_id", inst.ID, "position_id", step.ApproverRef)
			return
		}
		for _, id := range ids {
			_ = notif.Send(id, ntype, title, content, link)
		}
	case "permission":
		// permission 池步骤：广播通知所有拥有该审批权限点的启用用户
		ids, err := FindUsersByPermission(tx, step.ApproverPermission)
		if err != nil {
			slog.Warn("查询权限用户发送审批通知失败", "error", err, "instance_id", inst.ID, "permission", step.ApproverPermission)
			return
		}
		for _, id := range ids {
			_ = notif.Send(id, ntype, title, content, link)
		}
	case "role":
		// role 池步骤：广播通知该角色的所有启用用户
		ids, err := FindUsersByRoleID(tx, step.ApproverRoleID)
		if err != nil {
			slog.Warn("查询角色用户发送审批通知失败", "error", err, "instance_id", inst.ID, "role_id", step.ApproverRoleID)
			return
		}
		for _, id := range ids {
			_ = notif.Send(id, ntype, title, content, link)
		}
	case "manager":
		// 未指派 manager 步骤：按业务类型广播给拥有对应审批权限点的启用用户（hr 无权限即不再收到）
		permCode := bizApprovePermissions[inst.BizType]
		if permCode == "" {
			return
		}
		ids, err := FindUsersByPermission(tx, permCode)
		if err != nil {
			slog.Warn("查询未指派经理审批权限用户发送通知失败", "error", err, "instance_id", inst.ID, "biz_type", inst.BizType)
			return
		}
		for _, id := range ids {
			_ = notif.Send(id, ntype, title, content, link)
		}
	}
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
