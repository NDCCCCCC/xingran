package browser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// Phase 156 / NOCODE-02：元素拾取器（Worker 侧探测）
//
// 与 recorder.go 的关系：
//   - 复用同一份 `__rpaDescribe`（7 级阶梯候选计算），保证「录制出来的选择器」
//     与「拾取出来的选择器」是**同一套算法**，不会出现两种口径。
//   - 录制是「随时间累积事件」，拾取是「给定坐标/元素立刻返回阶梯结果」。
//
// 坐标语义（与 LIVEVIEW-02 完全一致）：
//   前端把画布像素换算成 **viewport CSS 像素** 后传过来，Worker 直接透传给
//   `document.elementFromPoint(x, y)` —— 不做二次换算，也不做 DPR 补偿。
//
// 为什么不用 CDP 的 DOM.getNodeForLocation：
//   它返回 nodeId，还要 DOM.describeNode + DOM.resolveNode 才能拿到属性，
//   而阶梯计算本来就在页面 JS 里做（recorder.go 已有一份）。用
//   elementFromPoint 一步到位，且能天然穿透到 shadow DOM 的宿主元素。
// ============================================================================

// PickedElement 一次拾取的结果（前端浮层直接渲染）。
type PickedElement struct {
	// Found 坐标处是否真的有元素。
	Found bool `json:"found"`
	// Tag 元素标签名（小写，仅诊断与前端提示，不直接展示给用户）。
	Tag string `json:"tag"`
	// Type input 的 type（password 等，决定前端提示「敏感字段」）。
	Type string `json:"type"`
	// Text 元素可见文本（截断到 40 字符）。
	Text string `json:"text"`
	// Tiers 7 级阶梯候选及各自可用性（NOCODE-02 浮层要求）。
	Tiers []RecordedTier `json:"tiers"`
	// Recommended 推荐选择器（= 阶梯首个可用级）。
	Recommended string `json:"recommended"`
	// RecommendedTier 推荐选择器所在的阶梯编号（1..7）。
	RecommendedTier int `json:"recommendedTier"`
	// RecommendedScore 推荐选择器的评分（0-100，前端显示「置信度」）。
	RecommendedScore int `json:"recommendedScore"`
	// Frame 元素所在的 iframe 路径（空表示主文档；NOCODE-02 要求 iframe 内也可用）。
	Frame string `json:"frame,omitempty"`
}

// Picker 在单个 page 上做元素拾取。
type Picker struct {
	page   *rod.Page
	logger logger.Logger
}

// NewPicker 创建拾取器。
func NewPicker(page *rod.Page, log logger.Logger) *Picker {
	return &Picker{page: page, logger: log}
}

// ensureRuntime 确保当前文档里有 __rpaDescribe（拾取与录制共用）。
//
// 拾取通常发生在「已经在执行/预览的页面」上，录制注入脚本不一定装过，
// 因此这里按需安装一次（幂等：脚本自己会判 __rpaRec 是否已存在）。
func (p *Picker) ensureRuntime() error {
	if p == nil || p.page == nil {
		return fmt.Errorf("拾取器未绑定页面")
	}
	if _, err := p.page.Eval("() => { " + recorderRuntimeJS + "\n return true; }"); err != nil {
		return fmt.Errorf("安装拾取运行时失败: %w", err)
	}
	return nil
}

// pickAtScript 按坐标拾取元素并返回完整阶梯结果（在页面内一次算完）。
//
// 用裸表达式而非函数：Source/IIFE 形态在注入路径是必须的，而这里是 Eval，
// 统一走箭头函数包装（与 recorder.Start 同款，避免 apply is not a function）。
const pickAtScript = `() => {
  var args = Array.prototype.slice.call(arguments);
  var x = args[0], y = args[1];
  var el = document.elementFromPoint(x, y);
  if (!el) { return {found: false}; }
  var d = window.__rpaDescribe(el);
  if (!d) { return {found: false}; }
  return {
    found: true,
    tag: d.tag,
    type: d.type,
    text: d.text,
    tiers: d.tiers
  };
}`

