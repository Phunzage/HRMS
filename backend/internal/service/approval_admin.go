package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// 审批流程配置业务错误
var (
	ErrApprovalFlowCodeInvalid = errors.New("审批流程编码无效")
	ErrApprovalFlowStepCount   = errors.New("审批步骤数须为 1~5 级")
	ErrApprovalFlowStepOrder   = errors.New("审批步骤顺序必须从 1 开始且连续")
	ErrApprovalFlowTypeInvalid = errors.New("审批步骤类型不合法（manager/role/permission/hr/position）")
	ErrApprovalFlowPosRefBad   = errors.New("指定岗位审批必须填写有效的岗位 ID")
	ErrApprovalFlowRefConflict = errors.New("manager/hr 步骤不得填写岗位引用")
	ErrApprovalFlowRoleBad     = errors.New("指定角色审批必须填写有效的角色 ID，且 manager/hr/position 步骤不得填写角色 ID")
	ErrApprovalFlowPermBad     = errors.New("指定权限审批必须填写有效的审批权限点，且 manager/hr/position/role 步骤不得填写权限点")
)

// validApprovalFlowCodes 可配置的流程编码（与业务申请 bizType 对应）
var validApprovalFlowCodes = map[string]bool{
	"leave":                 true,
	"attendance_correction": true,
}

// validApproverTypes 审批步骤合法审批人类型（hr/position/permission 为兼容旧配置保留，新配置建议 manager/role）
var validApproverTypes = map[string]bool{
	"manager":    true,
	"role":       true,
	"permission": true,
	"hr":         true,
	"position":   true,
}

// validApprovalPermissions 审批权限点合法取值（permission 类型步骤使用）
var validApprovalPermissions = map[string]bool{
	"leave:approve":                 true,
	"attendance_correction:approve": true,
}

// ApprovalFlowStepInput 流程步骤配置输入
type ApprovalFlowStepInput struct {
	StepOrder          int    `json:"step_order"`
	ApproverType       string `json:"approver_type"`
	ApproverRef        uint   `json:"approver_ref"`
	ApproverPermission string `json:"approver_permission"` // 审批权限点 code，approver_type=permission 时必填
	ApproverRoleID     uint   `json:"approver_role_id"`    // 审批角色 ID，approver_type=role 时必填
	Name               string `json:"name"`
}

// UpdateApprovalFlowRequest 更新审批流程请求体
type UpdateApprovalFlowRequest struct {
	Name  string                  `json:"name"`
	Steps []ApprovalFlowStepInput `json:"steps"`
}

// ApprovalStepDetail 审批步骤详情（含审批角色名称，供前端角色池下拉展示）
type ApprovalStepDetail struct {
	models.ApprovalStep
	ApproverRoleName string `json:"approver_role_name"` // 审批角色名称，approver_type=role 时 JOIN roles 带出
}

// ApprovalFlowDetail 审批流程详情（含步骤，按 step_order 升序）
type ApprovalFlowDetail struct {
	models.ApprovalFlow
	Steps []ApprovalStepDetail `json:"steps"`
}

// ListApprovalFlows 查询所有审批流程及步骤（admin 流程配置页用）
func ListApprovalFlows(db *gorm.DB) ([]ApprovalFlowDetail, error) {
	var flows []models.ApprovalFlow
	if err := db.Order("id ASC").Find(&flows).Error; err != nil {
		return nil, err
	}
	details := make([]ApprovalFlowDetail, 0, len(flows))
	for i := range flows {
		steps, err := loadFlowSteps(db, flows[i].ID)
		if err != nil {
			return nil, err
		}
		details = append(details, ApprovalFlowDetail{ApprovalFlow: flows[i], Steps: steps})
	}
	return details, nil
}

