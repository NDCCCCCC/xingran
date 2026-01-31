package operations

import (
	"context"

	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

type FloorPlanTextService interface {
	Create(ctx context.Context, text *operationsmodels.FloorPlanText) error
	Update(ctx context.Context, text *operationsmodels.FloorPlanText) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.FloorPlanText, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type floorPlanTextService struct {
	db        *gorm.DB
	validator Validator
}

func NewFloorPlanTextService(db *gorm.DB) FloorPlanTextService {
	return &floorPlanTextService{
		db:        db,
		validator: NewValidator(db),
	}
}

func (s *floorPlanTextService) Create(ctx context.Context, text *operationsmodels.FloorPlanText) error {
	if err := s.validator.ValidateFloor(ctx, text.FloorID.String()); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(text).Error
}

func (s *floorPlanTextService) Update(ctx context.Context, text *operationsmodels.FloorPlanText) error {
	if err := s.validator.ValidateFloor(ctx, text.FloorID.String()); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Save(text).Error
}

func (s *floorPlanTextService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operationsmodels.FloorPlanText{}, "id = ?", id).Error
}

func (s *floorPlanTextService) GetByID(ctx context.Context, id string) (*operationsmodels.FloorPlanText, error) {
	var text operationsmodels.FloorPlanText
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&text).Error
	if err != nil {
		return nil, err
	}
	return &text, nil
}

func (s *floorPlanTextService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operationsmodels.FloorPlanText

	query := s.db.WithContext(ctx).Model(&operationsmodels.FloorPlanText{})

	if floorId, ok := params["floorId"].(string); ok && floorId != "" {
		query = query.Where("floor_id = ?", floorId)
	}
	if content, ok := params["content"].(string); ok && content != "" {
		query = query.Where("content LIKE ?", "%"+content+"%")
	}

	pagination := extractPagination(params)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := calculateOffset(pagination)
	if err := query.Offset(offset).Limit(pagination.PageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		Total:    total,
		List:     list,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

func (s *floorPlanTextService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.FloorPlanText{}, "id IN ?", ids).Error
}
