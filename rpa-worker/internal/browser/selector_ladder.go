package browser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// ============================================================================
// Phase 154 / ENGINE-03：选择器阶梯
//
// 顺序（实证稳定性由高到低）：
//   data-testid > role+name > aria-label > 稳定属性(id/name) >
//   可见文本 > 结构路径 > 绝对 XPath
//
// 关键实现约束：
//   - 每一级都用「零等待探测」原语（Has / HasR / Search + 短超时 / HasX），
//     避免某一级耗光整段等待预算导致阶梯本身变成性能黑洞；
//   - 全部 7 级失败才返回 *LadderError，错误里逐级回显候选与原因；
//   - 命中层级写入 SelectorEventCollector（selector_events.RecordWithTier）。
// ============================================================================

// SelectorTier 选择器阶梯层级（编号越小越稳定）。
type SelectorTier int

const (
	TierTestID        SelectorTier = iota + 1 // data-testid
	TierRoleName                              // role + accessible name
	TierAriaLabel                             // aria-label
	TierStableAttr                            // 稳定属性 id / name
	TierVisibleText                           // 可见文本
	TierStructurePath                         // 结构路径 nth-child 链
	TierXPath                                 // 绝对 XPath
)

// tierNames 层级标识（写入选择器信封的 tier 维度）。
var tierNames = map[SelectorTier]string{
	TierTestID:        "data-testid",
	TierRoleName:      "role+name",
	TierAriaLabel:     "aria-label",
	TierStableAttr:    "stable-attr",
	TierVisibleText:   "visible-text",
	TierStructurePath: "structure-path",
	TierXPath:         "xpath",
}

// String 稳定标识。
func (t SelectorTier) String() string {
	if n, ok := tierNames[t]; ok {
		return n
	}
	return fmt.Sprintf("tier-%d", int(t))
}

// tierScores 阶梯评分（0–100，必须单调递减）。
// ladder_score_test.go 静态断言单调性，防止有人调换阶梯顺序。
var tierScores = map[SelectorTier]int{
	TierTestID:        100,
	TierRoleName:      90,
	TierAriaLabel:     80,
	TierStableAttr:    65,
	TierVisibleText:   45,
	TierStructurePath: 20,
	TierXPath:         5,
}

// TierScore 返回层级评分（0 表示未知层级）。
func TierScore(t SelectorTier) int { return tierScores[t] }

// tierProbeTimeout 单级探测预算。真实执行时 7 级 × 500ms = 3.5s 上限，
// 远小于 page 默认 30s 超时；单测可临时改小。
var tierProbeTimeout = 500 * time.Millisecond

// defaultElementOpTimeout 命中元素后的操作预算（W7.1 / 缺陷 A 第四层）。
//
// 探测用的 tierProbeTimeout 只该约束「找元素」，绝不能约束「用元素」——
// 否则一次点击/输入只要耗时超过 500ms 就报 context deadline exceeded。
// 这里给 30s（与 worker 默认元素超时同量级），调用方显式传 timeout 时优先用传值。
const defaultElementOpTimeout = 30 * time.Second

// ladderAttempt 单级尝试结果（错误信息里逐级回显）。
type ladderAttempt struct {
	Tier     SelectorTier
	Selector string
	Err      error
}

// LadderError 全部 7 级失败时的聚合错误。
type LadderError struct {
	Input    string
	Attempts []ladderAttempt
}

func (e *LadderError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "选择器阶梯全部失败（输入 %q，共 %d 级）:", e.Input, len(e.Attempts))
	for _, a := range e.Attempts {
		fmt.Fprintf(&b, " [%s %q", a.Tier.String(), a.Selector)
		if a.Err != nil {
			fmt.Fprintf(&b, ": %v", a.Err)
		}
		b.WriteString("]")
	}
	return b.String()
}

// selectorHints 从原始选择器提取可用于逐级降级的语义线索。
type selectorHints struct {
	raw     string
	hint    string // 核心标识（无 # . [ ] 引号，无 text= 前缀）
	isXPath bool
	isCSS   bool
	isPlain bool // 纯文本/自然语言（可作为可见文本级候选）
}

