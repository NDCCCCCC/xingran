package liveview

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-02/03/05 单元验收（无浏览器、无 Redis）
//
//   TestLiveView03_ModeStateMachine     三模式与闸门决策（copilot 默认 / manual 全停 / auto 放行）
//   TestLiveView03_GateBlocksOnTakeover 接管期间 RPA 动作暂停，释放后继续
//   TestLiveView03_ModeInvalidRejected  非法模式拒绝（不静默降级）
//   TestLiveView05_SingleTakeoverer     单 page 单接管者；抢占需 force
//   TestLiveView05_IdleStopsStream      N 秒无下行指令即停流
//   TestLiveView05_ConcurrentLimit      超 maxStreams 返回可读错误
// ============================================================================

// memSink 内存帧接收器（统计帧数，不落网络）。
type memSink struct {
	mu     sync.Mutex
	frames int
	contex *types.LiveViewContext
	events []types.LiveViewEvent
}

func (m *memSink) PushFrame(_ context.Context, f *types.LiveViewUpstreamFrame) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frames++
	if f.Context != nil {
		c := *f.Context
		m.contex = &c
	}
	return nil
}

func (m *memSink) PushEvent(_ context.Context, ev *types.LiveViewAuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev.Event)
	return nil
}

func (m *memSink) frameCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.frames
}

func (m *memSink) hasEvent(kind types.LiveViewEvent) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e == kind {
			return true
		}
	}
	return false
}

// stubPage 最小 Page 实现（只在需要 RawPage 的路径上返回 nil，其余无害）。
type stubPage struct {
	closed bool
}

func (s *stubPage) Goto(string) error                                   { return nil }
func (s *stubPage) Close() error                                        { s.closed = true; return nil }
func (s *stubPage) Screenshot() ([]byte, error)                         { return nil, nil }
func (s *stubPage) Click(string) error                                  { return nil }
func (s *stubPage) Fill(string, string) error                           { return nil }
func (s *stubPage) Select(string, string) error                         { return nil }
func (s *stubPage) WaitFor(string, time.Duration) error                 { return nil }
func (s *stubPage) Evaluate(string) (interface{}, error)                { return nil, nil }
func (s *stubPage) GetText(string) (string, error)                      { return "", nil }
func (s *stubPage) GetValue(string) (string, error)                     { return "", nil }
func (s *stubPage) GetHTML(string) (string, error)                      { return "", nil }
func (s *stubPage) GetAttribute(string, string) (string, error)         { return "", nil }
func (s *stubPage) ScrollDown(int) error                                { return nil }
func (s *stubPage) ScrollUp(int) error                                  { return nil }
func (s *stubPage) ScrollToElement(string) error                        { return nil }
func (s *stubPage) RawPage() *rod.Page                                   { return nil }
func (s *stubPage) ContextPage() *rod.Page                               { return nil }
func (s *stubPage) Browser() *rod.Browser                                { return nil }
func (s *stubPage) Closed() bool                                        { return s.closed }
func (s *stubPage) Reset()                                              {}
func (s *stubPage) Upload(string, []string) error                       { return nil }
func (s *stubPage) Hover(string) error                                  { return nil }
func (s *stubPage) ClickWith(string, proto.InputMouseButton, int) error       { return nil }
func (s *stubPage) Drag(string, string) error                           { return nil }
func (s *stubPage) PressKeys(string) error                              { return nil }
func (s *stubPage) SwitchTab(int) error                                 { return nil }
func (s *stubPage) EnterFrame(string) error                             { return nil }
func (s *stubPage) ExitFrame() error                                    { return nil }
func (s *stubPage) ScrollInElement(string, int, int) error              { return nil }
func (s *stubPage) Download(context.Context, string, func() error) (string, error) { return "", nil }
func (s *stubPage) CloseCurrentTab() error                              { return nil }

