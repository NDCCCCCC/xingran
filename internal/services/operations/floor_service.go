package operations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// floorSyncBuildingTimeout 楼层变更后异步同步工位 building_id 超时
const floorSyncBuildingTimeout = 30 * time.Second

// ErrFloorNotInExpectedBuilding 乐观锁冲突：楼层已被其他请求变更
var ErrFloorNotInExpectedBuilding = errors.New("floor was modified by another request")

type FloorService interface {
	Create(ctx context.Context, floor *operations.OpsFloor) error
	Update(ctx context.Context, floor *operations.OpsFloor) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsFloor, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	GetTree(ctx context.Context) ([]FloorTreeNode, error)
	BatchDelete(ctx context.Context, ids []string) error
	// Statistics 楼层统计(专用 COUNT 聚合,不依赖分页列表)。
	Statistics(ctx context.Context) (*FloorStatisticsResult, error)
	// SearchFloorOptions 楼层下拉数据源(LIKE 模糊 + buildingId 筛选,LIMIT 50)。
	// 设计给前端 Select/AutoComplete/Cascader children 远程搜索。
	SearchFloorOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// FloorStatisticsResult 楼层统计结果(status: 0=正常 1=停用)。
type FloorStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // operations.FloorStatusNormal
	Inactive int64 `json:"inactive"` // operations.FloorStatusStopped
}

