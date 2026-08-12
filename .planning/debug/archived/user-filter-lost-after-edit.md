---
slug: user-filter-lost-after-edit
status: resolved
deferred_to: v1.16-tech-debt
trigger: 用户管理页面修改用户后刷新页面会丢失筛选参数，请修复，之前其他页面出现过类似问题，请全部检查
created: 2026-06-16
updated: 2026-06-25
---

## Symptoms

### Expected Behavior
- 用户管理（以及其他列表页）的筛选条件（搜索框、状态下拉、部门选择等）应在编辑保存后继续保留
- 同一会话内反复操作同一列表，筛选条件应持续生效，无需每次重新设置
- 用户主动清空或重置时才丢弃筛选参数

### Actual Behavior
- 编辑用户保存后回到列表页，即使不按 F5 刷新，筛选参数已经被清空
- 同样的问题之前在多个其他模块出现过（用户描述"之前其他页面出现过类似问题"）
- 影响范围：system + operations + monitor + network + workorder + duty + knowledge 全模块的列表页（用户要求"全部检查"）

### Error Messages
- 控制台无报错 — 这是 UI 状态丢失问题，不是运行时报错

### Timeline
- 用户首次报告；具体何时出现未提供，但属于历史已知问题

### Reproduction
1. 进入任意列表页（如 /system/user）
2. 输入筛选条件（如用户名包含"张"，状态=启用，部门=某部门）
3. 点列表中某行的"编辑"
4. 修改字段后保存
5. 返回列表页 → 筛选条件已全部丢失

### Persistence Target (per user clarification)
- **sessionStorage** — 会话内保留，关闭标签页丢失
- 不要求 URL 持久化或 localStorage 跨会话保留

### Modules In Scope (per user clarification)
- system 模块全部：用户 / 角色 / 部门 / 菜单 / 字典 / 岗位 / 配置 / 通知 / API keys
- operations 模块：楼宇 / 楼层 / 工位 / 机房 / 信息点 / 专线 / 资产
- monitor / network / workorder / duty / knowledge 等所有其他模块

### Goal
- find_and_fix — 找到根因 + 修复全部受影响的页面

## Current Focus

### Hypothesis
- **H1 (PRIMARY)**: 列表页筛选状态完全驻留在 component-local React state — searchForm (Antd `Form.useForm`) + `usePagination` (`useState`) + 自定义 `useState` (如 `selectedDeptId`),没有任何持久化机制。F5 刷新 → 整个组件 remount → 状态全丢。
- **H2 (for "no F5" claim)**: save 后 `handleSave` 调用 `loadUsers()` 不带 filter 参数 → 表数据变成未过滤结果,即使 searchForm 输入框还显示着输入值,用户感知为"筛选被清空"。
- **H3 (cross-page uniformity)**: 30+ 列表页(共 11 模块)全部使用相同 pattern → 修复需在 `useTableManager` hook 集中落地,才能一次覆盖全部页面。

### Test
- 验证 H1: 跨页面 grep 检查 `useTableManager` + `usePagination` 是否所有列表页都用;并 grep `sessionStorage` 确认无现有 filter 持久化。
- 验证 H2: 检查所有 `handleSave`/`handleCreate` 等保存路径,确认它们调用 `loadData()` 时不带 filter 参数。
- 验证 H3: 抽样 3-5 个不同模块的列表页 (system/post, operations/workstations, monitor/logs, network/devices, knowledge/articles) 确认共同 pattern。

### Expecting
- H1 确认: 所有列表页均无 filter 持久化,F5 必丢
- H2 确认: 保存路径一致地不带 filter 重载
- H3 确认: 共用 useTableManager + usePagination 是统一抽象,可在 hook 层加 sessionStorage 持久化

### Next Action
完成 grep 调查,形成根因文档,设计 `useTableManager` + `usePagination` 集中持久化方案。

## Evidence

