package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Position 岗位处理器
type Position struct {
	service *service.PositionService
}

// NewPosition 创建岗位处理器
func NewPosition(svc *service.PositionService) *Position {
	return &Position{service: svc}
}

// List 岗位分页列表
func (h *Position) List(c *gin.Context) {
	keyword := c.Query("keyword")
	departmentID := parseUintQuery(c, "department_id")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(keyword, departmentID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取岗位列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Create 创建岗位
func (h *Position) Create(c *gin.Context) {
	var req service.CreatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	pos, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPosNameEmpty),
			errors.Is(err, service.ErrPosCodeEmpty),
			errors.Is(err, service.ErrPosDeptNotFound),
			errors.Is(err, service.ErrPosCodeExists):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "创建岗位失败")
		}
		return
	}
	response.OK(c, pos)
}

// Update 更新岗位
func (h *Position) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, req); err != nil {
		switch {
		case errors.Is(err, service.ErrPosNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrPosCodeExists),
			errors.Is(err, service.ErrPosDeptNotFound):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新岗位失败")
		}
		return
	}
	response.OK(c, nil)
}

// Delete 删除岗位
func (h *Position) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrPosNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrPosInUse):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除岗位失败")
		}
		return
	}
	response.OK(c, nil)
}
