package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Announcement 公告处理器
type Announcement struct {
	service *service.AnnouncementService
	db      *gorm.DB
}

// NewAnnouncement 创建公告处理器
func NewAnnouncement(svc *service.AnnouncementService, db *gorm.DB) *Announcement {
	return &Announcement{service: svc, db: db}
}

// List 公告分页列表（published 优先置顶+时间倒序），支持 keyword 搜索标题
func (h *Announcement) List(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(keyword, status, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取公告列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// Create 发布公告
func (h *Announcement) Create(c *gin.Context) {
	var req service.CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	ann, err := h.service.Create(middleware.GetUserID(c), middleware.GetUsername(c), req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "announcement", "create", fmt.Sprintf("announcement:%d", ann.ID), fmt.Sprintf("发布公告 标题:%s 置顶:%v", ann.Title, ann.IsPinned))
	response.OK(c, ann)
}

// Update 编辑公告（标题/内容/置顶/状态）
func (h *Announcement) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	ann, err := h.service.Update(id, req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "announcement", "update", fmt.Sprintf("announcement:%d", ann.ID), fmt.Sprintf("编辑公告 标题:%s", ann.Title))
	response.OK(c, ann)
}

// Delete 删除公告（软删除）
func (h *Announcement) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Delete(id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "announcement", "delete", fmt.Sprintf("announcement:%d", id), fmt.Sprintf("删除公告(ID:%d)", id))
	response.OK(c, nil)
}

// handleErr 公告模块错误映射
func (h *Announcement) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAnnouncementTitleRequired),
		errors.Is(err, service.ErrAnnouncementContentRequired),
		errors.Is(err, service.ErrAnnouncementStatusInvalid):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrAnnouncementNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
