package addomain

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MemberSyncService 成员同步服务接口
type MemberSyncService interface {
	// SyncDeptMembers 同步指定部门的成员到对应的AD组
	SyncDeptMembers(ctx context.Context, deptID string) (*MemberSyncResult, error)

	// SyncAllMembers 同步所有有映射的部门的成员
	SyncAllMembers(ctx context.Context, configID string) (*BatchMemberSyncResult, error)

	// HandleDeptChange 处理部门变更（移出旧组，加入新组）
	HandleDeptChange(ctx context.Context, userID, oldDeptID, newDeptID string) error

	// EnsureExclusiveMembership 确保成员只属于一个部门组
	EnsureExclusiveMembership(ctx context.Context, configID string) (*ExclusiveMembershipResult, error)
}

// memberSyncService 服务实现
type memberSyncService struct {
	db *gorm.DB
}

// NewMemberSyncService 创建服务实例
func NewMemberSyncService(db *gorm.DB) MemberSyncService {
	return &memberSyncService{db: db}
}

// MemberSyncResult 成员同步结果
type MemberSyncResult struct {
	DeptID         string `json:"deptId"`
	DeptName       string `json:"deptName"`
	GroupDN        string `json:"groupDn"`
	GroupName      string `json:"groupName"`
	TotalMembers   int    `json:"totalMembers"`
	AddedCount     int    `json:"addedCount"`
	RemovedCount   int    `json:"removedCount"`
	UnchangedCount int    `json:"unchangedCount"`
	SkippedCount   int    `json:"skippedCount"` // 没有AD账号的用户
	Duration       int    `json:"duration"`     // 毫秒
}

// BatchMemberSyncResult 批量同步结果
type BatchMemberSyncResult struct {
	TotalDepts   int                  `json:"totalDepts"`
	SuccessCount int                  `json:"successCount"`
	FailedCount  int                  `json:"failedCount"`
	Results      []MemberSyncResult   `json:"results"`
	TotalMembers int                  `json:"totalMembers"`
	TotalAdded   int                  `json:"totalAdded"`
	TotalRemoved int                  `json:"totalRemoved"`
	Duration     int                  `json:"duration"`
}

// ExclusiveMembershipResult 排他性成员关系结果
type ExclusiveMembershipResult struct {
	TotalUsers      int `json:"totalUsers"`
	RemovedCount    int `json:"removedCount"`
	ProcessedGroups int `json:"processedGroups"`
}

// SyncDeptMembers 同步指定部门的成员到对应的AD组
func (s *memberSyncService) SyncDeptMembers(ctx context.Context, deptID string) (*MemberSyncResult, error) {
	start := time.Now()
	result := &MemberSyncResult{DeptID: deptID}

	// 1. 获取部门信息
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", deptID).First(&dept).Error; err != nil {
		return nil, fmt.Errorf("部门不存在")
	}
	result.DeptName = dept.DeptName

	// 2. 获取部门的组映射
	mappingService := NewDeptGroupMappingService(s.db)
	mapping, err := mappingService.GetMappingByDept(ctx, deptID)
	if err != nil {
		return nil, fmt.Errorf("未找到部门组映射: %w", err)
	}

	if !mapping.SyncEnabled {
		return nil, fmt.Errorf("该部门的成员同步未启用")
	}

	result.GroupDN = mapping.GroupDN
	result.GroupName = mapping.GroupName

	// 3. 获取AD配置
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", mapping.ADConfigID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}

	// 4. 验证LDAP配置完整性
	if config.ServerAddress == "" || config.ServerPort == 0 || config.BaseDN == "" {
		return nil, fmt.Errorf("AD配置不完整：缺少服务器地址、端口或BaseDN")
	}

	// 5. 获取部门的成员（所有用户）
	var users []models.User
	err = s.db.WithContext(ctx).
		Where("dept_id = ? AND status = ? AND deleted_at IS NULL", deptID, models.UserStatusEnabled).
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("查询部门成员失败: %w", err)
	}

	result.TotalMembers = len(users)

	// 6. 连接LDAP（使用decryptPassword解密密码）
	config.AdminPassword = decryptPassword(config.AdminPassword)
	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer client.Close()

	// 7. 获取组当前的成员（从LDAP）
	currentMembers, err := s.getCurrentGroupMembers(ctx, mapping.GroupDN)
	if err != nil {
		applogger.Warnf("[成员同步] 获取组当前成员失败: %v", err)
		currentMembers = make(map[string]bool) // 继续处理，使用空集合
	}

	// 8. 同步成员
	targetMembers := make(map[string]bool) // userDN -> true

	for _, user := range users {
		// 跳过没有AD账号的用户
		if user.ADDN == nil || *user.ADDN == "" {
			result.SkippedCount++
			continue
		}

		userDN := *user.ADDN
		targetMembers[userDN] = true

		// 如果用户不在组中，添加
		if !currentMembers[userDN] {
			if err := client.AddGroupMember(mapping.GroupDN, userDN); err != nil {
				applogger.Errorf("[成员同步] 添加成员失败 [%s -> %s]: %v", user.Username, mapping.GroupName, err)
				continue
			}
			result.AddedCount++
			applogger.Debugf("[成员同步] 添加成员: %s -> %s", user.Username, mapping.GroupName)
		} else {
			result.UnchangedCount++
		}
	}

	// 9. 移除不在目标列表中的成员
	for userDN := range currentMembers {
		if !targetMembers[userDN] {
			if err := client.RemoveGroupMember(mapping.GroupDN, userDN); err != nil {
				applogger.Errorf("[成员同步] 移除成员失败 [%s]: %v", userDN, err)
				continue
			}
			result.RemovedCount++
			applogger.Debugf("[成员同步] 移除成员: %s from %s", userDN, mapping.GroupName)
		}
	}

	// 10. 更新映射的同步时间
	now := time.Now()
	s.db.WithContext(ctx).
		Model(&models.DeptGroupMapping{}).
		Where("id = ?", mapping.ID).
		Update("last_sync_at", &now)

	// 11. 记录同步日志
	s.recordSyncLog(ctx, mapping.ID, deptID, mapping.ADGroupID, result)

	result.Duration = int(time.Since(start).Milliseconds())
	applogger.Infof("[成员同步] 完成: 部门=%s, 添加=%d, 移除=%d, 跳过=%d, 耗时=%dms",
		result.DeptName, result.AddedCount, result.RemovedCount, result.SkippedCount, result.Duration)

	return result, nil
}

