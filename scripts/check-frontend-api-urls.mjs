#!/usr/bin/env node
/**
 * check-frontend-api-urls.mjs —— 前端 URL ↔ 后端路由 交叉校验（CI 守卫）。
 *
 * 背景：monitor/logs /clear 与 workorder 周期工单端点两个真实断链长期漏检，
 * 根因之一是前端调用的 URL 与后端注册路由（scripts/route-registry.json，由
 * scripts/scan-routes.cjs 生成）之间没有自动交叉校验。本脚本补上这一环。
 *
 * 设计要点（低误报优先，宁漏勿误）：
 *   1. 只提取「确定是后端 API 调用」上下文中的 URL 字面量：
 *      - ./api 导出的传输函数 post/get/put/del/postFormData（含 api.post 等）第一参；
 *      - downloadFile / downloadFilePost 第一参；
 *      - apiFactory 的 createResourceApi({ basePath }) —— 工厂派生端点也纳入校验，
 *        但改为「按工厂变量实际被消费的方法精确派生」（见第 5 点）。
 *      页面里的 navigate('/login')、<Link to=...>、路由表等前端路由不在上述
 *      函数上下文中，天然不会被提取。
 *   2. 注释内容先剥离（保持行结构），注释里的示例 URL 不校验。
 *   3. 模板字符串 ${...} 槽位一律归一化为 :p；gin 路由 :xxx 也归一化为 :p；
 *      纯数字段归一化为 :p。匹配用逐段通配（:p 匹配任意单段）。
 *   4. 前缀：axios baseURL 统一为 "/api/v1"（src/lib/api.ts），registry 路径
 *      不带该前缀；前端个别调用误带 /api/v1（双重前缀，本身就是隐患）剥离后
 *      参与比对，缺失即 FAIL。
 *   5. 工厂派生精确化（WARN 清零整改）：
 *      旧版对每个 createResourceApi 调用机械派生 8 条 URL，即使前端只 pick 了
 *      其中 1~3 个方法（其余是「工厂生成但从未被调用」的幽灵端点），于是产生
 *      16 条静默放行的 WARN。现按同文件内该工厂变量的实际消费形态派生：
 *        - const x = createResourceApi(...) + x.list / x.get / ... → 仅派生
 *          被引用方法对应的 URL；
 *        - export const x = createResourceApi(...)（导出对象，外部页面消费，
 *          文件内不可静态判定）→ 8 方法全派生；
 *        - { ...x } spread 展开 → 8 方法全派生；
 *        - 一个方法都没消费的局部工厂变量 → 不派生任何 URL，仅打印提示。
 *      这样「后端无实现 + 前端未调用」的死派生函数不再进入校验集合，从源头
 *      消除 WARN，而不是靠豁免掩盖。
 *
 * 已知漏检面（有意不报，控制误报）：
 *   - 非上述函数名的自定义封装（如 dataFetcher 内部拼 URL 后再调 api.*）；
 *   - 动态拼接 URL（a + '/' + b、数组 join 等）；
 *   - 测试文件（*.test.* / __tests__）中的 mock URL；
 *   - createResourceApi 之外的工厂/生成器若引入新派生端点模式，需在本脚本同步。
 *
 * 严格模式（整改后启用）：
 *   - 任何「显式调用字面量不匹配」或「工厂派生不匹配」都使 exit=1，不再静默
 *     放行 WARN。存量已清零，新出现的派生不匹配必须改 URL / 删死方法 / 补后端
 *     路由 / 登记待实现台账，四选一。
 *   - 「确有需求但后端暂未实现」的派生端点登记在下方 PENDING_DERIVED 台账，
 *     强制 reason + 清偿窗口（window），命中登记的派生不产生 WARN、不计入豁免
 *     命中、不写进 frontend-url-exemptions.json；到期须复核清除。
 *
 * 豁免：scripts/frontend-url-exemptions.json，条目 { url, file?, reason }，
 * reason 必填；url 为归一化前的原始 URL（可含模板槽原样写法）。
 * 治理规则见 scripts/EXEMPTIONS-POLICY.md。
 *
 * 用法：node scripts/check-frontend-api-urls.mjs（由 lint:ci 链尾调用）
 * 退出码：0 全过；1 存在不匹配/配置错误/工厂派生 WARN。
 */
