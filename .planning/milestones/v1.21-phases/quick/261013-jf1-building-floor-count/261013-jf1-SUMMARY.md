---
status: complete
phase: quick
plan: 261013-jf1
subsystem: operations
tags: [building, floor-count, realtime-aggregation, group-by, virtual-field, zero-n-plus-1]
requires: ["OpsBuilding/OpsFloor models", "buildingService.List repo 管道 (Phase 91-03)", "ops_floors.building_id varchar(64) 关联键"]
provides: ["OpsBuilding.FloorCount 实时楼层数虚拟字段", "buildingService attachFloorCount 批量聚合", "楼栋管理页实时楼层数列(nil → \"-\")"]
affects: ["xingran-react-frontend/src/pages/operations/buildings/index.tsx 楼层数列渲染"]
tech-stack:
  added: []
  patterns: ["虚拟字段 + List 末尾单条 GROUP BY 批量注入 (261012-l7k 同构复刻)"]
key-files:
  created:
    - internal/services/operations/building_floor_count_test.go
  modified:
    - internal/models/operations/building.go
    - internal/services/operations/building_service.go
    - xingran-react-frontend/src/types/operations.ts
    - xingran-react-frontend/src/pages/operations/buildings/index.tsx
decisions:
  - "楼层数口径 = ops_floors 非软删行数(实时 COUNT),使显示层对 total_floors 既有漂移路径(换楼不重算/Excel 直写)免疫"
  - "FloorCount 用 gorm:\"-\" 完全忽略而非 ->(selectScope 显式选列下 -> 会扫描不存在列)"
  - "前端排序参数保持发 totalFloors(后端白名单只认 total_floors),与 dataIndex 解耦"
  - "total_floors 持久化字段及其事件同步/Excel 导入导出零改动"
metrics:
  duration: ~11 min
  completed: 2026-10-13
---

# Quick 261013-jf1: 楼栋管理页每行显示该楼栋的楼层数（实时 COUNT）Summary

楼栋 List 每行注入实时楼层数（`ops_floors` 非软删行数，单条 GROUP BY 零 N+1），前端既有"楼层数"列切实时字段并支持无楼层渲染 "-"，`total_floors` 反范式持久化字段零改动。

## What Was Built

### Task 1: 后端 — FloorCount 虚拟字段 + List 批量聚合注入（commit `24af23d9`）

- `internal/models/operations/building.go`：`OpsBuilding` 末尾新增 `FloorCount *int64`，tag `gorm:"-"` + `json:"floorCount,omitempty"`，附注释说明必须 `-` 而非 `->` 的原因（selectScope 显式选列）与 nil 语义（聚合未覆盖/无楼层 → 序列化不出键）。`TotalFloors` 原样保留。
- `internal/services/operations/building_service.go`：
  - 新增 `attachFloorCount(ctx, buildings)`——逐字平移 `floor_service.go` `attachWorkstationStats` 结构：页内非空 ID 去重 → `Model(&operations.OpsFloor{})` 起链（自动附加 `deleted_at IS NULL`）+ `Select("building_id", "COUNT(*) AS floor_count")` + `IN ?` + `Group("building_id")` 单条聚合 → map 回填；失败 `logrus.WithError(err).Warn("[building] 楼层数聚合失败，按无统计降级")` 后 return（best-effort，行保持 nil）。新增 `sirupsen/logrus` import。
  - `List` 改为先接住 `s.repo.List(...)` 结果，类型断言 `[]operations.OpsBuilding` 成功且非空页才注入，失败/空页原样返回。
- `internal/services/operations/building_floor_count_test.go`（新建）：复用 `newFloor91TestDB` + `seedFloor91Building` 零新增基建，3 用例：
  - `TestBuildingFloorCount_BasicAggregation`：B1=2 层 / B2=1 层 / B3=0 层 → FloorCount 2 / 1 / nil，B3 JSON 不含 `floorCount` 键，B1 JSON 含 `"floorCount":2`
  - `TestBuildingFloorCount_SoftDeleteExcluded`：软删 1/2 层后 → FloorCount=1
  - `TestBuildingFloorCount_PaginationIsolation`：pageSize=1 → 第 1 页仅 BFC-PG-A 且 FloorCount=2（页外 BFC-PG-B 的楼层不混入）

### Task 2: 前端 — 类型扩展 + 列切实时字段（commit `b4bff80c`）

