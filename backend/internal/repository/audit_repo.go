package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// AuditRepo 审计日志数据仓库
type AuditRepo struct {
	db *gorm.DB
}

// NewAuditRepo 创建审计日志仓库
func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// List 审计日志分页列表，支持 module / action / username / start_date / end_date 过滤
// start_date、end_date 格式为 YYYY-MM-DD，过滤 created_at 日期
func (r *AuditRepo) List(module, action, username, startDate, endDate string, page, pageSize int) ([]models.AuditLog, int64, error) {
	q := r.db.Model(&models.AuditLog{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if username != "" {
		q = q.Where("username LIKE ?", "%"+username+"%")
	}
	if startDate != "" {
		q = q.Where("created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		q = q.Where("created_at <= ?", endDate+" 23:59:59")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.AuditLog
	err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
