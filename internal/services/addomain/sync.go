package addomain

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// SyncService AD数据同步服务
type SyncService struct {
	db *gorm.DB
}

// NewSyncService 创建同步服务
func NewSyncService(db *gorm.DB) *SyncService {
	return &SyncService{db: db}
}

// SyncResult 同步结果
type SyncResult struct {
	OUCount      int `json:"ouCount"`
	GroupCount   int `json:"groupCount"`
	UserCount    int `json:"userCount"`
	ComputerCount int `json:"computerCount"`
}

// SyncDataByID 根据配置ID同步AD数据
func (s *SyncService) SyncDataByID(ctx context.Context, configID string, syncType string) (*SyncResult, error) {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", configID, models.ADConfigStatusEnabled).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在或未启用")
	}
	return s.SyncData(ctx, &config, syncType)
}

// SyncData 同步AD数据
func (s *SyncService) SyncData(ctx context.Context, config *models.ADConfig, syncType string) (*SyncResult, error) {
	// 创建同步日志
	syncLog := &models.ADSyncLog{
		ADConfigID: config.ID,
		SyncType:   models.ADSyncType(syncType),
		SyncStatus: models.ADSyncStatusRunning,
		StartTime:  time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(syncLog).Error; err != nil {
		return nil, fmt.Errorf("创建同步日志失败: %w", err)
	}

	// 解密密码
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(config)
	if err := client.Connect(); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer client.Close()

	result := &SyncResult{}

	// 同步OU
	ous, err := client.SearchOUs(config.BaseDN)
	if err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
		return nil, fmt.Errorf("搜索OU失败: %w", err)
	}
	result.OUCount = len(ous)
	if err := s.syncOUs(ctx, config, ous); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, 0, 0, 0, err.Error())
		return nil, err
	}

	// 同步用户组
	groups, err := client.SearchGroups(config.BaseDN)
	if err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, 0, 0, 0, err.Error())
		return nil, fmt.Errorf("搜索用户组失败: %w", err)
	}
	result.GroupCount = len(groups)
	if err := s.syncGroups(ctx, config, groups); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, 0, 0, err.Error())
		return nil, err
	}

	// 同步用户
	users, err := client.SearchUsers(config.BaseDN)
	if err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, 0, 0, err.Error())
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}
	result.UserCount = len(users)
	if err := s.syncUsers(ctx, config, users); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, 0, err.Error())
		return nil, err
	}

	// 同步电脑设备
	computerService := NewComputerService(s.db)
	computers, err := client.SearchComputers(config.BaseDN)
	if err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, 0, err.Error())
		return nil, fmt.Errorf("搜索电脑设备失败: %w", err)
	}
	result.ComputerCount = len(computers)
	if err := computerService.syncComputers(ctx, config, computers); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount, err.Error())
		return nil, err
	}

	// 更新配置的最后同步时间
	now := time.Now()
	s.db.WithContext(ctx).Model(config).Updates(map[string]interface{}{
		"last_sync_at": now,
	})

	// 完成同步日志
	s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusSuccess, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount, "")

	return result, nil
}

// syncOUs 同步OU数据
func (s *SyncService) syncOUs(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	ouDNs := extractDNs(entries)
	existingOUs := s.getExistingOUs(ctx, config.ID, ouDNs)

	now := time.Now()
	ousToCreate, ousToUpdate := s.categorizeOUs(entries, existingOUs, config, now)

	if err := s.batchCreateOUs(ctx, ousToCreate); err != nil {
		return err
	}

	return s.batchUpdateOUs(ctx, ousToUpdate)
}

// extractDNs 提取所有DN
func extractDNs(entries []*ldap.Entry) []string {
	ouDNs := make([]string, 0, len(entries))
	for _, entry := range entries {
		ouDNs = append(ouDNs, entry.DN)
	}
	return ouDNs
}

