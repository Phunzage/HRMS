package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// Record 记录一条操作审计日志：从 gin 上下文读取当前登录用户（无登录用户时 UserID=0、Username="system"），
// 记录来源 IP 后写入 AuditLog。在各 handler 的关键操作成功后调用。
func Record(db *gorm.DB, c *gin.Context, module, action, resourceID, detail string) {
	username := GetUsername(c)
	if username == "" {
		username = "system"
	}
	log := &models.AuditLog{
		UserID:     GetUserID(c),
		Username:   username,
		Module:     module,
		Action:     action,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         c.ClientIP(),
	}
	if err := db.Create(log).Error; err != nil {
		slog.Error("写入审计日志失败", "err", err, "module", module, "action", action)
	}
}
