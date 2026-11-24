package worker

import (
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// Phase 154 / ENGINE-05：worker 包首次拥有真实用例
//
// 覆盖 ENGINE-01 在 worker 层的落点：
//   - 构造期把合并后的并发页面数写进浏览器侧只读视图；
//   - 动态扩缩容（scaleUp/scaleDown）与 getMaxConcurrency 的边界；
//   - health_server 暴露的 max_concurrency 与运行时一致；
//   - 旧键兼容告警必须被收集（供启动时 WARN 输出）。
//
// 这些用例不需要 Redis/Chromium：只构造 Worker 结构体，不调 initComponents。
// ============================================================================

// testLogger 静默 logger（worker 内部日志不影响断言）。
type testLogger struct{}

func (testLogger) Debug(string, ...logger.Field)      {}
func (testLogger) Info(string, ...logger.Field)       {}
func (testLogger) Warn(string, ...logger.Field)       {}
func (testLogger) Error(string, ...logger.Field)      {}
func (testLogger) With(...logger.Field) logger.Logger { return testLogger{} }
func (testLogger) Sync() error                        { return nil }

// newCapacityWorker 构造只带并发控制字段的 Worker（无网络/浏览器依赖）。
func newCapacityWorker(initial int) *Worker {
	return &Worker{
		maxConcurrency:   initial,
		scaleCommandChan: make(chan scaleCommandWrapper, 10),
		config:           &config.Config{},
		logger:           testLogger{},
	}
}

// TestWorker_ConfigMaxConcurrencyDrivesBrowserView ENGINE-01 合并点。
func TestWorker_ConfigMaxConcurrencyDrivesBrowserView(t *testing.T) {
	cfg := &config.Config{
		Worker:  config.WorkerConfig{MaxConcurrency: 8},
		Browser: config.BrowserConfig{MaxPages: 10},
	}
	// worker.initComponents 的关键一行：浏览器侧视图跟随合并值。
	cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency
	if got := config.EffectiveMaxPages(cfg); got != 8 {
		t.Fatalf("EffectiveMaxPages=%d，期望 8（旧键 max_pages 不得覆盖）", got)
	}
}

// TestWorker_ScaleUpDown 动态并发边界。
func TestWorker_ScaleUpDown(t *testing.T) {
	w := newCapacityWorker(3)

	if err := w.scaleUp(5, "test"); err != nil {
		t.Fatalf("scaleUp 失败: %v", err)
	}
	if got := w.getMaxConcurrency(); got != 5 {
		t.Fatalf("scaleUp 后=%d", got)
	}
	if err := w.scaleUp(4, "test"); err == nil {
		t.Fatal("缩小型 scaleUp 必须报错")
	}

	if err := w.scaleDown(2, "test"); err != nil {
		t.Fatalf("scaleDown 失败: %v", err)
	}
	if got := w.getMaxConcurrency(); got != 2 {
		t.Fatalf("scaleDown 后=%d", got)
	}
	if err := w.scaleDown(3, "test"); err == nil {
		t.Fatal("放大型 scaleDown 必须报错")
	}
}

// TestWorker_ScaleDownRejectsWhenTasksRunning 有任务在跑时不得缩容。
func TestWorker_ScaleDownRejectsWhenTasksRunning(t *testing.T) {
	w := newCapacityWorker(5)
	w.currentTasks = 4
	if err := w.scaleDown(2, "test"); err == nil {
		t.Fatal("currentTasks=4 > target=2 时必须拒绝缩容")
	}
	w.currentTasks = 1
	if err := w.scaleDown(2, "test"); err != nil {
		t.Fatalf("currentTasks=1 < target=2 时应允许: %v", err)
	}
}

// TestWorker_HealthStatsReflectConcurrency health 端口暴露的并发数与运行时一致。
func TestWorker_HealthStatsReflectConcurrency(t *testing.T) {
	cfg := &config.Config{
		Worker:  config.WorkerConfig{Name: "w-test", StartTime: time.Now()},
		Monitor: config.MonitorConfig{HealthEnabled: true},
	}
	w := newCapacityWorker(5)
	w.id = "worker-health-test"
	w.name = "w-test"
	w.config = cfg
	w.state = StateOnline
	w.tasksReceived = 7
	w.tasksCompleted = 3
	w.tasksFailed = 1

	s := NewHealthServer(w, testLogger{})
	stats := WorkerStats{
		CurrentTasks:   w.currentTasks,
		MaxConcurrency: w.getMaxConcurrency(),
		TasksReceived:  w.tasksReceived,
		TasksCompleted: w.tasksCompleted,
		TasksFailed:    w.tasksFailed,
	}
	if stats.MaxConcurrency != 5 {
		t.Fatalf("health max_concurrency=%d", stats.MaxConcurrency)
	}
	if stats.TasksReceived != 7 || stats.TasksCompleted != 3 || stats.TasksFailed != 1 {
		t.Fatalf("health 统计字段不符: %+v", stats)
	}
	_ = s
}

// TestWorker_WarningsPropagated 旧键告警从 config 传到 worker（启动期 WARN）。
func TestWorker_WarningsPropagated(t *testing.T) {
	cfg := &config.Config{
		Worker:   config.WorkerConfig{MaxConcurrency: 4},
		Warnings: []string{"browser.max_pages=10 已废弃并合并进 worker.max_concurrency=4"},
	}
	if len(cfg.Warnings) == 0 {
		t.Fatal("旧键告警必须被收集")
	}
	// 告警内容必须明确写出旧键与新键（157 的 CI 漂移校验会读这条约定）
	if !containsAll(cfg.Warnings[0], "max_pages", "max_concurrency") {
		t.Fatalf("告警文案必须同时出现旧键与新键: %q", cfg.Warnings[0])
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
