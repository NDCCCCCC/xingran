package browser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 156 / NOCODE-01：页面内录制器
//
// 产品形态（261123-v156-research §4）：一般员工的「录制 → 中文步骤 → 参数 →
// 一键运行」。不做流程图编辑器。
//
// 技术路线（NOCODE-01 硬要求：**禁止**浏览器扩展）：
//   proto.PageAddScriptToEvaluateOnNewDocument 注入监听脚本（= Puppeteer 的
//   evalOnNewDocument 语义）。脚本在**每个新文档**创建时执行，因此录制跨越
//   页面跳转/SPA 路由切换仍然有效，且零分发成本。
//
// 采集面：click / dblclick / contextmenu / input / change / select / submit /
//        scroll / 组合键（keydown）。
//
// 产出：[]types.Action —— schema 与 types.Action 完全一致，可直接落库执行，
//      无需二次转换（这是 NOCODE-01「录制结束产出可直接执行的 actions JSON」
//      的验收口径）。
//
// 选择器（ENGINE-03 阶梯约定，归属协议第 7 条：**只消费**）：
//   录制生成的选择器**优先落 testid/role**。JS 侧一次性算出全部 7 级候选并
//   标注「是否唯一可用」，Go 侧按阶梯顺序取第一个可用的 —— 与
//   selector_ladder.go 的 tierScores 单调性天然一致（单测断言该一致性）。
//
// 归属协议第 5 条：本文件是净新增，**不改** manager.go / page_manager.go 的
// 核心契约（154 已冻结）。复用其导出能力：TestChromePath / 阶梯层级常量。
// ============================================================================

// RecordedTier 录制/拾取时某一级阶梯候选的实测结果。
//
// 与 selector_ladder.go 的 SelectorTier 同编号（1..7），前端据此显示
// 「本次靠第几级命中」与「每级是否可用」（NOCODE-02 的浮层要求）。
type RecordedTier struct {
	Tier     int    `json:"tier"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
	// OK 该级候选是否存在且唯一（唯一性强的才值得作为首选）。
	OK bool `json:"ok"`
	// Score 该级评分（与 TierScore 同源，0 表示该级不适用）。
	Score int `json:"score"`
}

// RecordedEvent 页面内监听器上报的原始事件（JS → Go 的线格式）。
type RecordedEvent struct {
	Kind string `json:"kind"`
	Ts   int64  `json:"ts"`
	URL  string `json:"url"`

	Tag   string `json:"tag"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Text  string `json:"text"`
	// Checked 复选框/单选框的勾选态（录制 click 后据此产出 click 或 select）。
	Checked bool `json:"checked"`

	Key       string `json:"key"`
	Modifiers int    `json:"modifiers"`

	DeltaX float64 `json:"deltaX"`
	DeltaY float64 `json:"deltaY"`
	ScrollY float64 `json:"scrollY"`

	Tiers []RecordedTier `json:"tiers"`
}

// Recorder 单页录制器。
//
// 生命周期：NewRecorder → Install（注入，对新文档生效）→ Start（当前文档
// 也装一份，兼容「已经打开页面再开始录」）→ 用户操作 → Stop（拉事件并产出动作）。
type Recorder struct {
	page   *rod.Page
	logger logger.Logger
	// startURL 录制起点 URL（产出第一个 navigate 动作）。
	startURL string
	// startTS 录制开始时刻（过滤安装前残留事件）。
	startTS int64
	// scriptID 持久注入脚本句柄（Stop 时移除，避免 page 复用时继续录音）。
	scriptID  proto.PageScriptIdentifier
	recording bool
}

// NewRecorder 创建录制器。page 必须是非 nil 的 rod page（RawPage()）。
func NewRecorder(page *rod.Page, log logger.Logger) *Recorder {
	return &Recorder{page: page, logger: log}
}

