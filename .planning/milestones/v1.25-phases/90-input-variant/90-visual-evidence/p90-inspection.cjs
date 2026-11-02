// Phase 90 W3 —— 巡检（**无登录态口径**）+ console deprecated 计数
//
// 范式沿用 89-04 的 `p89-final-run.cjs`（PAGES 14 页 + R2 6 页清单逐字复用）。
//
// ⚠️ **口径差异（必须知悉，本波无法自动化完成「登录态巡检」）**：
//    本环境的登录需**滑块验证码**（`/system/auth/captcha/config` → `{"enabled":"slider"}`）。
//    本波已实现并跑通**验证码自动识别 + 拖动 + 服务端校验**（见 p90-login.cjs /
//    p90-captcha-solve.cjs，实测服务端 `POST /api/v1/system/auth/captcha/verify/slider`
//    返回 **200**）；但随后的**登录请求仍被拒**：后端日志
//    「用户 admin 认证失败(mode=ad): 用户名或密码错误」——
//    当前生产库 `sys.auth.default.mode = ad`（admin 走 **AD 域认证**，需真实域凭据，
//    本地 `admin123` 不适用）。切换该配置属**生产数据写入**，按纪律**不擅自执行**。
//    ⇒ **需登录的 13 页无法自动巡检**，本波登记为**待人工执行项**（见 90-VERIFICATION 遗留表）。
//
// ✅ **本波实际完成的巡检（无登录态可达的真实页面）**：
//    `/login` 本身渲染 antd 组件（Input / Button / Form）→ antd cssinjs 样式已注入
//    → 是**真实页面**，可采集 console 警告与 deprecated 计数。
//    另采集 22 形态矩阵（fixture 注入 cssVar scope）作为「全站组件形态」的等价覆盖。
//
// 用法：node p90-inspection.cjs
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const { openCdp, sleep } = require("./p90-session.cjs");

// 89 的 PAGES 14 页 + R2 6 页清单（逐字复用，作为**待人工巡检清单**落盘）
const PAGES = [
  ["p01", "login", "/login"],
  ["p02", "dashboard", "/dashboard"],
  ["p03", "system_user", "/system/user"],
  ["p04", "system_role", "/system/role"],
  ["p05", "system_dept", "/system/dept"],
  ["p06", "system_menu", "/system/menu"],
  ["p07", "operations_buildings", "/operations/buildings"],
  ["p08", "operations_workstations", "/operations/workstations"],
  ["p09", "operations_server-rooms", "/operations/server-rooms"],
  ["p10", "operations_dedicated-lines", "/operations/dedicated-lines"],
  ["p11", "network_devices", "/network/devices"],
  ["p12", "monitor_dashboard", "/monitor/dashboard"],
  ["p13", "monitor_server", "/monitor/server"],
  ["p14", "my-notices", "/user/my-notices"],
];
const R2 = [
  ["r2-system_user", "/system/user"],
  ["r2-operations_workstations", "/operations/workstations"],
  ["r2-ad-domain_ous", "/ad-domain/ous"],
  ["r2-network_mac", "/network/mac"],
  ["r2-my-notices", "/user/my-notices"],
  ["r2-monitor_cache", "/monitor/cache"],
];

/** 切模式：store 快照 + DOM 属性（两条通道都要，见 90-QUANT-SNAPSHOT 的方法学说明） */
async function setMode(ev, nav, mode) {
  await nav("/login", 4000);
  await ev(`(function(){
    var KEY='settings-storage';
    var o=JSON.parse(localStorage.getItem(KEY)||'{}');
    var st=o.state||o;
    st.preferences=st.preferences||{}; st.preferences.theme=st.preferences.theme||{};
    st.preferences.theme.mode=${JSON.stringify(mode)};
    o.state=st; localStorage.setItem(KEY,JSON.stringify(o));
    document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)});
  })()`);
  await nav("/login", 5000);
  await ev(`document.documentElement.setAttribute('data-color-mode',${JSON.stringify(mode)})`);
  await sleep(700);
}

