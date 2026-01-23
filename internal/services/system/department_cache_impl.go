package system

import (
	"context"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"gorm.io/gorm"
)

// departmentCacheService 部门缓存服务
type departmentCacheService struct {
	*departmentService
	cache CacheProvider
	CacheServiceBase
}

// NewDepartmentServiceWithCache 创建带缓存的部门服务
func NewDepartmentServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) DepartmentService {
	base := &departmentService{db: db}
	return &departmentCacheService{
		departmentService: base,
		cache:             cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetTree 获取部门树（覆盖基础方法，使用缓存）
func (s *departmentCacheService) GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	cacheKey := services.GetDeptTreeKey()
	if includeDisabled {
		cacheKey += ":all"
	}
	var result []*models.Department

	expiration := s.GetExpiration(services.CacheConfigDeptTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryTree(ctx, includeDisabled)
	})

	return result, err
}

// GetTreeWithCache 获取部门树（带缓存）- 保留以兼容性
func (s *departmentCacheService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	cacheKey := services.GetDeptTreeKey()
	if includeDisabled {
		cacheKey += ":all"
	}
	var result []*models.Department

	expiration := s.GetExpiration(services.CacheConfigDeptTree, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryTree(ctx, includeDisabled)
	})

	return result, err
}

// queryTree 查询部门树
func (s *departmentCacheService) queryTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if !includeDisabled {
		query = query.Where("status = ?", models.DeptStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, err
	}

	return s.departmentService.buildDeptTree(depts, nil), nil
}

// GetSelectDataWithCache 获取选择器数据（带缓存）
func (s *departmentCacheService) GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error) {
	cacheKey := services.CacheKeyDeptTreeSelect
	var result []*models.Department

	expiration := s.GetExpiration(services.CacheConfigDeptSelect, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.querySelectData(ctx)
	})

	return result, err
}

// querySelectData 查询选择器数据
func (s *departmentCacheService) querySelectData(ctx context.Context) ([]*models.Department, error) {
	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, err
	}

	return s.departmentService.buildDeptTree(depts, nil), nil
}

// InvalidateDeptCache 失效部门缓存
func (s *departmentCacheService) InvalidateDeptCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{
		services.CacheKeyDeptTree + "*",
		services.CacheKeyDeptList + "*",
		services.CacheKeyDeptTreeSelect + "*",
	}, "DEPT")
	return nil
}

// Create 创建部门（带缓存失效）
func (s *departmentCacheService) Create(ctx context.Context, req *DepartmentCreateRequest) error {
	if err := s.departmentService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

// Update 更新部门（带缓存失效）
func (s *departmentCacheService) Update(ctx context.Context, req *DepartmentUpdateRequest) error {
	if err := s.departmentService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

// Delete 删除部门（带缓存失效）
func (s *departmentCacheService) Delete(ctx context.Context, id string) error {
	if err := s.departmentService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

// BatchDelete 批量删除部门（带缓存失效）
func (s *departmentCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.departmentService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

// UpdateStatus 更新部门状态（带缓存失效）
func (s *departmentCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.departmentService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}
