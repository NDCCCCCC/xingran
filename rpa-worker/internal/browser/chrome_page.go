package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ChromePage Chrome浏览器页面实现
type ChromePage struct {
	browser *rod.Browser
	page    *rod.Page
	logger  logger.Logger
	config  *config.BrowserConfig
}

// NewChromePage 创建Chrome页面
func NewChromePage(ctx context.Context, cfg *config.BrowserConfig, log logger.Logger) (*ChromePage, error) {
	// 准备Chrome启动参数
	launcherPath := launcher.New()

	// 设置headless模式
	if cfg.Headless {
		launcherPath = launcherPath.Headless(true)
	}

	// 设置Chrome路径
	if chromePath := cfg.ChromePath; chromePath != "" {
		launcherPath = launcherPath.Bin(chromePath)
	}

	// 添加Chrome启动参数（容器模式必需）
	launcherPath = launcherPath.MustAppend(
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

	// 添加自定义flags
	for _, flag := range cfg.ChromeFlags {
		launcherPath = launcherPath.MustAppend(flag)
	}

	// 启动浏览器
	launcherURL := launcherPath.MustLaunch()
	browser := rod.New().ControlURL(launcherURL).MustConnect()

	// 创建空白页面
	page := browser.MustPage()

	// 设置视口大小
	err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  cfg.ViewportWidth,
		Height: cfg.ViewportHeight,
	})
	if err != nil {
		log.Warn("设置视口大小失败", logger.Err(err))
	}

	// 设置默认超时（使用导航超时作为默认值，确保所有操作都有足够时间）
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

	// 创建带有导航超时的页面实例
	navPage := p.page.Timeout(p.config.NavigationTimeout)

	// 导航到 URL（使用导航超时）
	err := navPage.Navigate(url)
	if err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}

	// 等待页面加载完成（WaitLoad 等待页面 load 事件触发）
	navPage.MustWaitLoad()

	// 额外等待网络空闲（确保所有异步请求完成）
	// 这对于单页应用(SPA)特别重要，使用较短的超时避免过度等待
	waitIdleTimeout := p.config.Timeout
	if waitIdleTimeout > p.config.NavigationTimeout/2 {
		waitIdleTimeout = p.config.NavigationTimeout / 2
	}
	err = navPage.WaitIdle(waitIdleTimeout)
	if err != nil {
		p.logger.Warn("等待网络空闲超时（继续执行）", logger.Err(err))
		// 不返回错误，因为页面可能已经基本可用
	}

	// 等待 body 元素存在（确保 DOM 已就绪）
	_, err = navPage.Timeout(p.config.Timeout).Element("body")
	if err != nil {
		return fmt.Errorf("等待 body 元素超时: %w", err)
	}

	p.logger.Debug("页面加载完成", logger.String("url", url))
	return nil
}

// Close 关闭页面
func (p *ChromePage) Close() error {
	p.logger.Debug("关闭Chrome页面")

	if p.page != nil {
		err := p.page.Close()
		if err != nil {
			p.logger.Warn("关闭页面失败", logger.Err(err))
		}
	}
	if p.browser != nil {
		return p.browser.Close()
	}
	return nil
}

// Click 点击元素
func (p *ChromePage) Click(selector string) error {
	p.logger.Debug("点击元素", logger.String("selector", selector))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	err = el.Click("left", 1)
	if err != nil {
		return fmt.Errorf("点击失败: %w", err)
	}

	return nil
}

// Fill 填写表单
func (p *ChromePage) Fill(selector, value string) error {
	p.logger.Debug("填写表单",
		logger.String("selector", selector),
		logger.String("value", maskValue(value)))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	err = el.Input(value)
	if err != nil {
		return fmt.Errorf("填写失败: %w", err)
	}

	return nil
}

