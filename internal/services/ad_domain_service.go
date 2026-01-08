package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// ADDomainService AD域管理服务
type ADDomainService struct {
	db *gorm.DB
}

// NewADDomainService 创建AD域管理服务
func NewADDomainService(db *gorm.DB) *ADDomainService {
	return &ADDomainService{db: db}
}

// ==================== 请求/响应结构 ====================

// ADConfigListRequest AD配置列表请求
type ADConfigListRequest struct {
	Status   *int `json:"status,omitempty"`
	Current  int  `json:"current"`
	PageSize int  `json:"pageSize"`
}

// ADConfigCreateRequest 创建AD配置请求
type ADConfigCreateRequest struct {
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	AdminUsername string `json:"adminUsername" binding:"required,max=255"`
	AdminPassword string `json:"adminPassword" binding:"required,max=500"`
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
}

// ADConfigUpdateRequest 更新AD配置请求
type ADConfigUpdateRequest struct {
	ID            string `json:"-"`
	ConfigName    string `json:"configName" binding:"required,max=100"`
	ServerAddress string `json:"serverAddress" binding:"required,max=255"`
	ServerPort    int    `json:"serverPort" binding:"required,min=1,max=65535"`
	DomainName    string `json:"domainName" binding:"required,max=255"`
	BaseDN        string `json:"baseDn" binding:"required,max=500"`
	AdminUsername string `json:"adminUsername" binding:"required,max=255"`
	AdminPassword string `json:"adminPassword,omitempty"` // 可选，为空表示不修改
	UseSSL        bool   `json:"useSsl"`
	UseTLS        bool   `json:"useTls"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncInterval  int    `json:"syncInterval" binding:"min=60"`
	Status        *int   `json:"status,omitempty"`
}

// ADSyncRequest 同步请求
type ADSyncRequest struct {
	SyncType string `json:"syncType"` // full/incremental
}

// ADUserListRequest AD用户列表请求
type ADUserListRequest struct {
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	Username  *string `json:"username,omitempty"`
	IsEnabled *bool   `json:"isEnabled,omitempty"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
}

