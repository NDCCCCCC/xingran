package browser

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// PageManager page manager
//
// ============================================================================
// Phase 154 / ENGINE-03：选择器阶梯与自愈
//
// 元素定位按实证稳定性从高到低依次尝试 7 级：
//
//  1. data-testid              （最高，录制首选）
//  2. role + accessible name   （内网系统无 testid 传统时最可用）
//  3. aria-label
//  4. 稳定属性（id / name，含 id 前缀/后缀容错，应对动态 id）
//  5. 可见文本
//  6. 结构路径（nth-child 链）
//  7. 绝对 XPath               （最后兜底）
//
// 约定（NOCODE-02 的录制器必须遵守）：**录制生成的选择器优先落 testid/role**，
// 只有两者都不存在时才退到 aria-label/属性/文本/路径/XPath。这一约定保证
// 新录制的脚本天然位于阶梯顶端，减少自愈开销。
//
// 命中层级经 recordSelectorHit 写入 SelectorEventCollector（复用
// selector_events.go 的同一信封），供后端选择器学习分析。
// ============================================================================
type PageManager struct {
	page   Page
	logger logger.Logger
	// selectorEvents TIDY-03 配套：选择器明细累积器（契约
	// docs/RPA-选择器上报明细契约.md；nil 安全——旧构造路径零影响）
	selectorEvents *SelectorEventCollector
	// ladderHint 当前阶梯尝试的核心标识（同一 PageManager 单执行单
	// goroutine 使用；仅供 role+name / 文本级的文案匹配）。
	ladderHint string
}

// NewPageManager create page manager
func NewPageManager(page Page, log logger.Logger) *PageManager {
	return &PageManager{
		page:           page,
		logger:         log,
		selectorEvents: NewSelectorEventCollector(),
	}
}

// SelectorEnvelopeLine TIDY-03 配套：执行收尾取契约信封行（无事件返回 ""）。
func (m *PageManager) SelectorEnvelopeLine() string {
	if m == nil || m.selectorEvents == nil {
		return ""
	}
	return m.selectorEvents.EnvelopeLine()
}

// recordSelector 统一埋点（定位方法收尾调用）。
func (m *PageManager) recordSelector(selector string, start time.Time, err error) {
	if m.selectorEvents != nil {
		m.selectorEvents.Record(selector, start, err)
	}
}

// recordSelectorHit 记录命中层级（ENGINE-03：学习分析的输入）。
func (m *PageManager) recordSelectorHit(selector string, tier SelectorTier, start time.Time, err error) {
	if m.selectorEvents == nil {
		return
	}
	m.selectorEvents.RecordWithTier(selector, tier.String(), start, err)
}

// Goto navigate to URL
func (m *PageManager) Goto(url string) error {
	m.logger.Debug("navigate to page", logger.String("url", url))
	if m.selectorEvents != nil {
		m.selectorEvents.ObserveNavigation(url)
	}
	return m.page.Goto(url)
}

// resolveElement 阶梯定位入口（ENGINE-03）。
//
// 所有元素级动作（click/fill/select/get*/hover/drag/upload）都走这里，
// 使「选择器阶梯 + 命中层级记录」在真实执行路径上生效。
func (m *PageManager) resolveElement(selector string) (*rod.Element, SelectorTier, error) {
	return m.findElement(selector)
}

// resolveOrErr 简化内部调用。
func (m *PageManager) resolveOrErr(selector string) (*rod.Element, error) {
	el, _, err := m.resolveElement(selector)
	return el, err
}

// Click click element
func (m *PageManager) Click(selector string) error {
	return m.ClickWith(selector, proto.InputMouseButtonLeft, 1)
}

// ClickWith 指定鼠标键与点击次数（双击 / 右键共用）。
func (m *PageManager) ClickWith(selector string, button proto.InputMouseButton, count int) error {
	m.logger.Debug("click element",
		logger.String("selector", selector),
		logger.String("button", string(button)),
		logger.Int("count", count))
	start := time.Now()

	el, err := m.resolveOrErr(selector)
	if err == nil {
		err = el.Click(button, count)
	}
	m.recordSelector(selector, start, err)
	return err
}

// DoubleClick 双击（Element.Click(left, 2)）。
func (m *PageManager) DoubleClick(selector string) error {
	return m.ClickWith(selector, proto.InputMouseButtonLeft, 2)
}

// RightClick 右键单击。
func (m *PageManager) RightClick(selector string) error {
	return m.ClickWith(selector, proto.InputMouseButtonRight, 1)
}

