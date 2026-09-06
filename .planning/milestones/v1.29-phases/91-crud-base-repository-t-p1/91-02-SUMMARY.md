---
phase: 91-crud-base-repository-t-p1
plan: 02
subsystem: api
tags: [gorm, generic-repository, typed-request, scope-based, pilot-migration, behavior-locks]

# Dependency graph
requires:
  - phase: 91-01
    provides: base.GORMRepository[T] 六方法 scope 仓储 + Scope/PageParams + SortScope（B 型） + base.PageResult 单一定义
  - phase: 77-03
    provides: workstation_floor_code_77_03_test.go 行为锁（TestImp77，pilot 零变更证明器）
provides:
  - workstation_service 全量 repo 化模板（pilot）：joinScope/filterScope + base.SortScope 组合
  - D-05 typed request 全套接线样板：struct 扩字段 → 接口 typed 化 → handler 降级 bind → 测试字面量改写（91-03/91-04 复制）
  - WorkstationListRequest +3 可选字段（FloorCode/Type *int/OrgID，加性 JSON 契约）
  - status/type -1 跳过语义在 typed 路径的行为锁（含新增 type<-1 用例）
affects: [91-03 building/floor/asset 迁移, 91-04 批量迁移收尾, 92 缓存层统一]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "pilot 迁移五步法：requests 扩字段 → 接口 typed 化 → struct 组 repo → 六方法委托 → filter/join scope 化（条件逐字平移）"
    - "handler typed bind 降级范式：bind 失败 → 零值 request 继续查询（不用 handleJSONBinding 的 400 路径）"
    - "-1 跳过语义 typed 化：req.GetStatus(-1) + *req.Type >= 0 双守卫"

key-files:
  created: []
  modified:
    - internal/api/v1/operations/requests/workstation_requests.go
    - internal/services/operations/workstation_service.go
    - internal/api/v1/operations/workstation_handler.go
    - internal/services/operations/workstation_floor_code_77_03_test.go
    - internal/api/v1/operations/workstation_handler_test.go
    - internal/api/v1/operations/workstation_handler_full_test.go
    - internal/services/operations/workstation_update_createdat_test.go

key-decisions:
  - "filterScope 签名保持 plan 规定的 base.Scope 单返回值；floorCode 白名单守卫上提至 List 函数体（scope 闭包无法 return error，错误文案与触发时机逐字保留）"
  - "SearchWorkstationOptions 复用 WorkstationListRequest（plan interfaces 节授权），不新增重复 struct"
  - "迁移后 joinScope 前置于 Count：6 表全为按主键 N:1 LEFT JOIN 无行增殖，Count 不变（91-01 spike + TestImp77 Total 断言双守护）"
  - "floor 测试的 4 处 map List 调用保持不动（floor 接口签名被装饰器锁定 P7，与 workstation 的 typed 化无关）"

requirements-completed: [CRUD-REUSE-02]

# Metrics
duration: 18min
completed: 2026-09-04
---

# Phase 91 Plan 02: workstation pilot 迁移 Summary

**workstation_service（456 行最异类服务）全量迁到 base.GORMRepository[models.Workstation]：六方法 CRUD 模板清零 + D-05 typed request 全套接线（struct 扩 3 字段/接口 typed 化/handler 降级 bind/3 测试文件 16 处字面量改写），14+1 个行为锁全绿证明零行为变更**

## Performance

- **Duration:** 18 min
- **Started:** 2026-09-04T10:14:41Z
- **Completed:** 2026-09-04T10:33Z
- **Tasks:** 3/3（Task 3 checkpoint auto-mode 自动 approved）
- **Files modified:** 7（6 计划内 + 1 Rule 3 阻塞修复）

## Accomplishments

- workstation 六方法 CRUD（Create/Update/Delete/GetByID/List/BatchDelete）全部经 `base.GORMRepository[models.Workstation]` 执行，直调 GORM 的重复查询管道清零；`db` 字段保留给 Statistics/SearchOptions/BatchUpdatePositions/Update First 回填（D-04）
- D-05 typed 化全套接线：接口 List/SearchWorkstationOptions 改 `requests.WorkstationListRequest`（其余方法签名逐字不动）；handler 2 处 typed bind，bind 失败降级零值请求（与迁移前空 map 语义一致，不返回 400）
- WorkstationListRequest 加性新增 FloorCode/Type(*int)/OrgID 三字段；BuildingID/Code 注释锁定「服务从不消费」；前端 JSON 契约零变更
- List 拆为 joinScope（Select+6 表 LEFT JOIN 常量逐字平移）+ filterScope（六条件逐字平移，含 EXISTS 子查询与 CAST 写法）+ base.SortScope（白名单 + 默认序 `sys_workstation.created_at DESC`）
- P6 语义锁保留：status 经 `req.GetStatus(-1)` 且 `st >= 0` 才过滤；type 经 `req.Type != nil && *req.Type >= 0`；新增 List type<-1 跳过用例补齐 typed 路径行为锁
- Statistics/GetWorkstationDeptOptions/BatchUpdatePositions/排序白名单/validateTableName/occupancy 联动/Update First 回填零改动
- workstation 消费点清零达成：extractPagination/calculateOffset/extractSortRequest/extractIntParam 在 workstation_service.go 零引用；extractStringParam 仅剩 Statistics 1 处（合法保留，F1 口径）
- 3 个测试文件改写：16 处 map 字面量 → typed 复合字面量（断言逐字不变）；handler stub/mock 签名同步（含 ListFunc/SearchWorkstationOptionsFunc 字段类型与 6 处内联实现）

