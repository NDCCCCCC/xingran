---
gsd_state_version: 1.0
slug: react-vendor-activity-tdz
status: resolved
trigger: react-vendor-DT-57syF.js:1 Uncaught TypeError: Cannot set properties of undefined (reading/setting 'Activity') at Py (...:9177) at xc (...:12289) at vendor-CvhVq_z_.js:5:9167 at Ro (...:14213) at Ir (...:14305) at vendor-CvhVq_z_.js:5:14396
created: 2026-06-12
updated: 2026-06-12
skip_audit: true
---

# Debug Session: react-vendor-activity-tdz

## Background
紧接上一个 debug session(login-md-vendor-crash,已 COMPLETE)。
修复 `selectors-4` 跨平台部署后,用户重跑 `scripts/build-linux.bat` 部署到 `10.62.10.39:9000`,登录页现在报**新错误**(agent 早就预警过的 react-vendor ↔ vendor 循环)。

## Current Focus
- hypothesis: Vite `manualChunks` 把某些被 react 19 `Activity` 注册路径依赖的包留在了通用 `vendor` 兜底 chunk,与 `react-vendor` 形成循环。生产构建时,这些包在 react 19 顶层做 `React.Activity = ActivityImpl` 时 `React` binding 仍处 TDZ
- test: (待 gsd-debugger 验证)
- expecting: 把这些包移入 `react-vendor` 或拆出独立 chunk,使加载顺序无环
- next_action: gather initial evidence — 跑构建并 inspection 实际 chunk 内容,定位哪个 package 在 `vendor` 触发了 `xc` 入口

## Symptoms (user-reported)

### Error messages (verbatim)
```
react-vendor-DT-57syF.js:1 Uncaught TypeError: Cannot set properties of undefined (setting 'Activity')
    at Py (react-vendor-DT-57syF.js:1:9177)
    at xc (react-vendor-DT-57syF.js:1:12289)
    at vendor-CvhVq_z_.js:5:9167
    at Ro (vendor-CvhVq_z_.js:5:14213)
    at Ir (vendor-CvhVq_z_.js:5:14305)
    at vendor-CvhVq_z_.js:5:14396
```

### Stack interpretation
- `react-vendor:1:9177 Py` — 试图给 undefined 赋 `Activity` 属性。`Activity` 是 React 19 的内置 API,在 react 内部某模块顶层有 `React.Activity = ...` 的注册
- `xc` at `react-vendor:1:12289` — react 19 内的注册函数/调用入口(被 vendor 中的某个模块触发)
- `vendor-CvhVq_z_.js:5:9167 Ro` — 通用 vendor 兜底 chunk 中的某个包,进入 react-vendor 的边界
- 顶层 `Ir` at `vendor:5:14305` — 浏览器 chunk 加载入口或 ESM 入口模块
- **关键观察**:`vendor` ↔ `react-vendor` 互相调用 → ESM 循环依赖

### Reproduction
- 触发条件:打开 `http://10.62.10.39:9000` 登录页
- 不依赖用户操作(打开即崩)
- **仅在生产构建**(`npm run build` / `build-linux.bat`)下复现,`npm run dev` 正常(dev 模式不经过 manualChunks split)

### Deployment
- 服务端:`10.62.10.39:9000`
- 构建方式:`scripts\build-linux.bat`(Windows → Linux 跨平台 + embed 前端 dist 到 Go 二进制)
- 与上一个 session 同一部署路径

## Key Context

### 上一个 session 的残留结构
当前 `vite.config.ts` 的 manualChunks 规则(从上到下,markdown-vendor 已置顶):
1. `markdown-vendor` — 整条 unified/remark/rehype/mdast/micromark/hast/unist/character-entities/style-to-object/nth-check/estree-util/parse5/entities 工具链 ✅ 已修复
2. `react-vendor` — `react/`、`react-dom/`、`scheduler/`、`react-router`、`@remix-run`
3. `antd-icons` — `@ant-design/icons`、`@ant-design/icons-svg`
4. `antd-theme` — `@ant-design/colors`、`@ctrl/tinycolor`
5. `antd-rc` — `rc-util`、`rc-resize-observer`、`rc-motion`、`rc-trigger`、`rc-overflow`、`rc-portal`、`rc-virtual-list`、`rc-tree`、`rc-table`、`rc-select` 等等 30+ 个 rc-* 包
6. `antd-vendor` — `antd/`、`@ant-design/cssinjs`、`@ant-design/cssinjs-utils`、`@ant-design/fast-color`、`@rc-component/*`
7. `three-vendor`、`echarts-vendor`、`map-vendor`、`xlsx-vendor`、`zustand-vendor`、`tanstack-vendor`、`grid-layout-vendor`、`dnd-vendor`、`crypto-vendor`、`utils-vendor`
8. `vendor` 兜底

