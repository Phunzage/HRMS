package service

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 通知类型常量
const (
	NotifTypeLeaveApproval  = "leave_approval"  // 请假待审批（给审批人）
	NotifTypeLeaveResult    = "leave_result"    // 请假审批结果（给申请人）
	NotifTypeAttCorrection  = "attendance_correction" // 补卡审批结果（给申请人）
	NotifTypeAnnouncement   = "announcement"    // 公告
	NotifTypeSystem         = "system"          // 系统消息
)

// 通知业务错误
var (
	ErrNotifNotFound = errors.New("通知不存在")
	ErrNotifNotOwner = errors.New("无权操作该通知")
)

// NotificationService 站内通知业务服务
type NotificationService struct {
	repo    *repository.NotificationRepo
	empRepo *repository.EmployeeRepo
}

// NewNotificationService 创建通知服务
func NewNotificationService(repo *repository.NotificationRepo, empRepo *repository.EmployeeRepo) *NotificationService {
	return &NotificationService{repo: repo, empRepo: empRepo}
}

// Send 创建通知（供其他服务调用）；userID 为 0 时静默跳过
func (s *NotificationService) Send(userID uint, ntype, title, content, link string) error {
	if userID == 0 {
		return nil
	}
	return s.repo.Create(&models.Notification{
		UserID:  userID,
		Type:    ntype,
		Title:   title,
		Content: content,
		Link:    link,
	})
}

// SendToEmployee 向员工关联的系统账号发送通知；
// 员工不存在或未关联账号时静默跳过（审批人可能无账号）
func (s *NotificationService) SendToEmployee(employeeID uint, ntype, title, content, link string) error {
	userID, err := s.empRepo.FindUserIDByEmployeeID(employeeID)
	if err != nil || userID == nil {
		return nil
	}
	return s.Send(*userID, ntype, title, content, link)
}

// List 当前用户通知分页列表；isRead 为 nil 表示不过滤
func (s *NotificationService) List(userID uint, isRead *bool, page, pageSize int) ([]models.Notification, int64, error) {
	return s.repo.List(userID, isRead, page, pageSize)
}

// UnreadCount 当前用户未读通知数量
func (s *NotificationService) UnreadCount(userID uint) (int64, error) {
	return s.repo.UnreadCount(userID)
}

// MarkRead 标记单条通知已读，校验归属当前用户
func (s *NotificationService) MarkRead(userID, id uint) error {
	n, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotifNotFound
		}
		return err
	}
	if n.UserID != userID {
		return ErrNotifNotOwner
	}
	_, err = s.repo.MarkRead(userID, id)
	return err
}

// MarkAllRead 当前用户全部通知标记已读
func (s *NotificationService) MarkAllRead(userID uint) error {
	return s.repo.MarkAllRead(userID)
}
