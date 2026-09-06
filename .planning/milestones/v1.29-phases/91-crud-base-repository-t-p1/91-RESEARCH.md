# Phase 91: CRUD 复用 base.Repository[T] (🔥 高优 P1) - Research

**Researched:** 2026-09-04
**Domain:** Go 后端泛型 Repository 重构（GORM scope 函数式、11 个同构 CRUD service 迁移、零行为变更）
**Confidence:** HIGH（全部结论基于本地代码逐文件审读 + GORM v1.30.5 模块源码 + 项目内 sqlite 运行时 spike 实证，无外部依赖面）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: gorm scope 函数式** — `GORMRepository[T]` 的 List 改为接受 gorm scope 函数（如 `List(ctx, scopes ...func(*gorm.DB) *gorm.DB, page PaginationParams)` 形态，最终签名 planner 定）。service 保留 typed request → scope 的转换（现有 `buildListQueryFromRequest` 直接改写为返回 scope）。理由：类型安全全保、GORM 生态惯例（`gorm.Scopes` 一等公民）、删代码最多。`Query`/`WhereCondition` DSL 零消费者直接删除。
- **D-02: 纯 struct 组合** — 不定义 `Repository[T]` interface，service 内部直接组合 `*GORMRepository[T]` struct。理由：零生产消费者无需 mock、项目测试直接打 sqlite in-memory 真库（v1.27 基建）、Go 泛型 interface 无法表达异构方法、YAGNI。现有 `Repository[T]` interface 定义删除。
- **D-03: 类型全统一到 base** — 删 `Query`/`WhereCondition`（零消费者）；`PageResult` 统一到 base 包单一定义（`base/service.go:38` 与 `operations/pagination_helper.go:17` 双定义合并，operations 侧删除、包内引用改 `base.PageResult`）；`GetPagination`/`ApplySort`/`BaseListRequest` 留 base 不动（101 处引用零风险）。
- **D-04: Statistics / SearchOptions 留 service 层** — `Repository[T]` 只管 T 中心 CRUD 六方法（Create/Update/Delete/GetByID/List/BatchDelete）。`Statistics`（返回 `*WorkstationStatisticsResult` / `*AssetStatisticsResult` 异构聚合）、`SearchWorkstationOptions`（返回 `[]DropdownOption`）、`GetWorkstationDeptOptions`、`BatchUpdatePositions` 均为业务查询非仓储抽象，各自 GORM 实现留 service 不动。
- **D-05: 统一 typed request** — workstation `List`/`SearchWorkstationOptions` 的 `map[string]interface{}` 参数改为 typed request。`WorkstationListRequest` struct 已存在（`workstation_requests.go:4`，已声明未接线），只需接线 + handler 2 个方法适配 bind。前端 JSON 契约不变（行为零变更）。随之消除 `extractIntParam`/`extractStringParam` 弱类型提取族（`pagination_helper.go`）。
- **D-06: 11 个服务全纳入** — ROADMAP 原 9 个 + `wall_service.go` + `floor_plan_text_service.go`。零额外设计成本，91-04 收尾 plan 扩容到 6 个服务。
- **D-07: LOC 标准 = 混合标准 ≥800** — Success Criteria 3 改为「11 services 全部复用 Repository，每服务 CRUD 模板清零 + **LOC 净减 ≥800 行**」。

### Claude's Discretion（planner/researcher 自主决定，无需再问用户）

- `xxxAllowedSortFields` 排序白名单 map 留在各 service，`base.ApplySort` 辅助不动
- soft-delete 语义不变（GORM DeletedAt 默认行为）
- 错误包装（`WrapError`/`IsNotFound`/`IsDuplicate`）的保留/使用方式
- Repository Create/Update 是否加钩子点（validate 前置校验保留在 service 层调用 repository 之前）
- 事务边界（单实体 CRUD 无跨表事务需求，维持现状）
- `GORMRepository` 改造后 `base_80_05_test.go` 的适配重写 + 新增 `base/service_test.go` 泛型契约测试的具体形态
- 11 个服务在 4 个 plan 中的分组微调（保持 pilot → 批量 → 收尾节奏即可）

### Deferred Ideas (OUT OF SCOPE)

- handler 层 CRUD 模板重复治理（独立后续 phase）
- excel_service / batch_upserter Repository 化（批量管道，Phase 92+ 评估）
- `pagination_helper.go` 的 `MaxOptionsPageSize`(10000) 上限语义拆分（业务决策延后）
- Phase 92 缓存层统一（`floor_cache_impl.go` / `cache_invalidator.go` 本期不触碰）
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CRUD-REUSE-01 | Repository[T] 完整 CRUD 抽象（**按 D-04 修订**：List scope 化 + interface 删除，Statistics/SearchOptions 不进 Repository） | 「GORMRepository 目标设计」节给出已验证签名；BatchDelete 已存在但空 ids 语义需反转（见 Pitfall P1） |
| CRUD-REUSE-02 | workstation_service 迁移（pilot） | 「11 服务形状盘点」#11 + D-05 接线清单（handler bind 2 处 + mock/stub 测试 2 文件 + 533 行行为锁测试改写） |
| CRUD-REUSE-03 | building_service 迁移 | 盘点 #2；注意 Table 链软删除 Count 分歧（Pitfall P2）+ typesafe 死文件处置决策 |
| CRUD-REUSE-04 | floor_service 迁移 | 盘点 #3；接口签名被 floorCacheService 装饰器锁定不可改；Create/Update/BatchDelete 业务逻辑保留 service |
| CRUD-REUSE-05 | asset_service 迁移 | 盘点 #4；Table 链软删除分歧同 building；无 AssetListRequest（D-05 不涉及，保持 map 签名） |
| CRUD-REUSE-06 | server_room + infopoint + dedicated_line + room_device + door 批量迁移 | 盘点 #5-#9；7 个 typed 服务之一族，scope 改写模板已验证 |
| CRUD-REUSE-07 | 每 service 迁移后 `go test ./internal/services/operations/...` 全过 + `base/service_test.go` 锁泛型契约 | 「Validation Architecture」节；Wave 0 缺口：floor / floor_plan_text 无直接 service 级测试 |
| CRUD-REUSE-08 | LOC 统计（按 D-07 修订 ≥800）+ SUMMARY + handler e2e 0 回归 | 「LOC 净减估算」节：严格范围 450-550L，达 800 需纳入补充来源（typesafe 死文件清理等），需 planner 显式决策 |
</phase_requirements>

## Summary

本 phase 是一个**纯内部重构**：把 `internal/services/operations/` 下 11 个同构 CRUD service 的重复查询管道（Model/WithContext/Count/Offset/Limit/Find + countRecords/fetchRecords/buildListQueryFromRequest 样板）收敛到改造后的 `base.GORMRepository[T]`（gorm scope 函数式），同时统一 `PageResult`、删除无人消费的 `Query`/`WhereCondition` DSL 与 `Repository[T]` interface。研究确认改造自由度极高：`GORMRepository[T]` 零生产消费者，`base.ApplySort`/`BaseListRequest` 才是 base 包的真资产（实测 54 文件 239 处引用，本期不动）。

