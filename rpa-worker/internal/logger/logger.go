package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	With(fields ...Field) Logger
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建64位整数字段
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Duration 创建时间段字段
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Err 创建错误字段
func Err(error error) Field {
	return Field{Key: "error", Value: error}
}

// Any 创建任意类型字段
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ZapLogger zap 实现
type ZapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	fields []Field
	mu     sync.RWMutex
}

// NewZapLogger 创建 zap logger
func NewZapLogger(level, format, output string) (*ZapLogger, error) {
	// 上海时区（优先使用 IANA 时区，失败则使用固定偏移）
	shanghaiLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果容器中没有时区数据，使用固定偏移量（UTC+8）
		shanghaiLocation = time.FixedZone("CST", 8*60*60)
	}

	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 编码器配置
	var encoderConfig zapcore.EncoderConfig
	if format == "json" {
		// JSON 格式
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000-07:00"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	} else {
		// Console 格式（人类可读）
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	// 时区编码器包装
	encodeTime := encoderConfig.EncodeTime
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		t = t.In(shanghaiLocation)
		encodeTime(t, enc)
	}

	// 选择编码器
	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 核心
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)

	logger := zap.New(core, zap.AddCaller())

	return &ZapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		fields: make([]Field, 0),
	}, nil
}

// With 添加字段
func (l *ZapLogger) With(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &ZapLogger{
		logger: l.logger,
		sugar:  l.sugar,
		fields: newFields,
	}
}

// Debug 记录调试日志
func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.log(zap.DebugLevel, msg, fields)
}

// Info 记录信息日志
func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.log(zap.InfoLevel, msg, fields)
}

// Warn 记录警告日志
func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.log(zap.WarnLevel, msg, fields)
}

// Error 记录错误日志
func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.log(zap.ErrorLevel, msg, fields)
}

// Sync 同步日志
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// log 内部日志方法
func (l *ZapLogger) log(level zapcore.Level, msg string, fields []Field) {
	l.mu.RLock()
	allFields := append([]Field{}, l.fields...)
	allFields = append(allFields, fields...)
	l.mu.RUnlock()

	// 构建 zap 字段
	zapFields := make([]zap.Field, len(allFields))
	for i, f := range allFields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}

	if level == zap.DebugLevel {
		l.logger.Debug(msg, zapFields...)
	} else if level == zap.InfoLevel {
		l.logger.Info(msg, zapFields...)
	} else if level == zap.WarnLevel {
		l.logger.Warn(msg, zapFields...)
	} else if level == zap.ErrorLevel {
		l.logger.Error(msg, zapFields...)
	}
}

// fieldsToZap 转换字段为 zap.Field
func fieldsToZap(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}
