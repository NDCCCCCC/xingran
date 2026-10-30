// Phase 90 W1 · M9 预算腾挪：压缩 index.css 纯冗余空行（只删块内空行，零声明改动）
// 用法：node p90-compress.cjs <csv 行号>      （行号为「删除前」的 1-based 行号）
// 约束：① 被删行必须是空行；② 前后两个非空行缩进均 > 0（块内空行）；③ 块间空行计数恒 801；
//       ④ 禁碰暗色只读区（删除前行号落在 :2254-2742 / :2780-4517 内）；⑤ 一行 CSS 声明都不许动。
const fs = require('fs');
const path = require('path');
const FE = path.resolve(__dirname, '../../../../ruoyi-react-frontend');
const file = path.join(FE, 'src/index.css');
const targets = String(process.argv[2] || '').split(',').filter(Boolean).map(Number);
if (!targets.length) { console.error('用法: node p90-compress.cjs <csv 行号>'); process.exit(1); }

const RO = [[2254, 2742], [2780, 4517]];
const text = fs.readFileSync(file, 'utf8');
const L = text.split('\n');
const before = { M9: L.length, wc: L.length - 1 };

const classify = (lines) => {
  let inBlock = 0, between = 0;
  const inLines = [];
  for (let x = 0; x < lines.length; x++) {
    if (lines[x].trim() !== '') continue;
    let p = x - 1; while (p >= 0 && lines[p].trim() === '') p--;
    let q = x + 1; while (q < lines.length && lines[q].trim() === '') q++;
    const pi = (lines[p] || '').match(/^\s*/)[0].length;
    const qi = (lines[q] || '').match(/^\s*/)[0].length;
    if (pi > 0 && qi > 0) { inBlock++; inLines.push(x + 1); } else between++;
  }
  return { inBlock, between, inLines };
};

const pre = classify(L);
console.log('压缩前：M9=' + before.M9 + ' wc-l=' + before.wc + ' 块内空行=' + pre.inBlock + ' 块间空行=' + pre.between);

// 逐条校验
const errs = [];
for (const n of targets) {
  if (L[n - 1] !== undefined && L[n - 1].trim() !== '') errs.push(`行 ${n} 非空行`);
  if (!pre.inLines.includes(n)) errs.push(`行 ${n} 不是块内空行`);
  if (RO.some(([a, b]) => n >= a && n <= b)) errs.push(`行 ${n} 落在暗色只读区`);
}
if (new Set(targets).size !== targets.length) errs.push('目标行号有重复');
if (errs.length) { console.error('FAIL:\n' + errs.join('\n')); process.exit(1); }

const del = new Set(targets);
const out = L.filter((_, i) => !del.has(i + 1)).join('\n');
fs.writeFileSync(file, out);

const L2 = fs.readFileSync(file, 'utf8').split('\n');
const post = classify(L2);
console.log('压缩后：M9=' + L2.length + ' wc-l=' + (L2.length - 1) + ' 块内空行=' + post.inBlock + ' 块间空行=' + post.between);
console.log('D=' + targets.length + ' G=' + (before.M9 - 5556) + ' 最终 M9=' + L2.length + '（预算 ≤5558）');
if (post.between !== 801) { console.error('FAIL: 块间空行计数变化 ' + post.between); process.exit(1); }
if (L2.length > 5558) { console.error('FAIL: M9 超预算 ' + L2.length); process.exit(1); }
console.log('被删行号清单：' + targets.join(','));
console.log('OK 压缩完成');