## Task Commits

1. **Task 1: WorkstationListRequest 扩 3 字段 + workstation_service repo 化 + D-05 typed 化** - `edc51fd` (refactor)
2. **Task 2: handler 2 处 typed bind（降级语义）+ 3 个测试文件改写** - `50577a5` (refactor)
3. **Task 3: Checkpoint workstation 分页语义收紧确认（A2）** - 无代码变更，auto-mode 自动 approved（证据见下节）

## Checkpoint（Task 3）：分页语义收紧 A2 — APPROVED（auto-mode）

- **内容:** workstation List/SearchOptions 分页归一化从 map 路径（clamp 上限 MaxOptionsPageSize=10000、current 不守卫）切换到 typed GetPagination（上限 MaxListPageSize=100、current<1 回退 1，修复负 offset latent bug）；另畸形 status/type 值从静默强转变为 bind 失败 → 降级零值请求（过滤全跳过）
- **验证证据:**
  1. `go test ./internal/services/operations/ -run TestImp77 -v` 全绿（14 个既有行为锁 + 1 个新增 type<-1 锁）
  2. 前端常规列表页发送 pageSize ≤100（分页组件口径）；SearchWorkstationOptions 自身另受 DropdownMaxRows=50 硬 LIMIT 约束，无依赖大 pageSize 拉全量的下游调用方
  3. 两项收紧（pageSize 上限 100、current<1 回退 1）按 plan 预授权接受
- **审批方式:** orchestrator 顺序执行 auto-mode（`workflow._auto_chain_active=true`），非 package-legitimacy 类 checkpoint 自动 approved

## Files Created/Modified

- `internal/api/v1/operations/requests/workstation_requests.go` - +FloorCode/Type(*int)/OrgID 三字段；BuildingID/Code 消费锁定注释（gofmt 对齐）
- `internal/services/operations/workstation_service.go` - 接口 typed 化 + repo 组合 + joinScope/filterScope scope 化 + 六方法委托；Statistics 等独有方法零改动
- `internal/api/v1/operations/workstation_handler.go` - List/SearchWorkstationOptions 2 处 typed bind（降级语义）+ requests import；struct 字段 gofmt 对齐
- `internal/services/operations/workstation_floor_code_77_03_test.go` - 16 处 map 字面量 → typed 字面量 + intPtr helper + 新增 type<-1 跳过用例；floor 段测试保持 map（P7）
- `internal/api/v1/operations/workstation_handler_test.go` - stubWorkstationService List/SearchWorkstationOptions 签名 typed 化
- `internal/api/v1/operations/workstation_handler_full_test.go` - mockWorkstationService 签名 + ListFunc/SearchWorkstationOptionsFunc 字段类型 + 6 处内联实现 typed 化
- `internal/services/operations/workstation_update_createdat_test.go` - [Rule 3] 2 处白盒 `&workstationService{db: db}` 构造改 `NewWorkstationService(db)`（repo 组合后白盒构造漏 repo 字段 → nil 指针 panic）

## Decisions Made

