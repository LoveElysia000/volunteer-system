package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别类型
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger 日志结构体
type Logger struct {
	mu      sync.Mutex
	logger  *log.Logger
	level   LogLevel
	console bool
}

var (
	instance *Logger
	once     sync.Once
)

// Init 初始化日志器
func Init(levelStr string, console bool, filePath string) error {
	var err error
	once.Do(func() {
		instance, err = newLogger(levelStr, console, filePath)
	})
	return err
}

// GetLogger 获取日志器实例，如果未初始化则返回默认logger
func GetLogger() *Logger {
	if instance == nil {
		// 返回一个默认的logger，输出到控制台
		return &Logger{
			logger:  log.New(os.Stdout, "", 0),
			level:   INFO,
			console: true,
		}
	}
	return instance
}

// newLogger 创建新的日志器
func newLogger(levelStr string, console bool, filePath string) (*Logger, error) {
	// 解析日志级别
	level := parseLevel(levelStr)

	// 确保日志目录存在
	logDir := filepath.Dir(filePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 创建多写入器
	var writers []io.Writer
	writers = append(writers, file)
	if console {
		writers = append(writers, os.Stdout)
	}

	return &Logger{
		logger:  log.New(io.MultiWriter(writers...), "", 0),
		level:   level,
		console: console,
	}, nil
}

// parseLevel 解析日志级别字符串
func parseLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// formatLevel 格式化日志级别
func (l *Logger) formatLevel(level LogLevel) string {
	switch level {
	case DEBUG:
		return "[DEBUG]"
	case INFO:
		return "[INFO]"
	case WARN:
		return "[WARN]"
	case ERROR:
		return "[ERROR]"
	default:
		return "[INFO]"
	}
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2) // skip 2 frames: log() -> Info/Warn/Error...
	var caller string
	if ok {
		// 简化文件路径，只保留相对于项目根目录的路径
		if idx := strings.Index(file, "volunteer-system"); idx >= 0 {
			file = file[idx:]
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	} else {
		caller = "unknown:0"
	}

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s %s %s %s\n", timestamp, l.formatLevel(level), caller, message)

	// 写入日志
	l.logger.Print(logLine)
}

// Debug 输出调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}
