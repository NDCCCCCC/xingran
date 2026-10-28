// Task 2.4 分组合采样：store 偏好(DB) 与 data-color-mode 对齐后采矩阵
// 用法: node tmp_sample_v6.cjs <mode>   （mode=dark|light，需 DB 偏好已设为该模式并重启后端）
const fs = require('fs');
const path = require('path');
const WebSocket = require(process.cwd() + '/ruoyi-react-frontend/node_modules/ws');
const fixtureJs = fs.readFileSync(path.join(__dirname, 'matrix-fixture.js'), 'utf8');
const sampleJs = fs.readFileSync(path.join(__dirname, 'matrix-sample.js'), 'utf8');
const mode = process.argv[2] || 'dark';
(async () => {
  const list = await (await fetch('http://127.0.0.1:9222/json')).json();
  const page = list.find((t) => t.type === 'page');
  const ws = new WebSocket(page.webSocketDebuggerUrl, { perMessageDeflate: false });
  let id = 0; const pend = new Map();
  const send = (m, p) => new Promise((r) => { const i = ++id; pend.set(i, r); ws.send(JSON.stringify({ id: i, method: m, params: p })); });
  ws.on('message', (d) => { const m = JSON.parse(d); if (m.id && pend.has(m.id)) { pend.get(m.id)(m.result); pend.delete(m.id); } });
  await new Promise((r) => ws.on('open', r));
  // 登录兜底：先去 dashboard，若被踢回 login 则登录后重进
  await send('Page.navigate', { url: 'http://localhost:4000/dashboard' });
  await new Promise((r) => setTimeout(r, 3500));
  let ev = await send('Runtime.evaluate', { expression: 'location.href', returnByValue: true });
  if (/login/.test(ev.result.value)) {
    const loginJs = `(async function(){
      function setNative(el, v){ var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype; var s = Object.getOwnPropertyDescriptor(proto,'value').set; s.call(el, v); el.dispatchEvent(new Event('input',{bubbles:true})); }
      var inputs = Array.from(document.querySelectorAll('input'));
      var u = inputs.find(i => i.type === 'text'); var p = inputs.find(i => i.type === 'password');
      if (!u || !p) return 'NO_INPUTS';
      setNative(u,'admin'); setNative(p,'admin123');
      await new Promise(r=>setTimeout(r,300));
      var btn = document.querySelector('button[type="submit"]') || Array.from(document.querySelectorAll('button')).find(b=>/登/.test(b.textContent));
      btn.click(); await new Promise(r=>setTimeout(r,3000)); return 'OK';
    })()`;
    ev = await send('Runtime.evaluate', { expression: loginJs, awaitPromise: true, returnByValue: true });
    console.log('login:', ev.result.value);
    // 登录成功后重新进 dashboard（initialize 拉偏好）
    await send('Page.navigate', { url: 'http://localhost:4000/dashboard' });
    await new Promise((r) => setTimeout(r, 4500));
  }
  // 对齐 attribute 并探针
  await send('Runtime.evaluate', { expression: `document.documentElement.setAttribute('data-color-mode','${mode}')` });
  await new Promise((r) => setTimeout(r, 800));
  const probe = `(function(){
    var scope = document.querySelector('.ant-app') || document.querySelector('[class*="css-var-"]');
    var sc = scope ? getComputedStyle(scope) : getComputedStyle(document.documentElement);
    var sel = document.querySelector('.ant-select .ant-select-selector') || document.querySelector('.ant-select-content');
    var card = document.querySelector('.ant-card');
    var btn = document.querySelector('.ant-btn-default');
    return JSON.stringify({
      domMode: document.documentElement.getAttribute('data-color-mode'),
      antBgContainer: sc.getPropertyValue('--ant-color-bg-container').trim(),
      antBorder: sc.getPropertyValue('--ant-color-border').trim(),
      antBgElevated: sc.getPropertyValue('--ant-color-bg-elevated').trim(),
      realSelectBg: sel ? getComputedStyle(sel).backgroundColor : null,
      cardBg: card ? getComputedStyle(card).backgroundColor : null,
      defaultBtnBorder: btn ? getComputedStyle(btn).borderTopColor : null
    });
  })()`;
  let r = await send('Runtime.evaluate', { expression: probe, returnByValue: true });
  console.log('PROBE:', r.result.value);
  // fixture + sample
  r = await send('Runtime.evaluate', { expression: fixtureJs, returnByValue: true });
  if (r.exceptionDetails) { console.error('fixture fail', JSON.stringify(r.exceptionDetails)); process.exit(1); }
  r = await send('Runtime.evaluate', { expression: sampleJs, returnByValue: true });
  if (r.exceptionDetails) { console.error('sample fail', JSON.stringify(r.exceptionDetails)); process.exit(1); }
  const data = JSON.parse(r.result.value);
  if (data.mode !== mode) { console.error('mode mismatch', mode, data.mode); process.exit(1); }
  fs.writeFileSync(path.join(__dirname, `after-${mode}.json`), JSON.stringify(data, null, 1));
  console.log(`WROTE after-${mode}.json`);
  ws.close(); process.exit(0);
})().catch((e) => { console.error('ERR', e.message); process.exit(1); });
