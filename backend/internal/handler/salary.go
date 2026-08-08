package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Salary 工资处理器
type Salary struct {
	service *service.SalaryService
	db      *gorm.DB
}

// NewSalary 创建工资处理器
func NewSalary(svc *service.SalaryService, db *gorm.DB) *Salary {
	return &Salary{service: svc, db: db}
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

// Export 导出工资表 Excel（附件下载，不走统一 JSON 响应）
func (h *Salary) Export(c *gin.Context) {
	month := c.Query("month")
	status := c.Query("status")
	departmentID := parseUintQuery(c, "department_id")
	headers, rows, err := h.service.ExportRows(middleware.GetRoleCode(c), middleware.GetUserID(c), month, status, departmentID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	filename := fmt.Sprintf("salaries_%s.xlsx", time.Now().Format("20060102_1504"))
	data, err := service.WriteExcel(filename, "工资表", headers, rows)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "生成 Excel 文件失败")
		return
	}
	writeXLSX(c, filename, data)
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
	middleware.Record(h.db, c, "salary", "create", fmt.Sprintf("salary:%d", record.ID), fmt.Sprintf("创建工资单 员工ID:%d 月份:%s", record.EmployeeID, record.Month))
	response.OK(c, record)
}

// BatchGenerate 批量生成某月工资单：为当月尚无工资单的在职员工生成 draft 工资单
func (h *Salary) BatchGenerate(c *gin.Context) {
	var req struct {
		Month string `json:"month"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	result, err := h.service.BatchGenerate(req.Month)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "salary", "batch_generate", "", fmt.Sprintf("批量生成 %s 工资单 %d 张", req.Month, result.Generated))
	response.OK(c, result)
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
	middleware.Record(h.db, c, "salary", "update", fmt.Sprintf("salary:%d", record.ID), fmt.Sprintf("更新工资单 员工ID:%d 月份:%s", record.EmployeeID, record.Month))
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
	middleware.Record(h.db, c, "salary", "confirm", fmt.Sprintf("salary:%d", id), fmt.Sprintf("确认工资单(ID:%d)", id))
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
	middleware.Record(h.db, c, "salary", "pay", fmt.Sprintf("salary:%d", id), fmt.Sprintf("发放工资(ID:%d)", id))
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
