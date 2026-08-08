package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/models"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Approval 通用审批处理器
// 权限靠任务归属校验（ApproverID==当前用户，或 hr 类型步骤由 hr/admin 审批），无独立权限点
type Approval struct {
	db        *gorm.DB
	callbacks map[string]service.ApprovalBizCallback // bizType → 终审回调
}

// NewApproval 创建通用审批处理器
func NewApproval(db *gorm.DB, callbacks map[string]service.ApprovalBizCallback) *Approval {
	return &Approval{db: db, callbacks: callbacks}
}

// approveRequest 审批通过请求体（comment 可选）
type approveRequest struct {
	Comment string `json:"comment"`
}

// rejectRequest 驳回请求体（comment 必填）
type rejectRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// Mine 我的待审批/历史任务列表（manager 与 hr 都从这里看待办）
func (h *Approval) Mine(c *gin.Context) {
	items, err := service.ListMyTasks(h.db, middleware.GetUserID(c), c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取我的审批任务失败")
		return
	}
	response.OK(c, items)
}

// InstanceTasks 某实例的审批进度时间线（步骤名/审批人/状态/时间/意见）
func (h *Approval) InstanceTasks(c *gin.Context) {
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if _, err := service.GetInstanceByID(h.db, instanceID); err != nil {
		h.handleError(c, err)
		return
	}
	tasks, err := service.ListInstanceTasks(h.db, instanceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "查询审批进度失败")
		return
	}
	response.OK(c, tasks)
}

// Biz 按业务类型+业务 ID 查询审批实例与进度（前端详情页展示）
func (h *Approval) Biz(c *gin.Context) {
	bizType := c.Query("biz_type")
	bizID := parseUintQuery(c, "biz_id")
	if bizType == "" || bizID == 0 {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	inst, err := service.GetInstanceByBiz(h.db, bizType, bizID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, 404, "审批实例不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "查询审批实例失败")
		return
	}
	tasks, err := service.ListInstanceTasks(h.db, inst.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "查询审批进度失败")
		return
	}
	response.OK(c, service.BizDetail{Instance: inst, Tasks: tasks})
}

// Approve 审批通过：事务内推进审批引擎，final 时同事务执行业务回调
func (h *Approval) Approve(c *gin.Context) {
	taskID, err := parseUintParam(c, "taskId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	userID := middleware.GetUserID(c)
	var inst *models.ApprovalInstance
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var e error
		inst, e = service.ApproveTask(tx, taskID, userID, req.Comment)
		if e != nil {
			return e
		}
		// 终审通过：同事务执行业务回调（更新状态/扣余额/联动考勤/通知）
		if inst.Status == "approved" {
			cb, ok := h.callbacks[inst.BizType]
			if !ok {
				return fmt.Errorf("未注册的审批终审回调: %s", inst.BizType)
			}
			return cb.ApplyApproved(tx, inst.BizID)
		}
		return nil
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	middleware.Record(h.db, c, "approval", "approve", fmt.Sprintf("task:%d", taskID), fmt.Sprintf("审批通过(ID:%d, %s)", taskID, inst.BizType))
	response.OK(c, inst)
}

// Reject 驳回：事务内任务→rejected、实例→rejected，同事务执行业务回调
func (h *Approval) Reject(c *gin.Context) {
	taskID, err := parseUintParam(c, "taskId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req rejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "驳回意见不能为空")
		return
	}
	userID := middleware.GetUserID(c)
	var bizType string
	var bizID uint
	err = h.db.Transaction(func(tx *gorm.DB) error {
		inst, e := service.GetInstanceByTask(tx, taskID)
		if e != nil {
			return e
		}
		bizType, bizID = inst.BizType, inst.BizID
		if e := service.RejectTask(tx, taskID, userID, req.Comment); e != nil {
			return e
		}
		cb, ok := h.callbacks[inst.BizType]
		if !ok {
			return fmt.Errorf("未注册的审批终审回调: %s", inst.BizType)
		}
		return cb.ApplyRejected(tx, inst.BizID, req.Comment)
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	middleware.Record(h.db, c, "approval", "reject", fmt.Sprintf("task:%d", taskID), fmt.Sprintf("驳回审批(ID:%d, %s:%d)", taskID, bizType, bizID))
	response.OK(c, nil)
}

// handleError 通用审批动作错误映射
func (h *Approval) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrApprovalInstanceNotFound),
		errors.Is(err, service.ErrApprovalTaskNotFound),
		errors.Is(err, service.ErrLeaveNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	case errors.Is(err, service.ErrApprovalTaskNotPending),
		errors.Is(err, service.ErrApprovalNotPending),
		errors.Is(err, service.ErrLeaveNotPending):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrApprovalNotAllowed):
		response.Error(c, http.StatusForbidden, 403, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "审批操作失败")
	}
}
