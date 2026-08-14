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

// GetUserMenus 获取用户的菜单列表（覆盖基础方法，使用缓存）
// 缓存键按 userID 隔离（GetMenuUserMenusKey），TTL 复用 CacheConfigMenuTree（菜单数据稳定）；
// query 闭包委托嵌入式 menuService.GetUserMenus（cache miss 时回源）。
// 返回值 []models.Menu 含嵌套 Children，与 GetTree 缓存形状一致，json round-trip 已验证。
func (s *menuCacheService) GetUserMenus(ctx context.Context, userID string) ([]models.Menu, error) {
	cacheKey := GetMenuUserMenusKey(userID)
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.menuService.GetUserMenus(ctx, userID)
	})

	return result, err
}

// GetAllUserMenus 获取用户的所有菜单（含隐藏）（覆盖基础方法，使用缓存）
// 缓存键按 userID 隔离（GetMenuUserAllMenusKey），TTL 复用 CacheConfigMenuTree。
func (s *menuCacheService) GetAllUserMenus(ctx context.Context, userID string) ([]models.Menu, error) {
	cacheKey := GetMenuUserAllMenusKey(userID)
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.menuService.GetAllUserMenus(ctx, userID)
	})

	return result, err
}

// GetUserPermissions 获取用户的权限列表（覆盖基础方法，使用缓存）
// 缓存键按 userID 隔离（GetMenuUserPermissionsKey），TTL 复用 CacheConfigMenuTree。
// 安全性说明：此缓存仅服务于 /system/my-menus/permissions 接口（前端按钮级 UI 控制）。
// 后端 RBAC 权限校验走 pkg/permission/service.go 的实时未缓存路径
// （pkg/middleware/permission.go、internal/middleware/apikey.go），不受此缓存影响，
// 因此 30min TTL 只影响前端权限标识展示的时效，不构成提权窗口。
// 角色↔菜单 / 用户↔角色变更时经 InvalidateUserMenuCache 主动失效（F-01），TTL 仅为兜底。
func (s *menuCacheService) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	cacheKey := GetMenuUserPermissionsKey(userID)
	var result []string

	expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.menuService.GetUserPermissions(ctx, userID)
	})

	return result, err
}

// InvalidateMenuCache 失效菜单缓存
// pattern 列表覆盖全部 6 个 menu: 命名空间下的缓存键前缀（含 3 个 user-scoped）；
// 菜单任意写操作（Create/Update/Delete/BatchDelete/UpdateStatus）均已调用此方法。
// 新 user-scoped key 用 ":*" 精确匹配（避免误伤同名前缀），既有 3 个保持 "*"(向后兼容)。
func (s *menuCacheService) InvalidateMenuCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{
		CacheKeyMenuTree + "*",
		CacheKeyMenuRouter + "*",
		CacheKeyMenuAll + "*",
		CacheKeyMenuUserMenus + ":*",
		CacheKeyMenuUserAllMenus + ":*",
		CacheKeyMenuUserPermissions + ":*",
	}, "MENU")
	return nil
}

// userMenuCacheInvalidationPatterns user-scoped 菜单缓存键前缀列表。
// 供 menuCacheService.InvalidateUserMenuCache 与 role/user 缓存服务复用，
// 避免 3 个前缀在多处硬编码后漂移（F-01）。
func userMenuCacheInvalidationPatterns() []string {
	return []string{
		CacheKeyMenuUserMenus + ":*",
		CacheKeyMenuUserAllMenus + ":*",
		CacheKeyMenuUserPermissions + ":*",
	}
}

// InvalidateUserMenuCacheByProvider 失效 user-scoped 菜单缓存（包级 helper）。
// role/user 缓存服务与 menu 服务在 core.go 中共享同一 CacheProvider，
// 因此直接用各自持有的 provider 清这 3 个前缀即可，无需互相注入 service 依赖
// （避免构造函数签名连锁改动）。
func InvalidateUserMenuCacheByProvider(ctx context.Context, cache CacheProvider) {
	InvalidateCacheByPattern(ctx, cache, userMenuCacheInvalidationPatterns(), "MENU-USER")
}

// InvalidateUserMenuCache 失效 user-scoped 菜单缓存（角色↔菜单 / 用户↔角色变更时调用）
// 只清 menu:user:* 三个前缀，不动全局 menu:tree/router/all（菜单本体未变，无需连带失效）。
func (s *menuCacheService) InvalidateUserMenuCache(ctx context.Context) error {
	InvalidateUserMenuCacheByProvider(ctx, s.cache)
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
