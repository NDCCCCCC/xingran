---
quick_id: 261106-wsd
status: complete
date: 2026-11-06
commit: none (按项目约定，提交前须经用户确认)
---

# 完成：工位导入部门填写不规范时按所属人员反推部门（仅兜底）

## 实现

- `internal/services/operations/excel_service.go`
  - `ImportResult` 新增 `InferredDeptCount`（`json:"inferredDeptCount,omitempty"`），用于审计反推条数。
  - 新增 `inferDeptFromUser(ctx, records) int`：仅工位导入、仅填补 `dept_id` 空缺，不覆盖显式填对的部门。批量 `IN` + `INNER JOIN sys_dept` 一次查询，天然过滤 `dept_id` 为 NULL / 部门名称 / 悬空 ID 的脏数据；按 400 分块；查询失败只记日志不阻断导入。JOIN 两侧 `CAST(... AS TEXT)`，规避 PG `uuid = varchar` 无运算符问题（范式见 `migration_217:201-206`）。
  - `ImportData` 在引用解析与 `validateReferenceFields` 之后调用该 hook（此时 `user_id` 已是 UUID）。
- `internal/services/operations/excel_workstation_dept_infer_test.go`（新）：6 个回归用例 —— 兜底生效 / 用户无部门 / 用户部门脏值 / 显式部门优先 / 未填用户 / 混合批次计数。
- `xingran-react-frontend/src/components/shared/ExcelImport.tsx`
  - `ImportResult` 加可选 `inferredDeptCount`；结果区在 >0 时显示「部门按人员反推: N」Tag，让兜底效果可感知。

## 未改动

- `excel_config.go` 列定义 / 表头 / 模板；`batch_upserter.go` upsert 语义；后端 status 联动；无关的 `internal/services/system/asset_columns_schema.json`（原有未提交改动，未触碰）。
- **未自动 commit**：项目 CLAUDE.md 要求提交前询问用户，diff 交由用户决定。

## 验证

| 命令 | 结果 |
|---|---|
| `go build ./...` | PASS |
| `go vet ./internal/services/operations/` | PASS |
| `go test ./internal/services/operations/ -run TestWorkstationImport_DeptInferFromUser -count=1 -v` | 6/6 PASS |
| `go test ./internal/services/operations/ -count=1` | PASS (5.034s) |
| `npm run type-check`（xingran-react-frontend） | PASS |
| `npx vitest run ExcelImport.render + ExcelImportLazy` | 10/10 PASS |

## 已知边界（非本次范围）

- `standardUpsert` 的 `ON CONFLICT DO UPDATE` 会把记录 map 中缺失的 `dept_id` 写为 NULL，因此「部门列留空重导已有工位」仍可能清空既有部门；反推回填缓解了「填了用户」的子集，但未改变 upsert 语义。若需彻底解决，应给工位配置 `PartialUpdate: true` 或让 `buildUpdateColumns` 跳过空值列 —— 需单独评估回归面。
- `sys_user` 姓名同名时命中谁由既有双向匹配决定，反推继承该结果，不加剧歧义。
