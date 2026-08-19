---
quick: 260703-eaj-b-schema-embed-xingran-react-frontend-src-
type: execute
---

# 目标

实现"前后端 defaultAssetColumns 单一真理源"：
1. 前端 `defaultAssetColumns`（51 列，260703-dkc 重写后）抽到独立 TS 文件 `xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts`，原 `index.tsx` 改为 `import` 该常量
2. 新增 Node 脚本 `xingran-react-frontend/scripts/sync-columns-schema.mjs` 把 schema 序列化为 `internal/services/system/asset_columns_schema.json`（**JSON 必须放在 column_config_service.go 同包目录以满足 go:embed 约束——禁 `..` 父目录引用**）
3. 后端 `column_config_service.go` 用 `//go:embed asset_columns_schema.json` 读取该 JSON，替换硬编码的 `defaultAssetColumns()` 函数（仅替换 asset 分支，user/role/dept 不动）
4. `package.json` 加 `sync-columns-schema` 脚本和 `prebuild` 钩子（`build` 前自动跑）
5. 验证：`npm run type-check` 0 错误 + `go build ./...` 0 错误 + `go test ./internal/services/system/...` 全绿

# 上下文

- 项目根 CLAUDE.md：已入仓
- `.planning/STATE.md`：v1.16 已闭环；当前 quick 流
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 当前 51 列 `defaultAssetColumns` 定义（260703-dkc commit 97b49ea7 后）
- `internal/services/system/column_config_service.go` — 当前 43 列硬编码 `defaultAssetColumns()`（已 stale：含 recipientName/assetIp/assetMac/assetValue/assetYear 不存在字段，缺 signOrgnoName/nowUserName/nowUserDeptCode/status/nbfStatus/drawingDate/machineUptime/lastInventoryDate 9 个真实字段）
- `internal/models/user_column_config.go` — 列配置 DB schema（`ColumnKey/Visible/DisplayOrder/Width`）
- `internal/models/system/requests/column_config_request.go` — `ColumnConfigItem` DTO 定义
- `internal/server/embed_frontend_prod.go` — 参考 `//go:embed all:xingran-react-frontend/dist` 的同仓库 go:embed 写法
- 参考 quick 计划：`.planning/quick/260703-dkc-xingran-react-frontend-src-pages-operation/260703-dkc-PLAN.md`

# 诊断摘要

## A. 前后端 drift 现状

| 来源 | 数量 | 关键差异 |
|------|------|----------|
| 前端 `defaultAssetColumns` (260703-dkc) | 51 列 | 真字段（deviceUserName/signOrgnoName/status/nbfStatus/drawingDate/machineUptime/lastInventoryDate） |
| 后端 `defaultAssetColumns()` (硬编码) | 43 列 | 含 `recipientName/assetIp/assetMac/assetValue/assetYear` 5 个不存在/僵尸 key，缺 9 个真实 key |

后果：用户首次打开资产列表（无 UserColumnConfig 行）后端 fallback 到后端 43 列 → 列与前端不齐；用 `useColumnConfig` 的 `defaultConfig` 传入前端 51 列 → "列配置"模态框 vs 实际渲染列结构不一致。resetConfig 行为不确定。

## B. go:embed 路径约束（关键！）

`//go:embed` **不允许** `..` 父目录引用，embed 源必须位于 embed 文件同包目录下（或子目录）。`column_config_service.go` 包路径是 `internal/services/system/`，所以：
- ✅ JSON 放在 `internal/services/system/asset_columns_schema.json`
- ❌ 不能放在 `xingran-react-frontend/scripts/...` 或 `internal/services/operations/...`

生成脚本 Node 进程运行（CWD 是项目根或 frontend 根），需要写绝对路径或相对项目根路径，**但生成出来的 JSON 落点必须在 backend embed 范围内**。

## C. 单一真理源流程图

```
columnsSchema.ts (前端 TS 常量, 51 列)
       │
       │ import (前端运行时)
       ▼
index.tsx (useColumnConfig defaultColumns)
       │
       │ sync-columns-schema.mjs (build 前运行)
       ▼
asset_columns_schema.json (生成产物, 提交进 git)
       │
       │ //go:embed (后端编译时)
       ▼
column_config_service.go (替代硬编码 defaultAssetColumns())
       │
       │ GetDefaultConfig("asset.list")
       ▼
UserColumnConfig DTOs (前端 useColumnConfig 接收)
```

# 任务

## Task 1: 抽前端 defaultAssetColumns 到独立 schema 文件 + index.tsx 改 import

