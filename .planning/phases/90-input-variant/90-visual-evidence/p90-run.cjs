// Phase 90 采样 / 巡检驱动
// 源自 89-visual-evidence/p89-final-run.cjs（骨架逐字沿用：CDP 连接 / 登录 admin·admin123 /
// 经 /user/settings 分段卡切 settingsStore 服务器偏好 / 采 fixture+sample / 14 页 + 6 页截图 /
// console deprecated 计数）。仅改：__dirname 相对路径（与 89 同深度，无需改）、fixture/sample 指向本目录、
// 输出文件名（before-*.json / after-*.json）、inspection-result-90.txt、截图前缀 p90-* / r3-*。
// 用法：
//   node p90-run.cjs --baseline   → 只采 fixture+sample，落 before-{light,dark}.json（不跑巡检截图）
//   node p90-run.cjs              → 采 fixture+sample + 巡检截图，落 after-{light,dark}.json
const fs = require('fs');
const path = require('path');
const FE_ROOT = path.resolve(__dirname, '../../../../ruoyi-react-frontend');
const WebSocket = require(FE_ROOT + '/node_modules/ws');

const BASELINE = process.argv.slice(2).includes('--baseline');
const OUT_PREFIX = BASELINE ? 'before' : 'after';

const fixtureJs = fs.readFileSync(path.join(__dirname, 'matrix-fixture.js'), 'utf8');
const sampleJs = fs.readFileSync(path.join(__dirname, 'matrix-sample.js'), 'utf8');