// recorderRuntimeJS 监听脚本（注入到每个新文档）。
//
// 设计约束：
//  1. **不拦截事件**：全部用捕获阶段 addEventListener 且不 preventDefault，
//     用户操作照常生效（录制不能改变被测系统的行为）。
//  2. **不产生副作用**：只在 window.__rpaRec 上累积，绝不改 DOM 属性。
//  3. **可重入**：脚本重复执行时先判 __rpaRec 已存在则直接返回，避免重复监听。
//  4. 事件体量有上限（MAX_EVENTS），防止长录制把页面内存吃满。
//
// 注意：本常量是 JS 源码，Go 反引号字符串内**不得**出现反引号。
const recorderRuntimeJS = `(function(){
  if (window.__rpaRec) { return true; }
  var MAX_EVENTS = 4000;

  function cssq(v){
    return String(v == null ? "" : v).replace(/["\\]/g, function(c){ return "\\" + c; });
  }
  function textOf(el){
    if (!el) return "";
    var t = el.innerText || el.textContent || "";
    return String(t).replace(/\s+/g, " ").trim();
  }
  function shortText(el){
    var t = textOf(el);
    return t.length > 40 ? "" : t;
  }
  function countSel(sel){
    try { return document.querySelectorAll(sel).length; } catch (e) { return -1; }
  }
  function pathOf(el){
    var parts = [];
    var cur = el;
    while (cur && cur.nodeType === 1 && cur !== document.documentElement) {
      var tag = cur.tagName.toLowerCase();
      var parent = cur.parentElement;
      if (!parent) { parts.unshift(tag); break; }
      var idx = 1;
      for (var i = 0; i < parent.children.length; i++) {
        if (parent.children[i] === cur) { break; }
        if (parent.children[i].tagName === cur.tagName) { idx++; }
      }
      parts.unshift(tag + ":nth-of-type(" + idx + ")");
      cur = parent;
    }
    return parts.join(" > ");
  }
  function xpathOf(el){
    if (!el || el.nodeType !== 1) { return ""; }
    if (el.id) { return "//*[@id=\"" + cssq(el.id) + "\"]"; }
    var parts = [];
    var cur = el;
    while (cur && cur.nodeType === 1) {
      var tag = cur.tagName.toLowerCase();
      var parent = cur.parentElement;
      if (!parent) { parts.unshift(tag); break; }
      var idx = 1;
      for (var i = 0; i < parent.children.length; i++) {
        if (parent.children[i] === cur) { break; }
        if (parent.children[i].tagName === cur.tagName) { idx++; }
      }
      parts.unshift(tag + "[" + idx + "]");
      cur = parent;
    }
    return "/" + parts.join("/");
  }
  function accessName(el){
    if (!el.getAttribute) { return ""; }
    var n = el.getAttribute("aria-label");
    if (n && n.trim()) { return n.trim(); }
    var t = shortText(el);
    if (t) { return t; }
    var ph = el.getAttribute("placeholder") || el.getAttribute("title");
    if (ph && ph.trim()) { return ph.trim(); }
    var v = el.getAttribute("value");
    if (el.tagName === "INPUT" && v && v.trim()) { return v.trim(); }
    return "";
  }
  function roleNameOK(el, name){
    if (!name) { return false; }
    var tags = ["[role]", "button", "a", "label", "input[type=submit]", "input[type=button]"];
    for (var i = 0; i < tags.length; i++) {
      var list = [];
      try { list = document.querySelectorAll(tags[i]); } catch (e) { continue; }
      for (var j = 0; j < list.length; j++) {
        if (list[j] === el && accessName(el) === name) { return true; }
      }
    }
    return false;
  }

  // describe 返回 7 级阶梯候选（与 selector_ladder.go 的层级编号一一对应）。
  window.__rpaDescribe = function(el){
    if (!el || el.nodeType !== 1) { return null; }
    var tiers = [];
    var testAttrs = ["data-testid", "data-test-id", "data-test", "data-cy", "data-qa"];

    // 1 data-testid
    var tsel = "";
    for (var a = 0; a < testAttrs.length; a++) {
      var tv = el.getAttribute && el.getAttribute(testAttrs[a]);
      if (tv && tv.trim()) {
        var cand = "[" + testAttrs[a] + "=\"" + cssq(tv.trim()) + "\"]";
        if (countSel(cand) === 1) { tsel = cand; }
        break;
      }
    }
    tiers.push({tier: 1, name: "data-testid", selector: tsel, ok: tsel !== "", score: 100});

    // 2 role + accessible name（选择器形态用 text=，由阶梯在 role 级消费）
    var name = accessName(el);
    var roleOK = roleNameOK(el, name);
    tiers.push({tier: 2, name: "role+name", selector: name ? ("text=" + name) : "", ok: roleOK, score: 90});

    // 3 aria-label
    var aria = el.getAttribute && el.getAttribute("aria-label");
    var asel = "";
    if (aria && aria.trim()) {
      var acand = "[aria-label=\"" + cssq(aria.trim()) + "\"]";
      if (countSel(acand) === 1) { asel = acand; }
    }
    tiers.push({tier: 3, name: "aria-label", selector: asel, ok: asel !== "", score: 80});

    // 4 稳定属性 id / name
    var ssel = "";
    var idv = el.getAttribute && el.getAttribute("id");
    if (idv && idv.trim() && !/^[0-9]/.test(idv) && !/:/.test(idv)) {
      var icand = "#" + idv.trim();
      if (countSel(icand) === 1) { ssel = icand; }
    }
    if (!ssel) {
      var nv = el.getAttribute && el.getAttribute("name");
      if (nv && nv.trim()) {
        var ncand = "[name=\"" + cssq(nv.trim()) + "\"]";
        if (countSel(ncand) === 1) { ssel = ncand; }
      }
    }
    tiers.push({tier: 4, name: "stable-attr", selector: ssel, ok: ssel !== "", score: 65});

    // 5 可见文本（只对短文本；过长文本不适合做锚点）
    var vt = shortText(el);
    var vsel = vt ? ("text=" + vt) : "";
    tiers.push({tier: 5, name: "visible-text", selector: vsel, ok: vsel !== "", score: 45});

    // 6 结构路径
    var psel = pathOf(el);
    var pathOK = psel !== "" && countSel(psel) === 1;
    tiers.push({tier: 6, name: "structure-path", selector: psel, ok: pathOK, score: 20});

    // 7 绝对 XPath
    var xsel = xpathOf(el);
    tiers.push({tier: 7, name: "xpath", selector: xsel, ok: xsel !== "", score: 5});

    return {
      tag: el.tagName ? el.tagName.toLowerCase() : "",
      type: (el.getAttribute && el.getAttribute("type")) || "",
      name: name,
      text: shortText(el),
      tiers: tiers
    };
  };

  window.__rpaRec = {
    events: [],
    enabled: true,
    overflow: false,
    push: function(ev){
      if (!window.__rpaRec.enabled) { return; }
      if (window.__rpaRec.events.length >= MAX_EVENTS) {
        window.__rpaRec.overflow = true;
        return;
      }
      window.__rpaRec.events.push(ev);
      // 关键事件（会触发导航的）必须立刻落盘：节流窗口内若不强制写，
      // 「点登录 → 立刻整页跳转」会在文档销毁时丢掉这一条。
      var critical = (ev.kind === "click" || ev.kind === "dblclick" ||
                      ev.kind === "submit" || ev.kind === "keydown" ||
                      ev.kind === "select");
      window.__rpaRec.persist(critical);
    },
    // persist 把事件落 sessionStorage。
    //
    // 这是跨导航录制的**必需**机制：window 变量随文档销毁，录制
    // 「登录 → 跳转 → 填表」时前一段事件会在导航瞬间丢失。sessionStorage
    // 在同源导航间保留，注入脚本启动时再从它恢复，从而把整段录制串起来。
    // 写入节流到 150ms，避免每个按键都做一次全量序列化。
    persist: function(force){
      var now = Date.now();
      if (!force && now - (window.__rpaRec.lastPersist || 0) < 150) { return; }
      window.__rpaRec.lastPersist = now;
      try {
        var cap = window.__rpaRec.events.slice(-MAX_EVENTS);
        sessionStorage.setItem("__rpaRecEvents", JSON.stringify(cap));
        sessionStorage.setItem("__rpaRecOn", "1");
      } catch (e) { /* 配额满：丢弃持久化，仅保留内存缓冲 */ }
    }
  };

  // 跨导航恢复：同一录制会话内把上一文档的事件接回来。
  try {
    if (sessionStorage.getItem("__rpaRecOn") === "1") {
      var saved = sessionStorage.getItem("__rpaRecEvents");
      if (saved) {
        var arr = JSON.parse(saved);
        if (arr && arr.length) { window.__rpaRec.events = arr; }
      }
    }
  } catch (e) { /* sessionStorage 不可用：退化为单文档录制 */ }

  function base(el){
    var d = window.__rpaDescribe(el);
    if (!d) { return null; }
    return {
      tag: d.tag, type: d.type, text: d.text, tiers: d.tiers,
      url: location.href, ts: Date.now()
    };
  }
  function target(el){
    if (!el) { return null; }
    if (el.nodeType !== 1) { el = el.parentElement; }
    return el;
  }

  document.addEventListener("click", function(e){
    var el = target(e.target);
    var b = el && base(el);
    if (!b) { return; }
    b.kind = "click";
    b.checked = !!el.checked;
    b.value = (el.value != null) ? String(el.value) : "";
    b.modifiers = (e.ctrlKey ? 2 : 0) | (e.shiftKey ? 8 : 0) | (e.altKey ? 1 : 0) | (e.metaKey ? 4 : 0);
    window.__rpaRec.push(b);
  }, true);

  document.addEventListener("dblclick", function(e){
    var el = target(e.target);
    var b = el && base(el);
    if (!b) { return; }
    b.kind = "dblclick";
    window.__rpaRec.push(b);
  }, true);

  document.addEventListener("contextmenu", function(e){
    var el = target(e.target);
    var b = el && base(el);
    if (!b) { return; }
    b.kind = "contextmenu";
    window.__rpaRec.push(b);
  }, true);

  function inputEvt(kind){
    return function(e){
      var el = target(e.target);
      var b = el && base(el);
      if (!b) { return; }
      b.kind = kind;
      b.value = (el.value != null) ? String(el.value) : "";
      b.checked = !!el.checked;
      if (el.tagName === "SELECT") {
        b.kind = "select";
        b.value = el.value;
        b.text = el.options && el.selectedIndex >= 0 ? el.options[el.selectedIndex].text : "";
      }
      window.__rpaRec.push(b);
    };
  }
  document.addEventListener("input", inputEvt("input"), true);
  document.addEventListener("change", inputEvt("change"), true);

  document.addEventListener("submit", function(e){
    var el = target(e.target);
    var b = el && base(el);
    if (!b) { return; }
    b.kind = "submit";
    window.__rpaRec.push(b);
  }, true);

  var lastScrollAt = 0;
  document.addEventListener("scroll", function(e){
    var now = Date.now();
    var el = target(e.target);
    var b = {kind: "scroll", ts: now, url: location.href,
             scrollY: window.scrollY || (el && el.scrollTop) || 0,
             deltaX: 0, deltaY: 0};
    if (el && el !== document && el !== document.documentElement) {
      var d = window.__rpaDescribe(el);
      if (d) { b.tiers = d.tiers; b.tag = d.tag; }
    }
    // 滚动事件高频：Go 侧还会聚类，这里只做粗粒度丢弃（同 200ms 内合并）
    if (now - lastScrollAt < 200) {
      var last = window.__rpaRec.events[window.__rpaRec.events.length - 1];
      if (last && last.kind === "scroll") { last.scrollY = b.scrollY; last.ts = now; return; }
    }
    lastScrollAt = now;
    window.__rpaRec.push(b);
  }, true);

  document.addEventListener("keydown", function(e){
    // 只录「组合键」与功能键：普通字符由 input 事件覆盖，否则会录出
    // 一串无意义的 keyboard 动作。
    var mods = (e.ctrlKey ? 2 : 0) | (e.shiftKey ? 8 : 0) | (e.altKey ? 1 : 0) | (e.metaKey ? 4 : 0);
    var fnKeys = ["Enter", "Tab", "Escape", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
                  "Home", "End", "PageUp", "PageDown", "Delete", "Backspace"];
    var isFn = fnKeys.indexOf(e.key) >= 0;
    if (mods === 0 && !isFn) { return; }
    if (mods === 8 && !isFn) { return; }
    var el = target(e.target);
    var b = el ? base(el) : null;
    if (!b) {
      b = {ts: Date.now(), url: location.href, tiers: []};
    }
    b.kind = "keydown";
    b.key = e.key;
    b.modifiers = mods;
    window.__rpaRec.push(b);
  }, true);

  return true;
})()`

