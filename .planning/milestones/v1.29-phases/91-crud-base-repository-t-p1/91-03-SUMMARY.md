---
phase: 91-crud-base-repository-t-p1
plan: 03
subsystem: api
tags: [gorm, generic-repository, map-signature, scope-based, batch-migration, behavior-baseline, latent-bugfix]

# Dependency graph
requires:
  - phase: 91-01
    provides: base.GORMRepository[T] 六方法 scope 仓储 + Scope/PageParams + SortScope（B 型） + base.PageResult 单一定义
  - phase: 91-02
    provides: workstation pilot 迁移模式（joinScope/filterScope 组合 + white-box 构造 nil-panic 教训）
provides:
  - building/floor/asset 三服务 repo 化（map 签名不变的迁移形状唯一样本，91-04 无此形状）
  - map→scope 转换模板：extractPagination/extractSortRequest + filterScope 闭包包 applyFilters + 恒定过滤 scope 首位
  - floor 行为基线测试 6 锁（floor_service_crud_test.go，迁移前后逐字节同结果）
  - listRepo() nil-safe 访问器模式（白盒构造装饰器兼容，P7 零触碰解法）
  - F3 软删 Total 语义修复（building/asset，checkpoint 显式确认）
affects: [91-04 批量迁移收尾, 92 缓存层统一]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "map 签名不变迁移形状：List(ctx, map) 接口零改动，service 内部 extract→PageParams + filterScope/恒定 scope/SortScope 组合"
    - "filterScope 闭包包裹既有 applyFilters helper（Statistics 共用方不受影响），条件逐字平移零漂移"
    - "listRepo() 惰性兜底：白盒 &service{db: db} 构造（装饰器/测试）漏 repo 字段时动态补建，替代触碰 Phase 92 文件"

key-files:
  created:
    - internal/services/operations/floor_service_crud_test.go
  modified:
    - internal/services/operations/building_service.go
    - internal/services/operations/asset_service.go
    - internal/services/operations/floor_service.go
    - internal/services/operations/asset_listfilter_test.go

key-decisions:
  - "Rule 1：floor Update 换楼异步同步工位 building_id 的分支自仓库初始化起即死代码（oldBuildingID 捕获的是新值），从库中回读旧 building_id 修复——由 Task 1 基线测试暴露"
  - "floor 用 listRepo() nil-safe 访问器而非改 floor_cache_impl.go：生产路径 router.go:579 走装饰器的白盒内嵌构造，P7 红线（git diff 零改动）下唯一零触碰解法"
  - "asset_listfilter_test.go 仅改 2 处 List 调用方的白盒构造为 NewAssetService(db)，Statistics-only 调用方保持不动（91-02 口径）"
  - "building/asset filterScope 保留 applyFilters/applyDeptFilter 原函数、闭包内调用：building Statistics 仍直调 applyFilters，且条件逐字平移零漂移（T-91-03-04 缓解）"

requirements-completed: [CRUD-REUSE-03, CRUD-REUSE-04, CRUD-REUSE-05, CRUD-REUSE-07]

# Metrics
duration: 35min
completed: 2026-09-04
---

# Phase 91 Plan 03: building/floor/asset 批量迁移（map 签名不变）Summary

**building/floor/asset 三个 map 参数服务（共 1147 行）迁入 base.GORMRepository[T]：List 保持 map[string]interface{} 签名（F1）、floor 全部公开签名零变更（P7 装饰器锁定）、floor 迁移前 6 锁行为基线先行（Wave 0 缺口补齐）；过程中 Rule 1 修复 floor 换楼异步同步死分支（自初始化起失效的现存 bug），F3 软删 Total 修复经 checkpoint 确认**

## Performance

- **Duration:** 35 min
- **Started:** 2026-09-04T10:45:12Z
- **Completed:** 2026-09-04T11:20Z
- **Tasks:** 4/4（Task 4 checkpoint auto-mode 自动 approved）
- **Files:** 1 新增 + 4 修改（全部计划内）

