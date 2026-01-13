package operations

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
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
	db *gorm.DB
}

// NewBuildingService 创建楼宇服务实例
func NewBuildingService(db *gorm.DB) BuildingService {
	return &buildingService{db: db}
}

// Create 创建楼宇
func (s *buildingService) Create(ctx context.Context, building *operations.OpsBuilding) error {
	// 检查编码唯一性
	if err := s.checkCodeUnique(ctx, building.Code, ""); err != nil {
		return err
	}

	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(building).Error
}

// Update 更新楼宇
func (s *buildingService) Update(ctx context.Context, building *operations.OpsBuilding) error {
	// 检查编码唯一性
	if err := s.checkCodeUnique(ctx, building.Code, building.ID); err != nil {
		return err
	}

	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Save(building).Error
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
	var total int64
	var list []operations.OpsBuilding

	query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{})

	// 添加筛选条件
	if name, ok := params["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if code, ok := params["code"].(string); ok && code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if orgId, ok := params["orgId"].(string); ok && orgId != "" {
		query = query.Where("org_id = ?", orgId)
	}
	if status, ok := params["status"].(int); ok {
		query = query.Where("status = ?", status)
	}
	if status, ok := params["status"].(float64); ok {
		query = query.Where("status = ?", int(status))
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	current := 1
	pageSize := 10
	if c, ok := params["current"].(int); ok {
		current = c
	}
	if c, ok := params["current"].(float64); ok {
		current = int(c)
	}
	if ps, ok := params["pageSize"].(int); ok {
		pageSize = ps
	}
	if ps, ok := params["pageSize"].(float64); ok {
		pageSize = int(ps)
	}

	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("order_num ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// BatchDelete 批量删除楼宇
func (s *buildingService) BatchDelete(ctx context.Context, ids []string) error {
	return s.db.WithContext(ctx).Delete(&operations.OpsBuilding{}, "id IN ?", ids).Error
}

// checkCodeUnique 检查编码唯一性
func (s *buildingService) checkCodeUnique(ctx context.Context, code string, excludeID string) error {
	var count int64
	query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{}).Where("code = ?", code)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("楼宇编码已存在")
	}
	return nil
}

// validateOrg 验证机构存在性
func (s *buildingService) validateOrg(ctx context.Context, orgID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", orgID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("所属机构不存在")
	}
	return nil
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}
