package auth

import (
	"encoding/json"
	"strings"
)

// webLoginPageTpl 交互登录页模板（内嵌资源，无外部文件依赖）。
// 纯 HTML/CSS/JS；占位符 __PRESET_USER__ 由 renderLoginPage 替换为 JSON 转义后的预填用户名。
const webLoginPageTpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>XingRan CLI 登录</title>
<style>
  :root { --primary:#2563eb; --danger:#dc2626; --bg:#f1f5f9; }
  * { box-sizing:border-box; margin:0; padding:0; }
  body { font-family:system-ui,-apple-system,"Segoe UI","Microsoft YaHei",sans-serif;
         background:var(--bg); display:flex; justify-content:center; align-items:center;
         min-height:100vh; padding:16px; }
  .card { background:#fff; border-radius:12px; box-shadow:0 4px 24px rgba(0,0,0,.08);
          padding:32px; width:464px; max-width:100%; }
  h1 { font-size:20px; margin-bottom:4px; }
  .sub { color:#64748b; font-size:13px; margin-bottom:20px; }
  label { display:block; font-size:13px; color:#334155; margin:12px 0 4px; }
  input[type=text],input[type=password] { width:100%; padding:10px 12px; border:1px solid #cbd5e1;
          border-radius:8px; font-size:14px; outline:none; }
  input:focus { border-color:var(--primary); }
  .captcha-box { margin-top:12px; }
  .slider-wrap { position:relative; width:400px; max-width:100%; user-select:none;
                 touch-action:none; border-radius:8px; overflow:hidden; }
  .slider-wrap img.bg { display:block; width:100%; pointer-events:none; }
  img.piece { position:absolute; cursor:grab; touch-action:none; opacity:.92; }
  img.piece:active { cursor:grabbing; }
  .normal-cap { display:flex; gap:8px; align-items:center; }
  .normal-cap img { border-radius:8px; border:1px solid #cbd5e1; cursor:pointer; }
  button { width:100%; margin-top:20px; padding:12px; border:none; border-radius:8px;
           background:var(--primary); color:#fff; font-size:15px; cursor:pointer; }
  button:disabled { background:#94a3b8; cursor:not-allowed; }
  .msg { margin-top:14px; padding:10px 12px; border-radius:8px; font-size:13px; display:none; }
  .msg.err { display:block; background:#fee2e2; color:var(--danger); white-space:pre-wrap; }
  .msg.ok { display:block; background:#dcfce7; color:#15803d; }
  .hint { font-size:12px; color:#94a3b8; margin-top:10px; }
</style>
</head>
<body>
<div class="card" id="app">
  <h1>XingRan CLI 登录</h1>
  <div class="sub">本页面仅在本机运行，密码不会经过命令行</div>
  <label for="u">用户名</label>
  <input type="text" id="u" autocomplete="username">
  <label for="p">密码</label>
  <input type="password" id="p" autocomplete="current-password">
  <div class="captcha-box" id="captchaBox"></div>
  <button id="btn" disabled>登录</button>
  <div class="msg" id="msg"></div>
  <div class="hint">遇到验证码失败可直接重新登录，会自动刷新验证码。</div>
</div>
<script>
var PRESET_USER = __PRESET_USER__;
var S = new URLSearchParams(location.search).get('s');
var mode = 'disabled', captchaId = '', capToken = '', dragX = 0, dragging = false, startX = 0;

function $(id){ return document.getElementById(id); }
function setMsg(text, ok){
  var m = $('msg');
  m.className = 'msg ' + (ok ? 'ok' : 'err');
  m.textContent = text;
}

function loadCaptcha(){
  dragX = 0;
  fetch('/captcha?s=' + encodeURIComponent(S)).then(function(r){ return r.json(); }).then(function(d){
    if (d.error) { setMsg('获取验证码失败：' + d.error); return; }
    mode = d.mode; captchaId = d.captchaId || ''; capToken = d.token || '';
    var box = $('captchaBox');
    if (mode === 'slider') {
      box.innerHTML = '<div class="slider-wrap" id="wrap">' +
        '<img class="bg" src="' + d.sliderImg + '" draggable="false">' +
        '<img class="piece" id="piece" src="' + d.pieceImg + '" style="left:0px" draggable="false">' +
        '</div><div class="hint">拖动拼图块到缺口位置后点击登录</div>';
      var bgEl = $('wrap').querySelector('.bg');
      var pieceEl = $('piece');
      // 底图按容器宽度显示；拼图块与 yPos 必须按同一缩放比换算，
      // 否则图片被缩放时视觉位置与后端原图坐标错位。
      function applyScale(){
        if (!bgEl.naturalWidth) return;
        var sc = bgEl.clientWidth / bgEl.naturalWidth;
        pieceEl.style.width = (pieceEl.naturalWidth * sc) + 'px';
        pieceEl.style.top = ((d.yPos||0) * sc) + 'px';
        pieceEl.style.left = (dragX) + 'px';
      }
      if (bgEl.complete) applyScale(); else bgEl.onload = applyScale;
      initDrag($('piece'));
    } else if (mode === 'normal') {
      box.innerHTML = '<label>验证码</label><div class="normal-cap">' +
        '<img id="capImg" src="' + d.captchaImg + '" title="点击刷新" alt="验证码">' +
        '<input type="text" id="code" style="flex:1" autocomplete="off"></div>';
      $('capImg').onclick = function(){ loadCaptcha(); };
    } else {
      box.innerHTML = '';
    }
    $('btn').disabled = false;
  }).catch(function(e){ setMsg('网络错误：' + e); });
}

function initDrag(piece){
  var wrap = $('wrap');
  piece.addEventListener('pointerdown', function(e){
    dragging = true; startX = e.clientX - dragX;
    piece.setPointerCapture(e.pointerId); e.preventDefault();
  });
  piece.addEventListener('pointermove', function(e){
    if (!dragging) return;
    var max = wrap.clientWidth - piece.clientWidth;
    dragX = Math.max(0, Math.min(max, e.clientX - startX));
    piece.style.left = dragX + 'px';
  });
  piece.addEventListener('pointerup', function(){ dragging = false; });
}

$('btn').onclick = function(){
  var btn = $('btn');
  btn.disabled = true; setMsg('');
  var body = { username: $('u').value.trim(), password: $('p').value,
               mode: mode, captchaId: captchaId, token: capToken, code: '', sliderX: 0 };
  if (mode === 'slider') {
    // 页面可能对 400px 原图做了响应式缩放显示；dragX 是显示像素，
    // 必须按 naturalWidth/clientWidth 换算回原图坐标，否则后端 ±8px 容差校验必然失败。
    var bg = document.querySelector('#wrap .bg');
    var scale = (bg && bg.naturalWidth && bg.clientWidth) ? (bg.clientWidth / bg.naturalWidth) : 1;
    body.sliderX = Math.round(dragX / scale);
  }
  if (mode === 'normal') { body.code = ($('code')||{}).value || ''; }
  fetch('/login?s=' + encodeURIComponent(S), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  }).then(function(r){ return r.json().then(function(d){ return { status: r.status, data: d }; }); })
  .then(function(resp){
    if (resp.data && resp.data.ok) {
      setMsg('登录成功，请返回终端，本页面可以关闭。', true);
      document.querySelectorAll('input,button').forEach(function(el){ el.disabled = true; });
    } else {
      setMsg((resp.data && resp.data.error) || ('登录失败(HTTP ' + resp.status + ')'));
      btn.disabled = false;
      loadCaptcha(); // 失败重试：刷新验证码
    }
  }).catch(function(e){
    setMsg('无法连接本地登录服务（' + e + '）。服务可能已超时关闭——请回到终端重新执行 login --web，本页面请关闭重开。');
    btn.disabled = false;
  });
};

if (PRESET_USER) { $('u').value = PRESET_USER; }
loadCaptcha();
</script>
</body>
</html>`

// renderLoginPage 渲染登录页（预填用户名经 JSON 转义防注入）。
func renderLoginPage(presetUsername string) string {
	encoded, _ := json.Marshal(presetUsername)
	return strings.Replace(webLoginPageTpl, "__PRESET_USER__", string(encoded), 1)
}
