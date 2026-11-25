// Package scheduler —— RPA 定时调度（sys_rpa_schedules）的调度层接线。
//
// 设计要点（Phase 158 / CLEANUP-02）：
//
//  1. 每个「启用的」RPA 调度在调度器里对应**一条** cron job，
//     invoke_target = "rpa_schedule:{scheduleID}"，cron 表达式取自
//     sys_rpa_schedules.cron_expr。到点后由本文件注册的 rpa_schedule handler
//     调用 services/rpa 的 ScheduleService.TriggerNow —— 与手动触发共用
//     TaskService.Execute 同一执行入口，执行记录的 trigger_type=schedule。
//
//  2. job 与 schedule 的对应关系靠 sys_job.remark 里的
//     "rpa_schedule_id={scheduleID}" 标记（幂等查找键）。
//     停用/删除 → RemoveScheduleJob（摘除 job，本调度即不再触发）。
//
//  3. cron 表达式位数：调度器引擎是 cron.WithSeconds()（6 位：秒 分 时 日 月 周），
//     而用户习惯写 5 位（分 时 日 月 周）。本层做位数归一：5 位 → 前置 "0 "
//     （即每分钟的第 0 秒触发），6 位原样保留。归一后的表达式写进 sys_job。
//
//  4. 幂等：EnsureScheduleJob 先按标记查重，命中即跳过（不重复 AddJob，
//     否则每次重启都会多挂一条 cron）。
//
//  5. 依赖方向：本包 → internal/services/rpa（单向）。services/rpa 不 import
//     本包，job 同步能力以 ScheduleJobSyncer 接口表达，由 internal/core/core.go 装配。
package scheduler

import (
	"context"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	rpav1 "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	// rpaScheduleInvokeTarget 调度器任务类型名（= RegisterTask 注册的 handler 名）。
	rpaScheduleInvokeTarget = "rpa_schedule"
	// rpaScheduleJobPrefix sys_job 展示名前缀（便于运维在「定时任务」页识别）。
	rpaScheduleJobPrefix = "RPA调度:"
	// rpaScheduleJobGroup sys_job 分组。
	rpaScheduleJobGroup = "RPA"
	// rpaScheduleIDMarker sys_job.remark 中的 schedule 标记前缀（幂等查找键）。
	rpaScheduleIDMarker = "rpa_schedule_id="
)

// cron 表达式解析器（6 位含秒 / 5 位不含秒），用于位数归一。
var (
	cronParser6 = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	cronParser5 = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

// NormalizeRPACronExpr 把用户写的 cron 表达式归一成调度器引擎可吃的 6 位形式。
//
// 调度器引擎是 cron.WithSeconds()（秒 分 时 日 月 周）；RPA 调度沿用运维习惯
// 写 5 位（分 时 日 月 周）。5 位表达式前置 "0 "（每分钟第 0 秒触发）。
func NormalizeRPACronExpr(expr string) (string, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return "", fmt.Errorf("cron 表达式不能为空")
	}
	if _, err := cronParser6.Parse(trimmed); err == nil {
		return trimmed, nil
	}
	if _, err := cronParser5.Parse(trimmed); err == nil {
		return "0 " + trimmed, nil
	}
	return "", fmt.Errorf("cron 表达式非法: %s", trimmed)
}

// rpaScheduleMarker 生成 job remark 里的 schedule 标记。
func rpaScheduleMarker(scheduleID string) string {
	return rpaScheduleIDMarker + scheduleID
}

// RPAScheduleJobSyncer 实现 rpav1.ScheduleJobSyncer —— 把 RPA 调度映射为调度器 job。
type RPAScheduleJobSyncer struct {
	scheduler *Scheduler
	db        *gorm.DB
}

// NewRPAScheduleJobSyncer 创建 RPA 调度 job 同步器。
func NewRPAScheduleJobSyncer(s *Scheduler, db *gorm.DB) *RPAScheduleJobSyncer {
	return &RPAScheduleJobSyncer{scheduler: s, db: db}
}

// EnsureScheduleJob 幂等地为该 schedule 确保一条调度 job。
func (y *RPAScheduleJobSyncer) EnsureScheduleJob(ctx context.Context, scheduleID, jobName, cronExpr string) error {
	if y == nil || y.scheduler == nil || y.db == nil {
		return fmt.Errorf("调度器未初始化")
	}
	if scheduleID == "" {
		return fmt.Errorf("scheduleID 不能为空")
	}

	normalized, err := NormalizeRPACronExpr(cronExpr)
	if err != nil {
		return err
	}

	existing, err := y.findJob(scheduleID)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.CronExpression == normalized {
			return nil // 已存在且 cron 一致 → 幂等跳过
		}
		// cron 变更：摘除旧 job（软删旧行）后重建（见 UpdateScheduleJobCron）
		return y.rebuildJob(existing, jobName, normalized)
	}

	return y.createJob(scheduleID, jobName, normalized)
}

// UpdateScheduleJobCron 更新该 schedule 对应 job 的 cron 表达式。
func (y *RPAScheduleJobSyncer) UpdateScheduleJobCron(ctx context.Context, scheduleID, cronExpr string) error {
	if y == nil || y.scheduler == nil || y.db == nil {
		return fmt.Errorf("调度器未初始化")
	}
	normalized, err := NormalizeRPACronExpr(cronExpr)
	if err != nil {
		return err
	}
	existing, err := y.findJob(scheduleID)
	if err != nil {
		return err
	}
	if existing == nil {
		// job 缺失（被人工删过）→ 补建
		return y.createJob(scheduleID, "", normalized)
	}
	if existing.CronExpression == normalized {
		return nil
	}
	return y.rebuildJob(existing, existing.JobName, normalized)
}

