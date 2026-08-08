package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Badge 导航栏角标处理器
type Badge struct {
	service *service.BadgeService
}

// NewBadge 创建角标处理器
func NewBadge(svc *service.BadgeService) *Badge {
	return &Badge{service: svc}
}

// Summary 当前用户导航栏角标汇总（待审批/未读公告/未读通知）
func (h *Badge) Summary(c *gin.Context) {
	counts, err := h.service.Summary(middleware.GetUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取角标汇总失败")
		return
	}
	response.OK(c, counts)
}