研究通过**运行时 spike**（项目自身 glebarez/sqlite + GORM v1.30.5，已验证 5 项后删除临时文件）和 GORM 模块源码审读，实证了 scope 函数式 List 的全部关键前提：scope 中携带 `Joins+Select` 时 Count 自动替换为 `count(*)` 并在 Find 恢复、多次 `Order()` 合并为复合 `ORDER BY`、Count→Find 链复用无污染、空切片 `IN` 删除是无害 no-op。同时发现了一个**此前无人知晓的现存行为分歧**：building/asset 的 List 用 `.Table(...)` 起链（非 `.Model(...)`），导致其 **Count 不过滤软删除、Find 过滤**——统一到 `Model(new(T))` 会改变这两处 Total 语义，必须由 planner 显式决策（推荐按 bugfix 处理并 checkpoint 确认）。

最重要的研究产出是**对 CONTEXT 前提的三处修正**：(1) map 参数服务实际是 **4 个**（workstation/building/floor/asset）不是 1 个，D-05 的「消除 extractXxxParam 提取族」在严格范围内不可达；(2) `WorkstationListRequest` 缺 `floorCode/type/orgId` 3 个字段且现有 `BuildingID/Code` 字段服务从不读取，「只需接线」低估了改动；(3) 严格迁移的 LOC 净减约 450-550 行，**低于 800 目标**，达标需纳入补充来源（如零生产消费者的 `building_service_typesafe.go` 死文件清理）。这些偏差均不推翻 D-01..D-07，但 planner 必须在 plan 中显式处理。

**Primary recommendation:** 按「接口签名不变 + 内部 scope 化」原则迁移 10 个服务（floor 签名被 `floorCacheService` 装饰器锁定，workstation 是 D-05 唯一例外）；`PageResult` 用 type alias 合并（`type PageResult = base.PageResult`）避免强制触碰 Phase 92 文件；repo 的 `BatchDelete` 空 ids 语义反转为返回 nil；building/asset 的软删除 Count 分歧按 latent bugfix 处理并加 `checkpoint:human-verify`；为达 LOC ≥800，将 `building_service_typesafe.go`（171 行零消费者死代码）纳入清理范围。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 单实体 CRUD 六方法（Create/Update/Delete/GetByID/List/BatchDelete） | Database/Storage（`base.GORMRepository[T]`） | API/Backend（service 层做 validate/preflight） | 纯数据访问管道收敛到泛型仓储；业务校验留 service |
| 查询条件构造（filter → scope） | API/Backend（service 层） | — | filter 语义是业务知识（字段名/EXISTS 子查询），repo 只消费 scope |
| 排序白名单解析 | API/Backend（service 层，`base.ApplySort` 不动） | — | `xxxAllowedSortFields` 每实体不同，CONTEXT 明确留 service |
| 分页参数归一化 | API/Backend（service 层 / requests.GetPagination） | — | 三种 clamp 语义并存（见 Pitfall P5），repo 必须直信入参 |
| Statistics / SearchXxxOptions / GetTree / BatchUpdatePositions | API/Backend（service 层，D-04 锁定） | — | 异构聚合/下拉查询，非仓储抽象 |
| handler JSON bind | Frontend Server（`internal/api/v1/operations/` handlers） | — | 仅 workstation List/SearchOptions 2 处因 D-05 改动 |
| 缓存装饰（floor_cache_impl） | API/Backend | — | Phase 92 范围，本期通过「floor 接口签名不变」实现零接触 |

## 关键发现：CONTEXT 前提修正（planner 必读）

研究过程中发现 CONTEXT/讨论基线的 5 处事实偏差。**均不推翻 D-01..D-07 决策本身**，但直接影响 plan 拆分与验收口径：

### F1. map 参数服务是 4 个，不是 1 个 [VERIFIED: 代码逐文件审读]

List 签名普查（`List(ctx context.Context,` 全量 grep）：

| List 参数形态 | 服务 |
|--------------|------|
| `map[string]interface{}`（4 个） | workstation、**building**、**floor**、**asset** |
| `requests.XxxListRequest`（7 个） | dedicated_line、door、floor_plan_text、infopoint、room_device、server_room、wall |

CONTEXT 写「唯 workstation 用 map」——实际 building/floor/asset 的**生产** List 也是 map（handler 直接 `ShouldBindJSON(&params)` 到 map：`building_handler.go:103`、`floor_handler.go:95`、`asset_handler.go:105`）。`requests.BuildingListRequest`/`FloorListRequest` 已存在但只被零生产消费者的 typesafe 变体使用。

**影响：** D-05 只锁了 workstation 的 typed 化。building/floor/asset 迁移时**接口签名保持 map 不变**（service 内部 map→scope 转换），否则违反「floor 接口签名被装饰器锁定」约束（floor）并扩大 handler 改动面（building/asset）。推论：`extractIntParam`/`extractStringParam`/`extractPagination`/`extractSortRequest` 在 3 个 map 服务 List + 全部 `SearchXxxOptions` + workstation/building Statistics 中仍被需要 → **D-05 的「随之消除提取族」只能部分达成**（workstation 消费点消除，helper 函数本体保留）。planner 应在 plan 中把「提取族消除」的验收口径改写为「workstation 消费点清零」。

### F2. WorkstationListRequest 不完整，「接线」实为「接线 + 扩字段」 [VERIFIED: workstation_requests.go:4 vs workstation_service.go:240-277]

现状字段：`PaginationParams`（嵌入）、`StatusRequest`（嵌入）、`BuildingID`、`FloorID`、`Name`、`Code`。
List 实际消费的 map key：`name`、`floorId`、`floorCode`、`status`、`type`、`orgId`、`current`、`pageSize`、`orderByColumn`、`isAsc`。

- **缺失 3 字段需新增**：`FloorCode string`、`Type *int`（保留 -1/缺失跳过语义，见 P6）、`OrgID string`
- **现有 `BuildingID`/`Code` 字段服务从不读取**（前端如发 `buildingId` 会被静默忽略——`workstation_floor_code_77_03_test.go:415` 正是验证这一忽略行为）。typed 化后**不得**开始消费它们，否则行为变更
- 新增字段是 JSON 加性变更（可选字段），前端契约零变化 ✓

### F3. building/asset List 存在「Count 无软删除、Find 有软删除」的现存分歧 [VERIFIED: GORM v1.30.5 源码 + 运行时 spike 双重实证]

building List（`building_service.go:131`）与 asset List（`asset_service.go:185`）以 `.Table("ops_buildings")`/`.Table("ops_asset")` 起链而非 `.Model(...)`。GORM 的 `callbacks.Execute`（callbacks.go:103-120）在 `stmt.Model == nil` 时令 `Model = Dest`：**Count 时 Dest 是 `*int64` → Schema 解析失败且 Table 已设置时错误被吞 → 无软删除过滤**；**Find 时 Dest 是 `&[]T` → Schema 解析成功 → 软删除过滤生效**。其余 9 个服务用 `.Model(&T{})` 起链，Count/Find 一致过滤。

Spike 实证（项目内 sqlite 运行）：Table 链 count=3（含软删）、find=2（排除）；Model 链 count=2。

