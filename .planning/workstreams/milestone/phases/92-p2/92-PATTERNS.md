# Phase 92: 缓存层三处架构统一 - Pattern Map

**Mapped:** 2026-09-05
**Files analyzed:** 16（4 新建 + 12 修改，含 D-04 涟漪 19 处涉及的 5 个文件）
**Analogs found:** 16 / 16（CLAUDE.md 修订段为 role-match，其余均 exact）

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/base/cache_service_base.go` (NEW) | utility（零依赖抽象层） | request-response（读穿透 TTL 解析） | `internal/services/base/service.go`（风格）+ `internal/services/system/cache_utils.go:50-61`（内容迁出） | exact |
| `internal/services/base/cache_provider.go` (NEW) | utility（接口定义） | request-response | `internal/services/system/cache_provider.go`（逐字搬迁源） | exact |
| `internal/services/base/cache_functions.go` (NEW) | utility（泛型函数族） | request-response + batch（失效） | `internal/services/base/service.go` 函数式风格 + `system/cache_utils.go:63-79` 失效循环 | role-match（新代码，薄包装） |
| `internal/services/base/cache_service_base_test.go` (NEW) | test | request-response | `internal/services/data_cache_service_79_01_test.go`（miniredis 双装配）+ `base/service_test.go`（契约锁） | exact |
| `internal/services/system/cache_utils.go` (MOD) | utility | — | 自身（CacheServiceBase/Invalidate* 迁出，余下工具函数保留） | exact |
| `internal/services/system/cache_provider.go` (MOD) | utility | — | 自身（内容迁 base，原位留 alias） | exact |
| `internal/services/system/*_cache_impl.go` ×9 (MOD) | service（缓存装饰器） | request-response（读穿透） | `system/user_cache_impl.go`（5 处样本）+ `notice_cache_impl.go`（逃兵样本） | exact |
| `internal/services/operations/floor_cache_impl.go` (MOD) | service（缓存装饰器） | request-response | 自身（跨包嵌入样本，3 处样板） | exact |
| `internal/services/operations/cache_invalidator.go` (MOD) | service（失效分发器） | event-driven（Excel 导入管道后置失效） | 自身（D-04 委托改造对象） | exact |
| `internal/services/data_cache_service.go` (MOD) | service（基础设施底座） | request-response | 自身（D-06/D-07：GetExpiration 委托 + 定位注释） | exact |
| `internal/services/monitor/cache_service.go` (MOD) | service（监控） | request-response | 自身（D-08 rename 对象 :48-58） | exact |
| `internal/api/v1/monitor/cache_router.go` (MOD) | controller | request-response | 自身（D-08 rename 同步点 :42） | exact |
| `internal/core/core.go` + duty/knowledge/network/workorder cache_impl (MOD) | service/handler | request-response | 自身（D-04 涟漪 19 处机械改写） | exact |
| CLAUDE.md (MOD) | docs | — | CLAUDE.md 既有 Convention 段（:317 Pagination / Phase 90 D-11 格式先例） | role-match |
| invariants 扫描测试 (NEW) | test | — | `pkg/constants/pagination_test.go`（Phase 89 AST 锁值先例） | exact |

---

## Pattern Assignments

### `internal/services/base/cache_service_base.go` (utility, NEW — D-01)

**Analog:** `internal/services/base/service.go`（Phase 91 零依赖基线）+ `internal/services/system/cache_utils.go`（迁出源）

**Imports 模式**（base/service.go:1-9 — 零依赖基线，新文件 import 面只允许 `time` + 本包类型）:
```go
package base

import (
	"context"
	"fmt"

	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)
```
注意：base 包现仅依赖 gorm + pkg/errors。cache_service_base.go 只需 `time`（TTLResolver/GetExpiration 签名），**禁止 import `internal/services`**（会引入 root→base 反向依赖，破坏零依赖；root import base 是安全方向——root 现有 18 文件走此方向，实证无环）。

**迁出源——CacheServiceBase 现状**（system/cache_utils.go:50-61）:
```go
// CacheServiceBase 缓存服务基础结构
type CacheServiceBase struct {
	Config *services.CacheConfigService
}

// GetExpiration 获取缓存过期时间（通用方法）
func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if b.Config != nil {
		return b.Config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}
```

**目标形态——TTLResolver consumer-defined interface（D-01）**:
```go
// Source: 92-RESEARCH.md "TTLResolver 隐式满足" + cache_utils.go:50-61 合成
type TTLResolver interface {
	GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration
}

type CacheServiceBase struct {
	Config TTLResolver // 原 *services.CacheConfigService；*CacheConfigService 隐式满足
}

func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if b.Config != nil {
		return b.Config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}
```

**隐式实现端（不动签名，只加 nil-receiver 防护）**（internal/services/cache_config_service.go:391-401）:
```go
// GetDurationWithDefault 获取缓存时间配置，支持自定义默认值
func (s *CacheConfigService) GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration {
	s.mu.RLock()          // ← :393 nil receiver 在此 panic（Pitfall 1，必须加 if s == nil { return defaultDuration }）
	defer s.mu.RUnlock()

	if duration, ok := s.configs[configKey]; ok {
		return duration
	}

	return defaultDuration
}
```

**注释风格先例**（base/service.go:11-14，:17-21 — "有意设计"决策就地文档化）:
```go
// Scope 与 gorm 官方 scope 函数签名一致（gorm.Scopes 一等公民形态）。
//
// service 层把业务过滤/排序/JOIN 构造为 Scope 传给 GORMRepository.List/GetByID，
// repo 只消费 scope，不做任何业务解释（filter 语义是业务知识，repo 只承接管道）。
type Scope = func(*gorm.DB) *gorm.DB
```

---

### `internal/services/base/cache_provider.go` (utility, NEW — D-02 逐字搬迁)

**Analog:** `internal/services/system/cache_provider.go`（整体搬迁源，129 行）

**CacheProvider 接口 9 方法**（cache_provider.go:9-44，迁 base 时逐字保留）:
```go
// CacheProvider 缓存提供者接口
// 由 system 模块外部实现，用于解耦缓存逻辑
type CacheProvider interface {
	// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
	GetOrSet(
		ctx context.Context,
		key string,
		dest interface{},
		expiration time.Duration,
		query func() (interface{}, error),
	) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// DeleteByPattern 根据模式删除缓存
	DeleteByPattern(ctx context.Context, pattern string) error

	// MGet 批量获取缓存
	MGet(ctx context.Context, keys ...string) (map[string]string, error)

	// MDelete 批量删除缓存
	MDelete(ctx context.Context, keys ...string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// SetTTL 设置缓存过期时间
	SetTTL(ctx context.Context, key string, expiration time.Duration) error

	// GetTTL 获取缓存过期时间
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// GetStats 获取缓存统计信息
	GetStats(ctx context.Context) (*CacheStats, error)
}
```

**NoOpCacheProvider + 反射 setValue**（cache_provider.go:46-81 — D-03 后 setValue 缺陷被泛型函数绕开，但 NoOp 本体逐字保留）:
```go
// NoOpCacheProvider 空缓存提供者（用于无缓存场景）
type NoOpCacheProvider struct{}

func (n *NoOpCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	// 直接执行查询，不使用缓存
	result, err := query()
	if err != nil {
		return err
	}
	// 将结果设置到目标变量（使用反射）
	setValue(dest, result)
	return nil
}

// setValue 使用反射设置目标变量的值（:62-81 —AssignableTo 不成立时静默跳过，即 Pitfall 5 的"返回零值"缺陷源）
```
伴生类型 `CacheStats`（:83-92）/ `CacheEntry`（:94-101）及 NoOp 其余 7 方法（:103-128）同文件逐字迁。

**alias 落点——system/cache_provider.go 迁移后**（Go spec alias 语义 + RESEARCH Pattern 2）:
```go
type CacheProvider = base.CacheProvider
type NoOpCacheProvider = base.NoOpCacheProvider
type CacheStats = base.CacheStats
type CacheEntry = base.CacheEntry
type CacheServiceBase = base.CacheServiceBase // discretion 建议：10 个嵌入点构造字面量零改动

// 编译期双保险（建议）
var _ CacheProvider = (*NoOpCacheProvider)(nil)
```

**alias 后零改动的外部消费者实证**（grep 验证的编译期断言 5 处，alias = 同一类型，断言自动继续成立）:
- `internal/services/duty/duty_cache_impl_test.go:48` — `var _ systemServices.CacheProvider = (*mockCacheProvider)(nil)`
- `internal/services/knowledge/knowledge_cache_impl_test.go:40`
- `internal/services/network/cache_impl_test.go:42`
- `internal/services/workorder/service_test.go:475`
- `internal/services/monitor/cache_service_test.go:35` — `var _ MultiLevelCacheProvider = ...`（monitor 包内，不受影响）

---

### `internal/services/base/cache_functions.go` (utility, NEW — D-03/D-04)

**Analog:** `internal/services/base/service.go` 包级函数风格（WrapError :182-188）+ `system/cache_utils.go:63-79` 失效循环（迁出源）+ RESEARCH Pattern 1 目标形态

**被消灭的样板——五段式现状**（system/user_cache_impl.go:38-52，32 处的统一形状）:
```go
func (s *userCacheService) GetByIDWithCache(ctx context.Context, id string) (*models.User, error) {
	cacheKey := GetUserByIDKey(id)
	var result models.User

	expiration := s.GetExpiration(services.CacheConfigUserByID, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.userService.GetByID(ctx, id)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}
```

**目标形态——GetOrSetJSON 泛型薄包装**（委托 provider.GetOrSet，P0 #9 同步写/JSON 往返语义逐字保留）:
```go
// Source: 92-RESEARCH.md Pattern 1（本机 go1.24.5 编译验证通过）
// GetOrSetJSON 类型安全的读穿透缓存：命中反序列化，未命中执行 query 并同步写缓存。
func GetOrSetJSON[T any](
	ctx context.Context,
	p CacheProvider,
	key string,
	ttl time.Duration,
	query func() (T, error),
) (T, error) {
	var result T
	err := p.GetOrSet(ctx, key, &result, ttl, func() (interface{}, error) {
		return query()
	})
	return result, err
}
```

**迁移后调用点（12-15 行 → 3-5 行单 return）**（92-CONTEXT.md D-03 预览，用户已确认）:
```go
func (s *userCacheService) GetByIDWithCache(ctx context.Context, id string) (*models.User, error) {
	return base.GetOrSetJSON(ctx, s.cache,
		GetUserByIDKey(id),
		s.TTL(services.CacheConfigUserByID, 30*time.Minute),
		func() (*models.User, error) {
			return s.userService.GetByID(ctx, id)
		})
}
```
（`s.GetExpiration(...)` 来自嵌入基类——选预解析 Duration 形态时，30 个嵌入点该行零改动，结果作为参数传入泛型函数。）

**短返回形态参考**（user_cache_impl.go:84-96 — `return result, err` 直收的两处站点，迁移更简单）:
```go
func (s *userCacheService) GetRolesWithCache(ctx context.Context, userID string) ([]models.Role, error) {
	cacheKey := GetUserRolesKey(userID)
	var result []models.Role
	expiration := s.GetExpiration(services.CacheConfigUserRoles, 30*time.Minute)
	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryRoles(ctx, userID)
	})
	return result, err
}
```

**InvalidatePattern 失效底层归一**（cache_utils.go:63-70 现状 + cache_invalidator.go:39-45 现状合并，D-04）:
```go
// Source: system/cache_utils.go:63-79（迁 base 后删除原件）
// InvalidatePattern 按模式列表失效缓存（唯一底层：nil 防护 + 统一日志）。
func InvalidatePattern(ctx context.Context, p CacheProvider, patterns []string, module string) {
	if p == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", module)
		return
	}
	for _, pattern := range patterns {
		if err := p.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", module, pattern, err)
		}
	}
}
func Invalidate(ctx context.Context, p CacheProvider, keys []string, module string) { /* 同构, 用 Delete */ }
```
（日志格式对齐 cache_invalidator.go:41 的 `[%s] 清除缓存失败: pattern=%s, error=%v`——比 cache_utils.go:67 的版本多 pattern 字段，取信息量大的格式为统一形态。`Invalidate`/`InvalidatePattern` 签名（error vs void）为 discretion 区，void+warn 对齐现状。）

**SetJSON[T]**：D-03 锁定函数族成员，但实测 32 处调用点零个直接 Set 使用者（A7）——薄包装 ~10 行兑现 D-03 即可，无迁移对象。

**跨包失效调用涟漪（D-04 删除强制，19 处机械 1 行/处，grep 全量验证）**:
| 文件 | 行号 | 现状调用 |
|------|------|---------|
| `internal/core/core.go` | :366 | `system.InvalidateCacheByPattern(context.Background(), system.NewCacheProvider(c.DataCacheService), ...)` |
| `internal/services/knowledge/knowledge_cache_impl.go` | :252, :259, :266, :273 | `systemServices.InvalidateCacheByPattern/ByKey(...)` |
| `internal/services/workorder/workorder_cache_impl.go` | :259, :266, :273, :279 | 同上 |
| `internal/services/duty/duty_cache_impl.go` | :300, :307, :314, :321, :328 | 同上 |
| `internal/services/network/cache_impl.go` | :321, :328, :335, :342, :348 | 同上 |

改写模式：`systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "DUTY")` → `base.Invalidate(ctx, s.cache, keys, "DUTY")`（import 增加 base 包，duty/knowledge/network/workorder 现仅 import services root + systemServices，root→base 方向已实证无环）。

---

### `internal/services/base/cache_service_base_test.go` (test, NEW — D-09 / CACHE-UNIFY-05)

**Analog:** `internal/services/data_cache_service_79_01_test.go`（miniredis 双装配模板）+ `internal/services/base/service_test.go`（Phase 91 契约锁）

**测试纪律文件头先例**（data_cache_service_79_01_test.go:3-24 — 新测试文件照抄此头部纪律）:
```go
// =====================================================================
// Phase 79-01 Task 1: DataCacheService 全方法双装配测试(MemoryCache + miniredis)
//
// 关键纪律:
//   - 双装配 helper,名字带 plan 后缀(R5 防同包重名)
//   - t.Cleanup 单次 Close
//   - 禁 t.Parallel()(装配含后台清理 goroutine 与 miniredis 实例)
//   - TTL 推进一律 miniredis mr.FastForward(R-1 纪律,禁裸 time.Sleep)
// =====================================================================
```

**miniredis 装配 helper（Phase 92 变体，RESEARCH 已给出直用形态）**（data_cache_service_79_01_test.go:58-70 改写）:
```go
func newBase92Redis(t *testing.T) (base.CacheProvider, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)
	rc, err := cache.NewRedisCache(&cache.CacheConfig{Host: host, Port: port}, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	return system.NewCacheProvider(services.NewDataCacheService(rc)), mr
}
```

**表驱动双装配结构**（data_cache_service_79_01_test.go:72-106 — MemoryCache 纯进程内 + RedisCache 真 go-redis 握手）:
```go
func dcs7901Assemblies() []struct {
	name  string
	setup func(t *testing.T) *DataCacheService
} {
	return []struct {
		name  string
		setup func(t *testing.T) *DataCacheService
	}{
		{"MemoryCache", func(t *testing.T) *DataCacheService { ... }},
		{"RedisCache", func(t *testing.T) *DataCacheService { ... }},
	}
}

