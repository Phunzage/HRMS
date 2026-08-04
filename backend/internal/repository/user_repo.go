package repository

import (
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
