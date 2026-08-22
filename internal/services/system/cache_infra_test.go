package system

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	requests "github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-07: cache_keys / cache_manager / cache_adapter /
// cache_provider(NoOp) / adapter(DataCacheService 适配) 测试。
// 底层缓存统一用 pkg/cache MemoryCache 真实现。
// =====================================================================

func newInfraMemCache() *pkgcache.MemoryCache {
	return pkgcache.NewMemoryCache(100, time.Minute)
}

func TestCacheKeyManager_BuildParsePattern(t *testing.T) {
	m := NewCacheKeyManager("xingran")
	assert.Equal(t, "xingran:user:id:u1", m.Build("user", "id", "u1"))
	// QUIRK(行为锁定):BuildPattern 只在结尾追加 *,不插 ":"
	// → "xingran:user*" 也会命中 "xingran:userX" 等跨命名空间键
	assert.Equal(t, "xingran:user*", m.BuildPattern("user"))
	assert.Equal(t, "xingran:user:id*", m.BuildPattern("user", "id"))
	assert.Equal(t, "user:id:u1", m.Parse("xingran:user:id:u1"))
	assert.Equal(t, "other:key", m.Parse("other:key"), "无前缀键原样返回")

	// 空前缀
	plain := NewCacheKeyManager("")
	assert.Equal(t, "user:id", plain.Build("user", "id"))
	assert.Equal(t, "user:id*", plain.BuildPattern("user", "id"))
	assert.Equal(t, "k", plain.Parse("k"))
}

func TestCacheKeyBuilders(t *testing.T) {
	assert.Equal(t, "cache:user:id:u1", BuildUserCacheKey("id", "u1"))
	assert.Equal(t, "cache:role:list", BuildRoleCacheKey("list"))
	assert.Equal(t, "cache:menu:tree", BuildMenuCacheKey("tree"))
	assert.Equal(t, "cache:dept:id:d1", BuildDeptCacheKey("id", "d1"))
	assert.Equal(t, "cache:post:all", BuildPostCacheKey("all"))
	assert.Equal(t, "cache:dict:data:status", BuildDictCacheKey("data", "status"))
	assert.Equal(t, "cache:config:key:k", BuildConfigCacheKey("key", "k"))
	assert.Equal(t, "cache:operations:building:id:b1", BuildBuildingCacheKey("id", "b1"))
	assert.Equal(t, "cache:operations:floor:list", BuildFloorCacheKey("list"))
	assert.Equal(t, "cache:operations:workstation:id:w1", BuildWorkstationCacheKey("id", "w1"))

	// 模式与失效 pattern
	assert.Equal(t, "cache:user:*", BuildModulePattern(ModuleUser))
	assert.Equal(t, "cache:menu:*", BuildInvalidatePattern(ModuleMenu, ""))
	assert.Equal(t, "cache:user:id:*", BuildInvalidatePattern(ModuleUser, "id"))

	// TTL 表
	assert.Equal(t, time.Duration(CacheTTLMedium)*time.Second, GetCacheTTL(CacheKeyUserByID))
	assert.Equal(t, time.Duration(CacheTTLLong)*time.Second, GetCacheTTL(CacheKeyRoleByID))
	assert.Equal(t, time.Duration(CacheTTLVeryLong)*time.Second, GetCacheTTL(CacheKeyPostByID))
	assert.Equal(t, time.Duration(CacheTTLShort)*time.Second, GetCacheTTL(CacheKeyConfigByKey))
	assert.Equal(t, time.Duration(CacheTTLMedium)*time.Second, GetCacheTTL("unknown"), "默认中期")

	assert.Equal(t, CacheKeyDictData+":status", GetDictDataByTypeKey("status"))
}