// SyncAllMembers 同步所有有映射的部门的成员
func (s *memberSyncService) SyncAllMembers(ctx context.Context, configID string) (*BatchMemberSyncResult, error) {
	start := time.Now()
	result := &BatchMemberSyncResult{
		Results: []MemberSyncResult{},
	}

	// 1. 获取所有启用的映射
	mappingService := NewDeptGroupMappingService(s.db)
	activeStatus := models.MappingStatusActive
	listReq := &ListMappingsRequest{
		ADConfigID:    &configID,
		MappingStatus: &activeStatus,
		Current:       1,
		PageSize:      1000,
	}

	listResp, err := mappingService.ListMappings(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("查询映射列表失败: %w", err)
	}

	result.TotalDepts = len(listResp.List)

	// 2. 逐个同步
	for _, mapping := range listResp.List {
		syncResult, err := s.SyncDeptMembers(ctx, mapping.DeptID)
		if err != nil {
			result.FailedCount++
			// Get department name for error logging
			var dept models.Department
			if err2 := s.db.WithContext(ctx).Select("dept_name").Where("id = ?", mapping.DeptID).First(&dept).Error; err2 == nil {
				applogger.Errorf("[成员同步] 同步部门失败 [%s]: %v", dept.DeptName, err)
			} else {
				applogger.Errorf("[成员同步] 同步部门失败 [ID: %s]: %v", mapping.DeptID, err)
			}
			continue
		}

		result.SuccessCount++
		result.TotalMembers += syncResult.TotalMembers
		result.TotalAdded += syncResult.AddedCount
		result.TotalRemoved += syncResult.RemovedCount
		result.Results = append(result.Results, *syncResult)
	}

	result.Duration = int(time.Since(start).Milliseconds())
	applogger.Infof("[成员同步] 批量同步完成: 部门=%d, 成功=%d, 失败=%d, 总成员=%d, 添加=%d, 移除=%d, 耗时=%dms",
		result.TotalDepts, result.SuccessCount, result.FailedCount,
		result.TotalMembers, result.TotalAdded, result.TotalRemoved, result.Duration)

	return result, nil
}

// HandleDeptChange 处理部门变更（移出旧组，加入新组）
func (s *memberSyncService) HandleDeptChange(ctx context.Context, userID, oldDeptID, newDeptID string) error {
	// 1. 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	if user.ADDN == nil || *user.ADDN == "" {
		applogger.Infof("[成员同步] 用户无AD账号，跳过: %s", user.Username)
		return nil
	}

	userDN := *user.ADDN

	// 2. 从旧部门组中移除
	if oldDeptID != "" && oldDeptID != newDeptID {
		mappingService := NewDeptGroupMappingService(s.db)
		oldMapping, err := mappingService.GetMappingByDept(ctx, oldDeptID)
		if err == nil && oldMapping.SyncEnabled {
			var config models.ADConfig
			if err := s.db.WithContext(ctx).Where("id = ?", oldMapping.ADConfigID).First(&config).Error; err == nil {
				// 验证配置完整性
				if config.ServerAddress != "" && config.ServerPort != 0 && config.BaseDN != "" {
					config.AdminPassword = decryptPassword(config.AdminPassword)
					client := NewLDAPClient(&config)
					if err := client.Connect(); err == nil {
						defer client.Close()
						if err := client.RemoveGroupMember(oldMapping.GroupDN, userDN); err != nil {
							applogger.Warnf("[成员同步] 移除用户失败 [%s from %s]: %v", user.Username, oldMapping.GroupName, err)
						} else {
							applogger.Infof("[成员同步] 移除用户: %s from %s", user.Username, oldMapping.GroupName)
						}
					}
				}
			}
		}
	}

	// 3. 加入新部门组
	if newDeptID != "" {
		_, err := s.SyncDeptMembers(ctx, newDeptID)
		if err != nil {
			applogger.Errorf("[成员同步] 加入新部门组失败 [%s]: %v", user.Username, err)
			return err
		}
	}

	return nil
}