// getExistingOUs 获取已存在的OU
func (s *SyncService) getExistingOUs(ctx context.Context, configID string, ouDNs []string) []models.ADOU {
	var existingOUs []models.ADOU
	s.db.WithContext(ctx).Where("ad_config_id = ? AND ou_dn IN ?", configID, ouDNs).Find(&existingOUs)
	return existingOUs
}

// categorizeOUs 分类OU为需要创建和更新的
func (s *SyncService) categorizeOUs(entries []*ldap.Entry, existingOUs []models.ADOU, config *models.ADConfig, now time.Time) ([]models.ADOU, map[string]*models.ADOU) {
	existingOUMap := make(map[string]*models.ADOU)
	for i := range existingOUs {
		existingOUMap[existingOUs[i].OUN] = &existingOUs[i]
	}

	var ousToCreate []models.ADOU
	ousToUpdate := make(map[string]*models.ADOU)

	for _, entry := range entries {
		ouDN := entry.DN
		ouName := entry.GetAttributeValue("ou")
		description := entry.GetAttributeValue("description")
		parentDN := extractParentDN(ouDN)
		ouPath := buildOUPath(ouDN, config.BaseDN)

		if existingOU, exists := existingOUMap[ouDN]; exists {
			existingOU.OUName = ouName
			existingOU.Description = description
			existingOU.ParentDN = parentDN
			existingOU.OUPath = ouPath
			existingOU.LastSyncAt = &now
			ousToUpdate[ouDN] = existingOU
		} else {
			ousToCreate = append(ousToCreate, models.ADOU{
				ADConfigID:  config.ID,
				OUN:         ouDN,
				OUName:      ouName,
				Description: description,
				ParentDN:    parentDN,
				OUPath:      ouPath,
				LastSyncAt:  &now,
			})
		}
	}

	return ousToCreate, ousToUpdate
}

// batchCreateOUs 批量创建OU
func (s *SyncService) batchCreateOUs(ctx context.Context, ous []models.ADOU) error {
	if len(ous) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Create(&ous).Error; err != nil {
		return fmt.Errorf("批量创建OU失败: %w", err)
	}
	return nil
}

// batchUpdateOUs 批量更新OU
func (s *SyncService) batchUpdateOUs(ctx context.Context, ous map[string]*models.ADOU) error {
	for _, ou := range ous {
		if err := s.db.WithContext(ctx).Save(ou).Error; err != nil {
			return fmt.Errorf("更新OU失败: %w", err)
		}
	}
	return nil
}

// syncGroups 同步用户组数据
func (s *SyncService) syncGroups(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	// 批量获取所有 Group DN
	groupDNs := make([]string, 0, len(entries))
	for _, entry := range entries {
		groupDNs = append(groupDNs, entry.DN)
	}

	// 批量查询已存在的 Group
	var existingGroups []models.ADGroup
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn IN ?", config.ID, groupDNs).Find(&existingGroups)

	// 创建 DN -> Group 的映射
	existingGroupMap := make(map[string]*models.ADGroup)
	for i := range existingGroups {
		existingGroupMap[existingGroups[i].GroupDN] = &existingGroups[i]
	}

	// 准备批量操作的数据
	var groupsToCreate []models.ADGroup
	groupsToUpdate := make(map[string]*models.ADGroup)
	groupMembersMap := make(map[string][]string) // groupDN -> members
	now := time.Now()

	for _, entry := range entries {
		groupDN := entry.DN
		groupName := entry.GetAttributeValue("cn")
		description := entry.GetAttributeValue("description")
		members := entry.GetAttributeValues("member")

		// 提取所属OU DN
		ouDN := extractParentDN(groupDN)

		// 保存成员信息供后续同步
		groupMembersMap[groupDN] = members

		if existingGroup, exists := existingGroupMap[groupDN]; exists {
			// 已存在，标记为需要更新
			existingGroup.GroupName = groupName
			existingGroup.Description = description
			existingGroup.MemberCount = len(members)
			existingGroup.OUN = ouDN
			existingGroup.LastSyncAt = &now
			groupsToUpdate[groupDN] = existingGroup
		} else {
			// 不存在，准备创建
			groupsToCreate = append(groupsToCreate, models.ADGroup{
				ADConfigID:  config.ID,
				GroupDN:     groupDN,
				GroupName:   groupName,
				Description: description,
				MemberCount: len(members),
				OUN:         ouDN,
				LastSyncAt:  &now,
			})
		}
	}

	// 批量创建新的 Group
	if len(groupsToCreate) > 0 {
		if err := s.db.WithContext(ctx).Create(&groupsToCreate).Error; err != nil {
			return fmt.Errorf("批量创建用户组失败: %w", err)
		}
	}

	// 批量更新已存在的 Group
	for _, group := range groupsToUpdate {
		if err := s.db.WithContext(ctx).Save(group).Error; err != nil {
			return fmt.Errorf("更新用户组失败: %w", err)
		}
	}

	// 批量同步成员关系
	for groupDN, members := range groupMembersMap {
		if err := s.syncGroupMembers(ctx, config, groupDN, members); err != nil {
			return err
		}
	}

	return nil
}

