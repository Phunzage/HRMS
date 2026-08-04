package database

import (
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"hrms/backend/internal/config"
	"hrms/backend/internal/models"
)

// Connect 建立 PostgreSQL 连接
func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort,
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// AutoMigrate 自动建表（新增模型时在此追加）
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.RolePermission{},
	)
}

// Seed 写入种子数据（幂等：已存在则跳过）
func Seed(db *gorm.DB) error {
	// 内置角色
	roles := []models.Role{
		{Name: "管理员", Code: "admin", Description: "系统管理员，拥有全部权限"},
		{Name: "人事", Code: "hr", Description: "人事专员，负责人事/考勤/薪酬"},
		{Name: "经理", Code: "manager", Description: "部门经理，负责本部门审批"},
		{Name: "员工", Code: "employee", Description: "普通员工，自助服务"},
	}

	roleIDs := make(map[string]uint, len(roles))
	for _, role := range roles {
		var existing models.Role
		err := db.Where("code = ?", role.Code).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			// 不存在则创建
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("创建角色 %s 失败: %w", role.Code, err)
			}
			roleIDs[role.Code] = role.ID
			slog.Info("种子角色已创建", "code", role.Code, "name", role.Name)
		case err != nil:
			return fmt.Errorf("查询角色 %s 失败: %w", role.Code, err)
		default:
			// 已存在则跳过
			roleIDs[role.Code] = existing.ID
		}
	}

	// 内置管理员账号 admin/admin123（bcrypt 哈希存储）
	var adminCount int64
	if err := db.Model(&models.User{}).Where("username = ?", "admin").Count(&adminCount).Error; err != nil {
		return fmt.Errorf("查询管理员账号失败: %w", err)
	}
	if adminCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成密码哈希失败: %w", err)
		}
		admin := models.User{
			Username:     "admin",
			PasswordHash: string(hash),
			Email:        "admin@example.com",
			FullName:     "系统管理员",
			RoleID:       roleIDs["admin"],
			Status:       1,
		}
		if err := db.Create(&admin).Error; err != nil {
			return fmt.Errorf("创建管理员账号失败: %w", err)
		}
		slog.Info("种子管理员账号已创建", "username", "admin", "role", "admin")
	}

	return nil
}
