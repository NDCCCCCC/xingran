---
slug: component-loader-invalid-path
name: component-loader-invalid-path
status: resolved
trigger: 浏览器控制台持续报 "[ComponentLoader] Invalid component path: network/discoveries.tsx"（及 system/list.tsx、system/notice.tsx、assets/dashboard.tsx），并抛出未捕获的 promise rejection
created: 2026-11-03
updated: 2026-11-03
resolved: 2026-11-03
---

## 症状 (Symptoms)

**期望行为:**
- 鼠标悬停在顶部多标签栏（TabBar）某个已打开 Tab 上时静默预取目标页面 chunk，控制台不应出现任何错误
- 组件加载器 `ComponentLoader.load()` 只接受以 `pages/` 或 `components/` 开头的**组件路径**，非法路径应被拒绝但不应污染控制台

**实际行为:**
- 每次悬停 Tab 标签都打印 `[ComponentLoader] Path not in whitelist`（warn）与 `[ComponentLoader] Invalid component path: <路径>`（error）
- 随后抛出 `Uncaught (in promise) Error: Invalid component path: ...`，异常冒泡到全局

**错误信息:**
```
[ComponentLoader] Path not in whitelist Object
isValidComponentPath @ index-BnADbsd9.js:2
[ComponentLoader] Invalid component path: network/discoveries.tsx
load @ index-BnADbsd9.js:2
15:22:25.227 index-BnADbsd9.js:2 Uncaught (in promise) Error: Invalid component path: network/discoveries.tsx
    at fe.load (index-BnADbsd9.js:2:51568)
    at index-BnADbsd9.js:2:53514
    at Array.map (<anonymous>)
    at Ma (index-BnADbsd9.js:2:53504)
    at onMouseEnter (index-BnADbsd9.js:2:88338)
    at xO (vendor-react-BLGl8JLG.js:8:128896)
```
同样路径还出现在：`system/list.tsx`、`system/notice.tsx`、`assets/dashboard.tsx`（assets/dashboard.tsx 出现两次）

**时间线:**
- 引入提交：`3dbd957f feat(frontend): 前端累积功能与测试(数周积累)`（2026-10-16）——该提交在 `TabBar.tsx` 首次加入 hover 预取调用（`git log -S "hover 预取"` / `git log -S "preloadComponents"` 定位）。此前的 `19ad3d4d feat(TabBar): 添加标签页滚动导航和右键上下文菜单功能` 只改了 TabBar 交互，未涉及预取。

**重现步骤:**
1. 登录运维管理系统，顶部打开多个页面 Tab
2. 鼠标悬停在任一已打开的 Tab 标签上
3. 控制台立即打印上述错误并抛出未捕获异常

---

## 当前关注点 (Current Focus)

**假设 (Hypothesis):** 已验证成立——`TabBar.tsx` 的 hover 预取把**路由 URL** 当作**组件路径**传给 `componentLoader.load()`，被白名单拒绝后 reject，且调用方未 `.catch()`。

**下一步行动 (Next Action):** 无（会话已解决，详见「解决方案 (Resolution)」）。

---

## 证据 (Evidence)

- timestamp: 2026-11-03 15:30:00
  source: code_review
  detail: |
    `xingran-react-frontend/src/router/componentLoader.tsx` 读解：
    - 第 18 行：`private static readonly ALLOWED_PREFIXES = ["pages/", "components/"]`
    - 第 63-66 行：`load()` 先做白名单校验，非法则 `console.error` + `throw new Error('Invalid component path: ...')`
    - 第 96 行：白名单判定就是 `path.startsWith(prefix)`，因此任何不以 `pages/` / `components/` 开头的路径必然失败
    - 第 108-122 行 `normalizePath()`：去掉前导 `/`、补 `.tsx`——这解释了日志中 `network/discoveries.tsx` 的形态（原始入参为 `/network/discoveries`）
    - 第 235-237 行 `preloadComponents()`：`Promise.all(componentPaths.map(p => componentLoader.load(p)))`——一个 reject 即整体 reject，且调用方 `TabBar.tsx:320` 没有 `.catch`，故变成 `Uncaught (in promise)`
    - 第 182-229 行 `createLazyComponent()`：已实现「去 pages/ 前缀 → 补 pages/ 前缀 → 补 .tsx」的归一化，与 `load()` 的白名单语义一致——两者契约不统一是根因所在