// syncUsers 同步用户数据
func (s *SyncService) syncUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	// 批量获取所有 User DN
	userDNs := make([]string, 0, len(entries))
	for _, entry := range entries {
		username := entry.GetAttributeValue("sAMAccountName")
		if username != "" {
			userDNs = append(userDNs, entry.DN)
		}
	}

	if len(userDNs) == 0 {
		return nil
	}

	// 批量查询已存在的 User
	var existingUsers []models.ADUser
	s.db.WithContext(ctx).Where("ad_config_id = ? AND user_dn IN ?", config.ID, userDNs).Find(&existingUsers)

	// 创建 DN -> User 的映射
	existingUserMap := make(map[string]*models.ADUser)
	for i := range existingUsers {
		existingUserMap[existingUsers[i].UserDN] = &existingUsers[i]
	}

	// 准备批量操作的数据
	var usersToCreate []models.ADUser
	usersToUpdate := make(map[string]*models.ADUser)
	now := time.Now()

	for _, entry := range entries {
		userDN := entry.DN
		username := entry.GetAttributeValue("sAMAccountName")
		if username == "" {
			continue
		}

		// 提取所属OU DN
		ouDN := extractParentDN(userDN)

		// 解析用户账户控制标志
		uacStr := entry.GetAttributeValue("userAccountControl")
		userAccountControl := parseIntOrDefault(uacStr, 512)

		// 解析memberOf（第一个）
		memberOf := entry.GetAttributeValue("memberOf")

		// 解析时间字段
		var lastLogon, pwdLastSet *time.Time
		if lastLogonStr := entry.GetAttributeValue("lastLogon"); lastLogonStr != "" {
			lastLogon = parseFileTime(lastLogonStr)
		}
		if pwdLastSetStr := entry.GetAttributeValue("pwdLastSet"); pwdLastSetStr != "" {
			pwdLastSet = parseFileTime(pwdLastSetStr)
		}

		if existingUser, exists := existingUserMap[userDN]; exists {
			// 已存在，标记为需要更新
			existingUser.Username = username
			existingUser.DisplayName = entry.GetAttributeValue("displayName")
			existingUser.Email = entry.GetAttributeValue("mail")
			existingUser.Phone = entry.GetAttributeValue("telephoneNumber")
			existingUser.Mobile = entry.GetAttributeValue("mobile")
			existingUser.Title = entry.GetAttributeValue("title")
			existingUser.Department = entry.GetAttributeValue("department")
			existingUser.Company = entry.GetAttributeValue("company")
			existingUser.Description = entry.GetAttributeValue("description")
			existingUser.OUN = ouDN
			existingUser.UserAccountControl = userAccountControl
			existingUser.IsEnabled = !existingUser.IsDisabledByUAC()
			existingUser.IsLocked = existingUser.IsLockedByUAC()
			existingUser.PasswordExpired = existingUser.IsPasswordExpiredByUAC()
			existingUser.MemberOf = memberOf
			existingUser.LastLogon = lastLogon
			existingUser.PasswordLastSet = pwdLastSet
			existingUser.LastSyncAt = &now
			usersToUpdate[userDN] = existingUser
		} else {
			// 不存在，准备创建
			newUser := models.ADUser{
				ADConfigID:         config.ID,
				UserDN:             userDN,
				Username:           username,
				DisplayName:        entry.GetAttributeValue("displayName"),
				Email:              entry.GetAttributeValue("mail"),
				Phone:              entry.GetAttributeValue("telephoneNumber"),
				Mobile:             entry.GetAttributeValue("mobile"),
				Title:              entry.GetAttributeValue("title"),
				Department:         entry.GetAttributeValue("department"),
				Company:            entry.GetAttributeValue("company"),
				Description:        entry.GetAttributeValue("description"),
				OUN:                ouDN,
				UserAccountControl: userAccountControl,
				MemberOf:           memberOf,
				LastLogon:          lastLogon,
				PasswordLastSet:    pwdLastSet,
				LastSyncAt:         &now,
			}
			// 设置从UAC派生的字段
			newUser.IsEnabled = !newUser.IsDisabledByUAC()
			newUser.IsLocked = newUser.IsLockedByUAC()
			newUser.PasswordExpired = newUser.IsPasswordExpiredByUAC()

			usersToCreate = append(usersToCreate, newUser)
		}
	}

	// 批量创建新的 User（分批）
	if len(usersToCreate) > 0 {
		batchSize := 100
		for i := 0; i < len(usersToCreate); i += batchSize {
			end := i + batchSize
			if end > len(usersToCreate) {
				end = len(usersToCreate)
			}
			if err := s.db.WithContext(ctx).Create(usersToCreate[i:end]).Error; err != nil {
				return fmt.Errorf("批量创建用户失败: %w", err)
			}
		}
	}

	// 批量更新已存在的 User
	for _, user := range usersToUpdate {
		if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
			return fmt.Errorf("更新用户失败: %w", err)
		}
	}

	return nil
}

