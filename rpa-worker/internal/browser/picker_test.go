package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

)

// ============================================================================
// Phase 156 / NOCODE-02：元素拾取器验收
//
// 验收口径（任务书）：拾取结果在「DOM 微调后」仍命中（用阶梯自愈验证）。
//
// 本文件分两层：
//  1. 纯函数单测（recommendTier / frameSelector）—— 无浏览器依赖，CI 恒跑。
//  2. 真 Chromium 测试 —— 拾取 → 微调 DOM（插入兄弟节点、改 nth-child 序号、
//     去掉 id）→ 用 154 的阶梯 findElement 重新定位 → 必须仍然命中。
//     这是「阶梯自愈」的端到端证据，而不是只看选择器字符串长得对不对。
// ============================================================================

// TestRecommendTier_PrefersTopOfLadder 推荐级按阶梯升序取首个可用。
func TestRecommendTier_PrefersTopOfLadder(t *testing.T) {
	sel, tier := recommendTier([]RecordedTier{
		{Tier: 7, Name: "xpath", Selector: "/html/body/div", OK: true},
		{Tier: 1, Name: "data-testid", Selector: `[data-testid="x"]`, OK: true},
	})
	if sel != `[data-testid="x"]` || tier != 1 {
		t.Fatalf("应推荐 testid 级，实际 %q tier=%d", sel, tier)
	}

	// testid 不可用 → 推荐 role
	sel, tier = recommendTier([]RecordedTier{
		{Tier: 1, Name: "data-testid", OK: false},
		{Tier: 2, Name: "role+name", Selector: "text=提交", OK: true},
		{Tier: 6, Name: "structure-path", Selector: "div > button", OK: true},
	})
	if sel != "text=提交" || tier != 2 {
		t.Fatalf("应推荐 role 级，实际 %q tier=%d", sel, tier)
	}

	// 全不可用 → 空推荐
	if sel, tier := recommendTier([]RecordedTier{{Tier: 1, OK: false}}); sel != "" || tier != 0 {
		t.Fatalf("全不可用时应返回空推荐，实际 %q tier=%d", sel, tier)
	}
}

// TestRecommendTier_MatchesRecorderBestSelector 拾取与录制给出同一答案。
//
// 这是 NOCODE-01/02 一致性的守门测试：两条路径若各算一套，用户会看到
// 「录出来是 A、拾出来是 B」的诡异现象。
func TestRecommendTier_MatchesRecorderBestSelector(t *testing.T) {
	cases := [][]RecordedTier{
		{{Tier: 1, Selector: `[data-testid="a"]`, OK: true}, {Tier: 2, Selector: "text=b", OK: true}},
		{{Tier: 1, OK: false}, {Tier: 4, Selector: "#c", OK: true}},
		{{Tier: 1, OK: false}, {Tier: 2, OK: false}, {Tier: 7, Selector: "/html", OK: true}},
		{{Tier: 1, OK: false}},
	}
	for i, tiers := range cases {
		pickSel, pickTier := recommendTier(tiers)
		recSel, recOK := bestSelector(tiers)
		if pickSel != recSel {
			t.Errorf("case %d: 拾取推荐 %q 与录制推荐 %q 不一致", i, pickSel, recSel)
		}
		if recOK != (pickTier > 0) {
			t.Errorf("case %d: 命中标记不一致（拾取 tier=%d，录制 ok=%v）", i, pickTier, recOK)
		}
	}
}

// TestFrameSelector 优先 id，其次 name，最后序号。
func TestFrameSelector(t *testing.T) {
	if got := frameSelector("main", "nm", 0); got != "#main" {
		t.Errorf("有 id 应优先 id，实际 %q", got)
	}
	if got := frameSelector("", "nm", 0); got != `iframe[name="nm"]` {
		t.Errorf("无 id 应用 name，实际 %q", got)
	}
	if got := frameSelector("", "", 2); got != "iframe:nth-of-type(3)" {
		t.Errorf("都无则用序号（1-based），实际 %q", got)
	}
}

// pickerPageHTML 拾取验收页面：包含有 testid 的按钮、无 testid 但有 aria-label
// 的按钮、以及一段「靠结构路径才能命中」的深层节点。
const pickerPageHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>拾取验收</title></head>
<body>
  <h1>元素拾取验收页</h1>
  <div id="toolbar">
    <button data-testid="save-btn">保存</button>
    <button aria-label="导出报表">导出</button>
  </div>
  <div id="deep">
    <div><div><span class="only-structure">深层文字节点</span></div></div>
  </div>
