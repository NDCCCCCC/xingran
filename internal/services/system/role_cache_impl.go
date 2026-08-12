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

// List 查询角色列表（带缓存）
func (s *roleCacheService) List(ctx context.Context, params requests.RoleListParams) (*PageResult, error) {
	// 构建缓存键
	cacheKey := s.buildListCacheKey(params)
	var result PageResult

	// 缓存时间：30分钟
	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.List(ctx, params)
	})

	return &result, err
}

// buildListCacheKey 构建列表查询的缓存键
// 注意:必须包含所有影响结果的参数(筛选/排序/分页),否则不同请求会
// 命中同一缓存返回错误数据(历史上遗漏了 BaseListRequest 排序参数)。
func (s *roleCacheService) buildListCacheKey(params requests.RoleListParams) string {
	// 格式:cache:role:list:<roleName>:<roleKey>:<status>:<orderBy>:<isAsc>:page:<cur>:size:<size>
	baseKey := CacheKeyRoleAll
	var keyPart string

	if params.RoleName != "" {
		keyPart += ":name:" + params.RoleName
	}
	if params.RoleKey != "" {
		keyPart += ":key:" + params.RoleKey
	}
	if params.Status != "" {
		keyPart += ":status:" + params.Status
	}

	// 排序参数(必须入 key,否则不同排序命中同一缓存)
	keyPart += ":orderBy:" + params.BaseListRequest.OrderByColumn
	if params.BaseListRequest.IsAsc != nil {
		keyPart += ":isAsc:" + fmt.Sprintf("%v", *params.BaseListRequest.IsAsc)
	} else {
		keyPart += ":isAsc:default"
	}

	// 分页
	keyPart += fmt.Sprintf(":page:%d:size:%d", params.Current, params.PageSize)

	return baseKey + keyPart
}

// GetByID 获取角色详情（带缓存）
func (s *roleCacheService) GetByID(ctx context.Context, id string) (*models.Role, error) {
	cacheKey := CacheKeyRoleAll + ":id:" + id
	var result models.Role

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.GetByID(ctx, id)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllEnabled 获取所有启用的角色（覆盖基础方法，使用缓存）
func (s *roleCacheService) GetAllEnabled(ctx context.Context) ([]*models.Role, error) {
	cacheKey := CacheKeyRoleEnabled
	var result []*models.Role

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.GetAllEnabled(ctx)
	})

	return result, err
}

// GetAllEnabledWithCache 获取所有启用的角色（带缓存）- 保留以兼容性
func (s *roleCacheService) GetAllEnabledWithCache(ctx context.Context) ([]*models.Role, error) {
	cacheKey := CacheKeyRoleEnabled
	var result []*models.Role

	expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.roleService.GetAllEnabled(ctx)
	})

	return result, err
}

// GetMenusWithCache 获取角色的菜单（带缓存）
func (s *roleCacheService) GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error) {
	cacheKey := GetRoleMenusKey(roleID)
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
	cacheKey := CacheKeyRoleDepts + ":" + roleID
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
	// F-18: 含通配符 "*" 的键必须走 DeleteByPattern,否则被当字面量精确删除,失效无效。
	patterns := []string{
		CacheKeyRoleAll + "*",
		CacheKeyRoleEnabled + "*",
	}
	InvalidateCacheByPattern(ctx, s.cache, patterns, "ROLE")

	if roleID != "" {
		keys := []string{
			CacheKeyRoleMenus + ":" + roleID,
			CacheKeyRoleDepts + ":" + roleID,
			CacheKeyRoleAll + ":id:" + roleID,
		}
		InvalidateCacheByKey(ctx, s.cache, keys, "ROLE")
	}
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
	InvalidateCacheByPattern(ctx, s.cache, []string{CacheKeyRoleAll + "*"}, "ROLE")
	return nil
}

// UpdateStatus 更新角色状态（带缓存失效）
func (s *roleCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.roleService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateRoleCache(ctx, "")
}