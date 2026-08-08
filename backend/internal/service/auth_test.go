package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
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

// TestLoginLockoutAfterFiveFailures 连续失败 5 次触发锁定，第 6 次即使密码正确也被拒绝
func TestLoginLockoutAfterFiveFailures(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "Passw0rd!", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	// 前 4 次失败：仍返回 ErrInvalidCredentials
	for i := 1; i <= 4; i++ {
		if _, err := svc.Login(u.Username, "wrong-pass"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("第 %d 次失败应返回 ErrInvalidCredentials，实际: %v", i, err)
		}
	}
	// 第 5 次失败：触发锁定
	if _, err := svc.Login(u.Username, "wrong-pass"); !errors.Is(err, ErrLockTriggered) {
		t.Fatalf("第 5 次失败应触发锁定返回 ErrLockTriggered，实际: %v", err)
	}
	// 锁定状态已写入：LockedUntil 为未来时间、失败计数清零
	var stored models.User
	if err := tx.First(&stored, u.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if stored.LockedUntil == nil || !stored.LockedUntil.After(time.Now()) {
		t.Errorf("锁定后 LockedUntil 应为未来时间，实际 %v", stored.LockedUntil)
	}
	if stored.FailedLoginCount != 0 {
		t.Errorf("锁定后失败计数应清零，实际 %d", stored.FailedLoginCount)
	}
	// 第 6 次（正确密码）：锁定期间仍拒绝
	if _, err := svc.Login(u.Username, "Passw0rd!"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("锁定期间正确密码也应被拒绝，实际: %v", err)
	}
}

// TestLoginSuccessResetsFailures 登录成功后清零失败计数与锁定状态
func TestLoginSuccessResetsFailures(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	u, err := createTestUser(tx, uniqueStr("u"), "Passw0rd!", "employee", nil, 1)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	svc := newTestAuthService(tx)
	// 2 次失败后再正确登录
	for i := 0; i < 2; i++ {
		if _, err := svc.Login(u.Username, "wrong-pass"); err == nil {
			t.Fatalf("第 %d 次失败应报错", i+1)
		}
	}
	if _, err := svc.Login(u.Username, "Passw0rd!"); err != nil {
		t.Fatalf("正确密码登录应成功: %v", err)
	}
	var stored models.User
	if err := tx.First(&stored, u.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if stored.FailedLoginCount != 0 {
		t.Errorf("登录成功后失败计数应清零，实际 %d", stored.FailedLoginCount)
	}
	if stored.LockedUntil != nil {
		t.Errorf("登录成功后锁定状态应清空，实际 %v", stored.LockedUntil)
	}
}

// TestRefreshToken 用 refresh_token 可换取新的 token 对（双刷新、滚动续期）
func TestRefreshToken(t *testing.T) {
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
	if result.RefreshToken == "" {
		t.Fatal("登录响应应包含 refresh_token")
	}
	refreshed, err := svc.RefreshToken(result.RefreshToken)
	if err != nil {
		t.Fatalf("刷新令牌应成功: %v", err)
	}
	if refreshed.Token == "" || refreshed.RefreshToken == "" {
		t.Fatal("刷新后应返回新的 token 对")
	}
	if refreshed.RefreshToken == result.RefreshToken {
		t.Error("双刷新模式下 refresh_token 应轮换（新老不同）")
	}
}

// TestRefreshTokenRejectsAccessToken 用 access token 调刷新应被拒绝
func TestRefreshTokenRejectsAccessToken(t *testing.T) {
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
	if _, err := svc.RefreshToken(result.Token); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("access token 调刷新应返回 ErrInvalidRefreshToken，实际: %v", err)
	}
}
