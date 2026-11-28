package recording

// ============================================================================
// Phase W6 / REC-01：录制会话真 Chromium 端到端验收。
//
// 验收口径（任务书）：真 Chromium 里录 3 个真实动作 → 停止 → 产出 3 条动作
// → 上行给后端。本测试覆盖「录制会话层」的完整链路（Hub.Start → 操作 →
// Hub.Stop → Upstream 收到权威产出）：
//
//	浏览器 Manager → Hub.Start（Acquire page + 注入录制器）
//	  → 真实鼠标键盘事件（不是 evaluate 改 DOM —— 那样绕过监听器）
//	→ Hub.Stop（recorder.Stop → BuildRecording → 上行）
//	  → 断言 Upstream 收到 actions + 中文步骤
//
// 缺 Chrome 时 Skip 而非 Fail（与 156 的集成测试同款约定）。
// ============================================================================

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// nopLogger 空日志器。
type nopLogger struct{}

func (nopLogger) Debug(string, ...logger.Field) {}
func (nopLogger) Info(string, ...logger.Field)  {}
func (nopLogger) Warn(string, ...logger.Field)  {}
func (nopLogger) Error(string, ...logger.Field) {}
func (l nopLogger) With(...logger.Field) logger.Logger {
	return l
}
func (nopLogger) Sync() error { return nil }

// testUpstream 收集上行（状态/产出/拾取）。
type testUpstream struct {
	mu      sync.Mutex
	results []*types.RecordingUpstreamResult
	picks   []*types.RecordingUpstreamPick
	status  []*types.RecordingUpstreamStatus
}

func (u *testUpstream) ReportStatus(_ context.Context, st *types.RecordingUpstreamStatus) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status = append(u.status, st)
	return nil
}

func (u *testUpstream) ReportResult(_ context.Context, res *types.RecordingUpstreamResult) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.results = append(u.results, res)
	return nil
}

func (u *testUpstream) ReportPick(_ context.Context, p *types.RecordingUpstreamPick) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.picks = append(u.picks, p)
	return nil
}

// simpleFixture 简单表单页（录 3 个动作：填金额 → 选类型 → 点提交）。
const simpleFormHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>报销填报</title></head>
<body>
  <h1>差旅报销填报</h1>
  <form id="expenseForm" method="get" action="/done">
    <label for="amt">金额</label>
    <input id="amt" name="amount" type="text" placeholder="请输入金额" />
    <label for="ty">类型</label>
    <select id="ty" name="type" aria-label="报销类型">
      <option value="traffic">交通费</option>
      <option value="hotel">住宿费</option>
    </select>
    <button type="submit" aria-label="提交报销单" id="submitExpense">提交</button>
  </form>
</body></html>`

const doneHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>完成</title></head>
<body><h1>提交成功</h1></body></html>`

