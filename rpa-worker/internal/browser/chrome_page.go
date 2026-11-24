package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ChromePage 单个页面（标签页）实现。
//
// Phase 154 / ENGINE-01 起，browser 字段是 Manager 共享的同一个
// *rod.Browser；本类型不再拥有 Chrome 生命周期：
//   - NewChromePage 仅供独立使用/测试路径（自起一个 Chrome 并由本类型负责）；
//   - Manager 路径走 NewChromePageFromBrowser，绝不关闭 browser。
type ChromePage struct {
	browser *rod.Browser
	page    *rod.Page
	logger  logger.Logger
	config  *config.BrowserConfig

	// ownsBrowser 标记本实例是否负责 Chrome 生命周期。Manager 创建的 page
	// 恒为 false —— Close() 只关 page（ENGINE-01 硬语义）。
	ownsBrowser bool

	// frame 当前 iframe 上下文；nil 表示主文档。
	frame *rod.Page
	// mainPage 本次借出的根页签（Reset 时把上下文恢复回它，避免上一个
	// 任务 switchTab / iframe 的残留影响下一个任务）。
	mainPage *rod.Page
	// baseTargets 借出时的页签快照：本次执行期间新开的页签 = 不在快照里的
	// target。用于让 switchTab 的序号在「池里有其它任务占用页签」时仍然
	// 可预测（index=1 永远是本次执行新开的第一个页签）。
	baseTargets map[string]bool
	closed      bool
}

// NewChromePage 创建「自持 Chrome」的页面（独立路径，保留给直接使用者与
// 需要一次性浏览器的场景）。Manager 不使用本函数。
func NewChromePage(ctx context.Context, cfg *config.BrowserConfig, log logger.Logger) (*ChromePage, error) {
	l := newLauncher(cfg)
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("启动 Chrome 失败: %w", err)
	}
	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		l.Kill()
		return nil, fmt.Errorf("连接 Chrome 失败: %w", err)
	}
	p, err := newChromePageOn(ctx, b, cfg, log)
	if err != nil {
		_ = b.Close()
		l.Kill()
		return nil, err
	}
	p.ownsBrowser = true
	return p, nil
}

// NewChromePageFromBrowser 在共享 browser 上新建一个 page。
// 这是 Manager 的唯一 page 创建入口 —— 1 Chrome + N page 的关键。
func NewChromePageFromBrowser(ctx context.Context, b *rod.Browser, cfg *config.BrowserConfig, log logger.Logger) (*ChromePage, error) {
	if b == nil {
		return nil, fmt.Errorf("browser 不能为空")
	}
	return newChromePageOn(ctx, b, cfg, log)
}

func newChromePageOn(ctx context.Context, b *rod.Browser, cfg *config.BrowserConfig, log logger.Logger) (*ChromePage, error) {
	page, err := b.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("创建 page 失败: %w", err)
	}
	// 注意：不要把 page 绑定到调用方的 ctx。Manager 会跨多次执行复用同一
	// page，绑定执行 ctx 会导致执行结束（ctx cancel）后 page 连带失效
	// （Close 也报 context canceled，池化复用实际报废）。
	_ = ctx

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  cfg.ViewportWidth,
		Height: cfg.ViewportHeight,
	}); err != nil {
		log.Warn("设置视口大小失败", logger.Err(err))
	}

	defaultTimeout := cfg.Timeout
	if cfg.NavigationTimeout > defaultTimeout {
		defaultTimeout = cfg.NavigationTimeout
	}
	page = page.Timeout(defaultTimeout)

	cp := &ChromePage{
		browser:  b,
		page:     page,
		mainPage: page,
		logger:   log,
		config:   cfg,
	}
	cp.snapshotTargets()
	log.Info("Chrome 页面创建成功",
		logger.String("headless", fmt.Sprintf("%v", cfg.Headless)))
	return cp, nil
}

