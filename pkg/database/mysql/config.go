package mysql

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
	Timezone string `mapstructure:"timezone"`
	Pool     struct {
		Min       int `mapstructure:"min"`
		Max       int `mapstructure:"max"`
		AcquireMs int `mapstructure:"acquire_ms"`
		IdleMs    int `mapstructure:"idle_ms"`
	} `mapstructure:"pool"`
}
