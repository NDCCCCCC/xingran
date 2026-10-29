// Phase 90 矩阵 diff（白名单归类模式 · 90 口径）
// 与 89（matrix-diff-89.cjs）的四处关键差别：
//   ① light 差异不再要求 = 0 → 改为「light 与 dark 各自差异数 > 0（防空跑/恒等）」；
//      89 的「浅色零漂移红线」在 90 必然误 FAIL（圆角 6→8 明暗双向）。仍保留「未归类 = exit 1」。
//   ② 登记表由 89 的 A/B 两类改为 90 的变更类别三类：a 圆角 6→8 / b InputNumber 错误态红 / c 可选 font-size。
//   ③ 采样属性由 5 个扩为 6 个（末尾追加 fontSize）→ 矩阵由 5 属性扩为 6 属性。
//   ④ 形态集合 22 形态（89 的 21 形态 + f22_input_number_status_error）；登记表形态清单
//      由本文件的常量维护，与 90-CHANGESET.md 双写（两处必须一致）。
// 口径：before/after × 明暗 × carrier/inner × 6 属性（bg/border/radius/shadow/btw/fontSize）
// 格数口径：(22 形态 × 2 part − 6 个 inner=null 形态) × 6 属性 = 每模式 228 格、明暗合计 456 格
// 用法：node matrix-diff-90.cjs [afterPrefix]   （默认 after；读取本目录 before-{light,dark}.json）
const fs = require('fs');
const path = require('path');

// 88 快照为双编码（stringify(stringify(obj))），89/90 采样为单编码 → 自适应解析
const p = (f) => {
  let v = JSON.parse(fs.readFileSync(path.join(__dirname, f), 'utf8'));
  if (typeof v === 'string') v = JSON.parse(v);
  return v.forms;
};
const props = ['bg', 'border', 'radius', 'shadow', 'btw', 'fontSize'];

// ===== 90-CHANGESET.md 登记表（与文档同步维护 · 代码内常量 + 文档双写纪律）=====
// 形态总数（22）—— 必须与 90-CHANGESET.md 一致
const FORMS_90 = [
  'f01_input_plain', 'f02_input_prefix', 'f03_input_suffix', 'f04_input_allowclear',
  'f05_input_addonbefore', 'f06_input_addonafter', 'f07_input_password', 'f08_input_search',
  'f09_textarea', 'f10_textarea_showcount', 'f11_input_number',
  'f12_select_single', 'f13_select_multiple', 'f14_datepicker', 'f15_range_picker',
  'f16_cascader', 'f17_treeselect', 'f18_autocomplete', 'f19_mentions',
  'f20_appinput', 'f21_apptextarea', 'f22_input_number_status_error',
];

// a 类落点（圆角 6px → 8px）：按 W0 before 实采快照逐格派生 —— 「before.radius === '6px'」的格
//   carrier：f01-f11（除 f12/f13 外全部 Input 族）+ f20 + f21 + f22
//   inner  ：f05 / f06（carrier 与 inner 同为 input.ant-input）+ f14 / f15（picker 内层 input.ant-input，
//            不在 VIS-01 复位块 `.ant-input-affix-wrapper > input.ant-input` 作用域内 → 仍吃 :885 的 6px）
//   内层 0px（f01/f02/f03/f04/f07/f08/f09/f10/f11/f19/f20/f21/f22）由 VIS-01 复位块保证 → 0→0，不产生差异
const A_CELLS = new Set([
  // carrier.radius
  'f01_input_plain.carrier.radius', 'f02_input_prefix.carrier.radius', 'f03_input_suffix.carrier.radius',
  'f04_input_allowclear.carrier.radius', 'f05_input_addonbefore.carrier.radius', 'f06_input_addonafter.carrier.radius',
  'f07_input_password.carrier.radius', 'f08_input_search.carrier.radius', 'f09_textarea.carrier.radius',
  'f10_textarea_showcount.carrier.radius', 'f11_input_number.carrier.radius', 'f20_appinput.carrier.radius',
  'f21_apptextarea.carrier.radius', 'f22_input_number_status_error.carrier.radius',
  // inner.radius
  'f05_input_addonbefore.inner.radius', 'f06_input_addonafter.inner.radius',
  'f14_datepicker.inner.radius', 'f15_range_picker.inner.radius',
]);

