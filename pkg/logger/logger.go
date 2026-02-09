package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
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
	mu    sync.RWMutex
	inner *slog.Logger
	level *slog.LevelVar
}

type rotationConfig struct {
	enabled   bool
	maxSizeMB int
	maxFiles  int
	maxAgeDay int
	compress  bool
}

var (
	instance = newDefaultLogger()
	once     sync.Once
	initErr  error

	rotationMu  sync.RWMutex
	rotationCfg = rotationConfig{
		enabled:   false,
		maxSizeMB: 100,
		maxFiles:  3,
		maxAgeDay: 28,
		compress:  true,
	}
)

// Init 初始化日志器
func Init(levelStr string, console bool, filePath string) error {
	once.Do(func() {
		initErr = instance.reconfigure(levelStr, console, filePath)
	})
	return initErr
}

// SetRotationConfig 设置日志切割配置（在 Init 前调用）
func SetRotationConfig(enabled bool, maxSizeMB, maxFiles int) {
	rotationMu.Lock()
	defer rotationMu.Unlock()

	rotationCfg.enabled = enabled
	if maxSizeMB > 0 {
		rotationCfg.maxSizeMB = maxSizeMB
	}
	if maxFiles > 0 {
		rotationCfg.maxFiles = maxFiles
	}
}

// GetLogger 获取日志器实例
func GetLogger() *Logger {
	return instance
}

func newDefaultLogger() *Logger {
	lv := &slog.LevelVar{}
	lv.Set(slog.LevelInfo)

	return &Logger{
		inner: slog.New(newHandler(os.Stdout, lv)),
		level: lv,
	}
}

func (l *Logger) reconfigure(levelStr string, console bool, filePath string) error {
	lv := &slog.LevelVar{}
	lv.Set(parseSlogLevel(levelStr))

	writer, err := buildWriter(console, filePath)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.inner = slog.New(newHandler(writer, lv))
	l.level = lv
	return nil
}

func buildWriter(console bool, filePath string) (io.Writer, error) {
	writers := make([]io.Writer, 0, 2)

	if strings.TrimSpace(filePath) != "" {
		logDir := filepath.Dir(filePath)
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		fileWriter, err := newFileWriter(filePath)
		if err != nil {
			return nil, err
		}
		writers = append(writers, fileWriter)
	}

	if console || len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	return io.MultiWriter(writers...), nil
}

func newFileWriter(filePath string) (io.Writer, error) {
	cfg := getRotationConfig()

	if cfg.enabled {
		return &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    cfg.maxSizeMB,
			MaxBackups: cfg.maxFiles,
			MaxAge:     cfg.maxAgeDay,
			Compress:   cfg.compress,
		}, nil
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	return file, nil
}

func getRotationConfig() rotationConfig {
	rotationMu.RLock()
	defer rotationMu.RUnlock()
	return rotationCfg
}

func newHandler(writer io.Writer, leveler slog.Leveler) slog.Handler {
	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource:   true,
		Level:       leveler,
		ReplaceAttr: replaceAttrs,
	})
}

func replaceAttrs(_ []string, attr slog.Attr) slog.Attr {
	switch attr.Key {
	case slog.TimeKey:
		t := attr.Value.Time()
		if !t.IsZero() {
			return slog.String(slog.TimeKey, t.Format("2006-01-02 15:04:05"))
		}
	case slog.LevelKey:
		if lv, ok := attr.Value.Any().(slog.Level); ok {
			return slog.String(slog.LevelKey, formatSlogLevel(lv))
		}
	case slog.SourceKey:
		if src, ok := attr.Value.Any().(*slog.Source); ok && src != nil {
			if idx := strings.Index(src.File, "volunteer-system"); idx >= 0 {
				src.File = src.File[idx:]
			}
			return slog.Any(slog.SourceKey, src)
		}
	}
	return attr
}

func formatSlogLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "[DEBUG]"
	case level < slog.LevelWarn:
		return "[INFO]"
	case level < slog.LevelError:
		return "[WARN]"
	default:
		return "[ERROR]"
	}
}

func parseSlogLevel(levelStr string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// log 内部日志方法
func (l *Logger) log(level slog.Level, format string, args ...interface{}) {
	l.mu.RLock()
	inner := l.inner
	l.mu.RUnlock()

	if inner == nil {
		return
	}

	ctx := context.Background()
	if !inner.Enabled(ctx, level) {
		return
	}

	msg := fmt.Sprintf(format, args...)

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	_ = inner.Handler().Handle(ctx, record)
}

// Debug 输出调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(slog.LevelDebug, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(slog.LevelInfo, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(slog.LevelWarn, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(slog.LevelError, format, args...)
}
