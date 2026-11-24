package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/xingran-next/rpa-worker/internal/config"
	lxtypes "github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01/02/05 验收（真 Chromium）
//
//   TestScreencast01_Unit_JPEGSize            轻量兜底（-short 也跑，无 Chromium）
//   TestScreencast01_FrameRateAndBandwidth    真跑：实测 fps / 单路带宽 / 端到端延迟
//   TestScreencast01_NoAckStopsFrames         真跑：不 ACK → 帧停（ACK 即背压的证伪）
//   TestLiveView02_InputInjection             真跑：注入点击+键盘 → 页面 DOM 真实变化
//   TestLiveView05_NoLeakAfterRepeatedWatch   真跑：100 次 start/stop 后 goroutine 与 page 数不增长
//   TestLiveView05_StopDoesNotClosePage       真跑：停流后 page 与 browser 仍可用
// ============================================================================

// liveFixture 起一个 httptest 页面（用于 screencast 与输入注入验收）。
func liveFixture(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	page := `<!doctype html><html><head><meta charset="utf-8"><title>live</title>
<style>
 body{margin:0;font-family:sans-serif;background:#fafafa}
 #box{position:absolute;left:20px;top:40px;width:300px;height:120px;background:linear-gradient(45deg,#ff9a9e,#fad0c4);}
 #field{margin-top:200px;margin-left:20px;width:320px;font-size:16px;padding:6px}
 #log{margin-left:20px;color:#333;font-size:14px}
</style></head><body>
<div id="box"></div>
<input id="field" placeholder="type here">
<div id="log">idle</div>
<div style="height:2000px"></div>
<script>
 const log = document.getElementById('log');
 const field = document.getElementById('field');
 window.__clicks = 0;
 document.getElementById('box').addEventListener('click', () => {
   window.__clicks++;
   log.textContent = 'clicked:' + window.__clicks;
 });
 field.addEventListener('input', () => { log.textContent = 'value:' + field.value; });
 // 持续动画让 screencast 有帧可发（静态页面 Chrome 只在变化时发帧）
 let n = 0;
 setInterval(() => { document.body.style.background = (n++ % 2) ? '#fafafa' : '#f5f5f5'; }, 120);
</script></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	t.Cleanup(srv.Close)
	return srv, srv.URL
}

// newLivePage 起一个真 Chromium + 一个 page，跳转到 fixture。
func newLivePage(t *testing.T, cfgMut func(*config.BrowserConfig)) (*Manager, Page, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	chrome := TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}
	srv, url := liveFixture(t)

	cfg := &config.BrowserConfig{
		Headless:          true,
		Timeout:           15 * time.Second,
		NavigationTimeout: 20 * time.Second,
		MaxConcurrency:    2,
		ViewportWidth:     1024,
		ViewportHeight:    768,
		ChromePath:        chrome,
		ChromeFlags:       []string{"--headless=new"},
	}
	if cfgMut != nil {
		cfgMut(cfg)
	}
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

	if err := p.Goto(url); err != nil {
		t.Fatalf("Goto: %v", err)
	}
	_ = srv
	return m, p, url
}

// TestScreencast01_Unit_JPEGSize JPEG SOF 解析（无浏览器，-short 也跑）。
func TestScreencast01_Unit_JPEGSize(t *testing.T) {
	// 最小合法 JPEG 头：SOI + SOF0 段（length=17, precision=8, h=720, w=1280）
	// 偏移：0=FF 1=D8(SOI) 2=FF 3=C0(SOF0) 4-5=段长(0x0011) 6=精度 7-8=高 9-10=宽
	// 段长 17 要求 buf 至少 i+2+17 = 21 字节，尾部补 4 个填充字节。
	buf := []byte{
		0xFF, 0xD8, // SOI
		0xFF, 0xC0, // SOF0
		0x00, 0x11, // 段长 17
		0x08,       // 精度
		0x02, 0xD0, // 高 720
		0x05, 0x00, // 宽 1280
		0x00, 0x00, 0x00, // 分量数与填充
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 填充到 24 字节
	}
	w, h := GuessImageSize(buf)
	if w != 1280 || h != 720 {
		t.Fatalf("JPEG 尺寸解析 = %dx%d，期望 1280x720", w, h)
	}
	if w2, h2 := GuessImageSize([]byte("not-a-jpeg")); w2 != 0 || h2 != 0 {
		t.Fatalf("非法 JPEG 应返回 (0,0)，实际 (%d,%d)", w2, h2)
	}
}

// TestScreencast01_OptionsNormalize 默认值归一（无浏览器）。
func TestScreencast01_OptionsNormalize(t *testing.T) {
	opts := ScreencastOptions{}.normalize()
	if opts.Quality != DefaultScreencastQuality {
		t.Fatalf("Quality=%d，期望 %d", opts.Quality, DefaultScreencastQuality)
	}
	if opts.MaxWidth != DefaultScreencastMaxWidth || opts.MaxHeight != DefaultScreencastMaxHeight {
		t.Fatalf("默认尺寸=%dx%d，期望 %dx%d", opts.MaxWidth, opts.MaxHeight,
			DefaultScreencastMaxWidth, DefaultScreencastMaxHeight)
	}
	if opts.EveryNthFrame != DefaultScreencastEveryNthFrame {
		t.Fatalf("EveryNthFrame=%d，期望 %d", opts.EveryNthFrame, DefaultScreencastEveryNthFrame)
	}
	if opts.AckTimeout != DefaultScreencastAckTimeout {
		t.Fatalf("AckTimeout=%v，期望 %v", opts.AckTimeout, DefaultScreencastAckTimeout)
	}
	// 越界值回落默认（Quality=200 非法）
	bad := ScreencastOptions{Quality: 200, MaxWidth: -1}.normalize()
	if bad.Quality != DefaultScreencastQuality || bad.MaxWidth != DefaultScreencastMaxWidth {
		t.Fatalf("越界值未回落默认: quality=%d width=%d", bad.Quality, bad.MaxWidth)
	}
}

// TestScreencast01_TuneMatrix 降档参数扫描（LIVEVIEW-01「可降档」要求的实证）。
//
// 目的：找出 headless 下能拉高 fps 的参数组合，以及各档位的带宽代价。
// 结论不预设阈值（调研 §1 已明示 headless 普遍 1–5fps），全部数字写入
// out/measure/liveview01-tune.json 供 SUMMARY 引用，禁止美化。
func TestScreencast01_TuneMatrix(t *testing.T) {
	_, p, _ := newLivePage(t, nil)

	cases := []struct {
		name    string
		quality int
		maxW    int
		maxH    int
		nth     int
	}{
		{"default-q50-1280-nth2", 50, 1280, 720, 2},
		{"q50-1280-nth1", 50, 1280, 720, 1},
		{"q40-854-nth1", 40, 854, 480, 1},
		{"q30-640-nth1", 30, 640, 360, 1},
		{"q60-1280-nth1", 60, 1280, 720, 1},
	}

	type row struct {
		Name      string  `json:"name"`
		Quality   int     `json:"quality"`
		MaxWidth  int     `json:"maxWidth"`
		MaxHeight int     `json:"maxHeight"`
		NthFrame  int     `json:"everyNthFrame"`
		Frames    int64   `json:"frames"`
		Seconds   float64 `json:"seconds"`
		FPS       float64 `json:"fps"`
		Mbps      float64 `json:"mbps"`
		AvgKB     float64 `json:"avgFrameKB"`
		JPEGW     int64   `json:"jpegWidth"`
		JPEGH     int64   `json:"jpegHeight"`
	}
	var rows []row

	for _, c := range cases {
		frames, _, stop, err := StartScreencast(p.RawPage(), ScreencastOptions{
			Quality:         c.quality,
			MaxWidth:        c.maxW,
			MaxHeight:       c.maxH,
			EveryNthFrame:   c.nth,
			AckTimeout:      2 * time.Second,
			ChannelCapacity: 4,
			Logger:          nopLogger{},
		})
		if err != nil {
			t.Fatalf("%s StartScreencast: %v", c.name, err)
		}

		const window = 4 * time.Second
		deadline := time.After(window)
		var got, bytes, wSum, hSum int64
		var first, last time.Time
	scan:
		for {
			select {
			case f, ok := <-frames:
				if !ok {
					break scan
				}
				if first.IsZero() {
					first = time.Now()
				}
				last = time.Now()
				got++
				bytes += int64(len(f.Data))
				if f.Width > 0 {
					wSum += int64(f.Width)
					hSum += int64(f.Height)
				}
				f.Ack()
			case <-deadline:
				break scan
			}
		}
		stop()

		elapsed := last.Sub(first).Seconds()
		if elapsed <= 0 {
			elapsed = 4.0
		}
		r := row{
			Name: c.name, Quality: c.quality, MaxWidth: c.maxW, MaxHeight: c.maxH, NthFrame: c.nth,
			Frames: got, Seconds: elapsed,
			FPS:   float64(got) / elapsed,
			Mbps:  float64(bytes) * 8 / elapsed / 1e6,
		}
		if got > 0 {
			r.AvgKB = float64(bytes) / float64(got) / 1024
			r.JPEGW = wSum / got
			r.JPEGH = hSum / got
		}
		rows = append(rows, r)
		t.Logf("%-22s fps=%.2f mbps=%.3f avgKB=%.1f jpeg=%dx%d frames=%d",
			r.Name, r.FPS, r.Mbps, r.AvgKB, r.JPEGW, r.JPEGH, r.Frames)
	}

	payload, err := json.MarshalIndent(map[string]interface{}{
		"req":       "LIVEVIEW-01",
		"purpose":   "降档参数扫描（headless Chromium，本机实测）",
		"windowSec": 4,
		"rows":      rows,
	}, "", "  ")
	if err != nil {
		t.Fatalf("序列化实测矩阵失败: %v", err)
	}
	if err := writeMeasure("liveview01-tune.json", string(payload)); err != nil {
		t.Logf("写实测矩阵失败（不影响用例）: %v", err)
	}
}

// TestScreencast01_FrameRateAndBandwidth 真跑实测：fps / 单路带宽 / 帧尺寸。
//
// 参数：quality=50、maxWidth=1280×720、everyNthFrame=2（LIVEVIEW-01 的降档甜点）。
// 消费方式：逐帧取 → 立即 Ack（诚实反映「消费者不阻塞」时的上限 fps）。
func TestScreencast01_FrameRateAndBandwidth(t *testing.T) {
	_, p, _ := newLivePage(t, nil)

	opts := ScreencastOptions{
		Quality:          DefaultScreencastQuality,
		MaxWidth:         DefaultScreencastMaxWidth,
		MaxHeight:        DefaultScreencastMaxHeight,
		EveryNthFrame:    DefaultScreencastEveryNthFrame,
		AckTimeout:       2 * time.Second,
		ChannelCapacity:  4,
		Logger:           nopLogger{},
	}
	frames, stats, stop, err := StartScreencast(p.RawPage(), opts)
	if err != nil {
		t.Fatalf("StartScreencast: %v", err)
	}
	defer stop()

	const window = 5 * time.Second
	deadline := time.After(window)
	var got int64
	var bytes int64
	var firstAt time.Time
	var lastAt time.Time
	var wSum, hSum int64
	var jpegMiss int64
	var latencySamples []time.Duration

loop:
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				break loop
			}
			if firstAt.IsZero() {
				firstAt = time.Now()
			}
			lastAt = time.Now()
			got++
			bytes += int64(len(f.Data))
			if f.Width > 0 && f.Height > 0 {
				wSum += int64(f.Width)
				hSum += int64(f.Height)
			} else {
				jpegMiss++
			}
			// 端到端延迟下限：Chrome 侧帧时间戳（metadata.timestamp）→ 本机收到。
			// 这是「截图编码 + CDP 传输 + rod 解码」的可测部分，不含前端渲染与
			// WS/HTTP 网络段，故 SUMMARY 里必须写清这是**下限**。
			if f.Metadata != nil && f.Metadata.Timestamp > 0 {
				capturedOnChrome := time.UnixMilli(int64(f.Metadata.Timestamp * 1000))
				if d := time.Since(capturedOnChrome); d > 0 && d < 5*time.Second {
					latencySamples = append(latencySamples, d)
				}
			}
			f.Ack() // 背压：消费完才 ACK
		case <-deadline:
			break loop
		}
	}

	elapsed := lastAt.Sub(firstAt)
	if got == 0 {
		t.Fatalf("5s 内未收到任何帧（headless screencast 可能被 Chrome 抑制）")
	}
	if elapsed <= 0 {
		elapsed = window
	}
	fps := float64(got) / elapsed.Seconds()
	mbps := float64(bytes) * 8 / elapsed.Seconds() / 1e6
	avgBytes := float64(bytes) / float64(got)
	snap := stats.Snapshot()

	var avgW, avgH int64
	if got-jpegMiss > 0 {
		avgW = wSum / (got - jpegMiss)
		avgH = hSum / (got - jpegMiss)
	}
	var maxLatency time.Duration
	var sumLatency time.Duration
	for _, d := range latencySamples {
		sumLatency += d
		if d > maxLatency {
			maxLatency = d
		}
	}
	avgLatency := time.Duration(0)
	if len(latencySamples) > 0 {
		avgLatency = sumLatency / time.Duration(len(latencySamples))
	}

	t.Logf("LIVEVIEW-01 实测（quality=%d maxW=%d maxH=%d nth=%d）：",
		opts.Quality, opts.MaxWidth, opts.MaxHeight, opts.EveryNthFrame)
	t.Logf("  帧数=%d 窗口=%v → 实测 %.2f fps", got, elapsed.Round(time.Millisecond), fps)
	t.Logf("  单路带宽=%.3f Mbps，平均帧 %.1f KB，JPEG 尺寸 %dx%d",
		mbps, avgBytes/1024, avgW, avgH)
	t.Logf("  帧内处理延迟 avg=%v max=%v（不含网络与前端渲染）",
		avgLatency.Round(time.Microsecond), maxLatency.Round(time.Microsecond))
	t.Logf("  stats: frames=%d acks=%d dropped=%d bytes=%d → fps=%.2f mbps=%.3f",
		snap.Frames, snap.Acks, snap.Dropped, snap.Bytes, snap.FPS, snap.Mbps)

	// 硬断言只覆盖「机制正确」，性能指标如实记录不设阈值（调研已说明 headless 常 1-5fps）。
	if snap.Acks < snap.Frames-int64(snap.Dropped) {
		t.Fatalf("ACK 数 %d 少于应收到的帧（frames=%d dropped=%d）", snap.Acks, snap.Frames, snap.Dropped)
	}
	if avgW > int64(opts.MaxWidth) || avgH > int64(opts.MaxHeight) {
		t.Fatalf("帧尺寸 %dx%d 超过 maxWidth/maxHeight %dx%d", avgW, avgH, opts.MaxWidth, opts.MaxHeight)
	}

	// 把实测值写进测试日志文件，供 SUMMARY 引用（out/ 为 gitignored 构建产物目录）。
	record := fmt.Sprintf(`{
  "req": "LIVEVIEW-01",
  "opts": {"quality": %d, "maxWidth": %d, "maxHeight": %d, "everyNthFrame": %d},
  "frames": %d,
  "windowSeconds": %.2f,
  "fps": %.3f,
  "mbps": %.3f,
  "avgFrameBytes": %.1f,
  "jpegWidth": %d,
  "jpegHeight": %d,
  "avgLatencyMs": %.3f,
  "maxLatencyMs": %.3f,
  "dropped": %d
}`,
		opts.Quality, opts.MaxWidth, opts.MaxHeight, opts.EveryNthFrame,
		got, elapsed.Seconds(), fps, mbps, avgBytes, avgW, avgH,
		float64(avgLatency.Microseconds())/1000, float64(maxLatency.Microseconds())/1000,
		snap.Dropped)
	if err := writeMeasure("liveview01-measure.json", record); err != nil {
		t.Logf("写实测文件失败（不影响用例）: %v", err)
	}
}

// TestScreencast01_AckTimeoutFallback 消费确认超时 → 计 drop 并强制 ACK（不永久锁死流）。
//
// **诚实结论（重要）**：调研 §1 说「不 ACK 就停帧」，那是 CDP 的原始行为；但本实现
// 额外加了 AckTimeout 兜底（默认 800ms，本用例放大到 2s）——因为一个卡死的消费者
// 若能把 rod 的整个事件泵（进而把 WaitLoad / 截图）永久挂住，代价比丢帧大得多。
// 故本用例验证的是「兜底语义」而非「帧完全停止」：
//   - 取出帧但长时间不 ACK → stats.Dropped 必须增长（背压信号被记录）；
//   - 流不会永久停摆 → 后续仍能取到帧（兜底生效，而不是死锁）。
//
// 真正的「ACK 即背压」体现为：消费慢 → ACK 慢 → Chrome 少发帧（fps 下降），
// 那个效果由 TestScreencast01_TuneMatrix 的 nth=1 vs nth=2 对比体现，不靠本用例。
func TestScreencast01_AckTimeoutFallback(t *testing.T) {
	_, p, _ := newLivePage(t, nil)

	opts := ScreencastOptions{
		Quality:         60,
		MaxWidth:        854,
		MaxHeight:       480,
		EveryNthFrame:   1,
		AckTimeout:      2 * time.Second, // 故意放大以观察兜底
		ChannelCapacity: 1,
		Logger:          nopLogger{},
	}
	frames, stats, stop, err := StartScreencast(p.RawPage(), opts)
	if err != nil {
		t.Fatalf("StartScreencast: %v", err)
	}
	defer stop()

	// 第一帧正常消费并 ACK → 建立流。
	first := mustFrame(t, frames, 10*time.Second)
	first.Ack()

	// 第二帧取出后**不 ACK**：AckTimeout 到点会强制 ACK 并计 drop。
	second := mustFrame(t, frames, 10*time.Second)
	_ = second // 故意不 Ack

	// 等一个 AckTimeout × 2 的窗口，让超时兜底至少触发一次。
	time.Sleep(opts.AckTimeout * 2)

	// 统计里必须出现 drop（背压信号被如实记录，不是静默吞掉）。
	before := stats.Snapshot()
	if before.Dropped < 1 {
		t.Fatalf("不 ACK 时 Dropped=0（超时兜底未生效或未计数）")
	}

	// 流仍然活着（兜底而不是死锁）：后续还能取到帧。
	count := 0
	timeout := time.After(3 * time.Second)
wait:
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				break wait
			}
			count++
			f.Ack()
		case <-timeout:
			break wait
		}
	}
	t.Logf("不 ACK 第 2 帧后：Dropped=%d，随后 3s 窗口内新帧数=%d（兜底生效、流未死锁）",
		before.Dropped, count)
	if count == 0 {
		t.Fatalf("超时兜底后流未恢复（成了死锁，比丢帧更糟）")
	}

	second.Ack()
	after := stats.Snapshot()
	t.Logf("最终统计: frames=%d acks=%d dropped=%d bytes=%d",
		after.Frames, after.Acks, after.Dropped, after.Bytes)
	if after.Dropped < 1 {
		t.Fatalf("最终 Dropped=0，背压信号未累计")
	}
}

// mustFrame 在期限内取一帧（超时 Fatal）。
func mustFrame(t *testing.T, ch <-chan *ScreencastFrame, d time.Duration) *ScreencastFrame {
	t.Helper()
	select {
	case f, ok := <-ch:
		if !ok {
			t.Fatalf("帧 channel 已关闭，未取到帧")
		}
		return f
	case <-time.After(d):
		t.Fatalf("%v 内未取到帧", d)
		return nil
	}
}

// TestLiveView02_InputInjection 注入点击与键盘 → 页面 DOM 真实响应。
//
// 这是 LIVEVIEW-02 的「前端点击 → 真实页面响应」在 Worker 侧的等价验收：
// 输入走 CDP Input.dispatch*（与前端注入完全同一条代码路径 DispatchLiveInput）。
func TestLiveView02_InputInjection(t *testing.T) {
	_, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	send := func(cmd *lxtypes.LiveInputCommand) {
		t.Helper()
		if err := DispatchLiveInput(raw, cmd); err != nil {
			t.Fatalf("%s 注入失败: %v", cmd.Kind, err)
		}
	}
	sendMouse := func(kind lxtypes.LiveInputKind, x, y float64) {
		t.Helper()
		send(&lxtypes.LiveInputCommand{Kind: kind, X: x, Y: y, Button: "left", ClickCount: 1})
	}

	// 1) 鼠标移到 box 中心并按下/抬起（box: left=20,top=40,w=300,h=120 → (170,100)）。
	boxX, boxY := 170.0, 100.0
	sendMouse(lxtypes.LiveInputMouseMove, boxX, boxY)
	sendMouse(lxtypes.LiveInputMouseDown, boxX, boxY)
	sendMouse(lxtypes.LiveInputMouseUp, boxX, boxY)

	clicks := waitEvalInt(t, p, "window.__clicks", 3*time.Second)
	if clicks < 1 {
		t.Fatalf("注入点击后 window.__clicks=%d，期望 >=1（CDP Input 未生效）", clicks)
	}

	// 2) 键盘：先点中输入框，再注入字符。
	fieldX, fieldY := 100.0, 210.0
	sendMouse(lxtypes.LiveInputMouseDown, fieldX, fieldY)
	sendMouse(lxtypes.LiveInputMouseUp, fieldX, fieldY)
	for _, k := range []struct {
		key  string
		code string
		vkey int
	}{
		{"a", "KeyA", 65},
		{"b", "KeyB", 66},
		{"c", "KeyC", 67},
	} {
		send(&lxtypes.LiveInputCommand{
			Kind: lxtypes.LiveInputKeyDown, Key: k.key, Code: k.code, VirtualKey: k.vkey,
		})
		send(&lxtypes.LiveInputCommand{
			Kind: lxtypes.LiveInputKeyUp, Key: k.key, Code: k.code, VirtualKey: k.vkey,
		})
	}

	value := waitEvalString(t, p, "document.getElementById('field').value", 3*time.Second)
	if value != "abc" {
		t.Fatalf("输入框值=%q，期望 \"abc\"（键盘注入未生效）", value)
	}

	// 3) 滚轮：页面有 2000px 内容，滚动后 scrollY 应 > 0。
	send(&lxtypes.LiveInputCommand{
		Kind: lxtypes.LiveInputMouseWheel, X: 400, Y: 300, DeltaY: 400,
	})
	scrollY := waitEvalInt(t, p, "window.scrollY", 3*time.Second)
	// Chrome 的 wheel 是异步合成，可能延迟到下一帧才生效；等待窗口内仍为 0 则如实失败。
	if scrollY <= 0 {
		t.Fatalf("滚轮注入后 window.scrollY=%d，期望 >0", scrollY)
	}

	t.Logf("LIVEVIEW-02 验收：点击 → __clicks=%d；键盘 → value=%q；滚轮 → scrollY=%d",
		clicks, value, scrollY)
}

// TestLiveView02_Unit_PrintableText 键盘 text 语义（组合键不带 text，无浏览器）。
func TestLiveView02_Unit_PrintableText(t *testing.T) {
	cases := []struct {
		key  string
		mods int
		want bool
	}{
		{"a", 0, true},
		{"A", ModifierShift, true},        // Shift+大写仍需 text
		{"a", ModifierControl, false},     // Ctrl 快捷键不带 text（否则打进输入框）
		{"a", ModifierMeta, false},
		{"a", ModifierAlt, false},
		{"Enter", 0, false},               // 非单字符
		{"", 0, false},
	}
	for _, c := range cases {
		_, ok := printableText(c.key, c.mods)
		if ok != c.want {
			t.Fatalf("printableText(%q, %d) = %v，期望 %v", c.key, c.mods, ok, c.want)
		}
	}
	// 虚拟键码兜底
	if got := fallbackVirtualKey("Enter"); got != 13 {
		t.Fatalf("fallbackVirtualKey(Enter)=%d，期望 13", got)
	}
	if got := fallbackVirtualKey("ArrowDown"); got != 40 {
		t.Fatalf("fallbackVirtualKey(ArrowDown)=%d，期望 40", got)
	}
}

// waitEvalInt 轮询取整数（Chrome 的输入合成是异步的，直接读会拿到旧值）。
func waitEvalInt(t *testing.T, p Page, expr string, d time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		v, err := p.Evaluate(expr)
		if err == nil {
			if n, ok := toInt(v); ok && n > 0 {
				return n
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	v, _ := p.Evaluate(expr)
	n, _ := toInt(v)
	return n
}

// waitEvalString 轮询取字符串。
func waitEvalString(t *testing.T, p Page, expr string, d time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(d)
	var last string
	for time.Now().Before(deadline) {
		v, err := p.Evaluate(expr)
		if err == nil {
			last, _ = v.(string)
			if last != "" {
				return last
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return last
}

// toInt 归一化 rod Evaluate 返回的数值（可能是 float64 / int / int64）。
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

// TestLiveView05_NoLeakAfterRepeatedWatch 泄漏验收（LIVEVIEW-05）。
//
// 反复 100 次 start/stop screencast，要求：
//   - goroutine 数增长有界（容差 8 个：Go runtime 与 rod 的少量后台 goroutine）
//   - Manager.PageCount 不变（停流不关 page）
//   - Chrome 主进程未重启（PID 不变）
func TestLiveView05_NoLeakAfterRepeatedWatch(t *testing.T) {
	m, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	// 预热：第一轮可能出现 rod 的懒启动 goroutine，不计入基线。
	preheat := func() {
		frames, _, stop, err := StartScreencast(raw, ScreencastOptions{
			Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 2,
			AckTimeout: 500 * time.Millisecond, ChannelCapacity: 1, Logger: nopLogger{},
		})
		if err != nil {
			t.Fatalf("预热 StartScreencast: %v", err)
		}
		_ = mustFrame(t, frames, 10*time.Second)
		for f := range frames {
			f.Ack()
			break
		}
		stop()
	}
	preheat()

	runtime.GC()
	time.Sleep(300 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()
	basePages := m.PageCount()
	basePID := m.PID()

	for i := 0; i < 100; i++ {
		frames, _, stop, err := StartScreencast(raw, ScreencastOptions{
			Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 2,
			AckTimeout: 500 * time.Millisecond, ChannelCapacity: 1, Logger: nopLogger{},
		})
		if err != nil {
			t.Fatalf("第 %d 轮 StartScreencast: %v", i, err)
		}
		// 读一帧（确保事件泵真的跑起来），然后立即停止。
		select {
		case f, ok := <-frames:
			if ok {
				f.Ack()
			}
		case <-time.After(3 * time.Second):
			// 本轮没帧（页面无变化）：不算失败，继续下一轮。
		}
		stop()
		if i%20 == 0 {
			runtime.GC()
		}
	}

	// 给后台 goroutine 一点收尾时间（rod 的事件订阅是异步退订的）。
	time.Sleep(1500 * time.Millisecond)
	runtime.GC()

	afterGoroutines := runtime.NumGoroutine()
	afterPages := m.PageCount()
	afterPID := m.PID()

	t.Logf("LIVEVIEW-05 泄漏测试：100 轮 watch/stop")
	t.Logf("  goroutine: base=%d after=%d delta=%d", baseGoroutines, afterGoroutines, afterGoroutines-baseGoroutines)
	t.Logf("  page 数:   base=%d after=%d", basePages, afterPages)
	t.Logf("  Chrome PID: base=%d after=%d", basePID, afterPID)

	if delta := afterGoroutines - baseGoroutines; delta > 8 {
		t.Fatalf("goroutine 增长 %d 个（> 容差 8），存在泄漏", delta)
	}
	if afterPages != basePages {
		t.Fatalf("page 数从 %d 变为 %d（停流不应关 page）", basePages, afterPages)
	}
	if basePID != 0 && afterPID != basePID {
		t.Fatalf("Chrome PID 从 %d 变为 %d（停流不应重启浏览器）", basePID, afterPID)
	}

	record := fmt.Sprintf(`{
  "req": "LIVEVIEW-05",
  "rounds": 100,
  "goroutinesBefore": %d,
  "goroutinesAfter": %d,
  "goroutineDelta": %d,
  "pagesBefore": %d,
  "pagesAfter": %d,
  "chromePidBefore": %d,
  "chromePidAfter": %d
}`, baseGoroutines, afterGoroutines, afterGoroutines-baseGoroutines, basePages, afterPages, basePID, afterPID)
	if err := writeMeasure("liveview05-leak.json", record); err != nil {
		t.Logf("写泄漏实测文件失败（不影响用例）: %v", err)
	}
}

// TestLiveView05_StopDoesNotClosePage 停流后 page 与 browser 必须仍可用（154 硬语义）。
func TestLiveView05_StopDoesNotClosePage(t *testing.T) {
	m, p, _ := newLivePage(t, nil)
	raw := p.RawPage()

	frames, _, stop, err := StartScreencast(raw, ScreencastOptions{
		Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 2,
		AckTimeout: 500 * time.Millisecond, ChannelCapacity: 1, Logger: nopLogger{},
	})
	if err != nil {
		t.Fatalf("StartScreencast: %v", err)
	}
	_ = mustFrame(t, frames, 10*time.Second)
	stop()

	if p.Closed() {
		t.Fatalf("停流后 page 被关闭（违反 154 「Close 只关 page / screencast 不关 page」语义）")
	}
	if m.Browser() == nil {
		t.Fatalf("停流后 browser 被关闭（违反 154 硬语义）")
	}
	// page 仍可正常执行动作。
	if _, err := p.Evaluate("1+1"); err != nil {
		t.Fatalf("停流后 page 不可用: %v", err)
	}
	// 可以再次开流（证明 stopScreencast 是可恢复的）。
	frames2, _, stop2, err := StartScreencast(raw, ScreencastOptions{
		Quality: 40, MaxWidth: 640, MaxHeight: 360, EveryNthFrame: 2,
		AckTimeout: 500 * time.Millisecond, ChannelCapacity: 1, Logger: nopLogger{},
	})
	if err != nil {
		t.Fatalf("停流后重启 screencast 失败: %v", err)
	}
	defer stop2()
	_ = mustFrame(t, frames2, 10*time.Second)
}

// writeMeasure 把实测数据写到 out/measure/（gitignored 构建产物目录）。
//
// 目录不存在时创建；写失败不阻断用例（实测数据由 SUMMARY 引用日志即可）。
func writeMeasure(name, content string) error {
	const dir = "../../../out/measure"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

// 保证 rod 与 sync/atomic 的导入被使用（避免各用例删除时编译器报未使用）。
var (
	_ = rod.New
	_ = atomic.LoadInt64
	_ sync.Mutex
)
