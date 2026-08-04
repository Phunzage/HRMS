package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Leave 请假处理器
type Leave struct {
	service *service.LeaveService
}

// NewLeave 创建请假处理器
func NewLeave(svc *service.LeaveService) *Leave {
	return &Leave{service: svc}
}

// rejectLeaveRequest 驳回请假请求体
type rejectLeaveRequest struct {
	RejectReason string `json:"reject_reason" binding:"required"`
}

// Create 提交请假申请
func (h *Leave) Create(c *gin.Context) {
	var req service.SubmitLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	leave, err := h.service.Create(middleware.GetUserID(c), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLeaveEmpNoUser),
			errors.Is(err, service.ErrLeaveTypeInvalid),
			errors.Is(err, service.ErrLeaveDateBad),
			errors.Is(err, service.ErrLeaveDateOrder),
			errors.Is(err, service.ErrLeaveReasonRequired):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			var balErr *service.BalanceInsufficientError
			if errors.As(err, &balErr) {
				response.Error(c, http.StatusBadRequest, 400, err.Error())
				return
			}
			response.Error(c, http.StatusInternalServerError, 500, "提交请假申请失败")
		}
		return
	}
	response.OK(c, leave)
}

// List 请假分页列表（按角色范围过滤）
func (h *Leave) List(c *gin.Context) {
	status := c.Query("status")
	employeeID := parseUintQuery(c, "employee_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(middleware.GetRoleCode(c), middleware.GetUserID(c), status, employeeID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取请假列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Mine 当前用户自己的申请列表（含余额信息）
func (h *Leave) Mine(c *gin.Context) {
	page, pageSize := response.Pagination(c)
	result, err := h.service.Mine(middleware.GetUserID(c), page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrLeaveEmpNoUser) {
			response.Error(c, http.StatusNotFound, 404, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "获取我的请假列表失败")
		return
	}
	response.OK(c, result)
}

// Balance 当前用户各类型假期余额
func (h *Leave) Balance(c *gin.Context) {
	items, err := h.service.Balance(middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, service.ErrLeaveEmpNoUser) {
			response.Error(c, http.StatusNotFound, 404, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "获取假期余额失败")
		return
	}
	response.OK(c, items)
}

// Approve 审批通过
func (h *Leave) Approve(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Approve(middleware.GetRoleCode(c), middleware.GetUserID(c), id); err != nil {
		h.handleActionError(c, err)
		return
	}
	response.OK(c, nil)
}

// Reject 驳回申请
func (h *Leave) Reject(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req rejectLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Reject(middleware.GetRoleCode(c), middleware.GetUserID(c), id, req.RejectReason); err != nil {
		h.handleActionError(c, err)
		return
	}
	response.OK(c, nil)
}

// Cancel 申请人取消 pending 申请
func (h *Leave) Cancel(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Cancel(middleware.GetUserID(c), id); err != nil {
		h.handleActionError(c, err)
		return
	}
	response.OK(c, nil)
}

// handleActionError 审批/取消类动作的通用错误映射
func (h *Leave) handleActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLeaveNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrLeaveEmpNoUser):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrLeaveNotApprover),
		errors.Is(err, service.ErrLeaveNotOwner):
		response.Error(c, http.StatusForbidden, 403, err.Error())
	case errors.Is(err, service.ErrLeaveNotPending):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
