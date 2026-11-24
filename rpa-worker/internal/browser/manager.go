package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// Phase 154 / ENGINE-01：浏览器承载模型（**契约已冻结，Phase 155 只依赖本文件**）
//
// 旧模型（已删除）：Pool 里每个 page 都 NewChromePage → MustLaunch() 起一个
// 独立 Chrome 进程，Close() 连带 browser.Close()。max_pages 实际语义是
// 「Chrome 进程数上限」，内存随并发线性爆炸（D17）。
//
// 新模型（承载形态 A：一 VM 一容器一 Chromium 多 page）：
//
//	Manager
//	  ├── launcher（1 个 Chrome 进程，懒启动一次）
//	  ├── browser  *rod.Browser（全 worker 共享）
//	  └── idle pages（N 个 page = N 个标签页，同一进程内共享）
//
// 155 可依赖的接口清单（精确签名 + 语义）：
//
//	func NewManager(cfg *config.BrowserConfig, log logger.Logger) (*Manager, error)
//	func (m *Manager) Acquire(ctx context.Context) (Page, error)
//	func (m *Manager) Release(p Page)
//	func (m *Manager) Close() error
//	func (m *Manager) Browser() *rod.Browser
//	func (m *Manager) PageCount() int
//	func (m *Manager) PID() int
//	func (m *Manager) SetCapacity(n int)
//	func (m *Manager) Capacity() int
//
//	Page 接口（browser/pool.go）：
//	  RawPage() *rod.Page       // 当前活动页签的 rod 句柄（screencast 出口）
//	  ContextPage() *rod.Page   // 当前操作上下文（iframe 内为 frame 句柄）
//	  Close() error             // **只关 page，绝不关 browser**
//	  Closed() bool
//	  Reset()                   // 复位 iframe/页签上下文并清理本次新开页签
//
// 硬语义：Page.Close() 只关 page。155 的 screencast 绑在 page 上，若回收时把
// Chrome 一起关掉，同容器内其它并发流会同时断掉。
// ============================================================================

// Manager 浏览器管理器：1 个 Chrome 进程 + N 个 page（N 动态可调）。
type Manager struct {
	config *config.BrowserConfig
	logger logger.Logger

	// mu 保护 browser/launcher/closed/idle/inUse/maxPages/notify。
	mu       sync.Mutex
	launcher *launcher.Launcher
	browser  *rod.Browser
	closed   bool

	idle     []Page // 空闲页（可复用）
	inUse    int    // 已借出页数
	maxPages int    // 并发页面数上限（= worker.max_concurrency）
	created  int    // 累计创建的 page 数（含空闲）
	// notify 是「容量或空闲页可能变化」的广播通道：每次 Release /
	// SetCapacity / Close 都 close 它并换新，阻塞中的 Acquire 立即醒来重试。
	notify chan struct{}

	// createMu 串行化浏览器启动与 page 创建，避免并发首启起多个 Chrome。
	createMu sync.Mutex
}

// Pool 是 Manager 的历史类型名薄封装（Phase 154 起两者等价）。
type Pool = Manager

// NewManager 创建浏览器管理器。Chrome 进程懒启动：首次 Acquire 时才拉起，
// 使 worker 初始化不阻塞、也使单测可在不启动浏览器的情况下构造。
func NewManager(cfg *config.BrowserConfig, log logger.Logger) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("browser config 不能为空")
	}
	capacity := cfg.MaxConcurrency
	if capacity <= 0 {
		capacity = cfg.MaxPages
	}
	if capacity <= 0 {
		capacity = 5
	}

	m := &Manager{
		config:   cfg,
		logger:   log,
		maxPages: capacity,
		notify:   make(chan struct{}),
	}
	log.Info("浏览器管理器创建成功（1 Chrome + N page 模型）",
		logger.Int("max_pages", capacity))
	return m, nil
}

// NewPool 兼容旧构造点（worker.go 已改用 NewManager）。
func NewPool(cfg *config.BrowserConfig, log logger.Logger) (*Pool, error) {
	return NewManager(cfg, log)
}

// broadcastLocked 唤醒所有等待者（调用方必须持有 mu）。
func (m *Manager) broadcastLocked() {
	close(m.notify)
	m.notify = make(chan struct{})
}

// Browser 返回共享的 rod 浏览器实例（155 若需 Browser 级 CDP 可直取）。
func (m *Manager) Browser() *rod.Browser {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.browser
}

// PID 返回 Chrome 主进程 PID（0 = 尚未启动）。RSS 基线与运维排查使用。
func (m *Manager) PID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launcher == nil {
		return 0
	}
	return m.launcher.PID()
}

// PageCount 当前已创建 page 数（含空闲）。
func (m *Manager) PageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.created
}

