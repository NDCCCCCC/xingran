---
slug: role-hooks-params-missing
name: role-hooks-params-missing
description: Frontend tsc -b fails — hooks/index.ts re-exports UseRoleDataParams but useRoleData.ts no longer defines it
status: resolved
trigger: "TypeScript build error: src/pages/system/role/hooks/index.ts:8:15 - error TS2305: Module '"./useRoleData"' has no exported member 'UseRoleDataParams'. The hooks/index.ts exports UseRoleDataParams, UseRoleDataReturn, RoleStatistics from ./useRoleData, but useRoleData.ts does not export UseRoleDataParams. Frontend build failed."
created: 2026-06-18
updated: 2026-06-18
resolved: 2026-06-18
---

# Debug Session: role-hooks-params-missing

## Symptoms (gathered)

1. **Expected behavior**: `.\build-linux.bat` 跑通，最终产出 Linux 嵌入式可执行。前端 `tsc -b && vite build` 应成功。
2. **Actual behavior**: 前端构建在 [3/5] 步失败，`tsc -b` 报 1 个 error，build 中断。
3. **Error message**:
   ```
   src/pages/system/role/hooks/index.ts:8:15 - error TS2305:
   Module '"./useRoleData"' has no exported member 'UseRoleDataParams'.

   8 export type { UseRoleDataParams, UseRoleDataReturn, RoleStatistics } from "./useRoleData";
                      ~~~~~~~~~~~~~~~~~

   Found 1 error.
   Frontend build FAILED
   ```
4. **Timeline**: 最近一次 commit `e215e3b feat(table): 增强 useTableManager 加排序，全项目统一列表数据层` 之后才出现的（重构 `useRoleData` 移除了参数）。
5. **Reproduction**: 在仓库根目录运行 `.\build-linux.bat`（PowerShell 上下文） 或在 `xingran-react-frontend/` 跑 `npm run build`。

## Current Focus

```yaml
hypothesis: barrel 文件 hooks/index.ts:8 重导出了已被 commit e215e3b 移除的 UseRoleDataParams 类型。修复：从重导出语句中删除 UseRoleDataParams（已 grep 确认全前端无任何消费者）。
test: cd xingran-react-frontend && npm run build
expecting: tsc -b 通过、vite build 成功产出 dist/
next_action: resolved — fix applied and build verified
```

## Evidence

- 2026-06-18: `git diff HEAD~5..HEAD -- xingran-react-frontend/src/pages/system/role/hooks/` 显示 useRoleData.ts 在 e215e3b 中:
  - 删除了 `export interface UseRoleDataParams { current; pageSize; }`
  - 函数签名从 `useRoleData(params: UseRoleDataParams)` 改为 `useRoleData()`
  - 同时把 `roles/loading/total/loadRoles/loadRoleMenus/loadRoleDepts` 移到了 useTableManager
- 2026-06-18: `grep -r "UseRoleDataParams" xingran-react-frontend` 全仓唯一一处就是 `hooks/index.ts:8` 的 re-export 本身。无任何 import 站点。
- 2026-06-18: session manager 再次 grep `UseRoleDataParams` 全前端，**唯一一处**就是 `hooks/index.ts:8`，确认零消费者。
- 2026-06-18: `npm run build` 两次均通过：5700 modules transformed，dist 全部产物生成，无 TS error。

## Eliminated

(无 — 根因直接由 git diff + grep 锁定，无需排除其他假设)

## Specialist Review

N/A — build-only re-export cleanup, no specialist dispatch needed (specialist_dispatch_enabled=true 但 goal=find_and_fix 且为 TypeScript 类型层面修复，无 idiomatic concerns 需外部审查)。

## Resolution

- **root_cause**: `xingran-react-frontend/src/pages/system/role/hooks/index.ts:8` 的 barrel 重导出语句仍引用 `UseRoleDataParams`，但 commit `e215e3b` 重构 `useRoleData` 后该 interface 已被删除（`useRoleData()` 改为零参，list/loading/total 移至 `useTableManager`），导致 `tsc -b` 报 TS2305 找不到导出。
- **fix**: 从 `hooks/index.ts:8` 的 `export type` 语句中删除孤儿 `UseRoleDataParams,`，重导出语句变为 `export type { UseRoleDataReturn, RoleStatistics } from "./useRoleData";`，与 `useRoleData.ts` 的实际导出对齐。
- **verification**:
  - `grep -r "UseRoleDataParams" xingran-react-frontend/` → 零匹配（fix 前唯一匹配就是 re-export 行；fix 后无残留引用）
  - `cd xingran-react-frontend && npm run build` → `tsc -b` 0 error，`vite build` 成功（`✓ 5700 modules transformed.` + `✓ built in 46.46s`），`dist/` 全量产物生成
  - 第二次 build 复测 2m 18s 全量构建同样通过
- **files_changed**:
  - `xingran-react-frontend/src/pages/system/role/hooks/index.ts` (line 8 — 移除 `UseRoleDataParams,`)

## Findings (not fixed, scope-constrained)

无额外发现。`grep` 仅定位到本会话目标的一行 re-export，无其他孤儿重导出残留。
