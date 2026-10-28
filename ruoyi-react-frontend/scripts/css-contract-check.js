/* eslint-disable */
// ^ dev-only 工具脚本：scripts/*.mjs 被 eslint 全局忽略，本文件按任务规格命名为 .js，
//   故以文件内禁用保持 lint 预算（errors≤17）不被本文件击穿。
/**
 * v1.24 双框修复（VIS-01/02/03）CSS 契约静态检查 —— dev-only，node 运行、零依赖
 * ==============================================================================
 * 用法：node scripts/css-contract-check.js   （退出码 0 = 契约完好）
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
  const EXEMPTED_UNREF_VARS = [
    // Phase 89 R11：--input-placeholder 仅在 light 块定义，暗色 2 处引用声明整条作废
    //（清偿 = 统一为 --input-text-placeholder 并 :root 兜底，之后删除这 2 条豁免）
    { name: '--input-placeholder', line: 2276, reason: 'Phase 89 R11 暗色 placeholder 引用未定义变量' },
    { name: '--input-placeholder', line: 2301, reason: 'Phase 89 R11 暗色 placeholder 引用未定义变量' },
    // --theme-warning-text：2 处引用 0 定义（1 处无 fallback 命中；另 1 处有 fallback 不命中）
    { name: '--theme-warning-text', line: 4472, reason: 'Phase 89 R11 变量无定义，需补 :root 定义' },
  ];

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
// PLAN 预估 ≈1395，实跑 1398——ratchet 只许低不许高，故定值取实测）。
const IMPORTANT_BUDGET = 1398;
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

console.log('----');
console.log(`结果: ${failures === 0 ? 'ALL PASS' : failures + ' 项 FAIL'}`);
process.exit(failures === 0 ? 0 : 1);
