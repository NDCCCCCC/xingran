package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheProvider 模拟缓存提供者
type MockCacheProvider struct {
	mock.Mock
}

func (m *MockCacheProvider) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	args := m.Called(ctx, key, dest, expiration, query)
	return args.Error(0)
}

func (m *MockCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return make(map[string]string), args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return time.Duration(args.Int(0)), args.Error(1)
}

func (m *MockCacheProvider) GetStats(ctx context.Context) (*system.CacheStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*system.CacheStats), args.Error(1)
}

// TestCacheInvalidator_InvalidateByEntityType 测试按实体类型清理缓存
func TestCacheInvalidator_InvalidateByEntityType(t *testing.T) {
	mockCache := new(MockCacheProvider)
	invalidator := NewCacheInvalidator(mockCache)

	ctx := context.Background()

	tests := []struct {
		name        string
		entityType  string
		patterns    []string
		expectCalls int
		expectError bool
		setupMock   func(*MockCacheProvider)
	}{
		{
			name:        "正常清理building缓存",
			entityType:  "building",
			patterns:    []string{"building:*", "buildings:*"},
			expectCalls: 2,
			expectError: false,
			setupMock: func(m *MockCacheProvider) {
				m.On("DeleteByPattern", ctx, "building:*").Return(nil)
				m.On("DeleteByPattern", ctx, "buildings:*").Return(nil)
			},
		},
		{
			name:        "部分缓存清理失败",
			entityType:  "floor",
			patterns:    []string{"floor:*", "floors:*"},
			expectCalls: 2,
			expectError: false, // 不返回错误，只记录警告
			setupMock: func(m *MockCacheProvider) {
				m.On("DeleteByPattern", ctx, "floor:*").Return(nil)
				m.On("DeleteByPattern", ctx, "floors:*").Return(errors.New("cache error"))
			},
		},
		{
			name:        "无缓存模式",
			entityType:  "test",
			patterns:    []string{},
			expectCalls: 0,
			expectError: false,
			setupMock: func(m *MockCacheProvider) {
				// 不设置任何期望
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockCache)

			err := invalidator.InvalidateByEntityType(ctx, tt.entityType, tt.patterns)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCache.AssertExpectations(t)
		})
	}
}

// TestCacheInvalidator_InvalidateByPatterns 测试按模式列表清理缓存
func TestCacheInvalidator_InvalidateByPatterns(t *testing.T) {
	mockCache := new(MockCacheProvider)
	invalidator := NewCacheInvalidator(mockCache)

	ctx := context.Background()

	tests := []struct {
		name        string
		patterns    []string
		module      string
		expectCalls int
		expectError bool
		setupMock   func(*MockCacheProvider)
	}{
		{
			name:        "清理多个模式",
			patterns:    []string{"user:*", "dept:*", "role:*"},
			module:      "system",
			expectCalls: 3,
			expectError: false,
			setupMock: func(m *MockCacheProvider) {
				m.On("DeleteByPattern", ctx, "user:*").Return(nil)
				m.On("DeleteByPattern", ctx, "dept:*").Return(nil)
				m.On("DeleteByPattern", ctx, "role:*").Return(nil)
			},
		},
		{
			name:        "清理单个模式",
			patterns:    []string{"building:*"},
			module:      "operations",
			expectCalls: 1,
			expectError: false,
			setupMock: func(m *MockCacheProvider) {
				m.On("DeleteByPattern", ctx, "building:*").Return(nil)
			},
		},
		{
			name:        "空模式列表",
			patterns:    []string{},
			module:      "test",
			expectCalls: 0,
			expectError: false,
			setupMock: func(m *MockCacheProvider) {
				// 不设置任何期望
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockCache)

			err := invalidator.InvalidateByPatterns(ctx, tt.patterns, tt.module)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCache.AssertExpectations(t)
		})
	}
}

// TestCacheInvalidator_NilCache 测试空缓存提供者
func TestCacheInvalidator_NilCache(t *testing.T) {
	invalidator := NewCacheInvalidator(nil)

	ctx := context.Background()

	// 不应该panic，只是不执行任何操作
	err := invalidator.InvalidateByPatterns(ctx, []string{"test:*"}, "test")
	assert.NoError(t, err)
}
