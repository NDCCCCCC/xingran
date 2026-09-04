# Phase 91: CRUD 复用 base.Repository[T] (🔥 高优 P1) - Context

**Gathered:** 2026-09-04
**Status:** Ready for planning

<domain>
## Phase Boundary

让 `internal/services/operations/` 下 **11 个**同构 CRUD services（workstation / building / floor / asset / server_room / infopoint / dedicated_line / room_device / door / **wall** / **floor_plan_text**）复用 `internal/services/base` 包的泛型 `GORMRepository[T]` 抽象（改造为 gorm scope 函数式），消除每服务重复的 CRUD 模板代码（直调 GORM 的 Create/Update/Delete/GetByID/BatchDelete + countRecords/fetchRecords + buildListQueryFromRequest 样板），统一 `PageResult` 单一定义；`go test ./internal/services/operations/...` 0 失败 + handler 端到端 smoke 通过 + 前端 JSON 契约零变更。

**关键现实校准（讨论确认，与 ROADMAP 原文有偏差，以本 CONTEXT 为准）：**
1. 现有 `Repository[T]` 接口**已含 BatchDelete**（`base/service.go:18`），真正缺失/需改造的是 List 的查询形态
2. `GORMRepository[T]` 目前**零生产消费者**（仅 `base_80_05_test.go` 测试引用）— 是已测试的 dead code，重构自由度极高
3. `base.ApplySort`/`BaseListRequest` 有 **101 处引用** — 是 base 包真正广泛使用的资产，本期不动
4. CRUD 服务家族实际是 **11 个**不是 9 个：`wall_service.go`（136 行/10 函数）与 `floor_plan_text_service.go`（完整六方法同构）是 ROADMAP 清单遗漏，本期全纳入
5. LOC 净减目标从 ≥2000 下调为 **≥800 行**（11 服务共 ~3000 行，减 2000 = 67% 数学不可达；800-1000 是诚实估算）

**不在本 phase：**
- 缓存层三处架构统一（`floor_cache_impl.go` / `cache_invalidator.go` / legacy root cache services）— Phase 92 范围
- config_backup 三处 TODO 闭环（Phase 93）
- handler 层的 CRUD handler 模板重复（若存在同类重复，属后续独立治理）
- Statistics / SearchOptions 等异构业务查询方法的抽象化（明确留 service 层，见 D-04）
- `excel_service.go` / `batch_upserter.go` 的 Repository 化（走独立批量管道，非单实体 CRUD 模式）

</domain>

<decisions>
## Implementation Decisions

### Repository 接口形态 (Area 1)

- **D-01: gorm scope 函数式** — `GORMRepository[T]` 的 List 改为接受 gorm scope 函数（如 `List(ctx, scopes ...func(*gorm.DB) *gorm.DB, page PaginationParams)` 形态，最终签名 planner 定）。service 保留 typed request → scope 的转换（现有 `buildListQueryFromRequest` 直接改写为返回 scope）。理由：类型安全全保、GORM 生态惯例（`gorm.Scopes` 一等公民）、删代码最多。`Query`/`WhereCondition` DSL 零消费者直接删除。
- **D-02: 纯 struct 组合** — 不定义 `Repository[T]` interface，service 内部直接组合 `*GORMRepository[T]` struct。理由：零生产消费者无需 mock、项目测试直接打 sqlite in-memory 真库（v1.27 基建）、Go 泛型 interface 无法表达异构方法、YAGNI。现有 `Repository[T]` interface 定义删除。
- **D-03: 类型全统一到 base** — 删 `Query`/`WhereCondition`（零消费者）；`PageResult` 统一到 base 包单一定义（`base/service.go:38` 与 `operations/pagination_helper.go:17` 双定义合并，operations 侧删除、包内引用改 `base.PageResult`）；`GetPagination`/`ApplySort`/`BaseListRequest` 留 base 不动（101 处引用零风险）。

### 异构方法归属 (Area 2)

- **D-04: Statistics / SearchOptions 留 service 层** — `Repository[T]` 只管 T 中心 CRUD 六方法（Create/Update/Delete/GetByID/List/BatchDelete）。`Statistics`（返回 `*WorkstationStatisticsResult` / `*AssetStatisticsResult` 异构聚合）、`SearchWorkstationOptions`（返回 `[]DropdownOption`）、`GetWorkstationDeptOptions`、`BatchUpdatePositions` 均为业务查询非仓储抽象，各自 GORM 实现留 service 不动。**ROADMAP Success Criteria 1 与 REQUIREMENTS CRUD-REUSE-01 的「补全 Statistics/SearchOptions」措辞同步修订**（沿 Phase 90 同步 REQUIREMENTS 先例 commit 3a2efe5）。

### workstation map 参数 (Area 3)

