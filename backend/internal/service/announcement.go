package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 公告业务错误
var (
	ErrAnnouncementNotFound        = errors.New("公告不存在")
	ErrAnnouncementTitleRequired   = errors.New("公告标题不能为空")
	ErrAnnouncementContentRequired = errors.New("公告内容不能为空")
	ErrAnnouncementStatusInvalid   = errors.New("公告状态不合法")
)

// CreateAnnouncementRequest 创建公告请求
type CreateAnnouncementRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	IsPinned bool   `json:"is_pinned"`
}

// UpdateAnnouncementRequest 更新公告请求（指针字段区分「未传」与「置零」）
type UpdateAnnouncementRequest struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	IsPinned *bool   `json:"is_pinned"`
	Status   *string `json:"status"`
}

// AnnouncementItem 公告列表项（附带当前用户是否已读）
type AnnouncementItem struct {
	models.Announcement
	IsRead bool `json:"is_read"`
}

// AnnouncementService 公告业务服务
type AnnouncementService struct {
	repo *repository.AnnouncementRepo
}

// NewAnnouncementService 创建公告服务
func NewAnnouncementService(repo *repository.AnnouncementRepo) *AnnouncementService {
	return &AnnouncementService{repo: repo}
}

// List 公告分页列表：published 优先 → 置顶优先 → 时间倒序，支持 keyword 搜索标题
// 每条附带当前用户是否已读（is_read）
func (s *AnnouncementService) List(userID uint, keyword, status string, page, pageSize int) ([]AnnouncementItem, int64, error) {
	rows, total, err := s.repo.ListWithRead(userID, keyword, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	items := make([]AnnouncementItem, 0, len(rows))
	for i := range rows {
		items = append(items, AnnouncementItem{
			Announcement: rows[i].Announcement,
			IsRead:       rows[i].IsRead,
		})
	}
	return items, total, nil
}

// MarkRead 标记当前用户已读某公告（幂等 upsert）
func (s *AnnouncementService) MarkRead(userID, id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnnouncementNotFound
		}
		return err
	}
	return s.repo.MarkRead(userID, id)
}

// MarkAllRead 标记当前用户已读全部 published 公告（幂等）
func (s *AnnouncementService) MarkAllRead(userID uint) error {
	return s.repo.MarkAllRead(userID)
}

// UnreadCount 当前用户未读公告数
func (s *AnnouncementService) UnreadCount(userID uint) (int64, error) {
	return s.repo.UnreadCount(userID)
}

// Create 发布公告（默认 published，PublishedAt=now）
func (s *AnnouncementService) Create(publisherID uint, publisherName string, req CreateAnnouncementRequest) (*models.Announcement, error) {
	if req.Title == "" {
		return nil, ErrAnnouncementTitleRequired
	}
	if req.Content == "" {
		return nil, ErrAnnouncementContentRequired
	}
	now := time.Now()
	ann := &models.Announcement{
		Title:         req.Title,
		Content:       req.Content,
		PublisherID:   publisherID,
		PublisherName: publisherName,
		IsPinned:      req.IsPinned,
		Status:        "published",
		PublishedAt:   &now,
	}
	if err := s.repo.Create(ann); err != nil {
		return nil, err
	}
	return ann, nil
}

// Update 编辑公告（标题/内容/置顶/状态）
func (s *AnnouncementService) Update(id uint, req UpdateAnnouncementRequest) (*models.Announcement, error) {
	ann, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, err
	}
	fields := map[string]interface{}{}
	if req.Title != nil {
		if *req.Title == "" {
			return nil, ErrAnnouncementTitleRequired
		}
		fields["title"] = *req.Title
	}
	if req.Content != nil {
		if *req.Content == "" {
			return nil, ErrAnnouncementContentRequired
		}
		fields["content"] = *req.Content
	}
	if req.IsPinned != nil {
		fields["is_pinned"] = *req.IsPinned
	}
	if req.Status != nil {
		if *req.Status != "published" && *req.Status != "draft" {
			return nil, ErrAnnouncementStatusInvalid
		}
		fields["status"] = *req.Status
		// 转回 published 时刷新发布时间；改为 draft 时清空发布时间
		if *req.Status == "published" && ann.PublishedAt == nil {
			now := time.Now()
			fields["published_at"] = &now
		}
		if *req.Status == "draft" {
			fields["published_at"] = nil
		}
	}
	if len(fields) == 0 {
		return ann, nil
	}
	if err := s.repo.Update(id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// Delete 删除公告（软删除）
func (s *AnnouncementService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnnouncementNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}
