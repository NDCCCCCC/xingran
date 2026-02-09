package operations

import (
	"context"

	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

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

func NewFloorService(db *gorm.DB) FloorService {
	return &floorService{db: db}
}

func (s *floorService) Create(ctx context.Context, floor *operations.OpsFloor) error {
	if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(floor).Error; err != nil {
		return err
	}
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

func (s *floorService) Update(ctx context.Context, floor *operations.OpsFloor) error {
	if floor.BuildingID != "" {
		if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Save(floor).Error
}

func (s *floorService) Delete(ctx context.Context, id string) error {
	var floor operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id = ?", id).Error; err != nil {
		return err
	}
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

func (s *floorService) GetByID(ctx context.Context, id string) (*operations.OpsFloor, error) {
	var floor operations.OpsFloor
	err := s.db.WithContext(ctx).
		Select("ops_floors.*, ops_buildings.name as building_name, '/uploads/' || sys_files.storage_path as plan_image_url").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid").
		Joins("LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id::uuid").
		Where("ops_floors.id = ?", id).
		First(&floor).Error
	if err != nil {
		return nil, err
	}
	return &floor, nil
}

func (s *floorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsFloor

	query := s.db.WithContext(ctx).Model(&operations.OpsFloor{})

	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("ops_floors.name LIKE ?", "%"+name+"%")
	}
	if buildingId, ok := params["buildingId"].(string); ok && buildingId != "" {
		query = query.Where("ops_floors.building_id = ?", buildingId)
	}

	current := 1
	pageSize := 10
	if c, ok := params["current"].(int); ok {
		current = c
	} else if c, ok := params["current"].(float64); ok {
		current = int(c)
	}
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	} else if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (current - 1) * pageSize
	if err := query.
		Select("ops_floors.*, ops_buildings.name as building_name, '/uploads/' || sys_files.storage_path as plan_image_url").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid").
		Joins("LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id::uuid").
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

	var tree []FloorTreeNode
	for _, node := range buildingMap {
		tree = append(tree, *node)
	}

	return tree, nil
}

func (s *floorService) BatchDelete(ctx context.Context, ids []string) error {
	var floors []operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&floors).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id IN ?", ids).Error; err != nil {
		return err
	}

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

func (s *floorService) validateBuilding(ctx context.Context, buildingID string) error {
	if buildingID == "" {
		return apperrors.ParamMissing("所属楼宇ID")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apperrors.BuildingNotFound()
	}
	return nil
}

func (s *floorService) updateBuildingFloorCount(ctx context.Context, buildingID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("building_id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Update("total_floors", count).Error
}
