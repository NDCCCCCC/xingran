package browser

import (
	"testing"

	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 156 / NOCODE-01：录制器纯函数单测
//
// 覆盖：聚类规则（输入合并 / submit 去重 / 滚动合并 / 密码脱敏 / URL 变化）、
//       阶梯优先 testid/role、中文描述零技术术语、与 154 阶梯评分单调性一致。
// ============================================================================

// tier 构造一个阶梯候选（测试辅助）。
func tier(n int, name, sel string, ok bool) RecordedTier {
	return RecordedTier{Tier: n, Name: name, Selector: sel, OK: ok, Score: TierScore(SelectorTier(n))}
}

// ev 构造一个基础事件（默认带全部 7 级候选）。
func ev(kind string, ts int64) RecordedEvent {
	return RecordedEvent{
		Kind: kind,
		Ts:   ts,
		URL:  "https://example.com/form",
		Tag:  "button",
		Text: "提交",
		Tiers: []RecordedTier{
			tier(1, "data-testid", `[data-testid="submit"]`, true),
			tier(2, "role+name", "text=提交", true),
			tier(4, "stable-attr", "#submit-btn", true),
		},
	}
}

// TestBuildActions_InsertsStartNavigate 录制必须以 navigate 开头（独立回放的前提）。
func TestBuildActions_InsertsStartNavigate(t *testing.T) {
	actions := BuildActions([]RecordedEvent{ev("click", 1000)})
	if len(actions) < 2 {
		t.Fatalf("期望至少 2 条动作（navigate + click），实际 %d", len(actions))
	}
	if actions[0].Type != types.ActionNavigate {
		t.Fatalf("首条动作必须是 navigate，实际 %s", actions[0].Type)
	}
	if got := paramString(actions[0].Params, "url"); got != "https://example.com/form" {
		t.Fatalf("navigate url 不符: %q", got)
	}
}

// TestBestSelector_PrefersTestIDThenRole 阶梯优先 testid/role（ENGINE-03 约定）。
func TestBestSelector_PrefersTestIDThenRole(t *testing.T) {
	// 全部可用 → 取 testid
	sel, ok := bestSelector([]RecordedTier{
		tier(7, "xpath", "/html/body/div[1]", true),
		tier(1, "data-testid", `[data-testid="a"]`, true),
		tier(2, "role+name", "text=登录", true),
	})
	if !ok || sel != `[data-testid="a"]` {
		t.Fatalf("应优先 testid，实际 %q ok=%v", sel, ok)
	}

	// testid 不可用 → 退到 role
	sel, ok = bestSelector([]RecordedTier{
		tier(1, "data-testid", "", false),
		tier(2, "role+name", "text=登录", true),
		tier(7, "xpath", "/html/body", true),
	})
	if !ok || sel != "text=登录" {
		t.Fatalf("应退到 role+name，实际 %q ok=%v", sel, ok)
	}

	// 只有兜底级可用
	sel, ok = bestSelector([]RecordedTier{
		tier(1, "data-testid", "", false),
		tier(2, "role+name", "", false),
		tier(7, "xpath", "/html/body", true),
	})
	if !ok || sel != "/html/body" {
		t.Fatalf("应退到 xpath，实际 %q ok=%v", sel, ok)
	}

	// 全不可用
	if _, ok := bestSelector([]RecordedTier{tier(1, "data-testid", "", false)}); ok {
		t.Fatal("全部不可用时不应返回命中")
	}
}

// TestBestSelector_TierOrderMatchesLadderScore 录制的 tier 升序 == 154 阶梯评分降序。
//
// 这条断言是「156 只消费 154 阶梯、不得自造顺序」的守门测试：
// 若 154 调换阶梯顺序（评分不再单调递减），本测试立刻失败。
func TestBestSelector_TierOrderMatchesLadderScore(t *testing.T) {
	prev := 101
	for tierNum := TierTestID; tierNum <= TierXPath; tierNum++ {
		score := TierScore(tierNum)
		if score >= prev {
			t.Fatalf("阶梯评分必须随 tier 编号严格递减：tier=%d score=%d 不小于上一级 %d",
				int(tierNum), score, prev)
		}
		prev = score
	}
}

// TestBuildActions_DebouncesInput 同一元素连续输入合并为一条（保留最终值）。
func TestBuildActions_DebouncesInput(t *testing.T) {
	sel := []RecordedTier{tier(4, "stable-attr", "#amount", true)}
	mk := func(ts int64, v string) RecordedEvent {
		return RecordedEvent{
			Kind: "input", Ts: ts, URL: "https://example.com/form",
			Tag: "input", Type: "text", Value: v, Tiers: sel,
		}
	}
	actions := BuildActions([]RecordedEvent{
		mk(1000, "1"),
		mk(1100, "12"),
		mk(1200, "123"),
	})
	fills := 0
	var last string
	for _, a := range actions {
		if a.Type == types.ActionFill {
			fills++
			last = paramString(a.Params, "value")
		}
	}
	if fills != 1 {
		t.Fatalf("窗口内逐字符输入应合并为 1 条 fill，实际 %d 条", fills)
	}
	if last != "123" {
		t.Fatalf("应保留最终值 123，实际 %q", last)
	}
}

// TestBuildActions_SeparateInputsOutsideWindow 超出窗口的输入各自成条。
func TestBuildActions_SeparateInputsOutsideWindow(t *testing.T) {
	sel := []RecordedTier{tier(4, "stable-attr", "#amount", true)}
	mk := func(ts int64, v string) RecordedEvent {
		return RecordedEvent{Kind: "input", Ts: ts, URL: "https://e.com", Type: "text", Value: v, Tiers: sel}
	}
	actions := BuildActions([]RecordedEvent{mk(1000, "a"), mk(1000 + inputDebounceMS + 1, "b")})
	fills := 0
	for _, a := range actions {
		if a.Type == types.ActionFill {
			fills++
		}
	}
	if fills != 2 {
		t.Fatalf("窗口外的两次输入应为 2 条 fill，实际 %d", fills)
	}
}

// TestBuildActions_DropsDerivedSubmit 点击提交后的派生 submit 被丢弃。
func TestBuildActions_DropsDerivedSubmit(t *testing.T) {
	click := ev("click", 1000)
	submit := RecordedEvent{
		Kind: "submit", Ts: 1200, URL: "https://example.com/form", Tag: "form",
		Tiers: []RecordedTier{tier(4, "stable-attr", "#form1", true)},
	}
	actions := BuildActions([]RecordedEvent{click, submit})
	for _, a := range actions {
		if a.Selector == "#form1" {
			t.Fatal("点击后 1.5s 内的派生 submit 应被丢弃，否则回放会提交两次")
		}
	}
}

// TestBuildActions_KeepsStandaloneSubmit 无前序点击的 submit 必须保留。
func TestBuildActions_KeepsStandaloneSubmit(t *testing.T) {
	submit := RecordedEvent{
		Kind: "submit", Ts: 5000, URL: "https://example.com/form", Tag: "form",
		Tiers: []RecordedTier{tier(4, "stable-attr", "#form1", true)},
	}
	actions := BuildActions([]RecordedEvent{submit})
	found := false
	for _, a := range actions {
		if a.Selector == "#form1" && a.Type == types.ActionClick {
			found = true
		}
	}
	if !found {
		t.Fatal("独立的 submit 应转成一条 click 动作")
	}
}

// TestBuildActions_MergesScroll 连续滚动合并为一条。
func TestBuildActions_MergesScroll(t *testing.T) {
	mk := func(ts int64, y float64) RecordedEvent {
		return RecordedEvent{Kind: "scroll", Ts: ts, URL: "https://e.com", ScrollY: y}
	}
	actions := BuildActions([]RecordedEvent{mk(1000, 200), mk(1300, 800), mk(1500, 600)})
	scrolls := 0
	var pixels int
	for _, a := range actions {
		if a.Type == types.ActionScroll {
			scrolls++
			pixels, _ = a.Params["pixels"].(int)
		}
	}
	if scrolls != 1 {
		t.Fatalf("连续滚动应合并为 1 条，实际 %d", scrolls)
	}
	if pixels != 800 {
		t.Fatalf("合并应取最大位移 800，实际 %d", pixels)
	}
}

// TestBuildActions_PasswordNeverPlaintext 密码框值必须替换为占位符。
func TestBuildActions_PasswordNeverPlaintext(t *testing.T) {
	pw := RecordedEvent{
		Kind: "input", Ts: 1000, URL: "https://e.com", Tag: "input", Type: "password",
		Value: "SuperSecret123", Tiers: []RecordedTier{tier(4, "stable-attr", "#pwd", true)},
	}
	actions := BuildActions([]RecordedEvent{pw})
	for _, a := range actions {
		if v := paramString(a.Params, "value"); v == "SuperSecret123" {
			t.Fatal("密码明文绝不能进入录制结果（必须替换为 ${password}）")
		}
		if v := paramString(a.Params, "value"); v == "${password}" {
			if a.Description != "在「」填入密码" && !contains(a.Description, "密码") {
				t.Fatalf("密码步骤描述应含「密码」，实际 %q", a.Description)
			}
			return
		}
	}
	t.Fatal("未找到密码占位动作")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// TestBuildActions_NavigateOnURLChange URL 变化插入 navigate。
func TestBuildActions_NavigateOnURLChange(t *testing.T) {
	a := ev("click", 1000)
	b := ev("click", 2000)
	b.URL = "https://example.com/done"
	actions := BuildActions([]RecordedEvent{a, b})
	navigates := 0
	for _, x := range actions {
		if x.Type == types.ActionNavigate {
			navigates++
		}
	}
	if navigates != 2 {
		t.Fatalf("起点 + 一次 URL 变化应产生 2 条 navigate，实际 %d", navigates)
	}
}

// TestBuildActions_DescriptionsAreBusinessChinese 中文步骤描述零技术术语。
//
// NOCODE-04 硬约束：不得出现 http 方法 / selector / JSON 这类词。
func TestBuildActions_DescriptionsAreBusinessChinese(t *testing.T) {
	events := []RecordedEvent{
		ev("click", 1000),
		{
			Kind: "input", Ts: 1100, URL: "https://example.com/form", Tag: "input",
			Type: "text", Text: "金额", Value: "1200",
			Tiers: []RecordedTier{tier(4, "stable-attr", "#amount", true)},
		},
		{
			Kind: "select", Ts: 1200, URL: "https://example.com/form", Tag: "select",
			Text: "报销类型", Value: "travel",
			Tiers: []RecordedTier{tier(4, "stable-attr", "#type", true)},
		},
	}
	actions := BuildActions(events)
	banned := []string{"selector", "CSS", "xpath", "JSON", "POST", "GET", "http://", "https://", "#", "div"}
	for _, a := range actions {
		if a.Description == "" {
			t.Errorf("动作 %s 缺少中文描述", a.Type)
			continue
		}
		for _, b := range banned {
			if contains(a.Description, b) {
				t.Errorf("步骤描述「%s」泄漏技术术语 %q", a.Description, b)
			}
		}
	}
}

// TestBuildActions_KeyboardCombos 组合键录制与解析语法一致。
func TestBuildActions_KeyboardCombos(t *testing.T) {
	cases := []struct {
		key  string
		mods int
		want string
	}{
		{"Enter", 0, "enter"},
		{"a", 2, "ctrl+a"},
		{"Tab", 0, "tab"},
		{"Escape", 0, "esc"},
		{"s", 10, "ctrl+shift+s"},
	}
	for _, c := range cases {
		got := keyCombo(c.key, c.mods)
		if got != c.want {
			t.Errorf("keyCombo(%q,%d) = %q，期望 %q", c.key, c.mods, got, c.want)
		}
	}
	// 单独修饰键无语义
	if keyCombo("Control", 0) != "" {
		t.Error("单独按修饰键不应产出动作")
	}
}

// TestBuildActions_EmptyInput 空事件不产生动作（不臆造 navigate）。
func TestBuildActions_EmptyInput(t *testing.T) {
	if got := BuildActions(nil); len(got) != 0 {
		t.Fatalf("空事件应产出 0 条动作，实际 %d", len(got))
	}
}

// TestBuildActions_DedupeAdjacent 相邻完全重复的动作被去重且 id 连续。
func TestBuildActions_DedupeAdjacent(t *testing.T) {
	e := ev("click", 1000)
	e2 := ev("click", 3000)
	actions := BuildActions([]RecordedEvent{e, e2})
	clicks := 0
	for _, a := range actions {
		if a.Type == types.ActionClick {
			clicks++
		}
	}
	if clicks != 1 {
		t.Fatalf("相邻重复点击应去重为 1 条，实际 %d", clicks)
	}
	for i, a := range actions {
		if want := actionID(i + 1); a.ID != want {
			t.Fatalf("动作 id 应连续：第 %d 条期望 %q 实际 %q", i, want, a.ID)
		}
	}
}

// TestBuildRecording_StepsMatchActions 步骤描述与动作一一对应。
func TestBuildRecording_StepsMatchActions(t *testing.T) {
	r := BuildRecording([]RecordedEvent{ev("click", 1000)})
	if len(r.Steps) != len(r.Actions) {
		t.Fatalf("steps(%d) 与 actions(%d) 数量必须一致", len(r.Steps), len(r.Actions))
	}
	if r.StartURL != "https://example.com/form" {
		t.Fatalf("startUrl 不符: %q", r.StartURL)
	}
	if r.EventCount != 1 {
		t.Fatalf("eventCount 不符: %d", r.EventCount)
	}
}

// TestKeyCombo_RoundTripWithLadderParser 组合键能被 154 的 parseKeyCombo 解析。
//
// 这条守住「录制产出 → 引擎执行」的跨 phase 契约：若 154 改了键名别名，
// 这里会立刻发现录制侧语法失配。
func TestKeyCombo_RoundTripWithLadderParser(t *testing.T) {
	for _, combo := range []string{"ctrl+a", "enter", "tab", "esc", "ctrl+shift+s"} {
		if _, err := parseKeyCombo(combo); err != nil {
			t.Errorf("录制产出的组合键 %q 无法被引擎解析: %v", combo, err)
		}
	}
}