import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, "..");
const frontendRoot = path.join(repoRoot, "xingran-react-frontend");
const registryPath = path.join(repoRoot, "scripts", "route-registry.json");
const exemptionsPath = path.join(repoRoot, "scripts", "frontend-url-exemptions.json");

// ---------- 待实现派生端点台账（不产生 WARN、不计入豁免命中） ----------
// 仅用于「确有需求但后端暂未实现」的工厂派生端点；强制 reason + 清偿窗口。
// file 为相对 repoRoot 的 POSIX 路径；method ∈ DERIVED_METHODS；
// window 为清偿窗口（如 "Phase N 之前" / "2026-Q4"），到期须复核清除。
// 整改当下为 0 条：存量 16 条 WARN 全部是「前端未调用的死派生」，已由精确派生
// 从校验集合中移除，无需登记任何待实现项。
const PENDING_DERIVED = [
  // { file: "xingran-react-frontend/src/lib/xxxApi.ts", basePath: "/xxx",
  //   method: "statistics", reason: "...", window: "...", }
];

// ---------- 后端 registry ----------
const registry = JSON.parse(readFileSync(registryPath, "utf8"));
const backendRoutes = (registry.routes || []).concat(
  (registry.deferred && registry.deferred.items) || [],
);
// 归一化集合（精确）与分段树（通配）
const normSeg = (s) => (s.startsWith(":") || /^\d+$/.test(s) ? ":p" : s);
const normPath = (p) =>
  p
    .split("/")
    .filter(Boolean)
    .map(normSeg)
    .join("/");
const exact = new Set();
const trees = new Map(); // method -> 树节点
for (const r of backendRoutes) {
  const np = normPath(r.path);
  exact.add(np);
  let node = trees.get(r.method) ?? (trees.set(r.method, new Map()), trees.get(r.method));
  for (const seg of np.split("/")) {
    if (!node.has(seg)) node.set(seg, new Map());
    node = node.get(seg);
  }
  node.set("__leaf__", true);
}
function matchTree(method, np) {
  const root = trees.get(method) ?? trees.get("POST");
  if (!root) return false;
  const segs = np.split("/");
  const walk = (node, i) => {
    if (i === segs.length) return node.get("__leaf__") === true;
    const s = segs[i];
    if (node.has(s) && walk(node.get(s), i + 1)) return true;
    if (s === ":p") {
      // 前端参数段可命中后端枚举字面量段（如 /ops/${entity}/template ↔
      // /ops/building/template 等 8 条逐实体注册），仅限前端归一化为 :p 的槽位
      for (const [k, child] of node) {
        if (k !== "__leaf__" && walk(child, i + 1)) return true;
      }
      return false;
    }
    if (node.has(":p")) return walk(node.get(":p"), i + 1);
    return false;
  };
  return walk(root, 0);
}
// 允许前端 GET/PUT/DELETE 调用命中后端同名或 POST 组；registry 已含全部方法，
// 先精确方法，失败再试全方法集合兜底（POST 为主路由形态）。
function backendHas(method, np) {
  if (exact.has(np)) return true;
  if (trees.has(method) && matchTree(method, np)) return true;
  for (const m of trees.keys()) if (matchTree(m, np)) return true;
  return false;
}

// ---------- 豁免 ----------
let exemptions = [];
if (existsSync(exemptionsPath)) {
  const raw = JSON.parse(readFileSync(exemptionsPath, "utf8"));
  if (!Array.isArray(raw)) fail(`豁免文件结构错误：${exemptionsPath} 顶层必须是数组`);
  raw.forEach((e, i) => {
    if (!e.url || typeof e.url !== "string") fail(`豁免条目 #${i} 缺少 url`);
    if (!e.reason || typeof e.reason !== "string" || !e.reason.trim())
      fail(`豁免条目 #${i}（${e.url}）缺少 reason —— 每条豁免必须写明理由与追踪事项`);
  });
  exemptions = raw;
}

