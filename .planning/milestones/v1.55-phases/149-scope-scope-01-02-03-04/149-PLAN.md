---
phase: 149
plan: 01
wave: 2
depends_on: []
files_modified:
  - internal/services/operations/wall_service.go
  - internal/services/operations/door_service.go
  - internal/services/operations/floor_plan_text_service.go
  - internal/services/operations/room_photo_service.go
  - internal/services/operations/location_alias_service.go
  - internal/services/operations/scope_entities.go
  - internal/api/router.go # 仅 :1174-1177 一行补挂，149 先改→151 后改
  - xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts
  - xingran-react-frontend/src/pages/operations/floors/__tests__/useFloorPlanEditor.test.tsx
  - .planning/milestones/v1.52-phases/133-scope-final/133-UAT.md # 仅 D1/D2 段 :77-91
  - internal/services/operations/wall_service_test.go
  - internal/services/operations/door_service_test.go
  - internal/services/operations/floor_plan_text_crud_test.go
  - internal/services/operations/room_photo_service_test.go
  - internal/services/operations/location_alias_service_test.go
  - internal/services/operations/scope_entities_scope149_test.go # 新增 403 矩阵 + 断链 + 双方言
  - .planning/phases/149-scope-scope-01-02-03-04/149-SUMMARY.md
autonomous: true
commit: false
---

# Phase 149: SCOPE 豁免面收口（SCOPE-01/02/03/04）— 可直接执行计划

> **执行约定**：本 plan 在 W2 与 148 / 150 并行，三个 executor 写同一工作区。
> `commit:false` —— **禁止任何 git 写操作**（`add` / `commit` / `stash` / `checkout` / `reset` / `cherry-pick`），由 Lead 在本波收口统一提交。
> 所有写操作严格限定在 `files_modified` 的独占文件面内；**唯一的跨 phase 共文件 `internal/api/router.go` 只允许改 `:1174-1177` 一行**（149 先改 → 151 后改，硬登记 ROADMAP `:145`）。
> 并行许可凭证 = `ROADMAP.md:119-137` 函数级归属表；任一格实测交集非空 → 整组降级串行，不得带冲突并行。

---

## 1. 必读顺序（开工前按序读完，均为只读）

| # | 文件 | 读什么 / 为什么 |
|---|------|-----------------|
| 1 | `.planning/STATE.md` | v1.55 active、W2 `148 ∥ 149 ∥ 150` 并行拓扑、79 条跟踪表保持 0 勾选、v1.52「按可见集裁剪 + 显式 403」语义基线 `:62-89` |
| 2 | `.planning/ROADMAP.md` | **Phase 149 详情 `:76-86`（5 条成功标准）**、并行冲突面/函数级归属表 `:113-148`、Constraints `:150-161`、禁区 `:173-181`、Known Gaps `:188-200`（尤其 `:200` UAT 重跑窗口） |
| 3 | `.planning/REQUIREMENTS.md` | **SCOPE-01~04 全文 `:30-36`** + 5 项裁定 `:7-17` + Constraints `:98-106` + Traceability `:112-115` |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` | **§② 根因 `:58-74`**（方案 D/E/F 对比表、根治定义、「回归面被高估」判定、唯一必须改的保存侧边界）、§0 结论速览 `:13-26`、共性架构结论 `:153-162`、未决问题段 |
| 5 | `.planning/notes/261107-v152-ds05-scope-decision.md` | **裁定 3 `:39-46`**（location-alias 显式豁免原文）、裁定 1 `:16-27`（roomDevice 两级链范本）、豁免清单终态 `:50-61`、fail-closed 总原则 `:12` |
| 6 | `.planning/milestones/v1.52-phases/133-scope-final/133-UAT.md:77-94` | **D1/D2「确认未收口」原文**（本 phase 只作废 D1/D2 段，不动 D3 与 150 的 132/133 backfill UAT 另一段） |
| 7 | `CLAUDE.md` / `AGENTS.md` | Handler-Service、`response.Success()/Error()`、operlog 写操作约定（success path 末尾、`response.Success` 之前 `operlog.Record`）、SQLite+PG 双方言（`CAST AS TEXT`、禁 `::uuid`/`ILIKE`）、前端 wrapped API（`@/lib/opsApi.ts`）、Go 单测就近 `*_test.go`、前端 `*.test.ts(x)` |
| 8 | 收口基建（只读范本） | `internal/services/operations/scope_entities.go:1-33`（契约注释）、`:81-112` `entityExistsSub`、`:139-185` `applyEntityReadScope`/`entityReadScope`、`:198-271` `entityEffectiveOrgs`、`:284-330` `entitySubmittedOrgs`、`:377-430` `checkEntityWriteOrgs`/`checkEntityWriteScope`/`checkEntitySubmittedScope`、`:555-659` `filterEntityVisibleIDs` |
| 9 | 只读依赖（**禁止写**） | `internal/services/operations/scope_filter.go:225-237` `wrapScopeExists`（**归 151**）、`:53-74` `scopeCacheKey`（**归 151**）、`:94-...` `orgScopeClause`；`internal/services/operations/scope_write.go:282-333` `visibleIDsForEntityWrite`（**归 150**，仅复用语义不写）；`internal/services/operations/dept_filter.go:17` `effectiveOrgExpr` 单一真实源；`internal/services/operations/workstation_service.go:232-236` `orgScopeClause` 直挂范式 |
| 10 | 代码现状 | `internal/api/router.go:969-981`（location-alias 读组已挂）、`:1174-1177`（rooms/photos 缺挂）、`:1269-1287`（walls 已挂）、`:1307-1323`（doors 已挂）、`:1343-1355`（texts 已挂）；5 个 `*_service.go` 的 `List`/`GetByID`/`BatchDelete`；`useFloorPlanEditor.ts:52-73`（整层拉取）、`:156-186`+`:200-279`（保存侧）、`__tests__/useFloorPlanEditor.test.tsx` |
| 11 | 既有 403 矩阵测试范式 | `internal/services/operations/scope_entities_403_test.go`、`scope_read_403_test.go`、`scope_write_403_test.go`、`v152_diff_coverage_test.go`（复用 fixture 装配与 `scopeFromContext` 注入手法） |

> 读完如发现根因报告/裁定与代码现状冲突 → 立即停手，返回 `## PLANNING BLOCKED` 并列出具体冲突。
> **本 plan 起草时已逐项核对**：5 个 service 的 `List` 不消费 `ScopeResult`、`filterScope` 只处理 `floor_id`/`type`、`scope_entities.go` 全部 helper 行号、`visibleIDsForEntityWrite` 的 4 域白名单 + `default` 报错、rooms/photos 组缺挂、4 个已挂组、`location-alias` 的 `DeptID`（`internal/models/sys_dept_location_alias.go:22`）、`useFloorPlanEditor.ts` 保存侧逐条 `create/update` 且不发全量 ID 删除集、5 个模型表名与外键列 —— **无冲突**。

