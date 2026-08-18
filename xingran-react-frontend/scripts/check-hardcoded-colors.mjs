/**
 * scripts/check-hardcoded-colors.mjs
 *
 * QA-02 硬编码色扫描器 (Phase 66 · 防回归门)
 *
 * 模式:
 *   - node scripts/check-hardcoded-colors.mjs           报告模式 (exit 1 if hits)
 *   - node scripts/check-hardcoded-colors.mjs --fix     修复模式 (替换确定性映射)
 *
 * 扫描范围: src tree (ts, tsx, css)
 * 路径 allowlist:
 *   - src/utils/three/colors.ts (Phase 65 锁定, VIS-01 v1.23+)
 *   - src/design-system/tokens/ (真相源本体)
 *   - scripts/ (本扫描器自身)
 * 值 allowlist (modern-tag dark 文字浅化, 深底 ≈7-9:1):
 *   - #95de64 / #ff7875 / #ffc53d / #69c0ff
 *
 * 零依赖 (node 内置 fs/path), 替代色全部基于 xingranBrand / brand-spec.md 实测值
 */

import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = process.cwd();
const SRC = join(ROOT, "src");
const FIX_MODE = process.argv.includes("--fix");

// 路径 allowlist (相对 ROOT, 跳过扫描)
const PATH_ALLOW = [
  "src/utils/three/colors.ts",
  "src/design-system/tokens/",
  "scripts/",
];

// 值 allowlist (允许这些 hex / rgba 出现, 例如 modern-tag dark 浅化文字)
const VALUE_ALLOW_HEX = new Set(["#95de64", "#ff7875", "#ffc53d", "#69c0ff"]);

// rgba denylist (捕获 alpha 通道)
const RGBA_MAPS = [
  [/rgba?\(\s*79\s*,\s*70\s*,\s*229\s*,\s*([\d.]+)\s*\)/gi, "rgba(21, 96, 49, $1)"],
  [/rgba?\(\s*37\s*,\s*42\s*,\s*63\s*,\s*([\d.]+)\s*\)/gi, "rgba(15, 46, 27, $1)"],
  [/rgba?\(\s*24\s*,\s*144\s*,\s*255\s*,\s*([\d.]+)\s*\)/gi, "rgba(51, 122, 176, $1)"],
  [/rgba?\(\s*82\s*,\s*196\s*,\s*26\s*,\s*([\d.]+)\s*\)/gi, "rgba(45, 137, 73, $1)"],
  [/rgba?\(\s*250\s*,\s*173\s*,\s*20\s*,\s*([\d.]+)\s*\)/gi, "rgba(176, 122, 32, $1)"],
  [/rgba?\(\s*255\s*,\s*77\s*,\s*79\s*,\s*([\d.]+)\s*\)/gi, "rgba(186, 54, 48, $1)"],
  [/rgba?\(\s*140\s*,\s*140\s*,\s*140\s*,\s*([\d.]+)\s*\)/gi, "rgba(112, 112, 104, $1)"],
];

// hex denylist → 映射 (lowercase → lowercase)
const HEX_MAPS = {
  "#4f46e5": "#156031",
  "#f1f5f9": "#dbd7ce",
  "#0f172a": "#101010",
  "#1e293b": "#101010",
  "#334155": "#707068",
  "#475569": "#707068",
  "#64748b": "#707068",
  "#94a3b8": "#c2bdb2",
  "#cbd5e1": "#dbd7ce",
  "#e2e8f0": "#e9efeb",
  "#f8fafc": "#ffffff",
  "#1677ff": "#156031",
  "#3b82f6": "#156031",
  "#2563eb": "#14542e",
  "#50a3ba": "#598e5e",
  "#fad252": "#fef3c7",
  "#eac736": "#b07a20",
  "#d94e5d": "#ba3630",
  "#6366f1": "#156031",
  "#8b5cf6": "#c09058",
  "#22c55e": "#2d8949",
  "#f59e0b": "#b07a20",
  "#ef4444": "#ba3630",
  "#1890ff": "#337ab0",
  "#ff4d4f": "#ba3630",
  "#52c41a": "#2d8949",
  "#faad14": "#b07a20",
  "#8c8c8c": "#707068",
  "#bfbfbf": "#707068",
  "#d9d9d9": "#dbd7ce",
  "#f0f0f0": "#e9efeb",
};

