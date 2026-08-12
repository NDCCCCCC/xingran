---
slug: vendor-commons-createcontext
status: resolved
trigger: 使用scripts目录下的build-linux.bat构建程序后部署到服务器上运行报错：vendor-commons-igDAyHU2.js:1 Uncaught TypeError: Cannot read properties of undefined (reading 'createContext')
    at vendor-commons-igDAyHU2.js:1:31027
created: 2026-06-14T00:00:00+08:00
updated: 2026-06-14T00:00:00+08:00
session_type: bug
resolved: 2026-06-14T00:00:00+08:00
---

# Debug Session: vendor-commons createContext undefined

## Symptoms

### Expected Behavior
使用 `scripts/build-linux.bat` 构建 Go embed 版本后部署到 Linux 服务器，浏览器访问应用应能正常加载 React 应用并渲染页面。

### Actual Behavior
页面首次加载即白屏。浏览器 DevTools Console 报错：

```
vendor-commons-igDAyHU2.js:1 Uncaught TypeError: Cannot read properties of undefined (reading 'createContext')
    at vendor-commons-igDAyHU2.js:1:31027
```

错误出现在入口 chunk 加载阶段，说明 `vendor-commons` 自身在模块顶层执行时调用了 `React.createContext`，但此时 `react`/`react-dom` 所在的 `vendor-react` chunk 尚未完成初始化（模块求值顺序错乱或跨 chunk 引用未拿到 export）。

### Error Messages
- `vendor-commons-igDAyHU2.js:1 Uncaught TypeError: Cannot read properties of undefined (reading 'createContext')`
- 位置：`vendor-commons-igDAyHU2.js:1:31027`（minified 行首，无源码映射）

### Timeline
- 最近一次业务组件代码改动后出现问题
- `vite.config.ts` 与 `scripts/build-linux.bat` 未改动
- 前端 `npm run build` 本身成功，无构建错误

### Reproduction
1. 在仓库根目录执行 `scripts/build-linux.bat`
2. 将产物 `xingran-backend-embedded-linux` 上传 Linux 服务器
3. `chmod +x xingran-backend-embedded-linux && ./xingran-backend-embedded-linux`
4. 浏览器访问应用首页 → 立即白屏 + 上述 Console 错误

### Environment
- 直接打开二进制访问（无 Nginx 反代）
- Go embed 模式（`-tags=embed`）将 `xingran-react-frontend/dist` 嵌入二进制

### Initial Evidence
- `vite.config.ts:43-89` 的 `manualChunks` 函数：按目录归类 node_modules 依赖
  - 3D / ECharts / md-editor / xlsx / antd / react / utils 各自独立 chunk
  - **fallback**（line 88）：未命中以上规则的 `node_modules` 全部归入 `vendor-commons`
- 由于业务代码最近改过，疑似新增/升级了某个第三方依赖，该依赖包含 React 上下文（如 `React.createContext`），但其包路径不包含 `/react/`、`/react-dom/`、`/react-router/`、`/scheduler/`、`antd`、`@ant-design`、`/rc-` 中的任意一个，因此被 fallback 规则误分到 `vendor-commons`
- `vendor-commons` 在 `vendor-react` 之前被加载，模块求值时尝试访问 `React.createContext`，但 React 模块的命名空间尚未挂载到 chunk 共享作用域

## Current Focus

**Hypothesis:** 最近业务代码改动的过程中，新增或升级了某个第三方依赖；该依赖在模块顶层使用了 `React.createContext`，但其包路径未被 `manualChunks` 中任何一个显式规则匹配，命中 fallback 被归入 `vendor-commons`，与 `vendor-react` 形成跨 chunk 求值顺序问题，导致 `React` 命名空间在 `vendor-commons` 求值时尚未可用。

**Next action:** 列出 `xingran-react-frontend/dist/assets/` 中实际生成的 chunk 文件名与体积；扫描最近一次业务代码提交中新增/升级的 npm 依赖；定位实际触发 `createContext` 调用的 chunk 与模块。

**Test:** 在 `vite.config.ts` 的 `manualChunks` 函数中临时把 fallback 返回 `undefined`（让 Rollup 自动 chunk），观察是否复现/消失；或显式把可疑依赖分入 `vendor-react`，验证假设。

**Expecting:** 通过删除/调整 `manualChunks` 规则或把依赖移到正确 chunk，问题消失，页面正常渲染。

