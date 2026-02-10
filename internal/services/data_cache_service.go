package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
)

// DataCacheService 数据缓存服务
type DataCacheService struct {
	cache       cache.Cache
	cacheConfig *CacheConfigService
}

// NewDataCacheService 创建数据缓存服务
func NewDataCacheService(cache cache.Cache) *DataCacheService {
	return &DataCacheService{cache: cache}
}

// SetCacheConfig 设置缓存配置服务
func (s *DataCacheService) SetCacheConfig(cacheConfig *CacheConfigService) {
	s.cacheConfig = cacheConfig
}

// GetExpiration 获取缓存过期时间
func (s *DataCacheService) GetExpiration(configKey string, defaultExpiration time.Duration) time.Duration {
	if s.cacheConfig != nil {
		return s.cacheConfig.GetDurationWithDefault(configKey, defaultExpiration)
	}
	return defaultExpiration
}

// Get 获取缓存数据（自动反序列化）
func (s *DataCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("获取缓存失败: %w", err)
	}
	if data == "" {
		return apperrors.CacheKeyNotFound()
	}
	return json.Unmarshal([]byte(data), dest)
}

// Set 设置缓存数据（自动序列化）
func (s *DataCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	return s.cache.Set(ctx, key, string(data), expiration)
}

// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
func (s *DataCacheService) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error {
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	data, err := query()
	if err != nil {
		return fmt.Errorf("查询数据失败: %w", err)
	}

	// 序列化到目标变量
	dataBytes, _ := json.Marshal(data)
	_ = json.Unmarshal(dataBytes, dest)

	// 异步写入缓存
	go func() {
		_ = s.Set(context.Background(), key, data, expiration)
	}()

	return nil
}

// Delete 删除缓存
func (s *DataCacheService) Delete(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}

// DeleteByPattern 根据模式删除缓存
func (s *DataCacheService) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("查询缓存键失败: %w", err)
	}
	if len(keys) > 0 {
		return s.cache.MDelete(ctx, keys...)
	}
	return nil
}

// MGet 批量获取缓存
func (s *DataCacheService) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	values, err := s.cache.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("批量获取缓存失败: %w", err)
	}
	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if i < len(values) && values[i] != "" {
			result[key] = values[i]
		}
	}
	return result, nil
}

// MDelete 批量删除缓存
func (s *DataCacheService) MDelete(ctx context.Context, keys ...string) error {
	return s.cache.MDelete(ctx, keys...)
}

// Exists 检查缓存是否存在
func (s *DataCacheService) Exists(ctx context.Context, key string) (bool, error) {
	return s.cache.Exists(ctx, key)
}

// SetTTL 设置缓存过期时间
func (s *DataCacheService) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return s.cache.Expire(ctx, key, expiration)
}

// GetTTL 获取缓存过期时间
func (s *DataCacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return s.cache.TTL(ctx, key)
}

// GetStats 获取缓存统计信息
func (s *DataCacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	// 返回基本统计信息
	stats := &CacheStats{}
	if keys, err := s.cache.Keys(ctx, "*"); err == nil {
		stats.KeyCount = len(keys)
		stats.Count = int64(len(keys))
	}
	return stats, nil
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits        int64                  // 命中次数
	Misses      int64                  // 未命中次数
	Count       int64                  // 缓存项数量
	MemorySize  int64                  // 内存占用（字节）
	HitRate     float64                // 命中率
	KeyCount    int                    // 键数量
	ExtendedStats map[string]interface{} // 扩展统计
}

// ==================== 缓存键构建器 ====================

// CacheKeyBuilder 缓存键构建器
type CacheKeyBuilder struct {
	prefix string
}

// NewCacheKeyBuilder 创建缓存键构建器
func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

// Build 构建缓存键
func (b *CacheKeyBuilder) Build(params ...interface{}) string {
	key := b.prefix
	for _, p := range params {
		key += ":" + fmt.Sprintf("%v", p)
	}
	return key
}

// ==================== 预定义的缓存键 ====================

const (
	CacheKeyDeptTree       = "dept:tree"
	CacheKeyDeptList       = "dept:list"
	CacheKeyDeptTreeSelect = "dept:tree:select"
	CacheKeyRoleMenus       = "role:menus"
	CacheKeyRoleMenuTree    = "role:menu:tree"
	CacheKeyRoleDepts       = "role:depts"
	CacheKeyRoleAll         = "role:all"
	CacheKeyRoleEnabled     = "role:enabled"
	CacheKeyDictType        = "dict:type"
	CacheKeyDictDataByType = "dict:data:type"
	CacheKeyMenuTree        = "menu:tree"
	CacheKeyMenuRouter      = "menu:router"
	CacheKeyMenuAll         = "menu:all"
	CacheKeyPostAll         = "post:all"
	CacheKeyPostEnabled     = "post:enabled"
	CacheKeyUserInfo       = "user:info"
	CacheKeyUserRoles      = "user:roles"
	CacheKeyUserPermissions = "user:permissions"
	CacheKeyUserByID       = "user:id"
	CacheKeyUserByUsername = "user:username"
	CacheKeyUserList       = "user:list"
)

// GetDeptTreeKey 获取部门树缓存键
func GetDeptTreeKey() string {
	return CacheKeyDeptTree
}

// GetRoleMenusKey 获取角色菜单权限缓存键
func GetRoleMenusKey(roleID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyRoleMenus, roleID)
}

// GetDictDataByTypeKey 获取字典数据缓存键
func GetDictDataByTypeKey(dictType string) string {
	return fmt.Sprintf("%s:%s", CacheKeyDictDataByType, dictType)
}

// GetMenuTreeKey 获取菜单树缓存键
func GetMenuTreeKey(includeHidden bool) string {
	if includeHidden {
		return CacheKeyMenuTree + ":all"
	}
	return CacheKeyMenuTree
}

// GetUserByIDKey 获取用户ID缓存键
func GetUserByIDKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserByID, userID)
}

// GetUserByUsernameKey 获取用户名缓存键
func GetUserByUsernameKey(username string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserByUsername, username)
}

// GetUserRolesKey 获取用户角色缓存键
func GetUserRolesKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserRoles, userID)
}

// GetUserPermissionsKey 获取用户权限缓存键
func GetUserPermissionsKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserPermissions, userID)
}