**文件**:
- `xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts` (新增, ~70 行)
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` (修改 1 行 import + 删除 67 行定义)

**action**:

1. 创建 `columnsSchema.ts`：
   - 复制 `index.tsx` line 59-66 的 `AssetColumnConfig` interface 完整定义（key/label/visible/order/width/group 六字段）
   - 复制 line 74-141 的 51 项 `defaultAssetColumns` 数组**完整内容**（不删不改，仅位移）
   - 顶部加 `export type { AssetColumnConfig }` 单独导出类型（解耦未来扩展）
   - 底部加 `export const defaultAssetColumns: AssetColumnConfig[] = [ ... ]`
   - 文件头注释：`/** 此文件为资产列表列定义的单一真理源 — 后端 sync-columns-schema.mjs 会序列化此文件到 internal/services/system/asset_columns_schema.json，编辑后必须跑 npm run sync-columns-schema 同步后端 */`

2. 修改 `index.tsx`：
   - 删除 line 59-66 的 `AssetColumnConfig` interface 定义（不再需要）
   - 删除 line 73-141 的 51 项 `defaultAssetColumns` 数组（搬走）
   - line 42-47 import 块下加一行：`import { defaultAssetColumns, type AssetColumnConfig } from "./columnsSchema";`
   - line 676 的 `defaultConfig={defaultAssetColumns}` prop 保持不变（同一引用）
   - line 222 的 `defaultColumns: defaultAssetColumns` prop 保持不变

**verify**:
```bash
cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend
npm run type-check
```
预期: 0 errors（关键 — TS 必须能解析新 import 路径 `./columnsSchema`）

**done**:
- `columnsSchema.ts` 存在且包含 51 项数组
- `index.tsx` 删除原定义后只剩 import
- type-check 通过

## Task 2: 新增 sync-columns-schema 脚本 + package.json 集成

**文件**:
- `xingran-react-frontend/scripts/sync-columns-schema.mjs` (新增, ~50 行)
- `xingran-react-frontend/package.json` (修改 scripts 段, 1-2 行新增)

**action**:

1. 创建 `xingran-react-frontend/scripts/sync-columns-schema.mjs`：
   - Node ES module（package.json 已有 `"type": "module"`，可用 `.mjs` 或 `.js`；用 `.mjs` 显式语义清晰）
   - 入口逻辑：
     ```js
     import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
     import { dirname, resolve } from "node:path";
     import { fileURLToPath } from "node:url";
     import { createRequire } from "node:module";
     
     // 解析项目根（scripts 的父目录的父目录）
     const __dirname = dirname(fileURLToPath(import.meta.url));
     const projectRoot = resolve(__dirname, "..", "..");
     
     // 1) 读 schema 源（TS）— 不能直接 require，TS 类型在运行时被擦除，
     //    用 esbuild? 不必要。改用更稳的方案: 改让 schema 文件同时 export 一个
     //    序列化 helper, 或者 script 直接静态扫描 .ts 提取数组字面量。
     //    选定: 重新设计 — script 读 schema.ts 用 ts.transpileModule? Node 没内置 TS。
     //    最简方案: 让 schema.ts 同时 export 纯数据 defaultAssetColumnsData (无类型注解),
     //    script 用 dynamic import + 字符串 eval? 也不稳。
     //    ★最终方案: 用正则从 columnsSchema.ts 文本提取 defaultAssetColumns 数组,
     //    然后 JSON.parse。注意: schema.ts 数组内字符串可能有中文逗号，
     //    用 robust 提取：取 `export const defaultAssetColumns: AssetColumnConfig[] = [` 起点
     //    和下一个 `];` 终点，对内层 `{` `}` 配对计数提取完整数组字面量。
     ```
   - 实际推荐实现（**避免 TS loader 依赖**）：
     ```js
     // Read schema.ts as text
     const schemaPath = resolve(projectRoot, "xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts");
     const src = readFileSync(schemaPath, "utf8");
     // Extract defaultAssetColumns array literal by bracket matching
     const startMarker = "export const defaultAssetColumns: AssetColumnConfig[] = [";
     const startIdx = src.indexOf(startMarker);
     if (startIdx < 0) throw new Error("defaultAssetColumns marker not found in columnsSchema.ts");
     const arrayStart = startIdx + startMarker.length;
     // Find matching `];` by depth counting `{` and `}`
     let depth = 0;
     let inString = null;
     let escaped = false;
     let endIdx = -1;
     for (let i = arrayStart; i < src.length; i++) {
       const c = src[i];
       if (escaped) { escaped = false; continue; }
       if (c === "\\") { escaped = true; continue; }
       if (inString) {
         if (c === inString) inString = null;
         continue;
       }
       if (c === '"' || c === "'" || c === "`") { inString = c; continue; }
       if (c === "{") depth++;
       else if (c === "}") depth--;
       else if (c === "]" && depth === 0) { endIdx = i; break; }
     }
     if (endIdx < 0) throw new Error("defaultAssetColumns array end not found");
     const arrayLiteral = src.slice(arrayStart, endIdx);
     // Convert TS object literal to JSON: keys are already quoted strings,
     // only difference is TS allows trailing commas and unquoted keys (we use quoted)
     // Strip trailing commas before `}` or `]`
     const cleaned = arrayLiteral
       .replace(/,(\s*[}\]])/g, "$1")
       .replace(/'/g, '"'); // single-quoted strings to double-quoted
     // But strings may contain escaped single quotes inside; safer: use Function() to evaluate
     // Simpler: wrap in parentheses and Function() to evaluate as JS object literal
     const parsed = (new Function(`return (${arrayLiteral});`))();
     ```
   - 包装为最终 JSON：
     ```js
     const out = {
       __generated__: new Date().toISOString(),
       source: "xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts",
       pageKey: "asset.list",
       columns: parsed,  // [{key, label, visible, order, width, group}, ...]
     };
     // 字段顺序固定: __generated__, source, pageKey, columns
     const outPath = resolve(projectRoot, "internal/services/system/asset_columns_schema.json");
     mkdirSync(dirname(outPath), { recursive: true });
     writeFileSync(outPath, JSON.stringify(out, null, 2) + "\n", "utf8");
     console.log(`[sync-columns-schema] Wrote ${parsed.length} columns to ${outPath}`);
     ```
   - 幂等性：相同 schema.ts 文本 → 相同 JSON（数组项顺序由 source 决定，无 random/hash）。`__generated__` 时间戳是唯一变化字段，但 Git diff 中能看到同步时间；构建产物 OK。

2. 修改 `package.json` scripts 段（line 6-18）：
   - 在 `"type-check"` 后加：
     ```json
     "sync-columns-schema": "node scripts/sync-columns-schema.mjs",
     ```
   - 把 `"build"` 从 `"tsc -b && vite build"` 改为 `"npm run sync-columns-schema && tsc -b && vite build"`（prebuild 钩子替代方案；npm 不原生支持 pre* 钩子，除非用 `prebuild` 同名 script）
   - 更稳：加 `"prebuild": "npm run sync-columns-schema"`，让 `"build"` 保持 `"tsc -b && vite build"`：
     ```json
     "sync-columns-schema": "node scripts/sync-columns-schema.mjs",
     "prebuild": "npm run sync-columns-schema",
     "build": "tsc -b && vite build",
     ```
   - 验证：`npm run sync-columns-schema` 独立可跑；`npm run build` 自动先跑 sync

**verify**:
```bash
cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend
npm run sync-columns-schema
ls -la ../internal/services/system/asset_columns_schema.json
cat ../internal/services/system/asset_columns_schema.json | head -20
```
预期:
- 命令 exit code 0
- JSON 文件存在
- 头部含 `"__generated__": "2026-07-03T..."` `"pageKey": "asset.list"` `"columns": [...]` (51 项)

**done**:
- 脚本可独立运行
- JSON 输出 51 列
- package.json 含 `sync-columns-schema` + `prebuild`

## Task 3: 后端 embed 替换硬编码 defaultAssetColumns + 兼容 test

**文件**:
- `internal/services/system/asset_columns_schema.json` (Task 2 生成的产物, 提交进 git)
- `internal/services/system/column_config_service.go` (修改 defaultAssetColumns 函数 + 新增 embed var)

**action**:

1. 修改 `column_config_service.go`:
   - 顶部 imports 加 `"embed"` (line 2 后)
   - 紧接 package 声明后 line 11 前加：
     ```go
     //go:embed asset_columns_schema.json
     var assetColumnsSchemaFS embed.FS
     ```
   - 删除 `defaultAssetColumns()` 函数定义 line 141-189（43 列硬编码）
   - 在 `getDefaultColumnsForPage` 内部 `case "asset.list":` 分支改：
     - 旧：`return defaultAssetColumns()`
     - 新：调用新函数 `return loadAssetColumnsFromEmbed()`
   - 新增 `loadAssetColumnsFromEmbed()` 函数：
     ```go
     // loadAssetColumnsFromEmbed 从 embed JSON 读取资产列表默认列配置
     // JSON 由前端 sync-columns-schema.mjs 维护,确保前后端单一真理源。
     // 注:此函数无 ctx 参数;embed 在进程启动时已加载到内存。
     func loadAssetColumnsFromEmbed() []ColumnConfigItem {
         data, err := assetColumnsSchemaFS.ReadFile("asset_columns_schema.json")
         if err != nil {
             // embed 失败 = 构建/部署错误,panic 暴露问题优于静默回退
             panic(fmt.Sprintf("asset_columns_schema.json embed read failed: %v", err))
         }
         var schema struct {
             Columns []ColumnConfigItem `json:"columns"`
         }
         if err := json.Unmarshal(data, &schema); err != nil {
             panic(fmt.Sprintf("asset_columns_schema.json parse failed: %v", err))
         }
         return schema.Columns
     }
     ```
   - imports 加 `"encoding/json"` (line 2 后, 与 embed 同行)
   - `defaultUserColumns/defaultRoleColumns/defaultDeptColumns` 三个函数保持不变 (out of scope, 后续 quick 可推广)

2. 检查 `internal/services/system/` 目录所有 `_test.go`:
   - 用 grep 检查是否有人 import 测试 `defaultAssetColumns` 函数:
     ```bash
     grep -rn "defaultAssetColumns" D:/code/ClaudeCode/xingran-go-backend/internal/services/system/
     ```
   - 预期: Task 3 完成后 grep 应只命中 `getDefaultColumnsForPage` 一处(在 `column_config_service.go` 内调用 `defaultAssetColumns()` 的行,需改名为 `loadAssetColumnsFromEmbed()` 或保留 `defaultAssetColumns` 作为 embed 版本的薄包装)
   - 方案选择: 保留 `defaultAssetColumns()` 函数名为 embed 版本薄壳（最小 diff）:
     ```go
     // defaultAssetColumns 资产列表默认配置 — 数据来源 asset_columns_schema.json (go:embed)
     // 真实源: xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts
     func defaultAssetColumns() []ColumnConfigItem {
         return loadAssetColumnsFromEmbed()
     }
     ```
   - 这样 line 121 `case "asset.list": return defaultAssetColumns()` 保持不变,无需改其他文件

3. 跑 `go test ./internal/services/system/...`:
   - 现有 test 不依赖具体列数 (确认用 `grep -l "defaultAssetColumns" internal/services/system/*_test.go`),如有依赖需更新 assertion
   - 跑 `go build ./...` 确认 embed 编译通过

**verify**:
```bash
cd D:/code/ClaudeCode/xingran-go-backend
go build ./...
go test ./internal/services/system/...
```
预期:
- build 0 error
- test 全绿 (任何红 → 改 test 期望,不删 test)

**done**:
- `column_config_service.go` 含 `//go:embed` + JSON 反序列化逻辑
- 硬编码 43 列数组删除
- `defaultAssetColumns()` 保留函数签名(改为 embed 薄壳)
- go build 0 error
- go test 全绿

# 最终验证

```bash
# 1. 重新生成 JSON 确保最新
cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend
npm run sync-columns-schema
# 2. 前端 TS 类型检查
npm run type-check
# 3. 后端 build + test
cd ..
go build ./...
go test ./internal/services/system/...
# 4. (可选) build 前端确认 prebuild 钩子触发
cd xingran-react-frontend
npm run build | head -20
```

**Success Criteria**:
- [ ] `columnsSchema.ts` 存在, 51 列
- [ ] `index.tsx` 用 `import { defaultAssetColumns, type AssetColumnConfig } from "./columnsSchema";` 不再含内联定义
- [ ] `npm run type-check` exit 0
- [ ] `npm run sync-columns-schema` exit 0,生成 51 列 JSON
- [ ] `internal/services/system/asset_columns_schema.json` 存在,顶部含 `__generated__` 时间戳
- [ ] `column_config_service.go` 含 `//go:embed asset_columns_schema.json`
- [ ] 原 43 列硬编码数组已删除
- [ ] `go build ./...` exit 0
- [ ] `go test ./internal/services/system/...` 全绿
- [ ] `npm run build` 自动先跑 sync-columns-schema
- [ ] 已有用户 UserColumnConfig 行不受影响(后端 GetDefaultConfig 仅在无配置时调用,DB 优先)

# 输出

创建 `.planning/quick/260703-eaj-b-schema-embed-xingran-react-frontend-src-/260703-eaj-SUMMARY.md`, 记录:
- 实际生成的列数(应=51)
- 嵌入 JSON 文件 size
- prebuild 钩子实际触发证据(`npm run build` 输出的 `[sync-columns-schema] Wrote N columns` 行)
- go test 实际通过数
- 任何 plan 之外的 deviation