- **floorCode 白名单守卫上提至 List 函数体**：scope 闭包签名为 `func(*gorm.DB) *gorm.DB` 无法返回 error；`validateTableName(floorTable)` 是编译期常量恒真死分支，守卫+错误文案 `invalid table name: %s` 逐字保留在 repo 调用前（SearchWorkstationOptions 路径内联保留原位）
- **joinScope 前置 Count 的行为等价论证写入 List/joinScope 注释**：6 表 JOIN 全部按主键 N:1 无行增殖，Count 结果不变；Count 时 Select 临时替换 count(*) 并恢复（91-01 TestBase91 契约锁定）
- **新增 List type<-1 跳过用例**：plan Task 2 提及「type:-1 用例」但迁移前测试文件并不存在该用例；typed 路径的 `*req.Type >= 0` 是新守卫分支，补 6 行用例使 plan 声称的行为锁成真（T-91-02-01/P6 缓解）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - 阻塞修复] 白盒构造 workstationService 漏 repo 字段导致 nil panic**
- **Found during:** Task 2 verify（TestWorkstationUpdate_PreservesCreatedAt panic）
- **Issue:** `workstation_update_createdat_test.go` 两处 `svc := &workstationService{db: db}` 白盒构造，Task 1 struct 增加 repo 字段后 repo 为 nil，Update 落库 `s.repo.Update` 触发 nil 指针 panic
- **Fix:** 改经 `NewWorkstationService(db)` 构造（黑盒，含 repo 初始化）；同型白盒构造 `operations_statistics_test.go:181` 仅调 Statistics（只用 db）不受影响，保持不动
- **Files modified:** internal/services/operations/workstation_update_createdat_test.go
- **Commit:** 50577a5

### Plan 事实修正（非代码偏差）

- **plan 引用的「:415 buildingId 忽略行为锁」实为 floor 的 List 测试**：`workstation_floor_code_77_03_test.go:415`（现 :436）是 `TestImp77_FloorGetTree_GetByID_List` 中 floor service 的 `List buildingId 等值` 用例——floor_service.go:190 真实消费 buildingId，并非 workstation 的忽略行为证明；workstation 测试段没有任何 buildingId 用例。处置：WorkstationListRequest.BuildingID/Code 按计划注释锁定不消费（真实行为不变），floor 段 map 调用保持原样（floor 接口签名被装饰器锁定 P7）
- **Task 2 验收「svc.List(ctx, map 计数为 0」按 workstation 口径达成**：文件内剩余 4 处 map List 调用全部属于 floor service 测试（不同接口、签名不变），workstation 段 16 处全部 typed 化 + SearchWorkstationOptions map 调用计数 0
- **gofmt 对齐**：workstation_handler.go 的 WorkstationHandler struct 与 mock struct 字段对齐为迁移前即存在的不规范缩进，随本次触碰文件一并 gofmt -w（格式零行为变更）

## Issues Encountered

- Task 2 首轮全包测试出现 `TestWorkstationUpdate_PreservesCreatedAt` nil panic（见 Rule 3 修复），修复后全绿——plan 未把该白盒构造测试文件列入 read_first/files_modified，属 pilot struct 变更的连带适配面
- 91-01 收尾时 ROADMAP Progress 表 Phase 91 行未同步（checkbox 已勾但 Plans 计数留 0/4），本次一并修正为 2/4

## Verification Results

- `go build ./...` 退出码 0（Task 1 收窄 build + Task 2 全量 build 均过）
- `go test ./internal/services/operations/ -run TestImp77 -v` 全绿（含 :145 status:-1 跳过、SearchWorkstationOptions status=-1 跳过、buildingId 无用例需忽略、orgId EXISTS 路径）
- `go test ./internal/services/operations/... ./internal/api/v1/operations/...` 0 失败（TestWorkstationHandler_Workstation 全套含 List_BindErrorFallback/SearchOptions_InvalidJSON 降级路径）
- `go test ./...` 全仓 0 失败（AST 锁值 + operlog 回归 + base TestBase91 契约 + 全量 1688+ tests 零回归）
- grep 验证：workstation_service.go 中 extractPagination/calculateOffset/extractSortRequest/extractIntParam 计数 0、extractStringParam 仅 Statistics 1 处；测试文件 workstation map 调用计数 0；handler List/SearchWorkstationOptions 无 map bind
- gofmt：本次触碰文件全部格式干净

## User Setup Required

None - 纯内部重构，无外部服务配置。

## Next Phase Readiness

- 91-03（building + floor + asset）可直接复制 pilot 模式：map 签名服务保持 `List(ctx, map)` 接口，内部 extract→scope 转换；floor 先补 service 级行为基线测试（Wave 0 缺口）；F3 软删 Total 修复 checkpoint 在 91-03 Task 4
- 91-04 收尾 7 个 typed 服务可全按 door 模板 + pilot 的 typed bind 降级范式批量处理
- A3（building/asset 软删 Total）checkpoint 尚未触发，随 91-03 执行
- 无阻塞项

## Self-Check: PASSED

- 7 个修改文件全部存在（含 Rule 3 修复文件）
- 2 个 task commit 在 git log：edc51fd / 50577a5
- 终验：`go build ./...` 0 错误；`go test ./...` 0 失败

---
*Phase: 91-crud-base-repository-t-p1*
*Completed: 2026-09-04*
