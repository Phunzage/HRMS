package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Update 客户端版本更新处理器
// check/latest 为公开接口（客户端启动时无需登录调用）；
// versions 为管理接口，受 update:read/update:write 权限点控制
type Update struct {
	service *service.UpdateService
	db      *gorm.DB
}

// NewUpdate 创建版本更新处理器
func NewUpdate(svc *service.UpdateService, db *gorm.DB) *Update {
	return &Update{service: svc, db: db}
}

// Check 检查更新（公开）：GET /update/check?platform=win&version=1.0.0&build=5
// 客户端 build 低于服务器最新 build 时返回 has_update=true 及最新版本信息
func (h *Update) Check(c *gin.Context) {
	platform := c.Query("platform")
	if platform == "" {
		response.Error(c, http.StatusBadRequest, 400, "platform 参数不能为空")
		return
	}
	latest, hasUpdate, err := h.service.Check(platform, parseBuildQuery(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "检查更新失败")
		return
	}
	if !hasUpdate || latest == nil {
		response.OK(c, gin.H{"has_update": false})
		return
	}
	response.OK(c, gin.H{
		"has_update": true,
		"latest": gin.H{
			"platform":      latest.Platform,
			"version":       latest.Version,
			"build_number":  latest.BuildNumber,
			"url":           latest.URL,
			"release_notes": latest.ReleaseNotes,
		},
	})
}

// Latest 查询某平台最新版本（公开）：GET /update/latest?platform=win
func (h *Update) Latest(c *gin.Context) {
	platform := c.Query("platform")
	if platform == "" {
		response.Error(c, http.StatusBadRequest, 400, "platform 参数不能为空")
		return
	}
	v, err := h.service.Latest(platform)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "查询最新版本失败")
		return
	}
	if v == nil {
		response.Error(c, http.StatusNotFound, 404, "该平台暂无版本记录")
		return
	}
	response.OK(c, v)
}

// Create 登记新版本（admin + update:write）：POST /update/versions
func (h *Update) Create(c *gin.Context) {
	var req service.CreateUpdateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	v, err := h.service.Create(req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "update", "create",
		fmt.Sprintf("version:%d", v.ID),
		fmt.Sprintf("登记版本 %s %s (build %d)", v.Platform, v.Version, v.BuildNumber))
	response.OK(c, v)
}

// List 版本列表（admin + update:read）：GET /update/versions?platform=
func (h *Update) List(c *gin.Context) {
	platform := c.Query("platform")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(platform, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "查询版本列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// handleErr 版本登记错误映射
func (h *Update) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUpdatePlatformInvalid),
		errors.Is(err, service.ErrUpdateVersionRequired),
		errors.Is(err, service.ErrUpdateBuildInvalid),
		errors.Is(err, service.ErrUpdateURLRequired),
		errors.Is(err, service.ErrUpdateBuildNotNewer),
		errors.Is(err, service.ErrUpdateVersionExists):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "登记版本失败")
	}
}

// parseBuildQuery 解析 build query 参数，缺失或非法时返回 0
func parseBuildQuery(c *gin.Context) int {
	if v, err := strconv.Atoi(c.Query("build")); err == nil {
		return v
	}
	return 0
}
