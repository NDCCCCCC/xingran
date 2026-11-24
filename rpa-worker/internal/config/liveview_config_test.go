package config

import (
	"path/filepath"
	"testing"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01/05：screencast 配置 schema 单测
//
//   TestLiveView_DefaultsWhenAbsent     未配置 liveview 节 → 默认启用 + 默认值
//   TestLiveView_ExplicitDisable        显式 enabled:false → 保持关闭（不被默认值覆盖）
//   TestLiveView_CustomValues           自定义取值被采纳
//   TestLiveView_InvalidDefaultMode     非法默认模式回落 copilot（不静默走错分支）
//   TestLiveView_ExampleYAML            仓库内的 config.example.yaml 可被真实解析
// ============================================================================

// TestLiveView_DefaultsWhenAbsent 未配置 liveview 时默认启用并填默认值。
func TestLiveView_DefaultsWhenAbsent(t *testing.T) {
	cfg := writeConfig(t, `
worker:
  max_concurrency: 4
`)
	if !cfg.LiveView.Enabled {
		t.Fatalf("未配置 liveview 时应默认启用（LIVEVIEW 是里程碑交付项）")
	}
	if cfg.LiveView.MaxStreams != DefaultLiveViewMaxStreams {
		t.Fatalf("MaxStreams=%d，期望默认 %d（LIVEVIEW-01/05 单 Worker 上限 2）",
			cfg.LiveView.MaxStreams, DefaultLiveViewMaxStreams)
	}
	if cfg.LiveView.Quality != DefaultLiveViewQuality ||
		cfg.LiveView.MaxWidth != DefaultLiveViewMaxWidth ||
		cfg.LiveView.MaxHeight != DefaultLiveViewMaxHeight {
		t.Fatalf("screencast 参数未填默认: q=%d w=%d h=%d",
			cfg.LiveView.Quality, cfg.LiveView.MaxWidth, cfg.LiveView.MaxHeight)
	}
	if cfg.LiveView.EveryNthFrame != DefaultLiveViewEveryNthFrame {
		t.Fatalf("EveryNthFrame=%d，期望 %d", cfg.LiveView.EveryNthFrame, DefaultLiveViewEveryNthFrame)
	}
	if cfg.LiveView.IdleTimeout != DefaultLiveViewIdleTimeout {
		t.Fatalf("IdleTimeout=%v，期望 %v（LIVEVIEW-05 断开 30s 停流）",
			cfg.LiveView.IdleTimeout, DefaultLiveViewIdleTimeout)
	}
	if cfg.LiveView.DefaultMode != DefaultLiveViewMode {
		t.Fatalf("DefaultMode=%q，期望 %q（copilot 默认）", cfg.LiveView.DefaultMode, DefaultLiveViewMode)
	}
}

// TestLiveView_ExplicitDisable 显式 false 必须保持关闭。
func TestLiveView_ExplicitDisable(t *testing.T) {
	cfg := writeConfig(t, `
liveview:
  enabled: false
`)
	if cfg.LiveView.Enabled {
		t.Fatalf("显式 enabled:false 被默认值覆盖（配置语义失效）")
	}
	if !cfg.LiveView.EnabledSet {
		t.Fatalf("显式配置时 EnabledSet 应为 true（用于区分「未配置」与「显式 false」）")
	}
	// 其它参数仍填默认值（关闭开关不影响 schema 完整性）。
	if cfg.LiveView.MaxStreams != DefaultLiveViewMaxStreams {
		t.Fatalf("关闭时 MaxStreams 仍应为默认 %d，实际 %d", DefaultLiveViewMaxStreams, cfg.LiveView.MaxStreams)
	}
}

// TestLiveView_CustomValues 自定义取值被采纳。
func TestLiveView_CustomValues(t *testing.T) {
	cfg := writeConfig(t, `
liveview:
  enabled: true
  max_streams: 1
  quality: 40
  max_width: 854
  max_height: 480
  every_nth_frame: 1
  ack_timeout: 500ms
  idle_timeout: 10s
  takeover_timeout: 60s
  default_mode: manual
`)
	if cfg.LiveView.MaxStreams != 1 {
		t.Fatalf("MaxStreams=%d，期望 1", cfg.LiveView.MaxStreams)
	}
	if cfg.LiveView.Quality != 40 || cfg.LiveView.MaxWidth != 854 || cfg.LiveView.MaxHeight != 480 {
		t.Fatalf("自定义 screencast 参数未生效: q=%d w=%d h=%d",
			cfg.LiveView.Quality, cfg.LiveView.MaxWidth, cfg.LiveView.MaxHeight)
	}
	if cfg.LiveView.EveryNthFrame != 1 {
		t.Fatalf("EveryNthFrame=%d，期望 1", cfg.LiveView.EveryNthFrame)
	}
	if cfg.LiveView.IdleTimeout.String() != "10s" {
		t.Fatalf("IdleTimeout=%v，期望 10s", cfg.LiveView.IdleTimeout)
	}
	if cfg.LiveView.DefaultMode != "manual" {
		t.Fatalf("DefaultMode=%q，期望 manual", cfg.LiveView.DefaultMode)
	}
}

// TestLiveView_InvalidDefaultMode 非法默认模式回落 copilot（不静默走错分支）。
func TestLiveView_InvalidDefaultMode(t *testing.T) {
	cfg := writeConfig(t, `
liveview:
  default_mode: wizard
`)
	if cfg.LiveView.DefaultMode != DefaultLiveViewMode {
		t.Fatalf("非法模式应回落 %q，实际 %q", DefaultLiveViewMode, cfg.LiveView.DefaultMode)
	}
}

// TestLiveView_ExampleYAML 仓库内的 configs/config.example.yaml 必须能被真实解析。
//
// 防止「文档里的配置块写错缩进 → 运维照抄后启动失败」这一类回归。
func TestLiveView_ExampleYAML(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "config.example.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("解析 config.example.yaml 失败（示例配置不可用）: %v", err)
	}
	if !cfg.LiveView.Enabled {
		t.Fatalf("示例配置里 liveview.enabled 应为 true")
	}
	if cfg.LiveView.MaxStreams != DefaultLiveViewMaxStreams {
		t.Fatalf("示例配置 MaxStreams=%d，期望 %d", cfg.LiveView.MaxStreams, DefaultLiveViewMaxStreams)
	}
	if cfg.LiveView.DefaultMode != DefaultLiveViewMode {
		t.Fatalf("示例配置 DefaultMode=%q，期望 %q", cfg.LiveView.DefaultMode, DefaultLiveViewMode)
	}
	if cfg.LiveView.Quality != DefaultLiveViewQuality {
		t.Fatalf("示例配置 Quality=%d，期望 %d", cfg.LiveView.Quality, DefaultLiveViewQuality)
	}
	if cfg.LiveView.IdleTimeout != DefaultLiveViewIdleTimeout {
		t.Fatalf("示例配置 IdleTimeout=%v，期望 %v", cfg.LiveView.IdleTimeout, DefaultLiveViewIdleTimeout)
	}
}
