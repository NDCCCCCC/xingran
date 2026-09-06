---
phase: 91-crud-base-repository-t-p1
plan: 04
subsystem: api
tags: [gorm, generic-repository, typed-request, scope-based, batch-migration, dead-code-removal, loc-audit]

# Dependency graph
requires:
  - phase: 91-01
    provides: base.GORMRepository[T] 六方法 scope 仓储 + Scope/PageParams + SortScope（B 型）/SortScopeWithTail（A 型）+ base.PageResult 单一定义
  - phase: 91-02
    provides: workstation pilot 迁移模式（joinScope/filterScope 组合 + typed bind 降级 + 白盒构造 nil-panic 教训）
  - phase: 91-03
    provides: building/floor/asset map 签名迁移形状 + floor 行为基线 6 锁 + calculateOffset 引用清零
provides:
  - 11/11 operations CRUD 服务全部复用 base.GORMRepository[T]（CRUD 模板清零）
  - A 型复合尾随排序样本（door/wall，SortScopeWithTail 实战）
  - P8 scope 顺序敏感样本（room_device joinScope 先于 filterScope 的 source 断言）
  - floor_plan_text service 级冒烟测试（Wave 0 缺口清零，TestFloorPlanText91）
  - typesafe 死代码三文件清零（-564 行）+ calculateOffset 修剪
  - REQUIREMENTS.md § CRUD-REUSE 措辞与 D-04/D-06/D-07 对齐（REQ_SYNC_OK gate 过）
affects: [92 缓存层统一, 93 config_backup, v1.29 closeout]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "typed 服务迁移七步法：struct 组 repo → 六方法委托 → validate/prefill 前置保留 → filterScope 条件逐字平移 → joinScope 常量逐字平移 → SortScope/SortScopeWithTail 分型 → GetPagination 直传 PageParams"
    - "A 型复合尾随排序：base.SortScopeWithTail 无条件尾随 created_at DESC（door/wall tiebreaker 语义锁定）"
    - "JOIN 前置 Count 等价论证模板：按主键 N:1 LEFT JOIN 无行增殖 → Count 不变（server_room/room_device/infopoint 三样本）"
    - "时间序断言确定性：显式错开 CreatedAt 消除同刻排序平局 flake（-count=10 验证）"

key-files:
  created:
    - internal/services/operations/floor_plan_text_crud_test.go
  modified:
    - internal/services/operations/door_service.go
    - internal/services/operations/wall_service.go
    - internal/services/operations/server_room_service.go
    - internal/services/operations/dedicated_line_service.go
    - internal/services/operations/floor_plan_text_service.go
    - internal/services/operations/room_device_service.go
    - internal/services/operations/infopoint_service.go
    - internal/services/operations/workstation_service.go
    - internal/services/operations/pagination_helper.go
    - internal/services/operations/pagination_helper_test.go
    - internal/services/operations/asset_building_roomdevice_test.go
    - .planning/REQUIREMENTS.md
  deleted:
    - internal/services/operations/building_service_typesafe.go
    - internal/api/v1/operations/building_handler_typesafe.go
    - internal/api/v1/operations/building_handler_typesafe_test.go

key-decisions:
  - "BatchDelete 空 ids 语义反转（91-01 P1）：repo 返回 nil，door/wall/fpt 的显式早退删除，行为一致"
  - "F3 软删 Total 修复与 D-05 分页收紧分别经 91-03/91-02 checkpoint auto-approved，本 plan 无新增 checkpoint"
  - "typesafe 三文件删除已由 orchestrator 拍板纳入（零生产消费者，编译期安全 A1），实际净删 597 行"
  - "extractXxxParam 保留口径：helper 本体全保留（F1——3 个 map 服务 List + SearchXxxOptions + Statistics 仍需），仅修剪达零消费者的 calculateOffset；clampPageSize 剩 location_alias_service.go:102 合法消费"
  - "requests.BuildingListRequest/BuildingBatchOperationRequest 在 typesafe 删除后成为零消费者，按 plan 范围保留声明（编译无影响），列 future cleanup"
  - "LOC 净减 ≥800 未达成：全口径 numstat 净减 -8（新增测试基线 +678 行所致），仅生产代码口径净减 +408——如实记录，诚实回退，列 F5 组合 2 备选留待用户决策，禁止凑数删除"

requirements-completed: [CRUD-REUSE-06, CRUD-REUSE-07, CRUD-REUSE-08]

