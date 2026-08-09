package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// UpdateRepo 客户端版本数据仓库
type UpdateRepo struct {
	db *gorm.DB
}

// NewUpdateRepo 创建版本仓库
func NewUpdateRepo(db *gorm.DB) *UpdateRepo {
	return &UpdateRepo{db: db}
}

// Latest 查询某平台 build_number 最大的版本记录（无记录时返回 nil）
func (r *UpdateRepo) Latest(platform string) (*models.AppVersion, error) {
	var v models.AppVersion
	err := r.db.Where("platform = ?", platform).
		Order("build_number DESC").
		First(&v).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// List 某平台版本分页列表（platform 为空查全部），按 build_number 倒序
func (r *UpdateRepo) List(platform string, page, pageSize int) ([]models.AppVersion, int64, error) {
	q := r.db.Model(&models.AppVersion{})
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]models.AppVersion, 0)
	err := q.Order("build_number DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Create 登记新版本
func (r *UpdateRepo) Create(v *models.AppVersion) error {
	return r.db.Create(v).Error
}