---

## 2. 目标

把 LOCKED 残余 ②「5 项显式豁免面」**从根源**修掉，使「5 类端点按可见集裁剪 + 写端点显式 403」由靠约定升级为靠机制，且**零迁移、响应形状不变、不引入整批 403**：

1. **SCOPE-02**：`router.go:1174-1177` 的 rooms/photos 组补挂 `middleware.DataScopePermission(core)`（修「比豁免更基础的接线遗漏」）；`GetByID`/count/primary/batch-delete 前置全走可见集校验。
2. **SCOPE-01**：5 个 service 的 `List`/`GetByID` 挂 `entityExistsSub` + `wrapScopeExists` 按可见集裁剪；walls/doors/texts 经楼层归属链、photos 经 `room_id → serverRoom → building` 两级链、location-alias 按已有 `dept_id ∈ 可见集` 直挂；断链 fail-closed 空集；All/未注入短路零回归。
3. **SCOPE-03**：写端点（Create/Update/Delete/Batch）越界 `floor_id`/`room_id` 单条显式 403、批量按可见集裁剪；**保存侧必改**（Three.js 编辑器整体覆写不得让非超管误删看不到几何）；响应形状不变（只裁剪行，不改 `list`/`total`/字段）。
4. **SCOPE-04**：双方言（PG + SQLite）403 矩阵 + 断链脏数据抽查断言一致；`133-UAT.md:79-91` D1/D2 旧结论作废 + 重跑窗口登记；将来加缓存 key 必含 `scopeCacheKey(ctx)`。
5. **SUMMARY**：4 REQ 满足情况、方案 D 与「回归面被高估」的实证、验证结果、Known Gaps、明确「未 git commit」。

**「根治」判定线**（按用户「不打补丁」标准）：读端点真的消费 `ScopeResult`、rooms/photos 真的注入、写端点真的 403/裁剪、保存侧真的不误删、双方言矩阵一致；**仅挂中间件不接线 / 只改前端不裁后端 / 加 `org_id`+回填（方案 E）/ 几何合并进楼层 JSONB（方案 F）/ 整批 403 一律不算根治**。

---

## 3. 已确认事实（直接作为执行输入，禁止重新调研）

### 5 类端点现状
- **已挂 `DataScopePermission` 但 service 不消费**：walls `router.go:1269-1287`（`:1283`）、doors `:1307-1323`（`:1319`）、floor-plan-texts `:1343-1355`（`:1351`）、location-alias `:969-981`（`:977`）。5 个 service 的 `List` 完全不消费 ctx 里的 `ScopeResult`：`wall_service.go:84-90`、`door_service.go:91-97`、`floor_plan_text_service.go:83-89`、`location_alias_service.go:96-128` 直接 `repo.List` / `query.Count+Find`，`filterScope` 只处理 `floor_id`/`type` 等业务筛选。
- **连中间件都没挂**：rooms/photos `router.go:1174-1177`，仅 ops 父组 JWT/OperLog，无 `DataScopePermission` → 比「豁免」更基础的接线遗漏。
- **归属链现成、零迁移**：walls/doors/texts `floor_id → ops_floors.org_id（可空覆盖）/ building_id → ops_buildings.org_id`，正是 `effectiveOrgExpr("f.org_id","b.org_id")`；photos `room_id → ops_server_rooms.building_id → ops_buildings.org_id`（与 ds05 裁定 1 的 roomDevice 两级链同款，`scope_entities.go:91-97` 已是现成骨架）。
- **location-alias 已有 `DeptID`**：`internal/models/sys_dept_location_alias.go:22`（`varchar(64)`，指向 `sys_dept.id`），裁定 3「5 类模型都不含归属字段」对它**不成立**；按 `dept_id ∈ 可见集` 直挂即可，`workstation_service.go:232-236` 的 `orgScopeClause` 是现成范式。
- **模型/表名（已核对）**：`ops_walls`（`wall.go:47`）、`ops_doors`（`door.go:62`）、`ops_floor_plan_texts`（`floor_plan_text.go:58`）、`ops_room_photos`（`room_photo.go:19`，`room_photo.go:10` `RoomID`）、`sys_dept_location_alias`（`sys_dept_location_alias.go:29`）。前三类外键列 `floor_id`，photos 外键列 `room_id`。

### 收口基建现成（零迁移）
- `scope_entities.go:81-112` `entityExistsSub`（EXISTS 子查询骨架，链断天然 0 行 = fail-closed）、`:139-185` `applyEntityReadScope` / `entityReadScope`（**149 独占文件，可按需扩展**）、`:231` `wrapScopeExists`（**在 `scope_filter.go`，归 151，149 只调用不写、不把唯一收口入口改成 ScopedDB**）。
- `scope_write.go:282-333` `visibleIDsForEntityWrite`（**归 150，仅复用语义不写**）：其 `switch` 白名单只有 `building/floor/workstation/infoPoint` 四域，`default` 返回「非法的写裁剪实体」→ **149 的 5 类实体不能靠改它落地**。
- **本 phase 批量裁剪落点（定案）**：`scope_entities.go:568-659` `filterEntityVisibleIDs` 已是「与 132 `filterVisibleIDs` 逐字同源、不放宽 132 白名单」的现成范式（`entityScopeTables` 表映射 + `orgScopeClause` 三态 + `EXISTS (?)` + All 短路零查询）。**T3 扩展 `entityScopeTables`/`entityExistsSub`/`entityEffectiveOrgs`/`entitySubmittedOrgs`/`filterEntityVisibleIDs` 到 5 类新实体常量，全部写在 149 独占的 `scope_entities.go`**，语义对齐 `visibleIDsForEntityWrite` 但不写 `scope_write.go`。
- `scope_filter.go:53-74` `scopeCacheKey`（归 151，仅登记「将来加缓存必含」）；`serverRoom`/`dedicatedLine`/`roomDevice` 已按楼宇链收口、`workstation-device` 已按工位可见集收口（同族实现，避免第 5 套）。
- **方案 D = 零迁移、响应形状不变、不引入整批 403**，与 v1.52「按可见集裁剪 + 显式 403」语义完全一致；方案 E（补 `org_id`+回填+双写）、F（合并进楼层 JSONB 重写前端）不推荐（根因报告 `:68-72`）。

### Three.js 编辑器边界（唯一必须改的前端/后端边界）
- `useFloorPlanEditor.ts:52-73` 按 `floorId` 一次拉整层（`pageSize:1000`）、`:156-186`/`:200-226` 保存时**对 `data.walls`/`data.doors`/`data.texts` 逐条 `create`/`update`**（不发送全量 ID 删除集），`:240-268` `saveFloorPlan` 并行三 save + `saveWorkstations`；同页 `getFloorWorkstationsAll`（`workstation_service.go:507-525`，`:517` `wrapScopeExists`）**已收口** → 收口后语义反而一致。
- **误删风险实质**：后端 `BatchOperation`（`wall_handler.go:165-181` → `wallService.BatchDelete` → `repo.BatchDelete`）按请求 ID 集删除；若前端/调用方以「整层拉取 → 整体覆写」口径提交删除集，非超管看不到的几何 ID 不在其可见集却会被删除。故**后端批量删除必须先裁剪**（T3），前端保存侧不得构造越界删除集（T3 定案）。

