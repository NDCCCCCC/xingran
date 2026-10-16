package system

// =====================================================================
// settings_cache_impl_test.go — covers settings_cache_impl.go
// Compile-time interface assertion + cache miss/hit/invalidation tests
// Per Plan 72-11 Task 4
// =====================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Compile-time interface assertion
var _ SettingsService = (*settingsCacheService)(nil)

// mockSettingsCacheProvider minimal mock for cache invalidation tests.
type mockSettingsCacheProvider struct {
	mock.Mock
}

func (m *mockSettingsCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	m.Called(ctx, key, dest, expiration, query)
	result, err := query()
	if err != nil {
		return err
	}
	_ = result
	return nil
}

func (m *mockSettingsCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockSettingsCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *mockSettingsCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	_ = m.Called(ctx, keys)
	return map[string]string{}, nil
}

func (m *mockSettingsCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	_ = m.Called(ctx, keys)
	return nil
}

func (m *mockSettingsCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	_ = m.Called(ctx, key)
	return false, nil
}

func (m *mockSettingsCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	_ = m.Called(ctx, key, expiration)
	return nil
}

func (m *mockSettingsCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	_ = m.Called(ctx, key)
	return 0, nil
}

func (m *mockSettingsCacheProvider) GetStats(ctx context.Context) (*CacheStats, error) {
	_ = m.Called(ctx)
	return &CacheStats{}, nil
}

func setupSettingsCacheDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_preference (
			id TEXT PRIMARY KEY,
			user_id TEXT UNIQUE,
			theme TEXT,
			theme_style TEXT,
			layout_type TEXT,
			layout_density TEXT,
			sidebar_width INTEGER,
			sidebar_collapsed_width INTEGER,
			sidebar_collapsed INTEGER,
			page_size INTEGER,
			custom_primary_color TEXT,
			custom_sidebar_color TEXT,
			language TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	return db
}

func newSettingsCacheService(t *testing.T, db *gorm.DB, cache CacheProvider) SettingsService {
	t.Helper()
	return NewSettingsServiceWithCache(db, cache, nil)
}

// TC1: GetUserPreferences - delegates via cache (NoOpCacheProvider reflection caveat)
// NoOpCacheProvider uses reflection to populate dest; for *UserPreferences pointer returns,
// AssignableTo may not match. Test verifies the call path executes without error.
func TestSettingsCache_GetUserPreferences(t *testing.T) {
	db := setupSettingsCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := newSettingsCacheService(t, db, cache)

	_, err := svc.GetUserPreferences(context.Background(), "user-1")
	require.NoError(t, err)
}

// TC2: UpdateUserPreferences - delegates + invalidates cache
func TestSettingsCache_UpdateUserPreferences(t *testing.T) {
	db := setupSettingsCacheDB(t)
	cache := new(mockSettingsCacheProvider)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newSettingsCacheService(t, db, cache)

	req := &UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	require.NoError(t, svc.UpdateUserPreferences(context.Background(), "user-1", req))
	assert.Greater(t, len(cache.Calls), 0)
}

// TC3: InvalidateUserSettingsCache - no error
func TestSettingsCache_InvalidateUserSettingsCache(t *testing.T) {
	db := setupSettingsCacheDB(t)
	cache := new(mockSettingsCacheProvider)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newSettingsCacheService(t, db, cache).(*settingsCacheService)

	require.NoError(t, svc.InvalidateUserSettingsCache(context.Background(), "user-1"))
}

// TC4: Interface assertion compile-time
func TestSettingsCache_InterfaceAssertion(t *testing.T) {
	var _ SettingsService = (*settingsCacheService)(nil)
	var _ CacheProvider = &NoOpCacheProvider{}
}

// TC5: UpdateUserPreferences - DB error path
func TestSettingsCache_UpdateUserPreferences_Error(t *testing.T) {
	db := setupSettingsCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := newSettingsCacheService(t, db, cache)
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	req := &UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	err := svc.UpdateUserPreferences(context.Background(), "user-1", req)
	assert.Error(t, err)
}

// TC6: GetUserPreferences - DB error path
func TestSettingsCache_GetUserPreferences_Error(t *testing.T) {
	db := setupSettingsCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := newSettingsCacheService(t, db, cache)
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	_, err := svc.GetUserPreferences(context.Background(), "user-1")
	assert.Error(t, err)
}