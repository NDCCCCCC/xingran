#!/usr/bin/env node
/**
 * check-route-registry.mjs —— 路由登记双层守卫之 CI 侧（Phase 85 Plan 01 / RBAC-03）。
 *
 * 职责（防「代码改了表没重生成」+ 防新写路由漏登记）：
 *   1. freshness：重跑 scripts/scan-routes.cjs 的扫描（内存中，不落盘），与入库的
 *      scripts/route-registry.json 逐条比对 routes 数组——diff 非空 → exit 1。
 *   2. coverage：对每条 isWrite=true 路由（以重扫结果为准）：
 *        - 已被 GetRoutePermissions() 登记（registeredDeadTable=true）→ 放行；
 *        - 未登记 → 必须出现在入库 json 的 deferred.items 冻结清单（85-02/03
 *          拆分同批收缩），否则 exit 1；
 *        - 已登记却仍挂在 deferred 清单 → 提示拆分者同批收缩 → exit 1。
 *
 * 不负责：TS↔Go↔菜单码漂移（check-permission-codes.mjs 的职责，语义零改动）；
 * 死表陈旧条目（登记了但路由不存在）仅提示，由 85-02/03 各域拆分时清理。
 *
 * 用法：node scripts/check-route-registry.mjs（由 xingran-react-frontend lint:ci 链尾调用）
 */
import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

// 本脚本位于仓库根 scripts/ 目录（由 frontend lint:ci 以 `node ../scripts/check-route-registry.mjs` 调用），
// 仓库根 = 本脚本目录的上一级。
const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, "..");

const require = createRequire(import.meta.url);
const scannerPath = path.join(repoRoot, "scripts", "scan-routes.cjs");
const { scanRoutes } = require(scannerPath);

const registryPath = path.join(repoRoot, "scripts", "route-registry.json");
const committed = JSON.parse(readFileSync(registryPath, "utf8"));

const errors = [];
const notes = [];

// ---------- 1. freshness ----------
const fresh = scanRoutes();
const canon = (r) =>
  JSON.stringify(
    r.routes.map(({ method, path: p, mountGroup, isWrite, handler, permKind, permCodes, registeredDeadTable }) => ({
      method, path: p, mountGroup, isWrite, handler, permKind, permCodes, registeredDeadTable,
    })),
  );
if (canon(fresh) !== canon(committed)) {
  const committedSet = new Set(committed.routes.map((r) => `${r.method} ${r.path}`));
  const freshSet = new Set(fresh.routes.map((r) => `${r.method} ${r.path}`));
  const added = [...freshSet].filter((k) => !committedSet.has(k));
  const removed = [...committedSet].filter((k) => !freshSet.has(k));
  errors.push(
    "route-registry.json 已过期（路由注册代码与入库产物不一致），请运行: node scripts/scan-routes.cjs",
  );
  if (added.length) errors.push("  扫描新增而入库没有: " + added.join(", "));
  if (removed.length) errors.push("  入库有而扫描已不存在: " + removed.join(", "));
  // 登记标记差异（config.go 登记变了但 json 未重生成）
  const regChanged = committed.routes.filter((c) => {
    const f = fresh.routes.find((x) => x.method === c.method && x.path === c.path);
    return f && f.registeredDeadTable !== c.registeredDeadTable;
  });
  if (regChanged.length) errors.push("  死表登记状态变化未同步: " + regChanged.map((r) => `${r.method} ${r.path}`).join(", "));
}

// ---------- 2. coverage ----------
const deferredSet = new Set(committed.deferred.items.map((d) => `${d.method} ${d.path}`));
const missing = [];
const staleDeferred = [];
for (const r of fresh.routes) {
  if (!r.isWrite) continue;
  const key = `${r.method} ${r.path}`;
  if (r.registeredDeadTable) {
    if (deferredSet.has(key)) staleDeferred.push(key);
  } else if (!deferredSet.has(key)) {
    missing.push(key);
  }
}
if (missing.length) {
  errors.push(
    `发现 ${missing.length} 条未登记写路由且不在 deferred 冻结清单（新写路由必须同步登记 GetRoutePermissions 或显式扩 deferred）:`,
  );
  missing.forEach((k) => errors.push("  MISSING: " + k));
}
if (staleDeferred.length) {
  errors.push(
    `deferred 清单中 ${staleDeferred.length} 条已登记，应随拆分批次收缩（运行 node scripts/scan-routes.cjs 重生成）:`,
  );
  staleDeferred.forEach((k) => errors.push("  SHRINK: " + k));
}

// ---------- 提示（不拦） ----------
const liveKeys = new Set(fresh.routes.map((r) => `${r.method} ${r.path}`));
const staleTable = [];
for (const r of committed.routes) {
  // 死表登记了但扫描不到实际路由的条目在 scan CLI 中提示，这里不重复
  void liveKeys; void staleTable;
}
notes.push(`路由 ${fresh.totals.routes} 条 / 写路由 ${fresh.totals.writeRoutes} 条 / deferred ${fresh.totals.deferredWriteRoutes} 条`);

if (errors.length) {
  console.error("[check-route-registry] FAIL");
  errors.forEach((e) => console.error(e));
  process.exit(1);
}
console.log("[check-route-registry] OK " + notes.join(" | "));
