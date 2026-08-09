package models

import "time"

// ApprovalFlow 审批流程定义（静态配置）
// Code 唯一，如 "leave" / "attendance_correction"；Status 控制启用/停用
type ApprovalFlow struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:64;uniqueIndex;not null" json:"code"`                  // 流程编码，如 leave / attendance_correction
	Name      string    `gorm:"size:128;not null" json:"name"`                             // 流程名称
	Status    string    `gorm:"size:16;default:active;not null" json:"status"`             // active / inactive
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApprovalStep 审批流程步骤定义
// 同一流程内 step_order 唯一；ApproverType 决定审批人解析方式
// （manager=申请人部门经理递归解析 / role=指定角色池（该角色所有启用用户）/ permission=拥有指定审批权限点的所有启用用户
//  / hr=HR池任意hr / position=指定岗位池该岗位所有在职员工）
// hr / position / permission 为兼容旧配置保留，新配置建议 manager / role
type ApprovalStep struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	FlowID             uint      `gorm:"uniqueIndex:idx_flow_step;not null" json:"flow_id"`  // 所属流程 ID
	StepOrder          int       `gorm:"uniqueIndex:idx_flow_step;not null" json:"step_order"` // 步骤顺序，从 1 开始
	ApproverType       string    `gorm:"size:16;not null" json:"approver_type"`               // manager / role / permission / hr / position
	ApproverRef        uint      `gorm:"default:0" json:"approver_ref"`                       // 审批人引用：approver_type=position 时为岗位 ID；其余为 0
	ApproverPermission string    `gorm:"size:64" json:"approver_permission"`                  // 审批权限点 code（如 leave:approve），approver_type=permission 时使用
	ApproverRoleID     uint      `gorm:"default:0" json:"approver_role_id"`                   // 审批角色 ID，approver_type=role 时使用
	Name               string    `gorm:"size:64;not null" json:"name"`                         // 步骤名，如 部门经理审批 / 人事专员审批 / HR终审
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ApprovalInstance 审批实例（一次业务申请对应一个实例）
// (biz_type, biz_id) 唯一，保证一个业务记录只有一个审批流
type ApprovalInstance struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	FlowID      uint      `gorm:"index;not null" json:"flow_id"`               // 所属流程 ID
	BizType     string    `gorm:"size:64;uniqueIndex:idx_biz;not null" json:"biz_type"` // 业务类型：leave / attendance_correction
	BizID       uint      `gorm:"uniqueIndex:idx_biz;not null" json:"biz_id"`           // 业务记录 ID
	Status      string    `gorm:"size:16;default:pending;not null" json:"status"`      // pending / approved / rejected / cancelled
	CurrentStep int       `gorm:"default:1;not null" json:"current_step"`              // 当前进行到的步骤顺序
	CreatedBy   uint      `gorm:"index;not null" json:"created_by"`                    // 提交人（用户 ID）
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ApprovalTask 审批任务（每个审批人一条）
// ApproverID 为审批人用户 ID；HR/角色/权限/岗位池步骤可为空（表示由池内任一审批人处理）
// ApproverRef 在 position 池任务中存岗位 ID（用于匹配该岗位在职员工），manager/hr 为 0
// ApproverPermission 在 permission 池任务中存审批权限点 code（用于匹配拥有该权限的用户）
// ApproverRoleID 在 role 池任务中存角色 ID（用于匹配该角色的用户）
type ApprovalTask struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	InstanceID         uint       `gorm:"index;not null" json:"instance_id"`   // 所属实例 ID
	StepOrder          int        `gorm:"not null" json:"step_order"`          // 步骤顺序
	ApproverType       string     `gorm:"size:16;not null" json:"approver_type"` // manager / role / permission / hr / position
	ApproverID         *uint      `gorm:"index" json:"approver_id"`            // 审批人用户 ID，池步骤可为空
	ApproverRef        uint       `gorm:"default:0" json:"approver_ref"`       // position 池任务存岗位 ID；其余为 0
	ApproverPermission string     `gorm:"size:64" json:"approver_permission"`  // permission 池任务存审批权限点 code；其余为空
	ApproverRoleID     uint       `gorm:"default:0" json:"approver_role_id"`   // role 池任务存角色 ID；其余为 0
	Status             string     `gorm:"size:16;default:pending;not null" json:"status"` // pending / approved / rejected
	Comment            string     `gorm:"size:500" json:"comment"`             // 审批意见
	ActedAt            *time.Time `json:"acted_at"`                            // 处理时间
	CreatedAt          time.Time  `json:"created_at"`
}
