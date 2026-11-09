package operations

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// ServerRoomService 机房服务接口
type ServerRoomService interface {
	Create(ctx context.Context, room *operations.OpsServerRoom) error
	Update(ctx context.Context, room *operations.OpsServerRoom) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsServerRoom, error)
	List(ctx context.Context, req requests.ServerRoomListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	// Statistics 机房统计(专用 COUNT 聚合,不依赖分页列表)。
	Statistics(ctx context.Context) (*ServerRoomStatisticsResult, error)
	// SearchServerRoomOptions 机房下拉数据源(LIKE 模糊 + floorId 筛选,LIMIT 50)。
	SearchServerRoomOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// ServerRoomStatisticsResult 机房统计结果(status: 0=正常 1=停用)。
type ServerRoomStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // operations.RoomStatusNormal
	Inactive int64 `json:"inactive"` // operations.RoomStatusStopped
}

// Statistics 统计机房(按 status 聚合,排除软删除)。
func (s *serverRoomService) Statistics(ctx context.Context) (*ServerRoomStatisticsResult, error) {
	var result ServerRoomStatisticsResult
	err := applyEntityReadScope(
		ScopedQuery(ctx, s.db).Model(&operations.OpsServerRoom{}), ctx,
		scopeEntityServerRoom, "ops_server_rooms",
	).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS active", int(operations.RoomStatusNormal)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS inactive", int(operations.RoomStatusStopped)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type serverRoomService struct {
	// repo 承接六方法 CRUD 管道（Phase 91-04 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[operations.OpsServerRoom]
	// db 保留给 Statistics / SearchServerRoomOptions / validateFloor /
	// getDeptAndChildDeptIDs（D-04 留 service 的直用场景）
	db *gorm.DB
}

// NewServerRoomService 创建机房服务实例
func NewServerRoomService(db *gorm.DB) ServerRoomService {
	return &serverRoomService{
		repo: base.NewGORMRepository[operations.OpsServerRoom](db),
		db:   db,
	}
}

// serverRoomAllowedSortFields 机房可排序字段白名单。
// 因 List 有 LEFT JOIN buildings/floors,字段值必须带表别名。
var serverRoomAllowedSortFields = map[string]string{
	"name":      "ops_server_rooms.name",
	"status":    "ops_server_rooms.status",
	"createdAt": "ops_server_rooms.created_at",
	"roomNo":    "ops_server_rooms.room_no",
}

// Create 创建机房
func (s *serverRoomService) Create(ctx context.Context, room *operations.OpsServerRoom) error {
	// Phase 133 / DS-05：单条写越界 403（提交侧，按 building_id 归属链解析有效机构）。
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityServerRoom, room.BuildingID); err != nil {
		return err
	}
	// 验证楼层存在性
	if err := s.validateFloor(ctx, room.FloorID); err != nil {
		return err
	}

	return s.repo.Create(ctx, room)
}

// Update 更新机房
func (s *serverRoomService) Update(ctx context.Context, room *operations.OpsServerRoom) error {
	// Phase 133 / DS-05：单条写越界 403（改前侧 + 提交侧双向校验，防搬移）。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityServerRoom, room.ID); err != nil {
		return err
	}
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityServerRoom, room.BuildingID); err != nil {
		return err
	}
	// 验证楼层存在性
	if err := s.validateFloor(ctx, room.FloorID); err != nil {
		return err
	}

	return s.repo.Update(ctx, room)
}

// Delete 删除机房
func (s *serverRoomService) Delete(ctx context.Context, id string) error {
	// Phase 133 / DS-05：单条写越界 403（改前侧按资源归属校验）。
	if err := checkEntityWriteScope(ctx, s.db, scopeEntityServerRoom, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// GetByID 根据ID获取机房
func (s *serverRoomService) GetByID(ctx context.Context, id string) (*operations.OpsServerRoom, error) {
	// Phase 133 / DS-05：读端点按楼宇链可见集校验，越界 → record not found。
	return s.repo.GetByID(ctx, id, entityReadScope(ctx, scopeEntityServerRoom, "ops_server_rooms"))
}

// joinScope List 的 Select + buildings/floors LEFT JOIN scope（常量逐字平移）。
//
// 迁移前 Select+Joins 即在 Count 之前（"必须在 Count 之前执行 JOIN,这样后面的
// 条件才能使用 JOIN 的表"），repo 化后 joinScope 前置——现状本就全量前置，
// 语义严格等价（Count 时自定义 Select 被临时替换为 count(*) 并在 Find 恢复，
// TestBase91 契约锁定；两 JOIN 均按主键 N:1 无行增殖）。
func (s *serverRoomService) joinScope() base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		// JOIN 关联表获取楼宇名称和楼层名称（需要类型转换：varchar -> uuid）
		return db.
			Select("ops_server_rooms.*, b.name as building_name, f.name as floor_name, f.floor_no").
			Joins("LEFT JOIN ops_buildings b ON CAST(b.id AS TEXT) = ops_server_rooms.building_id").
			Joins("LEFT JOIN ops_floors f ON CAST(f.id AS TEXT) = ops_server_rooms.floor_id")
	}
}

// filterScope List 的四个本表过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
// 使用表前缀避免与 JOIN 的表产生列名歧义。
func (s *serverRoomService) filterScope(req requests.ServerRoomListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.Name != "" {
			db = db.Where("ops_server_rooms.name LIKE ?", "%"+req.Name+"%")
		}
		if req.BuildingID != "" {
			db = db.Where("ops_server_rooms.building_id = ?", req.BuildingID)
		}
		if req.FloorID != "" {
			db = db.Where("ops_server_rooms.floor_id = ?", req.FloorID)
		}
		if req.HasStatus() {
			db = db.Where("ops_server_rooms.status = ?", req.GetStatus(0))
		}
		return db
	}
}

