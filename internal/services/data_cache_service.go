package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// DataCacheService 数据缓存服务
// 提供通用的缓存操作方法，支持序列化、反序列化和常见的缓存模式
type DataCacheService struct {
	cache       cache.Cache
	cacheConfig *CacheConfigService // 缓存配置服务（可选）
}

// NewDataCacheService 创建数据缓存服务
func NewDataCacheService(cache cache.Cache) *DataCacheService {
	return &DataCacheService{
		cache: cache,
	}
}

// SetCacheConfig 设置缓存配置服务
func (s *DataCacheService) SetCacheConfig(cacheConfig *CacheConfigService) {
	s.cacheConfig = cacheConfig
}

// GetExpiration 获取缓存过期时间
// 优先使用动态配置，如果没有配置则使用默认值
func (s *DataCacheService) GetExpiration(configKey string, defaultExpiration time.Duration) time.Duration {
	if s.cacheConfig != nil {
		duration := s.cacheConfig.GetDurationWithDefault(configKey, defaultExpiration)
		logger.Infof("[CACHE] GetExpiration: configKey=%s, duration=%v (%d秒)", configKey, duration, int(duration.Seconds()))
		return duration
	}
	logger.Infof("[CACHE] GetExpiration: configKey=%s, 使用默认值=%v (%d秒)", configKey, defaultExpiration, int(defaultExpiration.Seconds()))
	return defaultExpiration
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Key        string        // 缓存键
	Expiration time.Duration // 过期时间
}

// DefaultCacheConfig 默认缓存配置
var DefaultCacheConfig = CacheConfig{
	Expiration: 30 * time.Minute,
}

// Get 获取缓存数据（自动反序列化）
func (s *DataCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	// 先尝试从缓存获取
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	// 缓存不存在
	if data == "" {
		return fmt.Errorf("缓存不存在")
	}

	// 反序列化
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		logger.Warnf("[CACHE] 反序列化失败 key=%s: %v", key, err)
		return fmt.Errorf("反序列化失败: %w", err)
	}

	return nil
}

// Set 设置缓存数据（自动序列化）
func (s *DataCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 序列化
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	logger.Infof("[CACHE] Set调用: key=%s, expiration=%v (%d秒)", key, expiration, int(expiration.Seconds()))

	// 存入缓存
	if err := s.cache.Set(ctx, key, string(data), expiration); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	logger.Infof("[CACHE] Set成功: key=%s", key)

	return nil
}

// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
// 这是一个常用的 Cache-Aside 模式实现
func (s *DataCacheService) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error {
	// 先尝试从缓存获取
	err := s.Get(ctx, key, dest)
	if err == nil {
		// 缓存命中
		logger.Infof("[CACHE] 命中 key=%s", key)
		return nil
	}

	// 缓存未命中
	logger.Infof("[CACHE] 未命中 key=%s，开始查询数据库", key)

	// 缓存未命中，执行查询函数
	data, err := query()
	if err != nil {
		return fmt.Errorf("查询数据失败: %w", err)
	}

	// 将查询结果序列化到目标变量
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	if err := json.Unmarshal(dataBytes, dest); err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}

	logger.Infof("[CACHE] 查询成功，准备写入缓存 key=%s, TTL=%v", key, expiration)

	// 存入缓存（异步执行，不阻塞主流程）
	go func() {
		if err := s.Set(context.Background(), key, data, expiration); err != nil {
			logger.Warnf("[CACHE] 设置缓存失败 key=%s: %v", key, err)
		} else {
			logger.Infof("[CACHE] 设置缓存成功 key=%s", key)
		}
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

// ==================== 缓存键生成器 ====================

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
	// 部门相关缓存键
	CacheKeyDeptTree       = "dept:tree"
	CacheKeyDeptList       = "dept:list"
	CacheKeyDeptTreeSelect = "dept:tree:select"

	// 角色相关缓存键
	CacheKeyRoleMenus    = "role:menus"     // 角色菜单权限
	CacheKeyRoleMenuTree = "role:menu:tree" // 角色菜单树
	CacheKeyRoleDepts    = "role:depts"     // 角色部门权限

	// 字典相关缓存键
	CacheKeyDictType       = "dict:type"      // 字典类型
	CacheKeyDictDataByType = "dict:data:type" // 按类型查询字典数据

	// 用户相关缓存键
	CacheKeyUserInfo        = "user:info"
	CacheKeyUserRoles       = "user:roles"
	CacheKeyUserPermissions = "user:permissions"
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