// Capacity 当前并发页面数上限。
func (m *Manager) Capacity() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxPages
}

// SetCapacity 动态调整并发页面数上限（worker 扩缩容时同步）。
// 调小不会打断已借出的 page，只是不再新建；调大立即生效。
func (m *Manager) SetCapacity(n int) {
	if n <= 0 {
		return
	}
	m.mu.Lock()
	m.maxPages = n
	m.broadcastLocked()
	m.mu.Unlock()
	m.logger.Info("浏览器并发页面数已调整", logger.Int("max_pages", n))
}

// Acquire 取一个 page。
//
// 语义：
//  1. 有空闲 page 且未关闭 → 直接复用（零冷启动）；
//  2. 否则在并发页面数上限内新建 page（复用同一个 Chrome 进程）；
//  3. 到达上限则阻塞等待别人 Release 或 SetCapacity 调大，或 ctx 取消。
func (m *Manager) Acquire(ctx context.Context) (Page, error) {
	for {
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return nil, fmt.Errorf("浏览器管理器已关闭")
		}

		// 优先复用空闲页（丢弃已关闭的死页）。
		for len(m.idle) > 0 {
			p := m.idle[len(m.idle)-1]
			m.idle = m.idle[:len(m.idle)-1]
			if p == nil || p.Closed() {
				m.created--
				continue
			}
			m.inUse++
			m.mu.Unlock()
			p.Reset()
			return p, nil
		}

		if m.inUse < m.maxPages {
			m.inUse++
			m.mu.Unlock()
			p, err := m.createPage(ctx)
			if err != nil {
				m.mu.Lock()
				m.inUse--
				m.broadcastLocked()
				m.mu.Unlock()
				return nil, err
			}
			m.mu.Lock()
			m.created++
			m.mu.Unlock()
			return p, nil
		}

		wait := m.notify
		m.mu.Unlock()

		select {
		case <-wait:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Release 归还 page。
//
// 已关闭的 page 直接丢弃（不归还复用队列）；其余复位后进空闲队列。
func (m *Manager) Release(p Page) {
	if p == nil {
		return
	}

	m.mu.Lock()
	if m.inUse > 0 {
		m.inUse--
	}
	if m.closed {
		m.mu.Unlock()
		_ = p.Close()
		return
	}
	if p.Closed() {
		if m.created > 0 {
			m.created--
		}
		m.broadcastLocked()
		m.mu.Unlock()
		return
	}
	p.Reset()
	m.idle = append(m.idle, p)
	m.broadcastLocked()
	m.mu.Unlock()
}

// Close 关闭管理器：关闭所有 page，然后关闭 Chrome 进程。
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	idle := m.idle
	m.idle = nil
	b := m.browser
	l := m.launcher
	m.browser = nil
	m.launcher = nil
	m.broadcastLocked()
	m.mu.Unlock()

	var firstErr error
	for _, p := range idle {
		if p == nil {
			continue
		}
		if err := p.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if b != nil {
		if err := b.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if l != nil {
		l.Kill()
	}

	m.logger.Info("浏览器管理器已关闭")
	return firstErr
}

// createPage 新建一个 page（复用共享 Chrome 进程）。
func (m *Manager) createPage(ctx context.Context) (Page, error) {
	m.createMu.Lock()
	defer m.createMu.Unlock()

	b, err := m.ensureBrowser(ctx)
	if err != nil {
		return nil, err
	}

	page, err := NewChromePageFromBrowser(ctx, b, m.config, m.logger)
	if err != nil {
		m.logger.Error("创建 page 失败", logger.Err(err))
		return nil, fmt.Errorf("创建 page 失败: %w", err)
	}
	m.logger.Debug("新建 page（共享同一 Chrome 进程）")
	return page, nil
}

// ensureBrowser 懒启动共享 Chrome（仅一次）。
func (m *Manager) ensureBrowser(ctx context.Context) (*rod.Browser, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, fmt.Errorf("浏览器管理器已关闭")
	}
	if m.browser != nil {
		b := m.browser
		m.mu.Unlock()
		return b, nil
	}
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 启动过程不持锁（可能耗时数百毫秒）；createMu 已经保证只启动一次。
	l := newLauncher(m.config)
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("启动 Chrome 失败: %w", err)
	}
	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		l.Kill()
		return nil, fmt.Errorf("连接 Chrome 失败: %w", err)
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = b.Close()
		l.Kill()
		return nil, fmt.Errorf("浏览器管理器已关闭")
	}
	m.launcher = l
	m.browser = b
	m.mu.Unlock()

	m.logger.Info("Chrome 进程已启动（全 worker 共享）",
		logger.String("control_url", controlURL),
		logger.Int("pid", l.PID()))
	return b, nil
}
