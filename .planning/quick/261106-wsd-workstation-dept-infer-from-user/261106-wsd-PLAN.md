---
quick_id: 261106-wsd
slug: workstation-dept-infer-from-user
status: planned
date: 2026-11-06
---

# 工位导入：部门填写不规范时按所属人员反推部门（仅兜底）

## 用户决策（已锁定，不再重问）

- **优先级**：仅兜底。只有 `deptCode` / `deptName` 都没能解析出 `dept_id` 时，才用已解析的 `user_id` 反推部门；用户显式填对的部门保持不变。
- **实施范围**：现在实现，含回归测试。
- **提交**：不自动 commit。按项目 CLAUDE.md「提交前必须问用户」，完成后交 diff 由用户决定。

## 背景

工位 Excel 导入中 `dept_id` 与 `user_id` 是两条独立引用解析链（`excel_config.go:169-171`）。部门名称填写不规范时解析失败，而 `prepareRecordsForUpsert:939` 明确「引用解析失败不使用原始值」→ 静默不关联、不报错（部门列非必填，`validateReferenceFields:1180` 只校验必填引用）。

`sys_user.dept_id` 直接指向 `sys_dept.id`（`models/user.go:20`），项目已有现成反查范式：`workstation_device_service.go:989-1006` 用 `sys_user.id` LEFT JOIN `sys_dept` 取部门。

## 实现（1 个文件，最小改动）

文件：`internal/services/operations/excel_service.go`

1. 新增 `inferDeptFromUser(ctx, records, config) int`：
   - 仅当 `config.TableName == "sys_workstation"` 时执行；其他实体零影响。
   - 收集「`dept_id` 为空」且「`user_id` 非空」的 `user_id`，去重。
   - 一次批量查询：`SELECT su.id, su.dept_id FROM sys_user su INNER JOIN sys_dept d ON CAST(d.id AS TEXT) = CAST(su.dept_id AS TEXT) WHERE su.id IN (?) AND su.deleted_at IS NULL AND su.dept_id IS NOT NULL AND su.dept_id <> ''`，按 400 分块（沿用 `reference_resolver.go:282` 的 `maxBatchChunk` 约定）。
   - **JOIN 必须两侧 CAST 成 TEXT**：`sys_dept.id` 是 uuid、`sys_user.dept_id` 是 varchar(64)，PG 无 `uuid = varchar` 运算符（见 `migration_217_dept_ou_mismatch_view.go:201-206` 的既有范式）。
   - INNER JOIN `sys_dept` 天然过滤掉「dept_id 是部门名称 / 悬空 UUID」等脏数据，不需要额外的 UUID 正则。
   - 仅回填 `dept_id` 为空的记录（first-non-empty-wins，与 `applyReferenceResults:650` 一致），返回回填条数。
2. 在 `ImportData` 中 `validateReferenceFields` 之后、geocoding/事务之前调用；把条数写入 `ImportResult.InferredDeptCount`（新增 `json:"inferredDeptCount,omitempty"` 字段），便于审计与前端展示。
3. `logger.Infof` 记录回填条数；不产生 `ImportError`（这是兜底增强，不是失败）。

## 回归测试（新文件 `excel_workstation_dept_infer_test.go`）

复用 `setupImportRest77DB(t)`（`excel_import_rest_77_03_test.go:42`）与 `createWorkstationExcelFile(t, headers, rows)`（`excel_workstation_import_fix_test.go:18`）；工位表按既有测试 `AutoMigrate` + 唯一索引。

| 用例 | 断言 |
|---|---|
| 部门填错 + 有效用户 | `dept_id` = 用户所属部门；`user_id` 正确；`InferredDeptCount == 1` |
| 用户无 `dept_id`（NULL） | 不回填，`dept_id` 为空，不报错 |
| 用户 `dept_id` 是脏值（部门名称 / 不存在 UUID） | 不回填（JOIN 过滤），不报错 |
| 显式填对部门 | 显式部门优先，不被用户部门覆盖；`InferredDeptCount == 0` |
| 部门填错 + 未填用户 | 不回填（无 user_id 可反推），不报错 |
| 多用户混合 | 部分回填、部分跳过，计数准确 |

## 验证

- `go build ./...`
- `go test ./internal/services/operations/ -run 'TestWorkstationImport|TestDeptInfer' -count=1`
- `go test ./internal/services/operations/ -count=1`（全包回归）

## 不做

- 不修改前端（反推部门经既有 `sys_dept` LEFT JOIN 正常回显，`workstation_service.go:21`）。
- 不改 `excel_config.go` 的列定义 / 表头 / 模板。
- 不动 `batch_upserter.go`（`standardUpsert` 的 `ON CONFLICT DO UPDATE` 会把未提供的 `dept_id` 写成 NULL 是既有行为；反推回填恰好缓解「部门留空重导」，但不在本次范围内改动 upsert 语义）。
- 不碰无关的未提交改动 `internal/services/system/asset_columns_schema.json`。
