# Phase 103: 缓存闭包收敛 base 单一权威 - Research

**Researched:** 2026-09-07
**Domain:** Go backend — services layer cache abstraction migration
**Confidence:** HIGH

## Summary

Phase 103 是 v1.29 Phase 92 缓存统一的补遗收口——三域（mac_history / asset reconciliation / rpa selector）残余 legacy `interface{}` 闭包式 GetOrSet 与手写 cache-aside 全部迁移 `base.GetOrSetJSON[T]`。本相是**行为等价重构**，不引入新功能。

**核心发现：**
1. **行号漂移已确认**：CONTEXT.md 记录的行号(:307/:432/:837/:259-281/:118/:799-820/:169-174/:226)与当前代码完全吻合，无漂移
2. **CacheProvider 接口完全覆盖三域需求**：`CacheProvider` 的 9 方法（GetOrSet / Delete / DeleteByPattern / MGet / MDelete / Exists / SetTTL / GetTTL / GetStats）覆盖三域全部调用；无缺口
3. **手写 cache-aside 迁移路径明确**：selector_learner 的 `cache.Get` + `json.Unmarshal` 与 reconciliation 的 `cache.GetJSON` + `json.Marshal` + `cache.Set` 均通过 wrapper helper 收敛 `base.GetOrSetJSON[T]`
4. **错误语义 wrapper 有 4 种：D-103-6..9 已锁定**
5. **invariants 扩口可行**：在 `cache_invariants_92_test.go` 同一测试函数内扩展目录列表即可（mac_history_query_service.go 等 4 文件加入 hardDirs），新增 `cache_invariants_103_test.go` 不是必选

---

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-103-1**: 三域字段 `dataCache *DataCacheService` / `cache cache.Cache` 统一改为 `cache base.CacheProvider`
- **D-103-2**: 四个构造函数入参从 `*DataCacheService` 或 `cache.Cache` 改为 `base.CacheProvider`；所有调用方同步切
- **D-103-3**: 不引入 `cache.Cache → CacheProvider` adapter wrapper，保持结构性重构干净
- **D-103-4**: `mac_history_query_service.go:259-281` vendor manual cache-aside 纳入本相
- **D-103-5**: Vendor "Unknown Vendor" 占位缓存行为保留
- **D-103-6**: mac_history query_service 三处用 `base.GetOrSetJSON` 严格语义（cache 写失败即返回 error）
- **D-103-7**: reconciliation GetByWorkstation 用 wrapper 保留 warn-on-set 不阻断语义
- **D-103-8**: selector_learner 用 wrapper 保留 best-effort 语义（nil/fallback + warn 日志）
- **D-103-9**: mac_history_heatmap 用 wrapper 保留 fallback 直查语义
- **D-103-10**: mac_history 域 TTL 走 `s.perfCacheTTL()` 不变
- **D-103-11**: reconciliation TTL 走 `pkg/constants.reconciliationHealthCacheTTL` 不变
- **D-103-12**: selector_learner TTL 保持字面量 `30*time.Minute`
- **D-103-14**: 顺手接 `base.Invalidate` / `base.InvalidatePattern`
- **D-103-15**: 批量失效走 `base.InvalidatePattern`
- **D-103-16**: 扫描口径 = 显式文件清单 + 包路径
- **D-103-17**: 硬失败档——本相收敛面 4 文件 interface{} 闭包式 GetOrSet 计数必须 == 0
- **D-103-18**: 手写 cache-aside 扫描规则（`cache.Get` 后紧跟 `json.Unmarshal` + `cache.Set` 含 `json.Marshal`）
- **D-103-19**: 逐域回归测试锁等价语义
- **D-103-20**: constructor 签名变更兼容性测试
- **D-103-21**: TTL 等价 + cache key 字符串等价断言

### Claude's Discretion

- heatmap fallback helper 命名与包内位置
- reconciliation "脏缓存防御" `Workstation.ID != ""` 检查迁 wrapper 后的位置
- invariants 新测试文件的命名（`cache_invariants_103_test.go` vs 扩展现有）
- 三域 `Invalidate` 调用的 module 字符串常量
- 既有 `dataCache` 字段删除时机（同 commit 删除）