// List 查询机房列表（CRUD 管道经 base.GORMRepository 执行）。
//
// scopes 顺序保持原代码顺序（P8）：joinScope（Joins 在前）→ filterScope →
// orgId 过滤（引用 joined 表列 b.org_id，必须后于 joinScope）→ base.SortScope
// （B 型排他默认）。orgId 空命中早退分支保留在函数体内、scope 构造之前。
func (s *serverRoomService) List(ctx context.Context, req requests.ServerRoomListRequest) (*PageResult, error) {
	current, pageSize := req.GetPagination()

	// 部门筛选（包含子部门）
	var deptIDs []string
	if req.OrgID != "" {
		deptIDs = s.getDeptAndChildDeptIDs(ctx, req.OrgID)
		if len(deptIDs) == 0 {
			// 如果没有找到有效的部门ID，返回空结果
			return &PageResult{
				List:     []operations.OpsServerRoom{},
				Total:    0,
				Current:  current,
				PageSize: pageSize,
			}, nil
		}
	}

	scopes := []base.Scope{s.joinScope(), s.filterScope(req)}
	if req.OrgID != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			// 通过楼宇的 org_id 进行筛选（引用 JOIN 的表，保持原顺序在后）。
			// 请求维度（业务筛选）与下方 ScopeResult 维度（强制权限边界）AND 叠加。
			return db.Where("b.org_id IN ?", deptIDs)
		})
	}
	// Phase 133 / DS-05：楼宇链可见集（强制权限边界，后端注入、不信任前端 orgId）。
	// 与上方请求 orgId 业务筛选 AND 叠加；All / 未注入短路零额外 SQL（超管零回归）。
	scopes = append(scopes, entityReadScope(ctx, scopeEntityServerRoom, "ops_server_rooms"))
	scopes = append(scopes, base.SortScope(req.BaseListRequest, serverRoomAllowedSortFields, "ops_server_rooms.created_at DESC"))
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize}, scopes...)
}

// BatchDelete 批量删除机房
func (s *serverRoomService) BatchDelete(ctx context.Context, ids []string) error {
	// Phase 133 / DS-05：批量按可见集裁剪（不整批 403），越界行不进入删除集合。
	visible, _, err := filterEntityVisibleIDs(ctx, s.db, scopeEntityServerRoom, ids)
	if err != nil {
		return err
	}
	if len(visible) == 0 {
		return nil
	}
	return s.repo.BatchDelete(ctx, visible)
}

// SearchServerRoomOptions 机房下拉数据源(name LIKE 模糊 + buildingId/floorId/status/orgId 筛选,LIMIT 50)。
// 与 List 同款 WHERE 语义;orgId 经 sys_dept.ancestors 包含子部门。
func (s *serverRoomService) SearchServerRoomOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := applyEntityReadScope(
		ScopedQuery(ctx, s.db).Table("ops_server_rooms"), ctx,
		scopeEntityServerRoom, "ops_server_rooms",
	).
		Select("ops_server_rooms.id AS value, ops_server_rooms.name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("ops_server_rooms.name LIKE ?", "%"+name+"%")
	}
	if buildingId := extractStringParam(params, "buildingId"); buildingId != "" {
		query = query.Where("ops_server_rooms.building_id = ?", buildingId)
	}
	if floorId := extractStringParam(params, "floorId"); floorId != "" {
		query = query.Where("ops_server_rooms.floor_id = ?", floorId)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("ops_server_rooms.status = ?", status)
	}
	// orgId 部门筛选（请求维度，业务筛选）：与 List 同口径，走
	// getDeptAndChildDeptIDs（BuildDeptRecursiveFilter 单一真实源）展开子树后 IN。
	// 与上方注入的 ScopeResult 可见集（强制权限边界）AND 叠加。
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		deptIDs := s.getDeptAndChildDeptIDs(ctx, orgId)
		if len(deptIDs) == 0 {
			return []DropdownOption{}, nil
		}
		query = query.Where("EXISTS (SELECT 1 FROM ops_buildings b WHERE CAST(b.id AS TEXT) = ops_server_rooms.building_id AND b.org_id IN ? AND b.deleted_at IS NULL)", deptIDs)
	}

	if err := query.Order("ops_server_rooms.name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// validateFloor 验证楼层存在性
func (s *serverRoomService) validateFloor(ctx context.Context, floorID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("id = ?", floorID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apperrors.FloorNotFound()
	}
	return nil
}

// getDeptAndChildDeptIDs 展开「请求参数 orgId」的业务筛选部门集。
//
// Phase 133 收敛说明：这里不是「登录人可见部门集」的展开，而是既有请求参数
// req.OrgID 的业务筛选（与强制权限边界 ScopeResult 正交，二者 AND 叠加）。
// 展开逻辑本身复用 BuildDeptRecursiveFilter 单一真实源（dept_filter.go），
// 不内联复制四条件、不新增第 5 套子树实现；登录人可见集的展开只由
// pkg/datascope.Resolve 承担，读端点经 applyEntityReadScope / wrapScopeExists
// 注入。List 与 SearchServerRoomOptions 两处共用本 helper，口径已对齐。
func (s *serverRoomService) getDeptAndChildDeptIDs(ctx context.Context, orgId string) []string {
	var deptIDs []string

	err := s.db.WithContext(ctx).Table("sys_dept").
		Scopes(BuildDeptRecursiveFilter(orgId, "id")).
		Pluck("id", &deptIDs).Error

	if err != nil || len(deptIDs) == 0 {
		return []string{}
	}

	return deptIDs
}
