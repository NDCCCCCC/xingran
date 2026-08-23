package addomain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GroupSyncService handles incremental group synchronization from AD
type GroupSyncService struct {
	db          *gorm.DB
	pool        AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
	syncService *SyncService
}

// NewGroupSyncService creates a new GroupSyncService
// Phase 38 Wave 1: 注入 AccountPool 字段，并透传给内部 SyncService。
func NewGroupSyncService(db *gorm.DB, pool AccountPool) *GroupSyncService {
	return &GroupSyncService{
		db:          db,
		pool:        pool,
		syncService: NewSyncService(db, pool),
	}
}

// GroupSyncResult represents the result of a group sync operation
type GroupSyncResult struct {
	TotalGroups    int `json:"totalGroups"`
	CreatedGroups  int `json:"createdGroups"`
	UpdatedGroups  int `json:"updatedGroups"`
	DeletedGroups  int `json:"deletedGroups"`
	TotalMembers   int `json:"totalMembers"`
	CreatedMembers int `json:"createdMembers"`
	RemovedMembers int `json:"removedMembers"`
	Duration       int `json:"duration"` // milliseconds
}

// SyncGroupsByConfig syncs all groups for a specific AD configuration
// This is an incremental sync that only processes changed groups
func (s *GroupSyncService) SyncGroupsByConfig(ctx context.Context, configID string) (*GroupSyncResult, error) {
	start := time.Now()
	result := &GroupSyncResult{}

	// 1. Get AD config
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", configID, models.ADConfigStatusEnabled).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在或未启用")
	}

	// 2. Connect to LDAP（Phase 38 Wave 1: 改走 FailoverClient 账号池故障切换）
	fc := NewFailoverClient(s.pool, &config)
	var entries []*ldap.Entry
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		var err error
		entries, err = client.SearchGroups(config.BaseDN)
		if err != nil {
			return fmt.Errorf("搜索用户组失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}

	// 4. Sync groups
	if err := s.syncGroupEntries(ctx, &config, entries, result); err != nil {
		return nil, err
	}

	// 5. Detect and handle deleted groups (groups in DB but not in LDAP)
	if err := s.handleDeletedGroups(ctx, &config, entries, result); err != nil {
		applogger.Warnf("[组同步] 处理已删除组失败: %v", err)
		// Non-fatal: continue
	}

	result.Duration = int(time.Since(start).Milliseconds())
	applogger.Infof("[组同步] 完成: 总数=%d, 创建=%d, 更新=%d, 删除=%d, 耗时=%dms",
		result.TotalGroups, result.CreatedGroups, result.UpdatedGroups, result.DeletedGroups, result.Duration)

	return result, nil
}

// SyncSingleGroup syncs a single group by DN from LDAP
func (s *GroupSyncService) SyncSingleGroup(ctx context.Context, configID, groupDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", configID, models.ADConfigStatusEnabled).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在或未启用")
	}


	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	// 仅 LDAP Search 在闭包内；DB upsert/members 同步放闭包外（不依赖 LDAP 连接）
	fc := NewFailoverClient(s.pool, &config)
	var entry *ldap.Entry
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		// Search for the specific group by DN
		searchRequest := ldap.NewSearchRequest(
			groupDN,
			ldap.ScopeBaseObject,
			ldap.NeverDerefAliases,
			0, 0, false,
			"(objectClass=group)",
			[]string{"dn", "cn", "description", "member", "groupType"},
			nil,
		)
		sr, err := client.SearchWithRequest(searchRequest)
		if err != nil {
			return fmt.Errorf("搜索用户组失败: %w", err)
		}
		if len(sr.Entries) == 0 {
			return fmt.Errorf("用户组不存在: %s", groupDN)
		}
		entry = sr.Entries[0]
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}

	now := time.Now()
	members := entry.GetAttributeValues("member")
	groupScope, groupType := parseGroupTypeFromLDAP(entry.GetAttributeValue("groupType"))
	ouDN := extractParentDN(groupDN)

	group := &models.ADGroup{
		ADConfigID:  config.ID,
		GroupDN:     groupDN,
		GroupName:   entry.GetAttributeValue("cn"),
		Description: entry.GetAttributeValue("description"),
		MemberCount: len(members),
		OUN:         ouDN,
		GroupScope:  groupScope,
		GroupType:   groupType,
		LastSyncAt:  &now,
	}

	// Upsert the group
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "group_dn"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"group_name", "description", "member_count", "ou_dn",
			"group_scope", "group_type", "last_sync_at",
		}),
	}).Create(group).Error; err != nil {
		return fmt.Errorf("同步用户组失败: %w", err)
	}

	// Sync members
	if err := s.syncService.syncGroupMembers(ctx, &config, groupDN, members); err != nil {
		return fmt.Errorf("同步组成员失败: %w", err)
	}

	applogger.Infof("[组同步] 单组同步完成: %s, 成员数=%d", groupDN, len(members))
	return nil
}

