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
	ErrApprovalFlowTypeInvalid = errors.New("审批步骤类型不合法（manager/hr/position）")
	ErrApprovalFlowPosRefBad   = errors.New("指定岗位审批必须填写有效的岗位 ID")
	ErrApprovalFlowRefConflict = errors.New("manager/hr 步骤不得填写岗位引用")
)

// validApprovalFlowCodes 可配置的流程编码（与业务申请 bizType 对应）
var validApprovalFlowCodes = map[string]bool{
	"leave":                 true,
	"attendance_correction": true,
}

// validApproverTypes 审批步骤合法审批人类型
var validApproverTypes = map[string]bool{
	"manager":  true,
	"hr":       true,
	"position": true,
}

// ApprovalFlowStepInput 流程步骤配置输入
type ApprovalFlowStepInput struct {
	StepOrder    int    `json:"step_order"`
	ApproverType string `json:"approver_type"`
	ApproverRef  uint   `json:"approver_ref"`
	Name         string `json:"name"`
}

// UpdateApprovalFlowRequest 更新审批流程请求体
type UpdateApprovalFlowRequest struct {
	Name  string                  `json:"name"`
	Steps []ApprovalFlowStepInput `json:"steps"`
}

// ApprovalFlowDetail 审批流程详情（含步骤，按 step_order 升序）
type ApprovalFlowDetail struct {
	models.ApprovalFlow
	Steps []models.ApprovalStep `json:"steps"`
}

// ListApprovalFlows 查询所有审批流程及步骤（admin 流程配置页用）
func ListApprovalFlows(db *gorm.DB) ([]ApprovalFlowDetail, error) {
	var flows []models.ApprovalFlow
	if err := db.Order("id ASC").Find(&flows).Error; err != nil {
		return nil, err
	}
	details := make([]ApprovalFlowDetail, 0, len(flows))
	for i := range flows {
		var steps []models.ApprovalStep
		if err := db.Where("flow_id = ?", flows[i].ID).Order("step_order ASC").Find(&steps).Error; err != nil {
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
				FlowID:       flow.ID,
				StepOrder:    st.StepOrder,
				ApproverType: st.ApproverType,
				ApproverRef:  st.ApproverRef,
				Name:         st.Name,
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

// getApprovalFlowDetail 查询单个流程详情（含步骤）
func getApprovalFlowDetail(db *gorm.DB, flowID uint) (*ApprovalFlowDetail, error) {
	var flow models.ApprovalFlow
	if err := db.First(&flow, flowID).Error; err != nil {
		return nil, err
	}
	var steps []models.ApprovalStep
	if err := db.Where("flow_id = ?", flowID).Order("step_order ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	return &ApprovalFlowDetail{ApprovalFlow: flow, Steps: steps}, nil
}

// validateApprovalFlowSteps 校验步骤配置：
// 1~5 级；step_order 从 1 开始且连续；approver_type ∈ {manager, hr, position}；
// position 时 approver_ref>0 且岗位存在；hr/manager 时 approver_ref 必须为 0
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
		case "position":
			if st.ApproverRef == 0 {
				return ErrApprovalFlowPosRefBad
			}
			var pos models.Position
			if err := db.First(&pos, st.ApproverRef).Error; err != nil {
				return ErrApprovalFlowPosRefBad
			}
		default:
			if st.ApproverRef != 0 {
				return ErrApprovalFlowRefConflict
			}
		}
	}
	return nil
}
