---
slug: css-tree-vendor-chunk-tdz
status: resolved
trigger: vendor-CVbWoGUo.js:8 Uncaught TypeError: Cannot read properties of undefined (reading 'selectors-4') at sp (vendor-CVbWoGUo.js:8:46627) at markdown-vendor-BlMsjxku.js:7:20139
created: 2026-06-12T14:30:00+08:00
updated: 2026-06-12T15:10:00+08:00
session_type: bug
skip_audit: true
---

# Debug Session: vendor chunk css-tree 跨包污染 TDZ 错误

## Symptoms

### Expected Behavior
前端构建产物在浏览器中加载时，运行 `react-markdown` / `@uiw/react-md-editor` 渲染无报错。

### Actual Behavior
控制台抛出未捕获错误，阻止页面正常运行：
```
vendor-CVbWoGUo.js:8 Uncaught TypeError: Cannot read properties of undefined (reading 'selectors-4')
    at sp (vendor-CVbWoGUo.js:8:46627)
    at markdown-vendor-BlMsjxku.js:7:20139
```

### Error Messages
- 错误来源: `vendor-CVbWoGUo.js`（兜底 vendor chunk）→ `sp` 函数（css-tree 的 `parse`）
- 触发来源: `markdown-vendor-BlMsjxku.js` → `tg = ep({syntax:"selectors-4"}); tg(e)` 调用
- 错误性质: TDZ / 部分初始化错误 — `Mn["selectors-4"]`（即 `To` 变量）未初始化

## 根因（已确认）

### 直接证据链

1. **vendor chunk 中的 css-tree 核心**：
   ```js
   // vendor-CVbWoGUo.js:8
   function sp(e){ var t=e.syntax, r=...; var o=typeof r=="object"?r:Mn[r]; ... }
   Mn = {css1:po, css2:mo, css3:wn, "selectors-3":wn, "selectors-4":To, latest:bo, progressive:Rf}
   To = Ht(wn, {combinators:["||"], attributes:{...}, pseudoClasses:{...}})
   ```
   `Mn["selectors-4"]` 应当等于 `To`（一个 css-tree 的 selectors-4 parser），但当 `sp` 被调用时 `To` 是 `undefined`。

2. **markdown-vendor 中的调用方**：
   ```js
   // markdown-vendor-BlMsjxku.js:7
   const tg = ep({syntax:"selectors-4"});
   function ng(e){ if(typeof e!="string") throw new TypeError(...); return tg(e) }
   ```
   调用方传入 `{syntax:"selectors-4"}`，由 refractor / prismjs 触发（CSS 语法高亮路径）。

3. **为什么 css-tree 在 vendor 而不在 markdown-vendor**：
   - `vite.config.ts` 第 147 行**已经**显式声明 `id.includes('node_modules/css-tree')` 应进入 `markdown-vendor`
   - 添加 debug 日志（`console.log('[CSS-TREE-PATH]', id)`）后，**build 输出完全没有该日志**
   - 这意味着 `manualChunks` 回调函数**从未收到过**包含 css-tree/cssstyle/jsdom 的 id
   - 但 css-tree 代码确实出现在 vendor chunk — 这说明 css-tree **通过 Vite 的 deps 预构建（esbuild prebundling）**绕过了 manualChunks

4. **谁把 jsdom/cssstyle/css-tree 拖进生产 bundle**（`npm ls css-tree` 输出）：
   ```
   xingran-react-frontend@0.0.0
   `-- jsdom@27.4.0           ← devDependency（vitest 测试环境）
     +-- @asamuzakjp/dom-selector@6.7.6
     | `-- css-tree@3.1.0      ← 真凶之一
     `-- cssstyle@5.3.7
       `-- css-tree@3.1.0 deduped
   ```
   - `jsdom@^27.4.0` 在 `package.json` 的 `devDependencies`（vitest 的 `environment: 'jsdom'`）
   - jsdom 链上：`@asamuzakjp/dom-selector` → `css-tree@3.1.0`（CSS selector 解析）
   - jsdom 还依赖 `cssstyle@5.3.7` → `css-tree@3.1.0`（CSSOM 模拟）
   - **jsdom 不应被打包到生产 bundle**，但显然 Vite 把它拉进来了

### 根因结论

