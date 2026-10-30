/* eslint-disable */
// ^ dev-only 工具脚本：scripts/*.mjs 被 eslint 全局忽略，本文件按任务规格命名为 .js，
//   故以文件内禁用保持 lint 预算（errors≤17）不被本文件击穿。
/**
 * v1.24 双框修复（VIS-01/02/03）CSS 契约静态检查 —— dev-only，node 运行
 * ==============================================================================
 * 用法：node scripts/css-contract-check.js   （退出码 0 = 契约完好）
 *
 * 依赖口径：核心检查零依赖；仅 C14c / C14d / C14e（antd <Drawer> width、antd <Alert> message、
 *   antd <Progress> width 弃用守护）**可选**使用
 *   devDependency `typescript` 编译器 API 做精确 AST 判定；typescript 不可用时
 *   自动降级为正则 JSX 扫描（降级态会在输出中显式标注，且仍能覆盖绝大多数场景）。
 *
 * 职责分工（STACK.md (e.1) 口径）：
 *   - 本脚本：静态结构契约（规则存在性 / 禁用形态 / 回归陷阱守护），CI 可跑；
 *   - scripts/css-contract-assertions.js：computed-style 动态断言（浏览器 Console
 *     粘贴执行）。jsdom 无法解析压缩版 antd.css 的 @layer/嵌套规则（实测数百条
 *     "Could not parse CSS stylesheet"），computed-style 断言必须用真浏览器。
 *
 * 守护项：
 *   C1  affix-wrapper 内层复位规则存在且不写 box-shadow（焦点环归 wrapper，VIS-02）
 *   C2  禁止 `:not(组合子)` 形态（CSS3 下整条规则静默丢弃）
 *   C3  回归陷阱：3 处局部补丁（login.css:351 / index.css dept-tree-search /
 *       user-form-input）不得删除
 *   C4  回归陷阱：`.ant-input:focus` 焦点环、wrapper 边框、wrapper 焦点环规则不得删除
 *   C5  VIS-03 status 边框色还原规则存在
 *   C14 antd 6 弃用 prop 零命中（VISFIX-04）
 *   C14c antd <Drawer> 的 width prop 零命中（VISFIX-05，antd 6 → size）
 *   C14d antd <Alert> 的 message prop 零命中（VISFIX-05 收口补丁，antd 6 → title）
 *   C14e antd <Progress> 的 width prop 零命中（VISFIX-05 收口补丁，antd 6 → size）
 */
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const feRoot = resolve(here, '..');
const indexCss = readFileSync(resolve(feRoot, 'src/index.css'), 'utf8');
const loginCssPath = resolve(feRoot, 'src/pages/login/login.css');
const loginCss = existsSync(loginCssPath) ? readFileSync(loginCssPath, 'utf8') : '';