// installScript 注入监听脚本到每个新文档（evalOnNewDocument 语义）。
//
// 注意 rod 的 proto.PageAddScriptToEvaluateOnNewDocument 是**持久**的：
// 一次注入对后续所有导航生效。因此 Stop 时必须显式 RemoveScriptToEvaluate，
// 否则同一 page 复用于下一次执行时会继续录音（既浪费也可能泄漏数据）。
func (r *Recorder) installScript() (proto.PageScriptIdentifier, error) {
	if r == nil || r.page == nil {
		return "", fmt.Errorf("录制器未绑定页面")
	}
	res, err := proto.PageAddScriptToEvaluateOnNewDocument{
		Source: recorderRuntimeJS,
	}.Call(r.page)
	if err != nil {
		return "", fmt.Errorf("注入录制监听脚本失败: %w", err)
	}
	return res.Identifier, nil
}

// Install 注入监听脚本（仅对新文档生效）。
func (r *Recorder) Install() error {
	id, err := r.installScript()
	if err != nil {
		return err
	}
	r.scriptID = id
	return nil
}

// Start 开始录制：注入 + 在当前已加载文档里也装一份 + 记下起点。
//
// 「当前文档也装一份」是必要的一步：用户常常先打开目标页面再点「开始录制」，
// 若只依赖 evalOnNewDocument，不导航就永远录不到东西。
func (r *Recorder) Start() error {
	if r == nil || r.page == nil {
		return fmt.Errorf("录制器未绑定页面")
	}
	if err := r.Install(); err != nil {
		return err
	}
	// rod 的 Eval 把入参当函数体 apply()，裸 IIFE 会报 "apply is not a function"；
	// 包一层箭头函数即可（注入路径 Source 则必须是裸表达式，故两者不能共用形态）。
	if _, err := r.page.Eval("() => { " + recorderRuntimeJS + "\n return true; }"); err != nil {
		return fmt.Errorf("当前页面安装录制监听失败: %w", err)
	}
	// 清空并开启（含跨导航持久化缓冲，否则上一次录制的残留会被接回来）。
	if _, err := r.page.Eval(`() => {
      try {
        sessionStorage.removeItem("__rpaRecEvents");
        sessionStorage.setItem("__rpaRecOn", "1");
      } catch (e) {}
      if (window.__rpaRec) {
        window.__rpaRec.events = [];
        window.__rpaRec.enabled = true;
        window.__rpaRec.overflow = false;
        window.__rpaRec.lastPersist = 0;
      }
      return true;
    }`); err != nil {
		r.logger.Debug("清空录制缓冲失败（继续）", logger.Err(err))
	}
	r.startURL = r.currentURL()
	r.startTS = time.Now().UnixMilli()
	r.recording = true
	r.logger.Info("开始录制", logger.String("url", r.startURL))
	return nil
}

