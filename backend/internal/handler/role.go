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

// Role 角色管理处理器
type Role struct {
	service *service.RoleService
	db      *gorm.DB
}

// NewRole 创建角色管理处理器
func NewRole(svc *service.RoleService, db *gorm.DB) *Role {
	return &Role{service: svc, db: db}
}

// List 角色列表（含权限点 code 数组）
func (h *Role) List(c *gin.Context) {
	items, err := h.service.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取角色列表失败")
		return
	}
	response.OK(c, items)
}

// Permissions 全部权限点，按模块分组返回
func (h *Role) Permissions(c *gin.Context) {
	groups, err := h.service.PermissionTree()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取权限点列表失败")
		return
	}
	response.OK(c, groups)
}

// Create 创建角色
func (h *Role) Create(c *gin.Context) {
	var req service.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	role, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleCodeEmpty),
			errors.Is(err, service.ErrRoleCodeExists),
			errors.Is(err, service.ErrPermissionInvalid):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "创建角色失败")
		}
		return
	}
	middleware.Record(h.db, c, "role", "create", fmt.Sprintf("role:%d", role.ID), fmt.Sprintf("创建角色 %s", role.Name))
	response.OK(c, role)
}

// Update 更新角色信息与权限（permissions 全量替换）
func (h *Role) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	name := h.nameOf(id)
	var req service.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, req); err != nil {
		switch {
		case errors.Is(err, service.ErrRoleNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrPermissionInvalid):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新角色失败")
		}
		return
	}
	middleware.Record(h.db, c, "role", "update", fmt.Sprintf("role:%d", id), fmt.Sprintf("更新角色 %s", name))
	response.OK(c, nil)
}

// Delete 删除角色：角色下有用户时拒绝，admin 角色不可删
func (h *Role) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	name := h.nameOf(id)
	if err := h.service.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrRoleNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrRoleHasUsers),
			errors.Is(err, service.ErrRoleAdminNotDel):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除角色失败")
		}
		return
	}
	middleware.Record(h.db, c, "role", "delete", fmt.Sprintf("role:%d", id), fmt.Sprintf("删除角色 %s", name))
	response.OK(c, nil)
}

// nameOf 尽力获取角色名称用于审计描述，获取失败时返回空串
func (h *Role) nameOf(id uint) string {
	if r, err := h.service.GetByID(id); err == nil {
		return r.Name
	}
	return ""
}
