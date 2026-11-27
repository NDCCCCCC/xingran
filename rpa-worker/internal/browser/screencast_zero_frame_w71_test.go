package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// W7.1 / 缺陷 C：空页面 0 帧显式告知 —— worker 侧真 Chromium 验收
//
// 缺陷实证（生产）：页面停在 about:blank 时 Chrome 不产 screencast 帧，
// 前端只能无限显示「等待画面…」。Worker 侧必须在开流后 N 秒（默认 5s）
// 仍 0 帧时留下**可诊断的现场**（页面地址 + DOM 是否为空），而不是静默。
//
//   TestScreencastZeroFrame_BlankPage_Detected   about:blank 开流 → 5s 内 0 帧
//                                                → ZeroFrame 报告被设置 + WARN 日志
//   TestScreencastZeroFrame_ContentPage_NoHint   反向用例：有内容页面正常出帧
//                                                → 不判定、不误报
//
// 真 Chromium、真 CDP；无 Chromium 或 -short 时 skip（与同包其它集成用例一致）。
// ============================================================================

// warnCapture 捕获 WARN 日志文本（判定「是否留下了现场」）。
type warnCapture struct {
	mu    sync.Mutex
	warns []string
}

func (c *warnCapture) Debug(string, ...logger.Field) {}
func (c *warnCapture) Info(string, ...logger.Field)  {}
func (c *warnCapture) Warn(msg string, fields ...logger.Field) {
	c.mu.Lock()
	defer c.mu.Unlock()
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", f.Key, f.Value))
	}
	c.warns = append(c.warns, msg+" "+strings.Join(parts, " "))
}
func (c *warnCapture) Error(string, ...logger.Field)      {}
func (c *warnCapture) With(...logger.Field) logger.Logger { return c }
func (c *warnCapture) Sync() error                        { return nil }
func (c *warnCapture) all() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.warns, "\n")
}

