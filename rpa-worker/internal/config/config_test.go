package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// Phase 154 / ENGINE-01 + ENGINE-04：配置 schema 单测
//   - max_pages 与 max_concurrency 合并的 4 种组合（含旧键兼容告警）
//   - EffectiveMaxPages 是唯一真相
//   - undefined_variable_policy 取值校验
// 这份行为说明同时供 Phase 157 的 CI 漂移校验读取（见 154-SUMMARY「旧键兼容行为」）。
// ============================================================================

// writeConfig 写一个临时 yaml 并加载。
func writeConfig(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load 失败: %v\n配置:\n%s", err, body)
	}
	return cfg
}

// TestMerge_BothKeysMaxConcurrencyWins 同时给两个键 → max_concurrency 胜出 + 告警。
func TestMerge_BothKeysMaxConcurrencyWins(t *testing.T) {
	cfg := writeConfig(t, `
worker:
  max_concurrency: 4
browser:
  max_pages: 10
executor:
  undefined_variable_policy: "empty"
`)
	if cfg.Worker.MaxConcurrency != 4 {
		t.Fatalf("max_concurrency=%d，期望 4", cfg.Worker.MaxConcurrency)
	}
	if cfg.Browser.MaxConcurrency != 4 {
		t.Fatalf("浏览器侧只读视图=%d，期望跟随 4", cfg.Browser.MaxConcurrency)
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("应产生 1 条旧键告警，实际 %d: %v", len(cfg.Warnings), cfg.Warnings)
	}
}

// TestMerge_LegacyOnlyMigrates 只给旧键 → 迁移到 max_concurrency + 告警。
func TestMerge_LegacyOnlyMigrates(t *testing.T) {
	cfg := writeConfig(t, `
browser:
  max_pages: 8
`)
	if cfg.Worker.MaxConcurrency != 8 {
		t.Fatalf("旧键应迁移为 max_concurrency=8，实际 %d", cfg.Worker.MaxConcurrency)
	}
	if cfg.Browser.MaxConcurrency != 8 {
		t.Fatalf("浏览器侧视图=%d", cfg.Browser.MaxConcurrency)
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("应产生 1 条迁移告警: %v", cfg.Warnings)
	}
}

// TestMerge_NeitherUsesDefault 都不给 → 默认 5（不残留历史 3/10 分叉）。
func TestMerge_NeitherUsesDefault(t *testing.T) {
	cfg := writeConfig(t, "worker:\n  name: test\n")
	if cfg.Worker.MaxConcurrency != 5 {
		t.Fatalf("默认并发页面数应为 5，实际 %d", cfg.Worker.MaxConcurrency)
	}
	if cfg.Browser.MaxConcurrency != 5 {
		t.Fatalf("浏览器侧视图=%d", cfg.Browser.MaxConcurrency)
	}
	if len(cfg.Warnings) != 0 {
		t.Fatalf("无旧键时不应告警: %v", cfg.Warnings)
	}
}

// TestEffectiveMaxPages 唯一真相函数。
func TestEffectiveMaxPages(t *testing.T) {
	cfg := &Config{Worker: WorkerConfig{MaxConcurrency: 7}, Browser: BrowserConfig{MaxConcurrency: 7}}
	if got := EffectiveMaxPages(cfg); got != 7 {
		t.Fatalf("EffectiveMaxPages=%d", got)
	}
	if got := EffectiveMaxPages(nil); got != 0 {
		t.Fatalf("nil 配置应返回 0，实际 %d", got)
	}
}

// TestUndefinedVariablePolicy 策略默认值与非法值拒绝。
func TestUndefinedVariablePolicy(t *testing.T) {
	cfg := writeConfig(t, "worker:\n  name: test\n")
	if cfg.Executor.UndefinedVariablePolicy != UndefinedVariableEmpty {
		t.Fatalf("默认策略应为 empty，实际 %q", cfg.Executor.UndefinedVariablePolicy)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte("executor:\n  undefined_variable_policy: \"loose\"\n"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("非法策略值必须在加载期报错")
	}
}

// TestEnvOverrideMaxConcurrency 环境变量覆盖后浏览器侧视图同步。
func TestEnvOverrideMaxConcurrency(t *testing.T) {
	cfg := writeConfig(t, "worker:\n  max_concurrency: 3\n")
	t.Setenv("MAX_CONCURRENCY", "8")
	// 直接调用 override + 同步（Load 已读过文件，这里模拟 env 生效路径）
	overrideWithEnv(cfg)
	syncPageConcurrency(cfg)
	if cfg.Worker.MaxConcurrency != 8 {
		t.Fatalf("env 覆盖后 max_concurrency=%d", cfg.Worker.MaxConcurrency)
	}
	if cfg.Browser.MaxConcurrency != 8 {
		t.Fatalf("env 覆盖后浏览器侧视图=%d（必须同步）", cfg.Browser.MaxConcurrency)
	}
}
