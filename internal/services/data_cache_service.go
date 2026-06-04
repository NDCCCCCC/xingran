package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

// ==================== 缓存键命名规范迁移说明 ====================
//
// 本文件已更新为使用统一的缓存键命名规范（新规范）。
//
// **旧规范**（已弃用）:
//   - 格式: {type}:{params}
//   - 例如: user:id:uuid, role:menus:roleId, dept:tree
//   - 问题: 缺少统一前缀，Redis中键管理混乱
//
// **新规范**（当前使用）:
//   - 格式: cache:{module}:{type}:{params}
//   - 例如: cache:user:id:uuid, cache:role:menus:roleId, cache:dept:tree
//   - 优势: 统一前缀，便于管理和批量操作
//
// **迁移操作**:
//   1. 本文件中的键常量已更新为新规范格式
//   2. 键构建函数保持向后兼容，内部使用新规范
//   3. Redis中的旧键需要手动清理（见下方迁移脚本）
//
// **清理Redis旧键脚本**（在redis-cli中执行）:
//   // 删除旧规范的键（无cache:前缀）
//   SCAN 0 MATCH user:* COUNT 1000 | xargs -I {} DEL {}     // 删除用户旧键
//   SCAN 0 MATCH role:* COUNT 1000 | xargs -I {} DEL {}     // 删除角色旧键
//   SCAN 0 MATCH dept:* COUNT 1000 | xargs -I {} DEL {}    // 删除部门旧键
//   SCAN 0 MATCH menu:* COUNT 1000 | xargs -I {} DEL {}    // 删除菜单旧键
//   SCAN 0 MATCH dict:* COUNT 1000 | xargs -I {} DEL {}    // 删除字典旧键
//   SCAN 0 MATCH post:* COUNT 1000 | xargs -I {} DEL {}    // 删除岗位旧键
//
// **推荐清理方式**（更安全，先查看后删除）:
//   1. 查看旧键数量: SCAN 0 MATCH user:* COUNT 1000
//   2. 查看具体键名: KEYS user:id:*
//   3. 确认后删除: DEL user:id:uuid1 user:id:uuid2 ...
//
// **注意**: 新键会自动创建（格式: cache:user:id:uuid），无需手动迁移数据
//
// ===========================================================

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
	Hits          int64                  // 命中次数
	Misses        int64                  // 未命中次数
	Count         int64                  // 缓存项数量
	MemorySize    int64                  // 内存占用（字节）
	HitRate       float64                // 命中率
	KeyCount      int                    // 键数量
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
// 遵循 Go 最佳实践：使用 strings.Builder 避免循环中的多次内存分配
func (b *CacheKeyBuilder) Build(params ...interface{}) string {
	if len(params) == 0 {
		return b.prefix
	}

	// 预分配容量：前缀长度 + (参数数量 * (1个冒号 + 平均16字符))
	var sb strings.Builder
	sb.Grow(len(b.prefix) + len(params)*17)
	sb.WriteString(b.prefix)

	for _, p := range params {
		sb.WriteByte(':')
		// 使用类型断言进行更高效的类型转换
		// 注意：需要按具体类型断言，而不是使用逗号分隔的类型列表
		switch v := p.(type) {
		case string:
			sb.WriteString(v)
		case int:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int8:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int16:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int32:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int64:
			sb.WriteString(strconv.FormatInt(v, 10))
		case uint:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint8:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint16:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint32:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint64:
			sb.WriteString(strconv.FormatUint(v, 10))
		case float32:
			sb.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 64))
		case float64:
			sb.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			if v {
				sb.WriteString("true")
			} else {
				sb.WriteString("false")
			}
		default:
			// 对于其他类型，回退到 fmt.Sprintf
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}

	return sb.String()
}

// ==================== 预定义的缓存键 ====================
// 注意：这些键定义已迁移到 system/cache_keys.go 的新规范
// 使用 CacheKeyManager 构建，键格式为 cache:{module}:{type}:{params}
// 例如：cache:user:id:uuid, cache:role:menus:roleId

const (
	// 已弃用的旧规范键常量，保留用于向后兼容
	// 新代码应使用 system.BuildXxxCacheKey() 函数
	CacheKeyDeptTree        = "cache:dept:tree"
	CacheKeyDeptList        = "cache:dept:list"
	CacheKeyDeptTreeSelect  = "cache:dept:tree:select"
	CacheKeyRoleMenus       = "cache:role:menus"
	CacheKeyRoleMenuTree    = "cache:role:menu:tree"
	CacheKeyRoleDepts       = "cache:role:depts"
	CacheKeyRoleAll         = "cache:role:all"
	CacheKeyRoleEnabled     = "cache:role:enabled"
	CacheKeyDictType        = "cache:dict:type"
	CacheKeyDictDataByType  = "cache:dict:data:type"
	CacheKeyMenuTree        = "cache:menu:tree"
	CacheKeyMenuRouter      = "cache:menu:router"
	CacheKeyMenuAll         = "cache:menu:all"
	CacheKeyPostAll         = "cache:post:all"
	CacheKeyPostEnabled     = "cache:post:enabled"
	CacheKeyUserInfo        = "cache:user:info"
	CacheKeyUserRoles       = "cache:user:roles"
	CacheKeyUserPermissions = "cache:user:permissions"
	CacheKeyUserByID        = "cache:user:id"
	CacheKeyUserByUsername  = "cache:user:username"
	CacheKeyUserList        = "cache:user:list"
)

// GetDeptTreeKey 获取部门树缓存键
// 已更新为新规范：cache:dept:tree
func GetDeptTreeKey() string {
	return CacheKeyDeptTree
}

// GetRoleMenusKey 获取角色菜单权限缓存键
// 已更新为新规范：cache:role:menus:{roleID}
func GetRoleMenusKey(roleID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyRoleMenus, roleID)
}

// GetDictDataByTypeKey 获取字典数据缓存键
// 已更新为新规范：cache:dict:data:type:{dictType}
func GetDictDataByTypeKey(dictType string) string {
	return fmt.Sprintf("%s:%s", CacheKeyDictDataByType, dictType)
}

// GetMenuTreeKey 获取菜单树缓存键
// 已更新为新规范：cache:menu:tree 或 cache:menu:tree:all
func GetMenuTreeKey(includeHidden bool) string {
	if includeHidden {
		return CacheKeyMenuTree + ":all"
	}
	return CacheKeyMenuTree
}

// GetUserByIDKey 获取用户ID缓存键
// 已更新为新规范：cache:user:id:{userID}
func GetUserByIDKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserByID, userID)
}

// GetUserByUsernameKey 获取用户名缓存键
// 已更新为新规范：cache:user:username:{username}
func GetUserByUsernameKey(username string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserByUsername, username)
}

// GetUserRolesKey 获取用户角色缓存键
// 已更新为新规范：cache:user:roles:{userID}
func GetUserRolesKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserRoles, userID)
}

// GetUserPermissionsKey 获取用户权限缓存键
// 已更新为新规范：cache:user:permissions:{userID}
func GetUserPermissionsKey(userID string) string {
	return fmt.Sprintf("%s:%s", CacheKeyUserPermissions, userID)
}
