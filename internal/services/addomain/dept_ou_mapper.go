package addomain

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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
//
// Phase 40 修复（ad-login-deleted-dept-reuse）：JOIN sys_dept 过滤已软删除部门。
// 原实现只校验映射表存在性，未验证映射部门是否已被软删除，导致登录时把用户
// 挂到 deleted_at IS NOT NULL 的乱码部门（参见 .planning/debug/ad-login-deleted-dept-reuse.md）。
// JOIN 后若映射部门已被删除，本函数返回 ErrRecordNotFound，
// 上层 user_ou_service.resolveDeptFromOU 会落到 createDeptFromOUDN 重建部门链。
func (m *DeptOUmapper) FindDeptByOUDN(ctx context.Context, ouDN string) (string, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Joins("JOIN sys_dept ON sys_dept.id = sys_dept_ou_mapping.dept_id").
		Where("sys_dept_ou_mapping.ou_dn = ? AND sys_dept_ou_mapping.sync_enabled = ?", ouDN, true).
		Where("sys_dept.deleted_at IS NULL").
		First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到OU DN %s 对应的有效部门", ouDN)
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
// 处理数据库的两个唯一约束：
// 1. uni_dept_ou_mapping_dept (dept_id, ad_config_id) - 一个部门在一个AD配置下只能有一个映射
// 2. uni_dept_ou_mapping_ou (ad_config_id, ou_dn) - 一个OU在一个AD配置下只能映射到一个部门
func (m *DeptOUmapper) UpsertMapping(ctx context.Context, mapping *models.DeptOUMapping) error {
	// P1 fix: 把"先 delete 旧映射 → 再 Create 新映射"包在单事务中。
	// 原实现中两步在独立事务,delete 成功后 crash/网络抖动会让旧映射消失
	// 而新映射未建,等同于丢失整条映射关系。
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先检查是否已存在相同 ou_dn 的映射
		var existing models.DeptOUMapping
		err := tx.
			Where("ad_config_id = ? AND ou_dn = ?", mapping.ADConfigID, mapping.OUDN).
			First(&existing).Error

		if err == nil {
			// 找到已存在的 ou_dn 映射
			if existing.DeptID == mapping.DeptID {
				// 同一个部门，只更新其他字段
				// GORM 会自动更新 updated_at 字段
				return tx.
					Model(&existing).
					Updates(map[string]interface{}{
						"ou_name":      mapping.OUName,
						"parent_ou_dn": mapping.ParentOUDN,
						"sync_status":  mapping.SyncStatus,
						"last_sync_at": mapping.LastSyncAt,
						"sync_enabled": mapping.SyncEnabled,
					}).Error
			}
			// ou_dn 已映射到不同部门，事务内删除旧映射(失败回滚)
			applogger.Warnf("OU DN %s 已映射到部门 %s，现在重新映射到部门 %s",
				mapping.OUDN, existing.DeptID, mapping.DeptID)
			if err := tx.Delete(&existing).Error; err != nil {
				return fmt.Errorf("删除旧映射失败: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询现有映射失败: %w", err)
		}

		// 事务内创建新映射(使用 ON CONFLICT 处理 dept_id 约束)
		return tx.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "dept_id"}, {Name: "ad_config_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"ou_dn", "ou_name", "parent_ou_dn", "sync_status", "last_sync_at"}),
			}).
			Create(mapping).Error
	})
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
	// GORM 会自动更新 updated_at 字段
	return m.db.WithContext(ctx).
		Model(&models.DeptOUMapping{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sync_status": status,
		}).Error
}
