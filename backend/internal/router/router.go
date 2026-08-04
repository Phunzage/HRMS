package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/config"
	"hrms/backend/internal/handler"
	"hrms/backend/internal/middleware"
	"hrms/backend/internal/repository"
	"hrms/backend/internal/service"
)

// Register 注册所有路由
func Register(r *gin.Engine, db *gorm.DB) {
	cfg := config.Load()

	// 初始化各模块处理器
	healthHandler := handler.NewHealth(db)

	// 认证模块（handler → service → repository 分层）
	userRepo := repository.NewUserRepo(db)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.TokenExpireHours)
	authHandler := handler.NewAuth(authService)

	// API v1 分组
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", healthHandler.Check)

		// 认证（登录无需 JWT）
		v1.POST("/auth/login", authHandler.Login)

		// 需登录的认证接口
		authed := v1.Group("", middleware.JWTAuth(cfg.JWTSecret))
		{
			authed.GET("/auth/me", authHandler.Me)
			authed.PUT("/auth/password", authHandler.ChangePassword)
		}
	}
}
