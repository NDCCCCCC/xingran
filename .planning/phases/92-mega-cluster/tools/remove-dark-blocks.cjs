#!/usr/bin/env node
/**
 * Phase 92 · Wave 2/3 — 暗色规则块级删除脚本（D-04 块级操作）
 *
 * 用法：node remove-dark-blocks.cjs <r6|r7>
 * 读取 92-EQUIVALENCE-DATA.json，删除指定区间内 cls !== 'exception' 的规则
 * （连同紧邻的注释行），合并区间后一次性重写文件。
 */
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../../../..');
const CSS = path.join(ROOT, 'ruoyi-react-frontend/src/index.css');
const DATA = path.join(__dirname, '../92-EQUIVALENCE-DATA.json');

const SCOPES = {
  r6: { start: 2225, end: 2798 },   // Wave 2 已消费（旧行号基线）
  r7: { start: 2300, end: 3940 },   // Wave 3：R6 删除 500 行后的新行号基线（Premium Glow → Modern Tag 前）
  all: { start: 1, end: Infinity }, // 终扫：清除全部非例外暗色散存量（cluster 外围散点）
};

const scope = process.argv[2];
if (!SCOPES[scope]) {
  console.error('用法: node remove-dark-blocks.cjs <r6|r7>');
  process.exit(1);
}

const { start, end } = SCOPES[scope];
const data = JSON.parse(fs.readFileSync(DATA, 'utf8'));
const toDelete = data.rules.filter(
  (r) => r.startLine >= start && r.startLine <= end && r.cls !== 'exception'
);

if (toDelete.length === 0) {
  console.log(`${scope.toUpperCase()} 区间无需删除`);
  process.exit(0);
}

const lines = fs.readFileSync(CSS, 'utf8').split('\n');
const isCommentLine = (l) => /^\s*(\/\*|\*|\/\/)/.test(l);
const isBlank = (l) => l.trim() === '';

const del = new Set();
let deletedRules = 0;
for (const rule of toDelete) {
  // 标记规则主体（1-based → 0-based）
  for (let l = rule.startLine; l <= rule.endLine; l++) del.add(l);
  // 向前吃掉紧邻的注释行（含 /* ... */ 多行注释）
  let p = rule.startLine - 1;
  let sawComment = false;
  while (p >= 1) {
    const line = lines[p - 1];
    if (isCommentLine(line)) { del.add(p); sawComment = true; p--; continue; }
    // 注释块后的空行（仅当已吃到注释时）也删
    if (sawComment && isBlank(line)) { del.add(p); p--; continue; }
    break;
  }
  deletedRules++;
}

// 删除后的规则间多余连续空行收敛：额外把删除区后紧跟的空行标记（最多 1 行保留）
// 简化策略：直接删除后做一轮「3+ 连续空行 → 1 空行」收敛
const kept = lines.filter((_, idx) => !del.has(idx + 1));
const collapsed = [];
let blankRun = 0;
for (const line of kept) {
  if (line.trim() === '') {
    blankRun++;
    if (blankRun > 2) continue; // 保留最多 2 连续空行（章节边界）
  } else {
    blankRun = 0;
  }
  collapsed.push(line);
}

fs.writeFileSync(CSS, collapsed.join('\n'));
console.log(`${scope.toUpperCase()} 删除完成：规则 ${deletedRules} 条 / 行 ${lines.length} → ${collapsed.length}（净 -${lines.length - collapsed.length} 行）`);