**影响：** repo.List 统一用 `Model(new(T))` 后，building/asset 的 Total 将不再计入软删除行——这是**行为变更**（虽然方向是修 bug：现状 Total ≥ 实际列表行数）。两个选项：
- **推荐**：接受为 latent bugfix，plan 中加 `checkpoint:human-verify`（说明：前端分页 Total 与列表行数将严格一致）；现有测试不软化任何行，不会回归
- 保守：这两个服务 List 保留 Table 起链的自定义实现（不进 repo）——违背 D-06「全部复用」，不推荐

### F4. `operations.PageResult` 硬删除会强制触碰 Phase 92 文件 [VERIFIED: 全库 grep]

`opsServices.PageResult`/`operations.PageResult` 在 `internal/api/v1/operations/` 的 **12 个测试文件中共 76 处引用**，另有 2 个非迁移目标生产文件：`floor_cache_impl.go:163`（**Phase 92 明确不触碰**）和 `location_alias_service.go:45/95/125`。两个定义字段/JSON tag 完全一致。

**推荐：** operations 侧改为 **type alias**：`type PageResult = base.PageResult`（+ `type PaginationParams` 保持或逐步替换）。alias 下两种写法是同一类型，76 处测试引用与 2 个非目标文件零改动即可编译，同时满足 D-03「单一定义」的本质（唯一 struct 定义在 base）。若 planner 坚持字面执行「operations 侧删除」，则必须接受 1 行机械改动触碰 floor_cache_impl.go + location_alias_service.go + 12 个测试文件的 76 处替换（约 +90 行 diff 噪音，且与「本期不触碰 floor_cache_impl」的 phase boundary 冲突）。

### F5. LOC ≥800 在严格范围内不可达，需显式选择补充来源 [VERIFIED: 分项估算]

详见「LOC 净减估算」节。严格迁移（11 服务 + base 改造 + helper 修剪）≈ **450-550 行净减**。达 800 的可行组合（planner 决策）：
1. **删除 `building_service_typesafe.go`（171 行）+ `building_handler_typesafe.go` + 对应测试块** —— 零生产消费者死代码（router.go:546 只用 `NewBuildingService`；typesafe 构造器仅被测试引用），本身就是「消除重复」目标物，推荐纳入
2. building/floor typed request 接线扩展（struct 已存在，handler bind 各 1 处）—— D-05 的自然延伸，但属范围扩张，建议 checkpoint 确认
3. `pagination_helper_test.go` 中随实现删除的用例（最多 ~115 行）

## Standard Stack

### Core（全部为既有依赖，零新增安装）

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gorm.io/gorm | v1.30.5 [VERIFIED: go.mod:34 + 本地模块源码审读] | ORM；scope 函数式是一等公民（`Scopes(...)` 附加到 Statement，构建期执行） | 项目唯一 ORM；scope 模式即官方推荐的项目级查询复用方式 |
| github.com/glebarez/sqlite | 既有 | 测试用纯 Go sqlite（内存/临时文件库） | v1.27 测试基建标准，`crud_services_test.go`/`base_80_05_test.go` 已用 |
| github.com/stretchr/testify | 既有 | assert/require | 全项目测试标准 |
| gorm.io/gorm/logger | v1.30.5 | 测试静默日志 | 既有用法 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 自定义 `Scope` 类型别名 | 直接写 `func(*gorm.DB) *gorm.DB` | 别名提升可读性，零成本；纯命名问题 |
| type alias 合并 PageResult | 硬删除 + 76 处测试替换 | 见 F4；alias 零风险且不触碰 Phase 92 文件 |
| `Session(&gorm.Session{})` 每次新建会话 | 复用链（现状模式） | 新建会话更"洁癖"但改变现有共享语句行为路径；spike 已证明现状复用模式在 Count→Find 间无污染，保持现状 |

**Installation:** 无（本 phase 零外部包安装）。

## Package Legitimacy Audit

> 本 phase **不安装任何外部包**（纯内部重构）。gorm/glebarez-sqlite/testify 均为既有依赖且已在生产与测试中使用。审计结论：N/A — 无新增包，无需 checkpoint。

## GORMRepository 目标设计（spike 实证版）

### 已验证的行为前提（全部经 GORM v1.30.5 源码 + 项目内 sqlite 运行时 spike 确认，HIGH）

1. **Count 与自定义 Select/Joins 共存**：`Count` 检测到非 `count(` 开头的 Select 时临时替换为 `count(*)` 并在返回后恢复原 SELECT（finisher_api.go Count 的 defer 逻辑）→ scope 可同时携带 Joins+Select，Count 仍正确
2. **Count 剥离 ORDER BY**（无 GROUP BY 时）并在返回后恢复 → 排序 scope 在 Count 前应用不影响 count SQL
3. **多 `Order()` 合并为复合排序**：`ORDER BY name ASC,created_at DESC`（spike dry-run SQL 断言通过）→ door/wall 的「用户排序 + 尾随 created_at DESC」语义可直接用顺序 Order 复刻
4. **`Order("")` 是 no-op**（chainable_api.go：`if v != ""` 才 AddClause）→ door/wall 中的 `query.Order("")` 是死代码，迁移时可安全删除
5. **Count→Find 共享链复用无污染**：count(*)、ORDER BY 均被 defer 恢复（spike 通过）
6. **空切片 `Delete WHERE id IN ?` 是无害 no-op 返回 nil**（spike 通过）
7. **Table 链 vs Model 链的软删除 Count 分歧**（见 F3，spike 通过）

### 推荐签名（D-01/D-02 授权 planner 定稿，以下为研究推荐）

```go
// Scope 与 gorm 官方 scope 函数签名一致。
type Scope = func(*gorm.DB) *gorm.DB

// PageParams 已归一化的分页参数（由 service 层负责 clamp，repo 直信入参——见 P5）。
type PageParams struct {
    Current  int
    PageSize int
}

type GORMRepository[T any] struct {
    db *gorm.DB
}

func NewGORMRepository[T any](db *gorm.DB) *GORMRepository[T]

// 六方法（D-02：不定义 interface，service 直接组合 struct）
func (r *GORMRepository[T]) Create(ctx context.Context, entity *T) error
func (r *GORMRepository[T]) Update(ctx context.Context, entity *T) error          // Save 全量；Omit 变体见下
func (r *GORMRepository[T]) Delete(ctx context.Context, id string) error
func (r *GORMRepository[T]) GetByID(ctx context.Context, id string, scopes ...Scope) (*T, error)
func (r *GORMRepository[T]) List(ctx context.Context, page PageParams, scopes ...Scope) (*PageResult, error)
func (r *GORMRepository[T]) BatchDelete(ctx context.Context, ids []string) error   // 空 ids → nil（语义反转，见 P1）
```

**List 内部管道**（spike 验证的形状）：

```go
func (r *GORMRepository[T]) List(ctx context.Context, page PageParams, scopes ...Scope) (*PageResult, error) {
    query := r.db.WithContext(ctx).Model(new(T))
    for _, s := range scopes {
        if s != nil {
            query = s(query)
        }
    }
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }
    var list []T
    offset := (page.Current - 1) * page.PageSize
    if err := query.Offset(offset).Limit(page.PageSize).Find(&list).Error; err != nil {
        return nil, err
    }
    return &PageResult{List: list, Total: total, Current: page.Current, PageSize: page.PageSize}, nil
}
```

### 设计要点与依据

