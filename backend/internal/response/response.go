package response

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Body 统一响应体
type Body struct {
	Code    int         `json:"code"`           // 业务码，0 表示成功
	Message string      `json:"message"`        // 提示信息
	Data    interface{} `json:"data,omitempty"` // 业务数据
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`      // 当前页数据
	Total    int64       `json:"total"`     // 总条数
	Page     int         `json:"page"`      // 当前页
	PageSize int         `json:"page_size"` // 每页条数
}

// OK 成功响应，返回 200 + {code:0, message:"ok", data}
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Body{Code: 0, Message: "ok", Data: data})
}

// Error 统一错误响应
func Error(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Body{Code: code, Message: message})
}

// PageOK 分页成功响应
func PageOK(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(200, Body{Code: 0, Message: "ok", Data: PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}})
}

// Pagination 从 query 参数解析分页（默认 page=1, page_size=20，上限 100）
func Pagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 {
		pageSize = ps
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return
}
