package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error
}

// ZapLogger Zap 日志实现
type ZapLogger struct {
	logger *zap.SugaredLogger
	fields []Field
}

// NewLogger 创建日志器
func NewLogger(level, format, output string) (Logger, error) {
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
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	switch format {
	case "console":
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	switch output {
	case "stderr":
		writeSyncer = zapcore.AddSync(os.Stderr)
	default:
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, zapLevel)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &ZapLogger{
		logger: logger.Sugar(),
	}, nil
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.log(zapcore.DebugLevel, msg, fields)
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.log(zapcore.InfoLevel, msg, fields)
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.log(zapcore.WarnLevel, msg, fields)
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.log(zapcore.ErrorLevel, msg, fields)
}

func (l *ZapLogger) With(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &ZapLogger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *ZapLogger) log(level zapcore.Level, msg string, fields []Field) {
	allFields := make([]Field, len(l.fields)+len(fields))
	copy(allFields, l.fields)
	copy(allFields[len(l.fields):], fields)

	zapFields := make([]interface{}, len(allFields)*2)
	for i, f := range allFields {
		zapFields[i*2] = f.Key
		zapFields[i*2+1] = f.Value
	}

	switch level {
	case zapcore.DebugLevel:
		l.logger.Debugw(msg, zapFields...)
	case zapcore.InfoLevel:
		l.logger.Infow(msg, zapFields...)
	case zapcore.WarnLevel:
		l.logger.Warnw(msg, zapFields...)
	case zapcore.ErrorLevel:
		l.logger.Errorw(msg, zapFields...)
	}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
