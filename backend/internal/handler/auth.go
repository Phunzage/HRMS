package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Auth 认证处理器
type Auth struct {
	service *service.AuthService
}

// NewAuth 创建认证处理器
func NewAuth(svc *service.AuthService) *Auth {
	return &Auth{service: svc}
}

// loginRequest 登录请求体
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// changePasswordRequest 修改密码请求体
type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Login 登录接口：校验用户名密码并签发 JWT
func (h *Auth) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	result, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			response.Error(c, http.StatusUnauthorized, 401, "用户名或密码错误")
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "登录失败，请稍后重试")
		return
	}
	response.OK(c, result)
}

// Me 当前用户信息接口（需登录）
func (h *Auth) Me(c *gin.Context) {
	result, err := h.service.Me(middleware.GetUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取用户信息失败")
		return
	}
	response.OK(c, result)
}

// ChangePassword 修改密码接口（需登录）
func (h *Auth) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.ChangePassword(middleware.GetUserID(c), req.OldPassword, req.NewPassword); err != nil {
		switch err {
		case service.ErrUserNotFound:
			response.Error(c, http.StatusUnauthorized, 401, "用户不存在")
		case service.ErrOldPasswordWrong:
			response.Error(c, http.StatusBadRequest, 400, "旧密码不正确")
		default:
			response.Error(c, http.StatusInternalServerError, 500, "修改密码失败，请稍后重试")
		}
		return
	}
	response.OK(c, nil)
}
