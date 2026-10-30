package addomain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// DeptADCascadeService 部门删除/改名 → AD OU 级联（域控审查 P0-2）。
//
// 用户决策（m02757）：部门删除自动删除 OU（前提：部门下无用户且 OU 下用户也为空）；
// 部门改名 AD OU 自动重命名。
//
// 设计约束：
//   - 级联失败**不阻断**本地操作（本地事务已提交，AD 侧不一致可由「部门到AD同步」
//     下轮重建/管理员手动纠正），只 WARN；
//   - LDAP 一律走 FailoverClient 账号池（Pitfall 4：pool 必须是全局共享单例，
//     由 router 装配处注入 core.AuthFactory.GetAccountPool()）；
//   - 遍历 sys_dept_ou_mapping 的 (dept_id, ad_config_id) 全部映射（多域各一份）。
type DeptADCascadeService struct {
	db   *gorm.DB
	pool AccountPool
}

// NewDeptADCascadeService 创建级联服务。pool 为 nil 时所有方法直接 no-op（单元测试/未启用 AD 场景）。
func NewDeptADCascadeService(db *gorm.DB, pool AccountPool) *DeptADCascadeService {
	return &DeptADCascadeService{db: db, pool: pool}
}

// deptOUMapping 按部门取全部 OU 映射（联 AD 配置，只同步启用配置的映射）。
func (s *DeptADCascadeService) deptOUMappings(ctx context.Context, deptID string) ([]models.DeptOUMapping, []models.ADConfig, error) {
	var mappings []models.DeptOUMapping
	if err := s.db.WithContext(ctx).Where("dept_id = ?", deptID).Find(&mappings).Error; err != nil {
		return nil, nil, fmt.Errorf("查询部门OU映射失败: %w", err)
	}
	if len(mappings) == 0 {
		return nil, nil, nil
	}
	configIDs := make([]string, 0, len(mappings))
	for _, m := range mappings {
		configIDs = append(configIDs, m.ADConfigID)
	}
	var configs []models.ADConfig
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", configIDs, models.ADConfigStatusEnabled).
		Find(&configs).Error; err != nil {
		return nil, nil, fmt.Errorf("查询AD配置失败: %w", err)
	}
	return mappings, configs, nil
}

// configByID 映射 adConfigID → 配置。
func configByID(configs []models.ADConfig, id string) *models.ADConfig {
	for i := range configs {
		if configs[i].ID == id {
			return &configs[i]
		}
	}
	return nil
}

// RenameDeptOU 部门改名后重命名 AD OU（ModifyDN 子树随动），并把 DB 中受影响的
// DN 字符串（本部门及子孙的 sys_dept_ou_mapping.ou_dn / parent_ou_dn、
// sys_ou_group_mapping.ou_dn）同步替换。
func (s *DeptADCascadeService) RenameDeptOU(ctx context.Context, deptID, newDeptName string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	mappings, configs, err := s.deptOUMappings(ctx, deptID)
	if err != nil {
		return err
	}
	for _, m := range mappings {
		cfg := configByID(configs, m.ADConfigID)
		if cfg == nil {
			continue // 配置停用：不同步，保留映射
		}
		newOUDN := BuildOUDN(newDeptName, derefParent(m.ParentOUDN))
		if NormalizeDN(newOUDN) == NormalizeDN(m.OUDN) {
			continue // 名字未变（如只改了其他字段）
		}
		fc := NewFailoverClient(s.pool, cfg)
		err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
			return client.RenameOU(m.OUDN, newDeptName)
		})
		if err != nil {
			// 源 OU 不存在（OU 被手工删过/从未同步成功）→ 删除过期映射，让下轮部门同步重建
			if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
				applogger.Warnf("[dept_cascade] 部门 %s 的 OU %s 在AD中不存在，删除过期映射待重建", deptID, m.OUDN)
				s.db.WithContext(ctx).Delete(&models.DeptOUMapping{}, "id = ?", m.ID)
				continue
			}
			applogger.Errorf("[dept_cascade] 重命名部门 %s 的 OU %s → %s 失败: %v（本地已改名，可手动执行部门同步纠正）",
				deptID, m.OUDN, newOUDN, err)
			continue
		}
		s.replaceOUDNStrings(ctx, m.ADConfigID, m.OUDN, newOUDN, newDeptName)
		applogger.Infof("[dept_cascade] 部门 %s 的 OU 已重命名: %s → %s", deptID, m.OUDN, newOUDN)
	}
	return nil
}

