#!/usr/bin/env node
/**
 * v1.24 PERM-06 —— 「模块 × 操作 → 权限码」映射表一致性校验（**双向**）。
 *
 * 单一真相源：`ruoyi-react-frontend/src/constants/permissionCodes.ts`
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

/** 权限码形态：至少两段、段内小写字母数字下划线（如 system:user:resetPwd 的 resetPwd 不匹配 → 用宽松段） */
const PERM_LITERAL = /^[a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+$/;

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
    const columns = splitSqlTopLevel(stmt[1]).map((c) => c.trim().replace(/^"|"$/g, "").toLowerCase());
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
      // Go migration：权限码以字符串字面量出现（类型未知 → 仅参与存在性判断）
      const re = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+)["']/g;
      let m;
      while ((m = re.exec(text)) !== null) {
        menuUntyped.add(m[1]);
      }
    }
  }

  console.log("=== 权限码映射表双向校验（v1.24 PERM-06）===");
  console.log(`前端映射表: ${TS_FILE}`);
  console.log(`后端常量  : ${GO_CONFIG}（${goCodes.size} 个）`);
  console.log(`菜单种子  : ${MIGRATIONS_DIR}（F 型按钮 ${menuFType.size} 个）`);

  for (const { module, entries } of tsModules) {
    const codes = entries.map((e) => e.code);
    const prefixes = new Set(codes.map((c) => c.slice(0, c.lastIndexOf(":"))));
    if (prefixes.size !== 1) {
      errors.push(`模块 ${module}：权限码前缀不一致（${[...prefixes].join(" / ")}）`);
      continue;
    }
    const prefix = [...prefixes][0];
    const scope = `${prefix}:`;

    const tsCodes = new Set(codes);
    const backendScope = new Set([...goCodes].filter((c) => c.startsWith(scope)));
    const menuScope = new Set([...menuFType].filter((c) => c.startsWith(scope)));
    const menuAllScope = new Set(
      [...menuFType, ...menuUntyped].filter((c) => c.startsWith(scope))
    );

    console.log(`\n[${module}] 作用域 ${scope}`);

    // a. TS 声明但后端常量没有
    for (const code of tsCodes) {
      if (!backendScope.has(code)) {
        errors.push(`[${module}] TS 声明了 ${code}，但后端 pkg/permission/config.go 无对应常量`);
      }
    }
    // b. 后端常量有但 TS 没声明
    for (const code of backendScope) {
      if (!tsCodes.has(code)) {
        errors.push(`[${module}] 后端常量 ${code} 存在，但 TS 映射表未声明`);
      }
    }
    // c. TS 声明但 sys_menu 未声明
    for (const code of tsCodes) {
      if (!menuAllScope.has(code)) {
        errors.push(`[${module}] TS 声明了 ${code}，但 migrations 的 sys_menu.perms 未声明（该权限永远不会被授予）`);
      }
    }
    // d. 菜单 F 型按钮有但 TS 没声明
    for (const code of menuScope) {
      if (!tsCodes.has(code)) {
        errors.push(`[${module}] sys_menu 声明了 F 型权限 ${code}，但 TS 映射表未声明`);
      }
    }

    console.log(`  TS 声明 ${tsCodes.size} 个：${[...tsCodes].sort().join(", ")}`);
    console.log(`  后端常量 ${backendScope.size} 个 / 菜单 F 型 ${menuScope.size} 个`);
  }

  if (errors.length > 0) {
    console.log("\n=== 检出漂移 ===");
    for (const e of errors) console.log(`  ✗ ${e}`);
    console.log(`\n漂移 ${errors.length} 项 —— 退出码 1`);
    process.exit(1);
  }

  console.log("\n✓ 双向零漂移（TS ↔ 后端常量 ↔ sys_menu 菜单种子）");
  process.exit(0);
}

main();
