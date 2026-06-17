package operations

import (
	"context"
	"regexp"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
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
		uuidValidator: constants.UuidPattern,
	}
}

// buildingAllowedSortFields 楼宇可排序字段白名单(对应 ops_buildings 表列名)。
var buildingAllowedSortFields = map[string]string{
	"name":       "name",
	"address":    "address",
	"level":      "level",
	"orgId":      "org_id",
	"orderNum":   "order_num",
	"status":     "status",
	"createdAt":  "created_at",
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
