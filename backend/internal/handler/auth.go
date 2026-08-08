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
	service        *service.AuthService
	captchaService *service.CaptchaService
	db             *gorm.DB
}

// NewAuth 创建认证处理器
func NewAuth(svc *service.AuthService, captcha *service.CaptchaService, db *gorm.DB) *Auth {
	return &Auth{service: svc, captchaService: captcha, db: db}
}

// loginRequest 登录请求体（验证码必填）
type loginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

// refreshRequest 刷新令牌请求体
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// changePasswordRequest 修改密码请求体
type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Captcha 获取图形验证码接口（公开，无 JWT）
func (h *Auth) Captcha(c *gin.Context) {
	result, err := h.captchaService.Generate()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "验证码生成失败")
		return
	}
	response.OK(c, result)
}

// Refresh 刷新令牌接口（公开）：用 refresh_token 换取新的 token 对
func (h *Auth) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	result, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidRefreshToken:
			response.Error(c, http.StatusUnauthorized, 401, "刷新令牌无效或已过期")
		case service.ErrUserNotFound:
			response.Error(c, http.StatusUnauthorized, 401, "用户不存在")
		case service.ErrInvalidCredentials:
			response.Error(c, http.StatusUnauthorized, 401, "账号已禁用")
		default:
			response.Error(c, http.StatusInternalServerError, 500, "刷新失败，请稍后重试")
		}
		return
	}
	response.OK(c, result)
}

// Login 登录接口：先校验图形验证码，再校验用户名密码并签发 token 对
func (h *Auth) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	// 校验图形验证码（校验一次即消费，防止暴力撞码）
	if !h.captchaService.Verify(req.CaptchaID, req.CaptchaCode) {
		response.Error(c, http.StatusBadRequest, 400, "验证码错误或已过期")
		return
	}
	result, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			// 登录失败审计（此时无登录用户，UserID=0、Username=system）
			middleware.Record(h.db, c, "auth", "login", "", fmt.Sprintf("登录失败：用户名 %s", req.Username))
			response.Error(c, http.StatusUnauthorized, 401, "用户名或密码错误")
		case service.ErrLockTriggered:
			// 本次失败触发了账号锁定：对外仍返回通用错误，避免暴露锁定信息
			middleware.Record(h.db, c, "auth", "lock", "", fmt.Sprintf("连续登录失败，账号 %s 已锁定 15 分钟", req.Username))
			response.Error(c, http.StatusUnauthorized, 401, "用户名或密码错误")
		case service.ErrAccountLocked:
			// 账号处于锁定期间，即使密码正确也拒绝
			middleware.Record(h.db, c, "auth", "lock", "", fmt.Sprintf("账号 %s 处于锁定期间，拒绝登录", req.Username))
			response.Error(c, http.StatusBadRequest, 400, "账号已锁定，请 15 分钟后再试")
		default:
			response.Error(c, http.StatusInternalServerError, 500, "登录失败，请稍后重试")
		}
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