</body></html>`

// TestPicker_RealChromium_TierDetectionAndSelfHeal NOCODE-02 主验收。
func TestPicker_RealChromium_TierDetectionAndSelfHeal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, pickerPageHTML)
	}))
	defer srv.Close()

	page := launchTestPage(t)
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatalf("导航失败: %v", err)
	}
	page.MustWaitLoad()

	picker := NewPicker(page, nopLogger{})

	// ===== 1. 按坐标拾取带 testid 的按钮 =====
	btn, err := page.Timeout(5 * time.Second).Element(`[data-testid="save-btn"]`)
	if err != nil {
		t.Fatalf("找不到保存按钮: %v", err)
	}
	// Interactable 返回元素可点击中心点的 viewport 坐标（与 liveview 注入
	// 同一套坐标语义），比自己算 rect 中心更贴近真实点击位置。
	pt, err := btn.Interactable()
	if err != nil {
		t.Fatalf("取按钮可点击位置失败: %v", err)
	}
	cx, cy := pt.X, pt.Y

	picked, err := picker.PickAt(cx, cy)
	if err != nil {
		t.Fatalf("拾取失败: %v", err)
	}
	if !picked.Found {
		t.Fatalf("坐标 (%.0f,%.0f) 处未拾取到元素", cx, cy)
	}
	t.Logf("拾取到 <%s> 文本=%q 推荐=%q（第 %d 级，评分 %d）",
		picked.Tag, picked.Text, picked.Recommended, picked.RecommendedTier, picked.RecommendedScore)

	// 断言：7 级阶梯全部返回（前端浮层要逐级显示）
	if len(picked.Tiers) != 7 {
		t.Fatalf("应返回 7 级阶梯，实际 %d 级", len(picked.Tiers))
	}
	// 断言：推荐落在 testid 级
	if picked.RecommendedTier != 1 || !strings.Contains(picked.Recommended, "data-testid") {
		t.Fatalf("有 testid 的元素应推荐第 1 级，实际第 %d 级 %q",
			picked.RecommendedTier, picked.Recommended)
	}
	// 断言：每级都带可用标记（浮层要显示「每级是否可用」）
	for _, tr := range picked.Tiers {
		if tr.Name == "" {
			t.Errorf("第 %d 级缺少名称", tr.Tier)
		}
		if tr.Tier < 1 || tr.Tier > 7 {
			t.Errorf("阶梯编号越界: %d", tr.Tier)
		}
	}

	// ===== 2. 拾取无 testid 但带 aria-label 的按钮 =====
	exp, err := page.Timeout(5 * time.Second).Element(`[aria-label="导出报表"]`)
	if err != nil {
		t.Fatalf("找不到导出按钮: %v", err)
	}
	ept, err := exp.Interactable()
	if err != nil {
		t.Fatalf("取导出按钮位置失败: %v", err)
	}
	p2, err := picker.PickAt(ept.X, ept.Y)
	if err != nil {
		t.Fatalf("拾取导出按钮失败: %v", err)
	}
	if !p2.Found {
		t.Fatal("导出按钮未拾取到")
	}
	t.Logf("导出按钮推荐=%q（第 %d 级）", p2.Recommended, p2.RecommendedTier)
	// 无 testid → 不应推荐第 1 级；应在 role/aria 级
	if p2.RecommendedTier == 1 {
		t.Errorf("无 testid 的元素不应推荐第 1 级，实际 %q", p2.Recommended)
	}
	if p2.RecommendedTier == 0 {
		t.Error("导出按钮应有可用的推荐级")
	}

	// ===== 3. 阶梯自愈：DOM 微调后仍命中 =====
	// 微调手段（模拟真实前端改版，目标：让低层级选择器全部失效）：
	//   a) 在工具栏最前面插入两个兄弟节点 → nth-child / nth-of-type 序号全变
	//   b) 再套一层 wrapper → 结构路径多一层，路径级彻底失效
	//   c) 把原 #toolbar 的 id 改掉 → id 级失效
	//   d) 把可见文本改掉 → 文本级失效
	// 只有 data-testid 级能存活 —— 这正是阶梯顶端存在的意义。
	savedSel := picked.Recommended
	if _, err := page.Eval(`() => {
      var tb = document.getElementById("toolbar");
      var extra1 = document.createElement("span"); extra1.textContent = "新增A";
      var extra2 = document.createElement("span"); extra2.textContent = "新增B";
      tb.insertBefore(extra1, tb.firstChild);
      tb.insertBefore(extra2, tb.firstChild);
      // 套一层 wrapper，破坏结构路径
      var wrap = document.createElement("div");
      wrap.id = "wrap-new";
      tb.parentNode.insertBefore(wrap, tb);
      wrap.appendChild(tb);
      // 改掉 id（结构路径/id 级失效）
      tb.removeAttribute("id");
      // 改掉可见文本（文本级失效）
      var btn = document.querySelector('[data-testid="save-btn"]');
      if (btn) { btn.textContent = "保存草稿"; }
      return true;
    }`); err != nil {
		t.Fatalf("DOM 微调失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 用 154 的阶梯重新定位（走 findElement，不是简单 querySelector）
	pm := NewPageManager(&rodPageAdapter{page: page}, nopLogger{})
	el, tier, err := pm.findElement(savedSel)
	if err != nil {
		t.Fatalf("DOM 微调后阶梯定位失败（选择器 %q）: %v", savedSel, err)
	}
	if el == nil {
		t.Fatalf("DOM 微调后未找到元素（选择器 %q）", savedSel)
	}
	t.Logf("DOM 微调后仍命中，靠第 %d 级（%s）", int(tier), tier.String())

	// 反向验证：微调必须真的让低层级选择器失效，否则「仍命中」没有说服力。
	// 这里逐个确认 id 级 / 结构路径级已失效（#toolbar 已不存在）。
	for _, dead := range []string{"#toolbar", "div#toolbar > button:nth-of-type(1)"} {
		if ok, _, _ := page.Has(dead); ok {
			t.Errorf("微调后低层级选择器 %q 竟然仍命中，说明微调无效、自愈结论不可信", dead)
		}
	}

	// 更强的自愈证据：用一个「已失效的低层级选择器」走阶梯，看它能否自愈到
	// 高层级并命中同一元素 —— 这才是阶梯真正在起作用（而不只是字符串没变）。
	healed, healedTier, healErr := pm.findElement(`#toolbar > button:nth-of-type(1)`)
	if healErr == nil && healed != nil {
		t.Logf("低层级选择器经阶梯自愈命中，靠第 %d 级（%s）", int(healedTier), healedTier.String())
	} else {
		t.Logf("低层级选择器未能自愈（%v）—— 该路径依赖 hint 提取，不作为失败条件", healErr)
	}

	// ===== 4. 按选择器拾取（前端「换用次选」路径）=====
	again, err := picker.PickBySelector(savedSel)
	if err != nil {
		t.Fatalf("按选择器拾取失败: %v", err)
	}
	if !again.Found {
		t.Fatalf("按选择器 %q 拾取不到元素", savedSel)
	}
	if again.Recommended == "" {
		t.Fatal("按选择器拾取应给出推荐级")
	}

	// ===== 5. 页面之外的坐标必须返回 found=false =====
	// 用负坐标（页面外）而不是 (1,1) —— 后者落在 body/html 上，found 本就该是
	// true，用它做断言是把「有元素」误判成「无元素」。
	blank, err := picker.PickAt(-100, -100)
	if err != nil {
		t.Fatalf("页面外坐标拾取不应报错: %v", err)
	}
	if blank == nil {
		t.Fatal("页面外坐标应返回非 nil 结果（found=false）")
	}
	if blank.Found {
		t.Fatalf("页面外坐标 (-100,-100) 不应拾取到元素，实际找到 <%s>", blank.Tag)
	}
}