- **D-05: 统一 typed request** — workstation `List`/`SearchWorkstationOptions` 的 `map[string]interface{}` 参数改为 typed request。`WorkstationListRequest` struct **已存在**（`internal/api/v1/operations/requests/workstation_requests.go:4`，已声明未接线），只需接线 + handler 2 个方法适配 bind。前端 JSON 契约不变（行为零变更）。随之消除 `extractIntParam`/`extractStringParam` 弱类型提取族（`pagination_helper.go`）。

### 范围与目标校准 (Area 4)

- **D-06: 11 个服务全纳入** — ROADMAP 原 9 个 + `wall_service.go` + `floor_plan_text_service.go`。两者已是同构六方法模式（typed request、ApplySort、`crud_services_test.go` 已覆盖 wall），零额外设计成本，91-04 收尾 plan 扩容到 6 个服务。
- **D-07: LOC 标准 = 混合标准 ≥800** — Success Criteria 3 改为「11 services 全部复用 Repository，每服务 CRUD 模板（直调 GORM + countRecords/fetchRecords 重复）清零 + **LOC 净减 ≥800 行**」。ROADMAP SC-3 与 REQUIREMENTS 对应措辞同步修订。保住「消除重复」本质目标 + 可验收量化锚点。

### Claude's Discretion

以下细节由 planner/researcher 决定，无需再问用户：
- `xxxAllowedSortFields` 排序白名单 map 留在各 service（业务语义：每个实体可排序字段不同），`base.ApplySort` 辅助不动
- soft-delete 语义不变（GORM DeletedAt 默认行为，现有 Delete 实现已如此）
- 错误包装（`WrapError`/`IsNotFound`/`IsDuplicate`）的保留/使用方式
- Repository Create/Update 是否加 BeforeCreate/AfterUpdate 钩子点（若现有 service 有 validateXxxRelations 前置校验，保留在 service 层调用 repository 之前）
- 事务边界（单实体 CRUD 无跨表事务需求，维持现状）
- `GORMRepository` 改造后 `base_80_05_test.go` 的适配重写 + 新增 `base/service_test.go` 泛型契约锁定的具体测试形态
- 11 个服务在 4 个 plan 中的分组微调（保持 pilot → 批量 → 收尾节奏即可）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/REQUIREMENTS.md` § CRUD-REUSE（:55-66）— CRUD-REUSE-01..08 需求原文（**01 的「补全 Statistics/SearchOptions」措辞按 D-04 修订**）
- `.planning/workstreams/milestone/ROADMAP.md` § Phase 91 — Goal / 4 plans 拆分 / Success Criteria（**SC-1 按 D-04、SC-3 按 D-07 修订**）
- `.planning/PROJECT.md` — v1.29 Current Milestone 段，D-01..D-06 locked decisions + D-PRINCIPLE 行业最佳实践
- `.planning/workstreams/milestone/phases/90-timeouts-port-protocol-concurrency/90-CONTEXT.md` — Phase 90 决策风格 + REQUIREMENTS 同步先例（3a2efe5）+ D-PRINCIPLE 强化模式

### 本期改造核心代码
- `internal/services/base/service.go` — 现有 `Repository[T]` interface + `GORMRepository[T]`（**零生产消费者，D-01/D-02/D-03 改造对象**）
- `internal/services/base/base_80_05_test.go` — GORMRepository 现有测试（Phase 80-05 写，改造后需适配重写）
- `internal/services/base/list_request.go` — `BaseListRequest`/`ApplySort`（101 处引用的真正资产，不动）
- `internal/services/operations/pagination_helper.go` — operations 侧 `PageResult` 重复定义 + `extractXxxParam` 弱类型族（D-03/D-05 清理对象）

### 11 个迁移目标 service（形状参考）
- `internal/services/operations/door_service.go` — **最小同构样本**（148 行，六方法 + validateRelations + buildListQuery + countRecords/fetchRecords 全套模式）
- `internal/services/operations/workstation_service.go` — **pilot 目标，最大最异类**（456 行；List map 参数 + 4 个独有方法 Statistics/GetWorkstationDeptOptions/BatchUpdatePositions/SearchWorkstationOptions）
- `internal/services/operations/wall_service.go` / `floor_plan_text_service.go` — 清单遗漏的两个同构服务（D-06 新纳入）
- `internal/services/operations/crud_services_test.go` — 现有 CRUD 服务测试（wall/door/server_room/dedicated_line/infopoint 已覆盖，迁移后必须全绿）
- `internal/api/v1/operations/requests/workstation_requests.go:4` — `WorkstationListRequest` 已存在未接线（D-05）
- `internal/services/operations/dropdown_helper.go` — `DropdownOption` 类型 + 缓存 key 辅助（SearchOptions 共享部分，不动）

### 项目级约束
- `CLAUDE.md` § Go Code Patterns — Handler-Service 标准模式（service 接口对 handler 是契约，迁移不破坏）
- `CLAUDE.md` § Compilation & Build Verification — `go build ./...` 必须 0 错误
- `CLAUDE.md` § Testing — `go test ./...` 0 失败（1688+ 测试不回归，v1.29 D-03）
- `internal/models/status_constants_test.go` / `internal/utils/operlog/regression_test.go` — AST 锁值防线全程保持绿

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GORMRepository[T]`（`base/service.go:46`）— 骨干已在（Create/Update/Delete/GetByID/BatchDelete 五方法可保留，仅 List 需 scope 化改造），零消费者意味着无兼容包袱
- `base.ApplySort` + `BaseListRequest`（`list_request.go`）— 101 处引用的排序白名单机制，11 个服务全部已在用，本期原样保留
- `WorkstationListRequest`（`workstation_requests.go:4`）— typed struct 已声明，D-05 只需接线
- `crud_services_test.go` — 5 个服务的 CRUD 回归测试已存在，是「行为零变更」的现成验证器
- v1.27 测试基建（sqlite in-memory）— service 测试直接打真库，无需 repository mock（D-02 的依据）