### 并行冲突面（硬约束，ROADMAP `:119-148`）
- **149 独占**：5 个 `*_service.go`（wall/door/floor_plan_text/room_photo/location_alias）、`scope_entities.go`、`internal/api/router.go`（**仅 `:1174-1177` 补挂**）、`useFloorPlanEditor.ts`（含就近测试）、`133-UAT.md`（**仅 D1/D2 段**）、5 个 service 的 `*_test.go` + 新增 403 矩阵测试。
- **149 不得碰**（逐文件）：`excel_export_config.go` / `excel_export_service.go` / `workstation_device_service.go` / `.golangci.yml`（148）；`workstation_service.go` / `excel_service.go` / `scope_write.go`（`visibleIDsForEntityWrite` 仅复用不写）/ `useBackfillDept.tsx` / `132/133 backfill UAT` 另一段（150）；`scope_filter.go`（含 `wrapScopeExists`/`scopeCacheKey`）/ `internal/models/base/GORMRepository`（151）；`internal/utils/operlog/**` / `scripts/**` / `.github/workflows/ci.yml`（145）；`internal/core/security/**`（146）；`internal/services/asset/**`（147）；`xingran-react-frontend/src/lib/api.ts` barrel。
- **用户裁定相关**：① 的 A（150）+B（151）均需完成；本 phase 只做 ②。79 条跟踪表 0 勾选、LOCKED 裁定原文不动。

---

## 4. Lead 裁决表（执行期不得重新裁定）

| 裁决项 | 定案 | 依据 / 理由 |
|---|---|---|
| **总体方案** | **方案 D 零迁移**：5 类读端点统一父资源可见性校验按可见集裁剪，写端点 403/裁剪，响应形状不变、不引入整批 403 | 根因报告 `:66-72`、REQUIREMENTS `:36`、ROADMAP `:80/:86`；E 引入新一致性面、F 回归面与裁定担忧吻合，均不推荐 |
| **location-alias 归属** | **用已有 `dept_id ∈ 可见集` 直挂**（不新增列、不按楼宇链回退） | `sys_dept_location_alias.go:22` 已有 `DeptID`；裁定 3「无归属字段」对它不成立；`workstation_service.go:232-236` `orgScopeClause` 现成范式 |
| **rooms/photos 中间件** | `router.go:1174-1177` 组**补挂** `middleware.DataScopePermission(core)` 一行；其余 4 组已挂保持不动 | 比「豁免」更基础的接线遗漏（根因报告 `:24/:63`、SCOPE-02）；不碰 151 的 ScopedDB 挂载区 |
| **批量裁剪落点（关键）** | 扩展 **149 独占的 `scope_entities.go`**（`entityExistsSub`/`entityEffectiveOrgs`/`entitySubmittedOrgs`/`entityScopeTables`/`filterEntityVisibleIDs`）；**`scope_write.go::visibleIDsForEntityWrite` 仅复用语义、不写该文件** | `visibleIDsForEntityWrite`（`:292-333`）白名单只有 4 域、`default` 报错且文件归 150；`filterEntityVisibleIDs`（`:568`）本就是「不改 132 白名单」的同源范式 |
| **`wrapScopeExists` 归属** | **只调用不写**（在 `scope_filter.go:231`，归 151）；149 不把唯一收口入口改成 ScopedDB、不动 `scope_filter.go` | ROADMAP `:127/:146` 全局唯一归属硬登记 |
| **保存侧（二选一定案）** | **定案 = 后端批量裁剪为主 + 前端保存增量 diff 为辅，二者都做**：① 后端 `BatchDelete`/`BatchOperation` 一律先经 `filterEntityVisibleIDs` 裁剪（不可绕过）；② 前端 `useFloorPlanEditor.ts` 保存侧以「本次加载可见集」为基线做增量 diff，**只提交可见元素的 `create/update/delete`，绝不提交自己看不到的 ID 删除集** | 理由：只做后端裁剪，越界 ID 仍可能被前端拼进请求（虽被静默丢弃，但前端语义未对齐且 operlog 审计链出现「请求 N / 可见 M」噪声）；只做前端 diff，绕过前端直接调 API 仍可误删（后端才是权威边界）。**两层都要，后端为权威、前端为最小请求面**；不重写编辑器、不改「整层拉取 → 内存渲染」读取模型、不改响应形状 |
| **`effectiveOrgExpr` / 双方言** | 有效机构表达式**只走 `dept_filter.go:17 effectiveOrgExpr("f.org_id","b.org_id")` 单一真实源**，禁止内联复制 `COALESCE/NULLIF`；所有 uuid/varchar 比较 `CAST(... AS TEXT)`，禁 `::uuid`/`ILIKE`/裸 `ON CONFLICT` | v1.52 硬约束 `:77`、REQUIREMENTS `:100`、ROADMAP `:154`；`scope_entities.go:65-67` 已按此封装 `entityWorkstationOrgExpr()` |
| **断链 / All 短路** | 断链（floor/room 软删或不存在）→ EXISTS 0 行 → **fail-closed 空集**（读）/ 403（写）；All / 0 / unset（含未注入 ctx）**完全短路、零 DB 查询、零回归** | ds05 裁定 1 `:23`、根因报告 `:79-80`、`scope_entities.go:22/:138/:406-410` 契约 |
| **不碰 150/151 文件面** | 严格按 ROADMAP `:128-135/:141-148` 函数级归属表；`router.go` 只 `:1174-1177`；`scope_filter.go` + `base/GORMRepository` 一行不动 | W2 并行许可凭证；任一交集非空整组降级串行 |
| **UAT 范围** | **只改 `133-UAT.md` 的 D1/D2 段（`:79-91`）**：作废「确认未收口」结论 + 登记重跑窗口；**不动 D3、不动 150 的 132/133 backfill UAT 另一段** | ROADMAP `:130/:200`、REQUIREMENTS `:35` |
| **git / 进程 / 数据** | 禁所有 git 写操作；不启动长驻进程；不写生产数据；不动 79 条跟踪表与 LOCKED 裁定原文 | 三 executor 并行同工作区，Lead 统一提交；`commit:false` |

---

## 5. 任务（XML）

