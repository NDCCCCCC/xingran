package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	launcherflags "github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ChromePage Chrome浏览器页面实现（Phase 137 / FIX-08）。
// 对齐根副本 rpa-worker/internal/browser/chrome_page.go，但改用 go-rod 的
// error 版 API（Launch/Connect/Page/WaitLoad），遵循「返回 error 不 panic」约定。
type ChromePage struct {
	browser *rod.Browser
	page    *rod.Page
	logger  logger.Logger
	config  *config.BrowserConfig
}

// NewChromePage 创建Chrome页面（真实 Chrome，通过 go-rod launcher 启动）。
func NewChromePage(ctx context.Context, cfg *config.BrowserConfig, log logger.Logger) (*ChromePage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("创建Chrome页面失败: BrowserConfig 为空")
	}

	l := launcher.New()
	if cfg.Headless {
		l = l.Headless(true)
	}
	if cfg.ChromePath != "" {
		l = l.Bin(cfg.ChromePath)
	}
	// 容器模式必需的启动参数（与根副本一致）。
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
		l = l.Set(launcherflags.Flag(flag))
	}

	launcherURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("启动Chrome失败: %w", err)
	}

	browser := rod.New().ControlURL(launcherURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("连接Chrome失败: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		_ = browser.Close()
		return nil, fmt.Errorf("创建Chrome页面失败: %w", err)
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  cfg.ViewportWidth,
		Height: cfg.ViewportHeight,
	}); err != nil {
		log.Warn("设置视口大小失败", logger.Err(err))
	}

	// 使用导航超时作为默认超时，确保所有操作都有足够时间。
	defaultTimeout := cfg.Timeout
	if cfg.NavigationTimeout > defaultTimeout {
		defaultTimeout = cfg.NavigationTimeout
	}
	page = page.Timeout(defaultTimeout)

	log.Info("Chrome页面创建成功",
		logger.String("browser_url", launcherURL),
		logger.String("headless", fmt.Sprintf("%v", cfg.Headless)))

	return &ChromePage{
		browser: browser,
		page:    page,
		logger:  log,
		config:  cfg,
	}, nil
}

// Goto 导航到URL
func (p *ChromePage) Goto(url string) error {
	p.logger.Debug("导航到", logger.String("url", url))
	navPage := p.page.Timeout(p.config.NavigationTimeout)
	if err := navPage.Navigate(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}
	if err := navPage.WaitLoad(); err != nil {
		return fmt.Errorf("等待页面加载失败: %w", err)
	}
	// SPA 场景额外等待网络空闲，超时不阻断（页面可能已基本可用）。
	waitIdleTimeout := p.config.Timeout
	if waitIdleTimeout > p.config.NavigationTimeout/2 {
		waitIdleTimeout = p.config.NavigationTimeout / 2
	}
	if err := navPage.WaitIdle(waitIdleTimeout); err != nil {
		p.logger.Warn("等待网络空闲超时（继续执行）", logger.Err(err))
	}
	if _, err := navPage.Timeout(p.config.Timeout).Element("body"); err != nil {
		return fmt.Errorf("等待 body 元素超时: %w", err)
	}
	p.logger.Debug("页面加载完成", logger.String("url", url))
	return nil
}

// Close 关闭页面
func (p *ChromePage) Close() error {
	p.logger.Debug("关闭Chrome页面")
	if p.page != nil {
		if err := p.page.Close(); err != nil {
			p.logger.Warn("关闭页面失败", logger.Err(err))
		}
	}
	if p.browser != nil {
		return p.browser.Close()
	}
	return nil
}

// Screenshot 截图
func (p *ChromePage) Screenshot() ([]byte, error) {
	data, err := p.page.Screenshot(true, nil)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}
	return data, nil
}

// Click 点击元素
func (p *ChromePage) Click(selector string) error {
	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.Click("left", 1); err != nil {
		return fmt.Errorf("点击失败: %w", err)
	}
	return nil
}

// Fill 填写表单
func (p *ChromePage) Fill(selector, value string) error {
	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.Input(value); err != nil {
		return fmt.Errorf("填写失败: %w", err)
	}
	return nil
}

// Select 选择下拉框
func (p *ChromePage) Select(selector, value string) error {
	script := fmt.Sprintf(`
		(function() {
			const select = document.querySelector('%s');
			if (!select) throw new Error('Element not found');
			select.value = '%s';
			select.dispatchEvent(new Event('change', { bubbles: true }));
		})();
	`, selector, value)
	if _, err := p.page.Eval(script); err != nil {
		return fmt.Errorf("选择失败: %w", err)
	}
	return nil
}

// WaitFor 等待元素出现
func (p *ChromePage) WaitFor(selector string, timeout time.Duration) error {
	if _, err := p.page.Timeout(timeout).Element(selector); err != nil {
		return fmt.Errorf("等待元素超时: %w", err)
	}
	return nil
}

// Evaluate 执行JavaScript（返回 location.href / 文本内容等，供 Wait* 真实校验使用）
func (p *ChromePage) Evaluate(script string) (interface{}, error) {
	result, err := p.page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}
	return result.Value.Val, nil
}

// GetBrowser 获取浏览器实例
func (p *ChromePage) GetBrowser() *rod.Browser {
	return p.browser
}

// GetPage 获取页面实例
func (p *ChromePage) GetPage() *rod.Page {
	return p.page
}
