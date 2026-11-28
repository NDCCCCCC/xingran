---
phase: 151
plan: 01
wave: 3
depends_on: [145, 146, 147, 148, 149, 150]
files_modified:
  - pkg/query/scoped_db.go                                  # 新增：ScopedDB / ScopeCacheKey / Scopes 唯一查询起链点
  - pkg/query/scoped_registry.go                            # 新增：资源归属注册表（回调注册制，pkg 层不感知 ops 表名）
  - pkg/query/scoped_db_test.go                             # 新增：All 兜底短路 / 注册表 / 双方言 / 无重复谓词
  - internal/services/base/service.go                       # GORMRepository 的 List/GetByID 读链改走 ScopedDB；写链一字不动
  - internal/services/base/scoped_repo_test.go              # 新增：泛型 repo 读链同源 + 写链零回归 + 编译期守护
  - internal/services/operations/scope_registry.go          # 新增：ops 实体→归属链注册（引用 149 成果，不改 149 任何文件）
  - internal/services/operations/scope_filter.go            # 提升为不可绕过的查询层；新增导出 ScopedCacheKey 薄封装
  - internal/services/operations/scope_registry_test.go     # 新增：注册表覆盖 / 零裸起链静态断言 / 单一真实源断言
  - internal/services/operations/asset_service.go            # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/building_service.go        # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/floor_service.go           # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/workstation_service.go     # 8 核心 service 起链处（仅起链行；150 响应字段逻辑禁改）
  - internal/services/operations/infopoint_service.go       # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/server_room_service.go     # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/dedicated_line_service.go  # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/room_device_service.go     # 8 核心 service 起链处（仅起链行）
  - internal/services/operations/building_service_typesafe.go  # 类型安全旁路实现，起链挂 ScopedDB
  - internal/services/operations/excel_service.go           # 导入/导出起链（仅起链行；150 的 ImportResult / Errors[].Row 逻辑禁改）
  - internal/services/operations/floor_cache_impl.go        # floor:tree 固定 key 挂 scopeCacheKey 维度
  - internal/services/operations/sidechannel_homology_test.go   # 新增：8 statistics / total / counts-by-floor / dept-options / workstations-all 同源
  - internal/services/operations/response_contract_test.go      # 新增：响应契约层差分防复活静态断言
  - internal/services/operations/pg_contract_test.go            # 追加：同源 PG contract（TEST_PG_DSN unset → t.Skip 可观测）
  - internal/api/router.go                                  # 仅挂载相关行（1 注释 + 1 装配调用），基于 149 之后最新文件
  - .planning/phases/151-scopeddb-b-sidechannel-04-arch-01/151-SUMMARY.md
autonomous: true
commit: false
---

# Phase 151 · 统一 ScopedDB / B — ① 防回归 + ② 机制化（PLAN 01）

> **v1.55 最后一块（W3）**。本 plan 为**可直接执行**的交付物；Lead 统一提交，**任何 executor 不得执行 git 写操作**。输出与注释一律简体中文。
> **必须最后执行、不与其它 phase 并行**：`depends_on: [145,146,147,148,149,150]`；W1（145/146/147）与 W2（148/149/150）均已交付，`router.go` 的 149 补挂（`:1176 roomPhotos.Use(middleware.DataScopePermission(core))`）已落地。
> **B 未落地前 150-SUMMARY 已声明「不得宣称 ① 已根治」**。B 的职责**不是再删响应字段**（150 已做完），而是 **① 防回归 + ② 机制化**：把「同源」从靠约定升级为靠机制。

---

## 1. 必读顺序（开工前按序读完，均为只读）

| # | 文件 | 读什么 / 为什么 |
|---|------|----------------|
| 1 | `.planning/STATE.md` | **v1.55 active 段 `:20-26`**、`## Current Position :99-104`；**`## Deferred Items` 的 v1.55 W1 行 `:324` 与 W2 行 `:325-326`**（W1/W2 Known Gaps 权威口径，尤其 FUTURE-01 4 处引用、132/133 UAT 重跑、`TEST_PG_DSN` 前置依赖、149 router 契约基线更新）；`Constraints :53-59` |
| 2 | `.planning/ROADMAP.md` | **Phase 151 详情 `:100-111`（5 条成功标准逐条 = 本 plan `must_haves` 来源）**；**并行冲突面 / 函数级归属表 `:113-148`**（尤其 `:127` `scope_entities.go` 归 149、`:145` `router.go` 149 先改 151 后改、`:146` `scope_filter.go` 归 151、`:139` **151 必须最后执行**）；Constraints `:150-161`；5 项裁定 `:163-171`；禁区 `:173-181`；账务 `:183-186`；Known Gaps `:188-200` |
| 3 | `.planning/REQUIREMENTS.md` | **SIDECHANNEL-04 `:76` + ARCH-01 `:78-83` 全文**；根治定义 `:70`、A+B 组合 `:71`；5 项裁定 `:7-17`；**Constraints `:98-106`**；Out of Scope `:85-96`；Traceability `:108-136`（`ARCH-01` 已由「可选」升级**必需**） |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` | **§2 共性架构结论 `:153-162`（「① 的「统一架构」段」：统一解应用层 ScopedDB / QueryBuilder 作用域 + 资源归属注册表 + 响应契约层；DB 视图/RLS 不可行实证）**；§0 `:13-26`；§1 ① 方案对比 `:32-54`（尤其 `:48` B 行）；§4 未决问题 `:188-199`（`:197` 架构立项问题已由用户裁定为**必需**） |
| 5 | `.planning/phases/150-a-sidechannel-01-02-03/150-SUMMARY.md` | **A 已做什么、B 不能重复什么**：§1 改动清单、§2「不纳入本 phase」`:57`、**§4 明确声明 `:182-187`**（B 未落地前不得宣称 ① 已根治 / A 只完成去规模化不能防回归 / 未伪造统一架构已落地）、§5 Known Gaps `:191-197` |
| 6 | `.planning/phases/149-scope-scope-01-02-03-04/149-SUMMARY.md` | **② 已用 `scope_entities.go` 落地收口**：§1.1 实体常量 / `entityExistsSub` / `applyEntityReadScope` / `filterEntityVisibleIDs`；§1.2 5 个 service；§1.3 router `:1174-1177` +2 行；**§3.2「回归面被高估」实证 `:159-169`**（本 phase 影响面评估范本）；§3.3 `visibleIDsForEntityWrite` 白名单；§1.7 未触碰文件面（`:101` 明确 `scope_filter.go` / `base/GORMRepository` 归 151） |
| 7 | `pkg/datascope/datascope.go` | **`ScopeResult` `:43-49`、`Resolve` `:100-177`（All/0 短路 `:103-109`、异常枚举 fail-closed `:172-175`）、`FromContext` `:63-70`、`WithContext` `:59-61`、`AllScope` `:73-75`** —— 单一真实源，**禁止内联复制** |
| 8 | `internal/services/operations/scope_filter.go` | **`scopeIsAll` `:37-39`（未注入 ctx → All 兜底短路，零回归硬约束）**、`scopeCacheKey` `:53-74`、`orgScopeClause` `:94-102`、`applyScopeToSub` `:205-223`、**`wrapScopeExists` `:231-237`（目前是「一个函数」，靠每个 service 自觉调用）**、`applyBuildingScope` `:252-271`、`applyExcelWorkstationScope` `:291-313` |
| 9 | `internal/services/operations/scope_entities.go` | **149 的实体归属注册表雏形**：实体常量 `:39-54`、`entityScopeTables` `:57-69`、`entityBuildingOrgExpr` / `entityWorkstationOrgExpr` / `entityFloorOrgExpr` `:71-91`（引用 `effectiveOrgExpr` 单一真实源）、`entityExistsSub` `:105-`（归属链 EXISTS 骨架） |
| 10 | `internal/services/base/service.go` | **`GORMRepository[T]` `:38-45`**；`Scope = func(*gorm.DB) *gorm.DB` `:15`（与 GORM `Scopes` 一等公民同形）；`GetByID` `:71-98`、**`List` `:113-138`（scopes 在 `Count` 之前应用 → Count/Find 天然同源）**；`Create/Update/Delete/BatchDelete` `:48-60/:144-149`（**写链，禁挂读谓词**） |
| 11 | `pkg/query/`（`builder.go` / `pagination.go`） | 现有 builder / pagination 形态；**`pkg/query` 不得 import `internal/services/operations`**（分层 + 循环依赖），故注册表必须走回调注册制（见 §3 F-4） |
| 12 | `internal/api/router.go` | **149 已补挂 `roomPhotos` `:1176`**；8 个 `/statistics` 端点 `:799/:856/:912/:1063/:1106/:1147/:1199/:1240`；`counts-by-floor :914`、`dept-options :911`、`workstations-all :919`；ops 组定义位置（约 `:780` 前） |
| 13 | `pkg/middleware/permission.go` | **`DataScopePermission` `:497-541`** 已把 `ScopeResult` 注入 `c.Request.Context()` —— ScopedDB 无需新增中间件，只从 ctx 取值 |
| 14 | `CLAUDE.md` / `AGENTS.md` | Handler-Service、`response.Success()/Error()`、operlog 写操作约定（success path 末尾、`response.Success` 之前 `Record`）、25 个 OperType 常量、`0=正常 1=停用`、**SQLite+PG 双方言（`CAST AS TEXT`、禁 `::uuid`/`ILIKE`/裸 `ON CONFLICT`）**、Go 单测就近 `*_test.go`、构建产物只落 `out/`、前端 wrapped API |

> 读完如发现**根因报告 / ROADMAP 与代码现状存在语义冲突**（非笔误）→ 立即停手，返回 `## PLANNING BLOCKED` 并列出具体冲突。
> **本 plan 起草时已逐项实证核对**，结论见 §3 F-7：唯一偏差为 ROADMAP `:104` 的基座路径笔误与「485 Go 文件」口径，均**非阻塞**，以代码实证为准。

