package redis

import (
	"context"
	"fmt"
	"time"

	"volunteer-system/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// RedisClient 全局Redis客户端实例
var RedisClient *redis.Client

// InitRedis 初始化Redis连接
func InitRedis(conf *RedisConfig) (*redis.Client, error) {

	// 创建Redis客户端配置
	redisOpts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Password:     conf.Password,
		DB:           conf.DB,
		DialTimeout:  time.Duration(conf.ConnectTimeout) * time.Millisecond,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10, // 连接池大小
		MinIdleConns: 2,  // 最小空闲连接数
		MaxRetries:   3,  // 最大重试次数
	}

	// 创建Redis客户端
	client := redis.NewClient(redisOpts)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接测试失败: %v", err)
	}

	// 设置全局Redis客户端实例
	RedisClient = client

	if log := logger.GetLogger(); log != nil {
		log.Info("Redis连接成功: %s:%d (DB: %d)",
			conf.Host,
			conf.Port,
			conf.DB)
	}

	return client, nil
}

// GetRedis 获取Redis客户端实例
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseRedis 关闭Redis连接
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
