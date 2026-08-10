package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// NotificationRepo 站内通知数据仓库
type NotificationRepo struct {
	db *gorm.DB
}

// NewNotificationRepo 创建通知仓库
func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Create 创建通知
func (r *NotificationRepo) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

// UserEnabled 查询系统账号是否启用（status=1）；账号不存在或查询失败返回 false
func (r *NotificationRepo) UserEnabled(userID uint) bool {
	var user models.User
	if err := r.db.Select("status").First(&user, userID).Error; err != nil {
		return false
	}
	return user.Status == 1
}

// FindByID 根据 ID 查询通知
func (r *NotificationRepo) FindByID(id uint) (*models.Notification, error) {
	var n models.Notification
	if err := r.db.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// List 分页查询某用户的全部通知；isRead 为 nil 表示不过滤
func (r *NotificationRepo) List(userID uint, isRead *bool, page, pageSize int) ([]models.Notification, int64, error) {
	q := r.db.Model(&models.Notification{}).Where("user_id = ?", userID)
	if isRead != nil {
		q = q.Where("is_read = ?", *isRead)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]models.Notification, 0)
	err := q.Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// RecentUnread 查询某用户最近 N 条未读通知（按 id 倒序），供客户端增量检测新通知
func (r *NotificationRepo) RecentUnread(userID uint, limit int) ([]models.Notification, error) {
	items := make([]models.Notification, 0)
	err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Order("id DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// UnreadCount 查询某用户未读通知数量
func (r *NotificationRepo) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkRead 将某条通知标记为已读；返回是否成功更新（不存在或非本人返回 false）
func (r *NotificationRepo) MarkRead(userID, id uint) (bool, error) {
	res := r.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkAllRead 将某用户全部通知标记为已读
func (r *NotificationRepo) MarkAllRead(userID uint) error {
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
