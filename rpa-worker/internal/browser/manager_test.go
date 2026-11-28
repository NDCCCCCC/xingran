package browser

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/config"
)

// ============================================================================
// Phase 154 / ENGINE-01 验收：1 Chrome + N page
//   TestManager_Unit_*       轻量兜底（-short 也跑，无 Chromium 依赖）
//   TestManager_SingleChrome 真 Chromium：N page 共享同一 *rod.Browser；
//                            Page.Close 只关 page 不关 Chrome
//   TestManager_RSSBaseline  真 Chromium：N=1 与 N=8 的 RSS 基线（见 rss_test.go）
// ============================================================================

// testChromePath 探测可用 Chromium（复用生产侧的 TestChromePath）。
func testChromePath() string { return TestChromePath() }

// newTestManager 构造集成验证用 Manager（无 Chromium 或 -short 时 skip）。
func newTestManager(t *testing.T, maxPages int) *Manager {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	chrome := testChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}
	cfg := &config.BrowserConfig{
		Headless:          true,
		Timeout:           15 * time.Second,
		NavigationTimeout: 20 * time.Second,
		MaxConcurrency:    maxPages,
		ViewportWidth:     1024,
		ViewportHeight:    768,
		ChromePath:        chrome,
		ChromeFlags:       []string{"--headless=new"},
	}
	m, err := NewManager(cfg, nopLogger{})
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

