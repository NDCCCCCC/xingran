---
phase: quick
plan: 261013-lqb
subsystem: operations
tags: [operations, excel-import, migration, data-consistency, total-floors]
requires:
  - 261013-jf1 (楼栋列表 FloorCount 显示层免疫 — 本任务根治写路径)
provides:
  - floor Update 换楼双侧重算 + 普通编辑自愈重算
  - Excel 导入 building/floor 双实体收尾全量重算 (recomputeAllBuildingTotalFloors)
  - Migrate213 存量漂移一次性归正 (PG + sqlite 双方言注册)
affects:
  - ops_buildings.total_floors 消费方 (排序白名单 / BuildingCard / 概览抽屉 / Excel 导出)
tech-stack:
  added: []
  patterns: [关联子查询批量 UPDATE (BatchDelete SQL 形状复用), 导入事务收尾幂等重算]
key-files:
  created:
    - internal/services/operations/floor_update_total_floors_test.go
    - internal/services/operations/excel_total_floors_recompute_test.go
    - internal/core/db/migrations/migration_213_recompute_building_total_floors.go
    - internal/core/db/migrations/migration_213_recompute_building_total_floors_test.go
  modified:
    - internal/services/operations/floor_service.go
    - internal/services/operations/excel_service.go
    - internal/core/db/database.go
decisions:
  - 重算只用实时 COUNT (软删排除), 禁止 ±1 增量 (防历史脏值固化, 幂等自愈)
  - Update 不做事务重构 (乐观锁已保证换楼原子, 重算幂等 + 重试自愈)
  - Excel 导入重算放事务内, error 上抛回滚全量导入 (P2-A8 all-or-nothing)
  - 存量修复用一次性数据迁移 Migrate213, 零 DB TRIGGER / 常驻 schema object
  - 楼栋表单手工写值路径明确不修 (scope constrainment, 显示层已免疫 + 事件自愈)
metrics:
  duration: ~21 min
  completed: 2026-10-13
---

# Quick 261013-lqb: 根治 ops_buildings.total_floors 反范式列漂移 Summary

**One-liner:** 楼层 Update 换楼双侧重算 + 普通编辑自愈、Excel 导入 building/floor 双实体事务收尾全量重算、Migrate213 存量归正——三条写路径漂移全部关闭, 零 TRIGGER / 零前端 / 零 handler 改动。

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | floor Update 换楼双侧重算 + 普通编辑自愈 + 单测 | d41de5e4 | floor_service.go, floor_update_total_floors_test.go (new) |
| 2 | Excel 导入重算 hook (building+floor) + 共享 helper + 单测 | 4746cdc1 | floor_service.go, excel_service.go, excel_total_floors_recompute_test.go (new) |
| 3 | Migrate213 存量重算 + database.go 双方言注册 + 迁移单测 | 3a3163a6 | migration_213 x2 (new), database.go |

## Implementation Notes

- **路径 A (Update 换楼)**: `floor_service.go` `Update` 末尾 (goroutine 同步块之后) 追加 `updateBuildingFloorCount` — 当前楼栋必算, 换楼再算旧楼; error 上抛对齐 Delete fail-loud; 乐观锁分支 / Save / goroutine 块零触碰。
- **路径 B/C (Excel 导入)**: 包级 `recomputeAllBuildingTotalFloors(ctx, db)` — SQL 形状逐字拷贝 `BatchDelete` 关联子查询, 去 id 限定 + `WHERE deleted_at IS NULL` (软删楼栋跳过); `ImportData` 事务块内 user 默认角色 hook 之后触发 (`entityType == "building" || "floor"`), error `return err` 回滚全量导入。
- **存量修复**: `Migrate213RecomputeBuildingTotalFloors` 同一条幂等 UPDATE; database.go PG advisory-lock 块内 + sqlite 分支双注册 (Migrate212 之后), 非阻断模式; 无 `isPostgreSQL` 守卫 (纯 SQL 双方言通用)。

## Verification (真实输出)

| 命令 | 结果 |
|------|------|
| `go build ./...` | exit 0 (无输出, 每任务后均跑) |
| `go test ./internal/services/operations/ -run TestFloorUpdateTotalFloors -v` | 3/3 PASS (BuildingChangeRecountsBothSides / SameBuildingEditSelfHeals / MoveLastFloorOldBuildingZero) |
| `go test ./internal/services/operations/ -run TestRecomputeAllBuildingTotalFloors -v` | PASS (软删楼层排除 / 软删楼栋保持 777 / 二次幂等) |
| `go test ./internal/core/db/migrations/ -run TestMigrate213 -v` | PASS (脏值 99/0 归正为 1/1 + 二次幂等) |
| `go test ./internal/services/operations/` (全量) | ok 6.465s (最终) — 既有 `floor_service_crud_test.go` total_floors 编排用例与 `excel_import_rest_77_03_test.go` 零回归 |
| `go test ./internal/core/db/...` (最终门禁) | ok: core/db 7.435s + migrations 1.269s |

