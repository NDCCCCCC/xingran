#!/usr/bin/env node
/**
 * Phase 68.1 T7 — 豁免清单 ↔ 结构化理由 一一对应校验（只读脚本，不改任何配置）
 *
 * 背景（UI-SPEC §4 / Q3 裁定）：
 * - 豁免"机制"不动：still 由 eslint.config.js 的 __P65_EXEMPTION_FILES__ /
 *   __P66_EXEMPTION_FILES__ 两个锚点数组 + 生成脚本写回；本脚本**不写** eslint.config.js。
 * - 但每条剩余豁免必须有一条可核查理由（scripts/exemption-reasons.json），
 *   本脚本强制一一对应，防止"静默新增豁免"与"收缩后理由未同步"。
 *
 * 校验项：
 *  1. 清单文件集合 == 理由文件集合（缺一 / 多一均报错）
 *  2. 同一 (layer, file) 不得重复登记
 *  3. entry.layer 必须与该文件实际所在段一致
 *  4. userVisible=true 的条目 difference 必须非空
 *  5. 每条 reason 必须非空
 *
 * 用法：node scripts/check-exemption-reasons.mjs
 * 退出码：0 = 全部通过；1 = 存在差异（打印清单）
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const FE = path.resolve(here, "..");
const CONFIG = path.join(FE, "eslint.config.js");
const REASONS = path.join(FE, "scripts", "exemption-reasons.json");
const SCANNER = path.join(FE, "scripts", "check-hardcoded-colors.mjs");

const ANCHORS = {
  P65: "__P65_EXEMPTION_FILES__",
  P66: "__P66_EXEMPTION_FILES__",
};

// Phase 89 (W3) layer:"CSS" 对账基准：
// check-hardcoded-colors.mjs 的 PATH_ALLOW 中属于「结构性基础白名单」的条目（不需要理由登记）。
// 数组中减去本集合后的条目 = 「理由型豁免」，必须与 exemption-reasons.json 的 layer:"CSS" 一一对应。
const BASE_PATH_ALLOW = ["src/utils/three/colors.ts", "src/design-system/tokens/", "scripts/"];

/** 从扫描器源码提取 PATH_ALLOW 数组内的字符串字面量 */
function parseScannerPathAllow(scannerText) {
  const start = scannerText.indexOf("const PATH_ALLOW");
  if (start < 0)
    throw new Error(
      "未找到 check-hardcoded-colors.mjs 的 PATH_ALLOW（扫描器结构已变，校验脚本需同步）"
    );
  const end = scannerText.indexOf("];", start);
  if (end < 0) throw new Error("PATH_ALLOW 数组结束标记未找到");
  const segment = scannerText.slice(start, end);
  const items = [];
  const re = /"(src\/[^"]+)"/g;
  let m;
  while ((m = re.exec(segment)) !== null) items.push(m[1]);
  return items;
}

/** 从 eslint.config.js 的锚点段解析文件路径数组 */
function parseExemptionFiles(configText, anchor) {
  const start = configText.indexOf(anchor);
  if (start < 0)
    throw new Error(`未找到锚点 ${anchor}（eslint.config.js 结构已变，校验脚本需同步）`);
  const end = configText.indexOf("\n    ],", start);
  if (end < 0) throw new Error(`锚点 ${anchor} 的数组结束标记未找到`);
  const segment = configText.slice(start, end);
  const files = [];
  const re = /"(src\/[^"]+)"/g;
  let m;
  while ((m = re.exec(segment)) !== null) files.push(m[1]);
  return files;
}

