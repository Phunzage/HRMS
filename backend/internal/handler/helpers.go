package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseID 解析路径参数中的资源 ID
func parseID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// parseUintQuery 解析 query 参数为 uint，缺失或非法时返回 0
func parseUintQuery(c *gin.Context, key string) uint {
	if v, err := strconv.ParseUint(c.Query(key), 10, 64); err == nil {
		return uint(v)
	}
	return 0
}

// xlsxContentType Excel 附件 MIME 类型
const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// writeXLSX 以附件下载形式写出 Excel 二进制流（不走统一 JSON 响应）
func writeXLSX(c *gin.Context, filename string, data []byte) {
	c.Header("Content-Type", xlsxContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, xlsxContentType, data)
}
