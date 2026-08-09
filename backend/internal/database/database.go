package database

import (
	"fmt"

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
		&models.Department{},
		&models.Position{},
		&models.Employee{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.LeaveBalance{},
		&models.SalaryRecord{},
		&models.JobOpening{},
		&models.Candidate{},
		&models.AuditLog{},
		&models.Notification{},
		&models.AttendanceCorrection{},
		&models.Announcement{},
		&models.AnnouncementRead{},
		&models.PerformanceReview{},
		&models.ApprovalFlow{},
		&models.ApprovalStep{},
		&models.ApprovalInstance{},
		&models.ApprovalTask{},
		&models.AppVersion{},
	)
}
