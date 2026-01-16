package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MenuCacheService 菜单缓存服务
type MenuCacheService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewMenuCacheService 创建菜单缓存服务
func NewMenuCacheService(db *gorm.DB, dataCache *DataCacheService) *MenuCacheService {
	return &MenuCacheService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetAllMenusWithCache 获取所有菜单（带缓存）
func (s *MenuCacheService) GetAllMenusWithCache(ctx context.Context) ([]*models.Menu, error) {
	cacheKey := "menu:all"
	var menus []*models.Menu

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigMenuAll, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menus, expiration, func() (interface{}, error) {
		return s.queryAllMenus()
	})

	if err != nil {
		return nil, fmt.Errorf("获取菜单列表失败: %w", err)
	}

	return menus, nil
}

// queryAllMenus 从数据库查询所有菜单（包括停用的菜单，用于管理界面）
func (s *MenuCacheService) queryAllMenus() ([]*models.Menu, error) {
	var menus []models.Menu

	// 注意：不添加 status 过滤条件，确保管理界面能看到所有菜单（包括停用的）
	err := s.db.Order("order_num ASC").Find(&menus).Error
	if err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}

	return utils.ToSlicePtr(menus), nil
}

// GetMenuTreeWithCache 获取菜单树（带缓存）
func (s *MenuCacheService) GetMenuTreeWithCache(ctx context.Context) ([]models.Menu, error) {
	cacheKey := "menu:tree"
	var menuTree []models.Menu

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigMenuTree, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menuTree, expiration, func() (interface{}, error) {
		menus, err := s.queryAllMenus()
		if err != nil {
			return nil, err
		}
		return s.buildMenuTree(menus, nil), nil
	})

	if err != nil {
		return nil, fmt.Errorf("获取菜单树失败: %w", err)
	}

	return menuTree, nil
}

// buildMenuTree 构建菜单树
func (s *MenuCacheService) buildMenuTree(menus []*models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := s.buildMenuTree(menus, &menu.ID)
			menuCopy := *menu
			if len(children) > 0 {
				menuCopy.Children = children
			}
			tree = append(tree, menuCopy)
		}
	}
	return tree
}

// GetMenuByIDWithCache 根据ID获取菜单（带缓存）
func (s *MenuCacheService) GetMenuByIDWithCache(ctx context.Context, menuID string) (*models.Menu, error) {
	if menuID == "" {
		return nil, fmt.Errorf("菜单ID不能为空")
	}

	cacheKey := fmt.Sprintf("menu:id:%s", menuID)
	var menu models.Menu

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menu, 30*time.Minute, func() (interface{}, error) {
		return s.queryMenuByID(menuID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取菜单失败: %w", err)
	}

	return &menu, nil
}

// queryMenuByID 从数据库查询菜单
func (s *MenuCacheService) queryMenuByID(menuID string) (*models.Menu, error) {
	var menu models.Menu

	if err := s.db.Where("id = ?", menuID).First(&menu).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("菜单不存在")
		}
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}

	return &menu, nil
}

