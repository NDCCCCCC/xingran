#!/usr/bin/env node
/**
 * Phase 86 W0 —— 业务页「动作级权限门控」覆盖扫描器（**零依赖**，只读脚本）
 *
 * 用途：把 ROADMAP 里「76/76 页面接入三态门控」这句人话，变成**脚本可判定**的事实。
 *
 * 判定口径（与 86-00-PLAN.md 的判据表同源）：
 *   1. 页面 = `src/pages` 下任意层级的 `index.tsx`（递归）；
 *      ⚠️ 注释里**不要**直接贴目录通配 glob —— 星号紧邻斜杠会提前闭合块注释；
 *   2. 页面归属文件 = 该页 `index.tsx` + 同目录 `.ts/.tsx` + 同目录**一级子目录**下的
 *      `.ts/.tsx`（排除 `__tests__/`、`*.test.*`、`*.spec.*`）。
 *      ⚠️ 一级子目录若自身含 `index.tsx`（子页面目录，如 `asset/reconciliation/exceptions/`）
 *      则**不并入** —— 否则父页会吞掉子页的代码，把只读父页误判成有写入口；
 *   3. 有写入口 = 命中写入口特征集（`ActionButtons` / `createDeleteAction` / `confirmDelete(` /
 *      `batchDelete=` / `onBatchDelete` / `ExcelImport` / `ExcelExport` / `BatchExportModal` /
 *      `handleDelete|handleCreate|handleAdd|handleSave|handleSubmit|handleSync|handleExecute|
 *      handleRestore|handleUpdateStatus|handleSetDefault|handleClone|handleEnable|handleMove|
 *      handleGenerate` / `saveFloorPlan` / `wallApi.save` / `doorApi.save`）；
 *   4. 已门控 = 归属文件合并文本命中 `useActionPermission` **或** `permission:` + `PERMISSION_CODES`
 *      （两者取或：只看 hook 会漏掉「直接传 `permission:` 字段但没调 hook」的页）；
 *   5. N/A 豁免 = `scripts/gating-exemptions.json` 白名单（每条必须带 category / reason / evidence）。
 *
 * ⚠️ `<Switch>` 的特殊口径（86-00 裁定 7）：
 *   只有「绑定**写 handler** 的状态切换 Switch」才算写入口。纯视图切换（如
 *   `operations/building-spaces-3d/index.tsx` 的 `onChange={setUseWebGL}`）与表单字段 Switch
 *   （`<Form.Item valuePropName="checked">` 内、无 onChange）都**不算**业务写。
 *   因此本脚本**不**把 `<Switch` 关键字自动判成写入口命中 —— 它只**打印 Switch 命中明细**
 *   （文件:行 + onChange 绑定的 handler），由人工在豁免 `reason` 中标注性质。
 *   若把视图切换 Switch 误判为写入口，会让 86-05 的 R3 强制改判并推翻前两波结论。
 *
 * 退出码：
 *   0 = 应门控页面全部已门控（绿）；1 = 存在未覆盖页面 或 脚本自身错误（红）。
 *
 * 86-05（Task 5.2）硬断言升级：把「未覆盖即 exit 1」升为三条同时成立的硬断言 ——
 *   ① 覆盖率恒等式：已门控（不含豁免）+ 豁免 == EXPECTED_PAGE_COUNT（76）；
 *   ② 已门控覆盖完整：应门控页面数 == 已门控页面数；
 *   ③ 未覆盖页面数 == 0。
 *   另：豁免与已门控**双向互斥**（豁免里挂着已门控页 → exit 1；未豁免且未门控 → exit 1），
 *   使白名单**不能靠无限扩张让脚本变绿**。
 *
 * ⚠️ **能力边界登记（86-05 Task 5.2 第 5 条 —— 这是登记，不是待办）**：
 *   本脚本是**页级**门控判定器 —— 它只判「该页归属文件是否命中写入口特征集」且
 *   「是否出现门控特征（useActionPermission / permission: + PERMISSION_CODES）」，
 *   **不判单个入口是否已门控** ⇒ **施工图遗漏的入口不会被脚本捕获**。
 *   **入口级覆盖靠施工图 + 人工 review + 抽样测试兜底**。
 *   实证：86-02 曾漏列 assets「下载模板」`index.tsx:639-654`、workstations
 *   「下载部门映射表」`index.tsx:834-836` 两处导出类写入口，已由 86-02 补齐。
 *
 * 用法：node scripts/check-gating-coverage.mjs
 */
