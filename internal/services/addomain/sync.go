package addomain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncService AD数据同步服务
type SyncService struct {
	db *gorm.DB
	// Phase 38 Wave 1 DI 脚手架：AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）
	pool AccountPool
	// P1 fix: singleflight 防止同一 configID 的同步并发执行
	// (scheduler 5 分钟轮询 + 用户手动点击 + 调度漂移叠加),
	// 否则两份同步流程同时跑会产生:
	//   - LDAP 双倍请求加重 AD 服务器负载
	//   - 中间状态 DB 写冲突 (OU/Group/User 重复 upsert)
	//   - 部分软删除被另一份并发恢复导致状态错乱
	syncGroup singleflight.Group
}

// NewSyncService 创建同步服务
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）。
func NewSyncService(db *gorm.DB, pool AccountPool) *SyncService {
	return &SyncService{db: db, pool: pool}
}

// SyncResult 同步结果
type SyncResult struct {
	OUCount       int `json:"ouCount"`
	GroupCount    int `json:"groupCount"`
	UserCount     int `json:"userCount"`
	ComputerCount int `json:"computerCount"`
}

// SyncDataByID 根据配置ID同步数据
func (s *SyncService) SyncDataByID(ctx context.Context, configID string, syncType string) (*SyncResult, error) {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", configID, models.ADConfigStatusEnabled).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在或未启用")
	}
	return s.SyncData(ctx, &config, syncType)
}

// SyncData 同步数据
func (s *SyncService) SyncData(ctx context.Context, config *models.ADConfig, syncType string) (*SyncResult, error) {
	// P1 fix: 使用 singleflight 保证同 configID + syncType 的并发同步合并为一次。
	// scheduler 轮询 + 手动触发 + 调度漂移并发场景下:
	//   - 第一个 goroutine 真正执行同步
	//   - 后续相同 key 的 goroutine 共享结果,不重复发起 LDAP/写库
	//   - 极大降低 AD 负载与 DB 写冲突
	key := fmt.Sprintf("sync:%s:%s", config.ID, syncType)
	v, err, shared := s.syncGroup.Do(key, func() (interface{}, error) {
		return s.syncDataInternal(ctx, config, syncType)
	})
	if shared {
		applogger.Infof("[AD同步] 合并并发请求 key=%s, 复用同一次执行结果", key)
	}
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*SyncResult), nil
}

// syncDataInternal 实际执行同步逻辑(由 singleflight 包裹保证互斥)
func (s *SyncService) syncDataInternal(ctx context.Context, config *models.ADConfig, syncType string) (*SyncResult, error) {
	overallStart := time.Now()
	applogger.Infof("[AD同步] 开始同步配置: %s", config.ConfigName)

	syncLog := &models.ADSyncLog{
		ADConfigID: config.ID,
		SyncType:   models.ADSyncType(syncType),
		SyncStatus: models.ADSyncStatusRunning,
		StartTime:  time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(syncLog).Error; err != nil {
		return nil, fmt.Errorf("创建同步日志失败: %w", err)
	}


	// Phase 38 Wave 1: 整个同步流程（多次 LDAP Search）封装进 FailoverClient.ExecuteWithFailover
	// 一个 operation。所有 client.Search* 必须在闭包内（Pitfall 3：闭包返回后 client.Close()）；
	// DB 写入（syncOUs/syncGroups/...）依赖 Search 结果，因此在闭包内 collect 后传出。
	fc := NewFailoverClient(s.pool, config)
	var (
		ous      []*ldap.Entry
		groups   []*ldap.Entry
		users    []*ldap.Entry
		computers []*ldap.Entry
	)
	if err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		var err error
		// 2. 搜索 OU
		if ous, err = client.SearchOUs(config.BaseDN); err != nil {
			return fmt.Errorf("搜索OU失败: %w", err)
		}
		// 3. 搜索 Group
		if groups, err = client.SearchGroups(config.BaseDN); err != nil {
			return fmt.Errorf("搜索用户组失败: %w", err)
		}
		// 4. 搜索 User
		if users, err = client.SearchUsers(config.BaseDN); err != nil {
			return fmt.Errorf("搜索用户失败: %w", err)
		}
		// 5. 搜索 Computer
		if computers, err = client.SearchComputers(config.BaseDN); err != nil {
			return fmt.Errorf("搜索电脑设备失败: %w", err)
		}
		return nil
	}); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}

	result := &SyncResult{}
	result.OUCount = len(ous)

	if err := s.syncOUs(ctx, config, ous); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, 0, 0, 0, err.Error())
		return nil, err
	}

	result.GroupCount = len(groups)

	if err := s.syncGroups(ctx, config, groups); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, 0, 0, err.Error())
		return nil, err
	}

	result.UserCount = len(users)

	if err := s.syncUsers(ctx, config, users); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, 0, err.Error())
		return nil, err
	}

	computerService := NewComputerService(s.db)
	result.ComputerCount = len(computers)

	if err := computerService.syncComputers(ctx, config, computers); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount, err.Error())
		return nil, err
	}

	// 更新配置最后同步时间
	now := time.Now()
	updateResult := s.db.WithContext(ctx).Model(config).Updates(map[string]interface{}{
		"last_sync_at": now,
	})
	if updateResult.Error != nil {
		applogger.Warnf("更新AD配置last_sync_at失败: %v", updateResult.Error)
	}

	s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusSuccess, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount, "")
	applogger.Infof("[AD同步] 同步完成! 总耗时: %v | OU=%d Group=%d User=%d Computer=%d",
		time.Since(overallStart), result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount)

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

