package repository

import (
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// UserRepo 用户数据仓库，封装用户与关联员工的数据访问
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户仓库
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// FindByUsername 根据用户名查询用户（含角色）
func (r *UserRepo) FindByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Preload("Role").Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据 ID 查询用户（含角色）
func (r *UserRepo) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.Preload("Role").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindEmployeeByID 根据员工 ID 查询员工档案
func (r *UserRepo) FindEmployeeByID(id uint) (*models.Employee, error) {
	var emp models.Employee
	if err := r.db.First(&emp, id).Error; err != nil {
		return nil, err
	}
	return &emp, nil
}

// UpdatePassword 更新指定用户的密码哈希
func (r *UserRepo) UpdatePassword(userID uint, passwordHash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

// UpdateLoginLock 更新登录失败计数与锁定截止时间；lockedUntil 为 nil 表示仅更新计数不触发锁定
func (r *UserRepo) UpdateLoginLock(userID uint, failedCount int, lockedUntil *time.Time) error {
	fields := map[string]interface{}{"failed_login_count": failedCount}
	if lockedUntil != nil {
		fields["locked_until"] = lockedUntil
	}
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(fields).Error
}

// ResetLoginFailures 登录成功后清零失败计数与锁定状态
func (r *UserRepo) ResetLoginFailures(userID uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"failed_login_count": 0, "locked_until": nil}).Error
}

// UserItem 用户列表项（附带角色名称）
type UserItem struct {
	models.User
	RoleName string `json:"role_name"`
}

// List 用户分页列表，支持 keyword（username/full_name）/ role_id / status 过滤；status<0 表示不过滤
func (r *UserRepo) List(keyword string, roleID uint, status int, page, pageSize int) ([]UserItem, int64, error) {
	q := r.db.Model(&models.User{}).
		Joins("LEFT JOIN roles ON roles.id = users.role_id")
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("users.username LIKE ? OR users.full_name LIKE ?", kw, kw)
	}
	if roleID > 0 {
		q = q.Where("users.role_id = ?", roleID)
	}
	if status >= 0 {
		q = q.Where("users.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []UserItem
	err := q.Select("users.*, roles.name AS role_name").
		Order("users.id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByUsernameUnscoped 含软删除地查询用户名，用于创建/更新时的唯一性校验
func (r *UserRepo) FindByUsernameUnscoped(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Unscoped().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建用户
func (r *UserRepo) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update 按字段更新用户
func (r *UserRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除用户
func (r *UserRepo) SoftDelete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// CountByRoleID 统计指定角色下的用户数量（删除角色前校验用）
func (r *UserRepo) CountByRoleID(roleID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