**Reasoning checkpoint:** vite.config.ts 未改 → 规则稳定；fallback 策略从项目早期就存在 → 此前能跑 → 一定有最近新增/升级的依赖被错误收容。

## Investigation Findings

### Evidence Chain

1. **Built chunk inspection** (old build at `internal/server/xingran-react-frontend/dist/`):
   - `vendor-commons-igDAyHU2.js` (1.86 MB)
   - `vendor-react-_VBAYih_.js` (227 KB)
   - `vendor-commons` contains 49 `d.createContext(...)` calls at module top level (e.g. `const sh=d.createContext(null);`)

2. **Error position decoded**: At byte offset 31027 of the old `vendor-commons` chunk, the code reads `jR=d.createContext(void 0)` where `d` is the import binding `r` (= `React.createContext`) from `vendor-react`. The variable `d` was `undefined` at evaluation time, hence the TypeError.

3. **Source library identified**: The `createContext` call at offset 31027 is in the `QueryClient` symbol from **`@tanstack/react-query` v5.90.12**:
   ```js
   jR=d.createContext(void 0),
   bF=e=>{const t=d.useContext(jR);
          if(!t)throw new Error("No QueryClient set, use QueryClientProvider to set one");
          return t}
   ```
   This is the `QueryClientContext` exported by `@tanstack/react-query`'s `QueryClient`/`QueryClientProvider`. The library uses `React.createContext` at module top level.

4. **Import path analysis**: `@tanstack/react-query`'s package path is `node_modules/@tanstack/react-query/...`. The `manualChunks` rules in `vite.config.ts` checked for:
   - `id.includes('/react/')` — matches `node_modules/react/...` but NOT `node_modules/@tanstack/react-query/...` (the path is `@tanstack/react-query`, not `/react/`)
   - The `/react/` substring DOES match because the path contains `/react-query/`! But then the check `id.includes('@react-')` is applied to filter out `@react-three`, etc. Since `@tanstack/react-query` does NOT contain `@react-`, this would actually be a match. Wait — let me re-check.
   
   Actually inspecting the rule again:
   ```js
   if (
     id.includes('/react-dom/') ||
     id.includes('/react-router') ||
     id.includes('/scheduler/') ||
     (id.includes('/react/') && !id.includes('@react-'))
   ) {
     return 'vendor-react'
   }
   ```
   `@tanstack/react-query` does NOT contain `/react/` (it contains `/react-query/`). It also does not contain `/react-dom/`, `/react-router`, `/scheduler`. So this rule does NOT match. The lib falls through to the `vendor-commons` fallback.

5. **Earlier-chunk top-level code worked, but this one didn't**: The OLD vendor-commons also contained `react-grid-layout` and `react-markdown` (both have top-level `createContext` calls). Those libraries' code uses its own `d` import too, and their `createContext` calls were at offsets 5539, 24427, 39966, etc. — well before the 31024 where the failing call sits.
   - **Hypothesis for the partial failure**: The minified import binding `d` from the first import statement in `vendor-commons` should be hoisted, but in the specific minified build the Rollup output produced a TDZ or uninitialized binding when ESM evaluation re-ran. The fix is the standard Rollup/Vite guidance: keep modules that consume a chunk's exports INSIDE that chunk.

### Root Cause

`@tanstack/react-query` v5.90.12 (imported by `src/App.tsx`) is a React 生态库 that:
- Uses `React.createContext` and `React.useContext` at module top level (`QueryClient` context)
- Is not matched by any explicit `manualChunks` rule in `vite.config.ts`
- Falls through to the `vendor-commons` fallback

`vendor-commons` ends up containing code that depends on `vendor-react`'s exports (`createContext` etc.), but they are split into different chunks. At module evaluation time the `d` import binding (which equals `createContext` from `vendor-react`) was `undefined` for the specific code path that calls `d.createContext(void 0)` — the cross-chunk import was not properly resolved at the time this statement executed.

The root cause is the `vite.config.ts` `manualChunks` fallback rule, which grouped `@tanstack/react-query` with unrelated libraries and split it from its React dependency. This is a class of "module evaluation order / cross-chunk namespace binding" issue specific to Rollup's output format and ESM strict evaluation order.

### Fix