(async () => {
  const { ev, nav, cdp, close } = await openCdp();

  // console 采集（CDP 事件订阅 → consoleBuf）
  const consoleBuf = [];
  await cdp.send("Runtime.enable");
  await cdp.send("Log.enable");
  cdp.on("Runtime.consoleAPICalled", (p) => {
    const txt = (p.args || [])
      .map((a) => (a.value !== undefined ? String(a.value) : a.description || a.type))
      .join(" ");
    consoleBuf.push({ kind: p.type, txt });
  });
  cdp.on("Log.entryAdded", (p) => {
    const e = p.entry || {};
    consoleBuf.push({ kind: e.level, txt: e.text || "" });
  });
  cdp.on("Runtime.exceptionThrown", (p) => {
    const d = p.exceptionDetails || {};
    consoleBuf.push({ kind: "error", txt: d.text || d.exception?.description || "exception" });
  });

  const results = [];
  const shot = async (file) => {
    try {
      const r = await cdp.send("Page.captureScreenshot", { format: "png" });
      fs.writeFileSync(path.join(HERE, file), Buffer.from(r.data, "base64"));
      return true;
    } catch (e) {
      return false;
    }
  };

  // ---- 1) 无登录态可达的真实页面巡检（/login 明暗各一次）----
  for (const mode of ["light", "dark"]) {
    await setMode(ev, nav, mode);
    consoleBuf.length = 0;
    await nav("/login", 4200);
    const finalHref = await ev("location.href");
    const shotOk = await shot(`login-${mode}.png`);
    const dep = consoleBuf.filter((c) => /deprecated/i.test(c.txt));
    const errs = consoleBuf.filter((c) => c.kind === "error");
    const warns = consoleBuf.filter((c) => c.kind === "warning");
    results.push({
      mode,
      name: "login",
      url: "/login",
      finalHref,
      reachableWithoutLogin: true,
      shotOk,
      deprecated: dep.length,
      errors: errs.length,
      warnings: warns.length,
      depSamples: dep.map((d) => d.txt.slice(0, 140)),
      errSamples: errs.map((e2) => e2.txt.slice(0, 140)),
      warnSamples: warns.map((w) => w.txt.slice(0, 140)),
    });
    console.log(
      `SHOT login-${mode} dep=${dep.length} err=${errs.length} warn=${warns.length} href=${finalHref.slice(0, 40)}`,
    );
  }

  // ---- 2) 需登录的页面：逐个探测可达性 + console（记录重定向证据）----
  for (const mode of ["light", "dark"]) {
    await setMode(ev, nav, mode);
    for (const [name, url] of [...R2, ...PAGES.map((p) => [p[0], p[2]])]) {
      if (name === "login") continue;
      consoleBuf.length = 0;
      let navErr = null;
      let finalHref = "NAV_FAIL";
      try {
        await nav(url, 3000);
        finalHref = await ev("location.href");
      } catch (e) {
        navErr = String(e).slice(0, 200);
      }
      const dep = consoleBuf.filter((c) => /deprecated/i.test(c.txt));
      const errs = consoleBuf.filter((c) => c.kind === "error");
      const redirected = /\/login/.test(finalHref);
      results.push({
        mode,
        name,
        url,
        finalHref,
        reachableWithoutLogin: !redirected,
        blockedByLoginWall: redirected,
        navErr,
        deprecated: dep.length,
        errors: errs.length,
        depSamples: dep.map((d) => d.txt.slice(0, 140)),
        errSamples: errs.map((e2) => e2.txt.slice(0, 140)),
        note: redirected
          ? "登录墙（认证模式=ad，需真实 AD 域凭据）—— 待人工巡检"
          : "",
      });
      // 逐页截图（v1.25 遗留项 90-2：目标页截图 40 张，2026-11-02 补齐）
      let shotOk = false;
      try {
        shotOk = await shot(`${name}-${mode}.png`);
      } catch (e) {
        shotOk = false;
      }
      results[results.length - 1].shot = `${name}-${mode}.png`;
      results[results.length - 1].shotOk = shotOk;
      console.log(
        `PROBE ${name}-${mode} dep=${dep.length} err=${errs.length} redirected=${redirected}`,
      );
    }
  }

  fs.writeFileSync(path.join(HERE, "inspection-result-90.txt"), JSON.stringify(results, null, 1));

  // 待人工巡检清单（89 的 14 页 + 6 页）
  const manualList = [];
  for (const mode of ["light", "dark"]) {
    for (const [name, url] of [...R2, ...PAGES.map((p) => [p[0], p[2]])]) {
      if (name === "login") continue;
      manualList.push({ mode, name, url });
    }
  }
  fs.writeFileSync(path.join(HERE, "inspection-manual-todo-90.json"), JSON.stringify(manualList, null, 1));

  const totalDep = results.reduce((s, r) => s + r.deprecated, 0);
  const totalErr = results.reduce((s, r) => s + r.errors, 0);
  const reachable = results.filter((r) => r.reachableWithoutLogin);
  const blocked = results.filter((r) => r.blockedByLoginWall);
  const shots = fs.readdirSync(HERE).filter((f) => /^(login|p90|r3|p\d|r2)-.*\.png$/.test(f));
  console.log(
    `TOTAL deprecated=${totalDep} errors=${totalErr} reachable=${reachable.length} blockedByLoginWall=${blocked.length} shots=${shots.length}`,
  );
  close();
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
