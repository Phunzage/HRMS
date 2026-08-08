package service

import (
	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// AuditService 审计日志查询服务
type AuditService struct {
	repo *repository.AuditRepo
}

// NewAuditService 创建审计日志服务
func NewAuditService(repo *repository.AuditRepo) *AuditService {
	return &AuditService{repo: repo}
}

// List 审计日志分页列表，支持 module/action/username/start_date/end_date 过滤
func (s *AuditService) List(module, action, username, startDate, endDate string, page, pageSize int) ([]models.AuditLog, int64, error) {
	return s.repo.List(module, action, username, startDate, endDate, page, pageSize)
}
