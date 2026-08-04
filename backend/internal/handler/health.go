package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/response"
)

// Health 健康检查处理器
type Health struct {
	db *gorm.DB
}

// NewHealth 创建健康检查处理器
func NewHealth(db *gorm.DB) *Health {
	return &Health{db: db}
}

// Check 健康检查接口：探测服务与数据库状态
func (h *Health) Check(c *gin.Context) {
	// 探测数据库连接
	if err := h.db.Exec("SELECT 1").Error; err != nil {
		response.Error(c, http.StatusServiceUnavailable, 1, "数据库连接失败")
		return
	}
	response.OK(c, gin.H{
		"status": "ok",
		"db":     "up",
	})
}