<task id="1">
  <title>T1 · SCOPE-02：rooms/photos 组补挂 DataScopePermission + 前置可见集校验清单</title>
  <files>
    internal/api/router.go（**仅 :1174-1177**）
    internal/services/operations/room_photo_service.go
    internal/services/operations/scope_entities.go
  </files>
  <action>
    1. **一行补挂（硬约束）**：在 `internal/api/router.go:1174-1177` 的 `roomPhotos := ops.Group("/rooms/photos")` 之后、`operations.SetupRoomPhotoRouter(roomPhotos, core)` 之前，与兄弟读组逐字同款插入：
       ```
       // DS-02: 权限校验之后、handler 之前注入 data_scope/user_id（含 request context）。
       roomPhotos.Use(middleware.DataScopePermission(core))
       ```
       - **只改这一行 + 必要注释**；`:796/:852/:907/:977/:1004/:1050/:1102/:1143/:1193/:1208/:1234/:1249/:1283/:1292/:1319/:1328/:1351/:1360` 等所有既有 `DataScopePermission(core)` 挂载**保持原样**；**不得触碰 151 的 ScopedDB 挂载区**（本 phase 不存在该区，任何「顺手加 ScopedDB」都越界）。
       - 路由注册顺序不变（gin `Group()` 继承父组中间件链：这是读组，不是写组兄弟组；rooms/photos 无独立严格写组，本 phase 不新建组）。
    2. **注入生效断言**（写进 T4 测试）：构造 Dept 档 `ScopeResult` 注入 ctx，经 `router_test.go` 既有装配手法或新建就近断言，确认 `/rooms/photos` handler 取到的 ctx 含 `datascope.ScopeResult`（`scopeFromContext(ctx)` 的 `ok=true`、`All=false`、`DeptIDs` 非空）；补挂前 `ok=false`、补挂后 `ok=true` 的对照写进 SUMMARY 实证。
    3. **前置校验清单落地**：`room_photo_service.go` 的 `ListByRoom`（`:104`）、`GetPhotoWithFile`（`:262`）、`SetPrimary`（`:...`）、`UpdateSort`、`UpdateDescription`、`DeletePhoto`（`:186`）、`BatchDeletePhotos`（`:224`）、`GetPrimaryPhoto`（`:239`）、`CountPhotos`（`:280`）、`UploadPhotos`（`:43`）**逐方法登记并执行可见集前置校验**（具体挂接在 T2/T3 完成，本任务先列出「GetByID / count / primary / batch-delete 前置全部走可见集校验」的逐方法映射表写入 SUMMARY）。**不得遗漏 `CountPhotos`/`GetPrimaryPhoto`**（ROADMAP `:82` 点名 count / primary）。
    4. 本任务**不改** `scope_write.go` / `scope_filter.go` / `workstation_service.go`；`router.go` 改动后 `go build ./...` 必须通过（T5）。
  </action>
</task>

<task id="2">
  <title>T2 · SCOPE-01：5 个 service 的 List/GetByID 挂 entityExistsSub + wrapScopeExists（按可见集裁剪）</title>
  <files>
    internal/services/operations/wall_service.go
    internal/services/operations/door_service.go
    internal/services/operations/floor_plan_text_service.go
    internal/services/operations/room_photo_service.go
    internal/services/operations/location_alias_service.go
    internal/services/operations/scope_entities.go
  </files>
  <action>
    1. **在 149 独占的 `scope_entities.go` 扩展实体常量与归属链**（不动 `scope_filter.go`）：
       - 新增编译期白名单常量：`scopeEntityWall="wall"` / `scopeEntityDoor="door"` / `scopeEntityFloorPlanText="floorPlanText"` / `scopeEntityRoomPhoto="roomPhoto"` / `scopeEntityLocationAlias="locationAlias"`；`entityScopeTables` 增加 `ops_walls` / `ops_doors` / `ops_floor_plan_texts` / `ops_room_photos` / `sys_dept_location_alias` 映射。
       - **`entityExistsSub`（`:81`）新增 case**：
         - wall/door/floorPlanText（**楼层归属链**）：`sub.Table("ops_floors f").Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").Where("CAST(f.id AS TEXT) = "+mainAlias+".floor_id").Where("f.deleted_at IS NULL AND b.deleted_at IS NULL").Select("1")`；有效机构表达式 = **`effectiveOrgExpr("f.org_id","b.org_id")`**（**单一真实源，禁止内联 COALESCE/NULLIF**），复用 `entityWorkstationOrgExpr()` 同款封装为 `entityFloorOrgExpr()`。
         - roomPhoto（**room→serverRoom→building 两级链**）：复用 `scopeEntityRoomDevice` 同款骨架，主表外键列改为 `<mainAlias>.room_id`（`ops_room_photos.room_id`），有效机构 = `entityBuildingOrgExpr`（`b.org_id`）；与 ds05 裁定 1 的 roomDevice 两级链逐字同款。
         - locationAlias（**dept_id 直挂**）：不走 EXISTS，按 `applyAssetReadScope`（`:129-131`，`applyReadScope(query, ctx, "dept_id", "")`）同款在 `applyEntityReadScope` 增加 `case scopeEntityLocationAlias` 直挂分支。
       - **`applyEntityReadScope`（`:139`）**：wall/door/floorPlanText/roomPhoto 走 `wrapScopeExists(query, ctx, sub, <orgExpr>, "")`（**调用 `scope_filter.go:231`，归 151，只调用不写**）；locationAlias 走 `dept_id` 直挂；`default` 维持 `query.Where("1=0")`；All / 未注入 `scopeIsAll` 短路原样返回（`:144-147`）保持。
       - `entityReadScope`（`:181`）返回 `base.Scope`，供 `repo.List` / `repo.GetByID` 的 scopes 变参使用（wall/door/text 已是 `base.GORMRepository` 管道：`wall_service.go:84-90`）。
    2. **5 个 service 的 `List` 起链处挂可见集**（**响应形状不变，只裁剪行**）：
       - wall `wall_service.go:84-90`：在 `s.repo.List(ctx, pageParams, s.filterScope(req), sortScope)` 的 scopes 变参**追加** `entityReadScope(ctx, scopeEntityWall, "ops_walls")`；`filterScope`（业务 `floor_id`/`type` 筛选）保持不变、顺序不变。
       - door `door_service.go:91-97`、floor_plan_text `floor_plan_text_service.go:83-89` 同款（mainAlias 用各自表名）。
       - location_alias `location_alias_service.go:96-128`：在 `query := s.db.WithContext(ctx).Model(&models.SysDeptLocationAlias{}).Where("...deleted_at IS NULL")` 之后、`Count(&total)` **之前**对 `query` 追加 `entityReadScope(ctx, scopeEntityLocationAlias, "sys_dept_location_alias")` 作用域（用 `query.Scopes(entityReadScope(...))` 或等价 `applyEntityReadScope(query, ctx, ...)`），保证 **`Count` 与 `Find` 同源**（不出现「total 全库、list 裁剪」双口径）；`aliasListJoinSelect`/`aliasListJoinClause`/`Order`/`Offset`/`Limit` 与 `PageResult{List,Total,Current,PageSize}` 字段一律不动。
       - room_photo `room_photo_service.go:104 ListByRoom`：在 `s.db.WithContext(ctx).Model(...).Where("room_id = ?", roomID)` 之后追加 `entityReadScope(ctx, scopeEntityRoomPhoto, "ops_room_photos")`；`CountPhotos`（`:280`）、`GetPrimaryPhoto`（`:239`）同样挂可见集（count/primary 前置校验）。
    3. **GetByID / count / primary / batch-delete 前置全走可见集校验**：
       - wall/door/text 的 `GetByID`（`wall_service.go:57`、`door_service.go:64`、`floor_plan_text_service.go:64`）在 `s.repo.GetByID(ctx, id)` **之前**调 `checkEntityWriteScope` 同款读侧判定（或 `entityReadScope` 变参），越界返回「空/未找到」语义（**读端点空集不报错**，与 v1.52 一致）；断链 → 空。
       - location_alias `GetByID`（`:132`）、room_photo `GetPhotoWithFile`（`:262`）/ `SetPrimary` / `UpdateSort` / `UpdateDescription` / `DeletePhoto`（`:186`）/ `BatchDeletePhotos`（`:224`）/ `UploadPhotos`（`:43`）/ `GetPrimaryPhoto`（`:239`）/ `CountPhotos`（`:280`）逐方法前置（读侧空集、写侧见 T3 的 403）。
       - **断链 fail-closed**：floor/room 软删或不存在 → `entityExistsSub` 的 `JOIN ... deleted_at IS NULL` 不匹配 → `EXISTS=false` → 空集；`GetByID` 不得回退 All、不得猜测。
    4. **双方言**：所有新增 SQL 的 uuid/varchar 比较一律 `CAST(... AS TEXT)`（与 `scope_entities.go:21/:87/:94-96` 逐字同款），**禁止 `::uuid` / `ILIKE` / `isPostgres` 方言分支**；有效机构表达式只走 `effectiveOrgExpr` / `entityBuildingOrgExpr`，禁止内联复制。
    5. **零回归**：All 档 / ctx 未注入 `ScopeResult` → `scopeIsAll` 短路、零额外 DB 查询、超管行为逐字不变；`list` / `total` / 字段名 / 分页结构一律不改；**不引入整批 403**（读端点越界恒空集，不是 403）。
    6. 不得改 `scope_filter.go`、不得把 `wrapScopeExists` 复制进本 phase 文件、不得改 `base/GORMRepository`（归 151）、不得新增 `org_id` 列 / migration。
  </action>
