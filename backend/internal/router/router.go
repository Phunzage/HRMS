package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/handler"
)

// Register 注册所有路由
func Register(r *gin.Engine, db *gorm.DB) {
	// 初始化各模块处理器
	healthHandler := handler.NewHealth(db)

	// API v1 分组
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", healthHandler.Check)
	}
}