// Close 只关闭当前 page，绝不关闭浏览器（Phase 154 / ENGINE-01 硬语义）。
// 仅当 ownsBrowser=true（NewChromePage 独立路径）时才连带关闭 Chrome。
func (p *ChromePage) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	p.frame = nil

	p.logger.Debug("关闭页面（只关 page，保留 Chrome 进程）")
	var firstErr error
	if p.page != nil {
		if err := p.page.Close(); err != nil {
			p.logger.Warn("关闭页面失败", logger.Err(err))
			firstErr = err
		}
	}
	if p.ownsBrowser && p.browser != nil {
		if err := p.browser.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Closed 报告 page 是否已关闭。
func (p *ChromePage) Closed() bool { return p.closed }

// Reset 清理一次执行的瞬态状态：退出 iframe、关闭本次执行新开的页签、
// 页签上下文回到根页签，并重拍页签快照。
//
// 「关闭本次执行新开的页签」是池化复用的必要条件：switchTab 打开的新页签
// 若不复位，会在 Chrome 里累积成孤儿 target，最终把并发页面数吃满
// （ENGINE-05 实测：连续执行会卡在 Acquire）。
func (p *ChromePage) Reset() {
	p.frame = nil
	if p.mainPage != nil {
		p.page = p.mainPage
	}

	// 关闭本次执行期间新开的、且不属于根页签的 target。
	if pages, err := p.browser.Pages(); err == nil {
		mainID := ""
		if p.mainPage != nil {
			mainID = string(p.mainPage.TargetID)
		}
		for _, pg := range pages {
			id := string(pg.TargetID)
			if id == mainID {
				continue
			}
			if p.baseTargets != nil && p.baseTargets[id] {
				continue // 借出前就存在：属于池里其它 page
			}
			_ = pg.Close()
		}
	}

	// 借出时若根页签已被 close 动作销毁，mainPage 指向死 target：
	// 用实时列表里第一个可用页签兜底，避免后续动作全部打空。
	if p.page == nil || !targetAlive(p.page) {
		if pages, err := p.browser.Pages(); err == nil && len(pages) > 0 {
			p.mainPage = pages[0]
			p.page = pages[0]
		}
	}
	p.snapshotTargets()
}

// targetAlive 判定 target 是否仍然有效。
func targetAlive(pg *rod.Page) bool {
	if pg == nil {
		return false
	}
	_, err := pg.Info()
	return err == nil
}

// snapshotTargets 记录当前浏览器内全部页签 ID。
func (p *ChromePage) snapshotTargets() {
	p.baseTargets = map[string]bool{}
	pages, err := p.browser.Pages()
	if err != nil {
		return
	}
	for _, pg := range pages {
		p.baseTargets[string(pg.TargetID)] = true
	}
}

// RawPage 返回当前活动页签的 rod page（screencast / 高级 CDP 出口，155 依赖）。
func (p *ChromePage) RawPage() *rod.Page { return p.page }

// ContextPage 当前操作上下文（iframe 内为 frame 句柄）。
func (p *ChromePage) ContextPage() *rod.Page { return p.cur() }

// Browser 返回共享 rod 浏览器。
func (p *ChromePage) Browser() *rod.Browser { return p.browser }

// cur 返回当前有效的操作句柄（iframe 内则返回 frame 句柄）。
func (p *ChromePage) cur() *rod.Page {
	if p.frame != nil {
		return p.frame
	}
	return p.page
}

// Goto 导航到 URL
func (p *ChromePage) Goto(url string) error {
	p.logger.Debug("导航到", logger.String("url", url))

	p.frame = nil // 导航回主文档，旧 frame 句柄失效

	navPage := p.page.Timeout(p.config.NavigationTimeout)
	if err := navPage.Navigate(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}
	// WaitLoad 等待 load 事件；失败不致命（部分页面本就无 load 完成信号）。
	if err := navPage.WaitLoad(); err != nil {
		p.logger.Debug("等待 load 事件超时（继续）", logger.Err(err))
	}

	waitIdleTimeout := p.config.Timeout
	if waitIdleTimeout > p.config.NavigationTimeout/2 {
		waitIdleTimeout = p.config.NavigationTimeout / 2
	}
	if err := navPage.WaitIdle(waitIdleTimeout); err != nil {
		p.logger.Warn("等待网络空闲超时（继续执行）", logger.Err(err))
	}

	if _, err := navPage.Timeout(p.config.Timeout).Element("body"); err != nil {
		return fmt.Errorf("等待 body 元素超时: %w", err)
	}

	p.logger.Debug("页面加载完成", logger.String("url", url))
	return nil
}

// Click 点击元素
func (p *ChromePage) Click(selector string) error {
	return p.ClickWith(selector, proto.InputMouseButtonLeft, 1)
}

// ClickWith 指定鼠标键与点击次数（1=单击 / 2=双击 / 右键=right）。
func (p *ChromePage) ClickWith(selector string, button proto.InputMouseButton, count int) error {
	p.logger.Debug("点击元素",
		logger.String("selector", selector),
		logger.String("button", string(button)),
		logger.Int("count", count))

	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.Click(button, count); err != nil {
		return fmt.Errorf("点击失败: %w", err)
	}
	return nil
}

// Hover 悬停到元素（触发 mouseenter / CSS :hover）。
func (p *ChromePage) Hover(selector string) error {
	p.logger.Debug("悬停元素", logger.String("selector", selector))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.Hover(); err != nil {
		return fmt.Errorf("悬停失败: %w", err)
	}
	return nil
}

// Fill 填写表单
func (p *ChromePage) Fill(selector, value string) error {
	p.logger.Debug("填写表单",
		logger.String("selector", selector),
		logger.String("value", maskValue(value)))

	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	// 覆盖语义：先全选清空再输入（rod 的 Input 是「追加式」输入，直接
	// Input 会让第二次 fill 变成 "L1L2" —— 这是实测过的真实缺陷）。
	if err := el.SelectAllText(); err != nil {
		p.logger.Debug("全选清空失败（继续按追加输入）", logger.Err(err))
	}
	if err := el.Input(value); err != nil {
		return fmt.Errorf("填写失败: %w", err)
	}
	return nil
}

// Upload 用 Element.SetFiles 设置文件输入。
func (p *ChromePage) Upload(selector string, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("upload 需要至少一个文件路径")
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("上传文件不存在: %s: %w", f, err)
		}
	}
	p.logger.Debug("上传文件",
		logger.String("selector", selector),
		logger.Int("files", len(files)))

	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.SetFiles(files); err != nil {
		return fmt.Errorf("设置文件失败: %w", err)
	}
	return nil
}