func TestDcs7901_SetGetRoundTrip(t *testing.T) {
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) { ... assert.Equal(t, want, got) })
	}
}
```

**断言面**（D-09 锁定）: GetOrSetJSON 命中/穿透（query 调用计数）、`mr.FastForward(ttl)` 后穿透、Invalidate/InvalidatePattern 后穿透、NoOp 透传（query 必调 + 不写缓存）、GetStats 可读、nil-config → default 分支。

**契约锁测试风格**（base/service_test.go:3-17 — 契约编号 + 纪律头部）:
```go
// =====================================================================
// Phase 91-01 Task 3: GORMRepository 泛型契约锁值（CRUD-REUSE-07）。
//
// 把 91-RESEARCH spike 验证的 GORM 链语义固化为可重跑契约：
//   契约 1 (P4): ...
//
// 纪律（v1.27 规约）：... 零 sleep、零 t.Parallel ...
// =====================================================================
```

---

### `internal/services/system/user_cache_impl.go` 及 8 个同族 impl (service, MOD — D-03)

**Analog:** 自身（user 是 5 处样本 + 嵌入模式范本）

**装饰器嵌入模式（迁移后结构不变，仅方法体内部替换）**（user_cache_impl.go:15-35）:
```go
// userCacheService 用户缓存服务
type userCacheService struct {
	*userService
	cache CacheProvider
	CacheServiceBase
}

