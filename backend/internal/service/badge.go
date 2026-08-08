package service

import (
	"gorm.io/gorm"

	"hrms/backend/internal/repository"
)

// BadgeCounts 导航栏角标汇总
type BadgeCounts struct {
	PendingApprovals    int64 `json:"pending_approvals"`    // 待我审批任务数
	UnreadAnnouncements int64 `json:"unread_announcements"` // 未读公告数
	UnreadNotifications int64 `json:"unread_notifications"` // 未读站内通知数
}

// BadgeService 角标汇总服务（导航栏红点/数字角标）
type BadgeService struct {
	db               *gorm.DB // 审批任务统计用
	announcementRepo *repository.AnnouncementRepo
	notificationRepo *repository.NotificationRepo
}

// NewBadgeService 创建角标汇总服务
func NewBadgeService(db *gorm.DB, announcementRepo *repository.AnnouncementRepo, notificationRepo *repository.NotificationRepo) *BadgeService {
	return &BadgeService{db: db, announcementRepo: announcementRepo, notificationRepo: notificationRepo}
}

// Summary 汇总当前用户导航栏角标：
// pending_approvals 复用审批任务归属口径（ApproverID 匹配或 HR 池任务）；
// unread_announcements 复用公告未读数；unread_notifications 复用通知未读数
func (s *BadgeService) Summary(userID uint) (*BadgeCounts, error) {
	pending, err := CountMyPendingTasks(s.db, userID)
	if err != nil {
		return nil, err
	}
	unreadAnn, err := s.announcementRepo.UnreadCount(userID)
	if err != nil {
		return nil, err
	}
	unreadNotif, err := s.notificationRepo.UnreadCount(userID)
	if err != nil {
		return nil, err
	}
	return &BadgeCounts{
		PendingApprovals:    pending,
		UnreadAnnouncements: unreadAnn,
		UnreadNotifications: unreadNotif,
	}, nil
}