function fail(msg) {
  console.error("[frontend-api-urls] " + msg);
  process.exit(1);
}

// ---------- 待实现台账 schema 校验（reason + 清偿窗口必填） ----------
const DERIVED_METHODS = [
  "create",
  "list",
  "get",
  "update",
  "delete",
  "batch",
  "statistics",
  "searchOptions",
];
for (const [i, p] of PENDING_DERIVED.entries()) {
  if (!p.file || !p.basePath || !DERIVED_METHODS.includes(p.method))
    fail(`待实现台账 #${i} 缺少合法 file / basePath / method`);
  if (!p.reason || typeof p.reason !== "string" || !p.reason.trim())
    fail(`待实现台账 #${i}（${p.file} ${p.method}）缺少 reason`);
  if (!p.window || typeof p.window !== "string" || !p.window.trim())
    fail(`待实现台账 #${i}（${p.file} ${p.method}）缺少清偿窗口 window`);
}

// ---------- 前端扫描 ----------
const SCAN_DIRS = ["src/lib", "src/api", "src/pages", "src/components", "src/hooks"];
const EXTS = [".ts", ".tsx"];

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = path.join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) {
      if (name === "__tests__" || name === "node_modules") continue;
      yield* walk(p);
    } else if (EXTS.includes(path.extname(name)) && !/\.(test|spec)\./.test(name)) {
      yield p;
    }
  }
}

/** 剥离注释，保持行结构（替换为空格），返回 { text } */
function stripComments(src) {
  let out = "";
  let i = 0;
  const n = src.length;
  let inS = null; // ', ", `
  while (i < n) {
    const c = src[i];
    const c2 = src[i + 1];
    if (!inS && c === "/" && c2 === "/") {
      while (i < n && src[i] !== "\n") out += " ", i++;
      continue;
    }
    if (!inS && c === "/" && c2 === "*") {
      while (i < n && !(src[i] === "*" && src[i + 1] === "/")) {
        out += src[i] === "\n" ? "\n" : " ";
        i++;
      }
      out += "  ";
      i += 2;
      continue;
    }
    if (inS && c === "\\") {
      out += c + (c2 ?? "");
      i += 2;
      continue;
    }
    if (!inS && (c === "'" || c === '"' || c === "`")) inS = c;
    else if (inS === c) inS = null;
    out += c;
    i++;
  }
  return out;
}

/** 行号计算：offset 前的换行数 +1 */
function lineOf(text, offset) {
  let line = 1;
  for (let i = 0; i < offset; i++) if (text[i] === "\n") line++;
  return line;
}

