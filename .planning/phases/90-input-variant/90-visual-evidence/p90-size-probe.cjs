// Phase 90 · T4 实页字号探针（执行期取证）
// 用法：node p90-size-probe.cjs <outTag>       （输出 p90-size-<tag>.json）
// 探测三处真实页面的字号（PLAN 90-03 任务 3.2 裁定输入三）：
//   ① /knowledge/view 的 <Search size="large">（Input.Search wrapper 结构）→ 内层 input 字号
//   ② 全局搜索（button.search-trigger → Modal 内 <Search size="large">）→ 内层 input 字号
//   ③ /login 的 <Form size="large">（login.css K10 复位块接管）→ 输入框字号
//   ④ 普通 <Input>（/system/user 搜索框）与 InputNumber 的字号（对照）
const fs = require('fs');
const path = require('path');
const FE_ROOT = path.resolve(__dirname, '../../../../ruoyi-react-frontend');
const WebSocket = require(FE_ROOT + '/node_modules/ws');

const TAG = process.argv[2] || 'now';

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

const READ_FS = `(function(){
  function fs(el){ return el ? getComputedStyle(el).fontSize : null; }
  var out = { href: location.href, title: document.title };
  // ① knowledge/view 的 Search（Input.Search → wrapper 结构）
  var w1 = document.querySelector('.user-form-input.ant-input-affix-wrapper') || document.querySelector('.ant-input-affix-wrapper');
  out.kbSearchWrapper = fs(w1);
  out.kbSearchInner = fs(w1 ? w1.querySelector('input') : null);
  out.kbWrapperCls = w1 ? w1.className : null;
  // ④ 普通 Input
  var inp = document.querySelector('input.ant-input:not(.ant-input-number-input):not(.ant-select-selection-search-input)');
  out.plainInput = fs(inp);
  return JSON.stringify(out);
})()`;

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
  const res = {};

  // ③ 未登录态：/login（先清登录态，否则会被重定向到 /dashboard）
  await nav('/login', 1500);
  await ev(`(function(){ try { localStorage.clear(); sessionStorage.clear(); } catch(e){} return 'CLEARED'; })()`);
  await nav('/login', 4500);
  res.login = JSON.parse(await ev(`(function(){
    function fs(el){ return el ? getComputedStyle(el).fontSize : null; }
    var lf = document.querySelector('.login-form');
    var inputs = Array.from(document.querySelectorAll('input'));
    var u = inputs.find(i => i.type === 'text') || inputs[0];
    var p = inputs.find(i => i.type === 'password');
    var form = document.querySelector('form');
    return JSON.stringify({
      href: location.href,
      userInput: fs(u), passwordInput: fs(p),
      loginFormFound: !!lf,
      loginFormInnerInput: fs(lf ? lf.querySelector('input') : null),
      loginFormWrapper: fs(lf ? lf.querySelector('.ant-input-affix-wrapper') : null),
      formClass: form ? form.className : null,
      inputCount: inputs.length,
    });
  })()`));

  // 登录
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

  // ① knowledge 的 Search（Input.Search → wrapper 结构）；先从菜单里找真实路由
  const kbHref = await ev(`(function(){
    var as = Array.from(document.querySelectorAll('a[href]'));
    for (var i=0;i<as.length;i++){ if (/knowledge/i.test(as[i].getAttribute('href')||'')) return as[i].getAttribute('href'); }
    return '';
  })()`);
  await nav(kbHref || '/knowledge/view', 5000);
  res.knowledge = JSON.parse(await ev(READ_FS));
  res.knowledge.hrefUsed = kbHref || '/knowledge/view';

  // ② 全局搜索
  await nav('/system/user', 4500);
  const clicked = await ev(`(function(){
    var b = document.querySelector('button.search-trigger');
    if (!b) return 'NO_TRIGGER';
    b.click(); return 'CLICKED';
  })()`);
  await sleep(3000);
  res.globalSearch = JSON.parse(await ev(`(function(){
    function fs(el){ return el ? getComputedStyle(el).fontSize : null; }
    var m = document.querySelector('.ant-modal') || document.querySelector('.ant-modal-wrap');
    var w = m ? m.querySelector('.ant-input-affix-wrapper') : null;
    return JSON.stringify({
      trigger: ${JSON.stringify(clicked)},
      modalFound: !!m,
      modalCls: m ? String(m.className).slice(0,120) : null,
      wrapperFs: fs(w),
      innerFs: fs(w ? w.querySelector('input') : null),
      wrapperCls: w ? w.className : null,
      inputCount: m ? m.querySelectorAll('input').length : 0,
    });
  })()`));

  // ④ /system/user 普通 Input + InputNumber（若有）
  res.systemUser = JSON.parse(await ev(`(function(){
    function fs(el){ return el ? getComputedStyle(el).fontSize : null; }
    var inp = document.querySelector('input.ant-input:not(.ant-input-number-input):not(.ant-select-selection-search-input)');
    var num = document.querySelector('.ant-input-number');
    var numIn = num ? num.querySelector('input') : null;
    return JSON.stringify({
      plainInputFs: fs(inp),
      inputNumberRootFs: fs(num),
      inputNumberInnerFs: fs(numIn),
      inputNumberCount: document.querySelectorAll('.ant-input-number').length,
    });
  })()`));

  res.tag = TAG;
  fs.writeFileSync(path.join(__dirname, `p90-size-${TAG}.json`), JSON.stringify(res, null, 1));
  console.log(JSON.stringify(res, null, 1));
  cdp.close();
  process.exit(0);
})().catch((e) => { console.error('FATAL', e); process.exit(1); });
