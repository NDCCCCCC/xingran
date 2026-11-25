package browser

import (
	"strings"
	"testing"

	"github.com/xingran-next/rpa-worker/internal/config"
)

// TestNewLauncher_NoSandboxIsBareFlag 锁定 2026-11-25 生产 UAT 缺陷：
// launcher.Set(name, values...) 的语义是「一个 flag 名 + 它的值」。原写法把多个
// flag 名连写进一次 Set，产出 --no-sandbox=--disable-dev-shm-usage,--disable-gpu,...，
// Chrome 能启动但沙箱参数畸形，页面导航永久卡住（Progress 00%）。
func TestNewLauncher_NoSandboxIsBareFlag(t *testing.T) {
	cfg := &config.BrowserConfig{Headless: true, ChromePath: "/usr/bin/chromium-browser"}
	l := newLauncher(cfg)
	args := strings.Join(l.FormatArgs(), " ")

	if !strings.Contains(args, "--no-sandbox") {
		t.Fatalf("缺少 --no-sandbox: %s", args)
	}
	if strings.Contains(args, "--no-sandbox=") {
		t.Fatalf("--no-sandbox 被赋了值（应为裸 flag）: %s", args)
	}
	// 其余裸 flag 也应各自独立出现。
	for _, bare := range []string{"--disable-dev-shm-usage", "--disable-gpu", "--no-first-run"} {
		if !strings.Contains(args, bare) {
			t.Fatalf("缺少 %s: %s", bare, args)
		}
	}
	// 带值 flag 形态正确。
	if !strings.Contains(args, "--lang=zh-CN") {
		t.Fatalf("--lang 未正确设置: %s", args)
	}
}