// Select 选择下拉框
func (p *ChromePage) Select(selector, value string) error {
	p.logger.Debug("选择下拉框",
		logger.String("selector", selector),
		logger.String("value", value))

	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	// 优先按 option value 选；失败再按显示文本选（兼容业务下拉）。
	// 用 JS 做双匹配：rod 的 Element.Select 只支持单一匹配模式，
	// 而业务下拉常出现 value 与 text 混用。
	script := `function(want) {
		const opts = Array.from(this.options || []);
		const byValue = opts.find(o => o.value === want);
		const byText = opts.find(o => (o.textContent || '').trim() === want);
		const target = byValue || byText;
		if (!target) return false;
		el.value = target.value;
		el.dispatchEvent(new Event('input', {bubbles: true}));
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return true;
	}`
	res, err := el.Eval(script, value)
	if err != nil {
		return fmt.Errorf("选择失败: %w", err)
	}
	if !res.Value.Bool() {
		return fmt.Errorf("选择失败: 下拉框中找不到 value 或文本 %q 的选项", value)
	}
	return nil
}

// WaitFor 等待元素出现
func (p *ChromePage) WaitFor(selector string, timeout time.Duration) error {
	p.logger.Debug("等待元素",
		logger.String("selector", selector),
		logger.Duration("timeout", timeout))

	if _, err := p.cur().Timeout(timeout).Element(selector); err != nil {
		return fmt.Errorf("等待元素超时: %w", err)
	}
	return nil
}

// Screenshot 截图
func (p *ChromePage) Screenshot() ([]byte, error) {
	data, err := p.cur().Screenshot(true, nil)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}
	return data, nil
}

// Evaluate 执行 JavaScript。
//
// rod 的 Page.Eval 会把入参当作「函数定义」并 .apply(...)，因此裸表达式
// （如 "1+1"、"window.location.href"）会报 "apply is not a function"。
// 本方法做兼容：先按函数形态尝试；失败则包成 IIFE 重试，使录制/AI 生成的
// 两种写法都能执行（ENGINE-02 evaluate 的可用性底线）。
func (p *ChromePage) Evaluate(script string) (interface{}, error) {
	p.logger.Debug("执行JavaScript")

	result, err := p.cur().Eval(script)
	if err != nil {
		wrapped := fmt.Sprintf(`() => { return (%s); }`, strings.TrimSpace(script))
		result2, err2 := p.cur().Eval(wrapped)
		if err2 != nil {
			return nil, fmt.Errorf("执行脚本失败: %w", err)
		}
		return result2.Value.Val(), nil
	}
	return result.Value.Val(), nil
}

