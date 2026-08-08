package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 登录安全加固常量：连续失败阈值、锁定时长、刷新令牌有效期
const (
	maxLoginFailures   = 5               // 连续失败次数达到该值触发锁定
	lockDuration       = 15 * time.Minute // 锁定持续时长
	refreshTokenExpire = 7 * 24 * time.Hour // 刷新令牌有效期 7 天
)

// 认证相关业务错误
var (
	ErrInvalidCredentials  = errors.New("用户名或密码错误")
	ErrUserNotFound        = errors.New("用户不存在")
	ErrOldPasswordWrong    = errors.New("旧密码不正确")
	ErrAccountLocked       = errors.New("账号已锁定，请 15 分钟后再试") // 账号处于锁定期间
	ErrLockTriggered       = errors.New("登录失败已触发账号锁定")       // 本次失败达到阈值触发了锁定（内部哨兵）
	ErrInvalidRefreshToken = errors.New("刷新令牌无效或已过期")
)

// AuthService 认证服务：登录签发 JWT、令牌刷新、当前用户信息、修改密码
type AuthService struct {
	repo        *repository.UserRepo
	jwtSecret   string
	tokenExpire time.Duration
}

// NewAuthService 创建认证服务
func NewAuthService(repo *repository.UserRepo, jwtSecret string, tokenExpireHours int) *AuthService {
	return &AuthService{
		repo:        repo,
		jwtSecret:   jwtSecret,
		tokenExpire: time.Duration(tokenExpireHours) * time.Hour,
	}
}

// UserBrief 用户简要信息（登录与用户信息通用）
type UserBrief struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	RoleCode   string `json:"role_code"`
	EmployeeID *uint  `json:"employee_id"`
}

// LoginResult 登录成功返回体：访问令牌 + 刷新令牌 + 有效期（秒）
type LoginResult struct {
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int64      `json:"expires_in"`
	User         *UserBrief `json:"user"`
}

// Login 用户名密码登录：先校验账号是否被锁定，再校验密码；
// 密码错误累加失败计数，达阈值触发锁定；成功后清零计数并签发 token 对。
func (s *AuthService) Login(username, password string) (*LoginResult, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	// 锁定期间直接拒绝登录（即使密码正确）
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}
	// 校验密码：失败则记录一次失败计数（达阈值触发锁定）
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, s.recordLoginFailure(user)
	}
	// 校验账号状态：禁用账号拒绝登录（与密码错误统一返回，避免暴露账号状态）
	if user.Status != 1 {
		return nil, ErrInvalidCredentials
	}
	// 登录成功：清零失败计数与锁定状态
	if err := s.repo.ResetLoginFailures(user.ID); err != nil {
		return nil, err
	}
	// 签发 token 对（access + refresh）
	return s.issueLoginResult(user)
}

// recordLoginFailure 累加一次登录失败计数；达到阈值时设置锁定（锁定 15 分钟）并清零计数
func (s *AuthService) recordLoginFailure(user *models.User) error {
	count := user.FailedLoginCount + 1
	if count >= maxLoginFailures {
		lockedUntil := time.Now().Add(lockDuration)
		if err := s.repo.UpdateLoginLock(user.ID, 0, &lockedUntil); err != nil {
			return err
		}
		return ErrLockTriggered
	}
	if err := s.repo.UpdateLoginLock(user.ID, count, nil); err != nil {
		return err
	}
	// 未达锁定阈值：对外仍返回通用的用户名/密码错误
	return ErrInvalidCredentials
}

// issueToken 按指定用途（access/refresh）与有效期签发 JWT。
// 每次签发带随机 jti，保证同一秒内签发的令牌也不重复（双刷新轮换的唯一性依赖于此）。
func (s *AuthService) issueToken(user *models.User, tokenType string, expire time.Duration) (string, error) {
	jti, err := randomHex(16)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		RoleCode: user.Role.Code,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hrms",
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
}

// issueLoginResult 为已校验用户签发 access + refresh 令牌对
func (s *AuthService) issueLoginResult(user *models.User) (*LoginResult, error) {
	accessToken, err := s.issueToken(user, "access", s.tokenExpire)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.issueToken(user, "refresh", refreshTokenExpire)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.tokenExpire.Seconds()),
		User: &UserBrief{
			ID:         user.ID,
			Username:   user.Username,
			FullName:   user.FullName,
			RoleCode:   user.Role.Code,
			EmployeeID: user.EmployeeID,
		},
	}, nil
}

// RefreshToken 用刷新令牌换取新的 token 对（双刷新、滚动续期）。
// 仅接受 type=refresh 且未过期的令牌；签发前重新校验用户仍处于启用状态。
func (s *AuthService) RefreshToken(refreshToken string) (*LoginResult, error) {
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid || claims.Type != "refresh" {
		return nil, ErrInvalidRefreshToken
	}
	// 重新加载用户：存在性 + 启用状态
	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.Status != 1 {
		return nil, ErrInvalidCredentials
	}
	return s.issueLoginResult(user)
}

// MeResult 当前用户信息（含角色与关联员工）
type MeResult struct {
	ID         uint             `json:"id"`
	Username   string           `json:"username"`
	FullName   string           `json:"full_name"`
	Email      string           `json:"email"`
	RoleCode   string           `json:"role_code"`
	EmployeeID *uint            `json:"employee_id"`
	Role       *models.Role     `json:"role,omitempty"`
	Employee   *models.Employee `json:"employee,omitempty"`
}

// Me 获取当前登录用户信息
func (s *AuthService) Me(userID uint) (*MeResult, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	result := &MeResult{
		ID:         user.ID,
		Username:   user.Username,
		FullName:   user.FullName,
		Email:      user.Email,
		RoleCode:   user.Role.Code,
		EmployeeID: user.EmployeeID,
		Role:       user.Role,
	}
	if user.EmployeeID != nil {
		if emp, err := s.repo.FindEmployeeByID(*user.EmployeeID); err == nil {
			result.Employee = emp
		}
	}
	return result, nil
}

// ChangePassword 校验旧密码后，将密码更新为新密码（bcrypt 存储）
func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrOldPasswordWrong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(userID, string(hash))
}