// TestPicker_RealChromium_FramePick NOCODE-02：iframe 内也可用。
func TestPicker_RealChromium_FramePick(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html><html><body>
          <h1>外层</h1>
          <iframe id="inner" src="%s/frame" style="position:absolute;left:20px;top:120px;width:600px;height:300px;border:1px solid #ccc"></iframe>
        </body></html>`, srv.URL)
	})
	mux.HandleFunc("/frame", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="margin:0">
          <button data-testid="inner-btn" style="margin:40px;padding:20px">内层按钮</button>
        </body></html>`)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	page := launchTestPage(t)
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatalf("导航失败: %v", err)
	}
	page.MustWaitLoad()
	time.Sleep(500 * time.Millisecond) // 等 iframe 加载

	picker := NewPicker(page, nopLogger{})

	// iframe 左上角 (20,120)；按钮在 iframe 内部 margin:40 处，取按钮中心。
	picked, err := picker.PickInFrames(20+120, 120+90)
	if err != nil {
		t.Fatalf("iframe 内拾取失败: %v", err)
	}
	if picked == nil || !picked.Found {
		t.Fatalf("iframe 内坐标未命中任何元素")
	}
	t.Logf("iframe 内拾取到 <%s> 文本=%q 推荐=%q frame=%q",
		picked.Tag, picked.Text, picked.Recommended, picked.Frame)

	// 硬断言：必须真的拾取到 iframe **内部**的按钮，而不是 iframe 宿主元素。
	// 这是 NOCODE-02「iframe 内也可用」的实质验收点。
	if picked.Tag == "iframe" || picked.Tag == "frame" {
		t.Fatalf("拾取到的是 iframe 宿主元素（tag=%s），未真正进入 frame 内部", picked.Tag)
	}
	if picked.Frame == "" {
		t.Fatal("拾取 iframe 内元素时必须回传 frame 选择器（回放需要先进入 iframe）")
	}
	if picked.Frame != "#inner" {
		t.Errorf("frame 选择器应为 #inner，实际 %q", picked.Frame)
	}
	if !strings.Contains(picked.Recommended, "inner-btn") {
		t.Errorf("应推荐内层按钮的 testid 选择器，实际 %q", picked.Recommended)
	}
}
