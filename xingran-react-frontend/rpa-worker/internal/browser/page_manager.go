package browser

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
)

// PageManager 页面管理器
type PageManager struct {
	page   Page
	logger logger.Logger
}

// NewPageManager 创建页面管理器
func NewPageManager(page Page, log logger.Logger) *PageManager {
	return &PageManager{
		page:   page,
		logger: log,
	}
}

// Goto 导航到 URL
func (m *PageManager) Goto(url string) error {
	m.logger.Debug("导航到页面", logger.String("url", url))
	return m.page.Goto(url)
}

// Click 点击元素
func (m *PageManager) Click(selector string) error {
	m.logger.Debug("点击元素", logger.String("selector", selector))
	return m.page.Click(selector)
}

// Fill 填写表单
func (m *PageManager) Fill(selector, value string) error {
	m.logger.Debug("填写表单", 
		logger.String("selector", selector),
		logger.String("value", maskValue(value)))
	return m.page.Fill(selector, value)
}

// Select 选择下拉选项
func (m *PageManager) Select(selector, value string) error {
	m.logger.Debug("选择选项",
		logger.String("selector", selector),
		logger.String("value", value))
	return m.page.Select(selector, value)
}

// WaitFor 等待元素出现
func (m *PageManager) WaitFor(selector string, timeout time.Duration) error {
	m.logger.Debug("等待元素",
		logger.String("selector", selector),
		logger.Duration("timeout", timeout))
	return m.page.WaitFor(selector, timeout)
}

// Screenshot 截图
func (m *PageManager) Screenshot() ([]byte, error) {
	return m.page.Screenshot()
}

// Evaluate 执行 JavaScript
func (m *PageManager) Evaluate(script string) (interface{}, error) {
	return m.page.Evaluate(script)
}

// waitPollInterval 是 WaitForURL / WaitForText 的轮询间隔。
// 抽成包级变量，使单测可缩短轮询以覆盖超时路径（Phase 137 / FIX-08）。
var waitPollInterval = 100 * time.Millisecond

// currentURLScript 取当前页面 URL（供 WaitForURL 轮询校验）。
const currentURLScript = "window.location.href"

// matchURL 判断当前 URL 是否满足 pattern：
// 支持精确匹配与 "*" 通配符（filepath.Match 语义），也兼容子串包含。
func matchURL(current, pattern string) bool {
	if pattern == "" {
		return false
	}
	if ok, err := path.Match(pattern, current); err == nil && ok {
		return true
	}
	return strings.Contains(current, pattern)
}

// pollWait 按 waitPollInterval 轮询 probe，直到 probe 返回 true、ctx 超时或
// probe 返回错误。超时/失败路径返回带上下文的 error（不再首次 tick 即 return nil）。
func pollWait(ctx context.Context, timeout time.Duration, what string, probe func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()

	// 立即做一次探测，避免无条件等待首个 tick。
	if ok, err := probe(); err != nil {
		return fmt.Errorf("wait for %s probe failed: %w", what, err)
	} else if ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %s timeout after %s: %w", what, timeout, ctx.Err())
		case <-ticker.C:
			ok, err := probe()
			if err != nil {
				return fmt.Errorf("wait for %s probe failed: %w", what, err)
			}
			if ok {
				return nil
			}
		}
	}
}

// WaitForURL 等待 URL 变化（Phase 137 / FIX-08：真实轮询校验当前 URL）。
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
	m.logger.Debug("等待 URL",
		logger.String("pattern", pattern),
		logger.Duration("timeout", timeout))
	return pollWait(context.Background(), timeout, "URL("+pattern+")", func() (bool, error) {
		val, err := m.page.Evaluate(currentURLScript)
		if err != nil {
			return false, err
		}
		current, _ := val.(string)
		return matchURL(current, pattern), nil
	})
}

// WaitForText 等待文本出现（Phase 137 / FIX-08：真实轮询校验元素文本）。
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
	m.logger.Debug("等待文本",
		logger.String("selector", selector),
		logger.String("text", text),
		logger.Duration("timeout", timeout))
	script := fmt.Sprintf(`(function(){var el=document.querySelector(%q);return el?el.textContent:"";})()`, selector)
	return pollWait(context.Background(), timeout, "text("+text+")", func() (bool, error) {
		val, err := m.page.Evaluate(script)
		if err != nil {
			return false, err
		}
		got, _ := val.(string)
		return strings.Contains(got, text), nil
	})
}

// maskValue 隐藏敏感值
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