// UpdateApprovalFlow 更新指定编码的审批流程：校验步骤合法性，事务内删除旧步骤 + 批量插入新步骤。
// 只影响新申请，已存在的 ApprovalInstance / ApprovalTask 不受影响；返回更新后的流程详情。
func UpdateApprovalFlow(db *gorm.DB, code string, req UpdateApprovalFlowRequest) (*ApprovalFlowDetail, error) {
	if !validApprovalFlowCodes[code] {
		return nil, ErrApprovalFlowCodeInvalid
	}
	if err := validateApprovalFlowSteps(db, req); err != nil {
		return nil, err
	}
	var flow models.ApprovalFlow
	if err := db.Where("code = ?", code).First(&flow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalFlowNotFound
		}
		return nil, err
	}
	if req.Name != "" {
		flow.Name = req.Name
		if err := db.Model(&models.ApprovalFlow{}).Where("id = ?", flow.ID).
			Update("name", req.Name).Error; err != nil {
			return nil, err
		}
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("flow_id = ?", flow.ID).Delete(&models.ApprovalStep{}).Error; err != nil {
			return err
		}
		for _, st := range req.Steps {
			step := models.ApprovalStep{
				FlowID:             flow.ID,
				StepOrder:          st.StepOrder,
				ApproverType:       st.ApproverType,
				ApproverRef:        st.ApproverRef,
				ApproverPermission: st.ApproverPermission,
				ApproverRoleID:     st.ApproverRoleID,
				Name:               st.Name,
			}
			if err := tx.Create(&step).Error; err != nil {
				return fmt.Errorf("创建审批流程步骤失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return getApprovalFlowDetail(db, flow.ID)
}

// ApprovalPermissionRole 拥有审批权限点的角色
type ApprovalPermissionRole struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ApprovalPermissionInfo 审批权限点及拥有它的角色列表（流程配置页权限池下拉展示）
type ApprovalPermissionInfo struct {
	Code  string                   `json:"code"`
	Name  string                   `json:"name"`
	Roles []ApprovalPermissionRole `json:"roles"`
}

// ListApprovalPermissions 查询审批权限点及拥有它的角色列表（GET /approval-flows 顶层 approval_permissions 字段）
// 角色列表 = 拥有该权限点的所有角色（role_permissions JOIN roles），供前端显示哪些角色可审批
func ListApprovalPermissions(db *gorm.DB) ([]ApprovalPermissionInfo, error) {
	names := map[string]string{
		"leave:approve":                 "请假审批",
		"attendance_correction:approve": "补卡审批",
	}
	infos := make([]ApprovalPermissionInfo, 0, len(validApprovalPermissions))
	// 固定顺序返回，保证前端展示稳定
	for _, code := range []string{"leave:approve", "attendance_correction:approve"} {
		var perm models.Permission
		if err := db.Where("code = ?", code).First(&perm).Error; err != nil {
			// 权限点未 seed：兜底展示空角色列表，不阻断流程列表
			infos = append(infos, ApprovalPermissionInfo{Code: code, Name: names[code], Roles: []ApprovalPermissionRole{}})
			continue
		}
		var roles []struct {
			ID   uint
			Name string
		}
		if err := db.Table("roles").
			Joins("JOIN role_permissions ON role_permissions.role_id = roles.id").
			Where("role_permissions.permission_id = ?", perm.ID).
			Order("roles.id ASC").
			Scan(&roles).Error; err != nil {
			return nil, err
		}
		permRoles := make([]ApprovalPermissionRole, 0, len(roles))
		for _, r := range roles {
			permRoles = append(permRoles, ApprovalPermissionRole{ID: r.ID, Name: r.Name})
		}
		infos = append(infos, ApprovalPermissionInfo{Code: code, Name: perm.Name, Roles: permRoles})
	}
	return infos, nil
}

// loadFlowSteps 查询流程步骤（JOIN roles 带出审批角色名称），按 step_order 升序
func loadFlowSteps(db *gorm.DB, flowID uint) ([]ApprovalStepDetail, error) {
	steps := make([]ApprovalStepDetail, 0)
	err := db.Table("approval_steps").
		Select(`approval_steps.*, COALESCE(r.name, '') AS approver_role_name`).
		Joins("LEFT JOIN roles r ON r.id = approval_steps.approver_role_id").
		Where("approval_steps.flow_id = ?", flowID).
		Order("approval_steps.step_order ASC").
		Scan(&steps).Error
	if err != nil {
		return nil, err
	}
	return steps, nil
}

// getApprovalFlowDetail 查询单个流程详情（含步骤）
func getApprovalFlowDetail(db *gorm.DB, flowID uint) (*ApprovalFlowDetail, error) {
	var flow models.ApprovalFlow
	if err := db.First(&flow, flowID).Error; err != nil {
		return nil, err
	}
	steps, err := loadFlowSteps(db, flowID)
	if err != nil {
		return nil, err
	}
	return &ApprovalFlowDetail{ApprovalFlow: flow, Steps: steps}, nil
}

// validateApprovalFlowSteps 校验步骤配置：
// 1~5 级；step_order 从 1 开始且连续；approver_type ∈ {manager, role, permission, hr, position}（hr/position/permission 兼容旧配置）；
// role 时 approver_role_id>0 且角色存在、approver_permission 为空、approver_ref 为 0；
// permission 时 approver_permission 必须为合法审批权限点、approver_ref/approver_role_id 为 0；
// position 时 approver_ref>0 且岗位存在、approver_permission/approver_role_id 为空/0；
// manager/hr 时 approver_ref/approver_permission/approver_role_id 均为空/0
func validateApprovalFlowSteps(db *gorm.DB, req UpdateApprovalFlowRequest) error {
	n := len(req.Steps)
	if n < 1 || n > 5 {
		return ErrApprovalFlowStepCount
	}
	for i, st := range req.Steps {
		if st.StepOrder != i+1 {
			return ErrApprovalFlowStepOrder
		}
		if !validApproverTypes[st.ApproverType] {
			return ErrApprovalFlowTypeInvalid
		}
		switch st.ApproverType {
		case "role":
			if st.ApproverRoleID == 0 {
				return ErrApprovalFlowRoleBad
			}
			var role models.Role
			if err := db.First(&role, st.ApproverRoleID).Error; err != nil {
				return ErrApprovalFlowRoleBad
			}
			if st.ApproverPermission != "" {
				return ErrApprovalFlowPermBad
			}
			if st.ApproverRef != 0 {
				return ErrApprovalFlowRefConflict
			}
		case "permission":
			if !validApprovalPermissions[st.ApproverPermission] {
				return ErrApprovalFlowPermBad
			}
			if st.ApproverRef != 0 {
				return ErrApprovalFlowRefConflict
			}
			if st.ApproverRoleID != 0 {
				return ErrApprovalFlowRoleBad
			}
		case "position":
			if st.ApproverRef == 0 {
				return ErrApprovalFlowPosRefBad
			}
			var pos models.Position
			if err := db.First(&pos, st.ApproverRef).Error; err != nil {
				return ErrApprovalFlowPosRefBad
			}
			if st.ApproverPermission != "" {
				return ErrApprovalFlowPermBad
			}
			if st.ApproverRoleID != 0 {
				return ErrApprovalFlowRoleBad
			}
		default:
			if st.ApproverRef != 0 {
				return ErrApprovalFlowRefConflict
			}
			if st.ApproverPermission != "" {
				return ErrApprovalFlowPermBad
			}
			if st.ApproverRoleID != 0 {
				return ErrApprovalFlowRoleBad
			}
		}
	}
	return nil
}