---

## 2. 目标

**① 防回归（SIDECHANNEL-04）**：A(150) 只把当前响应体去规模化，**不能防回归** —— 未来新增端点仍可能漏挂可见集、仍可在明细/计数两条链上写出「全库 − 可见」差值。B 用**统一 ScopedDB** 把「所有 `COUNT(*)` / `SUM` 与明细查询同源」从「靠每个 service 自觉调 `wrapScopeExists`」升级为「**GORM `Scopes` 强制唯一查询起链点 + 静态断言零裸起链**」的机制保证。

**② 机制化（ARCH-01）**：149 已把 5 类豁免端点用 `scope_entities.go` 收口，但仍是「靠约定」（每个 service 自觉调用 `entityReadScope`）。B 把「靠约定」变「靠机制」：统一「资源归属注册表」，实体在注册表登记后**自动获得可见集谓词**，rooms/photos 的「中间件挂了没接线」由机制兜住；**不推翻、不重写 149 成果**。

**响应契约层**：静态断言任何端点响应体不含「全库规模 − 可见规模」派生数值，防 150 成果复活。

**根治声明门槛**：**A(150) + B(151) 都完成才算 ① 根治**。本 phase 完成前**不得**在任何产物写「统一架构已落地 / ① 已根治」；本 phase 完成后方可在 `151-SUMMARY.md` 明确「**A(150)+B(151) 均完成，① 侧信道根治达成**」。

**2 个 REQ 一一对应**：`SIDECHANNEL-04` → T4 + T5、`ARCH-01` → T1 + T2 + T3；T6 全量自验证、T7 SUMMARY。

---

## 3. 已确认事实（不要重新调研，直接作为 PLAN 输入）

- **F-1｜`pkg/datascope` 是单一真实源**。`DataScopePermission`（`pkg/middleware/permission.go:497-541`）已把 `ScopeResult` 注入 `c.Request.Context()`；`ScopeResult{All, DeptIDs, ResourceHook, FailClosed, UserID}`（`datascope.go:43-49`）；`All/0/unset` 短路零 DB 查询（`:103-109`）；异常枚举 fail-closed（`:172-175`）；`FromContext` 未注入返回 `(零值, false)`（`:63-70`）。**禁止内联复制 `datascope` 档位判定 / `effectiveOrgExpr` COALESCE-NULLIF 表达式 / 归属链 SQL 骨架**。
- **F-2｜「未注入 ctx → All 兜底」是零回归硬约束**。`scope_filter.go:37-39 scopeIsAll(scope, ok) = !ok || scope.All`；`datascope.go:6` 契约注释「0/unset → All 超管零回归」；`scope_filter.go:5-7` 明确「未挂注入中间件（`FromContext` 返回 false）同样按 All 兜底，保证存量调用方（直接以 `context.Background()` 调用的 SQLite 单测 / 内部任务）行为不变」。ScopedDB **必须原样保留该短路**。
- **F-3｜现状「统一收口入口」只是一个函数**。`scope_filter.go:231 wrapScopeExists` 靠每个 service 自觉调用（根因报告 `:156` 定性）；`:53-74 scopeCacheKey` 是缓存 key 的 scope 维度段。`scope_entities.go`（149）已有「实体→归属链」雏形：实体常量 10 类 + `entityScopeTables` + `entityExistsSub` + `entityFloorOrgExpr()` / `entityWorkstationOrgExpr()`（引用 `effectiveOrgExpr`）。
- **F-4｜分层硬约束 → 必须走「回调注册制」**。`pkg/query` 属 `pkg/` 层，**不得 import `internal/services/operations`**（会造成 import cycle 且违反 `pkg/datascope` 已确立的「不感知 ops 业务表名」契约，见 `datascope.go:1-11`）。故「资源归属注册表」拆两层：
  - `pkg/query/scoped_registry.go`：`Register(table string, f ScopeFactory)` / `Lookup(table)` —— **只存回调、零业务表名、零 SQL**；
  - `internal/services/operations/scope_registry.go`（151 新增）：在 ops 包内把 149 的实体常量 / `effectiveOrgExpr` / `entityExistsSub` / `wrapScopeExists` / `applyOrgScope`（`dept_id` 直挂）**注册为回调**，并注册 `ScopedCacheKey`（`scope_filter.go` 新增导出薄封装，委托既有 `scopeCacheKey`）—— **算法唯一实现仍留在 149/131 的既有文件里，零复制**。
  - `pkg/query.ScopedDB` 的 GORM `Scopes` 回调在 SQL 构建时按 `tx.Statement.Table`（空则回退 `tx.Statement.Schema.Table`，再空则 `Tabler` 断言）查注册表 → 调 factory 得谓词 → 应用。
- **F-5｜150 已完成，B 不得重复**：响应体 6 字段删除、全库 `countQ` 删除、越界行静默跳过、差分攻击测试（`sidechannel_diff_attack_test.go`）、`ScopeWriteOperParam` 审计链保留、`ImportResult.ScopeOutOfScope` 改 `json:"-"`。**151 只改 `workstation_service.go` / `excel_service.go` 的「起链行」，上述响应字段逻辑一字不动**。
- **F-6｜149 已完成，B 不得推翻**：5 个 service（`wall/door/floor_plan_text/room_photo/location_alias`）收口、`scope_entities.go`、`router.go:1176` roomPhotos 补挂、`useFloorPlanEditor.ts` 保存侧增量 diff。**151 不改这些文件**，只让它们在注册表登记后自动获得谓词（见 §4 归属表「保留」项）。
- **F-7｜ROADMAP 与代码现状的两处偏差（已实证核对，非阻塞）**：
  1. **`ROADMAP.md:104` 写 `internal/models/base/GORMRepository`，代码实际在 `internal/services/base/service.go:38`**（`internal/models/base/` 目录不存在；`internal/models/` 下无 `base/` 子目录）。149-SUMMARY `:101` 沿用同一笔误。ROADMAP 对基座**职责**的描述（Scopes 唯一查询起链点、禁止裸起链、All 兜底）与代码现状完全一致，且任务书已给定正确路径 → **按 `internal/services/base/service.go` 执行**，并在 `151-SUMMARY.md` 登记该笔误（不改 ROADMAP 原文，归 Lead 收档处理）。
  2. **`ROADMAP.md:104` / `:139` 称「485 Go 文件受影响」是全仓编译面，不是直接消费面**：`ops_grep` 实证 `GORMRepository` **全仓仅 17 个文件 / 56 处引用**（详见 §4.1），`internal/services/system/`、`pkg/`、`vdi` 相关包 **0 文件直接消费**。与 149-SUMMARY §3.2「回归面被高估」同理；本 phase 仍按 ROADMAP 要求跑**全量矩阵**（§T6），但零回归策略据实证收窄。
- **F-8｜统一解（根因报告 §2 `:159-160`）**：`DataScopePermission` → **ScopedDB**（GORM `Scopes` 强制唯一查询起链点，禁止裸 `s.db.WithContext(ctx)` 起链）+ 资源归属注册表（引用 `effectiveOrgExpr` / 归属链 EXISTS 骨架 / `dept_id` 直挂，**禁止内联复制**）+ 内建 `scopeCacheKey(ctx)` → **响应契约层**禁止「全库规模 − 可见规模」回显。零件全部现成，`EXISTS (?) + CAST AS TEXT` 已验证双方言。
- **F-9｜不可行方向（勿立项）**：DB 视图（可见集每用户动态、无法参数化）、PG RLS（SQLite 无 RLS、Go 共享连接池串味需每请求 `SET LOCAL` 独占事务、双方言两套安全模型）、按用户分表（`org_id` 可变、与 UUID 单表模型冲突）。见 ROADMAP `:111`、REQUIREMENTS `:83`、根因报告 `:160`。
- **F-10｜`TEST_PG_DSN` 本期仍为前置依赖**：用户裁定不建 `services: postgres`（REQUIREMENTS `:13`/`:92`、ROADMAP `:169`/`:180`/`:198`）。PG contract 用 `t.Skip` 可观测 + 静态断言，**不伪造双跑**；沿用 150 的 `pg_contract_test.go` 门控范式。
- **F-11｜`base.Scope` 与 GORM `Scopes` 同形**：`service.go:15 type Scope = func(*gorm.DB) *gorm.DB`；`List(ctx, page, scopes ...Scope)` 在 `Count` **之前**依序应用 scopes（`:113-124`）→ 只要 scope 谓词进入该链，`total` 与 `List` 行数天然同源。**这是 SIDECHANNEL-04 的最小改动杠杆**。
- **F-12｜8 个 `/statistics` 端点**（`router.go` 实证）：`building :799` / `floor :856` / `workstation :912` / `asset :1063` / `infoPoint :1106` / `serverRoom :1147` / `dedicatedLine :1199` / `roomDevice :1240`；另有 `counts-by-floor :914`、`dept-options :911`、`workstations-all :919`。**正好对应 8 个核心 service**（见 §4.3）。

---

## 4. 全仓基座影响面评估（本 phase 重点）

> 全部数字由本次 planning 用 `ops_grep` / `ops_glob` 实证统计（命令见 §4.5），非引用 ROADMAP 估算。

