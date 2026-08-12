package operations

import (
	"context"
	"regexp"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// BuildingService 楼宇服务接口
type BuildingService interface {
	Create(ctx context.Context, building *operations.OpsBuilding) error
	Update(ctx context.Context, building *operations.OpsBuilding) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsBuilding, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	// Statistics 楼宇统计(专用 COUNT 聚合,不依赖分页列表;复用 List 筛选支持按 name/orgId/status 统计)。
	Statistics(ctx context.Context, params map[string]interface{}) (*BuildingStatisticsResult, error)
	// SearchBuildingOptions 楼宇下拉数据源(LIKE 模糊 + orgId 部门筛选,LIMIT 50)。
	// 设计给前端 Select/AutoComplete 用,不支持分页;name="" 时走 5min Redis 缓存。
	SearchBuildingOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error)
}

// BuildingStatisticsResult 楼宇统计结果(status: 0=正常 1=停用)。
type BuildingStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // status = 0
	Inactive int64 `json:"inactive"` // status = 1
}

// Statistics 统计楼宇(按 status 聚合,排除软删除;复用 applyFilters 支持筛选,与 List 筛选语义一致)。
func (s *buildingService) Statistics(ctx context.Context, params map[string]interface{}) (*BuildingStatisticsResult, error) {
	var result BuildingStatisticsResult
	query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{})
	query = s.applyFilters(query, params)
	err := query.
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

type buildingService struct {
	db            *gorm.DB
	codeGenerator *CodeGenerator
	uuidValidator *regexp.Regexp
}

// NewBuildingService 创建楼宇服务实例
func NewBuildingService(db *gorm.DB) BuildingService {
	return &buildingService{
		db:            db,
		codeGenerator: NewCodeGenerator(db),
		uuidValidator: constants.UUIDPattern,
	}
}

// buildingAllowedSortFields 楼宇可排序字段白名单(对应 ops_buildings 表列名)。
var buildingAllowedSortFields = map[string]string{
	"name":        "name",
	"address":     "address",
	"level":       "level",
	"orgId":       "org_id",
	"orderNum":    "order_num",
	"totalFloors": "total_floors",
	"status":      "status",
	"createdAt":   "created_at",
}

// Create 创建楼宇
func (s *buildingService) Create(ctx context.Context, building *operations.OpsBuilding) error {
	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	// 验证楼宇名称唯一性（同一机构下不能有同名楼宇）
	if err := s.validateNameUnique(ctx, building.OrgID, building.Name, ""); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(building).Error
}

// Update 更新楼宇
func (s *buildingService) Update(ctx context.Context, building *operations.OpsBuilding) error {
	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	// 验证楼宇名称唯一性（同一机构下不能有同名楼宇，排除自身）
	if err := s.validateNameUnique(ctx, building.OrgID, building.Name, building.ID); err != nil {
		return err
	}

	// 使用 Omit 排除 CreatedAt 字段，只更新其他字段
	// UpdatedAt 会被 GORM 自动更新
	return s.db.WithContext(ctx).Omit("CreatedAt").Save(building).Error
}

// Delete 删除楼宇
func (s *buildingService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsBuilding{}, "id = ?", id).Error
}

// GetByID 根据ID获取楼宇
func (s *buildingService) GetByID(ctx context.Context, id string) (*operations.OpsBuilding, error) {
	var building operations.OpsBuilding
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&building).Error
	if err != nil {
		return nil, err
	}
	return &building, nil
}

