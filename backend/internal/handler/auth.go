package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Auth 认证处理器
type Auth struct {
	service *service.AuthService
	db      *gorm.DB
}

// NewAuth 创建认证处理器
func NewAuth(svc *service.AuthService, db *gorm.DB) *Auth {
	return &Auth{service: svc, db: db}
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
			// 登录失败审计（此时无登录用户，UserID=0、Username=system）
			middleware.Record(h.db, c, "auth", "login", "", fmt.Sprintf("登录失败：用户名 %s", req.Username))
			response.Error(c, http.StatusUnauthorized, 401, "用户名或密码错误")
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "登录失败，请稍后重试")
		return
	}
	// 登录成功审计：将登录用户写入上下文供 Record 读取
	c.Set(middleware.CtxUserID, result.User.ID)
	c.Set(middleware.CtxUsername, result.User.Username)
	middleware.Record(h.db, c, "auth", "login", fmt.Sprintf("user:%d", result.User.ID), fmt.Sprintf("登录成功：%s", result.User.Username))
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
	middleware.Record(h.db, c, "auth", "reset_password", fmt.Sprintf("user:%d", middleware.GetUserID(c)), "修改自己的密码")
	response.OK(c, nil)
}
