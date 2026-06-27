package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// RegisterReconciliationTasks 注册 Phase 42 R1 资产对账相关定时任务处理器。
//
// 关键修复: Phase 42-r1 plan 02 创建了 4 个 sys_job 记录(对账-物化视图刷新 / Layer3 检测 /
// 静默期重检测 / 例外规则清理)并实现了 Execute*Task 全局函数,但漏了把 handler 注册到
// Scheduler.RegisterTask(任务类型 → 实际函数映射)。结果: 调度器启动时找不到
// 'reconciliation:refreshView' 等 taskType 对应的 handler,即使 cron 表达式正确也
// 报 "未找到任务处理器: reconciliation"。
//
// 本文件补全 4 个 RegisterTask + sys_job 记录(若已存在则跳过)。
//
// R1 行为(R1 boundary):
//   - reconciliation:refreshView          → 真实执行 MV CONCURRENTLY refresh
//   - reconciliation:detectLayer3         → 真实执行 Layer 3 检测写入 sys_data_reconciliation
//   - reconciliation:detectExpiredSilence → R1 placeholder(R2 真实实现:修复回写后 7d 静默期)
//   - reconciliation:cleanupExpiredExceptions → R1 placeholder(R3 真实实现:临时例外到期清理)
func RegisterReconciliationTasks(s *Scheduler, db *gorm.DB) {
	// 实例化对账服务(无 core 依赖,只依赖 *gorm.DB)
	snapSvc := asset.NewReconciliationSnapshotService(db)
	detSvc := asset.NewReconciliationDetection(db)

	// === 1. 物化视图刷新 ===
	//
	// 关键: InvokeTarget 是 "reconciliation:refreshView",但 cron.JobExecutor
	// 用冒号前段作为 taskType 查表(parseInvokeTarget: "如果包含冒号则分割")。
	// 所以所有对账 cron 必须用相同的 taskType="reconciliation",在 handler 内部
	// 通过 params["command"] 或解析 InvokeTarget 区分具体子任务。
	//
	// 这里用 params["target"] 传子任务名,由单一路由函数 dispatch。
	s.RegisterTask("reconciliation", func(ctx context.Context, params map[string]interface{}) error {
		// cron.JobExecutor.parseInvokeTarget 把 InvokeTarget "reconciliation:refreshView"
		// 解析成 taskType="reconciliation" + params={"param":"refreshView"}
		target, _ := params["param"].(string)
		applogger.Infof("[reconciliation:%s] 开始执行", target)
		switch target {
		case "refreshView":
			if err := snapSvc.RefreshView(ctx); err != nil {
				applogger.Errorf("[reconciliation:refreshView] 失败: %v", err)
				return err
			}
		case "detectLayer3":
			created, skipped, err := detSvc.DetectLayer3(ctx)
			if err != nil {
				applogger.Errorf("[reconciliation:detectLayer3] 失败: %v", err)
				return err
			}
			applogger.Infof("[reconciliation:detectLayer3] 完成: created=%d skipped=%d", created, skipped)
		case "detectExpiredSilence":
			applogger.Infof("[reconciliation:detectExpiredSilence] R1 placeholder, R2 真实实现")
		case "cleanupExpiredExceptions":
			applogger.Infof("[reconciliation:cleanupExpiredExceptions] R1 placeholder, R3 真实实现")
		default:
			return fmt.Errorf("未知子任务: %s", target)
		}
		applogger.Infof("[reconciliation:%s] 完成", target)
		return nil
	})
	applogger.Infof("对账任务处理器已注册(单一 taskType 'reconciliation',内部分发 4 个子任务)")

	// === sys_job 记录: 4 条 R1 任务,若已存在(migration_169 seed 过)则跳过 ===
	// 注意: 这里只在 sys_job 不存在时 INSERT,避免与 migration_169 seed 冲突。
	// 若已存在,AddJob 会按 cron 表达式自动加载,无需重复插入。
	reconJobs := []struct {
		jobName        string
		cronExpression string
		remark         string
	}{
		{
			jobName:        "对账-物化视图刷新",
			cronExpression: "@every 5m",
			remark:         "Phase 42 R1: REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized (D-01 5min 周期)",
		},
		{
			jobName:        "对账-Layer3检测",
			cronExpression: "@every 6m",
			remark:         "Phase 42 R1: 遍历 reconciliation_normalized MV,按 Type A-F 分类写入 sys_data_reconciliation (D-07)",
		},
		{
			jobName:        "对账-静默期重检测",
			cronExpression: "0 0 2 * * *",
			remark:         "Phase 42 R1 placeholder: 修复回写后 7d 静默期重检测 (R2 真实实现)",
		},
		{
			jobName:        "对账-例外规则清理",
			cronExpression: "0 0 3 * * *",
			remark:         "Phase 42 R1 placeholder: 临时例外规则到期清理 (R3 真实实现)",
		},
	}

	for _, j := range reconJobs {
		invokeTarget := jobNameToInvokeTarget(j.jobName)
		var existingCount int64
		if err := db.Model(&models.Job{}).Where("job_name = ?", j.jobName).Count(&existingCount).Error; err == nil && existingCount > 0 {
			applogger.Debugf("对账任务 %s 已存在数据库,跳过创建", j.jobName)
			continue
		}

		nextMinute := time.Now().Add(5 * time.Minute)
		newJob := &models.Job{
			JobName:        j.jobName,
			JobGroup:       "RECONCILIATION",
			InvokeTarget:   invokeTarget,
			CronExpression: j.cronExpression,
			MisfirePolicy:  1, // MisfirePolicyImmediately
			Status:         0, // 0=启用
			NextRunTime:    &nextMinute,
			Remark:         &j.remark,
		}
		if err := db.Create(newJob).Error; err != nil {
			applogger.Warnf("创建对账任务 %s 失败: %v", j.jobName, err)
		} else {
			applogger.Infof("对账任务 %s 已创建", j.jobName)
		}
	}
}

// jobNameToInvokeTarget 把中文 job name 映射到 InvokeTarget 字符串
//
// migration_169 seed 写入 sys_job 时用的是以下格式: reconciliation:refreshView 等。
// 这里保持完全一致,否则 JobExecutor 找不到对应 handler。
func jobNameToInvokeTarget(jobName string) string {
	switch jobName {
	case "对账-物化视图刷新":
		return "reconciliation:refreshView"
	case "对账-Layer3检测":
		return "reconciliation:detectLayer3"
	case "对账-静默期重检测":
		return "reconciliation:detectExpiredSilence"
	case "对账-例外规则清理":
		return "reconciliation:cleanupExpiredExceptions"
	default:
		return ""
	}
}
