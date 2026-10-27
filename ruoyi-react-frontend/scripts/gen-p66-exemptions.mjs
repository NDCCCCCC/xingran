#!/usr/bin/env node
/**
 * Phase 66 — 存量豁免清单生成脚本（PLAN T7 生成 / T8 收口复跑对账；P67 收敛后清偿）
 *
 * 跑 eslint --format json 全量分析 src，收集命中 Phase 66 两条 hook import
 * 拦截（@/hooks/useTableManager / @/hooks/usePagination）的文件全集，并把
 * 清单写入 eslint.config.js 的「Phase 66 存量豁免 override」锚点段
 * （`__P66_EXEMPTION_FILES__`）。文件按路径排序，输出稳定可 diff。
 *
 * 关键设计（同 gen-p65-exemptions.mjs）：
 * ① 自清空悖论规避——存量豁免清单生效时 eslint 不再报这批文件 → 直接扫 src
 *   永远命中 0。因此扫描使用「禁用存量豁免」的临时 config
 *   （eslint.config.p66-scan.js：锚点段文件清单替换为占位 glob），分析完成
 *   后删除临时文件，再按真实命中集写回正式 config 的锚点段。
 * ② 临时 config 基于完整 config 复制、只替换 p66 锚点段，P65 段原样保留
 *   （p65 段只关 no-restricted-syntax，不影响 import 扫描）。
 * ③ 命中识别按 no-restricted-imports 的 message 标记（类比 P65_MESSAGE_MARKS）。
 *
 * Phase 74 收口（2026-10-28）：本脚本转为 **校验模式（VERIFICATION-ONLY）**——
 * 写回逻辑已移除。P66 存量豁免清单（37 条）经逐文件 grep 核实全部仍真实命中
 * useTableManager/usePagination import（PageTable 封装层为合法豁免点，不在清单），
 * 清单条目数 = 真实命中数，故豁免块保留为极小清单不再收缩。
 *
 * 现在脚本只做两件事：
 * ① 打印当前真实命中清单（等价旧 --dry）；
 * ② 与 eslint.config.js 锚点段现存清单比对：一致 → exit 0；不一致 → 打印差异并
 *    exit 1（防回退：新增 hook import 命中文件未登记、或清单残留已迁移条目均会报红）。
 *
 * 用法：node scripts/gen-p66-exemptions.mjs [--dry]（--dry 为兼容保留，行为相同——
 * 本脚本已无写回能力，天然幂等：连跑两次输出 diff 恒为空）
 */
import { spawnSync } from "node:child_process";
import { readFileSync, writeFileSync, rmSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const configPath = path.join(root, "eslint.config.js");
const scanConfigPath = path.join(root, "eslint.config.p66-scan.js");
const ANCHOR = "      // __P66_EXEMPTION_FILES__（脚本替换锚点）";
const PLACEHOLDER = '      "src/__p66_no_exemptions__/**",';

// Phase 66 两条 hook import 拦截的 message 前缀（与 eslint.config.js 保持同步）
const P66_MESSAGE_MARKS = [
  "新列表页请使用 @/components/shared 的 PageTable",
];

// 生成「禁用存量豁免」的临时扫描 config；同时校验正式 config 锚点存在
const config = readFileSync(configPath, "utf8");
const anchorIdx = config.indexOf(ANCHOR);
if (anchorIdx < 0) {
  console.error("未找到锚点注释，eslint.config.js 结构可能已被改动");
  process.exit(1);
}
const endIdx = config.indexOf("\n    ],", anchorIdx);
if (endIdx < 0) {
  console.error("未找到豁免 files 数组结束位置");
  process.exit(1);
}
const scanConfigText =
  config.slice(0, anchorIdx) + ANCHOR + "\n" + PLACEHOLDER + config.slice(endIdx);
writeFileSync(scanConfigPath, scanConfigText, "utf8");

const res = spawnSync(
  "npx",
  ["eslint", "src", "--config", "eslint.config.p66-scan.js", "--format", "json", "--no-error-on-unmatched-pattern"],
  {
    cwd: root,
    shell: process.platform === "win32",
    encoding: "utf8",
    maxBuffer: 512 * 1024 * 1024,
  }
);

// 无论成败都清理临时扫描 config
try {
  rmSync(scanConfigPath, { force: true });
} catch {
  /* 忽略清理失败 */
}

if (res.error) {
  console.error("eslint 执行失败:", res.error);
  process.exit(1);
}

// eslint --format json 输出可能混入警告行，取以 '[' 开头起的 JSON
const out = res.stdout ?? "";
const start = out.indexOf("[");
if (start < 0) {
  console.error("eslint 未输出 JSON。stderr:\n" + (res.stderr ?? ""));
  process.exit(1);
}
let results;
try {
  results = JSON.parse(out.slice(start));
} catch (e) {
  console.error("JSON 解析失败:", e.message);
  process.exit(1);
}

const hitFiles = [];
for (const r of results) {
  const has = r.messages.some(
    (m) => m.ruleId === "no-restricted-imports" && P66_MESSAGE_MARKS.some((k) => (m.message ?? "").includes(k))
  );
  if (has) hitFiles.push(r.filePath);
}

const rel = hitFiles
  .map((f) => path.relative(root, f).replaceAll("\\", "/"))
  .sort();

console.log(`命中 Phase 66 hook import 拦截的文件数: ${rel.length}`);
for (const f of rel) console.log("  " + f);

// ---- 校验模式：比对锚点段现存清单 ----
const anchorSection = config.slice(anchorIdx, endIdx);
const listed = [...anchorSection.matchAll(/"([^"]+)"/g)].map((m) => m[1]).sort();
const missing = rel.filter((f) => !listed.includes(f)); // 真实命中但清单未登记
const stale = listed.filter((f) => !rel.includes(f));   // 清单登记但已无命中
if (missing.length || stale.length) {
  console.error("\n[校验失败] 豁免清单与真实命中不一致:");
  for (const f of missing) console.error("  缺登记: " + f);
  for (const f of stale) console.error("  已失效: " + f);
  process.exit(1);
}
console.log(`\n[校验通过] 锚点清单 ${listed.length} 条 = 真实命中 ${rel.length} 条，逐条一致`);
process.exit(0);
