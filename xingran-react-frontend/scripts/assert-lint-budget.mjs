#!/usr/bin/env node
/**
 * Phase 74（HYG-04/SC2）：lint 预算脚本断言——errors ≤ 17（Phase 74 清偿后终态，
 * 原 SC 口径 ≤ 83 基线已被本轮清偿大幅收窄）+ no-unused-vars / no-console = 0
 * + no-explicit-any < 500（HYG-03 软目标，实际已 0）。
 *
 * 用法：node scripts/assert-lint-budget.mjs [--json <eslint-json-path>]
 *   不带 --json 时自行跑 `npx eslint . --format json`。
 * 任何一项超标 → process.exit(1)（供 CI/本地门禁使用，不靠人眼）。
 */
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const BUDGET = {
  maxErrors: 0, // Phase 82（HYG-02）清零：17 → 0
  maxWarnings: 0, // Phase 82（HYG-02/03）清零：34 → 0（与 package.json lint --max-warnings 0 同口径）
  zeroRules: ["@typescript-eslint/no-unused-vars", "no-console"],
  maxAny: 500,
};

const jsonArg = process.argv.indexOf("--json");
let results;
if (jsonArg > 0 && process.argv[jsonArg + 1]) {
  results = JSON.parse(readFileSync(process.argv[jsonArg + 1], "utf8"));
} else {
  const res = spawnSync("npx", ["eslint", ".", "--format", "json"], {
    cwd: root,
    shell: process.platform === "win32",
    encoding: "utf8",
    maxBuffer: 512 * 1024 * 1024,
  });
  const out = res.stdout ?? "";
  const start = out.indexOf("[");
  if (start < 0) {
    console.error("eslint 未输出 JSON。stderr:\n" + (res.stderr ?? ""));
    process.exit(1);
  }
  results = JSON.parse(out.slice(start));
}

let errors = 0;
let warnings = 0;
const byRule = {};
for (const f of results) {
  errors += f.errorCount;
  warnings += f.warningCount;
  for (const m of f.messages) {
    byRule[m.ruleId ?? "(unused-disable)"] = (byRule[m.ruleId ?? "(unused-disable)"] ?? 0) + 1;
  }
}

const failures = [];
if (errors > BUDGET.maxErrors) failures.push(`errors ${errors} > 预算 ${BUDGET.maxErrors}`);
if (warnings > BUDGET.maxWarnings)
  failures.push(`warnings ${warnings} > 预算 ${BUDGET.maxWarnings}`);
for (const rule of BUDGET.zeroRules) {
  const n = byRule[rule] ?? 0;
  if (n !== 0) failures.push(`${rule} ${n} ≠ 0（HYG-02 硬承诺）`);
}
const anyCount = byRule["@typescript-eslint/no-explicit-any"] ?? 0;
if (anyCount >= BUDGET.maxAny)
  failures.push(`no-explicit-any ${anyCount} ≥ ${BUDGET.maxAny}（HYG-03 软目标）`);

console.log("=== lint 预算断言（Phase 74）===");
console.log(`errors:    ${errors} / ≤ ${BUDGET.maxErrors}`);
console.log(`warnings:  ${warnings} / ≤ ${BUDGET.maxWarnings}`);
console.log(`no-unused-vars: ${byRule["@typescript-eslint/no-unused-vars"] ?? 0}（要求 0）`);
console.log(`no-console: ${byRule["no-console"] ?? 0}（要求 0）`);
console.log(`no-explicit-any: ${anyCount}（< ${BUDGET.maxAny}）`);
if (failures.length) {
  console.error("\n[预算超标]");
  for (const f of failures) console.error("  - " + f);
  process.exit(1);
}
console.log("\n[通过] 全部预算项达标");