// GetMenuChildrenWithCache 获取子菜单（带缓存）
func (s *MenuCacheService) GetMenuChildrenWithCache(ctx context.Context, parentID string) ([]*models.Menu, error) {
	cacheKey := fmt.Sprintf("menu:children:%s", parentID)
	var children []*models.Menu

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &children, 30*time.Minute, func() (interface{}, error) {
		return s.queryMenuChildren(parentID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取子菜单失败: %w", err)
	}

	return children, nil
}

// queryMenuChildren 从数据库查询子菜单
func (s *MenuCacheService) queryMenuChildren(parentID string) ([]*models.Menu, error) {
	var menus []models.Menu

	err := s.db.Where("parent_id = ? AND status = ?", parentID, models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error
	if err != nil {
		return nil, fmt.Errorf("查询子菜单失败: %w", err)
	}

	return utils.ToSlicePtr(menus), nil
}

// InvalidateMenuCache 清除菜单相关缓存
func (s *MenuCacheService) InvalidateMenuCache(ctx context.Context) error {
	patterns := []string{
		"menu:all",
		"menu:tree",
		"menu:routers",
		"menu:id:*",
		"menu:children:*",
		"menu:perms:*", // 清除按权限标识的菜单缓存
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateMenuCacheByID 清除指定菜单的缓存
func (s *MenuCacheService) InvalidateMenuCacheByID(ctx context.Context, menuID string) error {
	patterns := []string{
		"menu:all",
		"menu:tree",
		fmt.Sprintf("menu:id:%s", menuID),
	}

	// 如果有父菜单，也要清除父菜单的子菜单缓存
	menu, err := s.queryMenuByID(menuID)
	if err == nil && menu.ParentID != nil {
		patterns = append(patterns, fmt.Sprintf("menu:children:%s", *menu.ParentID))
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// CreateMenu 创建菜单
func (s *MenuCacheService) CreateMenu(ctx context.Context, menu *models.Menu) error {
	if err := s.db.Create(menu).Error; err != nil {
		return fmt.Errorf("创建菜单失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateMenuCache(ctx); err != nil {
		logger.Warnf("[MENU] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateMenu 更新菜单
func (s *MenuCacheService) UpdateMenu(ctx context.Context, menu *models.Menu) error {
	if err := s.db.Save(menu).Error; err != nil {
		return fmt.Errorf("更新菜单失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateMenuCacheByID(ctx, menu.ID); err != nil {
		logger.Warnf("[MENU] 清除缓存失败: %v", err)
	}

	return nil
}

// DeleteMenu 删除菜单
func (s *MenuCacheService) DeleteMenu(ctx context.Context, menuID string) error {
	if err := s.db.Delete(&models.Menu{}, "id = ?", menuID).Error; err != nil {
		return fmt.Errorf("删除菜单失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateMenuCacheByID(ctx, menuID); err != nil {
		logger.Warnf("[MENU] 清除缓存失败: %v", err)
	}

	return nil
}

// GetRouters 获取路由菜单（用于前端路由）
func (s *MenuCacheService) GetRouters(ctx context.Context) ([]models.Menu, error) {
	cacheKey := "menu:routers"
	var routers []models.Menu

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigMenuRouter, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &routers, expiration, func() (interface{}, error) {
		return s.queryRouters()
	})

	if err != nil {
		return nil, fmt.Errorf("获取路由菜单失败: %w", err)
	}

	return routers, nil
}

// queryRouters 从数据库查询路由菜单
func (s *MenuCacheService) queryRouters() ([]models.Menu, error) {
	var menus []models.Menu

	// 只查询菜单类型为目录或菜单的，且状态为正常的
	if err := s.db.Where("menu_type IN ? AND status = ? AND visible = ?",
		[]models.MenuType{models.MenuTypeDir, models.MenuTypeMenu},
		models.MenuStatusNormal,
		models.VisibleShow).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询路由菜单失败: %w", err)
	}

	return s.buildMenuTreeFromValues(menus, nil), nil
}

// buildMenuTreeFromValues 从值类型构建菜单树
func (s *MenuCacheService) buildMenuTreeFromValues(menus []models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := s.buildMenuTreeFromValues(menus, &menu.ID)
			menuCopy := menu
			if len(children) > 0 {
				menuCopy.Children = children
			}
			tree = append(tree, menuCopy)
		}
	}
	return tree
}

// GetMenuByPerms 根据权限标识获取菜单（带缓存）
func (s *MenuCacheService) GetMenuByPermsWithCache(ctx context.Context, perms string) (*models.Menu, error) {
	if perms == "" {
		return nil, fmt.Errorf("权限标识不能为空")
	}

	cacheKey := fmt.Sprintf("menu:perms:%s", perms)
	var menu models.Menu

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menu, 30*time.Minute, func() (interface{}, error) {
		return s.queryMenuByPerms(perms)
	})

	if err != nil {
		return nil, fmt.Errorf("获取菜单失败: %w", err)
	}

	return &menu, nil
}

// queryMenuByPerms 从数据库查询菜单
func (s *MenuCacheService) queryMenuByPerms(perms string) (*models.Menu, error) {
	var menu models.Menu

	if err := s.db.Where("perms = ? AND status = ?", perms, models.MenuStatusNormal).
		First(&menu).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("菜单不存在")
		}
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}

	return &menu, nil
}
