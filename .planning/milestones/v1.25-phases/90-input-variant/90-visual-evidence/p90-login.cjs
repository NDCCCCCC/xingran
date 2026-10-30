// Phase 90 —— 巡检前置：自动登录（滑块验证码识别 + 拖动 + 提交）
//
// 目的：W3 的 14 页 + 6 页巡检需要登录态；本环境 captcha config = {"enabled":"slider"}。
//
// ⚠️ **关键设计**：每次验证码的缺口位置**随机**（`xPos = r.Intn(maxX-minX+1)+minX`）
//    ⇒ 静态分析一张图无效 ⇒ 必须**同会话内**「取图 → 识别 → 拖动」，
//    且服务端返回「位置不正确」后要**刷新验证码**再试（识别失败会重新随机）。
//
// 流程：
//   1. 打开 /login，填账号密码，点「登录」→ 弹出「安全验证」模态（CaptchaModal）
//   2. 同一次求值内取出背景图 dataURL + 几何信息 → Node 侧解码识别 → 候选 x
//   3. 用 CDP 的 Input.dispatchMouseEvent 逐步拖动滑块（触发 pointer 事件）
//   4. 服务端「位置不正确」→ 点击背景图刷新验证码 → 回到 2，最多试 N 次
//   5. 通过 → CaptchaModal 自动提交 → 登录请求带 captcha 完成 → token 落盘
//
// 识别算法见 p90-captcha-solve.cjs（暗像素列密度法；服务端容差仅 8px）。
//
// 用法：node p90-login.cjs [最大尝试次数]
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");
const { decodePng, findGapCandidates } = require("./p90-captcha-solve.cjs");

function dataUrlToBuf(dataUrl) {
  const m = /^data:image\/png;base64,(.+)$/.exec(dataUrl || "");
  if (!m) throw new Error("非 PNG data URL");
  return Buffer.from(m[1], "base64");
}

