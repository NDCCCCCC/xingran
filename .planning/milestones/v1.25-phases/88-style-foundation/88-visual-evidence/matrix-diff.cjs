// Phase 88-03 矩阵 diff：before vs after（明/暗），逐格比对 carrier+inner 5 项
const fs = require('fs');
const p = (f) => JSON.parse(JSON.parse(fs.readFileSync(__dirname + '/' + f, 'utf8'))).forms;
const props = ['bg', 'border', 'radius', 'shadow', 'btw'];
let total = 0, diffs = [];
for (const mode of [['light', 'before-light.json', 'after-light.json'], ['dark', 'before-dark.json', 'after-dark.json']]) {
  const [m, bf, af] = mode;
  const B = p(bf), A = p(af);
  for (const id in B) {
    for (const part of ['carrier', 'inner']) {
      const b = B[id][part], a = A[id] && A[id][part];
      if (!b && !a) continue;
      if (!b || !a) { diffs.push(`${m}/${id}/${part}: 结构缺失 before=${!!b} after=${!!a}`); total++; continue; }
      for (const k of props) {
        total++;
        if (b[k] !== a[k]) diffs.push(`${m}/${id}/${part}.${k}: ${b[k]} -> ${a[k]}`);
      }
    }
  }
}
console.log(`比对单元格总数: ${total}`);
console.log(`差异数: ${diffs.length}`);
diffs.forEach((d) => console.log('DIFF', d));
fs.writeFileSync(__dirname + '/matrix-diff-result.txt', `单元格 ${total}，差异 ${diffs.length}\n${diffs.join('\n')}\n`);