// GetText 获取元素文本
func (p *ChromePage) GetText(selector string) (string, error) {
	p.logger.Debug("获取文本", logger.String("selector", selector))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	text, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("获取文本失败: %w", err)
	}
	return text, nil
}

// GetValue 获取输入框的值
func (p *ChromePage) GetValue(selector string) (string, error) {
	p.logger.Debug("获取值", logger.String("selector", selector))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	res, err := el.Eval(`function() { return this.value ?? '' }`)
	if err != nil {
		return "", fmt.Errorf("获取值失败: %w", err)
	}
	return res.Value.String(), nil
}

// GetHTML 获取元素HTML
func (p *ChromePage) GetHTML(selector string) (string, error) {
	p.logger.Debug("获取HTML", logger.String("selector", selector))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	html, err := el.HTML()
	if err != nil {
		return "", fmt.Errorf("获取HTML失败: %w", err)
	}
	return html, nil
}

// GetAttribute 获取元素属性
func (p *ChromePage) GetAttribute(selector, attrName string) (string, error) {
	p.logger.Debug("获取属性",
		logger.String("selector", selector),
		logger.String("attribute", attrName))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	attr, err := el.Attribute(attrName)
	if err != nil {
		return "", fmt.Errorf("获取属性失败: %w", err)
	}
	if attr == nil {
		return "", nil
	}
	return *attr, nil
}

// ScrollDown 向下滚动
func (p *ChromePage) ScrollDown(pixels int) error { return p.scrollBy(0, pixels) }

// ScrollUp 向上滚动
func (p *ChromePage) ScrollUp(pixels int) error { return p.scrollBy(0, -pixels) }

func (p *ChromePage) scrollBy(x, y int) error {
	script := fmt.Sprintf("window.scrollBy(%d, %d)", x, y)
	if _, err := p.cur().Eval(script); err != nil {
		return fmt.Errorf("滚动失败: %w", err)
	}
	return nil
}

// ScrollInElement 在元素内部滚动（ENGINE-02 的 scroll 元素内语义）。
func (p *ChromePage) ScrollInElement(selector string, deltaX, deltaY int) error {
	p.logger.Debug("元素内滚动",
		logger.String("selector", selector),
		logger.Int("dx", deltaX),
		logger.Int("dy", deltaY))

	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if _, err := el.Eval(`function(dx, dy) { this.scrollBy(dx, dy); return true; }`, deltaX, deltaY); err != nil {
		return fmt.Errorf("元素内滚动失败: %w", err)
	}
	return nil
}

// ScrollToElement 滚动到元素
func (p *ChromePage) ScrollToElement(selector string) error {
	p.logger.Debug("滚动到元素", logger.String("selector", selector))
	el, err := p.cur().Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找元素失败: %s, 错误: %w", selector, err)
	}
	if err := el.ScrollIntoView(); err != nil {
		return fmt.Errorf("滚动到元素失败: %w", err)
	}
	return nil
}

// Drag 拖拽：先走 CDP Input.dispatchMouseEvent 序列（原生 drag 事件链），
// 失败再降级为 HTML5 DataTransfer 的 JS 实现（适配自研内网页常见的
// dragstart/dragover/drop 手写实现）。
func (p *ChromePage) Drag(fromSelector, toSelector string) error {
	p.logger.Debug("拖拽",
		logger.String("from", fromSelector),
		logger.String("to", toSelector))

	from, err := p.cur().Timeout(p.config.Timeout).Element(fromSelector)
	if err != nil {
		return fmt.Errorf("查找拖拽源失败: %s: %w", fromSelector, err)
	}
	to, err := p.cur().Timeout(p.config.Timeout).Element(toSelector)
	if err != nil {
		return fmt.Errorf("查找拖拽目标失败: %s: %w", toSelector, err)
	}

	if err := p.dragByInput(from, to); err == nil {
		return nil
	} else {
		p.logger.Debug("原生拖拽失败，降级 JS DataTransfer", logger.Err(err))
	}

	if err := p.dragByJS(from, to); err != nil {
		return fmt.Errorf("拖拽失败（原生与 JS 降级均失败）: %w", err)
	}
	return nil
}

