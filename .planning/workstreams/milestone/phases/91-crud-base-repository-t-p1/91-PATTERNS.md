# Phase 91: CRUD 复用 base.Repository[T] - Pattern Map

**Mapped:** 2026-09-04
**Files analyzed:** 17（base 改造 1 + 测试 2 + helper 1 + 迁移 service 11 + handler 1 + requests 1，另含 1 个纯删除文件）
**Analogs found:** 16 / 17（`building_service_typesafe.go` 为纯删除目标，无需 analog）

> 本 phase 是纯内部重构：目标是把 11 个同构 CRUD service 的重复查询管道收敛到改造后的 `base.GORMRepository[T]`（D-01 gorm scope 函数式 + D-02 纯 struct 组合）。改造对象 `GORMRepository[T]` 零生产消费者，因此 **`internal/services/operations/door_service.go` 是迁移模板的最小同构样本**（本 PATTERNS 的核心 analog）。所有 filter 条件字符串、错误类型、JSON 契约必须**逐字保留**（零行为变更底线）。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/base/service.go` | service (generic repository) | CRUD | 自身重构（删 DSL 换 scope）；List 管道形状参考 `door_service.go` 的 countRecords/fetchRecords | exact (self) |
| `internal/services/base/service_test.go` (新建) | test | CRUD | `base_80_05_test.go`（sqlite 脚手架）+ `pkg/constants/pagination_test.go`（AST 锁值） | role-match |
| `internal/services/base/base_80_05_test.go` (适配) | test | CRUD | 自身（Query DSL 测试改写为 scope 测试 + BatchDelete 期望反转） | exact |
| `internal/services/operations/pagination_helper.go` | utility | transform | 自身修剪；`PageResult` 合并目标为 `base/service.go:38` | exact |
| `internal/services/operations/workstation_service.go` (pilot) | service | CRUD | `door_service.go`（同构骨架）+ RESEARCH D-05 接线清单 | exact + 扩展 |
| `internal/services/operations/building_service.go` / `floor_service.go` / `asset_service.go` | service | CRUD | `door_service.go`（骨架）；**List 保持 map 签名**（F1/P7 约束） | role-match |
| `internal/services/operations/{server_room,infopoint,dedicated_line,room_device,wall,floor_plan_text}_service.go` | service | CRUD | `door_service.go`（最小同构样本，7 个 typed 服务同族） | exact |
| `internal/api/v1/operations/workstation_handler.go` | controller | request-response | `door_handler.go:67-79`（typed bind）；降级语义来自自身现状 :139-152 | exact |
| `internal/api/v1/operations/requests/workstation_requests.go` | config (DTO) | request-response | `requests/door_requests.go` + `requests/common.go`（StatusRequest/GetStatus） | exact |
| `internal/services/operations/building_service_typesafe.go` (+handler_typesafe +测试块) | — | — | 纯删除（零生产消费者死代码，LOC 达标来源，见 RESEARCH F5/A1） | no analog (deletion) |

## Pattern Assignments

### `internal/services/base/service.go`（service, CRUD — D-01/D-02/D-03 改造对象）

**Analog:** 自身（保留骨架）+ `internal/services/operations/door_service.go`（List 管道的目标形状来源）

**现状要删除的部分**（`base/service.go`）：

- `Repository[T]` interface（:11-19）— D-02 明确删除，不 revival
- `Query`/`WhereCondition` DSL（:21-35）+ List 内的 switch 操作符管道（:88-107）— D-01 零消费者直接删除
- `BatchDelete` 空 ids 返回 `apperrors.BadRequest`（:139-141）— **语义反转为返回 nil**（P1：11 个服务全部 nil，`workstation_floor_code_77_03_test.go:173` 锁定 `BatchDelete(ctx, nil)` 为 NoError）

**现状要保留的部分**（逐字不动）：

```go
// base/service.go:56-78 — Create/Update/Delete/GetByID 四方法形状直接保留
func (r *GORMRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}
func (r *GORMRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}
func (r *GORMRepository[T]) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error
}
```

- `PageResult`（:38-43）— **成为全项目唯一定义**（D-03）
- `WrapError`/`IsNotFound`/`IsDuplicate`（:145-161）— 原样保留，`base_80_05_test.go:181-208` 有锁定测试
- `Update` 保持 Save 全量版；building/asset 的 `Omit("CreatedAt")` 变体与 workstation 的 First 回填**留 service 层**（各自有业务注释，不宜隐入通用 repo）

**List 的目标形状**（RESEARCH spike 实证版签名，planner 授权定稿）：

```go
// D-01: scope 函数式；D-02: service 直接组合 struct，无 interface
type Scope = func(*gorm.DB) *gorm.DB

