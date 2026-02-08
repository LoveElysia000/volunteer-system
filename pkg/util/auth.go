package util

import (
	"volunteer-system/config"
	"volunteer-system/pkg/auth"
	"volunteer-system/pkg/database/redis"
)

// GetJWTManager 获取JWT管理器实例
func GetJWTManager() *auth.Manager {
	cfg := config.GetConfig()
	if cfg == nil || cfg.Auth == nil {
		panic("JWT配置未找到")
	}

	jwtSecret := cfg.Auth.JWT.Secret
	if jwtSecret == "" {
		panic("JWT密钥未配置")
	}

	redisClient := redis.GetRedis()
	return auth.NewManager(jwtSecret, redisClient)
}