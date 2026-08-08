package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 角色管理业务错误
var (
	ErrRoleNotFound      = errors.New("角色不存在")
	ErrRoleCodeEmpty     = errors.New("角色编码不能为空")
	ErrRoleCodeExists    = errors.New("角色编码已存在")
	ErrRoleHasUsers      = errors.New("该角色下存在用户，无法删除")
	ErrRoleAdminNotDel   = errors.New("admin 角色不可删除")
	ErrPermissionInvalid = errors.New("存在无效的权限点")
)

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Code        string   `json:"code" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest 更新角色请求（Permissions 传空数组清空权限，不传则保持原权限）
type UpdateRoleRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

// PermissionGroup 权限点分组项（模块 → 权限点数组）
type PermissionGroup struct {
	Module string              `json:"module"`
	Items  []models.Permission `json:"items"`
}

// RoleService 角色管理服务
type RoleService struct {
	roleRepo *repository.RoleRepo
	userRepo *repository.UserRepo
}

// NewRoleService 创建角色管理服务
func NewRoleService(roleRepo *repository.RoleRepo, userRepo *repository.UserRepo) *RoleService {
	return &RoleService{roleRepo: roleRepo, userRepo: userRepo}
}

// List 角色列表（含权限点 code 数组）
func (s *RoleService) List() ([]repository.RoleItem, error) {
	return s.roleRepo.List()
}

// GetByID 查询角色，供处理器拼接审计描述等场景使用
func (s *RoleService) GetByID(id uint) (*models.Role, error) {
	return s.roleRepo.FindByID(id)
}

// PermissionTree 全部权限点，按模块分组返回
func (s *RoleService) PermissionTree() ([]PermissionGroup, error) {
	perms, err := s.roleRepo.ListPermissions()
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]models.Permission)
	order := make([]string, 0)
	for _, p := range perms {
		if _, ok := grouped[p.Module]; !ok {
			order = append(order, p.Module)
		}
		grouped[p.Module] = append(grouped[p.Module], p)
	}
	result := make([]PermissionGroup, 0, len(order))
	for _, mod := range order {
		result = append(result, PermissionGroup{Module: mod, Items: grouped[mod]})
	}
	return result, nil
}

// Create 创建角色：编码唯一校验（含软删除），permissions 关联写入
func (s *RoleService) Create(req CreateRoleRequest) (*models.Role, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, ErrRoleCodeEmpty
	}
	if _, err := s.roleRepo.FindByCodeUnscoped(code); err == nil {
		return nil, ErrRoleCodeExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	permIDs, err := s.resolvePermissionIDs(req.Permissions)
	if err != nil {
		return nil, err
	}
	role := &models.Role{
		Name:        req.Name,
		Code:        code,
		Description: req.Description,
	}
	if err := s.roleRepo.CreateWithPermissions(role, permIDs); err != nil {
		return nil, err
	}
	return role, nil
}

// Update 更新角色名称/描述/权限；permissions 全量替换（先删后插，事务）
func (s *RoleService) Update(id uint, req UpdateRoleRequest) error {
	if _, err := s.roleRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	var permIDs []uint
	if req.Permissions != nil {
		var err error
		permIDs, err = s.resolvePermissionIDs(req.Permissions)
		if err != nil {
			return err
		}
	}
	return s.roleRepo.UpdateWithPermissions(id, fields, permIDs)
}

// Delete 删除角色：admin 角色不可删，角色下有用户时拒绝
func (s *RoleService) Delete(id uint) error {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if role.Code == "admin" {
		return ErrRoleAdminNotDel
	}
	count, err := s.userRepo.CountByRoleID(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleHasUsers
	}
	return s.roleRepo.SoftDelete(id)
}

// resolvePermissionIDs 将权限点编码解析为 ID，并校验全部编码均有效
func (s *RoleService) resolvePermissionIDs(codes []string) ([]uint, error) {
	if len(codes) == 0 {
		return []uint{}, nil
	}
	ids, err := s.roleRepo.FindPermissionIDsByCodes(codes)
	if err != nil {
		return nil, err
	}
	if len(ids) != len(codes) {
		return nil, ErrPermissionInvalid
	}
	return ids, nil
}