func TestCacheManager_WarmUpAndInvalidate(t *testing.T) {
	ctx := context.Background()
	mem := newInfraMemCache()
	provider := NewCacheAdapter(mem)
	mgr := NewCacheManager(provider, "xingran", true)

	// 预热:任务成功/失败混合
	calls := map[string]int{}
	okFunc := func(name string) WarmUpFunc {
		return func(_ context.Context, c CacheProvider) error {
			calls[name]++
			return c.Delete(ctx, "warmup:"+name) // 顺带覆盖 Delete
		}
	}
	failFunc := func(_ context.Context, _ CacheProvider) error { return assert.AnError }

	err := mgr.WarmUp(ctx, map[string]WarmUpFunc{
		"a": okFunc("a"),
		"b": failFunc,
	})
	require.Error(t, err, "含失败任务应返回错误")
	assert.Equal(t, 1, calls["a"])
	assert.True(t, mgr.IsWarmedUp("a"), "成功任务应标记已预热")
	assert.False(t, mgr.IsWarmedUp("b"))

	// 重复预热:已预热的跳过
	err = mgr.WarmUp(ctx, map[string]WarmUpFunc{
		"a": okFunc("a"),
		"c": okFunc("c"),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls["a"], "已预热任务不应重复执行")
	assert.Equal(t, 1, calls["c"])

	// disabled → 直接跳过
	disabled := NewCacheManager(provider, "xingran", false)
	require.NoError(t, disabled.WarmUp(ctx, map[string]WarmUpFunc{"x": failFunc}))

	// 失效:ByModule / ByPattern / ByKey
	require.NoError(t, mem.Set(ctx, "cache:user:id:u1", "v", time.Minute))
	require.NoError(t, mem.Set(ctx, "cache:user:list", "v", time.Minute))
	require.NoError(t, mem.Set(ctx, "cache:role:id:r1", "v", time.Minute))
	require.NoError(t, mgr.InvalidateByModule(ctx, ModuleUser, ""))
	keys, _ := mem.Keys(ctx, "cache:user:*")
	assert.Empty(t, keys)
	require.NoError(t, mgr.InvalidateByModule(ctx, ModuleRole, "id"))
	keys, _ = mem.Keys(ctx, "cache:role:*")
	assert.Empty(t, keys)

	require.NoError(t, mem.Set(ctx, "k1", "1", time.Minute))
	require.NoError(t, mem.Set(ctx, "k2", "2", time.Minute))
	require.NoError(t, mgr.InvalidateByPattern(ctx, "k*"))
	keys, _ = mem.Keys(ctx, "k*")
	assert.Empty(t, keys)

	require.NoError(t, mem.Set(ctx, "j1", "1", time.Minute))
	require.NoError(t, mgr.InvalidateByKey(ctx, "j1"))
	exists, err := mgr.Exists(ctx, "j1")
	require.NoError(t, err)
	assert.False(t, exists)
	require.NoError(t, mgr.InvalidateByKey(ctx), "空键列表 no-op")

	// 键构建 + TTL
	assert.Equal(t, "xingran:user:id", mgr.BuildKey("user", "id"))
	assert.Equal(t, "xingran:user*", mgr.BuildPattern("user"))
	require.NoError(t, mem.Set(ctx, "ttl-key", "v", time.Minute))
	ttl, err := mgr.GetTTL(ctx, "ttl-key")
	require.NoError(t, err)
	assert.Positive(t, ttl)

	// 统计 + 预热记录清理
	stats, err := mgr.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, "xingran", stats.KeyManagerPrefix)
	assert.True(t, stats.WarmUpEnabled)
	assert.Positive(t, stats.WarmedUpCount)
	assert.Positive(t, stats.KeyCount, "MemoryCache 键数量应>0")

	mgr.ClearWarmUpCache()
	assert.False(t, mgr.IsWarmedUp("a"))
}

// stubXxxListSvc 嵌入接口 + 只实现 List,供 WarmUpXxxCache 使用。
type stubUserListSvc struct {
	UserService
	err error
}

func (s *stubUserListSvc) List(_ context.Context, _ requests.UserListParams) (*PageResult, error) {
	return nil, s.err
}

type stubRoleListSvc struct {
	RoleService
	err error
}

func (s *stubRoleListSvc) List(_ context.Context, _ requests.RoleListParams) (*PageResult, error) {
	return nil, s.err
}

type stubMenuListSvc struct {
	MenuService
	err error
}

func (s *stubMenuListSvc) List(_ context.Context, _ requests.MenuListParams) ([]models.Menu, error) {
	return nil, s.err
}

func (s *stubMenuListSvc) GetTree(_ context.Context) ([]models.Menu, error) {
	return nil, s.err
}

type stubDeptListSvc struct {
	DepartmentService
	err error
}

func (s *stubDeptListSvc) List(_ context.Context, _ requests.DepartmentListParams) ([]models.Department, error) {
	return nil, s.err
}

func (s *stubDeptListSvc) GetTree(_ context.Context, _ bool) ([]*models.Department, error) {
	return nil, s.err
}

type stubPostListSvc struct {
	PostService
	err error
}

func (s *stubPostListSvc) List(_ context.Context, _ requests.PostListParams) (*PageResult, error) {
	return nil, s.err
}

func TestCacheManager_PredefinedWarmUpFuncs(t *testing.T) {
	ctx := context.Background()
	provider := NewCacheAdapter(newInfraMemCache())

	require.NoError(t, WarmUpUserCache(&stubUserListSvc{})(ctx, provider))
	require.Error(t, WarmUpUserCache(&stubUserListSvc{err: assert.AnError})(ctx, provider))
	require.NoError(t, WarmUpRoleCache(&stubRoleListSvc{})(ctx, provider))
	require.Error(t, WarmUpRoleCache(&stubRoleListSvc{err: assert.AnError})(ctx, provider))
	require.NoError(t, WarmUpMenuCache(&stubMenuListSvc{})(ctx, provider))
	require.NoError(t, WarmUpDeptCache(&stubDeptListSvc{})(ctx, provider))
	require.NoError(t, WarmUpPostCache(&stubPostListSvc{})(ctx, provider))
}

