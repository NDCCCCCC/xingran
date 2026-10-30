package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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

// ⚠ Phase 58 D-03（HYGIENE-01）：本文件原有的「部门同步版默认配置选择函数」与本包
// ad_sync_tasks.go 的旧实现是逐字重复的两份拷贝，已删除并收敛到 resolveADConfigIDFromParams
// （参数提取层）+ addomain.FirstEnabledADConfigID（自动挑选，只判 status）。

// executeDeptToADSyncTask 执行部门到 AD 同步任务
// 将系统部门结构同步到 AD 域的 OU 结构
func executeDeptToADSyncTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 自动获取配置ID（支持参数传递或自动获取默认配置）
	adConfigID, err := resolveADConfigIDFromParams(ctx, db, params)
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
	// 域控审查 P2-3：ldapClient 参数已删除（SyncDeptStructureToAD 走 FailoverClient，不持有连接）
	pool := getGlobalADAccountPool()
	syncService := addomain.NewDeptToADSyncService(db, pool, mapper)

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
	adConfigID, err := resolveADConfigIDFromParams(ctx, db, params)
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
	// 所有 ldapClient.* 必须在闭包内（Pitfall 3）；DB 操作（查询/统计/更新时间）不依赖 LDAP 连接
	pool := getGlobalADAccountPool()
	fc := addomain.NewFailoverClient(pool, &adConfig)
	// 统计结果（域控审查 P2-1：totalProcessed 改按用户数统计，映射数不能反映工作量）
	totalProcessed := 0
	totalSuccess := 0
	totalFailed := 0
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient addomain.LDAPClientIface) error {
		// 域控审查 P2-6：last_sync_at / 同步日志统一走 OUGroupMappingService
		mappingService := addomain.NewOUGroupMappingService(db)
		// 对每个映射执行双向同步（域控审查 P0-1：先 diff AD 实际成员与影子表期望成员，多退少补）
		for _, mapping := range mappings {
			startTime := time.Now()

			applogger.Infof("开始同步映射: OU=%s → 组ID=%s", mapping.OUName, mapping.ADGroupID)

			// 查询属于该OU及其子OU的AD域控用户（后缀匹配）
			// 域控审查 P1-5：补 ad_config_id 过滤防多域串味 + LIKE 值转义 %/_
			var adUsers []models.ADUser
			if err := db.WithContext(ctx).
				Where("ad_config_id = ? AND ou_dn LIKE ? AND is_enabled = ?",
					adConfigID, "%"+escapeLikePattern(mapping.OUDN), true).
				Find(&adUsers).Error; err != nil {
				applogger.Errorf("查询OU %s 的AD用户失败: %v", mapping.OUName, err)
				totalFailed++
				continue
			}

			// 域控审查 P2-1：按用户数统计处理量
			totalProcessed += len(adUsers)

			// 期望成员集（影子表）
			expectedDNs := make([]string, 0, len(adUsers))
			expectedSet := make(map[string]bool, len(adUsers))
			for _, adUser := range adUsers {
				if adUser.UserDN == "" {
					applogger.Warnf("AD用户 %s 的DN为空，跳过", adUser.Username)
					continue
				}
				expectedDNs = append(expectedDNs, adUser.UserDN)
				expectedSet[adUser.UserDN] = true
			}

			if len(expectedDNs) == 0 {
				applogger.Infof("OU %s 没有启用的AD用户，跳过", mapping.OUName)
				continue
			}

			// 查询AD组信息
			var adGroup models.ADGroup
			if err := db.WithContext(ctx).Where("id = ?", mapping.ADGroupID).First(&adGroup).Error; err != nil {
				applogger.Errorf("查询AD组失败: %v", err)
				totalFailed++
				continue
			}

			// 记录组DN用于诊断
			applogger.Infof("目标AD组: 名称=%s, DN=%s", adGroup.GroupName, adGroup.GroupDN)

			// 读取 AD 组当前实际成员（域控审查 P0-1：diff 基准）
			actualMembers, err := ldapClient.GetGroupMembers(adGroup.GroupDN)
			if err != nil {
				applogger.Errorf("读取AD组 %s 当前成员失败: %v", adGroup.GroupName, err)
				addomain.WriteMemberSyncLog(ctx, mappingService, &mapping, startTime, 0, 0, len(expectedDNs), "failed", err.Error())
				totalFailed++
				continue
			}
			actualSet := make(map[string]bool, len(actualMembers))
			for _, dn := range actualMembers {
				actualSet[dn] = true
			}

			// 缺失成员 → 批量添加（批量失败降级逐个，域控审查 P1-6：错误码判断）
			missing := make([]string, 0)
			for _, dn := range expectedDNs {
				if !actualSet[dn] {
					missing = append(missing, dn)
				}
			}
			addedCount := 0
			skippedCount := 0
			if len(missing) > 0 {
				if err := ldapClient.AddGroupMembers(adGroup.GroupDN, missing); err != nil {
					applogger.Warnf("批量添加 %d 个成员失败，降级逐个添加: %v", len(missing), err)
					for _, dn := range missing {
						if err := ldapClient.AddGroupMember(adGroup.GroupDN, dn); err != nil {
							if ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
								skippedCount++
							} else {
								applogger.Errorf("添加AD成员 %s 到组 %s 失败: %v", dn, adGroup.GroupName, err)
							}
						} else {
							addedCount++
						}
					}
				} else {
					addedCount = len(missing)
				}
			}

			// 多余成员 → 逐个移除。
			// 保守约束（域控审查 P0-1）：只移除 DN 归一化后属于本 OU 的成员，
			// 防止误删手工加入的其他用户 / 其他映射管理的外部成员。
			normalizedOUDN := addomain.NormalizeDN(mapping.OUDN)
			removedCount := 0
			notFoundCount := 0
			otherFailedCount := 0
			for dn := range actualSet {
				if expectedSet[dn] {
					continue
				}
				if !strings.HasSuffix(addomain.NormalizeDN(dn), normalizedOUDN) {
					continue
				}
				if err := ldapClient.RemoveGroupMember(adGroup.GroupDN, dn); err != nil {
					if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
						// 成员已不存在 - 正常状态（域控审查 P1-6：错误码判断）
						notFoundCount++
					} else {
						applogger.Errorf("移除组 %s 的多余成员 %s 失败: %v", adGroup.GroupName, dn, err)
						otherFailedCount++
					}
				} else {
					removedCount++
				}
			}

			// 记录同步结果
			duration := time.Since(startTime)
			status := "success"
			errMsg := ""
			if otherFailedCount > 0 {
				status = "failed"
				errMsg = fmt.Sprintf("移除失败=%d", otherFailedCount)
				applogger.Errorf("映射同步失败: OU=%s, 添加=%d, 已存在=%d, 移除=%d, AD中不存在=%d, 其他错误=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, removedCount, notFoundCount, otherFailedCount, duration.Milliseconds())
			} else if notFoundCount > 0 {
				status = "partial"
				applogger.Warnf("映射同步部分完成: OU=%s, 添加=%d, 已存在=%d, 移除=%d, AD中不存在=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, removedCount, notFoundCount, duration.Milliseconds())
			} else {
				applogger.Infof("映射同步成功: OU=%s, 添加=%d, 已存在=%d, 移除=%d, 耗时=%dms",
					mapping.OUName, addedCount, skippedCount, removedCount, duration.Milliseconds())
			}

			// 域控审查 P0-1：每映射写同步日志（此前链路从未接线）+ P2-6：统一 UpdateSyncStatus
			addomain.WriteMemberSyncLog(ctx, mappingService, &mapping, startTime, addedCount, removedCount, len(expectedDNs), status, errMsg)

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

// escapeLikePattern 转义 LIKE 模式中的通配符（域控审查 P1-5）。
// OU DN 中出现 % _ \ 字符时防误匹配；仅用于 Postgres LIKE。
func escapeLikePattern(s string) string {
	return strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
