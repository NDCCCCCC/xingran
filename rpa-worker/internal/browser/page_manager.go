package browser

import (
    "context"
    "fmt"
    "time"

    "github.com/xingran-next/rpa-worker/internal/logger"
)

// PageManager page manager
type PageManager struct {
    page   Page
    logger logger.Logger
}

// NewPageManager create page manager
func NewPageManager(page Page, log logger.Logger) *PageManager {
    return &PageManager{
        page:   page,
        logger: log,
    }
}

// Goto navigate to URL
func (m *PageManager) Goto(url string) error {
    m.logger.Debug("navigate to page", logger.String("url", url))
    return m.page.Goto(url)
}

// Click click element
func (m *PageManager) Click(selector string) error {
    m.logger.Debug("click element", logger.String("selector", selector))
    return m.page.Click(selector)
}

// Fill fill form
func (m *PageManager) Fill(selector, value string) error {
    m.logger.Debug("fill form",
        logger.String("selector", selector),
        logger.String("value", maskValue(value)))
    return m.page.Fill(selector, value)
}

// Select select dropdown option
func (m *PageManager) Select(selector, value string) error {
    m.logger.Debug("select option",
        logger.String("selector", selector),
        logger.String("value", value))
    return m.page.Select(selector, value)
}

// WaitFor wait for element
func (m *PageManager) WaitFor(selector string, timeout time.Duration) error {
    m.logger.Debug("wait for element",
        logger.String("selector", selector),
        logger.Duration("timeout", timeout))
    return m.page.WaitFor(selector, timeout)
}

// Screenshot take screenshot
func (m *PageManager) Screenshot() ([]byte, error) {
    return m.page.Screenshot()
}

// Evaluate execute JavaScript
func (m *PageManager) Evaluate(script string) (interface{}, error) {
    return m.page.Evaluate(script)
}

// WaitForURL wait for URL change
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("wait for URL timeout")
        case <-ticker.C:
            // TODO: check if current URL matches
            return nil
        }
    }
}

// WaitForText wait for text appearance
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("wait for text timeout")
        case <-ticker.C:
            // TODO: check if text exists
            return nil
        }
    }
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
    return m.page.GetText(selector)
}

// GetValue get input element value
func (m *PageManager) GetValue(selector string) (string, error) {
    m.logger.Debug("get value", logger.String("selector", selector))
    return m.page.GetValue(selector)
}

// GetHTML get element HTML content
func (m *PageManager) GetHTML(selector string) (string, error) {
    m.logger.Debug("get HTML", logger.String("selector", selector))
    return m.page.GetHTML(selector)
}

// GetAttribute get element attribute value
func (m *PageManager) GetAttribute(selector, attrName string) (string, error) {
    m.logger.Debug("get attribute",
        logger.String("selector", selector),
        logger.String("attribute", attrName))
    return m.page.GetAttribute(selector, attrName)
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
    m.logger.Debug("scroll to element", logger.String("selector", selector))
    return m.page.ScrollToElement(selector)
}
