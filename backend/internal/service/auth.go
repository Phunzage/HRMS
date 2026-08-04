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

// 认证相关业务错误
var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrOldPasswordWrong   = errors.New("旧密码不正确")
)

// AuthService 认证服务：登录签发 JWT、当前用户信息、修改密码
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

// LoginResult 登录成功返回体
type LoginResult struct {
	Token string     `json:"token"`
	User  *UserBrief `json:"user"`
}

// Login 用户名密码登录，校验通过后签发 JWT
func (s *AuthService) Login(username, password string) (*LoginResult, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	// 签发 JWT
	now := time.Now()
	claims := middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		RoleCode: user.Role.Code,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hrms",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpire)),
		},
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		Token: tokenStr,
		User: &UserBrief{
			ID:         user.ID,
			Username:   user.Username,
			FullName:   user.FullName,
			RoleCode:   user.Role.Code,
			EmployeeID: user.EmployeeID,
		},
	}, nil
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
