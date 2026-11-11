#!/usr/bin/env node
/**
 * v1.24 PERM-06 —— 「模块 × 操作 → 权限码」映射表一致性校验（**双向**）。
 *
 * 单一真相源：`xingran-react-frontend/src/constants/permissionCodes.ts`
 *
 * 对照的三份声明：
 *   1. 前端映射表   src/constants/permissionCodes.ts（PERMISSION_CODES）
 *   2. 后端常量     pkg/permission/config.go（PermissionCode = "..."）
 *   3. 菜单种子     internal/core/db/migrations/** 的 sys_menu.perms 声明
 *
 * 报出的四类漂移（任一命中 → exit 1）：
 *   a. TS 声明了，但后端常量没有          → 前端永远拿不到该权限
 *   b. 后端常量有，但 TS 没声明          → 前端漏了按钮门控
 *   c. TS 声明了，但 sys_menu 未声明      → 该权限永远不会被授予（菜单无授权来源）
 *   d. sys_menu 声明了 F 型按钮，但 TS 没声明 → 前端漏了按钮门控
 *
 * 作用域口径：每个 TS 模块的权限码共享同一前缀（`system:user` → `system:user:list` …）。
 * 前缀本身（`system:user`）是**页面本体**权限（C 型菜单），不是操作码，
 * 不参与比对（比对作用域 = 前缀 + ":"）。
 *
 * 菜单侧提取口径：
 *   - `INSERT INTO sys_menu (...) VALUES (...)` → 按列位置解析，取 menu_type='F' 的行（精确）
 *   - `INSERT INTO sys_menu (...) SELECT ...`  → 取语句内所有字符串字面量（类型未知，
 *     仅用于 c 方向的存在性判断，不参与 d 方向，避免把 C 型页面权限误报成漏映射）
 *   - Go migration 里的权限字符串字面量      → 同上，仅用于 c 方向
 *   - `archive/` 目录（历史归档，不再执行）→ 不扫描
 *
 * 全量对账（feat/frontend-url-cross-check · task-5，修复三类作用域盲区）：
 *   盲区1（已修复）：旧版作用域仅由 TS 模块前缀驱动——凡 permissionCodes.ts 无对应
 *     模块前缀的域（ops:rpa、monitor、vdi、asset:reconciliation、ops:duty 等）四向检查
 *     全部静默跳过。现新增「全集驱动」逐码对账：全集 = TS 码 ∪ config.go 常量 ∪
 *     seed F 型 perms ∪ 路由挂载操作码，对**不属于任何 TS 模块作用域**的码做四向
 *     存在性检查（后端常量 / TS 声明 / sys_menu.perms / 路由挂载）；TS 作用域内的码
 *     仍由下方模块循环的 a-d 检查覆盖，二者互补、不重复报错。
 *   盲区2（已修复）：新增路由挂载层扫描——读取 scripts/route-registry.json（由
 *     scripts/scan-routes.cjs 生成，含每条路由的 permCodes）；路由使用的操作码必须
 *     ∈ 已定义集合（TS ∪ Go 常量 ∪ 菜单种子），否则报错；config.go 定义但
 *     「路由 ∪ 菜单种子」双零的码报「孤儿码」警告（warning 级，不阻断）。
 *     C 型页面本体码（恰好等于某 TS 模块前缀的路由码，如 system:user）沿用旧口径
 *     不参与操作码比对——它们由菜单种子 C 型声明覆盖。
 *   盲区3（TODO，不实现）：「前端 API 调用 ↔ 权限门控」维度（页面零门控检测）与
 *     scripts/check-gating-coverage.mjs 的页级门控覆盖职责重叠，由该脚本负责，
 *     本脚本不重复建设。
 *   存量豁免：修复后暴露的历史漂移登记在
 *     scripts/permission-drift-exemptions.json，每条必须带非空 reason 且
 *     grandfathered=true（schema 校验强制）——豁免仅限存量，新代码不允许进豁免。
 *
 * 用法：node scripts/check-permission-codes.mjs
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(scriptDir, "..");
const repoRoot = path.resolve(frontendRoot, "..");

const TS_FILE = path.join(frontendRoot, "src", "constants", "permissionCodes.ts");
const GO_CONFIG = path.join(repoRoot, "pkg", "permission", "config.go");
const MIGRATIONS_DIR = path.join(repoRoot, "internal", "core", "db", "migrations");
const ROUTE_REGISTRY = path.join(repoRoot, "scripts", "route-registry.json");
const DRIFT_EXEMPTIONS_FILE = path.join(scriptDir, "permission-drift-exemptions.json");

/** 权限码形态：至少两段、段内小写字母数字下划线连字符（放宽支持连字符 - 以适配 ops:floor-plan:edit 等码） */
const PERM_LITERAL = /^[a-z][a-z0-9_-]*(?::[A-Za-z0-9_-]+)+$/;

