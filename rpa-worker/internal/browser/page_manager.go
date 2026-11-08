package browser

import (
    "context"
    "fmt"
    "path"
    "strings"
    "time"

    "github.com/xingran-next/rpa-worker/internal/logger"
)

// PageManager page manager
type PageManager struct {
    page   Page
    logger logger.Logger
    // selectorEvents TIDY-03 配套：选择器明细累积器（契约
    // docs/RPA-选择器上报明细契约.md；nil 安全——旧构造路径零影响）
    selectorEvents *SelectorEventCollector
}

// NewPageManager create page manager
func NewPageManager(page Page, log logger.Logger) *PageManager {
    return &PageManager{
        page:           page,
        logger:         log,
        selectorEvents: NewSelectorEventCollector(),
    }
}

// SelectorEnvelopeLine TIDY-03 配套：执行收尾取契约信封行（无事件返回 ""）。
func (m *PageManager) SelectorEnvelopeLine() string {
    if m == nil || m.selectorEvents == nil {
        return ""
    }
    return m.selectorEvents.EnvelopeLine()
}

// recordSelector 统一埋点（定位方法收尾调用）。
func (m *PageManager) recordSelector(selector string, start time.Time, err error) {
    if m.selectorEvents != nil {
        m.selectorEvents.Record(selector, start, err)
    }
}

// Goto navigate to URL
func (m *PageManager) Goto(url string) error {
    m.logger.Debug("navigate to page", logger.String("url", url))
    if m.selectorEvents != nil {
        m.selectorEvents.ObserveNavigation(url)
    }
    return m.page.Goto(url)
}

// Click click element
func (m *PageManager) Click(selector string) error {
    m.logger.Debug("click element", logger.String("selector", selector))
    start := time.Now()
    err := m.page.Click(selector)
    m.recordSelector(selector, start, err)
    return err
}

// Fill fill form
func (m *PageManager) Fill(selector, value string) error {
    m.logger.Debug("fill form",
        logger.String("selector", selector),
        logger.String("value", maskValue(value)))
    start := time.Now()
    err := m.page.Fill(selector, value)
    m.recordSelector(selector, start, err)
    return err
}

// Select select dropdown option
func (m *PageManager) Select(selector, value string) error {
    m.logger.Debug("select option",
        logger.String("selector", selector),
        logger.String("value", value))
    start := time.Now()
    err := m.page.Select(selector, value)
    m.recordSelector(selector, start, err)
    return err
}

// WaitFor wait for element
func (m *PageManager) WaitFor(selector string, timeout time.Duration) error {
    m.logger.Debug("wait for element",
        logger.String("selector", selector),
        logger.Duration("timeout", timeout))
    start := time.Now()
    err := m.page.WaitFor(selector, timeout)
    m.recordSelector(selector, start, err)
    return err
}

// Screenshot take screenshot
func (m *PageManager) Screenshot() ([]byte, error) {
    return m.page.Screenshot()
}

// Evaluate execute JavaScript
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

// WaitForURL wait for URL change（Phase 137 / FIX-08：真实轮询校验当前 URL）。
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
    m.logger.Debug("wait for URL",
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

// WaitForText wait for text appearance（Phase 137 / FIX-08：真实轮询校验元素文本）。
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
    m.logger.Debug("wait for text",
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

// maskValue mask sensitive value
func maskValue(value string) string {
    if len(value) <= 4 {
        return "****"
    }
    return value[:2] + "****" + value[len(value)-2:]
}

// GetText get element text content
func (m *PageManager) GetText(selector string) (string, error) {
    m.logger.Debug("get text", logger.String("selector", selector))
    start := time.Now()
    text, err := m.page.GetText(selector)
    m.recordSelector(selector, start, err)
    return text, err
}

// GetValue get input element value
func (m *PageManager) GetValue(selector string) (string, error) {
    m.logger.Debug("get value", logger.String("selector", selector))
    start := time.Now()
    value, err := m.page.GetValue(selector)
    m.recordSelector(selector, start, err)
    return value, err
}

// GetHTML get element HTML content
func (m *PageManager) GetHTML(selector string) (string, error) {
    m.logger.Debug("get HTML", logger.String("selector", selector))
    start := time.Now()
    html, err := m.page.GetHTML(selector)
    m.recordSelector(selector, start, err)
    return html, err
}

// GetAttribute get element attribute value
func (m *PageManager) GetAttribute(selector, attrName string) (string, error) {
    m.logger.Debug("get attribute",
        logger.String("selector", selector),
        logger.String("attribute", attrName))
    start := time.Now()
    value, err := m.page.GetAttribute(selector, attrName)
    m.recordSelector(selector, start, err)
    return value, err
}

// ScrollDown scroll page down
func (m *PageManager) ScrollDown(pixels int) error {
    m.logger.Debug("scroll down", logger.Int("pixels", pixels))
    return m.page.ScrollDown(pixels)
}

// ScrollUp scroll page up
func (m *PageManager) ScrollUp(pixels int) error {
    m.logger.Debug("scroll up", logger.Int("pixels", pixels))
    return m.page.ScrollUp(pixels)
}

// ScrollToElement scroll to element
func (m *PageManager) ScrollToElement(selector string) error {
    start := time.Now()
    m.logger.Debug("scroll to element", logger.String("selector", selector))
    err := m.page.ScrollToElement(selector)
    m.recordSelector(selector, start, err)
    return err
}