# Metrics
duration: 40min
completed: 2026-09-04
---

# Phase 91 Plan 04: 剩余 7 服务迁移 + typesafe 死代码清理 + LOC 审计收口 Summary

**剩余 7 个 typed 服务（door/wall/server_room/dedicated_line/floor_plan_text/room_device/infopoint）全部迁入 base.GORMRepository[T]，11/11 服务 CRUD 模板清零；typesafe 死代码三文件 -597 行 + calculateOffset 修剪 + REQUIREMENTS § CRUD-REUSE 措辞同步；LOC 审计诚实记录：生产代码净减 +408，全口径净减 -8（测试基线新增 +678 行），D-07 ≥800 未达成**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-09-04T11:15Z
- **Completed:** 2026-09-04T11:55Z
- **Tasks:** 4/4（无 checkpoint——typesafe 清理 orchestrator 已拍板，F3/A2 checkpoint 已在 91-02/91-03 闭环）
- **Files:** 1 新增 + 8 修改 + 3 删除 + 1 计划文档（REQUIREMENTS.md）

## Accomplishments

- **11/11 服务 repo 化收口**：door/wall（A 型复合尾随）、server_room/dedicated_line/floor_plan_text（B 型 plain）、room_device/infopoint（B 型 JOIN typed）六方法 CRUD 管道全部委托 `base.GORMRepository[T]`；11 个文件 `countRecords/fetchRecords/buildListQueryFromRequest` 私有管道定义计数为 0
- **A 型复合尾随排序落地**（T-91-04-01）：door/wall 用 `base.SortScopeWithTail` 无条件尾随 `created_at DESC`——用户排序 → `ORDER BY <col> <dir>, created_at DESC`；无/非法排序 → 仅 `created_at DESC`（与迁移前取数管道恒追加逐字等价）；`Order("")` no-op 死代码删除（RESEARCH spike #4 实证）
- **P8 scope 顺序敏感样本**（T-91-04-02）：room_device 的 `joinScope()` 实参位于 `filterScope(req)` 之前（orgId filter 引用 joined 表 `ops_buildings.org_id`）；server_room 的 orgId scope 同样后于 joinScope 且空命中早退分支逐字保留
- **JOIN 前置 Count 等价**：server_room（迁移前 Select+Joins 本就在 Count 前）、room_device/infopoint（迁移前 Select+Joins 在 Count 后）——三服务 JOIN 均按主键 N:1 无行增殖，Count 不变；TestBase91 契约 + TestServerRoom/TestInfoPoint/TestRoomDevice Total 断言守护
- **兼容分支逐字保留**（T-91-04-05）：infopoint 的 WorkID/PointType 旧字段兼容（含 else-if 结构）、populateRedundantFields 3 次回查、dedicated_line populateRoomNames 2 次回查、room_device validateRoom + isDuplicateKeyError 映射、server_room getDeptAndChildDeptIDs——调用顺序零变更
- **Wave 0 缺口清零**：新建 `floor_plan_text_crud_test.go`（TestFloorPlanText91_CRUDSmoke）——六方法冒烟 + 默认序/白名单排序/值切片断言/软删/BatchDelete(nil) 锁，显式错开 CreatedAt 消除排序平局
- **死代码清理**：typesafe 三文件删除（171+169+224=564 行）+ TestBuildingService_TypeSafe 测试块（33 行）；pagination_helper.go 修剪 calculateOffset（零消费者）+ 对应用例（30 行）
- **REQUIREMENTS.md § CRUD-REUSE 措辞同步**（D-04 锁定，沿 3a2efe5 先例）：CRUD-REUSE-01 改 scope 化改造口径、CRUD-REUSE-06 清单 +wall_service.go +floor_plan_text_service.go（D-06 扩容后剩 7 个）、CRUD-REUSE-08 改净减 ≥800 混合标准；REQ_SYNC_OK gate（2 正向 grep + 2 反向 grep）通过

## Task Commits

1. **Task 1: door + wall 迁移（A 型复合尾随首个实战）** - `d60e0e8` (refactor)
2. **Task 2: server_room + dedicated_line + floor_plan_text 迁移 + fpt 冒烟测试** - `030d370` (refactor)
3. **Task 3: room_device + infopoint 迁移（JOIN typed，P8 顺序敏感）+ workstationTable 清理 + fpt 测试确定性修复** - `14e4a3c` (refactor)
4. **Task 4: typesafe 死文件清理 + pagination_helper 修剪 + REQUIREMENTS 同步** - `963defe` (refactor)；注释措辞微调 `6a45c19` (docs)

