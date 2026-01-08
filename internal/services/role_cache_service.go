package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// RoleCacheService 角色缓存服务
type RoleCacheService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewRoleCacheService 创建角色缓存服务
func NewRoleCacheService(db *gorm.DB, dataCache *DataCacheService) *RoleCacheService {
	return &RoleCacheService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetRoleMenusWithCache 获取角色菜单权限（带缓存）
func (s *RoleCacheService) GetRoleMenusWithCache(ctx context.Context, roleID string) ([]*models.Menu, error) {
	if roleID == "" {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	cacheKey := GetRoleMenusKey(roleID)
	var menus []*models.Menu

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menus, 30*time.Minute, func() (interface{}, error) {
		return s.queryRoleMenus(roleID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取角色菜单失败: %w", err)
	}

	return menus, nil
}

// queryRoleMenus 从数据库查询角色菜单权限
func (s *RoleCacheService) queryRoleMenus(roleID string) ([]*models.Menu, error) {
	var menuIDs []string

	// 查询角色关联的菜单ID
	if err := s.db.Table("sys_role_menu").
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, fmt.Errorf("查询角色菜单关联失败: %w", err)
	}

	if len(menuIDs) == 0 {
		return []*models.Menu{}, nil
	}

	// 查询菜单详情
	var menus []models.Menu
	if err := s.db.Where("id IN ? AND status = ?", menuIDs, models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单详情失败: %w", err)
	}

	// 转换为指针数组
	menuPtrs := make([]*models.Menu, len(menus))
	for i := range menus {
		menuPtrs[i] = &menus[i]
	}

	return menuPtrs, nil
}

// GetRoleMenuTreeWithCache 获取角色菜单树（带缓存）
func (s *RoleCacheService) GetRoleMenuTreeWithCache(ctx context.Context, roleID string) ([]models.Menu, error) {
	if roleID == "" {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	cacheKey := fmt.Sprintf("%s:%s", CacheKeyRoleMenuTree, roleID)
	var menuTree []models.Menu

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &menuTree, 30*time.Minute, func() (interface{}, error) {
		menus, err := s.queryRoleMenus(roleID)
		if err != nil {
			return nil, err
		}
		return s.buildMenuTree(menus, nil), nil
	})

	if err != nil {
		return nil, fmt.Errorf("获取角色菜单树失败: %w", err)
	}

	return menuTree, nil
}

// buildMenuTree 构建菜单树
func (s *RoleCacheService) buildMenuTree(menus []*models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := s.buildMenuTree(menus, &menu.ID)
			menuCopy := *menu // 复制值
			if len(children) > 0 {
				menuCopy.Children = children
			}
			tree = append(tree, menuCopy)
		}
	}
	return tree
}

// GetRoleDeptsWithCache 获取角色部门权限（带缓存）
func (s *RoleCacheService) GetRoleDeptsWithCache(ctx context.Context, roleID string) ([]*models.Department, error) {
	if roleID == "" {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	cacheKey := fmt.Sprintf("%s:%s", CacheKeyRoleDepts, roleID)
	var depts []*models.Department

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &depts, 30*time.Minute, func() (interface{}, error) {
		return s.queryRoleDepts(roleID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取角色部门失败: %w", err)
	}

	return depts, nil
}

// queryRoleDepts 从数据库查询角色部门权限
func (s *RoleCacheService) queryRoleDepts(roleID string) ([]*models.Department, error) {
	var deptIDs []string

	// 查询角色关联的部门ID
	if err := s.db.Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", &deptIDs).Error; err != nil {
		return nil, fmt.Errorf("查询角色部门关联失败: %w", err)
	}

	if len(deptIDs) == 0 {
		return []*models.Department{}, nil
	}

	// 查询部门详情
	var depts []models.Department
	if err := s.db.Where("id IN ? AND status = ?", deptIDs, models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门详情失败: %w", err)
	}

	// 转换为指针数组
	deptPtrs := make([]*models.Department, len(depts))
	for i := range depts {
		deptPtrs[i] = &depts[i]
	}

	return deptPtrs, nil
}

// InvalidateRoleCache 清除角色相关缓存
func (s *RoleCacheService) InvalidateRoleCache(ctx context.Context, roleID string) error {
	// 清除指定角色的缓存
	patterns := []string{
		fmt.Sprintf("%s:%s*", CacheKeyRoleMenus, roleID),
		fmt.Sprintf("%s:%s*", CacheKeyRoleMenuTree, roleID),
		fmt.Sprintf("%s:%s*", CacheKeyRoleDepts, roleID),
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateAllRoleCache 清除所有角色相关缓存
func (s *RoleCacheService) InvalidateAllRoleCache(ctx context.Context) error {
	patterns := []string{
		CacheKeyRoleMenus + "*",
		CacheKeyRoleMenuTree + "*",
		CacheKeyRoleDepts + "*",
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// AssignRoleMenus 分配角色菜单权限
func (s *RoleCacheService) AssignRoleMenus(ctx context.Context, roleID string, menuIDs []string) error {
	// 使用事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧的菜单关联
	if err := tx.Table("sys_role_menu").Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧菜单关联失败: %w", err)
	}

	// 添加新的菜单关联
	if len(menuIDs) > 0 {
		var roleMenus []models.RoleMenu
		for _, menuID := range menuIDs {
			roleMenus = append(roleMenus, models.RoleMenu{
				RoleID: roleID,
				MenuID: menuID,
			})
		}
		if err := tx.Create(&roleMenus).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("添加菜单关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateRoleCache(ctx, roleID); err != nil {
		logger.Warnf("[ROLE] 清除缓存失败: %v", err)
	}

	return nil
}

// AssignRoleDepts 分配角色部门权限
func (s *RoleCacheService) AssignRoleDepts(ctx context.Context, roleID string, deptIDs []string) error {
	// 使用事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧的部门关联
	if err := tx.Table("sys_role_dept").Where("role_id = ?", roleID).Delete(&models.RoleDept{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧部门关联失败: %w", err)
	}

	// 添加新的部门关联
	if len(deptIDs) > 0 {
		var roleDepts []models.RoleDept
		for _, deptID := range deptIDs {
			roleDepts = append(roleDepts, models.RoleDept{
				RoleID: roleID,
				DeptID: deptID,
			})
		}
		if err := tx.Create(&roleDepts).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("添加部门关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateRoleCache(ctx, roleID); err != nil {
		logger.Warnf("[ROLE] 清除缓存失败: %v", err)
	}

	return nil
}
