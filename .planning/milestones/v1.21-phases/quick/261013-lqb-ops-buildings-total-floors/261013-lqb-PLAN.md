# 根治 ops_buildings.total_floors 反范式列漂移 — 写路径全量重算 + 存量修复

**Mode**: quick
**Date**: 2026-10-13
**Slug**: 261013-lqb-ops-buildings-total-floors

## Context

前序任务 261013-jf1 已把楼栋列表"楼层数"列切实时 `FloorCount`（显示层免疫），本任务根治**写路径**，让 `total_floors` 列本身回归"根据创建的楼层自动计算"（`building.go:25` 模型注释的既定语义）。

**探索确认的现状（floor_service.go 实读）：**

| 写路径 | total_floors 重算 | 位置 |
|--------|------------------|------|
| 楼层 Create（含软删恢复） | ✅ 已有 | `floor_service.go:137/145` 调 `updateBuildingFloorCount` |
| 楼层 Delete | ✅ 已有 | `floor_service.go:220` |
| 楼层 BatchDelete | ✅ 已有（子查询批量 SQL） | `floor_service.go:468-487` |
| **楼层 Update 换楼（路径 A，主目标）** | ❌ **全程零重算** | `floor_service.go:148-210` 无任何调用；91-03 修复后 `oldBuildingID` 从库读取已生效（:154-160），乐观锁换楼 UPDATE 成功后新旧楼栋计数均不动 |
| **楼宇 Excel 导入（路径 B）** | ❌ **直写用户上传值** | `excel_service.go:1350-1352` `parseFieldValue` 对 `totalFloors` 走 `strconv.Atoi`，经 `BatchUpsert` 落库，绕过 service 层 |
| **楼层 Excel 导入（路径 C，本次探索新发现）** | ❌ **绕过 Create 重算** | `excel_config.go:118` 存在 `"floor"` ExcelConfig；楼层行经通用 `BatchUpsert` 直写 `ops_floors`，不触发 `floorService.Create` → 所属楼栋计数不更新 |

**total_floors 仍有的消费方（grep 全量确认 → 列不可退役，必须保准确 + 修存量）：**

1. 排序白名单 `building_service.go:82`（`"totalFloors": "total_floors"`，前端 `buildings/index.tsx:96/416` 仍发 totalFloors 排序）— 列漂移直接导致排序错乱
2. 楼宇空间卡片 `building-spaces/components/BuildingCard.tsx:30`（渲染 `building.totalFloors || 0`）
3. 楼栋概览抽屉 `buildings/index.tsx:524`（`{building.totalFloors || 0}层`）
4. Excel 导出 `excel_export_config.go:146`（`totalFloors` → `total_floors` 列映射）
5. 楼栋表单字段 `buildings/index.tsx:792`（`Form.Item name="totalFloors"`）

**既有重算基建（全部复用，零新概念）：**

- `updateBuildingFloorCount(ctx, buildingID)`（`floor_service.go:507-513`）：`Model(&OpsFloor{}).Count` 自动附 `deleted_at IS NULL` → 实时 COUNT 口径，幂等
- `BatchDelete` 的子查询批量 UPDATE SQL（`floor_service.go:468-487`）— 路径 B/C 与迁移的 SQL 形状直接拷贝它
- 测试基建 `newFloor91TestDB` / `seedFloor91Building` / `floor91BuildingTotalFloors`（`floor_service_crud_test.go:34/56/64`，同包直用）

**方案取舍（写明）：**

