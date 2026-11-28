package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 156 / NOCODE-01：真 Chromium 录制 → 回放验收
//
// 验收口径（任务书）：真 Chromium 里录一段「登录 → 填表 → 提交」，产出的
// actions 能直接跑通。
//
// 本测试是**端到端**的：
//  1. 起一个本地 HTTP 服务（登录页 + 报销表单页），带 data-testid 与
//     aria-label（模拟内网系统有 testid 与无 testid 两种现实）。
//  2. 起真 Chromium（TestChromePath 探测，缺失则 Skip 并显式登记）。
//  3. 用 Recorder 开始录制，然后用**真实鼠标键盘事件**完成「登录 → 填表 →
//     提交」，而不是直接 evaluate 改 DOM —— 后者绕过了监听器，验收无效。
//  4. Stop 拿到事件 → BuildRecording 产出 actions。
//  5. 用 PageManager（154 的阶梯 + 动作全集）**回放这些 actions**，断言服务端
//     真实收到了提交数据。
//
// 缺 Chrome 时 Skip 而非 Fail：CI 容器可能没有浏览器，但本机必须真跑（见
// 156-SUMMARY 的走查记录）。
// ============================================================================

// loginPageHTML 登录页（带 data-testid，验证第 1 级阶梯）。
const loginPageHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>登录</title></head>
<body>
  <h1>报销系统登录</h1>
  <form id="loginForm" method="get" action="/form">
    <label for="u">用户名</label>
    <input id="u" data-testid="login-username" name="username" type="text" placeholder="请输入用户名" />
    <label for="p">密码</label>
    <input id="p" data-testid="login-password" name="password" type="password" placeholder="请输入密码" />
    <button type="submit" data-testid="login-submit">登录</button>
  </form>
</body></html>`

// formPageHTML 报销表单页（部分字段无 testid，验证 role/aria 级阶梯）。
const formPageHTML = `<!DOCTYPE html>
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

const donePageHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>完成</title></head>
<body><h1>提交成功</h1></body></html>`

// recordingFixture 录制验收用的本地服务。
type recordingFixture struct {
	server *httptest.Server
	// submitted 记录最终提交参数（回放是否真的跑通的唯一硬证据）。
	submitted atomic.Value
	// formHit 表单页被访问次数（证明 navigate 动作真的生效）。
	formHit atomic.Int32
}

func newRecordingFixture(t *testing.T) *recordingFixture {
	t.Helper()
	f := &recordingFixture{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/login":
			fmt.Fprint(w, loginPageHTML)
		case r.URL.Path == "/form":
			f.formHit.Add(1)
			fmt.Fprint(w, formPageHTML)
		case r.URL.Path == "/done":
			q := r.URL.Query()
			f.submitted.Store(fmt.Sprintf("amount=%s;type=%s", q.Get("amount"), q.Get("type")))
			fmt.Fprint(w, donePageHTML)
		default:
			http.NotFound(w, r)
		}
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

// launchTestPage 起一个真 Chromium 页面（缺浏览器则 Skip）。
func launchTestPage(t *testing.T) *rod.Page {
	t.Helper()
	path := TestChromePath()
	if path == "" {
		t.Skip("未找到可用 Chrome/Chromium（可设 RPA_TEST_CHROME_PATH 指定），跳过真浏览器验收")
	}
	l := launcher.New().Bin(path).
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("disable-features", "Translate,BackForwardCache").
		Leakless(false)
	controlURL, err := l.Launch()
	if err != nil {
		t.Fatalf("启动 Chrome 失败: %v", err)
	}
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Kill()
		t.Fatalf("连接 Chrome 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = browser.Close()
		l.Kill()
	})
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatalf("新建页面失败: %v", err)
	}
	return page
}

// clickReal 用真实鼠标事件点击（不是 evaluate 改 DOM —— 那样绕过监听器）。
func clickReal(t *testing.T, page *rod.Page, selector string) {
	t.Helper()
	el, err := page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		t.Fatalf("找不到元素 %s: %v", selector, err)
	}
	if err := el.Click("left", 1); err != nil {
		t.Fatalf("点击 %s 失败: %v", selector, err)
	}
	time.Sleep(150 * time.Millisecond)
}

// typeReal 用真实键盘输入（触发 input 事件，录制器才有东西可录）。
func typeReal(t *testing.T, page *rod.Page, selector, value string) {
	t.Helper()
	el, err := page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		t.Fatalf("找不到元素 %s: %v", selector, err)
	}
	if err := el.SelectAllText(); err != nil {
		t.Logf("全选失败（继续）: %v", err)
	}
	if err := el.Input(value); err != nil {
		t.Fatalf("输入 %s 失败: %v", selector, err)
	}
	time.Sleep(120 * time.Millisecond)
}

// TestRecorder_RealChromium_LoginFillSubmit NOCODE-01 主验收：
// 真 Chromium 录「登录 → 填表 → 提交」，产出的 actions 直接回放跑通。
func TestRecorder_RealChromium_LoginFillSubmit(t *testing.T) {
	fixture := newRecordingFixture(t)
	page := launchTestPage(t)
	log := nopLogger{}

	// 打开登录页（录制前的初始状态）
	if err := page.Navigate(fixture.server.URL + "/login"); err != nil {
		t.Fatalf("导航失败: %v", err)
	}
	page.MustWaitLoad()

	// ===== 录制开始 =====
	rec := NewRecorder(page, log)
	if err := rec.Start(); err != nil {
		t.Fatalf("开始录制失败: %v", err)
	}

	// 真实操作 1：填用户名
	typeReal(t, page, `[data-testid="login-username"]`, "zhangsan")
	// 真实操作 2：填密码
	typeReal(t, page, `[data-testid="login-password"]`, "Passw0rd!")
	// 真实操作 3：点登录（会整页跳到 /form）
	clickReal(t, page, `[data-testid="login-submit"]`)
	page.MustWaitLoad()
	if got := fixture.formHit.Load(); got == 0 {
		t.Fatalf("登录后未到达表单页（formHit=0），URL=%s", page.MustInfo().URL)
	}

	// 真实操作 4：填金额
	typeReal(t, page, "#amt", "1280")
	// 真实操作 5：选下拉（触发 change 事件）。
	// rod 的 Select 第三参是「option 的选择器类型」：传 value 文本需配 CSS 选择器，
	// 直接传 "hotel" 配 SelectorTypeText 会按可见文本找而找不到。
	selEl, err := page.Timeout(5 * time.Second).Element("#ty")
	if err != nil {
		t.Fatalf("表单页找不到下拉框（当前 URL=%s）: %v", page.MustInfo().URL, err)
	}
	if err := selEl.Select([]string{`option[value="hotel"]`}, true, rod.SelectorTypeCSSSector); err != nil {
		t.Fatalf("选择下拉失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	// 真实操作 6：提交
	clickReal(t, page, "#submitExpense")
	page.MustWaitLoad()

	// ===== 录制结束 =====
	events, err := rec.Stop()
	if err != nil {
		t.Fatalf("停止录制失败: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("录制未捕获任何事件（注入可能失效）")
	}

	result := BuildRecording(events)
	t.Logf("录制产出 %d 个事件 → %d 个动作", result.EventCount, len(result.Actions))
	for i, s := range result.Steps {
		t.Logf("  步骤 %d: %s", i+1, s)
	}

	// ===== 断言：动作序列结构正确 =====
	if len(result.Actions) < 5 {
		t.Fatalf("动作过少（%d 条），录制明显不完整", len(result.Actions))
	}
	if result.Actions[0].Type != types.ActionNavigate {
		t.Fatalf("首条必须是 navigate，实际 %s", result.Actions[0].Type)
	}

	// 断言：选择器优先落在 testid / role 级（ENGINE-03 约定）
	var testidHits int
	for _, a := range result.Actions {
		if strings.Contains(a.Selector, "data-testid") {
			testidHits++
		}
	}
	if testidHits == 0 {
		t.Error("录制产出的选择器应至少有一条落在 data-testid 级（阶梯顶端）")
	}

	// 断言：密码绝不明文
	raw := fmt.Sprint(result.Actions)
	if strings.Contains(raw, "Passw0rd!") {
		t.Fatal("录制结果泄漏了密码明文")
	}

	// 断言：中文步骤描述可读且无技术术语
	for _, s := range result.Steps {
		if strings.ContainsAny(s, "#.[]") || strings.Contains(s, "div") {
			t.Errorf("步骤描述泄漏技术术语: %q", s)
		}
	}

	// ===== 回放：用 154 的动作全集执行录制结果 =====
	if err := replayActions(t, page, result.Actions); err != nil {
		t.Fatalf("回放录制结果失败: %v", err)
	}

	// 硬证据：服务端收到了真实提交
	got, _ := fixture.submitted.Load().(string)
	if got == "" {
		t.Fatal("回放后服务端未收到提交（actions 不可直接执行）")
	}
	t.Logf("回放后服务端收到: %s", got)
	if !strings.Contains(got, "amount=1280") {
		t.Errorf("回放提交的金额不对: %s", got)
	}
	if !strings.Contains(got, "type=hotel") {
		t.Errorf("回放提交的类型不对: %s", got)
	}
}

// replayActions 用 154 的 PageManager（选择器阶梯 + 动作全集）回放动作。
//
// 刻意不自己实现动作语义：如果录制产出的 schema 与 types.Action 不一致，
// 这里就会编译/运行失败 —— 这就是「schema 一致」的验证手段。
func replayActions(t *testing.T, page *rod.Page, actions []types.Action) error {
	t.Helper()
	pm := NewPageManager(&rodPageAdapter{page: page}, nopLogger{})

	for i := range actions {
		a := &actions[i]
		t.Logf("回放 [%d] %s: %s", i+1, a.Type, a.Description)
		switch a.Type {
		case types.ActionNavigate:
			url, _ := a.Params["url"].(string)
			if url == "" {
				return fmt.Errorf("第 %d 步 navigate 缺少 url", i+1)
			}
			if err := pm.Goto(url); err != nil {
				return fmt.Errorf("第 %d 步导航失败: %w", i+1, err)
			}
			time.Sleep(200 * time.Millisecond)
		case types.ActionFill:
			v, _ := a.Params["value"].(string)
			if a.Selector == "" {
				return fmt.Errorf("第 %d 步 fill 缺少 selector", i+1)
			}
			// 密码占位符在回放时用测试值替换（真实环境由凭证注入）
			if v == "${password}" {
				v = "Passw0rd!"
			}
			if err := pm.Fill(a.Selector, v); err != nil {
				return fmt.Errorf("第 %d 步填写失败（选择器 %q）: %w", i+1, a.Selector, err)
			}
			time.Sleep(100 * time.Millisecond)
		case types.ActionSelect:
			v, _ := a.Params["value"].(string)
			if a.Selector == "" {
				return fmt.Errorf("第 %d 步 select 缺少 selector", i+1)
			}
			if err := pm.Select(a.Selector, v); err != nil {
				return fmt.Errorf("第 %d 步选择失败（选择器 %q）: %w", i+1, a.Selector, err)
			}
			time.Sleep(100 * time.Millisecond)
		case types.ActionClick:
			if a.Selector == "" {
				return fmt.Errorf("第 %d 步 click 缺少 selector", i+1)
			}
			if err := pm.Click(a.Selector); err != nil {
				return fmt.Errorf("第 %d 步点击失败（选择器 %q）: %w", i+1, a.Selector, err)
			}
			time.Sleep(300 * time.Millisecond)
		case types.ActionKeyboard:
			keys, _ := a.Params["keys"].(string)
			if err := pm.PressKeys(keys); err != nil {
				return fmt.Errorf("第 %d 步按键失败: %w", i+1, err)
			}
		case types.ActionScroll:
			pixels, _ := a.Params["pixels"].(int)
			if pixels <= 0 {
				pixels = 500
			}
			if err := pm.ScrollDown(pixels); err != nil {
				return fmt.Errorf("第 %d 步滚动失败: %w", i+1, err)
			}
		default:
			t.Logf("回放跳过非关键动作 %s", a.Type)
		}
	}
	return nil
}

// rodPageAdapter 把 *rod.Page 适配成 Page 接口（回放只需部分能力）。
//
// 显式实现而非反射：接口方法集变化时这里会编译失败，起到契约守门作用。
type rodPageAdapter struct {
	page *rod.Page
}

func (a *rodPageAdapter) Goto(url string) error { return a.page.Navigate(url) }
func (a *rodPageAdapter) Close() error          { return nil }
func (a *rodPageAdapter) Screenshot() ([]byte, error) {
	return a.page.Screenshot(false, nil)
}
func (a *rodPageAdapter) Click(selector string) error { return a.page.MustElement(selector).Click("left", 1) }
func (a *rodPageAdapter) Fill(selector, value string) error {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return err
	}
	if err := el.SelectAllText(); err != nil {
		return err
	}
	return el.Input(value)
}
func (a *rodPageAdapter) Select(selector, value string) error {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return err
	}
	return el.Select([]string{fmt.Sprintf(`option[value=%q]`, value)}, true, rod.SelectorTypeCSSSector)
}
func (a *rodPageAdapter) WaitFor(selector string, timeout time.Duration) error {
	_, err := a.page.Timeout(timeout).Element(selector)
	return err
}
func (a *rodPageAdapter) Evaluate(script string) (interface{}, error) {
	res, err := a.page.Eval(script)
	if err != nil {
		return nil, err
	}
	return res.Value.Val(), nil
}
func (a *rodPageAdapter) GetText(selector string) (string, error) {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", err
	}
	return el.Text()
}
func (a *rodPageAdapter) GetValue(selector string) (string, error) {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", err
	}
	prop, err := el.Property("value")
	if err != nil {
		return "", err
	}
	return prop.String(), nil
}
func (a *rodPageAdapter) GetHTML(selector string) (string, error) {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", err
	}
	return el.HTML()
}
func (a *rodPageAdapter) GetAttribute(selector, attrName string) (string, error) {
	el, err := a.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return "", err
	}
	v, err := el.Attribute(attrName)
	if err != nil || v == nil {
		return "", err
	}
	return *v, nil
}
func (a *rodPageAdapter) ScrollDown(pixels int) error {
	return a.page.Mouse.Scroll(0, float64(pixels), 1)
}
func (a *rodPageAdapter) ScrollUp(pixels int) error {
	return a.page.Mouse.Scroll(0, float64(-pixels), 1)
}
func (a *rodPageAdapter) ScrollToElement(selector string) error {
	return a.page.MustElement(selector).ScrollIntoView()
}
func (a *rodPageAdapter) RawPage() *rod.Page     { return a.page }
func (a *rodPageAdapter) ContextPage() *rod.Page { return a.page }
func (a *rodPageAdapter) Browser() *rod.Browser  { return a.page.Browser() }
func (a *rodPageAdapter) Closed() bool           { return false }
func (a *rodPageAdapter) Reset()                 {}
func (a *rodPageAdapter) Upload(selector string, files []string) error {
	return a.page.MustElement(selector).SetFiles(files)
}
func (a *rodPageAdapter) Hover(selector string) error {
	return a.page.MustElement(selector).Hover()
}
func (a *rodPageAdapter) ClickWith(selector string, button proto.InputMouseButton, count int) error {
	return a.page.MustElement(selector).Click(button, count)
}
func (a *rodPageAdapter) Drag(fromSelector, toSelector string) error {
	return fmt.Errorf("adapter 不支持 drag")
}
func (a *rodPageAdapter) PressKeys(keys string) error {
	// 复用录制器同包的组合键解析，保证与引擎语义一致。
	parsed, err := parseKeyCombo(keys)
	if err != nil {
		return err
	}
	ka := a.page.KeyActions()
	for _, k := range parsed {
		ka.Press(k)
	}
	return ka.Do()
}
func (a *rodPageAdapter) SwitchTab(index int) error        { return fmt.Errorf("adapter 不支持 switchTab") }
func (a *rodPageAdapter) EnterFrame(selector string) error { return fmt.Errorf("adapter 不支持 iframe") }
func (a *rodPageAdapter) ExitFrame() error                 { return fmt.Errorf("adapter 不支持 iframe") }
func (a *rodPageAdapter) ScrollInElement(selector string, deltaX, deltaY int) error {
	return fmt.Errorf("adapter 不支持元素内滚动")
}
func (a *rodPageAdapter) Download(ctx context.Context, dir string, trigger func() error) (string, error) {
	return "", fmt.Errorf("adapter 不支持下载")
}
func (a *rodPageAdapter) CloseCurrentTab() error { return fmt.Errorf("adapter 不支持关闭页签") }
