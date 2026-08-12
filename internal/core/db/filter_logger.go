package db

import (
	"context"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm/logger"
)

// LogType 日志类型枚举
type LogType int

const (
	// LogTypeSQL SQL 查询日志
	LogTypeSQL LogType = iota
	// LogTypeError 错误日志
	LogTypeError
)

// LogFilterConfig 日志过滤配置
type LogFilterConfig struct {
	MinLevel      logger.LogLevel  // 最小日志级别（低于此级别的不输出）
	FilterTypes   map[LogType]bool // 要过滤的日志类型（true = 过滤掉，false = 保留）
	SlowThreshold int              // 慢查询阈值（毫秒），0 表示不启用
}

// DefaultLogFilterConfig 默认配置：完全静默 SQL 日志，保留数据库错误
var DefaultLogFilterConfig = LogFilterConfig{
	MinLevel: logger.Silent,
	FilterTypes: map[LogType]bool{
		LogTypeSQL:   true,  // 过滤普通 SQL
		LogTypeError: false, // 保留错误
	},
	SlowThreshold: 1000, // 慢查询阈值 1 秒
}

// FilterLogger 通用可配置日志过滤器
type FilterLogger struct {
	config LogFilterConfig
}

// NewFilterLogger 创建新的日志过滤器
func NewFilterLogger(config LogFilterConfig) *FilterLogger {
	return &FilterLogger{config: config}
}

// LogMode 实现 logger.Interface
func (l *FilterLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.config.MinLevel = level
	return &newLogger
}

// Info 实现 logger.Interface - 完全静默
func (l *FilterLogger) Info(_ context.Context, _ string, _ ...interface{}) {}

// Warn 实现 logger.Interface - 完全静默
func (l *FilterLogger) Warn(_ context.Context, _ string, _ ...interface{}) {}

// Error 实现 logger.Interface - 根据配置输出错误
func (l *FilterLogger) Error(_ context.Context, msg string, data ...interface{}) {
	if l.config.FilterTypes[LogTypeError] {
		return
	}

	if len(data) > 0 && data[0] != nil {
		if err, ok := data[0].(error); ok && err != logger.ErrRecordNotFound {
			applogger.Errorf("[GORM错误] %s: %v", msg, err)
		}
	}
}

// Trace 实现 logger.Interface，根据配置过滤 SQL 日志
func (l *FilterLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err == nil || err == logger.ErrRecordNotFound {
		return
	}

	if l.config.FilterTypes[LogTypeError] {
		return
	}

	sql, _ := fc()
	applogger.Errorf("[GORM错误] %s | 耗时: %v | 错误: %v", sql, time.Since(begin), err)
}