// DeleteDeptOU 部门删除后删除对应 AD OU。
// 双重空检：① 本地 sys_ad_user 中该 OU 下无影子用户；② LDAP DeleteOU 本身对
// 非空 OU 返回 Not Allowed On Non-Leaf（66），也按不可删降级 WARN。
// 删除成功后清理 sys_dept_ou_mapping 与 sys_ou_group_mapping 引用行。
func (s *DeptADCascadeService) DeleteDeptOU(ctx context.Context, deptID string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	mappings, configs, err := s.deptOUMappings(ctx, deptID)
	if err != nil {
		return err
	}
	for _, m := range mappings {
		cfg := configByID(configs, m.ADConfigID)
		if cfg == nil {
			continue
		}
		// 空检①：OU 下仍有 AD 用户则不删（与用户决策一致）
		var userCount int64
		if err := s.db.WithContext(ctx).Model(&models.ADUser{}).
			Where("ad_config_id = ? AND ou_dn LIKE ?", m.ADConfigID, "%"+escapeLike(m.OUDN)).
			Count(&userCount).Error; err != nil {
			applogger.Warnf("[dept_cascade] 预检 OU %s 用户数失败，跳过删除: %v", m.OUDN, err)
			continue
		}
		if userCount > 0 {
			applogger.Warnf("[dept_cascade] OU %s 下仍有 %d 个AD用户，保留OU与映射（请先处理用户）", m.OUDN, userCount)
			continue
		}
		fc := NewFailoverClient(s.pool, cfg)
		err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
			exists, err := client.DNExists(m.OUDN)
			if err != nil || !exists {
				return errNoDeleteNeeded
			}
			return client.DeleteOU(m.OUDN)
		})
		switch {
		case err == nil:
			s.cleanupDeptOURows(ctx, m)
			applogger.Infof("[dept_cascade] 部门 %s 的 OU %s 已删除", deptID, m.OUDN)
		case errors.Is(err, errNoDeleteNeeded):
			// AD 中本就不存在：只清本地映射
			s.cleanupDeptOURows(ctx, m)
		case ldap.IsErrorWithCode(err, ldap.LDAPResultNotAllowedOnNonLeaf):
			applogger.Warnf("[dept_cascade] OU %s 在AD中非空（含子对象），保留OU与映射（请先处理子OU/组）", m.OUDN)
		default:
			applogger.Errorf("[dept_cascade] 删除部门 %s 的 OU %s 失败: %v（映射保留，可手动执行部门同步纠正）", deptID, m.OUDN, err)
		}
	}
	return nil
}

// errNoDeleteNeeded 内部哨兵：AD 端无此 OU，无需删除（区别于真实失败）。
var errNoDeleteNeeded = errors.New("ad_ou_not_exists")

// replaceOUDNStrings 改名成功后替换 DB 中受影响的 DN 字符串。
// ModifyDN 使 AD 侧整棵子树 DN 随动，但 DB 中子孙映射行的 ou_dn / parent_ou_dn
// 仍是旧字符串 —— 组成员同步用 ou_dn 后缀匹配，不替换会把子孙判到旧 OU。
func (s *DeptADCascadeService) replaceOUDNStrings(ctx context.Context, adConfigID, oldOUDN, newOUDN, newDeptName string) {
	// sys_dept_ou_mapping：本部门 + 子孙（ou_dn 等于旧值，或以 ",旧值" 结尾）
	if err := s.db.WithContext(ctx).Exec(
		`UPDATE sys_dept_ou_mapping SET ou_dn = REPLACE(ou_dn, ?, ?),
			ou_name = CASE WHEN ou_dn = ? THEN ? ELSE ou_name END,
			parent_ou_dn = COALESCE(REPLACE(parent_ou_dn, ?, ?), parent_ou_dn),
			updated_at = ?
		 WHERE ad_config_id = ? AND (ou_dn = ? OR ou_dn LIKE ?)`,
		oldOUDN, newOUDN, oldOUDN, newDeptName, oldOUDN, newOUDN, time.Now(),
		adConfigID, oldOUDN, "%,"+escapeLike(oldOUDN),
	).Error; err != nil {
		applogger.Errorf("[dept_cascade] 替换 sys_dept_ou_mapping 中的旧 DN 失败: %v", err)
	}
	// sys_ou_group_mapping：OU 维度同步
	if err := s.db.WithContext(ctx).Exec(
		`UPDATE sys_ou_group_mapping SET ou_dn = REPLACE(ou_dn, ?, ?),
			ou_name = CASE WHEN ou_dn = ? THEN ? ELSE ou_name END,
			updated_at = ?
		 WHERE ad_config_id = ? AND (ou_dn = ? OR ou_dn LIKE ?)`,
		oldOUDN, newOUDN, oldOUDN, newDeptName, time.Now(),
		adConfigID, oldOUDN, "%,"+escapeLike(oldOUDN),
	).Error; err != nil {
		applogger.Errorf("[dept_cascade] 替换 sys_ou_group_mapping 中的旧 DN 失败: %v", err)
	}
}