// Statistics 统计楼层(按 status 聚合,排除软删除)。
func (s *floorService) Statistics(ctx context.Context) (*FloorStatisticsResult, error) {
	var result FloorStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsFloor{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS active", int(operations.FloorStatusNormal)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS inactive", int(operations.FloorStatusStopped)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type FloorTreeNode struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	FloorNo    string          `json:"floorNo"`
	BuildingID string          `json:"buildingId"`
	Children   []FloorTreeNode `json:"children"`
}

type floorService struct {
	db   *gorm.DB
	repo *base.GORMRepository[operations.OpsFloor]
}

func NewFloorService(db *gorm.DB) FloorService {
	return &floorService{db: db, repo: base.NewGORMRepository[operations.OpsFloor](db)}
}

// listRepo 返回仓储实例。
// floor_cache_impl.go 装饰器经白盒字面量 &floorService{db: db} 构造内嵌 base
// （不经过 NewFloorService，repo 字段为零值），而 router.go 生产路径走
// NewFloorServiceWithCache——此处惰性兜底补建，保证 P7（零触碰 Phase 92 文件）
// 约束下装饰器委托 List/GetByID 不 nil-panic。GORMRepository 无状态，惰性
// 构造与构造函数初始化语义等价。
func (s *floorService) listRepo() *base.GORMRepository[operations.OpsFloor] {
	if s.repo == nil {
		return base.NewGORMRepository[operations.OpsFloor](s.db)
	}
	return s.repo
}

// floorAllowedSortFields 楼层可排序字段白名单。
// 因 List 有 LEFT JOIN buildings/files,字段值带表别名。
var floorAllowedSortFields = map[string]string{
	"name":         "ops_floors.name",
	"floorNo":      "ops_floors.floor_no",
	"orderNum":     "ops_floors.order_num",
	"area":         "ops_floors.area",
	"buildingName": "ops_buildings.name",
	"status":       "ops_floors.status",
	"createdAt":    "ops_floors.created_at",
}

func (s *floorService) Create(ctx context.Context, floor *operations.OpsFloor) error {
	if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(floor).Error; err != nil {
		// 检查是否是唯一约束错误（同一楼宇下楼层号重复）
		if isDuplicateKeyError(err) {
			// 检查是否存在被软删除的记录
			var softDeletedFloor operations.OpsFloor
			err := s.db.WithContext(ctx).Unscoped().Where("building_id = ? AND floor_no = ? AND deleted_at IS NOT NULL",
				floor.BuildingID, floor.FloorNo).First(&softDeletedFloor).Error

			if err == nil && softDeletedFloor.ID != "" {
				// 存在被软删除的记录，恢复它
				recoverErr := s.db.WithContext(ctx).Exec(`
					UPDATE ops_floors
					SET name = ?, area = ?, status = ?, remark = ?,
							plan_image_id = ?, plan_image_url = ?,
							deleted_at = NULL, updated_at = NOW()
					WHERE id = ?
				`, floor.Name, floor.Area, floor.Status, floor.Remark,
					floor.PlanImageID, floor.PlanImageUrl, softDeletedFloor.ID).Error

				if recoverErr != nil {
					return apperrors.Wrap(recoverErr, apperrors.CodeServerError, "恢复楼层失败")
				}
				// 将恢复的 ID 赋值给 floor，以便后续更新
				floor.ID = softDeletedFloor.ID
				return s.updateBuildingFloorCount(ctx, floor.BuildingID)
			}

			// 不存在被软删除的记录，返回真正错误
			return apperrors.Wrap(err, apperrors.CodeParamError, "该楼宇已存在此楼层号")
		}
		return err
	}
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

func (s *floorService) Update(ctx context.Context, floor *operations.OpsFloor) error {
	// 保存旧的 buildingID 用于后续同步。
	// 91-03 修复（原实现 bug）：oldBuildingID := floor.BuildingID 捕获的是待写入的
	// 新值（同一变量），下方 oldBuildingID != floor.BuildingID 恒 false，换楼时异步
	// 同步挂靠工位 building_id 的分支自仓库初始化起即死代码（Wave 0 无 service 级
	// 测试未能暴露）。此处从库中读取修改前的 building_id，使换楼同步语义真正生效。
	var oldBuildingID string
	if floor.ID != "" {
		var prev operations.OpsFloor
		if err := s.db.WithContext(ctx).Select("building_id").Where("id = ?", floor.ID).First(&prev).Error; err == nil {
			oldBuildingID = prev.BuildingID
		}
	}

	if floor.BuildingID != "" {
		if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
			return err
		}
	}

	// 换楼时使用乐观锁：只有楼层仍在旧楼宇才能更新，防止并发 race 导致后到的请求静默覆盖先到的请求。
	// V130R-07: 如果 building_id 有变化，WHERE 条件包含 oldBuildingID 确保原子性。
	if oldBuildingID != "" && floor.BuildingID != "" && oldBuildingID != floor.BuildingID {
		// 乐观锁 UPDATE：只有楼层仍在 oldBuildingID 时才允许更新
		result := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).
			Where("id = ? AND building_id = ?", floor.ID, oldBuildingID).
			Updates(map[string]interface{}{
				"building_id": floor.BuildingID,
				"name":        floor.Name,
				"floor_no":    floor.FloorNo,
				"area":        floor.Area,
				"order_num":   floor.OrderNum,
				"status":      floor.Status,
				"remark":      floor.Remark,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// 楼层已不在预期的旧楼宇中（被并发修改），返回乐观锁冲突错误
			return ErrFloorNotInExpectedBuilding
		}
	} else {
		// 无换楼，保留原 Save 语义（普通字段更新）
		if err := s.db.WithContext(ctx).Save(floor).Error; err != nil {
			return err
		}
	}

	// 如果楼宇ID发生了变化，同步更新相关工位的 building_id
	if oldBuildingID != "" && floor.BuildingID != "" && oldBuildingID != floor.BuildingID {
		// 异步同步工位 building_id（非阻塞，避免影响主流程性能）
		// 使用带超时的衍生上下文，避免长时间运行
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), floorSyncBuildingTimeout)
			defer cancel()
			// best-effort: 同步工位 building_id，忽略错误以避免阻塞主流程
			_ = s.syncWorkstationBuildingID(syncCtx, floor.ID, floor.BuildingID)
		}()
	}

	// 楼栋 total_floors 重算（修复漂移路径 A：floor Update 换楼 261013-lqb 之前全程零重算，
	// 91-03 修好了换楼工位同步但漏了计数）：
	//   ① 换楼后新旧楼栋计数都要重算——旧楼失一层、新楼得一层，只算一侧必漂移；
	//   ② 普通编辑也重算当前楼栋是自愈设计——Update 非事务（乐观锁已保证换楼 UPDATE 本身
	//      原子），重算幂等，任何部分失败后的重试走 Save 分支仍会触发重算，历史漂移楼栋
	//      在任意一次楼层编辑后归正；
	//   ③ updateBuildingFloorCount 用 Model(&OpsFloor{}) 实时 COUNT（自动软删过滤），
	//      幂等可重入——禁止改成 ±1 增量（会把历史脏值固化）。
	// error 上抛（对齐 Delete 路径 fail-loud 行为）。
	if floor.BuildingID != "" {
		if err := s.updateBuildingFloorCount(ctx, floor.BuildingID); err != nil {
			return err
		}
		if oldBuildingID != "" && oldBuildingID != floor.BuildingID {
			if err := s.updateBuildingFloorCount(ctx, oldBuildingID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *floorService) Delete(ctx context.Context, id string) error {
	var floor operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id = ?", id).Error; err != nil {
		return err
	}
	return s.updateBuildingFloorCount(ctx, floor.BuildingID)
}

func (s *floorService) GetByID(ctx context.Context, id string) (*operations.OpsFloor, error) {
	// repo.GetByID 有 scope 时按 Tabler 断言生成 ops_floors.id = ?（与现状逐字等价），
	// joinScope 携带 Select+Joins（:229-231 三段原样平移）。
	return s.listRepo().GetByID(ctx, id, s.joinScope())
}

// joinScope List/GetByID 共用的 Select+Joins（原 :229-231 三段逐字平移）。
//
// 前置 Count 的等价性：现状 Select+Joins 在 Count 之后应用；repo 化后 scope 前置——
// 两 JOIN 均按主键 N:1 LEFT JOIN 无行增殖，Count 时 Select 被替换为 count(*) 并在
// Find 恢复，Total 不变（91-01 spike 实证 + TestFloor91_List_Filters 守护）。
func (s *floorService) joinScope() base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select("ops_floors.*, ops_buildings.name as building_name, '/uploads/' || sys_files.storage_path as plan_image_url").
			Joins("LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = ops_floors.building_id").
			// plan_image_id 在 PG 是 uuid 类型(model tag type:uuid,与 sys_files.id 同型),
			// 直接 uuid=uuid 比较;包 CAST(... AS TEXT) 会变 text=uuid → SQLSTATE 42883。
			// SQLite 两侧均为 TEXT,直接比较同样成立(双方言安全)。
			Joins("LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id")
	}
}

