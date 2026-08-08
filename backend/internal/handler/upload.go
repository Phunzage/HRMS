package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Upload 文件上传处理器
type Upload struct {
	service *service.UploadService
}

// NewUpload 创建上传处理器
func NewUpload(svc *service.UploadService) *Upload {
	return &Upload{service: svc}
}

// UploadImage 图片上传接口（需登录，无额外权限点）：multipart 字段名 file
func (h *Upload) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请选择要上传的文件")
		return
	}
	result, err := h.service.SaveImage(file)
	if err != nil {
		switch err {
		case service.ErrFileTooLarge:
			response.Error(c, http.StatusBadRequest, 400, "文件大小不能超过 5MB")
		case service.ErrFileTypeNotAllow:
			response.Error(c, http.StatusBadRequest, 400, "仅支持 jpg/png/webp/gif 图片")
		default:
			response.Error(c, http.StatusInternalServerError, 500, "文件上传失败，请稍后重试")
		}
		return
	}
	response.OK(c, result)
}
