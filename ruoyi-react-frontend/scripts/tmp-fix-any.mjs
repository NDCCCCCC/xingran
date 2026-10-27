// Phase 74 Wave C1：测试文件 no-explicit-any → unknown（精确位置替换 any token）
// 依据：tsconfig.app.json exclude 测试文件（type-check 不查），vitest esbuild 转译不做类型检查，
// 类型标注改动对运行时零影响。仅在 eslint 报告的 (line,col) 位置替换单词 any → unknown。
// 用法：node scripts/tmp-fix-any.mjs <lint-json>
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const lintJson = process.argv[2];
const results = JSON.parse(readFileSync(lintJson, "utf8"));
const byFile = new Map();
for (const f of results) {
  for (const m of f.messages) {
    if (m.ruleId !== "@typescript-eslint/no-explicit-any") continue;
    if (!f.filePath.includes("__tests__") && !/\.test\.(ts|tsx)$/.test(f.filePath)) continue;
    if (!byFile.has(f.filePath)) byFile.set(f.filePath, []);
    byFile.get(f.filePath).push({ line: m.line, col: m.column });
  }
}
let count = 0;
for (const [file, list] of byFile) {
  const lines = readFileSync(file, "utf8").split("\n");
  const perLine = new Map();
  for (const it of list) {
    if (!perLine.has(it.line)) perLine.set(it.line, []);
    perLine.get(it.line).push(it.col);
  }
  for (const [lineNo, cols] of perLine) {
    let line = lines[lineNo - 1];
    // 同行多个 any：从右往左替换
    cols.sort((a, b) => b - a);
    for (const col of cols) {
      const i = col - 1;
      if (line.slice(i, i + 3) !== "any") {
        console.log("MISMATCH", path.relative(process.cwd(), file) + ":" + lineNo + ":" + col);
        continue;
      }
      line = line.slice(0, i) + "unknown" + line.slice(i + 3);
      count++;
    }
    lines[lineNo - 1] = line;
  }
  writeFileSync(file, lines.join("\n"), "utf8");
}
console.log("replaced:", count, "files:", byFile.size);
