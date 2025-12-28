package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log            *logrus.Logger
	fileLogger     *logrus.Logger
	consoleLogger   *logrus.Logger
)

// Config 日志配置
type Config struct {
	Level         string  // 日志级别: debug, info, warn, error
	LogDir        string  // 日志目录
	MaxSize       int     // 单个日志文件最大大小(MB)
	MaxBackups    int     // 保留的旧日志文件最大数量
	MaxAge        int     // 保留旧日志文件的最大天数
	Compress      bool    // 是否压缩旧日志文件
	ConsoleOutput bool    // 是否同时输出到控制台
}

// DefaultConfig 默认日志配置
func DefaultConfig() *Config {
	return &Config{
		Level:         "info",
		LogDir:        "logs",
		MaxSize:       100,    // 100MB
		MaxBackups:    30,     // 保留30个备份
		MaxAge:        90,     // 保留90天
		Compress:      true,   // 压缩旧日志
		ConsoleOutput: true,   // 同时输出到控制台
	}
}

// Init 初始化日志系统
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 确保日志目录存在
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return err
	}

	// 初始化日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}

	// 创建文件日志写入器
	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, "app.log"),
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	// 创建文件日志记录器
	fileLogger = logrus.New()
	fileLogger.SetLevel(level)
	fileLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	fileLogger.SetOutput(fileWriter)

	// 如果需要同时输出到控制台
	if cfg.ConsoleOutput {
		// 创建多重写入器
		multiWriter := io.MultiWriter(os.Stdout, fileWriter)

		consoleLogger = logrus.New()
		consoleLogger.SetLevel(level)
		consoleLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     true,
		})
		consoleLogger.SetOutput(multiWriter)

		log = consoleLogger
	} else {
		log = fileLogger
	}

	return nil
}

// GetLogger 获取日志记录器
func GetLogger() *logrus.Logger {
	if log == nil {
		// 如果未初始化，使用默认配置初始化
		if err := Init(DefaultConfig()); err != nil {
			// 如果初始化失败，返回标准日志
			logrus.SetOutput(os.Stdout)
			logrus.SetLevel(logrus.InfoLevel)
			return logrus.StandardLogger()
		}
	}
	return log
}

// WithFields 创建带有字段的新日志记录器
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithField 创建带有单个字段的新日志记录器
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// Debug 记录调试日志
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf 记录格式化调试日志
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info 记录信息日志
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof 记录格式化信息日志
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn 记录警告日志
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf 记录格式化警告日志
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error 记录错误日志
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf 记录格式化错误日志
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal 记录致命错误日志并退出程序
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf 记录格式化致命错误日志并退出程序
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// Panic 记录恐慌日志并触发panic
func Panic(args ...interface{}) {
	GetLogger().Panic(args...)
}

// Panicf 记录格式化恐慌日志并触发panic
func Panicf(format string, args ...interface{}) {
	GetLogger().Panicf(format, args...)
}

// SetLevel 设置日志级别
func SetLevel(level string) error {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	GetLogger().SetLevel(l)
	return nil
}

// GetFileLogger 获取纯文件日志记录器（只写入文件）
func GetFileLogger() *logrus.Logger {
	return fileLogger
}

// Close 关闭日志系统
func Close() {
	if fileLogger != nil && fileLogger.Out != nil {
		if closer, ok := fileLogger.Out.(io.Closer); ok {
			closer.Close()
		}
	}
}
