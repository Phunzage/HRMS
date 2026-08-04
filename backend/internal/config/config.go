package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	Port             string // HTTP 监听端口
	DBHost           string // 数据库地址
	DBPort           string // 数据库端口
	DBUser           string // 数据库用户
	DBPassword       string // 数据库密码
	DBName           string // 数据库名
	JWTSecret        string // JWT 签名密钥
	TokenExpireHours int    // Token 过期时间（小时）
}

// Load 从 .env 加载配置，缺失项使用默认值
func Load() *Config {
	// 读取 .env（文件不存在时忽略错误，使用环境变量或默认值）
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env 文件未找到，使用环境变量或默认值")
	}

	return &Config{
		Port:             getEnv("PORT", "8080"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "3143"),
		DBName:           getEnv("DB_NAME", "hrms"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret-change-me"),
		TokenExpireHours: getEnvInt("TOKEN_EXPIRE_HOURS", 24),
	}
}

// getEnv 读取环境变量，为空时返回默认值
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getEnvInt 读取环境变量并转为 int，失败时返回默认值
func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