---

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CONV-01 | mac_history_query_service.go 4 处 legacy GetOrSet + :259-281 手写 cache-aside + heatmap_service.go:118 迁移 `base.GetOrSetJSON[T]` | 三域迁移路径、错误语义 wrapper 方案已确认 |
| CONV-02 | asset/reconciliation_service.go:799-820 GetByWorkstation 手写读穿透迁移 `base.GetOrSetJSON[T]` | 手写 cache-aside 形态确认（GetJSON + Marshal + Set），wrapper 语义保留方案已确认 |
| CONV-03 | rpa/selector_learner.go:169-174,226 GetBestSelector/SaveSelector 手写 JSON cache-aside 迁移 `base.GetOrSetJSON[T]` | 手写 cache-aside 形态确认（Get + Unmarshal / Marshal + Set），best-effort wrapper 方案已确认 |
| CONV-04 | `cache_invariants_92_test.go` 扫描口径扩至 services 根 / asset / rpa 包，interface{} 闭包式 GetOrSet 硬失败 | AST 扫描机制确认，扩口方案可行 |

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| mac_history query cache | Backend Service | — | 三域 service 层持有 cache 字段，迁 base.CacheProvider |
| asset reconciliation cache | Backend Service | — | 同上 |
| rpa selector cache | Backend Service | — | 同上 |
| invariants AST scan | Backend Test | — | 测试代码，独立执行 |
| cache key constants | Backend Service | — | Phase 102 已注册键常量（mac vendor / rpa selector）直接使用 |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `internal/services/base` | — | `GetOrSetJSON[T]` / `Invalidate` / `InvalidatePattern` 单一权威 | Phase 92 已确立，三域迁移目标 |
| `base.CacheProvider` | — | 9 方法接口（GetOrSet/Delete/DeleteByPattern/MGet/MDelete/Exists/SetTTL/GetTTL/GetStats） | Phase 92 确立，三域迁移目标接口 |
| `system.NewCacheProvider` | — | `*DataCacheService → CacheProvider` 适配器 | 用于 router.go 接线 |
| `base.NoOpCacheProvider` | — | nil-safe 空实现 | 测试场景传入 nil |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/constants/reconciliationHealthCacheTTL` | — | reconciliation 5min TTL 常量 | D-103-11 保持引用 |
| `CacheConfigService.perfCacheTTL()` | — | mac_history 域 TTL 来源 | D-103-10 保持引用 |
| `constants.MacVendorKeyFormat` | — | mac vendor 缓存键格式 | Phase 102 已注册，mac_history vendor lookup 直接使用 |
| `systemServices.GetRpaSelectorBestKey` | — | rpa selector 缓存键构建 | selector_learner.go:363 已使用，Phase 102 已注册 |
| `asset.GetReconciliationHealthByWorkstationKey` | — | reconciliation 工位健康度缓存键 | reconciliation_service.go:799 已使用 |

---

## Package Legitimacy Audit

> 本相不引入新的外部包，仅迁移既有代码到已验证的 `base` 抽象。无新增依赖。

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| 无新增 | — | — | — | — | — | — |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
# 迁移前（legacy interface{} 闭包）
┌─────────────────────────────────────────────────────────────────┐
│  macHistoryQueryServiceImpl / heatmapServiceImpl                │
│  ┌──────────────┐  ┌──────────────┐                           │
│  │ dataCache    │  │ cache       │  ← *DataCacheService /     │
│  │ *DataCacheService│ cache.Cache  │    cache.Cache 类型       │
│  └──────────────┘  └──────────────┘                           │
│         ↓                ↓                                    │
│  dataCache.GetOrSet(ctx, key, &result, ttl, func() (interface{}, error){...}) │
│  ← interface{} 闭包式 GetOrSet（编译警告，行为漂移风险）        │
└─────────────────────────────────────────────────────────────────┘

# 迁移后（base.CacheProvider + GetOrSetJSON[T]）
┌─────────────────────────────────────────────────────────────────┐
│  macHistoryQueryServiceImpl / heatmapServiceImpl /             │
│  reconciliationServiceImpl / selectorLearnerImpl                 │
│  ┌──────────────────────────────────┐                         │
│  │ cache  base.CacheProvider         │  ← 统一接口类型         │
│  └──────────────────────────────────┘                         │
│         ↓                                                       │
│  base.GetOrSetJSON[T](ctx, s.cache, key, ttl, func() (T, error){...}) │
│  ← 泛型 T，编译期类型安全，单 return                            │
└─────────────────────────────────────────────────────────────────┘

# 四种错误语义 wrapper（D-103-6..9）
┌──────────────────────────────────────────────────────────────────┐
│  严格语义（mac_history 三处）:                                    │
│    result, err := base.GetOrSetJSON[T](...)                      │
│    if err != nil { return nil, err }  // cache 写失败即返回       │
├──────────────────────────────────────────────────────────────────┤
│  warn-on-set 不阻断（reconciliation）:                           │
│    result, err := base.GetOrSetJSON[T](...)                      │
│    if err != nil { warn日志; return cachedResult, nil }         │
├──────────────────────────────────────────────────────────────────┤
│  best-effort 静默（selector_learner）:                            │
│    result, err := base.GetOrSetJSON[T](...)                      │
│    if err != nil { warn日志; return nil, nil }                   │
├──────────────────────────────────────────────────────────────────┤
│  fallback 直查（mac_history_heatmap）:                            │
│    result, err := base.GetOrSetJSON[T](...)                      │
│    if err != nil { warn日志; return fallback查询, nil }          │
└──────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
internal/services/
├── mac_history_query_service.go      # 迁移：字段 dataCache → cache base.CacheProvider
├── mac_history_heatmap_service.go    # 迁移：字段 dataCache → cache base.CacheProvider
├── asset/
│   ├── reconciliation_service.go     # 迁移：字段 cache → cache base.CacheProvider
│   └── cache_keys.go                 # 键常量已就位（Phase 102）
├── rpa/
│   └── selector_learner.go           # 迁移：字段 cache → cache base.CacheProvider
└── system/
    ├── cache_invariants_92_test.go   # 扩口：4 文件加入 hardDirs
    └── cache_invariants_103_test.go  # [可选] 新测试文件收纳扩口断言
```

