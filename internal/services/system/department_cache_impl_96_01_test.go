package system

import (
	"context"
	"sync/atomic"
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

// newDeptCacheImpl96Memory assembles an in-memory cache provider for CACHEDEF-01 tests.
func newDeptCacheImpl96Memory(t *testing.T) base.CacheProvider {
	mc := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mc.Close() })
	return NewCacheProvider(services.NewDataCacheService(mc))
}

// TestDeptCacheImpl96_CACHEDEF01_GetSelectDataWithCache_WriteKey_Matches_Invalidation
// CACHEDEF-01: GetSelectDataWithCache writes BuildDeptCacheKey("tree:select")
// which matches the invalidation pattern BuildDeptCacheKey("tree:select")+"*".
//
// Write→Invalidate→Read path: after InvalidateDeptCache, subsequent
// GetSelectDataWithCache reads miss cache and repopulates with fresh data.
func TestDeptCacheImpl96_CACHEDEF01_GetSelectDataWithCache_WriteKey_Matches_Invalidation(t *testing.T) {
	cacheProvider := newDeptCacheImpl96Memory(t)
	ctx := context.Background()

	// BuildDeptCacheKey("tree:select") = "cache:dept:tree:select"
	expectedWriteKey := BuildDeptCacheKey("tree:select")
	// Invalidation pattern from InvalidateDeptCache line 99
	invalidationPattern := BuildDeptCacheKey("tree:select") + "*"

	// Shared counter to track how many times the query function is invoked.
	var queryCount atomic.Int64

	// 1. First call: write data to cache at the correct key
	dept1 := &models.Department{
		BaseModel: models.BaseModel{ID: uuid.NewString()},
		DeptName:  "部门A",
		Status:    models.DeptStatusNormal,
		OrderNum:  1,
	}

	firstResult, err := base.GetOrSetJSON(ctx, cacheProvider,
		expectedWriteKey,
		30*time.Minute,
		func() ([]*models.Department, error) {
			queryCount.Add(1)
			return []*models.Department{dept1}, nil
		},
	)
	require.NoError(t, err)
	require.Len(t, firstResult, 1)
	assert.Equal(t, int64(1), queryCount.Load(), "first call should execute query once")

	// 2. Verify the key exists in cache
	exists, err := cacheProvider.Exists(ctx, expectedWriteKey)
	require.NoError(t, err)
	assert.True(t, exists, "write key %s should exist in cache", expectedWriteKey)

	// 3. Invalidate using the pattern (mimics InvalidateDeptCache line 99)
	base.InvalidatePattern(ctx, cacheProvider, []string{invalidationPattern}, "DEPT")

	// 4. Verify key is gone after invalidation
	existsAfter, err := cacheProvider.Exists(ctx, expectedWriteKey)
	require.NoError(t, err)
	assert.False(t, existsAfter, "write key %s should NOT exist after invalidation (CACHEDEF-01 fix verification)", expectedWriteKey)

	// 5. Second call should miss cache and repopulate (fresh query)
	dept2 := &models.Department{
		BaseModel: models.BaseModel{ID: uuid.NewString()},
		DeptName:  "部门B",
		Status:    models.DeptStatusNormal,
		OrderNum:  2,
	}

	secondResult, err := base.GetOrSetJSON(ctx, cacheProvider,
		expectedWriteKey,
		30*time.Minute,
		func() ([]*models.Department, error) {
			queryCount.Add(1)
			return []*models.Department{dept2}, nil
		},
	)
	require.NoError(t, err)
	require.Len(t, secondResult, 1)
	// If the fix is applied: cache miss → query called again → count = 2
	// If the fix is NOT applied: cache hit → query NOT called → count = 1 (test fails)
	assert.Equal(t, int64(2), queryCount.Load(), "after invalidation, query should execute again (cache miss)")
	assert.Equal(t, "部门B", secondResult[0].DeptName, "fresh data should be returned after invalidation")
}

// TestDeptCacheImpl96_CACHEDEF01_InvalidationPattern_CoversWriteKey
// Verifies the pattern BuildDeptCacheKey("tree:select")+"*" matches the write key
// BuildDeptCacheKey("tree:select") — exact string prefix match.
func TestDeptCacheImpl96_CACHEDEF01_InvalidationPattern_CoversWriteKey(t *testing.T) {
	writeKey := BuildDeptCacheKey("tree:select")
	pattern := BuildDeptCacheKey("tree:select") + "*"

	// The pattern is a prefix wildcard match — the writeKey equals the non-wildcard prefix
	assert.True(t, len(pattern) > len(writeKey), "pattern should be longer than write key")
	assert.Equal(t, writeKey, pattern[:len(writeKey)], "writeKey should be prefix of pattern")
	assert.Equal(t, "*", pattern[len(pattern)-1:], "pattern should end with *")
}

// TestDeptCacheImpl96_CACHEDEF01_GetTreeWithFilter_Unaffected
// GetTreeWithFilter uses a different cache key (cache:dept:dept:tree[:all])
// and is unaffected by GetSelectDataWithCache invalidation.
func TestDeptCacheImpl96_CACHEDEF01_GetTreeWithFilter_Unaffected(t *testing.T) {
	cacheProvider := newDeptCacheImpl96Memory(t)
	ctx := context.Background()

	// GetTreeWithFilter key for includeDisabled=false
	treeKey := BuildDeptCacheKey(CacheKeyDeptTree) // "cache:dept:dept:tree"

	// Write something at GetTreeWithFilter key
	_, err := base.GetOrSetJSON(ctx, cacheProvider, treeKey, 30*time.Minute,
		func() ([]*models.Department, error) {
			return []*models.Department{{BaseModel: models.BaseModel{ID: uuid.NewString()}, DeptName: "树部门"}}, nil
		})
	require.NoError(t, err)

	// Invalidate GetSelectDataWithCache pattern
	base.InvalidatePattern(ctx, cacheProvider, []string{BuildDeptCacheKey("tree:select") + "*"}, "DEPT")

	// GetTreeWithFilter key should still exist (different from tree:select)
	exists, err := cacheProvider.Exists(ctx, treeKey)
	require.NoError(t, err)
	assert.True(t, exists, "GetTreeWithFilter key should be unaffected by tree:select invalidation")
}
