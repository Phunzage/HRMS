package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"hrms/backend/internal/models"
)

// AnnouncementReadRow 公告列表行（附带当前用户是否已读）
type AnnouncementReadRow struct {
	models.Announcement
	IsRead bool `gorm:"column:is_read" json:"is_read"`
}

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
	items := make([]models.Announcement, 0)
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

// ListWithRead 公告分页列表（同 List 排序），附带当前用户是否已读（LEFT JOIN announcement_reads）
func (r *AnnouncementRepo) ListWithRead(userID uint, keyword, status string, page, pageSize int) ([]AnnouncementReadRow, int64, error) {
	var total int64
	if err := r.announcementQuery(keyword, status).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.db.Model(&models.Announcement{}).
		Select("announcements.*, CASE WHEN ar.id IS NULL THEN false ELSE true END AS is_read").
		Joins("LEFT JOIN announcement_reads ar ON ar.announcement_id = announcements.id AND ar.user_id = ?", userID)
	if status != "" {
		q = q.Where("announcements.status = ?", status)
	}
	if keyword != "" {
		q = q.Where("announcements.title LIKE ?", "%"+keyword+"%")
	}
	rows := make([]AnnouncementReadRow, 0)
	err := q.Order("CASE WHEN announcements.status = 'published' THEN 0 ELSE 1 END").
		Order("announcements.is_pinned DESC").
		Order("COALESCE(announcements.published_at, announcements.created_at) DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// MarkRead 标记某用户已读某公告（幂等 upsert，已读则忽略）
func (r *AnnouncementRepo) MarkRead(userID, announcementID uint) error {
	read := &models.AnnouncementRead{UserID: userID, AnnouncementID: announcementID}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "announcement_id"}},
		DoNothing: true,
	}).Create(read).Error
}

// MarkAllRead 标记当前用户已读全部 published 公告（幂等，已读跳过）
func (r *AnnouncementRepo) MarkAllRead(userID uint) error {
	return r.db.Exec(
		`INSERT INTO announcement_reads (user_id, announcement_id, created_at)
		 SELECT ?, id, ? FROM announcements
		 WHERE status = ? AND deleted_at IS NULL
		 ON CONFLICT (user_id, announcement_id) DO NOTHING`,
		userID, time.Now(), "published",
	).Error
}

// UnreadCount 当前用户未读公告数（published 总数 - 已读数）
func (r *AnnouncementRepo) UnreadCount(userID uint) (int64, error) {
	var n int64
	err := r.db.Model(&models.Announcement{}).
		Where("status = ? AND id NOT IN (?)", "published",
			r.db.Model(&models.AnnouncementRead{}).
				Select("announcement_id").
				Where("user_id = ?", userID)).
		Count(&n).Error
	return n, err
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