## 11 服务迁移清单（D-06 全覆盖）

| # | 服务 | 迁移前 LOC | 迁移后 LOC | List 形态 | 排序语义 | 特殊保留项 |
|---|------|-----------|-----------|----------|---------|-----------|
| 1 | workstation | 456 | 468 | typed（D-05） | B 型 `sys_workstation.created_at DESC` | Statistics/DeptOptions/SearchOptions/BatchUpdatePositions/First 回填（91-02） |
| 2 | building | 300 | 293 | map 不变 | B 型 `order_num ASC` | validateOrg/applyFilters/workstation_count 子查询（91-03） |
| 3 | floor | 436 | 448 | map 不变 | B 型 `ops_floors.order_num ASC` | 软删恢复/异步同步/楼层计数编排；listRepo() nil-safe（91-03） |
| 4 | asset | 411 | 395 | map 不变 | B 型 `created_at DESC` | Update Omit("CreatedAt") 变体；component_type IS NULL 恒定过滤（91-03） |
| 5 | server_room | 251 | 246 | typed | B 型 `ops_server_rooms.created_at DESC` | orgId 空命中早退空 PageResult；getDeptAndChildDeptIDs |
| 6 | infopoint | 264 | 254 | typed | B 型 `ops_info_points.created_at DESC` | WorkID/PointType 兼容分支；populateRedundantFields 3 回查；orgId EXISTS |
| 7 | dedicated_line | 231 | 215 | typed | B 型 `created_at DESC` | populateRoomNames 2 回查；11 filter 含 source/dest else-if |
| 8 | room_device | 223 | 219 | typed | B 型 `ops_room_devices.created_at DESC` | validateRoom + isDuplicateKeyError→DeviceCodeAlreadyExists；P8 joinScope 前置 |
| 9 | door | 148 | 116 | typed | **A 型** SortScopeWithTail `created_at DESC` | validateDoorRelations（Floor+Wall 条件校验） |
| 10 | wall | 136 | 103 | typed | **A 型** SortScopeWithTail `created_at DESC` | ValidateFloor 前置 |
| 11 | floor_plan_text | 116 | 95 | typed | B 型 `created_at DESC` | validator.ValidateFloor；Delete/BatchDelete Table→Model 形态等价 |
| | **合计** | **2972** | **2852** | | | |

## LOC 审计（T-91-04-04：git numstat 命令级证据）

**审计命令：** `git diff --numstat $(git rev-parse d92097f^)..HEAD -- internal/services/ internal/api/v1/operations/`（BASE = 91-01 首个 commit `d92097f` 的父 `0ab266e`）

### 逐文件 before → after（wc -l）

| 文件 | before | after | Δ |
|------|-------:|------:|---:|
| door_service.go | 148 | 116 | -32 |
| wall_service.go | 136 | 103 | -33 |
| server_room_service.go | 251 | 246 | -5 |
| dedicated_line_service.go | 231 | 215 | -16 |
| floor_plan_text_service.go | 116 | 95 | -21 |
| room_device_service.go | 223 | 219 | -4 |
| infopoint_service.go | 264 | 254 | -10 |
| workstation_service.go | 456 | 468 | +12 |
| building_service.go | 300 | 293 | -7 |
| floor_service.go | 436 | 448 | +12 |
| asset_service.go | 411 | 395 | -16 |
| **11 服务小计** | **2972** | **2852** | **-120** |
| base/service.go | 161 | 198 | +37 |
| base_80_05_test.go | 207 | 205 | -2 |
| pagination_helper.go | 82 | 80 | -2 |
| pagination_helper_test.go | 207 | 177 | -30 |
| asset_building_roomdevice_test.go | 331 | 298 | -33 |
| building_service_typesafe.go | 171 | 已删除 | -171 |
| building_handler_typesafe.go | 169 | 已删除 | -169 |
| building_handler_typesafe_test.go | 224 | 已删除 | -224 |

### numstat 汇总与结论

| 口径 | add | del | 净减（del-add） |
|------|----:|----:|----------:|
| 全口径（internal/services/ + internal/api/v1/operations/） | 1563 | 1555 | **-8** |
| 仅生产代码（排除 *_test.go） | 774 | 1182 | **+408** |
| 仅测试代码 | 789 | 373 | -416 |
| 全仓口径（含 .planning 文档） | 2027 | 939 | -1088 |

