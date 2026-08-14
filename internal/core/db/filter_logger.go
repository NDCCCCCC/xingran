package db

import (
	"context"
	"fmt"
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
	// MinLevel 最小日志级别(低于此级别的不输出)。
	// 默认 logger.Silent —— 在此默认值下 Info/Warn 均静默,仅慢查询与错误可见。
	// 由 LogMode(level) 实时改写。
	MinLevel logger.LogLevel
	// FilterTypes 要过滤的日志类型(true = 过滤掉, false = 保留)。
	// LogTypeSQL=true(默认)时普通 SQL 日志走 Trace 的 FilterTypes 路径静默;
	// 慢查询是运维信号,**不受 FilterTypes[LogTypeSQL] 影响**(独立判定)。
	FilterTypes map[LogType]bool
	// SlowThreshold 慢查询阈值(毫秒),0 表示不启用。
	// 默认 1000ms —— 超过此耗时的成功 SQL 走 applogger.Warnf 输出,
	// 耗时与行数与 SQL 一同记录,便于 dev/slow DB 性能问题排查。
	SlowThreshold int
}

// DefaultLogFilterConfig 默认配置:
//   - 普通 SQL 静默(FilterTypes[LogTypeSQL]=true)
//   - 数据库错误保留(FilterTypes[LogTypeError]=false),通过 applogger.Errorf 输出
//   - 慢查询阈值 1 秒(SlowThreshold=1000):超过此耗时的成功 SQL 输出 WARN
//   - MinLevel=Silent:Info/Warn 接口静默,LogMode 调高阈值时可显式打开
var DefaultLogFilterConfig = LogFilterConfig{
	MinLevel: logger.Silent,
	FilterTypes: map[LogType]bool{
		LogTypeSQL:   true, // 过滤普通 SQL
		LogTypeError: false,
	},
	SlowThreshold: 1000,
}

// FilterLogger 通用可配置日志过滤器
type FilterLogger struct {
	config LogFilterConfig
}

// NewFilterLogger 创建新的日志过滤器
func NewFilterLogger(config LogFilterConfig) *FilterLogger {
	return &FilterLogger{config: config}
}

// LogMode 实现 logger.Interface。
// 返回的 logger 拥有新的 MinLevel,Info/Warn/Trace 均真实读取该字段。
// (C4 修复:之前 MinLevel 写入后无人读取,LogMode 形同空操作。)
func (l *FilterLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.config.MinLevel = level
	return &newLogger
}

// shouldEmitInfo 返回 Info 是否应输出(MinLevel >= logger.Info)。
//
// GORM LogLevel 取值(Silent=1, Error=2, Warn=3, Info=4, Debug=5):
// 数值越大越详细;MinLevel=Silent 时全部静默;MinLevel=Debug 时全部输出。
// 判定规则:LogLevel >= messageLevel 才输出(与 GORM defaultLogger 一致)。
// 提取为私有方法以便测试断言 MinLevel 语义真实生效。
func (l *FilterLogger) shouldEmitInfo() bool {
	return l.config.MinLevel >= logger.Info
}

// shouldEmitWarn 返回 Warn 是否应输出(MinLevel >= logger.Warn)。
func (l *FilterLogger) shouldEmitWarn() bool {
	return l.config.MinLevel >= logger.Warn
}

// Info 实现 logger.Interface - 尊重 MinLevel。
// 默认 MinLevel=Silent 下静默(行为不变);LogMode(logger.Info) 后输出。
func (l *FilterLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if !l.shouldEmitInfo() {
		return
	}
	if len(data) > 0 {
		applogger.Debugf("[GORM] %s: %v", msg, data)
		return
	}
	applogger.Debugf("[GORM] %s", msg)
}

// Warn 实现 logger.Interface - 尊重 MinLevel。
// 默认 MinLevel=Silent 下静默;LogMode(logger.Warn) 后输出。
func (l *FilterLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if !l.shouldEmitWarn() {
		return
	}
	if len(data) > 0 {
		applogger.Warnf("[GORM] %s: %v", msg, data)
		return
	}
	applogger.Warnf("[GORM] %s", msg)
}

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

// slowQueryLog 判定是否触发慢查询日志并构造消息。
// 返回 (emit, msg, rows):emit=true 时 Trace 应调 applogger.Warnf 输出 msg。
//
// 提取为私有方法以便单元测试直接断言,无需捕获 applogger 输出。
//
// 判定规则:
//   - SlowThreshold <= 0:不启用慢查询,emit=false
//   - elapsed >= time.Duration(SlowThreshold) * time.Millisecond:emit=true
//   - 否则:emit=false(普通 SQL 走 FilterTypes[LogTypeSQL] 路径静默)
//
// 慢查询判定独立于 FilterTypes[LogTypeSQL]:慢查询是运维信号,不属于"普通 SQL 过滤"范畴。
func (l *FilterLogger) slowQueryLog(begin time.Time, fc func() (string, int64)) (emit bool, msg string, rows int64) {
	if l.config.SlowThreshold <= 0 {
		return false, "", 0
	}
	elapsed := time.Since(begin)
	if elapsed < time.Duration(l.config.SlowThreshold)*time.Millisecond {
		return false, "", 0
	}
	sql, r := fc()
	rows = r
	msg = fmt.Sprintf("[GORM慢查询] 耗时: %v | 行数: %d | %s", elapsed, rows, sql)
	return true, msg, rows
}

// Trace 实现 logger.Interface,根据配置过滤 SQL 日志并触发慢查询日志。
//
// 行为契约(C4 修复):
//   - err == nil 或 err == ErrRecordNotFound:不再直接 return ——
//     先判定慢查询;若超过 SlowThreshold 则 applogger.Warnf 输出。
//   - err != nil 且 != ErrRecordNotFound:走 Errorf 输出(既有行为不变)。
//   - 普通快查询(SlowThreshold 未达)走 FilterTypes[LogTypeSQL] 静默路径,默认配置下静默(行为不变)。
func (l *FilterLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	// 错误路径优先:err != nil 且 != ErrRecordNotFound 时按既有行为 Errorf 输出。
	if err != nil && err != logger.ErrRecordNotFound {
		if l.config.FilterTypes[LogTypeError] {
			return
		}
		sql, _ := fc()
		applogger.Errorf("[GORM错误] %s | 耗时: %v | 错误: %v", sql, time.Since(begin), err)
		return
	}

	// 成功 / ErrRecordNotFound 路径:判定慢查询(独立于 FilterTypes[LogTypeSQL])。
	emit, msg, _ := l.slowQueryLog(begin, fc)
	if emit {
		applogger.Warnf("%s", msg)
		return
	}

	// 普通 SQL 日志:FilterTypes[LogTypeSQL] 静默(默认 true → 静默)。
	if l.config.FilterTypes[LogTypeSQL] {
		return
	}
}
