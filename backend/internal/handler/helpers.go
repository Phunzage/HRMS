package handler

import (
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
