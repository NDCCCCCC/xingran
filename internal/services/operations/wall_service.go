package operations

import (
	"context"

	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

type WallService interface {
	Create(ctx context.Context, wall *operationsmodels.Wall) error
	Update(ctx context.Context, wall *operationsmodels.Wall) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.Wall, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type wallService struct {
	db        *gorm.DB
	validator Validator
}

func NewWallService(db *gorm.DB) WallService {
	return &wallService{
		db:        db,
		validator: NewValidator(db),
	}
}

func (s *wallService) Create(ctx context.Context, wall *operationsmodels.Wall) error {
	if err := s.validator.ValidateFloor(ctx, wall.FloorID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(wall).Error
}

func (s *wallService) Update(ctx context.Context, wall *operationsmodels.Wall) error {
	if err := s.validator.ValidateFloor(ctx, wall.FloorID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Save(wall).Error
}

func (s *wallService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operationsmodels.Wall{}, "id = ?", id).Error
}

func (s *wallService) GetByID(ctx context.Context, id string) (*operationsmodels.Wall, error) {
	var wall operationsmodels.Wall
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&wall).Error
	if err != nil {
		return nil, err
	}
	return &wall, nil
}

func (s *wallService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operationsmodels.Wall

	query := s.db.WithContext(ctx).Model(&operationsmodels.Wall{})

	if floorId, ok := params["floorId"].(string); ok && floorId != "" {
		query = query.Where("floor_id = ?", floorId)
	}
	if wallType, ok := params["type"].(string); ok && wallType != "" {
		query = query.Where("type = ?", wallType)
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

func (s *wallService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.Wall{}, "id IN ?", ids).Error
}