- timestamp: 2026-11-03 15:31:00
  source: code_review
  detail: |
    `xingran-react-frontend/src/components/layout/shared/TabBar.tsx:309-329` 读解：
    - 第 310-315 行 `createTabLabel(tab: { title; key; path; pinned? })`——`path` 字段属于 **tab（路由）** 数据结构，语义是路由 URL
    - 第 318-321 行：`onMouseEnter={() => { preloadComponents([tab.path]); }}`，注释写「hover 预取目标路由 chunk（仅预取，非导航）」——意图是预取路由 chunk，但实际传的是路由 URL 而非组件路径
    - 栈帧 `onMouseEnter → Ma → Array.map → fe.load` 与该调用链完全吻合

- timestamp: 2026-11-03 15:45:00
  source: lead_cross_check
  detail: |
    **[Lead 补充环境事实 — 已据此更新会话，未重复以下调查]**

    **① 用户实际访问的构建产物**
    - 用户浏览器里打开的是**后端 :9000 托管的前端构建产物**：`internal/server/xingran-react-frontend/dist/assets/index-BnADbsd9.js`（与 `xingran-react-frontend/dist/assets/index-BnADbsd9.js` 同名同 hash）。日志里的 `index-BnADbsd9.js` / `opsApi-Bmn_Aecq.js` / `vendor-react-BLGl8JLG.js` 就来自这个构建。
    - 另有 vite dev server 监听 :4000（PID 17240），后端 `ruoyi-backend.exe` PID 31076 监听 :9000。
    - **重要**：修复前端源码后，用户的运行环境（:9000 托管的构建产物）**不会自动更新**。见下方「验证 (Verification)」第 4 条，已写明经查证的构建 + 产物同步命令。

    **② 仓库里共有两份构建产物**
    `xingran-react-frontend/dist/` 与 `internal/server/xingran-react-frontend/dist/`；另有 worktree 副本 `.worktrees/main-fix/xingran-react-frontend/src/router/componentLoader.tsx` —— **本次未修改 worktree 副本**，只改主工作区（已核对：`.worktrees/main-fix/...` 不在本次变更文件列表中）。

    **③ 现有测试覆盖缺口**
    `xingran-react-frontend/src/router/__tests__/componentLoader.unit.test.ts` 目前只覆盖路径穿越 / 反斜杠 / `.html` 等拒绝行为，**没有覆盖「路由路径被当作组件路径」这一场景**。已补齐（见「变更文件」）。

    **④ 修复需同时覆盖两点**
    1. 传给 loader 的路径**语义正确**（组件路径而非路由 URL）——`createLazyComponent`（第 182-229 行）已有「去 pages/ 前缀 → 补 pages/ 前缀 → 补 .tsx」的归一化，而 `load()`（第 58-81 行）没有，两套契约不统一是根因。
    2. 预取失败必须**静默**：hover 预取是性能优化，不该污染控制台，也不该产生未捕获的 promise rejection（`preloadComponents` 第 235-237 行用 `Promise.all`，调用方 `TabBar.tsx:320` 没有 `.catch`）。

- timestamp: 2026-11-03 15:42:00
  source: code_review
  detail: |
    **独立验证 `tab.path` 的真实语义**（不依赖初始假设）：
    - `src/components/layout/shared/useRouteTabs.ts:98-105` 与 `src/hooks/useTabSync.ts:89-97` 两处注册 Tab 时均为
      `addTab({ key: location.pathname, ..., path: location.pathname, ... })` → **`tab.path` 就是路由 URL**
    - `src/store/tabsStore.ts` 的 `TabItem` 只有 `path` 字段（`src/types/layout.ts:72`），全项目不存在第二个「组件路径」字段
    - 结论：`preloadComponents([tab.path])` 传给 `componentLoader.load()` 的必然是路由 URL，与 `load()` 的「组件路径」契约不匹配

