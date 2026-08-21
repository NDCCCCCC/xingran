package monitor

// cache_service_test.go — Phase 73-04 Task 1 (D-02 portwrite 纯 mock 范本).
//
// 范本要点(镜像 internal/services/portwrite/port_write_service_test.go):
//   - 顶部 compile-time interface assertion 锁定 mockability 契约
//   - testify/mock 嵌入式 mock(mock.Mock + m.Called)
//   - 真实 cacheServiceImpl + mocked CacheProvider/CacheConfigProvider
//   - glebarez sqlite 仅用于不可避免 gorm 路径(DB fallback / 历史统计 / 配置持久化)
//
// 已锁定的业务 quirk(见 73-04-SUMMARY.md "Business-code quirks discovered — NOT fixed"):
//   Q1: normalizeCacheKeyForService 的 `key[:6] == "xingran:"` 用 6 字节切片比 8 字节字面量,
//       恒为 false —— 函数实际是恒等函数,前缀永不被剥离。测试按真实行为断言(恒等)。
//   Q2: getCachesFromSimpleCache / getCachesFromCacheWithLevel 对 Keys/KeysByLevel 的错误
//       静默吞掉(返回空列表 + nil error)。测试锁定该行为。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== Compile-time interface assertions ====================

var _ CacheProvider = (*mockMonitorCacheProvider)(nil)
var _ MultiLevelCacheProvider = (*mockMonitorFullCacheProvider)(nil)
var _ DirectRedisProvider = (*mockMonitorFullCacheProvider)(nil)
var _ StatsProvider = (*mockMonitorFullCacheProvider)(nil)
var _ CacheConfigProvider = (*mockCacheConfigProvider)(nil)

// mockMonitorCacheProvider 仅实现基础 CacheProvider(不实现可选接口),
// 用于 multiLevelCache/directRedis/statsProvider 均为 nil 的简单路径。
type mockMonitorCacheProvider struct {
	mock.Mock
}

func (m *mockMonitorCacheProvider) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockMonitorCacheProvider) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockMonitorCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockMonitorCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockMonitorCacheProvider) Expire(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

func (m *mockMonitorCacheProvider) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *mockMonitorCacheProvider) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockMonitorCacheProvider) FlushDB(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// mockMonitorFullCacheProvider 在基础 CacheProvider 之上实现全部可选接口
// (MultiLevelCacheProvider / DirectRedisProvider / StatsProvider),
// 用于验证 NewCacheService 的类型断言装配逻辑。
type mockMonitorFullCacheProvider struct {
	mockMonitorCacheProvider
}

func (m *mockMonitorFullCacheProvider) KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error) {
	args := m.Called(ctx, pattern, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockMonitorFullCacheProvider) DirectRedisKeys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockMonitorFullCacheProvider) DirectRedisGet(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockMonitorFullCacheProvider) DirectRedisTTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *mockMonitorFullCacheProvider) GetStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// mockCacheConfigProvider 实现 CacheConfigProvider。
type mockCacheConfigProvider struct {
	mock.Mock
}

func (m *mockCacheConfigProvider) GetConfigInfo() map[string]CacheConfigInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]CacheConfigInfo)
}

func (m *mockCacheConfigProvider) GetAllConfigs(ctx context.Context) map[string]int {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]int)
}

func (m *mockCacheConfigProvider) ReloadConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ==================== 测试夹具 ====================