// syncGroupMembers 同步用户组成员关系
func (s *SyncService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
	// 先删除旧的成员关系（使用 Unscoped 真正删除，因为表有唯一约束）
	s.db.WithContext(ctx).Unscoped().Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Delete(&models.ADGroupMember{})

	// 批量添加新的成员关系
	if len(memberDNs) == 0 {
		return nil
	}

	// 准备批量数据
	var members []models.ADGroupMember
	members = make([]models.ADGroupMember, 0, len(memberDNs))
	for _, userDN := range memberDNs {
		members = append(members, models.ADGroupMember{
			ADConfigID: config.ID,
			GroupDN:    groupDN,
			UserDN:     userDN,
		})
	}

	// 使用批量插入
	if err := s.db.WithContext(ctx).Create(&members).Error; err != nil {
		return fmt.Errorf("批量添加组成员失败: %w", err)
	}

	return nil
}

// updateSyncLog 更新同步日志
func (s *SyncService) updateSyncLog(ctx context.Context, logID string, status models.ADSyncStatus, ouCount, groupCount, userCount, computerCount int, errorMsg string) {
	updates := map[string]interface{}{
		"sync_status":    status,
		"ou_count":       ouCount,
		"group_count":    groupCount,
		"user_count":     userCount,
		"computer_count": computerCount,
		"error_count":    0,
	}

	if status != models.ADSyncStatusRunning {
		now := time.Now()
		updates["end_time"] = now
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
		updates["error_count"] = 1
	}

	s.db.WithContext(ctx).Model(&models.ADSyncLog{}).Where("id = ?", logID).Updates(updates)
}
