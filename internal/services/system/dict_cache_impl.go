package system

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// dictTypeCacheService 字典类型缓存服务
type dictTypeCacheService struct {
	*dictTypeService
	cache CacheProvider
	CacheServiceBase
}

// NewDictTypeServiceWithCache 创建带缓存的字典类型服务
func NewDictTypeServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) DictTypeService {
	base := &dictTypeService{db: db}
	return &dictTypeCacheService{
		dictTypeService:  base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetAllWithCache 获取所有字典类型（覆盖基础方法，使用缓存）
func (s *dictTypeCacheService) GetAllWithCache(ctx context.Context) ([]*models.DictType, error) {
	cacheKey := CacheKeyDictType
	var result []*models.DictType

	expiration := s.GetExpiration(services.CacheConfigDictType, 60*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryAllTypes(ctx)
	})

	if err != nil {
		return nil, fmt.Errorf("获取字典类型失败: %w", err)
	}

	return result, nil
}

// queryAllTypes 查询所有字典类型
func (s *dictTypeCacheService) queryAllTypes(ctx context.Context) ([]*models.DictType, error) {
	var types []models.DictType
	if err := s.db.WithContext(ctx).Where("status = ?", 0).
		Order("id ASC").Find(&types).Error; err != nil {
		return nil, err
	}
	result := make([]*models.DictType, len(types))
	for i := range types {
		result[i] = &types[i]
	}
	return result, nil
}

// Create 创建字典类型（带缓存失效）
func (s *dictTypeCacheService) Create(ctx context.Context, req *requests.DictTypeCreateRequest) error {
	if err := s.dictTypeService.Create(ctx, req); err != nil {
		return err
	}
	s.invalidateCache(ctx)
	return nil
}

// Update 更新字典类型（带缓存失效）
func (s *dictTypeCacheService) Update(ctx context.Context, req *requests.DictTypeUpdateRequest) error {
	if err := s.dictTypeService.Update(ctx, req); err != nil {
		return err
	}
	s.invalidateCache(ctx)
	return nil
}

// Delete 删除字典类型（带缓存失效）
func (s *dictTypeCacheService) Delete(ctx context.Context, id string) error {
	if err := s.dictTypeService.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateCache(ctx)
	return nil
}

// invalidateCache 失效缓存
func (s *dictTypeCacheService) invalidateCache(ctx context.Context) {
	InvalidateCacheByPattern(ctx, s.cache, []string{CacheKeyDictType + "*"}, "DICT")
}

// List 查询字典类型列表（全量缓存+内存筛选版本）
func (s *dictTypeCacheService) List(ctx context.Context, params requests.DictTypeListParams) (*PageResult, error) {
	allTypes, err := s.GetAllWithCache(ctx)
	if err != nil {
		return nil, err
	}

	filtered := s.filterDictTypes(allTypes, params)
	filtered = sortDictTypes(filtered, params.OrderByColumn, params.IsAsc)
	paged, total := paginate(filtered, params.Current, params.PageSize)

	return &PageResult{
		List:     paged,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// filterDictTypes 在内存中筛选字典类型
func (s *dictTypeCacheService) filterDictTypes(types []*models.DictType, params requests.DictTypeListParams) []*models.DictType {
	result := types

	if params.DictName != nil && *params.DictName != "" {
		result = filterSlice(result, func(t *models.DictType) bool {
			return contains(t.DictName, *params.DictName)
		})
	}

	if params.DictType != nil && *params.DictType != "" {
		result = filterSlice(result, func(t *models.DictType) bool {
			return contains(t.DictType, *params.DictType)
		})
	}

	if params.Status != nil {
		result = filterSlice(result, func(t *models.DictType) bool {
			return t.Status == *params.Status
		})
	}

	return result
}

// sortDictTypes 内存排序字典类型（缓存版 List 用，field 对应前端 sorterMeta 白名单）。
// 缓存版 List 不走数据库 ORDER BY，故在内存中对筛选结果排序，保证排序参数生效。
func sortDictTypes(types []*models.DictType, orderByColumn string, isAsc *bool) []*models.DictType {
	if orderByColumn == "" {
		return types
	}
	asc := isAsc == nil || *isAsc
	sort.SliceStable(types, func(i, j int) bool {
		a, b := types[i], types[j]
		var less bool
		switch orderByColumn {
		case "dictName":
			less = a.DictName < b.DictName
		case "dictType":
			less = a.DictType < b.DictType
		case "status":
			less = a.Status < b.Status
		case "createdAt":
			less = a.CreatedAt.Before(b.CreatedAt)
		default:
			return false
		}
		if !asc {
			return !less
		}
		return less
	})
	return types
}

// ==================== 字典数据缓存服务 ====================

// dictDataCacheService 字典数据缓存服务
type dictDataCacheService struct {
	*dictDataService
	cache CacheProvider
	CacheServiceBase
}

// NewDictDataServiceWithCache 创建带缓存的字典数据服务
func NewDictDataServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) DictDataService {
	base := &dictDataService{db: db}
	return &dictDataCacheService{
		dictDataService:  base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetByTypeWithCache 根据类型获取字典数据（带缓存）
func (s *dictDataCacheService) GetByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error) {
	cacheKey := GetDictDataByTypeKey(dictType)
	var result []*models.DictData

	expiration := s.GetExpiration(services.CacheConfigDictData, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryByType(ctx, dictType)
	})

	if err != nil {
		return nil, fmt.Errorf("获取字典数据失败: %w", err)
	}

	return result, nil
}

// queryByType 根据类型查询字典数据
func (s *dictDataCacheService) queryByType(ctx context.Context, dictType string) ([]*models.DictData, error) {
	var data []models.DictData
	if err := s.db.WithContext(ctx).Where("dict_type = ? AND status = ?", dictType, 0).
		Order("dict_sort ASC").Find(&data).Error; err != nil {
		return nil, err
	}
	result := make([]*models.DictData, len(data))
	for i := range data {
		result[i] = &data[i]
	}
	return result, nil
}

// Create 创建字典数据（带缓存失效）
func (s *dictDataCacheService) Create(ctx context.Context, req *requests.DictDataCreateRequest) error {
	if err := s.dictDataService.Create(ctx, req); err != nil {
		return err
	}
	s.invalidateCache(ctx, req.DictType)
	return nil
}

// Update 更新字典数据（带缓存失效）
func (s *dictDataCacheService) Update(ctx context.Context, req *requests.DictDataUpdateRequest) error {
	// 获取原数据以获取 dictType
	var oldData models.DictData
	if err := s.db.WithContext(ctx).First(&oldData, "id = ?", req.ID).Error; err != nil {
		return err
	}

	if err := s.dictDataService.Update(ctx, req); err != nil {
		return err
	}
	s.invalidateCache(ctx, oldData.DictType)
	return nil
}

// Delete 删除字典数据（带缓存失效）
func (s *dictDataCacheService) Delete(ctx context.Context, id string) error {
	// 获取原数据以获取 dictType
	var oldData models.DictData
	if err := s.db.WithContext(ctx).First(&oldData, "id = ?", id).Error; err != nil {
		return err
	}

	if err := s.dictDataService.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateCache(ctx, oldData.DictType)
	return nil
}

// invalidateCache 失效缓存
func (s *dictDataCacheService) invalidateCache(ctx context.Context, dictType string) {
	cacheKey := GetDictDataByTypeKey(dictType)
	InvalidateCacheByKey(ctx, s.cache, []string{cacheKey}, "DICT")
}
