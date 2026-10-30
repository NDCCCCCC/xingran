package browser

import (
	"context"
	"fmt"
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

// WaitForURL 等待 URL 变化
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待 URL 超时")
		case <-ticker.C:
			// TODO: 检查当前 URL 是否匹配
			return nil
		}
	}
}

// WaitForText 等待文本出现
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待文本超时")
		case <-ticker.C:
			// TODO: 检查文本是否存在
			return nil
		}
	}
}

// maskValue 隐藏敏感值
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
