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

	// 组织人事模块
	deptRepo := repository.NewDepartmentRepo(db)
	deptService := service.NewDepartmentService(deptRepo)
	deptHandler := handler.NewDepartment(deptService)

	posRepo := repository.NewPositionRepo(db)
	posService := service.NewPositionService(posRepo)
	posHandler := handler.NewPosition(posService)

	empRepo := repository.NewEmployeeRepo(db)
	empService := service.NewEmployeeService(empRepo, deptRepo, posRepo)
	empHandler := handler.NewEmployee(empService)

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

			// 部门管理
			departments := authed.Group("/departments")
			{
				departments.GET("", middleware.RequirePermission(db, "department:read"), deptHandler.List)
				departments.POST("", middleware.RequirePermission(db, "department:write"), deptHandler.Create)
				departments.PUT("/:id", middleware.RequirePermission(db, "department:write"), deptHandler.Update)
				departments.DELETE("/:id", middleware.RequirePermission(db, "department:write"), deptHandler.Delete)
			}

			// 岗位管理
			positions := authed.Group("/positions")
			{
				positions.GET("", middleware.RequirePermission(db, "position:read"), posHandler.List)
				positions.POST("", middleware.RequirePermission(db, "position:write"), posHandler.Create)
				positions.PUT("/:id", middleware.RequirePermission(db, "position:write"), posHandler.Update)
				positions.DELETE("/:id", middleware.RequirePermission(db, "position:write"), posHandler.Delete)
			}

			// 员工管理
			employees := authed.Group("/employees")
			{
				// 自助接口：仅需登录，无需额外权限（须先于 :id 注册，静态段优先匹配）
				employees.GET("/me", empHandler.Me)
				employees.GET("", middleware.RequirePermission(db, "employee:read"), empHandler.List)
				employees.POST("", middleware.RequirePermission(db, "employee:write"), empHandler.Create)
				employees.GET("/:id", middleware.RequirePermission(db, "employee:read"), empHandler.Get)
				employees.PUT("/:id", middleware.RequirePermission(db, "employee:write"), empHandler.Update)
				employees.DELETE("/:id", middleware.RequirePermission(db, "employee:write"), empHandler.Delete)
			}
		}
	}
}