// List 查询楼宇列表
func (s *buildingService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	query := s.db.WithContext(ctx).Table("ops_buildings")

	// 应用筛选条件
	query = s.applyFilters(query, params)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 应用分页
	pagination := extractPagination(params)
	offset := calculateOffset(pagination)

	// 应用用户排序(白名单);无 OrderByColumn 时保留 order_num ASC 默认
	sortReq := extractSortRequest(params)
	query = base.ApplySort(query, sortReq, buildingAllowedSortFields)
	if sortReq.OrderByColumn == "" {
		query = query.Order("order_num ASC")
	}

	// 附带工位计数(子查询,供 building-spaces 概览卡片;TotalFloors 为后端维护字段无需子查询)
	query = query.Select(`ops_buildings.*, (SELECT COUNT(*) FROM sys_workstation ws JOIN ops_floors f ON f.id = ws.floor_id::uuid WHERE f.building_id::uuid = ops_buildings.id AND ws.deleted_at IS NULL AND f.deleted_at IS NULL) AS workstation_count`)
	var list []operations.OpsBuilding
	if err := query.Offset(offset).Limit(pagination.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}

// applyFilters 应用查询筛选条件
func (s *buildingService) applyFilters(query *gorm.DB, params map[string]interface{}) *gorm.DB {
	// 名称筛选
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 部门筛选（包含子部门）
	if orgId, ok := params["orgId"].(string); ok && orgId != "" {
		query = s.applyDeptFilter(query, orgId)
	}

	// 状态筛选
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("status = ?", status)
	}

	return query
}

// applyDeptFilter 应用部门筛选（包含子部门）
func (s *buildingService) applyDeptFilter(query *gorm.DB, orgId string) *gorm.DB {
	var deptIDs []string

	// 查询该部门及其所有子部门的ID
	err := s.db.Table("sys_dept").
		Where("id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?",
			orgId, "%,"+orgId+",%", "%,"+orgId, orgId).
		Pluck("id", &deptIDs).Error

	if err != nil || len(deptIDs) == 0 {
		return query.Where("1 = 0")
	}

	return query.Where("org_id IN ?", deptIDs)
}

// BatchDelete 批量删除楼宇
func (s *buildingService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsBuilding{}, "id IN ?", ids).Error
}

// validateOrg 验证机构存在性
func (s *buildingService) validateOrg(ctx context.Context, orgID string) error {
	if orgID == "" {
		return nil
	}

	// 验证UUID格式
	if !s.uuidValidator.MatchString(orgID) {
		return apperrors.BuildingOrgInvalidWithMsg("所属机构ID格式无效：必须是有效的UUID格式")
	}

	// 验证机构是否存在
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", orgID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return apperrors.BuildingOrgInvalidWithMsg("所属机构不存在")
	}

	return nil
}

// validateNameUnique 验证楼宇名称唯一性（同一机构下不能有同名楼宇）
func (s *buildingService) validateNameUnique(ctx context.Context, orgID, name string, excludeID string) error {
	var count int64
	query := s.db.WithContext(ctx).Table("ops_buildings").
		Where("org_id = ? AND name = ?", orgID, name)

	// 更新时排除当前记录
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return apperrors.BuildingExistsWithMsg("同一机构下已存在同名楼宇")
	}

	return nil
}

// SearchBuildingOptions 楼宇下拉数据源(name LIKE 模糊 + orgId 部门含子部门,LIMIT 50)。
// 设计给前端 Select/AutoComplete 远程搜索;keyword 命中 >50 后截断,前端可继续 narrow。
//
// 不复用 buildingService.applyFilters:因下拉只需要 id+name 两列且要 ORDER BY name ASC,
// 而 applyFilters 不控制排序;直接手写 SQL 更清晰且无歧义。
func (s *buildingService) SearchBuildingOptions(ctx context.Context, params map[string]interface{}) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("ops_buildings").
		Select("id AS value, name AS label").
		Limit(DropdownMaxRows)

	if name := extractStringParam(params, "name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 部门筛选含子部门:复用 applyDeptFilter 的语义
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		var deptIDs []string
		err := s.db.Table("sys_dept").
			Where("id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?",
				orgId, "%,"+orgId+",%", "%,"+orgId, orgId).
			Pluck("id", &deptIDs).Error
		if err != nil {
			return nil, err
		}
		if len(deptIDs) == 0 {
			// 没有匹配的部门,返回空集而非全集
			return result, nil
		}
		query = query.Where("org_id IN ?", deptIDs)
	}

	// 状态筛选(0=正常 1=停用)
	if status := extractIntParam(params, "status", -1); status >= 0 {
		query = query.Where("status = ?", status)
	}

	// 按名称排序,保证首屏选项可预期
	if err := query.Order("name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}