---

## Migration Sites — Current Code Morphology

### 1. `mac_history_query_service.go` — 4 处 GetOrSet + vendor cache-aside

**当前 struct 字段（:138-143）：**
```go
type macHistoryQueryServiceImpl struct {
    db         *gorm.DB
    cache      cache.Cache   // vendor lookup 专用，nil-safe
    dataCache  *DataCacheService  // 三处 GetOrSet 使用
    perfConfig *CacheConfigService
}
```

**当前构造函数（:150-153）：**
```go
func NewMACHistoryQueryServiceWithCache(db *gorm.DB, dataCache *DataCacheService, perfConfig *CacheConfigService) MACHistoryQueryService {
    return &macHistoryQueryServiceImpl{db: db, cache: nil, dataCache: dataCache, perfConfig: perfConfig}
}
```

**迁移后 struct 字段：**
```go
type macHistoryQueryServiceImpl struct {
    db         *gorm.DB
    cache      base.CacheProvider  // vendor lookup 仍用此字段
    perfConfig *CacheConfigService // mac_history 域保留
}
```

**迁移后构造函数：**
```go
func NewMACHistoryQueryServiceWithCache(db *gorm.DB, cache base.CacheProvider, perfConfig *CacheConfigService) MACHistoryQueryService {
    return &macHistoryQueryServiceImpl{db: db, cache: cache, perfConfig: perfConfig}
}
```

**位点 A — vendor lookup (:259-281)：**
```go
// 当前代码（手写 cache-aside）：
if s.cache != nil {
    vendorName, err := s.cache.Get(ctx, cacheKey)  // cache 是 cache.Cache
    if err == nil && vendorName != "" { return vendorName, nil }
    // ...
    _ = s.cache.Set(ctx, cacheKey, "Unknown Vendor", 24*time.Hour)
}
// 迁后（使用 cache base.CacheProvider）：
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
// D-103-6 严格语义：cache 写失败即返回 error（vendor lookup 写失败仍返回结果，base.GetOrSetJSON 语义正确）
```

**位点 B — QueryPortHistory (:304-318)：**
```go
// 当前：
err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
    return s.queryPortHistoryFromDB(ctx, req)
})
// 迁后（base.GetOrSetJSON 严格语义，D-103-6）：
cached, err := base.GetOrSetJSON[MACHistoryQueryResult](ctx, s.cache, cacheKey, s.perfCacheTTL(), func() (MACHistoryQueryResult, error) {
    return s.queryPortHistoryFromDB(ctx, req)
})
if err != nil { return nil, err }
return &cached, nil
```

**位点 C — QueryDeviceHistory (:429-443)：** 同位点 B 形态，替换 `queryDeviceHistoryFromDB`

**位点 D — QueryConnectionStats (:832-846)：** 同位点 B 形态，替换 `queryConnectionStatsFromDB`

---

### 2. `mac_history_heatmap_service.go` — 1 处 GetOrSet + fallback

**当前 struct 字段（:50-54）：**
```go
type macHistoryHeatmapServiceImpl struct {
    db         *gorm.DB
    dataCache  *DataCacheService
    perfConfig *CacheConfigService
}
```

**当前构造函数（:57-59）：**
```go
func NewMACHistoryHeatmapService(db *gorm.DB, dataCache *DataCacheService, perfConfig *CacheConfigService) MACHistoryHeatmapService {
    return &macHistoryHeatmapServiceImpl{db: db, dataCache: dataCache, perfConfig: perfConfig}
}
```

**迁移后 struct 字段：**
```go
type macHistoryHeatmapServiceImpl struct {
    db         *gorm.DB
    cache      base.CacheProvider  // 替换 dataCache
    perfConfig *CacheConfigService
}
```

**迁移后构造函数：**
```go
func NewMACHistoryHeatmapService(db *gorm.DB, cache base.CacheProvider, perfConfig *CacheConfigService) MACHistoryHeatmapService {
    return &macHistoryHeatmapServiceImpl{db: db, cache: cache, perfConfig: perfConfig}
}
```

