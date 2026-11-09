package browser

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
)

// Phase 137 / FIX-08：WaitForURL / WaitForText 真实轮询校验的回归测试。
// 覆盖成功（立即命中 / 轮询后命中）、超时（恒不命中）、失败（probe 报错）三类路径，
// 锁死「首次 tick 即 return nil」不得回归。

// nopLogger 实现 logger.Logger，仅计数，不做真实 IO。
type nopLogger struct{}

func (nopLogger) Debug(string, ...logger.Field) {}
func (nopLogger) Info(string, ...logger.Field)  {}
func (nopLogger) Warn(string, ...logger.Field)  {}
func (nopLogger) Error(string, ...logger.Field) {}
func (n nopLogger) With(...logger.Field) logger.Logger { return n }
func (nopLogger) Sync() error                          { return nil }

// stubPage 是根副本 Page 接口的可编程假实现；只让 Evaluate 有行为。
type stubPage struct {
	evalFn func(script string) (interface{}, error)
}

func (s *stubPage) Goto(string) error                     { return nil }
func (s *stubPage) Close() error                          { return nil }
func (s *stubPage) Screenshot() ([]byte, error)           { return nil, nil }
func (s *stubPage) Click(string) error                    { return nil }
func (s *stubPage) Fill(string, string) error             { return nil }
func (s *stubPage) Select(string, string) error           { return nil }
func (s *stubPage) WaitFor(string, time.Duration) error   { return nil }
func (s *stubPage) Evaluate(script string) (interface{}, error) {
	if s.evalFn != nil {
		return s.evalFn(script)
	}
	return "", nil
}
func (s *stubPage) GetText(string) (string, error)         { return "", nil }
func (s *stubPage) GetValue(string) (string, error)        { return "", nil }
func (s *stubPage) GetHTML(string) (string, error)         { return "", nil }
func (s *stubPage) GetAttribute(string, string) (string, error) { return "", nil }
func (s *stubPage) ScrollDown(int) error                   { return nil }
func (s *stubPage) ScrollUp(int) error                     { return nil }
func (s *stubPage) ScrollToElement(string) error           { return nil }

// withFastPoll 把轮询间隔临时缩短，避免超时用例真的等 100ms。
func withFastPoll(t *testing.T, d time.Duration) {
	t.Helper()
	old := waitPollInterval
	waitPollInterval = d
	t.Cleanup(func() { waitPollInterval = old })
}

func TestWaitForURL_MatchesImmediately(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		return "https://example.com/dashboard", nil
	}}, nopLogger{})

	if err := m.WaitForURL("https://example.com/dashboard", 200*time.Millisecond); err != nil {
		t.Fatalf("URL 已匹配应立即成功: %v", err)
	}
}

func TestWaitForURL_WildcardAndSubstring(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		return "https://example.com/order/123/done", nil
	}}, nopLogger{})

	if err := m.WaitForURL("https://example.com/order/*/done", 200*time.Millisecond); err != nil {
		t.Fatalf("通配符应匹配: %v", err)
	}
	if err := m.WaitForURL("/done", 200*time.Millisecond); err != nil {
		t.Fatalf("子串应匹配: %v", err)
	}
}

func TestWaitForURL_TimesOutWhenNeverMatches(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	var calls int32
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return "https://example.com/login", nil
	}}, nopLogger{})

	err := m.WaitForURL("https://example.com/dashboard", 30*time.Millisecond)
	if err == nil {
		t.Fatal("URL 恒不匹配时必须返回超时错误，不能首次 tick 即 return nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("错误信息应含 timeout: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("必须真实轮询多次(>=2)，实际 probe 次数=%d", calls)
	}
}

func TestWaitForURL_ProbeErrorFailsFast(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	want := errors.New("page crashed")
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		return nil, want
	}}, nopLogger{})

	err := m.WaitForURL("https://example.com", time.Second)
	if err == nil {
		t.Fatal("probe 报错时必须返回 error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("应包装原始错误: %v", err)
	}
}

func TestWaitForURL_PollsUntilMatch(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	var calls int32
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		if atomic.AddInt32(&calls, 1) < 3 {
			return "https://example.com/login", nil
		}
		return "https://example.com/dashboard", nil
	}}, nopLogger{})

	if err := m.WaitForURL("https://example.com/dashboard", time.Second); err != nil {
		t.Fatalf("轮询到匹配后应成功: %v", err)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Fatalf("应至少轮询 3 次才命中，实际=%d", calls)
	}
}

func TestWaitForText_ContainsTarget(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	m := NewPageManager(&stubPage{evalFn: func(script string) (interface{}, error) {
		if !strings.Contains(script, "#status") {
			t.Fatalf("WaitForText 应查询目标 selector，实际脚本=%q", script)
		}
		return "订单提交成功", nil
	}}, nopLogger{})

	if err := m.WaitForText("#status", "成功", 200*time.Millisecond); err != nil {
		t.Fatalf("文本包含目标应立即成功: %v", err)
	}
}

func TestWaitForText_TimesOutWhenAbsent(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		return "加载中...", nil
	}}, nopLogger{})

	err := m.WaitForText("#status", "成功", 30*time.Millisecond)
	if err == nil {
		t.Fatal("文本恒不出现时必须返回超时错误，不能首次 tick 即 return nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("错误信息应含 timeout: %v", err)
	}
}

func TestWaitForText_ProbeErrorFailsFast(t *testing.T) {
	withFastPoll(t, time.Millisecond)
	want := errors.New("eval blocked")
	m := NewPageManager(&stubPage{evalFn: func(string) (interface{}, error) {
		return nil, want
	}}, nopLogger{})

	err := m.WaitForText("#status", "成功", time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("应包装原始 probe 错误: %v", err)
	}
}