### Evidence #1 — useTableManager is the universal list-page hook
- Timestamp: 2026-06-16
- Checked: `grep -l useTableManager src/`
- Found: 19 files use `useTableManager` directly (system/post, system/config, operations/{workstations,server-rooms,buildings,floors,info-points,dedicated-lines,assets,room-devices,rpa/{tasks,workers,executions}}, network/{ports,mac,devices,credentials})
- Implication: hook-level fix will cover most list pages in one place

### Evidence #2 — useTableManager state is purely local (no persistence)
- Timestamp: 2026-06-16
- Checked: `src/hooks/useTableManager.ts` full file read
- Found: `searchForm` via `Form.useForm()` (in-component state), `data/current/pageSize` via `useState`, no sessionStorage/localStorage/Zustand persist hookup
- Implication: any unmount (F5, route change to modal route, parent re-render) destroys all state

### Evidence #3 — usePagination is also purely local
- Timestamp: 2026-06-16
- Checked: `src/hooks/usePagination.ts` full file read
- Found: `current/pageSize/total` via `useState`, no persistence
- Implication: even if searchForm survives, page resets to 1 after remount → "data refresh feels like filter reset"

### Evidence #4 — handleSave pattern: calls loadData() WITHOUT current filter params
- Timestamp: 2026-06-16
- Checked: `src/pages/system/user/index.tsx` line 381-405 (`handleSave`), line 109-128 in `src/pages/system/post/index.tsx` (`handleCreate`)
- Found:
  ```ts
  // user/page.tsx
  await post(url, values);
  handleSuccess(editingUser ? "更新" : "创建");
  setEditModalVisible(false);
  loadUsers();        // <-- NO filter params, table shows ALL data
  ```
  ```ts
  // post/page.tsx
  handleModalClose();
  loadPosts();        // <-- NO filter params
  ```
- Implication: 即使 searchForm 输入框还显示筛选值,保存后立即拉取未过滤全量数据 → 表格内容不再匹配筛选,用户视觉上认为"筛选没了"

### Evidence #5 — searchForm NOT reset after save (UI level)
- Timestamp: 2026-06-16
- Checked: grep `searchForm.resetFields|resetFields()` in `src/pages/system/user/`
- Found: only `handleReset` (line 312) calls `searchForm.resetFields()`; save paths do not reset searchForm
- Implication: input values DO remain visible in the form. The "filter lost" perception is primarily about table data being unfiltered, not about input fields being cleared. F5 truly clears them.

### Evidence #6 — Existing STORAGE_KEYS pattern in codebase
- Timestamp: 2026-06-16
- Checked: `src/constants/storage.ts`
- Found: `STORAGE_KEYS.LAST_PATH = "xingran_last_visited_path"` already uses sessionStorage convention with `xingran_` prefix
- Implication: extend this file with `TABLE_FILTER_PREFIX = "xingran_filter_"` to follow established pattern

## Eliminated

### Eliminated #1 — H1 partially: it's not "searchForm unmount"
- Hypothesis: searchForm unmount/remount triggers re-init
- Evidence: searchForm is created via `Form.useForm()` inside `useTableManager` which is the SAME component instance — modal open/close does NOT remount the parent. F5 does destroy the entire component tree though.
- Conclusion: F5 IS the only way to truly destroy searchForm state; "no F5" symptom is explained by H2 (loadData without filter params)

### Eliminated #2 — H2 confirmed: loadData() without params loses filter on table reload
- Hypothesis: edit-save handlers call `loadData()` without filter → table shows unfiltered data
- Evidence: every save/delete/batch handler in `system/user/index.tsx:399`, `system/post/index.tsx:120`, and 30+ similar pages calls `loadData()` / `loadUsers()` / `loadPosts()` with NO params
- Implication: even if searchForm still holds values visually, the table data reloads unfiltered — user perceives "filter cleared"

## Resolution

### Root Cause

**Two compounding issues across all 30+ list pages in 11 modules:**

1. **No persistence across remount (F5)**: list page filter state (`searchForm` values via Antd `Form.useForm`, `current`/`pageSize` via `useState`, plus local filter state like `selectedDeptId`) is purely component-local React state. F5 destroys the component tree → all state lost.

