---
phase: 149
plan: 01
status: complete
commit: false
REQ: SCOPE-01, SCOPE-02, SCOPE-03, SCOPE-04
---

# Phase 149: SCOPE 豁免面收口（SCOPE-01/02/03/04）— 执行总结

> `commit:false` —— **未执行任何 git 写操作**（`add` / `commit` / `stash` / `checkout` /
> `reset` / `cherry-pick`），由 Lead 在 W2 收口统一提交。`git diff` / `git status` 仅只读核对。
> 3 个 executor 并行写同一工作区，本 phase 严格限定在独占文件面内。

---

## 0. 结论

**方案 D（零迁移）落地完成**：5 类显式豁免面由「靠约定（挂了中间件但不消费 / 连中间件都没挂）」
升级为「靠机制（读端点真的消费 `ScopeResult`、写端点真的 403 / 裁剪、保存侧真的不误删）」。
零迁移、响应形状不变、不引入整批 403、All / 未注入短路零回归。

- 6 Task（T1~T6）全部完成，7 must_haves 逐条对照全绿，12 禁区逐条未触碰。
- 4 REQ（SCOPE-01~04）全部满足，证据见 §3。

---

## 1. 改动清单（逐文件）

### 1.1 后端核心基建（149 独占：`internal/services/operations/scope_entities.go`）

| 落点 | 改动 |
|---|---|
| 实体常量 `:39-55` | 新增编译期白名单 `scopeEntityWall` / `scopeEntityDoor` / `scopeEntityFloorPlanText` / `scopeEntityRoomPhoto` / `scopeEntityLocationAlias` |
| `entityScopeTables` `:48-64` | 增加 `ops_walls` / `ops_doors` / `ops_floor_plan_texts` / `ops_room_photos` / `sys_dept_location_alias` 5 表映射 |
| 新增 `entityFloorOrgExpr()` | 楼层归属链「有效机构」表达式，直接复用 `effectiveOrgExpr("f.org_id","b.org_id")` **单一真实源**，禁止内联 COALESCE/NULLIF |
| `entityExistsSub` `:81-` | 新增 wall/door/floorPlanText **楼层归属链**（`ops_floors f JOIN ops_buildings b`，`CAST(f.id AS TEXT) = <mainAlias>.floor_id`）与 roomPhoto **两级链**（`ops_server_rooms sr JOIN ops_buildings b`，`CAST(sr.id AS TEXT) = <mainAlias>.room_id`）；断链（软删 / 不存在）→ JOIN 不上 → EXISTS 0 行 → fail-closed 空集 |
| `applyEntityReadScope` `:139-` | wall/door/text 走 `wrapScopeExists(..., entityFloorOrgExpr(), "")`；roomPhoto 走 `wrapScopeExists(..., entityBuildingOrgExpr, "")`；locationAlias 复用 `applyAssetReadScope` 的 `dept_id ∈ 可见集` 直挂；`default` 维持 `1=0`；All / 未注入 `scopeIsAll` 短路原样返回。**`wrapScopeExists` 定义在 `scope_filter.go:231`（归 151），149 只调用不写** |
| `entityEffectiveOrgs` / `entitySubmittedOrgs` | 新增 5 类 case：wall/door/text 按 `floor_id` 查楼层链有效机构；roomPhoto 按 `room_id` 查两级链 `b.org_id`；locationAlias 本表 `dept_id` 直取。断链 → 空切片 → fail-closed |
| `filterEntityVisibleIDs` `:568-` | switch 增加 5 类 case（wall/door/text 用楼层链 + `entityFloorOrgExpr()`；roomPhoto 用两级链 + `entityBuildingOrgExpr`；locationAlias 复用 asset 的 `dept_id` 直挂分支）。**批量裁剪定案落在这里，未写 `scope_write.go::visibleIDsForEntityWrite`（归 150，仅复用语义）**；All 短路零查询 / FailClosed 空集 / `WHERE id IN ? AND EXISTS (?)` 一次性 DB 自筛 / 禁「先取全量再内存过滤」逐字沿用 `:555-567` 契约 |

### 1.2 5 个 service（149 独占）

