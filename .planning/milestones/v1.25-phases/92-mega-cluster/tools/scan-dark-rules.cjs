#!/usr/bin/env node
/**
 * Phase 92 · Wave 1 — 暗色规则等价性扫描脚本（D-01）
 *
 * 用法：
 *   node scan-dark-rules.cjs                 # 全量扫描，输出三类统计与 92-EQUIVALENCE.md 数据
 *   node scan-dark-rules.cjs --check         # 校验提取完整性（区间规则数 vs 提取数）
 *   node scan-dark-rules.cjs --check-r6-cleared  # R6 区间（:2225-2799）暗色规则应为零
 *   node scan-dark-rules.cjs --check-r7-cleared  # R7 区间（:2799-4434）暗色规则应仅剩例外
 *
 * 输出：三类分类（covered / extendable / exception）+ 行号区间清单。
 * 行号为 1-based，与编辑器一致。
 */
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../../../..');
const CSS = path.join(ROOT, 'xingran-react-frontend/src/index.css');
const BRIDGE = path.join(ROOT, 'xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx');

// 与 92-RESEARCH.md §二 实测边界一致
const R6 = { start: 2225, end: 2798 };
const R7 = { start: 2799, end: 4434 };

// 例外白名单（92-RESEARCH §四：antd token 无法表达的场景，保留分区）
const EXCEPTION_PATTERNS = [
  /glow/i,            // Premium Glow Effects
  /scrollbar/i,       // Scrollbar 滚动条
  /::selection/i,     // Selection 选中文本
  /focus-visible/i,   // Focus Visible 可访问性
];

// antd 组件名 → Bridge components 键（用于 covered/extendable 判断）
const BRIDGE_COMPONENTS = [
  'Button', 'Table', 'Input', 'InputNumber', 'Select', 'Form', 'Menu',
  'Tabs', 'Tag', 'Card', 'Modal', 'Pagination', 'Dropdown',
];
const KNOWN_ANTD = [
  ...BRIDGE_COMPONENTS,
  'Alert', 'Transfer', 'Skeleton', 'Result', 'Descriptions', 'List', 'Collapse',
  'Slider', 'Rate', 'Spin', 'Empty', 'Steps', 'Upload', 'Calendar', 'Badge',
  'Progress', 'Carousel', 'QRCode', 'Comment', 'Notification', 'Typography',
  'Segmented', 'FloatButton', 'AutoComplete', 'Switch', 'TimePicker', 'Anchor',
  'Space', 'Divider', 'Tooltip', 'Popover', 'Drawer', 'Tree', 'Checkbox',
  'Radio', 'Message', 'Breadcrumb', 'Avatar', 'Layout', 'Cascader', 'DatePicker',
  'RangePicker', 'Timeline', 'Watermark', 'App', 'affix', 'login', 'cron',
];

function stripComments(text) {
  // 剥离 /* */ 注释但保留换行数（行号不变）
  return text.replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, ' '));
}

function extractDarkRules(text) {
  const lines = text.split('\n');
  const rules = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line.includes('[data-color-mode="dark"]')) {
      // 多行选择器支持：从 dark 行向后拼接直到出现 {
      let selEnd = i;
      let joined = line;
      while (!joined.includes('{') && selEnd + 1 < lines.length) {
        selEnd++;
        joined += '\n' + lines[selEnd];
      }
      if (!joined.includes('{')) { i++; continue; }
      // 块规则：向后找匹配的 }
      let depth = (joined.match(/\{/g) || []).length - (joined.match(/\}/g) || []).length;
      let end = selEnd;
      while (depth > 0 && end + 1 < lines.length) {
        end++;
        depth += (lines[end].match(/\{/g) || []).length - (lines[end].match(/\}/g) || []).length;
      }
      const selector = joined.replace(/\{[\s\S]*$/, '').replace(/\s+/g, ' ').trim();
      const body = lines.slice(i, end + 1).join('\n');
      const decls = (body.match(/[\w-]+\s*:\s*[^;{}]+;/g) || []).map((s) => s.trim());
      rules.push({
        startLine: i + 1,
        endLine: end + 1,
        selector,
        declCount: decls.length,
        importantCount: (body.match(/!important/g) || []).length,
        decls,
      });
      i = end + 1;
      continue;
    }
    i++;
  }
  return rules;
}

function classify(rule) {
  const sel = rule.selector;
  if (EXCEPTION_PATTERNS.some((p) => p.test(sel))) return 'exception';
  // 命中 Bridge 已配置组件 → covered；命中其余 antd 组件 → extendable；否则 exception
  for (const comp of BRIDGE_COMPONENTS) {
    if (sel.includes(`.ant-${comp.toLowerCase()}`) || sel.includes(`.ant-${comp.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase()).slice(1)}`)) {
      return 'covered';
    }
  }
  // 常见类名快查
  const lower = sel.toLowerCase();
  if (/\.ant-(btn|input|select|form|table|modal|dropdown|menu|tabs|tag|card|pagination|input-number)/.test(lower)) return 'covered';
  if (/\.ant-/.test(lower)) return 'extendable';
  return 'exception'; // 非 antd 类（登录页、自定义组件等）
}

