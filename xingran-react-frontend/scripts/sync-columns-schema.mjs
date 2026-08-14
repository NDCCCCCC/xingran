#!/usr/bin/env node
/**
 * sync-columns-schema.mjs
 *
 * 同步前端资产列表列定义到后端 embed JSON:
 *   src:  xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts
 *   dest: internal/services/system/asset_columns_schema.json
 *
 * 后端通过 //go:embed asset_columns_schema.json 在编译期嵌入该 JSON,
 * 替换原硬编码 defaultAssetColumns() 函数。运行时零依赖,无需 Node。
 *
 * 实现要点:
 *   - 不用 TS loader,直接对 columnsSchema.ts 文本做括号匹配提取数组字面量
 *     (避免引入 ts-node/esbuild 等额外依赖)
 *   - 用 new Function(`return (${arrayLiteral})`) 在隔离作用域求值 JS 对象字面量
 *     (避免逐字符 JSON 转换的边界 bug)
 *   - 路径以本脚本 __dirname 为基准,确保 npm run 在任意 cwd 调用都能正确解析
 */

import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(__dirname, "..", "..");

const SCHEMA_PATH = resolve(
  projectRoot,
  "xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts"
);
const OUTPUT_PATH = resolve(projectRoot, "internal/services/system/asset_columns_schema.json");
const PAGE_KEY = "asset.list";

/**
 * 从 columnsSchema.ts 文本中提取 defaultAssetColumns 数组字面量
 * 通过深度匹配的 `}` 计数定位数组结束点,正确处理字符串内的括号
 */
function extractArrayLiteral(src) {
  const marker = "export const defaultAssetColumns: AssetColumnConfig[] = [";
  const startIdx = src.indexOf(marker);
  if (startIdx < 0) {
    throw new Error(`[sync-columns-schema] marker "${marker}" not found in ${SCHEMA_PATH}`);
  }
  const arrayStart = startIdx + marker.length;

  let depth = 0;
  let inString = null;
  let escaped = false;
  let endIdx = -1;

  for (let i = arrayStart; i < src.length; i++) {
    const c = src[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (c === "\\") {
      escaped = true;
      continue;
    }
    if (inString !== null) {
      if (c === inString) inString = null;
      continue;
    }
    if (c === '"' || c === "'" || c === "`") {
      inString = c;
      continue;
    }
    if (c === "{") depth++;
    else if (c === "}") depth--;
    else if (c === "]" && depth === 0) {
      endIdx = i;
      break;
    }
  }

  if (endIdx < 0) {
    throw new Error("[sync-columns-schema] failed to locate end of defaultAssetColumns array");
  }
  return src.slice(arrayStart, endIdx);
}

/**
 * 将 TS 数组字面量转为 JS 对象数组。
 * 使用 new Function 隔离作用域,避免单/双引号转换的边界问题。
 *
 * 注意: TypeScript / 现代 JS 允许对象/数组字面量末尾的逗号(trailing comma),
 * 但 `new Function(...)` 创建的脚本在严格模式下会拒绝,需在求值前剥离。
 *
 * 需要剥离两类 trailing comma:
 *   1. 数组最末尾元素后的逗号 (`},\n` → `}\n`)
 *   2. 对象/数组嵌套元素末尾的逗号 (`},` → `}`, `],` → `]`)
 *
 * 拼接方式: 用 `[${cleaned}]` 把元素列表包成数组字面量;
 *   如果直接写 `return (${cleaned})`,逗号操作符会让返回值是最后一个对象,不是数组。
 */
function parseColumns(arrayLiteral) {
  const cleaned = arrayLiteral
    .replace(/,(\s*)$/, "$1") // 数组末尾 trailing comma
    .replace(/,(\s*[}\]])/g, "$1"); // 嵌套 trailing comma
  try {
    const parsed = new Function(`return [${cleaned}];`)();
    if (!Array.isArray(parsed)) {
      throw new Error("parsed value is not an array");
    }
    return parsed;
  } catch (err) {
    throw new Error(`[sync-columns-schema] failed to evaluate array literal: ${err.message}`);
  }
}

function main() {
  const src = readFileSync(SCHEMA_PATH, "utf8");
  const arrayLiteral = extractArrayLiteral(src);
  const columns = parseColumns(arrayLiteral);

  const output = {
    __generated__: new Date().toISOString(),
    source: "xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts",
    pageKey: PAGE_KEY,
    columns,
  };

  mkdirSync(dirname(OUTPUT_PATH), { recursive: true });
  writeFileSync(OUTPUT_PATH, JSON.stringify(output, null, 2) + "\n", "utf8");

  console.log(`[sync-columns-schema] Wrote ${columns.length} columns to ${OUTPUT_PATH}`);
}

main();