// TestManager_Unit_CapacityFromConfig 合并后的并发页面数驱动池容量。
func TestManager_Unit_CapacityFromConfig(t *testing.T) {
	m, err := NewManager(&config.BrowserConfig{MaxConcurrency: 8}, nopLogger{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer m.Close()
	if got, want := m.Capacity(), 8; got != want {
		t.Fatalf("容量=%d，期望 %d（应来自合并后的 max_concurrency）", got, want)
	}
	// 旧键 max_pages 单独存在时仍可驱动容量（兼容路径）
	legacy, err := NewManager(&config.BrowserConfig{MaxPages: 4}, nopLogger{})
	if err != nil {
		t.Fatalf("NewManager(legacy): %v", err)
	}
	defer legacy.Close()
	if got := legacy.Capacity(); got != 4 {
		t.Fatalf("旧键 max_pages 容量=%d，期望 4", got)
	}
}

// fakePage 仅用于「死页丢弃」逻辑（不启动 Chrome）。
type fakePage struct{ closed bool }

func (f *fakePage) Goto(string) error                                   { return nil }
func (f *fakePage) Close() error                                        { f.closed = true; return nil }
func (f *fakePage) Screenshot() ([]byte, error)                         { return nil, nil }
func (f *fakePage) Click(string) error                                  { return nil }
func (f *fakePage) Fill(string, string) error                           { return nil }
func (f *fakePage) Select(string, string) error                         { return nil }
func (f *fakePage) WaitFor(string, time.Duration) error                 { return nil }
func (f *fakePage) Evaluate(string) (interface{}, error)                { return nil, nil }
func (f *fakePage) GetText(string) (string, error)                      { return "", nil }
func (f *fakePage) GetValue(string) (string, error)                     { return "", nil }
func (f *fakePage) GetHTML(string) (string, error)                      { return "", nil }
func (f *fakePage) GetAttribute(string, string) (string, error)         { return "", nil }
func (f *fakePage) ScrollDown(int) error                                { return nil }
func (f *fakePage) ScrollUp(int) error                                  { return nil }
func (f *fakePage) ScrollToElement(string) error                        { return nil }
func (f *fakePage) RawPage() *rod.Page                                  { return nil }
func (f *fakePage) ContextPage() *rod.Page                              { return nil }
func (f *fakePage) Browser() *rod.Browser                               { return nil }
func (f *fakePage) Closed() bool                                        { return f.closed }
func (f *fakePage) Reset()                                              {}
func (f *fakePage) Upload(string, []string) error                       { return nil }
func (f *fakePage) Hover(string) error                                  { return nil }
func (f *fakePage) ClickWith(string, proto.InputMouseButton, int) error { return nil }
func (f *fakePage) Drag(string, string) error                           { return nil }
func (f *fakePage) PressKeys(string) error                              { return nil }
func (f *fakePage) SwitchTab(int) error                                 { return nil }
func (f *fakePage) EnterFrame(string) error                             { return nil }
func (f *fakePage) ExitFrame() error                                    { return nil }
func (f *fakePage) ScrollInElement(string, int, int) error              { return nil }
func (f *fakePage) Download(context.Context, string, func() error) (string, error) {
	return "", nil
}
func (f *fakePage) CloseCurrentTab() error { return nil }

// TestManager_Unit_AcquireDiscardsDeadPage 死页不得交给调用方。
func TestManager_Unit_AcquireDiscardsDeadPage(t *testing.T) {
	m, err := NewManager(&config.BrowserConfig{MaxConcurrency: 1}, nopLogger{})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	dead := &fakePage{closed: true}
	m.mu.Lock()
	m.idle = append(m.idle, dead)
	m.created = 1
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	got, err := m.Acquire(ctx)

	// 核心断言：死页绝不能被交给调用方。若本机有 Chromium 则新建页成功，
	// 若没有则返回 error；两种结果都合法，非法的是「返回了那条死页」。
	if err == nil {
		if got == Page(dead) {
			t.Fatal("死页被原样交给调用方（必须丢弃后新建）")
		}
		m.Release(got)
		return
	}

	m.mu.Lock()
	inUse, created := m.inUse, m.created
	m.mu.Unlock()
	if inUse != 0 {
		t.Fatalf("Acquire 失败后 inUse=%d，期望 0（必须归还借出计数）", inUse)
	}
	if created != 0 {
		t.Fatalf("死页应被丢弃，created=%d", created)
	}
}

// TestManager_SingleChrome N 个 page 共享同一 rod.Browser（1 Chrome 模型），
// 且 Page.Close 只关 page —— 关掉一个 page 后另一个仍可正常求值。
func TestManager_SingleChrome(t *testing.T) {
	m := newTestManager(t, 4)
	ctx := context.Background()

	p1, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire #1: %v", err)
	}
	p2, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire #2: %v", err)
	}

	if p1.Browser() == nil || p2.Browser() == nil {
		t.Fatal("Page.Browser() 不得为 nil（155 依赖该出口）")
	}
	if p1.Browser() != p2.Browser() {
		t.Fatal("两个 page 的 *rod.Browser 必须相同（证明 1 Chrome + N page）")
	}
	if p1.RawPage() == p2.RawPage() {
		t.Fatal("两个 page 的 *rod.Page 必须不同")
	}
	if got := m.PageCount(); got != 2 {
		t.Fatalf("PageCount=%d，期望 2", got)
	}

	// 「Close 只关 page」硬语义：关掉 p1 后 Chrome 仍存活，p2 继续可用。
	if err := p1.Close(); err != nil {
		t.Fatalf("关闭 page #1 失败: %v", err)
	}
	if !p1.Closed() {
		t.Fatal("Close 后 Closed() 必须为 true")
	}
	res, err := p2.Evaluate("1+1")
	if err != nil {
		t.Fatalf("关闭 page #1 后 page #2 不可用（Chrome 被误关）: %v", err)
	}
	t.Logf("关闭一个 page 后另一个 page 仍可求值: 1+1=%v", res)

	m.Release(p1)
	p3, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("Release 后重新 Acquire 失败: %v", err)
	}
	if p3.Browser() != p2.Browser() {
		t.Fatal("重新 Acquire 的 page 必须仍属于同一 Browser")
	}
	m.Release(p2)
	m.Release(p3)
}

// TestManager_CloseShutsBrowser Manager.Close 才关闭 Chrome 进程。
func TestManager_CloseShutsBrowser(t *testing.T) {
	m := newTestManager(t, 1)
	p, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	m.Release(p)
	if m.PID() == 0 {
		t.Skip("拿不到 Chrome PID，跳过进程关闭断言")
	}
	if err := m.Close(); err != nil {
		t.Fatalf("Manager.Close: %v", err)
	}
	if m.Browser() != nil {
		t.Fatal("Close 后 Browser() 必须为 nil")
	}
}