## Accomplishments

- building/asset repo 化：struct 组合 `base.GORMRepository[operations.OpsBuilding]` / `base.GORMRepository[models.Asset]`，Create/Delete/GetByID/BatchDelete 四方法委托 repo；Update 保留各自 `Omit("CreatedAt").Save` 变体（零值覆盖 created_at bug 的修复，按 plan 不隐入通用 repo）；List map 签名不变
- building List 拆为 filterScope（闭包内调用既有 applyFilters，name LIKE / orgId 子树 / status 三条件逐字）+ selectScope（workstation_count 子查询 Select 逐字）+ base.SortScope（白名单 + `order_num ASC` 默认）
- asset List：Phase 48 D-07 的 `component_type IS NULL` 恒定过滤进 scopes 首位（注释逐字保留）+ filterScope（9 个 filter 逐字）+ base.SortScope（`created_at DESC` 默认）；asset_listfilter_test.go（D-07 行为锁）零断言改动全绿
- floor repo 化：仅 GetByID/List 管道进 repo——joinScope（Select+两 JOIN 逐字）、filterScope（4 条件含 CAST 写法与 orgId EXISTS 子查询逐字）、base.SortScope（`ops_floors.order_num ASC`）；List 的无 clamp 内联分页断言逐字保留（P5）；Create 软删恢复 / Update 异步同步 / Delete·BatchDelete 楼层计数编排全部原位保留
- P7 红线全程守住：FloorService interface 块零变更、floor_cache_impl.go `git diff` 为空；新增 `listRepo()` nil-safe 访问器让装饰器的白盒内嵌构造（生产路径 router.go:579）零 panic
- floor 行为基线先行：floor_service_crud_test.go 6 个 TestFloor91_* 锁（CRUD+join 字段 / 软删恢复分支 / 异步工位同步 Eventually / 楼层计数编排 / List 过滤矩阵+默认序 / 无 clamp 分页契约），迁移前对旧实现全绿，迁移后逐字节同结果
- calculateOffset 在 building/asset 两文件引用清零（helper 本体留待 91-04 修剪）

## Task Commits

1. **Task 1a: [Rule 1] floor Update 换楼异步同步死分支修复** - `dbdacd9` (fix)
2. **Task 1b: floor 行为基线测试（Wave 0 缺口，6 锁全绿）** - `039e6e7` (test)
3. **Task 2: building/asset repo 化（map 签名不变）+ asset_listfilter 白盒构造修复** - `7fb4531` (refactor)
4. **Task 3: floor GetByID/List 管道 repo 化（P7 签名锁定）** - `a918f44` (refactor)
5. **Task 4: Checkpoint F3/A3 软删 Total 语义修复确认** - 无代码变更，auto-mode 自动 approved（证据见下节）

## Checkpoint（Task 4）：building/asset Total 软删语义修复（F3/A3）— APPROVED（auto-mode）

- **内容:** 两服务旧代码 `.Table("ops_buildings")`/`.Table("ops_asset")` 起链导致 Count 不过滤软删行、Find 过滤（GORM stmt.Model==nil 时 Count 的 Dest 解析失败，RESEARCH F3 spike 实证）——列表 Total 比实际行数虚高。repo 统一 `Model(new(T))` 后 Total 与列表行数严格一致（方向为修 bug）
- **验证证据:**
  1. `go test ./internal/services/operations/ -run "TestAsset|TestBuilding|TestFloor91" -v` → 14 PASS 0 FAIL（现有测试零软列数据，零回归）
  2. 行为差异面：仅当存在软删除的楼宇/资产行时 total 比旧版小（不再计入已删行）；无软删数据时逐字节一致
  3. 依赖排查：全库 grep 确认 building/asset List 的唯一消费方是 `building_handler.go:108` / `asset_handler.go:111`（total 透传前端分页展示），无任何业务逻辑依赖「Total 含软删行」——该行为是 GORM 链式副作用的意外产物，非设计意图