func TestCacheAdapter_AllMethods(t *testing.T) {
	ctx := context.Background()
	mem := newInfraMemCache()
	adapter := NewCacheAdapter(mem)

	// GetOrSet:未命中 → 查询 → 缓存;命中 → 反序列化
	type item struct {
		Name string `json:"name"`
	}
	var got item
	calls := 0
	require.NoError(t, adapter.GetOrSet(ctx, "g1", &got, time.Minute, func() (interface{}, error) {
		calls++
		return item{Name: "v1"}, nil
	}))
	assert.Equal(t, "v1", got.Name)
	require.NoError(t, adapter.GetOrSet(ctx, "g1", &got, time.Minute, func() (interface{}, error) {
		calls++
		return item{Name: "v2"}, nil
	}))
	assert.Equal(t, 1, calls, "缓存命中后不应重复查询")
	assert.Equal(t, "v1", got.Name, "命中应返回缓存值")

	// 查询报错透传
	require.Error(t, adapter.GetOrSet(ctx, "g-err", &got, time.Minute, func() (interface{}, error) {
		return nil, assert.AnError
	}))

	// 缓存值损坏 → 回源查询
	require.NoError(t, mem.Set(ctx, "g-bad", "not-json", time.Minute))
	require.NoError(t, adapter.GetOrSet(ctx, "g-bad", &got, time.Minute, func() (interface{}, error) {
		return item{Name: "fresh"}, nil
	}))
	assert.Equal(t, "fresh", got.Name)

	// MGet / MDelete / Exists / SetTTL / GetTTL / Delete / DeleteByPattern
	require.NoError(t, mem.Set(ctx, "m1", "1", time.Minute))
	require.NoError(t, mem.Set(ctx, "m2", "2", time.Minute))
	vals, err := adapter.MGet(ctx, "m1", "m2", "missing")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"m1": "1", "m2": "2"}, vals)
	empty, err := adapter.MGet(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)

	exists, err := adapter.Exists(ctx, "m1")
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, adapter.SetTTL(ctx, "m1", 30*time.Second))
	ttl, err := adapter.GetTTL(ctx, "m1")
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, int64(ttl), int64(30*time.Second))

	require.NoError(t, adapter.MDelete(ctx, "m2"))
	require.NoError(t, adapter.MDelete(ctx))

	require.NoError(t, adapter.Delete(ctx, "m1"))

	require.NoError(t, mem.Set(ctx, "pat:a", "1", time.Minute))
	require.NoError(t, mem.Set(ctx, "pat:b", "1", time.Minute))
	require.NoError(t, adapter.DeleteByPattern(ctx, "pat:*"))
	keys, _ := mem.Keys(ctx, "pat:*")
	assert.Empty(t, keys)
	require.NoError(t, adapter.DeleteByPattern(ctx, "nothing:*"), "无匹配键时 no-op")

	// GetStats:MemoryCache 无 GetStats 方法 → 走默认 map + Keys 计数
	require.NoError(t, mem.FlushDB(ctx))
	require.NoError(t, mem.Set(ctx, "stat-key", "v", time.Minute))
	stats, err := adapter.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.KeyCount)
	assert.Equal(t, int64(1), stats.Count)

	// setValue 反射赋值
	var dst int
	setValue(&dst, 42)
	assert.Equal(t, 42, dst)
	setValue(nil, 1)      // nil dest no-panic
	var notPtr int
	setValue(notPtr, 1)   // 非指针 no-panic
	var typeMismatch string
	setValue(&typeMismatch, 1) // 类型不匹配 no-panic
	_ = notPtr
}

func TestNoOpCacheProvider_AllMethods(t *testing.T) {
	ctx := context.Background()
	p := &NoOpCacheProvider{}

	var got int
	require.NoError(t, p.GetOrSet(ctx, "k", &got, time.Minute, func() (interface{}, error) { return 7, nil }))
	assert.Equal(t, 7, got)
	require.Error(t, p.GetOrSet(ctx, "k", &got, time.Minute, func() (interface{}, error) { return nil, assert.AnError }))

	vals, err := p.MGet(ctx, "a", "b")
	require.NoError(t, err)
	assert.Empty(t, vals)
	require.NoError(t, p.MDelete(ctx, "a"))
	exists, err := p.Exists(ctx, "a")
	require.NoError(t, err)
	assert.False(t, exists)
	require.NoError(t, p.SetTTL(ctx, "a", time.Minute))
	ttl, err := p.GetTTL(ctx, "a")
	require.NoError(t, err)
	assert.Zero(t, ttl)
	require.NoError(t, p.Delete(ctx, "a"))
	require.NoError(t, p.DeleteByPattern(ctx, "*"))
	stats, err := p.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, CacheStats{}, *stats)
}