// Hover 悬停。
func (m *PageManager) Hover(selector string) error {
	m.logger.Debug("hover element", logger.String("selector", selector))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	if err == nil {
		err = el.Hover()
	}
	m.recordSelector(selector, start, err)
	return err
}

// Fill fill form
func (m *PageManager) Fill(selector, value string) error {
	m.logger.Debug("fill form",
		logger.String("selector", selector),
		logger.String("value", maskValue(value)))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	if err == nil {
		// 覆盖语义：先全选清空再输入（rod 的 Input 是追加式，直接 Input
		// 会让第二次 fill 变成 "L1L2" —— ENGINE-05 fixture 实测到）。
		if selErr := el.SelectAllText(); selErr != nil {
			m.logger.Debug("全选清空失败（继续按追加输入）", logger.Err(selErr))
		}
		err = el.Input(value)
	}
	m.recordSelector(selector, start, err)
	return err
}

// Select select dropdown option（先按 value 再按文本）
func (m *PageManager) Select(selector, value string) error {
	m.logger.Debug("select option",
		logger.String("selector", selector),
		logger.String("value", value))
	start := time.Now()

	el, err := m.resolveOrErr(selector)
	if err == nil {
		script := `function(want) {
			const opts = Array.from(this.options || []);
			const target = opts.find(o => o.value === want) ||
			               opts.find(o => (o.textContent || '').trim() === want);
			if (!target) return false;
			this.value = target.value;
			this.dispatchEvent(new Event('input', {bubbles: true}));
			this.dispatchEvent(new Event('change', {bubbles: true}));
			return true;
		}`
		res, evalErr := el.Eval(script, value)
		if evalErr != nil {
			err = evalErr
		} else if !res.Value.Bool() {
			err = fmt.Errorf("下拉框找不到 value 或文本 %q 的选项", value)
		}
	}
	m.recordSelector(selector, start, err)
	return err
}

// Upload 设置文件输入（Element.SetFiles）。
func (m *PageManager) Upload(selector string, files []string) error {
	m.logger.Debug("upload files",
		logger.String("selector", selector),
		logger.Int("files", len(files)))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	if err == nil {
		if len(files) == 0 {
			err = fmt.Errorf("upload 需要至少一个文件路径")
		} else {
			err = el.SetFiles(files)
		}
	}
	m.recordSelector(selector, start, err)
	return err
}

// Drag 拖拽（from/to 都走阶梯）。
func (m *PageManager) Drag(fromSelector, toSelector string) error {
	m.logger.Debug("drag",
		logger.String("from", fromSelector),
		logger.String("to", toSelector))
	start := time.Now()

	from, err := m.resolveOrErr(fromSelector)
	if err != nil {
		m.recordSelector(fromSelector, start, err)
		return err
	}
	to, err := m.resolveOrErr(toSelector)
	if err != nil {
		m.recordSelector(toSelector, start, err)
		return err
	}
	err = m.dragElements(from, to)
	m.recordSelector(fromSelector, start, err)
	return err
}

// dragObserverScript 安装一个 DOM 变更计数器，作为「拖拽是否真的生效」的
// 通用见证（不放宽到页面特定语义，只判定「原生鼠标序列有没有引起 DOM 变化」）。
const dragObserverScript = `(function(){
  window.__rpaDragMutations = 0;
  if (!window.__rpaDragObserver) {
    window.__rpaDragObserver = new MutationObserver(function(ms){
      window.__rpaDragMutations += ms.length;
    });
    window.__rpaDragObserver.observe(document.documentElement, {
      subtree: true, childList: true, attributes: true, characterData: true
    });
  }
  return true;
})()`

const dragMutationCountScript = `window.__rpaDragMutations || 0`

// dragElements 原生 CDP 序列优先，HTML5 DataTransfer JS 兜底。
//
// 「原生失败」的判定不只是 CDP 报错：Chrome 的鼠标事件序列通常不会触发
// HTML5 dragstart/drop 链，但 dispatch 本身会成功返回。因此这里用 DOM
// 变更计数器做效果见证 —— 无变更即视为未生效，降级到 JS DataTransfer。
func (m *PageManager) dragElements(from, to *rod.Element) error {
	if _, err := m.page.Evaluate(dragObserverScript); err != nil {
		m.logger.Debug("安装拖拽变更计数器失败（继续）", logger.Err(err))
	}

	nativeErr := m.dragByInput(from, to)
	nativeEffect := false
	if nativeErr == nil {
		if n, err := m.page.Evaluate(dragMutationCountScript); err == nil {
			nativeEffect = isTruthyNumber(n)
		}
	}
	if nativeErr == nil && nativeEffect {
		return nil
	}
	m.logger.Debug("原生拖拽未生效，降级 JS DataTransfer",
		logger.Err(nativeErr), logger.Any("mutations", nativeEffect))

	if err := m.dragByJS(from, to); err != nil {
		if nativeErr != nil {
			return fmt.Errorf("拖拽失败（原生: %v；JS 降级: %w）", nativeErr, err)
		}
		return fmt.Errorf("拖拽失败（JS 降级）: %w", err)
	}
	return nil
}