### 4.1 `GORMRepository[T]` 消费方全量清单（17 文件 / 56 处）

| 包 / 域 | 文件 | 引用数 | 分类 | 是否需挂 scope |
|---|---|---:|---|---|
| `internal/services/base`（**定义 + 契约测试**） | `service.go` | 12 | 基座定义（`GetByID/List/Create/Update/Delete/BatchDelete`） | **改读链**（T3 基座最小化改造） |
| 同上 | `base_80_05_test.go` | 4 | 基座契约测试 | **只读回归**（不改，全量 `go test` 守护） |
| 同上 | `service_test.go` | 3 | 基座契约测试 | **只读回归**（同上） |
| `internal/services/operations`（**业务 service**） | `asset_service.go` | 2 | 8 核心之一 | **改**（T3） |
| 同上 | `building_service.go` | 2 | 8 核心之一 | **改**（T3） |
| 同上 | `dedicated_line_service.go` | 3 | 8 核心之一 | **改**（T3） |
| 同上 | `door_service.go` | 3 | **149 收口成果** | **保留**（不改；注册表登记自动获得） |
| 同上 | `floor_plan_text_service.go` | 3 | **149 收口成果** | **保留**（同上） |
| 同上 | `floor_service.go` | 5 | 8 核心之一 | **改**（T3） |
| 同上 | `infopoint_service.go` | 3 | 8 核心之一 | **改**（T3） |
| 同上 | `room_device_service.go` | 3 | 8 核心之一 | **改**（T3） |
| 同上 | `server_room_service.go` | 3 | 8 核心之一 | **改**（T3） |
| 同上 | `wall_service.go` | 3 | **149 收口成果** | **保留**（同上） |
| 同上 | `workstation_service.go` | 4 | 8 核心之一 | **改**（T3，**仅起链行**） |
| `internal/services/operations`（**辅助 + 测试**） | `pagination_helper.go` | 1 | 分页辅助 | **只读回归**（不改） |
| 同上 | `floor_service_crud_test.go` | 1 | 就近测试 | **只读回归**（不改） |
| 同上 | `workstation_update_createdat_test.go` | 1 | 就近测试 | **只读回归**（不改） |
| **其它域** | `internal/services/system/**`（87 文件扫描）/ `pkg/**`（114 文件扫描）/ `vdi` 相关包 | **0 / 0 / 0** | — | **零直接消费 → 零回归保证** |

**包分布结论**：直接消费面 = **2 个包**（`internal/services/base` 3 文件、`internal/services/operations` 14 文件），**共 17 文件 / 56 处**；其中需改的仅 **8 核心 service + 基座 `List/GetByID` + 辅助起链文件**（见 §4.3）。`system` / `vdi` / `pkg` 三域 **0 文件直接消费** → 基座改造对它们的影响只有「全仓编译」一层，由 `go build ./...` + `go test ./...` 全量守护（§T6）。这解释了为何 ROADMAP 「485 Go 文件」是编译面而非改造面（F-7）。

### 4.2 ScopedDB 的注入方式（定案：基座改动降到最小）

**定案 = 「context 携带 scope + GORM `Scopes` 回调」+「ops 域统一 helper」，不动 `GORMRepository` 构造签名**。

| 决策 | 定案 | 理由 / 回归守护 |
|---|---|---|
| 构造签名 | **`NewGORMRepository[T](db *gorm.DB)` 一字不动**，不新增 ctx / scope 参数 | 14 个 ops 消费方 + 3 个 base 契约测试零签名改动；避免 485 文件编译面连锁 |
| 唯一查询起链点 | `pkg/query.ScopedDB(ctx, db) *gorm.DB`：`datascope.FromContext(ctx)` → `ok=false \|\| All` **原样返回 `db.WithContext(ctx)`（零回归短路）**；否则 `db.WithContext(ctx).Scopes(scopeFn)` | `Scopes` 是 GORM 一等公民；`scopeFn` 在 SQL 构建时按 `Statement.Table` 查注册表挂谓词（F-4） |
| 基座改造范围 | `service.go` **仅 `List` `:114` 与 `GetByID` `:74` 两处**把 `r.db.WithContext(ctx).Model(new(T))` 改为 `query.ScopedDB(ctx, r.db).Model(new(T))` | 这两处是**读链**，且 `List` 的 scopes 在 `Count` 之前应用（F-11）→ `total` / `List` 天然同源 |
| **写链禁改** | `Create :49` / `Update :54` / `Delete :59` / `BatchDelete :148` **一律保持 `r.db.WithContext(ctx)`** | GORM `Scopes` 会把谓词带进 `UPDATE/DELETE` 的 WHERE，挂读 scope 会**破坏写语义**（越界行写不动、批量删除被裁剪）。写侧可见集校验已由 132/149 的 `ErrScopeDenied` / `filterEntityVisibleIDs` 承担，**不重复、不覆盖** |
| 命名空间 | `pkg/query` **不 import ops**；注册表回调由 ops 包 `scope_registry.go` 注册（F-4） | 避免 import cycle；保持 `pkg/datascope`「不感知表名」分层契约 |
| 缓存 key | `pkg/query.ScopeCacheKey(ctx)` **委托注册表回调** → ops 侧 `scope_filter.go` 新增导出 `ScopedCacheKey(ctx)` 薄封装 → 委托既有 `scopeCacheKey(ctx)` `:53-74` | 算法唯一实现不迁移、不复制；`floor:tree` 固定 key 因此能挂上 scope 维度（STATE `:79` 硬约束） |
| `scope_filter.go` 定位 | 从「一个函数」提升为「**不可绕过的查询层**」：`wrapScopeExists` 保留同名同签名（131/149 存量调用零改），新增强制层入口 + 导出 `ScopedCacheKey`；强制层由 `ScopedDB` 在起链时调用 | 符合根因报告 `:156` 定性；静态断言保证不可绕过（T3） |

**不采用「改泛型 repo 构造签名」方案**（回归矩阵过大）；**若执行期实测确需改泛型 repo**（例如 `Statement.Table` 在 `Scopes` 回调内不可靠），必须先在 `151-SUMMARY.md` 给出下列回归矩阵并全绿后方可实施：

| 回归矩阵（仅确需改签名时启用） | 守护方式 |
|---|---|
| 14 个 ops 消费方 + 3 个 base 契约测试全部编译 | `go build ./...`（全量，非定向） |
| `Create/Update/Delete/BatchDelete` 写语义零回归 | `internal/services/base/*_test.go` + `go test ./...` 全绿；新增 `TestScopedRepo_WriteChainUntouched` 断言写链不含 scope 谓词 |
| `All / 未注入 ctx` 短路零额外 SQL | `TestScopedDB_NoInjection_AllFallback`（`DryRun` SQL 逐字比对 + 查询计数 0） |
| 编译期守护 | `base.Scope` 类型别名不变 + 新增 `var _ base.Scope = query.ScopeFor(...)` 编译期断言；CI 由 `go build ./...` 天然硬门禁 |

### 4.3 16 个 operations 裸 `s.db.WithContext(ctx)` 起链文件逐一归属

> 实证口径：`ops_grep` 精确匹配 `s\.db\.WithContext\(ctx\)`，排除 `*_test.go` 与 `*.backup` 后 **15 个业务文件（147 处）**；含 `pg_contract_test.go`（1 处）共 **16 个文件 / 148 处**，与任务书「16 个文件」一致。`excel_service.go.backup`（8 处，非编译态）与 `WithContext(ctx)` 更宽口径的 26 文件（子查询构造 / Session 派生 / `scope_write.go` 等）**不计入「裸起链」判定**，但纳入 T4 同源审计。
> **8 核心 service** = 8 个 `/statistics` 端点对应的 service（F-12），是 T3/T4 的重点。

