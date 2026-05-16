package storage

import "sync"

var (
	globalStorage     Storage
	globalMaxFileSize int
	globalExtensions  []string
	once              sync.Once
)

type Config struct {
	Dir               string
	BaseURL           string
	MaxFileSizeMB     int
	AllowedExtensions []string
}

func Init(cfg Config) {
	once.Do(func() {
		globalStorage = NewLocalStorage(cfg.Dir, cfg.BaseURL)
		globalMaxFileSize = cfg.MaxFileSizeMB
		globalExtensions = cfg.AllowedExtensions
	})
}

func GetStorage() Storage {
	return globalStorage
}

func GetMaxFileSizeMB() int {
	return globalMaxFileSize
}

func GetAllowedExtensions() []string {
	return globalExtensions
}