(async () => {
  const maxTries = parseInt(process.argv[2] || "8", 10);
  const { ev, nav, cdp, close } = await openCdp();
  try {
    await nav("/login", 6000);

    const fillRes = await ev(`(function(){
      function setNative(el, v){
        var proto = el instanceof HTMLInputElement ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype;
        var s = Object.getOwnPropertyDescriptor(proto, 'value').set;
        s.call(el, v); el.dispatchEvent(new Event('input', {bubbles:true}));
      }
      var inputs = Array.from(document.querySelectorAll('input'));
      var u = inputs.find(i => i.type === 'text' || i.type === 'email');
      var p = inputs.find(i => i.type === 'password');
      if(!u||!p) return 'NO_INPUTS';
      setNative(u,'admin'); setNative(p,'admin123');
      var btn = Array.from(document.querySelectorAll('button')).find(b => /登/.test(b.textContent));
      if(!btn) return 'NO_BTN';
      btn.click();
      return 'CLICKED';
    })()`);
    console.log("fill:", fillRes);
    await sleep(3000);

    // 拦截验证请求 → 捕获前端实际提交的 xPos（诊断缩放与识别偏差）
    await cdp.send("Runtime.enable");
    await ev(`(function(){
      window.__p90captured = null;
      var orig = window.fetch;
      window.fetch = function(input, init){
        try{
          var url = typeof input === 'string' ? input : (input && input.url) || '';
          if (/captcha/i.test(url)) {
            var body = init && init.body ? init.body : null;
            window.__p90captured = { url: String(url).slice(0,120), body: body ? String(body).slice(0,300) : null };
          }
        }catch(e){}
        return orig.apply(this, arguments);
      };
    })()`);

    const mouse = async (type, x, y) => {
      await cdp.send("Input.dispatchMouseEvent", {
        type,
        x,
        y,
        button: "left",
        buttons: type === "mouseReleased" ? 0 : 1,
        clickCount: 1,
      });
    };

    let logged = false;
    for (let attempt = 1; attempt <= maxTries && !logged; attempt++) {
      // ---- 2) 取背景图 + 几何信息（一次性求值，保证与识别的是同一张图）----
      const snap = await ev(`(function(){
        var dialog = document.querySelector('.ant-modal-content, [role="dialog"]');
        if (!dialog) return JSON.stringify({error:'no modal'});
        var bg = dialog.querySelector('.slider-bg');
        var btn = dialog.querySelector('.slider-button');
        var canvas = dialog.querySelector('.slider-canvas-wrapper');
        if(!bg||!btn||!canvas) return JSON.stringify({error:'missing parts'});
        function r(el){var b=el.getBoundingClientRect();return {x:b.x,y:b.y,w:b.width,h:b.height};}
        return JSON.stringify({ bgSrc: bg.src, bgNatural:{w:bg.naturalWidth,h:bg.naturalHeight}, btn:r(btn), canvas:r(canvas) });
      })()`);
      const s = JSON.parse(snap);
      if (s.error) {
        const st = await ev(
          `(function(){var o=JSON.parse(localStorage.getItem('auth-storage')||'{}');var x=o.state||{};return JSON.stringify({hasToken:!!(x.accessToken||x.token),href:location.href})})()`,
        );
        console.log(`attempt ${attempt}: 模态消失 →`, st);
        if (/"hasToken":true/.test(st)) logged = true;
        else break;
        continue;
      }

      const buf = dataUrlToBuf(s.bgSrc);
      const png = decodePng(buf);
      const { candidates } = findGapCandidates(png, 70);
      // ⚠️ **缩放方向（踩坑记录）**：前端提交的是
      //      xPos = round(currentX * (naturalWidth / containerWidth))
      //    ⇒ 反解得 显示坐标 target = xPos * (containerWidth / naturalWidth)
      //    即缩放系数 = **显示宽 / 原始宽**（<1，因为图被缩小显示）。
      //    本脚本初版误用了倒数（canvas.w / png.width > 1）→ 目标位移偏大 6%，
      //    超出服务端 8px 容差 → 每次都「位置不正确」。
      const scale = s.canvas.w / png.width;
      const targetOf = (xPos) => (xPos * s.canvas.w) / png.width;
      console.log(
        `attempt ${attempt}: bg ${png.width}x${png.height} canvas.w=${s.canvas.w} ` +
          `scale=${scale.toFixed(4)} candidates=${JSON.stringify(
            candidates.slice(0, 3).map((c) => ({ xPos: c.xPos, w: c.width })),
          )}`,
      );
      if (!candidates.length) {
        await ev(`(function(){var b=document.querySelector('.slider-bg');if(b)b.click();})()`);
        await sleep(1500);
        continue;
      }

      // ---- 3) 逐候选拖动 ----
      // ⚠️ **偏差标定（服务端反馈闭环）**：服务端日志（`logs/app.log` 中
      //    `POST /api/v1/system/auth/captcha/verify/slider` 的 request_body）会记录
      //    前端**实际提交**的 xPos ⇒ 可用它测出「识别值 → 提交值」的偏差。
      //    实测：识别 220 → 提交 222（偏差 **+2**，**成功 200**）；识别 120/162/117 → 提交
      //    122/164/119（同样 +2，失败 400）⇒ **偏差 +2 是正常舍入，不是识别误差**；
      //    真正的失败来自「识别到了背景里的暗块而非真缺口」⇒ 由候选排序修正（见 solve 脚本）。
      //    ⇒ 不加任何固定标定偏移。
      for (const cand of candidates.slice(0, 3)) {
        const xPosAdj = cand.xPos;
        const targetDisplay = (xPosAdj * s.canvas.w) / png.width;
        const startX = s.btn.x + s.btn.w / 2;
        const startY = s.btn.y + s.btn.h / 2;
        await mouse("mousePressed", startX, startY);
        await sleep(150);
        const steps = 28;
        for (let i = 1; i <= steps; i++) {
          await mouse("mouseMoved", startX + (targetDisplay * i) / steps, startY);
          await sleep(35);
        }
        const moved = await ev(
          `(function(){var b=document.querySelector('.slider-button');return b?b.style.left:'?'})()`,
        );
        await mouse("mouseReleased", startX + targetDisplay, startY);
        await sleep(2200);

        // 诊断：拦截下一次验证请求，看前端实际提交的 xPos
        const submitted = await ev(`(function(){
          var w = window.__p90captured;
          return w ? JSON.stringify(w) : 'null';
        })()`);

        const state = await ev(`(function(){
          var o=JSON.parse(localStorage.getItem('auth-storage')||'{}');
          var x=o.state||{};
          var txt=document.body.innerText||'';
          return JSON.stringify({
            hasToken:!!(x.accessToken||x.token),
            failed:/验证失败|位置不正确/.test(txt),
            modalGone:!document.querySelector('.ant-modal-content, [role="dialog"]'),
            href:location.href
          });
        })()`);
        console.log(`  候选 xPos=${cand.xPos} → btn.left=${moved} → ${state}`);
        console.log(`    提交请求: ${submitted}`);
        // 验证码通过后，登录页会自动带上 captcha 重发登录请求 →
        // 此时等待登录结果（auth-storage 拿到 token）。
        if (/"modalGone":true/.test(state) && !/"hasToken":true/.test(state)) {
          // 等待登录请求完成（可能还在飞行中）
          for (let w = 0; w < 12; w++) {
            await sleep(1000);
            const st2 = await ev(
              `(function(){var o=JSON.parse(localStorage.getItem('auth-storage')||'{}');var x=o.state||{};return JSON.stringify({hasToken:!!(x.accessToken||x.token),href:location.href,txt:(document.body.innerText||'').slice(0,100).replace(/\\n/g,' | ')})})()`,
            );
            if (/"hasToken":true/.test(st2)) {
              console.log(`    等待登录完成 → ${st2}`);
              logged = true;
              break;
            }
          }
        }
        if (/"hasToken":true/.test(state)) {
          logged = true;
          break;
        }
        if (/"failed":true/.test(state) && /"modalGone":false/.test(state)) {
          await ev(`(function(){var b=document.querySelector('.slider-bg');if(b)b.click();})()`);
          await sleep(1500);
          break;
        }
      }
    }

    const final = await ev(`(function(){
      var o=JSON.parse(localStorage.getItem('auth-storage')||'{}');
      var x=o.state||{};
      return JSON.stringify({hasToken:!!(x.accessToken||x.token), user:x.user?String(x.user.userName||''):null, href:location.href});
    })()`);
    console.log("FINAL:", final);
    if (/"hasToken":true/.test(final)) {
      const tok = await ev(`(function(){
        var o=JSON.parse(localStorage.getItem('auth-storage')||'{}');
        var x=o.state||{};
        return JSON.stringify({accessToken:x.accessToken||x.token||null, refreshToken:x.refreshToken||null});
      })()`);
      fs.writeFileSync(path.join(HERE, "auth-token.json"), tok);
      console.log("token 已落盘 auth-token.json");
    }
  } finally {
    close();
  }
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
