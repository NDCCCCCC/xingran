package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)

// floorCacheService 楼层缓存服务
type floorCacheService struct {
	*floorService
	cache system.CacheProvider
	system.CacheServiceBase
}

// NewFloorServiceWithCache 创建带缓存的楼层服务
func NewFloorServiceWithCache(
	db *gorm.DB,
	cache system.CacheProvider,
	config *services.CacheConfigService,
) FloorService {
	base := &floorService{db: db}
	return &floorCacheService{
		floorService:     base,
		cache:            cache,
		CacheServiceBase: system.CacheServiceBase{Config: config},
	}
}

// GetTree 获取楼层树（带缓存）
// 这是最需要缓存的方法，被多个页面高频调用
func (s *floorCacheService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
	cacheKey := "floor:tree"
	var result []FloorTreeNode

	expiration := s.GetExpiration("cache.floor.tree", 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.floorService.GetTree(ctx)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetFloorsByBuildingID 获取指定楼宇的楼层列表（带缓存）
func (s *floorCacheService) GetFloorsByBuildingID(ctx context.Context, buildingID string) ([]operations.OpsFloor, error) {
	cacheKey := fmt.Sprintf("floor:building:%s", buildingID)
	var result []operations.OpsFloor

	expiration := s.GetExpiration("cache.floor.building", 15*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		// 查询指定楼宇的所有楼层
		var floors []operations.OpsFloor
		if err := s.db.WithContext(ctx).
			Where("building_id = ?", buildingID).
			Order("order_num ASC").
			Find(&floors).Error; err != nil {
			return nil, err
		}
		return floors, nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// InvalidateFloorCache 失效楼层缓存
func (s *floorCacheService) InvalidateFloorCache(ctx context.Context, buildingID string) error {
	keys := []string{
		"floor:tree",
	}
	if buildingID != "" {
		keys = append(keys, fmt.Sprintf("floor:building:%s", buildingID))
	}
	system.InvalidateCacheByKey(ctx, s.cache, keys, "FLOOR")
	return nil
}

// InvalidateAllFloorCache 失效所有楼层缓存
func (s *floorCacheService) InvalidateAllFloorCache(ctx context.Context) error {
	system.InvalidateCacheByPattern(ctx, s.cache, []string{"floor:*"}, "FLOOR")
	return nil
}

// Create 创建楼层（带缓存失效）
func (s *floorCacheService) Create(ctx context.Context, floor *operations.OpsFloor) error {
	if err := s.floorService.Create(ctx, floor); err != nil {
		return err
	}
	// 清除该楼宇的楼层缓存
	return s.InvalidateFloorCache(ctx, floor.BuildingID)
}

// Update 更新楼层（带缓存失效）
func (s *floorCacheService) Update(ctx context.Context, floor *operations.OpsFloor) error {
	if err := s.floorService.Update(ctx, floor); err != nil {
		return err
	}
	// 清除该楼宇的楼层缓存
	return s.InvalidateFloorCache(ctx, floor.BuildingID)
}

// Delete 删除楼层（带缓存失效）
func (s *floorCacheService) Delete(ctx context.Context, id string) error {
	// 先获取楼层以便获取 buildingID
	var floor operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error; err != nil {
		return err
	}

	if err := s.floorService.Delete(ctx, id); err != nil {
		return err
	}

	// 清除该楼宇的楼层缓存
	return s.InvalidateFloorCache(ctx, floor.BuildingID)
}

// BatchDelete 批量删除楼层（带缓存失效）
func (s *floorCacheService) BatchDelete(ctx context.Context, ids []string) error {
	// 先获取所有楼层以便获取 buildingID
	var floors []operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&floors).Error; err != nil {
		return err
	}

	if err := s.floorService.BatchDelete(ctx, ids); err != nil {
		return err
	}

	// 清除所有涉及楼宇的楼层缓存
	buildingIDMap := make(map[string]bool)
	for _, floor := range floors {
		buildingIDMap[floor.BuildingID] = true
	}

	for buildingID := range buildingIDMap {
		if err := s.InvalidateFloorCache(ctx, buildingID); err != nil {
			return err
		}
	}

	return nil
}

// GetByID 获取楼层详情（不缓存，查询频率低）
func (s *floorCacheService) GetByID(ctx context.Context, id string) (*operations.OpsFloor, error) {
	return s.floorService.GetByID(ctx, id)
}

// List 查询楼层列表（不缓存，参数多变）
func (s *floorCacheService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	return s.floorService.List(ctx, params)
}

// SearchFloorOptions 楼层下拉数据源(仅 name="" 路径走 5min Redis 缓存;
// keyword 查询绕过缓存避免 keyspace 爆炸)。参考 GetTree 的 cache.GetOrSet 模式。
func (s *floorCacheService) SearchFloorOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	// keyword 查询不走缓存(每次 keystroke 都会触发,keyspace 爆炸)
	if name, _ := params["name"].(string); name != "" {
		return s.floorService.SearchFloorOptions(ctx, params)
	}

	cacheKey := BuildDropdownCacheKey("floor", params)
	var result []DropdownOption
	expiration := s.GetExpiration("cache.floor.dropdown", 5*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.floorService.SearchFloorOptions(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