// scriptID 注入脚本句柄（Stop 时移除）。
func (r *Recorder) currentURL() string {
	if r == nil || r.page == nil {
		return ""
	}
	v, err := r.page.Eval(`() => location.href`)
	if err != nil {
		return ""
	}
	s, _ := v.Value.Val().(string)
	return s
}

// Events 拉取并清空当前缓冲的事件。
func (r *Recorder) Events() ([]RecordedEvent, error) {
	if r == nil || r.page == nil {
		return nil, fmt.Errorf("录制器未绑定页面")
	}
	// 拉取时优先取 sessionStorage 的全量（跨导航累积的那份），内存缓冲只是
	// 当前文档的增量。两者取并集会重复，故以持久化为准，内存作为兜底。
	res, err := r.page.Eval(`() => {
      var out = {events: [], overflow: false, fromStorage: false};
      try {
        if (sessionStorage.getItem("__rpaRecOn") === "1") {
          var saved = sessionStorage.getItem("__rpaRecEvents");
          if (saved) {
            var arr = JSON.parse(saved);
            if (arr && arr.length) { out.events = arr; out.fromStorage = true; }
          }
        }
      } catch (e) {}
      if (!out.fromStorage && window.__rpaRec) {
        out.events = window.__rpaRec.events.slice(0);
        out.overflow = window.__rpaRec.overflow;
      }
      // 清空两边，避免重复拉取
      try { sessionStorage.removeItem("__rpaRecEvents"); } catch (e) {}
      if (window.__rpaRec) {
        window.__rpaRec.events = [];
        window.__rpaRec.overflow = false;
        window.__rpaRec.lastPersist = 0;
      }
      return out;
    }`)
	if err != nil {
		return nil, fmt.Errorf("拉取录制事件失败: %w", err)
	}
	raw := res.Value.Val()
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	if ov, ok := m["overflow"].(bool); ok && ov {
		r.logger.Warn("录制事件超出上限，早期事件可能被丢弃")
	}
	list, ok := m["events"].([]interface{})
	if !ok {
		return nil, nil
	}
	out := make([]RecordedEvent, 0, len(list))
	for _, item := range list {
		buf, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var ev RecordedEvent
		if err := json.Unmarshal(buf, &ev); err != nil {
			continue
		}
		if ev.Ts < r.startTS {
			continue // 安装前残留
		}
		out = append(out, ev)
	}
	return out, nil
}

