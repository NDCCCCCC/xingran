// Package scheduler —— 全库备份定时任务的调度层接线。
//
// 设计要点（并行开发 D 层，只负责调度触发，不实现任何备份业务逻辑）：
//
//  1. 固定 tick + 每 tick 读配置门控（参考 ad_sync_tasks.go:193-220 的 Phase 58 方案 A 热重载范例）：
//     本文件只向调度器注册一个高频固定 cron（CronExpression="0 * * * * *"，每分钟整点
//     触发一次 RunScheduled），真正的「开不开、按什么周期跑」全部由 dbbackup.Service 内部
//     读取 sys_config 键（sys.database.backup.enabled / sys.database.backup.cron 等）并按
//     上次执行时间门控判定。tick 只是最小调度粒度：管理员配置「每小时 / 每 30 分钟 /
//     每 5 分钟」等更细周期时，每分钟的 tick 都会去问一次门控，到点才真正执行；
//     若用一天仅一次的低频 tick，细周期配置根本等不到触发，门控形同虚设。
//     代价是周期精度存在 ≤1 分钟抖动（放弃整周期对齐，与 Phase 58 方案 A 取舍一致）。
//     这样改备份计划无需重启应用，调度层不感知业务配置。
//
//  2. InvokeTarget 绝不能叫 "config_backup"：该名字已被网络设备 running-config 备份
//     占用（见 cron.go 的 registerNetworkTasks），这里固定用 "db_backup"。
//
//  3. 定时任务页「只读可见」：仅向 sys_job 写一行记录，不新增只读标记字段（sys_job
//     表结构无此字段），只读性由前端不暴露写按钮保证；本文件不新增 job 类型、不改表结构。
//
//  4. 跨实例互斥：现有 scheduler 引擎无跨实例互斥能力（ad_sync_tasks.go:195-196 实锤
//     多实例各自独立跑），因此不在本层实现分布式锁；多实例下的重复触发收敛由
//     dbbackup.Service 层的任务表（sys_db_backup_task）承担。
package scheduler

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	dbbackup "github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	// dbBackupJobName 定时任务页展示名，同时作为幂等查询键之一。
	dbBackupJobName = "数据库全量备份"
	// dbBackupInvokeTarget 对应 RegisterTask 注册的 handler 名；不可使用 config_backup。
	dbBackupInvokeTarget = "db_backup"
	// dbBackupCronExpression 6 字段含秒（robfig/cron/v3 以 WithSeconds 解析），高频固定 tick 语义：
	// 每分钟整点触发一次 svc.RunScheduled；实际执行周期由 sys.database.backup.cron 决定、
	// 开关由 sys.database.backup.enabled 决定，二者均在 Service 内部按 sys_config 与上次执行
	// 时间门控。tick 是最小调度粒度，故周期精度 ≤1 分钟抖动；改备份计划无需重启。
	dbBackupCronExpression = "0 * * * * *"
)

// RegisterDBBackupTasks 注册全库备份内置 cron 任务。
// 由 lead 在 internal/core/core.go 的 initSchedulerAndTasks 里调用。
//
// 参数：
//   - s:   调度器实例，用于 RegisterTask 与 AddJob。
//   - db:  GORM 句柄，用于 sys_job 幂等查询（AddJob 内部会再次 db.Create，必须先查重）。
//   - svc: 全库备份服务，依赖其 RunScheduled(ctx) / RecoverStaleTasks(ctx)，签名以
//     internal/services/dbbackup/contract.go 的 Service 接口为准。
func RegisterDBBackupTasks(s *Scheduler, db *gorm.DB, svc dbbackup.Service) {
	// 注册 handler：固定 tick 触发，周期/开关判定完全下沉到 Service 内部按 sys_config 门控。
	// defer recover() 防止单次 panic 击穿调度器 goroutine；错误统一用 applogger 记录后返回，
	// 由 JobExecutor 写入任务日志。
	s.RegisterTask(dbBackupInvokeTarget, func(ctx context.Context, _ map[string]interface{}) (err error) {
		defer func() {
			if r := recover(); r != nil {
				applogger.Errorf("[db_backup] 任务 panic 已恢复: %v", r)
				err = fmt.Errorf("db_backup 任务 panic: %v", r)
			}
		}()

		if svc == nil {
			applogger.Errorf("[db_backup] 服务未注入，跳过本轮定期备份")
			return fmt.Errorf("dbbackup.Service 未注入")
		}
		if runErr := svc.RunScheduled(ctx); runErr != nil {
			applogger.Errorf("[db_backup] 定期备份轮次失败: %v", runErr)
			return runErr
		}
		return nil
	})
	applogger.Infof("[db_backup] 任务处理器已注册（InvokeTarget=%s，固定 tick=%s）",
		dbBackupInvokeTarget, dbBackupCronExpression)

	// 出生时收敛残留 running/pending 任务；失败只 WARN，不阻断启动与其它定时任务注册。
	if svc != nil {
		if err := svc.RecoverStaleTasks(context.Background()); err != nil {
			applogger.Warnf("[db_backup] 残留任务收敛失败（不阻断启动）: %v", err)
		}
	} else {
		applogger.Warnf("[db_backup] 服务未注入，跳过残留任务收敛")
	}

	// 写一行 sys_job，令定时任务页只读可见。
	if db == nil {
		applogger.Warnf("[db_backup] 跳过 sys_job 注册（db 未设置）")
		return
	}
	if err := upsertDBBackupJob(s, db); err != nil {
		applogger.Warnf("[db_backup] 注册 sys_job 失败: %v", err)
	}
}

// upsertDBBackupJob 幂等地把全库备份任务写入 sys_job。
//
// 幂等关键：AddJob 内部会执行 db.Create(job)（见 cron.go:280-298），重复启动会插入
// 重复行并重复挂载 cron。因此先按 job_name 或 invoke_target 查重，命中即跳过 AddJob。
func upsertDBBackupJob(s *Scheduler, db *gorm.DB) error {
	var existing models.Job
	err := db.Where("job_name = ? OR invoke_target = ?", dbBackupJobName, dbBackupInvokeTarget).
		First(&existing).Error
	if err == nil {
		applogger.Infof("[db_backup] 任务已存在，跳过创建（不重复 AddJob）: %s", dbBackupJobName)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	// 下一次整分钟的估算值；调度器 addJob 会用 calculateNextRunTime 覆盖为真实下次执行时间。
	now := time.Now()
	nextRunTime := now.Truncate(time.Minute).Add(time.Minute)

	remark := "全库备份高频固定 tick：每分钟整点触发 RunScheduled，Service 内部按上次执行时间与 " +
		"sys.database.backup.cron 门控到点才真正执行（tick 为最小调度粒度，周期精度 ≤1 分钟抖动）；" +
		"实际执行周期由 sys.database.backup.cron 决定、开关由 sys.database.backup.enabled 决定，" +
		"改备份计划无需重启。定时任务页只读可见，不暴露写按钮。跨实例互斥由 Service 层任务表承担。"

	newJob := &models.Job{
		JobName:        dbBackupJobName,
		JobGroup:       "SYSTEM",
		InvokeTarget:   dbBackupInvokeTarget,
		CronExpression: dbBackupCronExpression,
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Concurrent:     false,
		Status:         models.JobStatusNormal,
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}
	if err := s.AddJob(newJob); err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	applogger.Infof("[db_backup] 任务已写入 sys_job（下次估算执行: %v）: %s",
		nextRunTime.Format("2006-01-02 15:04:05"), dbBackupJobName)
	return nil
}
