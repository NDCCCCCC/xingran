---
slug: role-custom-datascope-dept-tree-missing
status: resolved
trigger: 编辑角色时数据范围选"自定义数据"，应出现组织架构树供选择，实际没有
created: 2026-11-16
updated: 2026-11-16
---

# Debug: 角色自定义数据权限不显示部门树

## Symptoms

- Expected: 数据范围下拉选"自定义数据"(value=2) 后，"部门权限"Tree 出现
- Actual: Tree 始终不出现
- Error: 无（纯前端状态问题）
- Repro: 权限管理 → 编辑角色 → 数据范围切换为"自定义数据"

## Current Focus

（已解决）

## Evidence

- timestamp: 2026-11-16 — index.tsx:444 `{currentDataScope === 2 && (...)}` — 树的显示完全由 currentDataScope 状态驱动
- timestamp: 2026-11-16 — useRoleActions.ts handleDataScopeChange 原实现只有 `if (value !== 2) editForm.setFieldsValue({ deptIds: [] })`，无 setCurrentDataScope
- timestamp: 2026-11-16 — deptTree 数据无问题：页面初始化即调用 loadDeptTree()（index.tsx:208）

## Eliminated

- hypothesis: deptTree 为空 — 排除，初始化已加载
- hypothesis: 后端接口问题 — 排除，纯前端条件渲染

## Resolution

- root_cause: `handleDataScopeChange` 切换数据范围时未调用 `setCurrentDataScope(value)`，`currentDataScope` 停留在初始值 1，`currentDataScope === 2` 的条件渲染永远不成立，部门树无法出现
- fix: useRoleActions.ts `handleDataScopeChange` 增加 `setCurrentDataScope(value)`；顺带在非 2 时同步清空 `setCheckedDeptKeys([])`（避免残留勾选态在再次切回 2 时闪现）；补依赖数组
- verification: `npx vitest run src/pages/system/role/` 7 文件 38 测试全过（含新增回归测试「handleDataScopeChange 同步 currentDataScope」）；`tsc --noEmit` 通过