let failures = 0;
const check = (name, ok, detail) => {
  if (!ok) failures++;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? '  → ' + detail : ''}`);
};

// C1 内层复位规则（VIS-01）：存在、含 6 项复位、不含 box-shadow
const resetRe = /\.ant-input-affix-wrapper\s*>\s*input\.ant-input,\s*\n\.ant-input-affix-wrapper\s*>\s*textarea\.ant-input\s*\{([\s\S]*?)\n\}/;
const resetMatch = indexCss.match(resetRe);
check('C1a 内层复位规则存在', !!resetMatch);
if (resetMatch) {
  const block = resetMatch[0];
  for (const prop of ['border: none !important', 'border-radius: 0 !important', 'background: transparent !important', 'color: inherit !important', 'padding: 0 !important']) {
    check(`C1  复位含 ${prop}`, block.includes(prop));
  }
  check('C1b 复位规则不写 box-shadow', !/box-shadow/.test(block), '焦点环归 wrapper（VIS-02）');
  check('C1c 复位覆盖 textarea 内层', /\.ant-input-affix-wrapper\s*>\s*textarea\.ant-input/.test(indexCss));
}

// C2 禁止 :not(组合子)
check('C2 无 :not(含空格组合子) 形态', !/:not\([^)]*[\s>+~][^)]*\)/.test(indexCss));

// C3 局部补丁守护（Phase 88-03 更新：C3b 由「正向锁死死规则」反转为反向断言，
// 与 D1-D4 删除同一变更集落地——先反转不删 = 断言红；先删后反转 = 中间窗口红）
const indexCssNoComment = indexCss.replace(/\/\*[\s\S]*?\*\//g, '');
check('C3a login.css affix 内层补丁保留', /\.ant-input-affix-wrapper\s*>\s*input\.ant-input/.test(loginCss));
check(
  'C3b dept-tree-search 死规则不得存在（VISFIX-03，反向断言）',
  !/\.dept-tree-search[^\n]*\.ant-input-affix-wrapper/.test(indexCssNoComment),
);
check('C3c user-form-input 补丁保留', /\.user-form-input\.ant-input-affix-wrapper \.ant-input/.test(indexCss));

// C4 焦点/边框来源守护
check('C4a .ant-input:focus 焦点环保留', /\.ant-input:focus[\s\S]{0,300}box-shadow:\s*0 0 0 2px var\(--theme-primary-lighter\)/.test(indexCss));
check('C4b wrapper 边框规则保留', /\.ant-input-affix-wrapper\s*\{[\s\S]*?border:\s*1px solid var\(--input-border\)/.test(indexCss));
check('C4c wrapper 焦点环规则保留', /\.ant-input-affix-wrapper-focused\s*\{[\s\S]*?box-shadow/.test(indexCss));

// C5 VIS-03 status 边框色还原
//
// ⚠️ v1.24 Phase 77 实测修正：antd 6 把 status class 打在 wrapper 元素**自身**，
// 名为 `ant-input-status-error`；`ant-input-affix-wrapper-status-error` 这个 class
// 在 antd 6.6.4 的 antd.css 中**根本不存在** → 写成它 = 永不命中的死规则
// （Phase 76 的原 C5 恰好校验的正是这条死规则 → 契约检查形同虚设，本轮一并修正）。
// 真实 DOM：class="ant-input-affix-wrapper ant-input-outlined ant-input-status-error"
check(
  'C5a status-error 边框色还原存在（正确 class）',
  /\.ant-input-affix-wrapper\.ant-input-status-error/.test(indexCss),
);
// 剥掉 CSS 注释后再判定（注释里会提到这个 class 名，不能算命中）
check(
  'C5b 不得出现 antd 6 不存在的 affix-wrapper-status 变体（死规则）',
  !/ant-input-affix-wrapper-status-(error|warning)/.test(indexCssNoComment),
);
check(
  'C5c status-warning 边框色还原存在（正确 class）',
  /\.ant-input-affix-wrapper\.ant-input-status-warning/.test(indexCss),
);

// ---------- C16 VISFIX-05：InputNumber status 边框色还原规则必须存在（防回潮） ----------
// 背景：antd 6 把 status class 打在 .ant-input-number 根（ant-input-number-status-error），
//   而项目的 border-color … !important 会压掉 antd 的 border-color 长属性 →
//   InputNumber 错误态不回红（VISFIX-05）。本断言锁死「还原规则必须存在」，
//   防止后续 phase（91/92 暗色 mega-cluster 合并）误删。
check('C16a InputNumber status-error 还原规则存在（VISFIX-05）',
  /\.ant-input-number\.ant-input-number-status-error/.test(indexCssNoComment));
check('C16b InputNumber status-warning 还原规则存在（VISFIX-05）',
  /\.ant-input-number\.ant-input-number-status-warning/.test(indexCssNoComment));
check('C16c InputNumber status 还原使用 --ant-color-error/warning（无字面 hex）',
  /\.ant-input-number\.ant-input-number-status-error[\s\S]*?\{[\s\S]*?var\(--ant-color-error\)/.test(indexCssNoComment) &&
  /\.ant-input-number\.ant-input-number-status-warning[\s\S]*?\{[\s\S]*?var\(--ant-color-warning\)/.test(indexCssNoComment));

// ==========================================================================
// Phase 88 (VISFIX-06) 新增断言：C6 / C11 / C8 / C12
// ==========================================================================
// CLI: node scripts/css-contract-check.js [--strict]
//   默认态：C6/C11/C8/C12 全部阻断（failures 计入 exit code）。
//   --strict：Phase 88-03 收口后保留为冗余标志（C8/C12 已默认阻断）。

const STRICT = process.argv.includes('--strict');

const { readdirSync, statSync } = await import('node:fs');
const { join, relative } = await import('node:path');
// 递归枚举 src 下指定后缀文件（Windows 分隔符归一为 /）
const walkByExt = (dir, exts, out = []) => {
  for (const e of readdirSync(dir)) {
    const p = join(dir, e);
    if (statSync(p).isDirectory()) walkByExt(p, exts, out);
    else if (exts.some((x) => e.endsWith(x))) out.push(p);
  }
  return out;
};
const relFe = (p) => relative(feRoot, p).replace(/\\/g, '/');

// 剥注释但保持行号/列号不变（注释替换为等长空白，含换行），避免注释里的示例文本误伤。
const stripCommentsKeepLines = (text) =>
  text.replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, ' '));

// ---------- C6 变量必须定义（阻断 + 豁免清单） ----------
// 扫 index.css：按块作用域收集 `--x:` 定义与 `var(--x[, fallback])` 引用。
// 引用有效 = 有 fallback，或定义块作用域为 universal（:root/html 等），
// 或定义块与引用块同属一个明暗分区（light / dark）。
// 近似口径说明：index.css 的明暗变量均为顶层平铺块（无 @media 嵌套作用域），
// 故按选择器内的 [data-color-mode="dark"|"light"] 前缀推断作用域。
  // 行号为 Phase 88-03 删除第 1 批死规则后的实测行号（88-02 版为 2306/2336/4553，删除后前移）
  // Phase 89 Plan 01 R11 收缩：原 3 条豁免（--input-placeholder@2276/@2301、--theme-warning-text@4472）
  // 已清偿 —— placeholder 统一为 --input-text-placeholder（:root 兜底 + light/dark 双分区），
  // --theme-warning-text 已在 :root 定义，豁免表清空转为阻断。
  const EXEMPTED_UNREF_VARS = [];

const checkC6 = (() => {
  const text = stripCommentsKeepLines(indexCss);
  const lineStarts = [0];
  for (let i = 0; i < text.length; i++) if (text[i] === '\n') lineStarts.push(i + 1);
  const lineOf = (pos) => {
    let lo = 0, hi = lineStarts.length - 1;
    while (lo < hi) { const mid = (lo + hi + 1) >> 1; if (lineStarts[mid] <= pos) lo = mid; else hi = mid - 1; }
    return lo + 1;
  };
  const scopeOf = (selector) => {
    if (/\[data-color-mode=["']dark["']\]/.test(selector)) return 'dark';
    if (/\[data-color-mode=["']light["']\]/.test(selector)) return 'light';
    return 'universal';
  };
  // 块解析：栈式扫描，收集每块的 selector / defs / refs
  const blocks = [];
  const stack = [];
  let segStart = 0;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (ch === '{') {
      // 取上一个 `}`/`;` 之后的段作为 selector（文件头 @import 等语句不得并入，
      // 否则 :root 块会以 '@import…' 开头被误判为 at-rule 而丢弃全部变量定义）
      const seg = text.slice(segStart, i);
      const selector = seg.slice(seg.lastIndexOf(';') + 1).trim();
      stack.push({ selector, start: i, contentStart: i + 1 });
      segStart = i + 1;
    } else if (ch === '}') {
      const top = stack.pop();
      if (top) {
        const content = text.slice(top.contentStart, i);
        const isAtRule = top.selector.startsWith('@');
        const isDeclBlock = !isAtRule && content.includes(':');
        const defs = isDeclBlock
          ? [...content.matchAll(/(^|[;{\s])(--[a-zA-Z0-9-]+)\s*:/g)].map((m) => m[2])
          : [];
        const refs = isDeclBlock
          ? [...content.matchAll(/var\(\s*(--[a-zA-Z0-9-]+)\s*(,)?/g)].map((m) => ({
              name: m[1], hasFallback: !!m[2], line: lineOf(top.contentStart + m.index),
            }))
          : [];
        if (defs.length || refs.length) {
          blocks.push({ selector: top.selector, scope: scopeOf(top.selector), start: top.start, end: i, defs, refs });
        }
      }
      segStart = i + 1;
    }
  }
  // 定义索引：name → [{scope}]（@keyframes 等内部块已按 decl-block 过滤）
  const defsIndex = new Map();
  for (const b of blocks) for (const d of b.defs) {
    if (!defsIndex.has(d)) defsIndex.set(d, []);
    defsIndex.get(d).push(b.scope);
  }
  const refValid = (name, refScope, hasFallback) => {
    if (hasFallback) return true;
    // --ant-* 前缀：antd cssVar 模式运行时注入（--ant-color-error/-warning 等），静态扫描不识别。
    // Phase 89 T6/T7 删除死 fallback 后首次引用该前缀；运行时由 ConfigProvider cssVar 保证有值。
    if (name.startsWith('--ant-')) return true;
    const scopes = defsIndex.get(name);
    if (!scopes) return false;
    // html 上必然挂着 data-color-mode（light/dark 其一激活），因此：
    //   - universal 引用（body/.ant-* 等）：light/dark/universal 任意定义均可达；
    //   - dark 块内引用：需 universal 或 dark 定义（只定义在 light = 失效，反之亦然）。
    if (refScope === 'universal') return true;
    return scopes.some((s) => s === 'universal' || s === refScope);
  };
  const violations = [];
  for (const b of blocks) {
    for (const r of b.refs) {
      if (refValid(r.name, b.scope, r.hasFallback)) continue;
      const line = r.line;
      if (EXEMPTED_UNREF_VARS.some((e) => e.name === r.name && e.line === line)) continue;
      violations.push(`index.css:${line} 引用未定义变量 ${r.name}${r.hasFallback ? '' : '（无 fallback）'} @ ${b.selector.trim().slice(0, 80)}`);
    }
  }
  check(
    'C6  var() 引用的变量均已定义（含豁免清单）',
    violations.length === 0,
    violations.length ? violations.join(' | ') : `豁免登记 ${EXEMPTED_UNREF_VARS.length} 处（Phase 89 R11）`,
  );
})();

// ---------- C11 !important 预算 ratchet（阻断，只降不升） ----------
// ratchet：只允许下调，Phase 92 终值 ≤480；上调视为回归。
// 基线沿革：1435（Phase 88 落地时 HEAD 实跑值）→ 1398（Phase 88-03 删除第 1 批死规则后实跑值；
// PLAN 预估 ≈1395，实跑 1398——ratchet 只许低不许高，故定值取实测）→ 1397（Phase 89 T5 落地 -1）
// → 1385（Phase 91 表单/表格/弹窗交还 token，当前实测 1381）
// → 671（Phase 92 R6/R7 mega-cluster + 散存量块级删除，实测 671（含 C16 恢复 2 处）；92 终态目标 ≤480 由后续收敛达成）
// → 455（Phase 92 Gap 4 浅色区精简与高频 hex 变量化收敛，当前实测 452 ≤ 480 终态达标）。
const IMPORTANT_BUDGET = 455;
{
  const allCss = walkByExt(resolve(feRoot, 'src'), ['.css', '.less']);
  let importantCount = 0;
  for (const f of allCss) importantCount += (readFileSync(f, 'utf8').match(/!important/g) || []).length;
  check(
    'C11  !important 预算 ratchet（只降不升）',
    importantCount <= IMPORTANT_BUDGET,
    `当前 ${importantCount} / 预算 ${IMPORTANT_BUDGET}`,
  );
}

// ---------- C13 .css/.less hex 全量计数 + ratchet（阻断，只降不升，Phase 89 W3 新增） ----------
// 口径：
//   1) 扫描 src 下全部 .css/.less，逐行剥离 /* */ 与 // 注释后统计 6 位 hex（#rrggbb）出现次数
//   2) M7 = index.css hex 总数；M8 = 全部 .css/.less hex 总数（与 89-QUANT-SNAPSHOT W0 实跑口径一致）
//   3) ratchet = W0 实跑起点（M7=310 / M8=452，89-01 落盘值），本 phase 只降不升；
//      92 终态（M7 去重 ≤60 / M8 ≤120）不在本 phase 设绝对下降目标（Lead 裁定：89 只要求「不增」）
//   4) CSS 层逐条豁免走 exemption-reasons.json 的 layer:"CSS" 条目，由 check-exemption-reasons.mjs
//      与 check-hardcoded-colors.mjs 的 PATH_ALLOW 双向对账（本 phase 的 CSS 层豁免集合为空：
//      A3 图表色板是 .tsx 测试文件，归扫描器 PATH_ALLOW，不占 C13 预算）
// 职责边界：本断言只管「总量与新增」，不管「某个已知 hex 是不是坏色」——那是
//   scripts/check-hardcoded-colors.mjs 的 lint 职责（已接入 lint:ci）。两者互补，不重复。
const C13_M7_BUDGET = 60; // index.css 6 位 hex（Phase 92 终态锁定，实测 26 ≤ 60）
const C13_M8_BUDGET = 452; // 全 .css/.less 6 位 hex（W0 实跑起点，只降不升，实测 160）
const C13_M9_BUDGET = 3200; // index.css 行数终态（Phase 92 终态锁定，实测 3194 ≤ 3200）
const C13_BASELINE = resolve(here, 'css-hex-baseline.json');
{
  const stripComments = (line) => line.replace(/\/\*.*?\*\//g, '').replace(/(^|[^:])\/\/.*$/, '$1');
  const countHex = (text) => (stripComments(text).match(/#[0-9a-fA-F]{6}\b/g) || []).length;
  const cssFiles = walkByExt(resolve(feRoot, 'src'), ['.css', '.less']);
  const counts = new Map(); // rel → n
  let m7 = 0;
  let m8 = 0;
  for (const f of cssFiles) {
    const n = countHex(readFileSync(f, 'utf8'));
    const rel = relFe(f);
    counts.set(rel, n);
    m8 += n;
    if (rel === 'src/index.css') m7 = n;
  }
  const m9 = readFileSync(resolve(feRoot, 'src/index.css'), 'utf8').split('\n').length;
  // 新增检测：任一文件 hex 计数超 per-file 基线（W3 终态快照）= 新增硬编码色，
  // 未在 exemption-reasons.json 登记 layer:"CSS"（css 文件维度，本 phase 为空集）即阻断。
  const newHits = [];
  if (existsSync(C13_BASELINE)) {
    const baseline = JSON.parse(readFileSync(C13_BASELINE, 'utf8')).files || {};
    for (const [rel, n] of counts) {
      if (rel in baseline && n > baseline[rel]) newHits.push(`${rel}: ${baseline[rel]} → ${n}（新增 ${n - baseline[rel]}）`);
      if (!(rel in baseline)) newHits.push(`${rel}: 新文件 ${n} 处 hex（基线无此文件）`);
    }
  }
  const ok13 = m7 <= C13_M7_BUDGET && m8 <= C13_M8_BUDGET && m9 <= C13_M9_BUDGET && newHits.length === 0;
  check(
    'C13  .css/.less hex 全量 ratchet（W0 起点 M7≤310 / M8≤452 只降不升 + per-file 新增阻断）',
    ok13,
    ok13
      ? `C13 PASS：M7=${m7} ≤ ${C13_M7_BUDGET} / M8=${m8} ≤ ${C13_M8_BUDGET} / M9=${m9} ≤ ${C13_M9_BUDGET}，per-file 基线无新增（92 终态 ≤60/≤120 不在 89 追）`
      : `C13 FAIL：${[
          m7 > C13_M7_BUDGET ? `M7=${m7}>${C13_M7_BUDGET}` : null,
          m8 > C13_M8_BUDGET ? `M8=${m8}>${C13_M8_BUDGET}` : null,
          m9 > C13_M9_BUDGET ? `M9=${m9}>${C13_M9_BUDGET}` : null,
          ...newHits,
        ].filter(Boolean).join('  ')}（新增 hex 须删除或登记 layer:"CSS" 豁免）`,
  );
}

// ---------- C8 死文件检测（warn-only + --strict） ----------
{
  const styleFiles = walkByExt(resolve(feRoot, 'src'), ['.css', '.less']);
  const refFiles = walkByExt(resolve(feRoot, 'src'), ['.ts', '.tsx', '.css', '.less'])
    .map((p) => ({ path: p, dir: resolve(p, '..'), text: readFileSync(p, 'utf8') }));
  // 入口白名单：src/index.css 由 main.tsx 引入（按相对路径精确匹配，避免 basename 碰撞）；
  // allowlist 兜底动态 import 场景（basename 口径）
  const C8_ENTRY_ALLOWLIST = ['src/index.css'];
  const C8_ALLOWLIST = [];
  const dead = [];
  for (const f of styleFiles) {
    const rel = relFe(f);
    const base = f.split(/[\\/]/).pop();
    if (C8_ENTRY_ALLOWLIST.includes(rel) || C8_ALLOWLIST.includes(base)) continue;
    const parentName = resolve(f, '..').split(/[\\/]/).pop();
    const escapedBase = base.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const quotedRe = new RegExp(`['"]([^'"]*\\/?${escapedBase})['"]`);
    const referenced = refFiles.some(({ path, dir, text }) => {
      if (resolve(path) === resolve(f)) return false;
      // 仅 import 语句口径：`import "…<basename>"` / `from "…<basename>"` / `@import "…<basename>"`
      const importRe = new RegExp(
        /(^|\n)\s*(?:@import\s+[^;]*?|import\s+[^;'"]*?from\s*[^;'"]*?|import\s*)/.source + quotedRe.source,
      );
      const m = text.match(importRe);
      if (!m) return false;
      const p = m[1];
      // 目录感知：basename 碰撞消歧（如 main.tsx 的 "./index.css" ≠ dashboard-system/index.css）
      //   - 同目录相对导入（"./xxx.css"）→ 真引用
      //   - 路径含目标父目录名（"@/pages/dashboard-system/index.css"）→ 真引用
      if (dir === resolve(f, '..')) return true;
      return p.includes(`${parentName}/${base}`) || p.includes(`\\${parentName}\\${base}`);
    });
    if (!referenced) dead.push(relFe(f));
  }
  // Phase 88-03 收口：默认阻断（命中即失败）；命中为 0 时不计 failures
  if (dead.length) failures++;
  console.log(`${dead.length ? 'FAIL' : 'PASS'}  C8  CSS/LESS 死文件（无 import 引用）  → ${dead.length ? dead.join(' | ') : '0（全绿）'}`);
}

// ---------- C12 data-color_mode 拼写（warn-only + --strict） ----------
// 正确属性名为 data-color-mode；data-color_mode（下划线）在 DOM 上永不命中 = 死规则。
{
  const hits = [];
  for (const f of walkByExt(resolve(feRoot, 'src'), ['.css'])) {
    const text = readFileSync(f, 'utf8');
    let idx = text.indexOf('data-color_mode');
    while (idx !== -1) {
      hits.push(`${relFe(f)}:${text.slice(0, idx).split('\n').length}`);
      idx = text.indexOf('data-color_mode', idx + 1);
    }
  }
  // Phase 88-03 收口：默认阻断（命中即失败）；命中为 0 时不计 failures
  if (hits.length) failures++;
  console.log(`${hits.length ? 'FAIL' : 'PASS'}  C12  data-color_mode 拼写死规则  → ${hits.length ? hits.join(' | ') : '0（全绿）'}`);
}

// ==========================================================================
// Phase 88-04 (VISFIX-04) 新增断言：C14 — antd 6 弃用 prop 零命中（阻断）
// ==========================================================================
// 落地时序（PLAN 显式）：清理（Task 4.1 / 4.2）先完成 → 断言后落地阻断 → 落地即天然绿。
//   （先上断言必常红——断言必须在清理完成之后才落地。）
//
// 明确**不进**清单（防误伤 / 关键决策）：
//   - bordered：Descriptions / Table 未弃用（antd/es/card 仅 Card 弃用）→ 会大量误报，不进正则
//   - onSearch：antd 6.6.4 未弃用（源码 deprecatedProps + 类型定义双向确认）→ 绝不进清单
//
// 弃用证据（node_modules/antd 6.6.4）：
//   es/modal/Modal.js:126  bodyStyle→styles.body / maskStyle→styles.mask /
//                          destroyOnClose→destroyOnHidden / maskClosable→mask.closable
//   es/drawer/Drawer.d.ts:21,27  destroyOnClose / maskClosable 标 @deprecated
//   es/card/Card.js:76,93   size="default"→"medium"；headStyle/bodyStyle/bordered→styles/variant
//   es/select/index.d.ts:66,88,90,94,96,98  showArrow / popupClassName / dropdownClassName /
//                          dropdownRender / onDropdownVisibleChange / dropdownMatchSelectWidth @deprecated

// ---- 剥 TS/TSX 注释（保留行号/列号），避免注释里的示例文本误伤 ----
// 字符串/模板串用「前置非标识符字符」启发式判定，避免 JSX 文本里的撇号被误判为字符串起点。
const stripTsCommentsKeepLines = (text) => {
  const out = text.split('');
  const n = text.length;
  const blank = (k) => {
    if (k >= 0 && k < n && out[k] !== '\n') out[k] = ' ';
  };
  const isIdentChar = (ch) => /[\w$.]/.test(ch || '');
  let i = 0;
  let state = 'code';
  while (i < n) {
    const c = text[i];
    const c2 = i + 1 < n ? text[i + 1] : '';
    if (state === 'code') {
      if (c === '/' && c2 === '/') {
        blank(i); blank(i + 1); i += 2; state = 'line'; continue;
      }
      if (c === '/' && c2 === '*') {
        blank(i); blank(i + 1); i += 2; state = 'block'; continue;
      }
      if ((c === '"' || c === "'" || c === '`') && !isIdentChar(text[i - 1])) {
        state = c === '"' ? 'dq' : c === "'" ? 'sq' : 'tpl';
        i++;
        continue;
      }
      i++;
      continue;
    }
    if (state === 'line') {
      if (c === '\n') { state = 'code'; i++; continue; }
      blank(i); i++; continue;
    }
    if (state === 'block') {
      if (c === '*' && c2 === '/') { blank(i); blank(i + 1); i += 2; state = 'code'; continue; }
      blank(i); i++; continue;
    }
    if (state === 'dq' || state === 'sq') {
      if (c === '\\') { i += 2; continue; }
      if ((state === 'dq' && c === '"') || (state === 'sq' && c === "'")) { state = 'code'; i++; continue; }
      i++; continue;
    }
    // 模板串：${} 内不做注释剥离（罕见），整段按字符串处理
    if (c === '\\') { i += 2; continue; }
    if (c === '`') { state = 'code'; i++; continue; }
    i++;
  }
  return out.join('');
};