/**
 * 挂起集合（Phase 87 · OQ-6 · 用户裁定 2026-10-30）——
 * 第二批 7 个动作码已于 87-05 经用户确认父菜单为「楼层管理」并通过 Migration 227 完成补播。
 * 本集合与 PENDING_SEED_WHITELIST 已完全归零，进入规则 B「白名单必须归零」严格态。
 */
const PENDING_SEED_HOLD_CODES = [];
const PENDING_SEED_HOLD_SET = new Set(PENDING_SEED_HOLD_CODES);

/**
 * Phase 87 动作码待播种白名单 —— **已彻底归零**。
 * 第一批 65 条（Migration 226）+ 第二批 7 条（Migration 227）已全部完成 F 型按钮播种。
 */
const PENDING_SEED_WHITELIST = [];

const PENDING_SEED_SET = new Set(PENDING_SEED_WHITELIST.map((w) => w.code));

// ==================== TS 侧解析 ====================

/**
 * 解析 PERMISSION_CODES 常量 → [{ module, entries: [{ key, code }] }]
 */
function parseTsPermissionCodes(text) {
  const anchor = text.indexOf("PERMISSION_CODES");
  if (anchor < 0) throw new Error(`未在 ${TS_FILE} 找到 PERMISSION_CODES`);
  const open = text.indexOf("{", anchor);
  if (open < 0) throw new Error("PERMISSION_CODES 缺少对象字面量");

  let depth = 0;
  let end = -1;
  for (let i = open; i < text.length; i++) {
    const ch = text[i];
    if (ch === "{") depth++;
    else if (ch === "}") {
      depth--;
      if (depth === 0) {
        end = i;
        break;
      }
    }
  }
  if (end < 0) throw new Error("PERMISSION_CODES 对象字面量未闭合");

  const body = text.slice(open + 1, end);
  const modules = [];
  const moduleRe = /([A-Za-z_$][\w$]*)\s*:\s*\{([^{}]*)\}/g;
  let m;
  while ((m = moduleRe.exec(body)) !== null) {
    const entries = [];
    const entryRe = /([A-Za-z_$][\w$]*)\s*:\s*["']([^"']+)["']/g;
    let e;
    while ((e = entryRe.exec(m[2])) !== null) {
      entries.push({ key: e[1], code: e[2] });
    }
    modules.push({ module: m[1], entries });
  }
  if (modules.length === 0) throw new Error("PERMISSION_CODES 未解析出任何模块");
  return modules;
}

// ==================== Go 侧解析 ====================

/** 解析 pkg/permission/config.go 的 PermissionCode 常量 → Set<string> */
function parseGoPermissionCodes(text) {
  const codes = new Set();
  const re = /PermissionCode\s*=\s*"([^"]+)"/g;
  let m;
  while ((m = re.exec(text)) !== null) {
    codes.add(m[1]);
  }
  return codes;
}

// ==================== 菜单种子解析 ====================