// isTruthyNumber 判定 JS 数值结果是否非零（gson Val() 可能是 float64）。
func isTruthyNumber(v interface{}) bool {
	switch n := v.(type) {
	case float64:
		return n > 0
	case int:
		return n > 0
	case int64:
		return n > 0
	case string:
		return n != "" && n != "0"
	default:
		return false
	}
}

func (m *PageManager) dragByInput(from, to *rod.Element) error {
	fp, err := from.Interactable()
	if err != nil {
		return err
	}
	tp, err := to.Interactable()
	if err != nil {
		return err
	}
	page := m.page.ContextPage()
	buttons := 1
	if err := (proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMousePressed,
		X:       fp.X,
		Y:       fp.Y,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &buttons,
	}).Call(page); err != nil {
		return err
	}
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
	return (proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMouseReleased,
		X:       tp.X,
		Y:       tp.Y,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &buttons,
	}).Call(page)
}

func (m *PageManager) dragByJS(from, to *rod.Element) error {
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
	res, err := m.page.ContextPage().Eval(script, from.Object, to.Object)
	if err != nil {
		return err
	}
	if !res.Value.Bool() {
		return fmt.Errorf("JS 拖拽返回 false")
	}
	return nil
}

// ScrollInElement 元素内滚动（scroll 动作的 direction=in）。
func (m *PageManager) ScrollInElement(selector string, deltaX, deltaY int) error {
	m.logger.Debug("scroll in element", logger.String("selector", selector))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	if err == nil {
		_, err = el.Eval(`function(dx, dy) { this.scrollBy(dx, dy); return true; }`, deltaX, deltaY)
	}
	m.recordSelector(selector, start, err)
	return err
}

// WaitFor wait for element（阶梯轮询：每轮完整走 7 级）。
func (m *PageManager) WaitFor(selector string, timeout time.Duration) error {
	m.logger.Debug("wait for element",
		logger.String("selector", selector),
		logger.Duration("timeout", timeout))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()

	lastErr := fmt.Errorf("未开始")
	for {
		if _, _, err := m.resolveElement(selector); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待元素超时（%s）: %w", selector, lastErr)
		case <-ticker.C:
		}
	}
}

// Screenshot take screenshot
func (m *PageManager) Screenshot() ([]byte, error) { return m.page.Screenshot() }

// Evaluate execute JavaScript（走 Page 实现的兼容求值，支持裸表达式）。
func (m *PageManager) Evaluate(script string) (interface{}, error) {
	return m.page.Evaluate(script)
}

// PressKeys 组合键（keyboard 动作）。
func (m *PageManager) PressKeys(keys string) error {
	m.logger.Debug("press keys", logger.String("keys", keys))
	return m.page.PressKeys(keys)
}

// SwitchTab 页签切换。
func (m *PageManager) SwitchTab(index int) error {
	m.logger.Debug("switch tab", logger.Int("index", index))
	return m.page.SwitchTab(index)
}

// EnterFrame 进入 iframe。
func (m *PageManager) EnterFrame(selector string) error {
	start := time.Now()
	err := m.page.EnterFrame(selector)
	m.recordSelector(selector, start, err)
	return err
}

// ExitFrame 退出 iframe。
func (m *PageManager) ExitFrame() error { return m.page.ExitFrame() }

// CloseCurrentTab 关闭当前页签（close 动作）。
func (m *PageManager) CloseCurrentTab() error { return m.page.CloseCurrentTab() }

// Download 触发下载并归档。
func (m *PageManager) Download(ctx context.Context, dir string, trigger func() error) (string, error) {
	return m.page.Download(ctx, dir, trigger)
}

// waitPollInterval 是 WaitForURL / WaitForText / WaitFor 的轮询间隔。
// 抽成包级变量，使单测可缩短轮询以覆盖超时路径（Phase 137 / FIX-08）。
var waitPollInterval = 100 * time.Millisecond

// currentURLScript 取当前页面 URL（供 WaitForURL 轮询校验）。
const currentURLScript = "window.location.href"

