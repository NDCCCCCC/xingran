package operations

import (
	"context"
	"regexp"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// BuildingServiceTypeSafe 楼宇服务接口（类型安全版本）
// 这是新的类型安全接口定义，使用专门的请求结构体替代 map[string]interface{}
type BuildingServiceTypeSafe interface {
	Create(ctx context.Context, building *operations.OpsBuilding) error
	Update(ctx context.Context, building *operations.OpsBuilding) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsBuilding, error)
	// List 使用类型安全的请求结构体
	List(ctx context.Context, req requests.BuildingListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

type buildingServiceTypeSafe struct {
	db            *gorm.DB
	uuidValidator *regexp.Regexp
}

// NewBuildingServiceTypeSafe 创建楼宇服务实例（类型安全版本）
func NewBuildingServiceTypeSafe(db *gorm.DB) BuildingServiceTypeSafe {
	return &buildingServiceTypeSafe{
		db:            db,
		uuidValidator: constants.UuidPattern,
	}
}

// Create 创建楼宇
func (s *buildingServiceTypeSafe) Create(ctx context.Context, building *operations.OpsBuilding) error {
	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(building).Error
}

// Update 更新楼宇
func (s *buildingServiceTypeSafe) Update(ctx context.Context, building *operations.OpsBuilding) error {
	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	// 使用 Omit 排除 CreatedAt 字段，只更新其他字段
	// UpdatedAt 会被 GORM 自动更新
	return s.db.WithContext(ctx).Omit("CreatedAt").Save(building).Error
}

// Delete 删除楼宇
func (s *buildingServiceTypeSafe) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsBuilding{}, "id = ?", id).Error
}

// GetByID 根据ID获取楼宇
func (s *buildingServiceTypeSafe) GetByID(ctx context.Context, id string) (*operations.OpsBuilding, error) {
	var building operations.OpsBuilding
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&building).Error
	if err != nil {
		return nil, err
	}
	return &building, nil
}

// List 查询楼宇列表（类型安全版本）
func (s *buildingServiceTypeSafe) List(ctx context.Context, req requests.BuildingListRequest) (*PageResult, error) {
	query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{})

	// 应用筛选条件 - 现在不需要类型断言，直接访问字段
	query = s.applyFiltersTypeSafe(query, req)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 应用分页 - 使用请求结构体的方法
	offset := req.GetOffset()
	_, pageSize := req.GetPagination()

	var list []operations.OpsBuilding
	if err := query.Offset(offset).Limit(pageSize).Order("order_num ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	current, _ := req.GetPagination()
	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// applyFiltersTypeSafe 应用查询筛选条件（类型安全版本）
func (s *buildingServiceTypeSafe) applyFiltersTypeSafe(query *gorm.DB, req requests.BuildingListRequest) *gorm.DB {
	// 名称筛选 - 不需要类型断言！
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// 部门筛选（包含子部门）
	if req.OrgID != "" {
		query = s.applyDeptFilter(query, req.OrgID)
	}

	// 状态筛选 - 使用 HasStatus() 方法
	if req.HasStatus() {
		query = query.Where("status = ?", req.GetStatus(0))
	}

	return query
}

// applyDeptFilter 应用部门筛选（包含子部门）
func (s *buildingServiceTypeSafe) applyDeptFilter(query *gorm.DB, orgId string) *gorm.DB {
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
func (s *buildingServiceTypeSafe) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsBuilding{}, "id IN ?", ids).Error
}

// validateOrg 验证机构存在性
func (s *buildingServiceTypeSafe) validateOrg(ctx context.Context, orgID string) error {
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
