package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// menuCacheService 菜单缓存服务
type menuCacheService struct {
	*menuService
	cache CacheProvider
	CacheServiceBase
}

// NewMenuServiceWithCache 创建带缓存的菜单服务
func NewMenuServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) MenuService {
	base := &menuService{db: db}
	return &menuCacheService{
		menuService:      base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetTree 获取菜单树（覆盖基础方法，使用缓存）
func (s *menuCacheService) GetTree(ctx context.Context) ([]models.Menu, error) {
	cacheKey := GetMenuTreeKey(false) // 默认不包含隐藏菜单
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryTree(ctx, false)
	})

	return result, err
}

// GetTreeWithCache 获取菜单树（带缓存）- 保留以兼容性
func (s *menuCacheService) GetTreeWithCache(ctx context.Context, includeHidden bool) ([]models.Menu, error) {
	cacheKey := GetMenuTreeKey(includeHidden)
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryTree(ctx, includeHidden)
	})

	return result, err
}

// queryTree 查询菜单树
func (s *menuCacheService) queryTree(ctx context.Context, includeHidden bool) ([]models.Menu, error) {
	var menus []models.Menu
	query := s.db.WithContext(ctx).Model(&models.Menu{})

	if !includeHidden {
		query = query.Where("status = ?", models.MenuStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&menus).Error; err != nil {
		return nil, err
	}

	return s.menuService.buildMenuTree(menus, nil), nil
}

// GetRouterDataWithCache 获取路由数据（带缓存）
func (s *menuCacheService) GetRouterDataWithCache(ctx context.Context) ([]models.Menu, error) {
	cacheKey := CacheKeyMenuRouter
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigMenuRouter, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryRouterData(ctx)
	})

	return result, err
}

// queryRouterData 查询路由数据
func (s *menuCacheService) queryRouterData(ctx context.Context) ([]models.Menu, error) {
	var menus []models.Menu
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return s.menuService.buildMenuTree(menus, nil), nil
}

// InvalidateMenuCache 失效菜单缓存
func (s *menuCacheService) InvalidateMenuCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{
		CacheKeyMenuTree + "*",
		CacheKeyMenuRouter + "*",
		CacheKeyMenuAll + "*",
	}, "MENU")
	return nil
}

// Create 创建菜单（带缓存失效）
func (s *menuCacheService) Create(ctx context.Context, req *requests.MenuCreateRequest) error {
	if err := s.menuService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidateMenuCache(ctx)
}

// Update 更新菜单（带缓存失效）
func (s *menuCacheService) Update(ctx context.Context, req *requests.MenuUpdateRequest) error {
	if err := s.menuService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateMenuCache(ctx)
}

// Delete 删除菜单（带缓存失效）
func (s *menuCacheService) Delete(ctx context.Context, id string, cascade bool) error {
	if err := s.menuService.Delete(ctx, id, cascade); err != nil {
		return err
	}
	return s.InvalidateMenuCache(ctx)
}

// BatchDelete 批量删除菜单（带缓存失效）
func (s *menuCacheService) BatchDelete(ctx context.Context, ids []string, cascade bool) error {
	if err := s.menuService.BatchDelete(ctx, ids, cascade); err != nil {
		return err
	}
	return s.InvalidateMenuCache(ctx)
}

// UpdateStatus 更新菜单状态（带缓存失效）
func (s *menuCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.menuService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateMenuCache(ctx)
}