- timestamp: 2026-11-03 15:44:00
  source: code_review
  detail: |
    **全量调用点检索**（`preloadComponents|componentLoader.load|createLazyComponent|new ComponentLoader`，26 处命中）：
    - `preloadComponents` 只有 1 个调用点：`TabBar.tsx:320`（本次问题唯一来源）
    - `componentLoader.load` 只有 `preloadComponents` 内部 1 处调用
    - `createLazyComponent` 只有 `DynamicRoutes.tsx:88` 1 处调用
    - → 不存在第二个传错路径的调用点，修复面收敛在 TabBar + componentLoader

- timestamp: 2026-11-03 15:46:00
  source: git_history
  detail: |
    **引入点定位**（`git log -S`）：
    - `git log --oneline -S "hover 预取" -- xingran-react-frontend/src ruoyi-react-frontend/src` → `3dbd957f`（唯一命中）
    - `git show 3dbd957f -- .../TabBar.tsx` 的 diff 明确新增：
      `+ import { preloadComponents } from "@/router/componentLoader";`
      `+        preloadComponents([tab.path]);`
    - 即：预取功能自落地起就传错参数，**从未成功预取过任何 chunk**，报错是长期存在的既有缺陷（用户加载了含该 Tab 的页面后才暴露）

- timestamp: 2026-11-03 15:47:00
  source: code_review
  detail: |
    **为什么不能靠字符串规则推导组件路径**（用数据库菜单种子 `internal/core/db/migrations/menu_catalog_seed.sql` 核对）：
    - 菜单「设备发现」：`path='network/discoveries'`（父 network），`component='network/discoveries/index'` → 路由 `/network/discoveries` ↔ 组件 `pages/network/discoveries/index.tsx`（一致）
    - 菜单「资产管理」下 `path='dashboard'`：`component='asset/reconciliation/dashboard/index'` → 路由 `/assets/dashboard` ↔ 组件 `pages/asset/reconciliation/dashboard/index.tsx`
      **注意目录名是 `asset`（单数）而路由段是 `assets`（复数）**——字符串拼接会得到不存在的 `pages/assets/dashboard/index.tsx`
    - `routeGenerator.resolveComponent` 对 `path === "dashboard"` 还有渲染特例 → `pages/dashboard-system/index`（该文件确实存在）
    - → 路由路径必须经**菜单数据**（`routeConfigManager` → `RouteGenerator.resolveComponent`）映射，这是唯一可靠来源

- timestamp: 2026-11-03 15:52:00
  source: test
  detail: |
    **RED 验证（修复前）**：`src/router/__tests__/componentLoader.preload.test.ts` 3/3 失败，失败输出与用户报告逐字一致：
    `Error: Invalid component path: network/discoveries.tsx`（reject）+ `[ComponentLoader] Invalid component path: network/discoveries.tsx`（console.error）

- timestamp: 2026-11-03 16:02:00
  source: test
  detail: |
    **RED 有效性回验**：临时 `git stash` 掉 3 个源码修复文件（保留新测试）后重跑 4 个新测试文件 → **12/12 全红**，失败信息仍为原症状；`git stash pop` 恢复后 12/12 全绿。
    → 新测试确实能抓住该缺陷，不是「永远绿」的假回归测试。

