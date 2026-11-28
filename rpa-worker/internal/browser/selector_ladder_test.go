package browser

import (
	"strings"
	"testing"
)

// ============================================================================
// Phase 154 / ENGINE-03：阶梯评分单测（轻量兜底，-short 也跑）
//   - 评分单调递减（阶梯顺序的静态守护）
//   - 7 级覆盖完整、无缺号
//   - parseSelector 线索提取（CSS / XPath / 纯文本 / testid / 动态 id）
//   - ladderCandidates 每级候选形态正确
//   - LadderError 逐级回显
// ============================================================================

// TestLadderScoreMonotonic 评分必须随层级严格递减。
func TestLadderScoreMonotonic(t *testing.T) {
	prev := 1 << 30
	for tier := TierTestID; tier <= TierXPath; tier++ {
		score := TierScore(tier)
		if score <= 0 {
			t.Fatalf("层级 %s 缺评分", tier.String())
		}
		if score >= prev {
			t.Fatalf("阶梯评分非单调递减：%s=%d, 上一级=%d", tier.String(), score, prev)
		}
		prev = score
	}
	if TierScore(TierTestID) != 100 {
		t.Fatalf("data-testid 必须是最高分 100，实际 %d", TierScore(TierTestID))
	}
	if TierScore(TierXPath) != 5 {
		t.Fatalf("XPath 必须是兜底低分 5，实际 %d", TierScore(TierXPath))
	}
}

// TestLadderTiersComplete 7 级齐全，String() 可读。
func TestLadderTiersComplete(t *testing.T) {
	tiers := []SelectorTier{
		TierTestID, TierRoleName, TierAriaLabel, TierStableAttr,
		TierVisibleText, TierStructurePath, TierXPath,
	}
	if len(tiers) != 7 {
		t.Fatalf("阶梯必须恰好 7 级，实际 %d", len(tiers))
	}
	for _, tier := range tiers {
		if tier.String() == "" || strings.HasPrefix(tier.String(), "tier-") {
			t.Fatalf("层级 %d 缺少可读名称", int(tier))
		}
	}
	// 顺序语义断言（防止常量顺序被无意调换）
	if !(TierTestID < TierRoleName && TierRoleName < TierAriaLabel &&
		TierAriaLabel < TierStableAttr && TierStableAttr < TierVisibleText &&
		TierVisibleText < TierStructurePath && TierStructurePath < TierXPath) {
		t.Fatal("层级常量顺序被调换（必须是 data-testid > role > aria > attr > text > path > xpath）")
	}
}

// TestParseSelectorHints 线索提取。
func TestParseSelectorHints(t *testing.T) {
	cases := []struct {
		sel     string
		want    string
		isXPath bool
		isCSS   bool
		isPlain bool
	}{
		{`[data-testid="submit-btn"]`, "submit-btn", false, true, false},
		{`[aria-label="保存"]`, "保存", false, true, false},
		{`#user-row-42`, "user-row-42", false, true, false},
		{`//*[contains(text(),'登录')]`, "登录", true, false, false},
		{`text=提交订单`, "提交订单", false, false, true},
		{`提交订单`, "提交订单", false, false, true},
		{`div.nav > span`, "", false, true, false},
	}
	for _, tc := range cases {
		h := parseSelector(tc.sel)
		if h.hint != tc.want {
			t.Errorf("parseSelector(%q).hint=%q，期望 %q", tc.sel, h.hint, tc.want)
		}
		if h.isXPath != tc.isXPath || h.isCSS != tc.isCSS || h.isPlain != tc.isPlain {
			t.Errorf("parseSelector(%q) 形态判定错误: xpath=%v css=%v plain=%v",
				tc.sel, h.isXPath, h.isCSS, h.isPlain)
		}
	}
}

// TestLadderCandidatesPerTier 各级候选形态。
func TestLadderCandidatesPerTier(t *testing.T) {
	h := parseSelector(`[data-testid="submit-btn"]`)
	if c := ladderCandidates(TierTestID, h); len(c) == 0 || c[0] != `[data-testid="submit-btn"]` {
		t.Fatalf("testid 级候选错误: %v", c)
	}
	if c := ladderCandidates(TierAriaLabel, h); len(c) == 0 {
		t.Fatal("aria 级不应为空（应派生 [aria-label*=...]）")
	}
	// CSS 形态不得进入可见文本级（避免把 "div.whatever" 当文本全页搜索）
	if c := ladderCandidates(TierVisibleText, parseSelector(`div.whatever`)); len(c) != 0 {
		t.Fatalf("CSS 形态不应有文本级候选: %v", c)
	}
	// 纯文本形态必须有文本级候选
	if c := ladderCandidates(TierVisibleText, parseSelector(`提交订单`)); len(c) == 0 {
		t.Fatal("纯文本形态应有文本级候选")
	}
	// XPath 形态在 XPath 级直接用原串
	if c := ladderCandidates(TierXPath, parseSelector(`//div[@id='x']`)); len(c) == 0 || c[0] != `//div[@id='x']` {
		t.Fatalf("XPath 级候选错误: %v", c)
	}
}

// TestIDHintVariants 动态 id 容错变体。
func TestIDHintVariants(t *testing.T) {
	got := idHintVariants("user-row-42")
	want := []string{"user-row-42", "user-row", "user"}
	if len(got) != len(want) {
		t.Fatalf("变体数量=%d，期望 %d（%v）", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("变体[%d]=%q，期望 %q（全量 %v）", i, got[i], want[i], got)
		}
	}
}

// TestLadderErrorFormat 全失败错误必须逐级回显 7 条。
func TestLadderErrorFormat(t *testing.T) {
	e := &LadderError{Input: "#btn", Attempts: []ladderAttempt{
		{Tier: TierTestID, Selector: "[data-testid=btn]", Err: errNotFound},
		{Tier: TierRoleName, Selector: "btn", Err: errNotFound},
		{Tier: TierAriaLabel, Selector: "[aria-label=btn]", Err: errNotFound},
		{Tier: TierStableAttr, Selector: "#btn", Err: errNotFound},
		{Tier: TierVisibleText, Selector: "btn", Err: errNotFound},
		{Tier: TierStructurePath, Selector: "-", Err: errNotFound},
		{Tier: TierXPath, Selector: "//*[contains(@id,'btn')]", Err: errNotFound},
	}}
	msg := e.Error()
	if !strings.Contains(msg, "共 7 级") {
		t.Fatalf("错误信息必须声明 7 级: %s", msg)
	}
	for tier := TierTestID; tier <= TierXPath; tier++ {
		if !strings.Contains(msg, tier.String()) {
			t.Fatalf("错误信息缺少层级 %s: %s", tier.String(), msg)
		}
	}
}

type notFoundErr struct{}

func (notFoundErr) Error() string { return "not found" }

var errNotFound = notFoundErr{}