// Select 选择下拉框
func (p *ChromePage) Select(selector, value string) error {
	p.logger.Debug("选择下拉框",
		logger.String("selector", selector),
		logger.String("value", value))

	// 使用JavaScript选择
	script := fmt.Sprintf(`
		(function() {
			const select = document.querySelector('%s');
			if (!select) throw new Error('Element not found');
			select.value = '%s';
			select.dispatchEvent(new Event('change', { bubbles: true }));
		})();
	`, selector, value)

	_, err := p.page.Eval(script)
	if err != nil {
		return fmt.Errorf("选择失败: %w", err)
	}

	return nil
}

// WaitFor 等待元素出现
func (p *ChromePage) WaitFor(selector string, timeout time.Duration) error {
	p.logger.Debug("等待元素",
		logger.String("selector", selector),
		logger.Duration("timeout", timeout))

	_, err := p.page.Timeout(timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("等待元素超时: %w", err)
	}

	return nil
}

// Screenshot 截图
func (p *ChromePage) Screenshot() ([]byte, error) {
	p.logger.Debug("执行截图")

	data, err := p.page.Screenshot(true, nil)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}

	return data, nil
}

// Evaluate 执行JavaScript
func (p *ChromePage) Evaluate(script string) (interface{}, error) {
	p.logger.Debug("执行JavaScript")

	result, err := p.page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}

	return result.Value.Val, nil
}

// GetText 获取元素文本
func (p *ChromePage) GetText(selector string) (string, error) {
	p.logger.Debug("获取文本", logger.String("selector", selector))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	text, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("获取文本失败: %w", err)
	}

	return text, nil
}

// GetValue 获取输入框的值
func (p *ChromePage) GetValue(selector string) (string, error) {
	p.logger.Debug("获取值", logger.String("selector", selector))

	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return '';
			if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
				return el.value;
			}
			if (el.tagName === 'SELECT') {
				return el.value;
			}
			return '';
		})();
	`, selector)

	res, err := p.page.Eval(script)
	if err != nil {
		return "", fmt.Errorf("执行脚本失败: %w", err)
	}
	return res.Value.Str(), nil
}

// GetHTML 获取元素HTML
func (p *ChromePage) GetHTML(selector string) (string, error) {
	p.logger.Debug("获取HTML", logger.String("selector", selector))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	html, err := el.HTML()
	if err != nil {
		return "", fmt.Errorf("获取HTML失败: %w", err)
	}

	return html, nil
}

// GetAttribute 获取元素属性
func (p *ChromePage) GetAttribute(selector, attrName string) (string, error) {
	p.logger.Debug("获取属性",
		logger.String("selector", selector),
		logger.String("attribute", attrName))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	attr, err := el.Attribute(attrName)
	if err != nil {
		return "", fmt.Errorf("获取属性失败: %w", err)
	}

	if attr == nil {
		return "", nil
	}
	return *attr, nil
}

// ScrollDown 向下滚动
func (p *ChromePage) ScrollDown(pixels int) error {
	p.logger.Debug("向下滚动", logger.Int("pixels", pixels))

	script := fmt.Sprintf("window.scrollBy(0, %d)", pixels)
	_, err := p.page.Eval(script)
	if err != nil {
		return fmt.Errorf("滚动失败: %w", err)
	}

	return nil
}

// ScrollUp 向上滚动
func (p *ChromePage) ScrollUp(pixels int) error {
	p.logger.Debug("向上滚动", logger.Int("pixels", pixels))

	script := fmt.Sprintf("window.scrollBy(0, -%d)", pixels)
	_, err := p.page.Eval(script)
	if err != nil {
		return fmt.Errorf("滚动失败: %w", err)
	}

	return nil
}

// ScrollToElement 滚动到元素
func (p *ChromePage) ScrollToElement(selector string) error {
	p.logger.Debug("滚动到元素", logger.String("selector", selector))

	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	err = el.ScrollIntoView()
	if err != nil {
		return fmt.Errorf("滚动到元素失败: %w", err)
	}

	return nil
}

// GetBrowser 获取浏览器实例
func (p *ChromePage) GetBrowser() *rod.Browser {
	return p.browser
}

// GetPage 获取页面实例
func (p *ChromePage) GetPage() *rod.Page {
	return p.page
}
