package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/response"
)

// 上下文键名，用于在请求链路中传递当前登录用户信息
const (
	CtxUserID   = "auth_user_id"   // 用户ID
	CtxUsername = "auth_username"  // 用户名
	CtxRoleCode = "auth_role_code" // 角色编码
)

// Claims 自定义 JWT Claims，包含登录态所需的最小信息
// Type 区分令牌用途：access 为访问令牌，refresh 为刷新令牌（refresh 令牌不能当 access 用）
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	RoleCode string `json:"role_code"`
	Type     string `json:"type,omitempty"`
	jwt.RegisteredClaims
}

// JWTAuth 校验 Authorization: Bearer <token>，成功后将用户信息写入 gin.Context
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, 401, "未登录或登录已过期")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, 401, "未登录或登录已过期")
			c.Abort()
			return
		}
		// refresh 令牌仅用于换取新令牌，不能作为访问令牌使用
		if claims.Type == "refresh" {
			response.Error(c, http.StatusUnauthorized, 401, "未登录或登录已过期")
			c.Abort()
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRoleCode, claims.RoleCode)
		c.Next()
	}
}

// RequirePermission 校验当前角色是否拥有指定权限点，admin 角色直接放行
func RequirePermission(db *gorm.DB, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetRoleCode(c) == "admin" {
			c.Next()
			return
		}
		// 查询角色
		var role models.Role
		if err := db.Where("code = ?", GetRoleCode(c)).First(&role).Error; err != nil {
			response.Error(c, http.StatusForbidden, 403, "无权限访问")
			c.Abort()
			return
		}
		// 查询权限点
		var permission models.Permission
		if err := db.Where("code = ?", code).First(&permission).Error; err != nil {
			response.Error(c, http.StatusForbidden, 403, "无权限访问")
			c.Abort()
			return
		}
		// 校验角色-权限关联
		var rp models.RolePermission
		if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).First(&rp).Error; err != nil {
			response.Error(c, http.StatusForbidden, 403, "无权限访问")
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserID 从上下文读取当前登录用户 ID
func GetUserID(c *gin.Context) uint {
	if v, ok := c.Get(CtxUserID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文读取当前登录用户名
func GetUsername(c *gin.Context) string {
	if v, ok := c.Get(CtxUsername); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetRoleCode 从上下文读取当前登录用户角色编码
func GetRoleCode(c *gin.Context) string {
	if v, ok := c.Get(CtxRoleCode); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
