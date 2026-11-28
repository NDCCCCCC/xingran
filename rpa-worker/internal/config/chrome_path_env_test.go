package config

import (
	"testing"
)

// TestChromePathFromEnv 锁定 2026-11-25 生产 UAT 缺陷：
// 容器镜像只设 CHROMIUM_PATH / CHROME_BIN（Dockerfile ENV），从不写 config.yaml
// 的 browser.chrome_path。缺 env 回退时 ChromePath 为空，rod launcher 会去
// storage.googleapis.com 自动下载 chrome-linux.zip（国内不可达），表现为任务
// 永久卡在 "Progress: 00%"。
func TestChromePathFromEnv(t *testing.T) {
	cases := []struct {
		name string
		envs map[string]string
		want string
	}{
		{"CHROMIUM_PATH 优先", map[string]string{"CHROMIUM_PATH": "/usr/bin/chromium-browser", "CHROME_BIN": "/x"}, "/usr/bin/chromium-browser"},
		{"CHROME_BIN 次选", map[string]string{"CHROME_BIN": "/usr/bin/chromium"}, "/usr/bin/chromium"},
		{"CHROME_PATH 兜底", map[string]string{"CHROME_PATH": "/opt/chrome"}, "/opt/chrome"},
		{"全空则留空（由 launcher 决定行为）", map[string]string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, k := range []string{"CHROMIUM_PATH", "CHROME_BIN", "CHROME_PATH"} {
				t.Setenv(k, "")
			}
			for k, v := range c.envs {
				t.Setenv(k, v)
			}
			cfg := &Config{}
			overrideWithEnv(cfg)
			if got := cfg.Browser.ChromePath; got != c.want {
				t.Fatalf("ChromePath = %q, want %q", got, c.want)
			}
		})
	}
}

// TestChromePathConfigWins 配置显式指定时 env 不得覆盖。
func TestChromePathConfigWins(t *testing.T) {
	t.Setenv("CHROMIUM_PATH", "/from/env")
	cfg := &Config{}
	cfg.Browser.ChromePath = "/from/config"
	overrideWithEnv(cfg)
	if cfg.Browser.ChromePath != "/from/config" {
		t.Fatalf("显式配置被 env 覆盖: %q", cfg.Browser.ChromePath)
	}
}
