package system

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// configCacheService 系统配置缓存服务
type configCacheService struct {
	*configService
	cache CacheProvider
	CacheServiceBase
}

// NewConfigServiceWithCache 创建带缓存的系统配置服务
func NewConfigServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) ConfigService {
	base := &configService{db: db}
	return &configCacheService{
		configService:    base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetByID 获取配置详情（带缓存）
func (s *configCacheService) GetByID(ctx context.Context, id string) (*models.Config, error) {
	cacheKey := fmt.Sprintf("config:id:%s", id)
	var result models.Config

	expiration := s.GetExpiration(services.CacheConfigConfigByID, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.configService.GetByID(ctx, id)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByKey 根据配置键获取配置（带缓存）
func (s *configCacheService) GetByKey(ctx context.Context, configKey string) (*models.Config, error) {
	cacheKey := fmt.Sprintf("config:key:%s", configKey)
	var result models.Config

	expiration := s.GetExpiration(services.CacheConfigConfigByKey, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.configService.GetByKey(ctx, configKey)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllConfigs 获取所有配置（带缓存）
func (s *configCacheService) GetAllConfigs(ctx context.Context) ([]models.Config, error) {
	cacheKey := "config:all"
	var result []models.Config

	expiration := s.GetExpiration("cache.config.all", 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryAllConfigs(ctx)
	})

	return result, err
}

// queryAllConfigs 查询所有配置
func (s *configCacheService) queryAllConfigs(ctx context.Context) ([]models.Config, error) {
	var configs []models.Config
	err := s.db.WithContext(ctx).Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("查询所有配置失败: %w", err)
	}
	return configs, nil
}

// InvalidateConfigCache 失效配置缓存
func (s *configCacheService) InvalidateConfigCache(ctx context.Context, configKey string) error {
	keys := []string{
		"config:all",
		fmt.Sprintf("config:key:%s", configKey),
	}
	InvalidateCacheByKey(ctx, s.cache, keys, "CONFIG")
	return nil
}

// InvalidateAllConfigCache 失效所有配置缓存
func (s *configCacheService) InvalidateAllConfigCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{"config:*"}, "CONFIG")
	return nil
}

// Create 创建配置（带缓存失效）
func (s *configCacheService) Create(ctx context.Context, req *requests.ConfigCreateRequest) error {
	if err := s.configService.Create(ctx, req); err != nil {
		return err
	}
	// 清除所有配置缓存
	return s.InvalidateAllConfigCache(ctx)
}

// Update 更新配置（带缓存失效）
func (s *configCacheService) Update(ctx context.Context, req *requests.ConfigUpdateRequest) error {
	if err := s.configService.Update(ctx, req); err != nil {
		return err
	}
	// 清除所有配置缓存（因为可能影响键值映射）
	return s.InvalidateAllConfigCache(ctx)
}

// Delete 删除配置（带缓存失效）
func (s *configCacheService) Delete(ctx context.Context, id string) error {
	// 先获取配置以便获取 configKey
	config, err := s.configService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.configService.Delete(ctx, id); err != nil {
		return err
	}

	// 清除缓存
	return s.InvalidateConfigCache(ctx, config.ConfigKey)
}

// BatchDelete 批量删除配置（带缓存失效）
func (s *configCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.configService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	// 清除所有配置缓存
	return s.InvalidateAllConfigCache(ctx)
}

// List 查询配置列表（不使用缓存，因为查询条件多变）
func (s *configCacheService) List(ctx context.Context, params requests.ConfigListParams) (*PageResult, error) {
	return s.configService.List(ctx, params)
}

// RefreshCache 刷新缓存
func (s *configCacheService) RefreshCache(ctx context.Context) error {
	return s.InvalidateAllConfigCache(ctx)
}