// Stop 结束录制：停用监听、移除注入脚本、返回累积事件。
//
// 事件到动作的转换交给纯函数 BuildActions（可单测，不依赖浏览器）。
func (r *Recorder) Stop() ([]RecordedEvent, error) {
	if r == nil || r.page == nil {
		return nil, fmt.Errorf("录制器未绑定页面")
	}
	events, err := r.Events()
	if err != nil {
		return nil, err
	}
	// 停用 + 移除持久注入（避免 page 复用时继续录音）。
	if _, evalErr := r.page.Eval(`() => {
      if (window.__rpaRec) { window.__rpaRec.enabled = false; }
      try { sessionStorage.removeItem("__rpaRecOn"); } catch (e) {}
      return true;
    }`); evalErr != nil {
		r.logger.Debug("停用录制监听失败（继续）", logger.Err(evalErr))
	}
	if r.scriptID != "" {
		if err := (proto.PageRemoveScriptToEvaluateOnNewDocument{Identifier: r.scriptID}).Call(r.page); err != nil {
			r.logger.Debug("移除录制注入脚本失败（继续）", logger.Err(err))
		}
		r.scriptID = ""
	}
	r.recording = false
	r.logger.Info("录制结束", logger.Int("events", len(events)))
	return events, nil
}

// Recording 是否正在录制。
func (r *Recorder) Recording() bool { return r != nil && r.recording }

// ============================================================================
// 事件 → 动作（纯函数，无浏览器依赖，便于单测）
// ============================================================================