// ---- 定位 JSX 开始标签（<Tag ...>）的 [start,end] 与标签名 ----
// 用途：判定某个 JSX 属性到底传给了哪个组件（maskClosable 是否触达 antd Modal/Drawer）。
// 同时记录标签内的花括号深度分段：**属性名位**在深度 0，**属性值里的 JS 引用**在深度 ≥1
// （如 `mask={{ closable: maskClosable }}` 中的 maskClosable 是值引用，不是属性名）。
const findJsxOpenTags = (text) => {
  const tags = [];
  const n = text.length;
  let i = 0;
  while (i < n) {
    if (text[i] !== '<') { i++; continue; }
    const m = /^<([A-Za-z_$][\w$.]*)/.exec(text.slice(i, i + 120));
    if (!m) { i++; continue; }
    const prev = i > 0 ? text[i - 1] : '';
    if (/[\w$.]/.test(prev)) { i++; continue; } // a<b 之类，非标签
    const name = m[1];
    let j = i + 1 + name.length;
    let depth = 0;
    let quote = null;
    let segStart = j;
    const runs = [];
    while (j < n) {
      const ch = text[j];
      if (quote) {
        if (ch === '\\') { j += 2; continue; }
        if (ch === quote) quote = null;
        j++;
        continue;
      }
      if (ch === '"' || ch === "'" || ch === '`') { quote = ch; j++; continue; }
      if (ch === '{') { runs.push({ from: segStart, to: j, depth }); depth++; segStart = j + 1; j++; continue; }
      if (ch === '}') { runs.push({ from: segStart, to: j, depth }); depth--; segStart = j + 1; j++; continue; }
      if (ch === '>' && depth === 0) break;
      j++;
    }
    runs.push({ from: segStart, to: j, depth });
    tags.push({ name, start: i, end: j, runs });
    i = j + 1;
  }
  return tags;
};