| # | 文件 | 裸起链数 | 归属处置 | 原因 / 边界 |
|---:|---|---:|---|---|
| 1 | `asset_service.go` | 9 | **改 → ScopedDB**（8 核心） | `/statistics :1063` + 分页 `total`；`dept_id` 直挂（149 的 `applyAssetReadScope`） |
| 2 | `building_service.go` | 6 | **改 → ScopedDB**（8 核心） | `/statistics :799` + 分页 `total`；`applyBuildingScope` |
| 3 | `floor_service.go` | 21 | **改 → ScopedDB**（8 核心） | `/statistics :856` + 分页 `total`；`effectiveOrgExpr` 楼层覆盖链 |
| 4 | `workstation_service.go` | 10 | **改 → ScopedDB**（8 核心，**仅起链行**） | `/statistics :912` + `counts-by-floor :914` + `dept-options :911` + `workstations-all :919` + 分页 `total`；**150 的 6 字段删除 / 全库 `countQ` 删除 / `candidateQ` / `ToFill`/`ToCorrect` / `Samples` 一字不动** |
| 5 | `infopoint_service.go` | 7 | **改 → ScopedDB**（8 核心） | `/statistics :1106` + 分页 `total` |
| 6 | `server_room_service.go` | 4 | **改 → ScopedDB**（8 核心） | `/statistics :1147` + 分页 `total`；楼宇链 |
| 7 | `dedicated_line_service.go` | 4 | **改 → ScopedDB**（8 核心） | `/statistics :1199`；source/dest 双链 OR |
| 8 | `room_device_service.go` | 3 | **改 → ScopedDB**（8 核心） | `/statistics :1240`；两级链 |
| 9 | `building_service_typesafe.go` | 7 | **改 → ScopedDB**（旁路实现） | 不走 `GORMRepository` 的类型安全版（`:14-37` 自带 `db *gorm.DB`），同属 ops 读链；登记表填 `ops_buildings`，与 #2 共用谓词 |
| 10 | `excel_service.go` | 7 | **改 → 仅起链行** | 导入/导出起链；**150 的 `ImportResult.ScopeOutOfScope json:"-"` / `ImportError.Row` / `filterScopeWriteImportRows` / `result.ScopeOutOfScope = outOfScope` 一字不动**；`applyExcelWorkstationScope` 保留 |
| 11 | `excel_export_service.go` | 1 | **保留（禁改）** | **归 148 文件面**（`resolveAssociations` / `CAST(id AS TEXT)` 改写）；151 只在注册表登记导出路径，不改文件 |
| 12 | `floor_cache_impl.go` | 3 | **改 → 缓存 key 挂 `scopeCacheKey`** | `floor:tree` 固定 key 必须含 scope 维度（STATE `:79` / 149-SUMMARY `:115` 契约）；`CacheProvider.GetOrSet` 的 key 拼接 `query.ScopeCacheKey(ctx)` |
| 13 | `location_alias_service.go` | 7 | **保留（禁改）→ 注册表登记** | **149 已用 `dept_id ∈ 可见集` 直挂收口**；151 只让它在注册表登记后自动获得谓词，不重写 149 成果 |
| 14 | `room_photo_service.go` | 16 | **保留（禁改）→ 注册表登记** | **149 已 10 方法逐方法登记 `photoReadVisible` 等**；151 只登记「中间件挂了没接线」由机制兜住，不改 149 一行 |
| 15 | `workstation_device_service.go` | 42 | **保留（禁改）** | **归 148 文件面**（`DIALECT-02` CTE 改写 / Go 层合并）；151 只在注册表登记 workstation-device 归属链，不改文件 |
| 16 | `pg_contract_test.go` | 1 | **保留（纳入静态断言扫描 + 追加同源断言）** | 150 已写 `TestScopeWriteImport_StaticContract` / `TEST_PG_DSN` 门控；151 追加同源契约与「零裸起链」静态断言，`t.Skip` 可观测口径不变 |

> 另有 `scope_entities.go` / `scope_entities_scope149_test.go`（149）**不列入 `files_modified`**：用 151 新增的 `scope_registry.go` 做注册，避免触碰 149 成果与其测试契约；仅在 §T2「视需要」核对后保持零改动。`scope_write.go` / `useBackfillDept.tsx` / `workstation_handler.go` / `excel_handler.go`（150）**零改动**。

### 4.4 「All 兜底短路」与「system/vdi 零回归」验收方式