/**
 * 拆分 SQL 顶层逗号（尊重单引号字符串与 '' 转义）。
 * 返回原始片段数组。
 */
function splitSqlTopLevel(text, separator = ",") {
  const parts = [];
  let current = "";
  let inString = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (inString) {
      current += ch;
      if (ch === "'") {
        if (text[i + 1] === "'") {
          current += "'";
          i++;
        } else {
          inString = false;
        }
      }
      continue;
    }
    if (ch === "'") {
      inString = true;
      current += ch;
      continue;
    }
    if (ch === separator) {
      parts.push(current);
      current = "";
      continue;
    }
    current += ch;
  }
  parts.push(current);
  return parts;
}

/** 从 SQL 片段中取出单引号字符串字面量的内容（非字面量返回 null） */
function sqlLiteralValue(raw) {
  const trimmed = raw.trim();
  if (!trimmed.startsWith("'") || !trimmed.endsWith("'") || trimmed.length < 2) return null;
  return trimmed.slice(1, -1).replace(/''/g, "'");
}

/** 提取 VALUES 形态的元组列表 */
function extractSqlTuples(text) {
  const tuples = [];
  let depth = 0;
  let current = "";
  let inString = false;
  let started = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (inString) {
      current += ch;
      if (ch === "'") {
        if (text[i + 1] === "'") {
          current += "'";
          i++;
        } else {
          inString = false;
        }
      }
      continue;
    }
    if (ch === "'") {
      inString = true;
      current += ch;
      continue;
    }
    if (ch === "(") {
      depth++;
      if (depth === 1) {
        current = "";
        started = true;
        continue;
      }
    } else if (ch === ")") {
      depth--;
      if (depth === 0) {
        tuples.push(current);
        current = "";
        started = false;
        continue;
      }
    }
    if (depth >= 1) current += ch;
  }
  void started;
  return tuples;
}

/**
 * 解析 SQL 文件中的 sys_menu 声明。
 * @returns {{ fType: Set<string>, untyped: Set<string> }}
 */
function parseSqlMenuPerms(text) {
  const fType = new Set();
  const untyped = new Set();

  const stmtRe = /INSERT\s+INTO\s+"?sys_menu"?\s*\(([\s\S]*?)\)\s*(VALUES|SELECT)\s*([\s\S]*?);/gi;
  let stmt;
  while ((stmt = stmtRe.exec(text)) !== null) {
    const columns = splitSqlTopLevel(stmt[1]).map((c) =>
      c.trim().replace(/^"|"$/g, "").toLowerCase()
    );
    const permsIdx = columns.indexOf("perms");
    const typeIdx = columns.indexOf("menu_type");
    const body = stmt[3];
    const kind = stmt[2].toUpperCase();

    if (kind === "VALUES") {
      for (const tuple of extractSqlTuples(body)) {
        const cells = splitSqlTopLevel(tuple).map(sqlLiteralValue);
        const perms = permsIdx >= 0 ? cells[permsIdx] : null;
        if (!perms || !PERM_LITERAL.test(perms)) continue;
        const menuType = typeIdx >= 0 ? cells[typeIdx] : null;
        if (menuType === "F") fType.add(perms);
        else untyped.add(perms);
      }
    } else {
      // SELECT 形态：类型未知，只用于「存在性」判断
      for (const cell of splitSqlTopLevel(body)) {
        const value = sqlLiteralValue(cell);
        if (value && PERM_LITERAL.test(value)) untyped.add(value);
      }
    }
  }

  return { fType, untyped };
}

/** 递归收集 migrations 目录下的 .go / .sql 文件（跳过 archive/） */
function collectMigrationFiles(dir) {
  const files = [];
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return files;
  }
  for (const entry of entries) {
    if (entry === "archive") continue;
    const full = path.join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      files.push(...collectMigrationFiles(full));
    } else if (entry.endsWith(".sql") || entry.endsWith(".go")) {
      files.push(full);
    }
  }
  return files;
}