</task>

<task id="3">
  <title>T3 · SCOPE-03：写端点 403 / 批量裁剪 + 保存侧不误删（后端权威 + 前端增量 diff）</title>
  <files>
    internal/services/operations/wall_service.go
    internal/services/operations/door_service.go
    internal/services/operations/floor_plan_text_service.go
    internal/services/operations/room_photo_service.go
    internal/services/operations/location_alias_service.go
    internal/services/operations/scope_entities.go
    xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts
    xingran-react-frontend/src/pages/operations/floors/__tests__/useFloorPlanEditor.test.tsx
  </files>
  <action>
    1. **扩展 `scope_entities.go` 写侧 helper 到 5 类实体**（149 独占）：
       - `entityEffectiveOrgs`（`:198`）：新增 wall/door/floorPlanText 的「改前」解析（`ops_walls/doors/floor_plan_texts` → `ops_floors f` → `ops_buildings b`，取 `CAST(effectiveOrgExpr("f.org_id","b.org_id") AS TEXT)`，断链/软删 → 空切片）；roomPhoto（`ops_room_photos` → `ops_server_rooms sr` → `ops_buildings b`，取 `b.org_id`）；locationAlias（本表 `dept_id` 直取）。
       - `entitySubmittedOrgs`（`:284`）：新增「提交侧」解析 —— wall/door/floorPlanText 的 `fkIDs[0]=floorID` → 查 `ops_floors f` JOIN `ops_buildings b` 取有效机构；roomPhoto `fkIDs[0]=roomID` → 查 `ops_server_rooms sr` JOIN `ops_buildings b` 取 `b.org_id`；locationAlias `fkIDs[0]=deptID` 直挂。
       - `filterEntityVisibleIDs`（`:568`）：`entityScopeTables` 已含 5 表；在 switch 增加 5 类 case —— wall/door/floorPlanText 用楼层链 `sub = entityExistsSub(db, entity, table)` + `orgExpr = entityFloorOrgExpr()`；roomPhoto 用两级链 + `entityBuildingOrgExpr`；locationAlias 复用 `scopeEntityAsset` 的 `dept_id` 直挂分支。**All 短路零查询 / FailClosed 空集 / `WHERE id IN ? AND EXISTS (?)` 一次性 DB 自筛 / 禁「先取全量再内存过滤」**逐字沿用 `:555-567` 契约。
       - **语义对齐 `visibleIDsForEntityWrite`（`scope_write.go:292`，归 150，仅复用不写）**：返回 `(可见 ID, 越界行数, error)`，越界行数供 operlog 审计链用（不进 HTTP 响应）；**不改 `scope_write.go`**。
    2. **单条写端点越界显式 403**：
       - wall/door/text 的 `Create`（`wall_service.go:39` 等，先 `validator.ValidateFloor(ctx, ...)`）：在校验之后、`s.repo.Create/Update` 之前调 `checkEntitySubmittedScope(ctx, s.db, <entity>, <floorID>)`；`Delete`（`wall_service.go:53` 等）调 `checkEntityWriteScope(ctx, s.db, <entity>, id)`；越界返回 `ErrScopeDenied`（`scope_write.go:42`，handler 经 `response.HandleServiceError` 透传 **HTTP 403**）。
       - location_alias `Create`（`:144`）/ `Update`（`:185`）/ `Delete`（`:234`）：提交侧按 `req.DeptID` / `existing.DeptID` 直挂 `checkEntitySubmittedScope`，改前按 id 解析 `dept_id`；越界 403。
       - room_photo `UploadPhotos`（`:43`，`roomID`）、`DeletePhoto`（`:186`）、`SetPrimary` / `UpdateSort` / `UpdateDescription` / `BatchDeletePhotos`（`:224`）：按 `room_id` 两级链校验，越界 403；`BatchDeletePhotos` 先经批量裁剪（下条）再删。
       - **断链 / 不存在 / 空可见集 → fail-closed 403**（不回退 All）；All / 未注入 → `scopeIsAll` 零查询放行（存量零回归）。
       - **不引入整批 403**：单条越界 403；批量一律「按可见集裁剪后继续」，不是整批拒绝。
    3. **批量删除后端权威裁剪（必做）**：
       - wall/door/text 的 `BatchDelete`（`wall_service.go:92` → `s.repo.BatchDelete(ctx, ids)`）：改为先 `visible, _, err := filterEntityVisibleIDs(ctx, s.db, <entity>, ids)`，再 `s.repo.BatchDelete(ctx, visible)`；`len(visible)==0` 时按现有「空 ids 返回 nil」语义早退（**不改响应形状、不报错**）。
       - `BatchOperation`（`wall_handler.go:165-181` 等）经 service `BatchDelete` 已自动获得裁剪，**不改 handler**（handler 不属 149 独占文件面）；room_photo `BatchDeletePhotos`（`:224`）同款裁剪。
       - 越界 ID 被静默丢弃（只读裁剪，不回显 `outOfScope` 到 HTTP 响应；若需审计，仅服务端 operlog，参照 `scope_write.go:289-290` 口径，**不改 `scope_write.go`**）。
    4. **保存侧前端增量 diff（定案，必做）**：
       - `useFloorPlanEditor.ts` 的 `saveWalls`（`:156-186` 段）/ `saveDoors`（`:172-198`）/ `saveTexts`（`:200-226`）/ `saveFloorPlan`（`:240-268`）：以「本次 `loadFloorPlanData` 拉取到的可见元素集合」为基线（已有 `floorPlanData` state，`:52-73` 解析后保存），**只提交可见元素的 `create`/`update`/`delete`**；**绝不把「自己看不到的几何 ID」拼进删除集或更新集**。
       - 具体：给 `saveWalls/saveDoors/saveTexts` 增加「可见基线 ID 集」参数（从 `floorPlanData` 或 `loadFloorPlanData` 结果派生，用 `useMemo`/`useCallback` 稳定依赖，遵循 CLAUDE.md `useEffect` 依赖稳定约定）；删除集 = `基线可见 ID − 当前编辑态 ID`，**再与基线求交**；新建/更新集同样排除不在基线的 ID。不重写 Three.js 编辑器、不改「整层拉取 → 内存渲染」读取模型、不改 `wallApi/doorApi/floorPlanTextApi` 的 wrapped API 调用方式（`@/lib/opsApi.ts`）。
       - **理由（写进 SUMMARY）**：后端 `filterEntityVisibleIDs` 是权威边界（绕过前端直接调 API 也安全）；前端增量 diff 让非超管保存请求里根本不出现越界 ID，消除 operlog「请求 N / 可见 M」噪声与「静默丢弃」的语义歧义。两层都做，符合根因报告 `:66`「唯一必须改的前端/后端边界」。
    5. **响应形状不变**：`List` 只裁剪行，`list` / `total` / `current` / `pageSize` / 字段名 / 排序 / 分页结构一律不改；`Count` 与 `Find` 同源（T2 已保证），不得出现「total 全库 + list 裁剪」；写端点成功路径 `operlog.Record(...)` 仍在 `response.Success(...)` 之前（CLAUDE.md 约定），本 phase 不新增/不删 operlog 调用。
    6. **前端测试**：`__tests__/useFloorPlanEditor.test.tsx` 增加用例：构造「可见基线 3 面墙 + 越界 2 面墙（模拟后端未返回）」的编辑态，调 `saveFloorPlan`，断言 ① 提交的请求里**不含越界 ID 的 create/update/delete**；② 可见元素的 update/create 仍全部提交；③ 保存前后几何行数断言（越界元素不因整体覆写被误删）；④ `Promise.allSettled` 失败提示语义不变。用 wrapped API mock（`vi.mock("@/lib/opsApi")`，就近测试，不启动长驻进程）。
  </action>