**位点 E — QueryHeatmap (:113-127)：**
```go
// 当前：
err := s.dataCache.GetOrSet(ctx, cacheKey, &cached, s.perfCacheTTL(), func() (interface{}, error) {
    return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
})
if err == nil { return &cached, nil }
applogger.Warnf("[MAC热力图] 走直查: %v", err)
// 迁后（wrapper helper 保留 fallback 语义，D-103-9）：
func (s *macHistoryHeatmapServiceImpl) getHeatmapWithCache(ctx context.Context, cacheKey string, ttl time.Duration, req *HeatmapQuery, startTime, endTime time.Time) (*HeatmapResult, error) {
    result, err := base.GetOrSetJSON[*HeatmapResult](ctx, s.cache, cacheKey, ttl, func() (*HeatmapResult, error) {
        return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
    })
    if err != nil {
        applogger.Warnf("[MAC热力图] 走直查: %v", err)
        return s.queryHeatmapFromMV(ctx, startTime, endTime, req.TopN)
    }
    return result, nil
}
```

---

### 3. `asset/reconciliation_service.go` — 1 处手写 read-through

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

**当前构造函数（:227）：**
```go
func NewReconciliationService(db *gorm.DB, c cache.Cache, matcher ReconciliationExceptionService) ReconciliationService {
    return &reconciliationServiceImpl{db: db, cache: c, matcher: matcher, mvExists: -1}
}
```

**迁移后 struct 字段：**
```go
type reconciliationServiceImpl struct {
    db *gorm.DB
    cache base.CacheProvider  // 替换 cache.Cache
    matcher ReconciliationExceptionService
    mvExists int32
    mvExistsMux sync.Mutex
}
```

**迁移后构造函数：**
```go
func NewReconciliationService(db *gorm.DB, c base.CacheProvider, matcher ReconciliationExceptionService) ReconciliationService {
    return &reconciliationServiceImpl{db: db, cache: c, matcher: matcher, mvExists: -1}
}
```

**位点 F — GetByWorkstation (:793-824)：**

当前代码形态：
```go
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
        applogger.Warnf("[reconciliation] GetByWorkstation cache set failed: %v", setErr)
    }
}
```

迁后 wrapper helper（D-103-7 warn-on-set 不阻断）：
```go
func (s *reconciliationServiceImpl) getByWorkstationWithCache(ctx context.Context, cacheKey string, wsID string) (*ByWorkstationResponse, error) {
    var cached ByWorkstationResponse
    // D-103-7: 保留 Workstation.ID != "" 脏缓存防御
    result, err := base.GetOrSetJSON[*ByWorkstationResponse](ctx, s.cache, cacheKey, reconciliationHealthCacheTTL, func() (*ByWorkstationResponse, error) {
        return s.computeByWorkstation(ctx, wsID)
    })
    if err != nil {
        // cache 写失败 warn，不阻断
        applogger.Warnf("[reconciliation] GetByWorkstation cache set failed wsID=%s: %v", wsID, err)
        // 返回 result（即使 err != nil，result 仍可能有效）
        if result != nil {
            result.Visible = false
            return result, nil
        }
        return nil, err
    }
    if result != nil {
        result.Visible = false
    }
    return result, nil
}
```

**reconciliation SaveRecord 失效（:129-133）：**
```go
// 当前：InvalidateWorkstationHealth(ctx, s.cache, wsID)
// 迁后（D-103-14）：
base.Invalidate(ctx, s.cache, []string{asset.GetReconciliationHealthByWorkstationKey(wsID)}, "Reconciliation")
```

---

### 4. `rpa/selector_learner.go` — 2 处手写 JSON cache-aside

**当前 struct 字段（:26-32）：**
```go
type selectorLearnerImpl struct {
    db     *gorm.DB
    cache  cache.Cache
    config *config.Config
    mu     sync.RWMutex
}
```

**当前构造函数（:35）：**
```go
func NewSelectorLearner(db *gorm.DB, cache cache.Cache, cfg *config.Config) SelectorLearner {
```

**迁移后 struct 字段：**
```go
type selectorLearnerImpl struct {
    db     *gorm.DB
    cache  base.CacheProvider
    config *config.Config
    mu     sync.RWMutex
}
```

**迁移后构造函数：**
```go
func NewSelectorLearner(db *gorm.DB, cache base.CacheProvider, cfg *config.Config) SelectorLearner {
```

**位点 G — GetBestSelector (:167-175, 224-228)：**

