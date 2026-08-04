package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"hrms/backend/internal/config"
	"hrms/backend/internal/database"
	"hrms/backend/internal/router"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 连接数据库
	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("数据库连接失败", "err", err)
		os.Exit(1)
	}

	// 3. 自动建表
	if err := database.AutoMigrate(db); err != nil {
		slog.Error("自动建表失败", "err", err)
		os.Exit(1)
	}

	// 4. 写入种子数据
	if err := database.Seed(db); err != nil {
		slog.Error("种子数据写入失败", "err", err)
		os.Exit(1)
	}

	// 5. 设置 Gin release 模式
	gin.SetMode(gin.ReleaseMode)

	// 6. 注册路由
	r := gin.New()
	router.Register(r, db)

	// 7. 启动服务
	addr := ":" + cfg.Port
	slog.Info("服务启动", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("服务启动失败", "err", err)
		os.Exit(1)
	}
}