// PickAt 在 viewport 坐标 (x, y) 处拾取元素并计算 7 级阶梯。
//
// 返回的 PickedElement.Tiers 里每一级都带 OK 标记：前端浮层据此逐级显示
// 「可用 / 不可用」，这正是 NOCODE-02「展示生成的 7 级阶梯结果（每级是否可用）」
// 的数据来源。
func (p *Picker) PickAt(x, y float64) (*PickedElement, error) {
	if err := p.ensureRuntime(); err != nil {
		return nil, err
	}

	res, err := p.page.Eval(pickAtScript, x, y)
	if err != nil {
		return nil, fmt.Errorf("按坐标拾取失败（x=%.0f,y=%.0f）: %w", x, y, err)
	}

	raw := res.Value.Val()
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("拾取结果序列化失败: %w", err)
	}
	var picked PickedElement
	if err := json.Unmarshal(buf, &picked); err != nil {
		return nil, fmt.Errorf("拾取结果解析失败: %w", err)
	}
	if !picked.Found {
		// 坐标落在空白处：不是错误，前端显示「这里没有可拾取的元素」。
		return &picked, nil
	}

	p.decorate(&picked)
	p.logger.Debug("元素拾取成功",
		logger.String("tag", picked.Tag),
		logger.Int("tier", picked.RecommendedTier),
		logger.String("selector", picked.Recommended))
	return &picked, nil
}

// PickBySelector 按选择器拾取（前端「换用次选」后再验证，以及既有动作回填用）。
func (p *Picker) PickBySelector(selector string) (*PickedElement, error) {
	if err := p.ensureRuntime(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(selector) == "" {
		return nil, fmt.Errorf("选择器为空")
	}

	script := `() => {
      var sel = arguments[0];
      var el = null;
      try {
        if (sel.indexOf("text=") === 0) {
          var want = sel.slice(5);
          var all = document.querySelectorAll("[role],button,a,label,input,span,div");
          for (var i = 0; i < all.length; i++) {
            var t = (all[i].innerText || all[i].textContent || "").replace(/\s+/g, " ").trim();
            if (t === want) { el = all[i]; break; }
          }
        } else {
          el = document.querySelector(sel);
        }
      } catch (e) { el = null; }
      if (!el) { return {found: false}; }
      var d = window.__rpaDescribe(el);
      if (!d) { return {found: false}; }
      return {found: true, tag: d.tag, type: d.type, text: d.text, tiers: d.tiers};
    }`

	res, err := p.page.Eval(script, selector)
	if err != nil {
		return nil, fmt.Errorf("按选择器拾取失败（%q）: %w", selector, err)
	}
	buf, err := json.Marshal(res.Value.Val())
	if err != nil {
		return nil, err
	}
	var picked PickedElement
	if err := json.Unmarshal(buf, &picked); err != nil {
		return nil, err
	}
	if picked.Found {
		p.decorate(&picked)
	}
	return &picked, nil
}

// decorate 计算推荐级与评分（按 tier 升序取首个可用）。
//
// 与 recorder.go 的 bestSelector 是同一判定口径 —— 保证「拾取推荐」与
// 「录制生成」不会给出不同答案。
func (p *Picker) decorate(picked *PickedElement) {
	sel, tier := recommendTier(picked.Tiers)
	picked.Recommended = sel
	picked.RecommendedTier = tier
	if tier > 0 {
		picked.RecommendedScore = TierScore(SelectorTier(tier))
	}
	// 阶梯按 tier 升序返回，前端浮层直接按顺序渲染。
	sort.Slice(picked.Tiers, func(i, j int) bool { return picked.Tiers[i].Tier < picked.Tiers[j].Tier })
}

// recommendTier 按阶梯顺序取首个可用级，返回 (选择器, tier 编号)。
func recommendTier(tiers []RecordedTier) (string, int) {
	sorted := make([]RecordedTier, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Tier < sorted[j].Tier })
	for _, t := range sorted {
		if t.OK && strings.TrimSpace(t.Selector) != "" {
			return t.Selector, t.Tier
		}
	}
	return "", 0
}

