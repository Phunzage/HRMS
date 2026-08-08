package service

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/config"
	"hrms/backend/internal/database"
	"hrms/backend/internal/models"
)

// 测试库连接与初始化（惰性初始化一次）
var (
	testDBOnce sync.Once
	testDBConn *gorm.DB
	testDBErr  error
)

// TestMain 包级入口：初始化测试库，初始化失败则整个包直接退出
func TestMain(m *testing.M) {
	if _, err := testDB(); err != nil {
		fmt.Fprintf(os.Stderr, "测试数据库初始化失败: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// testDB 连接测试库并完成初始化（AutoMigrate + 基础角色/部门/岗位），全程只执行一次
func testDB() (*gorm.DB, error) {
	testDBOnce.Do(func() {
		cfg := config.Load()
		// 测试库名从 TEST_DB_NAME 读取，默认 hrms_test；其余连接参数与开发库一致（复用 config 默认值）
		cfg.DBName = os.Getenv("TEST_DB_NAME")
		if cfg.DBName == "" {
			cfg.DBName = "hrms_test"
		}
		testDBConn, testDBErr = database.Connect(cfg)
		if testDBErr != nil {
			return
		}
		// 只建表不跑 Seed，测试内自建数据保证隔离
		if err := database.AutoMigrate(testDBConn); err != nil {
			testDBErr = err
			return
		}
		if err := seedTestBasics(testDBConn); err != nil {
			testDBErr = err
		}
	})
	return testDBConn, testDBErr
}

// mustTestDB 获取测试库连接，失败直接终止测试
func mustTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := testDB()
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}
	return db
}

// seedTestBasics 写入测试库基础数据：内置角色 + 共享部门/岗位（并发工号测试用），幂等
func seedTestBasics(db *gorm.DB) error {
	roles := []models.Role{
		{Name: "管理员", Code: "admin", Description: "系统管理员"},
		{Name: "人事", Code: "hr", Description: "人事"},
		{Name: "经理", Code: "manager", Description: "部门经理"},
		{Name: "员工", Code: "employee", Description: "普通员工"},
	}
	for _, r := range roles {
		var existing models.Role
		err := db.Where("code = ?", r.Code).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&r).Error; err != nil {
				return fmt.Errorf("创建测试角色 %s 失败: %w", r.Code, err)
			}
		case err != nil:
			return fmt.Errorf("查询测试角色 %s 失败: %w", r.Code, err)
		}
	}
	// 共享部门：并发工号测试等需要提交状态的部门/岗位
	var dept models.Department
	err := db.Where("name = ?", "测试技术部").First(&dept).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		dept = models.Department{Name: "测试技术部", SortOrder: 99}
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建测试部门失败: %w", err)
		}
	case err != nil:
		return fmt.Errorf("查询测试部门失败: %w", err)
	}
	var pos models.Position
	err = db.Where("code = ?", "test_engineer").First(&pos).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		pos = models.Position{Name: "测试工程师", Code: "test_engineer", DepartmentID: dept.ID}
		if err := db.Create(&pos).Error; err != nil {
			return fmt.Errorf("创建测试岗位失败: %w", err)
		}
	case err != nil:
		return fmt.Errorf("查询测试岗位失败: %w", err)
	}
	return nil
}