| 设计点 | 决定 | 依据 |
|--------|------|------|
| repo 内是否 clamp 分页 | **否** | 三种现存 clamp 语义并存（P5）；repo 直信入参保持各自行为 |
| `PageResult.List` 装什么 | **值切片 `[]T`** | `crud_services_test.go:179` 有 `page.List.([]operationsmodels.OpsServerRoom)` 值切片断言；wall/door 现装 `*[]T` 但其测试只断言 Total，handler 不类型断言；JSON 序列化两者等价 |
| GetByID 是否支持 scopes | **是（变长参数，零 scope 时 SQL 与现状逐字等价）** | workstation/floor/infopoint/room_device 4 个服务的 GetByID 带 JOIN+Select；plain 服务（7 个）零 scope 时 `WHERE id = ?` 不变。**join 场景 id 过滤需表限定**：推荐 repo 内通过 Tabler 断言（`interface{ TableName() string }`，11 个模型全部实现 [VERIFIED: models 目录 grep]）在 scope 非空时用 `<table>.id = ?`，空 scope 时保持 `id = ?` |
| Update 变体（`Omit("CreatedAt")` / First 回填） | **repo 只提供 Save 全量版；building/asset 的 `Omit("CreatedAt")` 与 workstation 的 First 回填保留在 service 层**（或 planner 可选加 `UpdateOmit(ctx, entity, cols ...string)`） | building_service.go:111、asset_service.go:144、workstation_service.go:185-197；变体各有业务注释（零值覆盖 created_at bug 的修复），不宜隐入通用 repo |
| `Repository[T]` interface / `Query` / `WhereCondition` | **删除**（D-01/D-02） | 零生产消费者 [VERIFIED: `NewGORMRepository\|base.Query\|base.Repository\[` 非 test 全库 grep 仅命中定义处] |
| `WrapError`/`IsNotFound`/`IsDuplicate` | **原样保留** | CONTEXT discretion；`base_80_05_test.go` 有测试锁定 |

## 11 服务形状盘点（迁移映射表）

| # | 服务 | 行数 | List 参数 | GetByID | Create/Update 前置钩子 | 迁移要点 |
|---|------|------|-----------|---------|------------------------|----------|
| 1 | building | 300 | **map** | plain | validateOrg + validateNameUnique | F3 软删分歧；Order 默认 `order_num ASC`（排他语义）；List 有 workstation_count 子查询 Select |
| 2 | floor | 436 | **map** | **JOIN**（buildings+sys_files） | Create: validateBuilding + **唯一约束软删恢复逻辑**；Update: **异步 goroutine 同步工位 building_id**；Delete/BatchDelete: **updateBuildingFloorCount** | **接口签名不可改**（floorCacheService 装饰器实现同一接口，floor_cache_impl.go:163）；Create/Update/Delete/BatchDelete 业务逻辑整体保留 service，仅 GetByID/List 管道进 repo；floor 的分页提取是**无 clamp 内联断言**（P5） |
| 3 | asset | 411 | **map** | plain | validateDept + validateUser + validateDeviceSNUnique；Update 用 `Omit("CreatedAt")` | F3 软删分歧；List 有 `component_type IS NULL` 恒定过滤（必须进 scope）；applyFilters 9 个 filter → scope |
| 4 | workstation（pilot） | 456 | **map → D-05 typed** | **JOIN**（6 表） | applyWorkstationOccupancyLink（Create/Update）；Update 另有 First 回填 CreatedAt/CreatedBy | D-05 全套接线（见下节）；Statistics/GetWorkstationDeptOptions/SearchWorkstationOptions/BatchUpdatePositions 留 service（D-04）；`validateTableName` 白名单是常量防御死分支可留 |
| 5 | server_room | 251 | typed | plain | validateFloor | List 的 Select+Joins **在 Count 之前**（现状已证明 scope 全量前置可行）；orgId 空命中早退空 PageResult 保留 service |
| 6 | infopoint | 264 | typed | **JOIN**（workstation+floors+buildings） | populateRedundantFields（3 次回查回填） | WorkID/PointType 旧字段兼容逻辑保留在 scope 转换中；Order 默认 `ops_info_points.created_at DESC`（排他） |
| 7 | dedicated_line | 231 | typed | plain | populateRoomNames（2 次回查回填） | 11 个 filter 全部平移进 scope |
| 8 | room_device | 223 | typed | **JOIN**（server_rooms） | validateRoom（Create 另有 Create 后续逻辑，保留 service） | Joins 在 filter 之前（orgId 引用 joined 表）——scope 顺序敏感，保持原顺序 |
| 9 | door | 148 | typed | plain | validator.ValidateFloor + ValidateWall | **最小同构样本**；countRecords/fetchRecords/buildListQueryFromRequest 全套删除；**复合排序语义**（用户排序 + 恒尾随 created_at DESC） |
| 10 | wall | 136 | typed | plain | validator.ValidateFloor | 同 door；`Order("")` 死代码删除 |
| 11 | floor_plan_text | 116 | typed | plain | validator.ValidateFloor | Delete/BatchDelete 用 `.Table(...)` 形式（行为等价）；无 JOIN，最简单 |

**排序语义分型（迁移时最容易踩的差异，务必分型处理）：**

| 语义 | 服务 | 现状代码形状 |
|------|------|-------------|
| **A: 复合尾随**（用户排序时 `ORDER BY <user>, created_at DESC`；无用户排序时 `ORDER BY created_at DESC`） | door、wall | ApplySort 后无条件在 Find 前 `Order("created_at DESC")` |
| **B: 排他默认**（用户排序时仅 `ORDER BY <user>`；无用户排序时 `ORDER BY <default>`） | 其余 9 个 | `if OrderByColumn == "" { Order(default) }` |

推荐 base 提供两个排序 scope helper 消除 9 处重复块：
`base.SortScope(req BaseListRequest, allowed map[string]string, defaultOrder string) Scope`（B 型）与
`base.SortScopeWithTail(req, allowed, tail string) Scope`（A 型，内部即顺序两个 Order）。底座复用不动 `base.ApplySort`/`ResolveSort`。

## D-05 workstation 接线清单（改动全集）

