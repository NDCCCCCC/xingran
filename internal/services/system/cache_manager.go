package system

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// CacheManager 缓存管理器
// 负责缓存预热、失效和统计信息管理
type CacheManager struct {
	cache       CacheProvider
	keyManager  *CacheKeyManager
	warmUpCache map[string]bool // 记录已预热的缓存
	mu          sync.RWMutex
	enabled     bool // 是否启用缓存预热
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(cache CacheProvider, prefix string, enabled bool) *CacheManager {
	return &CacheManager{
		cache:       cache,
		keyManager:  NewCacheKeyManager(prefix),
		warmUpCache: make(map[string]bool),
		enabled:     enabled,
	}
}

// WarmUp 执行缓存预热
// 预热指定的缓存键列表
func (m *CacheManager) WarmUp(ctx context.Context, warmUpFuncs map[string]WarmUpFunc) error {
	if !m.enabled {
		logger.Infof("缓存预热已禁用，跳过")
		return nil
	}

	logger.Infof("开始缓存预热，共 %d 个预热任务", len(warmUpFuncs))

	var wg sync.WaitGroup
	errChan := make(chan error, len(warmUpFuncs))

	for name, warmUpFunc := range warmUpFuncs {
		// 检查是否已经预热过
		m.mu.RLock()
		if m.warmUpCache[name] {
			m.mu.RUnlock()
			logger.Infof("缓存 %s 已预热，跳过", name)
			continue
		}
		m.mu.RUnlock()

		wg.Add(1)
		go func(cacheName string, fn WarmUpFunc) {
			defer wg.Done()

			start := time.Now()
			logger.Infof("开始预热缓存: %s", cacheName)

			if err := fn(ctx, m.cache); err != nil {
				logger.Errorf("预热缓存失败 %s: %v", cacheName, err)
				errChan <- fmt.Errorf("预热 %s 失败: %w", cacheName, err)
				return
			}

			// 标记为已预热
			m.mu.Lock()
			m.warmUpCache[cacheName] = true
			m.mu.Unlock()

			logger.Infof("缓存预热完成: %s，耗时: %v", cacheName, time.Since(start))
		}(name, warmUpFunc)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("缓存预热完成，但有 %d 个失败: %v", len(errors), errors)
	}

	logger.Infof("缓存预热全部完成")
	return nil
}

// WarmUpFunc 预热函数类型
// ctx: 上下文
// cache: 缓存提供者
// 返回: 错误信息
type WarmUpFunc func(ctx context.Context, cache CacheProvider) error

// InvalidateByModule 按模块清除缓存
// module: 模块名称（如 "user", "role", "menu" 等）
// keyType: 可选，键类型（如 "id", "list" 等），为空则清除整个模块
func (m *CacheManager) InvalidateByModule(ctx context.Context, module string, keyType string) error {
	pattern := BuildInvalidatePattern(module, keyType)
	logger.Infof("清除缓存: module=%s, keyType=%s, pattern=%s", module, keyType, pattern)

	if err := m.cache.DeleteByPattern(ctx, pattern); err != nil {
		logger.Errorf("清除缓存失败: %v", err)
		return err
	}

	logger.Infof("缓存清除成功: %s", pattern)
	return nil
}

// InvalidateByKey 按键清除缓存
func (m *CacheManager) InvalidateByKey(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	logger.Infof("清除缓存键: %v", keys)

	if err := m.cache.MDelete(ctx, keys...); err != nil {
		logger.Errorf("批量删除缓存失败: %v", err)
		return err
	}

	logger.Infof("缓存键清除成功，共 %d 个", len(keys))
	return nil
}

// InvalidateByPattern 按模式清除缓存
func (m *CacheManager) InvalidateByPattern(ctx context.Context, pattern string) error {
	logger.Infof("按模式清除缓存: %s", pattern)

	if err := m.cache.DeleteByPattern(ctx, pattern); err != nil {
		logger.Errorf("按模式清除缓存失败: %v", err)
		return err
	}

	logger.Infof("按模式清除缓存成功: %s", pattern)
	return nil
}

// GetStats 获取缓存统计信息
func (m *CacheManager) GetStats(ctx context.Context) (*CacheManagerStats, error) {
	stats, err := m.cache.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	warmedUpCount := len(m.warmUpCache)
	m.mu.RUnlock()

	return &CacheManagerStats{
		CacheStats:       *stats,
		WarmedUpCount:    warmedUpCount,
		WarmUpEnabled:    m.enabled,
		KeyManagerPrefix: m.keyManager.prefix,
	}, nil
}

// IsWarmedUp 检查指定缓存是否已预热
func (m *CacheManager) IsWarmedUp(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.warmUpCache[name]
}

// ClearWarmUpCache 清除预热记录
func (m *CacheManager) ClearWarmUpCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warmUpCache = make(map[string]bool)
	logger.Infof("预热记录已清除")
}

