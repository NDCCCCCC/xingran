// Phase 89 矩阵 diff（白名单归类模式）
// 判据：① light 模式差异 = 0（浅色零漂移红线）
//       ② dark 模式差异 > 0（防恒等空跑）
//       ③ 每条差异必须归入 89-CHANGESET.md 登记的变更类别（A/B/…），未归类 = exit 1
//       ④ 归类后形态集合与登记表完全一致（不多不少）
// 口径沿用 88：before/after × 明暗 × carrier/inner × 5 属性（bg/border/radius/shadow/btw）= 360 格
const fs = require('fs');
const path = require('path');

// 88 快照为双编码（stringify(stringify(obj))），89 采样为单编码 → 自适应解析
const p = (f) => {
  let v = JSON.parse(fs.readFileSync(path.join(__dirname, f), 'utf8'));
  if (typeof v === 'string') v = JSON.parse(v);
  return v.forms;
};
const props = ['bg', 'border', 'radius', 'shadow', 'btw'];

// ===== 89-CHANGESET.md 登记表（与文档同步维护）=====
// A · 暗色输入类底色：--input-bg 串色修正（rgb(250,250,249) → rgb(15,46,27)）
//     形态集合按实跑校准；carrier.bg（部分含 inner.bg），prop=bg
const REGISTRY = [
  {
    id: 'A',
    desc: '暗色输入类底色（--input-bg 串色修正，VISFIX-01）',
    match: (m, form, part, prop, before, after) =>
      m === 'dark' &&
      prop === 'bg' &&
      before === 'rgb(250, 250, 249)' &&
      after === 'rgb(15, 46, 27)' &&
      A_FORMS.has(form) &&
      A_PARTS[form].includes(part),
  },
];
// A 类形态集合（实跑校准：研究 §1.5.1 推定 + f19/f05/f06/f11 实测补全）
const A_FORMS = new Set([
  'f01_input_plain', 'f02_input_prefix', 'f03_input_suffix', 'f04_input_allowclear',
  'f05_input_addonbefore', 'f06_input_addonafter', 'f07_input_password', 'f08_input_search',
  'f09_textarea', 'f10_textarea_showcount', 'f11_input_number',
  'f14_datepicker', 'f15_range_picker', 'f19_mentions', 'f20_appinput', 'f21_apptextarea',
]);
// carrier 全量；inner 实测仅 f05/f06/f11/f19（f01/f14/f15 内层被 VIS-01 复位为 transparent，不变）
const A_PARTS = {
  f01_input_plain: ['carrier'], f02_input_prefix: ['carrier'], f03_input_suffix: ['carrier'],
  f04_input_allowclear: ['carrier'], f05_input_addonbefore: ['carrier', 'inner'],
  f06_input_addonafter: ['carrier', 'inner'], f07_input_password: ['carrier'],
  f08_input_search: ['carrier'], f09_textarea: ['carrier'], f10_textarea_showcount: ['carrier'],
  f11_input_number: ['carrier', 'inner'], f14_datepicker: ['carrier'], f15_range_picker: ['carrier'],
  f19_mentions: ['carrier', 'inner'], f20_appinput: ['carrier'], f21_apptextarea: ['carrier'],
};

let total = 0;
const diffs = [];
for (const [m, bf, af] of [
  ['light', 'before-light.json', `${process.argv[2] || 'after'}-light.json`],
  ['dark', 'before-dark.json', `${process.argv[2] || 'after'}-dark.json`],
]) {
  const B = p(bf), A = p(af);
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
darkDiffs.forEach((d) => { (byForm[`${d.id}.${d.part}.${d.prop}`] ||= []).push(d.before + '->' + d.after); });
console.log('dark 差异形态清单:', JSON.stringify(byForm, null, 1));

let fail = false;
if (lightDiffs.length !== 0) { console.error(`FAIL: light 模式出现 ${lightDiffs.length} 处差异（浅色回归红线）`); fail = true; }
if (darkDiffs.length === 0) { console.error('FAIL: dark 差异为 0（疑似恒等空跑，变更未生效）'); fail = true; }
if (unclassified.length > 0) { console.error(`FAIL: ${unclassified.length} 条差异未归入登记类别`); fail = true; }

// ④ 归类后形态集合与登记表一致（不多不少）：A_FORMS × carrier.bg 必须全部实际出现
const actualA = new Set(classified.filter((c) => c.cls === 'A').map((c) => `${c.id}.${c.part}`));
const expectedA = new Set([...A_FORMS].flatMap((f) => A_PARTS[f].map((pt) => `${f}.${pt}`)));
const missing = [...expectedA].filter((x) => !actualA.has(x));
const extra = [...actualA].filter((x) => !expectedA.has(x));
if (missing.length) { console.error('FAIL: 登记的 A 类形态未在实测中出现:', missing.join(', ')); fail = true; }
if (extra.length) { console.error('FAIL: 实测出现未登记的 A 类形态:', extra.join(', ')); fail = true; }

fs.writeFileSync(path.join(__dirname, 'matrix-diff-result-89.txt'),
  `单元格 ${total}，差异 ${diffs.length}（light=${lightDiffs.length}/dark=${darkDiffs.length}），归类 ${classified.length}，未归类 ${unclassified.length}\n` +
  diffs.map((d) => (d.cls ? `[${d.cls}] ` : 'UNCLASSIFIED ') + d.text).join('\n') + '\n');

if (fail) process.exit(1);
console.log('RESULT: light 差异 0 / dark 差异全部归类 / 未归类 0 / A 类形态集合与登记一致 → PASS');