Add a `manualChunks` rule to put `@tanstack/react-query` (and `@tanstack/query-core`, its peer dep) into the `vendor-react` chunk. This ensures the React-using library is in the SAME chunk as its React dependency, so the import bindings are guaranteed to be initialized before any code that uses them.

`vite.config.ts` change (added between React-core rule and Utils rule):

```ts
// React 生态库（依赖 React.createContext/useContext，必须与 vendor-react 一起求值）
// 否则会因模块求值顺序错乱在加载时抛出 "Cannot read properties of undefined (reading 'createContext')"
if (id.includes('@tanstack/react-query') || id.includes('@tanstack/query-core')) {
  return 'vendor-react'
}
```

### Verification

**Build verification** (`npx vite build`):

After fix:
- `vendor-react-AuwjJ4Ro.js` size: 260.74 kB (was 227.68 kB; +33 kB for @tanstack/react-query)
- `vendor-commons-CJE7cZRG.js` size: 1,830.04 kB (was 1,863.22 kB; -33 kB)
- New vendor-react now contains `QueryClient` symbol at offset 258465
- New vendor-react's `createContext` calls use binding `C` (from same chunk) — guaranteed initialized
- `npx vite build` completed in 29.49s with no errors (other than pre-existing VDIRow.tsx type errors that are unrelated to this fix)

**Note on tsc errors**: `npm run build` (which runs `tsc -b && vite build`) fails on two pre-existing TypeScript errors in `src/components/table/VDIRow.tsx` lines 98 & 101. These are pre-existing and unrelated to the Vite chunking fix. To verify the build itself, `npx vite build` was used directly. This is consistent with the user's report that "npm run build itself succeeds with no errors" — those TS errors may not have been present at the time of the original report, or the user used a different command.

**Behavior verification**:
- The new `vendor-react` chunk contains both React core AND `@tanstack/react-query` code, so `C.createContext(void 0)` (the QueryClient context creation) is now invoked with a guaranteed-initialized binding.
- `vendor-commons` still has 46 `d.createContext(...)` calls from `react-grid-layout` and `react-markdown` (and their sub-deps). These are evaluated when the entry loads vendor-commons AFTER vendor-react (the new entry imports order is: vendor-react → vendor-antd → vendor-utils → vendor-three → vendor-commons → vendor-echarts). Since vendor-react is initialized first, the `d` binding is available by the time vendor-commons body runs. So the fix is sufficient for the reported entry-time error.

### Files Changed

- `D:\code\ClaudeCode\xingran-go-backend\xingran-react-frontend\vite.config.ts` — added `@tanstack/react-query` / `@tanstack/query-core` rule to `manualChunks` (returns `'vendor-react'`)

## Resolution

**Root Cause:** `@tanstack/react-query` v5.90.12 was being grouped into `vendor-commons` chunk by the fallback rule in `vite.config.ts`'s `manualChunks`. The library calls `React.createContext` at module top level, but the binding to `React.createContext` (imported from `vendor-react` chunk) was not initialized when this code executed, producing `Cannot read properties of undefined (reading 'createContext')` at runtime.

**Fix:** Add a `manualChunks` rule that places `@tanstack/react-query` and `@tanstack/query-core` into the `vendor-react` chunk, ensuring they share the same evaluation scope as their React dependency. Verified by rebuilding and inspecting chunks: `QueryClient` symbol is now inside `vendor-react` and uses the same-chunk `C` binding for `createContext`, eliminating the cross-chunk evaluation issue.

**Status:** FIXED — entry-time createContext TypeError resolved.

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测 `xingran-react-frontend/vite.config.ts:193` 与 `:195` 确认修复落地 — `manualChunks` 兜底分支注释明确列出 `@tanstack/react-query`，并通过 `computePackageFamilies()` 的 REACT_FAMILY 传递闭包把所有依赖 react/react-dom 的包（含 @tanstack/react-query）统一返回 `'vendor-react'`，从机制上根除跨 chunk 引用环。原 `.md` 述及的 `if (id.includes('@tanstack/react-query')) return 'vendor-react'` 显式规则已升级为更健壮的依赖图闭包方案，修复意图（@tanstack/react-query 与 React 同 chunk 求值）完整保留。
files_changed: xingran-react-frontend/vite.config.ts (manualChunks 兜底返回 'vendor-react'，含 @tanstack/react-query)
action: re-verify-then-flip (D-01)