// ==================== 路由挂载层解析（盲区2） ====================

/**
 * 从 scripts/route-registry.json 提取路由实际挂载的权限码。
 * @param {{ routes?: Array<{ permCodes?: string[] }> }} registry
 * @returns Set<string> 仅收符合 PERM_LITERAL 形态的操作码
 */
function parseRoutePermCodes(registry) {
  const codes = new Set();
  const routes = Array.isArray(registry?.routes) ? registry.routes : [];
  for (const route of routes) {
    for (const code of route.permCodes ?? []) {
      if (PERM_LITERAL.test(code)) codes.add(code);
    }
  }
  return codes;
}

/**
 * 加载存量漂移豁免清单（schema 强校验）。
 * 每条必须：code 非空、reason 非空、grandfathered === true。
 * 违反任一即抛错（exit 1）——新代码不允许进豁免（task-5 裁定）。
 */
function loadDriftExemptions() {
  const byCode = new Map();
  try {
    const raw = JSON.parse(readFileSync(DRIFT_EXEMPTIONS_FILE, "utf8"));
    const entries = Array.isArray(raw?.exemptions) ? raw.exemptions : [];
    for (const entry of entries) {
      if (!entry?.code || !String(entry.code).trim()) {
        throw new Error(`豁免条目缺少 code：${JSON.stringify(entry)}`);
      }
      if (!entry.reason || !String(entry.reason).trim()) {
        throw new Error(`豁免条目 ${entry.code} 缺少非空 reason`);
      }
      if (entry.grandfathered !== true) {
        throw new Error(
          `豁免条目 ${entry.code} 缺少 grandfathered:true —— 豁免仅限存量漂移，新代码不允许登记`
        );
      }
      if (byCode.has(entry.code)) {
        throw new Error(`豁免条目重复：${entry.code}`);
      }
      byCode.set(entry.code, entry);
    }
  } catch (e) {
    if (e instanceof SyntaxError) {
      throw new Error(`豁免清单 JSON 解析失败：${e.message}`);
    }
    throw e;
  }
  return byCode;
}

// ==================== 主流程 ====================

