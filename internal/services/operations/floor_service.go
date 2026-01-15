package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// FloorService 楼层服务接口
type FloorService interface {
	Create(ctx context.Context, floor *operations.OpsFloor) error
	Update(ctx context.Context, floor *operations.OpsFloor) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsFloor, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	GetTree(ctx context.Context) ([]FloorTreeNode, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type FloorTreeNode struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	FloorNo    string          `json:"floorNo"`
	BuildingID string          `json:"buildingId"`
	Children   []FloorTreeNode `json:"children"`
}

type floorService struct {
	db *gorm.DB
}

// NewFloorService 创建楼层服务实例
func NewFloorService(db *gorm.DB) FloorService {
	return &floorService{db: db}
}

// Create 创建楼层
func (s *floorService) Create(ctx context.Context, floor *operations.OpsFloor) error {
	// 验证楼宇存在性
	if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(floor).Error; err != nil {
		return err
	}
	// 更新楼宇的楼层数
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

// Update 更新楼层
func (s *floorService) Update(ctx context.Context, floor *operations.OpsFloor) error {
	if floor.BuildingID != "" {
		if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Save(floor).Error
}

// Delete 删除楼层
func (s *floorService) Delete(ctx context.Context, id string) error {
	// 先获取楼层信息以获取buildingID
	var floor operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id = ?", id).Error; err != nil {
		return err
	}
	// 更新楼宇的楼层数
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

// GetByID 根据ID获取楼层
func (s *floorService) GetByID(ctx context.Context, id string) (*operations.OpsFloor, error) {
	var floor operations.OpsFloor
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error
	if err != nil {
		return nil, err
	}
	return &floor, nil
}

// List 查询楼层列表
func (s *floorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsFloor

	query := s.db.WithContext(ctx).Model(&operations.OpsFloor{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("ops_floors.name LIKE ?", "%"+name+"%")
	}
	if buildingId, ok := params["buildingId"].(string); ok && buildingId != "" {
		query = query.Where("ops_floors.building_id = ?", buildingId)
	}

	// 分页
	current := 1
	pageSize := 10
	if c, ok := params["current"].(int); ok {
		current = c
	}
	if c, ok := params["current"].(float64); ok {
		current = int(c)
	}
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	}
	if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (current - 1) * pageSize
	// JOIN ops_buildings 表获取 building_name（类型转换：varchar -> uuid）
	if err := query.
		Select("ops_floors.*, ops_buildings.name as building_name").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid").
		Offset(offset).
		Limit(pageSize).
		Order("ops_floors.order_num ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// GetTree 获取楼宇-楼层树
func (s *floorService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
	type BuildingFloor struct {
		BuildingID   string
		BuildingName string
		FloorID      string
		FloorName    string
		FloorNo      string
	}

	var results []BuildingFloor
	err := s.db.WithContext(ctx).Raw(`
        SELECT
            b.id as building_id,
            b.name as building_name,
            f.id as floor_id,
            f.name as floor_name,
            f.floor_no
        FROM ops_buildings b
        LEFT JOIN ops_floors f ON b.id = f.building_id
        WHERE b.deleted_at IS NULL
        ORDER BY b.order_num ASC, f.order_num ASC
    `).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 构建树形结构
	buildingMap := make(map[string]*FloorTreeNode)

	for _, r := range results {
		if _, exists := buildingMap[r.BuildingID]; !exists {
			buildingMap[r.BuildingID] = &FloorTreeNode{
				ID:       r.BuildingID,
				Name:     r.BuildingName,
				Children: []FloorTreeNode{},
			}
		}

		if r.FloorID != "" {
			buildingMap[r.BuildingID].Children = append(buildingMap[r.BuildingID].Children, FloorTreeNode{
				ID:         r.FloorID,
				Name:       r.FloorName,
				FloorNo:    r.FloorNo,
				BuildingID: r.BuildingID,
			})
		}
	}

	// 转换为数组
	var tree []FloorTreeNode
	for _, node := range buildingMap {
		tree = append(tree, *node)
	}

	return tree, nil
}

// BatchDelete 批量删除楼层
func (s *floorService) BatchDelete(ctx context.Context, ids []string) error {
	// 先获取所有楼层信息以收集buildingID
	var floors []operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&floors).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id IN ?", ids).Error; err != nil {
		return err
	}
	// 收集所有受影响的楼宇ID并更新楼层数
	buildingIDMap := make(map[string]bool)
	for _, floor := range floors {
		buildingIDMap[floor.BuildingID] = true
	}
	for buildingID := range buildingIDMap {
		if err := s.updateBuildingFloorCount(ctx, buildingID); err != nil {
			return err
		}
	}
	return nil
}

// validateBuilding 验证楼宇存在性
func (s *floorService) validateBuilding(ctx context.Context, buildingID string) error {
	// 验证buildingID不为空
	if buildingID == "" {
		return errors.New("所属楼宇ID不能为空")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("所属楼宇不存在")
	}
	return nil
}

// updateBuildingFloorCount 更新楼宇的楼层数
func (s *floorService) updateBuildingFloorCount(ctx context.Context, buildingID string) error {
	// 统计该楼宇的楼层总数
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("building_id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	// 更新楼宇的totalFloors字段
	return s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Update("total_floors", count).Error
}