// dragByInput CDP 鼠标序列：pressed → 多步 moved → released。
func (p *ChromePage) dragByInput(from, to *rod.Element) error {
	fp, err := from.Interactable()
	if err != nil {
		return err
	}
	tp, err := to.Interactable()
	if err != nil {
		return err
	}

	page := p.cur()
	buttons := 1

	// 先把指针移到源点，避免 from/to 之间残留的 hover 状态干扰。
	if err := (proto.InputDispatchMouseEvent{
		Type:   proto.InputDispatchMouseEventTypeMouseMoved,
		X:      fp.X,
		Y:      fp.Y,
		Button: proto.InputMouseButtonNone,
	}).Call(page); err != nil {
		return err
	}
	if err := (proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMousePressed,
		X:       fp.X,
		Y:       fp.Y,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &buttons,
	}).Call(page); err != nil {
		return err
	}

	// 逐步移动，触发 dragover（一次性跳跃在部分实现里不触发）。
	const steps = 8
	for i := 1; i <= steps; i++ {
		x := fp.X + (tp.X-fp.X)*float64(i)/steps
		y := fp.Y + (tp.Y-fp.Y)*float64(i)/steps
		if err := (proto.InputDispatchMouseEvent{
			Type:    proto.InputDispatchMouseEventTypeMouseMoved,
			X:       x,
			Y:       y,
			Button:  proto.InputMouseButtonLeft,
			Buttons: &buttons,
		}).Call(page); err != nil {
			return err
		}
	}

	if err := (proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMouseReleased,
		X:       tp.X,
		Y:       tp.Y,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &buttons,
	}).Call(page); err != nil {
		return err
	}
	return nil
}

// dragByJS HTML5 DataTransfer 降级：手工构造 dragstart/dragover/drop。
func (p *ChromePage) dragByJS(from, to *rod.Element) error {
	script := `function(src, dst) {
		const dt = new DataTransfer();
		src.dispatchEvent(new DragEvent('dragstart', {bubbles: true, cancelable: true, dataTransfer: dt}));
		dst.dispatchEvent(new DragEvent('dragenter', {bubbles: true, cancelable: true, dataTransfer: dt}));
		dst.dispatchEvent(new DragEvent('dragover', {bubbles: true, cancelable: true, dataTransfer: dt}));
		dst.dispatchEvent(new DragEvent('drop', {bubbles: true, cancelable: true, dataTransfer: dt}));
		src.dispatchEvent(new DragEvent('dragend', {bubbles: true, cancelable: true, dataTransfer: dt}));
		return true;
	}`
	// rod 只把 *proto.RuntimeRemoteObject 当远程对象传参；*rod.Element
	// 会被序列化成普通 JSON，导致 src.dispatchEvent 不存在。必须传 Object。
	res, err := p.cur().Eval(script, from.Object, to.Object)
	if err != nil {
		return err
	}
	if !res.Value.Bool() {
		return fmt.Errorf("JS 拖拽返回 false")
	}
	return nil
}

// PressKeys 组合键：支持 "ctrl+shift+a" / "Enter" / "Control+c" 等形式。
func (p *ChromePage) PressKeys(keys string) error {
	p.logger.Debug("按键", logger.String("keys", keys))

	tokens, err := parseKeyCombo(keys)
	if err != nil {
		return err
	}
	ka := p.cur().KeyActions()
	for _, k := range tokens {
		ka.Press(k)
	}
	if err := ka.Do(); err != nil {
		return fmt.Errorf("按键失败: %w", err)
	}
	return nil
}

// keyAliases 组合键别名 → rod input.Key。
var keyAliases = map[string]input.Key{
	"ctrl": input.ControlLeft, "control": input.ControlLeft,
	"shift": input.ShiftLeft, "alt": input.AltLeft,
	"meta": input.MetaLeft, "cmd": input.MetaLeft, "win": input.MetaLeft,
	"enter": input.Enter, "return": input.Enter,
	"tab": input.Tab, "esc": input.Escape, "escape": input.Escape,
	"backspace": input.Backspace, "delete": input.Delete,
	"space": input.Space, "up": input.ArrowUp, "down": input.ArrowDown,
	"left": input.ArrowLeft, "right": input.ArrowRight,
	"home": input.Home, "end": input.End,
	"pageup": input.PageUp, "pagedown": input.PageDown,
	"f1": input.F1, "f2": input.F2, "f3": input.F3, "f4": input.F4,
	"f5": input.F5, "f6": input.F6, "f7": input.F7, "f8": input.F8,
	"f9": input.F9, "f10": input.F10, "f11": input.F11, "f12": input.F12,
}