- timestamp: 2026-11-03 16:03:00
  source: test
  detail: |
    **GREEN 验证**：
    - 新增/相关测试：`componentLoader.preload.test.ts`(3) + `componentLoader.preloadRoute.test.ts`(7) + `TabBar.hover-prefetch.test.tsx`(1) + `TabBar.hover-prefetch.contract.test.tsx`(1) = 12 passed
    - 既有单测：`componentLoader.unit.test.ts`(14) 全部通过，无回归
    - 目录级回归：`npx vitest run src/router src/components/layout` → **27 files / 161 tests passed**
    - `npx eslint <7 个改动文件>` → exit 0
    - `npm run type-check`（tsc --noEmit -p tsconfig.app.json）→ exit 0

- timestamp: 2026-11-03 16:08:00
  source: code_review
  detail: |
    **构建产物托管链路（经查证，非臆造）**：
    - `scripts/build-embedded.bat`（Windows，另有 `.sh`）= 项目既有的「前端构建 → 托管产物同步 → 后端内嵌构建」脚本：
      Step1 清 `xingran-react-frontend/dist` → Step2 `cd xingran-react-frontend && npm run build`（= `tsc -b && vite build`）
      → Step3 `rmdir /s /q internal\server\xingran-react-frontend` 后 `xcopy /e /i /y xingran-react-frontend\dist internal\server\xingran-react-frontend\dist`
      → Step4 `go build -tags=embed -ldflags="-s -w" -o out/xingran-backend-embedded.exe ./cmd/main.go`
    - 托管机制：`internal/server/embed_frontend_prod.go`（`//go:build embed`）用 `//go:embed all:xingran-react-frontend/dist` 把 `internal/server/xingran-react-frontend/dist` **嵌入二进制**；
      `internal/server/embed_frontend.go`（`//go:build !embed`）不托管前端（返回 503，提示改用 vite dev server :4000）。
    - ⇒ 用户在 :9000 能看到前端 ⇒ 其后端是 embed 构建 ⇒ **只改源码不够，必须重新构建前端 + 同步产物 + 重新 go build -tags=embed + 重启后端**。

---

## 已排除的假设 (Eliminated)

- **「加载器白名单前缀配错了」**——排除。`ALLOWED_PREFIXES = ["pages/", "components/"]` 与 `createLazyComponent` 的归一化语义一致，且菜单 component 值本就以 `pages/` 归一化后使用；白名单本身没有错，错的是调用方传入了非组件路径。
- **「路由路径可以用字符串规则推导组件路径」**——排除。`/assets/dashboard → pages/asset/reconciliation/dashboard/index.tsx`（目录名与路由段不一致）、`/dashboard → pages/dashboard-system/index.tsx`（渲染特例）两例证明必须走菜单数据映射。
- **「报错来自 DynamicRoutes 渲染链路」**——排除。`createLazyComponent` 对不存在的模块返回错误组件而非抛错，且调用链 `onMouseEnter → Array.map → load` 明确指向 hover 预取。
- **「存在第二个传错路径的调用点」**——排除。全项目检索确认 `preloadComponents` / `componentLoader.load` / `createLazyComponent` 各仅 1 个调用点。
- **「console.error 是必要的排障日志、只需吞掉异常」**——排除。用户明确期望「非法路径应被拒绝但不应污染控制台」，且报错源头是「把路由 URL 当组件路径」这一类型错误，日志只会掩盖真因。

---

## 解决方案 (Resolution)

**根本原因 (Root Cause):**
`TabBar.tsx` 的 hover 预取把 **Tab 的路由 URL**（`tab.path` = `location.pathname`）当作**组件路径**传给 `componentLoader.load()`；而 `load()` 的契约是「只接受 `pages/` 或 `components/` 开头的组件路径」，于是必然被白名单拒绝并 `throw`，`preloadComponents()` 用 `Promise.all` 聚合且调用方没有 `.catch()`，最终表现为控制台 `[ComponentLoader] Invalid component path: <路由路径>.tsx` 报错 + `Uncaught (in promise)` 未捕获异常。三处叠加：①调用方传错语义、②加载器缺少「路由路径 → 组件路径」的映射能力、③预取失败未被隔离（既打 error 日志又向外 reject）。

