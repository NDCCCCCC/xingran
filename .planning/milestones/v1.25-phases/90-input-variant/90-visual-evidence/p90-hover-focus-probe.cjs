// Phase 90 W2 Task 3.5 / W3 Task 4.2 —— 断言 ⑥「hover/focus 回归守护」采集（明暗各一次）
//
// 判据（90-03 Task 3.5 原文）：
//   light: .ant-input:hover borderTopColor = rgb(89, 142, 94)（#598e5e）
//          .ant-input:focus borderTopColor = rgb(21, 96, 49)（#156031）
//   dark : hover = rgb(89, 142, 94)（同值）
//          focus = rgb(26, 104, 57)（#1a6839）
//   ⇒ **这三个值在 90 前后必须逐字相等**（90 不改 hover/focus/disabled）
//
// ⚠️ hover / focus 态必须**真实触发**：hover 用 CDP 的 `CSS.forcePseudoState`，
//    focus 用真实的 `el.focus()` —— 读 computed style 不会自动进入这两态。
//
// 用法：node p90-hover-focus-probe.cjs <light|dark> <输出文件名>
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");

const FIXTURE_JS = `
(function(){
  var scope = document.querySelector('.ant-app, [class*="css-var-"]');
  if (!scope) return JSON.stringify({error:'no cssVar scope'});
  var prev = document.getElementById('__p90hf');
  if (prev) prev.remove();
  var c = document.createElement('div');
  c.id = '__p90hf';
  c.style.cssText = 'position:fixed;left:-9999px;top:0;width:1200px;';
  scope.appendChild(c);
  c.innerHTML = '<div data-k="inp"><input class="ant-input" value=""></div>';
  return JSON.stringify({mounted:true});
})()`;

const READ_JS = `
(function(){
  var c=document.getElementById('__p90hf');
  var el=c.querySelector("[data-k='inp'] > .ant-input");
  var s=getComputedStyle(el);
  return JSON.stringify({borderTopColor:s.borderTopColor,borderTopWidth:s.borderTopWidth,borderTopStyle:s.borderTopStyle,borderTopLeftRadius:s.borderTopLeftRadius,fontSize:s.fontSize,backgroundColor:s.backgroundColor});
})()`;

(async () => {
  const mode = process.argv[2] || "light";
  const outFile = process.argv[3] || `hover-focus-${mode}.json`;
  const { ev, nav, cdp, close } = await openCdp();
  try {
    await nav("/login", 4000);
    const wantDark = mode === "dark";
    await ev(`(function(){
      var KEY='settings-storage';
      var o=JSON.parse(localStorage.getItem(KEY)||'{}');
      var st=o.state||o;
      st.preferences=st.preferences||{}; st.preferences.theme=st.preferences.theme||{};
      st.preferences.theme.mode=${JSON.stringify(wantDark ? "dark" : "light")};
      o.state=st; localStorage.setItem(KEY,JSON.stringify(o));
      document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)});
    })()`);
    await nav("/login", 6000);
    await ev(`document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)})`);
    await sleep(900);

    const fx = await ev(FIXTURE_JS);
    const fxObj = typeof fx === "string" ? JSON.parse(fx) : fx;
    if (fxObj.error) throw new Error(fxObj.error);
    await sleep(500);

    // ⚠️ fixture 在视口外（left:-9999px）时 **:focus 伪类不生效**（浏览器对不可见元素
    //    不激活 focus 态）→ 读到的是常态色。实测证据：isActive=true 但 borderTopColor
    //    仍是常态 rgb(219,215,206) 而非 focus 的 rgb(21,96,49)。
    // ⇒ 修正：把 fixture 移到**视口内**（position:fixed; left:0; top:0; opacity 仍可保持，
    //    但不可用 display:none —— display:none 同样不激活 focus）。
    //    改用 z-index:-1 放到最底层，视觉上不可见但参与布局与伪类。
    await ev(`(function(){
      var c=document.getElementById('__p90hf');
      c.style.cssText='position:fixed;left:0;top:0;width:1200px;z-index:-1;opacity:0.01;pointer-events:none;';
    })()`);
    await sleep(300);

    const normal = JSON.parse(await ev(READ_JS));

    // hover：CDP forcePseudoState
    const { root } = await cdp.send("DOM.getDocument", { depth: -1 });
    const { nodeId } = await cdp.send("DOM.querySelector", {
      nodeId: root.nodeId,
      selector: "#__p90hf [data-k='inp'] > .ant-input",
    });
    let hover = null;
    if (nodeId) {
      await cdp.send("CSS.enable");
      await cdp.send("DOM.enable");
      await cdp.send("CSS.forcePseudoState", { nodeId, forcedPseudoClasses: ["hover"] });
      await sleep(500);
      hover = JSON.parse(await ev(READ_JS));
      await cdp.send("CSS.forcePseudoState", { nodeId, forcedPseudoClasses: [] });
      await sleep(200);
    }

    // focus：用 CDP 的 `CSS.forcePseudoState` 强制 `:focus` 伪类。
    // ⚠️ 踩坑记录（两条，都重要）：
    //   ① `el.focus()` 在 **Runtime.evaluate 的求值上下文**里执行后，求值一结束焦点即被重置
    //      → 实测 `document.activeElement === el` 为 true，但 `el.matches(':focus')` 为 **false**
    //      → `:focus` 规则不触发 → 读到的是常态色。CDP 的 `DOM.focus` 同样不生效。
    //   ② `Input.dispatchKeyEvent(Tab)` 会把焦点移到**页面上别的**元素（登录页自身有输入框，
    //      fixture 不是第一个可聚焦项）→ 实测 `isActive: false`。
    // ⇒ 唯一可靠路径 = 与 hover 同一机制的 `CSS.forcePseudoState`（它直接给元素打伪类标记，
    //    不依赖真实焦点归属）。
    await cdp.send("CSS.forcePseudoState", { nodeId, forcedPseudoClasses: ["focus"] });
    await sleep(500);
    const focus = JSON.parse(
      await ev(`(function(){
        var c=document.getElementById('__p90hf');
        var el=c.querySelector("[data-k='inp'] > .ant-input");
        var s=getComputedStyle(el);
        return JSON.stringify({
          borderTopColor:s.borderTopColor,borderTopWidth:s.borderTopWidth,borderTopStyle:s.borderTopStyle,
          borderTopLeftRadius:s.borderTopLeftRadius,boxShadow:s.boxShadow,fontSize:s.fontSize,
          backgroundColor:s.backgroundColor,
          matchesFocus: el.matches(':focus')
        });
      })()`),
    );
    await cdp.send("CSS.forcePseudoState", { nodeId, forcedPseudoClasses: [] });

    const payload = {
      mode,
      capturedAt: new Date().toISOString(),
      normal,
      hover,
      focus,
    };
    fs.writeFileSync(path.join(HERE, outFile), JSON.stringify(payload, null, 1));
    console.log(JSON.stringify(payload, null, 1));
    console.log(`\n→ ${outFile}`);
  } finally {
    close();
  }
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