import { existsSync, readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const feRoot = path.resolve(scriptDir, "..");
const pagesRoot = path.join(feRoot, "src", "pages");
const exemptionsFile = path.join(scriptDir, "gating-exemptions.json");

/** 口径断言：业务页面总数（漂移即 exit 1 —— 口径漂移必须重新裁决，不许悄悄改数字） */
const EXPECTED_PAGE_COUNT = 76;
/** 口径断言：N/A 豁免页数（增减须同批改本常量并在 86-STATE.md 说明理由） */
const EXPECTED_EXEMPTION_COUNT = 51;
/** 豁免分类白名单 */
const EXEMPTION_CATEGORIES = [
  "no-permission-code",
  "read-only",
  "non-route-page",
  "legacy-two-source",
];

/** 写入口特征集（PLAN 判据表全集） */
const WRITE_ENTRY_FEATURES = [
  { name: "ActionButtons", re: /\bActionButtons\b/ },
  { name: "createDeleteAction", re: /\bcreateDeleteAction\b/ },
  { name: "confirmDelete", re: /\bconfirmDelete\s*\(|\bConfirmDelete\b/ },
  { name: "batchDelete=", re: /\bbatchDelete\s*=/ },
  { name: "onBatchDelete", re: /\bonBatchDelete\b/ },
  { name: "ExcelImport", re: /\bExcelImport\b/ },
  { name: "ExcelExport", re: /\bExcelExport\b/ },
  { name: "BatchExportModal", re: /\bBatchExportModal\b/ },
  {
    name: "handleXxx",
    re: /\bhandle(?:Delete|Create|Add|Save|Submit|Sync|Execute|Restore|UpdateStatus|SetDefault|Clone|Enable|Move|Generate)\b/,
  },
  { name: "saveFloorPlan", re: /\bsaveFloorPlan\b/ },
  { name: "wallApi.save", re: /\bwallApi\.save\b/ },
  { name: "doorApi.save", re: /\bdoorApi\.save\b/ },
];

/** 已门控特征：`useActionPermission` 钩子 */
const GATING_HOOK_RE = /\buseActionPermission\b/;
/** 已门控特征：`permission:` 字段 + `PERMISSION_CODES` 单一真相源常量 */
const GATING_PERMISSION_RE = /permission\s*:[\s\S]{0,80}?PERMISSION_CODES/;
/** `<Switch` 命中（仅打印明细，不自动判为写入口 —— 见文件头裁定 7） */
const SWITCH_RE = /<Switch\b/g;

/** 相对前端根、正斜杠路径 */
function toPosix(p) {
  return p.split(path.sep).join("/");
}

/** 递归收集 `src/pages` 下任意层级的 `index.tsx` */
function collectPages(dir, acc) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      collectPages(full, acc);
    } else if (entry.name === "index.tsx") {
      acc.push(toPosix(path.relative(feRoot, full)));
    }
  }
  return acc;
}

