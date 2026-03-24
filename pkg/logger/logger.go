package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
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

	requestContexts sync.Map
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
	return slog.NewJSONHandler(writer, &slog.HandlerOptions{
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
func (l *Logger) log(level slog.Level, msg string, attrs ...any) {
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

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	if attrs := currentRequestAttrs(); len(attrs) > 0 {
		record.Add(attrs...)
	}
	if len(attrs) > 0 {
		record.Add(attrs...)
	}
	_ = inner.Handler().Handle(ctx, record)
}

func (l *Logger) logf(level slog.Level, format string, args ...interface{}) {
	l.log(level, fmt.Sprintf(format, args...))
}

// BindCurrentRequest binds request-scoped log fields to the current goroutine.
func BindCurrentRequest(fields map[string]string) func() {
	gid := currentGoroutineID()
	requestContexts.Store(gid, cloneRequestFields(fields))
	return func() {
		requestContexts.Delete(gid)
	}
}

// SetCurrentRequestField updates one request-scoped log field for the current goroutine.
func SetCurrentRequestField(key, value string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	gid := currentGoroutineID()
	current, _ := requestContexts.Load(gid)
	fields, _ := current.(map[string]string)
	if fields == nil {
		fields = make(map[string]string, 1)
	} else {
		fields = cloneRequestFields(fields)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		delete(fields, key)
	} else {
		fields[key] = value
	}

	if len(fields) == 0 {
		requestContexts.Delete(gid)
		return
	}
	requestContexts.Store(gid, fields)
}

func currentRequestAttrs() []any {
	current, ok := requestContexts.Load(currentGoroutineID())
	if !ok {
		return nil
	}

	fields, ok := current.(map[string]string)
	if !ok || len(fields) == 0 {
		return nil
	}

	keys := make([]string, 0, len(fields))
	for key, value := range fields {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	attrs := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		attrs = append(attrs, key, fields[key])
	}
	return attrs
}

func cloneRequestFields(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(fields))
	for key, value := range fields {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		cloned[key] = value
	}
	return cloned
}

func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	line := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	idField := line[:strings.IndexByte(line, ' ')]
	id, err := strconv.ParseUint(idField, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// Debug 输出调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.logf(slog.LevelDebug, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.logf(slog.LevelInfo, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.logf(slog.LevelWarn, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.logf(slog.LevelError, format, args...)
}

// InfoAttrs outputs an info log with structured attributes.
func (l *Logger) InfoAttrs(msg string, attrs ...any) {
	l.log(slog.LevelInfo, msg, attrs...)
}
