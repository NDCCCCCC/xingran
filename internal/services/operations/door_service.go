package operations

import (
	"context"

	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

type DoorService interface {
	Create(ctx context.Context, door *operationsmodels.Door) error
	Update(ctx context.Context, door *operationsmodels.Door) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.Door, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type doorService struct {
	db        *gorm.DB
	validator Validator
}

func NewDoorService(db *gorm.DB) DoorService {
	return &doorService{
		db:        db,
		validator: NewValidator(db),
	}
}

func (s *doorService) Create(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validator.ValidateFloor(ctx, door.FloorID); err != nil {
		return err
	}
	if door.WallID != nil && *door.WallID != "" {
		if err := s.validator.ValidateWall(ctx, *door.WallID); err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Create(door).Error
}

func (s *doorService) Update(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validator.ValidateFloor(ctx, door.FloorID); err != nil {
		return err
	}
	if door.WallID != nil && *door.WallID != "" {
		if err := s.validator.ValidateWall(ctx, *door.WallID); err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Save(door).Error
}

func (s *doorService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operationsmodels.Door{}, "id = ?", id).Error
}

func (s *doorService) GetByID(ctx context.Context, id string) (*operationsmodels.Door, error) {
	var door operationsmodels.Door
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&door).Error
	if err != nil {
		return nil, err
	}
	return &door, nil
}

func (s *doorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operationsmodels.Door

	query := s.db.WithContext(ctx).Model(&operationsmodels.Door{})

	if floorId, ok := params["floorId"].(string); ok && floorId != "" {
		query = query.Where("floor_id = ?", floorId)
	}
	if wallId, ok := params["wallId"].(string); ok && wallId != "" {
		query = query.Where("wall_id = ?", wallId)
	}
	if doorType, ok := params["type"].(string); ok && doorType != "" {
		query = query.Where("type = ?", doorType)
	}
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	pagination := extractPagination(params)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := calculateOffset(pagination)
	if err := query.
		Offset(offset).
		Limit(pagination.PageSize).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

func (s *doorService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.Door{}, "id IN ?", ids).Error
}