| 文件 | 改动 |
|---|---|
| `wall_service.go` | `Create`/`Update` 提交侧 `checkEntitySubmittedScope`（floorID）、`Update`/`Delete` 改前侧 `checkEntityWriteScope`（id）→ 单条越界 `ErrScopeDenied`(403)；`GetByID` 挂 `entityReadScope` → 越界空集语义；`List` scopes 变参追加 `entityReadScope` → **只裁剪行，Count/Find 同源，响应形状不变**；`BatchDelete` 先 `filterEntityVisibleIDs` 裁剪再 `repo.BatchDelete` → 后端权威边界（不整批 403） |
| `door_service.go` | 同 wall（实体常量 `scopeEntityDoor`，表 `ops_doors`） |
| `floor_plan_text_service.go` | 同 wall（`scopeEntityFloorPlanText` / `ops_floor_plan_texts`） |
| `location_alias_service.go` | `List` 在 `Count` **之前** 挂 `applyEntityReadScope(..., scopeEntityLocationAlias, ...)` 保证 Count/Find 同源；`GetByID` 挂 `.Scopes(entityReadScope(...))`；`Create` 提交侧 `checkEntitySubmittedScope(deptID)`、`Update` 改前侧 + 提交侧（req 覆盖优先）、`Delete` 改前侧 → 越界 403 |
| `room_photo_service.go` | 新增私有守卫 `photoReadVisible` / `photoSubmittedWriteGuard` / `photoExistingWriteGuard`；**10 个方法逐方法登记并执行前置校验**：`UploadPhotos`（提交侧 roomID 403）、`ListByRoom`（读裁剪）、`SetPrimary`/`UpdateDescription`/`DeletePhoto`（改前侧 403）、`UpdateSort`（按可见集裁剪后继续）、`BatchDeletePhotos`（后端权威裁剪）、`GetPrimaryPhoto`（primary 前置读裁剪，主图/回退两张查询都挂）、`GetPhotoWithFile`（读空集）、`CountPhotos`（count 同源） |

### 1.3 `internal/api/router.go`（**仅 `:1174-1177` 一行补挂**）

```diff
 		// 机房照片管理
 		roomPhotos := ops.Group("/rooms/photos")
+		// DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
+		roomPhotos.Use(middleware.DataScopePermission(core))
 		{
 			operations.SetupRoomPhotoRouter(roomPhotos, core)
 		}
```

- `git diff -- internal/api/router.go` = **+2 行（1 注释 + 1 补挂）**，其余任何行（含 151 的
  ScopedDB 挂载区、`:796/:852/:907/:977/...` 所有既有 `DataScopePermission` 挂载）**零改动**。
- 补挂前后对照（`TestScope149_Router_RoomPhotosMiddlewareContract`）：补挂前该组无
  `DataScopePermission` → ctx 不含 `ScopeResult`（`scopeFromContext ok=false`，service 恒按
  All 兜底）；补挂后 → 注入 `ScopeResult`（`ok=true`、`All=false`、`DeptIDs` 非空）→ 5 类
  service 的 `entityReadScope` / `checkEntityWriteScope` 才真正消费可见集。
- 路由注册顺序不变；`node scripts/check-route-registry.mjs` = `OK 路由 713 条 / 写路由 315 条
  / deferred 138 条`，**零漂移**。

### 1.4 前端（149 独占）

| 文件 | 改动 |
|---|---|
| `useFloorPlanEditor.ts` | 新增 `VisibleBaseline`（`walls`/`doors`/`texts` 三套 `Set<string>`）用 `useRef` 承载，`loadFloorPlanData` 成功后固化「本次加载可见集」、`resetFloorPlan`/加载失败时清空；新增 `computeIncrementalDiff`（toCreate=新建占位、toUpdate=可见基线内、toDelete=基线−当前编辑态、ignored=越界 ID 一律忽略）；`saveWalls`/`saveDoors`/`saveTexts` 接收基线参数，只提交可见元素的 create/update/delete，**越界几何 ID 绝不拼进任何请求集**；`saveFloorPlan` 从 ref 取基线传入。**不重写 Three.js 编辑器、不改「整层拉取→内存渲染」读取模型、不改 wrapped API 调用方式（`@/lib/opsApi.ts`）、不改响应形状** |
| `__tests__/useFloorPlanEditor.test.tsx` | mock 扩展 `create`/`update`/`delete`/`batch`；新增 2 个 `Phase 149 保存侧可见基线增量 diff` 用例（越界 ID 不进任何请求集、可见元素 update/create 全提交、删除集只含可见基线、保存前后几何行数不变、成功提示语义不变） |