// NewUserServiceWithCache 创建带缓存的用户服务
func NewUserServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
	pwdManager PasswordManager,
) UserService {
	base := &userService{db: db, pwdManager: pwdManager}
	return &userCacheService{
		userService:      base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},  // ← alias 后此字面量零改动
	}
}
```

**复杂站点 1——List（键构造已抽 helper，闭包返回 *PageResult）**（user_cache_impl.go:185-199）:
```go
func (s *userCacheService) List(ctx context.Context, params requests.UserListParams) (*PageResult, error) {
	// 构建缓存键
	cacheKey := s.buildListCacheKey(params)
	var result PageResult

	// 缓存时间：10分钟（列表数据变化较频繁）
	expiration := s.GetExpiration(services.CacheConfigUserList, 10*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.userService.List(ctx, params)
	})

	return &result, err
}
```
迁移注意：闭包返回 `*PageResult` 而 dest 为 `PageResult`——现状 NoOp 反射路径静默丢弃返回零值（Pitfall 5）；迁移为 `GetOrSetJSON[*PageResult]` 后 NoOp 路径反而修正为返回真实数据（单向改善，plan 注明防"修回去"）。

**逃逸样本——notice_cache_impl.go 未嵌入基类**（notice_cache_impl.go:42-70 — 唯一私有 getExpiration 的 impl，D-03 迁移时归队）:
```go
// noticeCacheService 通知公告缓存服务实现
type noticeCacheService struct {
	db     *gorm.DB
	base   *services.NoticeService
	cache  CacheProvider
	config *services.CacheConfigService
}

