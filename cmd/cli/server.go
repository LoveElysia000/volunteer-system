package cli

import (
	"fmt"
	"log"
	"time"

	"volunteer-system/config"
	"volunteer-system/internal/router"
	"volunteer-system/pkg/database/mysql"
	"volunteer-system/pkg/database/redis"
	"volunteer-system/pkg/logger"

	hz "github.com/cloudwego/hertz/pkg/app/server"
)

// StartServer 启动HTTP服务器
func StartServer() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化日志
	if cfg.Logging != nil {
		if err := logger.Init(cfg.Logging.Level, cfg.Logging.Console, cfg.Logging.File); err != nil {
			log.Fatalf("日志器初始化失败: %v", err)
		}
	}
	appLog := logger.GetLogger()
	appLog.Info("应用启动: %s (环境: %s)", cfg.App.Name, cfg.App.Env)

	// 初始化数据库连接
	if err := initDatabases(&cfg); err != nil {
		appLog.Error("数据库初始化失败: %v", err)
		log.Fatalf("无法启动应用")
	}
	defer closeDatabases()

	// 启动HTTP服务器
	initHttpServer(&cfg)
}

// initDatabases 初始化数据库连接
func initDatabases(cfg *config.Config) error {
	appLog := logger.GetLogger()

	// 初始化MySQL
	if cfg.MySQL != nil {
		if _, err := mysql.InitMySQL(cfg.MySQL); err != nil {
			appLog.Error("MySQL初始化失败: %v", err)
			return fmt.Errorf("MySQL初始化失败: %v", err)
		}
		appLog.Info("MySQL数据库初始化成功")
	}

	// 初始化Redis（如果不可用则记录警告，但不阻止启动）
	if cfg.Redis != nil {
		if _, err := redis.InitRedis(cfg.Redis); err != nil {
			appLog.Warn("Redis初始化失败: %v (应用将继续运行)", err)
		} else {
			appLog.Info("Redis初始化成功")
		}
	}

	return nil
}

// closeDatabases 关闭数据库连接
func closeDatabases() {
	appLog := logger.GetLogger()

	if err := mysql.CloseMySQL(); err != nil {
		appLog.Error("关闭MySQL连接失败: %v", err)
	}

	if err := redis.CloseRedis(); err != nil {
		appLog.Error("关闭Redis连接失败: %v", err)
	}

	appLog.Info("数据库连接已关闭")
}

func initHttpServer(cfg *config.Config) {
	appLog := logger.GetLogger()

	// 创建Hertz服务器配置
	h := hz.Default(
		hz.WithHostPorts(fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)),
		hz.WithReadTimeout(10*time.Second),
		hz.WithWriteTimeout(10*time.Second),
		hz.WithIdleTimeout(60*time.Second),
	)

	appLog.Info("Hertz服务器启动在 %s:%d", cfg.App.Host, cfg.App.Port)

	router.RegisterRouter(h)
	// 启动服务器
	h.Spin()
}