</task>

<task id="4">
  <title>T4 · SCOPE-04：双方言 403 矩阵 + 断链脏数据抽查 + UAT 作废重跑登记</title>
  <files>
    internal/services/operations/wall_service_test.go
    internal/services/operations/door_service_test.go
    internal/services/operations/floor_plan_text_crud_test.go
    internal/services/operations/room_photo_service_test.go
    internal/services/operations/location_alias_service_test.go
    internal/services/operations/scope_entities_scope149_test.go（新增 403 矩阵 + 断链 + 双方言）
    .planning/milestones/v1.52-phases/133-scope-final/133-UAT.md（**仅 D1/D2 段 :77-91**）
  </files>
  <action>
    1. **403 矩阵（PG + SQLite 双方言断言一致）**：复用 `scope_entities_403_test.go` / `scope_read_403_test.go` / `scope_write_403_test.go` 的 fixture 与 `scopeFromContext` 注入手法，在新增 `scope_entities_scope149_test.go` 建立 5 类实体 × 档位（All / Dept 可见 / Dept 越界 / FailClosed / Self 无用户列 / 未注入 ctx）× 操作（List / GetByID / Create / Update / Delete / BatchDelete / count / primary）矩阵：
       - 读端点越界 → **空集**（不是 403）；Dept 档请求越界 `floorId`/`roomId` → 列表只含可见行 / `GetByID` 空（对应 ROADMAP `:82/:83`「空/403」）。
       - 写端点单条越界 → **显式 403**（`errors.Is(err, ErrScopeDenied)`）；批量越界 → **裁剪后继续**（`len(visible) == 可见数`，越界行数 = `len(ids)-len(visible)`，无 error）；All / 未注入 → 原样返回、零额外 DB 查询。
       - location-alias 按 `dept_id ∈ 可见集` 直挂的可见/越界行断言；rooms/photos 补挂中间件后 `scopeFromContext(ctx)` 的 `ok=true` 对照。
       - **SQLite 实跑断言 + PG contract「skip 可观测」**：SQLite 用 `t.TempDir()` / 内存库跑全矩阵逐字段断言；PG 侧本期 `TEST_PG_DSN` unset（DECISION-CI-PG、ROADMAP `:180`），**只保留 `pg_contract_test.go` 同款 skip 可观测契约、不伪造 PG 双跑**；把「PG 版期望值」以 fixture 常量（从 `CAST AS TEXT` / EXISTS 语义推导）固化进 SQLite 断言，保证口径逐字一致。
       - **静态断言**：新增 SQL 不含 `::` / `::uuid` / `ILIKE` / `isPostgres`；含 `CAST(... AS TEXT)`；`effectiveOrgExpr` 调用点存在、无内联 COALESCE/NULLIF 复制。
    2. **断链脏数据抽查**：构造 ① `floor_id` 指向软删 `ops_floors` 的 wall/door/text 行、② `room_id` 指向软删 `ops_server_rooms` 或不存在 room 的 photo 行、③ `dept_id` 不在可见集的 alias 行，断言 Dept 档读端点**返回空集**（fail-closed）、写端点 **403**、批量**裁剪为空**，且 PG/SQLite 断言结果一致；All 档仍可见（短路语义不变）。
    3. **缓存 key 登记（不实现）**：在测试注释与 SUMMARY 明确「若 5 类端点将来加缓存，key 必含 `scopeCacheKey(ctx)`（`scope_filter.go:53-74`，归 151，本 phase 不写该文件、不加缓存）」（SCOPE-04 / ROADMAP `:85`）。
    4. **UAT 作废 + 重跑窗口（严格限定改动范围）**：
       - 只改 `.planning/milestones/v1.52-phases/133-scope-final/133-UAT.md` 的 **D1/D2 段（`:79-91`）**：把 D1「几何/附件面抽查…确认未收口」与 D2「location-alias 豁免复核…确认该面未收口」的**旧结论作废**，改写为「Phase 149 已按方案 D 收口（5 类端点按可见集裁剪 + 写 403/裁剪 + 保存侧不误删），D1/D2 原『确认未收口』结论作废，**待 QA 裁定重跑窗口与验收人后重跑**（只登记、不擅自勾选完成）」，并附重跑要点（Dept 档越界 `floorId`/`roomId` 空/403、断链 fail-closed、保存侧几何行数前后断言、双方言 403 矩阵）。
       - **不动**：`## D` 标题下 D3（`dept-mapping-template` 已收口复核 `:92-94`）、`## E` 及之后任何段、150 的 `132/133 backfill UAT` 另一段（ROADMAP `:130/:200` 硬登记）。改动前后用 `git diff --` 核对**仅 D1/D2 行变更**（`git diff` 为只读，允许；禁 `git add/commit`）。
    5. **就近测试归属**：5 个 service 的 `*_test.go` 就近放业务包内（AGENTS.md），新增矩阵测试放 `scope_entities_scope149_test.go`；**不建 `test/` 单数目录、不放 `tests/e2e/`（归 145）、不放 `tests/integration/` 除非确需多表装配**。测试函数名加 `Scope149` 前缀避免与 148/150 并行 executor 重名。
  </action>
