---
phase: 39-workstation-dept-location-alias
plan: 06
subsystem: frontend-workstation-edit-modal
tags: [frontend, workstation, dept-alias, tree-select, react-query]
requires:
  - 39-05 (useAliasByLocation hook + opsApi.deptOptions + DeptOption type)
  - 39-03 (backend union SQL: POST /ops/workstation/dept-options)
provides:
  - EditModal.subDeptTree union 派生 (baseTree + aliasNodes 带 [映射] 后缀)
  - WorkstationEditModalProps.aliasList 接口
  - index.tsx 顶层 watchedOrgId state lift + useAliasByLocation 钩入
affects:
  - 工位编辑模态框"所属部门"下拉 (用户可见 [映射] 后缀节点)
tech-stack:
  added: []
  patterns:
    - "union 派生: baseTree(D-LOCKED) ∪ aliasNodes(Phase 39)"
    - "顶层 state lift (watchedOrgId) → react-query enabled 守护"
    - "三路同步 setWatchedOrgId: 新增 / 编辑 / 重置"
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
decisions:
  - "watchedOrgId 顶层 state lift (三路同步) — 不暴露 EditModal 内部 Form.useWatch"
  - "alias 节点 isLeaf=true + key/value=deptId + is_alias 标记 (不影响渲染)"
  - "alias 顺序在 baseTree 之后 — 保留 Phase 37 D-LOCKED 子树顺序"
metrics:
  duration: 约 12 分钟
  completed: 2026-06-25
  tasks: 2
  files: 2
---

# Phase 39 Plan 06: 工位编辑模态框 subDeptTree union 注入 Summary

EditModal subDeptTree 在 D-LOCKED 基线子树之后追加 alias 映射节点 (带 ` [映射]` 后缀)，index.tsx 顶层 state lift (watchedOrgId) 钩入 useAliasByLocation，三路同步覆盖新增/编辑/重置三个 orgId 来源分支。

## 完成内容

### Task 1: EditModal aliasList props + subDeptTree union 派生 (commit 7a09a73)

**EditModal.tsx**:
- 新增 `import type { DeptOption } from "@/lib/opsApi"`
- `WorkstationEditModalProps` 接口追加 `aliasList?: DeptOption[]` 字段
- props 解构追加 `aliasList`
- `subDeptTree` useMemo 派生改造 (D-LOCKED 语义保留):
  - 基线: `baseTree = node?.children?.length ? trimTitleToLastSegment(node.children) : []` (与 Phase 37 等价)
  - Phase 39 注入: 当 `aliasList` 非空时,在 `baseTree` 之后追加 `aliasNodes`
  - 每个 alias 节点: `title = \`${trimTitleToLastSegment([a.deptName])[0]?.title ?? a.deptName} [映射]\`` (D-01 锁定的 UAT 断言字符串), `value/key = a.deptId`, `isLeaf: true`, 附带 `is_alias: true` 自定义字段
- `useMemo` deps 追加 `aliasList`
- **未修改**: user picker (REQ-39-08 零改动)、floor cascader、form rules、useLayoutEffect

### Task 2: index.tsx useAliasByLocation 钩入 + props 透传 (commit 0862470)

**index.tsx**:
- 新增 `import { useAliasByLocation } from "@/hooks/useAliasByLocation"`
- 顶层新增 `watchedOrgId` state + `const { data: aliasList = [] } = useAliasByLocation(watchedOrgId)`
- **三路 setWatchedOrgId 同步** (覆盖 orgId 所有来源分支):
  1. `handleOrgChange` (新增模式用户选机构) — 在最前同步,空 orgId 时清 undefined
  2. `setEditFormValues` 编辑模式 — 两处 `setFieldsValue(record/orgId)` 旁同步 (floorId 缺失早返分支 + building.get 拿到 orgId 主分支)
  3. `handleOpenModal` 新增模式 — 显式 `setWatchedOrgId(undefined)` 清空上一次编辑残留
