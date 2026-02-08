package logger

import (
	"context"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// GormLogger 自定义GORM日志记录器
type GormLogger struct {
	logger *Logger
}

// NewGormLogger 创建新的GORM日志记录器
func NewGormLogger() *GormLogger {
	return &GormLogger{
		logger: GetLogger(),
	}
}

// LogMode 设置日志级别
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

// Info 记录信息日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logger != nil {
		l.logger.Info(msg, data...)
	}
}

// Warn 记录警告日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logger != nil {
		l.logger.Warn(msg, data...)
	}
}

// Error 记录错误日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logger != nil {
		l.logger.Error(msg, data...)
	}
}

// Trace 记录SQL执行日志
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if l.logger != nil {
		if err != nil {
			l.logger.Error("[%.3fms] [rows:%d] %s | error: %v", float64(elapsed.Nanoseconds())/1e6, rows, sql, err)
		} else {
			l.logger.Info("[%.3fms] [rows:%d] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