// b 类落点（InputNumber 错误态红框）：仅 f22 的根元素边框（antd 6.6.4 把 status class 打在根，边框也画在根）
//   before：light = rgb(219, 215, 206)（--input-border #dbd7ce）/ dark = rgb(45, 77, 51)（--input-border #2d4d33）
//   after ：走 `var(--ant-color-error)`（与项目既有 VIS-03 规则 :967-974 同源）→ **明暗不同值**：
//           light = #BA3630 → rgb(186, 54, 48)；dark = #a1312c → rgb(161, 49, 44)
//   ⚠️ 该「明暗不同值」为 W0 实测裁定（probe-status-before.json），推翻 90-RESEARCH §7.3 的
//      「--ant-color-error 明暗同值 #BA3630」推定 —— antd darkAlgorithm 对 colorError 做了暗色派生。
const B_CELLS = new Set(['f22_input_number_status_error.carrier.border']);
const B_AFTER = {
  light: { before: 'rgb(219, 215, 206)', after: 'rgb(186, 54, 48)' },
  dark: { before: 'rgb(45, 77, 51)', after: 'rgb(161, 49, 44)' },
};

// c 类落点（font-size，可选）：T4（删 `:886` font-size: 14px !important）由 90-03 裁定；
//   裁定为「采纳且实测零差异」则恒为空集；裁定为「采纳且有差异」则在此登记 before/after 与形态集合。
//   W0 登记为空集占位 —— match 为 null 表示不接受任何 fontSize 差异（出现即未归类 → FAIL）。
const C_CELLS = new Set();

const REGISTRY = [
  {
    id: 'a',
    desc: '输入类圆角 6→8（--theme-radius-md → --theme-radius-lg，STYLE-03）',
    match: (m, form, part, prop, before, after) =>
      prop === 'radius' && before === '6px' && after === '8px' && A_CELLS.has(`${form}.${part}.radius`),
  },
  {
    id: 'b',
    desc: 'InputNumber 错误态边框（--input-border → --ant-color-error，VISFIX-05；明暗 after 值不同）',
    match: (m, form, part, prop, before, after) =>
      prop === 'border' && B_CELLS.has(`${form}.${part}.border`) &&
      B_AFTER[m] && before === B_AFTER[m].before && after === B_AFTER[m].after,
  },
  {
    id: 'c',
    desc: 'font-size（可选，T4 裁定后回填；W0 登记为空集占位）',
    match: null,
  },
];
const EXPECTED = { a: A_CELLS, b: B_CELLS, c: C_CELLS };

const afterPrefix = process.argv[2] || 'after';
let total = 0;
const diffs = [];
const formsBefore = new Set(), formsAfter = new Set();
for (const [m, bf, af] of [
  ['light', 'before-light.json', `${afterPrefix}-light.json`],
  ['dark', 'before-dark.json', `${afterPrefix}-dark.json`],
]) {
  const B = p(bf), A = p(af);
  Object.keys(B).forEach((k) => formsBefore.add(k));
  Object.keys(A).forEach((k) => formsAfter.add(k));
  for (const id in B) {
    for (const part of ['carrier', 'inner']) {
      const b = B[id][part], a = A[id] && A[id][part];
      if (!b && !a) continue;
      if (!b || !a) { diffs.push({ m, id, part, prop: '<structure>', text: `${m}/${id}/${part}: 结构缺失 before=${!!b} after=${!!a}` }); total++; continue; }
      for (const k of props) {
        total++;
        if (b[k] !== a[k]) diffs.push({ m, id, part, prop: k, before: b[k], after: a[k], text: `${m}/${id}/${part}.${k}: ${b[k]} -> ${a[k]}` });
      }
    }
  }
}

