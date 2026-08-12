package operations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// floorSyncBuildingTimeout 楼层变更后异步同步工位 building_id 超时
const floorSyncBuildingTimeout = 30 * time.Second

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
	Active   int64 `json:"active"`   // status = 0
	Inactive int64 `json:"inactive"` // status = 1
}

// Statistics 统计楼层(按 status 聚合,排除软删除)。
func (s *floorService) Statistics(ctx context.Context) (*FloorStatisticsResult, error) {
	var result FloorStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&operations.OpsFloor{}).
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS active",
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS inactive",
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
	db *gorm.DB
}

func NewFloorService(db *gorm.DB) FloorService {
	return &floorService{db: db}
}

// floorAllowedSortFields 楼层可排序字段白名单。
// 因 List 有 LEFT JOIN buildings/files,字段值带表别名。
var floorAllowedSortFields = map[string]string{
	"name":          "ops_floors.name",
	"floorNo":       "ops_floors.floor_no",
	"orderNum":      "ops_floors.order_num",
	"area":          "ops_floors.area",
	"buildingName":  "ops_buildings.name",
	"status":        "ops_floors.status",
	"createdAt":     "ops_floors.created_at",
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
	// 保存旧的 buildingID 用于后续同步
	oldBuildingID := floor.BuildingID

	if floor.BuildingID != "" {
		if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
			return err
		}
	}

	// 保存楼层（这会触发 updateBuildingFloorCount）
	if err := s.db.WithContext(ctx).Save(floor).Error; err != nil {
		return err
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
	var floor operations.OpsFloor
	err := s.db.WithContext(ctx).
		Select("ops_floors.*, ops_buildings.name as building_name, '/uploads/' || sys_files.storage_path as plan_image_url").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid").
		Joins("LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id::uuid").
		Where("ops_floors.id = ?", id).
		First(&floor).Error
	if err != nil {
		return nil, err
	}
	return &floor, nil
}

func (s *floorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	var total int64
	var list []operations.OpsFloor

	query := s.db.WithContext(ctx).Model(&operations.OpsFloor{})

	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("ops_floors.name LIKE ?", "%"+name+"%")
	}
	if buildingId, ok := params["buildingId"].(string); ok && buildingId != "" {
		query = query.Where("ops_floors.building_id = ?", buildingId)
	}
	if orgId, ok := params["orgId"].(string); ok && orgId != "" {
		// 通过关联楼宇的 orgId 筛选（使用 EXISTS 子查询避免 JOIN 冲突）
		query = query.Where("EXISTS (SELECT 1 FROM ops_buildings b WHERE b.id::text = ops_floors.building_id AND b.org_id = ? AND b.deleted_at IS NULL)", orgId)
	}

	// 状态筛选
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("ops_floors.status = ?", status)
	}

	current := 1
	pageSize := 10
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

	// 用户排序(白名单,带表别名);无 OrderByColumn 时保留原 order_num ASC 默认
	sortReq := extractSortRequest(params)
	query = base.ApplySort(query, sortReq, floorAllowedSortFields)
	if sortReq.OrderByColumn == "" {
		query = query.Order("ops_floors.order_num ASC")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (current - 1) * pageSize
	if err := query.
		Select("ops_floors.*, ops_buildings.name as building_name, '/uploads/' || sys_files.storage_path as plan_image_url").
		Joins("LEFT JOIN ops_buildings ON ops_buildings.id = ops_floors.building_id::uuid").
		Joins("LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id::uuid").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
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
		query = query.Where("EXISTS (SELECT 1 FROM ops_buildings b WHERE b.id::text = ops_floors.building_id AND b.org_id = ? AND b.deleted_at IS NULL)", orgId)
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
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsFloor{}).Where("building_id = ?", buildingID).Count(&count).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("id = ?", buildingID).Update("total_floors", count).Error
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
