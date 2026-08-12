package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// WallService 墙体服务接口
type WallService interface {
	Create(ctx context.Context, wall *operationsmodels.Wall) error
	Update(ctx context.Context, wall *operationsmodels.Wall) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.Wall, error)
	List(ctx context.Context, req requests.WallListRequest) (*PageResult, error)
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

func (s *wallService) List(ctx context.Context, req requests.WallListRequest) (*PageResult, error) {
	query := s.buildListQueryFromRequest(ctx, req)

	// 用户排序(白名单);无 OrderByColumn 时 fetchRecords 内部仍用 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, wallAllowedSortFields)
	if req.OrderByColumn != "" {
		// 用户选了排序时,移除 fetchRecords 内的硬编码 Order,避免两个 Order 冲突
		query = query.Order("")
	}

	total, err := s.countRecords(query)
	if err != nil {
		return nil, err
	}

	current, pageSize := req.GetPagination()

	list, err := s.fetchRecords(query, current, pageSize, &[]operationsmodels.Wall{})
	if err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

func (s *wallService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.Wall{}, "id IN ?", ids).Error
}

// buildListQueryFromRequest 从请求构建列表查询（类型安全版本）
func (s *wallService) buildListQueryFromRequest(ctx context.Context, req requests.WallListRequest) *gorm.DB {
	query := s.db.WithContext(ctx).Model(&operationsmodels.Wall{})

	if req.FloorID != "" {
		query = query.Where("floor_id = ?", req.FloorID)
	}
	if req.WallType != "" {
		query = query.Where("type = ?", req.WallType)
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
func (s *wallService) fetchRecords(query *gorm.DB, current, pageSize int, dest interface{}) (interface{}, error) {
	offset := (current - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(dest).Error
	return dest, err
}

// wallAllowedSortFields 墙体可排序字段白名单(对应 walls 表列名)。
var wallAllowedSortFields = map[string]string{
	"floorId":   "floor_id",
	"wallType":  "type",
	"createdAt": "created_at",
}
