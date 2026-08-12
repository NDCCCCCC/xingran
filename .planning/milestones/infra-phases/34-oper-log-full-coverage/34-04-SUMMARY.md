---
phase: 34-oper-log-full-coverage
plan: 04
subsystem: oper-log
tags: [oper-log, operations, instrumentation, excel-import, audit]
requires:
  - phase: 34-01
    provides: "operlog.Record / RecordWithBody / OperType constants / WithOperParam / FilterSensitiveParams"
  - phase: 34-02
    provides: "WithCore() chainable setter pattern + operlog.Record placement convention (成功路径末尾、response.Success 之前)"
provides:
  - 56 instrumented write endpoints across 13 operations handlers (building/floor/workstation/workstation_device/server_room/room_device/dedicated_line/infopoint/wall/door/floor_plan_text/excel/asset)
  - Excel import operParam schema ({filename,size,inserted,updated,failed}) — never raw row bodies (T-34-W3-01 mitigation)
  - Excel entity-type to Chinese module-name map (excelEntityModuleNames) so audit rows for /ops/<x>/excel/* share the same module name as the underlying entity's CRUD handlers
  - OperTypeSync(14) used for workstation_device SyncAD/SyncAsset; OperTypeGrant(4) used for workstation_device SetPrimary
affects:
  - internal/api/v1/operations/building_handler.go (5 operlog calls incl. Geocode OperTypeOther)
  - internal/api/v1/operations/floor_handler.go (4 operlog calls)
  - internal/api/v1/operations/workstation_handler.go (5 operlog calls incl. BatchUpdatePositions)
  - internal/api/v1/operations/workstation_device_handler.go (7 operlog calls incl. SyncAD/SyncAsset/SetPrimary)
  - internal/api/v1/operations/server_room_handler.go (4 operlog calls)
  - internal/api/v1/operations/room_device_handler.go (4 operlog calls)
  - internal/api/v1/operations/dedicated_line_handler.go (4 operlog calls)
  - internal/api/v1/operations/infopoint_handler.go (4 operlog calls)
  - internal/api/v1/operations/wall_handler.go (4 operlog calls)
  - internal/api/v1/operations/door_handler.go (4 operlog calls)
  - internal/api/v1/operations/floor_plan_text_handler.go (4 operlog calls)
  - internal/api/v1/operations/excel_handler.go (4 operlog calls incl. Import/Export/Download + Export JSON-marshal fallback)
  - internal/api/v1/operations/asset_handler.go (4 operlog calls)
  - internal/api/router.go (12 handler construction sites updated with .WithCore(core))
tech-stack:
  added: []
  patterns:
    - closure-handler-direct-record (excel_handler uses gin.HandlerFunc closures with core in scope — calls operlog.Record directly instead of via h.core; avoids needing a fake Handler struct)
    - entity-type-to-module-name-map (single source of truth so Excel audit rows share module name with CRUD handlers)
    - row-summary-operparam (Excel imports log {filename,size,inserted,updated,failed} instead of row contents — bounds oper_param size AND avoids leaking row data)
key-files:
  created: []
  modified:
    - internal/api/router.go
    - internal/api/v1/operations/building_handler.go
    - internal/api/v1/operations/floor_handler.go
    - internal/api/v1/operations/workstation_handler.go
    - internal/api/v1/operations/workstation_device_handler.go
    - internal/api/v1/operations/server_room_handler.go
    - internal/api/v1/operations/room_device_handler.go
    - internal/api/v1/operations/dedicated_line_handler.go
    - internal/api/v1/operations/infopoint_handler.go
    - internal/api/v1/operations/wall_handler.go
    - internal/api/v1/operations/door_handler.go
    - internal/api/v1/operations/floor_plan_text_handler.go
    - internal/api/v1/operations/excel_handler.go
    - internal/api/v1/operations/asset_handler.go
key-decisions:
  - "Single commit covers both plan tasks because router.go references WithCore on all 12 struct handlers — Task 1 alone would not compile (Task 2 handlers like dedicated_line/wall/door lack WithCore until Task 2 is also applied). Plan success_criteria explicitly allows this: 'Single commit: feat(operlog): instrument 63 operations endpoints (Wave 3)'"
  - "Excel import operParam schema is {filename,size,inserted,updated,failed} — never raw row data. This satisfies T-34-W3-01 (Information Disclosure: row data leak). Bounded size means it never exceeds 8192 bytes even for 1000-row imports"
  - "Excel entity-type to module name map (excelEntityModuleNames) — ensures audit rows for /ops/buildings/excel/import share module name 楼宇管理 with /ops/buildings CRUD handlers. Single source of truth avoids drift"
  - "Workstation_device SetPrimary uses OperTypeGrant(4) per T-34-W3-03 mitigation — semantically a permission grant (which device is primary). SyncAD/SyncAsset use OperTypeSync(14) — external system sync"
  - "Building Geocode endpoint uses OperTypeOther(0) with explicit WithOperParam({address:...}) — external Baidu Maps API call. Not a write to building data but an audit-worthy external call (data egress to third party)"
  - "Door module name 门窗管理 (not 门管理) per plan list — the Chinese CAD convention treats doors + windows as the same architectural element category"
requirements-completed: [F-OPLOG-W3]
metrics:
  duration: 22m
  completed: 2026-06-16T00:00:00Z
  tasks: 2
  files_created: 0
  files_modified: 14
  endpoints_instrumented: 56
---

# Phase 34 Plan 04: 运维模块操作日志全覆盖 (Wave 3) Summary

**One-liner:** 为 13 个运维 handler（building/floor/workstation/workstation_device/server_room/room_device/dedicated_line/infopoint/wall/door/floor_plan_text/excel/asset）的 56 个实际写端点各加一行 `operlog.Record`，用中文模块名，通过 `WithCore()` 链式注入 core 保留既有构造器签名；Excel 导入只记录 filename+size+inserted/updated/failed（绝不记录原始行 — T-34-W3-01 缓解），导出用 FilterSensitiveParams 过滤请求参数，模板下载用 OperTypeDownload(18)。

## What Was Built

### 56 个实际写端点全部埋点

| Handler | 模块名 | 端点（OperType） | 小计 |
|---------|--------|------------------|------|
| building_handler | 楼宇管理 | Create(1)/Update(2)/Delete(3)/Batch(16)/Geocode=Other(0) | 5 |
| floor_handler | 楼层管理 | Create/Update/Delete/Batch | 4 |
| workstation_handler | 工位管理 | Create/Update/Delete/Batch/BatchUpdatePositions=Batch | 5 |
| workstation_device_handler | 工位设备 | AddManual=Create/SetPrimaryAndSave=Update/SyncAD=Sync(14)/SyncAsset=Sync/Update/Delete/SetPrimary=Grant(4) | 7 |
| server_room_handler | 机房管理 | Create/Update/Delete/Batch | 4 |
| room_device_handler | 机房设备 | Create/Update/Delete/Batch | 4 |
| dedicated_line_handler | 专线管理 | Create/Update/Delete/Batch | 4 |
| infopoint_handler | 信息点管理 | Create/Update/Delete/Batch | 4 |
| wall_handler | 墙体管理 | Create/Update/Delete/Batch | 4 |
| door_handler | 门窗管理 | Create/Update/Delete/Batch | 4 |
| floor_plan_text_handler | 楼层文本 | Create/Update/Delete/Batch | 4 |
| excel_handler | 各实体名（按 entityType） | Import(6)/Export(5)+fallback/Download(18) | 3 端点 / 4 调用 |
| asset_handler | 资产管理 | Create/Update/Delete/Batch | 4 |
| **合计** | | | **56 端点 / 57 调用** |

每个 struct handler 写端点在成功路径末尾、`response.Success(...)` 之前插入一行：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeCreate)
```
Excel closure 端点（无 h.core 字段）直接传 core：
```go
operlog.Record(c, core.OperLogService, core.GetDB(), excelModuleName(entityType), operlog.OperTypeImport,
    operlog.WithOperParam(fmt.Sprintf(`{"filename":%q,"size":%d,"inserted":%d,"updated":%d,"failed":%d}`,
        file.Filename, file.Size, result.Inserted, result.Updated, result.Failed)))
```

### WithCore() 链式注入模式（沿用 34-02/34-03）

12 个 struct handler（building/floor/workstation/workstation_device/server_room/room_device/dedicated_line/infopoint/wall/door/floor_plan_text/asset）都加 core 字段 + WithCore() setter；router.go 在 12 个构造点链式注入：
```go
buildingHandler := operations.NewBuildingHandler(buildingService, geocodingService).WithCore(core)
floorHandler := operations.NewFloorHandler(floorService).WithCore(core)
// ... 10 more
```
excel_handler 是特例：它本就用 closure 模式 + 直接接收 core 参数（`importData(entityType, core)`），所以无需 WithCore — 直接调用 `operlog.Record(c, core.OperLogService, ...)`。

### Excel 导入 operParam 架构（T-34-W3-01 缓解）

Excel 导入是本波次审计价值最高的端点（"上周二谁导了哪批楼宇数据"）。设计：
```
{"filename":"buildings_20260615.xlsx","size":524288,"inserted":1000,"updated":50,"failed":3}
```
- **绝不**记录原始行内容（哪怕 FilterSensitiveParams 也无法遮蔽数千行业务数据）
- 大小有界（约 100 字节固定长度），永不触发 operlog 的 8192 字节截断告警
- 含 inserted/updated/failed 计数，审计可回答"这次导入产生了多少副作用"

### 威胁模型对照

| 威胁 ID | 缓解 | 证据 |
|---------|------|------|
| T-34-W3-01 (Excel 行数据泄露) | Excel 导入 operParam 只含 {filename,size,inserted,updated,failed}；绝不含原始行 | excel_handler.go:118-120 WithOperParam(...) |
| T-34-W3-02 (审计缺口) | 56 个写端点全部埋点 + 12 个中文模块名 | grep operlog.Record( = 57 调用 |
| T-34-W3-03 (绑定绕过审计) | workstation_device SetPrimary 用 OperTypeGrant(4)（语义匹配） | workstation_device_handler.go:309 |

## Deviations from Plan

### Architectural Decisions（非偏离，记录说明）

**1. "63 端点"目标 vs 实际 56 端点**
计划 must_haves 提到"All 63 operations endpoints trigger sys_oper_log inserts"。实际代码库中，这 13 个 handler **存在的写端点**只有 56 个：
- workstation: 5（plan 列 8 含 BindUser/UnbindUser/UpdateStatus/Import — 当前 handler 无这些方法，Excel 走 SetupExcelRouter）
- workstation_device: 7（plan 列 4，实际有 AddManual/SetPrimaryAndSave/SyncAD/SyncAsset/Update/Delete/SetPrimary）
- dedicated_line: 4（plan 列 5 含 UpdateStatus — 当前 handler 无）
- infopoint: 4（plan 列 5 含 BindPort — 当前 handler 无）
- excel: 3 个 handler 函数（plan 列 6 含 ImportBuilding/ImportWorkstation/ImportFloor/ImportInfoPoint — 实际是同一个 importData 闭包按 entityType 复用，共注册 8 次）
- asset: 4（plan 列 7 含 Import/Export/UpdateStatus — Excel 走 SetupExcelRouter，无 UpdateStatus 方法）

本计划对**所有存在的写端点**完成了 100% 埋点（56/56），完全满足"全模块覆盖"的实质要求。验证标准中的 `grep >= 63` 因端点总数本身只有 56 而无法达到，但 56/56 = 100% 覆盖了实际存在的写端点。此现象与 34-02（声称 47 实际 31）、34-03（声称 25 实际 23）相同 — 计划审计时基于权限定义/路由表，但 handler 方法实际不存在。

**2. 单一 commit 覆盖两个任务**
计划列出 Task 1（6 handler 文件）+ Task 2（7 handler 文件）。实际用单一 commit 覆盖，因为：
- router.go 在 Task 1 阶段就被改为对**所有 11 个** struct handler 调 `.WithCore(core)`（包括 Task 2 才动的 dedicated_line/infopoint/wall/door/floor_plan_text/asset）
- Task 1 单独 commit 无法编译 — 那 6 个 Task 2 handler 还没有 WithCore 方法
- 计划 success_criteria 明确允许："Single commit: feat(operlog): instrument 63 operations endpoints (Wave 3)"

如果分两个 commit，第一个 commit 必然是构建破坏的（red），违反"每个 commit 都构建通过"的硬性要求。选择单一 commit 是正确权衡。

**3. door 模块名用"门窗管理"而非"门管理"**
计划的 module-name 列表写"门窗管理"。CAD 建筑元素分类中门和窗通常同属一类（门窗表），保留计划的命名以保持审计视图与计划一致。

### Auto-fixed Issues

无。所有改动按计划执行，无需 Rule 1-3 修复。

## Known Stubs

无。所有 `operlog.Record` 调用均为完整实现，无占位、无 TODO、无 mock 数据。

## Threat Flags

无新增威胁面。计划 `<threat_model>` 中 T-34-W3-01 至 T-34-W3-03 全部已 mitigate（见上文威胁模型对照表）。Excel 导入 operParam 设计在审计完整性（够用回答"谁导了什么"）与数据最小化（不记录行内容）之间取得平衡。

## Verification Results

```
go build ./...                                  → exit 0
go vet ./...                                    → exit 0
go test -count=1 ./internal/api/v1/operations/  → ok (excel_magic_bytes_test PASS)
```

### operlog 调用计数

| Handler | 实际调用 | 实际写端点 | 状态 |
|---------|---------|-----------|------|
| building_handler.go | 5 | 5 | ✓ |
| floor_handler.go | 4 | 4 | ✓ |
| workstation_handler.go | 5 | 5 | ✓ |
| workstation_device_handler.go | 7 | 7 | ✓ |
| server_room_handler.go | 4 | 4 | ✓ |
| room_device_handler.go | 4 | 4 | ✓ |
| dedicated_line_handler.go | 4 | 4 | ✓ |
| infopoint_handler.go | 4 | 4 | ✓ |
| wall_handler.go | 4 | 4 | ✓ |
| door_handler.go | 4 | 4 | ✓ |
| floor_plan_text_handler.go | 4 | 4 | ✓ |
| excel_handler.go | 4 调用（含 1 个 Export JSON-marshal 失败回退）/ 3 端点 | 3 | ✓ |
| asset_handler.go | 4 | 4 | ✓ |
| **合计** | **57 调用 / 56 端点** | **56** | **✓ 100% 覆盖** |

### 预先存在的未提交 WIP（非本计划引入）

- `internal/api/v1/system/ad_domain_handler.go` — 修改但未提交（不属于本计划范围）
- `internal/scheduler/ad_sync_tasks.go` — 修改但未提交（不属于本计划范围）
- `xingran-react-frontend/src/types/operations.ts` 等前端文件 — 修改但未提交（不属于本计划范围）

这些 WIP 不影响本计划的 `go build ./...` / `go vet ./...` / `go test ./internal/api/v1/operations/...` 全部通过的验证结论。本计划只 stage 了 14 个本计划修改的文件，未触碰任何 WIP。

## Success Criteria 对照

- ✅ **F-OPLOG-W3**: 13 个运维 handler 的所有实际写端点（56 个）现在写 sys_oper_log 行
- ✅ Excel 导入用 OperTypeImport(6)，导出用 OperTypeExport(5)，模板下载用 OperTypeDownload(18)
- ✅ Excel 导入 operParam 只含 filename+size+inserted/updated/failed（T-34-W3-01 缓解）
- ✅ Excel 导出用 FilterSensitiveParams 过滤请求参数（防御性 — 即便当前无非敏感字段）
- ✅ 12 个中文模块名（楼宇管理/楼层管理/工位管理/工位设备/机房管理/机房设备/专线管理/信息点管理/墙体管理/门窗管理/楼层文本/资产管理）
- ✅ WithCore() 模式与 34-02/34-03 一致
- ✅ build / vet / operations 测试全绿
- ✅ workstation_device SetPrimary 用 OperTypeGrant(4)（T-34-W3-03 缓解）
- ✅ workstation_device SyncAD/SyncAsset 用 OperTypeSync(14)

## Self-Check: PASSED

- [x] `internal/api/v1/operations/building_handler.go` 存在且含 operlog.Record（FOUND，5 调用）
- [x] `internal/api/v1/operations/floor_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/workstation_handler.go` 存在且含 operlog.Record（FOUND，5 调用）
- [x] `internal/api/v1/operations/workstation_device_handler.go` 存在且含 operlog.Record（FOUND，7 调用含 Sync/Grant）
- [x] `internal/api/v1/operations/server_room_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/room_device_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/dedicated_line_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/infopoint_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/wall_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/door_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/floor_plan_text_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/operations/excel_handler.go` 存在且含 operlog.Record + WithOperParam（FOUND，4 调用含 Import/Export/Download）
- [x] `internal/api/v1/operations/asset_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/router.go` 12 个 handler 构造点全部 `.WithCore(core)`（FOUND）
- [x] commit `d7f8903` 存在于 git log（FOUND）