### 1.5 测试（就近 `*_test.go`，`Scope149` 前缀防并行重名）

| 文件 | 内容 |
|---|---|
| `scope_entities_scope149_test.go`（**新增**） | 5 类 × 六档（All / Dept 可见 / Dept 越界 / FailClosed / Self 无用户列 / 未注入）× 读操作矩阵；写侧单条 403 / 批量裁剪 + 越界行数；断链脏数据抽查（软删 floor / room / 越界 dept_id）；双方言静态断言（禁 `::`/`::uuid`/`ILIKE`/`isPostgres`、含 `CAST AS TEXT`、`effectiveOrgExpr` 单一真实源）；router 补挂静态契约；PG contract「skip 可观测」。13 PASS / 1 SKIP |
| `wall_service_test.go` / `door_service_test.go` / `room_photo_service_test.go`（**新增**） | 各 1 个 `TestScope149_*_ServiceWiring`：服务层接线（Create 越界 403 / List 裁剪 / BatchDelete 权威裁剪 / count 同源） |
| `floor_plan_text_crud_test.go` / `location_alias_service_test.go`（**追加**） | 各 1 个 `TestScope149_*_ServiceWiring`（+19 / +31 行） |

### 1.6 文档（本 phase 目录）

| 文件 | 内容 |
|---|---|
| `149-SUMMARY.md`（本文件） | 改动 / 4 REQ / 验证 / 方案 D 实证 / UAT 载体 / Known Gaps / 未提交 |
| `149-UAT-RECHECK.md`（**新建**） | D1/D2「确认未收口」结论作废 + 重跑要点 + 重跑窗口 / 验收人**待 QA 裁定**；见 §6 |

### 1.7 未触碰文件面核对表（逐条对照 PLAN §3 / ROADMAP `:119-148`）

| 归属 | 文件 | 149 状态 |
|---|---|---|
| 148 | `excel_export_config/service.go`（`service.go`）、`workstation_device_service.go`、`.golangci.yml` | **零改动** |
| 150 | `excel_service.go`、`workstation_service.go`、`scope_write.go`（`visibleIDsForEntityWrite` 仅复用语义）、`useBackfillDept.tsx`、`workstation_handler.go`、`excel_handler.go`、`pg_contract_test.go`、`132/133 backfill UAT` | **零改动** |
| 151 | `scope_filter.go`（含 `wrapScopeExists` / `scopeCacheKey`）、`internal/models/base/GORMRepository` | **零改动**（`wrapScopeExists` 只调用不写） |
| 145/146/147/W1 | `internal/utils/operlog/**`、`scripts/**`、CI、`internal/core/security/**`、`internal/services/asset/**` | **零改动** |
| 归档件 | `.planning/milestones/**`（含 `133-UAT.md`）、`.planning/notes/261108-*`（LOCKED 裁定原文）、79 条人工验证跟踪表 | **零改动 / 0 勾选** |
| barrel | `xingran-react-frontend/src/lib/api.ts` | **零改动** |

---

## 2. 4 REQ 满足情况（结论 + 证据）

