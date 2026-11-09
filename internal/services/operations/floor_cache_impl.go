package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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

// scopedFloorTreeKey 由请求 ctx 的 ScopeResult 派生楼层树缓存 key。
//
// PMODEL-01：生产路径（已挂 DataScopePermission 注入）一律带 scope 段，
// 不同可见集不共享同一份树缓存；All 档短路路径同样含 "all" 段
// （LOCKED §1 关键不变量 3）。
//
// 兼容策略：未注入 ctx（存量单测 / 内部任务以 context.Background() 调用）保持
// 旧精确 key "floor:tree"，保证既有断言与行为零回归——这与「未注入按 All 兜底」
// 的读路径语义一致，且这类调用方本就不跨用户，不存在泄漏面。
func scopedFloorTreeKey(ctx context.Context) string {
	if _, ok := scopeFromContext(ctx); !ok {
		return "floor:tree"
	}
	return fmt.Sprintf("floor:tree:s%s", scopeCacheKey(ctx))
}

// scopedFloorBuildingKey 同 scopedFloorTreeKey，作用于 floor:building:<id>。
func scopedFloorBuildingKey(ctx context.Context, buildingID string) string {
	if _, ok := scopeFromContext(ctx); !ok {
		return fmt.Sprintf("floor:building:%s", buildingID)
	}
	return fmt.Sprintf("floor:building:s%s:%s", scopeCacheKey(ctx), buildingID)
}

// GetTree 获取楼层树（带缓存）
// 这是最需要缓存的方法，被多个页面高频调用
func (s *floorCacheService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
	return base.GetOrSetJSON(ctx, s.cache,
		scopedFloorTreeKey(ctx),
		s.GetExpiration("cache.floor.tree", 30*time.Minute),
		func() ([]FloorTreeNode, error) {
			return s.floorService.GetTree(ctx)
		})
}

// GetFloorsByBuildingID 获取指定楼宇的楼层列表（带缓存）
func (s *floorCacheService) GetFloorsByBuildingID(ctx context.Context, buildingID string) ([]operations.OpsFloor, error) {
	return base.GetOrSetJSON(ctx, s.cache,
		scopedFloorBuildingKey(ctx, buildingID),
		s.GetExpiration("cache.floor.building", 15*time.Minute),
		func() ([]operations.OpsFloor, error) {
			// 查询指定楼宇的所有楼层
			var floors []operations.OpsFloor
			if err := ScopedQuery(ctx, s.db).
				Where("building_id = ?", buildingID).
				Order("order_num ASC").
				Find(&floors).Error; err != nil {
				return nil, err
			}
			return floors, nil
		})
}

// InvalidateFloorCache 失效楼层缓存。
//
// tree 与 building 均改用前缀通配：一次清掉旧 key（未注入路径遗留）与
// 所有 scope 变体，避免写操作后某个可见集仍命中陈旧缓存。
func (s *floorCacheService) InvalidateFloorCache(ctx context.Context, buildingID string) error {
	patterns := []string{"floor:tree*"}
	if buildingID != "" {
		// MemoryCache.Keys 只支持「末尾 *」前缀匹配，故按实体前缀整段清除，
		// 一次覆盖所有 scope 变体（写操作后全清属既有安全语义）。
		patterns = append(patterns, "floor:building:*")
	}
	base.InvalidatePattern(ctx, s.cache, patterns, "FLOOR")
	return nil
}

// InvalidateAllFloorCache 失效所有楼层缓存
func (s *floorCacheService) InvalidateAllFloorCache(ctx context.Context) error {
	base.InvalidatePattern(ctx, s.cache, []string{"floor:*"}, "FLOOR")
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

	// Phase 131：dropdown 缓存 key 增 scope 维度段（PMODEL-01 防跨范围泄漏）。
	cacheKey := BuildScopedDropdownCacheKey(ctx, "floor", params)
	return base.GetOrSetJSON(ctx, s.cache,
		cacheKey,
		s.GetExpiration("cache.floor.dropdown", 5*time.Minute),
		func() ([]DropdownOption, error) {
			return s.floorService.SearchFloorOptions(ctx, params)
		})
}