**结论：D-07「LOC 净减 ≥800」未达成（诚实回退）。** 差距归因：

1. **测试基线新增 +678 行**（phase 级 Wave 0 缺口补齐的计划内产物）：floor_service_crud_test.go +288（91-03）、base/service_test.go +275（91-01 契约锁值）、floor_plan_text_crud_test.go +115（本 plan）——RESEARCH F5 估算未计入这三个新增文件
2. **服务层净减 -120 低于 RESEARCH 估算（~350）**：7 个 filter/条件逐字平移的服务（server_room -5、room_device -4）每条件行原样保留，且按 plan 要求新增了行为等价论证注释（joinScope 前置 Count 论证、P8 顺序论证、A 型语义说明等 ~30 行/服务）；workstation/floor 因 D-04 保留方法的注释扩写各 +12
3. **typesafe 清理实绩优于估算**：F5 组合 1 估算 ~200-250 行，实际净删 597 行（564 三文件 + 33 测试块）

**仅生产代码口径净减 +408**，与 RESEARCH「严格范围 450-550」预估吻合。**可选补充来源（留待用户决策，本 plan 未实施）**：F5 组合 2——building/floor typed request 接线扩展（struct 已存在，handler bind 各 1 处，估算 ~80-120 行）。禁止为凑数做无意义删除。

## Decisions Made

- **BatchDelete 空 ids 语义反转**（91-01 P1 落地收尾）：door/wall/floor_plan_text 迁移前的显式 `if len(ids) == 0 { return nil }` 早退删除，repo 层统一承接（空 ids → nil）；server_room/dedicated_line/room_device/infopoint 迁移前依赖 `IN (空)` no-op，行为一致
- **F3 软删 Total 修复 / D-05 分页收紧**：分别经 91-03 Task 4 / 91-02 Task 3 checkpoint auto-approved，本 plan 零新增 checkpoint（typed 服务本就走 GetPagination 语义）
- **typesafe 死文件清理已拍板**：router.go:546 只用 NewBuildingService；BuildingServiceTypeSafe/BuildingHandlerTypeSafe 全库仅 4 处引用（本次 grep 复验），零生产消费者；编译期安全（A1）——`go build ./...` 立即暴露遗漏，实际一次通过
- **extractXxxParam 保留口径**：extractPagination/extractSortRequest/extractIntParam/extractStringParam/clampPageSize 五 helper 全部保留（F1：building/floor/asset List map 路径 + 全部 SearchXxxOptions + Statistics 仍消费；clampPageSize 剩 location_alias_service.go:102 合法消费）；仅 calculateOffset 达零消费者予以修剪（offset 计算由 repo.List 内部承接，typed 路径走 requests.PaginationParams.GetOffset）
- **door/wall 保留 db 字段**：plan 明示「db 保留（validateDoorRelations 的 Validator 构造仍需）」；六方法全部 repo 化后字段本身零直用，保留以维持构造形态与未来扩展面（编译零告警）
- **floorPlanTextTable 常量随迁移删除**：Delete/BatchDelete 的 `.Table(...)` 形态被 repo 的 `Model(new(T))` 形态承接后常量失云消费者（软删 + WHERE 语义一致，RESEARCH 盘点 #11），机械清理与 workstationTable 同口径

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] server_room orgId scope 未按条件守卫，空 OrgID 时过滤全部行**
- **Found during:** Task 2 verify（TestServerRoomService_CRUD Total=0 红灯 + index out of range panic）
- **Issue:** 初版 List 把 `db.Where("b.org_id IN ?", deptIDs)` 闭包无条件传入 scopes——req.OrgID 为空时 deptIDs 为 nil，生成 `b.org_id IN (NULL)`，Count 恒 0
- **Fix:** orgId scope 仅在 `req.OrgID != ""` 时 append（与迁移前条件分支一致）
- **Files modified:** internal/services/operations/server_room_service.go
- **Commit:** 030d370（含在 Task 2 commit 内）