type PageParams struct {
	Current  int
	PageSize int
}

// 关键内部管道（spike 已验证 Count 与 Joins+Select 共存、ORDER BY 剥离恢复、
// Count→Find 链复用无污染）：
func (r *GORMRepository[T]) List(ctx context.Context, page PageParams, scopes ...Scope) (*PageResult, error) {
	query := r.db.WithContext(ctx).Model(new(T)) // 注意：Model(new(T)) 统一起链（F3 软删 bugfix 方向）
	for _, s := range scopes {
		if s != nil {
			query = s(query)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []T // P4: 必须值切片 []T（crud_services_test.go:179 断言依赖）
	offset := (page.Current - 1) * page.PageSize
	if err := query.Offset(offset).Limit(page.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &PageResult{List: list, Total: total, Current: page.Current, PageSize: page.PageSize}, nil
}
```

**设计红线（Anti-Patterns，来自 RESEARCH）：**
- repo 内**不 clamp 分页**（三种 clamp 语义并存，P5）、**不加默认排序**（11 服务默认序各不相同）
- scope 内禁止 `fmt.Sprintf` 拼用户输入；条件保持 `?` 占位符
- GetByID 支持变长 scopes；scope 非空时用 Tabler 断言（`interface{ TableName() string }`）做 `<table>.id = ?` 表限定，空 scope 保持 `id = ?`

---

### 11 个迁移 service（service, CRUD）— 模板：`internal/services/operations/door_service.go`

**Analog:** `internal/services/operations/door_service.go`（148 行，最小同构样本）

**Imports/结构模式**（door_service.go:1-33）：

```go
import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// 接口形状（迁移后签名不变，除 workstation 2 方法）
type DoorService interface {
	Create(ctx context.Context, door *operationsmodels.Door) error
	Update(ctx context.Context, door *operationsmodels.Door) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.Door, error)
	List(ctx context.Context, req requests.DoorListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

// struct 组合迁移前: {db *gorm.DB, validator Validator}
// struct 组合迁移后: {repo *base.GORMRepository[operationsmodels.Door], db *gorm.DB, validator Validator}
//   （db 保留给 validate/populate 等业务逻辑；repo 承接六方法管道）
```

**排序白名单模式**（door_service.go:35-40）— 留在各 service，不动（CONTEXT Discretion）：

```go
var doorAllowedSortFields = map[string]string{
	"floorId":   "floor_id",
	"doorType":  "type",
	"createdAt": "created_at",
}
```

**List 迁移前**（door_service.go:69-96 — 本 phase 要消灭的样板形状）：

```go
func (s *doorService) List(ctx context.Context, req requests.DoorListRequest) (*PageResult, error) {
	query := s.buildListQueryFromRequest(ctx, req)
	query = base.ApplySort(query, req.BaseListRequest, doorAllowedSortFields)
	if req.OrderByColumn != "" {
		query = query.Order("") // 死代码：Order("") 是 no-op（GORM chainable_api 只在 v != "" 时 AddClause）
	}
	total, err := s.countRecords(query)   // ← 每服务重复定义
	if err != nil { return nil, err }
	current, pageSize := req.GetPagination()
	list, err := s.fetchRecords(query, current, pageSize, &[]operationsmodels.Door{}) // ← 每服务重复定义
	if err != nil { return nil, err }
	return &PageResult{List: list, Total: total, Current: current, PageSize: pageSize}, nil
}
// 另有 countRecords(:133-137) / fetchRecords(:140-148) / buildListQueryFromRequest(:119-130) 三个私有 helper —— 全套删除
```

**List 迁移后**（RESEARCH Pattern 1 — filter 条件字符串逐字平移进 scope）：

```go
func (s *doorService) filterScope(req requests.DoorListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.FloorID != "" { // 条件 "floor_id = ?" 与 door_service.go:122-124 逐字一致
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
		base.SortScopeWithTail(req.BaseListRequest, doorAllowedSortFields, "created_at DESC"), // A 型（door/wall 复合尾随）
	)
}
```

**BatchDelete 迁移前**（door_service.go:98-103）— 空 ids 返回 nil，迁移后直接委托 repo（repo 已反转为 nil 语义）：

```go
func (s *doorService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.Door{}, "id IN ?", ids).Error
}
```

**validate 前置钩子保留在 service**（door_service.go:42-47, 105-116）— `validateDoorRelations`（Validator 调用）在 repository 调用之前，迁移时保持该顺序。

**迁移分型速查（从 RESEARCH 盘点表提取，planner 分配 plan 时逐服务对照）：**

| 服务 | List 参数 | GetByID | 排序语义 | 特殊保留（不进 repo） |
|------|-----------|---------|----------|----------------------|
| door / wall | typed | plain | **A 型**复合尾随 | `Order("")` 死代码删除 |
| floor_plan_text | typed | plain | B 型 | Delete/BatchDelete 现用 `.Table(...)` 行为等价 |
| dedicated_line | typed | plain | B 型 | 11 个 filter 平移；populateRoomNames 回填留 service |
| server_room | typed | plain | B 型 | Select+Joins 在 Count 之前（现状已证可行）；orgId 早退留 service |
| infopoint | typed | **JOIN** | B 型（默认 `ops_info_points.created_at DESC`） | populateRedundantFields 留 service |
| room_device | typed | **JOIN** | B 型 | **scope 顺序敏感**：Joins 必须在引用 joined 表的 filter 之前（P8） |
| workstation (pilot) | map → **D-05 typed** | **JOIN**(6 表) | B 型 | Statistics/SearchOptions/GetWorkstationDeptOptions/BatchUpdatePositions 留 service（D-04） |
| building | **map（签名不变）** | plain | B 型（默认 `order_num ASC`） | validateOrg/validateNameUnique 留 service；**F3 软删 Count 分歧 → checkpoint** |
| floor | **map（签名不变，P7 装饰器锁定）** | **JOIN**(buildings+sys_files) | B 型 | 软删恢复/异步同步/楼层计数整体留 service，仅 GetByID/List 管道进 repo；分页为无 clamp 内联断言 |
| asset | **map（签名不变）** | plain | B 型 | `component_type IS NULL` 恒定过滤必须进 scope；validateDept/validateUser/Omit("CreatedAt") 留 service；**F3 → checkpoint** |

---

### `internal/api/v1/operations/workstation_handler.go`（controller, request-response — D-05 仅 2 处 bind）

**Analog:** `internal/api/v1/operations/door_handler.go:67-79`（typed bind 目标形状）+ 自身现状（降级语义基准）

**typed bind 目标形状**（door_handler.go:67-71）：

```go
func (h *DoorHandler) List(c *gin.Context) {
	var req requests.DoorListRequest
	if !handleJSONBinding(c, &req) {
		return
	}
	result, err := h.service.List(c.Request.Context(), req)
	// ...
}
```

**现状 map bind（要改写的 2 处）— 注意降级语义必须保留**（workstation_handler.go:139-152 与 :90-101）：

```go
// List（:139-152）现状：bind 失败 → 降级空 map 继续查询（不是 400）
func (h *WorkstationHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = make(map[string]interface{})
	}
	result, err := h.service.List(c.Request.Context(), params)
	// ...
}

// SearchWorkstationOptions（:90-101）现状同型
```

**迁移后 bind**（RESEARCH D-05 清单 #3 — typed 化但保留降级语义，不用 handleJSONBinding 的 400 路径）：

```go
var req requests.WorkstationListRequest
if err := c.ShouldBindJSON(&req); err != nil {
	req = requests.WorkstationListRequest{} // 零值 request = 空过滤全列表，与现状空 map 一致
}
```

---

### `internal/api/v1/operations/requests/workstation_requests.go`（config/DTO, request-response — D-05 + F2 扩 3 字段）

**Analog:** `requests/door_requests.go`（结构模板）+ `requests/common.go`（嵌入基类与语义方法）

**typed request 结构模板**（door_requests.go:3-9）：

```go
type DoorListRequest struct {
	PaginationParams        // 嵌入分页参数（内嵌 base.BaseListRequest，见 common.go:11-13）
	StatusRequest           // 嵌入状态筛选
	FloorID          string `json:"floorId"`
	DoorType         string `json:"doorType"`
}
```

**现状**（workstation_requests.go:4-11）已有 `PaginationParams` + `StatusRequest` + `BuildingID`/`FloorID`/`Name`/`Code`。**目标新增 3 字段**（RESEARCH F2 + D-05 清单 #1）：

```go
type WorkstationListRequest struct {
	PaginationParams
	StatusRequest
	BuildingID string `json:"buildingId"` // 保留声明；服务从不消费（锁行为，注释说明）
	FloorID    string `json:"floorId"`
	Name       string `json:"name"`
	Code       string `json:"code"`     // 保留声明；服务从不消费（锁行为）
	FloorCode  string `json:"floorCode"` // 新增（F2）
	Type       *int   `json:"type"`      // 新增；nil 或 -1 → 跳过过滤
	OrgID      string `json:"orgId"`     // 新增
}
```

**status/type 的 -1 跳过语义必须保留**（P6）— 依赖 `StatusRequest.GetStatus(defaultValue)` 默认值技巧（common.go:54-59）：

```go
// common.go:54-59 — typed 化后用 GetStatus(-1) 无损复刻 map 路径的 extractIntParam(params,"status",-1)
func (s *StatusRequest) GetStatus(defaultValue int) int {
	if s.Status != nil {
		return *s.Status
	}
	return defaultValue
}
// service 侧：if st := req.GetStatus(-1); st >= 0 { scope 过滤 }（status:-1 显式跳过有测试锁定）
```

**分页归一化**（common.go:16-29 `GetPagination`）：typed 路径 clamp 上限为 `MaxListPageSize`(100)，与 map 路径 `clampPageSize` 的 `MaxOptionsPageSize`(10000) 不同 — workstation 切换属语义收紧，需 checkpoint（RESEARCH A2 / D-05 清单 #6）。

---

### `internal/services/operations/pagination_helper.go`（utility, transform — D-03/D-05 清理对象）

**Analog:** 自身 + `internal/services/base/service.go:38`（合并目标）

**PageResult 双定义合并**（pagination_helper.go:16-22 与 base/service.go:38-43 字段/JSON tag 完全一致）— **用 type alias，不做硬删除**（F4：76 处测试引用 + floor_cache_impl.go 等 2 个 Phase 92 文件零改动）：

```go
// operations 侧改为：
type PageResult = base.PageResult // alias 下两种写法同一类型，唯一 struct 定义在 base（D-03 本质达成）
```

**extractXxxParam 族去留**（F1 修正后口径）：

- `extractPagination`（:25-34）/ `extractSortRequest`（:39-49）/ `extractIntParam`（:52-60）/ `extractStringParam`（:63-68）/ `clampPageSize`（:75-77）/ `calculateOffset`（:80-82）— **helper 本体保留**（3 个 map 服务 List + 全部 SearchXxxOptions + Statistics 仍需）
- 验收口径为「**workstation 消费点清零**」，不是 helper 函数删除
- `operations.PaginationParams`（:11-14，本地 struct）与 `requests.PaginationParams`（common.go:11-13，嵌 BaseListRequest）是两个不同类型，合并时注意不要混淆

---

### `internal/services/base/service_test.go`（新建 test, CRUD — 泛型契约锁）

**Analog A（sqlite 脚手架）:** `internal/services/base/base_80_05_test.go:27-55`

```go
// 测试行模型:嵌 BaseModel + 业务字段；TableName 显式声明（GetByID 表限定契约依赖 Tabler 断言）
type basRepoRow8005 struct {
	models.BaseModel
	Name string `gorm:"size:100;not null"`
	Age  int    `gorm:"default:0"`
}
func (basRepoRow8005) TableName() string { return "bas_repo_rows_8005" }

// 脚手架:glebarez sqlite t.TempDir 文件库 + AutoMigrate + Cleanup；零 sleep、零 t.Parallel
func newBasRepo8005(t *testing.T) (*GORMRepository[basRepoRow8005], *gorm.DB) {
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "base8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { /* close */ })
	require.NoError(t, gormDB.AutoMigrate(&basRepoRow8005{}))
	return NewGORMRepository[basRepoRow8005](gormDB), gormDB
}
```

**Analog B（AST 锁值双锁模式）:** `pkg/constants/pagination_test.go:60-97` — `TestPaginationConstantStability`（锁名→值映射，双向：缺常量报错 + 多余常量报错）+ `TestPaginationConstantCount`（锁常量个数）。`base/service_test.go` 锁泛型契约时参考此「Stability + Count」双锁结构。

**新测试必须覆盖的契约点**（RESEARCH Code Examples + P1/P4/P8）：
1. 空 scope → 全表分页，`page.List` 为值切片 `[]T`（P4）
2. Joins+Select scope 下 Count 正确且 Find 保留 select（spike 实证）
3. 复合排序：两个顺序 `Order()` = SQL `ORDER BY a, b`
4. `BatchDelete(nil)` → **nil**（P1 语义反转锁值，与旧 base_80_05 期望相反）
5. GetByID 无 scope → `WHERE id = ?`；有 scope → `<TableName()>.id = ?`（A4）
6. 软删除：Table 链 vs Model 链 Count 分歧的回归证明（F3）

---

### `internal/services/base/base_80_05_test.go`（适配 test）

**Analog:** 自身。改写点：
- `TestBas8005_List_Paged`（:88-155）的 `&Query{Where/OrderBy/Offset/Limit}` 调用 → scope + PageParams 形态（操作符覆盖用例中 `WhereCondition` DSL 部分删除，改 scope 组合用例）
- `TestBas8005_BatchDelete`（:158-178）空切片期望**反转**：`require.Error` + `CodeParamError`（:174-177）→ `require.NoError`
- 脚手架 `newBasRepo8005`/`seedBasRow8005`（:37-55）保留复用（`service_test.go` 也用）

---

### `internal/services/operations/building_service_typesafe.go`（纯删除）

**No analog.** 零生产消费者死代码（`router.go:546` 只用 `NewBuildingService`；typesafe 构造器仅测试引用，RESEARCH F5/A1 [VERIFIED]）。随同删除 `building_handler_typesafe.go` + 对应测试块（~200-250 行），是 D-07 LOC ≥800 的推荐达标来源。删除后 `go build ./...` 立即暴露任何遗漏引用（编译期零风险）。

## Shared Patterns

### 1. filter → scope 平移（逐字保留条件）
**Source:** `door_service.go:119-130`（迁移前）→ RESEARCH Pattern 1（迁移后）
**Apply to:** 全部 11 个 service 的 List/GetByID 管道
条件字符串（`"floor_id = ?"`、`"type = ?"`、`"component_type IS NULL"` 等）逐字平移；禁止重排 filter 顺序（room_device Joins 顺序敏感，P8）；禁止顺手改默认值/错误文案。

### 2. typed request 嵌入模式（PaginationParams + StatusRequest）
**Source:** `requests/common.go:11-59` + `requests/door_requests.go:3-9`
**Apply to:** `workstation_requests.go` 扩字段 + workstation service 接口 typed 化
嵌入即获得 current/pageSize/orderByColumn/isAsc 顶层 JSON 字段与 `GetPagination()`/`GetStatus(-1)` 语义方法。

### 3. 排序白名单 + ApplySort（不动资产）
**Source:** `internal/services/base/list_request.go:45-92`（`ResolveSort`/`ApplySort`）
**Apply to:** 全部 11 个 service（`xxxAllowedSortFields` 留在各 service）
101 处引用的真资产，本期零改动。若 base 新增 `SortScope`/`SortScopeWithTail` helper（RESEARCH 推荐的 B 型/A 型），底座必须复用 `ResolveSort`，不重写白名单逻辑。

### 4. 空值/零值语义族（零行为变更的关键锁点）
- `BatchDelete(nil)` → nil（Source: `door_service.go:99-101`；repo 反转 :139-141）
- `status/type: -1` → 跳过过滤（Source: `common.go:54-59` GetStatus 默认值技巧）
- handler bind 失败 → 零值请求继续（Source: `workstation_handler.go:141-143` 降级语义）

### 5. 错误分类 helpers（不动）
**Source:** `base/service.go:145-161`（`WrapError`/`IsNotFound`/`IsDuplicate`）
**Apply to:** repo 与 service 迁移全程原样保留；floor 的 `isDuplicateKeyError`（PG SQLSTATE 结构化判定）是独立机制，**不要合并**。

### 6. 测试基建（sqlite in-memory 真库，无 mock）
**Source:** `base_80_05_test.go:37-55`（脚手架）+ `pkg/constants/pagination_test.go:60-97`（AST 双锁）
**Apply to:** `base/service_test.go` 新建、`base_80_05_test.go` 适配、每服务迁移后 `crud_services_test.go` 全绿即行为不变证明（D-02：无 repository mock，测试打真库）。

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/operations/building_service_typesafe.go`（删除） | service (dead) | CRUD | 纯删除目标，无模式可提取；删除安全性由编译期保证（A1） |
| building/floor/asset 的 map→scope 转换细节 | service | CRUD | 代码库无「map 参数 service + scope」先例（其余 typed），形状按 RESEARCH Pattern 3 新建；但骨架仍套 door_service.go |

## Metadata

**Analog search scope:** `internal/services/base/`, `internal/services/operations/`, `internal/api/v1/operations/`（含 `requests/`）, `pkg/constants/`
**Files read in full:** 11（base/service.go, base_80_05_test.go, list_request.go, door_service.go, pagination_helper.go, workstation_requests.go, door_requests.go, requests/common.go, pagination_test.go + workstation_handler.go / door_handler.go 定向区段）
**Pattern extraction date:** 2026-09-04
**行号有效性:** 基于 main @ 917428a；若 operations 目录被其他 phase 改动需重验（RESEARCH valid until 2026-10-04）
