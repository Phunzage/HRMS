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

// Employee 员工处理器
type Employee struct {
	service *service.EmployeeService
	db      *gorm.DB
}

// NewEmployee 创建员工处理器
func NewEmployee(svc *service.EmployeeService, db *gorm.DB) *Employee {
	return &Employee{service: svc, db: db}
}

// List 员工分页列表
func (h *Employee) List(c *gin.Context) {
	keyword := c.Query("keyword")
	departmentID := parseUintQuery(c, "department_id")
	status := parseUintQuery(c, "status")
	positionID := parseUintQuery(c, "position_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(keyword, departmentID, int(status), positionID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取员工列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Get 员工详情
func (h *Employee) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	emp, err := h.service.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmpNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "获取员工详情失败")
		}
		return
	}
	response.OK(c, emp)
}

// Me 当前登录用户关联的员工档案（JWT 保护即可，无需额外权限）
func (h *Employee) Me(c *gin.Context) {
	emp, err := h.service.GetMyProfile(middleware.GetUserID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmpNoUser):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrEmpNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "获取员工档案失败")
		}
		return
	}
	response.OK(c, emp)
}

// Create 创建员工
func (h *Employee) Create(c *gin.Context) {
	var req service.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	result, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmpNameRequired),
			errors.Is(err, service.ErrEmpDeptNotFound),
			errors.Is(err, service.ErrEmpPosNotFound),
			errors.Is(err, service.ErrEmpDateBad):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "创建员工失败")
		}
		return
	}
	middleware.Record(h.db, c, "employee", "create", fmt.Sprintf("employee:%d", result.Employee.ID), fmt.Sprintf("创建员工 %s %s", result.Employee.Name, result.Employee.EmpNo))
	response.OK(c, result)
}

// Update 更新员工基本信息
func (h *Employee) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	name := h.empNameOf(id)
	var req service.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, req); err != nil {
		switch {
		case errors.Is(err, service.ErrEmpNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrEmpDeptNotFound),
			errors.Is(err, service.ErrEmpPosNotFound),
			errors.Is(err, service.ErrEmpDateBad):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新员工失败")
		}
		return
	}
	middleware.Record(h.db, c, "employee", "update", fmt.Sprintf("employee:%d", id), fmt.Sprintf("更新员工 %s", name))
	response.OK(c, nil)
}

// Delete 删除员工（软删除，禁用关联账号）
func (h *Employee) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	name := h.empNameOf(id)
	if err := h.service.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrEmpNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除员工失败")
		}
		return
	}
	middleware.Record(h.db, c, "employee", "delete", fmt.Sprintf("employee:%d", id), fmt.Sprintf("删除员工 %s", name))
	response.OK(c, nil)
}

// empNameOf 尽力获取员工姓名用于审计描述，获取失败时返回空串
func (h *Employee) empNameOf(id uint) string {
	if emp, err := h.service.GetByID(id); err == nil {
		return emp.Name
	}
	return ""
}