// filterScope List 筛选条件（原 :187-201 四条件逐字平移，含 CAST 写法与
// orgId EXISTS 子查询）。
func (s *floorService) filterScope(params map[string]interface{}) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if name, ok := params["name"].(string); ok && name != "" {
			db = db.Where("ops_floors.name LIKE ?", "%"+name+"%")
		}
		if buildingId, ok := params["buildingId"].(string); ok && buildingId != "" {
			db = db.Where("ops_floors.building_id = ?", buildingId)
		}
		if orgId, ok := params["orgId"].(string); ok && orgId != "" {
			// 通过关联楼宇的 orgId 筛选（使用 EXISTS 子查询避免 JOIN 冲突）
			db = db.Where("EXISTS (SELECT 1 FROM ops_buildings b WHERE CAST(b.id AS TEXT) = ops_floors.building_id AND b.org_id = ? AND b.deleted_at IS NULL)", orgId)
		}

		// 状态筛选
		if status := extractIntParam(params, "status", -1); status >= 0 {
			db = db.Where("ops_floors.status = ?", status)
		}
		return db
	}
}

func (s *floorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	// v129-recheck WR-05: 内联字面量改引 pkg/constants（值不变,零行为变更）
	current := constants.DefaultCurrent
	pageSize := constants.DefaultPageSize
	if c, ok := params["current"].(int); ok {
		current = c
	} else if c, ok := params["current"].(float64); ok {
		current = int(c)
	}
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	} else if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	// 分页为无 clamp 内联断言语契约（P5 floor 行）：current/pageSize 任意值直传
	// base.PageParams，repo 直信入参；排序 B 型排他默认（白名单 + ops_floors.order_num ASC）。
	// floor 起链本就是 Model 形态，无 building/asset 的 F3 软删 Count 分歧。
	sortReq := extractSortRequest(params)
	result, err := s.listRepo().List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(params),
		s.joinScope(),
		base.SortScope(sortReq, floorAllowedSortFields, "ops_floors.order_num ASC"),
	)
	if err != nil {
		return nil, err
	}
	// 每行工位统计注入（只读增强）：类型断言失败或空页直接返回原 result。
	if floors, ok := result.List.([]operations.OpsFloor); ok && len(floors) > 0 {
		s.attachWorkstationStats(ctx, floors)
	}
	return result, nil
}