// TestDataCacheProviderAdapter adapter.go:DataCacheService → CacheProvider。
func TestDataCacheProviderAdapter(t *testing.T) {
	ctx := context.Background()
	mem := newInfraMemCache()
	p := NewCacheProvider(services.NewDataCacheService(mem))

	type box struct {
		N int `json:"n"`
	}
	var got box
	require.NoError(t, p.GetOrSet(ctx, "d1", &got, time.Minute, func() (interface{}, error) {
		return box{N: 5}, nil
	}))
	assert.Equal(t, 5, got.N)
	// DataCacheService 写缓存后二次取命中
	require.NoError(t, p.GetOrSet(ctx, "d1", &got, time.Minute, func() (interface{}, error) {
		return box{N: 9}, nil
	}))
	assert.Equal(t, 5, got.N)

	require.NoError(t, p.SetTTL(ctx, "d1", time.Minute))
	require.NoError(t, p.Delete(ctx, "d1"))

	require.NoError(t, mem.Set(ctx, "ops:1", "a", time.Minute))
	vals, err := p.MGet(ctx, "ops:1", "ops:2")
	require.NoError(t, err)
	assert.Equal(t, "a", vals["ops:1"])
	exists, err := p.Exists(ctx, "ops:1")
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, mem.Set(ctx, "del:1", "a", time.Minute))
	require.NoError(t, p.DeleteByPattern(ctx, "del:*"))
	keys, _ := mem.Keys(ctx, "del:*")
	assert.Empty(t, keys)

	_, err = p.GetTTL(ctx, "ops:1")
	require.NoError(t, err)
	stats, err := p.GetStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

// =====================================================================
// column_config_service.go
// =====================================================================

func newColumnConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:colcfg_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.UserColumnConfig{}))
	return db
}

func TestColumnConfigService_CRUD(t *testing.T) {
	db := newColumnConfigDB(t)
	svc := NewColumnConfigService(db)
	ctx := context.Background()

	// 空配置 → 返回默认列(asset 走 embed JSON)
	cols, err := svc.GetByPageKey(ctx, "u1", "asset.list")
	require.NoError(t, err)
	assert.NotEmpty(t, cols, "无用户配置时应回退默认列")

	// Save:重建用户配置
	req := &requests.ColumnConfigSaveRequest{
		PageKey: "asset.list",
		ColumnConfigs: []requests.ColumnConfigItem{
			{ColumnKey: "devicesn", Visible: true, Width: 120},
			{ColumnKey: "ip", Visible: false, Width: 100},
		},
	}
	require.NoError(t, svc.Save(ctx, "u1", req))
	cols, err = svc.GetByPageKey(ctx, "u1", "asset.list")
	require.NoError(t, err)
	require.Len(t, cols, 2)
	assert.Equal(t, "devicesn", cols[0].ColumnKey)
	// QUIRK(D-12 记录不修):Visible=false 是零值且模型带 gorm default:true,
	// GORM Create 对零值+default 字段省略该列 → DB 默认值 true 覆盖显式 false。
	assert.True(t, cols[1].Visible, "零值 false 被 default:true 吞掉(quirk)")

	// 覆盖保存 → 仍为 2 条
	req.ColumnConfigs = req.ColumnConfigs[:1]
	require.NoError(t, svc.Save(ctx, "u1", req))
	cols, _ = svc.GetByPageKey(ctx, "u1", "asset.list")
	assert.Len(t, cols, 1)

	// Reset → 清空用户配置回到默认
	require.NoError(t, svc.Reset(ctx, "u1", "asset.list"))
	cols, err = svc.GetByPageKey(ctx, "u1", "asset.list")
	require.NoError(t, err)
	assert.NotEmpty(t, cols)

	// GetDefaultConfig:各页面 key
	for _, page := range []string{"asset.list", "user.list", "role.list", "dept.list", "unknown-page"} {
		def, err := svc.GetDefaultConfig(ctx, page)
		require.NoError(t, err)
		_ = def // unknown-page 回退 asset 默认(embed)
	}
	defAsset, _ := svc.GetDefaultConfig(ctx, "asset.list")
	defUser, _ := svc.GetDefaultConfig(ctx, "user.list")
	assert.Equal(t, getDefaultColumnsForPage("asset.list")[0].Key, defAsset[0].ColumnKey)
	assert.Equal(t, getDefaultColumnsForPage("user.list")[0].Key, defUser[0].ColumnKey)
}