1. **`requests/workstation_requests.go`**：新增 `FloorCode string \`json:"floorCode"\``、`Type *int \`json:"type"\``、`OrgID string \`json:"orgId"\``（F2；`BuildingID`/`Code` 保留声明但不消费，注释说明以锁行为）
2. **`WorkstationService` 接口**：`List(ctx, map)` → `List(ctx, req requests.WorkstationListRequest)`；`SearchWorkstationOptions(ctx, map)` → typed（推荐独立 `WorkstationSearchRequest` 或复用 ListRequest 皆可，D-05 只要求去 map）。Statistics/GetWorkstationDeptOptions **保持 map**（D-05 未授权改动，Statistics 仍需 extractStringParam）
3. **`workstation_handler.go`**：`List`（:139-152）与 `SearchWorkstationOptions`（:90-）2 处 bind 改 `ShouldBindJSON(&req)`；bind 失败行为从「降级空 map 继续查询」变为 typed 的绑定策略——**保持降级语义**（bind 失败 → 零值 request → 空过滤全列表），与现状一致
4. **status/type 的 -1 跳过语义**：现状 `extractIntParam(params,"status",-1); status >= 0` 才过滤（`status:-1` 显式跳过有测试锁定，workstation_floor_code_77_03_test.go:145）。typed 化后必须保留：`if s := req.GetStatus(-1); s >= 0 { ... }` / Type 同理
5. **测试改写**：`workstation_floor_code_77_03_test.go`（533 行，~14 处 `svc.List(ctx, map{...})` 调用改 typed 字面量）；`workstation_handler_test.go`（stubWorkstationService 的 List/SearchWorkstationOptions 2 个 stub）；`workstation_handler_full_test.go`（mockWorkstationService 的 ListFunc/SearchWorkstationOptionsFunc 签名 + 若干 ListFunc 内联实现）
6. **分页 clamp 语义**：现状 map 路径 clampPageSize 上限 10000（MaxOptionsPageSize）；typed `requests.PaginationParams.GetPagination()` 上限 100（MaxListPageSize）。若直接复用 GetPagination，workstation list 的 pageSize 上限 10000→100——**前端常规列表页发送 ≤100，实际无感，但属运行时语义收紧**。另：map 路径 `current=0` 会产生负 offset（latent），typed GetPagination 会 guard 为 1。建议 plan 中按「接受收紧 + 修复负 offset」处理并 checkpoint 说明

## Architecture Patterns

### System Architecture Diagram

```
                         Phase 91 数据流（迁移后）
 ─────────────────────────────────────────────────────────────────
 handler (internal/api/v1/operations/*_handler.go)
   │  ShouldBindJSON
   │  ├─ typed request (7+workstation=8 个服务)
   │  └─ map[string]interface{} (building/floor/asset — 签名不变)
   ▼
 XxxService interface（签名除 workstation 2 方法外全部不变）
   │  1. validate/preflight（Validator / validateOrg / populateXxx / occupancy link）
   │  2. 构造 scopes：filterScope(req) + sortScope(req) + joinScope（仅 4 个带 JOIN 服务）
   │  3. 归一化分页（GetPagination / extractPagination / 内联 —— 三种语义各自保留）
   ▼
 *base.GORMRepository[T]（struct 组合，D-02）
   │  Model(new(T)) → 应用 scopes → Count → Offset/Limit/Find
   ▼
 *base.PageResult{List []T, Total, Current, PageSize}   ← 单一定义（D-03，operations 侧 alias）
   │
   ▼
 response.Success(c, page) → JSON 契约零变更

 留在 service 层（D-04，不进 repo）：
   Statistics（5 服务）/ SearchXxxOptions（6 服务）/ GetTree / GetDeviceTypes 等
   floor 的软删恢复 / 异步同步 / 楼层计数编排
```

### Recommended Project Structure（改动面）

```
internal/services/base/
├── service.go             # 重写：删 Query/WhereCondition/Repository[T]；新增 Scope/PageParams/scope 版六方法
├── service_test.go         # 新建：泛型契约锁值（参考 pagination_test.go 的 Stability+Count 双锁）
├── base_80_05_test.go      # 适配重写：Query DSL 测试 → scope 测试；BatchDelete 空 ids 期望 BadRequest → nil
└── list_request.go         # 不动（54 文件 239 处引用的真资产）

internal/services/operations/
├── pagination_helper.go    # extractPagination/calculateOffset/extractSortRequest 视 map 服务去留修剪；extractInt/StringParam 保留（F1）
├── <11 个 service>.go      # 管道删除、filter 平移为 scope 函数、repo 组合
└── crud_services_test.go 等 # 保持全绿（行为不变性的现成验证器）

internal/api/v1/operations/
├── workstation_handler.go  # 仅 List + SearchWorkstationOptions 2 处 bind（D-05）
└── requests/workstation_requests.go  # +3 字段（F2）
```

### Pattern 1: filter → scope 平移（保持条件逐字不变）

```go
// 迁移前（door_service.go buildListQueryFromRequest）：
//   query := s.db.WithContext(ctx).Model(&operationsmodels.Door{})
//   if req.FloorID != "" { query = query.Where("floor_id = ?", req.FloorID) }
//   if req.DoorType != "" { query = query.Where("type = ?", req.DoorType) }

// 迁移后（条件字符串逐字保留）：
func (s *doorService) filterScope(req requests.DoorListRequest) base.Scope {
    return func(db *gorm.DB) *gorm.DB {
        if req.FloorID != "" {
            db = db.Where("floor_id = ?", req.FloorID)
        }
        if req.DoorType != "" {
            db = db.Where("type = ?", req.DoorType)
        }
        return db
    }
}

func (s *doorService) List(ctx context.Context, req requests.DoorListRequest) (*base.PageResult, error) {
    current, pageSize := req.GetPagination()
    return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
        s.filterScope(req),
        base.SortScopeWithTail(req.BaseListRequest, doorAllowedSortFields, "created_at DESC"),
    )
}
```

### Pattern 2: JOIN+Select 服务（scope 全量前置，含 workstation/floor/infopoint/room_device/server_room）

```go
// workstation：joins scope 只在 Find 生效 Select？不 —— spike 实证 Select 在 Count 时被替换为 count(*)，
// Find 时恢复。所以单 scope 列表即可（server_room 现状本就 Select+Joins 先于 Count）。
func (s *workstationService) joinScope() base.Scope {
    return func(db *gorm.DB) *gorm.DB {
        return db.Select(workstationJoinSelect).Joins(workstationJoinClause) // 常量原样平移
    }
}
// GetByID（带 join 的 4 个服务）：
got, err := s.repo.GetByID(ctx, id, s.joinScope()) // repo 内 scope 非空时用 <TableName()>.id = ? 限定
```

### Pattern 3: map → scope（building/floor/asset，接口签名不变）

```go
// 接口签名不动：List(ctx context.Context, params map[string]interface{}) (*base.PageResult, error)
func (s *assetService) List(ctx context.Context, params map[string]interface{}) (*base.PageResult, error) {
    pagination := extractPagination(params) // 现有 helper 保留（F1）
    return s.repo.List(ctx, base.PageParams{Current: pagination.Current, PageSize: pagination.PageSize},
        constantScope(func(db *gorm.DB) *gorm.DB { return db.Where("component_type IS NULL") }), // 恒定过滤
        s.filterScope(params), // 原 applyFilters 平移
        base.SortScope(*sortReq, assetAllowedSortFields, "created_at DESC"), // B 型排他默认
    )
}
```

### Anti-Patterns to Avoid