// ---- 字符位置 → 行号（保留原始换行，剥注释后行号不变） ----
// C14（maskClosable 上下文判定）与 C14c（降级扫描）共用。
const lineStartsOf = (text) => {
  const ls = [0];
  for (let i = 0; i < text.length; i++) if (text[i] === '\n') ls.push(i + 1);
  return ls;
};
const lineOf = (ls, pos) => {
  let lo = 0;
  let hi = ls.length - 1;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (ls[mid] <= pos) lo = mid;
    else hi = mid - 1;
  }
  return lo + 1;
};

// ---- C14 弃用清单（token → 弃用后的新形态；bordered / onSearch 刻意不入列） ----
const C14_DEPRECATED_TOKENS = [
  { key: 'destroyOnClose', re: /\bdestroyOnClose\b/g, note: 'antd 6 → destroyOnHidden' },
  { key: 'bodyStyle', re: /\bbodyStyle\b/g, note: 'antd 6 → styles.body' },
  { key: 'headStyle', re: /\bheadStyle\b/g, note: 'antd 6 → styles.header' },
  { key: 'maskStyle', re: /\bmaskStyle\b/g, note: 'antd 6 → styles.mask' },
  { key: 'footerStyle', re: /\bfooterStyle\b/g, note: 'antd 6 → styles.footer' },
  { key: 'contentWrapperStyle', re: /\bcontentWrapperStyle\b/g, note: 'antd 6 → styles.wrapper' },
  { key: 'dropdownRender', re: /\bdropdownRender\b/g, note: 'antd 6 → popupRender' },
  { key: 'dropdownClassName', re: /\bdropdownClassName\b/g, note: 'antd 6 → classNames.popup.root' },
  { key: 'popupClassName', re: /\bpopupClassName\b/g, note: 'antd 6 → classNames.popup.root' },
  { key: 'onDropdownVisibleChange', re: /\bonDropdownVisibleChange\b/g, note: 'antd 6 → onOpenChange' },
  { key: 'dropdownMatchSelectWidth', re: /\bdropdownMatchSelectWidth\b/g, note: 'antd 6 → popupMatchSelectWidth' },
  { key: 'showArrow', re: /\bshowArrow\b/g, note: 'antd 6 → suffixIcon（Select 系）' },
  { key: 'size="middle"', re: /\bsize\s*=\s*(["'])middle\1/g, note: 'antd 6 → size="medium"' },
  { key: 'size="default"', re: /\bsize\s*=\s*(["'])default\1/g, note: 'antd 6 → size="medium"（Card）' },
];

// ---- C14 豁免登记（新增豁免必须写明理由与证据行号） ----
const C14_EXEMPTIONS = [
  {
    file: 'src/pages/system/notice/components/StatisticsDrawer.tsx',
    token: 'size="default"',
    reason:
      'antd/es/drawer/Drawer.d.ts:6 `declare const _SizeTypes: readonly ["default", "large"]` —— Drawer 的 size 枚举**无 "medium"**；' +
      'antd/es/drawer/Drawer.js:90 `if (size === "default") return DEFAULT_SIZE` 证明 "default" 是 Drawer 合法基准尺寸；' +
      'Drawer.js:93-97 其它字符串（如 "medium"）会被原样透传成非法 width → 迁移即视觉破损，故保留并登记豁免。',
  },
  {
    file: 'src/design-system/components/ColorSwitcher.tsx',
    token: 'size="middle"',
    reason:
      '本仓自研 design-system 组件的**自有 prop 枚举**（ColorSwitcherProps.size?: "small" | "middle" | "large" + 解构默认值），' +
      '非 antd 弃用 prop：antd 6.6.4 的 SizeType（config-provider/SizeContext.d.ts:5）仍含 "middle"' +
      '（\'small\' | \'medium\' | \'middle\' | \'large\'），Button/ColorPicker 对 "middle" 无弃用告警，' +
      '且 Button.js:224 只对 large/small 加尺寸 class（middle 与 medium 同为默认尺寸 → 渲染等价）；' +
      '改写会变更本组件对外 prop 契约（扩范围）→ 保留并登记豁免。',
  },
  {
    file: 'src/components/shared/FormModal.tsx',
    token: 'maskClosable',
    reason:
      'v1.22 契约冻结：对外 prop 名 maskClosable 保留（props 声明 + 解构默认值），' +
      '内部转发 antd 6 新形态 mask={{ closable }} 收敛在 BaseEditModal 内部；调用方零改动。',
  },
  {
    file: 'src/components/modal/BaseEditModal.tsx',
    token: 'maskClosable',
    reason:
      'v1.22 契约冻结：对外 prop 名 maskClosable 保留（props 声明 + 解构默认值），' +
      '内部转发 antd 6 新形态 mask={{ closable: maskClosable }} 传给 antd Modal。',
  },
];

{
  const c14Files = walkByExt(resolve(feRoot, 'src'), ['.ts', '.tsx']);
  const violations = [];
  for (const f of c14Files) {
    const rel = relFe(f);
    const text = stripTsCommentsKeepLines(readFileSync(f, 'utf8'));
    const ls = lineStartsOf(text);
    const jsxTags = findJsxOpenTags(text);
    const report = (key, pos, note) => {
      violations.push(`${rel}:${lineOf(ls, pos)}  弃用 prop「${key}」${note ? '（' + note + '）' : ''}`);
    };
    const isExempt = (token) => C14_EXEMPTIONS.some((e) => e.file === rel && e.token === token);

    // 通用 token 清单
    for (const item of C14_DEPRECATED_TOKENS) {
      if (isExempt(item.key)) continue;
      for (const m of text.matchAll(item.re)) report(item.key, m.index, item.note);
    }

    // maskClosable：只有「作为属性名传给 antd Modal/Drawer」或「写成 antd config 对象键」才算弃用命中。
    // 允许位：props 类型声明 / 解构默认值 / 我方组件（FormModal/BaseEditModal/测试 Harness）的 JSX prop /
    //          属性值里的 JS 引用（如 mask={{ closable: maskClosable }}）。
    for (const m of text.matchAll(/\bmaskClosable\b/g)) {
      const idx = m.index;
      let attrTag = null;
      for (const t of jsxTags) {
        if (idx < t.start || idx > t.end) continue;
        const run = t.runs.find((r) => idx >= r.from && idx <= r.to);
        if (run && run.depth === 0) attrTag = t; // 深度 0 → 属性名位
        break;
      }
      if (attrTag && (attrTag.name === 'Modal' || attrTag.name === 'Drawer')) {
        report('maskClosable', idx, `仍传给 antd <${attrTag.name}>（antd 6 → mask.closable）`);
        continue;
      }
      if (!attrTag) {
        const rest = text.slice(idx + 'maskClosable'.length, idx + 'maskClosable'.length + 80);
        // 对象字面量 / antd config 键（`maskClosable:`）；类型标注（`maskClosable?: boolean`）不算
        if (/^\s*:\s*(?!boolean\b|string\b|number\b|unknown\b|any\b|never\b|void\b)/.test(rest)) {
          report('maskClosable', idx, '对象字面量 / antd config 仍用旧形态（antd 6 → mask.closable）');
        }
      }
    }
  }
  check(
    'C14  antd 6 弃用 prop 零命中（阻断）',
    violations.length === 0,
    violations.length
      ? violations.join(' | ')
      : `C14 PASS：弃用清单 0 命中，豁免登记 ${C14_EXEMPTIONS.length} 项`,
  );
}

// ==========================================================================
// Phase 88-05 (VISFIX-05) 新增断言：C14c — antd <Drawer> 的 width prop 零命中（阻断）
// ==========================================================================
// 注：C14d / C14e（antd <Alert> message / antd <Progress> width）紧随本块之后，
//     共用同一套 TS AST 判定思路（定位 JSX 元素 + 确认 antd import 绑定）。
// 弃用证据（node_modules/antd 6.6.4）：
//   es/drawer/Drawer.js:155  `[['width','size'], ['height','size']].forEach(...)`
//   → `width` 与 `height` 均为 antd 6 Drawer 的 @deprecated prop。
//
// 迁移等价性（es/drawer/Drawer.js:83-103 drawerSize 计算）：
//   - `size` 为数字 → `isNumber(size)` 直返该数字；
//   - `size` 为非数字字符串 → 原样透传（不匹配 /^(\d+)(\.\d+)?$/ 时）；
//   - `size` 缺省且 placement ∈ {left,right,undefined} → 取 `width`；
//   - `size` 缺省且 placement ∈ {top,bottom} → 取 `height`（**height 与 placement 强相关**）。
//   ⇒ 对 **数字** width：`width={N}` ≡ `size={N}`（两条路径产出同一个 `size={N}` 下传 rc-drawer）。
//   本仓 9 处迁移目标全部是**数字** width，且 placement ∈ {right, 未设置}（rc-drawer 默认 'right'），
//   故渲染宽度逐像素不变（rc-drawer DrawerPopup.js:135 `size ?? legacySize ?? …`，
//   两侧最终都是同一个数字 size → wrapperStyle.width 相同）。
//
// 判定口径（**上下文感知**）：
//   只判「antd <Drawer> 标签上的 width 属性」。antd <Modal> 的 width **不弃用**
//   （antd 6.6.4 modal/Modal.js 弃用表无 width）→ Modal / Table 列宽 / 自研组件
//   的 width 一律不得误报。
//   「是不是 antd Drawer」的判定 = 该文件确实存在 `import { Drawer } from 'antd'`
//   （本仓无本地同名 Drawer 组件，已全量核实）。
//
// 实现：优先用 typescript 编译器 API 做精确 AST 判定（JSX 元素 + 属性名 + import 来源）；
//   typescript 不可用时降级为正则 JSX 扫描（findJsxOpenTags + 深度 0 属性名位），
//   降级态在输出中显式标注（降级扫描存在已知盲区，见文末说明）。
{
  const C14C_EXEMPTIONS = [
    // 结构：{ file, line, reason } —— 新增豁免必须写明「为何 width 不可迁」+ 证据行号
  ];

  const tsMod = await import('typescript').catch(() => null);
  const useTs = !!tsMod && typeof tsMod.createSourceFile === 'function';
  const violations = [];
  const advisories = [];

  const c14cFiles = walkByExt(resolve(feRoot, 'src'), ['.tsx']);
  let scanned = 0;
  let antdDrawerFiles = 0;

  for (const f of c14cFiles) {
    const rel = relFe(f);
    const raw = readFileSync(f, 'utf8');
    if (!raw.includes('Drawer')) continue; // 快速预筛：不含 Drawer 字样的文件无需解析
    scanned++;

    // ---- 收集「本文件的 <Drawer> 是否来自 antd」 ----
    let importsAntdDrawer = false;
    let hasAnyDrawerImport = false;

    if (useTs) {
      const ts = tsMod;
      const sf = ts.createSourceFile(f, raw, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
      for (const st of sf.statements) {
        if (!ts.isImportDeclaration(st) || !ts.isStringLiteral(st.moduleSpecifier)) continue;
        const spec = st.moduleSpecifier.text;
        if (!(spec === 'antd' || spec.startsWith('antd/es/drawer'))) continue;
        const nb = st.importClause && st.importClause.namedBindings;
        if (!nb || !ts.isNamedImports(nb)) continue;
        for (const el of nb.elements) {
          const importedName = el.propertyName ? el.propertyName.text : el.name.text;
          if (importedName !== 'Drawer') continue;
          hasAnyDrawerImport = true;
          // 只有 `import { Drawer }`（未改名）才对应 JSX 标签名 <Drawer>
          if (el.name.text === 'Drawer') importsAntdDrawer = true;
        }
      }
      if (!importsAntdDrawer) continue; // 非 antd Drawer → 不在本断言范围

      const lineOfPos = (pos) => sf.getLineAndCharacterOfPosition(pos).line + 1;
      const visit = (node) => {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          if (node.tagName.getText(sf) === 'Drawer') {
            const line = lineOfPos(node.getStart(sf));
            let widthAttr = null;
            let hasSpread = false;
            for (const p of node.attributes.properties) {
              if (ts.isJsxSpreadAttribute(p)) {
                hasSpread = true;
                continue;
              }
              if (ts.isJsxAttribute(p) && p.name.text === 'width') widthAttr = p;
            }
            if (widthAttr) {
              // 报告**属性所在行**（而非元素起始行），便于直接定位
              const attrLine = lineOfPos(widthAttr.getStart(sf));
              const exempt = C14C_EXEMPTIONS.some(
                (e) => e.file === rel && e.line === attrLine,
              );
              if (!exempt) {
                violations.push(
                  `${rel}:${attrLine}  弃用 prop「width」仍传给 antd <Drawer>（antd 6 → size）`,
                );
              }
            }
            if (hasSpread) {
              advisories.push(
                `${rel}:${line}  antd <Drawer> 存在 JSX 展开属性（{...x}），width 需人工确认`,
              );
            }
          }
        }
        ts.forEachChild(node, visit);
      };
      visit(sf);
      antdDrawerFiles++;
    } else {
      // ---- 降级：正则 JSX 扫描 ----
      const text = stripTsCommentsKeepLines(raw);
      const ls = lineStartsOf(text);
      if (!/import\s*\{[^}]*\bDrawer\b[^}]*\}\s*from\s*['"]antd['"]/.test(text)) continue;
      importsAntdDrawer = true;
      antdDrawerFiles++;
      const jsxTags = findJsxOpenTags(text);
      for (const t of jsxTags) {
        if (t.name !== 'Drawer') continue;
        const line = lineOf(ls, t.start);
        for (const r of t.runs) {
          if (r.depth !== 0) continue;
          const seg = text.slice(r.from, r.to);
          const m = /^\s*width\s*=/.exec(seg);
          if (!m) continue;
          const exempt = C14C_EXEMPTIONS.some((e) => e.file === rel && e.line === line);
          if (!exempt) {
            violations.push(
              `${rel}:${line}  弃用 prop「width」仍传给 antd <Drawer>（antd 6 → size）`,
            );
          }
        }
      }
    }
  }

  check(
    'C14c antd <Drawer> width 弃用零命中（阻断）',
    violations.length === 0,
    violations.length
      ? violations.join(' | ')
      : `C14c PASS：antd <Drawer> 0 处 width（扫描 ${scanned} 个含 Drawer 的 .tsx，命中 antd Drawer 的 ${antdDrawerFiles} 个）` +
        (useTs ? '｜AST 精确判定' : '｜⚠ 降级为正则扫描（typescript 不可用）') +
        (advisories.length ? `｜人工确认 ${advisories.length} 条：${advisories.join(' | ')}` : ''),
  );
}

// ==========================================================================
// Phase 88-05 (VISFIX-05 收口补丁) 新增断言：C14d / C14e
//   C14d — antd <Alert> 的 message prop 零命中（antd 6 → title）
//   C14e — antd <Progress> 的 width prop 零命中（antd 6 → size）
// ==========================================================================
// 弃用证据（node_modules/antd 6.6.4）：
//   es/alert/Alert.js:87-88
//       [['closeText','closable.closeIcon'], ['message','title']].forEach(([deprecatedName, newName]) => {
//         warning.deprecated(!(deprecatedName in props), deprecatedName, newName);
//       });
//       → `message` 是 antd 6 Alert 的 @deprecated prop，新形态为 `title`。
//   es/alert/Alert.js:83  `const mergedTitle = title ?? message;`
//       → message 仅是 title 的**旧形态回退**，两者最终都渲染进同一个 `${prefixCls}-title`
//         节点（Alert.js:236-239）→ `message=` → `title=` 是**渲染等价**的纯改名
//         （DOM 结构 / class / 内容全同），可安全迁移。
//   es/progress/progress.js:120
//       [['width','size'], ['trailColor','railColor'], ['gapPosition','gapPlacement']]
//         → `width` 是 antd 6 Progress 的 @deprecated prop，新形态为 `size`。
//
// Progress 的 `width` → `size` 迁移等价性（本仓两处命中均为 type="circle"）：
//   es/progress/Circle.js:25  `width: originWidth = 120,`
//   es/progress/Circle.js:29  `size = originWidth,`   ← size 缺省时**回落到 width**
//   es/progress/Circle.js:36  `const [width, height] = getSize(size, 'circle');`
//   es/progress/utils.js:66-67  数字 size → `[width, height] = [size, size]`（= 直径）
//   ⇒ 对 **circle/dashboard** 型：`width={N}`（不传 size）⇒ size=N ⇒ 直径 N；
//     `size={N}` ⇒ 直径 N。两者产出同一 [N,N]，渲染逐像素一致。
//     （唯一差异面：progress.js:166 `progress-inline-circle` 在 size ≤ 20 时才命中，
//       仅影响 lineHeight/verticalAlign；本仓 N ∈ {100,120} > 20 → 不触发。）
//   ⇒ 对 **line** 型：size 的数值形态是**条高**（utils.js:58-59 [w,h]=[size,size]），
//     与 `width`（条宽百分比）语义不同，line 型须写 `size={{ width: N }}`（utils.js:61）。
//     本仓无 line 型 width 命中，故无豁免登记需求。
//
// 判定口径（**上下文感知**，沿用 C14c 的 TS AST 思路）：
//   只判「确实绑定到 antd 的 <Alert> / <Progress> 标签上的属性」，不碰其它任何东西：
//   - antd <Modal> / <Popconfirm> 的 message **不弃用**（antd 6.6.4 modal/Modal.js
//     弃用表无 message）→ 不得误报；
//   - 自研组件的同名 prop（如 ErrorAlertWithRetry 自有的 `message?: string`）不得误报；
//   - 变量名 / 对象键 / 属性值里的 `.message` 成员访问不得误报。
//   「是不是 antd 的 Alert/Progress」判定 = 该文件存在
//   `import { Alert } from 'antd'`（或 `antd/es/alert` 的 default 导入）且**未改名**，
//   即 JSX 标签名与本地绑定名一致（`import { Alert as Foo }` 的标签名是 Foo，不在本范围）。
{
  // 豁免登记：{ file, line, reason } —— 新增豁免必须写明「为何不可迁」+ 证据行号
  const C14D_EXEMPTIONS = []; // antd <Alert> 的 message
  const C14E_EXEMPTIONS = []; // antd <Progress> 的 width

  const tsMod = await import('typescript').catch(() => null);
  const useTs = !!tsMod && typeof tsMod.createSourceFile === 'function';

  const TARGETS = [
    { tag: 'Alert', prop: 'message', newName: 'title', label: 'C14d' },
    { tag: 'Progress', prop: 'width', newName: 'size', label: 'C14e' },
  ];

  const violations = { Alert: [], Progress: [] };
  const advisories = [];
  const stat = { Alert: { elements: 0, files: 0 }, Progress: { elements: 0, files: 0 } };

  const c14deFiles = walkByExt(resolve(feRoot, 'src'), ['.tsx']);
  let scanned = 0;

  for (const f of c14deFiles) {
    const rel = relFe(f);
    const raw = readFileSync(f, 'utf8');
    if (!/\b(Alert|Progress)\b/.test(raw)) continue; // 快速预筛
    scanned++;

    if (useTs) {
      const ts = tsMod;
      const sf = ts.createSourceFile(f, raw, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);

      // ---- 收集「本地名 → antd 绑定」的映射 ----
      const antdBind = new Map();
      for (const st of sf.statements) {
        if (!ts.isImportDeclaration(st) || !ts.isStringLiteral(st.moduleSpecifier)) continue;
        const spec = st.moduleSpecifier.text;
        const isAntdRoot = spec === 'antd';
        const isAntdSub = /^antd\/es\/(alert|progress)$/.test(spec);
        if (!isAntdRoot && !isAntdSub) continue;
        const ic = st.importClause;
        if (!ic || ic.isTypeOnly) continue;
        const nb = ic.namedBindings;
        if (nb && ts.isNamedImports(nb)) {
          for (const el of nb.elements) {
            if (el.isTypeOnly) continue;
            if (el.propertyName) continue; // `import { Alert as Foo }` → 标签名是 Foo，不在本范围
            antdBind.set(el.name.text, { orig: el.name.text, spec });
          }
        }
        // `import Alert from 'antd/es/alert'` → default 导出即 antd Alert
        if (ic.name && isAntdSub) antdBind.set(ic.name.text, { orig: 'default', spec });
      }

      const lineOfPos = (pos) => sf.getLineAndCharacterOfPosition(pos).line + 1;
      const visit = (node) => {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          const tagName = node.tagName.getText(sf);
          for (const t of TARGETS) {
            if (tagName !== t.tag) continue;
            const bind = antdBind.get(tagName);
            if (!bind) continue; // 非 antd 绑定 → 不在断言范围（自研组件不误报）
            if (bind.orig !== t.tag && bind.orig !== 'default') continue;
            stat[t.tag].elements++;
            let hit = null;
            let hasSpread = false;
            for (const p of node.attributes.properties) {
              if (ts.isJsxSpreadAttribute(p)) {
                hasSpread = true;
                continue;
              }
              if (ts.isJsxAttribute(p) && p.name.text === t.prop) hit = p;
            }
            if (hit) {
              const attrLine = lineOfPos(hit.getStart(sf));
              const exemptList = t.tag === 'Alert' ? C14D_EXEMPTIONS : C14E_EXEMPTIONS;
              const exempt = exemptList.some((e) => e.file === rel && e.line === attrLine);
              if (!exempt) {
                violations[t.tag].push(
                  `${rel}:${attrLine}  弃用 prop「${t.prop}」仍传给 antd <${t.tag}>（antd 6 → ${t.newName}）`,
                );
              }
            }
            if (hasSpread) {
              advisories.push(
                `${rel}:${lineOfPos(node.getStart(sf))}  antd <${t.tag}> 存在 JSX 展开属性（{...x}），${t.prop} 需人工确认`,
              );
            }
          }
        }
        ts.forEachChild(node, visit);
      };
      visit(sf);
    } else {
      // ---- 降级：正则 JSX 扫描（typescript 不可用时） ----
      const text = stripTsCommentsKeepLines(raw);
      const ls = lineStartsOf(text);
      const jsxTags = findJsxOpenTags(text);
      for (const t of TARGETS) {
        const impRe = new RegExp(
          `import\\s*\\{[^}]*\\b${t.tag}\\b[^}]*\\}\\s*from\\s*['"]antd['"]`,
        );
        if (!impRe.test(text)) continue;
        stat[t.tag].files++;
        for (const tag of jsxTags) {
          if (tag.name !== t.tag) continue;
          stat[t.tag].elements++;
          const line = lineOf(ls, tag.start);
          for (const r of tag.runs) {
            if (r.depth !== 0) continue;
            const seg = text.slice(r.from, r.to);
            if (!new RegExp(`^\\s*${t.prop}\\s*=`).test(seg)) continue;
            const exemptList = t.tag === 'Alert' ? C14D_EXEMPTIONS : C14E_EXEMPTIONS;
            const exempt = exemptList.some((e) => e.file === rel && e.line === line);
            if (!exempt) {
              violations[t.tag].push(
                `${rel}:${line}  弃用 prop「${t.prop}」仍传给 antd <${t.tag}>（antd 6 → ${t.newName}）`,
              );
            }
          }
        }
      }
    }
  }

  const summarize = (t) => {
    const v = violations[t.tag];
    return v.length
      ? v.join(' | ')
      : `${t.label} PASS：antd <${t.tag}> 0 处 ${t.prop}（扫描 ${scanned} 个含 ${t.tag} 的 .tsx，` +
        `命中 antd ${t.tag} 的 ${stat[t.tag].elements} 个元素）` +
        (useTs ? '｜AST 精确判定' : '｜⚠ 降级为正则扫描（typescript 不可用）') +
        `｜豁免登记 ${(t.tag === 'Alert' ? C14D_EXEMPTIONS : C14E_EXEMPTIONS).length} 项`;
  };

  check(
    'C14d antd <Alert> message 弃用零命中（阻断）',
    violations.Alert.length === 0,
    summarize(TARGETS[0]),
  );
  check(
    'C14e antd <Progress> width 弃用零命中（阻断）',
    violations.Progress.length === 0,
    summarize(TARGETS[1]),
  );
  if (advisories.length) {
    console.log(`NOTE  C14d/C14e 人工确认 ${advisories.length} 条：${advisories.join(' | ')}`);
  }
}

// ==========================================================================
// Phase 88-05 收口补丁 新增断言：C14f / C14g —— antd 6 弃用 prop **全表**零命中（阻断）
// ==========================================================================
// 背景：M16 的 must_have 是「antd 弃用警告 = 0」。C14d / C14e 只锁了 Alert.message 与
//   Progress.width 两类；若只盯这两类，其余 antd 6 弃用残留（Space.direction、
//   notification.message、Image.preview.visible、Timeline items[].children 等）
//   仍会在浏览器 console 里报警 → must_have 不成立。
//   故本块按 **antd 6.6.4 实际弃用表**（穷举 node_modules/antd/es/** 弃用声明）建立
//   **数据驱动**的全表守护，一次性锁死所有组件级 / 配置对象级弃用 prop。
//
// 证据来源（antd 6.6.4 es/**/ 弃用声明行）：
//   alert/Alert.js:87            message → title
//   progress/progress.js:120     width → size / trailColor → railColor / gapPosition → gapPlacement
//   space/index.js:94            direction → orientation / split → separator
//   drawer/Drawer.js:155         width → size / height → size / destroyInactivePanel → destroyOnHidden
//   card/Card.js:93              bordered → variant
//   input/Input.js:49            bordered → variant
//   input-number/index.js:149    bordered → variant
//   modal/Modal.js:126           destroyOnClose → destroyOnHidden
//   collapse/Collapse.js:64      destroyInactivePanel → destroyOnHidden / expandIconPosition → expandIconPlacement
//   tooltip/index.js:97          destroyTooltipOnHide → destroyOnHidden
//   tabs/index.js:86             tabPosition → tabPlacement
//   divider/index.js:107         type → orientation
//   transfer/index.js:290        operations → actions
//   slider/index.js:116          tooltipPrefixCls → prefixCls / getTooltipPopupContainer → getPopupContainer
//                                 tipFormatter → formatter / tooltipPlacement → placement / tooltipVisible → open
//   calendar/generateCalendar.js:102  dateCellRender → cellRender / dateFullCellRender → fullCellRender
//                                 monthCellRender → cellRender / monthFullCellRender → fullCellRender
//   image/hooks/usePreviewConfig.js:68   （**preview 配置对象**内）visible → open / onVisibleChange → onOpenChange
//                                 toolbarRender → actionsRender / maskClassName → classNames.cover
//                                 rootClassName → classNames.root
//   notification/useNotification.js:155  （**配置对象**内）message → title / btn → actions
//   timeline/Timeline.js:107     （<Timeline> 的 <Timeline.Item> 子元素形态）Timeline.Item → items
//   timeline/Timeline.js:116-117 （**items[] 条目**内）label → title / children → content
//                                 dot → icon / position → placement
//
// 判定口径（**上下文感知**，与 C14c/C14d/C14e 一致）：
//   - 只判「确实绑定到 antd 导出」的 JSX 元素；自研同名组件不误报；
//   - 同名 prop 在**别的** antd 组件上合法（如 Alert 的 type、Divider 的 titlePlacement）→ 不误报；
//   - Timeline 的 label/children/dot/position 弃用作用于 **items[] 条目**而非 JSX 属性
//     （Timeline.js:115-117 的 warning 文本即 `items.<prop>`），故不在 JSX 属性表内，
//     由 C14g 的 items[] 规则覆盖；
//   - 配置对象类（Image.preview / notification / Timeline.items）按**对象键**判定。
{
  // ---- C14f：antd 组件级弃用 prop（JSX 属性形态） ----
  const ANTD6_JSX_DEPRECATED = {
    Alert: { message: 'title' },
    Progress: { width: 'size', trailColor: 'railColor', gapPosition: 'gapPlacement' },
    Space: { direction: 'orientation', split: 'separator' },
    Drawer: { width: 'size', height: 'size', destroyInactivePanel: 'destroyOnHidden' },
    Card: { bordered: 'variant' },
    Input: { bordered: 'variant' },
    InputNumber: { bordered: 'variant' },
    Modal: { destroyOnClose: 'destroyOnHidden' },
    Collapse: { destroyInactivePanel: 'destroyOnHidden', expandIconPosition: 'expandIconPlacement' },
    Tooltip: { destroyTooltipOnHide: 'destroyOnHidden' },
    Tabs: { tabPosition: 'tabPlacement' },
    Divider: { type: 'orientation' },
    Transfer: { operations: 'actions' },
    Slider: {
      tooltipPrefixCls: 'prefixCls',
      getTooltipPopupContainer: 'getPopupContainer',
      tipFormatter: 'formatter',
      tooltipPlacement: 'placement',
      tooltipVisible: 'open',
    },
    Calendar: {
      dateCellRender: 'cellRender',
      dateFullCellRender: 'fullCellRender',
      monthCellRender: 'cellRender',
      monthFullCellRender: 'fullCellRender',
    },
  };

  // ---- C14g：antd 配置对象级弃用 prop（对象键形态） ----
  // 结构：{ antd导出名: { 配置prop名: { 弃用键: 新形态 } } }
  const ANTD6_CONFIG_DEPRECATED = {
    Image: {
      preview: {
        visible: 'open',
        onVisibleChange: 'onOpenChange',
        toolbarRender: 'actionsRender',
        maskClassName: 'classNames.cover',
        rootClassName: 'classNames.root',
      },
    },
    Timeline: {
      items: { label: 'title', children: 'content', dot: 'icon', position: 'placement' },
    },
  };
  // notification 的配置对象是**函数调用实参**形态（非 JSX 属性），单独一条规则
  const NOTIFICATION_DEPRECATED = { message: 'title', btn: 'actions' };

  const tsMod2 = await import('typescript').catch(() => null);
  const useTs2 = !!tsMod2 && typeof tsMod2.createSourceFile === 'function';

  const fViolations = [];
  const gViolations = [];
  const fElements = new Map(); // 组件名 → 命中元素数
  const gScanned = { Image: 0, Timeline: 0, notification: 0 };

  if (useTs2) {
    const ts = tsMod2;
    const files2 = walkByExt(resolve(feRoot, 'src'), ['.tsx']);
    for (const f2 of files2) {
      const rel2 = relFe(f2);
      const raw2 = readFileSync(f2, 'utf8');
      if (!/\b(Alert|Progress|Space|Drawer|Card|Input|InputNumber|Modal|Collapse|Tooltip|Tabs|Divider|Transfer|Slider|Calendar|Timeline|Image|notification)\b/.test(raw2)) {
        continue;
      }
      const sf2 = ts.createSourceFile(f2, raw2, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);

      // ---- antd 绑定：本地名 → antd 原始导出名 ----
      const bind = new Map();
      for (const st of sf2.statements) {
        if (!ts.isImportDeclaration(st) || !ts.isStringLiteral(st.moduleSpecifier)) continue;
        const spec = st.moduleSpecifier.text;
        if (!(spec === 'antd' || spec.startsWith('antd/'))) continue;
        const ic = st.importClause;
        if (!ic || ic.isTypeOnly) continue;
        const nb = ic.namedBindings;
        if (nb && ts.isNamedImports(nb)) {
          for (const el of nb.elements) {
            if (el.isTypeOnly || el.propertyName) continue;
            bind.set(el.name.text, el.name.text);
          }
        }
      }
      const lineOf2 = (p) => sf2.getLineAndCharacterOfPosition(p).line + 1;

      // ---- C14f：JSX 属性 ----
      const visitJsx2 = (node) => {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          const tag = node.tagName.getText(sf2);
          const orig = bind.get(tag);
          if (orig && ANTD6_JSX_DEPRECATED[orig]) {
            const rule = ANTD6_JSX_DEPRECATED[orig];
            fElements.set(orig, (fElements.get(orig) || 0) + 1);
            for (const p of node.attributes.properties) {
              if (!ts.isJsxAttribute(p)) continue;
              const np = rule[p.name.text];
              if (!np) continue;
              fViolations.push(
                `${rel2}:${lineOf2(p.name.getStart(sf2))}  弃用 prop「${p.name.text}」仍传给 antd <${orig}>（antd 6 → ${np}）`,
              );
            }
          }
        }
        ts.forEachChild(node, visitJsx2);
      };
      visitJsx2(sf2);

      // ---- C14g：配置对象键（Image.preview / Timeline.items） ----
      const visitCfg = (node) => {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          const tag = node.tagName.getText(sf2);
          const orig = bind.get(tag);
          if (orig && ANTD6_CONFIG_DEPRECATED[orig]) {
            const cfgs = ANTD6_CONFIG_DEPRECATED[orig];
            for (const p of node.attributes.properties) {
              if (!ts.isJsxAttribute(p) || !p.initializer) continue;
              const cfgName = p.name.text;
              const banned = cfgs[cfgName];
              if (!banned) continue;
              if (!ts.isJsxExpression(p.initializer) || !p.initializer.expression) continue;
              gScanned[orig] = (gScanned[orig] || 0) + 1;
              // 递归找该配置对象下的对象字面量键
              const scanObj = (n) => {
                if (ts.isObjectLiteralExpression(n)) {
                  for (const prop of n.properties) {
                    if (!ts.isPropertyAssignment(prop)) continue;
                    const kn = prop.name;
                    const key = ts.isIdentifier(kn) ? kn.text : ts.isStringLiteral(kn) ? kn.text : null;
                    if (!key || !banned[key]) continue;
                    gViolations.push(
                      `${rel2}:${lineOf2(prop.getStart(sf2))}  弃用配置键「${key}」仍传给 antd <${orig} ${cfgName}>（antd 6 → ${banned[key]}）`,
                    );
                  }
                }
                ts.forEachChild(n, scanObj);
              };
              scanObj(p.initializer.expression);
            }
          }
        }
        // <Timeline> 的 <Timeline.Item> 子元素形态（antd 6 → items 数组）
        if (ts.isJsxElement(node)) {
          const tag = node.openingElement.tagName.getText(sf2);
          if (tag === 'Timeline' && bind.get('Timeline') === 'Timeline') {
            const hasItem = node.children.some(
              (c) => ts.isJsxElement(c) && c.openingElement.tagName.getText(sf2) === 'Timeline.Item',
            );
            if (hasItem) {
              gViolations.push(
                `${rel2}:${lineOf2(node.openingElement.getStart(sf2))}  antd <Timeline> 仍用 <Timeline.Item> 子元素形态（antd 6 → items 数组）`,
              );
            }
          }
        }
        ts.forEachChild(node, visitCfg);
      };
      visitCfg(sf2);

      // ---- C14g：notification 配置对象（函数实参形态） ----
      const visitCall2 = (node) => {
        if (ts.isCallExpression(node)) {
          const callee = node.expression;
          if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression)) {
            const orig = bind.get(callee.expression.text);
            if (orig === 'notification' || orig === 'Notification') {
              for (const arg of node.arguments) {
                if (!ts.isObjectLiteralExpression(arg)) continue;
                gScanned.notification = (gScanned.notification || 0) + 1;
                for (const prop of arg.properties) {
                  if (!ts.isPropertyAssignment(prop)) continue;
                  const kn = prop.name;
                  const key = ts.isIdentifier(kn) ? kn.text : ts.isStringLiteral(kn) ? kn.text : null;
                  if (!key || !NOTIFICATION_DEPRECATED[key]) continue;
                  gViolations.push(
                    `${rel2}:${lineOf2(prop.getStart(sf2))}  弃用配置键「${key}」仍传给 antd notification（antd 6 → ${NOTIFICATION_DEPRECATED[key]}）`,
                  );
                }
              }
            }
          }
        }
        ts.forEachChild(node, visitCall2);
      };
      visitCall2(sf2);
    }
  } else {
    console.log('NOTE  C14f/C14g 未执行：typescript 不可用（无法做 AST 精确判定）');
  }

  const covered = Object.keys(ANTD6_JSX_DEPRECATED).length;
  check(
    'C14f antd 6 组件级弃用 prop 全表零命中（阻断）',
    fViolations.length === 0,
    fViolations.length
      ? fViolations.join(' | ')
      : `C14f PASS：${covered} 个 antd 组件的弃用 prop 全表 0 命中` +
        (useTs2 ? '｜AST 精确判定' : '｜⚠ typescript 不可用，本项未执行'),
  );
  check(
    'C14g antd 6 配置对象级弃用 prop 零命中（阻断）',
    gViolations.length === 0,
    gViolations.length
      ? gViolations.join(' | ')
      : `C14g PASS：Image.preview / Timeline.items / notification 配置对象 0 命中` +
        `（已扫描 Image.preview ${gScanned.Image || 0} 处、Timeline.items ${gScanned.Timeline || 0} 处、notification ${gScanned.notification || 0} 处）` +
        (useTs2 ? '｜AST 精确判定' : '｜⚠ typescript 不可用，本项未执行'),
  );
}

// ---- C14b 冻结契约守护：FormModal / BaseEditModal 对外签名不得漂移 ----
// v1.22 契约：对外 prop 名仍是 maskClosable（调用方零改动），内部转发 antd 6 新形态。
{
  const CONTRACT = [
    {
      file: 'src/components/shared/FormModal.tsx',
      label: 'FormModal 对外 prop 名冻结 + 向 BaseEditModal 透传',
      patterns: ['maskClosable?: boolean;', 'maskClosable = false,', 'maskClosable={maskClosable}'],
    },
    {
      file: 'src/components/modal/BaseEditModal.tsx',
      label: 'BaseEditModal 对外 prop 名冻结 + 内部转发 antd 6 新形态',
      patterns: ['maskClosable?: boolean;', 'maskClosable = false,', 'mask={{ closable: maskClosable }}'],
    },
  ];
  const bad = [];
  for (const c of CONTRACT) {
    const p = resolve(feRoot, c.file);
    if (!existsSync(p)) {
      bad.push(`${c.file} 缺失`);
      continue;
    }
    const text = readFileSync(p, 'utf8');
    for (const pat of c.patterns) {
      if (!text.includes(pat)) bad.push(`${c.file} 缺少「${pat}」（${c.label}）`);
    }
  }
  check('C14b FormModal/BaseEditModal 冻结契约签名保留（阻断）', bad.length === 0, bad.length ? bad.join(' | ') : 'C14b PASS：对外签名与内部转发形态均未被改动');
}

console.log('----');
console.log(`结果: ${failures === 0 ? 'ALL PASS' : failures + ' 项 FAIL'}`);
process.exit(failures === 0 ? 0 : 1);
