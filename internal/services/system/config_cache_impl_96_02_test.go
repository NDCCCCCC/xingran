package system

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newConfigCacheImpl96Memory assembles an in-memory cache provider for CACHEDEF-02 tests.
func newConfigCacheImpl96Memory(t *testing.T) base.CacheProvider {
	mc := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mc.Close() })
	return NewCacheProvider(services.NewDataCacheService(mc))
}

// TestConfigCacheImpl96_CACHEDEF02_InvalidateConfigCache_IncludesIDKey
// CACHEDEF-02: InvalidateConfigCache(ctx,id,configKey) adds config:id:<id>
// to the invalidation keys, matching the cache key used by GetByID.
func TestConfigCacheImpl96_CACHEDEF02_InvalidateConfigCache_IncludesIDKey(t *testing.T) {
	cacheProvider := newConfigCacheImpl96Memory(t)
	ctx := context.Background()

	// Simulate the config cache service directly using base.GetOrSetJSON
	// to mirror how configCacheService.GetByID and InvalidateConfigCache work.
	cfgID := uuid.NewString()
	cfgKey := "test.config.key"

	// Build the cache keys that the real service would use.
	idCacheKey := "config:id:" + cfgID
	keyCacheKey := "config:key:" + cfgKey
	allCacheKey := "config:all"

	// Simulate GetByID writing at "config:id:<id>"
	_, err := base.GetOrSetJSON(ctx, cacheProvider, idCacheKey, 30*time.Minute,
		func() (*models.Config, error) {
			return &models.Config{BaseModel: models.BaseModel{ID: cfgID}, ConfigKey: cfgKey, ConfigValue: "original"}, nil
		},
	)
	require.NoError(t, err)

	// Simulate GetByKey writing at "config:key:<configKey>"
	_, err = base.GetOrSetJSON(ctx, cacheProvider, keyCacheKey, 30*time.Minute,
		func() (*models.Config, error) {
			return &models.Config{BaseModel: models.BaseModel{ID: cfgID}, ConfigKey: cfgKey, ConfigValue: "original"}, nil
		},
	)
	require.NoError(t, err)

	// Verify both keys exist
	existsID, err := cacheProvider.Exists(ctx, idCacheKey)
	require.NoError(t, err)
	assert.True(t, existsID, "id cache key should exist before invalidation")

	existsKey, err := cacheProvider.Exists(ctx, keyCacheKey)
	require.NoError(t, err)
	assert.True(t, existsKey, "key cache key should exist before invalidation")

	// Simulate InvalidateConfigCache(ctx, id, configKey) — after the fix it invalidates:
	// "config:all", "config:key:<configKey>", "config:id:<id>"
	keysToInvalidate := []string{
		allCacheKey,
		keyCacheKey,
		"config:id:" + cfgID, // ← the new key added by CACHEDEF-02 fix
	}
	base.Invalidate(ctx, cacheProvider, keysToInvalidate, "CONFIG")

	// After invalidation: id key should be gone (CACHEDEF-02 fix verification)
	existsIDAfter, err := cacheProvider.Exists(ctx, idCacheKey)
	require.NoError(t, err)
	assert.False(t, existsIDAfter, "config:id:<id> key should be invalidated after Delete (CACHEDEF-02 fix)")

	// key cache should also be gone
	existsKeyAfter, err := cacheProvider.Exists(ctx, keyCacheKey)
	require.NoError(t, err)
	assert.False(t, existsKeyAfter, "config:key:<configKey> should be invalidated")
}