**修复方案 (Fix):**
1. `src/router/componentLoader.tsx`
   - 新增 `ComponentLoader.preload(componentPath)`：静默预取（不抛错、不打日志），命中则写入缓存；`preloadComponents()` 改用它，单项失败既不影响其它项也不再向外 reject（消除未捕获 rejection）。
   - `load()` 白名单拒绝时**只抛错、不再 `console.error`**；白名单不匹配的 `console.warn` 一并移除（危险字符告警保留，属安全信号）。
   - 新增导出 `resolveRouteComponentPath(routePath)`：经 `routeConfigManager`（菜单数据）+ `RouteGenerator.resolveComponent` 把路由路径映射为真实组件路径；新增 `preloadRouteComponent(routePath)`：映射后静默预取。
2. `src/router/routeGenerator.ts`：`resolveComponent` 由 private 提为 public static，使预取链路与渲染链路复用**同一套归一化规则**（保证预取到的 chunk 就是渲染用到的那个，且 dashboard 特例一致）。
3. `src/components/layout/shared/TabBar.tsx`：hover 预取改调 `void preloadRouteComponent(tab.path)`——传路由路径给「路由预取 API」，不再把路由 URL 当组件路径。

**验证 (Verification):**
1. **RED**：修复前新增测试 3/3 失败（失败输出与用户报告逐字一致）；`git stash` 回退 3 个源码文件后 4 个新测试文件 12/12 全红，确认测试有效、不是假回归。
2. **GREEN**：新增 4 个测试文件 12/12 通过；既有 `componentLoader.unit.test.ts` 14/14 通过；目录级回归 `npx vitest run src/router src/components/layout` 共 **27 files / 161 tests passed**；`npx eslint <7 个改动文件>` exit 0；`npm run type-check` exit 0。
3. **全量前端测试**：`cd xingran-react-frontend && npx vitest run`（结果见「验证补充」）。
4. **⚠️ 让修复在用户环境生效（源码修复不会自动进入 :9000 的构建产物）**——经查证的既有命令：
   ```bat
   :: 方式 A（一条命令走完构建+同步+后端内嵌构建，项目既有脚本）
   scripts\build-embedded.bat

   :: 方式 B（手工分步，与 build-embedded.bat 的 Step2/3/4 等价）
   cd xingran-react-frontend && npm run build
   xcopy /e /i /y xingran-react-frontend\dist internal\server\xingran-react-frontend\dist
   go build -tags=embed -ldflags="-s -w" -o out/xingran-backend-embedded.exe ./cmd/main.go
   :: 然后重启后端并强制刷新浏览器（Ctrl+F5）
   ```
   注意：前端资源经 `//go:embed all:xingran-react-frontend/dist` 嵌入后端二进制（`internal/server/embed_frontend_prod.go`），
   **不重新 `go build -tags=embed` 的话，新产物不会生效**；若改用 vite dev server（:4000，`npm run dev`）调试则无需构建步骤。
5. **未触碰**：`internal/server/xingran-react-frontend/dist/` 与 `xingran-react-frontend/dist/` 两份既有构建产物本次**未重新生成**（避免把构建产物噪声混入本次修复提交）；worktree 副本 `.worktrees/main-fix/` 未修改；工作区内其它模块的既有未提交改动（后端 swagger / xingran-cli 等）未触碰。

**变更文件 (Files Changed):**
- `xingran-react-frontend/src/router/componentLoader.tsx`（修改）
- `xingran-react-frontend/src/router/routeGenerator.ts`（修改：`resolveComponent` 提为 public）
- `xingran-react-frontend/src/components/layout/shared/TabBar.tsx`（修改：hover 预取改用路由预取 API）
- `xingran-react-frontend/src/router/__tests__/componentLoader.preload.test.ts`（新增：复现回归）
- `xingran-react-frontend/src/router/__tests__/componentLoader.preloadRoute.test.ts`（新增：路由→组件映射 + 静默失败）
- `xingran-react-frontend/src/components/layout/shared/__tests__/TabBar.hover-prefetch.test.tsx`（新增：悬停不报错/无未捕获异常）
- `xingran-react-frontend/src/components/layout/shared/__tests__/TabBar.hover-prefetch.contract.test.tsx`（新增：TabBar 调用契约）