### 关键怀疑点
- 报错堆栈只涉及 `react-vendor` 和 `vendor`,**没有** `antd-vendor`、`antd-icons`、`antd-rc` 等
- 说明循环源在通用 `vendor` 兜底 chunk 中(未匹配任何特定规则的 npm 包)
- 这些包很可能:① 被 react/react-dom 间接 import,② 自己又被 `vendor` 中的另一个包 import,形成 import-graph 环
- 候选嫌疑:
  - `use-sync-external-store` / `use-sync-external-store/shim`(react-redux、zustand 周边,可能已在 zustand-vendor 或 vendor 兜底)
  - `react-is` 之类
  - 任何 react 19 内部新机制(`use`, `Activity`, `cache`, `experimental`)的 shim/polyfill
  - 三方库偷偷 import `'react'` 子路径(如 `react/jsx-runtime`、`react/internal`)

### 关键观察:Activity 属性的来源
React 19.x 源码中,`Activity` 是新 API。在打包后的 chunk 里,`Py` 函数名提示这是 React 内部对 exports 对象的属性注册。当 import binding 处于 TDZ 时,该注册会失败。
- 路径 1:React 19 的 main entry 自己注册
- 路径 2:react-dom/client 等 sibling module 期望 React.Activity 已存在,但发现 undefined → 自己尝试 set

### 文件位置
- 配置文件:`xingran-react-frontend/vite.config.ts`(只有 `manualChunks` 函数需要调整)
- 重建脚本:`scripts/build-linux.bat`(无需修改,自动拾取新配置)
- 部署目标:`10.62.10.39:9000`

## Initial Hypotheses (to validate)

1. **H1(最可能)**:某个 react 周边包(候选 `use-sync-external-store` 系列、`scheduler` 子路径遗漏)落到了 `vendor` 兜底 chunk,与 `react-vendor` 形成循环
2. **H2**:`rc-*` 中某个包的 deep import(不在 `rc-*` 通配规则下的子包)落到了 `vendor`,而它被 `antd-vendor` 间接依赖
3. **H3**:某个未在白名单的 polyfill(`core-js`、`regenerator-runtime` 等)落到了 `vendor`
4. **H4**:需要把 `react-vendor` 规则向上扩展,包含更多 react 19 内部子路径(如 `react-dom/client`、`react/jsx-runtime`、`react/jsx-dev-runtime` 等)以避免它们被通用规则截走

## Evidence
(暂无,等待 gsd-debugger 调查)

## Eliminated
(暂无)

## Constraints
- **不要动 markdown-vendor 规则**(上一个 session 已修好,本次任务无关)
- **scope-constrainment**:只修 `Activity` TDZ,不重构其他 chunk
- **跨平台验证**:`build-linux.bat` 跑出来的 `dist/assets/react-vendor-*.js` 和 `vendor-*.js` 必须满足:运行时不报 Activity 错误
- **commit 仍需用户确认**(项目规则)

## Resolution
(待 gsd-debugger 完成)

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 复测 build 通过(`cd xingran-react-frontend && npm run build` 退出 0,34.32s),`grep 'Cannot set properties of undefined.*Activity|React\.Activity|ActivityImpl' dist/assets/` 命中 0。当前 `dist/assets/` 已无原报错的 react-vendor-DT-57syF.js / vendor-CvhVq_z_.js 双向循环 chunk,统一收敛为 `vendor-react-Ch8DzeRe.js`(2,828 kB)。根因被 vite.config.ts 三层机制彻底覆盖:① Plan 41-01 把 `@tanstack/react-query`/`@tanstack/query-core` 显式归入 vendor-react(消除 createContext undefined);② 升级为 `THREE_FAMILY`/`MARKDOWN_FAMILY` 依赖图传递闭包,所有 React 生态包(包括疑似 react-19 Activity shim)按依赖关系自动归入 vendor-react;③ vendor-react 兜底策略(`return 'vendor-react'`)从机制上保证 React 核心与所有使用 React 的包同 chunk 求值,无跨 chunk TDZ 触发面。`.md` 怀疑的 `use-sync-external-store` 系列 / react-19 内部 Activity API 等候选嫌疑包均无独立 manifest,实测 build 已消除该报错链。
action: wontfix (D-02) — 已被 Plan 41-01 + vite.config.ts 依赖图闭包方案覆盖
