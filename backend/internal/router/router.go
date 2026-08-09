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
	captchaService := service.NewCaptchaService()
	authHandler := handler.NewAuth(authService, captchaService, db)

	// 文件上传模块
	uploadService := service.NewUploadService("./uploads")
	uploadHandler := handler.NewUpload(uploadService)

	// 组织人事模块
	deptRepo := repository.NewDepartmentRepo(db)
	deptService := service.NewDepartmentService(deptRepo)
	deptHandler := handler.NewDepartment(deptService, db)

	posRepo := repository.NewPositionRepo(db)
	posService := service.NewPositionService(posRepo)
	posHandler := handler.NewPosition(posService, db)

	empRepo := repository.NewEmployeeRepo(db)
	empService := service.NewEmployeeService(db, empRepo, deptRepo, posRepo)
	empHandler := handler.NewEmployee(empService, db)

	// 考勤模块
	attendanceRepo := repository.NewAttendanceRepo(db)
	attendanceService := service.NewAttendanceService(db, attendanceRepo, empRepo)
	attendanceHandler := handler.NewAttendance(attendanceService, db)

	// 考勤补卡申请模块
	correctionRepo := repository.NewAttendanceCorrectionRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	notificationService := service.NewNotificationService(notificationRepo, empRepo)
	correctionService := service.NewAttendanceCorrectionService(db, correctionRepo, empRepo)
	correctionHandler := handler.NewAttendanceCorrection(correctionService, db)

	// 站内通知模块
	notificationHandler := handler.NewNotification(notificationService, db)

	// 请假审批流模块
	leaveRepo := repository.NewLeaveRepo(db)
	leaveService := service.NewLeaveService(db, leaveRepo, empRepo)
	leaveHandler := handler.NewLeave(leaveService, db)

	// 通用审批流模块（请假/补卡均已迁移，终审回调按业务类型注册）
	approvalHandler := handler.NewApproval(db, map[string]service.ApprovalBizCallback{
		"leave":                 leaveService,
		"attendance_correction": correctionService,
	})

	// 审批流程配置模块（admin 专属，approval_flow:read/write）
	approvalAdminHandler := handler.NewApprovalAdmin(db)

	// 薪酬模块
	salaryRepo := repository.NewSalaryRepo(db)
	salaryService := service.NewSalaryService(db, salaryRepo, empRepo)
	salaryHandler := handler.NewSalary(salaryService, db)

	// 招聘模块
	recruitmentRepo := repository.NewRecruitmentRepo(db)
	recruitmentService := service.NewRecruitmentService(db, recruitmentRepo, deptRepo, posRepo)
	recruitmentHandler := handler.NewRecruitment(recruitmentService, db)

	// 公告管理模块
	announcementRepo := repository.NewAnnouncementRepo(db)
	announcementService := service.NewAnnouncementService(announcementRepo)
	announcementHandler := handler.NewAnnouncement(announcementService, db)

	// 导航栏角标汇总模块（复用审批/公告/通知统计）
	badgeService := service.NewBadgeService(db, announcementRepo, notificationRepo)
	badgeHandler := handler.NewBadge(badgeService)

	// 绩效管理模块
	performanceRepo := repository.NewPerformanceRepo(db)
	performanceService := service.NewPerformanceService(db, performanceRepo, empRepo)
	performanceHandler := handler.NewPerformance(performanceService, db)

	// 统计仪表盘模块（需 db 解析 hr 数据范围）
	dashboardRepo := repository.NewDashboardRepo(db)
	dashboardService := service.NewDashboardService(db, dashboardRepo)
	dashboardHandler := handler.NewDashboard(dashboardService)

	// 用户/角色/审计模块
	roleRepo := repository.NewRoleRepo(db)
	userAdminService := service.NewUserService(userRepo, roleRepo)
	userAdminHandler := handler.NewUser(userAdminService, db)

	roleService := service.NewRoleService(roleRepo, userRepo)
	roleHandler := handler.NewRole(roleService, db)

	auditRepo := repository.NewAuditRepo(db)
	auditService := service.NewAuditService(auditRepo)
	auditHandler := handler.NewAudit(auditService)

	// API v1 分组
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", healthHandler.Check)

		// 认证（验证码、登录、刷新均无需 JWT）
		v1.GET("/auth/captcha", authHandler.Captcha)
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/refresh", authHandler.Refresh)

		// 需登录的认证接口
		authed := v1.Group("", middleware.JWTAuth(cfg.JWTSecret))
		{
			authed.GET("/auth/me", authHandler.Me)
			authed.PUT("/auth/password", authHandler.ChangePassword)

			// 文件上传（仅需登录，无额外权限点）
			authed.POST("/upload", uploadHandler.UploadImage)

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
				employees.PUT("/me/avatar", empHandler.UpdateMyAvatar)
				employees.GET("", middleware.RequirePermission(db, "employee:read"), empHandler.List)
				employees.POST("", middleware.RequirePermission(db, "employee:write"), empHandler.Create)
				// 导出须先于 :id 注册，否则 "export" 会被当作 id 匹配详情路由
				employees.GET("/export", middleware.RequirePermission(db, "employee:read"), empHandler.Export)
				employees.GET("/:id", middleware.RequirePermission(db, "employee:read"), empHandler.Get)
				employees.PUT("/:id", middleware.RequirePermission(db, "employee:write"), empHandler.Update)
				employees.DELETE("/:id", middleware.RequirePermission(db, "employee:write"), empHandler.Delete)
			}

			// 考勤管理
			attendance := authed.Group("/attendance")
			{
				attendance.GET("", middleware.RequirePermission(db, "attendance:read"), attendanceHandler.List)
				// 考勤月报（静态段必须先于 :id 注册，否则 report 会被当作 id 捕获）
				attendance.GET("/report", middleware.RequirePermission(db, "attendance:read"), attendanceHandler.Report)
				attendance.POST("/check-in", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.CheckIn)
				attendance.POST("/check-out", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.CheckOut)
				attendance.POST("/manual", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Manual)
				// 补卡申请（静态段必须先于 :id 注册，否则 corrections 会被当作 id 捕获）
				// 审批动作已迁移至 /approvals 通用接口，此处仅保留提交/列表/取消
				corrections := attendance.Group("/corrections")
				{
					corrections.POST("", middleware.RequirePermission(db, "attendance:write"), correctionHandler.Create)
					corrections.GET("", middleware.RequirePermission(db, "attendance:read"), correctionHandler.List)
					corrections.GET("/mine", correctionHandler.Mine)
					corrections.PUT("/:id/cancel", correctionHandler.Cancel)
				}
				attendance.PUT("/:id", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Update)
				attendance.DELETE("/:id", middleware.RequirePermission(db, "attendance:write"), attendanceHandler.Delete)
			}

			// 站内通知（登录即可，无权限点）
			notifications := authed.Group("/notifications")
			{
				notifications.GET("", notificationHandler.List)
				notifications.GET("/unread-count", notificationHandler.UnreadCount)
				notifications.PUT("/:id/read", notificationHandler.MarkRead)
				notifications.PUT("/read-all", notificationHandler.MarkAllRead)
			}

			// 导航栏角标汇总（登录即可，无权限点）
			authed.GET("/badges", badgeHandler.Summary)

			// 请假审批流（审批动作已迁移至 /approvals 通用接口）
			leaves := authed.Group("/leaves")
			{
				// 自助接口：仅需登录（静态段优先于 :id 匹配）
				leaves.GET("/mine", leaveHandler.Mine)
				leaves.GET("/balance", leaveHandler.Balance)
				leaves.POST("", middleware.RequirePermission(db, "leave:write"), leaveHandler.Create)
				leaves.GET("", middleware.RequirePermission(db, "leave:read"), leaveHandler.List)
				leaves.PUT("/:id/cancel", leaveHandler.Cancel)
			}

			// 通用审批流（JWT 保护，无新增权限点，靠任务归属校验）
			approvals := authed.Group("/approvals")
			{
				// 静态段须先于 :instanceId 注册，避免 "mine" / "biz" / "tasks" 被当作实例 ID 捕获
				approvals.GET("/mine", approvalHandler.Mine)
				approvals.GET("/biz", approvalHandler.Biz)
				approvals.PUT("/tasks/:taskId/approve", approvalHandler.Approve)
				approvals.PUT("/tasks/:taskId/reject", approvalHandler.Reject)
				approvals.GET("/:instanceId/tasks", approvalHandler.InstanceTasks)
			}

			// 审批流程配置（admin 专属权限点 approval_flow:read/write）
			approvalFlows := authed.Group("/approval-flows")
			{
				approvalFlows.GET("", middleware.RequirePermission(db, "approval_flow:read"), approvalAdminHandler.List)
				approvalFlows.PUT("/:code", middleware.RequirePermission(db, "approval_flow:write"), approvalAdminHandler.Update)
			}

			// 薪酬管理
			salaries := authed.Group("/salaries")
			{
				// 自助接口：员工查看自己的工资单（仅需登录）
				salaries.GET("/mine", salaryHandler.Mine)
				salaries.GET("", middleware.RequirePermission(db, "salary:read"), salaryHandler.List)
				salaries.GET("/export", middleware.RequirePermission(db, "salary:read"), salaryHandler.Export)
				salaries.POST("", middleware.RequirePermission(db, "salary:write"), salaryHandler.Create)
				// 批量生成须先于 /:id 注册，静态段优先匹配
				salaries.POST("/batch-generate", middleware.RequirePermission(db, "salary:write"), salaryHandler.BatchGenerate)
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

			// 公告管理（已读接口登录即可，无权限点；read-all 静态段须先于 :id 注册）
			announcements := authed.Group("/announcements")
			{
				announcements.PUT("/read-all", announcementHandler.MarkAllRead)
				announcements.GET("", middleware.RequirePermission(db, "announcement:read"), announcementHandler.List)
				announcements.POST("", middleware.RequirePermission(db, "announcement:write"), announcementHandler.Create)
				announcements.PUT("/:id/read", announcementHandler.MarkRead)
				announcements.PUT("/:id", middleware.RequirePermission(db, "announcement:write"), announcementHandler.Update)
				announcements.DELETE("/:id", middleware.RequirePermission(db, "announcement:write"), announcementHandler.Delete)
			}

			// 绩效管理
			performance := authed.Group("/performance")
			{
				// 自助接口：查看自己的绩效（仅需登录，静态段优先于 :id 匹配）
				performance.GET("/mine", performanceHandler.Mine)
				performance.GET("", middleware.RequirePermission(db, "performance:read"), performanceHandler.List)
				performance.POST("", middleware.RequirePermission(db, "performance:write"), performanceHandler.Create)
				performance.PUT("/:id/submit", middleware.RequirePermission(db, "performance:write"), performanceHandler.Submit)
				performance.PUT("/:id/approve", middleware.RequirePermission(db, "performance:write"), performanceHandler.Approve)
				performance.PUT("/:id", middleware.RequirePermission(db, "performance:write"), performanceHandler.Update)
				performance.DELETE("/:id", middleware.RequirePermission(db, "performance:write"), performanceHandler.Delete)
			}

			// 用户管理
			users := authed.Group("/users")
			{
				// 角色下拉（仅需登录，静态段优先于 :id 匹配）
				users.GET("/roles", userAdminHandler.Roles)
				users.GET("", middleware.RequirePermission(db, "user:read"), userAdminHandler.List)
				users.POST("", middleware.RequirePermission(db, "user:write"), userAdminHandler.Create)
				users.PUT("/:id", middleware.RequirePermission(db, "user:write"), userAdminHandler.Update)
				users.PUT("/:id/password", middleware.RequirePermission(db, "user:write"), userAdminHandler.ResetPassword)
				users.DELETE("/:id", middleware.RequirePermission(db, "user:write"), userAdminHandler.Delete)
			}

			// 角色管理
			roles := authed.Group("/roles")
			{
				roles.GET("", middleware.RequirePermission(db, "role:read"), roleHandler.List)
				roles.GET("/permissions", middleware.RequirePermission(db, "role:read"), roleHandler.Permissions)
				roles.POST("", middleware.RequirePermission(db, "role:write"), roleHandler.Create)
				roles.PUT("/:id", middleware.RequirePermission(db, "role:write"), roleHandler.Update)
				roles.DELETE("/:id", middleware.RequirePermission(db, "role:write"), roleHandler.Delete)
			}

			// 审计日志
			auditLogs := authed.Group("/audit-logs")
			{
				auditLogs.GET("", middleware.RequirePermission(db, "audit:read"), auditHandler.List)
			}

			// 统计仪表盘（admin/hr 可访问）
			dashboard := authed.Group("/dashboard")
			{
				dashboard.GET("/stats", middleware.RequirePermission(db, "dashboard:read"), dashboardHandler.Stats)
			}
		}
	}

	// 静态文件服务：上传的图片（须在 /api 路由外注册，Gin 原生提供目录服务）
	r.Static("/uploads", "./uploads")
}