/** 归属文件：index.tsx + 同目录 .ts/.tsx + 同目录一级子目录（子页面目录除外）的 .ts/.tsx */
function collectOwnedFiles(pageRel) {
  const dir = path.dirname(path.join(feRoot, pageRel));
  const out = [];
  const isCode = (n) => /\.tsx?$/.test(n) && !/[.](test|spec)[.]/.test(n);

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.isFile() && isCode(entry.name))
      out.push(toPosix(path.relative(feRoot, path.join(dir, entry.name))));
  }

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name === "__tests__") continue;
    const sub = path.join(dir, entry.name);
    // 一级子目录若自身是子页面目录（含 index.tsx），不并入父页 —— 否则父页会吞掉子页代码
    const isSubPage = readdirSync(sub, { withFileTypes: true }).some(
      (f) => f.isFile() && f.name === "index.tsx"
    );
    if (isSubPage) continue;
    for (const f of readdirSync(sub, { withFileTypes: true })) {
      if (f.isFile() && isCode(f.name))
        out.push(toPosix(path.relative(feRoot, path.join(sub, f.name))));
    }
  }
  return out.sort();
}

/** 提取 `<Switch` 命中明细：文件:行 + onChange 绑定的 handler 名（仅证据，不做判定） */
function collectSwitchHits(files) {
  const hits = [];
  for (const rel of files) {
    const lines = readFileSync(path.join(feRoot, rel), "utf8").split(/\r?\n/);
    lines.forEach((line, i) => {
      if (!/<Switch\b/.test(line)) return;
      const block = lines.slice(i, i + 20).join(" ");
      const m = block.match(/onChange=\{([\s\S]{0,400}?)\}/);
      const binding = m ? m[1].replace(/\s+/g, " ").trim() : "";
      hits.push({ file: rel, line: i + 1, binding, kind: classifySwitch(binding) });
    });
  }
  return hits;
}

/**
 * Switch 性质粗分类（**仅供人工判定的提示**，脚本不据此把页面判为有写入口）。
 *  - form-field  : 无 onChange → `<Form.Item valuePropName="checked">` 内的表单字段
 *  - view-toggle : 绑定 `setXxx` → 纯本地视图状态切换
 *  - local-state : 绑定 updateLayout / handleUpdate / onUpdateRow → 本地偏好或表单行更新
 *  - write-cand  : 绑定 handleXxx 形态 → **业务写候选**，必须门控
 */
function classifySwitch(binding) {
  if (!binding) return "form-field";
  if (/\bset[A-Z]/.test(binding)) return "view-toggle";
  if (/\b(updateLayout|handleUpdate|onUpdateRow)\b/.test(binding)) return "local-state";
  if (/\bhandle[A-Z]/.test(binding)) return "write-cand";
  return "unknown";
}