- **审批方式:** orchestrator 顺序执行 auto-mode（`workflow._auto_chain_active=true`），非 package-legitimacy 类 checkpoint 自动 approved

## Files Created/Modified

- `internal/services/operations/floor_service_crud_test.go` - 新增 288 行：6 个 TestFloor91_* 行为基线锁；t.TempDir 文件库脚手架（异步 goroutine 与 glebarez :memory: 每连接独立库不兼容，注释说明）
- `internal/services/operations/building_service.go` - repo 组合 + 四方法委托 + List scope 化（filterScope/selectScope/SortScope）+ F3 语义注释；validateOrg/validateNameUnique/applyFilters/applyDeptFilter/Statistics/SearchBuildingOptions 零改动
- `internal/services/operations/asset_service.go` - repo 组合 + 四方法委托 + List scope 化（恒定过滤首位 + filterScope + SortScope）+ F3 语义注释；Update Omit 变体/validate 族/GetByDeviceSN/GetDeviceTypes 等零改动
- `internal/services/operations/floor_service.go` - repo 字段 + listRepo() 访问器 + joinScope/filterScope + GetByID/List 委托；interface 块与业务编排方法零改动（Update 另见 Rule 1 修复 commit dbdacd9）
- `internal/services/operations/asset_listfilter_test.go` - 2 处白盒 `&assetService{db: db}`（List 调用方）改 `NewAssetService(db)`（repo 字段缺失会 nil panic，91-02 同型修复）+ gofmt import 排序；Statistics-only 白盒构造（:92）按 91-02 口径保持不动

## Decisions Made

- **Rule 1 修复 floor 换楼异步同步死分支**（commit dbdacd9）：`oldBuildingID := floor.BuildingID` 捕获的是待写入的新值，`oldBuildingID != floor.BuildingID` 恒 false——`syncWorkstationBuildingID` 的 Update 触发分支自 ea528c6（仓库初始化）起从未生效，全库唯一调用点就是这个死分支（TestImp77 只白盒直调）。Task 1 基线测试按 plan 要求锁定「Update 换楼 → 工位同步」行为时暴露。修复为 Save 前从库中 `Select("building_id")` 回读旧值；TestImp77 全套零回归
- **listRepo() nil-safe 访问器**：floor_cache_impl.go:27 白盒 `&floorService{db: db}` 是生产路径（router.go:579 `NewFloorServiceWithCache`），repo 组合后装饰器委托 List/GetByID 必 nil panic；P7 禁止触碰该文件 → floor_service.go 内惰性兜底（GORMRepository 无状态，语义等价），Phase 92 收敛后可移除
- **TestFloor91_Create_SoftDeleteRestore 锁定 sqlite 实际语义**：plan 预期「恢复软删行」，但 sqlite 无 NOW()（P-77-2 既有锁定）→ 恢复 UPDATE 必失败，基线按实现锁「恢复分支进入 + 错误归因『恢复楼层失败』」，与 TestImp77_FloorCreate 口径一致
- **基线脚手架用 t.TempDir 文件库**：Update 的异步 goroutine 与测试主 goroutine 并发访问连接池，glebarez :memory: 每条连接是独立空库（geocoding_photo_floor_test.go DeletePhoto quirk 同源），异步同步会被路由到空库；文件库 + 显式 Close（Windows TempDir 清理需释放句柄）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] floor Update 换楼异步同步分支为死代码**
- **Found during:** Task 1（TestFloor91_Update_SyncsWorkstation 对未迁移实现 FAIL——基线红灯）
- **Issue:** `oldBuildingID := floor.BuildingID` 与后续比较项为同一变量，恒 false；楼层换楼后挂靠工位的 building_id 从未被同步（孤立数据 bug，生产自初始化起存在）
- **Fix:** Save 前从库中回读修改前的 building_id（`Select("building_id").Where("id = ?")`），读失败保守跳过同步
- **Files modified:** internal/services/operations/floor_service.go
- **Commit:** dbdacd9（独立原子 commit，可单独 revert）

