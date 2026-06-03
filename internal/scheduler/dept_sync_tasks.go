package scheduler

import (
	"context"
	"fmt"

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

// executeDeptMemberToADGroupSyncTask 执行部门成员到AD域组同步任务
// 将系统部门成员同步到AD域组（系统 → AD）
// 注意：此任务已废弃，部门-组映射功能已被OU-组映射替代
func executeDeptMemberToADGroupSyncTask(ctx context.Context, params map[string]interface{}) error {
	applogger.Warn("部门成员到AD域组同步任务已废弃：部门-组映射功能已被OU-组映射替代")
	applogger.Warn("请使用新的OU组映射功能进行成员同步管理")
	return fmt.Errorf("该任务已废弃，请使用OU组映射功能")
}