// attachWorkstationStats 批量回填楼层工位统计（单条 GROUP BY 聚合，零 N+1）。
//
// 状态语义（sys_workstation.status 独立枚举，非全局 0=正常/1=停用惯例）：
//
//	0=空闲(未使用) / 1=占用(已使用) / 2=维护
//
// 实现要点：
//   - floor_id 为 varchar(64) 存楼层 UUID 字符串，按页内 id 集合 IN 过滤 + GROUP BY，
//     无需 JOIN（规避 PG uuid/varchar 类型转换问题）。
//   - Model(&models.Workstation{}) 让 GORM 自动附加 deleted_at IS NULL 软删过滤。
//   - 状态值引用 models.WorkstationStatus* 常量（禁止魔法数字内联）。
//   - 无工位的楼层保持 WorkstationStats=nil（json omitempty 不出键，前端渲染 "-"）。
//   - 只读增强：聚合失败仅告警降级（行保持 nil），不影响 List 主流程。
func (s *floorService) attachWorkstationStats(ctx context.Context, floors []operations.OpsFloor) {
	floorIDs := make([]string, 0, len(floors))
	seen := make(map[string]struct{}, len(floors))
	for i := range floors {
		id := floors[i].ID
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		floorIDs = append(floorIDs, id)
	}
	if len(floorIDs) == 0 {
		return
	}

	var rows []struct {
		FloorID     string
		Total       int64
		Used        int64
		Unused      int64
		Maintenance int64
	}
	err := s.db.WithContext(ctx).
		Model(&models.Workstation{}).
		Select(
			"floor_id",
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS used", int(models.WorkstationStatusOccupied)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS unused", int(models.WorkstationStatusAvailable)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS maintenance", int(models.WorkstationStatusMaintain)),
		).
		Where("floor_id IN ?", floorIDs).
		Group("floor_id").
		Scan(&rows).Error
	if err != nil {
		logrus.WithError(err).Warn("[floor] 工位统计聚合失败，按无统计降级")
		return
	}

	statsByFloor := make(map[string]*operations.FloorWorkstationStats, len(rows))
	for _, r := range rows {
		statsByFloor[r.FloorID] = &operations.FloorWorkstationStats{
			Total:       r.Total,
			Used:        r.Used,
			Unused:      r.Unused,
			Maintenance: r.Maintenance,
		}
	}
	for i := range floors {
		if st, ok := statsByFloor[floors[i].ID]; ok {
			floors[i].WorkstationStats = st
		}
	}
}