const PAGES = [
  ['p90-01', 'login', '/login'],
  ['p90-02', 'dashboard', '/dashboard'],
  ['p90-03', 'system_user', '/system/user'],
  ['p90-04', 'system_role', '/system/role'],
  ['p90-05', 'system_dept', '/system/dept'],
  ['p90-06', 'system_menu', '/system/menu'],
  ['p90-07', 'operations_buildings', '/operations/buildings'],
  ['p90-08', 'operations_workstations', '/operations/workstations'],
  ['p90-09', 'operations_server-rooms', '/operations/server-rooms'],
  ['p90-10', 'operations_dedicated-lines', '/operations/dedicated-lines'],
  ['p90-11', 'network_devices', '/network/devices'],
  ['p90-12', 'monitor_dashboard', '/monitor/dashboard'],
  ['p90-13', 'monitor_server', '/monitor/server'],
  ['p90-14', 'my-notices', '/user/my-notices'],
];
const R2 = [
  ['r3-system_user', '/system/user'],
  ['r3-operations_workstations', '/operations/workstations'],
  ['r3-ad-domain_ous', '/ad-domain/ous'],
  ['r3-network_mac', '/network/mac'],
  ['r3-my-notices', '/user/my-notices'],
  ['r3-monitor_cache', '/monitor/cache'],
];

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
  await cdp.send('Page.enable');
  await cdp.send('Runtime.enable');
  await cdp.send('Log.enable');

  const consoleBuf = [];
  // 全局 console/Log 捕获（无法带 sessionId 过滤来源，单页面单连接串行使用）
  const onMsg = (d) => {
    const m = JSON.parse(d);
    if (m.method === 'Runtime.consoleAPICalled') {
      const txt = (m.params.args || []).map((a) => a.value != null ? String(a.value) : (a.description || a.type)).join(' ');
      consoleBuf.push({ kind: 'console.' + m.params.type, txt });
    } else if (m.method === 'Log.entryAdded') {
      consoleBuf.push({ kind: 'log.' + m.params.entry.level, txt: m.params.entry.text });
    }
  };
  cdp._raw = (handler) => { /* expose */ };
  // 直接挂原生监听：绕开 connect 封装
  // （connect 里没有透出 message 事件 → 这里重新开一个底层 ws 太重，改为在 connect 内注入）
  // 简化：重新实现 message 监听
  // —— connect 未透出，退而求其次：用 Runtime.evaluate 轮询不可行 → 此处用独立轻量连接捕获 console
  const list2 = await (await fetch('http://127.0.0.1:9222/json')).json();
  const page2 = list2.find((t) => t.type === 'page');
  const ws2raw = new WebSocket(page2.webSocketDebuggerUrl, { perMessageDeflate: false });
  await new Promise((r) => ws2raw.on('open', r));
  let cid = 10000;
  ws2raw.on('message', onMsg);
  ws2raw.send(JSON.stringify({ id: ++cid, method: 'Runtime.enable', params: {} }));
  ws2raw.send(JSON.stringify({ id: ++cid, method: 'Log.enable', params: {} }));

  const ev = async (expr, awaitPromise = false) => {
    const r = await cdp.send('Runtime.evaluate', { expression: expr, awaitPromise, returnByValue: true });
    if (r.exceptionDetails) throw new Error('EVAL: ' + JSON.stringify(r.exceptionDetails).slice(0, 400));
    return r.result.value;
  };
  const nav = async (url, wait = 3500) => {
    await cdp.send('Page.navigate', { url: 'http://localhost:4000' + url });
    await sleep(wait);
  };

  // 1. 登录
  await nav('/login');
  let href = await ev('location.href');
  if (/login/.test(href)) {
    const r = await ev(`(async function(){
      function setNative(el, v){ var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype; var s = Object.getOwnPropertyDescriptor(proto,'value').set; s.call(el, v); el.dispatchEvent(new Event('input',{bubbles:true})); }
      var inputs = Array.from(document.querySelectorAll('input'));
      var u = inputs.find(i => i.type === 'text'); var p = inputs.find(i => i.type === 'password');
      if (!u || !p) return 'NO_INPUTS';
      setNative(u,'admin'); setNative(p,'admin123');
      await new Promise(r=>setTimeout(r,300));
      var btn = document.querySelector('button[type="submit"]') || Array.from(document.querySelectorAll('button')).find(b=>/登/.test(b.textContent));
      btn.click(); await new Promise(r=>setTimeout(r,3000)); return 'OK ' + location.href;
    })()`, true);
    console.log('Login:', r);
    if (!/OK/.test(r)) { console.error('LOGIN FAILED'); process.exit(1); }
  }

  // 2. 切换模式（设置页分段卡：index 0 = light, 1 = dark）
  const setMode = async (mode) => {
    await nav('/user/settings', 3000);
    const idx = mode === 'dark' ? 1 : 0;
    await ev(`(function(){
      var cards = document.querySelectorAll('button.xr-settings-segmented-card');
      if (cards.length < 2) throw new Error('segmented cards not found: ' + cards.length);
      cards[${idx}].click();
    })()`);
    await sleep(1500);
    const dom = await ev(`document.documentElement.getAttribute('data-color-mode')`);
    if (dom !== mode) { console.error('setMode mismatch:', mode, dom); process.exit(1); }
    console.log(`MODE SET: ${mode} (data-color-mode=${dom})`);
  };

  // 3. 探针（90 关注的 Input 域变量 + 圆角 + 错误态 token）
  const probeJs = `(function(){
    var root = getComputedStyle(document.documentElement);
    var scope = document.querySelector('.ant-app') || document.querySelector('[class*="css-var-"]');
    var sc = scope ? getComputedStyle(scope) : root;
    function pick(sel){ var el = document.querySelector(sel); return el ? getComputedStyle(el) : null; }
    var inp = pick('.ant-input'); var sel_ = pick('.ant-select'); var card = pick('.ant-card');
    var selSel = pick('.ant-select .ant-select-selector') || pick('.ant-select-content');
    return JSON.stringify({
      mode: document.documentElement.getAttribute('data-color-mode'),
      inputBgVar: root.getPropertyValue('--input-bg').trim(),
      placeholderVar: root.getPropertyValue('--input-text-placeholder').trim(),
      inputBorderVar: root.getPropertyValue('--input-border').trim(),
      radiusMdVar: root.getPropertyValue('--theme-radius-md').trim(),
      radiusLgVar: root.getPropertyValue('--theme-radius-lg').trim(),
      antColorError: sc.getPropertyValue('--ant-color-error').trim(),
      inputBg: inp && inp.backgroundColor,
      inputRadius: inp && inp.borderTopLeftRadius,
      inputFontSize: inp && inp.fontSize,
      pageBg: getComputedStyle(document.body).backgroundColor,
      selectBg: sel_ && sel_.backgroundColor,
      selectSelectorBg: selSel && selSel.backgroundColor,
      cardBg: card && card.backgroundColor,
      antBgContainer: sc.getPropertyValue('--ant-color-bg-container').trim(),
      antBorder: sc.getPropertyValue('--ant-color-border').trim(),
      antBgElevated: sc.getPropertyValue('--ant-color-bg-elevated').trim()
    }, null, 2);
  })();`;

  const sampleMode = async (mode) => {
    await nav('/system/user', 4000);
    const pj = await ev(probeJs);
    fs.writeFileSync(path.join(__dirname, `probe-${mode}.txt`), pj + '\n');
    console.log(`PROBE ${mode}:\n${pj}`);
    await ev(fixtureJs);
    const sj = await ev(sampleJs);
    const data = JSON.parse(sj);
    if (data.mode !== mode) { console.error('sample mode mismatch'); process.exit(1); }
    const formCount = Object.keys(data.forms).length;
    if (formCount !== 22) { console.error(`sample form count = ${formCount} (期望 22)`); process.exit(1); }
    if (!data.forms.f22_input_number_status_error) { console.error('f22 missing in sample'); process.exit(1); }
    fs.writeFileSync(path.join(__dirname, `${OUT_PREFIX}-${mode}.json`), JSON.stringify(data, null, 1));
    console.log(`WROTE ${OUT_PREFIX}-${mode}.json (forms=${formCount})`);
  };

  await setMode('light');
  await sampleMode('light');
  await setMode('dark');
  await sampleMode('dark');

  if (BASELINE) {
    console.log('BASELINE_DONE（只采基线，未跑巡检截图）');
    ws2raw.close();
    cdp.close();
    process.exit(0);
  }

  // 4. r3-* 截图（12 张）与 14 页巡检（28 张）+ console deprecated 计数
  const shot = async (file) => {
    const r = await cdp.send('Page.captureScreenshot', { format: 'png' });
    fs.writeFileSync(path.join(__dirname, file), Buffer.from(r.data, 'base64'));
  };
  const results = [];
  for (const mode of ['light', 'dark']) {
    await setMode(mode);
    for (const [name, url] of [...R2, ...PAGES.map((p) => [p[0], p[2]])]) {
      console.log(`NAV ${name} -> ${url}`);
      let navErr = null;
      let finalHref2 = 'NAV_FAIL';
      try {
        await nav(url, 4200);
        finalHref2 = await ev('location.href');
        await shot(`${name}-${mode}.png`);
      } catch (e) { navErr = String(e).slice(0, 200); }
      const dep = consoleBuf.filter((c) => /deprecated/i.test(c.txt));
      const errs = consoleBuf.filter((c) => c.kind.endsWith('error'));
      results.push({ mode, name, url, finalHref: finalHref2, navErr, deprecated: dep.length, errors: errs.length, depSamples: dep.map((d) => d.txt.slice(0, 140)), errSamples: errs.map((d) => d.txt.slice(0, 140)) });
      fs.writeFileSync(path.join(__dirname, 'inspection-result-90.txt'), JSON.stringify(results, null, 1));
      console.log(`SHOT ${name}-${mode} dep=${dep.length} err=${errs.length} href=${String(finalHref2).slice(0, 60)}${navErr ? ' NAVERR=' + navErr : ''}`);
    }
  }
  fs.writeFileSync(path.join(__dirname, 'inspection-result-90.txt'),
    JSON.stringify(results, null, 1));
  const totalDep = results.reduce((s, r) => s + r.deprecated, 0);
  const totalErr = results.reduce((s, r) => s + r.errors, 0);
  console.log(`TOTAL deprecated=${totalDep} errors=${totalErr}`);
  console.log('FINAL_DONE');
  ws2raw.close();
  cdp.close();
  process.exit(0);
})().catch((e) => { console.error('FATAL', e); process.exit(1); });
