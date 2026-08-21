package system

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/services"
)

// =====================================================================
// Phase 74-04: cache_adapter tests
//
// Scope (cache_adapter.go — 10 funcs):
//   - NewDataCacheAdapter
//   - GetOrSet, Delete, DeleteByPattern, MGet, MDelete
//   - Exists, SetTTL, GetTTL, GetStats
//
// Uses a tiny in-memory cache mock to cover all adapter methods.
// =====================================================================

type memCache struct {
	data map[string]string
}

func (m *memCache) Get(ctx context.Context, key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}
func (m *memCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.data[key] = value.(string)
	return nil
}
func (m *memCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}
func (m *memCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}
func (m *memCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i], _ = m.Get(ctx, k)
	}
	return out, nil
}
func (m *memCache) MSet(ctx context.Context, pairs ...interface{}) error { return nil }
func (m *memCache) MDelete(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}
func (m *memCache) Increment(ctx context.Context, key string) (int64, error)      { return 0, nil }
func (m *memCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}
func (m *memCache) Decrement(ctx context.Context, key string) (int64, error)      { return 0, nil }
func (m *memCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}
func (m *memCache) Expire(ctx context.Context, key string, expiration time.Duration) error { return nil }
func (m *memCache) TTL(ctx context.Context, key string) (time.Duration, error)              { return time.Minute, nil }
func (m *memCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	out := []string{}
	for k := range m.data {
		if matched, _ := filepath.Match(pattern, k); matched {
			out = append(out, k)
		}
	}
	return out, nil
}
func (m *memCache) FlushDB(ctx context.Context) error { return nil }
func (m *memCache) Close() error                      { return nil }
func (m *memCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *memCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	return nil
}
func (m *memCache) HGet(ctx context.Context, key, field string) (string, error) { return "", nil }
func (m *memCache) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return nil
}
func (m *memCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *memCache) HDel(ctx context.Context, key string, fields ...string) error  { return nil }
func (m *memCache) HKeys(ctx context.Context, key string) ([]string, error)       { return nil, nil }
func (m *memCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error {
	return nil
}
func (m *memCache) GetInt(ctx context.Context, key string) (int, error)          { return 0, nil }
func (m *memCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}
func (m *memCache) GetJSON(ctx context.Context, key string, dest interface{}) error { return nil }

func newMemCache() *memCache {
	return &memCache{data: make(map[string]string)}
}

func newDataCacheService() *services.DataCacheService {
	return services.NewDataCacheService(newMemCache())
}

func TestNewDataCacheAdapter(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)
	require.NotNil(t, adapter)
}

func TestDataCacheAdapter_GetOrSet(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	var dest string
	queryCalled := 0
	err := adapter.GetOrSet(context.Background(), "k1", &dest, time.Minute, func() (interface{}, error) {
		queryCalled++
		return "value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "value", dest)
	assert.Equal(t, 1, queryCalled)

	// Second call should be cached; query should not be called again.
	err = adapter.GetOrSet(context.Background(), "k1", &dest, time.Minute, func() (interface{}, error) {
		queryCalled++
		return "value2", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "value", dest, "cached value should remain unchanged")
	assert.Equal(t, 1, queryCalled)
}

func TestDataCacheAdapter_Delete(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	_ = adapter.GetOrSet(context.Background(), "k1", new(string), time.Minute, func() (interface{}, error) { return "v", nil })
	err := adapter.Delete(context.Background(), "k1")
	require.NoError(t, err)

	exists, err := adapter.Exists(context.Background(), "k1")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDataCacheAdapter_DeleteByPattern(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	_ = adapter.GetOrSet(context.Background(), "a:1", new(string), time.Minute, func() (interface{}, error) { return "v1", nil })
	_ = adapter.GetOrSet(context.Background(), "a:2", new(string), time.Minute, func() (interface{}, error) { return "v2", nil })
	_ = adapter.GetOrSet(context.Background(), "b:1", new(string), time.Minute, func() (interface{}, error) { return "v3", nil })

	err := adapter.DeleteByPattern(context.Background(), "a:*")
	require.NoError(t, err)

	existsA, _ := adapter.Exists(context.Background(), "a:1")
	existsB, _ := adapter.Exists(context.Background(), "b:1")
	assert.False(t, existsA, "a:1 should be deleted by pattern")
	assert.True(t, existsB, "b:1 should remain")
}

func TestDataCacheAdapter_MGet_MDelete(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	_ = adapter.GetOrSet(context.Background(), "k1", new(string), time.Minute, func() (interface{}, error) { return "v1", nil })
	_ = adapter.GetOrSet(context.Background(), "k2", new(string), time.Minute, func() (interface{}, error) { return "v2", nil })

	vals, err := adapter.MGet(context.Background(), "k1", "k2")
	require.NoError(t, err)
	assert.Equal(t, "\"v1\"", vals["k1"])
	assert.Equal(t, "\"v2\"", vals["k2"])

	err = adapter.MDelete(context.Background(), "k1", "k2")
	require.NoError(t, err)

	exists, _ := adapter.Exists(context.Background(), "k1")
	assert.False(t, exists)
}

func TestDataCacheAdapter_SetTTL_GetTTL(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	_ = adapter.GetOrSet(context.Background(), "k1", new(string), time.Minute, func() (interface{}, error) { return "v", nil })

	err := adapter.SetTTL(context.Background(), "k1", 2*time.Minute)
	require.NoError(t, err)

	ttl, err := adapter.GetTTL(context.Background(), "k1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, int64(ttl), int64(0))
}

func TestDataCacheAdapter_GetStats(t *testing.T) {
	dcs := newDataCacheService()
	adapter := NewDataCacheAdapter(dcs)

	_ = adapter.GetOrSet(context.Background(), "k1", new(string), time.Minute, func() (interface{}, error) { return "v", nil })

	stats, err := adapter.GetStats(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, stats)
}