function main() {
  const raw = fs.readFileSync(CSS, 'utf8');
  const text = stripComments(raw);
  const rules = extractDarkRules(text);
  const bridgeSrc = fs.readFileSync(BRIDGE, 'utf8');

  const classified = rules.map((r) => ({ ...r, cls: classify(r) }));
  const summary = {
    total: classified.length,
    covered: classified.filter((r) => r.cls === 'covered').length,
    extendable: classified.filter((r) => r.cls === 'extendable').length,
    exception: classified.filter((r) => r.cls === 'exception').length,
    importantTotal: rules.reduce((s, r) => s + r.importantCount, 0),
  };

  const mode = process.argv[2] || '';
  if (mode === '--check') {
    // 完整性校验：提取的规则行必须覆盖所有含 dark 的行
    const darkLines = text.split('\n')
      .map((l, idx) => (l.includes('[data-color-mode="dark"]') ? idx + 1 : 0))
      .filter(Boolean);
    const coveredLines = new Set();
    for (const r of rules) {
      for (let l = r.startLine; l <= r.endLine; l++) coveredLines.add(l);
    }
    const missed = darkLines.filter((l) => !coveredLines.has(l));
    if (missed.length > 0) {
      console.error(`FAIL 漏项 ${missed.length} 行：${missed.slice(0, 10).join(',')}`);
      process.exit(1);
    }
    console.log(`OK 提取 ${rules.length} 条规则，覆盖全部 ${darkLines.length} 个暗色行，零漏项`);
    console.log(JSON.stringify(summary));
    process.exit(0);
  }

  if (mode === '--check-r6-cleared') {
    // 口径：非例外暗色规则总数 = 全量基线 - R6 区间非例外数（不受删除后行号漂移影响）
    const base = JSON.parse(fs.readFileSync(path.join(__dirname, '../92-EQUIVALENCE-DATA.json'), 'utf8'));
    const r6NonEx = base.rules.filter((r) => r.startLine >= R6.start && r.startLine <= R6.end && r.cls !== 'exception').length;
    const now = extractDarkRules(text);
    const nowNonEx = now.filter((r) => classify(r) !== 'exception').length;
    const expected = base.rules.filter((r) => r.cls !== 'exception').length - r6NonEx;
    if (nowNonEx > expected) {
      console.error(`FAIL R6 未清干净：当前非例外 ${nowNonEx} 条 > 预期 ${expected} 条`);
      process.exit(1);
    }
    console.log(`OK R6 清零达成：非例外暗色规则 ${nowNonEx} / 预期 ≤${expected}`);
    process.exit(0);
  }

  if (mode === '--check-r7-cleared') {
    // 口径：非例外暗色规则仅剩 R6+R7 之外的部分（:4434 后与 R6 前的存量）
    const base = JSON.parse(fs.readFileSync(path.join(__dirname, '../92-EQUIVALENCE-DATA.json'), 'utf8'));
    const scopeNonEx = base.rules.filter((r) => ((r.startLine >= R6.start && r.startLine <= R6.end) || (r.startLine >= R7.start && r.startLine <= R7.end)) && r.cls !== 'exception').length;
    const now = extractDarkRules(text);
    const nowNonEx = now.filter((r) => classify(r) !== 'exception').length;
    const expected = base.rules.filter((r) => r.cls !== 'exception').length - scopeNonEx;
    if (nowNonEx > expected) {
      console.error(`FAIL R7 未清干净：当前非例外 ${nowNonEx} 条 > 预期 ${expected} 条`);
      process.exit(1);
    }
    const exceptions = now.filter((r) => classify(r) === 'exception');
    console.log(`OK R6+R7 清零达成：非例外暗色规则 ${nowNonEx} / 预期 ≤${expected}；例外保留 ${exceptions.length} 条`);
    process.exit(0);
  }

  // 默认：输出等价性清单数据
  const out = { summary, rules: classified.map(({ decls, ...r }) => ({ ...r, decls: decls.slice(0, 6) })) };
  const outPath = path.join(__dirname, '../92-EQUIVALENCE-DATA.json');
  fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
  console.log(`扫描完成：共 ${summary.total} 条 | covered=${summary.covered} extendable=${summary.extendable} exception=${summary.exception} | !important=${summary.importantTotal}`);
  console.log(`数据已写入 ${outPath}`);
}

main();
