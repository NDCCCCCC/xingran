package addomain

import (
	"context"
	"fmt"
	"strings"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// DeptGroupMappingService 部门组映射服务接口
type DeptGroupMappingService interface {
	// CreateMapping 创建部门组映射
	CreateMapping(ctx context.Context, req *CreateMappingRequest) (*models.DeptGroupMapping, error)

	// UpdateMapping 更新部门组映射
	UpdateMapping(ctx context.Context, id string, req *UpdateMappingRequest) error

	// DeleteMapping 删除部门组映射（软删除）
	DeleteMapping(ctx context.Context, id string) error

	// GetMapping 获取单个映射
	GetMapping(ctx context.Context, id string) (*models.DeptGroupMapping, error)

	// ListMappings 查询映射列表
	ListMappings(ctx context.Context, req *ListMappingsRequest) (*ListMappingsResponse, error)

	// AutoMapDepartment 自动为部门创建映射（查找cxhub-{dept}组）
	AutoMapDepartment(ctx context.Context, deptID, configID string) (*models.DeptGroupMapping, error)

	// GetMappingByDept 根据部门ID获取映射
	GetMappingByDept(ctx context.Context, deptID string) (*models.DeptGroupMapping, error)
}

// deptGroupMappingService 服务实现
type deptGroupMappingService struct {
	db                     *gorm.DB
	groupManagementService GroupManagementService
}

// NewDeptGroupMappingService 创建服务实例
func NewDeptGroupMappingService(db *gorm.DB) DeptGroupMappingService {
	return &deptGroupMappingService{
		db:                     db,
		groupManagementService: NewGroupManagementService(db),
	}
}

// CreateMappingRequest 创建映射请求
type CreateMappingRequest struct {
	DeptID      string                `json:"deptId" binding:"required"`
	ADGroupID   string                `json:"adGroupId" binding:"required"`
	ADConfigID  string                `json:"adConfigId" binding:"required"`
	MappingType models.MappingType    `json:"mappingType"`
	SyncEnabled bool                  `json:"syncEnabled"`
	CreatedBy   string                `json:"createdBy"`
}

// UpdateMappingRequest 更新映射请求
type UpdateMappingRequest struct {
	MappingStatus *models.MappingStatus `json:"mappingStatus"`
	SyncEnabled   *bool                 `json:"syncEnabled"`
	UpdatedBy     string                `json:"updatedBy"`
}

// ListMappingsRequest 查询映射列表请求
type ListMappingsRequest struct {
	ADConfigID    *string                 `json:"adConfigId,omitempty"`
	DeptID        *string                 `json:"deptId,omitempty"`
	MappingType   *models.MappingType     `json:"mappingType,omitempty"`
	MappingStatus *models.MappingStatus   `json:"mappingStatus,omitempty"`
	GroupName     *string                 `json:"groupName,omitempty"`
	Current       int                     `json:"current"`
	PageSize      int                     `json:"pageSize"`
}

// ListMappingsResponse 映射列表响应
type ListMappingsResponse struct {
	Total int64                      `json:"total"`
	List  []models.DeptGroupMapping  `json:"list"`
}

// CreateMapping 创建部门组映射
func (s *deptGroupMappingService) CreateMapping(ctx context.Context, req *CreateMappingRequest) (*models.DeptGroupMapping, error) {
	// 1. 验证部门是否存在
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", req.DeptID).First(&dept).Error; err != nil {
		return nil, fmt.Errorf("部门不存在")
	}

	// 2. 验证AD组是否存在
	var group models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", req.ADGroupID).First(&group).Error; err != nil {
		return nil, fmt.Errorf("AD组不存在")
	}

	// 3. 验证配置是否存在
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", req.ADConfigID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}

	// 4. 检查是否已存在映射
	var existing models.DeptGroupMapping
	err := s.db.WithContext(ctx).
		Where("dept_id = ? AND deleted_at IS NULL", req.DeptID).
		First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("该部门已存在映射关系")
	}

	// 5. 创建映射
	mapping := &models.DeptGroupMapping{
		DeptID:        req.DeptID,
		ADGroupID:     req.ADGroupID,
		ADConfigID:    req.ADConfigID,
		MappingType:   req.MappingType,
		MappingStatus: models.MappingStatusActive,
		GroupDN:       group.GroupDN,
		GroupName:     group.GroupName,
		SyncEnabled:   req.SyncEnabled,
		CreatedBy:     req.CreatedBy,
	}

	if mapping.MappingType == "" {
		mapping.MappingType = models.MappingTypeManual
	}

	if err := s.db.WithContext(ctx).Create(mapping).Error; err != nil {
		return nil, fmt.Errorf("创建映射失败: %w", err)
	}

	return mapping, nil
}