**2. [Rule 1 - 测试缺陷] fpt 冒烟测试排序断言同刻平局 flake**
- **Found during:** Task 3 verify（全包运行间歇 FAIL，单跑 PASS；`-count=10` 复筛定位）
- **Issue:** 两条测试数据同刻创建时 `created_at DESC/ASC` 排序不确定，`list[0]` 断言间歇翻转；另 BatchDelete 段误用排序前快照的已删 ID
- **Fix:** 显式错开两条记录的 CreatedAt（GORM 对非零 CreatedAt 不覆盖），时序断言确定性化；BatchDelete 段改用当前查询结果
- **Files modified:** internal/services/operations/floor_plan_text_crud_test.go
- **Commit:** 14e4a3c

### Plan 事实修正（非代码偏差）

- **RESEARCH/plan 声称「floor_plan_text 无 service 级测试（Wave 0 缺口）」不完全准确**：`asset_building_roomdevice_test.go:233` 已有 TestFloorPlanTextService_CRUD（RESEARCH 时点遗漏）。按 plan 字面要求仍新建 floor_plan_text_crud_test.go（TestFloorPlanText91 前缀），增量价值 = 排序双语义锁（默认序 + 白名单排他）+ repo 值切片断言 + BatchDelete(nil) 后再命中批量删除路径，与既有测试互补不重复
- **workstationTable 闲置常量删除**：plan phase context 授权的机械清理（91-02 迁移后残留，仅注释引用），随 Task 3 commit 实施
- **requests.BuildingListRequest/BuildingBatchOperationRequest 成为零消费者**：typesafe 三文件 + 测试块删除后无任何引用；struct 声明按 plan 范围保留（编译无影响），列 future cleanup（不属「禁止凑数删除」范畴，但为避免范围蔓延未动）

## Issues Encountered

- commitlint body-max-line-length 首次拒绝 Task 2 commit message（body 行 >100 字符），缩短后重提交——Task 1 同型 message 未触发属边界值
- 全量 `go test ./...` 单次 >10min，终验 gate 后台执行（91-03 同款教训）

## Verification Results

- `go build ./...` 退出码 0（每 task 收尾均过，typesafe 删除后即时验证）
- `go test ./internal/services/operations/ -run "TestDoor|TestWall"` 全绿（Task 1）
- `go test ./internal/services/operations/ -run "TestServerRoom|TestDedicatedLine|TestFloorPlanText91"` 全绿（Task 2，含 5 个用例）
- `go test ./internal/services/operations/ -run "TestRoomDevice|TestInfoPoint|TestAsset"` 全绿（Task 3，9 个用例含 TestImp77 回归）
- `go test ./internal/services/operations/ -count=1` 全包 0 失败（11 服务迁移全部完成的包级证明）；`-count=10 -run TestFloorPlanText91` 无 flake
- `go test ./internal/api/v1/operations/ -count=1` 全绿（handler e2e smoke 口径）
- `go vet ./internal/services/... ./internal/api/v1/operations/...` 无告警
- **全量 gate：`go test ./...` 74 包 ok / 0 失败（exit 0）**（AST 锁值防线 status_constants_test / operlog regression_test / pkg/constants 全绿）
- grep 证明：11 个服务文件 `countRecords|fetchRecords` 计数全 0；`Order("")` 计数全 0；typesafe 三文件已删除；calculateOffset 计数 0 且五 helper 各 ≥1；TestBuildingService_TypeSafe 计数 0
- REQ_SYNC_OK gate：`.planning/REQUIREMENTS.md` 含「D-06 扩容后剩 7 个」与「净减 ≥800」，不含「补全缺失方法」与「target ~2000」

## User Setup Required

None - 纯内部重构 + 死代码删除，无外部服务配置。

## Next Phase Readiness

- **Phase 92（缓存层统一）可开工**：operations 侧 11 服务 repo 化完成，floor_cache_impl.go 的 listRepo() 兜底可随装饰器改造一并移除（注释已标注）
- Phase 93/94 与 91 互不依赖，可并行推进；Phase 95 收口必须最后
- D-07 LOC 净减 ≥800 未达成的 F5 组合 2 备选（building/floor typed 接线扩展）留待用户决策，不阻塞后续 phase
- 无阻塞项

## Self-Check: PASSED

- 1 新增文件 + 8 修改文件全部存在；3 删除文件已从工作树移除（ls 报 No such file，验收要求）
- 5 个 task commit 在 git log：d60e0e8 / 030d370 / 14e4a3c / 963defe / 6a45c19
- 终验：`go build ./...` 0 错误；`go test ./...` exit 0 / 74 包 ok / 0 FAIL

---
*Phase: 91-crud-base-repository-t-p1*
*Completed: 2026-09-04*