const lightDiffs = diffs.filter((d) => d.m === 'light');
const darkDiffs = diffs.filter((d) => d.m === 'dark');
const unclassified = [];
const classified = [];
for (const d of diffs) {
  const hit = REGISTRY.find((r) => r.match && d.prop !== '<structure>' && r.match(d.m, d.id, d.part, d.prop, d.before, d.after));
  hit ? classified.push({ ...d, cls: hit.id }) : unclassified.push(d);
}

console.log(`比对单元格总数: ${total}`);
console.log(`差异数: ${diffs.length}（light=${lightDiffs.length} / dark=${darkDiffs.length}）`);
classified.forEach((d) => console.log(`CLASSIFIED [${d.cls}]`, d.text));
unclassified.forEach((d) => console.log('UNCLASSIFIED', d.text));

const byForm = {};
diffs.forEach((d) => { (byForm[`${d.m}/${d.id}.${d.part}.${d.prop}`] ||= []).push(d.before + '->' + d.after); });
console.log('差异形态清单:', JSON.stringify(byForm, null, 1));

let fail = false;
// ① 形态集合一致性：before 与 after 的形态集合必须完全相同，且等于登记的 22 形态
const onlyBefore = [...formsBefore].filter((x) => !formsAfter.has(x));
const onlyAfter = [...formsAfter].filter((x) => !formsBefore.has(x));
if (onlyBefore.length) { console.error('FAIL: after 缺失形态:', onlyBefore.join(', ')); fail = true; }
if (onlyAfter.length) { console.error('FAIL: after 多出未登记形态:', onlyAfter.join(', ')); fail = true; }
for (const f of FORMS_90) {
  if (!formsBefore.has(f) || !formsAfter.has(f)) { console.error(`FAIL: 登记的 22 形态之一缺失: ${f}`); fail = true; }
}
// ② 防空跑红线（保留 89 的恒等空跑检测，双侧）：light / dark 各自差异必须 > 0
if (lightDiffs.length === 0) { console.error('FAIL: light 差异为 0（疑似恒等空跑 / 变更未生效）'); fail = true; }
if (darkDiffs.length === 0) { console.error('FAIL: dark 差异为 0（疑似恒等空跑 / 变更未生效）'); fail = true; }
// ③ 全量归类：未归类差异 = FAIL
if (unclassified.length > 0) { console.error(`FAIL: ${unclassified.length} 条差异未归入登记类别`); fail = true; }
// ④ 归类后形态集合与登记表一致（不多不少）+ 明暗双向一致（90 的变更是明暗双向的）
for (const cls of Object.keys(EXPECTED)) {
  const expected = EXPECTED[cls];
  for (const m of ['light', 'dark']) {
    const actual = new Set(classified.filter((c) => c.cls === cls && c.m === m).map((c) => `${c.id}.${c.part}.${c.prop}`));
    const missing = [...expected].filter((x) => !actual.has(x));
    const extra = [...actual].filter((x) => !expected.has(x));
    if (missing.length) { console.error(`FAIL: [${cls}] ${m} 登记的落点未在实测中出现:`, missing.join(', ')); fail = true; }
    if (extra.length) { console.error(`FAIL: [${cls}] ${m} 实测出现未登记的落点:`, extra.join(', ')); fail = true; }
  }
}

fs.writeFileSync(path.join(__dirname, 'matrix-diff-result-90.txt'),
  `单元格 ${total}，差异 ${diffs.length}（light=${lightDiffs.length}/dark=${darkDiffs.length}），归类 ${classified.length}，未归类 ${unclassified.length}\n` +
  `形态数 before=${formsBefore.size} / after=${formsAfter.size}\n` +
  diffs.map((d) => (d.cls ? `[${d.cls}] ` : 'UNCLASSIFIED ') + d.text).join('\n') + '\n');

if (fail) process.exit(1);
console.log('RESULT: light 差异 > 0 / dark 差异 > 0 / 全部差异归类 / 未归类 0 / a·b·c 形态集合与登记一致 → PASS');