- `xingran-react-frontend/src/types/operations.ts`：`Building` interface 在 `totalFloors` 旁新增 `floorCount?: number | null;` + 注释（与 totalFloors 反范式字段并存）。
- `xingran-react-frontend/src/pages/operations/buildings/index.tsx`：既有"楼层数"列（407-415 行）`dataIndex`/`key` 切到 `floorCount`，render 改 `(v) => (v == null ? "-" : v)`；**排序保持** `sortOrder: getColumnSortOrder("totalFloors")` 不变（排序参数与 dataIndex 解耦——后端白名单 `building_allowedSortFields` 只认 `"totalFloors": "total_floors"`，发 `floorCount` 会静默失效）；列宽 100、列位置不动；第 521 行概览抽屉 `{building.totalFloors || 0}层` 按计划零改动。

### Task 3: 全量验证门禁

见下方 Verification（全绿）。

## Verification（真实运行输出）

| 命令 | 结果 |
|------|------|
| `go build ./...` | 退出码 0（BUILD_OK） |
| `go test ./internal/services/operations/ -run TestBuildingFloorCount -v` | 3 个新测试全 PASS（BasicAggregation 0.09s / SoftDeleteExcluded 0.07s / PaginationIsolation 0.07s），ok 2.604s |
| `go test ./internal/services/operations/` | `ok github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations 5.640s`（含 `building_service_test.go` / `floor_workstation_stats_test.go` / `building_total_softdelete_test.go` 等既有测试零回归） |
| `cd xingran-react-frontend && npm run type-check` | 退出码 0，零错误 |
| `cd xingran-react-frontend && npx vitest run src/pages/operations/buildings` | `Test Files 4 passed (4)` / `Tests 22 passed (22)`（`index.render.test.tsx` 等既有测试无需改动，按计划预期） |

## Success Criteria Check

- [x] 楼栋管理页"楼层数"列显示实时值（ops_floors 非软删行数）；无楼层显示 "-"
- [x] 后端 List 每页只多 1 条 GROUP BY building_id 聚合 SQL（零 N+1，由同构模式与测试锁定）
- [x] 软删楼层不计入；`total_floors` 持久化字段及其事件同步、Excel 导入导出零改动
- [x] 换楼不重算 total_floors 的预先存在缺陷未修（scope constrainment），显示层因实时 COUNT 对该漂移免疫
- [x] go build + operations 测试 + type-check + buildings vitest 全绿
- [x] 不涉及 operlog（只读扩展）、不涉及 DB migration（gorm:"-" 虚拟字段）

## Deviations from Plan

**无功能性偏差。** 两处执行期披露：

### 先前未提交修改一并入库（文件级 staging 副作用，预先知情）

- **发现于：** Task 1 / Task 2 提交时
- **情况：** `internal/services/operations/building_service.go` 与 `xingran-react-frontend/src/pages/operations/buildings/index.tsx` 在本任务开始前已携带大量先前未提交修改（git add 不支持交互式 partial staging），文件级 staging 将先前工作一并入库。与昨天任务 261012-l7k 同样先例。
- **具体卷入内容：**
  - `building_service.go`（commit 24af23d9 内 164+/50- 中约 150 行为先前工作）：Phase 91-03 repo 化（`repo.Create/Delete/GetByID/List` + `GORMRepository` 注入 + List 软删 Total 收紧 latent bugfix）、V130R-09 分页归一化（`pkg/query.NormalizePaginationWithMax` + `extractIntParam`）、`internal/constants` → `pkg/constants` 迁移（`UuidPattern` → `UUIDPattern`）、`Statistics` 魔法数字修复（status 0/1 内联 → `operations.BuildingStatus*` 常量）
  - `index.tsx`（commit b4bff80c 内 485+/210- 中约 680 行为先前工作）：该页面的既有未提交改造
- **验证口径：** 所有验证命令跑的都是工作区最终状态（即含先前工作的状态），全绿；先前工作本身即工作区现状，未被本任务改动。
- **文件：** 上述 2 个文件；`building.go` 与 `operations.ts` 干净（仅含本任务改动，commit diff 分别为 +8 行 / +2 行）。

### [微调] 测试变量编译修正

- `building_floor_count_test.go` 初版 `b3` 声明未使用导致编译失败，改为直接调用 `seedFloor91Building`（B3 经 byName map 查找断言），非逻辑偏差。

## Known Stubs

无。无楼层楼栋 `FloorCount=nil`（键缺省 → 前端 "-"）是计划锁定的设计行为，非 stub。

## Commits

| Task | Commit | 内容 |
|------|--------|------|
| 1 | `24af23d9` | feat(quick-261013-jf1): building List 注入实时楼层数 FloorCount(单条 GROUP BY 零 N+1) |
| 2 | `b4bff80c` | feat(quick-261013-jf1): 楼栋管理页楼层数列切实时 floorCount(nil 渲染 "-") |

## Self-Check: PASSED

- 全部 5 个任务文件 + SUMMARY 存在（FOUND）
- 两个任务 commit 均在 git log 中（24af23d9 / b4bff80c）
- 提交后删除检查：零意外删除；任务文件零残留未提交状态
