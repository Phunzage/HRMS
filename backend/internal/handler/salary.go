package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Salary 工资处理器
type Salary struct {
	service *service.SalaryService
}

// NewSalary 创建工资处理器
func NewSalary(svc *service.SalaryService) *Salary {
	return &Salary{service: svc}
}

// List 工资单分页列表（employee 角色仅看自己）
func (h *Salary) List(c *gin.Context) {
	month := c.Query("month")
	status := c.Query("status")
	employeeID := parseUintQuery(c, "employee_id")
	departmentID := parseUintQuery(c, "department_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(middleware.GetRoleCode(c), middleware.GetUserID(c), month, employeeID, status, departmentID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取工资单列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Create 创建工资单
func (h *Salary) Create(c *gin.Context) {
	var req service.CreateSalaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	record, err := h.service.Create(req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.OK(c, record)
}

// Update 更新工资单（仅 draft 状态可改，改后重算实发工资）
func (h *Salary) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateSalaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	record, err := h.service.Update(id, req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.OK(c, record)
}

// Confirm 确认工资单（draft → confirmed）
func (h *Salary) Confirm(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Confirm(id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Pay 发放工资（confirmed → paid）
func (h *Salary) Pay(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Pay(id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Mine 员工查看自己的工资单（登录即可）
func (h *Salary) Mine(c *gin.Context) {
	month := c.Query("month")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.Mine(middleware.GetUserID(c), month, page, pageSize)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// handleErr 工资模块错误映射
func (h *Salary) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSalaryMonthInvalid),
		errors.Is(err, service.ErrSalaryEmpNotFound),
		errors.Is(err, service.ErrSalaryAlreadyExists),
		errors.Is(err, service.ErrSalaryNotDraft),
		errors.Is(err, service.ErrSalaryNotConfirmed):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrSalaryEmpNoUser),
		errors.Is(err, service.ErrSalaryNotFound):
		// 自助读取（mine/list）时账号未关联员工档案按 404 处理，与其他模块自助接口一致
		response.Error(c, http.StatusNotFound, 404, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
