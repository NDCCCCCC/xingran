# Phase 103: 缓存闭包收敛 base 单一权威 - Pattern Map

**Mapped:** 2026-09-07
**Files analyzed:** 4 service files + 1 invariants test + 1 cache impl example + base API
**Analogs found:** 7 / 8

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/mac_history_query_service.go` | service | CRUD + cache | `internal/services/system/user_cache_impl.go` | role-match |
| `internal/services/mac_history_heatmap_service.go` | service | CRUD + cache | `internal/services/system/user_cache_impl.go` | role-match |
| `internal/services/asset/reconciliation_service.go` | service | CRUD + cache | `internal/services/system/user_cache_impl.go` | role-match |
| `internal/services/rpa/selector_learner.go` | service | CRUD + cache | `internal/services/system/user_cache_impl.go` | role-match |
| `internal/services/system/cache_invariants_92_test.go` | test | AST scan | (扩口宿主,自己) | exact |
| `internal/services/base/cache_functions.go` | utility | request-response | (base API 自身) | exact |
| `internal/services/base/cache_provider.go` | interface | — | (base 自身) | exact |

---

## Pattern Assignments

### `internal/services/mac_history_query_service.go` (service, CRUD + cache)

**迁移类型:** 行为等价重构（字段类型 + 4 处 GetOrSet 位点 + vendor cache-aside）

**迁移后目标形态参照:** `internal/services/system/user_cache_impl.go:17-21` + `base.GetOrSetJSON` 调用模式

**当前 struct 字段（:138-143）：**
```go
type macHistoryQueryServiceImpl struct {
    db         *gorm.DB
    cache      cache.Cache    // vendor lookup 专用，nil-safe
    dataCache  *DataCacheService  // 三处 GetOrSet 使用
    perfConfig *CacheConfigService
}
```

**迁移后 struct 字段（目标）：**
```go
type macHistoryQueryServiceImpl struct {
    db         *gorm.DB
    cache      base.CacheProvider  // 统一接口，替换 cache.Cache
    perfConfig *CacheConfigService
    // dataCache 字段删除（已迁移到 base.GetOrSetJSON）
}
```

**迁移后构造函数（目标）：**
```go
// NewMACHistoryQueryServiceWithCache 入参从 *DataCacheService 改为 base.CacheProvider
func NewMACHistoryQueryServiceWithCache(db *gorm.DB, cache base.CacheProvider, perfConfig *CacheConfigService) MACHistoryQueryService {
    return &macHistoryQueryServiceImpl{db: db, cache: cache, perfConfig: perfConfig}
}
```

**位点 A — vendor lookup (:259-281)：**

**当前代码（手写 cache-aside）：**
```go
// internal/services/mac_history_query_service.go:259-285
if s.cache != nil {
    vendorName, err := s.cache.Get(ctx, cacheKey)  // cache 是 cache.Cache
    if err == nil && vendorName != "" { return vendorName, nil }
    // ...
    _ = s.cache.Set(ctx, cacheKey, "Unknown Vendor", 24*time.Hour)
}
// DB 查询 + 缓存写入
```

**迁移后（使用 `base.GetOrSetJSON[T]`，D-103-5 保留 "Unknown Vendor" 占位）：**
```go
vendorName, err := base.GetOrSetJSON[string](ctx, s.cache, cacheKey, 24*time.Hour, func() (string, error) {
    var vendor models.MACOUIVendor
    if err := s.db.WithContext(ctx).Where("oui_prefix = ?", oui).First(&vendor).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return "Unknown Vendor", nil  // D-103-5 占位保留
        }
        return "", err
    }
    return vendor.VendorName, nil
})
// D-103-6 严格语义：cache 写失败即返回 error
if err != nil { return "", err }
return vendorName, nil
```

**位点 B — QueryPortHistory (:304-318)：**

**当前代码（`interface{}` 闭包式 GetOrSet）：**
```go
// internal/services/mac_history_query_service.go:305-318
if s.dataCache != nil {
    var cached MACHistoryQueryResult
    cacheKey, keyErr := BuildMACQueryCacheKey("port-history", req)
    if keyErr == nil {
        err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
            return s.queryPortHistoryFromDB(ctx, req)
        })
        if err == nil { return &cached, nil }
        applogger.Warnf("[MAC缓存] port-history 走直查: %v", err)
    }
}
return s.queryPortHistoryFromDB(ctx, req)
```

**迁移后（`base.GetOrSetJSON[T]` 严格语义）：**
```go
// 迁移后（base.GetOrSetJSON 严格语义，D-103-6）：
cacheKey, keyErr := BuildMACQueryCacheKey("port-history", req)
if keyErr != nil {
    return s.queryPortHistoryFromDB(ctx, req)
}
cached, err := base.GetOrSetJSON[MACHistoryQueryResult](ctx, s.cache, cacheKey, s.perfCacheTTL(), func() (MACHistoryQueryResult, error) {
    return s.queryPortHistoryFromDB(ctx, req)
})
if err != nil { return nil, err }
return &cached, nil
```

**位点 C — QueryDeviceHistory (:429-443)：** 同位点 B 形态，替换 `queryDeviceHistoryFromDB`

**位点 D — QueryConnectionStats (:832-846)：** 同位点 B 形态，替换 `queryConnectionStatsFromDB`

**TTL 来源（保持不变）：**
```go
// internal/services/mac_history_query_service.go:155-166
func (s *macHistoryQueryServiceImpl) perfCacheTTL() time.Duration {
    const fallback = 5 * time.Minute
    if s.perfConfig == nil { return fallback }
    d := s.perfConfig.GetDuration(MACPerfConfigCacheTTLSeconds)
    if d <= 0 { return fallback }
    return d
}
```

**结论:** 字段 `cache cache.Cache` → `cache base.CacheProvider`；`dataCache *DataCacheService` 删除；4 处 `s.dataCache.GetOrSet(ctx, key, &result, ttl, func() (interface{}, error){...})` → `base.GetOrSetJSON[T](ctx, s.cache, key, ttl, func() (T, error){...})`

---

### `internal/services/mac_history_heatmap_service.go` (service, CRUD + cache)

**迁移类型:** 行为等价重构（字段类型 + 1 处 GetOrSet + fallback wrapper）

**当前 struct 字段（:50-54）：**
```go
type macHistoryHeatmapServiceImpl struct {
    db         *gorm.DB
    dataCache  *DataCacheService
    perfConfig *CacheConfigService
}
```

**迁移后 struct 字段（目标）：**
```go
type macHistoryHeatmapServiceImpl struct {
    db         *gorm.DB
    cache      base.CacheProvider  // 替换 dataCache
    perfConfig *CacheConfigService
}
```

**迁移后构造函数（目标）：**
```go
func NewMACHistoryHeatmapService(db *gorm.DB, cache base.CacheProvider, perfConfig *CacheConfigService) MACHistoryHeatmapService {
    return &macHistoryHeatmapServiceImpl{db: db, cache: cache, perfConfig: perfConfig}
}
```

**位点 E — QueryHeatmap (:113-127)：**

**当前代码：**
```go
// internal/services/mac_history_heatmap_service.go:113-127
if s.dataCache != nil {
    var cached HeatmapResult
    cacheKey, keyErr := BuildMACQueryCacheKey("heatmap", req)
    if keyErr == nil {
        err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
            return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
        })
        if err == nil { return &cached, nil }
        applogger.Warnf("[MAC热力图] 走直查: %v", err)
    }
}
return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
```

**迁移后（D-103-9 fallback wrapper helper）：**
```go
// mac_history_heatmap_service.go 新增 wrapper helper
func (s *macHistoryHeatmapServiceImpl) getHeatmapWithCache(ctx context.Context, cacheKey string, ttl time.Duration, req *HeatmapQuery, startTime, endTime time.Time) (*HeatmapResult, error) {
    result, err := base.GetOrSetJSON[*HeatmapResult](ctx, s.cache, cacheKey, ttl, func() (*HeatmapResult, error) {
        return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
    })
    if err != nil {
        // D-103-9: fallback 直查语义
        applogger.Warnf("[MAC热力图] 走直查: %v", err)
        return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
    }
    return result, nil
}
```

**QueryHeatmap 改写后：**
```go
func (s *macHistoryHeatmapServiceImpl) QueryHeatmap(ctx context.Context, req *HeatmapQuery) (*HeatmapResult, error) {
    // ... 时间解析 + topN 处理 ...
    cacheKey, keyErr := BuildMACQueryCacheKey("heatmap", req)
    if keyErr != nil {
        return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
    }
    return s.getHeatmapWithCache(ctx, cacheKey, s.perfCacheTTL(), req, startTime, endTime)
}
```

**结论:** `dataCache *DataCacheService` → `cache base.CacheProvider`；新增 `getHeatmapWithCache` wrapper（D-103-9 fallback 语义）；`queryHeatmapFromMV` 复用为 fallback 路径

---

### `internal/services/asset/reconciliation_service.go` (service, CRUD + cache)

**迁移类型:** 行为等价重构（字段类型 + 1 处手写 read-through + warn-on-set wrapper）

**当前 struct 字段（:196-201）：**
```go
type reconciliationServiceImpl struct {
    db *gorm.DB
    cache cache.Cache  // GetByWorkstation 用
    matcher ReconciliationExceptionService
    mvExists int32
    mvExistsMux sync.Mutex
}
```

**迁移后 struct 字段（目标）：**
```go
type reconciliationServiceImpl struct {
    db *gorm.DB
    cache base.CacheProvider  // 替换 cache.Cache
    matcher ReconciliationExceptionService
    mvExists int32
    mvExistsMux sync.Mutex
}
```

**迁移后构造函数（目标）：**
```go
func NewReconciliationService(db *gorm.DB, c base.CacheProvider, matcher ReconciliationExceptionService) ReconciliationService {
    return &reconciliationServiceImpl{db: db, cache: c, matcher: matcher, mvExists: -1}
}
```

**位点 F — GetByWorkstation (:793-824)：**

**当前代码（手写 cache-aside，含 `Workstation.ID != ""` 脏缓存防御 + warn-on-set 不阻断）：**
```go
// internal/services/asset/reconciliation_service.go:798-824
// 缓存命中短路
var cached ByWorkstationResponse
if err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && cached.Workstation.ID != "" {
    cached.Visible = false
    return &cached, nil
}
// ... compute ...
// 写缓存
if data, mErr := json.Marshal(resp); mErr == nil {
    if setErr := s.cache.Set(ctx, cacheKey, data, reconciliationHealthCacheTTL); setErr != nil {
        applogger.Warnf("[reconciliation] GetByWorkstation cache set failed wsID=%s: %v", wsID, setErr)
    }
}
```

**迁移后 wrapper helper（D-103-7 warn-on-set 不阻断）：**
```go
// reconciliation_service.go 新增 wrapper helper
func (s *reconciliationServiceImpl) getByWorkstationWithCache(ctx context.Context, cacheKey string, wsID string) (*ByWorkstationResponse, error) {
    result, err := base.GetOrSetJSON[*ByWorkstationResponse](ctx, s.cache, cacheKey, reconciliationHealthCacheTTL, func() (*ByWorkstationResponse, error) {
        return s.computeByWorkstation(ctx, wsID)
    })
    // D-103-7: warn-on-set 不阻断语义
    if err != nil {
        applogger.Warnf("[reconciliation] GetByWorkstation cache set failed wsID=%s: %v", wsID, err)
        if result != nil {
            result.Visible = false
            return result, nil  // 即使 set 失败，结果仍可用
        }
        return nil, err
    }
    if result != nil {
        result.Visible = false
    }
    return result, nil
}
```

**GetByWorkstation 改写后：**
```go
func (s *reconciliationServiceImpl) GetByWorkstation(ctx context.Context, wsID string, window string) (*ByWorkstationResponse, error) {
    if wsID == "" { return nil, errors.New("工位ID不能为空") }
    cacheKey := GetReconciliationHealthByWorkstationKey(wsID)
    return s.getByWorkstationWithCache(ctx, cacheKey, wsID)
}
```

**TTL 来源（保持不变）：**
```go
// internal/services/asset/reconciliation_service.go:773-774
const reconciliationHealthCacheTTL = 5 * time.Minute  // D-103-11 保持引用 pkg/constants.reconciliationHealthCacheTTL
```

**结论:** `cache cache.Cache` → `cache base.CacheProvider`；手写 cache-aside → `getByWorkstationWithCache` wrapper；`Workstation.ID != ""` 脏缓存防御移至 wrapper 内；Marshal 错误静默保留

---

### `internal/services/rpa/selector_learner.go` (service, CRUD + cache)

**迁移类型:** 行为等价重构（字段类型 + 2 处手写 JSON cache-aside + best-effort wrapper）

**当前 struct 字段（:26-32）：**
```go
type selectorLearnerImpl struct {
    db     *gorm.DB
    cache  cache.Cache
    config *config.Config
    mu     sync.RWMutex
}
```

**迁移后 struct 字段（目标）：**
```go
type selectorLearnerImpl struct {
    db     *gorm.DB
    cache  base.CacheProvider  // 替换 cache.Cache
    config *config.Config
    mu     sync.RWMutex
}
```

**迁移后构造函数（目标）：**
```go
func NewSelectorLearner(db *gorm.DB, cache base.CacheProvider, cfg *config.Config) SelectorLearner {
```

**位点 G — GetBestSelector (:167-175, 224-228)：**

**当前代码（手写 Get + Unmarshal / Marshal + Set）：**
```go
// internal/services/rpa/selector_learner.go:167-231
// 读
cacheKey := l.getCacheKey(pageURL, elementID)
if cached, err := l.cache.Get(ctx, cacheKey); err == nil {
    var result SelectorRecommendation
    if err := json.Unmarshal([]byte(cached), &result); err == nil {
        return &result, nil
    }
}
// ... compute ...
// 写
if best != nil {
    if data, err := json.Marshal(best); err == nil {
        _ = l.cache.Set(ctx, cacheKey, string(data), 30*time.Minute)
    }
}
```

**迁移后 wrapper helper（D-103-8 best-effort）：**
```go
// selector_learner.go 新增 wrapper helper
func (l *selectorLearnerImpl) getBestSelectorCached(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
    cacheKey := l.getCacheKey(pageURL, elementID)
    result, err := base.GetOrSetJSON[*SelectorRecommendation](ctx, l.cache, cacheKey, 30*time.Minute, func() (*SelectorRecommendation, error) {
        // ... 原 DB 查询逻辑 ...
    })
    // D-103-8: best-effort 语义
    if err != nil {
        applogger.Warnf("[selector_learner] GetBestSelector cache failed: %v", err)
        return nil, nil  // 返回 nil, nil，调用方视为 cache miss
    }
    return result, nil
}
```

**SaveSelector (RecordSuccess :91-128) 失效改写：**

**当前代码：**
```go
// internal/services/rpa/selector_learner.go:124-126
cacheKey := l.getCacheKey(record.PageURL, record.ElementID)
_ = l.cache.Delete(ctx, cacheKey)
```

**迁移后（D-103-14）：**
```go
base.Invalidate(ctx, l.cache, []string{l.getCacheKey(record.PageURL, record.ElementID)}, "SelectorLearner")
```

**结论:** `cache cache.Cache` → `cache base.CacheProvider`；手写 JSON cache-aside → `getBestSelectorCached` wrapper（best-effort 语义）；TTL `30*time.Minute` 保持字面量（D-103-12）；Delete → `base.Invalidate`

---

## Shared Patterns

### Authentication / Authorization
**Source:** N/A（无 auth 变更）
**Apply to:** N/A

### Error Handling — 4 种语义 wrapper（D-103-6..9）

**Source:** `internal/services/base/cache_functions.go` + 本相决策

**严格语义（mac_history 三处，D-103-6）：**
```go
// 照抄 base.GetOrSetJSON 直接透传
result, err := base.GetOrSetJSON[T](ctx, s.cache, key, ttl, queryFn)
if err != nil { return nil, err }
return result, nil
```

**warn-on-set 不阻断（reconciliation，D-103-7）：**
```go
result, err := base.GetOrSetJSON[T](ctx, s.cache, key, ttl, queryFn)
if err != nil {
    applogger.Warnf("[MODULE] operation cache set failed: %v", err)
    if result != nil {
        result.Visible = false  // 或其他 post 处理
        return result, nil
    }
    return nil, err
}
```

**best-effort 静默（selector_learner，D-103-8）：**
```go
result, err := base.GetOrSetJSON[T](ctx, s.cache, key, ttl, queryFn)
if err != nil {
    applogger.Warnf("[selector_learner] operation cache failed: %v", err)
    return nil, nil  // 调用方视为 cache miss
}
```

**fallback 直查（mac_history_heatmap，D-103-9）：**
```go
result, err := base.GetOrSetJSON[T](ctx, s.cache, key, ttl, queryFn)
if err != nil {
    applogger.Warnf("[MODULE] 走直查: %v", err)
    return fallbackQueryFn()  // 复用既有直查函数
}
```

### Cache Invalidation
**Source:** `internal/services/base/cache_functions.go:50-76`

**单键失效：**
```go
base.Invalidate(ctx, s.cache, []string{cacheKey}, "MODULE")
```

**模式失效：**
```go
base.InvalidatePattern(ctx, s.cache, []string{pattern + "*"}, "MODULE")
```

**Apply to:** reconciliation SaveRecord、selector_learner RecordSuccess 等所有失效点

### Validation
**Source:** N/A（无新增验证）
**Apply to:** N/A

---

## invariants 扩口模式（CONV-04）

**宿主文件:** `internal/services/system/cache_invariants_92_test.go`

**扩口需要改的确切位置（:139-149）：**

```go
// Phase 92 既有硬档目录（零改动）：
hardDirs := []string{
    filepath.Join(servicesRoot, "system"),
    filepath.Join(servicesRoot, "operations"),
}
// Phase 103 扩口——在同一个 hardDirs slice 中追加（:600-602 RESEARCH 方案）：
hardDirs := []string{
    filepath.Join(servicesRoot, "system"),
    filepath.Join(servicesRoot, "operations"),
    // Phase 103 扩口：
    servicesRoot,                              // mac_history_query_service.go, mac_history_heatmap_service.go
    filepath.Join(servicesRoot, "asset"),       // reconciliation_service.go
    filepath.Join(servicesRoot, "rpa"),         // selector_learner.go
}
```

**AST 扫描逻辑骨架（:43-93）：**
```go
// cacheImplResidue92 检测 interface{} 闭包式 GetOrSet
func cacheImplResidue92(t *testing.T, path string) []string {
    fset := token.NewFileSet()
    f, err := parser.ParseFile(fset, path, nil, 0)
    // ...
    ast.Inspect(f, func(n ast.Node) bool {
        call, ok := n.(*ast.CallExpr)
        // 1. selector 为 .GetOrSet
        sel, ok := call.Fun.(*ast.SelectorExpr)
        if !ok || sel.Sel.Name != "GetOrSet" { return true }
        // 2. 实参是 FuncLit
        for _, arg := range call.Args {
            fl, ok := arg.(*ast.FuncLit)
            // 3. 闭包签名 (interface{}, error)
            if ft.Results == nil || len(ft.Results.List) != 2 { continue }
            if !isEmptyInterface(ft.Results.List[0].Type) { continue }
            errIdent, ok := ft.Results.List[1].Type.(*ast.Ident)
            if ok && errIdent.Name == "error" {
                hits = append(hits, fmt.Sprintf("%s:%d", filepath.Base(pos.Filename), pos.Line))
            }
        }
        return true
    })
    return hits
}
```

**手写 cache-aside 模式检测（D-103-18，需新增）：**
```go
// 新增函数检测 cache.Get → json.Unmarshal → cache.Set 序列
// 模式：cache.Get(ctx, key) → json.Unmarshal → cache.Set(ctx, key, json.Marshal(...))
// 误报风险低——该序列是手写 cache-aside 的标志性形态
func cacheAsideResidue(t *testing.T, path string) []string {
    // 检测逻辑：
    // 1. 找 cache.Get 调用（selector 为 Get，receiver 类型匹配 cache.Cache 相关）
    // 2. 找后续 json.Unmarshal 调用
    // 3. 找后续 cache.Set 调用（含 json.Marshal）
    // 命中即 fail
}
```

**白名单机制（:31-33, 186-191）：**
```go
var allowedResidues = map[string]int{}  // 初始全空 = 期望 0

// 白名单卫生检查（登记了额度却零残留提示可清理）
for name, allowed := range allowedResidues {
    if _, ok := perFile[name]; !ok && allowed > 0 {
        t.Logf("[ALLOWED] 白名单条目 %s=%d 当前无对应残留（可清理）", name, allowed)
    }
}
```

**结论:** `cache_invariants_92_test.go` 的 `hardDirs` 追加 3 项（servicesRoot + asset + rpa）；`allowedResidues` 初始保持全空；D-103-18 手写 cache-aside 检测新增独立函数

---

## Caller Audit 清单

### 生产调用点

| 文件:行号 | 当前调用 | 迁后应改为 |
|-----------|---------|-----------|
| `internal/api/v1/asset/reconciliation_router.go:34` | `asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvc)` | `asset.NewReconciliationService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService), exceptionSvc)` |
| `internal/api/router.go:619` | `asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvcForWs)` | `asset.NewReconciliationService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService), exceptionSvcForWs)` |
| `internal/api/v1/network/mac_history_router.go:19` | `NewMACHistoryHeatmapService(core.GetDB(), nil, nil)` | `NewMACHistoryHeatmapService(core.GetDB(), system.NewCacheProvider(core.DataCacheService), core.CacheConfigService)` |
| `internal/services/rpa/ai_service.go:84` | `NewSelectorLearner(db, cache, cfg)` | `NewSelectorLearner(db, system.NewCacheProvider(cache), cfg)` |

### 测试调用点

| 文件:行号 | 当前调用 | 适配方案 |
|-----------|---------|---------|
| `internal/services/mac_history_query_service_79_05_test.go:140` | `NewMACHistoryQueryServiceWithCache(db, NewDataCacheService(...), nil)` | 传入 `system.NewCacheProvider(NewDataCacheService(...))` |
| `internal/services/mac_history_tail_79_05_test.go:57,728` | `NewMACHistoryHeatmapService(db, NewDataCacheService(...), ...)` | 同上 |
| `internal/services/mac_history_tail_79_05_test.go:694,717` | `NewMACHistoryHeatmapService(db, nil, nil)` | 传入 `base.NoOpCacheProvider{}` |
| `internal/services/asset/reconciliation_sqlite_runtime_test.go:68,89,94,103` | `NewReconciliationService(db, nil, nil)` | 传入 `base.NoOpCacheProvider{}` |
| `internal/services/asset/asset_gapfill_test.go:203,210,241,300,346` | `NewReconciliationService(db, nil, ...)` | 传入 `base.NoOpCacheProvider{}` |
| `internal/services/rpa/ai_selector_excel_test.go:286,328,348` | `NewSelectorLearner(db, &fakeSelectorCache{}, ...)` | `fakeSelectorCache` 需实现 `base.CacheProvider` 接口 |
| `internal/api/v1/network/setup_routers_test.go:250` | 断言 `NewMACHistoryHeatmapService(core.GetDB(), nil, nil)` | 断言字面量需更新 |

**`base.NoOpCacheProvider` 现成可用:** `internal/services/base/cache_provider.go:59-146` 实现完整 `CacheProvider` 接口，query 函数直接执行不做缓存。

---

## 回归测试骨架

### 现有测试文件（需适配）

**mac_history_query_service_79_05_test.go** — query_service 测试：
```go
// 当前构造（:140）：
svc := NewMACHistoryQueryServiceWithCache(db, NewDataCacheService(mem), nil)
// 迁后：
svc := NewMACHistoryQueryServiceWithCache(db, system.NewCacheProvider(NewDataCacheService(mem)), nil)
```

**mac_history_tail_79_05_test.go** — heatmap 测试：
```go
// 当前（:57,728）：
heatmap := NewMACHistoryHeatmapService(db, NewDataCacheService(mem), NewCacheConfigService(db))
// 迁后：
heatmap := NewMACHistoryHeatmapService(db, system.NewCacheProvider(NewDataCacheService(mem)), NewCacheConfigService(db))
```

**reconciliation_sqlite_runtime_test.go** — reconciliation 测试：
```go
// 当前（:68,89,94,103）：
svc := NewReconciliationService(db, nil, nil)
// 迁后：
svc := NewReconciliationService(db, base.NoOpCacheProvider{}, nil)
```

**ai_selector_excel_test.go** — selector 测试：
```go
// 当前（:348）：
l := NewSelectorLearner(db, &fakeSelectorCache{}, newAICfg("", "", false, false))
// fakeSelectorCache 需要改为实现 base.CacheProvider：
type fakeSelectorCache struct { base.NoOpCacheProvider }
```

### Phase 92 等价语义回归测试先例

**`internal/services/system/cache_invariants_92_test.go`** — AST 扫描锁 0 残留：
```go
// 硬档：残留按文件计数，超出白名单即 fail
for _, name := range names {
    hits := perFile[name]
    allowed := allowedResidues[name]
    if len(hits) > allowed {
        for _, h := range hits[allowed:] {
            t.Errorf("interface{} 闭包式 GetOrSet 残留：%s", h)
        }
    }
}
```

**Phase 102 CACHE-01/02 等价测试模式（TTL + cache key 字符串等价断言）：**
```go
// TTL 等价断言示例
assert.Equal(t, 5*time.Minute, s.perfCacheTTL())
// Cache key 字符串等价
assert.Equal(t, "mac:vendor:AABBCC", fmt.Sprintf(constants.MacVendorKeyFormat, "AABBCC"))
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| D-103-18 手写 cache-aside AST 检测函数 | test | AST scan | Phase 92 AST 扫描仅检测 `interface{}` 闭包 GetOrSet，未覆盖手写 cache-aside 序列（`cache.Get` → `json.Unmarshal` → `cache.Set`）；本相首创此检测函数 |

---

## Metadata

**Analog search scope:** `internal/services/base/` + `internal/services/system/` + `internal/services/operations/` + `pkg/constants/`
**Files scanned:** 23
**Pattern extraction date:** 2026-09-07
