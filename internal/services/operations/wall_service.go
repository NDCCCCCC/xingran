package operations

import (
	"context"

	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"gorm.io/gorm"
)

// WallService 墙体服务接口
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

// NewWallService 创建墙体服务
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
	query := s.buildListQuery(ctx, params)
	pagination := extractPagination(params)

	total, err := s.countRecords(query)
	if err != nil {
		return nil, err
	}

	list, err := s.fetchRecords(query, pagination, &[]operationsmodels.Wall{})
	if err != nil {
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

// buildListQuery 构建列表查询
func (s *wallService) buildListQuery(ctx context.Context, params map[string]interface{}) *gorm.DB {
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

	return query
}

// countRecords 统计记录数
func (s *wallService) countRecords(query *gorm.DB) (int64, error) {
	var total int64
	err := query.Count(&total).Error
	return total, err
}

// fetchRecords 获取记录列表
func (s *wallService) fetchRecords(query *gorm.DB, pagination PaginationParams, dest interface{}) (interface{}, error) {
	offset := calculateOffset(pagination)
	err := query.
		Offset(offset).
		Limit(pagination.PageSize).
		Order("created_at DESC").
		Find(dest).Error
	return dest, err
}