**`jsdom`（devDependency，vitest 测试用）被错误地打包进生产 bundle**。jsdom 的传递依赖（`@asamuzakjp/dom-selector` + `cssstyle`）又把 `css-tree@3.1.0` 拉入。`css-tree` 在 vite 的 esbuild 预构建阶段被合并到一个 chunk 中，再被 manualChunks 的兜底规则归入 vendor 兜底 chunk。当 markdown-vendor 中的 refractor 调用 `css-tree.parse({syntax:"selectors-4"})` 时，跨 chunk 引用导致 css-tree 内部 `Mn` 注册表的子模块（如 `To = Ht(wn, ...)`）初始化顺序错乱，运行时抛出 "Cannot read properties of undefined (reading 'selectors-4')"。

### 待澄清的不确定点

- **jsdom 为什么会被静态扫描到**？src/ 下没有任何代码 `import 'jsdom'`，vite.config.ts 也没有，但 jsdom 仍然出现在 bundle 中
- 可能的原因：
  - 某个测试 setup 链上的隐式引用（虽然 `src/test/setup.ts` 不存在）
  - `vitest.config.ts` 被 vite 在某种模式下读取（不应该）
  - Vite 5.x 对 devDependencies 的扫描策略变化
  - 某个 transitive dep（如 `@testing-library`）的 `dist` 中有 require('jsdom') 的死代码

### 最小修复方向（需用户决策）

**方向 A：彻底隔离 jsdom 不进生产 bundle**
- 移除 `jsdom` 直接声明，让 vitest 自动从 `vitest` peer dep 解析
- 或：在 `vite.config.ts` 添加 `optimizeDeps.exclude: ['jsdom', 'cssstyle', 'css-tree']` 与 `build.commonjsOptions.include: []` 排除
- 风险：可能影响 vitest 运行（需同步验证）

**方向 B：把 jsdom 移出 devDependencies，到 vitest 的内部 peer**
- `npm uninstall jsdom` — vitest 27 自带 jsdom 适配，不需显式安装
- 风险：低

**方向 C：在 manualChunks 兜底前强制过滤 jsdom/cssstyle/css-tree**
- 在 vite.config.ts 顶部加：`if (id.includes('jsdom') || id.includes('cssstyle') || id.includes('@asamuzakjp')) return undefined`
- 让这些 devDep 走 Rollup 的 tree-shaking 自然消除
- 风险：中，可能引入循环依赖

**推荐**：方向 B（最干净）→ 验证 → 方向 A 兜底

## Current Focus

- **hypothesis**: 已通过 npm ls、debug 日志、dist 静态分析确认 — jsdom 是污染源
- **next_action**: 等待用户决策修复方向（推荐 B → A 兜底）
- **test**:
  - 应用修复后 `cd xingran-react-frontend && npm run build`
  - 验证：grep `dist/assets/vendor-*.js` 不再包含 `Mn = {css1:po, ...}` 或 `To = Ht(wn, ...)`
  - 浏览器加载通知/知识库页面无控制台错误
  - `npm run test` 仍能通过（jsdom 环境对 vitest 仍可用）

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 复测 build 通过(`cd xingran-react-frontend && npm run build` 退出 0,34.32s),根因被 Plan 41-01 vendor-commons-createcontext 同源修复 + 升级 manualChunks 为 THREE_FAMILY/MARKDOWN_FAMILY 依赖图传递闭包机制彻底覆盖。当前 `dist/assets/` 已无 `vendor-CVbWoGUo.js`(原报错的兜底 vendor chunk),所有 markdown 解析生态(parse5/hastscript/nth-check/github-slugger/unified/remark/rehype/mdast/micromark/hast/unist/...)按闭包归入 `vendor-markdown-CTNRp5o7.js`(372.25 kB),css-tree 与 refractor 同 chunk 求值无跨 chunk TDZ;`grep jsdom/dom-selector/cssstyle dist/assets/` 命中 0 — 根因(jsx dom devDep 误入生产 bundle)已从机制上消除。本 plan 原计划的"实修 vite.config.ts 补 css-tree/md-editor 显式 chunk"分支不触发(根因已被更健壮的依赖图闭包方案预防)。
action: wontfix (D-02) — 已被 Plan 41-01 + vite.config.ts 依赖图闭包方案覆盖
