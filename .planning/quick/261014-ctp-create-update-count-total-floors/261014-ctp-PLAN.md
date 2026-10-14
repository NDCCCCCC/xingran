# 楼栋 Create/Update 以实时 COUNT 覆盖 total_floors（关闭 261013-lqb 残余向量）

**Mode**: quick
**Date**: 2026-10-14
**Slug**: 261014-ctp-create-update-count-total-floors
**前序**: 261013-lqb（total_floors 漂移根治）SUMMARY 记录的已知残余向量：楼栋表单/手工 Create+Update 仍写用户声明值到 `total_floors`，可能与真实楼层数不符；4 个消费方（排序白名单 total_floors / BuildingCard / 概览抽屉 / Excel 导出）会看到脏值。

## Context

261013-lqb 已修复楼层事件路径（Update 换楼双侧重算 / Excel 导入收尾重算 / Migrate213 存量归正），重算口径 = 实时 COUNT（软删排除），幂等。本任务补最后一个写路径：楼栋自身的 Create/Update。

## Tasks

### Task 1: building_service Create/Update 末尾实时覆盖 total_floors + 测试

**Files:**
- `internal/services/operations/building_service.go`（修改）
- `internal/services/operations/building_total_floors_override_test.go`（新建）

**Action:**
1. 在 `building_service.go` 的 `Create` 与 `Update` 方法 success path（DB 写入成功后）调用与 261013-lqb 同款的单楼栋实时重算（优先复用 `floor_service.go` 已有的单楼栋重算 helper，如无则抽一个 `recomputeBuildingTotalFloorsByID(ctx, db, buildingID)` 放 `floor_service.go` 供两侧共用；口径：`SELECT COUNT(*) FROM ops_floors WHERE building_id=? AND deleted_at IS NULL`，实时查询非 ±1 增量）。
2. 覆盖语义：**用户表单声明的 totalFloors 值一律被实时 COUNT 覆盖**（total_floors 是派生列，不是用户输入位）。Create 后新楼栋 total_floors=0（新楼无楼层）；Update 后=当前真实楼层数。
3. 错误处理对齐 261013-lqb Task 1：重算失败上抛（fail-loud），不静默吞掉。
4. operlog：不新增（handler 层已有约定处理，本任务不动 handler）。

**测试（新建文件，参照 `building_floor_count_test.go` / `floor_workstation_stats_test.go` 的 `newFloor91TestDB` 基建）:**
- Create 带声明 totalFloors=999 → 落库后=0（实时 COUNT 覆盖用户值）
- Update 声明 totalFloors=999 → 落库后=真实楼层数（如 3）
- 楼栋带 2 层（1 层软删）→ Update 后=1（软删排除）

### Task 2: 全量门禁

**Action:**
- `go build ./...`
- `go test ./internal/services/operations/`（新测试 + 既有 total_floors 用例零回归）

## Constraints

- 零 DB TRIGGER / 零 schema object（用户偏好）
- 不动 handler / router / 前端
- floor_service.go / building_service.go 可能携带先前未提交 WIP——文件级 staging 卷入时在 SUMMARY Deviations 如实披露（261012-l7k / 261013-jf1 / 261013-lqb 同先例），不做 partial staging
