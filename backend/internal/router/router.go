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

	// 考勤模块
	attendanceRepo := repository.NewAttendanceRepo(db)
	attendanceService := service.NewAttendanceService(attendanceRepo, empRepo)
	attendanceHandler := handler.NewAttendance(attendanceService)

	// 请假审批流模块
	leaveRepo := repository.NewLeaveRepo(db)
	leaveService := service.NewLeaveService(leaveRepo, empRepo)
	leaveHandler := handler.NewLeave(leaveService)

	// 薪酬模块
	salaryRepo := repository.NewSalaryRepo(db)
	salaryService := service.NewSalaryService(salaryRepo, empRepo)
	salaryHandler := handler.NewSalary(salaryService)

	// 招聘模块
	recruitmentRepo := repository.NewRecruitmentRepo(db)
	recruitmentService := service.NewRecruitmentService(recruitmentRepo, deptRepo, posRepo)
	recruitmentHandler := handler.NewRecruitment(recruitmentService)

	// 统计仪表盘模块
	dashboardRepo := repository.NewDashboardRepo(db)
	dashboardService := service.NewDashboardService(dashboardRepo)
	dashboardHandler := handler.NewDashboard(dashboardService)

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

			// 考勤管理
			attendance := authed.Group("/attendance")
			{
				attendance.GET("", middleware.RequirePermission(db, "attendance:read"), attendanceHandler.List)
				attendance.POST("/check-in", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.CheckIn)
				attendance.POST("/check-out", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.CheckOut)
				attendance.POST("/manual", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Manual)
				attendance.PUT("/:id", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Update)
				attendance.DELETE("/:id", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Delete)
			}

			// 请假审批流
			leaves := authed.Group("/leaves")
			{
				// 自助接口：仅需登录（静态段优先于 :id 匹配）
				leaves.GET("/mine", leaveHandler.Mine)
				leaves.GET("/balance", leaveHandler.Balance)
				leaves.POST("", middleware.RequirePermission(db, "leave:write"), leaveHandler.Create)
				leaves.GET("", middleware.RequirePermission(db, "leave:read"), leaveHandler.List)
				leaves.PUT("/:id/approve", middleware.RequirePermission(db, "leave:write"), leaveHandler.Approve)
				leaves.PUT("/:id/reject", middleware.RequirePermission(db, "leave:write"), leaveHandler.Reject)
				leaves.PUT("/:id/cancel", leaveHandler.Cancel)
			}

			// 薪酬管理
			salaries := authed.Group("/salaries")
			{
				// 自助接口：员工查看自己的工资单（仅需登录）
				salaries.GET("/mine", salaryHandler.Mine)
				salaries.GET("", middleware.RequirePermission(db, "salary:read"), salaryHandler.List)
				salaries.POST("", middleware.RequirePermission(db, "salary:write"), salaryHandler.Create)
				salaries.PUT("/:id", middleware.RequirePermission(db, "salary:write"), salaryHandler.Update)
				salaries.PUT("/:id/confirm", middleware.RequirePermission(db, "salary:write"), salaryHandler.Confirm)
				salaries.PUT("/:id/pay", middleware.RequirePermission(db, "salary:write"), salaryHandler.Pay)
			}

			// 招聘管理
			recruitment := authed.Group("/recruitment")
			{
				jobs := recruitment.Group("/jobs")
				{
					jobs.GET("", middleware.RequirePermission(db, "recruitment:read"), recruitmentHandler.JobList)
					jobs.POST("", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.JobCreate)
					jobs.PUT("/:id", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.JobUpdate)
					jobs.DELETE("/:id", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.JobDelete)
				}
				candidates := recruitment.Group("/candidates")
				{
					candidates.GET("", middleware.RequirePermission(db, "recruitment:read"), recruitmentHandler.CandidateList)
					candidates.POST("", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.CandidateCreate)
					candidates.PUT("/:id", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.CandidateUpdate)
					candidates.DELETE("/:id", middleware.RequirePermission(db, "recruitment:write"), recruitmentHandler.CandidateDelete)
				}
			}

			// 统计仪表盘（admin/hr 可访问）
			dashboard := authed.Group("/dashboard")
			{
				dashboard.GET("/stats", middleware.RequirePermission(db, "dashboard:read"), dashboardHandler.Stats)
			}
		}
	}
}