- **在 repo 里 clamp/归一化分页**：会统一三种不同 clamp 语义（P5），破坏零行为变更
- **在 repo 里加默认排序**：11 个服务的默认序各不相同（`order_num ASC` / `created_at DESC` / `sys_workstation.created_at DESC` / 无），排序必须由 service scope 提供
- **用 `fmt.Sprintf` 拼条件字符串进 scope**：所有条件保持 `?` 占位符；ORDER BY 只能来自白名单 map value（`ResolveSort` 已保证）
- **迁移时顺手改 filter 条件/默认值/错误文案**：条件字符串、错误类型（`apperrors.*`）、`PageResult` 字段值必须逐字保留
- **为 mock 需要 revival `Repository[T]` interface**：D-02 明确禁止；测试打真 sqlite

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 分页+过滤+计数管道 | 每服务重复 Count/Offset/Limit/Find | `GORMRepository[T].List` | 11 处重复正是本 phase 目标；GORM 链语义陷阱（Count/Select/ORDER BY）已集中验证一次 |
| 排序白名单解析 | 手写 switch/字符串拼接 | `base.ApplySort`/`ResolveSort` + 新 SortScope helper | SQL 注入安全设计已内建（白名单 value 非用户输入） |
| 错误分类 | 自写 `strings.Contains("duplicate")` 判定 | `base.IsNotFound`/`IsDuplicate`/`WrapError` | 既有且被 base_80_05_test 锁定；floor 的 `isDuplicateKeyError` 是另一码事（PG SQLSTATE 结构化判定），**不要合并** |
| 泛型契约测试基建 | 手写 sqlite 脚手架 | 复用 `base_80_05_test.go` 的 `newBasRepo8005` 模式（t.TempDir 文件库 + AutoMigrate + BeforeCreate uuid） | v1.27 基建现成 |

**Key insight:** 这个领域最难的不是写 List，而是**GORM 链的隐式语义**（Count 的 Select 替换/恢复、ORDER BY 剥离、Table vs Model 的软删除差异、共享语句复用）。研究已用源码 + spike 一次性验证，planner 应把这些验证固化为 `base/service_test.go` 的契约测试，之后任何改动都能立即发现语义漂移。

## Common Pitfalls

### P1: BatchDelete 空 ids 语义冲突（必踩）
**What goes wrong:** `base.GORMRepository.BatchDelete` 现对空 ids 返回 `apperrors.BadRequest`；但 11 个服务全部返回 **nil**（door/wall/floor_plan_text 显式 `if len(ids)==0 {return nil}`，其余靠 `IN (空)` no-op；`workstation_floor_code_77_03_test.go:173` 有 `require.NoError(t, svc.BatchDelete(ctx, nil))` 锁定）。
**Why it happens:** base 是 Phase 80-05 写的独立设计，从未对过真实消费者。
**How to avoid:** repo.BatchDelete 空 ids 改为返回 nil；`base_80_05_test.go` 的 `TestBas8005_BatchDelete` 期望同步反转。
**Warning signs:** 迁移后批量删除空选择返回 400。

### P2: building/asset 软删除 Count 分歧（F3）
见上文。repo 统一 `Model(new(T))` 会收紧 Total 语义。checkpoint 确认或保留 Table 链。

### P3: door/wall 的复合排序 vs 其余 9 服务的排他默认（排序双语义）
**What goes wrong:** 用同一个排序 helper 复刻所有服务，door/wall 的 `created_at DESC` 尾随 tiebreaker 丢失（现状：`ORDER BY type ASC, created_at DESC`）。
**How to avoid:** 分 A/B 两型 helper（见排序语义分型表）；迁移 door/wall 后用多行同列名数据验证次级排序。
**Warning signs:** 同 created_at 的两行在用户排序下顺序翻转。

### P4: `PageResult.List` 类型断言
**What goes wrong:** repo 若装 `*[]T`，`crud_services_test.go:179` 的 `page.List.([]operationsmodels.OpsServerRoom)` panic。
**How to avoid:** repo 统一装值切片 `[]T`。

### P5: 三种分页 clamp 语义并存
| 服务族 | 下限 | 上限 | 附加 |
|--------|------|------|------|
| 7 个 typed 服务（`requests.PaginationParams.GetPagination`） | pageSize<10→10 | >100→100（MaxListPageSize） | current<1→1 |
| workstation/building/asset（`extractPagination`+`clampPageSize`） | <10→10 | >10000→10000（MaxOptionsPageSize） | current 不 guard（0 → 负 offset，latent） |
| floor（内联断言） | 无 | 无 | 任意值直传 |

**How to avoid:** repo 直信入参；各服务保持原提取路径；D-05 workstation 改用 GetPagination 的语义收紧需 checkpoint（见 D-05 清单第 6 点）。

### P6: workstation `status/type` 的 -1 跳过语义
typed 化后必须保留 `>= 0` 才过滤的判断（`status:-1` 显式跳过有测试锁定）。`StatusRequest.GetStatus(-1)` 默认值技巧即可无损保留。

### P7: floor 接口签名被装饰器锁定
`floorCacheService`（Phase 92 文件）实现同一 `FloorService` 接口并委托。改 floor 方法签名 = 强制触碰 Phase 92 文件。**floor 所有公开方法签名保持不变**（含 `List(ctx, map)`）。

### P8: room_device/server_room 的 scope 顺序敏感
room_device List 的 Joins 必须在引用 `ops_buildings.org_id` 的 filter 之前应用；server_room 的 orgId 早退分支发生在 Joins 之后。scope 按**原代码出现顺序**平移，不要重排。

### P9: handler 测试的 76 处 PageResult 引用
若选 alias 方案则零影响；若选硬删除则 12 个测试文件全部要机械替换（见 F4）。

### P10: 临时文件构建炸弹
根目录 `temp_*.go`/`test_*.go` 会导致 `main redeclared`（CLAUDE.md 既有警告）。本 phase 大量新建/改写文件，每个 plan 收尾必须 `go build ./...`（CLAUDE.md 强制）。

## Code Examples

### 泛型契约测试（`base/service_test.go` 推荐形态，参考既有 `newBasRepo8005` 脚手架）

```go
// Source: internal/services/base/base_80_05_test.go（既有模式）+ 本次 spike 验证
func TestBase91_RepoContract(t *testing.T) {
    repo, db := newBasRepo8005(t) // 复用既有 sqlite 脚手架
    ctx := context.Background()

    // 契约 1: scope 空 → 全表分页，List 为值切片
    page, err := repo.List(ctx, base.PageParams{Current: 1, PageSize: 2})
    require.NoError(t, err)
    _, ok := page.List.([]basRepoRow8005)
    require.True(t, ok, "List 必须是值切片")

    // 契约 2: Joins+Select scope 下 Count 正确且 Find 保留 select（spike 实证）
    // 契约 3: 复合排序（两 Order 顺序 = SQL 顺序）
    // 契约 4: BatchDelete(nil) → nil（P1 语义反转锁值）
    // 契约 5: GetByID 无 scope → WHERE id = ?；有 scope → <table>.id = ?
    _ = db
}
```

### workstation typed request 接线（D-05）

```go
// Source: requests/common.go:16-29 + workstation_service.go:240-292 现状语义
type WorkstationListRequest struct {
    PaginationParams
    StatusRequest
    BuildingID string `json:"buildingId"` // 保留声明；服务从不消费（锁行为）
    FloorID    string `json:"floorId"`
    Name       string `json:"name"`
    Code       string `json:"code"`     // 保留声明；服务从不消费（锁行为）
    FloorCode  string `json:"floorCode"` // 新增（F2）
    Type       *int   `json:"type"`      // 新增；nil 或 -1 → 跳过过滤
    OrgID      string `json:"orgId"`     // 新增
}

// handler：bind 失败降级为零值请求（与现状 params=空 map 一致）
var req requests.WorkstationListRequest
if err := c.ShouldBindJSON(&req); err != nil {
    req = requests.WorkstationListRequest{}
}
```

