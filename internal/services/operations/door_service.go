package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// DoorService 门服务接口
type DoorService interface {
	Create(ctx context.Context, door *operationsmodels.Door) error
	Update(ctx context.Context, door *operationsmodels.Door) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operationsmodels.Door, error)
	List(ctx context.Context, req requests.DoorListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type doorService struct {
	db        *gorm.DB
	validator Validator
}

// NewDoorService 创建门服务
func NewDoorService(db *gorm.DB) DoorService {
	return &doorService{
		db:        db,
		validator: NewValidator(db),
	}
}

// doorAllowedSortFields 门可排序字段白名单(对应 doors 表列名)。
var doorAllowedSortFields = map[string]string{
	"floorId":   "floor_id",
	"doorType":  "type",
	"createdAt": "created_at",
}

func (s *doorService) Create(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validateDoorRelations(ctx, door); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(door).Error
}

func (s *doorService) Update(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validateDoorRelations(ctx, door); err != nil {
		return err
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

func (s *doorService) List(ctx context.Context, req requests.DoorListRequest) (*PageResult, error) {
	query := s.buildListQueryFromRequest(ctx, req)

	// 用户排序(白名单);无 OrderByColumn 时 fetchRecords 内部仍用 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, doorAllowedSortFields)
	if req.OrderByColumn != "" {
		query = query.Order("")
	}

	total, err := s.countRecords(query)
	if err != nil {
		return nil, err
	}

	current, pageSize := req.GetPagination()

	list, err := s.fetchRecords(query, current, pageSize, &[]operationsmodels.Door{})
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

func (s *doorService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&operationsmodels.Door{}, "id IN ?", ids).Error
}

// validateDoorRelations 验证门的关联关系
func (s *doorService) validateDoorRelations(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validator.ValidateFloor(ctx, door.FloorID); err != nil {
		return err
	}
	if door.WallID != nil && *door.WallID != "" {
		if err := s.validator.ValidateWall(ctx, *door.WallID); err != nil {
			return err
		}
	}
	return nil
}

// buildListQueryFromRequest 从请求构建列表查询（类型安全版本）
func (s *doorService) buildListQueryFromRequest(ctx context.Context, req requests.DoorListRequest) *gorm.DB {
	query := s.db.WithContext(ctx).Model(&operationsmodels.Door{})

	if req.FloorID != "" {
		query = query.Where("floor_id = ?", req.FloorID)
	}
	if req.DoorType != "" {
		query = query.Where("type = ?", req.DoorType)
	}

	return query
}

// countRecords 统计记录数
func (s *doorService) countRecords(query *gorm.DB) (int64, error) {
	var total int64
	err := query.Count(&total).Error
	return total, err
}

// fetchRecords 获取记录列表
func (s *doorService) fetchRecords(query *gorm.DB, current, pageSize int, dest interface{}) (interface{}, error) {
	offset := (current - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(dest).Error
	return dest, err
}
