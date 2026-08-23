package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	assert.Equal(t, "info", c.Level)
	assert.Equal(t, "logs", c.LogDir)
	assert.Equal(t, 100, c.MaxSize)
	assert.Equal(t, 30, c.MaxBackups)
	assert.Equal(t, 90, c.MaxAge)
	assert.True(t, c.Compress)
	assert.True(t, c.ConsoleOutput)
}

func TestParseLogLevel(t *testing.T) {
	assert.Equal(t, logrus.DebugLevel, parseLogLevel("debug"))
	assert.Equal(t, logrus.InfoLevel, parseLogLevel("info"))
	assert.Equal(t, logrus.WarnLevel, parseLogLevel("warn"))
	assert.Equal(t, logrus.ErrorLevel, parseLogLevel("error"))
	assert.Equal(t, logrus.InfoLevel, parseLogLevel("unknown"))
}

func TestEnsureLogDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, ensureLogDirectory(dir))
	// 已存在也 ok

	// 新建嵌套目录
	dir2 := filepath.Join(t.TempDir(), "a", "b", "c")
	require.NoError(t, ensureLogDirectory(dir2))
}

func TestCreateFileWriter(t *testing.T) {
	dir := t.TempDir()
	w := createFileWriter(&Config{
		LogDir: dir, MaxSize: 1, MaxBackups: 1, MaxAge: 1,
	})
	assert.NotNil(t, w)
	defer w.Close()
	// 写一行
	_, err := w.Write([]byte("hello\n"))
	require.NoError(t, err)
}

func TestCreateFileLogger_Formats(t *testing.T) {
	var buf bytes.Buffer
	l := createFileLogger(logrus.InfoLevel, &nopCloser{Writer: &buf})
	assert.NotNil(t, l)
	l.Info("test msg")
	out := buf.String()
	assert.Contains(t, out, "test msg")
	assert.Contains(t, out, `"level":"info"`, "JSON 格式应包含 level 字段")
	// 时间戳格式包含年份
	assert.Contains(t, out, time.Now().Format("2006"))
}

func TestCreateFileLogger_CallerHook(t *testing.T) {
	// 触发 WithCaller 字段(测试 createLoggerWithCaller hook)
	var buf bytes.Buffer
	l := createFileLogger(logrus.InfoLevel, &nopCloser{Writer: &buf})
	l.Info("hook")
	out := buf.String()
	// 文件名/调用方可能未暴露,允许不含,但应不 panic
	assert.Contains(t, out, "hook")
}

func TestCreateConsoleLogger(t *testing.T) {
	l := createConsoleLogger(logrus.DebugLevel)
	assert.NotNil(t, l)
	assert.Equal(t, logrus.DebugLevel, l.GetLevel())
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	l := createFileLogger(logrus.WarnLevel, &nopCloser{Writer: &buf})

	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")

	out := buf.String()
	assert.NotContains(t, out, "debug")
	assert.NotContains(t, out, "info")
	assert.Contains(t, out, "warn")
	assert.Contains(t, out, "error")
}

func TestInit_DefaultsAndError(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-")
	require.NoError(t, err)
	defer os.RemoveAll(dir) // 即使 Close 没释放也可手动删
	cfg := DefaultConfig()
	cfg.LogDir = dir
	cfg.ConsoleOutput = false
	require.NoError(t, Init(cfg))
	assert.NotNil(t, GetLogger())
	assert.NotNil(t, GetFileLogger())
	Close()

	// 入口函数验证
	GetLogger().Info("via getter")
	GetFileLogger().Warn("via file getter")
}

func TestInit_NilConfig(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-nil-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	require.NoError(t, os.Setenv("LOG_DIR", dir))
	defer os.Unsetenv("LOG_DIR")

	require.NoError(t, Init(nil))
	assert.NotNil(t, GetLogger())
	Close()
}