// GetGroupSyncStatus returns the sync status for groups under a config
func (s *GroupSyncService) GetGroupSyncStatus(ctx context.Context, configID string) (*GroupSyncStatusInfo, error) {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}

	status := &GroupSyncStatusInfo{
		ConfigID:    configID,
		ConfigName:  config.ConfigName,
		LastSyncAt:  config.LastSyncAt,
		SyncEnabled: config.SyncEnabled,
	}

	// Count groups
	var totalGroups int64
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", configID).
		Count(&totalGroups)
	status.TotalGroups = totalGroups

	// Count groups synced in last 24 hours
	var recentlySynced int64
	cutoff := time.Now().Add(-24 * time.Hour)
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL AND last_sync_at > ?", configID, cutoff).
		Count(&recentlySynced)
	status.RecentlySynced = recentlySynced

	// Count total member relationships
	var totalMembers int64
	s.db.WithContext(ctx).Model(&models.ADGroupMember{}).
		Where("ad_config_id = ?", configID).
		Count(&totalMembers)
	status.TotalMemberRelations = totalMembers

	// Count groups that have never been synced
	var neverSynced int64
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL AND last_sync_at IS NULL", configID).
		Count(&neverSynced)
	status.NeverSynced = neverSynced

	return status, nil
}

// GroupSyncStatusInfo represents sync status for groups
type GroupSyncStatusInfo struct {
	ConfigID           string     `json:"configId"`
	ConfigName         string     `json:"configName"`
	LastSyncAt         *time.Time `json:"lastSyncAt,omitempty"`
	SyncEnabled        bool       `json:"syncEnabled"`
	TotalGroups        int64      `json:"totalGroups"`
	RecentlySynced     int64      `json:"recentlySynced"`
	TotalMemberRelations int64    `json:"totalMemberRelations"`
	NeverSynced        int64      `json:"neverSynced"`
}

// syncGroupEntries syncs LDAP group entries to the database
func (s *GroupSyncService) syncGroupEntries(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry, result *GroupSyncResult) error {
	if len(entries) == 0 {
		return nil
	}

	result.TotalGroups = len(entries)

	// Get existing groups from DB
	groupDNs := extractDNs(entries)
	var existingGroups []models.ADGroup
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn IN ?", config.ID, groupDNs).Find(&existingGroups)

	existingGroupMap := make(map[string]*models.ADGroup, len(existingGroups))
	for i := range existingGroups {
		existingGroupMap[existingGroups[i].GroupDN] = &existingGroups[i]
	}

	var groupsToCreate []models.ADGroup
	groupsToUpdate := make(map[string]*models.ADGroup)
	groupMembersMap := make(map[string][]string)
	now := time.Now()

	for _, entry := range entries {
		groupDN := entry.DN
		ouDN := extractParentDN(groupDN)
		members := entry.GetAttributeValues("member")
		groupMembersMap[groupDN] = members
		groupScope, groupType := parseGroupTypeFromLDAP(entry.GetAttributeValue("groupType"))

		if existingGroup, exists := existingGroupMap[groupDN]; exists {
			// Check if data actually changed before counting as update
			groupName := entry.GetAttributeValue("cn")
			description := entry.GetAttributeValue("description")
			memberCount := len(members)

			if existingGroup.GroupName != groupName ||
				existingGroup.Description != description ||
				existingGroup.MemberCount != memberCount ||
				existingGroup.OUN != ouDN ||
				existingGroup.GroupScope != groupScope ||
				existingGroup.GroupType != groupType {
				result.UpdatedGroups++
			}

			existingGroup.GroupName = groupName
			existingGroup.Description = description
			existingGroup.MemberCount = memberCount
			existingGroup.OUN = ouDN
			existingGroup.GroupScope = groupScope
			existingGroup.GroupType = groupType
			existingGroup.LastSyncAt = &now
			groupsToUpdate[groupDN] = existingGroup
		} else {
			groupsToCreate = append(groupsToCreate, models.ADGroup{
				ADConfigID:  config.ID,
				GroupDN:     groupDN,
				GroupName:   entry.GetAttributeValue("cn"),
				Description: entry.GetAttributeValue("description"),
				MemberCount: len(members),
				OUN:         ouDN,
				GroupScope:  groupScope,
				GroupType:   groupType,
				LastSyncAt:  &now,
			})
			result.CreatedGroups++
		}
	}

	// Batch create
	if len(groupsToCreate) > 0 {
		batchSize := 500
		for i := 0; i < len(groupsToCreate); i += batchSize {
			end := i + batchSize
			if end > len(groupsToCreate) {
				end = len(groupsToCreate)
			}
			if err := s.db.WithContext(ctx).Create(groupsToCreate[i:end]).Error; err != nil {
				return fmt.Errorf("批量创建用户组失败: %w", err)
			}
		}
	}

	// Batch update
	if len(groupsToUpdate) > 0 {
		groupSlice := make([]*models.ADGroup, 0, len(groupsToUpdate))
		for _, group := range groupsToUpdate {
			groupSlice = append(groupSlice, group)
		}

		batchSize := 500
		for i := 0; i < len(groupSlice); i += batchSize {
			end := i + batchSize
			if end > len(groupSlice) {
				end = len(groupSlice)
			}
			batch := groupSlice[i:end]
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "group_dn"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"group_name", "description", "member_count", "ou_dn", "group_scope", "group_type", "last_sync_at",
				}),
			}).Create(batch).Error; err != nil {
				return fmt.Errorf("批量更新用户组失败: %w", err)
			}
		}
	}

	// Sync group members
	for groupDN, members := range groupMembersMap {
		if err := s.syncService.syncGroupMembers(ctx, config, groupDN, members); err != nil {
			applogger.Warnf("[组同步] 同步组成员失败 [%s]: %v", groupDN, err)
			// Continue with other groups
		}
		result.TotalMembers += len(members)
	}

	return nil
}

