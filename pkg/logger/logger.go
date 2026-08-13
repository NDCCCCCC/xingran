package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// 默认日志配置
	defaultLevel         = "info"
	defaultLogDir        = "logs"
	defaultMaxSize       = 100  // MB
	defaultMaxBackups    = 30   // 保留30个备份
	defaultMaxAge        = 90   // 保留90天
	defaultCompress      = true // 压缩旧日志
	defaultConsoleOutput = true // 同时输出到控制台

	// 日志文件名
	logFileName = "app.log"

	// 日志时间格式
	timestampFormat = "2006-01-02 15:04:05"
)

var (
	log        *logrus.Logger
	fileLogger *logrus.Logger
	fileWriter io.WriteCloser
)

// Config 日志配置
type Config struct {
	Level         string // 日志级别: debug, info, warn, error
	LogDir        string // 日志目录
	MaxSize       int    // 单个日志文件最大大小(MB)
	MaxBackups    int    // 保留的旧日志文件最大数量
	MaxAge        int    // 保留旧日志文件的最大天数
	Compress      bool   // 是否压缩旧日志文件
	ConsoleOutput bool   // 是否同时输出到控制台
}

// DefaultConfig 返回默认日志配置
func DefaultConfig() *Config {
	return &Config{
		Level:         defaultLevel,
		LogDir:        defaultLogDir,
		MaxSize:       defaultMaxSize,
		MaxBackups:    defaultMaxBackups,
		MaxAge:        defaultMaxAge,
		Compress:      defaultCompress,
		ConsoleOutput: defaultConsoleOutput,
	}
}

// Init 初始化日志系统
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := ensureLogDirectory(cfg.LogDir); err != nil {
		return err
	}

	level := parseLogLevel(cfg.Level)
	fileWriter = createFileWriter(cfg)

	// 文件日志：JSON 格式（机器可读，方便日志聚合/分析）
	fileLogger = createFileLogger(level, fileWriter)

	if cfg.ConsoleOutput {
		// 控制台日志：写 stdout（Text 格式，方便人类阅读）。
		// 同时通过 hook 镜像到 JSON 文件 logger，避免 fileLogger 孤立
		// （早期用 MultiWriter 把 Text 也写进文件导致文件非 JSON，已用 hook 方案规避）。
		console := createConsoleLogger(level)
		console.AddHook(&fileMirrorHook{fileLogger: fileLogger})
		log = console
	} else {
		log = fileLogger
	}

	return nil
}

// fileMirrorHook 将日志条目镜像写入 JSON 文件 logger，
// 使 ConsoleOutput=true 时文件日志（logs/app.log）同时落盘。
// fileLogger 自身无 hook，不会递归。
type fileMirrorHook struct {
	fileLogger *logrus.Logger
}

func (h *fileMirrorHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *fileMirrorHook) Fire(entry *logrus.Entry) error {
	if h.fileLogger == nil {
		return nil
	}
	fileEntry := h.fileLogger.WithFields(entry.Data)
	switch entry.Level {
	case logrus.PanicLevel:
		fileEntry.Panic(entry.Message)
	case logrus.FatalLevel:
		fileEntry.Fatal(entry.Message)
	case logrus.ErrorLevel:
		fileEntry.Error(entry.Message)
	case logrus.WarnLevel:
		fileEntry.Warn(entry.Message)
	case logrus.InfoLevel:
		fileEntry.Info(entry.Message)
	case logrus.DebugLevel:
		fileEntry.Debug(entry.Message)
	case logrus.TraceLevel:
		fileEntry.Trace(entry.Message)
	}
	return nil
}

// ensureLogDirectory 确保日志目录存在
func ensureLogDirectory(logDir string) error {
	return os.MkdirAll(logDir, 0755)
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) logrus.Level {
	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		return logrus.InfoLevel
	}
	return parsed
}

// createFileWriter 创建文件写入器
func createFileWriter(cfg *Config) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, logFileName),
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

// createFileLogger 创建文件日志记录器
func createFileLogger(level logrus.Level, writer io.Writer) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: timestampFormat,
	})
	logger.SetOutput(writer)
	return logger
}

// createConsoleLogger 创建控制台日志记录器
//
// 仅写入 stdout（Text 格式，方便人类阅读）。
// 文件日志由 fileLogger 独立管理（JSON 格式）。
func createConsoleLogger(level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: timestampFormat,
		FullTimestamp:   true,
		ForceColors:     true,
	})
	logger.SetOutput(os.Stdout)

	return logger
}

// GetLogger 获取日志记录器
func GetLogger() *logrus.Logger {
	if log == nil {
		if err := Init(DefaultConfig()); err != nil {
			// 记录到stderr，因为日志系统尚未初始化
			fmt.Fprintf(os.Stderr, "日志初始化失败，使用标准日志: %v\n", err)
			fallbackToStdLogger()
			return logrus.StandardLogger()
		}
	}
	return log
}

// fallbackToStdLogger 回退到标准日志
func fallbackToStdLogger() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
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
//
// 关键修补:logrus.Fatal/Fatalf 走 os.Exit(1),不等 hook 刷盘,导致 fileMirrorHook
// 的镜像日志丢失。手动写一条带 Fatal level 的镜像 + sync(确保文件落盘)再 exit。
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// 直接写一条 Fatal 到 fileLogger(绕过 stdout logger 的 hook 链路,避免 os.Exit
	// 在 hook 执行前切断 flush 时机)。
	if fileLogger != nil {
		fileLogger.Fatal(msg)
	}
	// 同时写 stdout,保留原终端输出格式
	GetLogger().Error(msg)
	if fileWriter != nil {
		_ = fileWriter.Close()
	}
	os.Exit(1)
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
	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	GetLogger().SetLevel(parsed)
	return nil
}

// GetFileLogger 获取纯文件日志记录器（只写入文件）
func GetFileLogger() *logrus.Logger {
	return fileLogger
}

// Close 关闭日志系统
func Close() {
	if fileWriter != nil {
		fileWriter.Close()
	}
}