当前代码形态：
```go
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

迁后 wrapper helper（D-103-8 best-effort）：
```go
func (l *selectorLearnerImpl) getBestSelectorCached(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
    cacheKey := l.getCacheKey(pageURL, elementID)
    result, err := base.GetOrSetJSON[*SelectorRecommendation](ctx, l.cache, cacheKey, 30*time.Minute, func() (*SelectorRecommendation, error) {
        // ... 原 GetBestSelector DB 查询逻辑 ...
    })
    if err != nil {
        applogger.Warnf("[selector_learner] GetBestSelector cache failed: %v", err)
        return nil, nil  // D-103-8 best-effort：返回 nil, nil
    }
    return result, nil
}
```

**位点 H — SaveSelector（RecordSuccess :91-128）：**

当前代码：`_ = l.cache.Delete(ctx, cacheKey)`
迁后（D-103-14）：`base.Invalidate(ctx, l.cache, []string{l.getCacheKey(record.PageURL, record.ElementID)}, "SelectorLearner")`

---

## base 包 API 实测

### GetOrSetJSON 精确签名
```go
// internal/services/base/cache_functions.go:30-42
func GetOrSetJSON[T any](
    ctx context.Context,
    p CacheProvider,
    key string,
    ttl time.Duration,
    query func() (T, error),
) (T, error)
```

**泛型约束：** `T any`（无约束，任意类型）
**错误返回语义：** 底层 `p.GetOrSet` 返回 error 时直接透传；`DataCacheService.GetOrSet` 在 cache 写失败时**返回 warn 日志但仍返回 nil**（:125-127 `if err := s.Set(...); err != nil { Warnf(...); }` return nil）——因此 `base.GetOrSetJSON` 的 cache 写失败路径**返回 nil error**（写失败不阻断）
**Marshal/Unmarshal 路径：** `DataCacheService.GetOrSet` 内部 `json.Marshal` + `json.Unmarshal`（:116-117）；`base.GetOrSetJSON` 薄委托此路径

### Invalidate / InvalidatePattern
```go
// nil 防护内聚，warn 日志
func Invalidate(ctx context.Context, p CacheProvider, keys []string, module string) {
    if p == nil { logger.Debugf("未配置缓存提供者"); return }
    for _, key := range keys {
        if err := p.Delete(ctx, key); err != nil {
            logger.Warnf("[%s] 清除缓存失败: key=%s, error=%v", module, key, err)
        }
    }
}
```

### CacheProvider 9 方法接口
```go
type CacheProvider interface {
    GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error
    Delete(ctx context.Context, key string) error
    DeleteByPattern(ctx context.Context, pattern string) error
    MGet(ctx context.Context, keys ...string) (map[string]string, error)
    MDelete(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) (bool, error)
    SetTTL(ctx context.Context, key string, expiration time.Duration) error
    GetTTL(ctx context.Context, key string) (time.Duration, error)
    GetStats(ctx context.Context) (*CacheStats, error)
}
```

**无 `GetJSON`/`SetJSON` 方法**——这是 `cache.Cache` 的扩展方法，三域手写 cache-aside 依赖 `GetJSON` 的 JSON 反序列化能力。迁移后由 `base.GetOrSetJSON[T]` 内部处理 JSON 往返，不再需要直接调用。

---

## CacheProvider vs 现有字段类型的方法面差集

| 域 | 现有字段类型 | 调用的 cache 方法 | CacheProvider 是否覆盖 |
|---|------------|------------------|----------------------|
| mac_history query | `*DataCacheService` | GetOrSet, Delete | 是（GetOrSet + Delete） |
| mac_history vendor | `cache.Cache` | Get, Set | 是（通过 adapter 封装） |
| mac_history heatmap | `*DataCacheService` | GetOrSet | 是 |
| reconciliation | `cache.Cache` | GetJSON, Set, Delete | GetJSON 否（通过 base.GetOrSetJSON 绕开）；Set/Delete 是 |
| selector_learner | `cache.Cache` | Get, Set, Delete | 是 |

**结论：无缺口。** `GetJSON` 是 `cache.Cache` 的扩展方法（`pkg/cache/cache.go:53`），但三域迁移后不再直接调用——`base.GetOrSetJSON[T]` 内部处理 JSON 序列化/反序列化。

---

## Caller Audit 清单

| 调用方 | 文件:行号 | 当前传入实参 | 迁后应传实参 |
|--------|----------|-------------|-------------|
| `mac_history_router.go` | :13 | `NewMACHistoryQueryService(core.GetDB())` | 不变（无缓存构造） |
| `mac_history_router.go` | :19 | `NewMACHistoryHeatmapService(core.GetDB(), nil, nil)` | `NewMACHistoryHeatmapService(core.GetDB(), system.NewCacheProvider(core.DataCacheService), core.CacheConfigService)` |
| `cmd/main.go` | :333 | `NewMACHistoryQueryService(db)`（用于 importOUIData） | 不变（无缓存构造，OUI 导入不需要缓存） |
| `reconciliation_router.go` | :34 | `asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvc)` | `asset.NewReconciliationService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService), exceptionSvc)` |
| `router.go` | :619 | `asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvcForWs)` | `asset.NewReconciliationService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService), exceptionSvcForWs)` |
| `ai_service.go` | :84 | `NewSelectorLearner(db, cache, cfg)` | `NewSelectorLearner(db, system.NewCacheProvider(cache), cfg)` |

**测试文件调用点（需适配）：**

| 测试文件 | 调用 | 适配方案 |
|---------|------|---------|
| `mac_history_query_service_79_05_test.go:140` | `NewMACHistoryQueryServiceWithCache(db, NewDataCacheService(...), nil)` | 传入 `system.NewCacheProvider(NewDataCacheService(...))` |
| `mac_history_tail_79_05_test.go:57,728` | `NewMACHistoryHeatmapService(db, NewDataCacheService(...), ...)` | 同上 |
| `mac_history_tail_79_05_test.go:694,717` | `NewMACHistoryHeatmapService(db, nil, nil)` | 传入 `base.NoOpCacheProvider{}` |
| `reconciliation_sqlite_runtime_test.go:68,89,94,103` | `NewReconciliationService(db, nil, nil)` | 传入 `base.NoOpCacheProvider{}` |
| `asset_gapfill_test.go:203,210,241,300,346` | `NewReconciliationService(db, nil, mem, nil)` / `NewReconciliationService(db, nil, nil)` | 传入 `base.NoOpCacheProvider{}` 或 mock |
| `ai_selector_excel_test.go:286,328,348` | `NewSelectorLearner(db, &fakeSelectorCache{}, ...)` | `fakeSelectorCache` 需实现 `base.CacheProvider` 接口；最简方案是创建一个实现 `CacheProvider` 接口的 fake |
| `setup_routers_test.go:250` | 断言 `NewMACHistoryHeatmapService(core.GetDB(), nil, nil)` | 断言字面量需更新 |

**`base.NoOpCacheProvider` 可用性：**
`base.NoOpCacheProvider` 实现完整 `CacheProvider` 接口（:59-146），可直接传入测试，query 函数会被执行（不做缓存）。

---

## invariants 扩口的 AST 扫描机制

### 现有扫描逻辑（`cache_invariants_92_test.go`）

**检测原理：** AST 遍历，匹配：
1. `CallExpr.Fun` 是 `.GetOrSet` selector
2. 任一实参是 `FuncLit`（闭包）
3. 闭包签名返回类型第一项是 `interface{}`（`isEmptyInterface` 检测）
4. 闭包签名第二项是 `error`

**双档机制：**
- **硬档**（`hardDirs` = system + operations）：残留 > `allowedResidues[name]` 即 `t.Errorf`
- **warning 档**（`warnDirs` = duty/knowledge/network/workorder）：仅 `t.Logf` 计数，不 fail

**allowedResidues 白名单结构：**
```go
var allowedResidues = map[string]int{}  // 初始全空 = 期望 0
```

### D-103-16..18 扩口方案

**方案 A（推荐）：** 在 `cache_invariants_92_test.go` 的 `TestNoInterfaceGetOrSetResidue` 函数中扩展 `hardDirs`，追加：
```go
hardDirs := []string{
    filepath.Join(servicesRoot, "system"),
    filepath.Join(servicesRoot, "operations"),
    // Phase 103 扩口：
    servicesRoot,                    // mac_history_query_service.go, mac_history_heatmap_service.go
    filepath.Join(servicesRoot, "asset"),   // reconciliation_service.go
    filepath.Join(servicesRoot, "rpa"),     // selector_learner.go
}
```

**方案 B（独立文件）：** 创建 `cache_invariants_103_test.go`，包含针对三域的独立测试函数。优点：不影响 Phase 92 既有扫描的稳定性；缺点：两文件并存增加维护成本。

**手写 cache-aside AST 模式检测（D-103-18）：**

检测 `cache.Get` 后紧跟 `json.Unmarshal` + `cache.Set` 含 `json.Marshal` 序列：
```
// 目标模式：
cache.Get(ctx, key) → json.Unmarshal → cache.Set(ctx, key, json.Marshal(...))
```

AST 可行性分析：
- `cache.Get` 调用匹配：`CallExpr.Fun.Sel.Name == "Get"` + receiver 类型检查
- `json.Unmarshal` 调用匹配：`CallExpr.Fun.Sel.Sel.Name == "Unmarshal"` + `Ident.Name == "json"`
- `cache.Set` + `json.Marshal` 模式匹配：同文件内顺序检测

**误报风险：** 低——该模式是手写 cache-aside 的标志性序列，Phase 92 迁移后应归零。

---

## Common Pitfalls

### Pitfall 1: cache.Cache vs base.CacheProvider 类型混淆
**What goes wrong:** 三域字段原有 `cache cache.Cache` 类型，直接赋值 `base.CacheProvider` 接口值会编译失败（类型不兼容）。必须改字段类型为 `base.CacheProvider`。
**How to avoid:** 字段类型从 `cache.Cache` 改为 `base.CacheProvider` 是 D-103-1 锁定决策，按决策执行即可。

### Pitfall 2: vendor lookup 迁后 `s.cache` 可能为 nil
**What goes wrong:** `mac_history_query_service.go` 的 vendor lookup 使用 `s.cache` 字段（:260-284），当前代码已有 nil 检查。迁后 `base.GetOrSetJSON` 内部已处理 nil（`NoOpCacheProvider` query 直接执行），但传入 `base.NoOpCacheProvider{}` vs `nil` 有差异。
**How to avoid:** `NewMACHistoryQueryServiceWithCache` 若传入 `nil`，`base.GetOrSetJSON` 内部走 `NoOpCacheProvider`（传入 nil 时 `CacheProvider` 接口值为 nil，调用会 panic）。正确做法：若缓存不可用，显式传 `base.NoOpCacheProvider{}`。

### Pitfall 3: reconciliation Marshal 错误静默丢失
**What goes wrong:** 当前 `reconciliation_service.go:815` Marshal 错误被静默忽略（`if mErr == nil { _ = s.cache.Set(...)}`）。迁后 wrapper 内 `base.GetOrSetJSON` 内部 `DataCacheService.GetOrSet` 同样静默忽略 Marshal 错误（D-103-7 锁定保留）。
**How to avoid:** D-103-7 已锁定此行为，wrapper 内保持与现状等价即可。

### Pitfall 4: selector_learner `nil, nil` 返回值被调用方误用
**What goes wrong:** D-103-8 锁定 selector_learner best-effort 返回 `nil, nil`。若调用方检查 `err != nil` 则正常；若调用方只检查 `result != nil` 会误认为 cache miss 而走 DB。
**How to avoid:** D-103-8 锁定此行为；调用方（`selector_learner.go:169-174` 自己）检查 `err == nil && result != nil`。

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `interface{}` 闭包 GetOrSet | `base.GetOrSetJSON[T]` 泛型 | Phase 92 v1.29 | 编译期类型安全，消除 var 中转 |
| 手写 cache-aside (GetJSON + Marshal + Set) | wrapper helper + `base.GetOrSetJSON[T]` | Phase 103 v1.31 | 统一代码形态，行为等价 |
| `cache.Cache` 字段类型 | `base.CacheProvider` 接口类型 | Phase 103 v1.31 | 统一抽象层，零依赖具体实现 |
| invariants 扫描仅覆盖 system/operations | 扩口至 services 根 / asset / rpa | Phase 103 v1.31 | 守护新收敛面不回潮 |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `base.NoOpCacheProvider` 在传入 nil 时 panic（而应传入 `base.NoOpCacheProvider{}` 实例） | Caller Audit | 高——若误传 nil 导致 panic，测试失败。需在 `base.GetOrSetJSON` 调用前做 nil 检查，或显式传 `NoOpCacheProvider{}` |
| A2 | `system.NewCacheProvider(core.DataCacheService)` 返回的 adapter 实现了完整的 `CacheProvider` 9 方法 | Caller Audit | 低——Phase 92 已验证 |

---

## Open Questions

1. **heatmap fallback helper 命名与位置**
   - What we know: `queryHeatmapFromMV` 已存在，fallback wrapper 需要决定放在 service impl 内作为私有方法还是独立的包级函数
   - What's unclear: helper 命名（`getHeatmapWithFallback` vs `tryCacheThenQuery`）
   - Recommendation: Claude's Discretion，按 `getHeatmapWithFallback` 作为 service impl 私有方法

2. **invariants 新测试文件命名**
   - What we know: 方案 A 扩口 `cache_invariants_92_test.go` vs 方案 B 新建 `cache_invariants_103_test.go`
   - Recommendation: 方案 A 更优（扩口而非新建），避免两文件并存

3. **`dataCache` 字段删除时机**
   - What we know: D-103-1 锁定删除，但字段可能同时被多个代码路径引用
   - What's unclear: 是否有其他代码路径（非迁移位点）引用了 `dataCache` 字段
   - Recommendation: 迁移前 grep 确认 `dataCache` 仅被迁移位点使用

---

## Environment Availability

> Step 2.6: SKIPPED — 本相为代码重构，无外部工具/服务依赖

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + `testify` |
| Config file | 无独立配置 |
| Quick run command | `go test ./internal/services/... -run "TestNoInterfaceGetOrSetResidue" -v` |
| Full suite command | `go test ./internal/services/...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|---------------|
| CONV-01 | mac_history 4 处 GetOrSet + vendor cache-aside 迁 `base.GetOrSetJSON[T]` | 等价语义验证 | `go test ./internal/services/ -run "MAC|Heatmap" -v` | 部分 |
| CONV-02 | reconciliation GetByWorkstation 手写读穿透迁 `base.GetOrSetJSON[T]` + warn-on-set 不阻断 | 等价语义验证 | `go test ./internal/services/asset/ -run "Reconciliation" -v` | 部分 |
| CONV-03 | selector_learner 手写 JSON cache-aside 迁 `base.GetOrSetJSON[T]` + best-effort | 等价语义验证 | `go test ./internal/services/rpa/ -run "Selector" -v` | 部分 |
| CONV-04 | invariants 扩口：4 文件 interface{} 闭包 GetOrSet 硬失败档归零 | AST 扫描验证 | `go test ./internal/services/system/ -run "TestNoInterfaceGetOrSetResidue" -v` | 是 |