共新增 5 个测试用例。无人工 UAT (按 PLAN: 写路径语义由测试锁定, 消费方零改动)。

## Deviations from Plan

### 1. [中断恢复] 删除中断期产生的计划外测试文件

- **Found during:** 恢复执行时 (协调者通知 API 中断)
- **Issue:** 中断前已创建 `internal/services/operations/floor_move_building_test.go` (未跟踪) — 内容完整但非 PLAN 指定文件 (4 个换楼/乐观锁测试), 且含空断言 (`StaleReadReturnsError` 的 `rowsAffected` 局部变量从未赋值, `assert.Equal(t, int64(0), rowsAffected)` 恒真) 与名不副实用例 (`OptimisticLockError` 实际断言 NoError)
- **Fix:** 删除该文件 (本任务自产, 非用户工作), 按 PLAN 落地指定测试 `floor_update_total_floors_test.go` (3 用例, 复用 newFloor91TestDB 基建)
- **Commit:** d41de5e4 (净效果与 PLAN 一致)

### 2. [文件级 staging 卷入] excel_service.go 携带先前未提交修改入库 (4746cdc1)

- **Issue:** 本任务修改 ~11 行 (hook), 文件级 staging 另卷入先前未提交修改 ~30 行: QUIRK-77-2 合并单元格锚点按行计算修复、`models.DeptStatusNormal` 常量替换魔法数字 (2 处)、Scan struct 字段 tag 对齐空白
- **Fix:** 按环境约定不做 partial staging, 整文件入库, 在此披露 (先例: 261012-l7k / 261013-jf1)
- **Commit:** 4746cdc1

### 3. [文件级 staging 卷入 - 重大] database.go 携带先前未提交 DB 层大重构入库 (3a3163a6)

- **Issue:** 本任务在 database.go 仅改 2×4 行 (Migrate213 双注册), 但该文件携带先前未提交大重构 (715+/1003-): 连接池 keepalive (`startPoolKeepalive`/`poolKeepaliveLoop`/`pingPoolOnce`)、`FilterLogger`、`sanitizeSQLiteModelDefaults`、`MigrateModelList()` 抽取、advisory-lock 方法化、`InitData`→`BootstrapMissingTables` 拆分 (拆出的 `init_data.go`/`sqlite_reconciliation_views.go`/migrations 205-212 系列仍是未跟踪文件)、migrations 205-212 注册块 (HEAD 基线为零注册)
- **Fix:** 按环境约定整文件入库 (禁止 partial staging), 在此披露。**注意**: 重构依赖的未跟踪文件未随之入库 → 提交 3a3163a6 不可独立构建 (checkout 该 commit 单独 `go build` 会因缺 init_data.go 等失败); 工作树状态全绿 (最终门禁实测)。261013-lqb 在该文件的净变更仅为 Migrate213 双注册
- **Commit:** 3a3163a6

### 测试编写中的迭代修正 (非偏离, 如实记录)

- Task 2 测试: GORM `Update` 自动附 `deleted_at IS NULL` scope → 软删 B3 后置脏 777 是静默 0 行, 改为先置脏再软删; 软删楼栋断言需 `Unscoped()` 读取
- Task 3 测试: seed 初版误建 B1=3 行楼层 (2 活 1 软删 → 活行数 2), 按计划语义「2 层含 1 软删 = 活行数 1」修正为 2 行

## Known Stubs

None — 无占位值/空数据源/TODO。

## Intentional Residual (per PLAN scope constrainment)

- **楼栋表单手工 Create+Update 路径** (`buildings/index.tsx:792` → building_service) 仍写用户声明值 — planning context 未列此路径, 不扩 scope; 显示层已免疫 (jf1 FloorCount), 任意后续楼层事件或 Excel 导入经本任务重算路径自愈该楼栋。如需根治约 5 行 follow-up (building_service Create/Update 末尾调 `updateBuildingFloorCount`)。

## Self-Check: PASSED

- [x] floor_service.go Update 重算块 + recomputeAllBuildingTotalFloors helper 已提交 (d41de5e4, 4746cdc1)
- [x] excel_service.go 导入 hook 已提交 (4746cdc1)
- [x] migration_213_recompute_building_total_floors.go + _test.go 已提交 (3a3163a6)
- [x] database.go Migrate213 双注册已提交 (3a3163a6)
- [x] 3 个 commit hash 均在 git log (--all 确认): d41de5e4 / 4746cdc1 / 3a3163a6
- [x] 7 个任务文件 git status 干净 (全部已提交, 无未跟踪残留)
- [x] 最终门禁: go build ./... + operations + core/db + migrations 全绿
