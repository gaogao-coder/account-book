package config

import "os"

// AppConfig 保存 API 服务启动所需的最小配置。
type AppConfig struct {
	Port string
}

// Load 从环境变量加载服务配置。
func Load() AppConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return AppConfig{Port: port}
}

// Addr 返回 Gin 监听地址。
func (c AppConfig) Addr() string {
	return ":" + c.Port
}

// FirstEnv 返回第一个非空环境变量值。
func FirstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
