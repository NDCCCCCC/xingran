// 汇总 eslint JSON 输出：总数 + 按文件分组
let d = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (d += c));
process.stdin.on('end', () => {
  const r = JSON.parse(d);
  let e = 0, w = 0;
  const rows = [];
  for (const f of r) {
    if (f.errorCount || f.warningCount) {
      e += f.errorCount;
      w += f.warningCount;
      const short = f.filePath.split('ruoyi-react-frontend')[1] || f.filePath;
      rows.push(`${f.errorCount}E/${f.warningCount}W ${short}`);
      for (const m of f.messages) {
        rows.push(`   :${m.line}:${m.column} ${m.severity === 2 ? 'E' : 'W'} ${m.ruleId} — ${m.message.slice(0, 110)}`);
      }
    }
  }
  console.log('TOTAL', e, 'errors', w, 'warnings');
  console.log(rows.join('\n'));
});
