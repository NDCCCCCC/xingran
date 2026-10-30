#!/usr/bin/env node
/**
 * Phase 65 — 存量豁免清单生成脚本（PLAN T2/T6/T8 复跑；P67 收敛后清偿）
 *
 * 跑 eslint --format json 全量分析 src，收集命中 Phase 65 新增 4 selector
 * （裸 Input / 裸 Select / Modal.confirm / Popconfirm）的文件全集，并把清单
 * 写入 eslint.config.js 的「Phase 65 存量豁免 override」锚点段
 * （`__P65_EXEMPTION_FILES__`）。文件按路径排序，输出稳定可 diff。
 *
 * 关键设计（自清空悖论规避）：存量豁免清单生效时 eslint 不再报这批文件 →
 * 直接扫 src 永远命中 0，复跑会把清单清空。因此扫描使用「禁用存量豁免」的
 * 临时 config（eslint.config.p65-scan.js：锚点段文件清单替换为占位 glob），
 * 分析完成后删除临时文件，再按真实命中集写回正式 config 的锚点段。
 *
 * 用法：node scripts/gen-p65-exemptions.mjs [--dry]
 *   --dry  仅打印清单，不写回 eslint.config.js
 */
import { spawnSync } from "node:child_process";
import { readFileSync, writeFileSync, rmSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const dry = process.argv.includes("--dry");
const configPath = path.join(root, "eslint.config.js");
const scanConfigPath = path.join(root, "eslint.config.p65-scan.js");
const ANCHOR = "      // __P65_EXEMPTION_FILES__（脚本替换锚点）";
const PLACEHOLDER = '      "src/__p65_no_exemptions__/**",';

// Phase 65 新增 selector 的 message 前缀（与 eslint.config.js 中的文案保持同步；
// Phase 70 追加 TextArea 双形态拦截的前缀）
const P65_MESSAGE_MARKS = [
  "禁止直接使用 antd <Input>",
  "禁止直接使用 antd <Select>",
  "禁止直接调用 Modal.confirm",
  "禁止直接使用 Popconfirm",
  "禁止直接使用 antd 多行输入框",
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
  [
    "eslint",
    "src",
    "--config",
    "eslint.config.p65-scan.js",
    "--format",
    "json",
    "--no-error-on-unmatched-pattern",
  ],
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
    (m) =>
      m.ruleId === "no-restricted-syntax" &&
      P65_MESSAGE_MARKS.some((k) => (m.message ?? "").includes(k))
  );
  if (has) hitFiles.push(r.filePath);
}

const rel = hitFiles.map((f) => path.relative(root, f).replaceAll("\\", "/")).sort();

console.log(`命中 Phase 65 selector 的文件数: ${rel.length}`);
for (const f of rel) console.log("  " + f);

if (dry) process.exit(0);

// 写回正式 config：保留锚点行本身，保证脚本可重复执行（T7 迁移后复跑收缩）
const listing = rel.length
  ? rel.map((f) => `      "${f}",`).join("\n")
  : "      // （当前无存量命中）\n" + PLACEHOLDER;
const updated = config.slice(0, anchorIdx) + ANCHOR + "\n" + listing + config.slice(endIdx);
writeFileSync(configPath, updated, "utf8");
console.log(`\n已写回 ${path.relative(root, configPath)}（${rel.length} 个文件）`);