| REQ | 结论 | 证据（测试名 / 命令 / 断言） |
|---|---|---|
| **SCOPE-01** 5 类读端点统一父资源可见性校验按可见集裁剪；walls/doors/texts 经楼层链、photos 经两级链、alias 按 `dept_id` 直挂；断链 fail-closed 空集 | ✅ 满足 | `TestScope149_Read_FiveEntities_OutOfScopeEmpty`（DeptA/B 各 1 行、FailClosed/Self 全 0、All/未注入 3 行零回归）、`TestScope149_Read_GetByID_Precheck`（GetByID / CountPhotos / GetPrimaryPhoto 前置）、`TestScope149_BrokenChain_SoftDeletedParents`（软删 floor/room → 4 类空集）、`TestScope149_Static_DualDialectAssertions`（CAST AS TEXT / 单一真实源 / 禁方言 token）；5 个 `TestScope149_*_ServiceWiring` 就近接线 |
| **SCOPE-02** `router.go:1174-1177` rooms/photos 组补挂 `DataScopePermission`；GetByID / count / primary / batch-delete 前置全走可见集校验 | ✅ 满足 | `git diff -- internal/api/router.go` = +2 行仅补挂；`TestScope149_Router_RoomPhotosMiddlewareContract`（未注释补挂行的逐行断言，变异 3 必红）；逐方法映射表：`ListByRoom`/`GetPhotoWithFile`/`SetPrimary`/`UpdateSort`/`UpdateDescription`/`DeletePhoto`/`BatchDeletePhotos`/`GetPrimaryPhoto`/`CountPhotos`/`UploadPhotos` 全部登记并执行前置校验（见 §1.2） |
| **SCOPE-03** 写端点越界 `floor_id`/`room_id` 单条 403 / 批量按可见集裁剪；**必须修保存侧**不得误删越界几何；响应形状不变 | ✅ 满足 | `TestScope149_Write_Single_OutOfScope403`（`errors.Is(err, ErrScopeDenied)`）、`TestScope149_Write_Batch_FilterEntityVisibleIDs`（`len(visible)==可见数`、越界行数正确、无 error、非整批 403）、`TestScope149_BatchDelete_ServiceIntegration`（保存前后几何行数：越界 + 断链 2 行不被误删）、前端 2 个增量 diff 用例（越界 ID 不进任何请求集）；后端权威裁剪 + 前端最小请求面双层都做；`PageResult` 的 `list`/`total`/`current`/`pageSize` 字段与排序逐字未改 |
| **SCOPE-04** 双方言 403 矩阵 + 断链脏数据抽查断言一致；`133-UAT.md:79-91` D1/D2 作废 + 重跑窗口；将来加缓存 key 必含 `scopeCacheKey(ctx)` | ✅ 满足 | SQLite 全矩阵逐字段断言 + PG contract `TestScope149_PGContract_SkipObservable`（`TEST_PG_DSN` unset → 可观测 skip，不伪造双跑）；静态断言禁 `::`/`ILIKE`/`isPostgres`、含 `CAST AS TEXT`、`effectiveOrgExpr` 调用点存在且无内联 COALESCE/NULLIF 复制；**UAT 作废登记载体按 Lead 收口裁决落 `149-UAT-RECHECK.md`**（见 §6）；缓存 key 契约在测试注释 + 本 SUMMARY 明确登记（本 phase 未加缓存、未写 `scope_filter.go`） |

### ROADMAP Phase 149 五条成功标准（`:81-86`）逐条对照

| # | 成功标准 | 落点 | 结论 |
|---|---|---|---|
| 1 | `:82` Dept 档 5 类列表只得到可见集内行；断链空集；GetByID/count/primary/batch-delete 前置全走可见集校验 | T1/T2/T4 | ✅ 矩阵 + 就近接线全绿 |
| 2 | `:83` 补挂后实测生效；Dept 档越界空/403；批量按可见集裁剪 | T1/T3/T4 | ✅ router diff +2 行、静态契约（变异 3 红绿）、403/裁剪矩阵全绿 |
| 3 | `:84` 非超管保存整层几何不误删越界元素（前后行数断言）；响应形状不变 | T3 | ✅ 后端 BatchDelete 裁剪 + 前端增量 diff 双层；前端几何行数断言 + 后端集成断言 |
| 4 | `:85` PG+SQLite 双方言矩阵 + 断链抽查一致；D1/D2 作废 + 重跑窗口；缓存 key 契约 | T4 | ✅ SQLite 实跑 + PG skip 可观测 + 静态双方言断言；`149-UAT-RECHECK.md` 登记；缓存 key 契约登记 |
| 5 | `:86` 零迁移（方案 D）：不新增 `org_id`(E)、不合并进楼层 JSONB(F)；`visibleIDsForEntityWrite` 仅复用不改；`scope_filter.go` 归 151 不触碰 | §4 裁决表 / T2/T3 | ✅ `git diff` 无 migration / 无新列 / 无 JSONB 重写；`scope_write.go`/`scope_filter.go`/`workstation_service.go`/`excel_service.go`/`useBackfillDept.tsx`/`.golangci.yml` 零改动 |

### must_haves 补充（PLAN §6 #6/#7）

