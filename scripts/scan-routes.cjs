#!/usr/bin/env node
/**
 * scan-routes.cjs —— 路由注册静态扫描器（Phase 85 Plan 01 / RBAC-03 验收底座）。
 *
 * 事实源 = internal/api/** 路由注册源码重扫（tmp_routes.json 仅盘点参考，不作事实源）。
 *
 * 解析能力（相对 tmp_scan_final.cjs 正规化）：
 *   ① 组前缀链展开：var := parent.Group("/x") 逐级拼接（router.go: system → authorized → roles …）
 *   ② SetupXxxRouter 跨文件绑定：挂载组前缀 + Require* 权限中间件随实参传入被调函数
 *   ③ struct-literal 写组：SetupUserWriteRouter(UserWriteRouteGroups{Add: strictUserRouteGroup(...)})
 *   ④ 内联中间件：r.POST("/x", middleware.RequireXxx([]string{"..."}), handler)
 *   ⑤ setup 函数体内局部子组（g := r.Group(...) / g.Use(...)）
 *
 * 产物：scripts/route-registry.json（入库，幂等可重复生成）
 *   { totals, routes: [{method, path, mountGroup, isWrite, ...}], deferred: {items} }
 *   deferred = isWrite=true 但 GetRoutePermissions()（pkg/permission/config.go）未登记的写路由，
 *   由扫描器按当前死表现状派生并冻结入库；85-02/03 补登记时应同步收缩该清单。
 *
 * 用法：
 *   node scripts/scan-routes.cjs                 # 写 scripts/route-registry.json
 *   node scripts/scan-routes.cjs --out <file>    # 写到指定文件（供校验方做 freshness 比对）
 *   const { scanRoutes } = require("./scan-routes.cjs")  # 供 check-route-registry.mjs / go test 复用
 */
"use strict";
const fs = require("fs");
const path = require("path");

const REPO_ROOT = path.resolve(__dirname, "..");
const API_DIR = path.join(REPO_ROOT, "internal", "api");
const CONFIG_GO = path.join(REPO_ROOT, "pkg", "permission", "config.go");

// 写操作 handler 名判定（沿用 tmp_scan_final.cjs WRITE_HANDLER 口径，不扩不缩）
const WRITE_HANDLER = /(Create|Update|Delete|BatchDelete|Batch|Remove|Import|Export|Publish|Withdraw|Reset|Sync|Execute|Dispatch|Generate|Merge|Adjust|Apply|Rollback|Accept|Reject|Resolve|Toggle|Enable|Disable|Unlock|Assign|Convert|Refresh|Invalidate|Warmup|Clear|Clean|Operate|Start|Stop|Restart|Move|Clone|Preview|Quick|Manual|Swap|SetDefault|SetPrimary|SetDescription|SetAccessVlan|PortBinding|Shutdown|UndoShutdown|BindUser|UnbindUser|AddManual|AddComment|Like|TestConnection|SyncData|SyncAD|SyncAsset|SyncFromVDI|SyncAll|Upload|ImportUser|ImportDevices|Write|Store|Persist)/;

// ---------------------------------------------------------------- 基础工具

function walkGoFiles(dir, acc = []) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) walkGoFiles(p, acc);
    else if (e.name.endsWith(".go") && !e.name.endsWith("_test.go")) acc.push(p);
  }
  return acc;
}

const rel = (p) => path.relative(REPO_ROOT, p).split(path.sep).join("/");

/** 从字符串 s 的 fromIndex（须停在某个 "(" 上）做括号平衡，返回 { body, end } */
function balancedParens(s, fromIndex) {
  let depth = 0, i = fromIndex, inStr = false, strCh = "";
  for (; i < s.length; i++) {
    const ch = s[i];
    if (inStr) {
      if (ch === "\\") { i++; continue; }
      if (ch === strCh) inStr = false;
      continue;
    }
    if (ch === '"' || ch === "'" || ch === "`") { inStr = true; strCh = ch; continue; }
    if (ch === "(") depth++;
    else if (ch === ")") { depth--; if (depth === 0) return { body: s.slice(fromIndex + 1, i), end: i }; }
  }
  return { body: s.slice(fromIndex + 1), end: s.length };
}

/** 提取 middleware.RequireXxx(...) 参数里的权限码（字符串字面量 + permission.X 常量） */
function extractPermCodes(constMap, argBody) {
  const codes = [];
  for (const m of argBody.matchAll(/"([a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+)"/g)) codes.push(m[1]);
  for (const m of argBody.matchAll(/permission\.([A-Za-z_]\w*)/g)) {
    const v = constMap.get(m[1]);
    if (v) codes.push(v);
  }
  return codes;
}

