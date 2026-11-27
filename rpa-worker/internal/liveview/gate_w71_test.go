package liveview

// ============================================================================
// W7.1 / 缺陷 B 的前后对比证据（闸门侧）
//
// 缺陷 B 第 ⑤ 环的生产铁证：`liveview gate waiting {"mode":"copilot","force":true}`
// 之后 10 分钟内无任何后续动作，最终 context deadline exceeded。
//
// 本文件证明「修复前那条路一定卡到超时」：force=true + copilot 模式下
// BeforeAction 既不返回也不放行，直到 ctx 预算耗尽。修复后 executor 在进入
// 本闸门之前用 types.IsNonRetryable 拦掉配置错误（见
// rpa-worker/internal/executor/engine_w71_test.go 的 FailFast 用例，实测 47ms），
// 因此本闸门继续只为「真实运行时失败」服务 —— 设计意图保留，误触发消失。
//
//   TestGate_ForceCopilotBlocksUntilTimeout   对照 A：仍在阻塞 → 直到 ctx 超时
//   TestGate_ForceAutoDoesNotBlock            反向：auto 不打断（既有语义不变）
// ============================================================================

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	app_logger "github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// recordingLogger 记录日志文本（用于断言闸门是否真的进入等待）。
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) append(level, msg string, fields ...app_logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", f.Key, f.Value))
	}
	l.lines = append(l.lines, level+" "+msg+" "+strings.Join(parts, " "))
}

func (l *recordingLogger) Debug(msg string, f ...app_logger.Field) { l.append("DEBUG", msg, f...) }
func (l *recordingLogger) Info(msg string, f ...app_logger.Field)  { l.append("INFO", msg, f...) }
func (l *recordingLogger) Warn(msg string, f ...app_logger.Field)  { l.append("WARN", msg, f...) }
func (l *recordingLogger) Error(msg string, f ...app_logger.Field) { l.append("ERROR", msg, f...) }
func (l *recordingLogger) With(...app_logger.Field) app_logger.Logger {
	return l
}
func (l *recordingLogger) Sync() error { return nil }

func (l *recordingLogger) has(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func (l *recordingLogger) all() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

// TestGate_ForceCopilotBlocksUntilTimeout 对照 A。
func TestGate_ForceCopilotBlocksUntilTimeout(t *testing.T) {
	rec := &recordingLogger{}
	h := NewHub(&config.LiveViewConfig{
		Enabled:         true,
		MaxStreams:      2,
		IdleTimeout:     30 * time.Second,
		TakeoverTimeout: 30 * time.Second,
		DefaultMode:     string(types.LiveModeCopilot),
	}, nil, nil, rec)
	t.Cleanup(h.Close)

	const execID = "w71-gate-force"
	h.Attach(execID, &stubPage{}, types.NewLiveViewContext(execID))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	started := time.Now()
	done := make(chan error, 1)
	go func() { done <- h.BeforeAction(ctx, execID, &types.Action{Type: types.ActionClick}, true) }()

	// 2.5s 内必须仍在阻塞（生产上这里是 10 分钟）。
	select {
	case err := <-done:
		t.Fatalf("copilot + force=true 本应阻塞，却在 %v 后返回 %v", time.Since(started), err)
	case <-time.After(2500 * time.Millisecond):
	}

	// ctx 耗尽后以 ctx 错误结束 —— 这正是生产日志里的形态。
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ctx 耗尽后应返回错误")
		}
		t.Logf("对照 A：阻塞 %v 后以 %v 结束（生产对应 10 分钟 task_timeout）",
			time.Since(started).Round(time.Millisecond), err)
	case <-time.After(3 * time.Second):
		t.Fatal("ctx 耗尽后闸门仍未返回")
	}

	if !rec.has("liveview gate waiting") {
		t.Fatalf("闸门必须留下 gate waiting 日志（生产铁证即此条）:\n%s", rec.all())
	}
	t.Logf("对照 A 日志:\n%s", rec.all())
}

// TestGate_ForceAutoDoesNotBlock 反向保护：auto 模式不打断（既有语义）。
func TestGate_ForceAutoDoesNotBlock(t *testing.T) {
	rec := &recordingLogger{}
	h := NewHub(&config.LiveViewConfig{
		Enabled:         true,
		MaxStreams:      2,
		IdleTimeout:     30 * time.Second,
		TakeoverTimeout: 30 * time.Second,
		DefaultMode:     string(types.LiveModeAuto),
	}, nil, nil, rec)
	t.Cleanup(h.Close)

	const execID = "w71-gate-auto"
	h.Attach(execID, &stubPage{}, types.NewLiveViewContext(execID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started := time.Now()
	if err := h.BeforeAction(ctx, execID, &types.Action{Type: types.ActionClick}, true); err != nil {
		t.Fatalf("auto + force=true 应直接放行，实际 err=%v", err)
	}
	if elapsed := time.Since(started); elapsed > 300*time.Millisecond {
		t.Fatalf("auto 模式被阻塞了 %v（force 不得打断自动任务）", elapsed)
	}
}