// PickInFrames 在 iframe 内拾取（NOCODE-02 明确要求 iframe 内也可用）。
//
// 实现：先在主文档按坐标拾取；若命中的元素**本身是 iframe**（用户在画面上
// 点 iframe 区域时 elementFromPoint 会返回 iframe 宿主元素，这是浏览器语义，
// 不是 bug），则深入该 frame 内部继续拾取，并把 frame 的 CSS 选择器路径写进
// 结果的 Frame 字段 —— 回放时引擎需要先 iframe(enter) 再操作。
func (p *Picker) PickInFrames(x, y float64) (*PickedElement, error) {
	// 先在主文档试
	picked, err := p.PickAt(x, y)
	if err != nil {
		return nil, err
	}

	// 主文档命中的是 iframe 宿主 → 必须深入，否则用户拿到的是 iframe 而不是
	// 他真正想点的内层元素（这是「iframe 内也可用」的关键路径）。
	hostIsFrame := picked.Found && (picked.Tag == "iframe" || picked.Tag == "frame")
	if picked.Found && !hostIsFrame {
		return picked, nil
	}

	// 枚举所有 iframe 的矩形（含主文档命中的那个）
	frames, err := p.page.Eval(`() => {
      var out = [];
      var list = document.querySelectorAll("iframe,frame");
      for (var i = 0; i < list.length; i++) {
        var f = list[i];
        var r = f.getBoundingClientRect();
        out.push({index: i, x: r.left, y: r.top, w: r.width, h: r.height,
                  id: f.id || "", name: f.getAttribute("name") || ""});
      }
      return out;
    }`)
	if err != nil {
		return picked, nil
	}
	buf, _ := json.Marshal(frames.Value.Val())
	var rects []struct {
		Index int     `json:"index"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		W     float64 `json:"w"`
		H     float64 `json:"h"`
		ID    string  `json:"id"`
		Name  string  `json:"name"`
	}
	if err := json.Unmarshal(buf, &rects); err != nil {
		return picked, nil
	}

	for _, r := range rects {
		// 坐标是否落在该 iframe 的矩形内（留 1px 容差，避免边界浮点误差）
		if x < r.X-1 || x > r.X+r.W+1 || y < r.Y-1 || y > r.Y+r.H+1 {
			continue
		}
		sel := frameSelector(r.ID, r.Name, r.Index)
		// 必须**真的进入 frame** 才能拾取内层元素：
		// elementFromPoint 只作用于当前文档，在主文档里用它永远只会返回
		// iframe 宿主元素本身（这是上一版的实测缺陷）。
		inner, err := p.pickInsideFrame(sel, x-r.X, y-r.Y)
		if err != nil || inner == nil || !inner.Found {
			continue
		}
		// 内层又命中 iframe（嵌套）→ 保留当前结果，避免无限深入。
		if inner.Tag == "iframe" || inner.Tag == "frame" {
			continue
		}
		inner.Frame = sel
		return inner, nil
	}
	return picked, nil
}

// pickInsideFrame 进入 iframe 内部按相对坐标拾取，取完立刻退回主文档。
//
// 用 rod 的 Element.Frame() 而不是 CDP 的 DOM 域手写穿透：前者已经把
// frame 的执行上下文（ExecutionContextId）处理好，且与引擎的 iframe 动作
// 走同一条路径（154 的 EnterFrame 也是 Element.Frame()），语义一致。
func (p *Picker) pickInsideFrame(frameSel string, x, y float64) (*PickedElement, error) {
	host, err := p.page.Timeout(frameTimeout()).Element(frameSel)
	if err != nil {
		return nil, fmt.Errorf("查找 iframe %q 失败: %w", frameSel, err)
	}
	frame, err := host.Frame()
	if err != nil {
		return nil, fmt.Errorf("进入 iframe %q 失败: %w", frameSel, err)
	}
	// 在 frame 上下文里安装同一份运行时（每个文档独立）
	if _, err := frame.Eval("() => { " + recorderRuntimeJS + "\n return true; }"); err != nil {
		return nil, fmt.Errorf("iframe 内安装拾取运行时失败: %w", err)
	}
	res, err := frame.Eval(pickAtScript, x, y)
	if err != nil {
		return nil, fmt.Errorf("iframe 内按坐标拾取失败: %w", err)
	}
	buf, err := json.Marshal(res.Value.Val())
	if err != nil {
		return nil, err
	}
	var picked PickedElement
	if err := json.Unmarshal(buf, &picked); err != nil {
		return nil, err
	}
	if picked.Found {
		p.decorate(&picked)
	}
	return &picked, nil
}

// frameTimeout iframe 探测超时（headless 下 iframe 加载可能慢于主文档）。
func frameTimeout() time.Duration { return 5 * time.Second }

// frameSelector 生成 iframe 的选择器（优先 id/name，退到序号）。
func frameSelector(id, name string, index int) string {
	if id != "" {
		return "#" + id
	}
	if name != "" {
		return fmt.Sprintf(`iframe[name="%s"]`, name)
	}
	return fmt.Sprintf("iframe:nth-of-type(%d)", index+1)
}