- **#6 All 档零响应形状/数值回归**：`scopeIsAll` 短路零额外 DB 查询，All / 未注入档
  `List`/`total`/字段/排序逐字不变（矩阵断言 All=3 行、未注入=3 行）；`go build ./...` +
  三包 `go test` + `type-check` + vitest 全绿；反向变异 3 项红绿全通过。✅
- **#7 无整批 403 / 不引入 403 面扩张**：读越界恒空集（非 403）；批量裁剪后继续
  （非整批 403）；仅单条写越界显式 403。✅

---

## 3. 方案 D 与「回归面被高估」实证

### 3.1 方案 D 零迁移（对照根因报告 `:60-72`）

- **零迁移**：`git diff` 无 migration / 无新增 `org_id` 冗余列 / 无回填 / 无双写 /
  无 JSONB 合并（方案 E / F 均未采用）。5 类归属链**现成**：walls/doors/texts
  `floor_id → ops_floors.org_id（可空覆盖）/ building_id → ops_buildings.org_id`
  = `effectiveOrgExpr("f.org_id","b.org_id")`；photos `room_id → ops_server_rooms.building_id
  → ops_buildings.org_id`（与 ds05 裁定 1 的 roomDevice 两级链同款）；location-alias
  **已有 `DeptID`**（`sys_dept_location_alias.go:22`，裁定 3「5 类模型都不含归属字段」
  对它不成立）。
- **收口基建现成**：`scope_entities.go` 的 `entityExistsSub` + `applyEntityReadScope` +
  `wrapScopeExists`（`scope_filter.go:231`，归 151 只调用）+ `filterEntityVisibleIDs`
  = 现成范式，本期只是「扩展实体常量 / case」，**没有写第 5 套实现**。
- **响应形状 diff = 0**：`PageResult` 的 `list`/`total`/`current`/`pageSize`、排序、
  分页结构、字段名一律未改；只裁剪行。Count 与 Find 同源（`repo.List` 的 scopes 在
  Count 之前应用；location_alias 的 `applyEntityReadScope` 挂在 `Count` 之前）。
- **不引入整批 403**：读越界恒空集；批量一律裁剪后继续；仅单条写越界 403。
  与 v1.52「按可见集裁剪 + 显式 403」语义完全一致。
- **同页语义反而一致**：同页 `getFloorWorkstationsAll`（`workstation_service.go:517`
  `wrapScopeExists`）已收口，几何收口后「工位受 scope、几何不受」的不一致消失。

### 3.2 「回归面被高估」实证（LOCKED 裁定理由 = 成本判断，非技术不可行）

| 维度 | LOCKED 裁定假设 | Phase 149 实测 |
|---|---|---|
| 数据迁移 | 方案 E 需 4 表 + 回填 + 双写一致性 | **0 迁移**（方案 D） |
| 前端面 | 「Three.js 联动回归面大、重写编辑器」 | **仅 `useFloorPlanEditor.ts` 保存侧增量 diff**（`VisibleBaseline` + `computeIncrementalDiff` + 3 个 save 函数签名加 1 个参数）；不重写编辑器、不改读取模型、不改 API 调用方式 |
| 唯一必须改的边界 | 根因报告 `:66`「保存侧 `BatchOperation` 可见集裁剪」 | **两层都做**：后端 `BatchDelete`/`BatchOperation`/`BatchDeletePhotos` 的 `filterEntityVisibleIDs` 裁剪（权威边界）+ 前端增量 diff（最小请求面，消除 operlog「请求 N / 可见 M」噪声与静默丢弃语义歧义） |
| 改动文件数 / 行数 | 假设「大回归面」 | 后端 6 个 `.go`（1 基建 + 5 service）+ router.go **+2 行**；前端 1 个 hook + 1 个测试；**无 handler 改动、无 migration、无模型改动** |

结论：实测改动行数 / 文件数**远低于** LOCKED 裁定「Three.js 联动回归面大」的假设；
根因报告 `:16`「成本被高估、可根治」判定成立。

### 3.3 `visibleIDsForEntityWrite` 白名单发现（定案依据）

- `scope_write.go:292-333` 的 `visibleIDsForEntityWrite` switch 白名单**只有
  `building/floor/workstation/infoPoint` 四域**，`default` 返回「非法的写裁剪实体」；
  且该文件**归 Phase 150**。故 149 的 5 类实体**不能靠改它落地**。
