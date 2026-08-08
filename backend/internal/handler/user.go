package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// User 用户管理处理器
type User struct {
	service *service.UserService
	db      *gorm.DB
}

// NewUser 创建用户管理处理器
func NewUser(svc *service.UserService, db *gorm.DB) *User {
	return &User{service: svc, db: db}
}

// List 用户分页列表，支持 keyword / role_id / status 过滤
func (h *User) List(c *gin.Context) {
	keyword := c.Query("keyword")
	roleID := parseUintQuery(c, "role_id")
	status := -1 // 默认不过滤；status=0/1 分别过滤禁用/启用
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(keyword, roleID, status, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取用户列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Roles 全部角色列表（供前端下拉，仅需登录）
func (h *User) Roles(c *gin.Context) {
	roles, err := h.service.ListRoles()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取角色列表失败")
		return
	}
	response.OK(c, roles)
}

// Create 创建用户
func (h *User) Create(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	user, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameExists),
			errors.Is(err, service.ErrUsernameRequired),
			errors.Is(err, service.ErrPasswordTooShort),
			errors.Is(err, service.ErrRoleNotFound):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "创建用户失败")
		}
		return
	}
	middleware.Record(h.db, c, "user", "create", fmt.Sprintf("user:%d", user.ID), fmt.Sprintf("创建用户 %s（%s）", user.Username, user.FullName))
	response.OK(c, user)
}

// Update 更新用户基本信息
func (h *User) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	username := h.usernameOf(id)
	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, middleware.GetUserID(c), req); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrRoleNotFound),
			errors.Is(err, service.ErrCannotDisableSelf):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新用户失败")
		}
		return
	}
	middleware.Record(h.db, c, "user", "update", fmt.Sprintf("user:%d", id), fmt.Sprintf("更新用户 %s", username))
	response.OK(c, nil)
}

// ResetPassword 重置用户密码
func (h *User) ResetPassword(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	username := h.usernameOf(id)
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.ResetPassword(id, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrPasswordTooShort):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "重置密码失败")
		}
		return
	}
	middleware.Record(h.db, c, "user", "reset_password", fmt.Sprintf("user:%d", id), fmt.Sprintf("重置用户 %s 的密码", username))
	response.OK(c, nil)
}

// Delete 删除用户（软删除）
func (h *User) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	username := h.usernameOf(id)
	if err := h.service.Delete(id, middleware.GetUserID(c)); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrCannotDeleteSelf):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除用户失败")
		}
		return
	}
	middleware.Record(h.db, c, "user", "delete", fmt.Sprintf("user:%d", id), fmt.Sprintf("删除用户 %s", username))
	response.OK(c, nil)
}

// usernameOf 尽力获取用户名用于审计描述，获取失败时返回空串
func (h *User) usernameOf(id uint) string {
	if u, err := h.service.GetByID(id); err == nil {
		return u.Username
	}
	return ""
}
