package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegisterReconciliationTasks 注册 Phase 42 R1 + Phase 43 R2 资产对账相关定时任务处理器。
//
// 关键修复: Phase 42-r1 plan 02 创建了 4 个 sys_job 记录(对账-物化视图刷新 / Layer3 检测 /
// 静默期重检测 / 例外规则清理)并实现了 Execute*Task 全局函数,但漏了把 handler 注册到
// Scheduler.RegisterTask(任务类型 → 实际函数映射)。结果: 调度器启动时找不到
// 'reconciliation:refreshView' 等 taskType 对应的 handler,即使 cron 表达式正确也
// 报 "未找到任务处理器: reconciliation"。
//
// 本文件补全 RegisterTask + sys_job 记录(若已存在则跳过)。
//
// R1 行为(R1 boundary):
//   - reconciliation:refreshView          → 真实执行 MV CONCURRENTLY refresh
//   - reconciliation:detectLayer3         → 真实执行 Layer 3 检测写入 sys_data_reconciliation
//   - reconciliation:detectExpiredSilence → R1 placeholder(R2 真实实现:修复回写后 7d 静默期)
//   - reconciliation:cleanupExpiredExceptions → R1 placeholder(R3 真实实现:临时例外到期清理)
//
// R2 新增(Phase 43 / D-A1-01 + D-A1-02):
//   - reconciliation:createWorkorderCritical → 扫 critical 异常(@every 2m)→ 调
//     asset.ReconciliationWorkorderService.CreateWorkorderFromException
//   - reconciliation:createWorkorderHigh    → 扫 high 异常(@every 5m)→ 同上
func RegisterReconciliationTasks(s *Scheduler, db *gorm.DB) {
	// 实例化对账服务(无 core 依赖,只依赖 *gorm.DB)
	snapSvc := asset.NewReconciliationSnapshotService(db)
	detSvc := asset.NewReconciliationDetection(db)
	woSvc := asset.NewReconciliationWorkorderService(db)

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
			// R2 返回值扩展(Phase 43 / D-A3-03):inserted / skipped / skippedSilence / skippedThrottle
			created, skipped, skippedSilence, skippedThrottle, err := detSvc.DetectLayer3(ctx)
			if err != nil {
				applogger.Errorf("[reconciliation:detectLayer3] 失败: %v", err)
				return err
			}
			applogger.Infof("[reconciliation:detectLayer3] 完成: created=%d skipped=%d skippedSilence=%d skippedThrottle=%d",
				created, skipped, skippedSilence, skippedThrottle)
		case "detectExpiredSilence":
			applogger.Infof("[reconciliation:detectExpiredSilence] R1 placeholder, R2 真实实现")
		case "cleanupExpiredExceptions":
			applogger.Infof("[reconciliation:cleanupExpiredExceptions] R1 placeholder, R3 真实实现")
		case "createWorkorderCritical":
			// Phase 43 R2 / D-A1-01: critical 异常 ≤2min 自动转工单
			// 扫 sys_data_reconciliation WHERE severity='critical' AND deleted_at IS NULL
			//   AND resolved_at IS NULL AND workorder_id IS NULL
			// ORDER BY detected_at ASC LIMIT 50 (避免单次 cron 周期堆积)
			if err := createWorkorderBySeverity(ctx, db, woSvc, "critical", 50); err != nil {
				applogger.Errorf("[reconciliation:createWorkorderCritical] 失败: %v", err)
				return err
			}
		case "createWorkorderHigh":
			// Phase 43 R2 / D-A1-02: high 异常 ≤5min 自动转工单
			// 同 createWorkorderCritical 但 severity='high',LIMIT 30
			if err := createWorkorderBySeverity(ctx, db, woSvc, "high", 30); err != nil {
				applogger.Errorf("[reconciliation:createWorkorderHigh] 失败: %v", err)
				return err
			}
		default:
			return fmt.Errorf("未知子任务: %s", target)
		}
		applogger.Infof("[reconciliation:%s] 完成", target)
		return nil
	})
	applogger.Infof("对账任务处理器已注册(单一 taskType 'reconciliation',内部分发 6 个子任务)")

	// === sys_job 记录: 4 条 R1 + 2 条 R2 任务,若已存在(migration_169/171 seed 过)则跳过 ===
	// 注意: 这里只在 sys_job 不存在时 INSERT,避免与 migration_169/171 seed 冲突。
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
		// === Phase 43 R2 新增 2 条 cron(续写,见 plan 43-01 Task 2) ===
		{
			jobName:        "对账-自动转工单critical",
			cronExpression: "@every 2m",
			remark:         "Phase 43 R2: critical 异常自动转工单(D-A1-01 2min 周期,符合 ROADMAP SC 1)",
		},
		{
			jobName:        "对账-自动转工单high",
			cronExpression: "@every 5m",
			remark:         "Phase 43 R2: high 异常自动转工单(D-A1-02 5min 周期,符合 ROADMAP SC 2)",
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

// createWorkorderBySeverity 扫描指定 severity 的未转单异常并调转单 service
//
// 设计考量:
//   - 单函数支持 critical / high 两种 severity,避免重复 switch 代码
//   - SELECT 含 deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL 4 条件,
//     确保只处理真正"开放 + 未转单"的异常(防重复转单)
//   - ORDER BY detected_at ASC LIMIT N:按时间顺序处理(先到先服务);LIMIT 避免单次 cron 周期
//     处理过多(预留 1-2min 时间给单条转单流程)
//   - 单条转单失败仅 logrus.Errorf + continue(D-A1-03 不写 SysNotice),其他正常异常继续处理
//
// 输入:
//   - ctx: 上下文
//   - db: *gorm.DB
//   - woSvc: 已构造的 ReconciliationWorkorderService
//   - severity: "critical" 或 "high"
//   - limit: 单次扫描上限(避免 cron 周期堆积)
func createWorkorderBySeverity(ctx context.Context, db *gorm.DB, woSvc *asset.ReconciliationWorkorderService, severity string, limit int) error {
	var exceptions []models.SysDataReconciliation
	if err := db.WithContext(ctx).
		Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL", severity).
		Order("detected_at ASC").
		Limit(limit).
		Find(&exceptions).Error; err != nil {
		return fmt.Errorf("查询 %s 异常失败: %w", severity, err)
	}

	if len(exceptions) == 0 {
		applogger.Infof("[reconciliation:createWorkorder%s] 无待转单异常", severity)
		return nil
	}

	successCount := 0
	failureCount := 0
	for _, rec := range exceptions {
		if _, err := woSvc.CreateWorkorderFromException(ctx, rec.ID); err != nil {
			// 单条失败仅 log,继续下一条(D-A1-03)
			failureCount++
			logrus.Errorf("[reconciliation:createWorkorder%s] 转单失败 exceptionID=%s: %v", severity, rec.ID, err)
			continue
		}
		successCount++
	}

	applogger.Infof("[reconciliation:createWorkorder%s] 完成: success=%d failure=%d total=%d",
		severity, successCount, failureCount, len(exceptions))
	return nil
}

// jobNameToInvokeTarget 把中文 job name 映射到 InvokeTarget 字符串
//
// migration_169/171 seed 写入 sys_job 时用的是以下格式: reconciliation:refreshView 等。
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
	case "对账-自动转工单critical":
		return "reconciliation:createWorkorderCritical"
	case "对账-自动转工单high":
		return "reconciliation:createWorkorderHigh"
	default:
		return ""
	}
}
