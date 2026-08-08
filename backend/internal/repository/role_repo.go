package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// RoleItem 角色列表项（附带该角色的权限点 code 数组，前端用于勾选）
type RoleItem struct {
	models.Role
	Permissions []string `json:"permissions"`
}

// RoleRepo 角色数据仓库
type RoleRepo struct {
	db *gorm.DB
}

// NewRoleRepo 创建角色仓库
func NewRoleRepo(db *gorm.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

// ListAll 查询全部角色（供用户下拉选择，登录即可）
func (r *RoleRepo) ListAll() ([]models.Role, error) {
	var roles []models.Role
	if err := r.db.Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// List 角色列表（含权限点 code 数组）
func (r *RoleRepo) List() ([]RoleItem, error) {
	var roles []models.Role
	if err := r.db.Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	items := make([]RoleItem, 0, len(roles))
	for _, role := range roles {
		codes, err := r.findPermissionCodes(role.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, RoleItem{Role: role, Permissions: codes})
	}
	return items, nil
}

// ListPermissions 查询全部权限点（按模块、ID 排序，供前端分组勾选）
func (r *RoleRepo) ListPermissions() ([]models.Permission, error) {
	var perms []models.Permission
	if err := r.db.Order("module ASC, id ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// FindByID 根据 ID 查询角色
func (r *RoleRepo) FindByID(id uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindByCode 根据编码查询角色
func (r *RoleRepo) FindByCode(code string) (*models.Role, error) {
	var role models.Role
	if err := r.db.Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindByCodeUnscoped 含软删除地查询角色编码，用于创建时的唯一性校验
func (r *RoleRepo) FindByCodeUnscoped(code string) (*models.Role, error) {
	var role models.Role
	if err := r.db.Unscoped().Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindPermissionIDsByCodes 根据权限点编码批量查询 ID
func (r *RoleRepo) FindPermissionIDsByCodes(codes []string) ([]uint, error) {
	if len(codes) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	if err := r.db.Model(&models.Permission{}).
		Where("code IN ?", codes).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// CreateWithPermissions 事务内创建角色并关联权限
func (r *RoleRepo) CreateWithPermissions(role *models.Role, permissionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		return r.insertRolePermissions(tx, role.ID, permissionIDs)
	})
}

// UpdateWithPermissions 事务内更新角色信息并全量替换权限（先删后插）；
// permissionIDs 为 nil 时不改动权限，为空切片时清空该角色权限
func (r *RoleRepo) UpdateWithPermissions(id uint, fields map[string]interface{}, permissionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			if err := tx.Model(&models.Role{}).Where("id = ?", id).Updates(fields).Error; err != nil {
				return err
			}
		}
		if permissionIDs != nil {
			if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
				return err
			}
			if err := r.insertRolePermissions(tx, id, permissionIDs); err != nil {
				return err
			}
		}
		return nil
	})
}

// SoftDelete 软删除角色
func (r *RoleRepo) SoftDelete(id uint) error {
	return r.db.Delete(&models.Role{}, id).Error
}

// findPermissionCodes 查询角色拥有的权限点编码数组
func (r *RoleRepo) findPermissionCodes(roleID uint) ([]string, error) {
	var codes []string
	if err := r.db.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.id ASC").
		Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	if codes == nil {
		codes = []string{}
	}
	return codes, nil
}

// insertRolePermissions 批量插入角色-权限关联
func (r *RoleRepo) insertRolePermissions(tx *gorm.DB, roleID uint, permissionIDs []uint) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	rps := make([]models.RolePermission, 0, len(permissionIDs))
	for _, pid := range permissionIDs {
		rps = append(rps, models.RolePermission{RoleID: roleID, PermissionID: pid})
	}
	return tx.Create(&rps).Error
}
