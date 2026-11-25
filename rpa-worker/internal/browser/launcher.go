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
	// 注：ChromePath 为空时 rod launcher 会尝试自动下载 chrome-linux.zip
	// （storage.googleapis.com，国内不可达）→ 任务永久卡在 Progress 00%。
	// 配置侧已在 config.overrideWithEnv 做 CHROMIUM_PATH/CHROME_BIN/CHROME_PATH
	// 回退（2026-11-25 生产 UAT 修复），此处不再重复告警。
	// 逐项 Set：launcher.Set(name, values...) 的语义是「一个 flag 名 + 它的值」，
	// 把多个 flag 名连写会把它们变成第一个 flag 的**值**。
	// 2026-11-25 生产 UAT 实测缺陷：原写法产出
	//   --no-sandbox=--disable-dev-shm-usage,--disable-gpu,...
	// Chrome 仍能启动但沙箱参数畸形，页面导航永久卡住（Progress 00%）。
	for _, flag := range []string{
		"no-sandbox",
		"disable-dev-shm-usage",
		"disable-gpu",
		"disable-extensions",
		"disable-background-networking",
		"no-first-run",
		"no-default-browser-check",
		"disable-translate",
		"disable-save-password-bubble",
		"disable-autofill",
		"disable-sync",
	} {
		l = l.Set(launcherflags.Flag(flag))
	}
	// 带值的 flag 单独设置。
	l = l.Set("lang", "zh-CN")
	l = l.Set("accept-lang", "zh-CN,zh,en-US,en")
	l = l.Set("disable-blink-features", "AutomationControlled")
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