- **重算口径** = 实时 `COUNT(*)`（软删排除），禁止 ±1 增量 — 防历史脏数据固化，重算幂等且天然自愈
- **Update 不做事务重构**：乐观锁（V130R-07）已保证换楼 UPDATE 本身原子；重算幂等 + 每次普通编辑也重算当前楼栋 → 任何部分失败后的重试自愈（重试走 Save 分支仍会重算）。保持 Update 现有非事务结构，最小改动
- **路径 B/C 纳入**（评估结论：纳入）：模型注释锁定派生语义 + 上述 5 个消费方都会被导入值立即污染；修复 = 事务内一条幂等 UPDATE（复用 BatchDelete SQL 形状），成本 ~10 行。反方向顾虑（导入楼宇时楼层尚未导入、用户声明值被清零）不成立——楼层导入（路径 C）同样触发重算，导入闭环后计数即为真值
- **存量修复 = migration（非 DB TRIGGER）**：用户偏好代码层根因修复、不改 PG schema object（user-prefers-code-fixes-no-db-triggers）；一次性数据修复迁移是本仓库标准实践（`migration_205` 等先例），迁移结束即退出，不留任何常驻 schema object
- **明确不修（scope constrainment）**：楼栋表单/手工 Create+Update 路径（`index.tsx:792` → building_service）写用户声明值 — planning context 未列此路径，不主动扩scope；显示层已免疫（jf1 FloorCount），且任何后续楼层事件或 Excel 导入会经本任务新增的重算路径自愈该楼栋。留作已知残余向量，如需根治是 5 行 follow-up

## Constraints

| 约束 | 来源 |
|------|------|
| 禁止 DB TRIGGER / PG schema object 修改；存量修复只允许一次性数据迁移（跑完即完，无常驻对象） | planning context + user-prefers-code-fixes-no-db-triggers |
| 重算只用实时 COUNT（软删排除），禁止 ±1 增量 | planning context 方向 3 |
| 不新增 operlog 调用（楼层 handler 现状保持，不因本任务加） | planning context |
| Handler/Router 零改动，纯 service + migration 层；**前端零改动**（消费方自动受益于列变准） | CLAUDE.md Handler-Service |
| 迁移必须 `migration_213_*.go` Go 函数 + `database.go` 显式注册（.sql 文件不会被自动加载） | MEMORY: ruoyi-migrations-no-sql-autoloader |
| 迁移 SQL 方言中立（PG + sqlite 双分支注册，`Migrate212` 先例）；禁止 `isPostgreSQL` 守卫（纯 SQL 无需） | `database.go:869/893` 双分支先例 |
| Excel 导入重算放事务内、错误上抛回滚全量导入（P2-A8 all-or-nothing，对齐 `:475-487` user 默认角色 hook） | `excel_service.go:427-492` 既有事务语义 |
| 改 Go 后必须 `go build ./...`；一次修一个文件再 build，禁止批量修 | CLAUDE.md |
| Windows 环境；主树执行（非 worktree）；工作区有大量未提交修改属正常现状 | planning context |

## Tasks

### Task 1: floor_service.Update 换楼双侧重算 + 普通编辑自愈重算 + 单测

**Files:**
- `internal/services/operations/floor_service.go`（修改）
- `internal/services/operations/floor_update_total_floors_test.go`（新建）

**Action:**

1. `floor_service.go` `Update` 方法（:148-210）：在函数末尾 `return nil` 之前、工位同步 goroutine 块（:198-207）之后，追加重算逻辑：
   - 若 `floor.BuildingID != ""`：调 `s.updateBuildingFloorCount(ctx, floor.BuildingID)`，error 上抛（对齐 Delete 路径 fail-loud 行为）。
   - 若同时 `oldBuildingID != "" && oldBuildingID != floor.BuildingID`（换楼）：再调 `s.updateBuildingFloorCount(ctx, oldBuildingID)`，error 上抛。
   - 附注释说明三点：① 换楼后新旧楼栋计数都要重算（修复漂移路径 A — 91-03 修好了换楼同步但漏了计数）；② 普通编辑也重算当前楼栋是**自愈设计** — 部分失败后重试走 Save 分支仍触发重算，历史漂移楼栋在任何楼层编辑后归正；③ `updateBuildingFloorCount` 用 `Model(&OpsFloor{})` 实时 COUNT（自动软删过滤），幂等，禁止改 ±1 增量。
   - **不要**动乐观锁分支结构、`Save` 分支逻辑、goroutine 同步块（:198-207 原样保留）。