// 聚类参数（抽成变量便于单测缩短窗口）。
var (
	// inputDebounceMS 同一元素连续输入合并窗口：窗口内只保留最后一个值。
	inputDebounceMS int64 = 600
	// scrollDebounceMS 连续滚动合并窗口。
	scrollDebounceMS int64 = 800
	// submitAfterClickMS 点击后紧跟的 submit 视为派生事件（丢弃 submit）。
	submitAfterClickMS int64 = 1500
)

// BuildActions 把原始录制事件聚类成可执行动作序列。
//
// 聚类规则（每条都对应一个真实录制噪声源）：
//  1. 起点插入 navigate 动作（录制必须能独立回放，不能依赖「当前已在某页」）。
//  2. 同一元素 input/change 在窗口内合并 —— 否则「逐字符输入」会产出几十条 fill。
//  3. click 后紧邻的 submit 丢弃 —— 点击提交按钮已经触发了提交。
//  4. 连续 scroll 合并为一条，且换算成方向 + 像素（回放语义）。
//  5. 密码框的值替换为 ${password} 占位 —— 录制绝不把明文密码写进任务。
//  6. URL 变化插入 navigate 动作（SPA 跳转与整页跳转统一处理）。
func BuildActions(events []RecordedEvent) []types.Action {
	actions := make([]types.Action, 0, len(events)+1)
	if len(events) == 0 {
		return actions
	}

	// 起点 URL：取第一个事件里的 url（事件都带 location.href）。
	startURL := events[0].URL
	if startURL != "" {
		actions = append(actions, types.Action{
			ID:          "recorded_1",
			Type:        types.ActionNavigate,
			Description: "打开网页",
			Params:      map[string]interface{}{"url": startURL},
			Timeout:     defaultRecordedTimeoutMS,
		})
	}

	lastURL := startURL
	// pendingInput 同一元素的待合并输入（key = 选择器 → actions 下标）。
	pendingInput := map[string]int{}
	// pendingInputTs 对应下标最近一次更新的时刻（决定是否仍在合并窗口内）。
	pendingInputTs := map[string]int64{}
	lastScrollIdx := -1
	lastScrollTs := int64(0)
	lastClickTs := int64(0)
	lastClickSel := ""

	for _, ev := range events {
		// URL 变化 → 插入 navigate（与上一个动作同 URL 则跳过）。
		if ev.URL != "" && ev.URL != lastURL {
			lastURL = ev.URL
			actions = append(actions, types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        types.ActionNavigate,
				Description: "打开网页",
				Params:      map[string]interface{}{"url": ev.URL},
				Timeout:     defaultRecordedTimeoutMS,
			})
			pendingInput = map[string]int{}
			pendingInputTs = map[string]int64{}
			lastScrollIdx = -1
		}

		switch ev.Kind {
		case "click", "dblclick", "contextmenu":
			sel, ok := bestSelector(ev.Tiers)
			if !ok || sel == "" {
				continue
			}
			// 复选框/单选框：点击态变化更适合用 select 语义（幂等）。
			at := types.ActionClick
			desc := "点击按钮"
			switch ev.Kind {
			case "dblclick":
				at = types.ActionDoubleClick
				desc = "双击"
			case "contextmenu":
				at = types.ActionRightClick
				desc = "右键点击"
			}
			if ev.Kind == "click" && (ev.Type == "checkbox" || ev.Type == "radio") {
				at = types.ActionSelect
				desc = "勾选选项"
			}
			a := types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        at,
				Description: describeClick(desc, ev),
				Selector:    sel,
				Timeout:     defaultRecordedTimeoutMS,
			}
			if at == types.ActionSelect {
				a.Params = map[string]interface{}{"value": boolText(ev.Checked)}
			}
			actions = append(actions, a)
			lastClickTs = ev.Ts
			lastClickSel = sel
			delete(pendingInput, sel)
			delete(pendingInputTs, sel)
			lastScrollIdx = -1

		case "input", "change", "select":
			sel, ok := bestSelector(ev.Tiers)
			if !ok || sel == "" {
				continue
			}
			// 复选框的 change 与 click 重复 → 已在 click 分支处理。
			if ev.Type == "checkbox" || ev.Type == "radio" {
				continue
			}
			value := ev.Value
			desc := "填入内容"
			at := types.ActionFill
			if ev.Kind == "select" || ev.Tag == "select" {
				at = types.ActionSelect
				desc = "选择选项"
			}
			// 密码框：绝不明文落库。
			if ev.Type == "password" {
				value = "${password}"
			}
			if idx, ok := pendingInput[sel]; ok && ev.Ts-pendingInputTs[sel] <= inputDebounceMS {
				// 窗口内同一元素：覆盖上一条（只保留最终值，避免逐字符产出几十条）
				actions[idx].Params["value"] = value
				actions[idx].Description = describeFill(desc, ev)
				pendingInputTs[sel] = ev.Ts
				continue
			}
			a := types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        at,
				Description: describeFill(desc, ev),
				Selector:    sel,
				Params:      map[string]interface{}{"value": value},
				Timeout:     defaultRecordedTimeoutMS,
			}
			actions = append(actions, a)
			pendingInput[sel] = len(actions) - 1
			pendingInputTs[sel] = ev.Ts
			lastScrollIdx = -1

		case "submit":
			// 点击提交按钮后的派生 submit 丢弃（否则回放会提交两次）。
			if lastClickSel != "" && ev.Ts-lastClickTs <= submitAfterClickMS {
				continue
			}
			actions = append(actions, types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        types.ActionClick,
				Description: "提交表单",
				Selector:    bestSelectorOrEmpty(ev.Tiers),
				Timeout:     defaultRecordedTimeoutMS,
			})
			lastScrollIdx = -1

		case "scroll":
			// 连续滚动合并
			if lastScrollIdx >= 0 && ev.Ts-lastScrollTs <= scrollDebounceMS {
				mergeScroll(&actions[lastScrollIdx], ev)
				lastScrollTs = ev.Ts
				continue
			}
			a := types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        types.ActionScroll,
				Description: "向下滚动页面",
				Params:      map[string]interface{}{"direction": "down", "pixels": 500},
				Timeout:     defaultRecordedTimeoutMS,
			}
			if ev.ScrollY > 0 {
				a.Params["pixels"] = int(ev.ScrollY)
			}
			actions = append(actions, a)
			lastScrollIdx = len(actions) - 1
			lastScrollTs = ev.Ts

		case "keydown":
			combo := keyCombo(ev.Key, ev.Modifiers)
			if combo == "" {
				continue
			}
			actions = append(actions, types.Action{
				ID:          actionID(len(actions) + 1),
				Type:        types.ActionKeyboard,
				Description: describeKey(combo),
				Params:      map[string]interface{}{"keys": combo},
				Timeout:     defaultRecordedTimeoutMS,
			})
			lastScrollIdx = -1
		}
	}

	// 二次清洗：合并被 navigate 切断的同元素输入（已由 pendingInput 处理），
	// 并去掉连续重复的完全相同动作（录制抖动）。
	return dedupeAdjacent(actions)
}