// RemoveScheduleJob 摘除该 schedule 对应 job（停用/删除时调用）。
func (y *RPAScheduleJobSyncer) RemoveScheduleJob(ctx context.Context, scheduleID string) error {
	if y == nil || y.scheduler == nil || y.db == nil {
		return fmt.Errorf("调度器未初始化")
	}
	existing, err := y.findJob(scheduleID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil // 已不存在 → 幂等
	}
	if err := y.scheduler.RemoveJob(existing.ID); err != nil {
		return fmt.Errorf("摘除调度任务失败: %w", err)
	}
	return nil
}

// findJob 按 remark 标记查找该 schedule 对应的 job（未找到返回 nil,nil）。
func (y *RPAScheduleJobSyncer) findJob(scheduleID string) (*models.Job, error) {
	var job models.Job
	err := y.db.Where("remark LIKE ?", "%"+rpaScheduleMarker(scheduleID)+"%").
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询调度任务失败: %w", err)
	}
	return &job, nil
}

// createJob 新建一条调度 job 并挂载到 cron 引擎。
func (y *RPAScheduleJobSyncer) createJob(scheduleID, jobName, normalizedCron string) error {
	name := rpaScheduleJobPrefix + jobName
	if jobName == "" {
		name = rpaScheduleJobPrefix + scheduleID
	}
	remark := fmt.Sprintf("%s%s", rpaScheduleMarker(scheduleID),
		"（RPA 定时调度，Phase 158 / CLEANUP-02 播种；停用/删除本调度会自动摘除）")

	job := &models.Job{
		JobName:        name,
		JobGroup:       rpaScheduleJobGroup,
		InvokeTarget:   rpaScheduleInvokeTarget + ":" + scheduleID,
		CronExpression: normalizedCron,
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Concurrent:     false,
		Status:         models.JobStatusNormal,
		Remark:         &remark,
	}
	if err := y.scheduler.AddJob(job); err != nil {
		return fmt.Errorf("创建调度任务失败: %w", err)
	}
	applogger.Infof("[rpa_schedule] 已注册调度任务: %s (schedule=%s, cron=%s)", job.JobName, scheduleID, normalizedCron)
	return nil
}

// rebuildJob 重建 job（cron 变更时）：摘除旧条目后按新表达式重建。
//
// 不用 Scheduler.UpdateJob —— 它的内部 removeJob 会软删 sys_job 行，
// 而随后的 Save 对软删行不生效（Update 带 deleted_at IS NULL 条件），
// 结果是「cron 内存里换了、DB 行却没了」→ 下次重启调度丢失。
// 这里改为「摘除 + 重建」，DB 留一行软删旧记录 + 一行新记录，语义清晰。
func (y *RPAScheduleJobSyncer) rebuildJob(existing *models.Job, jobName, normalizedCron string) error {
	if err := y.scheduler.RemoveJob(existing.ID); err != nil {
		return fmt.Errorf("摘除旧调度任务失败: %w", err)
	}
	scheduleID := strings.TrimPrefix(existing.InvokeTarget, rpaScheduleInvokeTarget+":")
	if scheduleID == existing.InvokeTarget || scheduleID == "" {
		return fmt.Errorf("调度任务的 invoke_target 格式非法: %s", existing.InvokeTarget)
	}
	return y.createJob(scheduleID, jobName, normalizedCron)
}

// RegisterRPAScheduleTasks 注册 RPA 定时调度的调度器接线。
//
// 由 lead 在 internal/core/core.go 的 initSchedulerAndTasks 里调用。
//
// 参数：
//   - s:   调度器实例（注册 task type + 挂 job）
//   - db:  GORM 句柄（job 幂等查找）
//   - svc: RPA 调度服务（触发执行 + 同步启用的调度为 job）
func RegisterRPAScheduleTasks(s *Scheduler, db *gorm.DB, svc rpav1.ScheduleService) {
	s.RegisterTask(rpaScheduleInvokeTarget, func(ctx context.Context, params map[string]interface{}) error {
		scheduleID, _ := params["param"].(string)
		if scheduleID == "" {
			return fmt.Errorf("rpa_schedule 处理器缺少 schedule_id 参数")
		}
		if svc == nil {
			return fmt.Errorf("RPA 调度服务未注入")
		}
		exec, err := svc.TriggerNow(ctx, scheduleID)
		if err != nil {
			applogger.Errorf("[rpa_schedule] 调度触发失败 (schedule=%s): %v", scheduleID, err)
			return err
		}
		applogger.Infof("[rpa_schedule] 调度已触发 (schedule=%s, execution=%s)", scheduleID, exec.ID)
		return nil
	})
	applogger.Infof("[rpa_schedule] 任务处理器已注册（InvokeTarget=%s）", rpaScheduleInvokeTarget)

	if svc == nil {
		applogger.Warnf("[rpa_schedule] 调度服务未注入，跳过调度同步")
		return
	}
	// 启动时把启用的调度同步为 job（幂等；停用的调度在此被摘除）。
	if err := svc.SyncEnabledJobs(context.Background()); err != nil {
		applogger.Warnf("[rpa_schedule] 调度同步失败（不阻断启动）: %v", err)
	}
}