---

## Specialist Review

**specialist_hint:** typescript / react
**说明:** 准则要求映射到 `typescript-expert` skill，但当前会话的 skill 目录未安装该 skill（可用列表中不存在），且本会话 delegation depth 已达上限、无法再派生子代理执行独立评审。改由本代理按 TypeScript/React 视角做内联评审，并如实标注评审主体。

**结论:** LOOKS_GOOD

- 方向正确：根因是「调用方传了错误语义的参数」，修复落在调用方语义修正 + 加载器补齐映射能力 + 失败隔离三处，未做无关重构。
- 幂等与开销：`preload()` 命中 `ComponentLoader.cache` 直接返回，重复 hover 不会重复 import chunk；失败项每次 hover 会重试一次（模块不存在属配置错误，代价可忽略）。
- 时序安全：菜单未就绪时 `routeConfigManager.getRouteByPath()` 返回 undefined → 静默跳过；菜单加载后自动恢复，不需要额外的状态协调。
- 依赖方向无环：`componentLoader → routeConfigManager / routeGenerator`，而二者均不 import `componentLoader`（已核对 import 列表）。
- 公共 API 面变化可控：`RouteGenerator.resolveComponent` 由 private 提为 public static，行为未变，只扩大可见性。
- 常见陷阱已规避：①未采用「给 load() 放宽白名单」的做法（会削弱安全边界，且无法解决 `assets/dashboard → asset/reconciliation/...` 的映射错位）；②未把 `console.error` 降级为 `console.warn`（仍是污染）；③测试刻意不断言真实 chunk import —— vitest 下真实 import 需 >30s（实测 `pages/network/discoveries/index.tsx` transform 超时），改以「映射结果 + glob key 命中」等效断言，避免引入 flaky/超慢用例。

---

## 无责复盘 (Postmortem)

- **为什么没被抓住:** 该缺陷自 `3dbd957f`（2026-10-16）落地起就存在，但此前**没有任何测试覆盖 hover 预取链路**；`componentLoader` 的既有单测只覆盖「白名单拒绝路径」，且 `load()` 与 `createLazyComponent()` 两套路径归一化契约长期不一致却无一致性测试锁定。
- **守门产物:** `xingran-react-frontend/src/router/__tests__/componentLoader.preload.test.ts`（预取静默/不污染控制台）、`.../componentLoader.preloadRoute.test.ts`（路由→组件映射正确性 + 静默失败）、`.../TabBar.hover-prefetch.test.tsx`（悬停不产生未捕获异常）、`.../TabBar.hover-prefetch.contract.test.tsx`（TabBar 必须走路由预取 API 的调用契约）。

---

## 验证补充 (Verification Appendix)

- **全量前端测试**（`cd xingran-react-frontend && npx vitest run`）：**589 个测试文件 / 4101 个测试全部通过（exit 0），无任何新失败**（Duration 1179.62s / import 2254.63s / tests 1307.08s）。
- 本次改动的影响面同时由目录级回归**完整覆盖**：`componentLoader` 的全部调用点只落在 `src/router` 与 `src/components/layout` 内（已逐个检索确认），这两个目录共 27 files / 161 tests 全绿；另有 ESLint（exit 0）与 `tsc --noEmit`（exit 0）。
- 说明：第一轮全量测试因未指定工作目录而在仓库根执行，把 `.worktrees/main-fix/` 副本下的测试一并收集，产生大量 `ERR_MODULE_NOT_FOUND` —— 属执行环境错误（副本缺少依赖解析），**非本次改动导致**；已在正确目录 `xingran-react-frontend/` 重跑并全绿。