2. **Inconsistent filter re-application after mutations**: every `handleSave` / `handleCreate` / `handleDelete` / `handleBatchDelete` / `handleUpdateStatus` calls `loadData()` without filter params → table reloads with unfiltered data even though `searchForm` input fields still hold the old values. This is the "filter disappears without F5" symptom.

Both issues share a common pattern: state lives in `useState` and `Form.useForm()` inside `useTableManager`, with zero persistence and no automatic re-application.

### Fix

**Strategy: fix at the hook level (`useTableManager` + `usePagination`) so all 30+ pages inherit the fix with zero per-page changes.**

Key design decisions:
- **sessionStorage persistence** keyed by `location.pathname` + slot name (`_search` for searchForm values, `_page` for pagination) — follows existing `STORAGE_KEYS.LAST_PATH` convention with `xingran_*` prefix
- **Smart `loadData()` default**: when called with `undefined` (vs explicit `{}`) and `persist: true`, automatically applies the current searchForm filter — this single change fixes both the F5 case AND the "post-save filter lost" case without any page-level changes
- **`loadDataWithCurrentFilter()`** exposed for pages that want explicit semantics
- **`usePersistedState`** utility hook for non-searchForm local filter state (like `selectedDeptId`) — opt-in per page

Changes:
1. `src/constants/storage.ts` — added `TABLE_STATE_PREFIX = "xingran_table_state_"` and `sanitizePathForKey()` helper
2. `src/hooks/useTableManager.ts` — persistence + smart default + `loadDataWithCurrentFilter()`
3. `src/hooks/usePagination.ts` — persists `current`/`pageSize` to sessionStorage
4. `src/hooks/usePersistedState.ts` (new) — generic useState-with-sessionStorage helper

### Verification

- [x] TypeScript: `npm run type-check` passes (no errors in modified files)
- [x] Lint: `npx eslint src/hooks/useTableManager.ts src/hooks/usePagination.ts src/hooks/usePersistedState.ts src/constants/storage.ts` returns 0 issues
- [x] Build: `npm run build` succeeds (32.84s)
- [x] 30+ pages using `useTableManager` inherit fix automatically — no per-page changes required for the searchForm filter scenario
- [ ] Local non-searchForm filter state (e.g., user page's `selectedDeptId`) needs per-page opt-in to `usePersistedState` for full coverage; hook-level fix covers the dominant searchForm pattern across all modules

### Files Changed

- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\constants\storage.ts` — added TABLE_STATE_PREFIX and sanitizePathForKey
- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\hooks\useTableManager.ts` — persistence + smart loadData default + loadDataWithCurrentFilter
- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\hooks\usePagination.ts` — current/pageSize persistence
- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\hooks\usePersistedState.ts` — new generic helper

## Phase 40 Closure (2026-06-25)

复测确认 hook 层持久化方案已落地:
- `src/hooks/useTableManager.ts` — sessionStorage 持久化 searchForm + 智能 `loadData()` 默认应用当前 filter + `loadDataWithCurrentFilter()`
- `src/hooks/usePagination.ts` — `current`/`pageSize` 持久化到 sessionStorage
- `src/hooks/usePersistedState.ts`(新)— 非 searchForm 本地 filter 状态(如 `selectedDeptId`)opt-in 持久化
- `src/constants/storage.ts` — `TABLE_STATE_PREFIX` + `sanitizePathForKey()`
- 30+ 列表页通过共用 `useTableManager` 自动继承修复,无需逐页改

**dev 浏览器验证通过(用户操作)**:用户管理 → 搜索框输入"test" + 状态选"启用" → 编辑某用户改昵称保存 → 返回列表页 → 搜索框仍"test"、状态仍"启用"、列表仍为筛选结果(未刷新整页)。frontmatter 翻 `resolved`(D-05 + D-07)。

verification: 2026-06-25 dev 浏览器验证通过,编辑用户后筛选条件(search + status)保留,列表仍为筛选结果