// UpdateMapping 更新部门组映射
func (s *deptGroupMappingService) UpdateMapping(ctx context.Context, id string, req *UpdateMappingRequest) error {
	updates := make(map[string]interface{})

	if req.MappingStatus != nil {
		updates["mapping_status"] = *req.MappingStatus
	}
	if req.SyncEnabled != nil {
		updates["sync_enabled"] = *req.SyncEnabled
	}
	if req.UpdatedBy != "" {
		updates["updated_by"] = req.UpdatedBy
	}

	if len(updates) == 0 {
		return fmt.Errorf("没有需要更新的字段")
	}

	result := s.db.WithContext(ctx).
		Model(&models.DeptGroupMapping{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新映射失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("映射不存在")
	}

	return nil
}

// DeleteMapping 删除部门组映射（软删除）
func (s *deptGroupMappingService) DeleteMapping(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.DeptGroupMapping{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除映射失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("映射不存在")
	}
	return nil
}

// GetMapping 获取单个映射
func (s *deptGroupMappingService) GetMapping(ctx context.Context, id string) (*models.DeptGroupMapping, error) {
	var mapping models.DeptGroupMapping
	err := s.db.WithContext(ctx).
		Preload("Dept").
		Preload("ADGroup").
		Where("id = ?", id).
		First(&mapping).Error
	if err != nil {
		return nil, fmt.Errorf("映射不存在")
	}
	return &mapping, nil
}

// ListMappings 查询映射列表
func (s *deptGroupMappingService) ListMappings(ctx context.Context, req *ListMappingsRequest) (*ListMappingsResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.DeptGroupMapping{}).Where("deleted_at IS NULL")

	// 过滤条件
	if req.ADConfigID != nil && *req.ADConfigID != "" {
		query = query.Where("ad_config_id = ?", *req.ADConfigID)
	}
	if req.DeptID != nil && *req.DeptID != "" {
		query = query.Where("dept_id = ?", *req.DeptID)
	}
	if req.MappingType != nil && *req.MappingType != "" {
		query = query.Where("mapping_type = ?", *req.MappingType)
	}
	if req.MappingStatus != nil && *req.MappingStatus != "" {
		query = query.Where("mapping_status = ?", *req.MappingStatus)
	}
	if req.GroupName != nil && *req.GroupName != "" {
		query = query.Where("group_name LIKE ?", "%"+*req.GroupName+"%")
	}

	// 总数
	var total int64
	query.Count(&total)

	// 分页查询
	var list []models.DeptGroupMapping
	offset := (req.Current - 1) * req.PageSize
	err := query.
		Preload("Dept").
		Preload("ADGroup").
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&list).Error

	if err != nil {
		return nil, fmt.Errorf("查询映射列表失败: %w", err)
	}

	return &ListMappingsResponse{
		Total: total,
		List:  list,
	}, nil
}

// AutoMapDepartment 自动为部门创建映射（查找cxhub-{dept}组）
func (s *deptGroupMappingService) AutoMapDepartment(ctx context.Context, deptID, configID string) (*models.DeptGroupMapping, error) {
	// 1. 获取部门信息
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", deptID).First(&dept).Error; err != nil {
		return nil, fmt.Errorf("部门不存在")
	}

	// 2. 构造预期的组名：cxhub-{dept}
	// 组名规则：去除"部"字后缀，小写，例如：科技创新部 -> cxhub-科技创新
	groupName := fmt.Sprintf("cxhub-%s", strings.TrimSuffix(dept.DeptName, "部"))

	// 3. 查找对应的AD组
	var group models.ADGroup
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND group_name = ?", configID, groupName).
		First(&group).Error
	if err != nil {
		return nil, fmt.Errorf("未找到匹配的AD组: %s (部门: %s)", groupName, dept.DeptName)
	}

	// 4. 检查是否已存在映射
	var existing models.DeptGroupMapping
	err = s.db.WithContext(ctx).
		Where("dept_id = ? AND deleted_at IS NULL", deptID).
		First(&existing).Error
	if err == nil {
		return &existing, nil // 已存在，直接返回
	}

	// 5. 创建自动映射
	mapping := &models.DeptGroupMapping{
		DeptID:        deptID,
		ADGroupID:     group.ID,
		ADConfigID:    configID,
		MappingType:   models.MappingTypeAuto,
		MappingStatus: models.MappingStatusActive,
		GroupDN:       group.GroupDN,
		GroupName:     group.GroupName,
		SyncEnabled:   true,
	}

	if err := s.db.WithContext(ctx).Create(mapping).Error; err != nil {
		return nil, fmt.Errorf("创建自动映射失败: %w", err)
	}

	return mapping, nil
}

// GetMappingByDept 根据部门ID获取映射
func (s *deptGroupMappingService) GetMappingByDept(ctx context.Context, deptID string) (*models.DeptGroupMapping, error) {
	var mapping models.DeptGroupMapping
	err := s.db.WithContext(ctx).
		Where("dept_id = ? AND deleted_at IS NULL", deptID).
		First(&mapping).Error
	if err != nil {
		return nil, fmt.Errorf("未找到部门映射")
	}
	return &mapping, nil
}