function main() {
  const errors = [];

  if (!fs.existsSync(CONFIG)) {
    console.error(`[FAIL] 找不到 ${CONFIG}`);
    process.exit(1);
  }
  if (!fs.existsSync(REASONS)) {
    console.error(`[FAIL] 找不到 ${REASONS}（--phase 68.1 T7 要求：每条剩余豁免都要有一条理由）`);
    process.exit(1);
  }

  const configText = fs.readFileSync(CONFIG, "utf8");
  const json = JSON.parse(fs.readFileSync(REASONS, "utf8"));
  const entries = Array.isArray(json.entries) ? json.entries : [];

  if (!entries.length)
    errors.push("理由清单 entries 为空（若确实已无豁免，请同时清空 eslint 锚点段并保留本脚本）");

  // 1) eslint 侧集合
  const configSet = new Set();
  for (const [layer, anchor] of Object.entries(ANCHORS)) {
    for (const f of parseExemptionFiles(configText, anchor)) {
      const key = `${layer}|${f}`;
      if (configSet.has(key)) errors.push(`[重复] eslint.config.js ${layer} 段重复列出：${f}`);
      configSet.add(key);
    }
  }

  // 2) 理由侧集合 + 条目自身校验
  const reasonSet = new Set();
  for (const e of entries) {
    const { file, layer, dimension, reason, difference } = e ?? {};
    if (!file || !layer) {
      errors.push(`[结构] 条目缺少 file 或 layer：${JSON.stringify(e)}`);
      continue;
    }
    if (!["P65", "P66", "CSS"].includes(layer))
      errors.push(`[结构] ${file} 的 layer=${layer} 非法（只允许 P65 / P66 / CSS）`);
    if (layer === "CSS" && !(dimension || "").startsWith("hex:")) {
      errors.push(`[结构] ${file} 的 layer=CSS 条目 dimension 必须以 hex: 开头`);
    }
    const key = `${layer}|${file}`;
    if (layer !== "CSS") {
      if (reasonSet.has(key)) errors.push(`[重复] 理由清单重复登记：${layer} ${file}`);
      reasonSet.add(key);
    }

    if (!reason || !String(reason).trim()) errors.push(`[理由] ${layer} ${file} 的 reason 为空`);
    if (!dimension || !String(dimension).trim())
      errors.push(`[维度] ${layer} ${file} 的 dimension 为空`);
    if (e.userVisible === true && (!difference || !String(difference).trim())) {
      errors.push(
        `[差异] ${layer} ${file} userVisible=true 但 difference 为空（UI-SPEC §4.3 要求登记用户可感知差异）`
      );
    }
    if (typeof e.userVisible !== "boolean") {
      errors.push(`[结构] ${layer} ${file} 的 userVisible 必须是布尔值`);
    }
  }

  // 3) 集合相等（缺一 / 多一）—— P65/P66 对 eslint 锚点；CSS 对扫描器 PATH_ALLOW
  const onlyConfig = [...configSet].filter((k) => !reasonSet.has(k));
  const onlyReasons = [...reasonSet].filter((k) => !configSet.has(k));
  for (const k of onlyConfig) errors.push(`[缺理由] 仍被豁免但理由清单无对应条目：${k}`);
  for (const k of onlyReasons)
    errors.push(`[多理由] 理由清单有条目但已不在豁免清单（已收缩，应删除）：${k}`);

  // 3b) Phase 89：layer:"CSS" ↔ 扫描器 PATH_ALLOW（双向）
  const cssEntries = entries.filter((e) => e?.layer === "CSS");
  const scannerItems = parseScannerPathAllow(fs.readFileSync(SCANNER, "utf8"));
  const scannerExemptions = scannerItems.filter((p) => !BASE_PATH_ALLOW.includes(p));
  const cssReasonSet = new Set(cssEntries.map((e) => e.file));
  const scannerSet = new Set(scannerExemptions);
  for (const f of cssReasonSet)
    if (!scannerSet.has(f))
      errors.push(
        `[CSS缺放行] layer:"CSS" 条目未加入 check-hardcoded-colors.mjs 的 PATH_ALLOW：${f}`
      );
  for (const f of scannerSet)
    if (!cssReasonSet.has(f))
      errors.push(`[CSS缺理由] 扫描器 PATH_ALLOW 有理由型豁免但理由清单无 layer:"CSS" 条目：${f}`);

  const p65 = [...configSet].filter((k) => k.startsWith("P65|")).length;
  const p66 = [...configSet].filter((k) => k.startsWith("P66|")).length;

  if (errors.length) {
    console.error(`\n[FAIL] 豁免清单 ↔ 理由校验失败，共 ${errors.length} 项：\n`);
    for (const e of errors) console.error("  - " + e);
    console.error(
      `\n当前 eslint 豁免：P65 ${p65} / P66 ${p66}（合计 ${configSet.size}）；理由清单 ${reasonSet.size} 条`
    );
    process.exit(1);
  }

  console.log(`[OK] 豁免清单 ↔ 理由一一对应`);
  console.log(
    `     P65 ${p65} / P66 ${p66}（合计 ${configSet.size} 条），理由清单 ${entries.length} 条`
  );
  console.log(
    `     userVisible=true ${entries.filter((e) => e.userVisible === true).length} 条（均登记了 difference）`
  );
  process.exit(0);
}

main();