// newCacheTestDB 构造内存 sqlite + cache service DB 依赖表
// (sys_cache_info / sys_cache_stats / sys_config, 列对齐 internal/models/monitor.go 与 config.go)。
func newCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_cache_info (
			key TEXT PRIMARY KEY,
			value TEXT,
			ttl INTEGER,
			size INTEGER,
			type TEXT,
			location TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_cache_stats (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			cache_type TEXT,
			hit_count INTEGER,
			miss_count INTEGER,
			hit_rate REAL,
			total_memory INTEGER,
			used_memory INTEGER,
			key_count INTEGER,
			expired_count INTEGER,
			collect_time DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			config_name TEXT,
			config_key TEXT UNIQUE,
			config_value TEXT,
			config_type TEXT,
			is_system INTEGER,
			remark TEXT
		)
	`).Error)
	return db
}

// newTestCacheService 通过 NewCacheService 构造真实 cacheServiceImpl。
func newTestCacheService(db *gorm.DB, provider CacheProvider, config CacheConfigProvider) *cacheServiceImpl {
	return NewCacheService(db, provider, config).(*cacheServiceImpl)
}

// ==================== 构造器 ====================

// TestCacheService_NewService_Wiring 验证 NewCacheService 的可选接口类型断言装配:
// full provider → 三个可选字段非 nil;基础 provider → 三个可选字段均为 nil。
func TestCacheService_NewService_Wiring(t *testing.T) {
	db := newCacheTestDB(t)

	full := &mockMonitorFullCacheProvider{}
	svcFull := newTestCacheService(db, full, nil)
	assert.NotNil(t, svcFull.multiLevelCache, "full provider 应装配 multiLevelCache")
	assert.NotNil(t, svcFull.directRedis, "full provider 应装配 directRedis")
	assert.NotNil(t, svcFull.statsProvider, "full provider 应装配 statsProvider")

	basic := &mockMonitorCacheProvider{}
	svcBasic := newTestCacheService(db, basic, nil)
	assert.Nil(t, svcBasic.multiLevelCache, "基础 provider 不装配 multiLevelCache")
	assert.Nil(t, svcBasic.directRedis, "基础 provider 不装配 directRedis")
	assert.Nil(t, svcBasic.statsProvider, "基础 provider 不装配 statsProvider")
}

// TestCacheService_CompileOnly 冒烟测试(编译期契约已由顶部 var _ 断言锁定)。
func TestCacheService_CompileOnly(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	assert.NotNil(t, svc)
}

// ==================== GetCacheList ====================

func TestCacheService_GetCacheList_NilProvider_FromDB_Empty(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), nil, nil)
	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
}

func TestCacheService_GetCacheList_NilProvider_FromDB_FiltersAndSort(t *testing.T) {
	db := newCacheTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&models.CacheInfo{Key: "user:1", Value: "a", Type: "string", Location: "l2", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&models.CacheInfo{Key: "dept:2", Value: "bb", Type: "hash", Location: "l2", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&models.CacheInfo{Key: "user:3", Value: "ccc", Type: "string", Location: "l1", CreatedAt: now, UpdatedAt: now}).Error)

	svc := newTestCacheService(db, nil, nil)

	// Key LIKE + Type = + 白名单排序(key ASC)
	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{
		Key: "user", Type: "string", Current: 1, PageSize: 10, OrderByColumn: "key", IsAsc: true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, "user:1", list[0].Key)
	assert.Equal(t, "user:3", list[1].Key)

	// 非白名单排序列 → 回退 created_at DESC(不注入)
	list2, total2, err := svc.GetCacheList(context.Background(), CacheListParams{
		Current: 1, PageSize: 2, OrderByColumn: "evil; DROP TABLE sys_cache_info",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total2)
	require.Len(t, list2, 2)
}

func TestCacheService_GetCacheList_NilProvider_DBError(t *testing.T) {
	db := newCacheTestDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_cache_info").Error)
	svc := newTestCacheService(db, nil, nil)
	_, _, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 10})
	assert.Error(t, err)
}

func TestCacheService_GetCacheList_SimpleCache_Success(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	provider.On("Keys", mock.Anything, "*").Return([]string{"k1", "k2"}, nil)
	provider.On("Get", mock.Anything, "k1").Return("value1", nil)
	provider.On("TTL", mock.Anything, "k1").Return(90*time.Second, nil)
	provider.On("Get", mock.Anything, "k2").Return("v2", nil)
	provider.On("TTL", mock.Anything, "k2").Return(time.Duration(-1), nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	assert.Equal(t, "k1", list[0].Key)
	assert.Equal(t, "value1", list[0].Value)
	assert.Equal(t, "string", list[0].Type)
	assert.Equal(t, int64(len("value1")), list[0].Size)
	assert.Equal(t, int64(90), list[0].TTL)
	assert.Equal(t, "l2", list[0].Location)

	// TTL <= 0 → -1
	assert.Equal(t, int64(-1), list[1].TTL)

	provider.AssertExpectations(t)
}

// Q2 quirk lock: Keys 出错被静默吞掉 → 空列表 + nil error。
func TestCacheService_GetCacheList_SimpleCache_KeysError_Swallowed(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Keys", mock.Anything, "*").Return(nil, errors.New("redis down"))

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 10})
	assert.NoError(t, err, "Keys 错误应被吞掉(quirk Q2)")
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheList_SimpleCache_GetError_SkipsKey(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	provider.On("Keys", mock.Anything, "*").Return([]string{"bad", "good"}, nil)
	provider.On("Get", mock.Anything, "bad").Return("", errors.New("get fail"))
	provider.On("Get", mock.Anything, "good").Return("ok", nil)
	provider.On("TTL", mock.Anything, "good").Return(time.Duration(0), nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "good", list[0].Key)
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheList_SimpleCache_KeyPatternForwarded(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Keys", mock.Anything, "*user*").Return([]string{}, nil)

	_, _, err := svc.GetCacheList(context.Background(), CacheListParams{Key: "user", Current: 1, PageSize: 10})
	require.NoError(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_L1(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	full.On("KeysByLevel", mock.Anything, "*", "l1").Return([]string{"mem:1"}, nil)
	full.On("Get", mock.Anything, "mem:1").Return("mv", nil)
	full.On("TTL", mock.Anything, "mem:1").Return(30*time.Second, nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "l1", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "l1", list[0].Location)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_L2_DirectRedis(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	full.On("DirectRedisKeys", mock.Anything, "*").Return([]string{"r:1"}, nil)
	full.On("DirectRedisGet", mock.Anything, "r:1").Return("rv", nil)
	full.On("DirectRedisTTL", mock.Anything, "r:1").Return(60*time.Second, nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "l2", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "r:1", list[0].Key)
	assert.Equal(t, "rv", list[0].Value)
	assert.Equal(t, int64(60), list[0].TTL)
	assert.Equal(t, "l2", list[0].Location)
	// 直连 Redis 命中时不再走通用 KeysByLevel
	full.AssertNotCalled(t, "KeysByLevel", mock.Anything, mock.Anything, mock.Anything)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_L2_DirectRedisFails_Fallback(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	// DirectRedisKeys 出错 → 回退 KeysByLevel(l2)
	full.On("DirectRedisKeys", mock.Anything, "*").Return(nil, errors.New("direct fail"))
	full.On("KeysByLevel", mock.Anything, "*", "l2").Return([]string{"k:l2"}, nil)
	full.On("Get", mock.Anything, "k:l2").Return("v", nil)
	full.On("TTL", mock.Anything, "k:l2").Return(time.Duration(0), nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "l2", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_L2_DirectRedisEmpty_Fallback(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	// DirectRedisKeys 成功但 keys 为空 → 回退
	full.On("DirectRedisKeys", mock.Anything, "*").Return([]string{}, nil)
	full.On("KeysByLevel", mock.Anything, "*", "l2").Return(nil, nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "l2", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_All_BothLocation(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	full.On("KeysByLevel", mock.Anything, "*", "all").Return([]string{"shared:1"}, nil)
	full.On("Get", mock.Anything, "shared:1").Return("sv", nil)
	full.On("TTL", mock.Anything, "shared:1").Return(time.Duration(0), nil)
	// all 级别逐 key 探测 l1 命中 → location = both
	full.On("KeysByLevel", mock.Anything, "shared:1", "l1").Return([]string{"shared:1"}, nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "all", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "both", list[0].Location)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_All_L1Miss_L2Location(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	full.On("KeysByLevel", mock.Anything, "*", "all").Return([]string{"only:l2"}, nil)
	full.On("Get", mock.Anything, "only:l2").Return("v", nil)
	full.On("TTL", mock.Anything, "only:l2").Return(time.Duration(0), nil)
	full.On("KeysByLevel", mock.Anything, "only:l2", "l1").Return(nil, nil)

	list, _, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "all", Current: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "l2", list[0].Location)
	full.AssertExpectations(t)
}

// Q2 quirk lock: KeysByLevel 出错 → 空列表 + nil error。
func TestCacheService_GetCacheList_MultiLevel_KeysByLevelError_EmptyResult(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("KeysByLevel", mock.Anything, "*", "l1").Return(nil, errors.New("lvl fail"))

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "l1", Current: 1, PageSize: 10})
	assert.NoError(t, err, "KeysByLevel 错误应被吞掉(quirk Q2)")
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_MultiLevel_SystemKeysAndEmptyValuesSkipped(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)

	// redis: 前缀 + 数字开头 = 系统键(不触发 Get);empty:val 取回空值被跳过
	full.On("KeysByLevel", mock.Anything, "*", "all").Return([]string{"redis:sys", "9session", "empty:val", "user:ok"}, nil)
	full.On("Get", mock.Anything, "empty:val").Return("", nil)
	// 注意: TTL 在 value=="" 跳过判定之前就被调用(先取 TTL 再判空)
	full.On("TTL", mock.Anything, "empty:val").Return(time.Duration(0), nil)
	full.On("Get", mock.Anything, "user:ok").Return("v", nil)
	full.On("TTL", mock.Anything, "user:ok").Return(time.Duration(0), nil)
	full.On("KeysByLevel", mock.Anything, "user:ok", "l1").Return(nil, nil)

	list, total, err := svc.GetCacheList(context.Background(), CacheListParams{Level: "all", Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "user:ok", list[0].Key)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheList_Pagination(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	provider.On("Keys", mock.Anything, "*").Return([]string{"a", "b", "c"}, nil)
	provider.On("Get", mock.Anything, mock.Anything).Return("v", nil)
	provider.On("TTL", mock.Anything, mock.Anything).Return(time.Duration(0), nil)

	// 第 1 页:2 条
	p1, total, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, p1, 2)

	// 第 2 页:1 条
	p2, _, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, p2, 1)

	// 越界页:start >= len → nil 列表,total 不变
	p3, total3, err := svc.GetCacheList(context.Background(), CacheListParams{Current: 5, PageSize: 2})
	require.NoError(t, err)
	assert.Nil(t, p3)
	assert.Equal(t, int64(3), total3)
	provider.AssertExpectations(t)
}

// ==================== GetCacheInfo ====================

func TestCacheService_GetCacheInfo_EmptyKey(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	_, err := svc.GetCacheInfo(context.Background(), "")
	assert.ErrorIs(t, err, ErrCacheKeyRequired)
}

func TestCacheService_GetCacheInfo_ProviderError(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "k").Return("", errors.New("get fail"))

	_, err := svc.GetCacheInfo(context.Background(), "k")
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheInfo_ProviderHit_TTLPositive(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "k").Return("val", nil)
	provider.On("TTL", mock.Anything, "k").Return(45*time.Second, nil)

	info, err := svc.GetCacheInfo(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "k", info.Key)
	assert.Equal(t, "val", info.Value)
	assert.Equal(t, int64(45), info.TTL)
	assert.Equal(t, int64(len("val")), info.Size)
	assert.Equal(t, "l2", info.Location)
	provider.AssertExpectations(t)
}

// Q1 quirk lock: 前缀永不被剥离 —— "xingran:k" 原样传给 provider。
func TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "xingran:k").Return("v", nil)
	provider.On("TTL", mock.Anything, "xingran:k").Return(time.Duration(0), nil)

	info, err := svc.GetCacheInfo(context.Background(), "xingran:k")
	require.NoError(t, err)
	assert.Equal(t, "xingran:k", info.Key, "quirk Q1: key 应原样保留(前缀剥离永不生效)")
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheInfo_EmptyValue_FallsToDB_Found(t *testing.T) {
	db := newCacheTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&models.CacheInfo{Key: "dbk", Value: "dbval", Type: "string", CreatedAt: now, UpdatedAt: now}).Error)

	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(db, provider, nil)
	provider.On("Get", mock.Anything, "dbk").Return("", nil)

	info, err := svc.GetCacheInfo(context.Background(), "dbk")
	require.NoError(t, err)
	assert.Equal(t, "dbk", info.Key)
	assert.Equal(t, "dbval", info.Value)
	provider.AssertExpectations(t)
}

func TestCacheService_GetCacheInfo_NilProvider_DBNotFound(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), nil, nil)
	_, err := svc.GetCacheInfo(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheService_GetCacheInfo_EmptyValue_FallsToDB_NotFound(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "missing").Return("", nil)

	_, err := svc.GetCacheInfo(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrCacheNotFound)
	provider.AssertExpectations(t)
}

// ==================== OperateCache ====================

func TestCacheService_OperateCache_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), nil, nil)
	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "get"})
	assert.ErrorIs(t, err, ErrCacheServiceUnavailable)
}

func TestCacheService_OperateCache_Get_Success(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "k").Return("v", nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "get"})
	require.NoError(t, err)
	assert.Equal(t, "v", result)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Get_Error(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "k").Return("", errors.New("get fail"))

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "get"})
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Set_DefaultTTL(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	// TTL 未指定 → 默认 1 小时
	provider.On("Set", mock.Anything, "k", "v", time.Hour).Return(nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Value: "v", Operation: "set"})
	require.NoError(t, err)
	assert.Equal(t, "设置成功", result)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Set_ExplicitTTL(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Set", mock.Anything, "k", "v", 120*time.Second).Return(nil)

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Value: "v", TTL: 120, Operation: "set"})
	require.NoError(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Set_ValueRequired(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Value: "", Operation: "set"})
	assert.ErrorIs(t, err, ErrCacheValueRequired)
	provider.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCacheService_OperateCache_Set_Error(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Set", mock.Anything, "k", "v", time.Hour).Return(errors.New("set fail"))

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Value: "v", Operation: "set"})
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Del_Success(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Delete", mock.Anything, "k").Return(nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "del"})
	require.NoError(t, err)
	assert.Equal(t, "删除成功", result)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Del_Error(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Delete", mock.Anything, "k").Return(errors.New("del fail"))

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "del"})
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Exists_True(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Exists", mock.Anything, "k").Return(true, nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "exists"})
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["exists"])
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Exists_Error(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Exists", mock.Anything, "k").Return(false, errors.New("exists fail"))

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "exists"})
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Expire_TTLRequired(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", TTL: 0, Operation: "expire"})
	assert.ErrorIs(t, err, ErrCacheTTLRequired)
	provider.AssertNotCalled(t, "Expire", mock.Anything, mock.Anything, mock.Anything)
}

func TestCacheService_OperateCache_Expire_Success(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Expire", mock.Anything, "k", 300*time.Second).Return(nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", TTL: 300, Operation: "expire"})
	require.NoError(t, err)
	assert.Equal(t, "设置成功", result)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_Expire_Error(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Expire", mock.Anything, "k", 300*time.Second).Return(errors.New("expire fail"))

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", TTL: 300, Operation: "expire"})
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_OperateCache_TTL_ReturnsMap(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	result, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "ttl"})
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, -1, m["ttl"])
	provider.AssertNotCalled(t, "TTL", mock.Anything, mock.Anything)
}

func TestCacheService_OperateCache_UnsupportedOperation(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	_, err := svc.OperateCache(context.Background(), CacheOperateParams{Key: "k", Operation: "flushall"})
	assert.ErrorIs(t, err, ErrOperationUnsupported)
}

// ==================== BatchOperateCache ====================

func TestCacheService_BatchOperateCache_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), nil, nil)
	_, err := svc.BatchOperateCache(context.Background(), CacheBatchOperateParams{Keys: []string{"k"}, Operation: "get"})
	assert.ErrorIs(t, err, ErrCacheServiceUnavailable)
}

func TestCacheService_BatchOperateCache_EmptyKeys(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	_, err := svc.BatchOperateCache(context.Background(), CacheBatchOperateParams{Keys: []string{}, Operation: "get"})
	assert.ErrorIs(t, err, ErrCacheKeysRequired)
}

func TestCacheService_BatchOperateCache_UnsupportedOperation(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)

	_, err := svc.BatchOperateCache(context.Background(), CacheBatchOperateParams{Keys: []string{"k"}, Operation: "set"})
	assert.ErrorIs(t, err, ErrOperationUnsupported)
}

func TestCacheService_BatchOperateCache_Get_MixedResults(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Get", mock.Anything, "ok").Return("v", nil)
	provider.On("Get", mock.Anything, "bad").Return("", errors.New("get fail"))

	results, err := svc.BatchOperateCache(context.Background(), CacheBatchOperateParams{Keys: []string{"ok", "bad"}, Operation: "get"})
	require.NoError(t, err)
	assert.Equal(t, "v", results["ok"])
	assert.Equal(t, "获取失败: get fail", results["bad"])
	provider.AssertExpectations(t)
}

func TestCacheService_BatchOperateCache_Del_MixedResults(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("Delete", mock.Anything, "ok").Return(nil)
	provider.On("Delete", mock.Anything, "bad").Return(errors.New("del fail"))

	results, err := svc.BatchOperateCache(context.Background(), CacheBatchOperateParams{Keys: []string{"ok", "bad"}, Operation: "del"})
	require.NoError(t, err)

	deleted, dok := results["deleted"].([]string)
	require.True(t, dok)
	assert.Equal(t, []string{"ok"}, deleted)

	failed, fok := results["failed"].(map[string]string)
	require.True(t, fok)
	assert.Equal(t, "del fail", failed["bad"])
	provider.AssertExpectations(t)
}

// ==================== ClearCache ====================

func TestCacheService_ClearCache_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), nil, nil)
	err := svc.ClearCache(context.Background())
	assert.ErrorIs(t, err, ErrCacheServiceUnavailable)
}

func TestCacheService_ClearCache_FlushDBError(t *testing.T) {
	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), provider, nil)
	provider.On("FlushDB", mock.Anything).Return(errors.New("flush fail"))

	err := svc.ClearCache(context.Background())
	assert.Error(t, err)
	provider.AssertExpectations(t)
}

func TestCacheService_ClearCache_Success(t *testing.T) {
	db := newCacheTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&models.CacheInfo{Key: "k", Value: "v", CreatedAt: now, UpdatedAt: now}).Error)

	provider := &mockMonitorCacheProvider{}
	svc := newTestCacheService(db, provider, nil)
	provider.On("FlushDB", mock.Anything).Return(nil)

	require.NoError(t, svc.ClearCache(context.Background()))

	var count int64
	require.NoError(t, db.Model(&models.CacheInfo{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "ClearCache 应清空 sys_cache_info")
	provider.AssertExpectations(t)
}

// ==================== GetCacheStats ====================

func TestCacheService_GetCacheStats_Realtime_NoStatsProvider(t *testing.T) {
	// 基础 provider 不实现 StatsProvider → statsProvider == nil
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	_, _, err := svc.GetCacheStats(context.Background(), CacheStatsParams{})
	assert.ErrorIs(t, err, ErrCacheStatsUnsupported)
}

func TestCacheService_GetCacheStats_RealtimeByDefault_TimesNil(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	// IsRealtime=false 但 StartTime/EndTime 均 nil → 仍走实时
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(nil, nil)

	_, _, err := svc.GetCacheStats(context.Background(), CacheStatsParams{IsRealtime: false})
	assert.NoError(t, err)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheStats_Realtime_GetStatsError(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(nil, errors.New("stats fail"))

	_, _, err := svc.GetCacheStats(context.Background(), CacheStatsParams{IsRealtime: true})
	assert.Error(t, err)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheStats_Realtime_Success(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(map[string]interface{}{
		"l1": map[string]interface{}{
			"keyspace_hits":   int64(10),
			"keyspace_misses": int64(2),
			"hit_rate":        0.83,
			"used_memory":     int64(512),
			"key_count":       int64(7),
		},
		"l2": map[string]interface{}{
			"keyspace_hits":   int64(100),
			"keyspace_misses": int64(50),
			"hit_rate":        0.66,
			"used_memory":     int64(2048),
			"key_count":       int64(64),
		},
		"other": "wrong-shape", // 非 map 的值应被跳过
	}, nil)

	list, total, err := svc.GetCacheStats(context.Background(), CacheStatsParams{IsRealtime: true})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	assert.Equal(t, "L1(内存)", list[0].CacheType)
	assert.Equal(t, int64(10), list[0].HitCount)
	assert.Equal(t, int64(2), list[0].MissCount)
	assert.Equal(t, 0.83, list[0].HitRate)
	assert.Equal(t, int64(512), list[0].UsedMemory)
	assert.Equal(t, int64(1024), list[0].TotalMemory, "convertToCacheStats: TotalMemory = usedMemory*2")
	assert.Equal(t, int64(7), list[0].KeyCount)

	assert.Equal(t, "L2(Redis)", list[1].CacheType)
	assert.Equal(t, int64(100), list[1].HitCount)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheStats_History_WithFiltersAndPagination(t *testing.T) {
	db := newCacheTestDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&models.CacheStats{
			CacheType:   "L1(内存)",
			HitCount:    int64(i),
			CollectTime: base.Add(time.Duration(i) * time.Hour),
		}).Error)
	}
	require.NoError(t, db.Create(&models.CacheStats{CacheType: "L2(Redis)", HitCount: 1, CollectTime: base}).Error)

	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(db, full, nil)

	start := "2026-08-20 09:00:00"
	end := "2026-08-20 12:00:00"

	// CacheType 过滤
	list, total, err := svc.GetCacheStats(context.Background(), CacheStatsParams{
		CacheType: "L2(Redis)", StartTime: &start, EndTime: &end, Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "L2(Redis)", list[0].CacheType)

	// 时间范围 + 分页(collect_time DESC)
	list2, total2, err := svc.GetCacheStats(context.Background(), CacheStatsParams{
		StartTime: &start, EndTime: &end, Current: 1, PageSize: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total2, "L1 3 条在时间窗内(L2 被类型过滤排除)")
	require.Len(t, list2, 2)
	assert.GreaterOrEqual(t, list2[0].CollectTime.Unix(), list2[1].CollectTime.Unix(), "默认 collect_time DESC")
}

func TestCacheService_GetCacheStats_History_DBError(t *testing.T) {
	db := newCacheTestDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_cache_stats").Error)
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(db, full, nil)

	start := "2026-08-20 00:00:00"
	_, _, err := svc.GetCacheStats(context.Background(), CacheStatsParams{StartTime: &start, Current: 1, PageSize: 10})
	assert.Error(t, err)
}

// ==================== GetCacheMonitor ====================

func TestCacheService_GetCacheMonitor_NoStatsProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	_, err := svc.GetCacheMonitor(context.Background())
	assert.ErrorIs(t, err, ErrCacheStatsUnsupported)
}

func TestCacheService_GetCacheMonitor_GetStatsError(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(nil, errors.New("stats fail"))

	_, err := svc.GetCacheMonitor(context.Background())
	assert.Error(t, err)
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheMonitor_L1Only(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(map[string]interface{}{
		"l1": map[string]interface{}{
			"keyspace_hits":   int64(4),
			"keyspace_misses": int64(1),
			"used_memory":     int64(256),
			"key_count":       int64(3),
			"hit_rate":        0.8,
		},
	}, nil)

	monitor, err := svc.GetCacheMonitor(context.Background())
	require.NoError(t, err)
	assert.Contains(t, monitor, "l1")
	assert.NotContains(t, monitor, "l2")

	l1 := monitor["l1"].(map[string]interface{})
	status := l1["status"].(map[string]interface{})
	assert.Equal(t, true, status["connected"])
	assert.Equal(t, "memory", status["type"])

	stats := l1["stats"].(map[string]interface{})
	assert.Equal(t, int64(4), stats["hitCount"])
	assert.Equal(t, int64(512), stats["totalMemory"], "无 total_system_memory/maxmemory → usedMemory*2")
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheMonitor_L2WithVersionAndUptime(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(map[string]interface{}{
		"l2": map[string]interface{}{
			"keyspace_hits":     int64(40),
			"keyspace_misses":   int64(10),
			"hit_rate":          0.8,
			"used_memory":       int64(1024),
			"key_count":         int64(20),
			"total_system_memory": int64(4096),
			"redis_version":     "7.4.0",
			"uptime_in_seconds": int64(120),
		},
	}, nil)

	monitor, err := svc.GetCacheMonitor(context.Background())
	require.NoError(t, err)
	assert.Contains(t, monitor, "l2")
	assert.NotContains(t, monitor, "l1")

	l2 := monitor["l2"].(map[string]interface{})
	status := l2["status"].(map[string]interface{})
	assert.Equal(t, "redis", status["type"])
	assert.Equal(t, "7.4.0", status["version"])
	assert.Equal(t, "120s", status["uptime"])

	stats := l2["stats"].(map[string]interface{})
	assert.Equal(t, int64(4096), stats["totalMemory"], "total_system_memory 优先")
	full.AssertExpectations(t)
}

func TestCacheService_GetCacheMonitor_EmptyStats(t *testing.T) {
	full := &mockMonitorFullCacheProvider{}
	svc := newTestCacheService(newCacheTestDB(t), full, nil)
	full.On("GetStats", mock.Anything).Return(map[string]interface{}{}, nil)

	monitor, err := svc.GetCacheMonitor(context.Background())
	require.NoError(t, err)
	assert.Empty(t, monitor)
	full.AssertExpectations(t)
}

// ==================== ExportCache ====================

func TestCacheService_ExportCache_NoFilters(t *testing.T) {
	db := newCacheTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&models.CacheInfo{Key: "a", Value: "1", Type: "string", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&models.CacheInfo{Key: "b", Value: "2", Type: "hash", CreatedAt: now, UpdatedAt: now}).Error)

	svc := newTestCacheService(db, nil, nil)
	caches, err := svc.ExportCache(context.Background(), CacheExportParams{})
	require.NoError(t, err)
	assert.Len(t, caches, 2)
}

func TestCacheService_ExportCache_WithFilters(t *testing.T) {
	db := newCacheTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&models.CacheInfo{Key: "user:1", Value: "1", Type: "string", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&models.CacheInfo{Key: "dept:2", Value: "2", Type: "hash", CreatedAt: now, UpdatedAt: now}).Error)

	svc := newTestCacheService(db, nil, nil)
	caches, err := svc.ExportCache(context.Background(), CacheExportParams{Key: "user", Type: "string"})
	require.NoError(t, err)
	require.Len(t, caches, 1)
	assert.Equal(t, "user:1", caches[0].Key)
}

func TestCacheService_ExportCache_DBError(t *testing.T) {
	db := newCacheTestDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_cache_info").Error)
	svc := newTestCacheService(db, nil, nil)

	_, err := svc.ExportCache(context.Background(), CacheExportParams{})
	assert.Error(t, err)
}

// ==================== GetCacheConfigs ====================

func TestCacheService_GetCacheConfigs_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	_, _, err := svc.GetCacheConfigs(context.Background())
	assert.ErrorIs(t, err, ErrCacheConfigUnavailable)
}

func TestCacheService_GetCacheConfigs_Success(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)

	infoMap := map[string]CacheConfigInfo{
		"cache.user.ttl": {Key: "cache.user.ttl", Name: "用户缓存", Min: 1, Max: 120, Default: 30},
	}
	values := map[string]int{"cache.user.ttl": 45}
	config.On("GetConfigInfo").Return(infoMap)
	config.On("GetAllConfigs", mock.Anything).Return(values)

	gotInfo, gotValues, err := svc.GetCacheConfigs(context.Background())
	require.NoError(t, err)
	assert.Equal(t, infoMap, gotInfo)
	assert.Equal(t, values, gotValues)
	config.AssertExpectations(t)
}

// ==================== UpdateCacheConfig ====================

func TestCacheService_UpdateCacheConfig_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	err := svc.UpdateCacheConfig(context.Background(), "k", 1)
	assert.ErrorIs(t, err, ErrCacheConfigUnavailable)
}

func TestCacheService_UpdateCacheConfig_InvalidKey(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)
	config.On("GetConfigInfo").Return(map[string]CacheConfigInfo{})

	err := svc.UpdateCacheConfig(context.Background(), "nope", 5)
	assert.ErrorIs(t, err, ErrInvalidConfigKey)
	config.AssertExpectations(t)
}

func TestCacheService_UpdateCacheConfig_OutOfRange(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)
	config.On("GetConfigInfo").Return(map[string]CacheConfigInfo{
		"cache.user.ttl": {Key: "cache.user.ttl", Min: 1, Max: 120},
	})

	err := svc.UpdateCacheConfig(context.Background(), "cache.user.ttl", 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 到 120")
	config.AssertExpectations(t)
}

func TestCacheService_UpdateCacheConfig_CreatesWhenMissing(t *testing.T) {
	db := newCacheTestDB(t)
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(db, &mockMonitorCacheProvider{}, config)

	config.On("GetConfigInfo").Return(map[string]CacheConfigInfo{
		"cache.user.ttl": {Key: "cache.user.ttl", Name: "用户缓存", Description: "desc", Min: 1, Max: 120},
	})
	config.On("ReloadConfig", mock.Anything).Return(nil)

	require.NoError(t, svc.UpdateCacheConfig(context.Background(), "cache.user.ttl", 30))

	var cfg models.Config
	require.NoError(t, db.Where("config_key = ?", "cache.user.ttl").First(&cfg).Error)
	assert.Equal(t, "30", cfg.ConfigValue)
	assert.Equal(t, "用户缓存", cfg.ConfigName)
	assert.Equal(t, models.ConfigTypeYes, cfg.ConfigType)
	config.AssertExpectations(t)
}

func TestCacheService_UpdateCacheConfig_UpdatesExisting(t *testing.T) {
	db := newCacheTestDB(t)
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(db, &mockMonitorCacheProvider{}, config)

	require.NoError(t, db.Create(&models.Config{
		BaseModel:   models.BaseModel{ID: "cfg-1"},
		ConfigKey:   "cache.user.ttl",
		ConfigName:  "用户缓存",
		ConfigValue: "10",
	}).Error)

	config.On("GetConfigInfo").Return(map[string]CacheConfigInfo{
		"cache.user.ttl": {Key: "cache.user.ttl", Name: "用户缓存", Min: 1, Max: 120},
	})
	config.On("ReloadConfig", mock.Anything).Return(nil)

	require.NoError(t, svc.UpdateCacheConfig(context.Background(), "cache.user.ttl", 60))

	var cfg models.Config
	require.NoError(t, db.Where("config_key = ?", "cache.user.ttl").First(&cfg).Error)
	assert.Equal(t, "60", cfg.ConfigValue)
	config.AssertExpectations(t)
}

func TestCacheService_UpdateCacheConfig_ReloadErrorPropagates(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)

	config.On("GetConfigInfo").Return(map[string]CacheConfigInfo{
		"cache.user.ttl": {Key: "cache.user.ttl", Min: 1, Max: 120},
	})
	config.On("ReloadConfig", mock.Anything).Return(errors.New("reload fail"))

	err := svc.UpdateCacheConfig(context.Background(), "cache.user.ttl", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reload fail")
	config.AssertExpectations(t)
}

// ==================== ReloadCacheConfigs ====================

func TestCacheService_ReloadCacheConfigs_NilProvider(t *testing.T) {
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, nil)
	err := svc.ReloadCacheConfigs(context.Background())
	assert.ErrorIs(t, err, ErrCacheConfigUnavailable)
}

func TestCacheService_ReloadCacheConfigs_Success(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)
	config.On("ReloadConfig", mock.Anything).Return(nil)

	assert.NoError(t, svc.ReloadCacheConfigs(context.Background()))
	config.AssertExpectations(t)
}

func TestCacheService_ReloadCacheConfigs_Error(t *testing.T) {
	config := &mockCacheConfigProvider{}
	svc := newTestCacheService(newCacheTestDB(t), &mockMonitorCacheProvider{}, config)
	config.On("ReloadConfig", mock.Anything).Return(errors.New("reload fail"))

	assert.Error(t, svc.ReloadCacheConfigs(context.Background()))
	config.AssertExpectations(t)
}

// ==================== 辅助函数 ====================

// TestNormalizeCacheKeyForService — quirk Q1 lock:
// `key[:6] == "xingran:"`(6 字节切片 vs 8 字节字面量)恒为 false,函数为恒等函数。
func TestNormalizeCacheKeyForService(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"xingran:user:1", "xingran:user:1"}, // 前缀不被剥离(quirk)
		{"user:1", "user:1"},
		{"xingran", "xingran"}, // len<=6 短键原样返回
		{"xingr", "xingr"},
		{"", ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, normalizeCacheKeyForService(tc.in), "input=%q", tc.in)
	}
}

// TestIsSystemKeyForService 表驱动覆盖全部系统前缀 + 数字开头 + 正常键。
func TestIsSystemKeyForService(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"__:lock", true},
		{"redis:sys", true},
		{"monitor:x", true},
		{"perf:x", true},
		{"stats:x", true},
		{"cluster:x", true},
		{"node:x", true},
		{"replication:x", true},
		{"sentinel:x", true},
		{"0abc", true},
		{"9", true},
		{"user:1", false},
		{"session:abc", false},
		{"abc", false},
		{"", false},
		{"_:short", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isSystemKeyForService(tc.key), "key=%q", tc.key)
	}
}

// TestFormatCacheStatsForService 覆盖 totalMemory 三分支:
// total_system_memory 优先 → maxmemory 次之 → 兜底 usedMemory*2。
func TestFormatCacheStatsForService(t *testing.T) {
	// 分支 1: total_system_memory
	got := formatCacheStatsForService(map[string]interface{}{
		"keyspace_hits": int64(5), "keyspace_misses": int64(3), "hit_rate": 0.625,
		"used_memory": int64(100), "key_count": int64(9),
		"total_system_memory": int64(1000), "maxmemory": int64(2000),
	})
	assert.Equal(t, int64(5), got["hitCount"])
	assert.Equal(t, int64(3), got["missCount"])
	assert.Equal(t, 0.625, got["hitRate"])
	assert.Equal(t, int64(1000), got["totalMemory"])
	assert.Equal(t, int64(100), got["usedMemory"])
	assert.Equal(t, int64(9), got["keyCount"])

	// 分支 2: maxmemory(total_system_memory 缺失或 <=0)
	got2 := formatCacheStatsForService(map[string]interface{}{
		"used_memory": int64(100), "maxmemory": int64(2000),
	})
	assert.Equal(t, int64(2000), got2["totalMemory"])

	// 分支 3: 兜底 usedMemory*2(两者都缺失/<=0)
	got3 := formatCacheStatsForService(map[string]interface{}{
		"used_memory": int64(100), "total_system_memory": int64(-1), "maxmemory": int64(0),
	})
	assert.Equal(t, int64(200), got3["totalMemory"])

	// 空表 → 全零
	got4 := formatCacheStatsForService(map[string]interface{}{})
	assert.Equal(t, int64(0), got4["hitCount"])
	assert.Equal(t, int64(0), got4["totalMemory"])
}

// TestConvertToCacheStats 覆盖全字段映射 + TotalMemory = usedMemory*2。
func TestConvertToCacheStats(t *testing.T) {
	got := convertToCacheStats(map[string]interface{}{
		"keyspace_hits":   int64(7),
		"keyspace_misses": int64(3),
		"hit_rate":        0.7,
		"used_memory":     int64(300),
		"key_count":       int64(12),
	}, "L1(内存)")

	assert.Equal(t, "L1(内存)", got.CacheType)
	assert.Equal(t, int64(7), got.HitCount)
	assert.Equal(t, int64(3), got.MissCount)
	assert.Equal(t, 0.7, got.HitRate)
	assert.Equal(t, int64(300), got.UsedMemory)
	assert.Equal(t, int64(600), got.TotalMemory)
	assert.Equal(t, int64(12), got.KeyCount)
	assert.False(t, got.CollectTime.IsZero())
}