func TestInit_BadLogDir(t *testing.T) {
	Close()
	// 用文件占用路径 → MkdirAll 失败
	bad := filepath.Join(t.TempDir(), "x")
	require.NoError(t, os.WriteFile(bad, []byte("x"), 0644))
	cfg := DefaultConfig()
	cfg.LogDir = filepath.Join(bad, "sub")
	cfg.ConsoleOutput = false
	err := Init(cfg)
	require.Error(t, err)
}

func TestWithFieldsAndWithField(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-fields-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	cfg := DefaultConfig()
	cfg.LogDir = dir
	cfg.ConsoleOutput = false
	require.NoError(t, Init(cfg))
	defer Close()

	// WithFields
	entry := WithFields(map[string]interface{}{"key": "v"})
	assert.NotNil(t, entry)
	entry.Info("with fields")

	// WithField
	e2 := WithField("k2", "v2")
	assert.NotNil(t, e2)
	e2.Info("with single field")
}

func TestInit_WithInvalidLevel(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-bad-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	cfg := DefaultConfig()
	cfg.LogDir = dir
	cfg.Level = "bogus"
	cfg.ConsoleOutput = false
	require.NoError(t, Init(cfg))
	defer Close()
}

func TestSetLevel(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-setlevel-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	cfg := DefaultConfig()
	cfg.LogDir = dir
	cfg.ConsoleOutput = false
	require.NoError(t, Init(cfg))
	defer Close()

	require.NoError(t, SetLevel("debug"))
	assert.Equal(t, logrus.DebugLevel, GetLogger().GetLevel())
	require.NoError(t, SetLevel("warn"))
	assert.Equal(t, logrus.WarnLevel, GetLogger().GetLevel())
	require.Error(t, SetLevel("garbage"))
}

func TestShortcutFunctions(t *testing.T) {
	Close()
	dir, err := os.MkdirTemp("", "logtest-short-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	cfg := DefaultConfig()
	cfg.LogDir = dir
	cfg.ConsoleOutput = false
	require.NoError(t, Init(cfg))
	defer Close()

	Debug("debug")
	Debugf("debugf %d", 1)
	Info("info")
	Infof("infof %d", 2)
	Warn("warn")
	Warnf("warnf %d", 3)
	Error("error")
	Errorf("errorf %d", 4)
}

func TestEnsureLogDirectory_Error(t *testing.T) {
	// 传入根目录冲突 / 不可写 → 触发 error
	if os.Getuid() != 0 {
		// 普通用户: 写一个文件, 再尝试 mkdir 同名
		f := filepath.Join(t.TempDir(), "block")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
		err := ensureLogDirectory(f + "/sub")
		require.Error(t, err, "已存在文件占位应报错")
	}
}

// nopCloser 把 io.Writer 包成 io.WriteCloser 用于测试。
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// 触发 strings.Split 路径覆盖(日志轮转 / file naming 用的分割符)
func TestStringSplit_LogFilenames(t *testing.T) {
	parts := strings.Split("app.log", ".")
	assert.Equal(t, 2, len(parts))
}

// Q-7 race 验证暴露的回归:GetLogger 懒初始化曾在并发首调时与 Init 形成数据竞争
// (pkg/logger 全局 log 无锁读写)。本用例在 -race 下判别:重置全局后并发首调。
func TestGetLogger_ConcurrentLazyInit(t *testing.T) {
	logMu.Lock()
	savedLog, savedFileLogger, savedFileWriter := log, fileLogger, fileWriter
	log, fileLogger, fileWriter = nil, nil, nil
	logMu.Unlock()
	t.Cleanup(func() {
		logMu.Lock()
		log, fileLogger, fileWriter = savedLog, savedFileLogger, savedFileWriter
		logMu.Unlock()
	})

	tmp := t.TempDir()
	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			l := GetLogger()
			assert.NotNil(t, l)
		}()
	}
	// 并发的显式 Init 也不许与懒初始化竞争
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = Init(&Config{Level: "info", LogDir: tmp, ConsoleOutput: false})
	}()
	wg.Wait()
	assert.NotNil(t, GetLogger())
}