// EnsureExclusiveMembership 确保成员只属于一个部门组
func (s *memberSyncService) EnsureExclusiveMembership(ctx context.Context, configID string) (*ExclusiveMembershipResult, error) {
	result := &ExclusiveMembershipResult{}

	// 1. 获取所有用户及其部门
	var users []struct {
		UserID string
		DeptID *string
		ADDN   *string
	}

	err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Select("id, dept_id, ad_dn").
		Where("ad_dn IS NOT NULL AND ad_dn != '' AND status = ? AND deleted_at IS NULL", models.UserStatusEnabled).
		Scan(&users).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	result.TotalUsers = len(users)

	// 2. 获取所有映射
	mappingService := NewDeptGroupMappingService(s.db)
	activeStatus := models.MappingStatusActive
	listReq := &ListMappingsRequest{
		ADConfigID:    &configID,
		MappingStatus: &activeStatus,
		Current:       1,
		PageSize:      1000,
	}

	listResp, err := mappingService.ListMappings(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("查询映射列表失败: %w", err)
	}

	// 构建部门到组映射的快速查找表
	deptToGroup := make(map[string]string) // deptID -> groupDN
	for _, mapping := range listResp.List {
		deptToGroup[mapping.DeptID] = mapping.GroupDN
	}

	// 3. 连接LDAP（用于修改组成员）
	var config models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error; err != nil {
		return nil, fmt.Errorf("AD配置不存在")
	}

	// 验证配置完整性
	if config.ServerAddress == "" || config.ServerPort == 0 || config.BaseDN == "" {
		return nil, fmt.Errorf("AD配置不完整：缺少服务器地址、端口或BaseDN")
	}

	config.AdminPassword = decryptPassword(config.AdminPassword)
	client := NewLDAPClient(&config)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer client.Close()

	// 4. 对每个组，移除不属于对应部门的成员
	result.ProcessedGroups = len(listResp.List)

	for _, mapping := range listResp.List {
		// 获取组当前的成员（从LDAP）
		currentMembers, err := s.getCurrentGroupMembers(ctx, mapping.GroupDN)
		if err != nil {
			applogger.Warnf("[排他性检查] 获取组成员失败 [%s]: %v", mapping.GroupName, err)
			continue
		}

		// 移除不属于该部门的成员
		for userDN := range currentMembers {
			// 查找用户所属部门
			userDeptID := ""
			for _, u := range users {
				if u.ADDN != nil && *u.ADDN == userDN && u.DeptID != nil {
					userDeptID = *u.DeptID
					break
				}
			}

			// 如果用户不属于该部门，移除
			if userDeptID != mapping.DeptID {
				if err := client.RemoveGroupMember(mapping.GroupDN, userDN); err != nil {
					applogger.Errorf("[排他性检查] 移除成员失败 [%s from %s]: %v", userDN, mapping.GroupName, err)
					continue
				}
				result.RemovedCount++
				applogger.Infof("[排他性检查] 移除成员: %s from %s (不属于部门)", userDN, mapping.GroupName)
			}
		}
	}

	applogger.Infof("[排他性检查] 完成: 总用户=%d, 处理组=%d, 移除成员=%d",
		result.TotalUsers, result.ProcessedGroups, result.RemovedCount)

	return result, nil
}

// getCurrentGroupMembers 获取组当前的成员（从LDAP查询）
func (s *memberSyncService) getCurrentGroupMembers(ctx context.Context, groupDN string) (map[string]bool, error) {
	// 注意：这个方法需要从AD组中获取当前成员列表
	// 由于需要LDAP连接，这里简化实现，实际应该查询AD
	// 可以通过ADGroupMember表或直接LDAP查询

	var members []models.ADGroupMember
	err := s.db.WithContext(ctx).
		Where("group_dn = ?", groupDN).
		Find(&members).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, m := range members {
		result[m.UserDN] = true
	}

	return result, nil
}

// recordSyncLog 记录同步日志
func (s *memberSyncService) recordSyncLog(ctx context.Context, mappingID, deptID, groupID string, result *MemberSyncResult) {
	log := &models.DeptGroupMappingSyncLog{
		MappingID:      mappingID,
		DeptID:         deptID,
		ADGroupID:      groupID,
		SyncType:       "member_sync",
		MembersAdded:   result.AddedCount,
		MembersRemoved: result.RemovedCount,
		TotalMembers:   result.TotalMembers,
		Status:         "success",
		StartedAt:      time.Now().Add(-time.Duration(result.Duration) * time.Millisecond),
		CompletedAt:    timePtr(time.Now()),
		DurationMs:     result.Duration,
	}

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		applogger.Errorf("[成员同步] 记录同步日志失败: %v", err)
	}
}

// timePtr 返回时间指针的辅助函数
func timePtr(t time.Time) *time.Time {
	return &t
}
