// B 类视觉证据：真实页面（/system/user）dark 下 Select 容器底色
const WebSocket = require(process.cwd() + '/ruoyi-react-frontend/node_modules/ws');
(async () => {
  const list = await (await fetch('http://127.0.0.1:9222/json')).json();
  const page = list.find((t) => t.type === 'page');
  const ws = new WebSocket(page.webSocketDebuggerUrl, { perMessageDeflate: false });
  let id = 0; const pend = new Map();
  const send = (m, p) => new Promise((r) => { const i = ++id; pend.set(i, r); ws.send(JSON.stringify({ id: i, method: m, params: p })); });
  ws.on('message', (d) => { const m = JSON.parse(d); if (m.id && pend.has(m.id)) { pend.get(m.id)(m.result); pend.delete(m.id); } });
  await new Promise((r) => ws.on('open', r));
  await send('Page.navigate', { url: 'http://localhost:4000/system/user' });
  await new Promise((r) => setTimeout(r, 5000));
  const expr = `(function(){
    function grab(sel){ var el=document.querySelector(sel); return el?getComputedStyle(el).backgroundColor:null; }
    return JSON.stringify({
      href: location.href.slice(0,60),
      selectContent: grab('.ant-select-content'),
      selectSelector: grab('.ant-select-selector'),
      cardBody: grab('.ant-card-body'),
      card: grab('.ant-card'),
      input: grab('.ant-input'),
      antBgContainer: getComputedStyle(document.querySelector('.ant-app')||document.documentElement).getPropertyValue('--ant-color-bg-container').trim()
    });
  })()`;
  const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true });
  console.log(r.result.value);
  ws.close(); process.exit(0);
})().catch((e) => { console.error('ERR', e.message); process.exit(1); });