// getExpiration 获取缓存过期时间
func (s *noticeCacheService) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}
```
归队动作：struct 增加 `CacheServiceBase` 嵌入、构造函数增加 config→嵌入赋值、删除私有 getExpiration、调用点改 `s.GetExpiration`。

**复杂站点 2——匿名 struct 缓存值**（notice_cache_impl.go:202-233，Pitfall 6）:
```go
func (s *noticeCacheService) GetUserNotices(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error) {
	cacheKey := fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d", userID, page, pageSize)
	if status != nil {
		cacheKey = fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d:status:%s", userID, page, pageSize, *status)
	}
	var result struct {
		List  []models.Notice
		Total int64
	}

	expiration := s.getExpiration("cache.notice.my_notices", 1*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		list, total, err := s.base.GetUserNotices(ctx, userID, page, pageSize, status)
		if err != nil {
			return nil, err
		}
		return struct {
			List  []models.Notice
			Total int64
		}{List: list, Total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}
	return result.List, result.Total, nil
}
```
迁移建议（Pitfall 6）：就地声明具名局部类型 `type noticeListPage struct{ List []models.Notice; Total int64 }`，T 用具名类型（JSON 字段名不变，缓存数据兼容）；键 if/else 分支可抽 `buildMyNoticesKey(...)` helper 顺带减行。

**9 文件分布**（RESEARCH 实测 29 处）: menu(6) / role(6) / user(5) / config、department、dict、post、settings（各 1-3，dict 含双 struct dictTypeCacheService + dictDataCacheService）/ notice(2 + 逃兵归队)。

---

### `internal/services/operations/floor_cache_impl.go` (service, MOD — D-03/CACHE-UNIFY-03)

**Analog:** 自身（跨包嵌入唯一样本，实测 3 处真实调用点 :43/:60/:179）

**跨包嵌入模式**（floor_cache_impl.go:14-33 — alias 后零改动的关键验证点）:
```go
// floorCacheService 楼层缓存服务
type floorCacheService struct {
	*floorService
	cache system.CacheProvider
	system.CacheServiceBase
}

func NewFloorServiceWithCache(
	db *gorm.DB,
	cache system.CacheProvider,
	config *services.CacheConfigService,
) FloorService {
	base := &floorService{db: db}
	return &floorCacheService{
		floorService:     base,
		cache:            cache,
		CacheServiceBase: system.CacheServiceBase{Config: config},
	}
}
```

**短样板站点 GetTree**（:37-51）与**闭包直查 db 站点 GetFloorsByBuildingID**（:54-76，闭包内非委托 floorService，原样搬进 query 闭包形状仍匹配）:
```go
func (s *floorCacheService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
	cacheKey := "floor:tree"
	var result []FloorTreeNode
	expiration := s.GetExpiration("cache.floor.tree", 30*time.Minute)
	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.floorService.GetTree(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *floorCacheService) GetFloorsByBuildingID(ctx context.Context, buildingID string) ([]operations.OpsFloor, error) {
	cacheKey := fmt.Sprintf("floor:building:%s", buildingID)   // ← 内联键，本期不改（Redis 数据兼容）
	var result []operations.OpsFloor
	expiration := s.GetExpiration("cache.floor.building", 15*time.Minute)
	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		// 查询指定楼宇的所有楼层（闭包内直查 db，原样保留）
		var floors []operations.OpsFloor
		if err := s.db.WithContext(ctx).
			Where("building_id = ?", buildingID).
			Order("order_num ASC").
			Find(&floors).Error; err != nil {
			return nil, err
		}
		return floors, nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
```

**grep 假阳性锚点**（:167-168 — CONTEXT 的"4 处"多计来源）:
```go
// SearchFloorOptions 楼层下拉数据源(仅 name="" 路径走 5min Redis 缓存;
// keyword 查询绕过缓存避免 keyspace 爆炸)。参考 GetTree 的 cache.GetOrSet 模式。  ← :168 注释行
```
真实第 3 处在 :179（SearchFloorOptions 方法体内）。

---

### `internal/services/operations/cache_invalidator.go` (service, MOD — D-04 委托)

**Analog:** 自身（分发器保留，底层循环委托 base）

**现状完整逻辑**（cache_invalidator.go:21-48）:
```go
// InvalidateByEntityType 根据实体类型清理缓存
// patterns从ExcelConfig的CachePatterns字段获取
func (c *CacheInvalidator) InvalidateByEntityType(
	ctx context.Context,
	entityType string,
	patterns []string,
) error {
	if len(patterns) == 0 {
		logger.Debugf("[%s] 没有配置缓存清理模式", entityType)
		return nil
	}

	// 如果没有配置缓存，直接返回
	if c.cache == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", entityType)
		return nil
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", entityType, pattern, err)
		} else {
			logger.Debugf("[%s] 清除缓存成功: pattern=%s", entityType, pattern)
		}
	}

	return nil
}
```

**目标形态**（RESEARCH Pattern 3 — for 循环替换为 base 委托，nil 防护内聚于 base）:
```go
func (c *CacheInvalidator) InvalidateByEntityType(ctx context.Context, entityType string, patterns []string) error {
	if len(patterns) == 0 { /* Debugf 保留 */ }
	base.InvalidatePattern(ctx, c.cache, patterns, entityType) // nil 防护已内聚
	return nil
}
```
InvalidateByPatterns（:50-68）同构委托。struct 定义与构造函数（:10-19）零改动。

---

### `internal/services/data_cache_service.go` (service, MOD — D-06/D-07)

**Analog:** 自身（原地定性，仅两处动作）

**消除对象——平行 GetExpiration**（data_cache_service.go:54-60，生产代码 0 调用者，仅 2 测试锁定）:
```go
// GetExpiration 获取缓存过期时间
func (s *DataCacheService) GetExpiration(configKey string, defaultExpiration time.Duration) time.Duration {
	if s.cacheConfig != nil {
		return s.cacheConfig.GetDurationWithDefault(configKey, defaultExpiration)
	}
	return defaultExpiration
}
```
D-06 处置（A3）：保留方法签名，内部委托 base TTL 逻辑（root import base 无循环），2 个锁定测试（data_cache_service_79_01_test.go:351-359、cache_config_service_79_01_test.go:225-237）保持绿。

**P0 #9 语义锚点（薄包装红线——禁止在 GetOrSetJSON 里重写此逻辑）**（data_cache_service.go:99-107）:
```go
// P0 #9: 不再起裸 goroutine 异步写缓存。底层 cache（MultiLevelCache）的 Set
// 已自带 L1 同步写入 + L2 经 L2Writer worker pool 异步写入（带重试/队列/降级）。
if err := s.Set(context.Background(), key, data, expiration); err != nil {
	applogger.Warnf("[CACHE] GetOrSet 缓存写入失败: key=%s, err=%v", key, err)
}
```

**文件头定位注释先例（D-07 照抄格式）**（data_cache_service.go:16-36 — P2-A2 迁移说明块，双定位注释加在同位置）:
```go
// ==================== 缓存键命名规范迁移说明 ====================
//
// **状态 (P2-A2)**: 本文件 (data_cache_service.go) 已不再定义任何 CacheKey*
// 常量 — 缓存键定义已统一迁移到 internal/services/system/cache_keys.go,
// 该文件是项目内缓存键的 **单一真实来源 (single source of truth)**。
// ...
// ===========================================================
```

**装配链不动（回归验证范围）**: `internal/core/core.go:356-366,900,958` + `internal/core/core_services.go:33`；12+ API 文件 `services.DataCacheService` 引用零改动。`internal/services/system/adapter.go:12-17` Adaptee 适配不动:
```go
func NewCacheProvider(dataCache *services.DataCacheService) CacheProvider {
	if dataCache != nil {
		return &cacheProviderAdapter{dataCache: dataCache}
	}
	return &NoOpCacheProvider{}
}
```

---

### `internal/services/monitor/cache_service.go` + `internal/api/v1/monitor/cache_router.go` (MOD — D-08 rename)

**Analog:** 自身（纯包内 rename，语义零变更）

**rename 对象——monitor.CacheProvider**（cache_service.go:48-58）:
```go
// CacheProvider 缓存提供者接口          → 建议改名 CacheOperator（discretion，最终名 planner 定）
type CacheProvider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error
}
```

**不撞名不动**（cache_service.go:60-82）: `CacheConfigProvider`（:61）/ `MultiLevelCacheProvider`（:68）/ `DirectRedisProvider`（:73）/ `StatsProvider`（:80）。

**adapter 同步点**（cache_router.go:42 — 返回类型引用处）:
```go
// NewCacheProviderAdapter 创建缓存提供者适配器
func NewCacheProviderAdapter(core *core.Core) monitorServices.CacheProvider {   // → monitorServices.CacheOperator
	adapter := &CacheProviderAdapter{cache: core.Cache}
	...
	return adapter
}
```
rename 面实测 3 文件 ~58 处引用（cache_service.go 8 + cache_router.go 27 + cache_router_test.go 23，RESEARCH Open Question 3）。

---

### CLAUDE.md (docs, MOD — D-10①)

**Analog:** CLAUDE.md 既有 Convention 段格式（:317 "Pagination Constants Convention" / "Timeout/Port/Protocol Constants Convention"，Phase 90 D-11 先例同格式）

**修订落点（行号实测）:**
- `:188` — "2. **Dual Cache Architecture**"（root 6 件套 Legacy 描述失实 → 重写为 base 单一权威描述）
- `:682` — "### Cache System"（"Legacy"/"New" 两分法 → base 路径单权威）
- `:774` — "**Legacy Services (still used by core):**" 列表（dept/role/dict/menu/user/post root 文件已不存在 → 删除或改述）
- `:807` — "### Working with Cache"（"New code: Use internal/services/system/ pattern" → 改锁 `base.GetOrSetJSON` + `base.CacheProvider` 单一权威路径）
- `:1151` — "### CacheProvider Interface"（Architecture reference 段，接口已迁 base → 路径同步）

**新增 Cache Service Convention 段内容锚点（D-10①）:** 锁 `base.GetOrSetJSON[T]`/`SetJSON[T]`/`Invalidate`/`InvalidatePattern` + `base.CacheProvider` 为唯一权威；cache_impl 无 interface{} 闭包式 GetOrSet；键构造唯一真相源 `system/cache_keys.go`。

---

### invariants 扫描测试 (test, NEW — D-10②)

**Analog:** `pkg/constants/pagination_test.go`（Phase 89 AST 锁值先例，Stability + Count 双锁）

**AST 解析模式**（pagination_test.go:22-56）:
```go
func readPaginationConsts(filename string) (map[string]int, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		...
	}
	return result, nil
}
```

**Phase 92 变体**（RESEARCH 已定义）: 扫描 system/operations 的 `*_cache_impl.go`，检测 `.GetOrSet(ctx` 且闭包签名为 `func() (interface{}, error)` 的残留 → 期望 0 处；**warning 不 fail**（t.Logf + 宽松断言，Phase 89/90 模式）；白名单机制 `map[文件]允许数`（初始全 0，未来豁免显式登记）。

---

## Shared Patterns

### 1. nil-receiver 防护（Pitfall 1 — D-01 最高风险配套，全 plan 前置）
**Source:** `internal/services/cache_config_service.go:392-394`
**Apply to:** 92-01 base 抽取同 commit 必做
```go
func (s *CacheConfigService) GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration {
	if s == nil { // ← 新增：typed-nil interface 防护
		return defaultDuration
	}
	s.mu.RLock()
	...
}
```
**影响面实测（远超 RESEARCH 引用的 2 处）**: grep 验证 nil-config 测试构造 ~100 处——workorder/service_test.go（20+）、department_cache_impl_test.go（17）、dict_cache_impl_test.go（15）、menu_cache_impl_test.go（14）、post_cache_impl_test.go（14）、role_cache_impl_test.go:151、settings_cache_impl_test.go:108、asset/asset_gapfill_test.go:746，另 geocoding_photo_floor_test.go:379、notice_service_gapfill_test.go:169。现状 `Config` 是具体指针字段，nil 判空有效；改 TTLResolver 接口后 typed-nil 使 `if b.Config != nil` 失效 → `s.mu.RLock()` panic。不加密防护，迁移后 `go test ./internal/services/...` 将出现 nil pointer dereference（非断言失败）。

### 2. 缓存键构造逐字保留（CONTEXT 锁定）
**Source:** `internal/services/system/cache_keys.go`（367 行单一真相源，:9-52 CacheKeyManager + :54+ 常量）+ `floor_cache_impl.go:55` 内联键先例
**Apply to:** 32 处 GetOrSet 迁移——key 字符串 diff 级零变更；floor 的 `fmt.Sprintf("floor:building:%s", ...)` 内联键与 notice 的 `notice:my_notices:...` 内联键均不改（Redis 数据兼容）。

### 3. P0 #9 薄委托红线
**Source:** `internal/services/data_cache_service.go:99-107`
**Apply to:** base/cache_functions.go——GetOrSetJSON 只做类型包装，禁止重写 Get/Set/JSON/同步写逻辑（重新实现 = 行为漂移，违背 v1.29 D-05 零行为变更底线）。

### 4. type alias 翻转迁移机制（D-02）
**Source:** `system/cache_provider.go` 迁移后 alias 块（见 Pattern Assignments）+ 编译期断言 5 处（duty:48 / knowledge:40 / network:42 / workorder:475 / monitor:35）
**Apply to:** 92-01——alias 同 commit 翻转，40+ 引用文件零改动由 Go alias 语义保证；duty/knowledge/network/workorder/cache_impl_test.go 的 `var _ systemServices.CacheProvider` 断言自动继续成立。

### 5. miniredis 测试纪律
**Source:** `internal/services/data_cache_service_79_01_test.go:3-24`（头部纪律）+ :58-70（装配）
**Apply to:** base/cache_service_base_test.go——禁 t.Parallel、TTL 用 `mr.FastForward` 禁裸 sleep、t.Cleanup 单次 Close、双装配 helper 带 plan 后缀防同包重名。

### 6. 泛型语法红线（编译实证）
**Source:** 92-RESEARCH.md 编译实验（go1.24.5）
**Apply to:** base 包新代码——`func (b *CacheServiceBase) Get[T any]()` 非法（`method must have no type parameters`）；泛型只能放包级函数；匿名 struct 作类型实参合法但建议用具名局部类型（Pitfall 6）；接口 type alias 合法。

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| CLAUDE.md Cache Service Convention 新段 | docs | — | 无仓内代码类比；格式先例为 CLAUDE.md 自身 Convention 段（:317）+ Phase 90 D-11 收口惯例（planning 工件，commit 3a2efe5） |
| `internal/services/system/cache_adapter.go`（NewDataCacheAdapter） | — | — | 非迁移对象——生产 0 调用者死代码（A6），本期不动仅记录，v1.30 清理候选 |

## Metadata

**Analog search scope:** `internal/services/`（base / system / operations / root / monitor / duty / knowledge / network / workorder）、`internal/api/v1/monitor/`、`internal/core/`、`internal/services/system/`、`pkg/constants/`、`internal/services/cache_config_service.go`
**Files read:** 14（service.go / cache_utils.go / cache_provider.go / cache_invalidator.go / user_cache_impl.go / notice_cache_impl.go / floor_cache_impl.go / data_cache_service.go / adapter.go / monitor cache_service.go / cache_router.go / data_cache_service_79_01_test.go / base service_test.go / pagination_test.go + cache_keys.go + cache_config_service.go 定点段）
**Grep 验证:** 19 处外部失效调用面（逐 file:line）／nil-config 测试构造 ~100 处／编译期 mock 断言 5 处／CLAUDE.md 修订落点 6 处行号
**Pattern extraction date:** 2026-09-05
