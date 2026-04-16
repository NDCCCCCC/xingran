package system

import (
	"context"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"gorm.io/gorm"
)

// roleCacheService 角色缓存服务
type roleCacheService struct {
	*roleService
	cache CacheProvider
	CacheServiceBase
}

// NewRoleServiceWithCache 创建带缓存的角色服务
func NewRoleServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) RoleService {
	base := &roleService{db: db}
	return &roleCacheService{
		roleService:      base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetAllEnabled 获取所有启用的角色（覆盖基础方法，使用缓存）
func (s *roleCacheService) GetAllEnabled(ctx context.Context) ([]*models.Role, error) {
	cacheKey := services.CacheKeyRoleEnabled
	var result []*models.Role

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.GetAllEnabled(ctx)
	})

	return result, err
}

// GetAllEnabledWithCache 获取所有启用的角色（带缓存）- 保留以兼容性
func (s *roleCacheService) GetAllEnabledWithCache(ctx context.Context) ([]*models.Role, error) {
	cacheKey := services.CacheKeyRoleEnabled
	var result []*models.Role

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.GetAllEnabled(ctx)
	})

	return result, err
}

// GetMenusWithCache 获取角色的菜单（带缓存）
func (s *roleCacheService) GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error) {
	cacheKey := services.GetRoleMenusKey(roleID)
	var result []models.Menu

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryMenus(ctx, roleID)
	})

	return result, err
}

// queryMenus 查询角色的菜单
func (s *roleCacheService) queryMenus(ctx context.Context, roleID string) ([]models.Menu, error) {
	var menuIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_role_menu").
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, err
	}

	if len(menuIDs) == 0 {
		return []models.Menu{}, nil
	}

	var menus []models.Menu
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", menuIDs, models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// GetDeptsWithCache 获取角色的部门（带缓存）
func (s *roleCacheService) GetDeptsWithCache(ctx context.Context, roleID string) ([]models.Department, error) {
	cacheKey := services.CacheKeyRoleDepts + ":" + roleID
	var result []models.Department

	expiration := s.GetExpiration(services.CacheConfigRoleDepts, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryDepts(ctx, roleID)
	})

	return result, err
}

// queryDepts 查询角色的部门
func (s *roleCacheService) queryDepts(ctx context.Context, roleID string) ([]models.Department, error) {
	var deptIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", &deptIDs).Error; err != nil {
		return nil, err
	}

	if len(deptIDs) == 0 {
		return []models.Department{}, nil
	}

	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", deptIDs, models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, err
	}

	return depts, nil
}

// InvalidateRoleCache 失效角色缓存
func (s *roleCacheService) InvalidateRoleCache(ctx context.Context, roleID string) error {
	keys := []string{
		services.CacheKeyRoleAll + "*",
		services.CacheKeyRoleEnabled + "*",
	}
	if roleID != "" {
		keys = append(keys,
			services.CacheKeyRoleMenus+":"+roleID,
			services.CacheKeyRoleDepts+":"+roleID,
		)
	}
	InvalidateCacheByKey(ctx, s.cache, keys, "ROLE")
	return nil
}

// Create 创建角色（带缓存失效）
func (s *roleCacheService) Create(ctx context.Context, req *requests.RoleCreateRequest) error {
	if err := s.roleService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidateRoleCache(ctx, "")
}

// Update 更新角色（带缓存失效）
func (s *roleCacheService) Update(ctx context.Context, req *requests.RoleUpdateRequest) error {
	if err := s.roleService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateRoleCache(ctx, req.ID)
}

// Delete 删除角色（带缓存失效）
func (s *roleCacheService) Delete(ctx context.Context, id string) error {
	if err := s.roleService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidateRoleCache(ctx, id)
}

// BatchDelete 批量删除角色（带缓存失效）
func (s *roleCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.roleService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	// 清除所有角色相关缓存
	InvalidateCacheByPattern(ctx, s.cache, []string{services.CacheKeyRoleAll + "*"}, "ROLE")
	return nil
}

// UpdateStatus 更新角色状态（带缓存失效）
func (s *roleCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.roleService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateRoleCache(ctx, "")
}
