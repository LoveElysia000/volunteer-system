package config

import (
	"fmt"
	"log"

	"volunteer-system/pkg/database/mysql"
	"volunteer-system/pkg/database/redis"

	"github.com/spf13/viper"
)

// AppConfig 应用基础配置
type AppConfig struct {
	Name      string `mapstructure:"name"`
	Env       string `mapstructure:"env"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Timezone  string `mapstructure:"timezone"`
	SecretKey string `mapstructure:"secret_key"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled bool `mapstructure:"enabled"`
	SMTP    struct {
		Host   string `mapstructure:"host"`
		Port   int    `mapstructure:"port"`
		Secure bool   `mapstructure:"secure"`
		User   string `mapstructure:"user"`
		Pass   string `mapstructure:"pass"`
	} `mapstructure:"smtp"`
	From struct {
		Name    string `mapstructure:"name"`
		Address string `mapstructure:"address"`
	} `mapstructure:"from"`
}

// UploadConfig 文件上传配置
type UploadConfig struct {
	Dir               string   `mapstructure:"dir"`
	MaxFileSizeMB     int      `mapstructure:"max_file_size_mb"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	Console  bool   `mapstructure:"console"`
	File     string `mapstructure:"file"`
	Rotation struct {
		Enabled   bool `mapstructure:"enabled"`
		MaxSizeMB int  `mapstructure:"max_size_mb"`
		MaxFiles  int  `mapstructure:"max_files"`
	} `mapstructure:"rotation"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWT struct {
		Secret               string `mapstructure:"secret"`
		AccessExpiryMinutes  int    `mapstructure:"access_expiry_minutes"`
		RefreshExpiryDays    int    `mapstructure:"refresh_expiry_days"`
		Issuer               string `mapstructure:"issuer"`
		MaxConcurrentTokens  int    `mapstructure:"max_concurrent_tokens"`
		TokenRotationEnabled bool   `mapstructure:"token_rotation_enabled"`
		AnomalyDetection     struct {
			Enabled                  bool `mapstructure:"enabled"`
			SuspiciousLoginThreshold int  `mapstructure:"suspicious_login_threshold"`
			RefreshRateLimit         int  `mapstructure:"refresh_rate_limit"`
			RefreshWindowMinutes     int  `mapstructure:"refresh_window_minutes"`
		} `mapstructure:"anomaly_detection"`
	} `mapstructure:"jwt"`
}

// Config 完整的配置结构
type Config struct {
	App     AppConfig          `mapstructure:"app"`
	MySQL   *mysql.MySQLConfig `mapstructure:"mysql"`
	Redis   *redis.RedisConfig `mapstructure:"redis"`
	Email   *EmailConfig       `mapstructure:"email"`
	Upload  *UploadConfig      `mapstructure:"upload"`
	Logging *LoggingConfig     `mapstructure:"logging"`
	Auth    *AuthConfig        `mapstructure:"auth"`
}

var conf Config

func LoadConfig() Config {
	// 设置 Viper 配置
	viper.SetConfigName("config")   // 配置文件名称 (不需要扩展名)
	viper.SetConfigType("yaml")     // 配置文件类型
	viper.AddConfigPath("./config") // 配置文件路径
	viper.AddConfigPath(".")        // 当前目录

	// 设置环境变量前缀
	viper.SetEnvPrefix("VOLUNTEER")
	viper.AutomaticEnv() // 自动读取环境变量

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal("配置文件未找到: config.yaml")
		} else {
			log.Fatalf("读取配置文件错误: %v", err)
		}
	}

	// 将配置绑定到结构体
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("配置解析错误: %v", err)
	}

	// 设置全局配置变量
	conf = config

	// 打印加载的配置信息（开发环境）
	if config.App.Env == "development" {
		fmt.Printf("配置文件加载成功: %s\n", viper.ConfigFileUsed())
		fmt.Printf("应用名称: %s\n", config.App.Name)
		fmt.Printf("环境: %s\n", config.App.Env)
		fmt.Printf(
			"服务地址: %s:%d\n",
			config.App.Host,
			config.App.Port,
		)
	}

	return config
}

func GetConfig() *Config {
	return &conf
}