- `<WorkstationEditModal>` 调用处透传 `aliasList={aliasList}` (默认 [] 守护)
- **未修改**: useWorkstationModals、Table 列定义、StatisticsCards、DeptSidebar、Cascader 预加载逻辑

## 调查结论 (watchedOrgId 落点)

Plan 中标注的"watchedOrgId 位置不确定"经实地核查澄清:

- `watchedOrgId` 真实落点在 **EditModal.tsx:86** (`Form.useWatch("orgId", form)`),并非在 `useWorkstationModals.ts`
- `useWorkstationModals.ts` 仅负责 open/close + CRUD,不持有表单字段订阅
- 因此采用 **顶层 state lift** 方案: 在 index.tsx 新增独立 `watchedOrgId` state,通过三个回调同步点 (handleOrgChange / setEditFormValues / handleOpenModal) 写入,喂给 useAliasByLocation
- EditModal 内部的 `Form.useWatch("orgId", form)` **保留不变** (subDeptTree useMemo 派生仍依赖它),两者并行不冲突 — 顶层 state 只用于 react-query 启用条件

## 验证

| 检查项 | 命令 | 结果 |
|--------|------|------|
| TypeScript 类型检查 | `npm run type-check` (tsc --noEmit) | 退出码 0,无输出 |
| ESLint (修改文件) | `npx eslint EditModal.tsx index.tsx` | 0 error,2 warning (均为 pre-existing 非本次改动) |
| 全量 lint | `npm run lint` | 3859 problems 全部为 pre-existing (非本次引入) |
| 关键字面量 `[映射]` | (代码内嵌) | EditModal.tsx 内 `\`${trimmed} [映射]\`` 模板字符串存在 |
| 关键调用 `useAliasByLocation` | (代码内嵌) | index.tsx 内调用 + props 透传存在 |

## Deviations from Plan

### 调查期间对 Plan 推测代码的适配 (非规则触发的计划调整)

**1. watchedOrgId 三路同步 (超出 Plan 单点 onOrgChange 建议)**

- **Plan 建议**: 仅在 EditModal 的 `onOrgChange` 回调里 `setWatchedOrgId(orgId)`
- **实际情况**: `setEditFormValues` (编辑模式) 通过 `building.get` 拿到 orgId 后直接 `workstationForm.setFieldsValue({ ...record, orgId })`,**完全绕过 onOrgChange**,且 EditModal 的 onOrgChange 仅在用户主动改机构时触发
- **适配**: 增加两处额外同步点 (`setEditFormValues` 主分支 + 早返分支) + `handleOpenModal` 新增模式重置,否则编辑既有工位时 alias 下拉会缺失
- **类型**: 计划调整 (Plan 推测代码 vs 真实代码差异),非 Rule 1-4 触发

**2. 未实现 onAliasInvalidate callback (Plan 已声明可省略)**

- Plan 明确写出:"不需要 onAliasInvalidate callback,alias 失效由 useInvalidateDept 一把全清处理"
- 本 plan 仅交付基础注入,Drawer CRUD 失效逻辑在 Plan 39-07 阶段实现
- 符合 Plan 预期,无偏差

## Known Stubs

无 — 本 plan 所有改动均连接到真实数据源 (`useAliasByLocation` → `workstationApi.deptOptions` → 后端 union SQL),无 placeholder / mock 数据。

## Threat Flags

无 — 本 plan 未引入新的网络端点、auth 路径、文件访问模式或信任边界 schema 变更。仅消费 Plan 39-03 已交付的 `POST /ops/workstation/dept-options` (后端已鉴权)。

## Self-Check: PASSED

- `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx` 存在 (FOUND via git diff)
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` 存在 (FOUND via git diff)
- commit `7a09a73` 存在 (FOUND via `git log --oneline -3`)
- commit `0862470` 存在 (FOUND via `git log --oneline -3`)
- `npm run type-check` 退出码 0 (PASSED)
- `npx eslint <修改文件>` 0 error (PASSED)