// handleDeletedGroups detects groups that exist in DB but not in LDAP and soft-deletes them
func (s *GroupSyncService) handleDeletedGroups(ctx context.Context, config *models.ADConfig, ldapEntries []*ldap.Entry, result *GroupSyncResult) error {
	// P1 fix: 安全闸门 — 如果 LDAP 返回为空,直接拒绝任何删除。
	// 否则 AD 不可达/网络抖动/搜索 BaseDN 配错都会导致整个 DB 组全表软删除,
	// 此时用户彻底失去所有 AD 组关联,恢复极其困难。
	if len(ldapEntries) == 0 {
		applogger.Warnf("[组同步] 跳过过期组清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
		return nil
	}

	// Build a set of current LDAP group DNs
	ldapGroupDNs := make(map[string]bool, len(ldapEntries))
	for _, entry := range ldapEntries {
		ldapGroupDNs[entry.DN] = true
	}

	// Find groups in DB that are not in LDAP
	var staleGroups []models.ADGroup
	s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
		Find(&staleGroups)

	var deletedDNs []string
	for _, group := range staleGroups {
		if !ldapGroupDNs[group.GroupDN] {
			deletedDNs = append(deletedDNs, group.GroupDN)
		}
	}

	if len(deletedDNs) > 0 {
		// P1 fix: 阈值保护 — 若拟删除超过现有 DB 组的 50%,拒绝并要求人工确认。
		// 防止 LDAP 返回严重不完整(部分 OU 搜索失败/分页中断)时误删大量数据。
		if len(staleGroups) > 0 && len(deletedDNs)*2 > len(staleGroups) {
			applogger.Warnf(
				"[组同步] 跳过过期组清理: 拟删除 %d/%d 个组(>50%%),"+
					"可能是 LDAP 返回不完整。请人工核实后手动清理",
				len(deletedDNs), len(staleGroups))
			return nil
		}

		// Soft-delete stale groups
		if err := s.db.WithContext(ctx).
			Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
			Delete(&models.ADGroup{}).Error; err != nil {
			return fmt.Errorf("删除过期用户组失败: %w", err)
		}

		// Also clean up member relations (hard delete since they are derived data)
		if err := s.db.WithContext(ctx).Unscoped().
			Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
			Delete(&models.ADGroupMember{}).Error; err != nil {
			applogger.Warnf("[组同步] 清理过期组成员关系失败: %v", err)
		}

		result.DeletedGroups = len(deletedDNs)
		applogger.Infof("[组同步] 清理了 %d 个已不存在的用户组", len(deletedDNs))
	}

	return nil
}