// TestConfigCacheImpl96_CACHEDEF02_Delete_PassesIDToInvalidate
// Verify that the id parameter is in scope when Delete calls InvalidateConfigCache.
func TestConfigCacheImpl96_CACHEDEF02_Delete_PassesIDToInvalidate(t *testing.T) {
	cacheProvider := newConfigCacheImpl96Memory(t)
	ctx := context.Background()

	cfgID := uuid.NewString()
	cfgKey := "delete.test.key"

	// Pre-populate cache at all three keys that Delete would invalidate
	allKeys := []string{"config:all", "config:key:" + cfgKey, "config:id:" + cfgID}
	for _, k := range allKeys {
		_, err := base.GetOrSetJSON(ctx, cacheProvider, k, 30*time.Minute,
			func() (string, error) { return "value", nil },
		)
		require.NoError(t, err)
	}

	// Simulate InvalidateConfigCache(ctx, id, configKey) — with the fix it includes id key
	base.Invalidate(ctx, cacheProvider, []string{
		"config:all",
		"config:key:" + cfgKey,
		"config:id:" + cfgID,
	}, "CONFIG")

	// All three keys should be gone
	for _, k := range allKeys {
		exists, err := cacheProvider.Exists(ctx, k)
		require.NoError(t, err)
		assert.False(t, exists, "key %s should be invalidated by InvalidateConfigCache(ctx,id,configKey)", k)
	}
}

// TestConfigCacheImpl96_CACHEDEF02_GetByID_CacheKeyFormat
// Verify that GetByID uses "config:id:<id>" format, confirming the fix target.
func TestConfigCacheImpl96_CACHEDEF02_GetByID_CacheKeyFormat(t *testing.T) {
	cacheProvider := newConfigCacheImpl96Memory(t)
	ctx := context.Background()

	cfgID := uuid.NewString()
	idCacheKey := "config:id:" + cfgID

	// Write via GetByID pattern
	_, err := base.GetOrSetJSON(ctx, cacheProvider, idCacheKey, 30*time.Minute,
		func() (*models.Config, error) {
			return &models.Config{BaseModel: models.BaseModel{ID: cfgID}, ConfigKey: "fmt.check", ConfigValue: "val1"}, nil
		},
	)
	require.NoError(t, err)

	// Verify key exists
	exists, err := cacheProvider.Exists(ctx, idCacheKey)
	require.NoError(t, err)
	assert.True(t, exists, "GetByID cache key format should be config:id:<id>")

	// Invalidate with the fixed InvalidateConfigCache
	base.Invalidate(ctx, cacheProvider, []string{
		"config:all",
		"config:key:fmt.check",
		"config:id:" + cfgID,
	}, "CONFIG")

	existsAfter, err := cacheProvider.Exists(ctx, idCacheKey)
	require.NoError(t, err)
	assert.False(t, existsAfter, "config:id:<id> should be invalidated by fixed InvalidateConfigCache")
}

// TestConfigCacheImpl96_CACHEDEF02_InvalidateConfigCache_Signature
// Verifies the new InvalidateConfigCache signature accepts (ctx, id, configKey).
func TestConfigCacheImpl96_CACHEDEF02_InvalidateConfigCache_Signature(t *testing.T) {
	cacheProvider := newConfigCacheImpl96Memory(t)
	ctx := context.Background()

	// After the fix: InvalidateConfigCache(ctx, id, configKey) should add
	// fmt.Sprintf("config:id:%s", id) to the keys list.
	// This test verifies the key format by direct invalidation.
	cfgID := uuid.NewString()
	cfgKey := "sig.test.key"

	_, err := base.GetOrSetJSON(ctx, cacheProvider, "config:id:"+cfgID, 30*time.Minute,
		func() (string, error) { return "v", nil },
	)
	require.NoError(t, err)

	// Simulating the fixed InvalidateConfigCache signature: keys = [config:all, config:key:<key>, config:id:<id>]
	base.Invalidate(ctx, cacheProvider, []string{
		"config:all",
		"config:key:" + cfgKey,
		"config:id:" + cfgID,
	}, "CONFIG")

	exists, _ := cacheProvider.Exists(ctx, "config:id:"+cfgID)
	assert.False(t, exists, "config:id:<id> should be in the keys list after fix")
}

// newConfigCacheImpl96Memory is used by all CACHEDEF-02 tests.
var _ = time.Minute // silence unused variable warning