var (
	testidAttrRe = regexp.MustCompile(`(?i)\[data-testid\s*=\s*["']?([^"'\]]+)`)
	ariaAttrRe   = regexp.MustCompile(`(?i)\[aria-label\s*=\s*["']?([^"'\]]+)`)
	nameAttrRe   = regexp.MustCompile(`(?i)\[name\s*=\s*["']?([^"'\]]+)`)
	idAttrRe     = regexp.MustCompile(`#([A-Za-z0-9_\-:.]+)`)
	classAttrRe  = regexp.MustCompile(`\.([A-Za-z0-9_\-]+)`)
	// trailingDigits 去掉末尾的动态序号（user-row-42 → user-row）。
	trailingDigits = regexp.MustCompile(`[-_]?\d+$`)
	// naturalText 判断是否像自然语言/中文标签（可走可见文本级）。
	naturalText = regexp.MustCompile(`[\p{Han}]|\s`)
	// xpathTextRe 抽取 XPath 里的文本字面量。
	xpathTextRe = regexp.MustCompile(`(?i)contains\(\s*(?:text\(\)|@[\w-]+)\s*,\s*["']([^"']+)`)
)

// parseSelector 解析选择器形态并提取核心标识。
func parseSelector(selector string) selectorHints {
	s := strings.TrimSpace(selector)
	h := selectorHints{raw: s, hint: s}

	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "xpath="):
		h.isXPath = true
		h.hint = strings.TrimSpace(s[len("xpath="):])
		return h
	case strings.HasPrefix(lower, "css="):
		h.isCSS = true
		h.hint = strings.TrimSpace(s[len("css="):])
		return h
	case strings.HasPrefix(lower, "text="):
		h.isPlain = true
		h.hint = strings.TrimSpace(s[len("text="):])
		return h
	case strings.HasPrefix(s, "/") || strings.HasPrefix(s, "("):
		h.isXPath = true
		// xpath: 抽取文本字面量作为 hint
		if m := xpathTextRe.FindStringSubmatch(s); len(m) == 2 {
			h.hint = m[1]
		}
		return h
	}

	// 属性语义优先（testid / aria-label / name）。
	for _, pair := range []struct {
		re *regexp.Regexp
	}{{testidAttrRe}, {ariaAttrRe}, {nameAttrRe}} {
		if m := pair.re.FindStringSubmatch(s); len(m) == 2 {
			h.hint = m[1]
			h.isCSS = true
			return h
		}
	}

	if m := idAttrRe.FindStringSubmatch(s); len(m) == 2 {
		h.hint = m[1]
		h.isCSS = true
		return h
	}

	// 纯 CSS 但只有 class/tag：没有稳定语义线索，hint 置空（只允许原始
	// 选择器在 stable-attr 级直接尝试）。
	if strings.HasPrefix(s, ".") || classAttrRe.MatchString(s) || strings.ContainsAny(s, ">[]: ") {
		if m := classAttrRe.FindStringSubmatch(s); len(m) == 2 && !strings.ContainsAny(s, "> [") {
			h.hint = m[1]
		} else {
			h.hint = ""
		}
		h.isCSS = true
		return h
	}

	// 其余按「纯文本」处理（录制器常直接给出可见文本）。
	if naturalText.MatchString(s) || !strings.ContainsAny(s, "#.[]>:") {
		h.isPlain = true
	}
	return h
}

// quoteAttr 转义属性值里的双引号，供构造 CSS 选择器。
func quoteAttr(v string) string { return strings.ReplaceAll(v, `"`, `\"`) }

// idHintVariants 生成 id 类提示的容错变体（应对动态 id）。
//   - 原值
//   - 去掉末尾序号（user-row-42 → user-row）
//   - 取第一个 / 最后一个连字符段（order-2024-0001 → order）
func idHintVariants(hint string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(hint)
	add(trailingDigits.ReplaceAllString(hint, ""))
	if i := strings.Index(hint, "-"); i > 0 {
		add(hint[:i])
	}
	return out
}

