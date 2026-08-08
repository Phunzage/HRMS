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

// Performance 绩效处理器
type Performance struct {
	service *service.PerformanceService
	db      *gorm.DB
}

// NewPerformance 创建绩效处理器
func NewPerformance(svc *service.PerformanceService, db *gorm.DB) *Performance {
	return &Performance{service: svc, db: db}
}

// List 绩效分页列表（manager 仅本部门；employee 仅自己）
func (h *Performance) List(c *gin.Context) {
	period := c.Query("period")
	status := c.Query("status")
	employeeID := parseUintQuery(c, "employee_id")
	departmentID := parseUintQuery(c, "department_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(middleware.GetRoleCode(c), middleware.GetUserID(c), period, status, employeeID, departmentID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取绩效列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Mine 当前用户自己的绩效记录（登录即可，含评级中文名）
func (h *Performance) Mine(c *gin.Context) {
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.Mine(middleware.GetUserID(c), page, pageSize)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Create 创建绩效
func (h *Performance) Create(c *gin.Context) {
	var req service.CreatePerformanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	record, err := h.service.Create(middleware.GetRoleCode(c), middleware.GetUserID(c), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "performance", "create", fmt.Sprintf("performance:%d", record.ID), fmt.Sprintf("创建绩效 员工ID:%d 周期:%s 分数:%.1f", record.EmployeeID, record.Period, record.Score))
	response.OK(c, record)
}

// Update 更新绩效（仅 draft 可改）
func (h *Performance) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdatePerformanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	record, err := h.service.Update(id, req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "performance", "update", fmt.Sprintf("performance:%d", record.ID), fmt.Sprintf("更新绩效 员工ID:%d 周期:%s 分数:%.1f", record.EmployeeID, record.Period, record.Score))
	response.OK(c, record)
}

// Submit 提交绩效（draft → submitted）
func (h *Performance) Submit(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Submit(id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "performance", "submit", fmt.Sprintf("performance:%d", id), fmt.Sprintf("提交绩效(ID:%d)", id))
	response.OK(c, nil)
}

// Approve 批准绩效（submitted → approved，仅 hr/admin）
func (h *Performance) Approve(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Approve(middleware.GetRoleCode(c), id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "performance", "approve", fmt.Sprintf("performance:%d", id), fmt.Sprintf("批准绩效(ID:%d)", id))
	response.OK(c, nil)
}

// Delete 删除绩效（仅 draft 可删）
func (h *Performance) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Delete(id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "performance", "delete", fmt.Sprintf("performance:%d", id), fmt.Sprintf("删除绩效(ID:%d)", id))
	response.OK(c, nil)
}

// handleErr 绩效模块错误映射
func (h *Performance) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPerformanceNotApprover):
		response.Error(c, http.StatusForbidden, 403, err.Error())
	case errors.Is(err, service.ErrPerformancePeriodInvalid),
		errors.Is(err, service.ErrPerformanceScoreInvalid),
		errors.Is(err, service.ErrPerformanceAlreadyExists),
		errors.Is(err, service.ErrPerformanceNotDraft),
		errors.Is(err, service.ErrPerformanceNotSubmitted),
		errors.Is(err, service.ErrPerformanceNotInDept),
		errors.Is(err, service.ErrPerformanceEmpNotFound):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrPerformanceEmpNoUser),
		errors.Is(err, service.ErrPerformanceNotFound):
		// 自助读取（mine）时账号未关联员工档案按 404 处理，与其他模块自助接口一致
		response.Error(c, http.StatusNotFound, 404, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