// safeAttr 从 LDAP entry 读取属性值并清洗为可安全写入 PostgreSQL 的字符串。
//
// 与 utils.SanitizeForDB 的区别:此处同时裁剪到对应模型列宽上限,
// 防止 AD 返回超长 description/displayName 触发 GORM Data too long 错误。
// 长度参考 internal/models/ad_domain.go 的 gorm:"size:XXX" 标签:
//   - size:255 → 255
//   - size:500 → 500
//   - size:50  → 50
//   - type:text → 4000(防止单字段爆炸)
//
// 调用方不需要再二次清洗。
func safeAttr(s string, maxLen int) string {
	return utils.SanitizeAndTruncate(s, maxLen)
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
		ouName := safeAttr(entry.GetAttributeValue("ou"), 255)
		description := safeAttr(entry.GetAttributeValue("description"), 4000)
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

// batchUpdateOUs 批量更新OU - 使用 upsert（分批处理）
func (s *SyncService) batchUpdateOUs(ctx context.Context, ous map[string]*models.ADOU) error {
	if len(ous) == 0 {
		return nil
	}

	ouSlice := make([]*models.ADOU, 0, len(ous))
	for _, ou := range ous {
		ouSlice = append(ouSlice, ou)
	}

	// 分批处理，避免超过 PostgreSQL 参数限制
	batchSize := 500
	for i := 0; i < len(ouSlice); i += batchSize {
		end := i + batchSize
		if end > len(ouSlice) {
			end = len(ouSlice)
		}
		batch := ouSlice[i:end]
		if len(batch) > 0 {
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "ou_dn"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"ou_name", "description", "parent_dn", "ou_path", "last_sync_at",
				}),
			}).Create(batch).Error; err != nil {
				return fmt.Errorf("批量更新OU失败: %w", err)
			}
		}
	}

	return nil
}

// syncGroups 同步用户组数据
func (s *SyncService) syncGroups(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

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

		// Parse groupType from LDAP
		groupScope, groupType := parseGroupTypeFromLDAP(entry.GetAttributeValue("groupType"))

		if existingGroup, exists := existingGroupMap[groupDN]; exists {
			existingGroup.GroupName = safeAttr(entry.GetAttributeValue("cn"), 255)
			existingGroup.Description = safeAttr(entry.GetAttributeValue("description"), 4000)
			existingGroup.MemberCount = len(members)
			existingGroup.OUN = ouDN
			existingGroup.GroupScope = groupScope
			existingGroup.GroupType = groupType
			existingGroup.LastSyncAt = &now
			groupsToUpdate[groupDN] = existingGroup
		} else {
			groupsToCreate = append(groupsToCreate, models.ADGroup{
				ADConfigID:  config.ID,
				GroupDN:     groupDN,
				GroupName:   safeAttr(entry.GetAttributeValue("cn"), 255),
				Description: safeAttr(entry.GetAttributeValue("description"), 4000),
				MemberCount: len(members),
				OUN:         ouDN,
				GroupScope:  groupScope,
				GroupType:   groupType,
				LastSyncAt:  &now,
			})
		}
	}

	if err := s.createGroupsInBatches(ctx, groupsToCreate); err != nil {
		return err
	}

	if err := s.updateGroupsInBatches(ctx, groupsToUpdate); err != nil {
		return err
	}

	for groupDN, members := range groupMembersMap {
		if err := s.syncGroupMembers(ctx, config, groupDN, members); err != nil {
			return err
		}
	}

	return nil
}

// createGroupsInBatches 批量创建用户组
func (s *SyncService) createGroupsInBatches(ctx context.Context, groups []models.ADGroup) error {
	if len(groups) == 0 {
		return nil
	}

	batchSize := 500
	for i := 0; i < len(groups); i += batchSize {
		end := i + batchSize
		if end > len(groups) {
			end = len(groups)
		}
		if err := s.db.WithContext(ctx).Create(groups[i:end]).Error; err != nil {
			return fmt.Errorf("批量创建用户组失败: %w", err)
		}
	}

	return nil
}

