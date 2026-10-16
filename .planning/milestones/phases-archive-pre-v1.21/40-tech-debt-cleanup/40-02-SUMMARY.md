---
phase: 40-tech-debt-cleanup
plan: 02
subsystem: ui
tags: [frontend, react, dev-verification, session-persistence, hmr, theme]

requires:
  - phase: 历史前端修复 phase(useTableManager / authStore / defaultThemeApi / opsApi)
    provides: 5 个 awaiting_human_verify session 的代码修复已在前序 commit 落地
provides:
  - 5 个 awaiting_human_verify session 全部 dev 浏览器验证通过 + 翻 resolved
  - 3 个 session 补齐 slug 字段(D-09 resolved/ 风格,保证 40-03 validator 100% pass)
  - 每个 session Resolution 章节追加 Phase 40 Closure 验证证据(D-07)
affects: [40-03, verify_phase40.sh, audit-open awaiting_human_verify 桶]

tech-stack:
  added: []
  patterns: ["dev 浏览器验证闭环:代码修复(历史已落)→ 用户 dev 环境复现 → 通过则翻 resolved + 证据,失败则 D-06 反复修"]

key-files:
  created: []
  modified:
    - .planning/debug/workstation-device-build-errors.md(Task 1,build 验证)
    - .planning/debug/workstation-expand-device-load-empty.md(Task 2,补 slug)
    - .planning/debug/reset-theme-hardcoded-colors.md(Task 3)
    - .planning/debug/user-filter-lost-after-edit.md(Task 4,补 slug)
    - .planning/debug/vite-hmr-token-blank-page.md(Task 5,补 slug)

key-decisions:
  - "5 个 session 全部为'已修待验证':代码修复在历史 commit 已落地(D-08 正确),40-02 走 dev 浏览器复现 → 通过翻 resolved"
  - "3 个 session 补 slug 字段(workstation-expand / user-filter / vite-hmr):原 frontmatter 缺 slug,翻状态时一并补齐,对齐 D-09 + 保证 40-03 validator 100% pass"
  - "workstation-device-build-errors(Task 1)用 npm run build 验证(非浏览器),因本质是 TS 编译错误(D-05 build 即可)"
  - "plan 引用的 8ad0f4a/16372f5 commit 与部分 session 文件位置不符(api.ts vs authStore.ts),实际修复在正确文件,executor/orchestrator 核查确认"

patterns-established:
  - "验证证据落地:每 session Resolution 章节追加 'Phase 40 Closure' 块,含复测确认 + dev 操作步骤 + verification 时间戳行"

requirements-completed: [TECH-02]

duration: ~25min
completed: 2026-06-25
---

# Phase 40 Plan 02: 5 个 awaiting_human_verify dev 验证闭环

**5 个前端 awaiting_human_verify session 全部 dev 浏览器验证通过(用户操作),代码修复确认在历史 commit 已落地,frontmatter 全翻 resolved 并补齐 3 个缺失 slug。**

## Performance

- **Tasks:** 5/5(1 build 验证 + 4 dev 浏览器验证,全部用户回报"全过")
- **Commits:** 5 个原子 fix(40) commit
- **Build:** 无代码变更(仅 .planning/debug/*.md),build 不受影响;Task 1 npm run build 验证通过(built in 1m 29s)

## Accomplishments

### 5 个 fix(40) commit

| # | Slug | Commit | 验证方式 | 结果 |
|---|------|--------|---------|------|
| 1 | workstation-device-build-errors | `67c1fabd` | npm run build(D-05) | ✓ exit 0,built 1m 29s |
| 2 | workstation-expand-device-load-empty | `6446387c` | dev 浏览器 | ✓ 展开工位加载设备成功 |
| 3 | reset-theme-hardcoded-colors | `06c339f9` | dev 浏览器 | ✓ 恢复 admin 配置色,非硬编码 |
| 4 | user-filter-lost-after-edit | `f97a2fbb` | dev 浏览器 | ✓ 编辑后筛选保留 |
| 5 | vite-hmr-token-blank-page | `a6b314f9` | dev 浏览器 | ✓ HMR 不报 token 错 |

### 修复位置确认(全部历史已落地)

- **workstation-expand**:Layer1 前端 `opsApi.ts:657-672`(getManual/getAD/getAsset + getByWorkstation 兼容别名)+ Layer2 后端 `workstation_device_handler.go` c.Param("id")(line 43/97/124)
- **reset-theme**:`settings/index.tsx` handleReset + 3 处 reset 路径均调 `getDefaultThemeConfig()`(line 96/142/204/223)
- **user-filter**:`useTableManager.ts`(sessionStorage 持久化 + 智能 loadData)+ `usePagination.ts` + `usePersistedState.ts`(新)+ `storage.ts`(TABLE_STATE_PREFIX)
- **vite-hmr**:`authStore.ts`(initializeFromStorage try/finally + clearTokens + logout 去 circular import)+ `DynamicRoutes.tsx`(3 秒 safety setTimeout)
- **workstation-device-build-errors**:antd import 含 Modal + pathId 空值检查

### slug 字段补齐(D-09)

`workstation-expand-device-load-empty` / `user-filter-lost-after-edit` / `vite-hmr-token-blank-page` 原 frontmatter 缺 `slug:`,翻状态时一并补齐,保证 40-03 validator(D-11 必填 slug)对这些文件 100% pass。

## Surprises / Deviations

- **plan 文件引用不准**:40-02 plan 对 vite-hmr 说"前序 commit 已修 api.ts",实际修复在 `authStore.ts` + `DynamicRoutes.tsx`;对 workstation-expand 引用 8ad0f4a(实为引入 bug 的 commit)/16372f5(实为不完整修改的 commit)。修复确实在历史落地,但 plan 的文件指引与真实位置不符,executor/orchestrator 通过核查 Resolution + 实际代码确认。
- **5/5 全部已修待验证**:与 plan 假设一致(D-08),无 session 需要本 plan 实修代码 —— 全部走"验证 → 翻 resolved"。
- **用户一次回报"全过"**:5 个场景用户在单一 dev 会话验证全部通过,无 FAIL 需 D-06 反复修。

## Verification

- `npm run build` 退出码 0(built in 1m 29s)✓
- 5/5 `.planning/debug/<slug>.md` frontmatter `status: resolved` ✓
- 5/5 含 slug 字段 ✓(validator D-11 必填)
- 用户 dev 浏览器验证 4 个运行时场景全部通过 ✓
- audit-open awaiting_human_verify 桶清零(从 5 降到 0)✓

## Next

进入 Wave 3(Plan 40-03):6 个 audit 数据质量文件 frontmatter 规范化 + 2 个验证脚本(validator + verify_phase40)。TECH-04 + TECH-05。
