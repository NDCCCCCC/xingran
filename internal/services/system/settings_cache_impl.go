package system

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// settingsCacheService 用户设置缓存服务
type settingsCacheService struct {
	*settingsService
	cache CacheProvider
	CacheServiceBase
}

// NewSettingsServiceWithCache 创建带缓存的用户设置服务
func NewSettingsServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
	configService ConfigService,
) SettingsService {
	base := &settingsService{db: db, configService: configService}
	return &settingsCacheService{
		settingsService:  base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetUserPreferences 获取用户设置（带缓存）
func (s *settingsCacheService) GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	cacheKey := fmt.Sprintf("settings:user:%s", userID)
	var result UserPreferences

	expiration := s.GetExpiration("cache.settings.user", 15*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.settingsService.GetUserPreferences(ctx, userID)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateUserPreferences 更新用户设置（带缓存失效）
func (s *settingsCacheService) UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferences) error {
	if err := s.settingsService.UpdateUserPreferences(ctx, userID, req); err != nil {
		return err
	}

	// 清除该用户的设置缓存
	cacheKey := fmt.Sprintf("settings:user:%s", userID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		// 记录日志但不影响更新操作
		applogger.Warnf("[SETTINGS_CACHE] 清除用户设置缓存失败: %v", err)
	}

	return nil
}

// InvalidateUserSettingsCache 失效用户设置缓存
func (s *settingsCacheService) InvalidateUserSettingsCache(ctx context.Context, userID string) error {
	cacheKey := fmt.Sprintf("settings:user:%s", userID)
	return s.cache.Delete(ctx, cacheKey)
}
