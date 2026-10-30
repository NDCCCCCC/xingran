package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ADPartitionService AD登录历史分区服务接口(避免循环依赖,mac 同款防环手法)
// 定义为最小接口,由 internal/services/addomain/login_history_partition.go 实现
// (quick 261016-ad-login-history)
type ADPartitionService interface {
	CreateMonthlyPartition(ctx context.Context, year int, month int) error
	EnsurePartitionsExist(ctx context.Context, monthsAhead int) error
	DropExpiredPartitions(ctx context.Context) error
	GetRetentionDays(ctx context.Context) int
}

// 全局 AD 登录历史分区服务实例(依赖注入,sync.Once 确保线程安全)
var (
	globalADLoginHistoryPartitionService     ADPartitionService
	globalADLoginHistoryPartitionServiceOnce sync.Once
)

// SetADLoginHistoryPartitionService 设置全局 AD 登录历史分区服务(线程安全)
// 在 Core 初始化时调用,注入分区服务实例
func SetADLoginHistoryPartitionService(service ADPartitionService) {
	globalADLoginHistoryPartitionServiceOnce.Do(func() {
		globalADLoginHistoryPartitionService = service
		applogger.Infof("AD登录历史分区服务已注入")
	})
}

// RegisterADLoginHistoryTasks 注册 AD 登录历史分区维护定时任务
// 调度: 每天 02:30(7 段含秒;避开 02:00 的 mac_history_cleanup,PLAN 风险 10)
func RegisterADLoginHistoryTasks(scheduler *Scheduler, db *gorm.DB) {
	scheduler.RegisterTask("ad_login_history_cleanup", func(ctx context.Context, params map[string]interface{}) error {
		return executeADLoginHistoryCleanup(ctx, params)
	})

	applogger.Infof("AD登录历史分区维护任务处理器已注册")

	if db == nil {
		applogger.Warnf("AD登录历史分区维护任务数据库注册跳过（数据库未设置）")
		return
	}

	if err := upsertADLoginHistoryJob(db, scheduler, "AD登录历史分区维护", "ad_login_history_cleanup", "0 30 2 * * ?",
		"AD设备登录历史分区维护任务（quick 261016），每天 02:30 执行，确保未来2个月分区存在并清理过期月度分区（保留期默认120天，sys_config ops.ad.login_history.retention_days 可调，最小30天）", 2); err != nil {
		applogger.Warnf("注册 AD登录历史分区维护 任务失败: %v", err)
	}
}

// upsertADLoginHistoryJob 注册或更新 AD 登录历史分区维护任务到 sys_job 表
// (仿 upsertMACHistoryJob:存在即跳过,不存在则创建)
func upsertADLoginHistoryJob(db *gorm.DB, scheduler *Scheduler, jobName, invokeTarget, cronExpression, remark string, nextRunHour int) error {
	var existingJob models.Job
	err := db.Where("job_name = ?", jobName).First(&existingJob).Error
	if err == nil {
		applogger.Infof("任务已存在于数据库,跳过创建: %s", jobName)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	now := time.Now()
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day()+1, nextRunHour, 0, 0, 0, now.Location())

	newJob := &models.Job{
		JobName:        jobName,
		JobGroup:       "SYSTEM",
		InvokeTarget:   invokeTarget,
		CronExpression: cronExpression,
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         models.JobStatusNormal,
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}
	if err := scheduler.AddJob(newJob); err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	applogger.Infof("任务已添加到数据库(下次执行: %v): %s",
		nextRunTime.Format("2006-01-02 15:04:05"), jobName)
	return nil
}

// executeADLoginHistoryCleanup 执行 AD 登录历史分区维护任务:
// 1) 确保未来 2 个月的分区存在 2) 清理超过保留期的过期分区
func executeADLoginHistoryCleanup(ctx context.Context, params map[string]interface{}) error {
	if globalADLoginHistoryPartitionService == nil {
		return fmt.Errorf("AD登录历史分区服务未初始化,请检查 Core 启动日志中的 SetADLoginHistoryPartitionService 调用")
	}

	applogger.Infof("[AD登录历史] 开始分区维护")

	// 1. 确保未来 2 个月分区存在(防月末月初边界漏分区)
	if err := globalADLoginHistoryPartitionService.EnsurePartitionsExist(ctx, 2); err != nil {
		applogger.Errorf("[AD登录历史] 确保分区存在失败: %v", err)
		return fmt.Errorf("确保AD登录历史分区存在失败: %w", err)
	}

	// 2. 清理过期分区
	if err := globalADLoginHistoryPartitionService.DropExpiredPartitions(ctx); err != nil {
		applogger.Errorf("[AD登录历史] 清理过期分区失败: %v", err)
		return fmt.Errorf("清理AD登录历史过期分区失败: %w", err)
	}

	applogger.Infof("[AD登录历史] 分区维护完成")
	return nil
}