// updateGroupsInBatches 批量更新用户组
func (s *SyncService) updateGroupsInBatches(ctx context.Context, groups map[string]*models.ADGroup) error {
	if len(groups) == 0 {
		return nil
	}

	groupSlice := make([]*models.ADGroup, 0, len(groups))
	for _, group := range groups {
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

	return nil
}

// syncUsers 同步用户数据
func (s *SyncService) syncUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	totalStart := time.Now()
	applogger.Infof("[用户同步] 开始同步 %d 个用户", len(entries))

	// 1. 提取DN列表
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

	// 2. 批量查询已存在的用户（分批查询以优化性能）
	var existingUsers []models.ADUser
	queryBatchSize := 500 // 每批查询 500 个 DN，避免 IN 子句过大
	for i := 0; i < len(userDNs); i += queryBatchSize {
		end := i + queryBatchSize
		if end > len(userDNs) {
			end = len(userDNs)
		}
		batch := userDNs[i:end]
		var batchUsers []models.ADUser
		s.db.WithContext(ctx).Where("ad_config_id = ? AND user_dn IN ?", config.ID, batch).Find(&batchUsers)
		existingUsers = append(existingUsers, batchUsers...)
	}

	// 3. 创建映射
	existingUserMap := make(map[string]*models.ADUser)
	for i := range existingUsers {
		existingUserMap[existingUsers[i].UserDN] = &existingUsers[i]
	}

	// 4. 处理条目
	var usersToCreate []models.ADUser
	usersToUpdate := make(map[string]*models.ADUser)
	now := time.Now()

	for _, entry := range entries {
		userDN := entry.DN
		username := safeAttr(entry.GetAttributeValue("sAMAccountName"), 255)
		if username == "" {
			continue
		}

		// 提取所属OU DN
		ouDN := extractParentDN(userDN)

		// 解析用户账户控制标志
		uacStr := entry.GetAttributeValue("userAccountControl")
		userAccountControl := parseIntOrDefault(uacStr, 512)

		// memberOf is multi-valued, store as semicolon-separated
		// (对每个值单独清洗,然后 join,避免 NUL/无效 UTF-8 污染连接结果)
		memberOfValues := entry.GetAttributeValues("memberOf")
		memberOfParts := make([]string, 0, len(memberOfValues))
		for _, v := range memberOfValues {
			if s := safeAttr(v, 500); s != "" {
				memberOfParts = append(memberOfParts, s)
			}
		}
		memberOf := strings.Join(memberOfParts, ";")

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
			existingUser.DisplayName = safeAttr(entry.GetAttributeValue("displayName"), 255)
			existingUser.Email = safeAttr(entry.GetAttributeValue("mail"), 255)
			existingUser.Phone = safeAttr(entry.GetAttributeValue("telephoneNumber"), 50)
			existingUser.Mobile = safeAttr(entry.GetAttributeValue("mobile"), 50)
			existingUser.Title = safeAttr(entry.GetAttributeValue("title"), 100)
			existingUser.Department = safeAttr(entry.GetAttributeValue("department"), 255)
			existingUser.Company = safeAttr(entry.GetAttributeValue("company"), 255)
			existingUser.Description = safeAttr(entry.GetAttributeValue("description"), 4000)
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
				DisplayName:        safeAttr(entry.GetAttributeValue("displayName"), 255),
				Email:              safeAttr(entry.GetAttributeValue("mail"), 255),
				Phone:              safeAttr(entry.GetAttributeValue("telephoneNumber"), 50),
				Mobile:             safeAttr(entry.GetAttributeValue("mobile"), 50),
				Title:              safeAttr(entry.GetAttributeValue("title"), 100),
				Department:         safeAttr(entry.GetAttributeValue("department"), 255),
				Company:            safeAttr(entry.GetAttributeValue("company"), 255),
				Description:        safeAttr(entry.GetAttributeValue("description"), 4000),
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

	// 5. 批量创建
	if len(usersToCreate) > 0 {
		batchSize := 500
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

	// 6. 批量更新
	if len(usersToUpdate) > 0 {
		batchSize := 500
		usersSlice := make([]*models.ADUser, 0, len(usersToUpdate))
		for _, user := range usersToUpdate {
			usersSlice = append(usersSlice, user)
		}

		for i := 0; i < len(usersSlice); i += batchSize {
			end := i + batchSize
			if end > len(usersSlice) {
				end = len(usersSlice)
			}
			batch := usersSlice[i:end]
			if len(batch) > 0 {
				if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "user_dn"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"username", "display_name", "email", "phone", "mobile",
						"title", "department", "company", "description", "ou_dn",
						"user_account_control", "is_enabled", "is_locked",
						"password_expired", "member_of", "last_logon",
						"password_last_set", "last_sync_at",
					}),
				}).Create(batch).Error; err != nil {
					return fmt.Errorf("批量更新用户失败: %w", err)
				}
			}
		}
	}

	applogger.Infof("[用户同步] 完成: 创建 %d 个, 更新 %d 个, 耗时 %.2fs",
		len(usersToCreate), len(usersToUpdate), time.Since(totalStart).Seconds())
	return nil
}

