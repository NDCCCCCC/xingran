// Phase 89 CDP 驱动：登录 → 分模式（light/dark）挂 fixture 采样 + 变量探针
// 用法: node p89-cdp-run.cjs <outPrefix>
const fs = require('fs');
const path = require('path');
const FE_ROOT = path.resolve(__dirname, '../../../../xingran-react-frontend');
const WebSocket = require(FE_ROOT + '/node_modules/ws');

const fixtureJs = fs.readFileSync(path.join(__dirname, 'matrix-fixture.js'), 'utf8');
const sampleJs = fs.readFileSync(path.join(__dirname, 'matrix-sample.js'), 'utf8');
const prefix = process.argv[2] || 'after';

const probeJs = `(function(){
  var root = getComputedStyle(document.documentElement);
  var inp = document.querySelector('.ant-input');
  var sel = document.querySelector('.ant-select');
  return JSON.stringify({
    mode: document.documentElement.getAttribute('data-color-mode'),
    inputBgVar: root.getPropertyValue('--input-bg').trim(),
    inputBgFocusVar: root.getPropertyValue('--input-bg-focus').trim(),
    placeholderVar: root.getPropertyValue('--input-text-placeholder').trim(),
    warnTextVar: root.getPropertyValue('--theme-warning-text').trim(),
    inputBgComputed: inp ? getComputedStyle(inp).backgroundColor : null,
    selectBgComputed: sel ? getComputedStyle(sel).backgroundColor : null,
    bodyBg: getComputedStyle(document.body).backgroundColor
  });
})();`;

function connect(wsUrl) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl, { perMessageDeflate: false });
    let id = 0;
    const pending = new Map();
    ws.on('open', () => resolve({
      send(method, params, sessionId) {
        return new Promise((res2, rej2) => {
          const mid = ++id;
          pending.set(mid, { res: res2, rej: rej2 });
          ws.send(JSON.stringify({ id: mid, method, params, sessionId }));
        });
      },
      close() { ws.close(); },
    }));
    ws.on('message', (d) => {
      const m = JSON.parse(d);
      if (m.id && pending.has(m.id)) {
        const p = pending.get(m.id); pending.delete(m.id);
        m.error ? p.rej(new Error(JSON.stringify(m.error))) : p.res(m.result);
      }
    });
    ws.on('error', reject);
  });
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

(async () => {
  const list = await (await fetch('http://127.0.0.1:9222/json')).json();
  const page = list.find((t) => t.type === 'page');
  const cdp = await connect(page.webSocketDebuggerUrl);

  // 1. 打开登录页
  await cdp.send('Page.navigate', { url: 'http://localhost:4000/login' });
  await sleep(3500);
  let ev = await cdp.send('Runtime.evaluate', { expression: 'location.href', returnByValue: true });
  console.log('URL after nav:', ev.result.value);

  // 2. 登录（验证码 disabled；React 受控输入用原生 setter 触发）
  const loginJs = `(async function(){
    function setNative(el, v){
      var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype;
      var setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
      setter.call(el, v);
      el.dispatchEvent(new Event('input', { bubbles: true }));
    }
    var inputs = Array.from(document.querySelectorAll('input'));
    var u = inputs.find(i => i.type === 'text' || i.id.includes('username'));
    var p = inputs.find(i => i.type === 'password');
    if (!u || !p) return 'INPUTS_NOT_FOUND: ' + inputs.map(i=>i.id||i.type).join(',');
    setNative(u, 'admin'); setNative(p, 'admin123');
    await new Promise(r=>setTimeout(r,300));
    var btn = document.querySelector('button[type="submit"]') || Array.from(document.querySelectorAll('button')).find(b=>/登|login|Login/.test(b.textContent));
    if (!btn) return 'BTN_NOT_FOUND';
    btn.click();
    await new Promise(r=>setTimeout(r,3000));
    return 'OK ' + location.href;
  })()`;
  ev = await cdp.send('Runtime.evaluate', { expression: loginJs, awaitPromise: true, returnByValue: true });
  console.log('Login:', ev.result.value);
  if (!/OK/.test(ev.result.value)) { console.error('LOGIN FAILED'); process.exit(1); }
  await sleep(2000);

  // 3. 分模式采样
  const results = {};
  for (const mode of ['light', 'dark']) {
    await cdp.send('Runtime.evaluate', { expression: `document.documentElement.setAttribute('data-color-mode','${mode}')` });
    await sleep(600);
    let r = await cdp.send('Runtime.evaluate', { expression: fixtureJs, returnByValue: true });
    if (r.exceptionDetails || (r.result.value && r.result.value.error)) {
      console.error('fixture fail', mode, JSON.stringify(r.exceptionDetails || r.result.value));
      process.exit(1);
    }
    r = await cdp.send('Runtime.evaluate', { expression: sampleJs, returnByValue: true });
    if (r.exceptionDetails) { console.error('sample fail', mode, JSON.stringify(r.exceptionDetails)); process.exit(1); }
    const data = JSON.parse(r.result.value);
    if (data.mode !== mode) { console.error('mode mismatch', mode, data.mode); process.exit(1); }
    results[mode] = data;
    const pj = await cdp.send('Runtime.evaluate', { expression: probeJs, returnByValue: true });
    console.log(`PROBE ${mode}:`, pj.result.value);
  }
  for (const mode of ['light', 'dark']) {
    fs.writeFileSync(path.join(__dirname, `${prefix}-${mode}.json`), JSON.stringify(results[mode], null, 1));
  }
  console.log('SAMPLES_WRITTEN');
  cdp.close();
  process.exit(0);
})().catch((e) => { console.error('FATAL', e); process.exit(1); });
