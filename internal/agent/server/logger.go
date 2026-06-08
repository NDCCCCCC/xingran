package server

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// InitLogger 初始化日志系统
func InitLogger(logLevel string, logPath string) error {
	logger = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 设置 JSON 格式（结构化日志）
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
			logrus.FieldKeyFile:  "file",
		},
	})

	// 设置日志输出
	if logPath != "" {
		logFile, err := os.OpenFile(logPath+"/agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logger.SetOutput(logFile)
		}
	}

	return nil
}

// WithContext 创建带上下文的日志条目
func WithContext(ctx context.Context) *logrus.Entry {
	entry := logger.WithFields(logrus.Fields{})

	// 提取请求 ID（如果有）
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	// 提取用户 ID（如果有）
	if userID := ctx.Value("user_id"); userID != nil {
		entry = entry.WithField("user_id", userID)
	}

	// 提取 Agent ID（如果有）
	if agentID := ctx.Value("agent_id"); agentID != nil {
		entry = entry.WithField("agent_id", agentID)
	}

	return entry
}

// WithRequestID 创建带请求 ID 的日志条目
func WithRequestID(requestID string) *logrus.Entry {
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return logger.WithField("request_id", requestID)
}

// Debug 调试日志
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info 信息日志
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn 警告日志
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error 错误日志
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Fatal 致命错误日志（程序退出）
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// WithFields 创建带字段的日志条目
func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.WithFields(fields)
}