</task>

<task id="5">
  <title>T5 · 自验证（只跑相关包，不跑全仓）+ 反向变异红绿</title>
  <files>
    （只读验证，不改文件；变异为临时改动后立即还原）
  </files>
  <action>
    1. **后端相关包测试**（按实际包，不跑全仓 `go test ./...` —— 三 executor 并行同工作区，全仓会读到其它 phase 未完成文件）：
       - `go test -count=1 ./internal/services/operations/`
       - `go test -count=1 ./internal/api/v1/operations/`
       - `go test -count=1 ./pkg/middleware/`
       - 若 5 类 service 有子包：`go list ./internal/services/operations/...` 后按实际结果补跑（**不跑 `./...` 全仓**）。
    2. **构建检查**：`go build ./...`（AGENTS.md/CLAUDE.md 硬约定：改 Go 后必跑；`go build` 不落盘二进制、产物不落根目录）。若构建因其它 phase 并行中途文件报错，记录并待 Lead 集成期复核，**不擅自改其它 phase 文件**。
    3. **前端验证**：
       - `cd xingran-react-frontend && npm run type-check`
       - `cd xingran-react-frontend && npx vitest run src/pages/operations/floors/__tests__/useFloorPlanEditor.test.tsx`（**只跑 useFloorPlanEditor 相关**，不跑全量 vitest；同波 150 会改前端其它文件）
    4. **反向变异（红绿验证，必须还原）**：
       - 临时去掉 `wall_service.go::List` 的 `entityReadScope`（或 `scope_entities.go` 的 `applyEntityReadScope` 中 `wrapScopeExists` 调用）→ Dept 档越界 `floorId` 断言（T4 矩阵）**必红** → 还原 → **复跑全绿**。
       - 临时把 `BatchDelete` 的 `filterEntityVisibleIDs` 裁剪改回 `repo.BatchDelete(ctx, ids)` → 「批量越界 ID 被裁剪」断言**必红** → 还原 → 绿。
       - 临时把 `router.go:1174-1177` 补挂注释掉 → rooms/photos `scopeFromContext(ctx) ok=true` 断言**必红** → 还原 → 绿。
       - 每次变异只动一处、立即还原，最终工作区必须回到全绿状态。
    5. **路由登记核对**：`node scripts/check-route-registry.mjs`（若本机可执行）——本 phase 只补挂中间件、未增删路由，确认**零漂移**；不可执行则记录 skip（不伪造）。
    6. 汇总 T1~T5 全部命令与红绿结果，写入 T6 SUMMARY「验证结果」段。
  </action>
</task>

<task id="6">
  <title>T6 · 写 149-SUMMARY.md（改动 / 4 REQ / 验证 / 方案 D 实证 / Known Gaps / 未提交）</title>
  <files>
    .planning/phases/149-scope-scope-01-02-03-04/149-SUMMARY.md
  </files>
  <action>
    1. **改动清单**：逐文件列 T1~T4 实际改动（含 `router.go` 仅 `:1174-1177` 一行的 diff 摘要、`scope_entities.go` 新增的实体常量/case、`useFloorPlanEditor.ts` 增量 diff 逻辑、`133-UAT.md` 仅 D1/D2 段）；附「未触碰文件面」核对表（逐条对照本 plan §3 并行冲突面 + ROADMAP `:119-148`）。
    2. **4 REQ 满足情况**：SCOPE-01（读裁剪 + 断链 fail-closed + All 短路）/ SCOPE-02（补挂 + 前置校验）/ SCOPE-03（写 403/裁剪 + 保存侧不误删 + 响应形状不变）/ SCOPE-04（双方言矩阵 + UAT 作废重跑 + 缓存 key 登记）逐条给「结论 + 证据（测试名/命令/断言）」；附 ROADMAP Phase 149 **5 条成功标准 `:81-86`** 的逐条对照表。
    3. **方案 D 与「回归面被高估」的实证**：
       - 零迁移（无新列 / 无 migration / 无回填）；响应形状 diff = 0（只裁剪行）；不引入整批 403（读越界恒空集）；前端改动仅 `useFloorPlanEditor.ts` 保存侧增量 diff（不重写编辑器）；
       - 与根因报告 `:60-66` 对照：归属链现成、`scope_entities.go` 基建现成、同页 `getFloorWorkstationsAll` 已收口 → 收口后语义反而一致；**唯一必须改的边界 = 后端批量裁剪 + 前端保存 diff**，实测改动行数/文件数远低于 LOCKED 裁定「Three.js 联动回归面大」的假设；
       - 附 `visibleIDsForEntityWrite` 白名单发现（`scope_write.go:292-333` 仅 4 域、`default` 报错、归 150）→ 定案落 `scope_entities.go::filterEntityVisibleIDs` 扩展，未写 150 文件。
    4. **验证结果**：T5 全部命令 + 红绿结果逐条粘贴（含 `go build ./...`、`go test` 三包、`npm run type-check`、`npx vitest run ...`、反向变异 3 项）；PG contract「skip 可观测 / `TEST_PG_DSN` unset / 不伪造双跑」显式登记。
    5. **Known Gaps**：① D1/D2 UAT 重跑窗口由 QA 裁定、本 phase 只登记不勾选；② PG 实机双跑仍为前置依赖（DECISION-CI-PG）；③ 缓存 key `scopeCacheKey(ctx)` 仅为契约登记、本 phase 未加缓存；④ 若生产存在断链脏数据，非超管按 fail-closed 看不到这些行（ds05 裁定 1 `:76` 已登记，不为脏数据开旁路）；⑤ ① 的 A(150)+B(151) 不在本 phase 范围，本 phase 不宣称「统一架构已落地」。
    6. **明确「未 git commit」**：写「本 plan `commit:false`，**未执行任何 `git add` / `commit` / `stash` / `checkout` / `reset`**，由 Lead 在 W2 收口统一提交；`internal/api/router.go` 的 149→151 先后顺序按 ROADMAP `:145` 硬登记（151 在 149 commit 之后基于最新文件改）」。
    7. 若任一 REQ 未满足 / 验证未全绿 / 出现跨 phase 写交集：**不得伪造完成**，在 SUMMARY 标 `gaps_found` 并返回 `## PLANNING BLOCKED` + 具体原因。
  </action>
