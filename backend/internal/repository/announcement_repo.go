package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// AnnouncementRepo 公告数据仓库
type AnnouncementRepo struct {
	db *gorm.DB
}

// NewAnnouncementRepo 创建公告仓库
func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo {
	return &AnnouncementRepo{db: db}
}

// announcementQuery 公告基础查询（软删除自动过滤）
// status 为空表示不过滤状态；keyword 搜索标题
func (r *AnnouncementRepo) announcementQuery(keyword, status string) *gorm.DB {
	q := r.db.Model(&models.Announcement{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		q = q.Where("title LIKE ?", "%"+keyword+"%")
	}
	return q
}

// List 公告分页列表：published 优先 → 置顶优先 → 发布时间倒序
func (r *AnnouncementRepo) List(keyword, status string, page, pageSize int) ([]models.Announcement, int64, error) {
	var total int64
	if err := r.announcementQuery(keyword, status).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Announcement
	err := r.announcementQuery(keyword, status).
		Order("CASE WHEN status = 'published' THEN 0 ELSE 1 END").
		Order("is_pinned DESC").
		Order("COALESCE(published_at, created_at) DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID 根据 ID 查询公告
func (r *AnnouncementRepo) FindByID(id uint) (*models.Announcement, error) {
	var ann models.Announcement
	if err := r.db.First(&ann, id).Error; err != nil {
		return nil, err
	}
	return &ann, nil
}

// Create 创建公告
func (r *AnnouncementRepo) Create(ann *models.Announcement) error {
	return r.db.Create(ann).Error
}

// Update 按字段更新公告
func (r *AnnouncementRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Announcement{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 软删除公告
func (r *AnnouncementRepo) Delete(id uint) error {
	return r.db.Delete(&models.Announcement{}, id).Error
}
