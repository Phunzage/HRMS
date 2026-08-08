package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Audit 审计日志处理器
type Audit struct {
	service *service.AuditService
}

// NewAudit 创建审计日志处理器
func NewAudit(svc *service.AuditService) *Audit {
	return &Audit{service: svc}
}

// List 审计日志分页列表，支持 module / action / username / start_date / end_date 过滤
func (h *Audit) List(c *gin.Context) {
	module := c.Query("module")
	action := c.Query("action")
	username := c.Query("username")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(module, action, username, startDate, endDate, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取审计日志失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}