// ---------------------------------------------------------------- 常量表

function parseConstMap() {
  const src = fs.readFileSync(CONFIG_GO, "utf8");
  const map = new Map();
  const re = /([A-Za-z_]\w*)\s+PermissionCode\s*=\s*"([^"]+)"/g;
  let m;
  while ((m = re.exec(src)) !== null) map.set(m[1], m[2]);
  return map;
}

// ---------------------------------------------------------------- 死表登记解析

/** 解析 GetRoutePermissions() 返回的登记条目：{"/path", "METHOD", Const, "desc"} */
function parseDeadTable(constMap) {
  const src = fs.readFileSync(CONFIG_GO, "utf8");
  const start = src.indexOf("func GetRoutePermissions()");
  const end = src.indexOf("\nfunc ", start + 1);
  const body = src.slice(start, end === -1 ? src.length : end);
  const items = [];
  const re = /\{\s*"([^"]+)"\s*,\s*"([A-Z]+)"\s*,\s*([A-Za-z_]\w*)\s*,\s*"[^"]*"\s*\}/g;
  let m;
  while ((m = re.exec(body)) !== null) {
    items.push({ path: m[1], method: m[2], code: constMap.get(m[3]) || m[3] });
  }
  return items;
}

/** 与 pkg/permission/config.go matchPath 语义一致：精确相等，或段数一致逐段、:前缀段跳过 */
function matchPath(pattern, p) {
  if (pattern === p) return true;
  const a = pattern.split("/"), b = p.split("/");
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i].startsWith(":")) continue;
    if (a[i] !== b[i]) return false;
  }
  return true;
}

// ---------------------------------------------------------------- 单文件静态结构

/**
 * 解析一个 .go 文件，产出：
 *   groups: Map var -> { parent, prefix }
 *   uses:   Map var -> [{ kind, codes }]
 *   registrations: [{ var, method, argStart(在整文件文本中的 index), line, inlineKind, inlineCodes, handler }]
 *   setupCalls:    [{ name, argsText, line, argIndex(文本 index) }]
 *   setupDefs:     Map name -> { param, bodyStart, bodyEnd(文本 index 范围) }
 */