/**
 * 递归列出 src/ 下 .ts/.tsx/.css 文件
 */
function walk(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      walk(full, acc);
    } else if (/\.(ts|tsx|css)$/.test(entry)) {
      acc.push(full);
    }
  }
  return acc;
}

/**
 * 路径是否在 allowlist 中
 */
function isPathAllowed(relPath) {
  return PATH_ALLOW.some((p) => relPath === p || relPath.startsWith(p));
}

/**
 * 在文本中查找所有 denylist 命中
 * 返回 { line, col, match, replacement } 数组
 */
function findHits(text) {
  const hits = [];
  // rgba 命中
  for (const [re, replacement] of RGBA_MAPS) {
    re.lastIndex = 0;
    let m;
    while ((m = re.exec(text)) !== null) {
      const before = text.slice(0, m.index);
      const line = (before.match(/\n/g) || []).length + 1;
      const col = m.index - (before.lastIndexOf("\n") + 1) + 1;
      hits.push({ line, col, match: m[0], replacement: replacement.replace("$1", m[1] || "") });
    }
  }
  // hex 命中
  const hexRe = /#[0-9a-fA-F]{6}\b/g;
  hexRe.lastIndex = 0;
  let m;
  while ((m = hexRe.exec(text)) !== null) {
    const lower = m[0].toLowerCase();
    if (VALUE_ALLOW_HEX.has(lower)) continue;
    if (!(lower in HEX_MAPS)) continue;
    const before = text.slice(0, m.index);
    const line = (before.match(/\n/g) || []).length + 1;
    const col = m.index - (before.lastIndexOf("\n") + 1) + 1;
    hits.push({ line, col, match: m[0], replacement: HEX_MAPS[lower] });
  }
  return hits;
}

const files = walk(SRC);
const SKIPPED = [];
const REMAINING = [];
let totalFixed = 0;

for (const file of files) {
  const rel = relative(ROOT, file).replace(/\\/g, "/");
  if (isPathAllowed(rel)) {
    SKIPPED.push(rel);
    continue;
  }
  const original = readFileSync(file, "utf-8");
  const hits = findHits(original);
  if (hits.length === 0) continue;

  if (FIX_MODE) {
    let updated = original;
    for (const [re, replacement] of RGBA_MAPS) {
      updated = updated.replace(re, (m, a) => replacement.replace("$1", a || ""));
    }
    for (const [orig, dest] of Object.entries(HEX_MAPS)) {
      const re = new RegExp(`${orig}\\b`, "gi");
      updated = updated.replace(re, dest);
    }
    writeFileSync(file, updated, "utf-8");
    totalFixed += hits.length;
  } else {
    REMAINING.push({ file: rel, hits });
  }
}

if (FIX_MODE) {
  console.log(`[fix] applied ${totalFixed} replacements across ${files.length - SKIPPED.length} scanned files`);
  console.log(`[skip] ${SKIPPED.length} paths allowlisted: ${SKIPPED.join(", ")}`);
  process.exit(0);
}

if (REMAINING.length === 0) {
  console.log(`[ok] no hardcoded colors found in ${files.length - SKIPPED.length} scanned files`);
  console.log(`[skip] ${SKIPPED.length} paths allowlisted: ${SKIPPED.join(", ")}`);
  process.exit(0);
}

console.error(`[fail] hardcoded colors found in ${REMAINING.length} files:`);
for (const { file, hits } of REMAINING) {
  for (const h of hits) {
    console.error(`  ${file}:${h.line}:${h.col}  ${h.match}  →  ${h.replacement}`);
  }
}
process.exit(1);
