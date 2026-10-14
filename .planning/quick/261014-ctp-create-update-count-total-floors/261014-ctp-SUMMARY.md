---
phase: quick
plan: 261014-ctp
subsystem: operations
tags: [operations, building, total-floors, derived-column, data-consistency]
requires:
  - 261013-lqb (total_floors 漂移根治 — 本任务关闭其 SUMMARY 记录的最后残余向量)
provides:
  - recomputeBuildingTotalFloorsByID 包级共享单楼栋重算 helper (floor_service.go)
  - buildingService.Create/Update success path 实时 COUNT 覆盖 total_floors (声明值一律失效)
affects:
  - ops_buildings.total_floors 消费方 (排序白名单 / BuildingCard / 概览抽屉 / Excel 导出) — 全部写路径闭合后恒为真实值
tech-stack:
  added: []
  patterns: [派生列 service 层覆盖 (写路径末尾 fail-loud 重算), method→包级 helper 抽取共用]
key-files:
  created:
    - internal/services/operations/building_total_floors_override_test.go
  modified:
    - internal/services/operations/building_service.go
    - internal/services/operations/floor_service.go
decisions:
  - total_floors 定性为派生列而非用户输入位 — 表单声明值一律被实时 COUNT 覆盖 (Create→0, Update→真实层数)
  - 抽取 recomputeBuildingTotalFloorsByID 包级 helper 而非跨 service 调 method — updateBuildingFloorCount 委托之, 单一真实源
  - 重算失败上抛 fail-loud, 对齐 261013-lqb 路径 A (Delete/Update) 行为
metrics:
  duration: ~5 min
  completed: 2026-10-14
---

# Quick 261014-ctp: 楼栋 Create/Update 实时 COUNT 覆盖 total_floors Summary

**One-liner:** buildingService.Create/Update success path 末尾调用新抽取的包级 helper `recomputeBuildingTotalFloorsByID`（实时 COUNT、软删排除、幂等），用户表单声明的 totalFloors 值一律被覆盖——261013-lqb 的最后一个写路径漂移向量关闭，total_floors 四条写路径（楼层 CRUD / Excel 导入 / 存量迁移 / 楼栋表单）全部闭合。

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | building_service Create/Update 末尾实时覆盖 + 共享 helper 抽取 + 3 单测 | 7b64fc06 | building_service.go, floor_service.go, building_total_floors_override_test.go (new) |
| 2 | 全量门禁 (go build ./... + go test ./internal/services/operations/) | — (无文件变更) | — |

## Implementation Notes

- **helper 抽取**: PLAN 指定的 `recomputeBuildingTotalFloorsByID(ctx, db, buildingID)` 落在 `floor_service.go`，紧邻 `updateBuildingFloorCount`；后者改为一行委托（行为逐字节不变，由既有 `floor_update_total_floors_test.go` 3 用例守护）。COUNT 口径与 `recomputeAllBuildingTotalFloors` 一致：`Model(&OpsFloor{})` 实时 COUNT（自动软删过滤），禁止 ±1 增量。
- **Create**: `s.repo.Create` 后调 helper——新楼栋无楼层 → total_floors=0，声明值 999 被覆盖。
- **Update**: `Save`（Omit CreatedAt 语义保留）后调 helper——覆盖为当前真实楼层数。
- **fail-loud**: 两处重算错误均 `return err` 上抛，不静默吞掉（对齐 261013-lqb 路径 A）。
- **handler/router/前端零改动**（per PLAN）；operlog 零新增（handler 层既有约定已覆盖）。
- 调用面核查: 生产侧 `buildingHandler.Create/Update`（router.go:546）与 Excel handler 均经此路径或 BatchUpsert（后者已由 261013-lqb 收尾重算覆盖）；测试侧 3 处 `NewBuildingService` 调用点的测试库均含 ops_floors/ops_buildings 表（newFloor91TestDB / newCRUDTestDB AutoMigrate），无破坏。

## Verification (真实输出)

| 命令 | 结果 |
|------|------|
| `go build ./...` | exit 0 ("BUILD OK", 每任务后均跑) |
| `go test ./internal/services/operations/ -run TestBuildingTotalFloorsOverride -v` | 3/3 PASS (CreateOverridesDeclaredValue / UpdateOverridesWithRealCount / UpdateExcludesSoftDeletedFloors) |
| `go test ./internal/services/operations/` (全量) | ok 13.138s — 既有 total_floors 用例零回归 |
| 定向回归 (`-run "TestFloorUpdateTotalFloors\|TestRecomputeAllBuildingTotalFloors\|TestBuildingFloorCount\|TestBuildingTotalFloorsOverride"`) | 10/10 PASS (3 新 + 7 既有 261013-lqb / 261013-jf1 用例) |

无人工 UAT (按 PLAN: 写路径语义由测试锁定, 消费方零改动)。

## Deviations from Plan

**None — plan executed exactly as written.**

补充如实记录（非偏离）：环境提示 "building_service.go / floor_service.go 已携带先前未提交 WIP，文件级 staging 会卷入"。实测 `git diff --stat` vs HEAD 逐 hunk 核对，两文件差异 100% 为本任务变更（building_service.go +18/-2、floor_service.go +16/-2，全部落在 Create/Update/helper 抽取 hunk），**无先前 WIP 卷入**，提交 7b64fc06 内容纯净。

## Known Stubs

None — 无占位值/空数据源/TODO。

## Intentional Residual (per PLAN scope constrainment)

None — 261013-lqb SUMMARY 记录的残余向量已由本任务关闭。total_floors 全部已知写路径（楼层 Create/Update/Delete/BatchDelete、Excel 导入 building+floor、Migrate213 存量、楼栋表单 Create/Update）均收敛到同一实时 COUNT 口径。

## Self-Check: PASSED

- [x] building_total_floors_override_test.go (new) 已提交 (7b64fc06)
- [x] building_service.go Create/Update 覆盖块已提交 (7b64fc06)
- [x] floor_service.go recomputeBuildingTotalFloorsByID helper + 委托已提交 (7b64fc06)
- [x] commit 7b64fc06 在 git log (--all 确认)
- [x] 3 个任务文件 git status 干净 (无未跟踪残留)
- [x] 门禁真实运行: go build ./... exit 0 + operations 全量 ok 13.138s + 定向 10/10 PASS