function parseFile(file, constMap) {
  const text = fs.readFileSync(file, "utf8");
  const lineAt = (idx) => text.slice(0, idx).split(/\r?\n/).length;

  const groups = new Map();
  const uses = new Map();
  const registrations = [];
  const setupCalls = [];
  const setupDefs = new Map();

  const reGroup = /([A-Za-z_][\w.]*)\s*:?=\s*([A-Za-z_][\w.]*)\.Group\("([^"]*)"\)/g;
  const reUse = /([A-Za-z_][\w.]*)\.Use\(middleware\.(Require\w+)\s*\(/g;
  const reReg = /([A-Za-z_][\w.]*)\.(POST|GET|PUT|DELETE|PATCH)\s*\(/g;
  const reCall = /\b(?:[A-Za-z_][\w.]*\.)?(Setup[A-Z]\w*)\s*\(/g;
  const reDef = /func\s+(Setup[A-Z]\w*)\s*\(\s*([A-Za-z_]\w*)\s+(?:\*gin\.RouterGroup|UserWriteRouteGroups|[A-Za-z_*][\w.]*)/g;

  let m;
  while ((m = reGroup.exec(text)) !== null) groups.set(m[1], { parent: m[2], prefix: m[3] });

  while ((m = reUse.exec(text)) !== null) {
    const { body } = balancedParens(text, reUse.lastIndex - 1);
    const codes = extractPermCodes(constMap, body);
    if (!uses.has(m[1])) uses.set(m[1], []);
    uses.get(m[1]).push({ kind: m[2], codes, line: lineAt(m.index) });
  }

  while ((m = reReg.exec(text)) !== null) {
    const { body, end } = balancedParens(text, reReg.lastIndex - 1);
    // handler 名 = 参数列表最后一个标识符（如 "handler.List" / "core.Handler"）
    const hm = body.match(/([A-Za-z_]\w*)\s*[,)]\s*$/) || body.match(/([A-Za-z_]\w*)$/);
    // 内联权限中间件
    let inlineKind = null, inlineCodes = [];
    const im = body.match(/middleware\.(Require\w+)\s*\(/);
    if (im) {
      const ib = balancedParens(body, im.index + im[0].length - 1);
      inlineKind = im[1];
      inlineCodes = extractPermCodes(constMap, ib.body);
    }
    registrations.push({
      varName: m[1], method: m[2], line: lineAt(m.index), span: [m.index, end],
      handler: hm ? hm[1] : "", inlineKind, inlineCodes,
    });
  }

  while ((m = reCall.exec(text)) !== null) {
    // 排除函数定义行本身
    const before = text.slice(Math.max(0, m.index - 6), m.index);
    if (/func\s+$/.test(before)) continue;
    const { body } = balancedParens(text, reCall.lastIndex - 1);
    setupCalls.push({ name: m[1], argsText: body, line: lineAt(m.index), index: m.index });
  }

  while ((m = reDef.exec(text)) !== null) {
    // 函数体范围：从签名后第一个 "{" 到配对 "}"
    const bo = text.indexOf("{", reDef.lastIndex);
    let depth = 0, i = bo, end = text.length;
    for (; i < text.length; i++) {
      if (text[i] === "{") depth++;
      else if (text[i] === "}") { depth--; if (depth === 0) { end = i; break; } }
    }
    setupDefs.set(m[1], { param: m[2], start: bo, end });
  }

  return { file: rel(file), text, groups, uses, registrations, setupCalls, setupDefs };
}

// ---------------------------------------------------------------- 路由实例化

function scanRoutes() {
  const constMap = parseConstMap();
  const deadTable = parseDeadTable(constMap);
  const files = walkGoFiles(API_DIR);
  const parsed = files.map((f) => parseFile(f, constMap));

  const routes = [];
  const visited = new Set();

  /**
   * 解析 var 的挂载（prefix 链 + 权限码链）。
   * ctx = { analysis, paramVar, paramMount, fieldMounts(Map "param.Field" -> mount), depth }
   * 返回 { prefix, codes, kinds, leafVar }
   */
  function resolveVar(ctx, varName, depth = 0) {
    const { analysis } = ctx;
    const mount = { prefix: "", codes: [], kinds: [], leafVar: varName };
    if (depth > 12) return mount;

    let chain = [];
    let cur = varName;
    while (true) {
      if (cur === ctx.paramVar) break; // 链根 = 参数挂载
      // struct 字段挂载：var 形如 "<param>.<Field>"
      if (cur.startsWith(ctx.paramVar + ".")) {
        const fieldMount = ctx.fieldMounts.get(cur.slice(ctx.paramVar.length + 1));
        if (fieldMount) {
          mount.prefix = fieldMount.prefix + mount.prefix;
          mount.codes.push(...fieldMount.codes, ...mount.codes);
          mount.kinds.push(...(fieldMount.kinds || []), ...mount.kinds);
          mount.leafVar = fieldMount.leafVar || cur;
          return mount;
        }
        break;
      }
      const g = analysis.groups.get(cur);
      if (!g) break;
      chain.unshift({ varName: cur, g });
      if (g.parent === ctx.paramVar || ctx.fieldMounts.has(g.parent) || analysis.groups.has(g.parent)) {
        cur = g.parent;
      } else break;
    }

    // 链根挂载：参数 / struct 字段 / 内联调用点匿名组（记录在 ctx.anonMounts）
    let base = null;
    if (cur === ctx.paramVar) base = ctx.paramMount;
    else if (ctx.fieldMounts.has(cur)) base = ctx.fieldMounts.get(cur);
    else if (ctx.anonMounts && ctx.anonMounts.has(cur)) base = ctx.anonMounts.get(cur);
    if (base) {
      mount.prefix = base.prefix;
      mount.codes.push(...base.codes);
      mount.kinds.push(...(base.kinds || []));
      mount.leafVar = base.leafVar || varName;
    }

    for (const { varName: v } of chain) {
      const g = analysis.groups.get(v);
      mount.prefix += g.prefix;
      mount.leafVar = v;
      const us = analysis.uses.get(v) || [];
      for (const u of us) { mount.codes.push(...u.codes); mount.kinds.push(u.kind); }
    }
    const selfUses = analysis.uses.get(varName) || [];
    for (const u of selfUses) { mount.codes.push(...u.codes); mount.kinds.push(u.kind); }
    return mount;
  }

  /**
   * 实例化一个 setup 函数：paramMount 为实参挂载，fieldMounts 为 struct 字段挂载。
   */
  function instantiate(setupName, paramMount, fieldMounts, depth) {
    let def = null, analysis = null;
    for (const a of parsed) {
      if (a.setupDefs.has(setupName)) { def = a.setupDefs.get(setupName); analysis = a; break; }
    }
    if (!def || depth > 4) return;

    // 防重复实例化（同一 setup + 同一挂载）
    const key = `${setupName}|${paramMount.prefix}|${paramMount.codes.join(",")}|${JSON.stringify(Object.fromEntries(fieldMounts))}`;
    if (visited.has(key)) return;
    visited.add(key);

    // 函数体切片
    const bodyText = analysis.text.slice(def.start, def.end);
    const bodyOffset = def.start;

    const ctx = {
      analysis, paramVar: def.param, paramMount, fieldMounts,
      anonMounts: new Map(),
    };

    // 1) 收集 setup 调用点（先于 registration，避免把调用点里的 .Group() 误当注册）
    const calls = [];
    const reCall = /\b(?:[A-Za-z_][\w.]*\.)?(Setup[A-Z]\w*)\s*\(/g;
    let m;
    while ((m = reCall.exec(bodyText)) !== null) {
      const before = bodyText.slice(Math.max(0, m.index - 6), m.index);
      if (/func\s+$/.test(before)) continue;
      const { body: argsText } = balancedParens(bodyText, reCall.lastIndex - 1);
      calls.push({ name: m[1], argsText, lineRel: bodyText.slice(0, m.index).split(/\r?\n/).length });
    }

    // 2) 函数体内局部组定义 / Use
    const localGroups = new Map(); // var -> { parent, prefix }
    const reGroup = /([A-Za-z_][\w.]*)\s*:?=\s*([A-Za-z_][\w.]*)\.Group\("([^"]*)"\)/g;
    while ((m = reGroup.exec(bodyText)) !== null) localGroups.set(m[1], { parent: m[2], prefix: m[3] });
    const reUse = /([A-Za-z_][\w.]*)\.Use\(middleware\.(Require\w+)\s*\(/g;
    while ((m = reUse.exec(bodyText)) !== null) {
      const { body } = balancedParens(bodyText, reUse.lastIndex - 1);
      const codes = extractPermCodes(constMap, body);
      if (!analysis.usesLocal) analysis.usesLocal = new Map();
      const k = analysis.file + "@" + setupName;
      if (!analysis.usesLocal.has(k)) analysis.usesLocal.set(k, new Map());
      const lm = analysis.usesLocal.get(k);
      if (!lm.has(m[1])) lm.set(m[1], []);
      lm.get(m[1]).push({ kind: m[2], codes });
    }
    // 85-03：局部 strict 写组赋值形态 var := (pkg.)strictUserRouteGroup(parent, permission.X, core)
    // （router.go 内联兄弟写组，非 struct-literal 实参形态）。prefix 继承 parent 组
    // （strictUserRouteGroup 是 parent.Group("")），权限码 = 单一常量，kind = Strict。
    const localStrict = new Map(); // var -> { parentVar, code }
    const reLocalStrict = /([A-Za-z_]\w*)\s*:?=\s*(?:[A-Za-z_][\w.]*\.)?strict\w*Group\(\s*([A-Za-z_]\w*)\s*,\s*(?:permission\.)?([A-Za-z_]\w*)/g;
    while ((m = reLocalStrict.exec(bodyText)) !== null) {
      const code = constMap.get(m[3]);
      localStrict.set(m[1], { parentVar: m[2], code: code || null });
    }

    // 局部 use 叠加：包装 resolveVar（局部组/局部 use 优先查函数体切片）
    const localResolve = (varName) => {
      // 85-03：局部 strict 写组（strictUserRouteGroup 赋值形态）——prefix 继承
      // parent 组（Group("")），叠加单一严格权限码后即止（不继承 parent 的 ANY-of）。
      const ls = localStrict.get(varName);
      if (ls) {
        const base = localResolve(ls.parentVar);
        return {
          prefix: base.prefix,
          codes: ls.code ? [ls.code] : [],
          kinds: ["RequireStrictPermissions(local)"],
          leafVar: varName,
        };
      }
      const mount = resolveVar(ctx, varName, 0);
      const lm = analysis.usesLocal && analysis.usesLocal.get(analysis.file + "@" + setupName);
      if (lm) {
        // 沿局部链叠加
        let cur = varName, guard = 0;
        while (guard++ < 12) {
          const us = lm.get(cur);
          if (us) for (const u of us) { mount.codes.push(...u.codes); mount.kinds.push(u.kind); }
          const lg = localGroups.get(cur);
          if (!lg) break;
          cur = lg.parent;
        }
      }
      return mount;
    };

    // 3) 处理 setup 调用点：解析实参挂载并递归
    for (const call of calls) {
      // 形态 A：struct literal  SetupX(pkg.UserWriteRouteGroups{Field: strictUserRouteGroup(var, permission.X, ...), ...})
      // 85-02：泛化为 *WriteRouteGroups（Role/Menu/Post/Dept 各域写组结构体同名后缀）
      const litMatch = call.argsText.match(/\w*WriteRouteGroups\s*\{([\s\S]*?)\}/);
      if (litMatch) {
        const fieldMounts2 = new Map();
        const fre = /(\w+)\s*:\s*(?:\w+\.)*strict\w*Group\(\s*([A-Za-z_]\w*)\s*,\s*permission\.([A-Za-z_]\w*)/g;
        let fm;
        while ((fm = fre.exec(litMatch[1])) !== null) {
          const base = localResolve(fm[2]);
          const code = constMap.get(fm[3]);
          fieldMounts2.set(fm[1], {
            prefix: base.prefix, codes: code ? [code] : [], kinds: ["RequireStrictPermissions(field)"], leafVar: fm[1],
          });
        }
        // 兜底：字段直接是普通变量
        const fre2 = /(\w+)\s*:\s*([A-Za-z_]\w*)\s*[,}]/g;
        while ((fm = fre2.exec(litMatch[1])) !== null) {
          if (fieldMounts2.has(fm[1])) continue;
          const base = localResolve(fm[2]);
          fieldMounts2.set(fm[1], base);
        }
        instantiate(call.name, paramMount, fieldMounts2, depth + 1);
        continue;
      }
      // 形态 B：实参 = 内联 X.Group("/prefix")
      const inlineGroup = call.argsText.match(/^\s*([A-Za-z_]\w*)\.Group\("([^"]*)"\)/);
      if (inlineGroup) {
        const base = localResolve(inlineGroup[1]);
        instantiate(call.name, { prefix: base.prefix + inlineGroup[2], codes: [...base.codes], kinds: [...base.kinds] }, new Map(), depth + 1);
        continue;
      }
      // 形态 C：实参 = 标识符（取第一个标识符实参）
      // 85-02：支持「写组字段直传」形态 SetupX(groups.Import, ...)（如 departments
      // 域 Excel import/export，字段经 resolveVar 的 <param>.<Field> 分支解析）
      const argVar = call.argsText.match(/^\s*((?:[A-Za-z_]\w*\.)?[A-Za-z_]\w*)\s*[,)]/);
      if (argVar) {
        const base = localResolve(argVar[1]);
        instantiate(call.name, { prefix: base.prefix, codes: [...base.codes], kinds: [...base.kinds] }, new Map(), depth + 1);
      }
    }

    // 4) 函数体内的路由注册（排除 setup 调用点区间内的误匹配）
    const callSpans = [];
    const reCallSpan = /\b(?:[A-Za-z_][\w.]*\.)?(Setup[A-Z]\w*)\s*\(/g;
    while ((m = reCallSpan.exec(bodyText)) !== null) {
      const { end } = balancedParens(bodyText, reCallSpan.lastIndex - 1);
      callSpans.push([m.index, end]);
    }
    const inCallSpan = (idx) => callSpans.some(([a, b]) => idx >= a && idx <= b);

    const reReg = /([A-Za-z_][\w.]*)\.(POST|GET|PUT|DELETE|PATCH)\s*\(/g;
    while ((m = reReg.exec(bodyText)) !== null) {
      if (inCallSpan(m.index)) continue;
      const { body: argsText, end } = balancedParens(bodyText, reReg.lastIndex - 1);
      const hm = argsText.match(/([A-Za-z_]\w*)\s*[,)]\s*$/) || argsText.match(/([A-Za-z_]\w*)$/);
      let inlineKind = null, inlineCodes = [];
      const im = argsText.match(/middleware\.(Require\w+)\s*\(/);
      if (im) {
        const ib = balancedParens(argsText, im.index + im[0].length - 1);
        inlineKind = im[1];
        inlineCodes = extractPermCodes(constMap, ib.body);
      }
      const mount = localResolve(m[1]);
      const pm = argsText.match(/^\s*"([^"]*)"/);
      const regPath = pm ? pm[1] : "";
      const fullPath = mount.prefix + regPath;
      const handlerName = hm ? hm[1] : "";
      const isWrite = m[2] !== "GET" && WRITE_HANDLER.test(handlerName);
      const codes = inlineCodes.length ? inlineCodes : mount.codes;
      const kinds = inlineKind ? [inlineKind] : mount.kinds;
      const registered = deadTable.some((d) => d.method === m[2] && matchPath(d.path, fullPath));
      routes.push({
        method: m[2],
        path: fullPath,
        mountGroup: mount.leafVar === m[1] ? mount.leafVar : `${mount.leafVar}>${m[1]}`,
        setup: setupName,
        isWrite,
        handler: handlerName,
        permKind: kinds.length ? kinds[kinds.length - 1] : null,
        permCodes: [...new Set(codes)],
        registeredDeadTable: registered,
        file: analysis.file,
        line: bodyText.slice(0, m.index).split(/\r?\n/).length + lineNoOfFuncStart(analysis, setupName),
      });
    }
  }

  function lineNoOfFuncStart(analysis, setupName) {
    const idx = analysis.text.indexOf("func " + setupName + "(");
    if (idx === -1) return 0;
    return analysis.text.slice(0, idx).split(/\r?\n/).length - 1;
  }

  // 顶层驱动：SetupRouter 挂在根（prefix=""，无 Require* 权限）
  instantiate("SetupRouter", { prefix: "", codes: [], kinds: [] }, new Map(), 0);

  routes.sort((a, b) => a.path.localeCompare(b.path) || a.method.localeCompare(b.method));

  // deferred：isWrite 且死表未登记
  const deferredItems = routes
    .filter((r) => r.isWrite && !r.registeredDeadTable)
    .map((r) => ({ path: r.path, method: r.method, handler: r.handler }));

  const mountPoints = new Set(routes.map((r) => `${r.file}:${r.setup}:${r.mountGroup}`));
  return {
    scanner: "scripts/scan-routes.cjs",
    sourceOfTruth: "internal/api/** route registration source (rescan), not tmp_routes.json",
    writeHandlerBasis: "WRITE_HANDLER regex on handler name (same as tmp_scan_final.cjs)",
    totals: {
      mounts: mountPoints.size,
      routes: routes.length,
      writeRoutes: routes.filter((r) => r.isWrite).length,
      deferredWriteRoutes: deferredItems.length,
    },
    routes,
    deferred: {
      note: "isWrite=true 但 GetRoutePermissions() 未登记的写路由（85-02/03 拆分同批收缩本清单）",
      items: deferredItems,
    },
  };
}

// ---------------------------------------------------------------- 死表漂移核对（重建质量自检）

function deadTableDriftCheck(result) {
  // 死表中登记了、但扫描不到对应实际路由的条目（本 plan 重建后应为 0）
  const liveKeys = new Set(result.routes.map((r) => `${r.method} ${r.path}`));
  const constMap = parseConstMap();
  const dead = parseDeadTable(constMap);
  const stale = [];
  for (const d of dead) {
    const hit = result.routes.some((r) => r.method === d.method && matchPath(d.path, r.path));
    if (!hit) stale.push(`${d.method} ${d.path} -> ${d.code}`);
  }
  return { stale, liveKeys };
}

module.exports = { scanRoutes, matchPath, deadTableDriftCheck, REPO_ROOT };

// ---------------------------------------------------------------- CLI

if (require.main === module) {
  const outIdx = process.argv.indexOf("--out");
  const outFile = outIdx !== -1 ? path.resolve(process.argv[outIdx + 1]) : path.join(REPO_ROOT, "scripts", "route-registry.json");
  const result = scanRoutes();

  fs.writeFileSync(outFile, JSON.stringify(result, null, 2) + "\n");

  console.log(`总路由: ${result.totals.routes}  写路由: ${result.totals.writeRoutes}  挂载点: ${result.totals.mounts}`);
  console.log(`死表未登记写路由(deferred): ${result.totals.deferredWriteRoutes}`);
  // 陈旧条目仅打印提示（死表重建质量自检）；拦截职责在 check-route-registry.mjs /
  // route_registry_test.go 的 coverage 校验，scan 本身永远产出事实、不设退出码。
  const { stale } = deadTableDriftCheck(result);
  console.log(`死表登记但无对应实际路由(陈旧条目, 仅提示): ${stale.length}`);
  stale.forEach((s) => console.log("  STALE: " + s));
  console.log(`written: ${path.relative(REPO_ROOT, outFile)}`);
}