const defaultRecordedTimeoutMS = 30000

func actionID(n int) string { return fmt.Sprintf("recorded_%d", n) }

// bestSelector 按阶梯顺序取第一个「唯一可用」的候选。
//
// 阶梯顺序即 tier 升序（1..7），与 selector_ladder.go 的 tierScores 单调递减
// 一一对应 —— 因此这里只按 tier 升序取第一个 ok，不需要再比分数。
func bestSelector(tiers []RecordedTier) (string, bool) {
	sorted := make([]RecordedTier, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Tier < sorted[j].Tier })
	for _, t := range sorted {
		if t.OK && strings.TrimSpace(t.Selector) != "" {
			return t.Selector, true
		}
	}
	return "", false
}

func bestSelectorOrEmpty(tiers []RecordedTier) string {
	s, _ := bestSelector(tiers)
	return s
}

// mergeScroll 把后一条滚动并入前一条（取绝对值更大的位移）。
func mergeScroll(a *types.Action, ev RecordedEvent) {
	prev, _ := a.Params["pixels"].(int)
	next := int(ev.ScrollY)
	if next > prev {
		a.Params["pixels"] = next
	}
}

// boolText 布尔 → select 的 value 文本。
func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// keyCombo 事件 → 组合键字符串（与 parseKeyCombo 的 "ctrl+shift+a" 语法一致）。
func keyCombo(key string, mods int) string {
	if key == "" {
		return ""
	}
	parts := make([]string, 0, 4)
	if mods&2 != 0 {
		parts = append(parts, "ctrl")
	}
	if mods&1 != 0 {
		parts = append(parts, "alt")
	}
	if mods&8 != 0 {
		parts = append(parts, "shift")
	}
	if mods&4 != 0 {
		parts = append(parts, "meta")
	}
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case "control", "ctrl":
		return "" // 单独按修饰键没有语义
	case "shift", "alt", "meta":
		return ""
	case " ":
		k = "space"
	case "escape", "esc":
		k = "esc"
	}
	parts = append(parts, k)
	return strings.Join(parts, "+")
}

// dedupeAdjacent 去掉完全相同的相邻动作（录制抖动产生的重复）。
func dedupeAdjacent(actions []types.Action) []types.Action {
	if len(actions) < 2 {
		return actions
	}
	out := make([]types.Action, 0, len(actions))
	for _, a := range actions {
		if len(out) > 0 {
			p := out[len(out)-1]
			if p.Type == a.Type && p.Selector == a.Selector &&
				paramString(p.Params, "value") == paramString(a.Params, "value") &&
				paramString(p.Params, "url") == paramString(a.Params, "url") &&
				paramString(p.Params, "keys") == paramString(a.Params, "keys") {
				continue
			}
		}
		out = append(out, a)
	}
	// 重新编号，保持 id 连续（前端步骤列表按 id 排序）。
	for i := range out {
		out[i].ID = actionID(i + 1)
	}
	return out
}