// syncGroupMembers 同步用户组成员关系
func (s *SyncService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
	// 先删除旧的成员关系
	if err := s.db.WithContext(ctx).Unscoped().Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Delete(&models.ADGroupMember{}).Error; err != nil {
		return fmt.Errorf("清理旧成员关系失败: %w", err)
	}

	if len(memberDNs) == 0 {
		return nil
	}

	// 准备批量数据
	members := make([]models.ADGroupMember, 0, len(memberDNs))
	for _, userDN := range memberDNs {
		members = append(members, models.ADGroupMember{
			ADConfigID: config.ID,
			GroupDN:    groupDN,
			UserDN:     userDN,
		})
	}

	if err := s.db.WithContext(ctx).Create(&members).Error; err != nil {
		return fmt.Errorf("批量添加组成员失败: %w", err)
	}

	return nil
}

// updateSyncLog 更新同步日志
//
// DB 安全:errorMsg 在写入 PostgreSQL TEXT 列之前必须经过 SanitizeForDB,
// 因为 go-ldap v3.4.12 在 error.go:223 直接把 AD 服务器的原始诊断字节
// 包装成 error message,某些 AD 服务器返回的诊断消息含有 NUL (0x00) 或
// 非 UTF-8 序列,会触发 PostgreSQL SQLSTATE 22021 拒绝写入,导致整个
// handler 返回 500。
// 触发场景参见 .planning/debug/ad-sync-500-nul-byte.md (待写)。
func (s *SyncService) updateSyncLog(ctx context.Context, logID string, status models.ADSyncStatus, ouCount, groupCount, userCount, computerCount int, errorMsg string) {
	// 先获取原始日志记录以计算耗时
	var log models.ADSyncLog
	if err := s.db.WithContext(ctx).Where("id = ?", logID).First(&log).Error; err != nil {
		// 如果查询失败，仍然尝试更新（但不计算耗时）
		log = models.ADSyncLog{StartTime: time.Now()}
	}

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
		// 计算耗时（秒）
		duration := int(now.Sub(log.StartTime).Seconds())
		updates["duration"] = duration
	}

	if errorMsg != "" {
		// 关键:剥离 NUL 字节与无效 UTF-8,避免 PostgreSQL 拒绝写入
		safe := utils.SanitizeAndTruncate(errorMsg, 4000)
		updates["error_message"] = safe
		updates["error_count"] = 1
		if safe != errorMsg {
			applogger.Warnf("[AD同步] sync_log.error_message 含非法字符已被清洗 (id=%s)", logID)
		}
	}

	if err := s.db.WithContext(ctx).Model(&models.ADSyncLog{}).Where("id = ?", logID).Updates(updates).Error; err != nil {
		// 写入失败仅记录,不影响同步主流程的错误向上传播
		applogger.Errorf("[AD同步] 写入 sync_log 失败 (id=%s): %v", logID, err)
	}
}

// parseGroupTypeFromLDAP parses the AD groupType integer flag into GroupScope and GroupType
// AD groupType values:
//   - 0x00000002 (2)   = Global Distribution Group
//   - 0x00000004 (4)   = Domain Local Distribution Group
//   - 0x00000008 (8)   = Universal Distribution Group
//   - 0x80000002 (-2147483646) = Global Security Group
//   - 0x80000004 (-2147483644) = Domain Local Security Group
//   - 0x80000008 (-2147483640) = Universal Security Group
func parseGroupTypeFromLDAP(groupTypeStr string) (models.ADGroupScope, models.ADGroupType) {
	if groupTypeStr == "" {
		return models.ADGroupScopeGlobal, models.ADGroupTypeSecurity
	}

	val := parseIntOrDefault(groupTypeStr, -2147483646)
	isSecurity := val&0x80000000 != 0
	scopeVal := val & 0x0FFFFFFF

	var scope models.ADGroupScope
	switch scopeVal {
	case 2:
		scope = models.ADGroupScopeGlobal
	case 4:
		scope = models.ADGroupScopeLocal
	case 8:
		scope = models.ADGroupScopeUniversal
	default:
		scope = models.ADGroupScopeGlobal
	}

	var groupType models.ADGroupType
	if isSecurity {
		groupType = models.ADGroupTypeSecurity
	} else {
		groupType = models.ADGroupTypeDistribution
	}

	return scope, groupType
}
