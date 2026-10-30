package addomain

import (
	"context"
	"errors"
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
// 上层 resolveDeptFromOU（internal/services/system/user_sync_service.go）查不到映射时返回空串，走默认部门兜底。
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

// FindOUDNByDeptIDForConfig 配置作用域的 OU DN 查询（Phase 58 F-B，阻断级）。
//
// 为什么必须用这个而不是简单的 (dept_id, sync_enabled) 查询：
//   - sys_dept_ou_mapping 的唯一键是 (dept_id, ad_config_id) → 一个 dept 在 N 个配置下
//     有 N 行合法记录
//   - 多 AD 配置环境下不带 ad_config_id 的 First() 顺序不确定 → 排水器会把 A 域用户移进 B 域的 OU
//     （跨域写入，是 phase 58 最严重的误路由形态）
//
// ⚠ 不带 ad_config_id 的旧方法 FindOUDNByDeptID 已随 moveUserToNewOU 一起删除：
//   它唯一的生产调用方 moveUserToNewOU 属于已下线的 SyncUserUpdateToAD 路径，
//   保留会留下一条「无配置作用域」的跨域误路由隐患。
func (m *DeptOUmapper) FindOUDNByDeptIDForConfig(ctx context.Context, deptID, adConfigID string) (string, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("dept_id = ? AND ad_config_id = ? AND sync_enabled = ?", deptID, adConfigID, true).
		First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到部门 %s 在 AD 配置 %s 下的 OU DN", deptID, adConfigID)
		}
		return "", fmt.Errorf("查询部门映射失败: %w", err)
	}
	return mapping.OUDN, nil
}

// ErrOUDNConflict 哨兵错误：OU DN 已映射到其他部门（HYGIENE-05）。
// 用 errors.Is(err, ErrOUDNConflict) 或 errors.As(err, &*OUDNConflictError) 识别。
var ErrOUDNConflict = errors.New("OU DN 已映射到其他部门")

// OUDNConflictError 携带冲突双方详情的 typed error。
// UpsertMapping 严格化后：同名兄弟部门争用同一 OU DN 时不再静默互相覆盖，
// 冲突显式返回，映射保留给 incumbent（先占方）。
type OUDNConflictError struct {
	OUDN             string
	ADConfigID       string
	ExistingDeptID   string
	ExistingDeptName string
	IncomingDeptID   string
	IncomingDeptName string
}

func (e *OUDNConflictError) Error() string {
	return fmt.Sprintf("OU DN %s 已映射到部门 [%s](%s)，不能重新映射到部门 [%s](%s)",
		e.OUDN, e.ExistingDeptName, e.ExistingDeptID, e.IncomingDeptName, e.IncomingDeptID)
}

// Unwrap 使 errors.Is(err, ErrOUDNConflict) 成立。
func (e *OUDNConflictError) Unwrap() error { return ErrOUDNConflict }

// UpsertMapping 创建或更新映射关系（幂等操作，冲突严格化 HYGIENE-05）。
// 处理数据库的两个唯一约束：
// 1. uni_dept_ou_mapping_dept (dept_id, ad_config_id) - 一个部门在一个AD配置下只能有一个映射
// 2. uni_dept_ou_mapping_ou (ad_config_id, ou_dn) - 一个OU在一个AD配置下只能映射到一个部门
//
// 冲突语义：若 (ad_config_id, ou_dn) 已映射到其他部门，返回 *OUDNConflictError
// 且不做任何删除（映射保留给先占部门）。显式覆盖走 UpsertMappingForce（仅手工重绑 API）。
func (m *DeptOUmapper) UpsertMapping(ctx context.Context, mapping *models.DeptOUMapping) error {
	return m.upsert(ctx, mapping, false)
}

// UpsertMappingForce 强制建立映射：发现异部门冲突时告警 + 删旧 + 建新。
// 仅供手工重绑 API（ou_mapping_handler）使用；自动同步路径禁止调用。
// P1 fix 保留：delete+create 包在单事务中，delete 成功后 crash 不会丢映射。
func (m *DeptOUmapper) UpsertMappingForce(ctx context.Context, mapping *models.DeptOUMapping) error {
	return m.upsert(ctx, mapping, true)
}

func (m *DeptOUmapper) upsert(ctx context.Context, mapping *models.DeptOUMapping, allowRebind bool) error {
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
			if !allowRebind {
				// HYGIENE-05：不删除，返回携带双方部门名的冲突错误
				var conflict OUDNConflictError
				conflict.OUDN = mapping.OUDN
				conflict.ADConfigID = mapping.ADConfigID
				conflict.ExistingDeptID = existing.DeptID
				conflict.IncomingDeptID = mapping.DeptID
				var dept models.Department
				if err := tx.Select("dept_name").Where("id = ?", existing.DeptID).First(&dept).Error; err == nil {
					conflict.ExistingDeptName = dept.DeptName
				}
				if err := tx.Select("dept_name").Where("id = ?", mapping.DeptID).First(&dept).Error; err == nil {
					conflict.IncomingDeptName = dept.DeptName
				}
				return &conflict
			}
			// 显式 Force：ou_dn 已映射到不同部门，事务内删除旧映射(失败回滚)
			applogger.Warnf("OU DN %s 已映射到部门 %s，现在重新映射到部门 %s（显式 Force 覆盖）",
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