## Runtime State Inventory

> 重构 phase 按规程显式回答；本 phase 无标识符改名落盘问题。

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — 六方法名/签名不落库；无表名/列名/键名变更 [VERIFIED: 改造面仅为 Go 函数组织] | none |
| Live service config | None — 无 n8n/外部服务配置引用这些 service 符号 | none |
| OS-registered state | None — 无计划任务/服务注册涉及 | none |
| Secrets/env vars | None — 不触碰 `SM4_KEY`/`BAIDU_MAP_AK` 等任何配置键 | none |
| Build artifacts | None — 纯源码重构，无 codegen/egg-info/二进制产物依赖旧符号；唯一注意：`configs/config.yaml` 等不涉及 | none（`go build ./...` 每 plan 收尾） |

## LOC 净减估算（分项，诚实口径）

| 来源 | 估算净减 | 备注 |
|------|---------|------|
| door + wall（countRecords/fetchRecords/buildListQuery 全套删除） | ~90 | 最小同构样本，删除面最大 |
| floor_plan_text | ~30 | |
| dedicated_line | ~20 | filter 平移守恒，仅删管道 |
| server_room | ~25 | |
| room_device | ~25 | |
| infopoint | ~35 | |
| asset | ~35 | |
| building | ~35 | |
| floor | ~23 | 业务编排（恢复/异步/计数）保留，仅 GetByID/List 管道 |
| workstation | ~35 | pilot；4 个独有方法不动（D-04） |
| **services 小计** | **~350** | |
| base/service.go 改造（删 DSL ~85L，增 scope 版 ~60L） | ~25 | |
| base_80_05_test.go 适配（旧 Query 测试删、scope 测试增） | ~30 | |
| pagination_helper.go 修剪（extractSortRequest/extractPagination 若 map 服务 List 全迁后） | 0~40 | F1 约束下部分保留 |
| pagination_helper_test.go 对应用例 | 0~115 | |
| **严格范围合计** | **~450-550** | **低于 D-07 的 800** |
| + `building_service_typesafe.go`(171L) + `building_handler_typesafe.go` + 对应测试块删除（零生产消费者死代码 [VERIFIED: router.go:546 仅用 NewBuildingService；typesafe 构造器仅测试引用]） | ~200-250 | **推荐纳入**（本身就是消除重复） |
| + building/floor typed 接线扩展（struct 已存在） | ~80-120 | 属 D-05 延伸，建议 checkpoint |
| **合计** | **~750-900** | 覆盖 ≥800 目标 |

**给 planner 的口径建议：** D-07 的 LOC 锚点达成依赖范围决策。推荐组合 = 严格迁移 + typesafe 死文件清理（合计 ~650-800，贴线）；如需保险再加 building/floor typed 扩展。SUMMARY 统计用 `git diff --stat` 删除行 − 新增行。

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | go test + testify（assert/require）+ glebarez/sqlite 内存库（v1.27 基建） |
| Config file | 无专门配置（标准 go test） |
| Quick run command | `go test ./internal/services/operations/... ./internal/services/base/...` |
| Full suite command | `go test ./...`（1688+ tests，v1.29 D-03 零回归底线）；每 plan 另跑 `go build ./...`（CLAUDE.md 强制） |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CRUD-REUSE-01 | GORMRepository 六方法契约（值切片 List、BatchDelete(nil)=nil、scope 组合、GetByID 表限定、排序双型 helper） | unit（泛型契约锁值，参考 Stability+Count 双锁模式） | `go test ./internal/services/base/ -run TestBase91 -v` | ❌ Wave 0（`base/service_test.go` 新建） |
| CRUD-REUSE-01 | 旧 base_80_05 测试适配 scope 形态 | unit | `go test ./internal/services/base/ -run TestBas8005 -v` | ✅（改写 `base_80_05_test.go`） |
| CRUD-REUSE-02 | workstation CRUD + 全部 List filter（name/floorId/floorCode/status -1 跳过/type/orgId）行为不变 | unit（行为锁，改写为 typed 调用） | `go test ./internal/services/operations/ -run TestImp77_Workstation -v` | ✅ 改写（`workstation_floor_code_77_03_test.go`） |
| CRUD-REUSE-02 | workstation handler List/SearchOptions bind（D-05） | unit（handler mock） | `go test ./internal/api/v1/operations/ -run Workstation -v` | ✅ 改写 2 个 mock 文件 |
| CRUD-REUSE-03..06 | 5 个 typed 服务 CRUD/过滤矩阵行为不变 | unit（回归即验证器） | `go test ./internal/services/operations/ -run "TestWall|TestDoor|TestServerRoom|TestDedicatedLine|TestInfoPoint" -v` | ✅（`crud_services_test.go` 全绿即行为不变证明） |
| CRUD-REUSE-03..05 | building/floor/asset 迁移行为不变 | unit | `go test ./internal/services/operations/ -run "TestAsset|TestBuilding" -v` | ⚠️ 部分（`asset_building_roomdevice_test.go` 覆盖 building/asset/room_device；floor 无直接测试 → Wave 0） |
| CRUD-REUSE-06 | room_device 迁移 | unit | 同上（含在 asset_building_roomdevice_test） | ✅ |
| CRUD-REUSE-07 | 每 service 迁移后包级全绿 | gate | `go test ./internal/services/operations/...` | ✅ 既有 |
| CRUD-REUSE-08 | handler e2e 0 回归 + LOC 统计 | gate + manual | `go test ./internal/api/v1/operations/...`；LOC 用 `git diff --stat` 记入 SUMMARY | ✅ / manual（理由：LOC 是构建时统计非运行时行为） |

### 行为不变性采样维度（Nyquist）

1. **每 service 迁移 commit**：包级 quick suite（< 30s）——现有 5 服务 CRUD 矩阵 + workstation 行为锁 + asset/building/roomdevice 测试是现成的行为采样器，**全绿即行为不变的直接证据**（同一批测试在迁移前后必须逐字节同结果）
2. **每 wave merge**：`go build ./...` + `go test ./...` 全量（AST 锁值防线 status_constants_test / operlog regression_test 全程绿）
3. **Phase gate**：全量 + `go vet`；handler 层测试包全绿充当「端到端 smoke」（本项目无独立 e2e harness，handler 测试经 mock service 覆盖 bind/response 路径；真正打 DB 的 e2e 由 service 级 sqlite 测试承担——两层拼起来即完整采样）

### Sampling Rate