2. 新建 `floor_update_total_floors_test.go`（package `operations`），复用 `newFloor91TestDB` / `seedFloor91Building` / `floor91BuildingTotalFloors`，service 用 `NewFloorService(db)`。测试用例（楼层用 `db.Create(&operations.OpsFloor{Name:..., FloorNo:..., BuildingID: b.ID})` 直建，同楼 FloorNo 互异；Update 入参须带全字段 — 换楼分支按 map 写 name/floor_no/area/order_num/status/remark）：
   - `TestFloorUpdateTotalFloors_BuildingChangeRecountsBothSides`：B1 挂 2 层、B2 挂 0 层；`Update` 把其中一层换到 B2 → 断言 B1 total=1、B2 total=1。
   - `TestFloorUpdateTotalFloors_SameBuildingEditSelfHeals`：B1 挂 2 层，先 `db.Model(&OpsBuilding{}).Update("total_floors", 99)` 人工制造漂移；`Update` 同楼改名（BuildingID 不变）→ 断言 B1 total=2（漂移被编辑动作自愈）。
   - `TestFloorUpdateTotalFloors_MoveLastFloorOldBuildingZero`：B1 挂 1 层；换楼到 B2 → 断言 B1 total=0、B2 total=1（旧楼清零边界）。

**Verify:**

```bash
go build ./...
go test ./internal/services/operations/ -run TestFloorUpdateTotalFloors -v
go test ./internal/services/operations/
```

**Done:** 3 个新测试 PASS；`floor_service_crud_test.go` 既有 total_floors 用例（:171-194 编排测试）零回归。

### Task 2: Excel 导入重算 hook（building + floor 双实体）+ 共享重算 helper + 单测

**Files:**
- `internal/services/operations/floor_service.go`（修改 — 追加 helper）
- `internal/services/operations/excel_service.go`（修改 — 导入 hook）
- `internal/services/operations/excel_total_floors_recompute_test.go`（新建）

**Action:**

1. `floor_service.go` 在 `updateBuildingFloorCount`（:507-513）之后追加包级函数 `recomputeAllBuildingTotalFloors(ctx context.Context, db *gorm.DB) error`：单条 `db.WithContext(ctx).Exec` 执行全楼栋重算 UPDATE — SQL 形状逐字拷贝 `BatchDelete` 的 :468-475 关联子查询部分，去掉 `WHERE id IN (...)` 限定，外加 `WHERE deleted_at IS NULL`（只重算未删楼栋）。文档注释写明：① 供 Excel 导入事务调用（building + floor 双实体，修路径 B/C）；② 幂等可重入；③ 实时 COUNT 口径与 `updateBuildingFloorCount` 一致；④ ops_buildings 量级小（百级行），全表重算无性能顾虑。

2. `excel_service.go` `ImportData` 事务块内（:427-492）：在 user 默认角色 hook（:475-487）之后、`return nil` 之前追加：

   - 条件 `entityType == "building" || entityType == "floor"`（building 修路径 B、floor 修路径 C）时调 `recomputeAllBuildingTotalFloors(ctx, tx)`，error 直接 `return err` → 整个导入事务回滚（对齐 P2-A8 all-or-nothing 语义与 user hook 的错误处理）。附注释说明双实体触发原因（building 直写上传值 / floor 绕过 service Create）与回滚语义。

3. 新建 `excel_total_floors_recompute_test.go`（package `operations`），直测 helper（newFloor91TestDB）：B1 挂 2 层（软删其 1）、B2 挂 1 层、B3 楼栋本身软删（挂 1 层）；先把 B1/B2/B3 的 total_floors 人工置为 99/0/777；调 `recomputeAllBuildingTotalFloors(ctx, db)` → 断言 B1=1、B2=1、B3 **保持 777**（软删楼栋不在重算范围）；再跑一次 helper 断言结果不变（幂等）。