// cleanupDeptOURows 删除 OU 成功后清理本地引用行。
func (s *DeptADCascadeService) cleanupDeptOURows(ctx context.Context, m models.DeptOUMapping) {
	if err := s.db.WithContext(ctx).Delete(&models.DeptOUMapping{}, "id = ?", m.ID).Error; err != nil {
		applogger.Warnf("[dept_cascade] 删除部门OU映射失败(mapping=%s): %v", m.ID, err)
	}
	if err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND ou_dn = ?", m.ADConfigID, m.OUDN).
		Delete(&models.OUGroupMapping{}).Error; err != nil {
		applogger.Warnf("[dept_cascade] 删除OU组映射失败(ou_dn=%s): %v", m.OUDN, err)
	}
}

// derefParent 取 parent_ou_dn 字符串值。
func derefParent(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// PurgeMappingMembers 删除 OU-组映射前，把 AD 组中属于该 OU 的成员移除
// （域控审查 P1-6：DeleteMapping 原本只删映射行，AD 组里已加入的成员成为
// 无映射管理的孤儿 —— 调岗/离职后无人再从组里移除）。
//
// 保守约束与组成员同步任务一致：只移除 DN 归一化后属于本 OU 的成员，
// 手工加入的外部用户不动。映射或组不存在时静默返回（幂等，供删除前置调用）。
func (s *DeptADCascadeService) PurgeMappingMembers(ctx context.Context, mappingID string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	var mapping models.OUGroupMapping
	if err := s.db.WithContext(ctx).Where("id = ?", mappingID).First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("查询OU组映射失败: %w", err)
	}
	var adGroup models.ADGroup
	if err := s.db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("查询AD组失败: %w", err)
	}
	config := &models.ADConfig{}
	if err := s.db.WithContext(ctx).
		Where("id = ? AND status = ?", mapping.ADConfigID, models.ADConfigStatusEnabled).
		First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 配置停用/不存在：不清理，保留 AD 现状
		}
		return fmt.Errorf("查询AD配置失败: %w", err)
	}
	normalizedOUDN := NormalizeDN(mapping.OUDN)
	fc := NewFailoverClient(s.pool, config)
	return fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		members, err := client.GetGroupMembers(adGroup.GroupDN)
		if err != nil {
			// 组可能已不存在：不阻断映射删除
			applogger.Warnf("[dept_cascade] 清理映射成员时读取组 %s 失败: %v", adGroup.GroupDN, err)
			return nil
		}
		removed := 0
		for _, dn := range members {
			if !strings.HasSuffix(NormalizeDN(dn), normalizedOUDN) {
				continue
			}
			if err := client.RemoveGroupMember(adGroup.GroupDN, dn); err != nil {
				if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
					continue
				}
				applogger.Warnf("[dept_cascade] 从组 %s 移除成员 %s 失败: %v", adGroup.GroupDN, dn, err)
				continue
			}
			removed++
		}
		if removed > 0 {
			applogger.Infof("[dept_cascade] 已从组 %s 移除 OU %s 下的 %d 个成员（映射删除前置清理）", adGroup.GroupDN, mapping.OUDN, removed)
		}
		return nil
	})
}