// newTestHub 构造测试 hub（不启用真浏览器；sink 为内存实现）。
func newTestHub(t *testing.T, mut func(*config.LiveViewConfig)) (*Hub, *memSink) {
	t.Helper()
	cfg := &config.LiveViewConfig{
		Enabled:         true,
		MaxStreams:      2,
		IdleTimeout:     2 * time.Second,
		TakeoverTimeout: 3 * time.Second,
		DefaultMode:     "copilot",
	}
	if mut != nil {
		mut(cfg)
	}
	sink := &memSink{}
	h := NewHub(cfg, sink, sink, nil)
	t.Cleanup(func() { h.Close() })
	return h, sink
}

// TestLiveView03_ModeStateMachine 三模式默认值与取值校验。
func TestLiveView03_ModeStateMachine(t *testing.T) {
	if _, ok := types.NormalizeLiveMode(""); !ok {
		t.Fatalf("空模式应归一为默认")
	}
	if got, _ := types.NormalizeLiveMode(""); got != types.LiveModeCopilot {
		t.Fatalf("空模式归一 = %q，期望 copilot（LIVEVIEW-03 默认档位）", got)
	}
	if ok := types.IsValidLiveMode("hover"); ok {
		t.Fatalf("非法模式 hover 被判合法")
	}
	if _, ok := types.NormalizeLiveMode("wizard"); ok {
		t.Fatalf("非法模式 wizard 应被拒绝（不静默降级）")
	}
	for _, m := range []types.LiveMode{types.LiveModeAuto, types.LiveModeCopilot, types.LiveModeManual} {
		if !types.IsValidLiveMode(m) {
			t.Fatalf("合法模式 %q 被判非法", m)
		}
	}
}

