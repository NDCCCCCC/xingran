package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== 部门到 AD 同步任务注册函数 ====================

// RegisterDeptSyncTasks 注册部门到 AD 同步定时任务
func RegisterDeptSyncTasks(scheduler *Scheduler) {
	// 部门到 AD 同步任务 - 将系统部门结构同步到 AD OU
	scheduler.RegisterTask("dept_to_ad_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeDeptToADSyncTask(ctx, params)
	})

	// 部门成员到AD域组同步任务 - 将系统部门成员同步到AD域组
	scheduler.RegisterTask("dept_member_to_ad_group_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeDeptMemberToADGroupSyncTask(ctx, params)
	})
}

// getDefaultADConfigID 获取默认的 AD 配置 ID（部门同步版本）
// 优先从参数中获取，如果没有则查询第一个启用的配置
func getDefaultADConfigIDForDept(ctx context.Context, db *gorm.DB, params map[string]interface{}) (string, error) {
	// 1. 尝试从参数中获取配置ID（支持两种参数名）
	if configID, ok := params["configId"].(string); ok && configID != "" {
		return configID, nil
	}
	if configID, ok := params["adConfigId"].(string); ok && configID != "" {
		return configID, nil
	}

	// 2. 自动获取第一个启用的 AD 配置
	var adConfig models.ADConfig
	err := db.WithContext(ctx).
		Where("status = ?", models.ADConfigStatusEnabled).
		Order("created_at ASC").
		First(&adConfig).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到启用的AD配置，请先在AD域配置中启用至少一个配置")
		}
		return "", fmt.Errorf("查询AD配置失败: %w", err)
	}

	applogger.Infof("自动使用AD配置: %s (ID: %s)", adConfig.ConfigName, adConfig.ID)
	return adConfig.ID, nil
}

// executeDeptToADSyncTask 执行部门到 AD 同步任务
// 将系统部门结构同步到 AD 域的 OU 结构
func executeDeptToADSyncTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 自动获取配置ID（支持参数传递或自动获取默认配置）
	adConfigID, err := getDefaultADConfigIDForDept(ctx, db, params)
	if err != nil {
		return err
	}

	applogger.Infof("开始执行部门到 AD 同步任务，AD 配置 ID: %s", adConfigID)

	// 查询 AD 配置
	var adConfig models.ADConfig
	if err := db.WithContext(ctx).Where("id = ?", adConfigID).First(&adConfig).Error; err != nil {
		return fmt.Errorf("查询 AD 配置失败: %w", err)
	}

	// 检查 AD 配置是否启用
	if adConfig.Status != models.ADConfigStatusEnabled {
		return fmt.Errorf("AD 配置未启用: %s", adConfig.ConfigName)
	}

	// 创建 LDAP 客户端
	adConfig.AdminPassword = addomain.DecryptPassword(adConfig.AdminPassword)
	ldapClient := addomain.NewLDAPClient(&adConfig)
	if err := ldapClient.Connect(); err != nil {
		return fmt.Errorf("连接 AD 失败: %w", err)
	}
	defer ldapClient.Close()

	// 创建部门 OU 映射器
	mapper := addomain.NewDeptOUmapper(db)

	// 创建部门同步服务
	syncService := addomain.NewDeptToADSyncService(db, ldapClient, mapper)

	// 执行同步
	result, err := syncService.SyncDeptStructureToAD(ctx, adConfigID)
	if err != nil {
		applogger.Errorf("部门到 AD 同步失败: %v", err)
		return err
	}

	// 记录同步结果
	applogger.Infof("部门到 AD 同步完成: 总数=%d, 成功=%d, 失败=%d, 跳过=%d, 耗时=%dms",
		result.TotalDepts, result.SuccessDepts, result.FailedDepts, result.SkippedDepts, result.Duration.Milliseconds())

	return nil
}

