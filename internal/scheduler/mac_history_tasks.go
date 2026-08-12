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

// PartitionService 分区服务接口（避免循环依赖）
// 定义为最小接口，由 internal/services/mac_history_partition 实现
type PartitionService interface {
	CreateMonthlyPartition(ctx context.Context, year int, month int) error
	EnsurePartitionsExist(ctx context.Context, monthsAhead int) error
	DropExpiredPartitions(ctx context.Context) error
	GetRetentionDays(ctx context.Context) int
}

// MACHistoryPurgeService MAC历史"无意义记录"清理服务接口(避免循环依赖)
// 由 internal/services/mac_history_service.PurgeMeaninglessRecords 实现。
// 注入后由 cron 任务 mac_history_purge_monthly 调用。
type MACHistoryPurgeService interface {
	PurgeMeaninglessRecords(ctx context.Context, dryRun bool) (deletedCount int64, backupTable string, err error)
}

// 全局分区服务实例（依赖注入）
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
var (
	globalPartitionService     PartitionService
	globalPartitionServiceOnce sync.Once

	globalPurgeService     MACHistoryPurgeService
	globalPurgeServiceOnce sync.Once
)

// SetPartitionService 设置全局分区服务（线程安全）
// 在 Core 初始化时调用，注入分区服务实例
func SetPartitionService(service PartitionService) {
	globalPartitionServiceOnce.Do(func() {
		globalPartitionService = service
		applogger.Infof("MAC历史分区服务已注入")
	})
}

// SetMACHistoryPurgeService 设置全局 purge 服务(线程安全)
// 在 Core 初始化时调用,注入 PurgeMeaninglessRecords 实现。
func SetMACHistoryPurgeService(service MACHistoryPurgeService) {
	globalPurgeServiceOnce.Do(func() {
		globalPurgeService = service
		applogger.Infof("MAC历史 purge 服务已注入")
	})
}

// RegisterMACHistoryTasks 注册MAC地址历史相关的定时任务
// 包括：
// - mac_history_cleanup: 每天凌晨2点清理过期分区
// - mac_history_purge_monthly: 每月1号凌晨3点清理"无意义记录"(2026-06-30 新增)
func RegisterMACHistoryTasks(scheduler *Scheduler, db *gorm.DB) {
	// 注册MAC历史清理任务处理器
	scheduler.RegisterTask("mac_history_cleanup", func(ctx context.Context, params map[string]interface{}) error {
		return executeMACHistoryCleanup(ctx, params)
	})

	// 注册MAC历史"无意义记录"清理任务处理器(2026-06-30 新增)
	scheduler.RegisterTask("mac_history_purge_monthly", func(ctx context.Context, params map[string]interface{}) error {
		return executeMACHistoryPurge(ctx, params)
	})

	applogger.Infof("MAC历史清理任务处理器已注册")

	// 向数据库添加任务记录
	if db == nil {
		applogger.Warnf("MAC历史清理任务数据库注册跳过（数据库未设置）")
		return
	}

	// 任务 1: 过期分区清理 (existing)
	if err := upsertMACHistoryJob(db, scheduler, "MAC历史数据清理", "mac_history_cleanup", "0 0 2 * * ?",
		"MAC地址历史数据清理任务，每天凌晨2点执行，清理过期的月度分区", 2); err != nil {
		applogger.Warnf("注册 MAC历史数据清理 任务失败: %v", err)
	}

	// 任务 2: 月度无意义记录清理 (2026-06-30 新增)
	// 调度:每月 1 号 03:00(避开 02:00 的分区清理 + collector 早高峰)
	// 设计依据:
	//   - L1/L2 去重(commit 467df3fc)已抑制稳定 MAC 90%+ 噪声
	//   - 但 disappeared 仍会写入(真离开场景),appeared 跨设备也会写
	//   - 月度清理防表再次堆积到百万级
	if err := upsertMACHistoryJob(db, scheduler, "MAC历史无意义记录清理", "mac_history_purge_monthly", "0 0 3 1 * ?",
		"MAC历史无意义记录清理任务(2026-06-30 新增),每月 1 号 03:00 执行,删除所有 disappeared + 冗余 appeared,保留 moved + vlan_changed + 每 MAC 首条 appeared。执行前自动备份 sys_device_mac_history_purge_backup_YYYYMMDD_HHMMSS", 3); err != nil {
		applogger.Warnf("注册 MAC历史无意义记录清理 任务失败: %v", err)
	}
}

// upsertMACHistoryJob 注册或更新 MAC 历史定时任务到 sys_job 表(2026-06-30 新增 helper)
//
// 参数:
//   - jobName:任务中文名(唯一)
//   - invokeTarget:对应 scheduler.RegisterTask 注册的 handler 名
//   - cronExpression:Cron 表达式(7 段,含秒)
//   - remark:任务说明
//   - nextRunHour:首次执行小时数(用于计算 next_run_time)
func upsertMACHistoryJob(db *gorm.DB, scheduler *Scheduler, jobName, invokeTarget, cronExpression, remark string, nextRunHour int) error {
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
		Status:         0,
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

// executeMACHistoryCleanup 执行MAC历史清理任务
// 清理过期的MAC历史表分区，释放存储空间
func executeMACHistoryCleanup(ctx context.Context, params map[string]interface{}) error {
	// 检查分区服务是否已初始化
	if globalPartitionService == nil {
		return fmt.Errorf("MAC历史分区服务未初始化")
	}

	applogger.Infof("[MAC历史清理] 开始清理过期分区")

	// 执行分区清理
	if err := globalPartitionService.DropExpiredPartitions(ctx); err != nil {
		applogger.Errorf("[MAC历史清理] 清理失败: %v", err)
		return fmt.Errorf("清理MAC历史分区失败: %w", err)
	}

	applogger.Infof("[MAC历史清理] 清理完成")
	return nil
}

// executeMACHistoryPurge 执行 MAC 历史"无意义记录"清理任务(2026-06-30 新增)
//
// 流程:
//   1. 校验全局 purge 服务是否已注入
//   2. 调用 PurgeMeaninglessRecords(ctx, dryRun=false)
//   3. 返回 (deletedCount, backupTable, error)
//
// 失败处理:
//   - 服务未注入:返回 error,任务标记失败
//   - DELETE 失败:服务内部已记录 backupTable,本函数返回 error 便于 scheduler 重试
func executeMACHistoryPurge(ctx context.Context, params map[string]interface{}) error {
	if globalPurgeService == nil {
		return fmt.Errorf("MAC历史 purge 服务未初始化,请检查 Core 启动日志中的 SetMACHistoryPurgeService 调用")
	}

	applogger.Infof("[MAC历史 purge] 开始执行月度无意义记录清理")
	deleted, backupTable, err := globalPurgeService.PurgeMeaninglessRecords(ctx, false)
	if err != nil {
		applogger.Errorf("[MAC历史 purge] 清理失败: %v (备份表: %s,如需回滚 INSERT INTO sys_device_mac_history SELECT * FROM %s)",
			err, backupTable, backupTable)
		return fmt.Errorf("MAC历史 purge 失败: %w", err)
	}

	applogger.Infof("[MAC历史 purge] 清理完成:删除 %d 行,备份表 %s", deleted, backupTable)
	return nil
}
