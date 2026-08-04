package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Department 部门处理器
type Department struct {
	service *service.DepartmentService
}

// NewDepartment 创建部门处理器
func NewDepartment(svc *service.DepartmentService) *Department {
	return &Department{service: svc}
}

// List 部门树形列表
func (h *Department) List(c *gin.Context) {
	tree, err := h.service.ListTree()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取部门列表失败")
		return
	}
	response.OK(c, tree)
}

// Create 创建部门
func (h *Department) Create(c *gin.Context) {
	var req service.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	dept, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeptNameEmpty):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		case errors.Is(err, service.ErrDeptParentMissing):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "创建部门失败")
		}
		return
	}
	response.OK(c, dept)
}

// Update 更新部门
func (h *Department) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, req); err != nil {
		switch {
		case errors.Is(err, service.ErrDeptNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrDeptParentMissing),
			errors.Is(err, service.ErrDeptCycle):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新部门失败")
		}
		return
	}
	response.OK(c, nil)
}

// Delete 删除部门
func (h *Department) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrDeptNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrDeptHasChildren),
			errors.Is(err, service.ErrDeptHasEmployees):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除部门失败")
		}
		return
	}
	response.OK(c, nil)
}
