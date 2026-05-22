package addomain

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeptOUmapper 部门-OU映射服务
// 提供系统部门与AD域控OU的双向映射查询和管理功能
type DeptOUmapper struct {
	db *gorm.DB
}

// NewDeptOUmapper 创建DeptOUmapper实例
func NewDeptOUmapper(db *gorm.DB) *DeptOUmapper {
	return &DeptOUmapper{db: db}
}

// FindDeptByOUDN 通过OU DN查找部门ID
// 用于用户AD登录时，根据用户所在OU反向查找系统部门
func (m *DeptOUmapper) FindDeptByOUDN(ctx context.Context, ouDN string) (string, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("ou_dn = ? AND sync_enabled = ?", ouDN, true).
		First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到OU DN %s 对应的部门", ouDN)
		}
		return "", fmt.Errorf("查询OU DN映射失败: %w", err)
	}
	return mapping.DeptID, nil
}

// FindOUDNByDeptID 通过部门ID查找OU DN
// 用于部门同步到AD时，查找已存在的映射关系
func (m *DeptOUmapper) FindOUDNByDeptID(ctx context.Context, deptID string) (string, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("dept_id = ? AND sync_enabled = ?", deptID, true).
		First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到部门 %s 对应的OU DN", deptID)
		}
		return "", fmt.Errorf("查询部门映射失败: %w", err)
	}
	return mapping.OUDN, nil
}

// UpsertMapping 创建或更新映射关系（幂等操作）
// 使用ON CONFLICT确保同一个部门+AD配置只有一个映射
func (m *DeptOUmapper) UpsertMapping(ctx context.Context, mapping *models.DeptOUMapping) error {
	return m.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dept_id"}, {Name: "ad_config_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"ou_dn", "ou_name", "parent_ou_dn", "sync_status", "last_sync_at", "updated_at"}),
		}).
		Create(mapping).Error
}

// GetMappingByDept 获取部门的映射关系
func (m *DeptOUmapper) GetMappingByDept(ctx context.Context, deptID string) (*models.DeptOUMapping, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("dept_id = ?", deptID).
		First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetMappingByOU 获取OU DN的映射关系
func (m *DeptOUmapper) GetMappingByOU(ctx context.Context, ouDN string) (*models.DeptOUMapping, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("ou_dn = ?", ouDN).
		First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// ListMappings 列出指定AD配置的所有映射
func (m *DeptOUmapper) ListMappings(ctx context.Context, adConfigID string) ([]*models.DeptOUMapping, error) {
	var mappings []*models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("ad_config_id = ?", adConfigID).
		Order("created_at DESC").
		Find(&mappings).Error
	return mappings, err
}

// DeleteMapping 删除映射关系（软删除）
func (m *DeptOUmapper) DeleteMapping(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).
		Delete(&models.DeptOUMapping{}, "id = ?", id).Error
}

// DisableMapping 禁用映射关系（设置sync_enabled=false）
func (m *DeptOUmapper) DisableMapping(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).
		Model(&models.DeptOUMapping{}).
		Where("id = ?", id).
		Update("sync_enabled", false).Error
}

// UpdateSyncStatus 更新映射的同步状态
func (m *DeptOUmapper) UpdateSyncStatus(ctx context.Context, id string, status string) error {
	return m.db.WithContext(ctx).
		Model(&models.DeptOUMapping{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sync_status": status,
			"updated_at":  "CURRENT_TIMESTAMP",
		}).Error
}
