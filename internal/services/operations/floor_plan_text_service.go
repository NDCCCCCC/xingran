package operations

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

const (
	floorPlanTextTable = "ops_floor_plan_texts"
)

type FloorPlanTextService interface {
	Create(ctx context.Context, text *operationsmodels.FloorPlanText) error
	Update(ctx context.Context, text *operationsmodels.FloorPlanText) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.FloorPlanText, error)
	List(ctx context.Context, req requests.FloorPlanTextListRequest) (*PageResult, error)
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
	if err := s.validator.ValidateFloor(ctx, text.FloorID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(text).Error
}

func (s *floorPlanTextService) Update(ctx context.Context, text *operationsmodels.FloorPlanText) error {
	if err := s.validator.ValidateFloor(ctx, text.FloorID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Save(text).Error
}

func (s *floorPlanTextService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Table(floorPlanTextTable).Where("id = ?", id).Delete(&operationsmodels.FloorPlanText{}).Error
}

func (s *floorPlanTextService) GetByID(ctx context.Context, id string) (*operationsmodels.FloorPlanText, error) {
	var text operationsmodels.FloorPlanText
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&text).Error
	if err != nil {
		return nil, err
	}
	return &text, nil
}

func (s *floorPlanTextService) List(ctx context.Context, req requests.FloorPlanTextListRequest) (*PageResult, error) {
	var total int64
	var list []operationsmodels.FloorPlanText

	query := s.db.WithContext(ctx).Model(&operationsmodels.FloorPlanText{})

	// 添加筛选条件 - 类型安全，无需类型断言
	if req.FloorID != "" {
		query = query.Where("floor_id = ?", req.FloorID)
	}
	if req.Content != "" {
		query = query.Where("content LIKE ?", "%"+req.Content+"%")
	}

	// 分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()
	current, _ := req.GetPagination()

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		Total:    total,
		List:     list,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

func (s *floorPlanTextService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Table(floorPlanTextTable).Where("id IN ?", ids).Delete(&operationsmodels.FloorPlanText{}).Error
}