### Established Patterns
- **同构 CRUD 六方法模式**：11 个 service 全部是 `struct{db, validator}` + Create/Update/Delete/GetByID/List/BatchDelete + `buildListQueryFromRequest` + `countRecords`/`fetchRecords` + `xxxAllowedSortFields` 白名单 — 重复模板即迁移对象
- **typed request 模式**：10 个服务用 `requests.XxxListRequest`；唯 workstation 用 map（D-05 统一）
- **Handler-Service 契约**：handler 依赖 service 接口；迁移只改 service 内部实现 + workstation 2 个 handler 方法的 bind 适配，其余 handler 零改动
- **AST 锁值测试先例**（Phase 89/90）：`base/service_test.go` 锁泛型契约可参考 `pagination_test.go` 模式

### Integration Points
- `internal/api/v1/operations/` handlers → service 接口（除 workstation List/SearchOptions 2 处 bind 适配外零改动）
- `excel_service.go` / `batch_upserter.go` — 批量管道独立，不迁 Repository（明确排除项）
- `floor_cache_impl.go` / `cache_invalidator.go` — Phase 92 范围，本期不触碰（但 service 方法签名不变保证其零影响）
- `internal/api/v1/operations/requests/` — workstation typed request 接线点

</code_context>

<specifics>
## Specific Ideas

### 「已声明未接线」asset（用户确认利用）
- `WorkstationListRequest`（workstation_requests.go:4）已存在 — D-05 的 typed 化不是新建 struct 而是接线，成本远低于预估

### 现实偏差必须体现在 plan 里（用户确认）
- ROADMAP 91-01 原文「补全缺失方法（Statistics/SearchOptions/BatchDelete）」→ 实际改造是 **List scope 化 + interface 删除**，BatchDelete 已存在，Statistics/SearchOptions 不进 Repository（D-04）
- ROADMAP SC-3「LOC 净减少 ≥2000 行」→ **≥800 行**（D-07），planner 按 11 服务现实估算

### 估算依据（供 planner 参考）
- 11 个 service 文件合计 ~3000 行（2720 + wall 136 + fpt ~150）
- 每服务可消除：5 个 CRUD 直调方法体（~30-60 行）+ countRecords/fetchRecords（~20 行）+ map 提取残留；pilot 后 91-03/91-04 批量复制模式
- 净减 800-1000 行为诚实区间；「CRUD 模板清零」是定性底线（每个 service 不再有直调 GORM 的重复六方法体）

</specifics>

<deferred>
## Deferred Ideas

### 范围外但相关的后续候选
- **handler 层 CRUD 模板重复治理** — `internal/api/v1/operations/` 下各 entity handler 若存在同构 CRUD handler 样板，属独立后续 phase（本 phase 仅 services 层；侦察发现 `base_handler.go` 已有部分共享）
- **excel_service / batch_upserter Repository 化** — 批量管道模式与单实体 CRUD 不同构，如需抽象属 Phase 92+ 评估
- **`pagination_helper.go` 的 `MaxOptionsPageSize`(10000) 上限语义** — 注释中已标注「若未来拆分 options 端点可改用 MaxListPageSize」，属业务决策延后
- **Phase 92 缓存层统一** — 与本 phase 同在 `internal/services/`，顺序执行避免 git diff 冲突（ROADMAP 已锁定顺序依赖）

None — discussion stayed within phase scope（无 scope creep 提案）

</deferred>

---

*Phase: 91-CRUD 复用 base.Repository[T]*
*Context gathered: 2026-09-04*