// matchURL 判断当前 URL 是否满足 pattern：
// 支持精确匹配与 "*" 通配符（filepath.Match 语义），也兼容子串包含。
func matchURL(current, pattern string) bool {
	if pattern == "" {
		return false
	}
	if ok, err := path.Match(pattern, current); err == nil && ok {
		return true
	}
	return strings.Contains(current, pattern)
}

// pollWait 按 waitPollInterval 轮询 probe，直到 probe 返回 true、ctx 超时或
// probe 返回错误。超时/失败路径返回带上下文的 error（不再首次 tick 即 return nil）。
func pollWait(ctx context.Context, timeout time.Duration, what string, probe func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()

	// 立即做一次探测，避免无条件等待首个 tick。
	if ok, err := probe(); err != nil {
		return fmt.Errorf("wait for %s probe failed: %w", what, err)
	} else if ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %s timeout after %s: %w", what, timeout, ctx.Err())
		case <-ticker.C:
			ok, err := probe()
			if err != nil {
				return fmt.Errorf("wait for %s probe failed: %w", what, err)
			}
			if ok {
				return nil
			}
		}
	}
}

// WaitForURL wait for URL change（Phase 137 / FIX-08：真实轮询校验当前 URL）。
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
	m.logger.Debug("wait for URL",
		logger.String("pattern", pattern),
		logger.Duration("timeout", timeout))
	return pollWait(context.Background(), timeout, "URL("+pattern+")", func() (bool, error) {
		val, err := m.page.Evaluate(currentURLScript)
		if err != nil {
			return false, err
		}
		current, _ := val.(string)
		return matchURL(current, pattern), nil
	})
}

// WaitForText wait for text appearance（Phase 137 / FIX-08：真实轮询校验元素文本）。
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
	m.logger.Debug("wait for text",
		logger.String("selector", selector),
		logger.String("text", text),
		logger.Duration("timeout", timeout))
	script := fmt.Sprintf(`(function(){var el=document.querySelector(%q);return el?el.textContent:"";})()`, selector)
	return pollWait(context.Background(), timeout, "text("+text+")", func() (bool, error) {
		val, err := m.page.Evaluate(script)
		if err != nil {
			return false, err
		}
		got, _ := val.(string)
		return strings.Contains(got, text), nil
	})
}

// maskValue mask sensitive value
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

// GetText get element text content
func (m *PageManager) GetText(selector string) (string, error) {
	m.logger.Debug("get text", logger.String("selector", selector))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	var text string
	if err == nil {
		text, err = el.Text()
	}
	m.recordSelector(selector, start, err)
	return text, err
}

// GetValue get input element value
func (m *PageManager) GetValue(selector string) (string, error) {
	m.logger.Debug("get value", logger.String("selector", selector))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	var value string
	if err == nil {
		res, evalErr := el.Eval(`function() { return this.value ?? '' }`)
		if evalErr != nil {
			err = evalErr
		} else {
			value = res.Value.String()
		}
	}
	m.recordSelector(selector, start, err)
	return value, err
}

// GetHTML get element HTML content
func (m *PageManager) GetHTML(selector string) (string, error) {
	m.logger.Debug("get HTML", logger.String("selector", selector))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	var html string
	if err == nil {
		html, err = el.HTML()
	}
	m.recordSelector(selector, start, err)
	return html, err
}

// GetAttribute get element attribute value
func (m *PageManager) GetAttribute(selector, attrName string) (string, error) {
	m.logger.Debug("get attribute",
		logger.String("selector", selector),
		logger.String("attribute", attrName))
	start := time.Now()
	el, err := m.resolveOrErr(selector)
	var value string
	if err == nil {
		attr, attrErr := el.Attribute(attrName)
		if attrErr != nil {
			err = attrErr
		} else if attr != nil {
			value = *attr
		}
	}
	m.recordSelector(selector, start, err)
	return value, err
}

// ScrollDown scroll page down
func (m *PageManager) ScrollDown(pixels int) error {
	m.logger.Debug("scroll down", logger.Int("pixels", pixels))
	return m.page.ScrollDown(pixels)
}

// ScrollUp scroll page up
func (m *PageManager) ScrollUp(pixels int) error {
	m.logger.Debug("scroll up", logger.Int("pixels", pixels))
	return m.page.ScrollUp(pixels)
}

// ScrollToElement scroll to element
func (m *PageManager) ScrollToElement(selector string) error {
	start := time.Now()
	m.logger.Debug("scroll to element", logger.String("selector", selector))
	el, err := m.resolveOrErr(selector)
	if err == nil {
		err = el.ScrollIntoView()
	}
	m.recordSelector(selector, start, err)
	return err
}
