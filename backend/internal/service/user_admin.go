package service

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 用户管理业务错误
var (
	ErrUsernameExists   = errors.New("用户名已存在")
	ErrUsernameRequired = errors.New("用户名不能为空")
	ErrPasswordTooShort = errors.New("密码长度不能少于6位")
	ErrCannotDeleteSelf = errors.New("不能删除当前登录账号")
	ErrCannotDisableSelf = errors.New("不能禁用当前登录账号")
)

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	RoleID   uint   `json:"role_id" binding:"required"`
}

// UpdateUserRequest 更新用户请求（指针字段区分「未传」与「置空」）
type UpdateUserRequest struct {
	FullName *string `json:"full_name"`
	Email    *string `json:"email"`
	RoleID   *uint   `json:"role_id"`
	Status   *int    `json:"status"`
}

// UserService 用户管理服务
type UserService struct {
	userRepo *repository.UserRepo
	roleRepo *repository.RoleRepo
}

// NewUserService 创建用户管理服务
func NewUserService(userRepo *repository.UserRepo, roleRepo *repository.RoleRepo) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo}
}

// List 用户分页列表
func (s *UserService) List(keyword string, roleID uint, status int, page, pageSize int) ([]repository.UserItem, int64, error) {
	return s.userRepo.List(keyword, roleID, status, page, pageSize)
}

// GetByID 查询用户（含角色），供处理器拼接审计描述等场景使用
func (s *UserService) GetByID(id uint) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

// Create 创建用户：校验用户名唯一（含软删除）与角色存在，bcrypt 存储密码
func (s *UserService) Create(req CreateUserRequest) (*models.User, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, ErrUsernameRequired
	}
	if len(req.Password) < 6 {
		return nil, ErrPasswordTooShort
	}
	if _, err := s.userRepo.FindByUsernameUnscoped(username); err == nil {
		return nil, ErrUsernameExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if _, err := s.roleRepo.FindByID(req.RoleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Username:     username,
		PasswordHash: string(hash),
		Email:        req.Email,
		FullName:     req.FullName,
		RoleID:       req.RoleID,
		Status:       1,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Update 更新用户基本信息：校验角色存在；禁止将当前登录账号自己禁用
func (s *UserService) Update(id, currentUserID uint, req UpdateUserRequest) error {
	if _, err := s.userRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	if req.FullName != nil {
		fields["full_name"] = *req.FullName
	}
	if req.Email != nil {
		fields["email"] = *req.Email
	}
	if req.RoleID != nil {
		if _, err := s.roleRepo.FindByID(*req.RoleID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleNotFound
			}
			return err
		}
		fields["role_id"] = *req.RoleID
	}
	if req.Status != nil {
		if id == currentUserID && *req.Status == 0 {
			return ErrCannotDisableSelf
		}
		fields["status"] = *req.Status
	}
	if len(fields) == 0 {
		return nil
	}
	return s.userRepo.Update(id, fields)
}

// ResetPassword 重置指定用户的密码（bcrypt 存储）
func (s *UserService) ResetPassword(id uint, password string) error {
	if len(password) < 6 {
		return ErrPasswordTooShort
	}
	if _, err := s.userRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(id, string(hash))
}

// Delete 软删除用户：禁止删除当前登录账号自己
func (s *UserService) Delete(id, currentUserID uint) error {
	if id == currentUserID {
		return ErrCannotDeleteSelf
	}
	if _, err := s.userRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return s.userRepo.SoftDelete(id)
}

// ListRoles 查询全部角色（供前端下拉选择，仅需登录）
func (s *UserService) ListRoles() ([]models.Role, error) {
	return s.roleRepo.ListAll()
}