- **Per task commit:** `go test ./internal/services/operations/... ./internal/services/base/...`
- **Per wave merge:** `go build ./... && go test ./...`
- **Phase gate:** full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/services/base/service_test.go` — 泛型契约锁值（六方法 + 排序双型 + P1/P4 语义），覆盖 CRUD-REUSE-01
- [ ] `internal/services/operations/floor_service` 直接 CRUD 测试缺失 — floor 仅被 workstation 测试间接使用；迁移 floor 前先补 service 级测试（含软删恢复/楼层计数路径），否则 floor 迁移无行为采样
- [ ] `internal/services/operations/floor_plan_text_service` 无 service 级测试（仅 handler mock）— 建议随 91-04 补一条 CRUD 冒烟
- [ ] Framework install：无需（既有基建）

## Security Domain

> security_enforcement 未显式关闭，按规程给出。本 phase 为内部重构，无新增攻击面；安全不变量如下。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes（迁移涉及查询构造路径） | 保持 `?` 占位符参数化；ORDER BY 仅来自白名单 map value（`base.ResolveSort` 安全设计）；`validateTableName` 白名单原样保留 |
| V4 Access Control | no（权限中间件层不触碰） | — |
| V2/V3/V6 | no（无认证/会话/密码学改动） | — |

### Known Threat Patterns for 此 stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL 注入（scope 改写时动态拼条件） | Tampering | 条件字符串逐字平移 + `?` 占位符；禁止 `fmt.Sprintf` 拼用户输入进 scope |
| ORDER BY 注入 | Tampering | `xxxAllowedSortFields` 白名单机制原样保留（db_col 来自 map value 非用户输入） |
| 软删除绕过 | Information Disclosure | repo 统一 `Model(new(T))` 后软删除过滤反而更完备（F3 是修复方向）；Unscoped 不引入 |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `building_service_typesafe.go`/`building_handler_typesafe.go` 可安全删除（零生产消费者） | LOC 估算 / F5 | 若有未 grep 到的动态引用会断编译——但 `go build` 会立即暴露，风险为零（编译期） |
| A2 | D-05 workstation 分页上限 10000→100 收紧可被用户接受（前端常规发送 ≤100） | D-05 清单 #6 | 若有下游脚本依赖大 pageSize 拉全量会被截断；建议 checkpoint:human-verify |
| A3 | building/asset Total 不再计软删行属可接受 bugfix | Pitfall P2 / F3 | 若业务依赖旧行为（Total 含软删）则列表分页数变化；建议 checkpoint:human-verify |
| A4 | GetByID 表限定用 Tabler 断言可行（11 模型均实现 TableName()） | repo 设计 | 个别模型若无 TableName() 会退化为 `id = ?` 在 join 下 ambiguous——已 grep 确认 11 个模型族 TableName 普遍存在，个别遗漏由 service_test 契约 5 锁定 |
| A5 | handler 测试充当 e2e smoke 已足够（无独立 e2e harness） | Validation | 若用户期望真浏览器级 smoke 则需手动验证步骤补充（CONTEXT 提到的「handler 端到端 smoke」按现有测试口径解释） |

## Open Questions (RESOLVED)

> 3 项均已在 planning 阶段解决并落到对应 plan/task（下附 RESOLVED 指针），保留原始问题描述以供溯源。

1. **D-05 分页上限收紧（A2）与 building/asset 软删 Total 修复（A3）是否需要用户显式确认？**
   - What we know: 两者都是「零行为变更」字面承诺下的微小语义变化，方向均正确
   - What's unclear: 用户对「字面零变更」的容忍度
   - Recommendation: 两个 plan checkpoint:human-verify，一次说清
   - **RESOLVED → 91-02 Task 3 + 91-03 Task 4**：两个 checkpoint:human-verify 已分别落 plan——A2（workstation 分页收紧 pageSize 上限 10000→100 + current<1→1）见 91-02「Checkpoint: workstation 分页语义收紧确认（A2）」；A3（building/asset List Total 不再计入软删除行）见 91-03「Checkpoint: building/asset Total 软删语义修复确认（F3/A3）」
2. **LOC ≥800 的范围决策（F5）**
   - What we know: 严格范围 450-550
   - What's unclear: typesafe 死文件清理是否算本 phase（属「消除重复」精神内，但不在 11 服务清单）
   - Recommendation: 纳入 91-04 收尾 plan；如讨论需要再确认
   - **RESOLVED → 91-04 Task 4**：typesafe 死文件清理已由 orchestrator 拍板纳入 91-04 Task 4（无需 checkpoint，91-04 must_haves truths 显式记录该拍板），作为 LOC 净减 ≥800 的补充来源（F5 组合 1）
3. **`extractIntParam`/`extractStringParam` 最终去留**
   - What we know: F1 证明严格范围无法全删（SearchXxxOptions + Statistics + 3 个 map 服务仍需）
   - What's unclear: 是否值得为删 2 个小 helper 把 SearchXxxOptions 改内联断言（负收益）
   - Recommendation: 保留 helper，plan 验收口径写「workstation 消费点清零」
   - **RESOLVED → 91-02 Task 1 + 91-04 Task 4**：采用推荐口径——extract 族 helper 本体保留（F1：3 个 map 服务 List + SearchXxxOptions + Statistics 仍需），验收口径 = 「workstation 消费点清零」：91-02 Task 1 第八步（List/SearchWorkstationOptions 零 extract 调用）+ 91-02 success_criteria 5；91-04 Task 4 修剪红线明确 extractPagination/extractSortRequest/extractIntParam/extractStringParam/clampPageSize 全部保留

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部 | ✓ | go.mod Go 1.24 [VERIFIED: CLAUDE.md + go build/test 本地运行成功] | — |
| gorm.io/gorm | repo 改造 | ✓ | v1.30.5 [VERIFIED: go.mod:34] | — |
| glebarez/sqlite | 测试 | ✓ | 既有 | — |
| git | LOC 统计/SUMMARY | ✓ | 既有 | — |

**Missing dependencies with no fallback:** 无
**Missing dependencies with fallback:** 无

## Sources

### Primary (HIGH confidence)
- 本地代码逐文件审读：`internal/services/base/{service,list_request,list_request_test,base_80_05_test}.go`；`internal/services/operations/` 11 个目标 service + pagination_helper(+test) + validation_helper + dropdown_helper + floor_cache_impl + building_service_typesafe + crud_services_test + workstation_floor_code_77_03_test + workstation_service_test + asset_building_roomdevice_test；`internal/api/v1/operations/`（workstation/building/floor/asset handler、requests/ 全包、12 个 PageResult 测试文件）；`internal/api/router.go` 接线；`internal/models/workstation.go`
- GORM v1.30.5 模块源码（`C:\Users\CPIC\go\pkg\mod\gorm.io\gorm@v1.30.5`）：`finisher_api.go` Count 实现（Select 替换/恢复、ORDER BY 剥离/恢复）、`chainable_api.go` Order/Scopes 实现、`callbacks.go` Execute（Model=Dest 解析与软删除注入）
- 运行时 spike（项目内 sqlite，5 项断言全部通过后已删除临时文件）：scope List with Joins+Select Count 正确性、Table vs Model 软删 Count 分歧（3 vs 2 实证）、Order 复合排序 SQL 断言、Count→Find 无污染、空 IN 删除 no-op

### Secondary (MEDIUM confidence)
- CONTEXT.md / REQUIREMENTS.md / STATE.md（计划文档；其中 3 处事实前提被本研究修正，见 F1/F2/F5）

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新增依赖，gorm 版本与测试基建均本地实证
- Architecture: HIGH — repo 签名与 List 管道经源码 + 运行时 spike 双重验证；11 服务形状逐文件盘点
- Pitfalls: HIGH — P1-P10 全部有代码行号或源码级依据，其中 P1/P3/P4 有测试锁定证据

**Research date:** 2026-09-04
**Valid until:** 2026-10-04（内部代码重构无外部快变面；若 main 上 operations 目录被其他 phase 改动需重验行号）