// parseKeyCombo 把 "ctrl+shift+a" 解析成 rod key 序列。
func parseKeyCombo(keys string) ([]input.Key, error) {
	parts := strings.Split(keys, "+")
	out := make([]input.Key, 0, len(parts))
	for _, raw := range parts {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			return nil, fmt.Errorf("组合键含空段: %q", keys)
		}
		if k, ok := keyAliases[name]; ok {
			out = append(out, k)
			continue
		}
		// 单字符（含大写字母/数字/符号）直接按其 rune 输入。
		if r := []rune(name); len(r) == 1 {
			out = append(out, input.Key(r[0]))
			continue
		}
		return nil, fmt.Errorf("未知按键: %q", raw)
	}
	return out, nil
}

// switchTarget 按「稳定顺序」解析页签序号。
//
// rod 的 Browser.Pages() 返回顺序不保证跨调用稳定（真实执行里出现过
// 索引漂移，index=1 拿到已被销毁的 target）。这里固定为：
//
//	0 = 当前页签，其余按 TargetID 字典序升序。
//
// 编号一旦确定，同一时刻的多次调用结果一致，脚本可预测。
func (p *ChromePage) switchTarget(index int) (*rod.Page, error) {
	pages, err := p.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("列出页签失败: %w", err)
	}
	// 排序规则（对脚本可预测）：
	//  0 = 当前页签
	//  1..k = 本次执行期间新开的页签（不在借出快照里），按 TargetID 升序
	//  k+1.. = 其它既有页签（池里属于其它任务或空闲页），按 TargetID 升序
	newTabs := make([]*rod.Page, 0, len(pages))
	others := make([]*rod.Page, 0, len(pages))
	curID := string(p.page.TargetID)
	for _, pg := range pages {
		if string(pg.TargetID) == curID {
			continue
		}
		if p.baseTargets != nil && !p.baseTargets[string(pg.TargetID)] {
			newTabs = append(newTabs, pg)
			continue
		}
		others = append(others, pg)
	}
	sort.Slice(newTabs, func(i, j int) bool {
		return string(newTabs[i].TargetID) < string(newTabs[j].TargetID)
	})
	sort.Slice(others, func(i, j int) bool {
		return string(others[i].TargetID) < string(others[j].TargetID)
	})
	ordered := append([]*rod.Page{p.page}, append(newTabs, others...)...)
	if index < 0 || index >= len(ordered) {
		return nil, fmt.Errorf("页签序号越界: %d（共 %d 个）", index, len(ordered))
	}
	return ordered[index], nil
}

// SwitchTab 按页签序号切换（0 = 当前页签，其余按 TargetID 升序）。
func (p *ChromePage) SwitchTab(index int) error {
	p.logger.Debug("切换页签", logger.Int("index", index))

	target, err := p.switchTarget(index)
	if err != nil {
		return err
	}
	if _, err := target.Activate(); err != nil {
		return fmt.Errorf("激活页签失败: %w", err)
	}
	// 新开的页签常见「先 about:blank、再导航到目标 URL」两段式：激活后
	// 必须等它 load 完成，否则紧接着的断言会打在空白页上（ENGINE-05 实测）。
	if err := target.Timeout(p.config.NavigationTimeout).WaitLoad(); err != nil {
		p.logger.Debug("等待新页签 load 超时（继续）", logger.Err(err))
	}
	p.page = target
	p.frame = nil
	return nil
}

// EnterFrame 进入 iframe（按 frame 元素选择器）。
func (p *ChromePage) EnterFrame(selector string) error {
	p.logger.Debug("进入 iframe", logger.String("selector", selector))
	if p.frame != nil {
		return fmt.Errorf("已在 iframe 内，请先 exitFrame")
	}
	el, err := p.page.Timeout(p.config.Timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("查找 iframe 失败: %s, 错误: %w", selector, err)
	}
	frame, err := el.Frame()
	if err != nil {
		return fmt.Errorf("进入 iframe 失败: %w", err)
	}
	p.frame = frame.Timeout(p.config.Timeout)
	return nil
}

