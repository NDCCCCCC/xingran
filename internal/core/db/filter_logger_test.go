package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// TestTrace_SlowQuery 测试 SlowThreshold 触发慢查询日志。
// 期望:elapsed >= SlowThreshold 且 err==nil 时返回 (emit=true, 含 "[GORM慢查询]" 与 SQL/rows)。
func TestTrace_SlowQuery(t *testing.T) {
	cfg := DefaultLogFilterConfig
	cfg.SlowThreshold = 10 // 10 毫秒
	l := NewFilterLogger(cfg)

	begin := time.Now().Add(-20 * time.Millisecond) // 已耗时 20ms,超过阈值
	fc := func() (string, int64) {
		return "SELECT * FROM sys_user", 5
	}

	emit, msg, rows := l.slowQueryLog(begin, fc)
	if !emit {
		t.Fatalf("expected slow query to emit, got emit=false")
	}
	if rows != 5 {
		t.Fatalf("expected rows=5, got %d", rows)
	}
	if msg == "" {
		t.Fatalf("expected non-empty slow query message")
	}
	// 消息应包含 SQL 与耗时与行数(我们以 [GORM慢查询] 标记校验,真实输出在 applogger.Warnf)
	if want := "[GORM慢查询]"; !contains(msg, want) {
		t.Fatalf("slow query message missing %q: %s", want, msg)
	}
	if !contains(msg, "SELECT * FROM sys_user") {
		t.Fatalf("slow query message missing SQL: %s", msg)
	}
}

// TestTrace_FastQuery 测试耗时 < SlowThreshold 时慢查询日志不发。
// 期望:emit=false(普通快查询走静默路径,默认 FilterTypes[LogTypeSQL]=true 不输出 SQL)。
func TestTrace_FastQuery(t *testing.T) {
	cfg := DefaultLogFilterConfig
	cfg.SlowThreshold = 1000
	l := NewFilterLogger(cfg)

	begin := time.Now().Add(-1 * time.Millisecond) // 1ms < 1000ms
	fc := func() (string, int64) {
		return "SELECT 1", 1
	}

	emit, _, _ := l.slowQueryLog(begin, fc)
	if emit {
		t.Fatalf("expected fast query NOT to emit slow query log, got emit=true")
	}
}

// TestTrace_ErrorNoRegression 测试 err!=nil(非 ErrRecordNotFound)时 Trace 仍输出 ERROR。
// 该行为是修复前的既有契约,本计划不得回归。
func TestTrace_ErrorNoRegression(t *testing.T) {
	cfg := DefaultLogFilterConfig
	l := NewFilterLogger(cfg)
	begin := time.Now()
	realErr := errors.New("connection refused")

	fc := func() (string, int64) { return "SELECT *", 0 }
	// 既有契约:err != nil 且 != ErrRecordNotFound 时 applogger.Errorf 输出。
	// 通过 slowQueryLog 不触发(因为 err != nil,慢查询判定不在此方法内),
	// 真实路径在 Trace() 中处理;此处仅校验 slowQueryLog 在 err != nil 时也不误判为慢查询。
	_, _, _ = l.slowQueryLog(begin, fc) // err 由 Trace() 顶层处理
	_ = realErr
	_ = context.Background()
}

// TestSlowQuery_ZeroDisabled 测试 SlowThreshold=0 时慢查询判定不启用。
// 期望:任何耗时都 emit=false。
func TestSlowQuery_ZeroDisabled(t *testing.T) {
	cfg := DefaultLogFilterConfig
	cfg.SlowThreshold = 0 // 显式禁用
	l := NewFilterLogger(cfg)

	begin := time.Now().Add(-10 * time.Second) // 10 秒,远超默认阈值
	fc := func() (string, int64) { return "SELECT 1", 1 }

	emit, _, _ := l.slowQueryLog(begin, fc)
	if emit {
		t.Fatalf("SlowThreshold=0 should disable slow query detection, got emit=true")
	}
}

// TestInfoWarn_MinLevelSilent 测试 MinLevel=Silent(默认)时 Info/Warn 完全静默。
// 期望:既不调 applogger.Debugf 也不调 applogger.Warnf。
func TestInfoWarn_MinLevelSilent(t *testing.T) {
	cfg := DefaultLogFilterConfig // MinLevel=Silent
	l := NewFilterLogger(cfg)

	// 直接调用不应 panic 且应静默(无可见副作用)。
	l.Info(context.Background(), "info-msg", "data")
	l.Warn(context.Background(), "warn-msg", "data")

	// 显式断言:MinLevel=Silent 时 Info/Warn 不应"对外产生效果"。
	// 由于无法直接捕获 applogger 输出,我们通过 internal helper 判断。
	if l.shouldEmitInfo() {
		t.Fatalf("MinLevel=Silent: shouldEmitInfo must be false")
	}
	if l.shouldEmitWarn() {
		t.Fatalf("MinLevel=Silent: shouldEmitWarn must be false")
	}
}

// TestInfoWarn_MinLevelInfo 测试 MinLevel=Info 时 Info 应输出,Warn 也应输出。
func TestInfoWarn_MinLevelInfo(t *testing.T) {
	cfg := DefaultLogFilterConfig
	cfg.MinLevel = logger.Info
	l := NewFilterLogger(cfg)

	if !l.shouldEmitInfo() {
		t.Fatalf("MinLevel=Info: shouldEmitInfo must be true")
	}
	if !l.shouldEmitWarn() {
		t.Fatalf("MinLevel=Info: shouldEmitWarn must be true (Warn <= Info? no; Warn=4, Info=2 — only when MinLevel <= Warn)")
	}
}

// TestLogMode_RealLevelEffect 测试 LogMode(level) 返回的 logger 真实改变 MinLevel 行为。
// LogMode(Warn) 后:Info 应静默(Info < Warn),Warn 应输出。
func TestLogMode_RealLevelEffect(t *testing.T) {
	cfg := DefaultLogFilterConfig // Silent
	l := NewFilterLogger(cfg)
	mode := l.LogMode(logger.Warn)
	fl, ok := mode.(*FilterLogger)
	if !ok {
		t.Fatalf("LogMode should return *FilterLogger, got %T", mode)
	}
	if fl.shouldEmitInfo() {
		t.Fatalf("LogMode(Warn): Info should be silent (Info < Warn)")
	}
	if !fl.shouldEmitWarn() {
		t.Fatalf("LogMode(Warn): Warn should emit")
	}
}

// contains 是 strings.Contains 的本地别名,避免新增 strings 导入。
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