// SearchFloorOptions 楼层下拉数据源(name LIKE 模糊 + buildingId/orgId/status 筛选,LIMIT 50)。
// 设计给前端 Select/AutoComplete/Cascader children 远程搜索。
//
// 与 List 语义对齐(同样的 WHERE 闭包),但只 SELECT id+name 两列且无分页无 JOIN,
// 保证 dropdown 单次响应 < 5KB。
func (s *floorService) SearchFloorOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("ops_floors").
		Select("ops_floors.id AS value, ops_floors.name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("ops_floors.name LIKE ?", "%"+name+"%")
	}
	if buildingId := extractStringParam(params, "buildingId"); buildingId != "" {
		query = query.Where("ops_floors.building_id = ?", buildingId)
	}
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		query = query.Where("EXISTS (SELECT 1 FROM ops_buildings b WHERE CAST(b.id AS TEXT) = ops_floors.building_id AND b.org_id = ? AND b.deleted_at IS NULL)", orgId)
	}
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("ops_floors.status = ?", status)
	}

	if err := query.Order("ops_floors.name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *floorService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
	type BuildingFloor struct {
		BuildingID   string
		BuildingName string
		FloorID      string
		FloorName    string
		FloorNo      string
	}

	var results []BuildingFloor
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			b.id as building_id,
			b.name as building_name,
			f.id as floor_id,
			f.name as floor_name,
			f.floor_no
		FROM ops_buildings b
		LEFT JOIN ops_floors f ON b.id = f.building_id
		WHERE b.deleted_at IS NULL
		ORDER BY b.order_num ASC, f.order_num ASC
	`).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	buildingMap := make(map[string]*FloorTreeNode)

	for _, r := range results {
		if _, exists := buildingMap[r.BuildingID]; !exists {
			buildingMap[r.BuildingID] = &FloorTreeNode{
				ID:       r.BuildingID,
				Name:     r.BuildingName,
				Children: []FloorTreeNode{},
			}
		}

		if r.FloorID != "" {
			buildingMap[r.BuildingID].Children = append(buildingMap[r.BuildingID].Children, FloorTreeNode{
				ID:         r.FloorID,
				Name:       r.FloorName,
				FloorNo:    r.FloorNo,
				BuildingID: r.BuildingID,
			})
		}
	}

	var tree []FloorTreeNode
	for _, node := range buildingMap {
		tree = append(tree, *node)
	}

	return tree, nil
}

func (s *floorService) BatchDelete(ctx context.Context, ids []string) error {
	var floors []operations.OpsFloor
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&floors).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&operations.OpsFloor{}, "id IN ?", ids).Error; err != nil {
		return err
	}

	// 遵循 Go 最佳实践：使用子查询批量更新，避免 N+1 问题
	// 不再循环调用 updateBuildingFloorCount，而是一次性更新所有相关楼宇
	sql := `
		UPDATE ops_buildings
		SET total_floors = (
			SELECT COUNT(*)
			FROM ops_floors
			WHERE ops_floors.building_id = ops_buildings.id
				AND ops_floors.deleted_at IS NULL
		)
		WHERE id IN (
			SELECT DISTINCT building_id
			FROM ops_floors
			WHERE building_id IN (
				SELECT building_id FROM ops_floors WHERE id IN ?
			)
		)
	`

	if err := s.db.WithContext(ctx).Exec(sql, ids).Error; err != nil {
		return err
	}

	return nil
}

func (s *floorService) validateBuilding(ctx context.Context, buildingID string) error {
	if buildingID == "" {
		return apperrors.ParamMissing("所属楼宇ID")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apperrors.BuildingNotFound()
	}
	return nil
}

func (s *floorService) updateBuildingFloorCount(ctx context.Context, buildingID string) error {
	return recomputeBuildingTotalFloorsByID(ctx, s.db, buildingID)
}

// recomputeBuildingTotalFloorsByID 单楼栋 total_floors 实时重算（包级共享 helper，
// 261014-ctp 抽取自 updateBuildingFloorCount，供 floor_service 与 building_service
// Create/Update 两侧共用）。
//
// 口径（与 recomputeAllBuildingTotalFloors 一致）：
//   - 实时 COUNT ops_floors 非软删行数（Model(&OpsFloor{}) 自动附软删过滤），
//     禁止 ±1 增量——会把历史脏值固化；
//   - 幂等可重入：重复执行结果一致，失败重试安全。
func recomputeBuildingTotalFloorsByID(ctx context.Context, db *gorm.DB, buildingID string) error {
	var count int64
	if err := db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("building_id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Update("total_floors", count).Error
}

// recomputeAllBuildingTotalFloors 全量重算所有未删楼栋的 total_floors 反范式列
// （SQL 形状拷贝 BatchDelete 的关联子查询 UPDATE，去掉 id 限定 + 加楼栋软删过滤）。
//
// 设计要点：
//   ① 供 Excel 导入事务收尾调用（building + floor 双实体，修漂移路径 B/C）——
//     building 导入直写用户上传的 totalFloors 值、floor 导入经通用 BatchUpsert
//     绕过 floorService.Create 的重算，两者都会污染派生列，导入完成统一归正；
//   ② 幂等可重入：重复执行结果一致，事务重试/并发导入均安全；
//   ③ 实时 COUNT 口径与 updateBuildingFloorCount 一致（ops_floors 非软删行数，
//     禁止 ±1 增量——会把历史脏值固化）；
//   ④ ops_buildings 量级小（百级行），全表重算无性能顾虑。
func recomputeAllBuildingTotalFloors(ctx context.Context, db *gorm.DB) error {
	sql := `
		UPDATE ops_buildings
		SET total_floors = (
			SELECT COUNT(*)
			FROM ops_floors
			WHERE ops_floors.building_id = ops_buildings.id
				AND ops_floors.deleted_at IS NULL
		)
		WHERE deleted_at IS NULL
	`
	return db.WithContext(ctx).Exec(sql).Error
}

// syncWorkstationBuildingID 同步更新工位的楼宇ID
// 当楼层所属楼宇变更时，更新该楼层下所有工位的 building_id
func (s *floorService) syncWorkstationBuildingID(ctx context.Context, floorID string, newBuildingID string) error {
	// 更新该楼层下所有工位的 building_id
	sql := `
		UPDATE sys_workstation
		SET building_id = ?
		WHERE floor_id = ?
			AND (building_id IS NULL OR building_id != ?);
	`
	if err := s.db.WithContext(ctx).Exec(sql, newBuildingID, floorID, floorID).Error; err != nil {
		return err
	}

	return nil
}

// isDuplicateKeyError 检查是否是 PostgreSQL 唯一约束错误
//
// P2 fix: 改用结构化错误类型(pgconn.PgError.Code == "23505")替代
// 字符串匹配 ("duplicate key" / "unique constraint" / "23505").
// 后者:
// - 受 DB 错误信息本地化影响(中文版/i18n 中可能完全不出现这些英文 token)
// - 不同 PG 驱动可能返回略不同 wording
// - "23505" 子串在错误中可能巧合出现但不是真正约束错误
//
// pgconn.PgError 是 jackc/pgx 标准化的错误类型,Code 字段就是 PG SQLSTATE,
// 23505 = unique_violation,精确无歧义。
// 仍保留字符串 fallback 作为退路(SQLite 测试、非 pgconn 错误)。
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// 1) 优先用结构化错误类型 (生产 PostgreSQL 路径)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	// 2) Fallback 字符串匹配 (SQLite 测试 / 第三方 wrapper)
	// P2 fix: 用 strings.Contains + ToLower 替代手写 containsIgnoreCase,
	// 删除 30+ 行手写循环,与 stdlib 等价但更简洁。
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "23505")
}