// newBlankPage 起一个真 Chromium + page，并导航到 about:blank。
func newBlankPage(t *testing.T) Page {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	chrome := TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}
	cfg := newTestBrowserConfig(chrome)
	m, err := NewManager(cfg, nopLogger{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	t.Cleanup(func() { m.Release(p) })

	// 显式回到 about:blank（生产事故的现场形态）。
	if err := p.Goto("about:blank"); err != nil {
		t.Fatalf("Goto about:blank: %v", err)
	}
	return p
}

// TestScreencastZeroFrame_BlankPage_Detected 主断言（缺陷 C / worker 侧）。
func TestScreencastZeroFrame_BlankPage_Detected(t *testing.T) {
	p := newBlankPage(t)
	raw := p.RawPage()
	if raw == nil {
		t.Fatal("RawPage 为 nil")
	}

	cap := &warnCapture{}
	frames, stats, stop, err := StartScreencast(raw, ScreencastOptions{
		ZeroFrameWarnAfter: 2 * time.Second, // 缩短等待，保持语义（默认 5s）
		Logger:             cap,
	})
	if err != nil {
		t.Fatalf("StartScreencast: %v", err)
	}
	defer stop()

	// 1) 2.5s 内必须 0 帧（about:blank 不产帧 —— 这也是缺陷能被复现的前提）。
	frameCount := 0
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatal("frames channel 意外关闭")
			}
			f.Ack()
			frameCount++
		case <-time.After(100 * time.Millisecond):
		}
		if frameCount > 0 {
			break
		}
	}
	if frameCount > 0 {
		t.Fatalf("about:blank 竟然产生了 %d 帧 —— 该环境无法复现「空页面无帧」，本用例前提不成立", frameCount)
	}

	// 2) 判定必须被设置（ZeroFrame 报告可被后端/日志消费）。
	var rep ZeroFrameReport
	pollDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(pollDeadline) {
		if rep = stats.ZeroFrame(); rep.Detected {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !rep.Detected {
		t.Fatalf("开流后 2s 零帧未判定为「暂无画面」（日志=%s）", cap.all())
	}
	if rep.FramesSeen != 0 {
		t.Fatalf("判定时的帧数应为 0，实际 %d", rep.FramesSeen)
	}
	if !strings.Contains(strings.ToLower(rep.PageURL), "about:blank") {
		t.Fatalf("判定现场必须带页面地址（about:blank），实际 %q", rep.PageURL)
	}
	if !rep.DOMEmpty {
		t.Fatalf("about:blank 应被判为 DOM 为空，实际 domEmpty=%v", rep.DOMEmpty)
	}

	// 3) WARN 日志必须真的打出来（运维在日志里能看见，而不是只存内存字段）。
	if !strings.Contains(cap.all(), "无画面帧") {
		t.Fatalf("缺少「无画面帧」WARN 日志，运维无法在日志里识别:\n%s", cap.all())
	}
	if !strings.Contains(cap.all(), "page_url") || !strings.Contains(cap.all(), "dom_empty") {
		t.Fatalf("WARN 日志必须带 page_url 与 dom_empty 现场:\n%s", cap.all())
	}
	t.Logf("零帧判定现场: url=%s domEmpty=%v framesSeen=%d waitMs=%d",
		rep.PageURL, rep.DOMEmpty, rep.FramesSeen, rep.WaitMS)
	t.Logf("WARN 原文:\n%s", cap.all())
}

// TestScreencastZeroFrame_ContentPage_NoHint 反向用例（不能误报）。
//
// 有内容且持续动画的页面 → 正常出帧 → ZeroFrame 必须保持未判定。
func TestScreencastZeroFrame_ContentPage_NoHint(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	chrome := TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}

	// 与同包 fixture 同款：带持续动画，保证 Chrome 有变化可发帧。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8"><title>content</title>
<style>body{margin:0;font-family:sans-serif}#box{width:320px;height:160px;background:#4a9}</style></head>
<body><div id="box">有内容</div>
<script>let n=0;setInterval(()=>{document.body.style.background=(n++%2)?'#fff':'#eee';},100);</script>
</body></html>`)
	}))
	t.Cleanup(srv.Close)

	cfg := newTestBrowserConfig(chrome)
	m, err := NewManager(cfg, nopLogger{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := m.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	t.Cleanup(func() { m.Release(p) })
	if err := p.Goto(srv.URL); err != nil {
		t.Fatalf("Goto: %v", err)
	}

	cap := &warnCapture{}
	frames, stats, stop, err := StartScreencast(p.RawPage(), ScreencastOptions{
		ZeroFrameWarnAfter: 2 * time.Second,
		Logger:             cap,
	})
	if err != nil {
		t.Fatalf("StartScreencast: %v", err)
	}
	defer stop()

	// 收几帧证明「有画面」（不 ACK 会停流，所以边收边 ACK）。
	got := 0
	timeout := time.After(8 * time.Second)
	for got < 3 {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("拿到 %d 帧后 channel 关闭", got)
			}
			f.Ack()
			got++
		case <-timeout:
			t.Fatalf("8s 内只拿到 %d 帧（有内容页面应正常出帧）", got)
		}
	}

	// 等过判定阈值后再确认没有误报。
	time.Sleep(1200 * time.Millisecond)
	if rep := stats.ZeroFrame(); rep.Detected {
		t.Fatalf("有画面帧的页面被误判为「暂无画面」: %+v", rep)
	}
	if strings.Contains(cap.all(), "无画面帧") {
		t.Fatalf("有画面帧的页面被误报 WARN:\n%s", cap.all())
	}
	t.Logf("有内容页面正常出帧 %d 帧，未触发空页面判定", got)
}

// newTestBrowserConfig 统一的真 Chromium 配置（与同包用例的取值一致）。
func newTestBrowserConfig(chrome string) *config.BrowserConfig {
	return &config.BrowserConfig{
		Headless:          true,
		Timeout:           15 * time.Second,
		NavigationTimeout: 20 * time.Second,
		MaxConcurrency:    2,
		ViewportWidth:     1024,
		ViewportHeight:    768,
		ChromePath:        chrome,
		ChromeFlags:       []string{"--headless=new"},
	}
}
