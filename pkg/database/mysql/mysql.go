package mysql

import (
	"context"
	"fmt"
	"time"

	"volunteer-system/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB 全局GORM数据库连接实例
var DB *gorm.DB

// InitMySQL 初始化MySQL数据库连接（使用GORM）
func InitMySQL(conf *MySQLConfig) (*gorm.DB, error) {

	// 构建DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Database,
		conf.Charset,
	)

	// 配置GORM
	gormConfig := &gorm.Config{
		// 禁用默认事务（对于写操作可以提升性能）
		SkipDefaultTransaction: true,
		// 配置自定义日志
		Logger: logger.NewGormLogger(),
		// 禁用自动为表名添加复数形式
		// NamingStrategy: gorm.NamingStrategy{
		// 	SingularTable: true,
		// },
	}

	// 打开数据库连接
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 获取底层的sql.DB连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(conf.Pool.Max)                                            // 最大连接数
	sqlDB.SetMaxIdleConns(conf.Pool.Min)                                            // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Duration(conf.Pool.IdleMs) * time.Millisecond)    // 连接最大生存时间
	sqlDB.SetConnMaxIdleTime(time.Duration(conf.Pool.AcquireMs) * time.Millisecond) // 连接最大空闲时间

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 设置全局数据库实例
	DB = db

	if log := logger.GetLogger(); log != nil {
		log.Info("MySQL数据库连接成功: %s:%d/%s",
			conf.Host,
			conf.Port,
			conf.Database)
	}

	return db, nil
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// GetDBWithContext 获取带上下文的数据库连接实例
func GetDBWithContext(ctx context.Context, c *app.RequestContext) *gorm.DB {
	return DB.WithContext(ctx)
}

// CloseMySQL 关闭数据库连接
func CloseMySQL() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
