---
status: in-progress
created: 2026-07-03
---

# Quick Task: 资产列表字段一致性 4 项修复

## 范围

按 plan 文件 `C:\Users\CPIC\.claude\plans\glittery-discovering-kernighan.md` 执行 4 个 atomic commits。

## 4 个 Atomic Commits

### Commit 1: P0-1 收尾
- 文件：`xingran-react-frontend/src/pages/operations/assets/columnsSchema.ts`
- 改动：line 72 `useStatusName` → `useStatusLabel`
- 不需要 sync JSON（只有 key 名变化，不影响 schema 结构，但 sync 一下安全）

### Commit 2: P0-2 前端 deptName 错位
- 文件 1：`columnsSchema.ts`
  - line 40 改 `key: "deptName"` → `key: "usefulDeptName"`（label 不变"所属部门"）
  - 新增 `{ key: "deptName", label: "受益部门", visible: false, order: 53, width: 120, group: "部门与用户" }`
- 文件 2：`xingran-react-frontend/src/pages/operations/assets/index.tsx`
  - line 374 改 dataIndex `deptName` → `usefulDeptName`
  - 在 usefulDeptName 之后新增 `deptName` 列渲染（受益部门）
- 同步后端 `internal/services/system/asset_columns_schema.json`

### Commit 3: 新 P0-4 first-non-empty-wins
- 文件：`internal/services/operations/excel_service.go`
  - 行 820-827 `prepareRecordsForUpsert` 加 first-non-empty-wins 逻辑
  - 新增 `isEmptyValue` helper 函数
  - 检查并加 `strings` import
- 文件：`internal/services/operations/excel_service_test.go`（如不存在则新建）
  - 加 first-non-empty-wins 单元测试 5 个场景

### Commit 4: P1-5 列清理补全
- 文件 1：`columnsSchema.ts` 删 31 phantom + 加 22 缺失字段
- 文件 2：`index.tsx` columns 数组同步
- 文件 3：同步后端 `asset_columns_schema.json`

## 不改范围

- DB schema / Go model asset 字段
- Excel config asset 配置
- batch_upsert.go
- validateAndParseRow

## 验证

每 commit 后：
- `go build ./...`
- `go test ./internal/services/operations/... -count=1 -run "Excel|Asset"`
- `cd xingran-react-frontend && npm run type-check`（每 frontend commit 后）

最终验证：
- `npm run lint`（确认与基线一致）
- `npm run sync-columns-schema` 同步 JSON
- 对比 embed JSON 变化

## 参考

- Plan: `C:\Users\CPIC\.claude\plans\glittery-discovering-kernighan.md`
- 三层对齐矩阵: 已诊断完成（43 字段）
