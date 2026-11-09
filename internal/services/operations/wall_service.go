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
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operationsmodels.Wall]
	// db 保留给 Validator 构造来源（Create/Update 的 ValidateFloor 前置校验）
	db        *gorm.DB
	validator Validator
}

// NewWallService 创建墙体服务
func NewWallService(db *gorm.DB) WallService {
	return &wallService{
		repo:      base.NewGORMRepository[operationsmodels.Wall](db),
		db:        db,
		validator: NewValidator(db),
	}
}

func (s *wallService) Create(ctx context.Context, wall *operationsmodels.Wall) error {
	if err := s.validator.ValidateFloor(ctx, wall.FloorID); err != nil {
		return err
	}
	// Phase 149 / SCOPE-03：提交侧 floor_id 越界 → 单条显式 403（All/未注入短路放行）。
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityWall, wall.FloorID); err != nil {
		return err
	}
	return s.repo.Create(ctx, wall)
}

func (s *wallService) Update(ctx context.Context, wall *operationsmodels.Wall) error {
	if err := s.validator.ValidateFloor(ctx, wall.FloorID); err != nil {
		return err
	}
	// Phase 149 / SCOPE-03：改前侧按已存在 wall.ID 解析归属链 + 提交侧 floor_id，
	// 任一侧越界 → 单条显式 403；All/未注入零查询短路放行（存量零回归）。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityWall, wall.ID); err != nil {
		return err
	}
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityWall, wall.FloorID); err != nil {
		return err
	}
	return s.repo.Update(ctx, wall)
}

func (s *wallService) Delete(ctx context.Context, id string) error {
	// Phase 149 / SCOPE-03：改前侧越界 → 单条显式 403。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityWall, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *wallService) GetByID(ctx context.Context, id string) (*operationsmodels.Wall, error) {
	// Phase 149 / SCOPE-01：读端点按可见集裁剪，越界 / 断链 → gorm.ErrRecordNotFound
	// （handler 统一转 404「未找到」空集语义，不报 403）。Count/Find 同源由 repo 保证。
	return s.repo.GetByID(ctx, id, entityReadScope(ctx, scopeEntityWall, "ops_walls"))
}

// filterScope List 的两个过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
func (s *wallService) filterScope(req requests.WallListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.FloorID != "" {
			db = db.Where("floor_id = ?", req.FloorID)
		}
		if req.WallType != "" {
			db = db.Where("type = ?", req.WallType)
		}
		return db
	}
}

// List 查询墙体列表（CRUD 管道经 base.GORMRepository 执行）。
//
// A 型复合尾随排序语义（Phase 91-04，T-91-04-01 缓解）：SortScopeWithTail
// 无条件尾随 created_at DESC——
//   - 用户排序（白名单命中）→ ORDER BY <col> <dir>, created_at DESC
//   - 无/非法排序 → 仅 ORDER BY created_at DESC
//
// 与迁移前取数管道恒追加 Order("created_at DESC") 逐字等价。迁移前 List
// 中「用户排序时追加空串 Order」的分支是 GORM no-op 死代码（空串不进 clause，
// 原注释"移除硬编码 Order 避免冲突"的前提不成立），随迁移删除。
func (s *wallService) List(ctx context.Context, req requests.WallListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()
	// Phase 149 / SCOPE-01：scopes 变参追加可见集裁剪（只裁剪行，响应形状不变）；
	// filterScope（业务 floor_id/type 筛选）与排序 scope 顺序不变；Count 与 Find 同源。
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(req),
		base.SortScopeWithTail(req.BaseListRequest, wallAllowedSortFields, "created_at DESC"),
		entityReadScope(ctx, scopeEntityWall, "ops_walls"),
	)
}

func (s *wallService) BatchDelete(ctx context.Context, ids []string) error {
	// Phase 149 / SCOPE-03：后端权威边界 —— 按可见集裁剪后继续（不整批 403）。
	// 越界 / 断链 ID 静默丢弃；All / 未注入短路零查询原样返回（超管零回归）。
	visible, _, err := filterEntityVisibleIDs(ctx, s.db, scopeEntityWall, ids)
	if err != nil {
		return err
	}
	// 空 ids 早退已由 repo 承接（91-01 P1 语义反转：空 ids 返回 nil），
	// 与迁移前 `if len(ids) == 0 { return nil }` 行为一致。
	return s.repo.BatchDelete(ctx, visible)
}

// wallAllowedSortFields 墙体可排序字段白名单(对应 walls 表列名)。
var wallAllowedSortFields = map[string]string{
	"floorId":   "floor_id",
	"wallType":  "type",
	"createdAt": "created_at",
}