/** 前端 URL 归一化：剥 /api/v1 前缀、模板槽/参数段/数字段 → :p */
function normalizeFrontendUrl(raw) {
  let u = raw.trim();
  u = u.replace(/^\/api\/v1(?=\/)/, ""); // axios baseURL 已含 /api/v1
  u = u.replace(/[?#].*$/, ""); // query/hash 不参与路由匹配（如 ?${params}）
  const np = normPath(u);
  return { original: raw, normalized: np };
}

/** 传输函数名（第一参为 URL） */
const CALL_RE =
  /(?:^|[^\w.$])(post|get|put|del|postFormData|downloadFile|downloadFilePost|api\.(?:post|get|put|delete))\s*(?:<[^>()]*>)?\s*\(\s*(["'`])((?:\\.|(?!\2).)*?)\2/gs;
/**
 * 工厂调用：捕获 (export 前缀, 变量名, config body)。
 * 变量名用于在同一文件内查找实际被消费的方法，实现精确派生。
 */
const FACTORY_RE =
  /(export\s+)?const\s+([A-Za-z_$][\w$]*)\s*=\s*createResourceApi\s*(?:<[^>()]*>)?\s*\(\s*\{([\s\S]*?)\}\s*\)\s*;/g;

/** 工厂方法 → 派生 URL（b 为已去尾斜杠的 basePath，d 为 dropdownPath） */
function derivedUrl(method, b, d) {
  switch (method) {
    case "create": return b;
    case "list": return `${b}/list`;
    case "get": return `${b}/:p`;
    case "update": return `${b}/:p/update`;
    case "delete": return `${b}/:p/delete`;
    case "batch": return `${b}/batch`;
    case "statistics": return `${b}/statistics`;
    case "searchOptions": return `${b}${d}`;
    default: return null;
  }
}

/** 转义正则元字符（变量名安全嵌入动态正则） */
function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function extractFromFile(absPath) {
  const rel = path.relative(repoRoot, absPath).replace(/\\/g, "/");
  const text = stripComments(readFileSync(absPath, "utf8"));
  const found = [];

  for (const m of text.matchAll(CALL_RE)) {
    const fn = m[1];
    const rawUrl = m[3];
    if (!rawUrl.startsWith("/")) continue; // 相对路径/空串不校验
    // 模板字符串槽位 → :p；普通字符串中的 ${} 不存在
    const withSlots = rawUrl.replace(/\$\{[^}]*\}/g, ":p");
    if (withSlots.includes("${")) continue; // 拿不准不报
    found.push({
      file: rel,
      line: lineOf(text, m.index),
      url: withSlots,
      rawUrl,
      context: fn,
    });
  }

  // apiFactory 派生端点（精确派生：只包含该工厂变量实际被消费的方法）
  for (const m of text.matchAll(FACTORY_RE)) {
    const isExported = Boolean(m[1]);
    const varName = m[2];
    const body = m[3];
    const base = body.match(/basePath\s*:\s*["'`]([^"'`]+)["'`]/);
    const dropdown = body.match(/dropdownPath\s*:\s*["'`]([^"'`]+)["'`]/);
    if (!base || !base[1].startsWith("/")) continue;
    const b = base[1].replace(/\/$/, "");
    const d = dropdown && dropdown[1].startsWith("/") ? dropdown[1] : "/dropdown-options";
    const line = lineOf(text, m.index);

    // 该变量是否被 spread 展开（{ ...x }）——展开即 8 方法全暴露，静态不可再收窄
    const spread = new RegExp(`\\.\\.\\.${escapeRegExp(varName)}\\b`).test(text);
    // 该变量被显式引用的方法集合（x.list / x.get / ...）
    const used = new Set();
    const refRe = new RegExp(`\\b${escapeRegExp(varName)}\\.([A-Za-z]+)\\b`, "g");
    for (const r of text.matchAll(refRe)) {
      if (DERIVED_METHODS.includes(r[1])) used.add(r[1]);
    }

    // 导出对象（外部消费）或 spread → 8 方法全派生；否则仅派生实际引用的方法
    const methods = isExported || spread ? DERIVED_METHODS : DERIVED_METHODS.filter((x) => used.has(x));
    if (methods.length === 0) {
      console.log(`[frontend-api-urls] 提示：${rel}:${line} 工厂变量 ${varName} 未消费任何方法（疑似死变量，建议删除）`);
    }

    for (const method of methods) {
      const u = derivedUrl(method, b, d);
      if (u === null) continue;
      found.push({
        file: rel,
        line,
        url: u,
        rawUrl: u,
        context: "createResourceApi",
        derived: true,
        derivedMethod: method,
        derivedBase: b,
      });
    }
  }
  return found;
}

// ---------- 主流程 ----------
const all = [];
const files = [];
for (const d of SCAN_DIRS) {
  const abs = path.join(frontendRoot, d);
  if (!existsSync(abs)) continue;
  for (const f of walk(abs)) files.push(f);
}
for (const f of files) all.push(...extractFromFile(f));

// 去重（同文件同 URL 同行只报一次；apiFactory basePath 生成物按 (file,url) 去重）
const seen = new Set();
const uniq = all.filter((e) => {
  const k = `${e.file}:${e.line}:${e.url}`;
  if (seen.has(k)) return false;
  seen.add(k);
  return true;
});

const isPending = (e) =>
  PENDING_DERIVED.some(
    (p) =>
      p.file === e.file &&
      p.basePath === e.derivedBase &&
      p.method === e.derivedMethod,
  );

const usedExemptions = new Set();
const failures = [];
const warns = [];
const pendingHits = [];
const matchedCount = { total: 0 };
for (const e of uniq) {
  const { normalized } = normalizeFrontendUrl(e.url);
  if (backendHas("POST", normalized)) {
    matchedCount.total++;
    continue;
  }
  const exIdx = exemptions.findIndex(
    (x) =>
      x.url === e.url ||
      x.url === normalized ||
      normPath(String(x.url).replace(/\$\{[^}]*\}/g, ":p")) === normalized,
  );
  if (exIdx >= 0 && (!exemptions[exIdx].file || exemptions[exIdx].file === e.file)) {
    usedExemptions.add(exIdx);
    continue;
  }
  // 已登记「待实现」的派生端点：不产生 WARN、不阻断，仅提示到期复核
  if (e.derived && isPending(e)) {
    pendingHits.push(e);
    continue;
  }
  // 严格模式：工厂派生不匹配与显式调用字面量不匹配一律阻断（exit=1）
  if (e.derived) {
    warns.push({ ...e, normalized });
    continue;
  }
  failures.push({ ...e, normalized });
}

// ---------- 输出 ----------
console.log(`[frontend-api-urls] 扫描 ${files.length} 个文件，提取 ${uniq.length} 条前端 API URL；` +
  `后端 registry ${backendRoutes.length} 条路由；命中 ${matchedCount.total} 条，` +
  `豁免命中 ${usedExemptions.size} 条，工厂派生告警 ${warns.length} 条，` +
  `待实现登记 ${pendingHits.length} 条，不匹配 ${failures.length} 条。`);

for (const p of pendingHits) {
  const entry = PENDING_DERIVED.find(
    (x) => x.file === p.file && x.basePath === p.derivedBase && x.method === p.derivedMethod,
  );
  console.log(`  PENDING(待实现，清偿窗口 ${entry.window}) ${p.file}:${p.line}  ${p.url}  —— ${entry.reason}`);
}

for (const w of warns)
  console.log(`  WARN(工厂派生) ${w.file}:${w.line}  ${w.url}  后端无匹配（${w.derivedMethod}）`);

for (const f of failures) {
  console.error(`  FAIL ${f.file}:${f.line}  ${f.url}  （归一化 ${f.normalized}，上下文 ${f.context}，后端路由无匹配）`);
}

if (warns.length) {
  console.error(`\n[frontend-api-urls] 存在 ${warns.length} 条工厂派生 URL 在后端 route-registry.json 中无匹配。`);
  console.error("  处置：删除该死派生方法 / 修正 URL / 补后端路由 / 在 PENDING_DERIVED 登记待实现（须写 reason + 清偿窗口）。");
}
if (failures.length) {
  console.error(`\n[frontend-api-urls] 存在 ${failures.length} 条前端 URL 在后端 route-registry.json 中无匹配。`);
  console.error("  处置：改 URL / 下线调用 / 在 scripts/frontend-url-exemptions.json 登记（必须写 reason）。");
}
// 严格模式：任何 WARN 或不匹配都使 exit=1
if (warns.length || failures.length) process.exit(1);

const unused = exemptions.filter((_, i) => !usedExemptions.has(i));
for (const u of unused)
  console.log(`[frontend-api-urls] 提示：豁免条目已无对应不匹配项，可清理：${u.url}（${u.file ?? "任意文件"}）`);

console.log("[frontend-api-urls] 全部前端 API URL 与后端路由匹配，检查通过。");
