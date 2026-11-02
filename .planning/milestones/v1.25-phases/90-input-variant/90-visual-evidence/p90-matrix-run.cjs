// Phase 90 W3 —— 矩阵终采驱动（**无登录态**）
//
// 口径依据（与 90-01 的 before 快照同源）：
//   矩阵 fixture 只需 antd 的 **cssVar scope**（`[class*="css-var-"]`），而**登录页本身就渲染
//   antd 组件**（Input / Button）→ antd cssinjs 样式已注入 → fixture 可直接挂载并采样。
//   实测证据（本脚本在 2026-10-29 的运行输出）：
//     scope = "css-dev-only-do-not-override-19u5a7b ant-app css-var-_r_0_"；
//     形态数 = 22（全部命中）；f22 carrier.border = rgb(186, 54, 48)（= VISFIX-05 目标值）；
//     CSS 变量 --input-bg = #fafaf9 / --theme-radius-lg = 8px / --theme-radius-md = 6px
//   ⇒ **矩阵终跑不依赖登录态**，绕开本环境「登录需滑块验证码、无法自动化」的环境约束。
//
// ⚠️ 明暗模式切换：登录页未登录时设置页不可达 → **暗色矩阵通过在 <html> 上写
//    `data-color-mode="dark"` + 触发 store 侧的 token 重算**（见 setDarkMode 注释）。
//    若暗色采样与 89 的 after-dark 口径不一致，**以本脚本实测为准并在 CHANGESET 登记差异**。
//
// 用法：node p90-matrix-run.cjs <light|dark> <输出文件名>
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");

const fixtureJs = fs.readFileSync(path.join(HERE, "matrix-fixture.js"), "utf8");
const sampleJs = fs.readFileSync(path.join(HERE, "matrix-sample.js"), "utf8");

/**
 * 暗色切换：登录页未登录时设置页不可达（`/user/settings` 会被路由守卫踢回 /login），
 * 且登录需滑块验证码（本环境 captcha config = {"enabled":"slider"}）→ 无法自动化。
 *
 * 因此暗色走 **DOM 属性 + CSS 变量覆写** 的等效路径：
 *   1. `document.documentElement.setAttribute('data-color-mode','dark')`
 *      → 触发项目 CSS 里 `[data-color-mode="dark"]` 作用域的暗色分区变量（项目暗色变量
 *      定义在该属性选择器下，见 index.css 的暗色分区）；
 *   2. antd 的 seed/derived token 在 cssVar 层由 `--ant-*` 变量承载，其暗色派生值由
 *      antd 在 runtime 注入 —— 无登录态下 antd 的 `darkAlgorithm` 不一定生效。
 *      ⇒ 因此**暗色矩阵的 `--ant-color-error` 等 antd 派生值需要在注入前显式覆写**，
 *      否则会读到 light 值。本函数只做 (1)；(2) 由调用方按实测需要补。
 */
async function setDarkMode(ev) {
  await ev(`(function(){
    document.documentElement.setAttribute('data-color-mode','dark');
    return document.documentElement.getAttribute('data-color-mode');
  })()`);
  await sleep(800);
}

(async () => {
  const mode = process.argv[2] || "light";
  const outFile = process.argv[3] || `after-${mode}.json`;
  const { ev, nav, close } = await openCdp();
  try {
    await nav("/login", 4000);

    // ⚠️ **模式必须显式钉死**：settings-storage 的 persist 快照可能被上一轮采样改成 dark
    //    （p90-matrix-dark.cjs 会写它）→ 若不重置，本轮的「light」采样会读到 dark 派生值，
    //    导致与 before 快照口径错位。实测教训：antd 的 `--ant-color-error` 在 DOM mode=light
    //    时仍可能返回 `#a1312c`（暗色派生值）—— 因为 antd algorithm 由 store 的 mode 决定，
    //    与 DOM 属性是两条通道。
    //    ⇒ 本函数统一做两件事：① 把 store 快照写成目标模式；② 写 DOM 属性。
    const wantDark = mode === "dark";
    await ev(`(function(){
      var KEY='settings-storage';
      var o=JSON.parse(localStorage.getItem(KEY)||'{}');
      var st=o.state||o;
      st.preferences=st.preferences||{}; st.preferences.theme=st.preferences.theme||{};
      st.preferences.theme.mode=${JSON.stringify(wantDark ? "dark" : "light")};
      o.state=st; localStorage.setItem(KEY,JSON.stringify(o));
      document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)});
      return 'SET';
    })()`);
    await nav("/login", 6000); // 刷新让 store rehydrate → antd 用目标 algorithm
    await ev(`document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)})`);
    await sleep(800);

    const scopeInfo = await ev(`(function(){
      var s = document.querySelector('.ant-app, [class*="css-var-"]');
      return s ? s.className.slice(0,80) : 'NONE';
    })()`);
    if (scopeInfo === "NONE") throw new Error("无 cssVar scope —— antd 样式未注入，采样不可信");

    const fixtureOut = await ev(fixtureJs);
    const fx = typeof fixtureOut === "string" ? JSON.parse(fixtureOut) : fixtureOut;
    if (fx.error) throw new Error("fixture 挂载失败：" + fx.error);
    await sleep(800);

    const sampleOut = await ev(sampleJs);
    const parsed = JSON.parse(sampleOut);
    const forms = parsed.forms || {};
    const formCount = Object.keys(forms).length;

    const vars = await ev(`(function(){
      var root = getComputedStyle(document.documentElement);
      var scope = document.querySelector('.ant-app, [class*="css-var-"]');
      var sc = scope ? getComputedStyle(scope) : root;
      return JSON.stringify({
        inputBg: root.getPropertyValue('--input-bg').trim(),
        radiusLg: root.getPropertyValue('--theme-radius-lg').trim(),
        radiusMd: root.getPropertyValue('--theme-radius-md').trim(),
        antColorError: sc.getPropertyValue('--ant-color-error').trim(),
        antColorWarning: sc.getPropertyValue('--ant-color-warning').trim(),
        antColorBgContainer: sc.getPropertyValue('--ant-color-bg-container').trim(),
        antColorBorder: sc.getPropertyValue('--ant-color-border').trim()
      });
    })()`);

    const payload = {
      mode: parsed.mode,
      domMode: mode,
      scope: scopeInfo,
      capturedAt: new Date().toISOString(),
      cssVars: JSON.parse(vars),
      forms: parsed.forms,
    };
    fs.writeFileSync(path.join(HERE, outFile), JSON.stringify(payload, null, 1));

    console.log(`MODE=${parsed.mode} scope=${scopeInfo}`);
    console.log(`形态数=${formCount} → ${outFile}`);
    console.log(`CSS 变量: ${vars}`);
    // 摘要
    const f22 = forms.f22_input_number_status_error;
    if (f22 && f22.carrier) {
      console.log(`f22 carrier.border = ${f22.carrier.border} | radius = ${f22.carrier.radius}`);
    }
  } finally {
    close();
  }
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