- **定案**：批量裁剪落在 **149 独占的 `scope_entities.go::filterEntityVisibleIDs`**
  扩展实体常量 / case（PLAN §4 裁决表已定案），语义对齐 `visibleIDsForEntityWrite`
  （返回 `(可见 ID, 越界行数, error)`）但**未写 `scope_write.go`**。已核对零改动。

---

## 4. 验证结果（T5 全量命令 + 红绿）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `go build ./...` | ✅ **PASS**（并行中间态解除后全量编译通过；期间 148/150 并行文件 `excel_export_service.go` / `workstation_device_service.go` 曾出现未完成态，按 PLAN T5「记录待 Lead 集成期复核，不擅自改其它 phase 文件」处理，最终复跑全绿） |
| 2 | `go test -count=1 ./internal/services/operations/` | ✅ **ok 8.040s**（全量 operations 包，含 149 的 13 PASS + 1 SKIP + 5 个就近接线 + 存量全绿） |
| 3 | `go test -count=1 ./internal/api/v1/operations/` | ✅ **ok 0.377s** |
| 4 | `go test -count=1 ./pkg/middleware/` | ✅ **ok 0.658s** |
| 5 | `cd xingran-react-frontend && npm run type-check` | ✅ **PASS**（`tsc --noEmit -p tsconfig.app.json`） |
| 6 | `npx vitest run src/pages/operations/floors/__tests__/useFloorPlanEditor.test.tsx` | ✅ **10 tests PASS**（8 存量 + 2 个 Phase 149 增量 diff 用例） |
| 7 | `node scripts/check-route-registry.mjs` | ✅ **OK 路由 713 条 / 写路由 315 条 / deferred 138 条**（零漂移：本 phase 只补挂中间件、未增删路由） |
| 8 | `go test -count=1 -run '^TestScope149' ./internal/services/operations/` | ✅ **13 PASS / 1 SKIP**（`TestScope149_PGContract_SkipObservable` 可观测 skip） |

### 4.1 反向变异 3 项（红 → 还原 → 绿，每次只动一处、立即还原）

| # | 变异 | 必红证据 | 还原后复绿 |
|---|---|---|---|
| 1 | 去掉 `applyEntityReadScope` 中 wall/door/text case 的 `wrapScopeExists` 调用（改 `return query`） | `TestScope149_Read_FiveEntities_OutOfScopeEmpty` + `TestScope149_Wall_ServiceWiring` **FAIL**（Not equal ×12） | ✅ `cp` 还原后 `ok` |
| 2 | `BatchDelete` 的 `filterEntityVisibleIDs` 裁剪改回 `repo.BatchDelete(ctx, ids)` | `TestScope149_BatchDelete_ServiceIntegration` + `TestScope149_Wall_ServiceWiring` **FAIL**（「越界行仍被删 / Should be false」） | ✅ 还原后 `ok` |
| 3 | 注释掉 `router.go:1176` 的 `roomPhotos.Use(middleware.DataScopePermission(core))` | `TestScope149_Router_RoomPhotosMiddlewareContract` **FAIL**（逐行「未注释补挂行」断言 Should be true） | ✅ 还原后 `ok`，`grep` 确认 `:1176` 补挂行存在 |

> 变异 3 初次用 `Contains` 断言时因「注释后字符串仍存在」出现假绿 → **已强化为逐行
> 「trim 后不以 `//` 开头」的 activeMount 断言**，重做变异 3 必红、还原复绿。最终工作区全绿。

### 4.2 PG contract「skip 可观测」

- `TEST_PG_DSN` **unset**（DECISION-CI-PG、ROADMAP `:180`）：`TestScope149_PGContract_SkipObservable`
  显式 `t.Skip(...)`，可观测；**未建 `services: postgres`、未伪造 PG 双跑、未写假证据**。
- PG 版期望值以 fixture 常量（从 `CAST AS TEXT` / `EXISTS` 语义推导）固化进 SQLite 断言，
  保证双方言口径逐字一致。

---

## 5. UAT 载体说明（Lead 收口裁决 / checker 小问题 3）

