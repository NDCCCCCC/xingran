#!/usr/bin/env node
/**
 * Phase 65 — 存量豁免清单生成脚本（PLAN T2/T6/T8 复跑；P67 收敛后清偿）
 *
 * 跑 eslint --format json 全量分析 src，收集命中 Phase 65 新增 4 selector
 * （裸 Input / 裸 Select / Modal.confirm / Popconfirm）的文件全集，并把清单
 * 写入 eslint.config.js 的「Phase 65 存量豁免 override」锚点段
 * （`__P65_EXEMPTION_FILES__`）。文件按路径排序，输出稳定可 diff。
 *
 * 用法：node scripts/gen-p65-exemptions.mjs [--dry]
 *   --dry  仅打印清单，不写回 eslint.config.js
 */
import { spawnSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const dry = process.argv.includes("--dry");

// Phase 65 新增 4 selector 的 message 前缀（与 eslint.config.js 中的文案保持同步）
const P65_MESSAGE_MARKS = [
  "禁止直接使用 antd <Input>",
  "禁止直接使用 antd <Select>",
  "禁止直接调用 Modal.confirm",
  "禁止直接使用 Popconfirm",
];

const res = spawnSync("npx", ["eslint", "src", "--format", "json", "--no-error-on-unmatched-pattern"], {
  cwd: root,
  shell: process.platform === "win32",
  encoding: "utf8",
  maxBuffer: 512 * 1024 * 1024,
});

if (res.error) {
  console.error("eslint 执行失败:", res.error);
  process.exit(1);
}

// eslint --format json 输出可能混入警告行，取最后一个以 '[' 开头的行起的 JSON
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
    (m) => m.ruleId === "no-restricted-syntax" && P65_MESSAGE_MARKS.some((k) => (m.message ?? "").includes(k))
  );
  if (has) hitFiles.push(r.filePath);
}

const rel = hitFiles
  .map((f) => path.relative(root, f).replaceAll("\\", "/"))
  .sort();

console.log(`命中 Phase 65 selector 的文件数: ${rel.length}`);
for (const f of rel) console.log("  " + f);

if (dry) process.exit(0);

const configPath = path.join(root, "eslint.config.js");
const config = readFileSync(configPath, "utf8");
const anchor = "      // __P65_EXEMPTION_FILES__（脚本替换锚点）";
const anchorIdx = config.indexOf(anchor);
if (anchorIdx < 0) {
  console.error("未找到锚点注释，eslint.config.js 结构可能已被改动");
  process.exit(1);
}
// 数组结束于锚点之后的第一个 "\n    ],"
const endIdx = config.indexOf("\n    ],", anchorIdx);
if (endIdx < 0) {
  console.error("未找到豁免 files 数组结束位置");
  process.exit(1);
}
const listing = rel.length
  ? rel.map((f) => `      "${f}",`).join("\n")
  : "      // （当前无存量命中）\n      \"src/__p65_no_exemptions__/**\",";
const updated = config.slice(0, anchorIdx) + listing + config.slice(endIdx);
writeFileSync(configPath, updated, "utf8");
console.log(`\n已写回 ${path.relative(root, configPath)}（${rel.length} 个文件）`);