### Sampling Rate
- **Per task commit:** `go test ./internal/services/... -count=1`
- **Per wave merge:** `go test ./internal/services/...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

**不存在** — 现有测试基础设施已覆盖三域：
- `internal/services/mac_history_query_service_79_05_test.go` — mac_history query service 测试
- `internal/services/mac_history_tail_79_05_test.go` — heatmap service 测试
- `internal/services/asset/reconciliation_sqlite_runtime_test.go` — reconciliation 测试
- `internal/services/asset/asset_gapfill_test.go` — asset gapfill + reconciliation 集成测试
- `internal/services/rpa/ai_selector_excel_test.go` — selector_learner 测试
- `internal/services/system/cache_invariants_92_test.go` — AST 扫描测试

### 验证维度（可执行断言）

| 维度 | 断言手段 | 通过标准 |
|------|---------|---------|
| TTL 等价 | 对比迁移前后 TTL 来源（`s.perfCacheTTL()` vs `constants.reconciliationHealthCacheTTL` vs 字面量 `30*time.Minute`） | 值完全一致 |
| Cache key 字符串等价 | 对比迁移前后 cache key 值（`constants.MacVendorKeyFormat` vs `asset.GetReconciliationHealthByWorkstationKey` vs `systemServices.GetRpaSelectorBestKey`） | 字符串完全一致 |
| 错误语义等价（4 种） | 为每种语义编写单测：mac_history 严格 / reconciliation warn-on-set / selector best-effort / heatmap fallback | 每种错误场景下行为与迁移前等价 |
| 命中/回源/回填时序等价 | 为 mac_history query / reconciliation / selector 各写一个 cache hit/miss 单测 | hit 返回 cached；miss 执行 query 并回填 |
| `Workstation.ID != ""` 脏缓存防御保留 | 在 reconciliation 测试中构造 `Workstation.ID == ""` 的 cached 值 | 防御逻辑保留（hit 时返回有效 cached） |
| "Unknown Vendor" 占位保留 | 测试 vendor lookup 命 DB 中不存在 OUI | 返回 "Unknown Vendor"，写缓存成功 |
| Constructor 签名变更编译通过 | `go build ./...` | 编译 0 错误 |

### 现有 AST 扫描回归测试
```bash
# 运行 Phase 92 既有 invariants 测试（应绿）
go test ./internal/services/system/ -run "TestNoInterfaceGetOrSetResidue" -v