// CacheManagerStats 缓存管理器统计信息
type CacheManagerStats struct {
	CacheStats              // 基础缓存统计
	WarmedUpCount    int    // 已预热缓存数量
	WarmUpEnabled    bool   // 是否启用预热
	KeyManagerPrefix string // 键管理器前缀
}

// BuildKey 构建缓存键（委托给 CacheKeyManager）
func (m *CacheManager) BuildKey(parts ...string) string {
	return m.keyManager.Build(parts...)
}

// BuildPattern 构建缓存键模式（委托给 CacheKeyManager）
func (m *CacheManager) BuildPattern(parts ...string) string {
	return m.keyManager.BuildPattern(parts...)
}

// Exists 检查缓存键是否存在
func (m *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	return m.cache.Exists(ctx, key)
}

// GetTTL 获取缓存键的剩余存活时间
func (m *CacheManager) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return m.cache.GetTTL(ctx, key)
}

// ==================== 预定义的预热函数 ====================
// 注意：这些函数需要导入对应的 requests 包

// WarmUpUserCache 预热用户缓存
func WarmUpUserCache(userSvc UserService) WarmUpFunc {
	return func(ctx context.Context, cache CacheProvider) error {
		// 获取所有用户列表并缓存
		params := requests.UserListParams{
			BaseListRequest: base.BaseListRequest{
				Current:  1,
				PageSize: 1000,
			},
		}
		_, err := userSvc.List(ctx, params)
		if err != nil {
			return fmt.Errorf("预热用户列表失败: %w", err)
		}
		return nil
	}
}

// WarmUpRoleCache 预热角色缓存
func WarmUpRoleCache(roleSvc RoleService) WarmUpFunc {
	return func(ctx context.Context, cache CacheProvider) error {
		// 获取所有角色列表并缓存
		params := requests.RoleListParams{
			BaseListRequest: base.BaseListRequest{
				Current:  1,
				PageSize: 1000,
			},
		}
		_, err := roleSvc.List(ctx, params)
		if err != nil {
			return fmt.Errorf("预热角色列表失败: %w", err)
		}
		return nil
	}
}

// WarmUpMenuCache 预热菜单缓存
func WarmUpMenuCache(menuSvc MenuService) WarmUpFunc {
	return func(ctx context.Context, cache CacheProvider) error {
		// 获取菜单树并缓存
		_, err := menuSvc.GetTree(ctx)
		if err != nil {
			return fmt.Errorf("预热菜单树失败: %w", err)
		}
		return nil
	}
}

// WarmUpDeptCache 预热部门缓存
func WarmUpDeptCache(deptSvc DepartmentService) WarmUpFunc {
	return func(ctx context.Context, cache CacheProvider) error {
		// 获取部门树并缓存
		_, err := deptSvc.GetTree(ctx, false)
		if err != nil {
			return fmt.Errorf("预热部门树失败: %w", err)
		}
		return nil
	}
}

// WarmUpPostCache 预热岗位缓存
func WarmUpPostCache(postSvc PostService) WarmUpFunc {
	return func(ctx context.Context, cache CacheProvider) error {
		// 获取所有岗位列表并缓存
		params := requests.PostListParams{
			BaseListRequest: base.BaseListRequest{
				Current:  1,
				PageSize: 1000,
			},
		}
		_, err := postSvc.List(ctx, params)
		if err != nil {
			return fmt.Errorf("预热岗位列表失败: %w", err)
		}
		return nil
	}
}
