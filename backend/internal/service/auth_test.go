package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/repository"
)

// newTestAuthService 构造认证服务（基于事务内的仓库）
func newTestAuthService(tx *gorm.DB) *AuthService {
	return NewAuthService(repository.NewUserRepo(tx), "test-secret", 24)
}

// TestLoginSuccess 正确密码 → 返回 token + user
func TestLoginSuccess(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "Passw0rd!", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	result, err := svc.Login(u.Username, "Passw0rd!")
	if err != nil {
		t.Fatalf("登录应成功: %v", err)
	}
	if result.Token == "" {
		t.Error("token 不应为空")
	}
	if result.User == nil || result.User.Username != u.Username {
		t.Errorf("返回的用户信息不匹配: %+v", result.User)
	}
	if result.User.RoleCode != "employee" {
		t.Errorf("角色编码期望 employee，实际 %s", result.User.RoleCode)
	}
}

// TestLoginWrongPassword 错误密码 → ErrInvalidCredentials
func TestLoginWrongPassword(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "Passw0rd!", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	if _, err := svc.Login(u.Username, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("错误密码应返回 ErrInvalidCredentials，实际: %v", err)
	}
}

// TestLoginDisabledUser 禁用账号（status=0）拒绝登录
func TestLoginDisabledUser(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "Passw0rd!", "employee", nil, 0)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	if _, err := svc.Login(u.Username, "Passw0rd!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("禁用账号登录应被拒绝，实际: %v", err)
	}
}

// TestChangePassword 修改密码后旧密码不可登录、新密码可登录
func TestChangePassword(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "OldPass123!", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	if err := svc.ChangePassword(u.ID, "OldPass123!", "NewPass456!"); err != nil {
		t.Fatalf("修改密码失败: %v", err)
	}
	if _, err := svc.Login(u.Username, "OldPass123!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("旧密码应无法登录，实际: %v", err)
	}
	if _, err := svc.Login(u.Username, "NewPass456!"); err != nil {
		t.Fatalf("新密码应可登录: %v", err)
	}
}