# Phase 103 扩口后，4 文件应加入 hardDirs，interface{} 闭包 GetOrSet 计数应为 0
# 手写 cache-aside 模式扫描新增测试（若实现 cache_invariants_103_test.go）：
go test ./internal/services/system/ -run "TestNoHandwrittenCacheAside" -v
```

---

## Security Domain

> 本相为行为等价重构，无安全相关变更。手写 cache-aside 迁移到 `base.GetOrSetJSON[T]` 不改变安全态势。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | 否 | 不涉及用户输入验证变更 |
| V6 Cryptography | 否 | 不涉及加密操作变更 |

---

## Sources

### Primary (HIGH confidence)
- `internal/services/base/cache_functions.go` — `GetOrSetJSON[T]` / `Invalidate` / `InvalidatePattern` 实现
- `internal/services/base/cache_provider.go` — `CacheProvider` interface 9 方法定义
- `internal/services/base/cache_service_base.go` — `CacheServiceBase` + `TTLResolver`
- `internal/services/system/cache_invariants_92_test.go` — Phase 92 AST 扫描实现（扩口参照）
- `internal/services/system/adapter.go` — `cacheProviderAdapter` 实现（`NewCacheProvider`）

### Secondary (MEDIUM confidence)
- `internal/services/mac_history_query_service.go` — 迁移位点当前代码
- `internal/services/mac_history_heatmap_service.go` — 迁移位点当前代码
- `internal/services/asset/reconciliation_service.go` — 迁移位点当前代码
- `internal/services/rpa/selector_learner.go` — 迁移位点当前代码
- `pkg/constants/cache.go` — Phase 102 已注册键常量

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — `base.GetOrSetJSON[T]` Phase 92 已验证
- Architecture: HIGH — 迁移路径明确，4 种 wrapper helper 语义已锁定
- Pitfalls: HIGH — 已知类型替换风险和 nil 防护需求

**Research date:** 2026-09-07
**Valid until:** 2026-10-07（30 天，技术债清偿类 phase 无快速演进风险）
