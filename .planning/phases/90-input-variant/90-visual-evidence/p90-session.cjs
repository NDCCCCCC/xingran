// Phase 90 W2 执行期辅助脚本 —— 登录 + 设置页模式切换 + 探针取样的公共底座
// 用途：W2 的 T4 实测（删 font-size 前后各采一份）与 W3 的 computed 六断言 / 巡检都复用它。
// 依赖：CDP 已在 127.0.0.1:9222 监听；dev server 在 localhost:4000 运行。
const path = require("path");
const fs = require("fs");
const FE_ROOT = path.resolve(__dirname, "../../../../ruoyi-react-frontend");
const WebSocket = require(FE_ROOT + "/node_modules/ws");

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function connect(wsUrl) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl, { perMessageDeflate: false });
    let id = 0;
    const pending = new Map();
    ws.on("open", () =>
      resolve({
        send(method, params) {
          return new Promise((res, rej) => {
            const mid = ++id;
            pending.set(mid, { res, rej });
            ws.send(JSON.stringify({ id: mid, method, params }));
          });
        },
        close() {
          ws.close();
        },
      }),
    );
    ws.on("message", (d) => {
      const m = JSON.parse(d);
      if (m.id && pending.has(m.id)) {
        const p = pending.get(m.id);
        pending.delete(m.id);
        m.error ? p.rej(new Error(JSON.stringify(m.error))) : p.res(m.result);
      }
    });
    ws.on("error", reject);
  });
}

/** 连接 CDP 并返回 { cdp, ev, nav, close } —— ev(expr, awaitPromise) 求值，nav(url, wait) 导航 */
async function openCdp() {
  const list = await (await fetch("http://127.0.0.1:9222/json")).json();
  const page = list.find((t) => t.type === "page");
  if (!page) throw new Error("CDP 无 page target —— 9222 端口是否有 Chrome 在跑？");
  const cdp = await connect(page.webSocketDebuggerUrl);
  await cdp.send("Page.enable");
  await cdp.send("Runtime.enable");
  const ev = async (expr, awaitPromise = false) => {
    const r = await cdp.send("Runtime.evaluate", {
      expression: expr,
      awaitPromise,
      returnByValue: true,
    });
    if (r.exceptionDetails) {
      throw new Error("EVAL: " + JSON.stringify(r.exceptionDetails).slice(0, 600));
    }
    return r.result.value;
  };
  const nav = async (url, wait = 4000) => {
    await cdp.send("Page.navigate", { url: "http://localhost:4000" + url });
    await sleep(wait);
  };
  return { cdp, ev, nav, close: () => cdp.close() };
}

/**
 * 登录：admin / admin123。
 * ⚠️ 登录页的 input 在 React 挂载完成前不存在 —— 必须**轮询等待**，
 *    不能导航完就立即查（这是 89/90 探针脚本首次执行报 "NO_INPUTS" 的根因）。
 */
async function login(ev, nav) {
  await nav("/login", 5000);
  let href = await ev("location.href");
  if (!/login/.test(href)) return "ALREADY_LOGGED_IN " + href;

  const ready = await ev(`(function(){
    var inputs = Array.from(document.querySelectorAll('input'));
    var u = inputs.find(i => i.type === 'text' || i.type === 'email');
    var p = inputs.find(i => i.type === 'password');
    return (u && p) ? 'READY' : 'WAIT';
  })()`);
  if (ready !== "READY") {
    // 再等一轮（React 懒加载 + 首屏渲染）
    await sleep(3000);
    const again = await ev(`(function(){
      var inputs = Array.from(document.querySelectorAll('input'));
      return (inputs.some(i=>i.type==='text'||i.type==='email') && inputs.some(i=>i.type==='password')) ? 'READY' : 'WAIT';
    })()`);
    if (again !== "READY") return "NO_INPUTS";
  }

  const r = await ev(
    `(async function(){
      function setNative(el, v){
        var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype;
        var s = Object.getOwnPropertyDescriptor(proto, 'value').set;
        s.call(el, v); el.dispatchEvent(new Event('input', {bubbles:true}));
      }
      var inputs = Array.from(document.querySelectorAll('input'));
      var u = inputs.find(i => i.type === 'text' || i.type === 'email');
      var p = inputs.find(i => i.type === 'password');
      setNative(u, 'admin'); setNative(p, 'admin123');
      await new Promise(r=>setTimeout(r,500));
      var btn = Array.from(document.querySelectorAll('button')).find(b => /登/.test(b.textContent));
      if (!btn) return 'NO_BTN';
      btn.click();
      await new Promise(r=>setTimeout(r,5000));
      return 'OK ' + location.href;
    })()`,
    true,
  );
  return r;
}

/**
 * 切换明暗模式：走设置页的分段卡（antd token 的真实来源 = settingsStore 的服务器偏好）。
 * idx 0 = light，1 = dark。
 */
async function setMode(ev, nav, mode) {
  await nav("/user/settings", 4000);
  const count = await ev(`document.querySelectorAll('button.xr-settings-segmented-card').length`);
  if (count < 2) {
    // 设置页未渲染出分段卡 —— 可能是路由或权限问题，显式报错而非静默跳过
    throw new Error(
      `setMode(${mode}): 设置页分段卡未找到（count=${count}），href=${await ev("location.href")}`,
    );
  }
  const idx = mode === "dark" ? 1 : 0;
  await ev(`(function(){
    var cards = document.querySelectorAll('button.xr-settings-segmented-card');
    cards[${idx}].click();
  })()`);
  await sleep(1500);
  const dom = await ev(`document.documentElement.getAttribute('data-color-mode')`);
  if (dom !== mode) throw new Error(`setMode mismatch: want=${mode} got=${dom}`);
  return dom;
}

module.exports = { openCdp, login, setMode, sleep };

// CLI 用法：node p90-session.cjs <mode>   → 登录 + 切模式 + 打印探针关键值
if (require.main === module) {
  (async () => {
    const mode = process.argv[2] || "light";
    const { ev, nav, close } = await openCdp();
    try {
      const loginResult = await login(ev, nav);
      console.log("LOGIN:", loginResult);
      if (!/OK|ALREADY/.test(loginResult)) {
        console.error("LOGIN FAILED");
        process.exit(1);
      }
      const m = await setMode(ev, nav, mode);
      console.log("MODE:", m);
      await nav("/system/user", 4500);
      const out = await ev(`(function(){
        var scope = document.querySelector('.ant-app') || document.querySelector('[class*="css-var-"]');
        var sc = scope ? getComputedStyle(scope) : getComputedStyle(document.documentElement);
        var inp = document.querySelector('.ant-input');
        var c = inp ? getComputedStyle(inp) : null;
        return JSON.stringify({
          mode: document.documentElement.getAttribute('data-color-mode'),
          antColorError: sc.getPropertyValue('--ant-color-error').trim(),
          antColorWarning: sc.getPropertyValue('--ant-color-warning').trim(),
          inputFontSize: c ? c.fontSize : null,
          inputRadius: c ? c.borderTopLeftRadius : null,
        }, null, 1);
      })()`);
      console.log(out);
    } finally {
      close();
    }
    process.exit(0);
  })().catch((e) => {
    console.error("FATAL", e);
    process.exit(1);
  });
}
