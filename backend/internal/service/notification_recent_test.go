package service

import (
	"testing"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// TestNotificationRecentUnread 最近未读通知增量接口：按 id 倒序、只含未读、limit 生效、
// 无未读返回空数组
func TestNotificationRecentUnread(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	user, err := createTestUser(tx, uniqueStr("recent_"), "pass123", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	repo := repository.NewNotificationRepo(tx)
	svc := NewNotificationService(repo, repository.NewEmployeeRepo(tx))

	// 直接落库 3 条未读 + 1 条已读，未读按创建先后 id 递增
	for i := 1; i <= 3; i++ {
		if err := repo.Create(&models.Notification{
			UserID:  user.ID,
			Type:    "system",
			Title:   uniqueStr("通知"),
			Content: "内容",
		}); err != nil {
			t.Fatalf("创建未读通知失败: %v", err)
		}
	}
	read := &models.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   uniqueStr("已读通知"),
		Content: "内容",
		IsRead:  true,
	}
	if err := repo.Create(read); err != nil {
		t.Fatalf("创建已读通知失败: %v", err)
	}

	// limit=5：应返回 3 条未读，id 严格递减
	items, err := svc.RecentUnread(user.ID, 5)
	if err != nil {
		t.Fatalf("RecentUnread 失败: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("期望 3 条未读，实际 %d", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i].ID >= items[i-1].ID {
			t.Errorf("通知应按 id 倒序，实际 [%d, %d]", items[i-1].ID, items[i].ID)
		}
	}
	for _, it := range items {
		if it.IsRead {
			t.Errorf("不应返回已读通知 id=%d", it.ID)
		}
	}

	// limit=2：只返回最新 2 条
	two, err := svc.RecentUnread(user.ID, 2)
	if err != nil {
		t.Fatalf("RecentUnread limit=2 失败: %v", err)
	}
	if len(two) != 2 {
		t.Fatalf("期望 2 条未读，实际 %d", len(two))
	}

	// 无未读用户：返回空数组而非 nil
	other, err := createTestUser(tx, uniqueStr("recent_empty_"), "pass123", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建空用户失败: %v", err)
	}
	empty, err := svc.RecentUnread(other.ID, 5)
	if err != nil {
		t.Fatalf("RecentUnread 空用户失败: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Errorf("无未读时应返回空数组，实际 %v", empty)
	}
}

// TestNotificationRecentUnreadInvalidLimit limit 非法值回退/封顶：0 → 默认 5，超大 → 50
func TestNotificationRecentUnreadInvalidLimit(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	user, err := createTestUser(tx, uniqueStr("recent_limit_"), "pass123", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	repo := repository.NewNotificationRepo(tx)
	svc := NewNotificationService(repo, repository.NewEmployeeRepo(tx))

	for range 6 {
		if err := repo.Create(&models.Notification{
			UserID:  user.ID,
			Type:    "system",
			Title:   uniqueStr("通知"),
			Content: "内容",
		}); err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	// limit<=0 回退默认 5
	def, err := svc.RecentUnread(user.ID, 0)
	if err != nil {
		t.Fatalf("RecentUnread(0) 失败: %v", err)
	}
	if len(def) != 5 {
		t.Errorf("limit=0 应回退为 5，实际 %d", len(def))
	}

	// limit 超大封顶 50（6 条全返回）
	cap, err := svc.RecentUnread(user.ID, 999)
	if err != nil {
		t.Fatalf("RecentUnread(999) 失败: %v", err)
	}
	if len(cap) != 6 {
		t.Errorf("limit=999 应封顶 50 并返回全部 6 条，实际 %d", len(cap))
	}
}