function main() {
  const errors = [];

  const tsModules = parseTsPermissionCodes(readFileSync(TS_FILE, "utf8"));
  const goCodes = parseGoPermissionCodes(readFileSync(GO_CONFIG, "utf8"));

  const menuFType = new Set();
  const menuUntyped = new Set();
  for (const file of collectMigrationFiles(MIGRATIONS_DIR)) {
    const text = readFileSync(file, "utf8");
    if (file.endsWith(".sql")) {
      const parsed = parseSqlMenuPerms(text);
      for (const code of parsed.fType) menuFType.add(code);
      for (const code of parsed.untyped) menuUntyped.add(code);
    } else {
      // Go migration：权限码以字符串字面量出现（类型未知 → 仅参与存在性判断；放宽支持连字符 -）
      const re = /["']([a-z][a-z0-9_-]*(?::[A-Za-z0-9_-]+)+)["']/g;
      let m;
      while ((m = re.exec(text)) !== null) {
        menuUntyped.add(m[1]);
      }

      // Phase 87 (RBAC-06): 解析 Go 侧 F 型声明块 (// F_TYPE_SEED_BEGIN ... // F_TYPE_SEED_END)
      // 普通 Go 字面量只参与 c 类（存在性）；F_TYPE_SEED 块内的码按 F 型按钮参与 d 类（菜单 F 型 vs TS）
      const fBlockMatch = /\/\/\s*F_TYPE_SEED_BEGIN([\s\S]*?)\/\/\s*F_TYPE_SEED_END/.exec(text);
      if (fBlockMatch) {
        const blockText = fBlockMatch[1];
        const blockRe = /["']([a-z][a-z0-9_-]*(?::[A-Za-z0-9_-]+)+)["']/g;
        let bm;
        while ((bm = blockRe.exec(blockText)) !== null) {
          menuFType.add(bm[1]);
          menuUntyped.add(bm[1]);
        }
      }
    }
  }

  console.log("=== 权限码映射表双向校验（v1.24 PERM-06 + 全量对账 task-5）===");
  console.log(`前端映射表: ${TS_FILE}`);
  console.log(`后端常量  : ${GO_CONFIG}（${goCodes.size} 个）`);
  console.log(`菜单种子  : ${MIGRATIONS_DIR}（F 型按钮 ${menuFType.size} 个）`);

  // 路由挂载层 + 存量豁免（盲区1/2 修复）
  const routeRegistry = JSON.parse(readFileSync(ROUTE_REGISTRY, "utf8"));
  const routeCodes = parseRoutePermCodes(routeRegistry);
  const driftExemptions = loadDriftExemptions();
  console.log(
    `路由挂载  : ${ROUTE_REGISTRY}（操作码 ${routeCodes.size} 个，源自 ${routeRegistry.totals?.routes ?? "?"} 条路由）`
  );
  console.log(
    `存量豁免  : ${driftExemptions.size} 条（permission-drift-exemptions.json，全部 grandfathered）`
  );

  const exempt = (code) => driftExemptions.has(code);

  for (const { module, entries } of tsModules) {
    const codes = entries.map((e) => e.code);
    // 85-03：允许「主前缀的更深嵌套」同区块（如 opsBuilding 区块内的
    // ops:building:spaces:query / ops:building:spaces:3d:query —— scope 圈定
    // 按主前缀 ops:building: 本来就会把这两个 F 型码圈进 opsBuilding，
    // 若禁止深嵌套则 c/d 类与前缀一致性结构性互斥）。仅放宽嵌套深度，
    // 不同命名空间仍然报错；漂移检查（a-d）语义零变化。
    const mainPrefix = codes[0].slice(0, codes[0].lastIndexOf(":"));
    const foreign = codes.filter((c) => !c.startsWith(`${mainPrefix}:`));
    if (foreign.length > 0) {
      errors.push(
        `模块 ${module}：权限码前缀不一致（${mainPrefix} 之外出现 ${[...new Set(foreign)].join(" / ")}）`
      );
      continue;
    }
    const prefix = mainPrefix;
    const scope = `${prefix}:`;

    const tsCodes = new Set(codes);
    const backendScope = new Set([...goCodes].filter((c) => c.startsWith(scope)));
    const menuScope = new Set([...menuFType].filter((c) => c.startsWith(scope)));
    const menuAllScope = new Set([...menuFType, ...menuUntyped].filter((c) => c.startsWith(scope)));

    console.log(`\n[${module}] 作用域 ${scope}`);

    // a. TS 声明但后端常量没有（存量豁免码跳过）
    for (const code of tsCodes) {
      if (!backendScope.has(code)) {
        if (exempt(code)) {
          console.log(`  ⏸ ${code}（存量豁免：后端常量缺失）`);
          continue;
        }
        errors.push(`[${module}] TS 声明了 ${code}，但后端 pkg/permission/config.go 无对应常量`);
      }
    }
    // b. 后端常量有但 TS 没声明（存量豁免码跳过）
    for (const code of backendScope) {
      if (!tsCodes.has(code)) {
        if (exempt(code)) {
          console.log(`  ⏸ ${code}（存量豁免：TS 声明缺失）`);
          continue;
        }
        errors.push(`[${module}] 后端常量 ${code} 存在，但 TS 映射表未声明`);
      }
    }
    // c. TS 声明但 sys_menu 未声明（PENDING_SEED_WHITELIST 豁免码跳过——85-02
    //    新码待播种，87 播种后清空白名单，届时漏报会自然恢复）
    for (const code of tsCodes) {
      if (menuAllScope.has(code)) continue;
      if (PENDING_SEED_SET.has(code)) {
        console.log(`  ⏳ ${code} 在 85 新码待播种白名单内（c 类跳过；87 播种后必须清空白名单）`);
        continue;
      }
      if (exempt(code)) {
        console.log(`  ⏸ ${code}（存量豁免：sys_menu 声明缺失）`);
        continue;
      }
      errors.push(
        `[${module}] TS 声明了 ${code}，但 migrations 的 sys_menu.perms 未声明（该权限永远不会被授予）`
      );
    }
    // d. 菜单 F 型按钮有但 TS 没声明（存量豁免码跳过）
    for (const code of menuScope) {
      if (!tsCodes.has(code)) {
        if (exempt(code)) {
          console.log(`  ⏸ ${code}（存量豁免：F 型按钮无 TS 声明）`);
          continue;
        }
        errors.push(`[${module}] sys_menu 声明了 F 型权限 ${code}，但 TS 映射表未声明`);
      }
    }

    console.log(`  TS 声明 ${tsCodes.size} 个：${[...tsCodes].sort().join(", ")}`);
    console.log(`  后端常量 ${backendScope.size} 个 / 菜单 F 型 ${menuScope.size} 个`);
  }

  // ---------- 全量对账（盲区1 修复：全集驱动，不限 TS 前缀作用域） ----------
  // 全集 = TS 码 ∪ config.go 常量 ∪ seed F 型 perms ∪ 路由操作码。
  // TS 模块作用域内的码已由上方 a-d 检查覆盖；此处只对**不属于任何 TS
  // 模块作用域**的码做四向存在性检查，二者互补不重复。
  const tsAllCodes = new Set(tsModules.flatMap(({ entries }) => entries.map((e) => e.code)));
  const tsScopes = tsModules.map(({ entries }) => {
    const c0 = entries[0]?.code ?? "";
    return `${c0.slice(0, c0.lastIndexOf(":"))}:`;
  });
  const menuAll = new Set([...menuFType, ...menuUntyped]);
  const allCodes = new Set([...tsAllCodes, ...goCodes, ...menuFType, ...routeCodes]);
  // C 型页面本体码（旧口径：不参与操作码比对）——段数 < 3 的码（monitor、
  // system:user、system:dict 等）是页面/目录访问权限，不是操作码；操作码对账
  // 仅针对 ≥3 段的「域:资源(可选):动作」码。它们由菜单 C 型声明与路由页面级
  // RequirePermissions 保证，不参与四向存在性检查与孤儿判定。
  const isPageLevelCode = (code) => code.split(":").length < 3;
  const unscoped = [...allCodes].filter(
    (code) => !tsScopes.some((scope) => code.startsWith(scope)) && !isPageLevelCode(code)
  );

  console.log(
    `\n=== 全量对账（全集 ${allCodes.size} 码，其中 TS 作用域外 ${unscoped.length} 码）===`
  );
  let unscopedExempt = 0;
  for (const code of unscoped.sort()) {
    const missing = [];
    if (!goCodes.has(code)) missing.push("后端常量(config.go)");
    if (!tsAllCodes.has(code)) missing.push("TS声明(permissionCodes.ts)");
    if (!menuAll.has(code)) missing.push("sys_menu.perms");
    if (missing.length === 0) continue;
    if (exempt(code)) {
      unscopedExempt++;
      continue;
    }
    errors.push(
      `[全量对账] ${code} 缺失声明方向：${missing.join(" / ")}（存量漂移须登记豁免，新代码必须补全声明）`
    );
  }

  // ---------- 路由挂载层检查（盲区2 修复） ----------
  // 路由使用的操作码必须 ∈ 已定义集合（TS ∪ Go 常量 ∪ 菜单种子）；
  // C 型页面本体码不参与（由菜单种子 C 型声明保证）。
  const defined = new Set([...tsAllCodes, ...goCodes, ...menuAll]);
  for (const code of [...routeCodes].sort()) {
    if (isPageLevelCode(code)) continue;
    if (defined.has(code) || exempt(code)) continue;
    errors.push(
      `[路由挂载] 路由使用了未定义的权限码 ${code}（不在 TS / config.go / sys_menu 任何一处）`
    );
  }

  // 孤儿码：config.go 定义但「路由挂载 ∪ 菜单种子」双零 —— warning 级，不阻断。
  // C 型页面本体码不参与孤儿判定（它们经菜单 C 型声明使用，不走路由 permCodes）。
  const orphanCodes = [...goCodes]
    .filter((code) => !routeCodes.has(code) && !menuAll.has(code) && !isPageLevelCode(code))
    .sort();
  if (orphanCodes.length > 0) {
    console.log(
      `\n⚠ 孤儿码 ${orphanCodes.length} 个（config.go 定义但路由 + 菜单种子均未使用，warning 不阻断）：`
    );
    for (const code of orphanCodes) console.log(`  ⚠ ${code}`);
  }
  if (unscopedExempt > 0) {
    console.log(
      `\nℹ 全量对账中 ${unscopedExempt} 码按存量豁免跳过 —— 后续 task-3 整改落地后须逐条清理豁免。`
    );
  }

  // 白名单硬断言（RBAC-06 防永久化）。
  // 规则 A：白名单每一条都必须在「挂起集合」内 —— 挂起集合之外任何一条残留即 exit 1。
  // 规则 B：挂起集合已清空（87-05 收口）⇒ 白名单必须完全为空。
  // 规则 C：白名单里的码若已在 F_TYPE_SEED 块中（其实已播种）⇒ 警告提示移除（不阻断）。
  const illegalWhitelistEntries = PENDING_SEED_WHITELIST.filter(
    (w) => !PENDING_SEED_HOLD_SET.has(w.code)
  );
  if (illegalWhitelistEntries.length > 0) {
    errors.push(
      `PENDING_SEED_WHITELIST 含挂起集合之外的条目（${illegalWhitelistEntries.length} 项：` +
        `${illegalWhitelistEntries.map((w) => w.code).join(", ")}）—— ` +
        `Phase 87 第一批已播种，豁免只允许保留在挂起集合内（OQ-6 裁定）`
    );
  }
  if (PENDING_SEED_HOLD_CODES.length === 0 && PENDING_SEED_WHITELIST.length > 0) {
    errors.push(
      `PENDING_SEED_WHITELIST 非空（${PENDING_SEED_WHITELIST.length} 项）但挂起集合已清空 —— ` +
        `第二批补播完成后白名单必须归零`
    );
  }
  const alreadySeededHoldCodes = PENDING_SEED_WHITELIST.filter(
    (w) => PENDING_SEED_HOLD_SET.has(w.code) && menuFType.has(w.code)
  );
  if (alreadySeededHoldCodes.length > 0) {
    console.log(
      `\n⚠ 挂起集合中已有 ${alreadySeededHoldCodes.length} 条出现在 F 型声明块内（说明已播种）：` +
        `${alreadySeededHoldCodes.map((w) => w.code).join(", ")} —— 请从白名单移除`
    );
  }

  if (PENDING_SEED_WHITELIST.length > 0) {
    console.log(
      `\nℹ 当前白名单仅保留挂起集合（${PENDING_SEED_WHITELIST.length} 项，OQ-6 用户裁定挂起）——` +
        "待第二批补播完成后白名单归零。"
    );
  }

  if (errors.length > 0) {
    console.log("\n=== 检出漂移 ===");
    for (const e of errors) console.log(`  ✗ ${e}`);
    console.log(`\n漂移 ${errors.length} 项 —— 退出码 1`);
    process.exit(1);
  }

  console.log("\n✓ 双向零漂移 + 全量对账零漂移（TS ↔ 后端常量 ↔ sys_menu ↔ 路由挂载）");
  process.exit(0);
}

main();