**Verify:**

```bash
go build ./...
go test ./internal/services/operations/ -run TestRecomputeAllBuildingTotalFloors -v
go test ./internal/services/operations/
```

**Done:** helper 测试 3 组断言 PASS；既有 `excel_import_rest_77_03_test.go` 等导入测试零回归。

### Task 3: Migrate213 存量重算迁移 + database.go 双方言注册 + 迁移单测 + 全量验证门禁

**Files:**
- `internal/core/db/migrations/migration_213_recompute_building_total_floors.go`（新建）
- `internal/core/db/migrations/migration_213_recompute_building_total_floors_test.go`（新建）
- `internal/core/db/database.go`（修改 — 两处注册）

**Action:**

1. 新建 `Migrate213RecomputeBuildingTotalFloors(db *gorm.DB) error`，结构逐字对照 `migration_205_rpa_worker_id_default.go`（`log.Println` 起止记账 + `fmt.Errorf` 包装 + `applogger.Infof` 完成日志）。**不加** `isPostgreSQL` 守卫（纯 SQL 双方言通用）。文档注释写明背景：floor Update 换楼全程不重算（261013-lqb 之前的历史缺陷）+ Excel 导入直写造成的存量漂移一次性归正；幂等，重复执行结果一致；一次性数据修复、不产生任何常驻 schema object（禁 TRIGGER 是用户明确偏好）。函数体一条 `db.Exec` 执行与 Task 2 helper 相同的重算 UPDATE。

2. `database.go` 两处注册（`Migrate212` 之后，先例 `:869-871` / `:893-895`）：
   - PG advisory-lock 块内（Migrate212 注册之后）
   - sqlite 分支块内（sqlite Migrate212 注册之后、`ensureSQLiteReconciliationViews` 之前）
   - 两处均按非阻断模式：`applogger.Errorf("楼栋 total_floors 存量重算失败 (非阻断,留待下次启动): %v", err)`，各附一行中文注释说明用途。

3. 新建迁移单测（package `migrations`），参照 `migration_210_normalize_menu_parent_id_test.go` 的测试脚手架与 sqlite 建库方式；业务表用**裸 SQL 建最小表**（先例 `excel_import_rest_77_03_test.go:52`，避免 AutoMigrate 耦合）：`ops_buildings(id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME, total_floors INTEGER DEFAULT 0)` + `ops_floors(id TEXT PRIMARY KEY, building_id TEXT, deleted_at DATETIME)`。用例：seed 2 栋楼（B1 两层含一软删、B2 一层）+ total_floors 脏值（99/0）→ 跑 `Migrate213` → 断言 B1=1、B2=1；**再跑一次**断言不变（幂等）。

4. 全量验证门禁（全绿才算 Done）：

```bash
go build ./...
go test ./internal/services/operations/ ./internal/core/db/...
```

**Verify:** 上述命令全部退出码 0；`git status` 确认迁移文件与 database.go 改动在工作树。

**Done:** Migrate213 落地且双方言注册；迁移单测含幂等断言全绿；operations 包 + db 包测试零回归。不要求人工 UAT（写路径语义由 Task 1/2/3 共 5 个新测试锁定；消费方零改动无需前端验证）。

## Success Criteria

- 楼层换楼后，旧/新楼栋 total_floors 立即等于各自 `ops_floors` 非软删行数（路径 A 关闭）
- 楼层普通编辑触发所在楼栋重算 — 历史漂移楼栋随任意楼层编辑自愈
- 楼宇/楼层 Excel 导入完成后，受影响楼栋 total_floors 与实际楼层行数一致（路径 B/C 关闭；用户上传值被纠正）
- 应用启动时 Migrate213 一次性归正存量漂移 — 排序白名单（building_service.go:82）、BuildingCard、概览抽屉、Excel 导出消费到正确值
- 全程零 DB TRIGGER / schema object、零 operlog 改动、零前端改动
- `go build ./...` + operations 包 + core/db 包测试全绿