// ladderCandidates 生成某一级的候选选择器（可为空 = 该级不适用）。
func ladderCandidates(tier SelectorTier, h selectorHints) []string {
	raw := h.raw
	hint := strings.TrimSpace(h.hint)

	switch tier {
	case TierTestID:
		if testidAttrRe.MatchString(raw) {
			return []string{raw}
		}
		if hint == "" {
			return nil
		}
		return []string{
			fmt.Sprintf(`[data-testid="%s"]`, quoteAttr(hint)),
			fmt.Sprintf(`[data-testid*="%s"]`, quoteAttr(hint)),
		}

	case TierRoleName:
		// role + accessible name：只有「纯文本」输入有可匹配的名字；
		// CSS/XPath 输入在 role 级退化为「标签名同名」的 aria 语义查询
		// （见 locateByTier），此处统一给出 role 容器候选。
		if hint == "" {
			return nil
		}
		return []string{hint}

	case TierAriaLabel:
		if ariaAttrRe.MatchString(raw) {
			return []string{raw}
		}
		if hint == "" {
			return nil
		}
		return []string{
			fmt.Sprintf(`[aria-label="%s"]`, quoteAttr(hint)),
			fmt.Sprintf(`[aria-label*="%s"]`, quoteAttr(hint)),
		}

	case TierStableAttr:
		// 原始选择器本身是 CSS 时永远先直试（它可能就是 #id/[name=...]）。
		cands := make([]string, 0, 8)
		if h.isCSS || (!h.isXPath && !h.isPlain) {
			cands = append(cands, raw)
		}
		if hint != "" {
			for _, v := range idHintVariants(hint) {
				q := quoteAttr(v)
				cands = append(cands,
					fmt.Sprintf(`[name="%s"]`, q),
					"#"+v,
					fmt.Sprintf(`[id^="%s"]`, q),
					fmt.Sprintf(`[id*="%s"]`, q),
					fmt.Sprintf(`[id$="%s"]`, q),
				)
			}
		}
		return cands

	case TierVisibleText:
		// 只对自然语言/中文标签尝试文本查询；CSS 形态强行走文本会把
		// 「div.whatever」当成字面量全文搜索，是纯粹的耗时来源。
		if h.isCSS || h.isXPath || !h.isPlain {
			return nil
		}
		if hint == "" {
			return nil
		}
		return []string{hint}

	case TierStructurePath:
		if !h.isCSS || !strings.Contains(raw, ">") {
			return nil
		}
		return []string{raw}

	case TierXPath:
		if h.isXPath {
			return []string{raw}
		}
		if hint == "" {
			return nil
		}
		esc := strings.ReplaceAll(hint, "'", "&apos;")
		return []string{
			fmt.Sprintf(`//*[contains(text(), '%s')]`, esc),
			fmt.Sprintf(`//*[contains(@id, '%s')]`, esc),
			fmt.Sprintf(`//*[contains(@name, '%s')]`, esc),
		}
	}
	return nil
}

