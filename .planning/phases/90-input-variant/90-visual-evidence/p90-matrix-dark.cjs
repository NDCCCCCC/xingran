// Phase 90 W3 —— 暗色矩阵采样（无登录态，走 settingsStore 的持久化通道）
//
// 机制说明：
//   暗色由 `settingsStore.preferences.theme.mode` 驱动 → `AntdThemeBridge` 据此选
//   `antdTheme.darkAlgorithm` → antd 在 runtime 重算 cssVar 层的 `--ant-*` 派生值。
//   该 store 用 zustand persist 落在 localStorage 的 `settings-storage` 键。
//   ⇒ **未登录态下也能切暗色**：改 localStorage 的 persist 快照 → 刷新页面 →
//     store 的 rehydrate 读出新 mode → antd 以 darkAlgorithm 渲染。
//   这比「直接改 data-color-mode 属性」更准：后者只切项目 CSS 的暗色分区，
//   **不会**触发 antd 的 darkAlgorithm 重算（实测 `--ant-color-error` 仍是 light 值 #ba3630）。
//
// 用法：node p90-matrix-dark.cjs <输出文件名>
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");

const fixtureJs = fs.readFileSync(path.join(HERE, "matrix-fixture.js"), "utf8");
const sampleJs = fs.readFileSync(path.join(HERE, "matrix-sample.js"), "utf8");

/** 把 settings-storage 的 persist 快照改成 dark 模式 */
async function forceDarkStore(ev) {
  return await ev(`(function(){
    var KEY = 'settings-storage';
    var raw = localStorage.getItem(KEY);
    if (!raw) return 'NO_SETTINGS_STORAGE';
    var o = JSON.parse(raw);
    var st = o.state || o;
    st.preferences = st.preferences || {};
    st.preferences.theme = st.preferences.theme || {};
    st.preferences.theme.mode = 'dark';
    o.state = st;
    localStorage.setItem(KEY, JSON.stringify(o));
    return 'SET_DARK';
  })()`);
}

(async () => {
  const outFile = process.argv[2] || "after-dark-nologin.json";
  const { ev, nav, close } = await openCdp();
  try {
    await nav("/login", 4000);
    const setRes = await forceDarkStore(ev);
    console.log("store:", setRes);
    if (setRes !== "SET_DARK") throw new Error("无法改写 settings-storage");

    // 刷新让 store rehydrate
    await nav("/login", 6000);

    // store 的 rehydrate 决定了 antd 的 algorithm（暗色派生值已生效，见下），
    // 但**项目 CSS 的暗色分区**由 <html data-color-mode="dark"> 选择器驱动
    // （`--input-bg` 等项目变量的暗色值在该作用域下定义）——
    // store 的 mode 与 DOM 属性是**两条独立通道**（ThemeProvider 负责写 DOM 属性，
    // 但它订阅的是 themeStore.mode 而非 settingsStore.preferences.theme.mode）。
    // ⇒ 此处显式补写 DOM 属性，让项目变量也进入暗色分区。
    await ev(`(function(){ document.documentElement.setAttribute('data-color-mode','dark'); })()`);
    await sleep(800);

    const domMode = await ev(`document.documentElement.getAttribute('data-color-mode')`);
    console.log("data-color-mode:", domMode);

    const vars1 = await ev(`(function(){
      var scope = document.querySelector('.ant-app, [class*="css-var-"]');
      var sc = scope ? getComputedStyle(scope) : getComputedStyle(document.documentElement);
      return JSON.stringify({
        antColorError: sc.getPropertyValue('--ant-color-error').trim(),
        antColorBgContainer: sc.getPropertyValue('--ant-color-bg-container').trim(),
        antColorBorder: sc.getPropertyValue('--ant-color-border').trim()
      });
    })()`);
    console.log("刷新后 antd 变量:", vars1);

    const fixtureOut = await ev(fixtureJs);
    const fx = typeof fixtureOut === "string" ? JSON.parse(fixtureOut) : fixtureOut;
    if (fx.error) throw new Error("fixture 挂载失败：" + fx.error);
    await sleep(800);

    const sampleOut = await ev(sampleJs);
    const parsed = JSON.parse(sampleOut);
    const forms = parsed.forms || {};

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
      domMode,
      capturedAt: new Date().toISOString(),
      cssVars: JSON.parse(vars),
      forms: parsed.forms,
    };
    fs.writeFileSync(path.join(HERE, outFile), JSON.stringify(payload, null, 1));
    console.log(`MODE=${parsed.mode} domMode=${domMode} 形态数=${Object.keys(forms).length} → ${outFile}`);
    console.log(`CSS 变量: ${vars}`);
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
