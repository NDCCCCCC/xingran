package browser

import (
	"context"
	"encoding/json"
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

// clickMouseBudget 真实鼠标点击的预算（失败即退化到页面内触发）。
//
// 取值依据（生产诊断实证）：同一按钮「Eval click」3ms 生效、「rod 鼠标点击」
// 在 30s 预算下耗尽——卡在可交互检测/坐标。正常点击远小于 1s，因此 5s
// 既不会误伤慢机器，也不会让整条任务在白等 30s。
const clickMouseBudget = 5 * time.Second

// interactBudget 键盘输入/悬停等交互元素的预算（失败即退化到页面内触发）。
//
// 与 clickMouseBudget 同口径：正常交互远小于 1s；被遮挡时 rod 会一直等到
// 预算耗尽，5s 让兜底及时接管，又不至于误伤慢机器。
const interactBudget = 5 * time.Second

// ClickWith 指定鼠标键与点击次数（双击 / 右键共用）。
//
// W7.1 / 缺陷 A 同族（第五层）：**真实鼠标点击失败时的 JS 兜底**。
//
// 真跑生产库 6 个 goto 任务时定位到（诊断实证）：
//
//	[A] Eval 直接 el.click()   → 2ms 成功
//	[B] PageManager.Click(#su) → context deadline exceeded（耗尽 30s 预算）
//
// rod 的 Element.Click 走「可交互检测 + 坐标计算 + Input.dispatchMouseEvent」，
// 目标元素被浮层/遮罩覆盖、或在视口外时，它会一直等到超时。此时**页面本身
// 完全正常**（JS 触发立刻生效），失败原因不是脚本配置、也不是我们的归一化。
//
// 兜底语义（保守且显式）：
//   - 只在鼠标路径**失败**时启用，成功后永不替换（不改变既有成功路径的语义）；
//   - 只用于左键单击（右键菜单/双击有独立的语义预期，不做静默替换）；
//   - 必须留 WARN 日志，运维能看到「这一步用的是 JS 触发而非真实鼠标」。
func (m *PageManager) ClickWith(selector string, button proto.InputMouseButton, count int) error {
	m.logger.Debug("click element",
		logger.String("selector", selector),
		logger.String("button", string(button)),
		logger.Int("count", count))
	start := time.Now()

	el, err := m.resolveOrErr(selector)
	if err == nil {
		// 鼠标派发单独给一个**短预算**（不是元素操作总预算 30s）：
		// 命中的元素被浮层遮挡时 rod 会一直等到超时，而退化到页面内触发
		// 只要几毫秒。真跑实证：30s 预算下整步耗时 30.0s；给 5s 预算后
		// 退化接管，整步仍在可接受范围内，且成功路径不受影响（正常点击秒回）。
		err = el.Context(el.GetContext()).Timeout(clickMouseBudget).Click(button, count)
	}
	if err != nil && button == proto.InputMouseButtonLeft && count == 1 {
		// 真实鼠标路径失败 → 退化为页面内 JS 触发（等价「用户点了一下」）。
		jsErr := m.clickByJS(selector)
		if jsErr == nil {
			m.logger.Warn("鼠标点击失败，已退化为页面内触发（元素可能被浮层遮挡）",
				logger.String("selector", selector),
				logger.Err(err))
			m.recordSelector(selector, start, nil)
			return nil
		}
		err = jsErr
	}
	m.recordSelector(selector, start, err)
	return err
}

// waitElementForJS 在页面内短轮询等待元素出现（最多 d，每 200ms 一次）。
//
// 为什么需要（真跑实证）：点击搜索后页面开始导航，紧接着的下一步在**旧文档
// 或加载中**的文档上 querySelector → not-found。这不是「元素不存在」，而是
// 「此刻还没渲染完」。生产库的历史 goto 任务全是「搜索→再搜索」的连续动作，
// 正好踩在这个时序上。
//
// **每次探查都带独立预算**：页面若已假死（执行中途 Chrome 掉线），
// m.page.Evaluate 本身会阻塞到 page 超时（15~30s）；不加预算的话这个
// 5s 等待会被拖成分钟级，把「兜底」变成新的卡点（真跑踩到过）。
func (m *PageManager) waitElementForJS(selector string, d time.Duration) bool {
	selJSON, err := json.Marshal(selector)
	if err != nil {
		return false
	}
	script := fmt.Sprintf(`() => !!(document.querySelector(%s))`, string(selJSON))

	deadline := time.Now().Add(d)
	for {
		ok := m.evaluateBoolWithBudget(script, 1*time.Second)
		if ok {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// evaluateBoolWithBudget 带硬预算的布尔求值（求值本身阻塞也不会超过 budget）。
//
// goroutine 可能短暂滞留到 page 自身 ctx 结束（最多 15~30s），但调用方立即
// 拿到「false」继续走错误路径 —— 对「诊断/兜底」路径这是可接受的取舍。
func (m *PageManager) evaluateBoolWithBudget(script string, budget time.Duration) bool {
	ch := make(chan bool, 1)
	go func() {
		res, err := m.page.Evaluate(script)
		if err != nil {
			ch <- false
			return
		}
		ch <- isTruthyAny(res)
	}()
	select {
	case v := <-ch:
		return v
	case <-time.After(budget):
		return false
	}
}

// isTruthyAny 粗判 JS 求值结果的真值（rod 返回 bool / string / float 等）。
func isTruthyAny(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != "" && x != "false" && x != "0"
	case float64:
		return x != 0
	}
	return false
}

// clickByJS 在页面内直接触发一次点击（不派发真实鼠标事件）。
//
// 用于「元素被遮挡/不可交互，但页面逻辑本身正常」的场景 —— 生产实证：同一
// 元素 Eval click 2ms 生效、rod 鼠标点击 30s 超时。失败时返回可读错误
// （找不到元素 / 脚本异常），交由上层按原有错误路径处理。
func (m *PageManager) clickByJS(selector string) error {
	// 先等元素出现（最多 clickJSWaitBudget）：上一步点击触发导航时，
	// 本步可能落在旧文档/加载中的文档上。
	if !m.waitElementForJS(selector, clickJSWaitBudget) {
		return fmt.Errorf("页面内触发点击失败：找不到元素 %s", selector)
	}
	selJSON, err := json.Marshal(selector)
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`() => {
		const el = document.querySelector(%s);
		if (!el) return "not-found";
		el.click();
		return "clicked";
	}`, string(selJSON))
	res, evalErr := m.page.Evaluate(script)
	if evalErr != nil {
		return fmt.Errorf("页面内触发点击失败: %w", evalErr)
	}
	if res == "not-found" {
		return fmt.Errorf("页面内触发点击失败：找不到元素 %s", selector)
	}
	return nil
}

// clickJSWaitBudget JS 兜底路径等待元素出现的预算。
//
// 与「找元素」的语义区分：真实交互路径靠 rod 的 Element 等待（预算充足）；
// JS 兜底是「页面逻辑本来就能跑」的快速路径，给 5s 足够等一次导航完成，
// 又不会把配置错误（选择器真的过期）拖成几十秒。
const clickJSWaitBudget = 5 * time.Second

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
//
// W7.1 / 缺陷 A 同族（第五层续）：与 Click 同源的**交互层兜底**。
//
// 真跑生产库 goto 任务时定位到：搜索结果页的 #kw 输入框能被阶梯命中，但
// rod 的 Input（可交互检测 + 焦点 + 键盘派发）挂到 30s 超时；同一元素用
// 页面内脚本设值 + 派发 input/change 立刻生效。这不是脚本配置错误，也不是
// 我们的归一化 —— 是 rod 交互层在「元素被浮层遮挡/在视口外」时的行为。
//
// 兜底语义与 Click 一致：只在鼠标/键盘路径失败时启用，成功路径完全不改；
// 必须留 WARN 日志。
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
		err = el.Context(el.GetContext()).Timeout(interactBudget).Input(value)
	}
	if err != nil {
		// 退化：页面内设值 + 派发 input/change（等价「用户输完并触发校验」）。
		jsErr := m.setValueByJS(selector, value)
		if jsErr == nil {
			m.logger.Warn("键盘输入失败，已退化为页面内设值（元素可能被浮层遮挡）",
				logger.String("selector", selector), logger.Err(err))
			m.recordSelector(selector, start, nil)
			return nil
		}
		err = jsErr
	}
	m.recordSelector(selector, start, err)
	return err
}

// setValueByJS 页面内设值并派发事件（不模拟键盘）。
//
// 只用于「元素被遮挡导致 Input 挂住」的兜底；正常路径仍走真实键盘输入，
// 以保证需要真实键入事件的站点行为不被改变。
func (m *PageManager) setValueByJS(selector, value string) error {
	if !m.waitElementForJS(selector, clickJSWaitBudget) {
		return fmt.Errorf("页面内设值失败：找不到元素 %s", selector)
	}
	selJSON, err := json.Marshal(selector)
	if err != nil {
		return err
	}
	valJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`() => {
		const el = document.querySelector(%s);
		if (!el) return "not-found";
		try { el.focus(); } catch (e) {}
		el.value = %s;
		el.dispatchEvent(new Event("input", { bubbles: true }));
		el.dispatchEvent(new Event("change", { bubbles: true }));
		return "set";
	}`, string(selJSON), string(valJSON))
	res, evalErr := m.page.Evaluate(script)
	if evalErr != nil {
		return fmt.Errorf("页面内设值失败: %w", evalErr)
	}
	if res == "not-found" {
		return fmt.Errorf("页面内设值失败：找不到元素 %s", selector)
	}
	return nil
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
