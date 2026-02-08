package redis

// RedisConfig Redis配置
type RedisConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Password        string `mapstructure:"password"`
	DB              int    `mapstructure:"db"`
	KeyPrefix       string `mapstructure:"key_prefix"`
	ConnectTimeout  int    `mapstructure:"connect_timeout_ms"`
	CacheTTLSeconds int    `mapstructure:"cache_ttl_seconds"`
}
