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
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operationsmodels.Door]
	// db 保留给 Validator 构造来源（validateDoorRelations 前置校验）
	db        *gorm.DB
	validator Validator
}

// NewDoorService 创建门服务
func NewDoorService(db *gorm.DB) DoorService {
	return &doorService{
		repo:      base.NewGORMRepository[operationsmodels.Door](db),
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
	return s.repo.Create(ctx, door)
}

func (s *doorService) Update(ctx context.Context, door *operationsmodels.Door) error {
	if err := s.validateDoorRelations(ctx, door); err != nil {
		return err
	}
	return s.repo.Update(ctx, door)
}

func (s *doorService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *doorService) GetByID(ctx context.Context, id string) (*operationsmodels.Door, error) {
	return s.repo.GetByID(ctx, id)
}

// filterScope List 的两个过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
func (s *doorService) filterScope(req requests.DoorListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.FloorID != "" {
			db = db.Where("floor_id = ?", req.FloorID)
		}
		if req.DoorType != "" {
			db = db.Where("type = ?", req.DoorType)
		}
		return db
	}
}

// List 查询门列表（CRUD 管道经 base.GORMRepository 执行）。
//
// A 型复合尾随排序语义（Phase 91-04，T-91-04-01 缓解）：SortScopeWithTail
// 无条件尾随 created_at DESC——
//   - 用户排序（白名单命中）→ ORDER BY <col> <dir>, created_at DESC
//   - 无/非法排序 → 仅 ORDER BY created_at DESC
//
// 与迁移前取数管道恒追加 Order("created_at DESC") 逐字等价（无排序时
// ApplySort 无 clauses，仅剩尾随序）。迁移前 List 中「用户排序时追加空串
// Order」的分支是 GORM no-op 死代码（空串不进 clause），随迁移删除。
func (s *doorService) List(ctx context.Context, req requests.DoorListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(req),
		base.SortScopeWithTail(req.BaseListRequest, doorAllowedSortFields, "created_at DESC"),
	)
}

func (s *doorService) BatchDelete(ctx context.Context, ids []string) error {
	// 空 ids 早退已由 repo 承接（91-01 P1 语义反转：空 ids 返回 nil），
	// 与迁移前 `if len(ids) == 0 { return nil }` 行为一致
	return s.repo.BatchDelete(ctx, ids)
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
