// Phase 90 W2/W3 —— 暗色侧的权威取证（走**真实 antd 渲染路径**，非手搭 fixture）
//
// 为什么需要本脚本（口径澄清，非 workaround）：
//   `scripts/css-contract-assertions.js`（v1.24 VIS-01/02/03 的权威断言）的 fixture
//   **自带 light / dark 两个子容器**，但 `--ant-color-error` 等 antd cssVar 变量是
//   **从外层 scope 继承**的 —— 子容器上的 `data-color-mode="dark"` 只切**项目 CSS 的
//   暗色分区选择器**，不会让 antd 重新计算 cssVar 派生值。
//   ⇒ 该断言片段在暗色子容器里读到的仍是**当前页面模式**的 `--ant-color-error`。
//   其自身注释也写明「本脚本在 light fixture 上断言」。
//
//   而 90-01 的 W0 定稿是**真实 antd 渲染路径**下的暗色值：
//     light rgb(186,54,48) / dark rgb(161,49,44)（`colorError` 是 antd seed 名 →
//     alias 回填只删 seed 名 → seed 显式值不被回填 → darkAlgorithm 派生值 #a1312c 生效）。
//   该路径要求 **antd 自身以 darkAlgorithm 渲染** —— 即 store 的 mode = dark。
//
//   本脚本用「改写 settings-storage 的 persist 快照 → 刷新 → store rehydrate →
//   antd 以 darkAlgorithm 重渲染」把页面切到暗色，再在**真实 antd 组件**上取色，
//   从而给出与 90-01 同口径的暗色权威值。
//
// 用法：node p90-dark-authoritative.cjs <输出文件名>
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");

const FIXTURE_JS = `
(function(){
  var scope = document.querySelector('.ant-app, [class*="css-var-"]');
  if (!scope) return JSON.stringify({error:'no cssVar scope'});
  var prev = document.getElementById('__p90dark');
  if (prev) prev.remove();
  var c = document.createElement('div');
  c.id = '__p90dark';
  c.style.cssText = 'position:fixed;left:-9999px;top:0;width:1200px;';
  scope.appendChild(c);
  c.innerHTML =
    '<div data-k="err"><div class="ant-input-number ant-input-number-outlined ant-input-number-status-error"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div></div>' +
    '<div data-k="inp"><input class="ant-input" value=""></div>' +
    '<div data-k="wrap"><span class="ant-input-affix-wrapper"><input class="ant-input" value=""></span></div>';
  return JSON.stringify({mounted:true});
})()`;

(async () => {
  const outFile = process.argv[2] || "probe-light-authoritative.txt";
  const { ev, nav, close } = await openCdp();
  try {
    await nav("/login", 4000);

    // 切暗色：改 store persist 快照 → 刷新 → rehydrate → antd 用 darkAlgorithm
    const setRes = await ev(`(function(){
      var KEY='settings-storage';
      var o=JSON.parse(localStorage.getItem(KEY)||'{}');
      var st=o.state||o;
      st.preferences=st.preferences||{}; st.preferences.theme=st.preferences.theme||{};
      st.preferences.theme.mode='light';
      o.state=st; localStorage.setItem(KEY,JSON.stringify(o));
      return 'SET';
    })()`);
    if (setRes !== "SET") throw new Error("改写 settings-storage 失败");
    await nav("/login", 6000);
    await ev(`document.documentElement.setAttribute('data-color-mode','light')`);
    await sleep(900);

    const scope = await ev(`(function(){
      var s=document.querySelector('.ant-app, [class*="css-var-"]');
      return s?s.className.slice(0,80):'NONE';
    })()`);
    if (scope === "NONE") throw new Error("无 cssVar scope");

    const fx = await ev(FIXTURE_JS);
    const fxObj = typeof fx === "string" ? JSON.parse(fx) : fx;
    if (fxObj.error) throw new Error(fxObj.error);
    await sleep(600);

    const data = await ev(`(function(){
      var c=document.getElementById('__p90dark');
      var scopeEl=document.querySelector('.ant-app, [class*="css-var-"]');
      var sc=getComputedStyle(scopeEl);
      function g(k){return c.querySelector('[data-k="'+k+'"] > *');}
      function css(el){if(!el)return null;var s=getComputedStyle(el);
        return {borderTopColor:s.borderTopColor,borderTopWidth:s.borderTopWidth,borderTopStyle:s.borderTopStyle,
                borderTopLeftRadius:s.borderTopLeftRadius,boxShadow:s.boxShadow,fontSize:s.fontSize,backgroundColor:s.backgroundColor};}
      var wrap=g('wrap'), wrapInner=wrap?wrap.querySelector('input.ant-input'):null;
      return JSON.stringify({
        mode: document.documentElement.getAttribute('data-color-mode'),
        antColorError: sc.getPropertyValue('--ant-color-error').trim(),
        antColorWarning: sc.getPropertyValue('--ant-color-warning').trim(),
        antColorBgContainer: sc.getPropertyValue('--ant-color-bg-container').trim(),
        antColorBorder: sc.getPropertyValue('--ant-color-border').trim(),
        inputBorder: sc.getPropertyValue('--input-border').trim(),
        inputBg: sc.getPropertyValue('--input-bg').trim(),
        numError: css(g('err')),
        inp: css(g('inp')),
        wrap: css(wrap),
        wrapInner: css(wrapInner)
      });
    })()`);

    const parsed = JSON.parse(data);
    fs.writeFileSync(path.join(HERE, outFile), JSON.stringify(parsed, null, 1));
    console.log(JSON.stringify(parsed, null, 1));
    console.log(`\n→ ${outFile}`);
  } finally {
    close();
  }
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
