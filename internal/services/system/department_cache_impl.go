package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

type departmentCacheService struct {
	*departmentService
	cache CacheProvider
	CacheServiceBase
}

func NewDepartmentServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) DepartmentService {
	base := &departmentService{db: db}
	return &departmentCacheService{
		departmentService: base,
		cache:             cache,
		CacheServiceBase:  CacheServiceBase{Config: config},
	}
}

func (s *departmentCacheService) GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTreeWithFilter(ctx, includeDisabled, requests.DepartmentListParams{})
}

func (s *departmentCacheService) GetTreeWithFilter(ctx context.Context, includeDisabled bool, params requests.DepartmentListParams) ([]*models.Department, error) {
	if params.DeptName != "" || params.Status != nil {
		return s.departmentService.GetTreeWithFilter(ctx, includeDisabled, params)
	}

	cacheKey := BuildDeptCacheKey(CacheKeyDeptTree)
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

func (s *departmentCacheService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTree(ctx, includeDisabled)
}

func (s *departmentCacheService) queryTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if !includeDisabled {
		query = query.Where("status = ?", models.DeptStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, err
	}

	s.departmentService.fillLeaderInfo(ctx, depts)

	return s.departmentService.buildDeptTree(depts, nil), nil
}

func (s *departmentCacheService) GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error) {
	cacheKey := CacheKeyDeptTree
	var result []*models.Department

	expiration := s.GetExpiration(services.CacheConfigDeptSelect, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.querySelectData(ctx)
	})

	return result, err
}

func (s *departmentCacheService) querySelectData(ctx context.Context) ([]*models.Department, error) {
	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, err
	}

	s.departmentService.fillLeaderInfo(ctx, depts)

	return s.departmentService.buildDeptTree(depts, nil), nil
}

func (s *departmentCacheService) InvalidateDeptCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{
		BuildDeptCacheKey(CacheKeyDeptTree) + "*",
		BuildDeptCacheKey(CacheKeyDeptList) + "*",
		BuildDeptCacheKey("tree:select") + "*",
	}, "DEPT")
	return nil
}

func (s *departmentCacheService) Create(ctx context.Context, req *requests.DepartmentCreateRequest) error {
	if err := s.departmentService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error {
	if err := s.departmentService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) Delete(ctx context.Context, id string) error {
	if err := s.departmentService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.departmentService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.departmentService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) GetDB() *gorm.DB {
	return s.departmentService.GetDB()
}