</task>

---

## 6. 验证标准（must_haves，逐条对应 ROADMAP Phase 149 的 5 条成功标准）

| # | ROADMAP 成功标准 | 本 plan 落点 | 可观察验收 |
|---|---|---|---|
| 1 | `:82` Dept 档 5 类列表只得到可见集内行；断链 fail-closed 空集；`GetByID`/count/primary/batch-delete 前置全走可见集校验 | T1/T2/T4 | T4 矩阵：walls/doors/texts/rooms-photos/location-alias 的 Dept 档越界 `floorId`/`roomId` 列表**只含可见行**；软删 floor/room 脏数据 → 空集；`GetByID`/`CountPhotos`/`GetPrimaryPhoto`/`BatchDeletePhotos` 前置断言全绿 |
| 2 | `:83` `router.go:1174-1177` 补挂后实测生效（注入 `ScopeResult`）；Dept 档越界 `floorId`/`roomId` 返回空/403；批量按可见集裁剪 | T1/T3/T4 | 补挂前后 `scopeFromContext(ctx) ok=false→true` 对照；单条写越界 `errors.Is(ErrScopeDenied)` → HTTP 403；批量 `filterEntityVisibleIDs` 裁剪后 `len(visible)==可见数`、越界行数正确、无 error |
| 3 | `:84` 非超管在 Three.js 编辑器保存整层几何后，看不到的几何元素不因整体覆写被误删（前后几何行数断言）；响应形状不变 | T3 | `useFloorPlanEditor.test.tsx` 保存前后几何行数断言；后端 `BatchDelete` 裁剪 + 前端增量 diff 双层；`list`/`total`/字段 diff = 0 |
| 4 | `:85` PG + SQLite 双方言 403 矩阵与断链脏数据抽查断言一致；`133-UAT.md:79-91` D1/D2 作废 + 重跑窗口登记；将来加缓存 key 必含 `scopeCacheKey(ctx)` | T4 | SQLite 全矩阵逐字段断言 + PG contract skip 可观测；`133-UAT.md` diff **仅 D1/D2 段**；SUMMARY 登记缓存 key 契约（不实现、不写 `scope_filter.go`） |
| 5 | `:86` **零迁移（方案 D）**：不新增 `org_id` 冗余列（E）、不把几何合并进楼层 JSONB（F）；`visibleIDsForEntityWrite` 仅复用不改（归 150）；`scope_filter.go` 归 151 不得触碰 | §4 裁决表 / T2/T3 | `git diff` 无 migration / 无新列 / 无 JSONB 重写；`scope_write.go`、`scope_filter.go`、`workstation_service.go`、`excel_service.go`、`useBackfillDept.tsx`、`.golangci.yml` **零改动**；`wrapScopeExists` 只调用不写 |
| 6 | **All 档零响应形状/数值回归**（Constraints `:158`、ROADMAP `:84`） | T2/T4/T5 | `scopeIsAll` 短路零额外 DB 查询；All 档 `List`/`total`/字段/排序逐字不变；`go build ./...` + 三包 `go test` + `type-check` + vitest 全绿；反向变异红绿全通过 |
| 7 | **无整批 403 / 不引入 403 面扩张**（方案 D 语义） | T2/T3 | 读越界恒空集（非 403）；批量裁剪后继续（非整批 403）；仅单条写越界显式 403 |

---

## 7. 禁止（执行期硬红线）

1. **git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git reset` / `git cherry-pick` 一律禁止（`commit:false`，Lead 统一提交）；`git diff` / `git status` 只读允许。
2. **改其它 phase 文件面**：尤其 `internal/api/router.go` **只能改 `:1174-1177` 一行**，不得碰 151 的 ScopedDB 挂载区、不得顺手改其它 `DataScopePermission` 行；`scope_filter.go`（含 `wrapScopeExists`/`scopeCacheKey`）、`scope_write.go`、`base/GORMRepository` 一行不动；148/150/145/146/147 文件面逐条见 §3。
3. **方案 E**：新增 `org_id` 冗余列 / migration / 回填 / 双写 —— 禁止（零迁移）。
4. **方案 F**：把几何合并进楼层 JSONB、重写 Three.js 编辑器 —— 禁止（保存侧只做增量 diff）。
5. **引入整批 403**：批量写端点必须「按可见集裁剪后继续」，不得整批拒绝；读端点越界恒空集不是 403。
6. **动 79 条跟踪表与 LOCKED 裁定原文**（`.planning/notes/261108-*` 归档件只读）；`133-UAT.md` 只改 D1/D2 段、不勾完成、不改 D3/E 段、不动 150 的 132/133 backfill UAT 另一段。
7. **启动长驻进程**（dev server / watcher / DB）：不落 `&` / `nohup` / `Start-Process`；一次性命令前台跑或 `ops_bash run_in_background` 托管。
8. **写生产数据**：不切生产认证模式、不写生产库；测试一律 `t.TempDir()` / 内存 SQLite / mock。
9. **方言分叉**：禁 `::uuid` / `ILIKE` / 裸 `ON CONFLICT` / `isPostgres` 分支；禁内联复制 `effectiveOrgExpr` 的 COALESCE/NULLIF。
10. **伪造 PG 双跑**：本期 `TEST_PG_DSN` unset，PG contract 只做「skip 可观测」，不建 `services: postgres`、不写假双跑证据。
11. **不跑全仓**：禁 `go test ./...` 与全量 `vitest`（三 executor 并行同工作区会读到其它 phase 未完成文件）；只跑 T5 列出的相关包/用例。
12. **不新增跨 phase 文件**：403 矩阵测试就近落 `internal/services/operations/scope_entities_scope149_test.go`，前端测试就近落 `floors/__tests__/`；不建 `test/` 单数目录、不放 `tests/e2e/`（归 145）。

---

> **完成条件**：`149-PLAN.md` 已落盘；executor 按 T1→T6 执行，T5 全绿、T6 SUMMARY 写明「未 git commit」后，返回 `## PLANNING COMPLETE (Phase 149)` + 任务数 / 验证命令 / 文件面 / `router.go` 改动范围一句话。
> 冲突（跨 phase 写交集、裁决与代码现状矛盾、必读缺失）返回 `## PLANNING BLOCKED` + 具体原因。