func paramString(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	s, _ := params[key].(string)
	return s
}

// ============================================================================
// 中文步骤描述（NOCODE-01/04：业务语义，零技术术语）
// ============================================================================

// describeClick 生成「点击按钮/点击链接」这类业务语义描述。
//
// 硬约束（NOCODE-04）：**不暴露** http 方法 / selector / JSON 术语。
func describeClick(base string, ev RecordedEvent) string {
	target := humanTarget(ev)
	if target == "" {
		return base
	}
	switch ev.Tag {
	case "a":
		return "点击链接「" + target + "」"
	case "button":
		return "点击按钮「" + target + "」"
	case "input":
		if ev.Type == "checkbox" || ev.Type == "radio" {
			return "勾选「" + target + "」"
		}
		return "点击「" + target + "」"
	case "label":
		return "点击「" + target + "」"
	default:
		return "点击「" + target + "」"
	}
}

// describeFill 生成「在「用户名」填入内容」这类描述。
//
// 密码框的「密码」语义不能因为拿不到可读标签而丢失 —— 兜底文案也要区分，
// 否则用户看到「填入内容」根本不知道这一步是敏感字段。
// 下拉选择用「选择」，与「填入」区分（回放时前者走 select 语义）。
func describeFill(base string, ev RecordedEvent) string {
	isPassword := ev.Type == "password"
	isSelect := ev.Kind == "select" || ev.Tag == "select"
	// 下拉的「人话标签」优先用选项文本（用户选的是「住宿费」而不是字段名）。
	target := humanTarget(ev)
	if isSelect && strings.TrimSpace(ev.Text) != "" && len([]rune(ev.Text)) <= 20 {
		target = strings.TrimSpace(ev.Text)
	}
	if target == "" {
		if isPassword {
			return "填入密码"
		}
		if isSelect {
			return "选择选项"
		}
		return base
	}
	if isPassword {
		return "在「" + target + "」填入密码"
	}
	if isSelect {
		return "选择「" + target + "」"
	}
	return "在「" + target + "」填入内容"
}

// describeKey 组合键 → 中文描述。
func describeKey(combo string) string {
	switch combo {
	case "enter":
		return "按回车键"
	case "tab":
		return "按 Tab 键"
	case "esc":
		return "按 Esc 键"
	case "ctrl+a":
		return "全选"
	case "ctrl+c":
		return "复制"
	case "ctrl+v":
		return "粘贴"
	}
	return "按下 " + combo + " 组合键"
}

// humanTarget 从事件里挑一个「人话标签」：优先可见文本，其次占位提示，
// 最后才用标签名（且必须转成中文业务名，绝不出现 "div"/"input"）。
func humanTarget(ev RecordedEvent) string {
	if t := strings.TrimSpace(ev.Text); t != "" && len([]rune(t)) <= 20 {
		return t
	}
	// 从阶梯候选里找 role+name 级的名字（它带 aria-label / placeholder）
	for _, tr := range ev.Tiers {
		if tr.Tier != 2 || tr.Selector == "" {
			continue
		}
		name := strings.TrimPrefix(tr.Selector, "text=")
		name = strings.TrimSpace(name)
		if name != "" && len([]rune(name)) <= 20 {
			return name
		}
	}
	return ""
}

// ============================================================================
// 录制结果（供 API / 前端消费）
// ============================================================================

// RecordingResult 一次录制的产出（NOCODE-01 的对外结构）。
type RecordingResult struct {
	Actions []types.Action `json:"actions"`
	// Steps 中文步骤描述（与 Actions 一一对应，前端步骤列表直接渲染）。
	Steps []string `json:"steps"`
	// StartURL 录制起点（任务 targetUrl 字段的来源）。
	StartURL string `json:"startUrl"`
	// EventCount 原始事件数（诊断用：骤减说明注入失效）。
	EventCount int `json:"eventCount"`
}

// BuildRecording 从事件产出完整录制结果。
func BuildRecording(events []RecordedEvent) *RecordingResult {
	actions := BuildActions(events)
	steps := make([]string, len(actions))
	for i, a := range actions {
		steps[i] = a.Description
	}
	startURL := ""
	if len(events) > 0 {
		startURL = events[0].URL
	}
	return &RecordingResult{
		Actions:    actions,
		Steps:      steps,
		StartURL:   startURL,
		EventCount: len(events),
	}
}