**2. [Rule 3 - 阻塞修复] asset_listfilter_test.go 白盒构造漏 repo 字段**
- **Found during:** Task 2 实现时预判（91-02 同型教训：`&assetService{db: db}` 在 List 委托 s.repo 后 nil panic）
- **Fix:** 2 处 List 调用方改 `NewAssetService(db)`；plan 验收「零改动」按字面不可达（结构性冲突），phase context 已预授权该替代（「如遇同类测试用 NewXxxService(db) 替代」）
- **Files modified:** internal/services/operations/asset_listfilter_test.go
- **Commit:** 7fb4531

### Plan 事实修正（非代码偏差）

- **plan 验收 grep 口径**：`grep -c "List(ctx context.Context, params map\[string\]interface{})"` 三文件各为 **2**（interface 声明 + 方法实现各 1），plan 写「各为 1」低估——意图（签名不变）达成，迁移前后计数一致
- **gofmt**：asset_listfilter_test.go 的 import 排序为迁移前即存在的不规范，随触碰一并 gofmt -w（包内其余 4+ 个未触碰文件的历史 gofmt 问题按 scope constrainment 不动）

## Issues Encountered

- Task 1 基线首次运行 2 类红灯：①sync 用例 FAIL（Rule 1 真 bug，非测试基建——file DB 复测排除 ：memory: 假设后定位死分支）；②file DB 在 Windows 下 TempDir 清理失败（补 t.Cleanup 显式 Close）——均已修复，教训：glebarez :memory: 多连接隔离 + Windows 文件句柄是 floor 异步路径测试的两个基建坑
- 全量 `go test ./...` 单次 >10min（74 包），终验放后台执行

## Verification Results

- `go build ./...` 退出码 0（每 task 收尾均过）
- `go test ./internal/services/operations/ -run "TestAsset|TestBuilding|TestFloor91" -v` → 14 PASS 0 FAIL
- `go test ./internal/services/operations/...` 全包 0 失败（TestFloor91 基线 + TestImp77 14+1 锁 + asset/building/roomdevice/statistics/listfilter 全绿）
- `go test ./internal/api/v1/operations/...` + `./internal/services/base/...` 全绿（handler smoke + base 泛型契约）
- **全量：`go test ./...` 74 包 ok / 0 失败**（exit 0，status_constants AST 锁值 + operlog 回归含内）
- `git diff floor_cache_impl.go` 为空（P7 红线实证）
- map 签名 grep：三文件 interface + 实现 List 签名迁移前后逐字一致；calculateOffset 两文件引用 0；`component_type IS NULL` 在 asset List scopes 首位
- gofmt：本次触碰文件全部格式干净

## User Setup Required

None - 纯内部重构 + 1 个 latent bugfix，无外部服务配置。

## Next Phase Readiness

- 91-04 收尾（7 个 typed 服务 + typesafe 死文件清理 + pagination_helper 修剪）：door 模板（PATTERNS Pattern 1）+ 本 plan 的 map 形状已全备；calculateOffset/extractPagination 修剪时注意 floor 的内联断言不消费这些 helper（无依赖）
- 92 缓存层统一可安全触碰 floor_cache_impl.go：届时 listRepo() 兜底可随装饰器改造移除（注释已标注）
- F3/A3 checkpoint 已闭环，RESEARCH Open Questions 1/2/3 全部 RESOLVED 落地
- 无阻塞项

## Self-Check: PASSED

- 5 个文件全部存在（1 新增 + 4 修改）
- 4 个 task commit 在 git log：dbdacd9 / 039e6e7 / 7fb4531 / a918f44
- 终验：`go build ./...` 0 错误；`go test ./...` 74 包 ok 0 失败

---
*Phase: 91-crud-base-repository-t-p1*
*Completed: 2026-09-04*
