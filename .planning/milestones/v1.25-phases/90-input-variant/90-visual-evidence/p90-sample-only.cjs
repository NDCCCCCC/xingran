// Phase 90 采样（仅 fixture+sample，不跑巡检截图）—— 执行期快速复采用
// 用法：node p90-sample-only.cjs <prefix>   （默认 after）
const fs = require('fs');
const path = require('path');
const FE_ROOT = path.resolve(__dirname, '../../../../xingran-react-frontend');
const WebSocket = require(FE_ROOT + '/node_modules/ws');

const OUT_PREFIX = process.argv[2] || 'after';
const fixtureJs = fs.readFileSync(path.join(__dirname, 'matrix-fixture.js'), 'utf8');
const sampleJs = fs.readFileSync(path.join(__dirname, 'matrix-sample.js'), 'utf8');

function connect(wsUrl) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl, { perMessageDeflate: false });
    let id = 0;
    const pending = new Map();
    ws.on('open', () => resolve({
      send(method, params) {
        return new Promise((res2, rej2) => {
          const mid = ++id;
          pending.set(mid, { res: res2, rej: rej2 });
          ws.send(JSON.stringify({ id: mid, method, params }));
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
  await cdp.send('Page.enable');
  await cdp.send('Runtime.enable');
  const ev = async (expr, awaitPromise = false) => {
    const r = await cdp.send('Runtime.evaluate', { expression: expr, awaitPromise, returnByValue: true });
    if (r.exceptionDetails) throw new Error('EVAL: ' + JSON.stringify(r.exceptionDetails).slice(0, 600));
    return r.result.value;
  };
  const nav = async (url, wait = 4000) => { await cdp.send('Page.navigate', { url: 'http://localhost:4000' + url }); await sleep(wait); };

  await nav('/login');
  let href = await ev('location.href');
  if (/login/.test(href)) {
    await ev(`(async function(){
      function setNative(el, v){ var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype; var s = Object.getOwnPropertyDescriptor(proto,'value').set; s.call(el, v); el.dispatchEvent(new Event('input',{bubbles:true})); }
      var inputs = Array.from(document.querySelectorAll('input'));
      var u = inputs.find(i => i.type === 'text'); var p = inputs.find(i => i.type === 'password');
      if (!u || !p) return 'NO_INPUTS';
      setNative(u,'admin'); setNative(p,'admin123');
      await new Promise(r=>setTimeout(r,300));
      var btn = document.querySelector('button[type="submit"]') || Array.from(document.querySelectorAll('button')).find(b=>/登/.test(b.textContent));
      btn.click(); await new Promise(r=>setTimeout(r,3500)); return 'OK ' + location.href;
    })()`, true);
  }
  const setMode = async (mode) => {
    await nav('/user/settings', 3000);
    const idx = mode === 'dark' ? 1 : 0;
    await ev(`(function(){ var cards = document.querySelectorAll('button.xr-settings-segmented-card'); if (cards.length < 2) throw new Error('no cards'); cards[${idx}].click(); })()`);
    await sleep(1500);
    const dom = await ev(`document.documentElement.getAttribute('data-color-mode')`);
    if (dom !== mode) { console.error('setMode mismatch', mode, dom); process.exit(1); }
    console.log('MODE SET ' + mode);
  };
  const sampleMode = async (mode) => {
    await nav('/system/user', 4500);
    await ev(fixtureJs);
    const sj = await ev(sampleJs);
    const data = JSON.parse(sj);
    if (data.mode !== mode) { console.error('sample mode mismatch'); process.exit(1); }
    if (Object.keys(data.forms).length !== 22) { console.error('form count != 22'); process.exit(1); }
    fs.writeFileSync(path.join(__dirname, `${OUT_PREFIX}-${mode}.json`), JSON.stringify(data, null, 1));
    console.log(`WROTE ${OUT_PREFIX}-${mode}.json`);
  };
  await setMode('light');
  await sampleMode('light');
  await setMode('dark');
  await sampleMode('dark');
  cdp.close();
  process.exit(0);
})().catch((e) => { console.error('FATAL', e); process.exit(1); });
