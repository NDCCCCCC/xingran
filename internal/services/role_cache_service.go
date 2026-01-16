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

// GetAllRolesWithCache 获取所有启用的角色（带缓存）
func (s *RoleCacheService) GetAllRolesWithCache(ctx context.Context) ([]*models.Role, error) {
	cacheKey := "role:all:enabled"
	var roles []*models.Role

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigRoleMenus, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &roles, expiration, func() (interface{}, error) {
		return s.queryAllRoles()
	})

	if err != nil {
		return nil, fmt.Errorf("获取角色列表失败: %w", err)
	}

	return roles, nil
}

// queryAllRoles 从数据库查询所有启用的角色
func (s *RoleCacheService) queryAllRoles() ([]*models.Role, error) {
	var roles []models.Role

	err := s.db.Where("status = ?", models.RoleStatusEnabled).
		Order("role_sort ASC, created_at DESC").
		Find(&roles).Error

	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}

	return utils.ToSlicePtr(roles), nil
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

	err := s.db.Table("sys_role_menu").
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error
	if err != nil {
		return nil, fmt.Errorf("查询角色菜单关联失败: %w", err)
	}

	if len(menuIDs) == 0 {
		return []*models.Menu{}, nil
	}

	var menus []models.Menu
	err = s.db.Where("id IN ? AND status = ?", menuIDs, models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error
	if err != nil {
		return nil, fmt.Errorf("查询菜单详情失败: %w", err)
	}

	return utils.ToSlicePtr(menus), nil
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

	err := s.db.Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", &deptIDs).Error
	if err != nil {
		return nil, fmt.Errorf("查询角色部门关联失败: %w", err)
	}

	if len(deptIDs) == 0 {
		return []*models.Department{}, nil
	}

	var depts []models.Department
	err = s.db.Where("id IN ? AND status = ?", deptIDs, models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error
	if err != nil {
		return nil, fmt.Errorf("查询部门详情失败: %w", err)
	}

	return utils.ToSlicePtr(depts), nil
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
		"role:all:enabled",         // 角色列表缓存
		CacheKeyRoleMenus + "*",    // 角色菜单缓存
		CacheKeyRoleMenuTree + "*", // 角色菜单树缓存
		CacheKeyRoleDepts + "*",    // 角色部门缓存
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
	if err := s.assignRoleRelations(ctx, roleID, menuIDs, "sys_role_menu", &models.RoleMenu{}, func(id string) interface{} {
		return &models.RoleMenu{RoleID: roleID, MenuID: id}
	}); err != nil {
		return err
	}

	// 清除缓存
	if err := s.InvalidateRoleCache(ctx, roleID); err != nil {
		logger.Warnf("[ROLE] 清除缓存失败: %v", err)
	}

	return nil
}

// AssignRoleDepts 分配角色部门权限
func (s *RoleCacheService) AssignRoleDepts(ctx context.Context, roleID string, deptIDs []string) error {
	if err := s.assignRoleRelations(ctx, roleID, deptIDs, "sys_role_dept", &models.RoleDept{}, func(id string) interface{} {
		return &models.RoleDept{RoleID: roleID, DeptID: id}
	}); err != nil {
		return err
	}

	// 清除缓存
	if err := s.InvalidateRoleCache(ctx, roleID); err != nil {
		logger.Warnf("[ROLE] 清除缓存失败: %v", err)
	}

	return nil
}

// assignRoleRelations 通用的角色关联分配函数
func (s *RoleCacheService) assignRoleRelations(_ context.Context, roleID string, ids []string, tableName string, model interface{}, relationFactory func(string) interface{}) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除旧关联
		if err := tx.Table(tableName).Where("role_id = ?", roleID).Delete(model).Error; err != nil {
			return fmt.Errorf("删除旧关联失败: %w", err)
		}

		// 添加新关联
		if len(ids) > 0 {
			relations := make([]interface{}, len(ids))
			for i, id := range ids {
				relations[i] = relationFactory(id)
			}
			if err := tx.Table(tableName).Create(&relations).Error; err != nil {
				return fmt.Errorf("添加关联失败: %w", err)
			}
		}

		return nil
	})
}

