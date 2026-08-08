package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Dashboard 仪表盘处理器
type Dashboard struct {
	service *service.DashboardService
}

// NewDashboard 创建仪表盘处理器
func NewDashboard(svc *service.DashboardService) *Dashboard {
	return &Dashboard{service: svc}
}

// Stats 获取仪表盘统计（按登录用户数据范围过滤，hr 仅统计负责部门）
func (h *Dashboard) Stats(c *gin.Context) {
	stats, err := h.service.Stats(middleware.GetRoleCode(c), middleware.GetUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取统计数据失败")
		return
	}
	response.OK(c, stats)
}