// TestLiveView03_GateBlocksOnTakeover 接管期间闸门阻塞，释放后放行。
func TestLiveView03_GateBlocksOnTakeover(t *testing.T) {
	h, _ := newTestHub(t, func(c *config.LiveViewConfig) { c.DefaultMode = "auto" })
	p := &stubPage{}
	h.Attach("exec-1", p, nil)

	// auto 模式：不接管 → 立即放行。
	if err := h.BeforeAction(context.Background(), "exec-1", &types.Action{Type: types.ActionClick}, false); err != nil {
		t.Fatalf("auto 模式无接管时闸门应放行: %v", err)
	}

	// 接管 → 后续动作必须阻塞。
	if err := h.Takeover("exec-1", "alice", false); err != nil {
		t.Fatalf("Takeover: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- h.BeforeAction(context.Background(), "exec-1", &types.Action{Type: types.ActionClick}, false)
	}()

	select {
	case err := <-done:
		t.Fatalf("接管期间闸门未阻塞，直接返回 err=%v", err)
	case <-time.After(300 * time.Millisecond):
		// 期望：仍在阻塞
	}

	// 释放接管 → 阻塞的动作应被放行。
	if err := h.Release("exec-1", "alice", "测试释放"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("释放后放行应返回 nil，实际 %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("释放接管后闸门仍未放行（RPA 无法从断点继续）")
	}
}

// TestLiveView03_ManualWaitsEveryAction manual 模式每个动作都等人工继续。
func TestLiveView03_ManualWaitsEveryAction(t *testing.T) {
	h, _ := newTestHub(t, func(c *config.LiveViewConfig) { c.DefaultMode = "manual" })
	h.Attach("exec-2", &stubPage{}, nil)

	done := make(chan error, 1)
	go func() {
		done <- h.BeforeAction(context.Background(), "exec-2", &types.Action{Type: types.ActionClick}, false)
	}()

	select {
	case err := <-done:
		t.Fatalf("manual 模式应阻塞，实际直接返回 err=%v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if err := h.Continue("exec-2", "", "人工继续"); err != nil {
		t.Fatalf("Continue: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("继续后应放行: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Continue 后未放行")
	}
}

// TestLiveView03_CopilotCheckpoint copilot 只在关键节点（pause / confirm 标记）暂停。
func TestLiveView03_CopilotCheckpoint(t *testing.T) {
	h, _ := newTestHub(t, nil) // 默认 copilot
	h.Attach("exec-3", &stubPage{}, nil)

	// 普通动作：copilot 不暂停。
	if err := h.BeforeAction(context.Background(), "exec-3", &types.Action{Type: types.ActionClick}, false); err != nil {
		t.Fatalf("copilot 普通动作不应暂停: %v", err)
	}

	// pause 动作：必须暂停。
	done := make(chan error, 1)
	go func() {
		done <- h.BeforeAction(context.Background(), "exec-3", &types.Action{Type: types.ActionPause}, false)
	}()
	select {
	case err := <-done:
		t.Fatalf("copilot 的 pause 动作应暂停，实际直接返回 err=%v", err)
	case <-time.After(300 * time.Millisecond):
	}
	h.Continue("exec-3", "", "确认")
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("确认后应放行: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("copilot 确认后未放行")
	}

	// confirm 标记动作：同样暂停。
	done2 := make(chan error, 1)
	action := &types.Action{Type: types.ActionClick, Params: map[string]interface{}{"confirm": true}}
	go func() {
		done2 <- h.BeforeAction(context.Background(), "exec-3", action, false)
	}()
	select {
	case err := <-done2:
		t.Fatalf("params.confirm=true 的动作应暂停，实际直接返回 err=%v", err)
	case <-time.After(300 * time.Millisecond):
	}
	h.Release("exec-3", "", "释放") // 无接管者时幂等，不报错
	h.Continue("exec-3", "", "确认")
	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		t.Fatalf("confirm 动作确认后未放行")
	}
}

// TestLiveView03_ModeInvalidRejected 非法模式必须报错（不静默降级）。
func TestLiveView03_ModeInvalidRejected(t *testing.T) {
	h, _ := newTestHub(t, nil)
	h.Attach("exec-4", &stubPage{}, nil)
	if err := h.SetMode("exec-4", "unknown", "alice"); err == nil {
		t.Fatalf("非法模式应报错")
	}
	if err := h.SetMode("exec-4", types.LiveModeAuto, "alice"); err != nil {
		t.Fatalf("合法模式 auto 应成功: %v", err)
	}
	if got := h.Mode("exec-4"); got != types.LiveModeAuto {
		t.Fatalf("模式=%q，期望 auto", got)
	}
}

// TestLiveView05_SingleTakeoverer 单 page 单接管者；抢占需显式 force。
func TestLiveView05_SingleTakeoverer(t *testing.T) {
	h, sink := newTestHub(t, nil)
	h.Attach("exec-5", &stubPage{}, nil)

	if err := h.Takeover("exec-5", "alice", false); err != nil {
		t.Fatalf("alice 接管: %v", err)
	}
	if got := h.TakeoverBy("exec-5"); got != "alice" {
		t.Fatalf("接管者=%q，期望 alice", got)
	}
	// bob 无 force：必须被拒（已有接管者）。
	if err := h.Takeover("exec-5", "bob", false); err == nil {
		t.Fatalf("bob 无 force 抢占应被拒绝")
	}
	// bob 带 force：允许（显式确认）。
	if err := h.Takeover("exec-5", "bob", true); err != nil {
		t.Fatalf("bob 带 force 抢占应成功: %v", err)
	}
	if got := h.TakeoverBy("exec-5"); got != "bob" {
		t.Fatalf("抢占后接管者=%q，期望 bob", got)
	}

	// 审计：至少一条 takeover 事件（worker 侧 sink 可能异步，等一小会）。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sink.hasEvent(types.LiveViewEventTakeover) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !sink.hasEvent(types.LiveViewEventTakeover) {
		t.Fatalf("接管未产生审计事件（LIVEVIEW-05 四类审计）")
	}
}

// TestLiveView05_IdleStopsStream N 秒无下行指令即停流。
func TestLiveView05_IdleStopsStream(t *testing.T) {
	h, _ := newTestHub(t, func(c *config.LiveViewConfig) { c.IdleTimeout = 300 * time.Millisecond })
	h.Attach("exec-6", &stubPage{}, nil)

	// 直接把 lastSignal 设成过去时刻（模拟「已建立流但无人发指令」）。
	h.mu.Lock()
	s := h.sessions["exec-6"]
	h.mu.Unlock()
	if s == nil {
		t.Fatalf("Attach 后会话不存在")
	}

	// 伪造一个「正在推流」的状态：用一个 fake stopStream 计数。
	var stopped int
	s.mu.Lock()
	s.stopStream = func() { stopped++ }
	s.framesDone = make(chan struct{})
	close(s.framesDone)
	s.lastSignal = time.Now().Add(-1 * time.Second)
	s.mu.Unlock()

	h.sweep(time.Now())
	time.Sleep(100 * time.Millisecond)

	if stopped != 1 {
		t.Fatalf("空闲超时后 stopStream 调用次数=%d，期望 1（LIVEVIEW-05 断流）", stopped)
	}
}

// TestLiveView05_ConcurrentLimit 超 maxStreams 返回可读错误。
func TestLiveView05_ConcurrentLimit(t *testing.T) {
	h, _ := newTestHub(t, func(c *config.LiveViewConfig) { c.MaxStreams = 1 })
	h.Attach("exec-a", &stubPage{}, nil)
	h.Attach("exec-b", &stubPage{}, nil)

	// 人为占满一个名额（不真开 screencast：RawPage 为 nil，Watch 会失败）。
	h.mu.Lock()
	s := h.sessions["exec-a"]
	h.mu.Unlock()
	s.mu.Lock()
	s.stopStream = func() {}
	s.mu.Unlock()

	err := h.Watch("exec-b", "alice")
	if err == nil {
		t.Fatalf("超 maxStreams 应返回错误")
	}
	var limit *LimitError
	if !asLimit(err, &limit) {
		t.Fatalf("错误类型应为 *LimitError，实际 %T: %v", err, err)
	}
	if limit.Max != 1 {
		t.Fatalf("LimitError.Max=%d，期望 1", limit.Max)
	}
	t.Logf("超限错误文案：%s", limit.Error())
}

// asLimit 判定错误是否为 *LimitError（避免手写 errors.As 的类型断言样板）。
func asLimit(err error, out **LimitError) bool {
	if e, ok := err.(*LimitError); ok {
		*out = e
		return true
	}
	return false
}

// TestLiveView05_WatchDisabledNotEnabled 未启用时 Watch 返回可读错误。
func TestLiveView05_WatchDisabledNotEnabled(t *testing.T) {
	h, _ := newTestHub(t, func(c *config.LiveViewConfig) { c.Enabled = false })
	h.Attach("exec-x", &stubPage{}, nil)
	if err := h.Watch("exec-x", "alice"); err == nil {
		t.Fatalf("未启用时应返回可读错误")
	} else {
		t.Logf("未启用文案：%s", err.Error())
	}
}

// TestLiveView05_DetachStopsAndAudits 执行结束 Detach 停流并写 watch_stop 审计。
func TestLiveView05_DetachStopsAndAudits(t *testing.T) {
	h, sink := newTestHub(t, nil)
	h.Attach("exec-d", &stubPage{}, nil)

	h.Detach("exec-d")
	if n := h.SessionCount(); n != 0 {
		t.Fatalf("Detach 后会话数=%d，期望 0", n)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if sink.hasEvent(types.LiveViewEventWatchStop) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !sink.hasEvent(types.LiveViewEventWatchStop) {
		t.Fatalf("Detach 未写 watch_stop 审计")
	}
}

// TestLiveView05_MemorySinkContract 编译期断言：APIClient 风格 sink 契约（内存实现）。
var _ FrameSink = (*memSink)(nil)
var _ AuditSink = (*memSink)(nil)