// ADGroupListRequest AD用户组列表请求
type ADGroupListRequest struct {
	ConfigID  string  `json:"configId" binding:"required"`
	OUDN      *string `json:"ouDn,omitempty"`
	GroupName *string `json:"groupName,omitempty"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
}

// ADUserUpdateRequest AD用户更新请求
type ADUserUpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Mobile      *string `json:"mobile,omitempty"`
	Title       *string `json:"title,omitempty"`
	Department  *string `json:"department,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ADGroupUpdateRequest AD用户组更新请求
type ADGroupUpdateRequest struct {
	GroupName   *string `json:"groupName,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ADUserMoveRequest 移动用户请求
type ADUserMoveRequest struct {
	NewOUDN string `json:"newOuDn" binding:"required"`
}

// ADGroupMemberRequest 用户组成员操作请求
type ADGroupMemberRequest struct {
	UserDN string `json:"userDn" binding:"required"`
}

// ADSyncResult 同步结果
type ADSyncResult struct {
	OUCount    int `json:"ouCount"`
	GroupCount int `json:"groupCount"`
	UserCount  int `json:"userCount"`
}

// ADOUNode OU树节点
type ADOUNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	DN       string      `json:"dn"`
	Path     string      `json:"path,omitempty"`
	Children []ADOUNode  `json:"children,omitempty"`
}

// ==================== AD配置管理 ====================

// GetADConfigList 获取AD配置列表
func (s *ADDomainService) GetADConfigList(ctx context.Context, status *int, current, pageSize int) ([]models.ADConfig, int64, error) {
	var configs []models.ADConfig
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADConfig{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	query.Count(&total)

	offset := (current - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询AD配置列表失败: %w", err)
	}

	return configs, total, nil
}

// GetADConfigByID 根据ID获取AD配置
func (s *ADDomainService) GetADConfigByID(ctx context.Context, id string) (*models.ADConfig, error) {
	var config models.ADConfig
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("AD配置不存在")
		}
		return nil, fmt.Errorf("查询AD配置失败: %w", err)
	}
	return &config, nil
}

// CreateADConfig 创建AD配置
func (s *ADDomainService) CreateADConfig(ctx context.Context, req *ADConfigCreateRequest, creatorID string) (*models.ADConfig, error) {
	// 验证配置是否可以连接
	tempConfig := &models.ADConfig{
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		DomainName:    req.DomainName,
		BaseDN:        req.BaseDN,
		AdminUsername: req.AdminUsername,
		AdminPassword: req.AdminPassword,
		UseSSL:        req.UseSSL,
		UseTLS:        req.UseTLS,
	}

	client := NewLDAPClient(tempConfig)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("连接测试失败: %w", err)
	}
	client.Close()

	// 保存配置
	config := &models.ADConfig{
		BaseModel: models.BaseModel{
			CreatedBy: creatorID,
			Version:   0,
		},
		ConfigName:    req.ConfigName,
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		DomainName:    req.DomainName,
		BaseDN:        req.BaseDN,
		AdminUsername: req.AdminUsername,
		AdminPassword: encryptPassword(req.AdminPassword), // 加密存储
		UseSSL:        req.UseSSL,
		UseTLS:        req.UseTLS,
		SyncEnabled:   req.SyncEnabled,
		SyncInterval:  req.SyncInterval,
		Status:        models.ADConfigStatusEnabled,
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return nil, fmt.Errorf("保存AD配置失败: %w", err)
	}

	// 清除密码字段后返回
	config.AdminPassword = ""
	return config, nil
}

// UpdateADConfig 更新AD配置
func (s *ADDomainService) UpdateADConfig(ctx context.Context, req *ADConfigUpdateRequest, updaterID string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	// 如果提供了新密码，需要验证连接
	if req.AdminPassword != "" {
		tempConfig := config
		tempConfig.ServerAddress = req.ServerAddress
		tempConfig.ServerPort = req.ServerPort
		tempConfig.AdminUsername = req.AdminUsername
		tempConfig.AdminPassword = req.AdminPassword
		tempConfig.UseSSL = req.UseSSL
		tempConfig.UseTLS = req.UseTLS

		client := NewLDAPClient(&tempConfig)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("连接测试失败: %w", err)
		}
		client.Close()
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"config_name":    req.ConfigName,
		"server_address": req.ServerAddress,
		"server_port":    req.ServerPort,
		"domain_name":    req.DomainName,
		"base_dn":        req.BaseDN,
		"admin_username": req.AdminUsername,
		"use_ssl":        req.UseSSL,
		"use_tls":        req.UseTLS,
		"sync_enabled":   req.SyncEnabled,
		"sync_interval":  req.SyncInterval,
		"updated_by":     updaterID,
	}

	// 只有提供了新密码才更新密码
	if req.AdminPassword != "" {
		updates["admin_password"] = encryptPassword(req.AdminPassword)
	}

	// 更新状态
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 更新版本号
	updates["version"] = config.Version + 1

	if err := s.db.WithContext(ctx).Model(&config).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新AD配置失败: %w", err)
	}

	return nil
}

// DeleteADConfig 删除AD配置
func (s *ADDomainService) DeleteADConfig(ctx context.Context, id string) error {
	// 软删除
	result := s.db.WithContext(ctx).Delete(&models.ADConfig{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("删除AD配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("AD配置不存在")
	}
	return nil
}

// TestADConnection 测试AD连接
func (s *ADDomainService) TestADConnection(ctx context.Context, configID string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	// 解密密码
	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	return nil
}

// ==================== AD数据同步 ====================

// SyncADData 同步AD数据
func (s *ADDomainService) SyncADData(ctx context.Context, configID string, syncType string) (*ADSyncResult, error) {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", configID, models.ADConfigStatusEnabled).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在或未启用")
	}

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

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer client.Close()

	result := &ADSyncResult{}

	// 同步OU
	ous, err := client.SearchOUs(config.BaseDN)
	if err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
		return nil, fmt.Errorf("搜索OU失败: %w", err)
	}
	result.OUCount = len(ous)
	if err := s.syncOUs(ctx, &config, ous); err != nil {
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
	if err := s.syncGroups(ctx, &config, groups); err != nil {
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
	if err := s.syncUsers(ctx, &config, users); err != nil {
		s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, result.OUCount, result.GroupCount, result.UserCount, 0, err.Error())
		return nil, err
	}

	// 更新配置的最后同步时间
	now := time.Now()
	s.db.WithContext(ctx).Model(&config).Updates(map[string]interface{}{
		"last_sync_at": now,
	})

	// 完成同步日志
	s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusSuccess, result.OUCount, result.GroupCount, result.UserCount, 0, "")

	return result, nil
}

// syncOUs 同步OU数据（优化批量操作）
func (s *ADDomainService) syncOUs(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	// 批量获取所有 OU DN
	ouDNs := make([]string, 0, len(entries))
	for _, entry := range entries {
		ouDNs = append(ouDNs, entry.DN)
	}

	// 批量查询已存在的 OU
	var existingOUs []models.ADOU
	s.db.WithContext(ctx).Where("ad_config_id = ? AND ou_dn IN ?", config.ID, ouDNs).Find(&existingOUs)

	// 创建 DN -> OU 的映射
	existingOUMap := make(map[string]*models.ADOU)
	for i := range existingOUs {
		existingOUMap[existingOUs[i].OUN] = &existingOUs[i]
	}

	// 准备批量操作的数据
	var ousToCreate []models.ADOU
	ousToUpdate := make(map[string]*models.ADOU)
	now := time.Now()

	for _, entry := range entries {
		ouDN := entry.DN
		ouName := entry.GetAttributeValue("ou")
		description := entry.GetAttributeValue("description")

		// 提取父OU DN
		parentDN := s.extractParentDN(ouDN)

		// 构建OU路径
		ouPath := s.buildOUPath(ouDN, config.BaseDN)

		if existingOU, exists := existingOUMap[ouDN]; exists {
			// 已存在，标记为需要更新
			existingOU.OUName = ouName
			existingOU.Description = description
			existingOU.ParentDN = parentDN
			existingOU.OUPath = ouPath
			existingOU.LastSyncAt = &now
			ousToUpdate[ouDN] = existingOU
		} else {
			// 不存在，准备创建
			ousToCreate = append(ousToCreate, models.ADOU{
				ADConfigID: config.ID,
				OUN:        ouDN,
				OUName:     ouName,
				Description: description,
				ParentDN:   parentDN,
				OUPath:     ouPath,
				LastSyncAt: &now,
			})
		}
	}

	// 批量创建新的 OU
	if len(ousToCreate) > 0 {
		if err := s.db.WithContext(ctx).Create(&ousToCreate).Error; err != nil {
			return fmt.Errorf("批量创建OU失败: %w", err)
		}
	}

	// 批量更新已存在的 OU（使用 Save 逐个更新，因为需要更新特定字段）
	for _, ou := range ousToUpdate {
		if err := s.db.WithContext(ctx).Save(ou).Error; err != nil {
			return fmt.Errorf("更新OU失败: %w", err)
		}
	}

	return nil
}

// syncGroups 同步用户组数据（优化批量操作）
func (s *ADDomainService) syncGroups(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
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
		ouDN := s.extractParentDN(groupDN)

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
				ADConfigID: config.ID,
				GroupDN:    groupDN,
				GroupName:  groupName,
				Description: description,
				MemberCount: len(members),
				OUN:        ouDN,
				LastSyncAt: &now,
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

// syncUsers 同步用户数据（优化批量操作）
func (s *ADDomainService) syncUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
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
			continue // 跳过没有用户名的条目
		}

		// 提取所属OU DN
		ouDN := s.extractParentDN(userDN)

		// 解析用户账户控制标志
		uacStr := entry.GetAttributeValue("userAccountControl")
		userAccountControl := s.parseIntOrDefault(uacStr, 512)

		// 解析memberOf（第一个）
		memberOf := entry.GetAttributeValue("memberOf")

		// 解析时间字段（AD使用FileTime格式）
		var lastLogon, pwdLastSet *time.Time
		if lastLogonStr := entry.GetAttributeValue("lastLogon"); lastLogonStr != "" {
			lastLogon = s.parseFileTime(lastLogonStr)
		}
		if pwdLastSetStr := entry.GetAttributeValue("pwdLastSet"); pwdLastSetStr != "" {
			pwdLastSet = s.parseFileTime(pwdLastSetStr)
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
				ADConfigID:       config.ID,
				UserDN:          userDN,
				Username:        username,
				DisplayName:     entry.GetAttributeValue("displayName"),
				Email:           entry.GetAttributeValue("mail"),
				Phone:           entry.GetAttributeValue("telephoneNumber"),
				Mobile:          entry.GetAttributeValue("mobile"),
				Title:           entry.GetAttributeValue("title"),
				Department:      entry.GetAttributeValue("department"),
				Company:         entry.GetAttributeValue("company"),
				Description:     entry.GetAttributeValue("description"),
				OUN:             ouDN,
				UserAccountControl: userAccountControl,
				MemberOf:        memberOf,
				LastLogon:       lastLogon,
				PasswordLastSet: pwdLastSet,
				LastSyncAt:      &now,
			}
			// 设置从UAC派生的字段
			newUser.IsEnabled = !newUser.IsDisabledByUAC()
			newUser.IsLocked = newUser.IsLockedByUAC()
			newUser.PasswordExpired = newUser.IsPasswordExpiredByUAC()

			usersToCreate = append(usersToCreate, newUser)
		}
	}

	// 批量创建新的 User
	if len(usersToCreate) > 0 {
		// 分批插入，避免单次插入过多数据
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
func (s *ADDomainService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
	// 先删除旧的成员关系
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Delete(&models.ADGroupMember{})

	// 批量添加新的成员关系，避免N+1查询
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

	// 使用批量插入，GORM会自动分批处理
	if err := s.db.WithContext(ctx).Create(&members).Error; err != nil {
		// 记录错误但继续执行
		return fmt.Errorf("批量添加组成员失败: %w", err)
	}

	return nil
}

// updateSyncLog 更新同步日志
func (s *ADDomainService) updateSyncLog(ctx context.Context, logID string, status models.ADSyncStatus, ouCount, groupCount, userCount, errorCount int, errorMsg string) {
	updates := map[string]interface{}{
		"sync_status": status,
		"ou_count":     ouCount,
		"group_count":  groupCount,
		"user_count":   userCount,
		"error_count":  errorCount,
	}

	if status != models.ADSyncStatusRunning {
		now := time.Now()
		updates["end_time"] = now
		// 计算持续时间需要从数据库获取开始时间，这里简化处理
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	s.db.WithContext(ctx).Model(&models.ADSyncLog{}).Where("id = ?", logID).Updates(updates)
}

// ==================== OU管理 ====================

// GetOUTree 获取OU树形结构
func (s *ADDomainService) GetOUTree(ctx context.Context, configID string) ([]ADOUNode, error) {
	var ous []models.ADOU
	if err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", configID).
		Find(&ous).Error; err != nil {
		return nil, fmt.Errorf("查询OU失败: %w", err)
	}

	return s.buildOUTree(ous, ""), nil
}

// buildOUTree 构建OU树
func (s *ADDomainService) buildOUTree(ous []models.ADOU, parentDN string) []ADOUNode {
	var tree []ADOUNode

	// 创建DN到OU的映射
	ouMap := make(map[string]*models.ADOU)
	for i := range ous {
		ouMap[ous[i].OUN] = &ous[i]
	}

	// 构建树
	for _, ou := range ous {
		if ou.ParentDN == parentDN {
			node := ADOUNode{
				ID:   ou.ID,
				Name: ou.OUName,
				DN:   ou.OUN,
				Path: ou.OUPath,
			}
			node.Children = s.buildOUTree(ous, ou.OUN)
			tree = append(tree, node)
		}
	}

	return tree
}

// GetOUByDN 根据DN获取OU
func (s *ADDomainService) GetOUByDN(ctx context.Context, configID, ouDN string) (*models.ADOU, error) {
	var ou models.ADOU
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND ou_dn = ? AND deleted_at IS NULL", configID, ouDN).
		First(&ou).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OU不存在")
		}
		return nil, fmt.Errorf("查询OU失败: %w", err)
	}
	return &ou, nil
}

// ==================== 用户管理 ====================

// GetADUserList 获取AD用户列表
func (s *ADDomainService) GetADUserList(ctx context.Context, req *ADUserListRequest) ([]models.ADUser, int64, error) {
	var users []models.ADUser
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID)

	if req.OUDN != nil {
		query = query.Where("ou_dn = ?", *req.OUDN)
	}

	if req.Username != nil {
		query = query.Where("username LIKE ?", "%"+*req.Username+"%")
	}

	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	query.Count(&total)

	offset := (req.Current - 1) * req.PageSize
	err := query.Offset(offset).Limit(req.PageSize).Order("username ASC").Find(&users).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询AD用户列表失败: %w", err)
	}

	return users, total, nil
}

// GetADUserByDN 根据DN获取用户
func (s *ADDomainService) GetADUserByDN(ctx context.Context, configID, userDN string) (*models.ADUser, error) {
	var user models.ADUser
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND user_dn = ? AND deleted_at IS NULL", configID, userDN).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// UpdateADUser 更新AD用户
func (s *ADDomainService) UpdateADUser(ctx context.Context, configID, userDN string, req *ADUserUpdateRequest) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer client.Close()

	// 构建更新属性
	attrs := make(map[string]string)
	if req.DisplayName != nil {
		attrs["displayName"] = *req.DisplayName
	}
	if req.Email != nil {
		attrs["mail"] = *req.Email
	}
	if req.Phone != nil {
		attrs["telephoneNumber"] = *req.Phone
	}
	if req.Mobile != nil {
		attrs["mobile"] = *req.Mobile
	}
	if req.Title != nil {
		attrs["title"] = *req.Title
	}
	if req.Department != nil {
		attrs["department"] = *req.Department
	}
	if req.Description != nil {
		attrs["description"] = *req.Description
	}

	if err := client.UpdateUserAttribute(userDN, attrs); err != nil {
		return err
	}

	// 更新本地缓存
	updateData := make(map[string]interface{})
	if req.DisplayName != nil {
		updateData["display_name"] = *req.DisplayName
	}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Phone != nil {
		updateData["phone"] = *req.Phone
	}
	if req.Mobile != nil {
		updateData["mobile"] = *req.Mobile
	}
	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Department != nil {
		updateData["department"] = *req.Department
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}

	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", configID, userDN).
		Updates(updateData)

	return nil
}

// EnableADUser 启用AD用户
func (s *ADDomainService) EnableADUser(ctx context.Context, configID, userDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.EnableUser(userDN); err != nil {
		return err
	}

	// 更新本地缓存
	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", configID, userDN).
		Update("is_enabled", true)

	return nil
}

// DisableADUser 禁用AD用户
func (s *ADDomainService) DisableADUser(ctx context.Context, configID, userDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.DisableUser(userDN); err != nil {
		return err
	}

	// 更新本地缓存
	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", configID, userDN).
		Update("is_enabled", false)

	return nil
}

// MoveADUser 移动AD用户到其他OU
func (s *ADDomainService) MoveADUser(ctx context.Context, configID, userDN, newOUDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.MoveUser(userDN, newOUDN); err != nil {
		return err
	}

	// 更新本地缓存
	s.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("ad_config_id = ? AND user_dn = ?", configID, userDN).
		Update("ou_dn", newOUDN)

	return nil
}

// ==================== 用户组管理 ====================

// GetADGroupList 获取AD用户组列表
func (s *ADDomainService) GetADGroupList(ctx context.Context, req *ADGroupListRequest) ([]models.ADGroup, int64, error) {
	var groups []models.ADGroup
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID)

	if req.OUDN != nil {
		query = query.Where("ou_dn = ?", *req.OUDN)
	}

	if req.GroupName != nil {
		query = query.Where("group_name LIKE ?", "%"+*req.GroupName+"%")
	}

	query.Count(&total)

	offset := (req.Current - 1) * req.PageSize
	err := query.Offset(offset).Limit(req.PageSize).Order("group_name ASC").Find(&groups).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询AD用户组列表失败: %w", err)
	}

	return groups, total, nil
}

// GetADGroupByDN 根据DN获取用户组
func (s *ADDomainService) GetADGroupByDN(ctx context.Context, configID, groupDN string) (*models.ADGroup, error) {
	var group models.ADGroup
	err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND group_dn = ? AND deleted_at IS NULL", configID, groupDN).
		First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户组不存在")
		}
		return nil, fmt.Errorf("查询用户组失败: %w", err)
	}
	return &group, nil
}

// GetADGroupMembers 获取用户组成员
func (s *ADDomainService) GetADGroupMembers(ctx context.Context, configID, groupDN string, current, pageSize int) ([]models.ADUser, int64, error) {
	// 从成员关系表获取用户DN列表
	var members []models.ADGroupMember
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADGroupMember{}).
		Where("ad_config_id = ? AND group_dn = ?", configID, groupDN)

	query.Count(&total)

	offset := (current - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&members)

	// 根据用户DN获取用户详情
	var userDNs []string
	for _, m := range members {
		userDNs = append(userDNs, m.UserDN)
	}

	var users []models.ADUser
	if len(userDNs) > 0 {
		s.db.WithContext(ctx).
			Where("ad_config_id = ? AND user_dn IN ? AND deleted_at IS NULL", configID, userDNs).
			Find(&users)
	}

	return users, total, nil
}

// AddADGroupMember 添加用户组成员
func (s *ADDomainService) AddADGroupMember(ctx context.Context, configID, groupDN, userDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.AddGroupMember(groupDN, userDN); err != nil {
		return err
	}

	// 更新本地成员关系
	member := &models.ADGroupMember{
		ADConfigID: config.ID,
		GroupDN:    groupDN,
		UserDN:     userDN,
	}
	s.db.WithContext(ctx).Create(member)

	// 更新用户组成员数
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", configID, groupDN).
		UpdateColumn("member_count", gorm.Expr("member_count + 1"))

	return nil
}

// RemoveADGroupMember 移除用户组成员
func (s *ADDomainService) RemoveADGroupMember(ctx context.Context, configID, groupDN, userDN string) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	if err := client.RemoveGroupMember(groupDN, userDN); err != nil {
		return err
	}

	// 删除本地成员关系
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn = ? AND user_dn = ?", configID, groupDN, userDN).
		Delete(&models.ADGroupMember{})

	// 更新用户组成员数
	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", configID, groupDN).
		UpdateColumn("member_count", gorm.Expr("member_count - 1"))

	return nil
}

// UpdateADGroup 更新AD用户组
func (s *ADDomainService) UpdateADGroup(ctx context.Context, configID, groupDN string, req *ADGroupUpdateRequest) error {
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return fmt.Errorf("AD配置不存在")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)

	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	// 构建更新属性
	attrs := make(map[string]string)
	if req.GroupName != nil {
		attrs["cn"] = *req.GroupName
	}
	if req.Description != nil {
		attrs["description"] = *req.Description
	}

	if err := client.UpdateGroupAttribute(groupDN, attrs); err != nil {
		return err
	}

	// 更新本地缓存
	updateData := make(map[string]interface{})
	if req.GroupName != nil {
		updateData["group_name"] = *req.GroupName
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}

	s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND group_dn = ?", configID, groupDN).
		Updates(updateData)

	return nil
}

// ==================== 同步日志 ====================

// GetADSyncLogList 获取同步日志列表
func (s *ADDomainService) GetADSyncLogList(ctx context.Context, configID string, current, pageSize int) ([]models.ADSyncLog, int64, error) {
	var logs []models.ADSyncLog
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ADSyncLog{})

	// 只在指定了configID时才添加过滤条件
	if configID != "" {
		query = query.Where("ad_config_id = ?", configID)
	}

	query.Count(&total)

	offset := (current - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("start_time DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询同步日志失败: %w", err)
	}

	return logs, total, nil
}

// ==================== 辅助方法 ====================

// extractParentDN 提取父DN
func (s *ADDomainService) extractParentDN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], ",")
}

// buildOUPath 构建OU路径
func (s *ADDomainService) buildOUPath(ouDN, baseDN string) string {
	if ouDN == baseDN {
		return "/"
	}
	// 简化处理：去掉baseDN后反转DN部分
	parts := strings.Split(ouDN, ",")
	var pathParts []string
	for _, part := range parts {
		if strings.Contains(part, "OU=") {
			name := strings.TrimPrefix(part, "OU=")
			pathParts = append([]string{name}, pathParts...)
		}
	}
	return "/" + strings.Join(pathParts, "/")
}

// parseIntOrDefault 解析整数，失败返回默认值
func (s *ADDomainService) parseIntOrDefault(str string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(str, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// parseFileTime 解析AD FileTime格式
func (s *ADDomainService) parseFileTime(fileTime string) *time.Time {
	// AD FileTime是100纳秒为单位，从1601-01-01开始的计数
	// 这里简化处理，实际需要完整的转换
	var ft int64
	_, err := fmt.Sscanf(fileTime, "%d", &ft)
	if err != nil || ft == 0 {
		return nil
	}

	// 转换为Unix时间戳（简化版）
	// 实际需要：unix = (ft / 10000000) - 11644473600
	unix := (ft / 10000000) - 11644473600
	if unix < 0 {
		return nil
	}

	t := time.Unix(unix, 0)
	return &t
}

// encryptPassword 加密密码（简化实现，实际应使用AES）
func encryptPassword(password string) string {
	// TODO: 实际应使用AES等加密算法
	// 这里为了演示，简单进行base64编码
	return password
}

// decryptPassword 解密密码
func decryptPassword(encrypted string) string {
	// TODO: 实际应使用AES等解密算法
	return encrypted
}