func newSimpleFixture(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/", "/form":
			fmt.Fprint(w, simpleFormHTML)
		case "/done":
			fmt.Fprint(w, doneHTML)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newRecordingTestManager 起一个共享 Chrome 的 Manager（缺浏览器则 Skip）。
func newRecordingTestManager(t *testing.T) *browser.Manager {
	t.Helper()
	path := browser.TestChromePath()
	if path == "" {
		t.Skip("未找到可用 Chrome/Chromium（可设 RPA_TEST_CHROME_PATH 指定），跳过真浏览器验收")
	}

	// 直接用 rod 起一个共享 browser，再把它注入 Manager 的私有字段 —— 不可行
	// （browser 字段私有）。改用 Manager 自带的懒启动：配置 chrome 路径即可。
	cfg := &config.BrowserConfig{
		Headless:     true,
		ChromePath:   path,
		Timeout:      30 * time.Second,
		MaxConcurrency: 5,
	}
	mgr, err := browser.NewManager(cfg, nopLogger{})
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

// TestRecordingHub_RealChromium_PickTierLadder REC-01 拾取链路验收：
// 按 viewport 坐标拾取元素 → 返回 7 级阶梯（逐级可用性）→ 推荐级可命中。
func TestRecordingHub_RealChromium_PickTierLadder(t *testing.T) {
	srv := newSimpleFixture(t)
	mgr := newRecordingTestManager(t)

	up := &testUpstream{}
	hub := NewHub(&config.RecordingConfig{Enabled: true}, up, nopLogger{})
	defer hub.Close()

	if err := hub.Start(mgr, &types.RecordingCommand{
		Action:    types.RecordingActionStart,
		SessionID: "test-session-pick",
		TargetURL: srv.URL + "/form",
	}); err != nil {
		t.Fatalf("开始录制失败: %v", err)
	}

	hub.mu.Lock()
	sess := hub.sessions["test-session-pick"]
	hub.mu.Unlock()
	if sess == nil || sess.page == nil {
		t.Fatal("会话未登记")
	}
	raw := sess.page.RawPage()

	// 定位提交按钮的中心坐标（viewport CSS 像素）。
	// Interactable 返回元素可点击中心点的 viewport 坐标（与 liveview 注入
	// 同一套坐标语义）—— 与 156 拾取测试同款先例。
	btn, err := raw.Timeout(10 * time.Second).Element("#submitExpense")
	if err != nil {
		t.Fatalf("找不到提交按钮: %v", err)
	}
	pt, err := btn.Interactable()
	if err != nil {
		t.Fatalf("取可点击中心失败: %v", err)
	}
	cx, cy := pt.X, pt.Y

	// 拾取（经 Hub.Pick —— 与前端「画布点元素」走同一条链路）。
	pick, err := hub.Pick("test-session-pick", cx, cy)
	if err != nil {
		t.Fatalf("拾取失败: %v", err)
	}
	if !pick.Found {
		t.Fatalf("拾取按钮中心 (%.0f, %.0f) 未命中元素", cx, cy)
	}
	if pick.Tag != "button" {
		t.Errorf("拾取到的标签应为 button，实际 %q", pick.Tag)
	}
	t.Logf("拾取到 <%s> 文本=%q 推荐=%q（第 %d 级，评分 %d）",
		pick.Tag, pick.Text, pick.Recommended, pick.RecommendedTier, pick.RecommendedScore)

	// 断言：阶梯非空且至少一级可用。
	if len(pick.Tiers) == 0 {
		t.Fatal("拾取未返回选择器阶梯")
	}
	usable := 0
	for _, tier := range pick.Tiers {
		if tier.OK {
			usable++
		}
	}
	if usable == 0 {
		t.Fatal("阶梯没有任何可用层级")
	}

	// 上行断言：拾取结果已交给 Upstream（后端推给前端浮层）。
	up.mu.Lock()
	picks := up.picks
	up.mu.Unlock()
	if len(picks) != 1 {
		t.Fatalf("Upstream 应恰收到 1 次拾取上报，实际 %d 次", len(picks))
	}
	if len(picks[0].Tiers) == 0 {
		t.Error("上行拾取结果的阶梯为空")
	}
}

// TestRecordingHub_SessionLimit 并发上限：超过 max_sessions 的 start 被拒。
func TestRecordingHub_SessionLimit(t *testing.T) {
	path := browser.TestChromePath()
	if path == "" {
		t.Skip("未找到可用 Chrome/Chromium，跳过真浏览器验收")
	}

	cfg := &config.BrowserConfig{
		Headless:       true,
		ChromePath:     path,
		Timeout:        30 * time.Second,
		MaxConcurrency: 5,
	}
	mgr, err := browser.NewManager(cfg, nopLogger{})
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	up := &testUpstream{}
	hub := NewHub(&config.RecordingConfig{
		Enabled:     true,
		MaxSessions: 1,
		IdleTimeout: 5 * time.Minute,
	}, up, nopLogger{})
	defer hub.Close()

	if err := hub.Start(mgr, &types.RecordingCommand{
		Action: types.RecordingActionStart, SessionID: "s1",
	}); err != nil {
		t.Fatalf("第一个会话应能开始: %v", err)
	}
	err = hub.Start(mgr, &types.RecordingCommand{
		Action: types.RecordingActionStart, SessionID: "s2",
	})
	limitErr, ok := err.(*LimitError)
	if err == nil || !ok {
		t.Fatalf("第二个会话应被上限拒绝（LimitError），实际: %v", err)
	}
	t.Logf("超限错误文案: %s", limitErr.Error())
}
func TestRecordingHub_RealChromium_ThreeActions(t *testing.T) {
	srv := newSimpleFixture(t)
	mgr := newRecordingTestManager(t)

	up := &testUpstream{}
	hub := NewHub(&config.RecordingConfig{
		Enabled:     true,
		MaxSessions: 2,
		IdleTimeout: 5 * time.Minute,
	}, up, nopLogger{})
	defer hub.Close()

	// ===== 1) 开始录制会话 =====
	cmd := &types.RecordingCommand{
		Action:    types.RecordingActionStart,
		SessionID: "test-session-e2e",
		Operator:  "张三",
		TargetURL: srv.URL + "/form",
	}
	if err := hub.Start(mgr, cmd); err != nil {
		t.Fatalf("开始录制失败: %v", err)
	}

	// 拿到会话的 page（用于模拟员工的真实操作）。
	hub.mu.Lock()
	sess := hub.sessions["test-session-e2e"]
	hub.mu.Unlock()
	if sess == nil || sess.page == nil {
		t.Fatal("会话未登记或页面为空")
	}
	raw := sess.page.RawPage()
	if raw == nil {
		t.Fatal("RawPage 为空")
	}

	// ===== 2) 真实操作 3 个动作（真实鼠标键盘事件，不 evaluate） =====
	// 动作 1：填金额
	el, err := raw.Timeout(10 * time.Second).Element("#amt")
	if err != nil {
		t.Fatalf("找不到金额输入框: %v", err)
	}
	if err := el.SelectAllText(); err != nil {
		t.Logf("全选失败（继续）: %v", err)
	}
	if err := el.Input("1280"); err != nil {
		t.Fatalf("输入金额失败: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	// 动作 2：选下拉（触发 change 事件）
	selEl, err := raw.Timeout(10 * time.Second).Element("#ty")
	if err != nil {
		t.Fatalf("找不到下拉框: %v", err)
	}
	if err := selEl.Select([]string{`option[value="hotel"]`}, true, rod.SelectorTypeCSSSector); err != nil {
		t.Fatalf("选择下拉失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 动作 3：点提交（真实点击）
	btn, err := raw.Timeout(10 * time.Second).Element("#submitExpense")
	if err != nil {
		t.Fatalf("找不到提交按钮: %v", err)
	}
	if err := btn.Click("left", 1); err != nil {
		t.Fatalf("点击提交失败: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	// ===== 3) 停止录制 → 产出 → 上行 =====
	result, err := hub.Stop(mgr, "test-session-e2e")
	if err != nil {
		t.Fatalf("停止录制失败: %v", err)
	}

	t.Logf("录制产出 %d 个事件 → %d 个动作", result.EventCount, len(result.Actions))
	for i, s := range result.Steps {
		t.Logf("  步骤 %d: %s", i+1, s)
	}

	// ===== 4) 断言 =====
	if len(result.Actions) < 3 {
		t.Fatalf("动作过少（%d 条）：录制 3 个真实操作至少应产出 3 条动作", len(result.Actions))
	}
	if result.EventCount == 0 {
		t.Fatal("事件数为 0（录制器注入可能失效）")
	}
	// 中文步骤与动作等长且可读（零技术术语）
	if len(result.Steps) != len(result.Actions) {
		t.Fatalf("步骤数（%d）与动作数（%d）不等", len(result.Steps), len(result.Actions))
	}
	for _, s := range result.Steps {
		if s == "" {
			t.Error("步骤描述为空")
		}
	}

	// 上行断言：权威产出已交给 Upstream（后端据此写任务脚本）
	up.mu.Lock()
	results := up.results
	up.mu.Unlock()
	if len(results) != 1 {
		t.Fatalf("Upstream 应恰收到 1 次产出上报，实际 %d 次", len(results))
	}
	if len(results[0].Actions) != len(result.Actions) {
		t.Errorf("上行产出动作数不一致：本地产出 %d / 上行 %d",
			len(result.Actions), len(results[0].Actions))
	}
	if results[0].SessionID != "test-session-e2e" {
		t.Errorf("上行产出的 sessionId 不对: %q", results[0].SessionID)
	}

	// 会话状态收尾：page 已归还
	hub.mu.Lock()
	st := hub.sessions["test-session-e2e"]
	hub.mu.Unlock()
	if st != nil && st.page != nil {
		t.Error("停止后会话页面未释放（page 泄漏）")
	}
}