// ExitFrame 退出 iframe 回到主文档。
func (p *ChromePage) ExitFrame() error {
	if p.frame == nil {
		return fmt.Errorf("当前不在 iframe 内")
	}
	p.frame = nil
	return nil
}

// Download 触发下载并归档产物。
//
// 实现：Browser.WaitDownload(dir) 返回一次性提交函数，它内部用 EachEvent
// 等待 Page.downloadProgress 到达 completed；我们先跑触发动作（通常是
// Click），再调用提交函数阻塞等到完成。
func (p *ChromePage) Download(ctx context.Context, dir string, trigger func() error) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("download 需要目录参数")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建下载目录失败: %w", err)
	}

	done := p.browser.WaitDownload(dir)

	if trigger != nil {
		if err := trigger(); err != nil {
			return "", err
		}
	}

	type result struct {
		info *proto.PageDownloadWillBegin
	}
	ch := make(chan result, 1)
	go func() { ch <- result{info: done()} }()

	select {
	case r := <-ch:
		if r.info == nil {
			return "", fmt.Errorf("下载未开始（触发动作可能未产生下载）")
		}
		// AllowAndName 模式下落盘文件名是 GUID，归档为建议文件名。
		src := filepath.Join(dir, r.info.GUID)
		dst := filepath.Join(dir, r.info.SuggestedFilename)
		if r.info.SuggestedFilename == "" {
			dst = src
		}
		if _, err := os.Stat(src); err == nil && src != dst {
			if err := os.Rename(src, dst); err != nil {
				// 改名失败不算致命：仍返回 GUID 路径。
				return src, nil
			}
		} else if _, err := os.Stat(dst); err != nil {
			return src, nil
		}
		return dst, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// CloseCurrentTab 关闭当前页签。
//
// 语义（ENGINE-02 close）：
//  1. 浏览器内还有其它页签 → 关闭当前页签并激活第一个剩余页签；
//  2. 这是最后一个页签 → 关闭后新建空白页继续承载执行（避免后续动作全失败）。
//
// 注意：本方法关的是「页签」，不是本次执行借出的 Page 句柄；Manager 的
// Release 仍能正常回收（page 未标记 closed）。
func (p *ChromePage) CloseCurrentTab() error {
	p.logger.Debug("关闭当前页签")

	closing := p.cur()
	if err := closing.Close(); err != nil {
		return fmt.Errorf("关闭页签失败: %w", err)
	}

	// 关掉后重新列举，而不是复用关闭前的快照：Chrome 的 target 销毁是
	// 异步的，快照里的其它 target 也可能已失效（实测 Activate 报
	// "No target with given id found"）。这里以「关闭后的实时列表」为准。
	live, listErr := p.browser.Pages()
	if listErr != nil {
		live = nil
	}
	var remain []*rod.Page
	closingID := string(closing.TargetID)
	for _, pg := range live {
		if string(pg.TargetID) == closingID {
			continue
		}
		if _, err := pg.Info(); err != nil {
			continue // 已失效的 target
		}
		remain = append(remain, pg)
	}

	if len(remain) > 0 {
		sort.Slice(remain, func(i, j int) bool {
			return string(remain[i].TargetID) < string(remain[j].TargetID)
		})
		activated := false
		for _, pg := range remain {
			if _, err := pg.Activate(); err != nil {
				continue
			}
			if closing == p.mainPage {
				p.mainPage = pg
			}
			p.page = pg
			p.frame = nil
			activated = true
			break
		}
		if activated {
			return nil
		}
	}

	// 最后一页保护：新建空白页，保持执行链不断。
	fresh, err := p.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return fmt.Errorf("新建空白页失败: %w", err)
	}
	if err := fresh.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  p.config.ViewportWidth,
		Height: p.config.ViewportHeight,
	}); err != nil {
		p.logger.Warn("设置新页视口失败", logger.Err(err))
	}
	p.page = fresh.Timeout(p.config.Timeout)
	p.mainPage = p.page
	p.frame = nil
	return nil
}

// GetBrowser 获取浏览器实例
func (p *ChromePage) GetBrowser() *rod.Browser { return p.browser }

// GetPage 获取页面实例
func (p *ChromePage) GetPage() *rod.Page { return p.cur() }

// jsonString 辅助：把任意值转成紧凑 JSON（调试用）。
func jsonString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