function main() {
  const errors = [];

  // ---------- 1. 页面清单 + 总数断言 ----------
  if (!existsSync(pagesRoot)) {
    console.error(`[错误] 找不到页面目录 ${pagesRoot}`);
    process.exit(1);
  }
  const pages = collectPages(pagesRoot, []).sort();

  console.log("=== Phase 86 W0 门控覆盖扫描 ===\n");
  console.log(`页面清单：${pages.length} 页（断言 EXPECTED_PAGE_COUNT = ${EXPECTED_PAGE_COUNT}）`);
  if (pages.length !== EXPECTED_PAGE_COUNT) {
    errors.push(
      `页面总数漂移：实测 ${pages.length}，断言 ${EXPECTED_PAGE_COUNT} —— 口径漂移必须重新裁决，` +
        `不许悄悄改数字（若确为口径变更，须同批改 EXPECTED_PAGE_COUNT 并在 86-STATE.md 说明理由）`
    );
  }

  // ---------- 2. 豁免清单读取 + 结构校验 ----------
  if (!existsSync(exemptionsFile)) {
    console.error(`[错误] 找不到豁免清单 ${exemptionsFile}`);
    process.exit(1);
  }
  let exemptions = [];
  try {
    const raw = JSON.parse(readFileSync(exemptionsFile, "utf8"));
    exemptions = Array.isArray(raw?.exemptions) ? raw.exemptions : [];
  } catch (e) {
    console.error(`[错误] 豁免清单 JSON 解析失败：${e.message}`);
    process.exit(1);
  }

  const pageSet = new Set(pages);
  const exemptByPage = new Map();
  for (const entry of exemptions) {
    const page = entry?.page;
    if (!page) {
      errors.push(`[豁免结构] 条目缺少 page：${JSON.stringify(entry)}`);
      continue;
    }
    if (!pageSet.has(page)) {
      errors.push(`[悬空豁免] ${page} 不在页面清单中（页面可能已被删除或改名）`);
      continue;
    }
    if (exemptByPage.has(page)) {
      errors.push(`[重复豁免] ${page} 在豁免清单中出现多次`);
      continue;
    }
    if (!EXEMPTION_CATEGORIES.includes(entry.category)) {
      errors.push(
        `[豁免分类] ${page} 的 category=${entry.category} 非法（只允许 ${EXEMPTION_CATEGORIES.join(" / ")}）`
      );
    }
    if (!entry.reason || !String(entry.reason).trim())
      errors.push(`[豁免理由] ${page} 的 reason 为空`);
    if (!entry.evidence || !String(entry.evidence).trim())
      errors.push(`[豁免证据] ${page} 的 evidence 为空`);
    exemptByPage.set(page, entry);
  }
  if (exemptByPage.size !== EXPECTED_EXEMPTION_COUNT) {
    errors.push(
      `豁免条数漂移：实测 ${exemptByPage.size}，断言 ${EXPECTED_EXEMPTION_COUNT} —— ` +
        `页数增减须同批改 EXPECTED_EXEMPTION_COUNT 并在 86-STATE.md 说明理由`
    );
  }

  // ---------- 3. 逐页判定（有写入口 / 已门控 / Switch 明细） ----------
  const gatedPages = new Set();
  const uncovered = [];
  const switchPages = [];

  for (const page of pages) {
    const files = collectOwnedFiles(page);
    const text = files.map((f) => readFileSync(path.join(feRoot, f), "utf8")).join("\n");

    const featureHits = WRITE_ENTRY_FEATURES.filter((f) => f.re.test(text)).map((f) => f.name);
    const hookHit = GATING_HOOK_RE.test(text);
    const permissionHit = GATING_PERMISSION_RE.test(text);
    const gated = hookHit || permissionHit;
    if (gated) gatedPages.add(page);

    const switches = collectSwitchHits(files);
    if (switches.length) switchPages.push({ page, switches });

    if (!exemptByPage.has(page) && !gated) {
      uncovered.push({ page, files, featureHits, switches });
    }
  }

  // 门控与豁免互斥
  for (const page of gatedPages) {
    if (exemptByPage.has(page)) {
      errors.push(`[门控/豁免冲突] ${page} 同时命中门控特征与豁免清单 —— 二者必须互斥`);
    }
  }

  // ---------- 4. 统计输出 ----------
  const byCategory = new Map();
  for (const e of exemptByPage.values()) {
    byCategory.set(e.category, (byCategory.get(e.category) ?? 0) + 1);
  }
  const shouldGate = pages.filter((p) => !exemptByPage.has(p));
  const covered = shouldGate.filter((p) => gatedPages.has(p));
  const coveredExempt = [...gatedPages].filter((p) => exemptByPage.has(p));

  console.log(
    `豁免清单：${exemptByPage.size} 条（` +
      EXEMPTION_CATEGORIES.map((c) => `${c} ${byCategory.get(c) ?? 0}`).join(" / ") +
      "）"
  );
  console.log(
    `应门控页面：${shouldGate.length}（= ${pages.length} 页面 − ${exemptByPage.size} 豁免）`
  );
  console.log(`已门控页面：${gatedPages.size}（${[...gatedPages].sort().join(", ") || "无"}）`);
  console.log(
    `已门控 ∩ 豁免：${coveredExempt.length}${coveredExempt.length ? `（${coveredExempt.join(", ")}）` : ""}`
  );
  console.log(`未覆盖页面：${uncovered.length}`);

  // ---------- 4b. 覆盖率硬断言（86-05 Task 5.2：把「软提示」升为「硬失败」）----------
  // 恒等式：已门控（不含豁免）+ 豁免 == EXPECTED_PAGE_COUNT
  // 三条同时成立才算覆盖率达成；任一不满足 → errors → exit 1
  const coveredOnly = shouldGate.filter((p) => gatedPages.has(p)).length;
  if (gatedPages.size !== EXPECTED_PAGE_COUNT - exemptByPage.size) {
    errors.push(
      `覆盖率恒等式不成立：已门控 ${gatedPages.size} + 豁免 ${exemptByPage.size} = ` +
        `${gatedPages.size + exemptByPage.size} ≠ ${EXPECTED_PAGE_COUNT}（断言 EXPECTED_PAGE_COUNT）`
    );
  }
  if (coveredOnly !== shouldGate.length) {
    errors.push(
      `已门控覆盖率不足：应门控 ${shouldGate.length} 页中仅 ${coveredOnly} 页命中门控特征`
    );
  }
  if (uncovered.length !== 0) {
    errors.push(`未覆盖页面 ${uncovered.length} 页 —— 应门控页面必须全部接入三态门控`);
  }
  console.log(
    `覆盖率恒等式：已门控 ${coveredOnly} + 豁免 ${exemptByPage.size} = ${coveredOnly + exemptByPage.size}` +
      `（断言 ${EXPECTED_PAGE_COUNT}）${coveredOnly + exemptByPage.size === EXPECTED_PAGE_COUNT ? " ✓" : " ✗"}`
  );

  // ---------- 5. 未覆盖清单（业务性未完成 —— 与「脚本自身错误」分区呈现）----------
  if (uncovered.length) {
    console.log(`\n=== 未覆盖页面清单（${uncovered.length} 页，需铺开三态门控）===`);
    for (const u of uncovered) {
      console.log(`  ✗ ${u.page}`);
      console.log(`      归属文件：${u.files.join(", ")}`);
      console.log(
        `      写入口特征：${u.featureHits.length ? u.featureHits.join(", ") : "（无 —— 需人工确认该页写入口形态）"}`
      );
      for (const s of u.switches) {
        console.log(
          `      Switch 命中：${s.file}:${s.line} onChange=${s.binding || "(无 —— 表单字段)"} ` +
            `[${s.kind}]${s.kind === "write-cand" ? " ← 业务写候选，必须门控" : ""}`
        );
      }
    }
  }

  // ---------- 6. Switch 人工判定台账（脚本不判定，仅列证据）----------
  if (switchPages.length) {
    console.log(
      `\n=== Switch 命中台账（${switchPages.length} 页 —— 需人工标注「视图切换 / 表单字段 / 业务写」）===`
    );
    for (const sp of switchPages) {
      for (const s of sp.switches) {
        console.log(`  · ${sp.page}`);
        console.log(
          `      ${s.file}:${s.line} onChange=${s.binding || "(无 —— 表单字段)"} [${s.kind}]`
        );
      }
    }
  }

  // ---------- 7. 豁免移除时限提示 ----------
  if (exemptByPage.size > 0) {
    console.log(
      `\n⚠ 豁免清单非空（${exemptByPage.size} 条）—— 仅限 Phase 86 铺开期间的窗口期。` +
        "各条豁免的 reason 已写明「等后端拆分 phase 回补」的时限；后端拆分 / 权限码回补落地后" +
        "必须逐条移除对应豁免，否则门控缺口将被永久掩盖。"
    );
  }

  // ---------- 8. 错误汇总 + 退出码 ----------
  if (errors.length > 0) {
    console.error(`\n=== 脚本自身错误（${errors.length} 项，结构性问题）===`);
    for (const e of errors) console.error(`  ✗ ${e}`);
  }

  if (errors.length > 0 || uncovered.length > 0) {
    console.error(
      `\n✗ 门控覆盖未达成：未覆盖 ${uncovered.length} 页 / 脚本错误 ${errors.length} 项 —— 退出码 1`
    );
    process.exit(1);
  }

  console.log("\n✓ 门控覆盖达成：应门控页面全部已命中三态门控特征（退出码 0）");
  process.exit(0);
}

main();
