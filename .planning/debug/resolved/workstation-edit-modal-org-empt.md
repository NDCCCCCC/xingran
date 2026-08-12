---
slug: workstation-edit-modal-org-empt
status: resolved
trigger: "工位管理编辑工位模态框，所属机构为空，没有正确显示，所属部门显示的是最末级机构，需要使用xxx/xxx/xxx的样式显示全部部门"
created: 2026-06-15
updated: 2026-06-15
---

# Workstation Edit Modal — Org/Dept Display Bug

## Symptoms

**S1 — 所属机构 (Organization) field is empty**
- 在工位管理页面，点击编辑按钮打开模态框时，"所属机构"字段不显示任何内容
- 期望：应当显示该工位对应的所属机构信息

**S2 — 所属部门 (Department) shows only leaf node**
- 同一个模态框中，"所属部门"仅显示最末级部门名称（如 "财务部"）
- 期望：使用 `xxx/xxx/xxx` 路径样式显示完整部门链（如 "集团总部/财务中心/财务部"）

**Scope (user-confirmed):**
- 仅出现在**编辑模态框**（点击编辑按钮触发的表单）
- org_id 字段同时支持 Excel 导入（部门名→UUID）和表单部门树选择两种填充方式

## Current Focus

resolution: applied minimal fixes; build/type-check pass.

## Symptoms

expected: |
  Edit modal: "所属机构" shows the building's org; "所属部门" shows full
  department path like "集团总部/财务中心/财务部".
actual: |
  "所属机构" is empty. "所属部门" shows only the leaf name.
errors: none
reproduction: Open workstation list → click edit on any row → observe orgId
  field empty and deptName field shows leaf only.
started: known

## Eliminated

- hypothesis: Backend omits orgId / deptName on the list/get endpoint
  evidence: workstation_service.go:15 includes dept_name in workstationJoinSelect;
    building_service JOINs dept name when applicable; building response has
    orgId/orgName. The data exists in the response shape.
  timestamp: 2026-06-15

## Evidence

- timestamp: 2026-06-15
  checked: EditModal.tsx
  found: form fields `orgId` and `deptId` mapped to DepartmentTreeSelect;
    value is set externally by parent.
- timestamp: 2026-06-15
  checked: workstations/index.tsx setEditFormValues
  found: Fetches building to get orgId, but `setFieldsValue({ ...record,
    floorId: [...] })` did NOT include orgId. FIX APPLIED.
- timestamp: 2026-06-15
  checked: useWorkstationData.ts buildTreeData
  found: Sets `title: dept.deptName || dept.name || ''` — no path.
    FIX APPLIED: rewritten to build full path labels.
- timestamp: 2026-06-15
  checked: DepartmentTreeSelect.tsx
  found: When `treeData` prop is provided, the internal
    `convertDeptTreeData` (which builds full path) is skipped.
- timestamp: 2026-06-15
  checked: WorkstationOps type
  found: Has `deptId`/`deptName`/`buildingId`/`buildingName`/`floorId` etc.,
    but no `orgId` field. orgId comes from the building (via floor→building).
    FIX APPLIED: added optional `orgId?: string` field for edit-form echo.
- timestamp: 2026-06-15
  checked: buildingService and buildingApi
  found: Building.get returns `{ orgId, orgName, ... }` — orgId IS available
    after the buildingApi.get call.

## Resolution

root_cause: |
  S1: setEditFormValues in workstations/index.tsx did not include orgId in the
  setFieldsValue payload, even though it had already fetched it from
  buildingApi.get. The "orgId" form field stayed empty in edit mode.

  S2: buildTreeData in useWorkstationData.ts set the dept tree node title to
  the leaf name only. The DepartmentTreeSelect's built-in path-building helper
  was bypassed because the workstation page passes a pre-converted
  treeData prop, so the title shown for the matched deptId node contained
  only the leaf name.
fix: |
  S1: Explicitly inject `orgId` into the pendingCascaderWrite record (and
      into the immediate setFieldsValue payload for the no-orgId branch),
      so the form's "orgId" field is populated when edit modal opens.

  S2: Rewrite buildTreeData to construct full path labels per node,
      joining the ancestor chain with " / " — yielding titles like
      "集团总部/财务中心/财务部" instead of "财务部".

  Type: Added `orgId?: string` to WorkstationOps interface to allow the
        injected field to type-check cleanly.
verification: |
  - `npm run type-check` in xingran-react-frontend: clean (no errors)
  - `npm run build` in xingran-react-frontend: succeeded
  - `go build ./...` in repo root: exit 0 (no backend changes needed;
    sanity check)
  - No unit tests exist for these frontend files
  - User-side UI verification pending (modal still needs to be opened
    in the running app to confirm the values are echoed as expected)
files_changed:
  - xingran-react-frontend/src/pages/operations/workstations/index.tsx
  - xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts
  - xingran-react-frontend/src/types/operations.ts
