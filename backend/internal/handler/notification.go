package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Notification 站内通知处理器
type Notification struct {
	service *service.NotificationService
	db      *gorm.DB
}

// NewNotification 创建通知处理器
func NewNotification(svc *service.NotificationService, db *gorm.DB) *Notification {
	return &Notification{service: svc, db: db}
}

// List 当前用户通知分页列表，支持 is_read 过滤
func (h *Notification) List(c *gin.Context) {
	var isRead *bool
	if v := c.Query("is_read"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			isRead = &b
		}
	}
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(middleware.GetUserID(c), isRead, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取通知列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Recent 当前用户最近 N 条未读通知（按 id 倒序），供客户端增量检测新通知；
// 无未读时返回空数组而非 null
func (h *Notification) Recent(c *gin.Context) {
	limit := 5
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := h.service.RecentUnread(middleware.GetUserID(c), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取最近通知失败")
		return
	}
	response.OK(c, items)
}

// UnreadCount 当前用户未读通知数量（顶栏铃铛用）
func (h *Notification) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(middleware.GetUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取未读数量失败")
		return
	}
	response.OK(c, gin.H{"count": count})
}

// MarkRead 标记单条通知已读（校验归属当前用户）
func (h *Notification) MarkRead(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.MarkRead(middleware.GetUserID(c), id); err != nil {
		switch {
		case errors.Is(err, service.ErrNotifNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrNotifNotOwner):
			response.Error(c, http.StatusForbidden, 403, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "标记已读失败")
		}
		return
	}
	response.OK(c, nil)
}

// MarkAllRead 当前用户全部通知标记已读
func (h *Notification) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(middleware.GetUserID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "全部标记已读失败")
		return
	}
	response.OK(c, nil)
}
