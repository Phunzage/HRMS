package service

import (
	"encoding/json"
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

// OptionalUint 可区分「未传」与「显式 null」的可空 uint 字段。
// 以值类型嵌入请求体：key 未出现时 Set=false；key 出现（含 null）时必然调用 UnmarshalJSON 置 Set=true，
// Value 为 nil 表示显式置空，非 nil 表示指定值。标准 *uint 无法区分「未传」与「显式 null」（二者均为 nil）。
type OptionalUint struct {
	Set   bool
	Value *uint
}

// UnmarshalJSON 实现 json.Unmarshaler：null 置 Set=true 且 Value=nil；数字解析为值
func (o *OptionalUint) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v uint
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// UpdateUserRequest 更新用户请求（指针字段区分「未传」与「置空」）
// DeptScopeID 用 OptionalUint 区分三种状态：未传 / 显式 null 清空 / 指定部门 ID
type UpdateUserRequest struct {
	FullName    *string      `json:"full_name"`
	Email       *string      `json:"email"`
	RoleID      *uint        `json:"role_id"`
	Status      *int         `json:"status"`
	DeptScopeID OptionalUint `json:"dept_scope_id"`
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
	if req.DeptScopeID.Set {
		if req.DeptScopeID.Value == nil {
			// 显式传 null：清空数据范围（回退全量）
			fields["dept_scope_id"] = nil
		} else {
			fields["dept_scope_id"] = *req.DeptScopeID.Value
		}
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
