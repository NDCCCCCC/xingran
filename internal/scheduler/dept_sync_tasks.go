package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// getGlobalADAccountPool 返回 globalADSyncScheduler 持有的全局账号池单例（W-04 / Pitfall 4）。
// dept_sync_tasks 作为外部包，必须复用全局单例，**禁止**在本文件内临时创建账号池
// （acceptance_criteria: 本文件不得直接创建账号池实例）。
// 若 globalADSyncScheduler 未初始化（调度器启动前被调用的极端场景），返回 nil，
// caller 的 FailoverClient 会因 ListAvailable 调用而 panic —— 这是预期的前置 init 错误，
// 应通过先调用 StartADSyncScheduler 修复，而非在此临时创建。
func getGlobalADAccountPool() addomain.AccountPool {
	if globalADSyncScheduler == nil {
		applogger.Errorf("[dept_sync] globalADSyncScheduler 未初始化，账号池不可用；请先调用 StartADSyncScheduler")
		return nil
	}
	return globalADSyncScheduler.getPool()
}

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

	// Phase 38 Wave 1 (W-04): 不再创建单管理员 ldapClient，改由 SyncDeptStructureToAD 内部走 FailoverClient 账号池
	// decryptPassword 行暂保留（38-03 删）

	// 创建部门 OU 映射器
	mapper := addomain.NewDeptOUmapper(db)

	// 创建部门同步服务
	// Phase 38 Wave 1 (W-04): 复用 globalADSyncScheduler.pool 全局单例（Pitfall 4：避免临时创建独立缓存的账号池）
	// ldapClient 参数传 nil（SyncDeptStructureToAD 已改用 FailoverClient，不再使用注入的 ldapClient）
	pool := getGlobalADAccountPool()
	syncService := addomain.NewDeptToADSyncService(db, pool, nil, mapper)

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

	// 检查 AD 配置是否启用
	if adConfig.Status != models.ADConfigStatusEnabled {
		return fmt.Errorf("AD配置未启用: %s", adConfig.ConfigName)
	}

	// 查询启用的OU-组映射
	var mappings []models.OUGroupMapping
	if err := db.WithContext(ctx).
		Where("ad_config_id = ? AND mapping_status = ? AND sync_enabled = ?",
			adConfigID, models.OUGroupMappingStatusActive, true).
		Find(&mappings).Error; err != nil {
		return fmt.Errorf("查询OU-组映射失败: %w", err)
	}

	if len(mappings) == 0 {
		applogger.Infof("没有找到活动的OU-组映射，跳过同步")
		return nil
	}

	applogger.Infof("找到 %d 个活动的OU-组映射，开始同步", len(mappings))

	// 解密密码（Phase 38 Wave 1: decryptPassword 暂保留，38-03 统一删）

	// Phase 38 Wave 1 (W-04): 改走 FailoverClient 账号池故障切换
	// operation 边界 = 整个批量同步流程（所有 mappings × 所有 users 一个 operation，SP-3）
	// 所有 ldapClient.AddGroupMember 必须在闭包内（Pitfall 3）；DB 操作（查询/统计/更新时间）不依赖 LDAP 连接
	pool := getGlobalADAccountPool()
	fc := addomain.NewFailoverClient(pool, &adConfig)
	// 统计结果
	totalProcessed := 0
	totalSuccess := 0
	totalFailed := 0
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient *addomain.LDAPClient) error {
		// 对每个映射执行同步
		for _, mapping := range mappings {
			totalProcessed++
			startTime := time.Now()

			applogger.Infof("开始同步映射: OU=%s → 组ID=%s", mapping.OUName, mapping.ADGroupID)

			// 查询属于该OU及其子OU的AD域控用户（后缀匹配）
			var adUsers []models.ADUser
			if err := db.WithContext(ctx).
				Where("ou_dn LIKE ? AND is_enabled = ?", "%"+mapping.OUDN, true).
				Find(&adUsers).Error; err != nil {
				applogger.Errorf("查询OU %s 的AD用户失败: %v", mapping.OUName, err)
				totalFailed++
				continue
			}

			if len(adUsers) == 0 {
				applogger.Infof("OU %s 没有启用的AD用户，跳过", mapping.OUName)
				continue
			}

			applogger.Infof("OU %s 找到 %d 个AD用户，开始添加到AD组", mapping.OUName, len(adUsers))

			// 查询AD组信息
			var adGroup models.ADGroup
			if err := db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
				applogger.Errorf("查询AD组失败: %v", err)
				totalFailed++
				continue
			}

			// 记录组DN用于诊断
			applogger.Infof("目标AD组: 名称=%s, DN=%s", adGroup.GroupName, adGroup.GroupDN)

			// 执行LDAP添加成员操作
			addedCount := 0      // 成功添加
			skippedCount := 0    // 已存在（正常）
			notFoundCount := 0   // 用户在AD中不存在
			otherFailedCount := 0 // 其他错误

			for _, adUser := range adUsers {
				if adUser.UserDN == "" {
					applogger.Warnf("AD用户 %s 的DN为空，跳过", adUser.Username)
					otherFailedCount++
					continue
				}

				// 添加AD用户到AD组
				if err := ldapClient.AddGroupMember(adGroup.GroupDN, adUser.UserDN); err != nil {
					errMsg := err.Error()
					// 检查错误类型
					if strings.Contains(errMsg, "Entry Already Exists") || strings.Contains(errMsg, "68") {
						// 用户已存在 - 正常状态
						applogger.Infof("AD用户 %s 已在组中（跳过）", adUser.Username)
						skippedCount++
					} else if strings.Contains(errMsg, "No Such Object") || strings.Contains(errMsg, "32") {
						// 用户在AD中不存在
						applogger.Warnf("AD用户 %s 在AD中不存在: %v", adUser.Username, err)
						notFoundCount++
					} else {
						// 其他错误
						applogger.Errorf("添加AD用户 %s 到组 %s 失败: %v",
							adUser.Username, adGroup.GroupName, err)
						otherFailedCount++
					}
				} else {
					addedCount++
				}
			}

			// 记录同步结果
			duration := time.Since(startTime)
			if otherFailedCount == 0 && notFoundCount == 0 {
				applogger.Infof("映射同步成功: OU=%s, 添加=%d, 已存在=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, duration.Milliseconds())
			} else if otherFailedCount == 0 {
				applogger.Warnf("映射同步部分完成: OU=%s, 添加=%d, 已存在=%d, AD中不存在=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, notFoundCount, duration.Milliseconds())
			} else {
				applogger.Errorf("映射同步失败: OU=%s, 添加=%d, 已存在=%d, AD中不存在=%d, 其他错误=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, notFoundCount, otherFailedCount, duration.Milliseconds())
			}

			// 更新映射的最后同步时间
			if err := db.WithContext(ctx).Model(&mapping).
				Update("last_sync_at", time.Now()).Error; err != nil {
				applogger.Errorf("更新映射同步时间失败: %v", err)
			}

			// 统计结果（只有真正的失败才算失败）
			if otherFailedCount == 0 {
				totalSuccess++
			} else {
				totalFailed++
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, addomain.ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return fmt.Errorf("连接AD失败: %w", err)
	}

	applogger.Infof("OU成员到AD域组同步完成: 处理=%d, 成功=%d, 失败=%d",
		totalProcessed, totalSuccess, totalFailed)

	return nil
}
