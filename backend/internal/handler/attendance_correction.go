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

// AttendanceCorrection 补卡申请处理器
type AttendanceCorrection struct {
	service *service.AttendanceCorrectionService
	db      *gorm.DB
}

// NewAttendanceCorrection 创建补卡申请处理器
func NewAttendanceCorrection(svc *service.AttendanceCorrectionService, db *gorm.DB) *AttendanceCorrection {
	return &AttendanceCorrection{service: svc, db: db}
}

// rejectCorrectionRequest 驳回补卡请求体
type rejectCorrectionRequest struct {
	RejectReason string `json:"reject_reason" binding:"required"`
}

// Create 员工提交补卡申请
func (h *AttendanceCorrection) Create(c *gin.Context) {
	var req service.SubmitCorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	cor, err := h.service.Create(middleware.GetUserID(c), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCorrEmpNoUser),
			errors.Is(err, service.ErrCorrTypeBad),
			errors.Is(err, service.ErrCorrDateBad),
			errors.Is(err, service.ErrCorrDateTooOld),
			errors.Is(err, service.ErrCorrDateFuture),
			errors.Is(err, service.ErrCorrTimeBad),
			errors.Is(err, service.ErrCorrReasonRequired):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "提交补卡申请失败")
		}
		return
	}
	middleware.Record(h.db, c, "attendance", "create", fmt.Sprintf("correction:%d", cor.ID),
		fmt.Sprintf("提交补卡申请(ID:%d, %s %s)", cor.ID, cor.WorkDate.Format("2006-01-02"), cor.CorrectionType))
	response.OK(c, cor)
}

// List 补卡申请分页列表（按角色范围过滤）
func (h *AttendanceCorrection) List(c *gin.Context) {
	status := c.Query("status")
	employeeID := parseUintQuery(c, "employee_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(middleware.GetRoleCode(c), middleware.GetUserID(c), status, employeeID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取补卡申请列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Mine 当前用户自己的补卡记录（登录即可）
func (h *AttendanceCorrection) Mine(c *gin.Context) {
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.Mine(middleware.GetUserID(c), page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrCorrEmpNoUser) {
			response.Error(c, http.StatusNotFound, 404, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "获取我的补卡记录失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Approve 审批通过补卡申请
func (h *AttendanceCorrection) Approve(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Approve(middleware.GetRoleCode(c), middleware.GetUserID(c), id); err != nil {
		h.handleActionError(c, err)
		return
	}
	middleware.Record(h.db, c, "attendance", "approve", fmt.Sprintf("correction:%d", id), fmt.Sprintf("审批通过补卡(ID:%d)", id))
	response.OK(c, nil)
}

// Reject 驳回补卡申请
func (h *AttendanceCorrection) Reject(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req rejectCorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Reject(middleware.GetRoleCode(c), middleware.GetUserID(c), id, req.RejectReason); err != nil {
		h.handleActionError(c, err)
		return
	}
	middleware.Record(h.db, c, "attendance", "reject", fmt.Sprintf("correction:%d", id), fmt.Sprintf("驳回补卡(ID:%d)", id))
	response.OK(c, nil)
}

// Cancel 申请人取消 pending 补卡申请
func (h *AttendanceCorrection) Cancel(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Cancel(middleware.GetUserID(c), id); err != nil {
		h.handleActionError(c, err)
		return
	}
	middleware.Record(h.db, c, "attendance", "cancel", fmt.Sprintf("correction:%d", id), fmt.Sprintf("取消补卡(ID:%d)", id))
	response.OK(c, nil)
}

// handleActionError 审批/取消类动作的通用错误映射
func (h *AttendanceCorrection) handleActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCorrNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrCorrEmpNoUser):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrCorrNotApprover),
		errors.Is(err, service.ErrCorrNotOwner):
		response.Error(c, http.StatusForbidden, 403, err.Error())
	case errors.Is(err, service.ErrCorrNotPending):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
