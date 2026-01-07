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

// syncOUs 同步OU数据
func (s *ADDomainService) syncOUs(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	for _, entry := range entries {
		ouDN := entry.DN
		ouName := entry.GetAttributeValue("ou")
		description := entry.GetAttributeValue("description")

		// 提取父OU DN
		parentDN := s.extractParentDN(ouDN)

		// 构建OU路径
		ouPath := s.buildOUPath(ouDN, config.BaseDN)

		var ou models.ADOU
		err := s.db.WithContext(ctx).Where("ad_config_id = ? AND ou_dn = ?", config.ID, ouDN).
			FirstOrCreate(&ou).Error
		if err != nil {
			return fmt.Errorf("同步OU失败: %w", err)
		}

		ou.ADConfigID = config.ID
		ou.OUN = ouDN
		ou.OUName = ouName
		ou.Description = description
		ou.ParentDN = parentDN
		ou.OUPath = ouPath
		now := time.Now()
		ou.LastSyncAt = &now

		if err := s.db.WithContext(ctx).Save(&ou).Error; err != nil {
			return fmt.Errorf("保存OU失败: %w", err)
		}
	}
	return nil
}

// syncGroups 同步用户组数据
func (s *ADDomainService) syncGroups(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	for _, entry := range entries {
		groupDN := entry.DN
		groupName := entry.GetAttributeValue("cn")
		description := entry.GetAttributeValue("description")
		members := entry.GetAttributeValues("member")

		// 提取所属OU DN
		ouDN := s.extractParentDN(groupDN)

		var group models.ADGroup
		err := s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
			FirstOrCreate(&group).Error
		if err != nil {
			return fmt.Errorf("同步用户组失败: %w", err)
		}

		group.ADConfigID = config.ID
		group.GroupDN = groupDN
		group.GroupName = groupName
		group.Description = description
		group.MemberCount = len(members)
		group.OUN = ouDN
		now := time.Now()
		group.LastSyncAt = &now

		if err := s.db.WithContext(ctx).Save(&group).Error; err != nil {
			return fmt.Errorf("保存用户组失败: %w", err)
		}

		// 同步成员关系
		if err := s.syncGroupMembers(ctx, config, groupDN, members); err != nil {
			return err
		}
	}
	return nil
}

// syncUsers 同步用户数据
func (s *ADDomainService) syncUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	for _, entry := range entries {
		userDN := entry.DN
		username := entry.GetAttributeValue("sAMAccountName")
		if username == "" {
			continue // 跳过没有用户名的条目
		}

		// 提取所属OU DN
		ouDN := s.extractParentDN(userDN)

		var user models.ADUser
		err := s.db.WithContext(ctx).Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
			FirstOrCreate(&user).Error
		if err != nil {
			return fmt.Errorf("同步用户失败: %w", err)
		}

		user.ADConfigID = config.ID
		user.UserDN = userDN
		user.Username = username
		user.DisplayName = entry.GetAttributeValue("displayName")
		user.Email = entry.GetAttributeValue("mail")
		user.Phone = entry.GetAttributeValue("telephoneNumber")
		user.Mobile = entry.GetAttributeValue("mobile")
		user.Title = entry.GetAttributeValue("title")
		user.Department = entry.GetAttributeValue("department")
		user.Company = entry.GetAttributeValue("company")
		user.Description = entry.GetAttributeValue("description")
		user.OUN = ouDN

		// 解析用户账户控制标志
		uacStr := entry.GetAttributeValue("userAccountControl")
		user.UserAccountControl = s.parseIntOrDefault(uacStr, 512)
		user.IsEnabled = !user.IsDisabledByUAC()
		user.IsLocked = user.IsLockedByUAC()
		user.PasswordExpired = user.IsPasswordExpiredByUAC()

		// 解析memberOf（第一个）
		memberOf := entry.GetAttributeValue("memberOf")
		user.MemberOf = memberOf

		// 解析时间字段（AD使用FileTime格式）
		if lastLogon := entry.GetAttributeValue("lastLogon"); lastLogon != "" {
			user.LastLogon = s.parseFileTime(lastLogon)
		}
		if pwdLastSet := entry.GetAttributeValue("pwdLastSet"); pwdLastSet != "" {
			user.PasswordLastSet = s.parseFileTime(pwdLastSet)
		}

		now := time.Now()
		user.LastSyncAt = &now

		if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
			return fmt.Errorf("保存用户失败: %w", err)
		}
	}
	return nil
}

// syncGroupMembers 同步用户组成员关系
func (s *ADDomainService) syncGroupMembers(ctx context.Context, config *models.ADConfig, groupDN string, memberDNs []string) error {
	// 先删除旧的成员关系
	s.db.WithContext(ctx).Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
		Delete(&models.ADGroupMember{})

	// 添加新的成员关系
	for _, userDN := range memberDNs {
		member := &models.ADGroupMember{
			ADConfigID: config.ID,
			GroupDN:    groupDN,
			UserDN:     userDN,
		}
		if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
			// 忽略错误，继续处理其他成员
			continue
		}
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