// executeDeptMemberToADGroupSyncTask 执行OU成员到AD域组同步任务
// 将系统中属于指定OU的用户同步到对应的AD域组（系统 → AD）
func executeDeptMemberToADGroupSyncTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 自动获取配置ID（支持参数传递或自动获取默认配置）
	adConfigID, err := getDefaultADConfigIDForDept(ctx, db, params)
	if err != nil {
		return err
	}

	applogger.Infof("开始执行OU成员到AD域组同步任务，AD配置ID: %s", adConfigID)

	// 查询AD配置
	var adConfig models.ADConfig
	if err := db.WithContext(ctx).Where("id = ?", adConfigID).First(&adConfig).Error; err != nil {
		return fmt.Errorf("查询AD配置失败: %w", err)
	}

	// 检查AD配置是否启用
	if adConfig.Status != models.ADConfigStatusEnabled {
		return fmt.Errorf("AD配置未启用: %s", adConfig.ConfigName)
	}

	// 查询启用的OU-组映射
	var mappings []models.OUGroupMapping
	if err := db.WithContext(ctx).
		Where("ad_config_id = ? AND mapping_status = ? AND sync_enabled = ? AND deleted_at IS NULL",
			adConfigID, models.OUGroupMappingStatusActive, true).
		Find(&mappings).Error; err != nil {
		return fmt.Errorf("查询OU-组映射失败: %w", err)
	}

	if len(mappings) == 0 {
		applogger.Infof("没有找到活动的OU-组映射，跳过同步")
		return nil
	}

	applogger.Infof("找到 %d 个活动的OU-组映射，开始同步", len(mappings))

	// 解密密码
	adConfig.AdminPassword = addomain.DecryptPassword(adConfig.AdminPassword)

	// 创建LDAP客户端
	ldapClient := addomain.NewLDAPClient(&adConfig)
	if err := ldapClient.Connect(); err != nil {
		return fmt.Errorf("连接AD失败: %w", err)
	}
	defer ldapClient.Close()

	// 统计结果
	totalProcessed := 0
	totalSuccess := 0
	totalFailed := 0

	// 对每个映射执行同步
	for _, mapping := range mappings {
		totalProcessed++
		startTime := time.Now()

		applogger.Infof("开始同步映射: OU=%s → 组ID=%s", mapping.OUName, mapping.ADGroupID)

		// 查询属于该OU的用户
		var users []models.User
		if err := db.WithContext(ctx).
			Where("ad_ou_dn = ? AND status = ?", mapping.OUDN, models.UserStatusEnabled).
			Find(&users).Error; err != nil {
			applogger.Errorf("查询OU %s 的用户失败: %v", mapping.OUName, err)
			totalFailed++
			continue
		}

		if len(users) == 0 {
			applogger.Infof("OU %s 没有启用的用户，跳过", mapping.OUName)
			continue
		}

		applogger.Infof("OU %s 找到 %d 个用户，开始添加到AD组", mapping.OUName, len(users))

		// 查询AD组信息
		var adGroup models.ADGroup
		if err := db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
			applogger.Errorf("查询AD组失败: %v", err)
			totalFailed++
			continue
		}

		// 执行LDAP添加成员操作
		addedCount := 0
		failedCount := 0

		for _, user := range users {
			if user.AdDn == nil || *user.AdDn == "" {
				applogger.Warnf("用户 %s 的AD DN为空，跳过", user.Username)
				failedCount++
				continue
			}

			// 添加用户到AD组
			if err := ldapClient.AddGroupMember(adGroup.GroupDN, *user.AdDn); err != nil {
				applogger.Errorf("添加用户 %s 到组 %s 失败: %v", user.Username, adGroup.GroupName, err)
				failedCount++
			} else {
				addedCount++
			}
		}

		// 记录同步日志
		duration := time.Since(startTime)
		syncLog := &models.OUGroupMappingSyncLog{
			MappingID:      mapping.ID,
			OUdn:           mapping.OUDN,
			ADGroupID:      mapping.ADGroupID,
			SyncType:       "ou_member_sync",
			MembersAdded:   addedCount,
			MembersRemoved: 0,
			TotalMembers:   len(users),
			Status:         "success",
			StartedAt:      startTime,
			CompletedAt:    &startTime,
			DurationMs:     int(duration.Milliseconds()),
		}

		if failedCount > 0 {
			status := "partial"
			syncLog.Status = status
			errorMsg := fmt.Sprintf("%d/%d 用户添加失败", failedCount, len(users))
			syncLog.ErrorMsg = &errorMsg
		}

		if err := db.WithContext(ctx).Create(syncLog).Error; err != nil {
			applogger.Errorf("创建同步日志失败: %v", err)
		}

		// 更新映射的最后同步时间
		if err := db.WithContext(ctx).Model(&mapping).
			Update("last_sync_at", time.Now()).Error; err != nil {
			applogger.Errorf("更新映射同步时间失败: %v", err)
		}

		if failedCount == 0 {
			totalSuccess++
			applogger.Infof("映射同步成功: OU=%s, 添加用户=%d", mapping.OUName, addedCount)
		} else {
			totalFailed++
		}
	}

	applogger.Infof("OU成员到AD域组同步完成: 处理=%d, 成功=%d, 失败=%d",
		totalProcessed, totalSuccess, totalFailed)

	return nil
}
