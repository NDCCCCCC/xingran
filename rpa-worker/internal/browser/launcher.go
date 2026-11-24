package browser

import (
	"strings"

	"github.com/go-rod/rod/lib/launcher"
	launcherflags "github.com/go-rod/rod/lib/launcher/flags"
	"github.com/xingran-next/rpa-worker/internal/config"
)

// newLauncher 按 BrowserConfig 组装 Chrome 启动器（容器内必需 flags 全在这）。
// Phase 154 / ENGINE-01：Chrome 进程只由 Manager 启一次；NewChromePage 的
// 独立路径也复用本函数，保证两条路径启动参数零漂移。
func newLauncher(cfg *config.BrowserConfig) *launcher.Launcher {
	l := launcher.New()
	if cfg == nil {
		return l
	}
	if cfg.Headless {
		l = l.Headless(true)
	}
	if chromePath := cfg.ChromePath; chromePath != "" {
		l = l.Bin(chromePath)
	}
	l = l.Set(
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--disable-extensions",
		"--disable-background-networking",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-translate",
		"--disable-save-password-bubble",
		"--disable-autofill",
		"--lang=zh-CN",
		"--accept-lang=zh-CN,zh,en-US,en",
		"--disable-sync",
		"--disable-blink-features=AutomationControlled",
	)
	for _, flag := range cfg.ChromeFlags {
		// 支持 "--headless=new" / "--disable-gpu" 两种书写：launcher.Set 的
		// 键名不允许含 '='，故显式拆成 (Flag, Value) 两段。
		name := strings.TrimSpace(flag)
		name = strings.TrimPrefix(name, "--")
		if i := strings.Index(name, "="); i >= 0 {
			l = l.Set(launcherflags.Flag(name[:i]), name[i+1:])
			continue
		}
		l = l.Set(launcherflags.Flag(name))
	}
	return l
}