**ROADMAP `:186`「不要动 `.planning/milestones/` 归档件」是对 roadmapper 的写作约束，
而 `:85` 是 149 的执行期成功标准（D1/D2 作废）。两全做法：**

- **不修改 `133-UAT.md` 原文**（归档件只读），改为在 149 phase 目录**新建
  `149-UAT-RECHECK.md`**，登记「D1/D2『确认未收口』结论作废 + 重跑要点 + 重跑窗口 /
  验收人**待 QA 裁定**」，并在本 SUMMARY 写明「因 ROADMAP:186 归档件只读，作废登记落在
  本 phase 目录，**等效满足 :85**」。
- 与 150 新建 `150-UAT-BACKFILL.md` 做法一致。
- **未改任何 `.planning/milestones/` 归档件、未勾选任何 UAT 完成项**；79 条人工验证
  跟踪表保持 0 勾选，LOCKED 裁定原文不动。
- 真正改动落点以 `:79-91`（D1/D2，不含 D3 `:92-94`）为准；`saveWalls` 实际 `:143` 起、
  `saveDoors` `:172` 起按代码实际定位（checker 小问题 2/5，不阻塞）。

---

## 6. Known Gaps（登记，非阻断）

1. **D1/D2 UAT 重跑窗口由 QA 裁定**：本 phase 只登记、不勾选完成（`149-UAT-RECHECK.md`）。
2. **PG 实机双跑仍为前置依赖**（DECISION-CI-PG）：`TEST_PG_DSN` unset，PG contract 只做
   「skip 可观测」；需新建 PG service + 迁移引导后方可升级为零真实双跑，**本期不伪造**。
3. **缓存 key `scopeCacheKey(ctx)` 仅为契约登记**：`scope_filter.go:53-74`（归 151），
   本 phase 未加缓存、未写该文件；若 5 类端点将来加缓存，key 必含该 scope 维度段。
4. **断链脏数据对非超管 fail-closed 不可见**：若生产存在「floor/room 软删或不存在」的
   历史几何/照片行，Dept 档按 fail-closed 看不到这些行（ds05 裁定 1 `:76` 已登记，
   **不为脏数据开旁路**）；All 档短路仍可见。
5. **① 的 A(150) + B(151) 不在本 phase 范围**：本 phase 只做 ②；**不宣称「统一架构已落地」**
   （B 未落地前不得宣称，ROADMAP `:103`）。
6. **HTTP 层 403 透传口径登记**：wall/door/floor-plan-text handler 走
   `response.HandleServiceError`（透传 `ErrScopeDenied` 的 HTTP 403）；room_photo /
   location_alias handler 现为 `response.Error(c, response.ErrServerError/ErrBadRequest, ...)`
   固定状态码形态。service 层已统一返回 `ErrScopeDenied`（`errors.Is` 可判定、满足
   T4 矩阵与「service 层 403 语义」），但这两个 handler 的 HTTP 状态码透传属
   **handler 文件面**（不在 149 独占清单，也不在明确禁区），本期按 PLAN「不改 handler」
   约束未改；建议 Lead 集成期核对是否需后续 phase 统一（不属 SCOPE-01~04 成功标准阻断项）。

---

## 7. 明确「未 git commit」

- 本 plan `commit:false`，**未执行任何 `git add` / `commit` / `stash` / `checkout` /
  `reset` / `cherry-pick`**；`git diff` / `git status` 仅用于只读核对文件面。
- 由 **Lead 在 W2 收口统一提交**；`internal/api/router.go` 的 149→151 先后顺序按
  ROADMAP `:145` 硬登记（**151 在 149 commit 之后基于最新文件改**）。
- 三 executor 并行同工作区期间曾观察到 148/150 的未完成中间态（`excel_export_service.go`
  的 `RightCast`、`workstation_device_service.go` 的 unused import、`scope_write.go` 的
  中间态测试），本 executor **未擅自修改任何其它 phase 文件**，最终 `go build ./...` +
  三包 `go test` 全绿，交回 Lead 集成。

---

> **完成条件**：T1~T6 全部完成，T5 全绿、T6 SUMMARY 写明「未 git commit」。
> `gaps_found`：无阻断项（§6 均为登记性 Known Gaps）。返回 `## PHASE 149 COMPLETE`。