// tryTier 在某一级内逐个候选尝试。返回命中的元素与命中的候选。
func (m *PageManager) tryTier(p *rod.Page, tier SelectorTier, cands []string) (*rod.Element, string, error) {
	if len(cands) == 0 {
		return nil, "", fmt.Errorf("该级无候选")
	}
	var lastErr error
	for _, cs := range cands {
		el, err := m.locateByTier(p, tier, cs)
		if err == nil && el != nil {
			return el, cs, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未命中")
	}
	return nil, "", lastErr
}

// locateByTier 按层级的探测原语定位（Has / HasR / Search / HasX）。
func (m *PageManager) locateByTier(p *rod.Page, tier SelectorTier, cand string) (*rod.Element, error) {
	switch tier {
	case TierTestID, TierAriaLabel, TierStableAttr, TierStructurePath:
		ok, el, err := p.Has(cand)
		if err != nil {
			return nil, err
		}
		if ok && el != nil {
			return el, nil
		}
		return nil, fmt.Errorf("css 未命中: %s", cand)

	case TierRoleName:
		// role + accessible name：HasR([role], 文本正则) 优先，再退化为
		// 任意元素文本匹配（内网系统常只有语义化标签而缺 role 属性）。
		re := regexp.QuoteMeta(cand)
		// 先要求元素确有 role 语义；再退到「语义化标签」（自带隐式 role）。
		// 绝不使用裸 "*"：那会把整页任意文本节点都算命中，破坏分层价值。
		for _, css := range []string{
			"[role]",
			"button",
			"a",
			"label",
			"input[type=submit]",
			"input[type=button]",
		} {
			ok, el, err := p.HasR(css, re)
			if err != nil {
				return nil, err
			}
			if ok && el != nil {
				return el, nil
			}
		}
		return nil, fmt.Errorf("role/name 未命中: %s", cand)

	case TierVisibleText:
		// Search 走 DOM plain-text 查询；命中后校验可见性（隐藏文本不算）。
		sr, err := p.Search(cand)
		if err != nil {
			return nil, err
		}
		defer sr.Release()
		if sr.First == nil {
			return nil, fmt.Errorf("文本未命中: %s", cand)
		}
		vis, err := sr.First.Visible()
		if err != nil || !vis {
			return nil, fmt.Errorf("文本命中但元素不可见: %s", cand)
		}
		return sr.First, nil

	case TierXPath:
		ok, el, err := p.HasX(cand)
		if err != nil {
			return nil, err
		}
		if ok && el != nil {
			return el, nil
		}
		return nil, fmt.Errorf("xpath 未命中: %s", cand)
	}
	return nil, fmt.Errorf("未知层级: %d", int(tier))
}

// findElement 选择器阶梯入口：按 7 级依次尝试，命中即返回元素并记录层级。
//
// 返回 (element, 命中层级, error)。全部失败时 error 为 *LadderError；同时
// 最后一次失败以 success=false + tier 写入选择器事件信封（后端累计 ≥3 次
// 触发 selector_update_required 告警）。
func (m *PageManager) findElement(selector string) (*rod.Element, SelectorTier, error) {
	return m.findElementWithTimeout(selector, 0)
}

// findElementWithTimeout timeout>0 时给命中的元素句柄设置超时。
func (m *PageManager) findElementWithTimeout(selector string, timeout time.Duration) (*rod.Element, SelectorTier, error) {
	if m == nil || m.page == nil {
		return nil, 0, fmt.Errorf("PageManager 未初始化")
	}
	rawPage := m.page.ContextPage()
	if rawPage == nil {
		return nil, 0, fmt.Errorf("page 无底层 rod.Page 句柄")
	}

	hints := parseSelector(selector)
	m.ladderHint = hints.hint
	defer func() { m.ladderHint = "" }()

	start := time.Now()
	attempts := make([]ladderAttempt, 0, 7)

	for tier := TierTestID; tier <= TierXPath; tier++ {
		cands := ladderCandidates(tier, hints)
		if len(cands) == 0 {
			attempts = append(attempts, ladderAttempt{
				Tier: tier, Selector: "-", Err: fmt.Errorf("该级不适用"),
			})
			continue
		}

		// ================================================================
		// W7.1 / 缺陷 A（第四层修复，续）：**每级各自一份 500ms 预算**。
		//
		// 修复前 probe 在循环外创建一次，7 级共用同一个带 deadline 的 page 句柄
		// → 第一级耗尽预算后，后面每一级都立刻 "context deadline exceeded"，
		// 阶梯形同虚设（真跑 6 个 goto 任务时，凡第一级没命中的动作全部报
		// deadline 而不是「未命中」）。注释里写的「7 级 × 500ms = 3.5s 上限」
		// 正是每级独立预算的语义，实现与设计在此处漂移。
		// ================================================================
		probe := rawPage.Context(context.Background()).Timeout(tierProbeTimeout)

		el, hit, err := m.tryTier(probe, tier, cands)
		if err != nil {
			attempts = append(attempts, ladderAttempt{
				Tier: tier, Selector: strings.Join(cands, " | "), Err: err,
			})
			continue
		}

		if timeout > 0 {
			el = el.Context(context.Background()).Timeout(timeout)
		} else {
			// ================================================================
			// W7.1 / 缺陷 A（第四层）：探测 ctx 泄漏到**元素句柄**上。
			//
			// 这里的 el 是从 probe（只给了 tierProbeTimeout=500ms）派生的，
			// 带着那个一次性 deadline。修复前 timeout<=0 时不重绑，于是后续
			// el.Click() / el.Input() / el.Text() 只要超过 500ms 就立刻
			// "context deadline exceeded" —— 与慢网络无关，是句柄自己的 ctx
			// 已过期。
			//
			// 为什么历史上一路没暴露：goto 任务在第一步（类型不支持）就失败，
			// step2 的 fill/click 从来没执行过。别名修好后它立刻成为新的
			// 失败点（生产库 6 个任务真跑时 step2/step3 全部 deadline exceeded）。
			//
			// 修法：命中的元素**始终**改用干净 ctx（Context 是纯替换，不继承
			// probe 的 deadline），再按操作预算设超时。默认预算取 worker 的
			// 元素操作超时（30s），与 page 创建时不再设全局 Timeout 的口径一致。
			// ================================================================
			el = el.Context(context.Background()).Timeout(defaultElementOpTimeout)
		}
		m.recordSelectorHit(hit, tier, start, nil)
		m.logger.Debug("选择器阶梯命中",
			logger.String("input", selector),
			logger.String("tier", tier.String()),
			logger.Int("score", TierScore(tier)),
			logger.String("selector", hit))
		return el, tier, nil
	}

	ladderErr := &LadderError{Input: selector, Attempts: attempts}
	m.recordSelectorHit(selector, TierXPath, start, ladderErr)
	m.logger.Warn("选择器阶梯全部失败",
		logger.String("input", selector),
		logger.String("attempts", ladderErr.Error()))
	return nil, 0, ladderErr
}
