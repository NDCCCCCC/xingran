package addomain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// OUGroupMappingService OU组映射服务
type OUGroupMappingService struct {
	db *gorm.DB
}

// NewOUGroupMappingService 创建OU组映射服务
func NewOUGroupMappingService(db *gorm.DB) *OUGroupMappingService {
	return &OUGroupMappingService{
		db: db,
	}
}

// ListMappingsRequest 查询映射列表请求
type ListMappingsRequest struct {
	ADConfigID string `json:"adConfigId,omitempty"`
	OUDn       string `json:"ouDn,omitempty"`
	GroupName  string `json:"groupName,omitempty"`
	Status     string `json:"status,omitempty"`
	Current    int    `json:"current"`
	PageSize   int    `json:"pageSize"`
}

// ListMappingsResponse 查询映射列表响应
type ListMappingsResponse struct {
	List     []*models.OUGroupMapping `json:"list"`
	Total    int64                    `json:"total"`
	Current  int                      `json:"current"`
	PageSize int                      `json:"pageSize"`
}

// CreateMappingRequest 创建映射请求
type CreateMappingRequest struct {
	ADConfigID  string                       `json:"adConfigId" binding:"required"`
	OUDn        string                       `json:"ouDn" binding:"required"`
	OUName      string                       `json:"ouName" binding:"required"`
	ADGroupID   string                       `json:"adGroupId" binding:"required"`
	SyncEnabled bool                         `json:"syncEnabled"`
	CreatedBy   string                       `json:"createdBy,omitempty"`
}

// UpdateMappingRequest 更新映射请求
type UpdateMappingRequest struct {
	SyncEnabled *bool                         `json:"syncEnabled"`
	Status      *models.OUGroupMappingStatus `json:"status"`
	UpdatedBy   string                        `json:"updatedBy,omitempty"`
}

// ListMappings 查询映射列表
func (s *OUGroupMappingService) ListMappings(ctx context.Context, req *ListMappingsRequest) (*ListMappingsResponse, error) {
	var list []*models.OUGroupMapping
	var total int64

	query := s.db.WithContext(ctx).Model(&models.OUGroupMapping{})

	// 过滤条件
	if req.ADConfigID != "" {
		query = query.Where("ad_config_id = ?", req.ADConfigID)
	}
	if req.OUDn != "" {
		query = query.Where("ou_dn = ?", req.OUDn)
	}
	if req.Status != "" {
		query = query.Where("mapping_status = ?", req.Status)
	}
	if req.GroupName != "" {
		// 关联查询组名
		query = query.Joins("JOIN sys_ad_group ON sys_ad_group.id = sys_ou_group_mapping.ad_group_id").
			Where("sys_ad_group.group_name LIKE ?", "%"+req.GroupName+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count mappings: %w", err)
	}

	// 分页查询
	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	offset := (req.Current - 1) * req.PageSize
	if err := query.Preload("ADGroup").Preload("ADConfig").
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("failed to list mappings: %w", err)
	}

	return &ListMappingsResponse{
		List:     list,
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}, nil
}

// CreateMapping 创建映射
func (s *OUGroupMappingService) CreateMapping(ctx context.Context, req *CreateMappingRequest) (*models.OUGroupMapping, error) {
	// 验证AD组是否存在
	var adGroup models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", req.ADGroupID).First(&adGroup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("AD group not found")
		}
		return nil, fmt.Errorf("failed to find AD group: %w", err)
	}

	mapping := &models.OUGroupMapping{
		ADConfigID:    req.ADConfigID,
		OUDN:          req.OUDn,
		OUName:        req.OUName,
		ADGroupID:     req.ADGroupID,
		MappingStatus: models.OUGroupMappingStatusActive,
		SyncEnabled:   req.SyncEnabled,
		CreatedBy:     req.CreatedBy,
		UpdatedBy:     req.CreatedBy, // 创建时 updated_by 与 created_by 相同
	}

	// 直接创建，依赖数据库的唯一约束处理重复
	if err := s.db.WithContext(ctx).Create(mapping).Error; err != nil {
		// 检查是否是唯一约束冲突（重复映射）
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("mapping already exists for this OU and group")
		}
		return nil, fmt.Errorf("failed to create mapping: %w", err)
	}

	// 重新加载关联数据
	if err := s.db.WithContext(ctx).Preload("ADGroup").Preload("ADConfig").
		First(mapping, "id = ?", mapping.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load mapping associations: %w", err)
	}

	return mapping, nil
}

// GetMapping 获取单个映射
func (s *OUGroupMappingService) GetMapping(ctx context.Context, id string) (*models.OUGroupMapping, error) {
	var mapping models.OUGroupMapping
	if err := s.db.WithContext(ctx).Preload("ADGroup").Preload("ADConfig").
		First(&mapping, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("mapping not found")
		}
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}

	return &mapping, nil
}

// UpdateMapping 更新映射
func (s *OUGroupMappingService) UpdateMapping(ctx context.Context, id string, req *UpdateMappingRequest) error {
	updates := make(map[string]interface{})

	if req.SyncEnabled != nil {
		updates["sync_enabled"] = *req.SyncEnabled
	}
	if req.Status != nil {
		updates["mapping_status"] = *req.Status
	}
	if req.UpdatedBy != "" {
		updates["updated_by"] = req.UpdatedBy
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	result := s.db.WithContext(ctx).Model(&models.OUGroupMapping{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update mapping: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("mapping not found")
	}

	return nil
}

// DeleteMapping 删除映射
func (s *OUGroupMappingService) DeleteMapping(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.OUGroupMapping{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete mapping: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("mapping not found")
	}

	return nil
}

// GetMappingsByOU 获取OU的所有关联组
func (s *OUGroupMappingService) GetMappingsByOU(ctx context.Context, ouDn string) ([]*models.OUGroupMapping, error) {
	var mappings []*models.OUGroupMapping

	if err := s.db.WithContext(ctx).
		Preload("ADGroup").
		Preload("ADConfig").
		Where("ou_dn = ? AND mapping_status = ?", ouDn, models.OUGroupMappingStatusActive).
		Order("created_at DESC").
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get mappings by OU: %w", err)
	}

	return mappings, nil
}

// CreateSyncLog 创建同步日志
func (s *OUGroupMappingService) CreateSyncLog(ctx context.Context, log *models.OUGroupMappingSyncLog) error {
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}
	return nil
}

// UpdateSyncStatus 更新同步状态
func (s *OUGroupMappingService) UpdateSyncStatus(ctx context.Context, mappingID string, lastSyncAt time.Time) error {
	result := s.db.WithContext(ctx).Model(&models.OUGroupMapping{}).
		Where("id = ?", mappingID).
		Update("last_sync_at", lastSyncAt)

	if result.Error != nil {
		return fmt.Errorf("failed to update sync status: %w", result.Error)
	}

	return nil
}

// isUniqueConstraintError 检查是否是唯一约束冲突错误
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL 唯一约束错误代码
	// "unique constraint" 错误消息
	return err.Error() != "" && (containsIgnoreCase(err.Error(), "duplicate key") ||
		containsIgnoreCase(err.Error(), "unique constraint"))
}

// containsIgnoreCase 不区分大小写检查字符串包含
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}