package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
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
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operationsmodels.FloorPlanText]
	// db 保留给 Validator 构造来源（Create/Update 的 ValidateFloor 前置校验）
	db        *gorm.DB
	validator Validator
}

func NewFloorPlanTextService(db *gorm.DB) FloorPlanTextService {
	return &floorPlanTextService{
		repo:      base.NewGORMRepository[operationsmodels.FloorPlanText](db),
		db:        db,
		validator: NewValidator(db),
	}
}

// floorPlanTextAllowedSortFields 楼层平面文本可排序字段白名单。
var floorPlanTextAllowedSortFields = map[string]string{
	"floorId":   "floor_id",
	"textType":  "text_type",
	"createdAt": "created_at",
}

func (s *floorPlanTextService) Create(ctx context.Context, text *operationsmodels.FloorPlanText) error {
	if err := s.validator.ValidateFloor(ctx, text.FloorID); err != nil {
		return err
	}
	// Phase 149 / SCOPE-03：提交侧 floor_id 越界 → 单条显式 403。
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityFloorPlanText, text.FloorID); err != nil {
		return err
	}
	return s.repo.Create(ctx, text)
}

func (s *floorPlanTextService) Update(ctx context.Context, text *operationsmodels.FloorPlanText) error {
	if err := s.validator.ValidateFloor(ctx, text.FloorID); err != nil {
		return err
	}
	// Phase 149 / SCOPE-03：改前侧按 text.ID + 提交侧 floor_id，任一侧越界 403。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityFloorPlanText, text.ID); err != nil {
		return err
	}
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityFloorPlanText, text.FloorID); err != nil {
		return err
	}
	return s.repo.Update(ctx, text)
}

func (s *floorPlanTextService) Delete(ctx context.Context, id string) error {
	// Phase 149 / SCOPE-03：改前侧越界 → 单条显式 403。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityFloorPlanText, id); err != nil {
		return err
	}
	// 迁移前 .Table(ops_floor_plan_texts).Delete(&T{}) 形态与 repo 的
	// Model(new(T)) 形态行为等价（软删 + WHERE 语义一致，RESEARCH 盘点 #11）
	return s.repo.Delete(ctx, id)
}

func (s *floorPlanTextService) GetByID(ctx context.Context, id string) (*operationsmodels.FloorPlanText, error) {
	// Phase 149 / SCOPE-01：读端点按可见集裁剪，越界 / 断链 → record not found（空集语义）。
	return s.repo.GetByID(ctx, id, entityReadScope(ctx, scopeEntityFloorPlanText, "ops_floor_plan_texts"))
}

// filterScope List 的两个过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
func (s *floorPlanTextService) filterScope(req requests.FloorPlanTextListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.FloorID != "" {
			db = db.Where("floor_id = ?", req.FloorID)
		}
		if req.Content != "" {
			db = db.Where("content LIKE ?", "%"+req.Content+"%")
		}
		return db
	}
}

// List 查询楼层平面文本列表（CRUD 管道经 base.GORMRepository 执行；
// B 型排他默认排序，无 OrderByColumn 时保留 created_at DESC 默认）。
func (s *floorPlanTextService) List(ctx context.Context, req requests.FloorPlanTextListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()
	// Phase 149 / SCOPE-01：scopes 追加可见集裁剪（只裁剪行，响应形状不变）；
	// 业务 filterScope 与排序 scope 顺序不变；Count 与 Find 同源。
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(req),
		base.SortScope(req.BaseListRequest, floorPlanTextAllowedSortFields, "created_at DESC"),
		entityReadScope(ctx, scopeEntityFloorPlanText, "ops_floor_plan_texts"),
	)
}

func (s *floorPlanTextService) BatchDelete(ctx context.Context, ids []string) error {
	// Phase 149 / SCOPE-03：后端权威边界 —— 按可见集裁剪后继续（不整批 403）。
	visible, _, err := filterEntityVisibleIDs(ctx, s.db, scopeEntityFloorPlanText, ids)
	if err != nil {
		return err
	}
	// 空 ids 早退已由 repo 承接（91-01 P1 语义反转：空 ids 返回 nil）。
	return s.repo.BatchDelete(ctx, visible)
}
