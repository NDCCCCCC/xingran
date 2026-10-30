// Phase 90 临时探针 v2（执行期取证用，非终跑脚本）
// 用法：node p90-probe.cjs <mode>
// 目的：① VISFIX-05 规则是否生效（fixture 必须注入 cssVar scope 内）；② antd 6.6.4 是否生成 InputNumber status 边框规则；
//       ③ 批次 C 去 important 后 border-width/style 的接管来源；④ 内层 radius 保持 0。
const fs = require('fs');
const path = require('path');
const FE_ROOT = path.resolve(__dirname, '../../../../ruoyi-react-frontend');
const WebSocket = require(FE_ROOT + '/node_modules/ws');

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
  const mode = process.argv[2] || 'light';
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
  await nav('/user/settings', 3000);
  const idx = mode === 'dark' ? 1 : 0;
  await ev(`(function(){ var cards = document.querySelectorAll('button.xr-settings-segmented-card'); if (cards.length < 2) throw new Error('no cards'); cards[${idx}].click(); })()`);
  await sleep(1500);
  console.log('MODE=' + (await ev(`document.documentElement.getAttribute('data-color-mode')`)));
  await nav('/system/user', 4500);

  const out = await ev(`(function(){
    function css(el){ var c = getComputedStyle(el); return { bc: c.borderTopColor, bw: c.borderTopWidth, bs: c.borderTopStyle, r: c.borderTopLeftRadius, fs: c.fontSize, bg: c.backgroundColor }; }
    var scope = document.querySelector('.ant-app, [class*="css-var-"]');
    var res = { scopeClass: scope ? scope.className.slice(0,80) : null };
    // 1) 页面上是否已存在真实 InputNumber（决定 antd input-number 样式是否已注入）
    res.realInputNumberCount = document.querySelectorAll('.ant-input-number').length;
    // 2) fixture 注入到 cssVar scope 内（与 matrix-fixture.js 同口径）
    var prev = document.getElementById('__p90probe');
    if (prev) prev.remove();
    var c = document.createElement('div');
    c.id = '__p90probe';
    c.style.cssText = 'position:fixed;left:-9999px;top:0;width:1200px;';
    scope.appendChild(c);
    c.innerHTML =
      '<div data-k="num"><div class="ant-input-number ant-input-number-outlined"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div></div>' +
      '<div data-k="err"><div class="ant-input-number ant-input-number-outlined ant-input-number-status-error"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div></div>' +
      '<div data-k="warn"><div class="ant-input-number ant-input-number-outlined ant-input-number-status-warning"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div></div>' +
      '<div data-k="inp"><input class="ant-input" value=""></div>';
    var g = function(k){ return c.querySelector('[data-k="'+k+'"] > *'); };
    res.num = css(g('num'));
    res.numInner = css(g('num').querySelector('input'));
    res.err = css(g('err'));
    res.warn = css(g('warn'));
    res.inp = css(g('inp'));
    res.antColorError = getComputedStyle(scope).getPropertyValue('--ant-color-error').trim();
    res.antColorWarning = getComputedStyle(scope).getPropertyValue('--ant-color-warning').trim();
    res.antColorBorder = getComputedStyle(scope).getPropertyValue('--ant-color-border').trim();
    // 3) 扫描所有样式表，收集含 input-number-status / .ant-input-number 的规则（找 antd 自身规则）
    var found = [];
    for (var i=0;i<document.styleSheets.length;i++){
      var ss; try { ss = document.styleSheets[i]; } catch(e){ continue; }
      var rs; try { rs = ss.cssRules; } catch(e){ continue; }
      if (!rs) continue;
      for (var j=0;j<rs.length;j++){
        var txt = rs[j].cssText || '';
        if (/input-number-status-(error|warning)/.test(txt) && found.length < 30) {
          found.push({ sheet:i, rule:j, len: txt.length, text: txt.slice(0,300) });
        }
      }
    }
    res.rulesWithInputNumberStatus = found;
    // 4) antd 自身的 .ant-input-number 基础规则（判断 border-width/style 来源）
    var base = [];
    for (var i2=0;i2<document.styleSheets.length;i2++){
      var ss2; try { ss2 = document.styleSheets[i2]; } catch(e){ continue; }
      var rs2; try { rs2 = ss2.cssRules; } catch(e){ continue; }
      if (!rs2) continue;
      for (var j2=0;j2<rs2.length;j2++){
        var t2 = rs2[j2].cssText || '';
        if (/\\.ant-input-number(?![-\\w])/.test(t2) && /border/.test(t2) && base.length < 12) {
          base.push({ sheet:i2, rule:j2, text: t2.slice(0,300) });
        }
      }
    }
    res.antdInputNumberBaseRules = base;
    // 5) 样式表顺序：项目 index.css 与 antd cssinjs 的先后
    res.sheetOwners = [];
    for (var k=0;k<document.styleSheets.length;k++){
      var s2; try { s2=document.styleSheets[k]; } catch(e){ continue; }
      var owner = s2.ownerNode ? String(s2.ownerNode.id || s2.ownerNode.getAttribute('data-css-hash') || s2.ownerNode.tagName) : '?';
      var n=0; try{ n=s2.cssRules.length; }catch(e){}
      res.sheetOwners.push(k + ':' + owner.slice(0,40) + '(' + n + ')');
    }
    return JSON.stringify(res, null, 1);
  })()`);
  fs.writeFileSync(path.join(__dirname, `probe90-${mode}.json`), out);
  console.log(out);
  cdp.close();
  process.exit(0);
})().catch((e) => { console.error('FATAL', e); process.exit(1); });
