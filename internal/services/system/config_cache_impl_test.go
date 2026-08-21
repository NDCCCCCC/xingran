package system

// =====================================================================
// config_cache_impl_test.go — covers config_cache_impl.go
// Compile-time interface assertion + cache invalidation tests
// Per Plan 72-10 Task 3
// =====================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// Compile-time interface assertion
var _ ConfigService = (*configCacheService)(nil)

// mockConfigCacheProvider minimal mock for cache invalidation tests.
type mockConfigCacheProvider struct {
	mock.Mock
}

func (m *mockConfigCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	m.Called(ctx, key, dest, expiration, query)
	result, err := query()
	if err != nil {
		return err
	}
	_ = result
	return nil
}

func (m *mockConfigCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockConfigCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *mockConfigCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	_ = m.Called(ctx, keys)
	return map[string]string{}, nil
}

func (m *mockConfigCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	_ = m.Called(ctx, keys)
	return nil
}

func (m *mockConfigCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	_ = m.Called(ctx, key)
	return false, nil
}

func (m *mockConfigCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	_ = m.Called(ctx, key, expiration)
	return nil
}

func (m *mockConfigCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	_ = m.Called(ctx, key)
	return 0, nil
}

func (m *mockConfigCacheProvider) GetStats(ctx context.Context) (*CacheStats, error) {
	_ = m.Called(ctx)
	return &CacheStats{}, nil
}

func setupConfigCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			config_name TEXT NOT NULL,
			config_key TEXT NOT NULL UNIQUE,
			config_value TEXT,
			config_type TEXT DEFAULT 'Y',
			is_system INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

func newConfigCacheService(t *testing.T, db *gorm.DB, cache CacheProvider) ConfigService {
	t.Helper()
	// Pass nil to avoid CacheConfigService inserting default rows into sys_config
	// which would pollute Statistics/List/Count assertions.
	return NewConfigServiceWithCache(
		db,
		cache,
		nil,
	)
}

// TC1: queryAllConfigs - directly
func TestConfigCache_QueryAllConfigs(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cs := &configCacheService{configService: &configService{db: db}}
	configs, err := cs.queryAllConfigs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, configs)
}

// TC2: GetAllConfigs - empty
func TestConfigCache_GetAllConfigs_Empty(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	cs := svc.(*configCacheService)
	configs, err := cs.GetAllConfigs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, configs)
}

// TC3: InvalidateConfigCache - no panic
func TestConfigCache_InvalidateConfigCache(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	cs := svc.(*configCacheService)
	require.NoError(t, cs.InvalidateConfigCache(context.Background(), "sys.k1"))
}

// TC4: InvalidateAllConfigCache - no panic
func TestConfigCache_InvalidateAllConfigCache(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	cs := svc.(*configCacheService)
	require.NoError(t, cs.InvalidateAllConfigCache(context.Background()))
}

// TC5: Create - delegates + invalidates cache
func TestConfigCache_Create(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	req := &requests.ConfigCreateRequest{
		ConfigName:  "new",
		ConfigKey:   "sys.new",
		ConfigValue: "v",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("config_key = ?", "sys.new").First(&stored).Error)
	assert.Equal(t, "v", stored.ConfigValue)
}

// TC6: Update - delegates + invalidates cache
func TestConfigCache_Update(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, 'k1', 'sys.k1', 'v1', 'Y', 0, datetime('now'), datetime('now'))`, id).Error)

	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "k1",
		ConfigKey:   "sys.k1",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "v2", stored.ConfigValue)
}

// TC7: Delete - delegates + invalidates cache
func TestConfigCache_Delete(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, 'k1', 'sys.k1', 'v1', 'Y', 0, datetime('now'), datetime('now'))`, id).Error)

	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, svc.Delete(context.Background(), id))

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_config WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt)
}

// TC8: BatchDelete - delegates + invalidates cache
func TestConfigCache_BatchDelete(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	id1 := uuid.NewString()
	id2 := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, 'k1', 'sys.k1', 'v1', 'Y', 0, datetime('now'), datetime('now'))`, id1).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, 'k2', 'sys.k2', 'v2', 'Y', 0, datetime('now'), datetime('now'))`, id2).Error)

	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_config WHERE id = ?", id1).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt)
}

// TC9: RefreshCache - invalidates all
func TestConfigCache_RefreshCache(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, svc.RefreshCache(context.Background()))
}

// TC10: Create - error path returns error
func TestConfigCache_Create_DuplicateKeyError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, 'dup', 'sys.dup', 'v1', 'Y', 0, datetime('now'), datetime('now'))`, uuid.NewString()).Error)

	cache := new(mockConfigCacheProvider)
	svc := newConfigCacheService(t, db, cache)

	req := &requests.ConfigCreateRequest{
		ConfigName:  "dup",
		ConfigKey:   "sys.dup",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC11: Update - error path returns error
func TestConfigCache_Update_NotFound(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	svc := newConfigCacheService(t, db, cache)

	req := &requests.ConfigUpdateRequest{
		ID:          uuid.NewString(),
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "y",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC12: Delete - lookup error before delete
func TestConfigCache_Delete_LookupError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := new(mockConfigCacheProvider)
	svc := newConfigCacheService(t, db, cache)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC13: GetByID - cache miss returns error
func TestConfigCache_GetByID_NotFound(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	cfg, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TC14: GetByKey - not found
func TestConfigCache_GetByKey_NotFound(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	cfg, err := svc.GetByKey(context.Background(), "sys.missing")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TC15: List - delegates to base service
func TestConfigCache_List(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	result, err := svc.List(context.Background(), requests.DefaultConfigListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC16: Statistics - delegates
func TestConfigCache_Statistics(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC17: Interface assertion compile-time
func TestConfigCache_InterfaceAssertion(t *testing.T) {
	var _ ConfigService = (*configCacheService)(nil)
	var _ CacheProvider = &NoOpCacheProvider{}
}

// TC18: Create service error path
func TestConfigCache_Create_ServiceError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	req := &requests.ConfigCreateRequest{
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "v",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC19: Update service error path
func TestConfigCache_Update_ServiceError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	req := &requests.ConfigUpdateRequest{
		ID:          uuid.NewString(),
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "y",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC20: Delete service error
func TestConfigCache_Delete_ServiceError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC21: BatchDelete service error
func TestConfigCache_BatchDelete_ServiceError(t *testing.T) {
	db := setupConfigCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newConfigCacheService(t, db, cache)

	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	err := svc.BatchDelete(context.Background(), []string{uuid.NewString()})
	assert.Error(t, err)
}
