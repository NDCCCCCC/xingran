package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// Page 浏览器页面接口
type Page interface {
	Goto(url string) error
	Close() error
	Screenshot() ([]byte, error)
	Click(selector string) error
	Fill(selector, value string) error
	Select(selector, value string) error
	WaitFor(selector string, timeout time.Duration) error
	Evaluate(script string) (interface{}, error)
}

// Pool 浏览器池
type Pool struct {
	config    *config.BrowserConfig
	pages     chan *PooledPage
	logger    logger.Logger
	mu        sync.RWMutex
	created   int
	closed    bool
}

// PooledPage 池化的页面
type PooledPage struct {
	page      Page
	inUse     bool
	lastUsed  time.Time
	pool      *Pool
}

// NewPool 创建浏览器池
func NewPool(cfg *config.BrowserConfig, log logger.Logger) (*Pool, error) {
	p := &Pool{
		config: cfg,
		pages:  make(chan *PooledPage, cfg.MaxPages),
		logger: log,
	}

	log.Info("浏览器池创建成功", logger.Int("max_pages", cfg.MaxPages))
	return p, nil
}

// Acquire 获取页面
func (p *Pool) Acquire(ctx context.Context) (*PooledPage, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("浏览器池已关闭")
	}
	p.mu.RUnlock()

	select {
	case page := <-p.pages:
		p.logger.Debug("复用现有页面")
		page.inUse = true
		return page, nil
	default:
		// 没有空闲页面，创建新的
		p.mu.Lock()
		if p.created >= p.config.MaxPages {
			p.mu.Unlock()
			// 等待空闲页面
			select {
			case page := <-p.pages:
				page.inUse = true
				return page, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		page, err := p.createPage()
		if err != nil {
			p.mu.Unlock()
			return nil, err
		}
		p.created++
		p.mu.Unlock()
		
		pooledPage := &PooledPage{
			page:     page,
			inUse:    true,
			lastUsed: time.Now(),
			pool:     p,
		}
		
		p.logger.Debug("创建新页面", logger.Int("total", p.created))
		return pooledPage, nil
	}
}

// Release 释放页面
func (p *Pool) Release(page *PooledPage) {
	if !page.inUse {
		return
	}
	
	page.inUse = false
	page.lastUsed = time.Now()
	
	select {
	case p.pages <- page:
		// 成功放回池中
	default:
		// 池已满，关闭页面
		p.logger.Debug("池已满，关闭页面")
		page.page.Close()
		p.mu.Lock()
		p.created--
		p.mu.Unlock()
	}
}

// Close 关闭浏览器池
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	close(p.pages)
	
	for page := range p.pages {
		if page.page != nil {
			page.page.Close()
		}
	}
	
	p.logger.Info("浏览器池已关闭")
	return nil
}

// createPage 创建新页面（需要实际实现）
func (p *Pool) createPage() (Page, error) {
	// TODO: 实现实际的浏览器页面创建
	// 这里返回模拟实现
	return &mockPage{}, nil
}

// mockPage 模拟页面（用于开发测试）
type mockPage struct{}

func (m *mockPage) Goto(url string) error {
	// 模拟导航
	return nil
}

func (m *mockPage) Close() error {
	return nil
}

func (m *mockPage) Screenshot() ([]byte, error) {
	return []byte("fake screenshot"), nil
}

func (m *mockPage) Click(selector string) error {
	return nil
}

func (m *mockPage) Fill(selector, value string) error {
	return nil
}

func (m *mockPage) Select(selector, value string) error {
	return nil
}

func (m *mockPage) WaitFor(selector string, timeout time.Duration) error {
	return nil
}

func (m *mockPage) Evaluate(script string) (interface{}, error) {
	return nil, nil
}
