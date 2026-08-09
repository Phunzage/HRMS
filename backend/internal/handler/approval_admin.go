package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// ApprovalAdmin 审批流程配置处理器（admin 专属，approval_flow:read/write 权限点）
type ApprovalAdmin struct {
	db *gorm.DB
}

// NewApprovalAdmin 创建审批流程配置处理器
func NewApprovalAdmin(db *gorm.DB) *ApprovalAdmin {
	return &ApprovalAdmin{db: db}
}

// List 查询所有审批流程及步骤（GET /approval-flows）
func (h *ApprovalAdmin) List(c *gin.Context) {
	list, err := service.ListApprovalFlows(h.db)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "查询审批流程失败")
		return
	}
	response.OK(c, gin.H{"list": list})
}

// Update 更新指定编码的审批流程（PUT /approval-flows/:code）
func (h *ApprovalAdmin) Update(c *gin.Context) {
	code := c.Param("code")
	var req service.UpdateApprovalFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	detail, err := service.UpdateApprovalFlow(h.db, code, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	middleware.Record(h.db, c, "approval_flow", "update", code, fmt.Sprintf("更新审批流程(%s)", code))
	response.OK(c, detail)
}

// handleError 审批流程配置错误映射
func (h *ApprovalAdmin) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrApprovalFlowNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrApprovalFlowCodeInvalid),
		errors.Is(err, service.ErrApprovalFlowStepCount),
		errors.Is(err, service.ErrApprovalFlowStepOrder),
		errors.Is(err, service.ErrApprovalFlowTypeInvalid),
		errors.Is(err, service.ErrApprovalFlowPosRefBad),
		errors.Is(err, service.ErrApprovalFlowRefConflict):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "更新审批流程失败")
	}
}