| 保证 | 验收方式（静态断言 + 全量测试，二者缺一不可） |
|---|---|
| **未注入 ctx → All 兜底** | ① **静态断言**：`scope_registry_test.go` / `scoped_db_test.go` 断言 `scope_filter.go:37-39 scopeIsAll` 的 `!ok \|\| scope.All` 短路分支仍在、`datascope.go:103-109` 的 `scope == DataScopeAll \|\| scope == 0` 短路未被改动；断言 `ScopedDB` 源码含 `FromContext` + `!ok \|\| All` 早返回。② **行为断言**：`TestScopedDB_NoInjection_AllFallback`（`context.Background()` 起链，`DryRun` SQL 与改造前逐字相等、零额外 `EXISTS`/`WHERE`、查询计数 0）、`TestScopedDB_AllTier_ZeroRegression`（`AllScope` 注入 → 同上）。③ **全量**：`go test ./...` 全部既有 `context.Background()` 直调单测（base 契约测试 / 132/149/150 矩阵）保持全绿 |
| **system/vdi 零回归** | ① **静态断言**：`scope_registry_test.go` 断言 `internal/services/system/**`、`pkg/**`、vdi 相关包对 `GORMRepository` / `ScopedDB` 的**直接消费数 = 0**（`go/parser` 扫描，与本次 planning 实证 0/0/0 对齐）；② **专项**：`go test -count=1 ./internal/services/system/... ./internal/api/v1/system/... ./pkg/middleware/... ./pkg/datascope/...` 全绿（vdi 相关包按实际路径追加）；③ **全量**：`go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `npx vitest run`（全量）全绿（§T6） |
| **写链零回归** | `TestScopedRepo_WriteChainUntouched`：`Create/Update/Delete/BatchDelete` 的 `DryRun` SQL 中**不含** `EXISTS`/`CAST(... AS TEXT) IN` 等 scope 谓词；`internal/services/base/*_test.go` 既有 7 处引用全绿 |

### 4.5 影响面实证命令（可复核）

```
ops_grep(pattern="GORMRepository", path="internal", outputMode="count")                 → 17 文件 / 56 处
ops_grep(pattern="GORMRepository", path="internal/services/system", ...)               → 0 文件
ops_grep(pattern="GORMRepository", path="pkg", ...)                                    → 0 文件
ops_grep(pattern="s\\.db\\.WithContext\\(ctx\\)", path="internal/services/operations",
         glob=["*.go","!*.backup"], outputMode="count")                                → 16 文件 / 148 处
ops_glob(pattern=["**/base/*.go"], path="internal/models")                            → 0（路径笔误实证）
```

---

## 5. Lead 裁决表（本 phase 必须写入 `151-SUMMARY.md`）

| # | 裁决 | 落地要求 |
|---|------|---------|
| D-1 | **A(150) + B(151) 都完成才算 ① 根治** | 本 phase 完成前任何产物不得写「① 已根治」；`151-SUMMARY.md` 完成后才写「**A(150)+B(151) 均完成，① 侧信道根治达成**」 |
| D-2 | **no 伪造「统一架构已落地」直到本 phase 完成** | 150-SUMMARY §4 已声明；151 在 T1~T6 全部通过前不得宣称；`go build ./...` + 全量矩阵全绿是硬门槛 |
| D-3 | **B 不再删响应字段** | 150 的 6 字段 / `countQ` / `json:"-"` / 越界静默跳过一字不动；151 只做起链机制 + 静态契约 |
| D-4 | **不推翻 149 成果** | 5 个 service / `scope_entities.go` / router `:1176` / `useFloorPlanEditor.ts` 零改动；只登记注册表 |
| D-5 | **禁区**（ROADMAP `:173-181`） | 禁 DB 视图 / PG RLS / 按用户分表；禁告警通道 / 粗粒度桶 / 频限代替根治；禁 `isPostgres` 方言分支；禁内联复制 `effectiveOrgExpr` / `datascope` 逻辑 |
| D-6 | **PG 双跑不伪造** | `TEST_PG_DSN` unset → `t.Skip` 可观测 + 静态断言；不建 `services: postgres`、不注入 DSN、不把 `t.Skip` 改 `t.Fatal` |
| D-7 | **router.go 只改挂载相关行** | 基于 **149 commit 之后的最新文件**；仅 1 注释 + 1 装配调用（±2 行），不碰 `:1176` 补挂行、不碰任何既有 `DataScopePermission` 挂载、不增删路由（`node scripts/check-route-registry.mjs` 零漂移） |
| D-8 | **F-7 路径笔误 / 485 口径** | 按 `internal/services/base/service.go:38` 执行；在 `151-SUMMARY.md` 登记偏差，**不改 ROADMAP 原文**（归 Lead 收档处理） |
| D-9 | **提交** | `commit:false`，**禁止 git add / commit / stash / checkout / reset / cherry-pick**；反向变异还原用 `edit` 手动改回；Lead 统一提交 |
| D-10 | **全仓基座 → 必须全量矩阵** | 触及 `GORMRepository` 基座，T6 **必须**跑全量 `go test ./...`（不同于 150 只跑相关包）；不启动长驻进程、不写生产数据 |

---

## 6. 任务（XML）

<tasks>

  <task id="T1" req="ARCH-01">
    <title>新建 pkg/query/scoped_db.go：ScopedDB + GORM Scopes 唯一查询起链点 + 内建 scopeCacheKey 维度</title>
    <files>
      pkg/query/scoped_db.go（新增）;
      pkg/query/scoped_registry.go（新增）;
      pkg/query/scoped_db_test.go（新增）;
      internal/services/operations/scope_filter.go（新增导出 ScopedCacheKey 薄封装）;
      internal/services/operations/scope_registry.go（新增，T2 共用）
    </files>
    <action>
      1. `pkg/query/scoped_registry.go` 定义（**零业务表名、零 SQL、零 ops import**）：
         - `type ScopeFactory func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB`；
         - `type Entry struct { Predicate ScopeFactory; CacheKey func(ctx context.Context) string }`；
         - `func Register(table string, e Entry)`（重复注册 panic 或返回 error，项目禁 panic → 返回 error 并由 `MustRegister` 包装；编译期白名单由 ops 侧常量保证）；
         - `func Lookup(table string) (Entry, bool)`；`func RegisteredTables() []string`（供静态断言测试枚举）。
         - **分层红线**：本文件只允许 import `context` / `gorm.io/gorm` / `pkg/datascope`；**不得 import `internal/**`**。
      2. `pkg/query/scoped_db.go` 定义 `func ScopedDB(ctx context.Context, db *gorm.DB) *gorm.DB`：
         - `r, ok := datascope.FromContext(ctx)`；**`if !ok || r.All { return db.WithContext(ctx) }`**（All / 未注入短路，原样返回，**零额外 SQL、零回归**）；
         - 失败闭合 `r.FailClosed` → 由 registry 的 Predicate 统一处理（`1=0`），本函数不另写 `1=0`（避免第二套实现）；
         - 非短路 → `db.WithContext(ctx).Scopes(func(tx *gorm.DB) *gorm.DB { table := resolveTable(tx); e, found := Lookup(table); if !found { return tx }; return e.Predicate(tx, r) })`；
         - `resolveTable(tx)`：优先 `tx.Statement.Table`，空则 `tx.Statement.Schema.Table`，再空则对 `tx.Statement.Model` 做 `interface{ TableName() string }` 断言（与 `service.go:83-89` 同款）；**取不到表名返回 `tx` 不放行也不加谓词**（由 T3 静态断言保证 ops 起链必有表名）。
         - 定义 `func ScopeCacheKey(ctx context.Context) string`：先取 `FromContext`；`!ok || All` → 直接返回 `"all"`（与 `scope_filter.go:59-62` 一致，此处为短路常量非算法复制）；否则委托注册表的 `CacheKey` 回调（由 ops 侧注册 `ScopedCacheKey`），**零算法复制**。
         - 定义 `func ScopeFor(ctx context.Context) base.Scope` 亦可（若需与 `List` 的 `scopes ...Scope` 衔接）：返回 `func(q *gorm.DB) *gorm.DB { return ScopedDBPending(q, ctx) }`；但**主路径以 `service.go` 起链替换为准**（T3），`ScopeFor` 仅作兼容备选，避免双起链。
      3. `pkg/query/scoped_db_test.go`（就近 `*_test.go`，SQLite 真库 + `DryRun`，零新依赖）：
         - `TestScopedDB_NoInjection_AllFallback`：`context.Background()` → SQL 与基线逐字相等、零 `EXISTS`、零额外 WHERE；
         - `TestScopedDB_AllTier_ZeroRegression`：`datascope.WithContext(ctx, datascope.AllScope("u1"))` → 同上；
         - `TestScopedDB_RegistryApplied`：注册一个测试用 factory → Dept 档注入后 SQL 含该谓词且 `CAST(... AS TEXT)` 双方言形态；
         - `TestScopedDB_NoDuplicatePredicate_CountAndFind`：同一链 `Count` + `Find` 各执行一次，断言谓词不重复叠加（GORM Scopes 每次执行各应用一次）；
         - `TestScopedDB_UnknownTable_NoPredicate`：未注册表 → 原样返回（不静默放行全库，由 T3 静态断言兜住「ops 起链必登记」）；
         - **静态断言**：`scoped_db.go` 源码含 `datascope.FromContext` 与 `!ok || r.All` 短路；`scoped_registry.go` 源码**不含** `internal/services/` import（防分层被突破）。
      4. `scope_filter.go`（151 独占）**新增导出薄封装**：`func ScopedCacheKey(ctx context.Context) string { return scopeCacheKey(ctx) }`，注释写明「唯一算法实现仍是本文件 `:53-74 scopeCacheKey`；`pkg/query` 经注册表回调引用，禁止复制」。**既有 `scopeIsAll` / `scopeCacheKey` / `orgScopeClause` / `wrapScopeExists` 同名同签名一字不动**（131/149 存量调用零改）。
      5. **禁止**：不复制 `effectiveOrgExpr` 的 COALESCE/NULLIF 表达式、不复制 `datascope` 档位判定、不复制归属链 SQL；不改 `pkg/datascope/datascope.go`（不在本 phase 文件面）；不新增中间件（`DataScopePermission` 已注入 ctx，F-13）。
    </action>
    <acceptance>
      `go build ./pkg/query/ ./internal/services/operations/` 通过；6 个 `TestScopedDB*` 全绿；`ScopedDB` 未注入 / All 档 SQL 零漂移；`pkg/query` 零 `internal/**` import（静态断言）。
    </acceptance>
  </task>

  <task id="T2" req="ARCH-01">
    <title>资源归属注册表：把 scope_entities.go（149）与 scope_filter.go 的实体→归属链映射收敛为统一表</title>
    <files>
      internal/services/operations/scope_registry.go（新增）;
      internal/services/operations/scope_registry_test.go（新增）;
      internal/services/operations/scope_filter.go（仅强制层入口，与 T1 同文件）
    </files>
    <action>
      1. `scope_registry.go` 在 ops 包内定义 `func SetupScopedDBRegistry() error`（供 T5 `router.go` 装配调用），用 `query.MustRegister(table, query.Entry{Predicate: ..., CacheKey: ScopedCacheKey})` 逐项登记。**谓词一律引用既有单一真实源，禁止内联复制**：
         - `building/floor/workstation/asset/...`（直挂 `org_id` 或有效机构）→ 引用 `effectiveOrgExpr("f.org_id","b.org_id")`（`dept_filter.go`，经 `scope_filter.go` 的 `orgScopeClause` / `applyOrgScope` / `applyBuildingScope`）；
         - `wall/door/floor-plan-text/room-photo/server-room/room-device/dedicated-line/workstation-device` → 引用 149 的 `entityExistsSub` 归属链 EXISTS 骨架 + `wrapScopeExists`（`scope_filter.go:231`）+ `entityFloorOrgExpr()` / `entityBuildingOrgExpr` / `entityWorkstationOrgExpr()`（`scope_entities.go:71-91`）；
         - `location-alias` → `dept_id` 直挂（复用 149 的 `applyEntityReadScope` / `applyAssetReadScope` 分支）；
         - `workstation-device` / `excel` 导出路径：仅登记，不改 148 文件（§4.3 #11/#15）。
      2. **登记表覆盖清单**（表名 → 归属形态，逐项可枚举）：
         `ops_buildings` / `ops_floors` / `sys_workstation` / `ops_asset` / `ops_info_points` / `ops_server_rooms` / `ops_dedicated_lines` / `ops_room_devices` / `ops_workstation_device` / `ops_walls` / `ops_doors` / `ops_floor_plan_texts` / `ops_room_photos` / `sys_dept_location_alias` / `ops_buildings`（typesafe 旁路同表共用）。
         **注意**：`Register` 若同表重复注册 → `SetupScopedDBRegistry` 返回 error；typesafe 旁路共用 `ops_buildings` 条目，**不重复注册**。
      3. **不动 149 的 5 个 service 收口成果**：`wall_service.go` / `door_service.go` / `floor_plan_text_service.go` / `room_photo_service.go` / `location_alias_service.go` / `scope_entities.go` / `scope_entities_scope149_test.go` **零改动**；它们「在注册表登记后自动获得可见集谓词」由 `ScopedDB` 起链机制实现（T3 基座替换 + 8 核心起链替换），**不靠改这 5 个文件**。
      4. `scope_filter.go` 提升为「**不可绕过的查询层**」：新增 `func ScopedQuery(ctx context.Context, db *gorm.DB) *gorm.DB { return query.ScopedDB(ctx, db) }` 作为 ops 包内统一 helper（内部任务 / 写侧前置校验可复用），并在文件头契约注释补「本层由 `pkg/query.ScopedDB` 在起链时强制调用，`wrapScopeExists` 不再是唯一可选入口」。**`wrapScopeExists` 保持同名同签名**。
      5. `scope_registry_test.go`：
         - `TestScopeRegistry_CoversAllEntities`：`RegisteredTables()` 与上表逐项比对，缺一即红；
         - `TestScopeRegistry_SingleSourceOfTruth`：静态断言 `scope_registry.go` **不含** `COALESCE(` / `NULLIF(` / `1=0` 内联、不含 `::uuid` / `ILIKE` / `isPostgres`；必须出现 `effectiveOrgExpr` / `entityExistsSub` / `wrapScopeExists` / `dept_id` 的引用 token；
         - `TestScopeRegistry_NoBareChain`（静态断言，与 T3 共用）：扫描 `internal/services/operations/*.go`（排除 `*_test.go` / `*.backup`），断言 `s.db.WithContext(ctx)` 作为**起链点**的出现数 = **0**（改造后口径；未改造的 148/149/150 禁改文件按 §4.3 白名单排除并逐项登记原因）；
         - `TestScopeRegistry_149FilesUntouched`：静态读取 5 个 149 service 的 `entityReadScope` / `photoReadVisible` 等 token 仍在（防误改 149 成果）；
         - `TestScopeRegistry_DualDialect`：谓词 SQL 含 `CAST(... AS TEXT)`、不含 `::` / `ILIKE` / `ON CONFLICT`（沿用 149 `TestScope149_Static_DualDialectAssertions` 手法）。
      6. **禁止**：不把 149 的 `applyEntityReadScope` 改成调用 ScopedDB（149 成果不推翻）；不改 `scope_entities.go` 常量；不新增第 5 套归属链实现。
    </action>
    <acceptance>
      `SetupScopedDBRegistry()` 返回 nil 且覆盖全部登记项；6 个 `TestScopeRegistry*` 全绿；149 的 5 个 service 与 `scope_entities.go` `git diff` 为零（只读核对）；单一真实源静态断言通过。
    </acceptance>
  </task>

  <task id="T3" req="ARCH-01">
    <title>8 个 operations service 起链改走 ScopedDB；base.GORMRepository 读链最小化改造；scope_filter.go 提升为强制层</title>
    <files>
      internal/services/base/service.go;
      internal/services/base/scoped_repo_test.go（新增）;
      internal/services/operations/{asset,building,floor,workstation,infopoint,server_room,dedicated_line,room_device}_service.go;
      internal/services/operations/building_service_typesafe.go;
      internal/services/operations/excel_service.go;
      internal/services/operations/floor_cache_impl.go;
      internal/services/operations/scope_filter.go
    </files>
    <action>
      1. **`base/service.go` 仅 2 行**：`:74 GetByID` 与 `:114 List` 的 `r.db.WithContext(ctx).Model(new(T))` → `query.ScopedDB(ctx, r.db).Model(new(T))`；新增 import `github.com/xingran-next/xingran-go-backend/pkg/query`。**`Create :49` / `Update :54` / `Delete :59` / `BatchDelete :148` 一字不动**（§4.2 写链禁改）。`Scope` 类型别名 / `PageParams` / `PageResult` / `SortScope` / `WrapError` 全部不动。
      2. **8 核心 service 起链处**：逐文件把「读请求起链」的 `s.db.WithContext(ctx)` 改为 `query.ScopedDB(ctx, s.db)` 或 ops 统一 helper `ScopedQuery(ctx, s.db)`（T2）。**只改起链行**，其余：
         - `workstation_service.go`：**150 的响应字段逻辑零改动** —— `BackfillDeptResult` 6 字段删除状态、全库 `countQ` 删除状态、`candidateQ` / `result.Candidates` / `ToFill` / `ToCorrect` / `Samples` 非 All 恒空逻辑一字不动；
         - `excel_service.go`：**150 的 `ImportResult.ScopeOutOfScope json:"-"` / `ImportError.Row` / `filterScopeWriteImportRows` 调用与 `:460` 赋值 / 越界静默跳过一字不动**；
         - **不改 148 的 `excel_export_service.go` / `workstation_device_service.go` CTE 改写**（§4.3 #11/#15）；
         - **不改 149 的 5 个 service**（§4.3 #13/#14）。
      3. `building_service_typesafe.go`：7 处裸起链改走 `ScopedQuery`（其 `db *gorm.DB` 自带，登记表 `ops_buildings`）。
      4. `floor_cache_impl.go`：`CacheProvider.GetOrSet` 的 key 拼接 `query.ScopeCacheKey(ctx)` 维度段（`:39-45 floor:tree` 固定 key）；**不改缓存值形状 / TTL / 失效语义**。
      5. `scope_filter.go`：完成 T1 的 `ScopedCacheKey` 薄封装 + T2 的 `ScopedQuery` 强制层入口；**不改 `scopeIsAll :37-39` 短路**（零回归硬约束）。
      6. **静态强制（新增源码静态断言测试，写进 `scope_registry_test.go`）**：
         - `TestScopedLayer_ZeroBareChain`：对 `internal/services/operations/*.go`（排除白名单：`excel_export_service.go`〔148〕、`workstation_device_service.go`〔148〕、`location_alias_service.go`〔149〕、`room_photo_service.go`〔149〕、`pg_contract_test.go`〔测试〕、`*.backup`〔非编译〕）断言 **`s.db.WithContext(ctx)` 起链出现数 = 0**；白名单逐项含「保留原因」注释，缺注释即红；
         - `TestScopedLayer_AllFallbackIntact`：断言 `scope_filter.go:37-39` 与 `datascope.go:103-109` 的短路 token 仍在；
         - `TestScopedLayer_BaseReadChainUsesScopedDB`：断言 `service.go` 的 `List` / `GetByID` 含 `query.ScopedDB`，`Create/Update/Delete/BatchDelete` **不含** `ScopedDB`；
         - `TestScopedLayer_WriteChainUntouched`（写进 `base/scoped_repo_test.go`）：`Create/Update/Delete/BatchDelete` 的 `DryRun` SQL 不含 scope 谓词；
         - **反向变异守护**（T6 执行）：临时去掉一个 `ScopedDB` 起链 → 上述静态断言**必红** → 用 `edit` 还原 → **必绿**。
      7. `base/scoped_repo_test.go`：`TestScopedRepo_CountAndFindHomologous`（Dept 档注入 → `PageResult.Total == len(List)`）、`TestScopedRepo_AllFallback`（`context.Background()` → 既有 base 契约测试结果逐字不变）、`TestScopedRepo_ScopesStillApplied`（`SortScope` / 变参 scopes 与 `ScopedDB` 叠加后排序不变）。
      8. **禁止**：不改 `GORMRepository` 构造签名（§4.2）；不改 25 个 OperType 常量、不删 `operlog.Record`；不改 8 个 service 的响应字段 / 排序 / 分页结构；构建产物不落根目录。
    </action>
    <acceptance>
      `go build ./...` 全量通过；零裸起链静态断言全绿；`PageResult.Total == len(List)`；写链 SQL 零 scope 谓词；All 短路专项全绿；反向变异「红→还原→绿」证据完整（T6 贴入 SUMMARY）。
    </acceptance>
  </task>

  <task id="T4" req="SIDECHANNEL-04">
    <title>所有 COUNT(*)/SUM 与明细同源：8 个 /statistics + 分页 total + counts-by-floor + dept-options + workstations-all + PG/SQLite 双跑断言</title>
    <files>
      internal/services/operations/sidechannel_homology_test.go（新增）;
      internal/services/operations/pg_contract_test.go（追加，不改既有静态断言 token）;
      internal/services/operations/scope_registry_test.go（追加同源静态断言）
    </files>
    <action>
      1. `sidechannel_homology_test.go`（SQLite 真跑 + `DryRun` 静态 + 零新依赖）：对 **8 个 `/statistics`**（`building/floor/workstation/asset/infoPoint/serverRoom/dedicatedLine/roomDevice`）、**分页 `total`**、**`counts-by-floor`（`router.go:914`）**、**`dept-options`（`:911`）**、**`workstations-all`（`:919`）** 逐端点构造「全库 N 行 / 可见 M 行」夹具（N>M），以同一 `datascope.ScopeResult`（Dept / DeptChild 两相邻档）注入 ctx，断言：
         - **同源恒等式**：每个端点的 `COUNT(*)` / `SUM` 结果与 `List`/明细行数**相等**，且都 == M；
         - **差分不变式**：改变 N 保持 M 不变 → 所有数值字段（`total` / `count` / `sum` / 数组长度 / 状态码）**逐字节相等**（复用 150 `sidechannel_diff_attack_test.go` 的两相邻 scope 思路）；
         - **无全库链**：`DryRun` SQL 中不存在不带可见集谓词的 `COUNT(*)` / `SUM(` 子查询（静态 + 行为双断言）。
      2. `pg_contract_test.go` **追加** `TestPGContract_ScopedDB_Homology`（**不改** 150 已落的 `TestScopeWriteImport_StaticContract` 的 `filterScopeWriteImportRows(` / `result.ScopeOutOfScope = outOfScope` 两 token，不改 `TestPGContract_ScopeWriteImportResolve` 的 `t.Skip` 行为）：
         - `TEST_PG_DSN` **unset** → `t.Skip("TEST_PG_DSN 未设置：PG 双跑为 DIALECT-04③ 前置依赖，skip 可观测")`（**可观测、不伪造**）；
         - DSN 存在时才连真 PG 跑同一 `ScopeResult` 下 COUNT 与 List 行数一致；
         - **同时恒跑静态断言**（不依赖 DSN）：`workstation_service.go` 不含 `COUNT(*) AS total` 全库口径 token（防 150 D-3 复活）、8 个 service 起链含 `ScopedDB`/`ScopedQuery`、`scope_filter.go` 短路 token 存在。
      3. `scope_registry_test.go` 追加 `TestHomology_StaticChainContract`：断言 `service.go:113-124` 的 `Count(&total)` 位于 `for _, s := range scopes` **之后**（scopes 在 Count 之前应用 = 同源结构性保证，F-11）。
      4. **operlog 审计链不降级**：断言 `ScopeWriteOperParam` 的 `requested/visible/outOfScope` 三字段仍在（`scope_write.go:730-738` 零改动）、`excel_handler.go` 的 `operlog.Record` 仍含 `outOfScope`、25 个 OperType 常量锁 `internal/utils/operlog/regression_test.go` 不红；`workstation_handler.go` 的 OperParam 格式串（150 D-4）一字不动。
      5. **禁止**：不伪造 PG 双跑（不建 `services: postgres`、不注入 DSN、不把 `t.Skip` 改 `t.Fatal`）；不改 150 的差分攻击测试；不写生产数据。
    </action>
    <acceptance>
      13 类端点（8 statistics + total + counts-by-floor + dept-options + workstations-all）同源恒等式与差分不变式全绿；`TestPGContract_*` 在 DSN unset 时 `t.Skip` 可观测且静态断言恒跑；operlog 审计链断言全绿。
    </acceptance>
  </task>

  <task id="T5" req="SIDECHANNEL-04 / ARCH-01">
    <title>响应契约层静态断言（防复活）+ router.go 只改挂载相关行</title>
    <files>
      internal/services/operations/response_contract_test.go（新增）;
      internal/api/router.go（仅挂载相关行）
    </files>
    <action>
      1. `response_contract_test.go`（**复用 150 的差分攻击测试思路**，全部静态断言恒跑，零 DSN 依赖）：
         - `TestResponseContract_NoDerivedScale`：扫描 `internal/api/v1/operations/**/*_handler.go` 与 `internal/services/operations/**` 的响应构造点（`response.Success` / `gin.H` / `PageResult` / `ImportResult` / `BackfillDeptResult`），断言**任何端点响应体不含「全库规模 − 可见规模」派生数值**：6 个被删字段名 0 命中、`ScopeOutOfScope` 保持 `json:"-"`、`result.Errors = append` 越界分支不复活、不含 `全库` / `总数-可见` / `总数 - 可见` / `outOfScope` 的响应体键；
         - `TestResponseContract_NoRevival`（防复活断言）：把 150 的 5 个 `TestSideChannel*` 与 `TestScopeWriteImport_StaticContract` 以**测试名清单**静态登记，缺一即红（防执行期误删）；断言 `sidechannel_diff_attack_test.go` 存在且含 `OnlyVariesWithM` / `CannotEnumerateN` token；
         - `TestResponseContract_OperlogChain`：断言 `ScopeWriteOperParam` 三字段、`excel_handler.go` 的 `outOfScope` operlog 参数、`workstation_handler.go` 的 OperParam 格式串仍在；
         - **差分攻击复跑**：复用 150 夹具，Dept / DeptChild 两相邻 scope 各请求一次，断言响应体所有数值字段仅随 M 变化、不随 N−M 变化（N=5/M=2 vs N=9/M=2 逐字节相等）。
      2. `internal/api/router.go`：**基于 149 commit 之后的最新文件**，定位 `ops :=` 组定义行（约 `:780` 前）与 149 的 `:1176 roomPhotos` 补挂行；**仅新增 1 条注释 + 1 条装配调用**（±2 行），例如在 ops 组初始化后紧邻处：
         ```go
         // Phase 151 / ARCH-01：ScopedDB 资源归属注册表装配（唯一查询起链点由 pkg/query 承载）。
         if err := operations.SetupScopedDBRegistry(); err != nil { panic(err) } // 项目禁 panic → 改用 logrus.Fatalf 或返回 error
         ```
         **定案用无 panic 形态**：`router.go` 的 `Setup...` 系列函数返回 error 时按现有惯例处理；若当前 `SetupRouter` 无 error 返回，则改为 `operations.MustSetupScopedDBRegistry()`（内部 `logrus.Fatalf`，与项目既有启动期失败处理一致）—— 由执行期复核 `router.go` 签名后二选一，**不得新增中间件、不得增删路由、不得改 `:1176` 补挂行、不得改任何既有 `DataScopePermission` 挂载**。
         - 静态契约测试 `TestRouter151_ScopedDBMountContract`：断言装配调用恰好 1 处、位于 ops 组定义之后、roomPhotos 补挂行**仍存在且未改**（与 149 `TestScope149_Router_RoomPhotosMiddlewareContract` 同思路）；
         - `node scripts/check-route-registry.mjs` 零漂移（`OK 路由 713 条 / 写路由 315 条 / deferred 138 条` 基线，按 149 收口后实际值核对）。
      3. **禁止**：不在 `router.go` 写业务 SQL、不加 `middleware.DataScopePermission` 重复挂载、不碰 148 的 `ci.yml` / `.golangci.yml`。
    </action>
    <acceptance>
      4 个 `TestResponseContract*` 全绿；`router.go` diff = +2 行（1 注释 + 1 装配调用）；roomPhotos 补挂行零改动；路由注册零漂移；防复活断言覆盖 150 全部 5 个测试 + 静态契约测试。
    </acceptance>
  </task>

  <task id="T6" req="验证">
    <title>自验证：全量矩阵（触及全仓基座）+ system/vdi 零回归专项 + All 兜底短路专项 + 反向变异红绿</title>
    <files>
      （无新增生产文件；验证输出写入 151-SUMMARY.md）
    </files>
    <action>
      > **本 phase 触及 `GORMRepository` 全仓基座，必须跑全量矩阵**（D-10）。不同于 150「只跑相关包」。所有命令**前台依次执行**，不并发跑同一 `go test`；**不启动长驻进程 / dev server / DB**；构建产物不落根目录。

      1. **后端全量**（必跑）：
         ```
         go build ./...
         go test -count=1 ./...
         go test -count=1 ./internal/services/base/ ./pkg/query/ ./internal/services/operations/ ./internal/api/v1/operations/
         go test -count=1 ./internal/services/system/... ./internal/api/v1/system/... ./pkg/middleware/... ./pkg/datascope/...
         ```
         （vdi 相关包按实际路径追加；`go test ./...` 若命中 `scripts/e2e/TestE2EAllEndpointsLogged` 的 Windows/bash 路径拼接环境既有红灯，按 150-PLAN `:240` / 132-SUMMARY.md:87 先例**如实登记为既有环境红灯**，不擅自改其它 phase 文件、不伪造全绿。）
      2. **前端全量**（必跑，在 `xingran-react-frontend/` 下）：
         ```
         npm run type-check
         npm run lint:ci
         npx vitest run
         ```
         （**全量 vitest**，非定向单文件；本 phase 不改前端文件，用于守护零回归。）
      3. **专项**（必跑）：
         ```
         go test -count=1 -run 'TestScopedDB|TestScopedRepo|TestScopeRegistry|TestScopedLayer|TestSideChannel|TestResponseContract|TestHomology|TestPGContract|TestRouter151' -v ./pkg/query/ ./internal/services/base/ ./internal/services/operations/
         ```
         - **system/vdi 零回归专项**：上述第 1 组第 4 条命令 + §4.4 静态断言「直接消费数 = 0」；
         - **「未注入 ctx → All 兜底」短路专项**：`TestScopedDB_NoInjection_AllFallback` + `TestScopedRepo_AllFallback` + `TestScopedLayer_AllFallbackIntact`；
         - `node scripts/check-route-registry.mjs`（路由零漂移）。
      4. **反向变异（必做，红 → 还原 → 绿）**：临时用 `edit` 把**一个** `ScopedDB` 起链（建议 `building_service.go` 一处）改回 `s.db.WithContext(ctx)`，重跑 `TestScopedLayer_ZeroBareChain` / `TestScopedLayer_BaseReadChainUsesScopedDB` / `TestSideChannel*` 同源断言 → **必须红**；随后**立即用 `edit` 手动还原**（**严禁 `git checkout`**）→ 重跑 → **必须绿**。把「红 → 还原 → 绿」三段真实输出原样贴进 `151-SUMMARY.md`。
      5. **不伪造**：`TEST_PG_DSN` unset → `t.Skip` 可观测，SUMMARY 明写「本期零真实 PG 双跑」；不勾 79 条跟踪表；不写生产数据。
    </action>
    <acceptance>
      全量命令输出齐全且全绿（PG contract 的 `t.Skip` 与已登记的既有环境红灯属预期、如实披露）；三项专项全绿；反向变异「红→还原→绿」证据完整；`151-SUMMARY.md` 记录每条命令的真实输出结论。
    </acceptance>
  </task>

  <task id="T7" req="SUMMARY">
    <title>写 151-SUMMARY.md（改动 / 2 REQ 满足 / 根治声明 / 验证结果 / 全仓基座影响面结论 / Known Gaps / 未 git commit）</title>
    <files>
      .planning/phases/151-scopeddb-b-sidechannel-04-arch-01/151-SUMMARY.md（新增）
    </files>
    <action>
      新建 `151-SUMMARY.md`，必含 7 段：
      1. **改动清单**：按 T1~T6 分组逐文件列出改了什么（含 `pkg/query` 两新增、`base/service.go` 仅 2 行、8 核心起链、注册表、静态断言测试、`router.go` +2 行）；**逐条列「未触碰文件面」**（148 的 `excel_export_service.go`/`workstation_device_service.go`、149 的 5 个 service/`scope_entities.go`/`useFloorPlanEditor.ts`、150 的响应字段逻辑、W1 全部、`.golangci.yml`/`ci.yml`、79 条跟踪表、LOCKED 裁定原文、`.planning/milestones/**`）。
      2. **2 REQ 满足情况**：`SIDECHANNEL-04` / `ARCH-01` 逐条对照 ROADMAP `:106-111` 五条成功标准，逐条给「满足 + 证据（测试名 / 命令 / 断言）」。
      3. **根治声明（加粗）**：「**A(150)+B(151) 均完成，① 侧信道根治达成**」（**仅在本 phase 全部任务与 must_haves 通过后写**）；并写明「② 由靠约定升级为靠机制」「**未伪造『统一架构已落地』直到本 phase 完成**」；明确 B 的职责是防回归 + 机制化、**未重复删响应字段**。
      4. **验证结果**：贴 T6 全量矩阵输出结论 + 三个专项 + 反向变异「红 → 还原 → 绿」三段输出；写明 `TEST_PG_DSN` 未设置、PG contract `t.Skip` 可观测、**本期零真实 PG 双跑、不伪造**。
      5. **全仓基座影响面结论**：照抄 §4.1 实证（**17 文件 / 56 处 / 2 个包**，`system`/`vdi`/`pkg` **0 文件直接消费**）、§4.2 注入方式定案（不动构造签名 / 只改 `List`/`GetByID` / 写链禁改）、§4.3 16 文件归属表、§4.4 两项零回归验收；**登记 F-7 路径笔误与「485 文件」为编译面口径**（D-8，不改 ROADMAP 原文）。
      6. **Known Gaps（照抄 ROADMAP `:159`/`:198`/`:200` 与 STATE `:325` 口径）**：
         - 「**PG 双跑前置依赖**：`TEST_PG_DSN` unset、`ci.yml` 无 `services: postgres`，需新建 PG service + 迁移引导后方可升级（DIALECT-04③）」；
         - 「**FUTURE-01 收档清理 4 处引用由 Lead 执行**：`.planning/REQUIREMENTS.md`(v1.54) / `ROADMAP.md:91` / `STATE.md` / `.planning/notes/261108-v153-locked-residuals.md:63`，本期不动」；
         - 「**132/133 UAT 重跑窗口待 QA 裁定**：见 `149-UAT-RECHECK.md` / `150-UAT-BACKFILL.md`，本期不擅自勾选、79 条跟踪表保持 0 勾选、不写生产数据」；
         - 承接 W1 `:324` / W2 `:325` 的其余登记项（portwrite operlog 缺口、forbidigo 能力边界、handler 固定状态码形态等），**不重复关闭、不伪造闭环**。
      7. **明确「未 git commit」**：写明「**本 phase 未执行 `git add` / `git commit` / `git stash` / `git checkout` / `git reset` / `git cherry-pick` 等任何 git 写操作**；由 Lead 统一提交（frontmatter `commit:false`）」。
      8. **禁止**在 T1~T6 未全绿时写「① 已根治 / 统一架构已落地 / PG 双跑通过 / UAT 已重跑完成」等不实闭环。
    </action>
    <acceptance>
      `151-SUMMARY.md` 7 段齐全；含加粗的「A+B 均完成，① 侧信道根治达成」与「未 git commit」声明；三项 Known Gaps 逐条登记；无任何伪造闭环措辞。
    </acceptance>
  </task>

</tasks>

---

## 7. 验证标准（must_haves，逐条对应 ROADMAP `:106-111` 五条 + 补充硬约束）

| # | must_have | 对应 ROADMAP 成功标准 | 可观察证据 |
|---|-----------|---------------------|-----------|
| **MH-1** | **COUNT/SUM 与明细同源**：8 个 `/statistics`、分页 `total`、`counts-by-floor`、 `dept-options`、`workstations-all` 全部满足 `COUNT == len(List) == M`；`pg_contract_test.go` + SQLite 双跑对同一 `ScopeResult` 断言一致；`TEST_PG_DSN` unset 时 `t.Skip` 可观测 + 静态断言恒跑，**不伪造双跑** | **:107（成功标准 1 / SIDECHANNEL-04）** | T4 四类测试 + T6 专项；`TestPGContract_ScopedDB_Homology` 的 skip 输出；静态「无全库 `COUNT(*) AS total`」断言 |
| **MH-2** | **零裸起链 + 注册表机制化**：service 起链处静态断言「零裸 `s.db.WithContext(ctx)` 起链」，全部经 ScopedDB；② 的 5 类端点（walls/doors/texts/photos/location-alias）在注册表登记后自动获得可见集谓词，rooms/photos「中间件挂了没接线」由机制兜住 | **:108（成功标准 2 / ARCH-01）** | T2 `TestScopeRegistry_CoversAllEntities` + T3 `TestScopedLayer_ZeroBareChain`；`scope_entities.go` 与 5 个 149 service `git diff` 为零；`router.go:1176` 补挂行未改 |
| **MH-3** | **All 兜底短路 + system/vdi 零回归 + 全量矩阵全绿**：保留 `scope_filter.go:37-39` / `datascope.go:6` 短路；`GORMRepository` 改造后全量 `go build ./...` + `go test ./...` + `tsc` + `lint:ci`（含 forbidigo）+ `npx vitest run`（全量）全绿 | **:109（成功标准 3）** | T6 全量命令输出；`TestScopedDB_NoInjection_AllFallback` / `TestScopedRepo_AllFallback`；静态断言「system/vdi/pkg 直接消费数 = 0」；写链 `TestScopedRepo_WriteChainUntouched` |
| **MH-4** | **响应契约层 + operlog 不降级**：任何端点响应体不含「全库规模 − 可见规模」派生数值；防复活断言覆盖 150 的 5 个差分攻击测试与静态契约测试；`ScopeWriteOperParam` 三字段 / Excel 导入 `outOfScope` operlog / 25 个 OperType 常量锁不红 | **:110（成功标准 4）** | T5 `TestResponseContract_*` 四个测试；`internal/utils/operlog/regression_test.go` 未红；150 文件 diff 核对表 |
| **MH-5** | **不可行方向勿立项**：DB 视图 / PG RLS / 按用户分表零代码；`EXISTS (?) + CAST AS TEXT` 双方言；禁 `::uuid` / `ILIKE` / `isPostgres` / 裸 `ON CONFLICT` | **:111（成功标准 5）** | T2 `TestScopeRegistry_DualDialect` / `TestScopeRegistry_SingleSourceOfTruth`；`git diff` 无 migration / 无视图 / 无 `SET LOCAL`；SUMMARY 显式声明 |
| **MH-6** | **「未注入 ctx → All 兜底」短路零回归**（硬约束） | ROADMAP `:105`/`:109`、REQUIREMENTS `:82`/`:104` | §4.4 静态 + 行为 + 全量三重验收；`context.Background()` 直调单测全绿 |
| **MH-7** | **system/vdi 零回归**（硬约束） | ROADMAP `:109`、REQUIREMENTS `:82` | §4.4 实证 0 直接消费 + 专项 `go test` + 全量矩阵 |
| **MH-8** | **A+B 完成**：B 未落地前不宣称 ① 已根治；本 phase 全绿后 `151-SUMMARY.md` 明确「A(150)+B(151) 均完成，① 侧信道根治达成」；B **不重复删响应字段**、**不推翻 149 成果** | ROADMAP `:91`/`:103`/`:139`、150-SUMMARY §4 | T7 第 3 段加粗声明 + 第 1 段未触碰文件面表；T5 防复活断言 |
| **MH-9** | **不伪造闭环**：不伪造 PG 双跑、不勾 79 条跟踪表、不写生产数据、`commit:false` 未执行任何 git 写操作、FUTURE-01 4 处引用与 132/133 UAT 重跑窗口由 Lead/QA 处理 | ROADMAP `:159`/`:180`/`:198`/`:200`、STATE `:325` | T7 第 6 段 Known Gaps + 第 7 段「未 git commit」；T6 反向变异用 `edit` 还原（非 `git checkout`） |

---

## 8. 禁止清单（违反即 PLANNING BLOCKED / 执行判红）

1. **git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git reset` / `cherry-pick` / 任何写 `.git` 的动作（`commit:false`，Lead 统一提交）。反向变异还原**用 `edit` 工具手动改回**。
2. **改其它 phase 无关文件**：
   - **W1**：145（`internal/utils/operlog/**`、`scripts/e2e`、`ci.yml`、146（`internal/core/security/password.go`）、147（`internal/services/asset/**`）全部零改动；
   - **148**：`excel_export_config.go` / `excel_export_service.go` / `workstation_device_service.go` / 4+ 测试 / **`.golangci.yml`**；
   - **149**：5 个 `wall/door/floor_plan_text/room_photo/location_alias` service、`scope_entities.go`、`scope_entities_scope149_test.go`、`useFloorPlanEditor.ts`、`133-UAT.md`、`router.go:1176` 补挂行；
   - **150**：`workstation_service.go` / `excel_service.go` / `scope_write.go` 的**响应字段逻辑**、`useBackfillDept.tsx`、`workstation_handler.go`、`excel_handler.go`、`sidechannel_diff_attack_test.go`、132/133 backfill UAT；
   - `.planning/milestones/**`、`.planning/notes/261108-*`（LOCKED 裁定原文）、**79 条人工验证跟踪表保持 0 勾选**。
3. **DB 视图 / PG RLS / 按用户分表**（ROADMAP `:111`、REQUIREMENTS `:83`、根因报告 `:160`）：不得立项、不得写 migration / 视图 / `SET LOCAL` / 分表。
4. **破坏单一真实源**：禁止内联复制 `effectiveOrgExpr` 的 COALESCE/NULLIF 表达式、`pkg/datascope` 档位判定、`scope_entities.go` 归属链 SQL 骨架、`scopeCacheKey` 算法；一律引用 / 回调注册。
5. **伪造 PG 双跑**：不建 `services: postgres`、不注入 / 伪造 `TEST_PG_DSN`、不把 `t.Skip` 改 `t.Fatal`；不勾 UAT 完成、不写生产数据。
6. **启动长驻进程**（dev server / watcher / DB / 后台 `&`）；构建产物只落 `out/`；不新增 `.xlsx` 夹具落盘；Go 单测就近 `*_test.go`，不建 `test/`（单数）。
7. **不新增中间件重复挂载**：`DataScopePermission` 已注入 ctx（`permission.go:497-541`），`router.go` 只做注册表**装配调用**（±2 行），不新增 `Use(...)`、不增删路由。
8. **不动 `GORMRepository` 构造签名**、**不改 4 个写链方法**（写链挂读谓词会破坏 UPDATE/DELETE 语义）；不改 25 个 operlog 常量、不删 `operlog.Record`。
9. **不改 ROADMAP / REQUIREMENTS / STATE 原文**（F-7 笔误只在 `151-SUMMARY.md` 登记，归 Lead 收档处理）。

---

## 9. 完成条件与交付

- 全部 **7 个任务（T1~T7）**完成、`must_haves` **MH-1~MH-9** 逐条有证据、T6 全量矩阵 + 三项专项 + 反向变异红绿齐全 → 返回：
  `## PLANNING COMPLETE (Phase 151)` + 任务数（7）/ 全仓基座影响面结论（消费方数量与零回归策略）/ 验证命令 / 文件面一句话。
- 若出现下列任一情形 → 返回 `## PLANNING BLOCKED` + 具体冲突：
  - 根因报告 / ROADMAP 与代码现状存在**语义冲突**（非 F-7 已登记的笔误/口径偏差）；
  - 151 与 148/149/150 出现跨 phase 写交集（逐格核对 ROADMAP `:119-148`）；
  - `pkg/query` 被迫 import `internal/**`（分层被突破，回调注册制失效）；
  - 任一禁止项无法回避。

**本 plan 落盘即视为 Phase 151 planning 产物；执行由 `/gsd-execute-phase 151` 在 W1/W2 全部交付后单独执行（必须最后），Lead 汇总提交。**
