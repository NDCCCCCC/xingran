package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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
//
// R2 闭环(Phase 43 / D-A4-01 + D-A4-03):
//   - wsSvc + noticeSvc 注入到 ReconciliationWorkorderService,critical 转单成功后
//     触发 WS 广播 + sys_notice 写入(双通道)
//   - 兼容性:wsSvc/noticeSvc 为 nil 时跳过 WS/SysNotice(单测 / 非 production)
//
// R4 扩展(Phase 45 / D-A4-04 缓存失效):
//   - cacheSvc 注入到 ReconciliationWorkorderService,createWorkorderBySeverity 在
//     CreateWorkorderFromException 成功后通过 woSvc.WorkstationIDForException 拿
//     wsID,再调 woSvc.InvalidateWorkstationHealth 主动失效缓存
//   - cacheSvc 为 nil 时跳过(单测 / 非 production),失效失败仅 logrus.Warnf
func RegisterReconciliationTasks(s *Scheduler, db *gorm.DB, cacheSvc cache.Cache, wsSvc *websocket.NoticeHub, noticeSvc *services.NoticeService) {
	// 实例化对账服务(无 core 依赖,只依赖 *gorm.DB)
	snapSvc := asset.NewReconciliationSnapshotService(db)
	detSvc := asset.NewReconciliationDetection(db)
	woSvc := asset.NewReconciliationWorkorderServiceWithCache(db, cacheSvc, wsSvc, noticeSvc)
	// Phase 46 R5: 半自动修复建议生成器
	// 注入 configService + noticeHub(D-A3 读 sys_config / D-C5 误修复率告警)
	configSvc := system.NewConfigService(db)
	fixSuggestionSvc := asset.NewFixSuggestionService(db, cacheSvc, configSvc, wsSvc)
	// Phase 46 R5 误修复率监控(46-02 / D-C5)
	// 注入 db + configSvc + wsSvc + noticeSvc + fixSuggestionSvc
	fixSuggestionMonitor := asset.NewFixSuggestionMonitor(db, configSvc, wsSvc, noticeSvc, fixSuggestionSvc)

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
			// Phase 44 R3 / Plan 44-02 Task 1 — 过期例外规则软停用(D-R3-A4-03 + T-44-07)
			//
			// 行为:
			//   - WHERE expires_at IS NOT NULL AND expires_at < NOW() AND is_active=0 AND deleted_at IS NULL
			//   - UPDATE is_active=1 (Status Convention: 0=启用→1=停用)
			//   - **软停用不软删除**:deleted_at 保持 NULL,历史 sys_data_reconciliation.exception_rule_id
			//     仍指向有效(虽停用)记录,审计链不断(D-R3-A4-03)
			//   - 幂等:WHERE 含 is_active=0(仅启用规则),二次 cron 调用 rowsAffected=0
			rowsAffected, err := cleanupExpiredExceptionsDirect(ctx, db, time.Now())
			if err != nil {
				applogger.Errorf("[reconciliation:cleanupExpiredExceptions] 失败: %v", err)
				return err
			}
			applogger.Infof("[reconciliation:cleanupExpiredExceptions] 软停用 %d 条过期例外规则", rowsAffected)
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
		case "generateFixSuggestions":
			// Phase 46 R5: 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion
			// (D-A4 触发条件:D-A3 sys_config confidence_threshold / 紧急熔断 enabled=0)
			inserted, err := fixSuggestionSvc.GenerateFixSuggestions(ctx)
			if err != nil {
				applogger.Errorf("[reconciliation:generateFixSuggestions] 失败: %v", err)
				return err
			}
			applogger.Infof("[reconciliation:generateFixSuggestions] 本轮生成 %d 条建议", inserted)
		case "monitorFixSuggestionMisFix":
			// Phase 46 R5 / 46-02: 误修复率监控(D-C5)
			// 1h 节流防告警风暴 + WS + SysNotice + operlog 三通道(46-02 监控 service 内部处理)
			if err := fixSuggestionMonitor.CheckAndNotify(ctx); err != nil {
				applogger.Errorf("[reconciliation:monitorFixSuggestionMisFix] 失败: %v", err)
				return err
			}
		default:
			return fmt.Errorf("未知子任务: %s", target)
		}
		applogger.Infof("[reconciliation:%s] 完成", target)
		return nil
	})
	applogger.Infof("对账任务处理器已注册(单一 taskType 'reconciliation',内部分发 6 个子任务)")

	// === sys_job 记录: 8 条对账任务 seed + 历史脏 cron 自愈 ===
	//
	// 历史背景(2026-07-05 修订):
	//   - 260704-ne5 已删除启动期 migrations.MigrateNNN(d.DB) 调用块,故
	//     migration_169/200 的 seedReconciliationSysJobs / seedFixSuggestionSysJobs
	//     不再启动期执行 —— 本循环是 sys_job 唯一启动期 seed 途径。
	//   - cron / InvokeTarget / JobGroup 表达式权威源在 reconciliation_crons.go
	//     (常量集中管理,避免双权威源漂移)。
	//
	// 三态自愈策略:
	//   1. sys_job 不存在 → INSERT (全新环境首次启动)
	//   2. sys_job 存在 + cron 命中 legacyCronOverrides 黑名单 → UPDATE 修正历史脏值
	//   3. sys_job 存在 + JobGroup 大小写不一致 / InvokeTarget 漂移 → UPDATE 统一
	//
	// 不动运维手动改的自定义 cron(尊重运维自治)—— 只命中黑名单的脏值才覆盖。
	reconJobs := []struct {
		jobName        string
		cronExpression string
		invokeTarget   string // 子任务名,完整 InvokeTarget = "reconciliation:" + invokeTarget
		remark         string
	}{
		{"对账-物化视图刷新", reconCronRefreshView, reconInvokeRefreshView, "Phase 42 R1: REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized (D-01 5min 周期)"},
		{"对账-Layer3检测", reconCronDetectLayer3, reconInvokeDetectLayer3, "Phase 42 R1: 遍历 reconciliation_normalized MV,按 Type A-F 分类写入 sys_data_reconciliation (D-07)"},
		{"对账-静默期重检测", reconCronDetectExpiredSilence, reconInvokeDetectExpiredSilence, "Phase 42 R1 placeholder: 修复回写后 7d 静默期重检测 (R2 真实实现)"},
		{"对账-例外规则清理", reconCronCleanupExpiredExceptions, reconInvokeCleanupExpiredExceptions, "Phase 42 R1 placeholder: 临时例外规则到期清理 (R3 真实实现)"},
		{"对账-自动转工单critical", reconCronCreateWorkorderCritical, reconInvokeCreateWorkorderCritical, "Phase 43 R2: critical 异常自动转工单(D-A1-01 2min 周期,符合 ROADMAP SC 1)"},
		{"对账-自动转工单high", reconCronCreateWorkorderHigh, reconInvokeCreateWorkorderHigh, "Phase 43 R2: high 异常自动转工单(D-A1-02 5min 周期,符合 ROADMAP SC 2)"},
		{"对账-修复建议生成", reconCronGenerateFixSuggestions, reconInvokeGenerateFixSuggestions, "Phase 46 R5: 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion (D-A4 触发器)"},
		{"对账-误修复率监控", reconCronMonitorFixSuggestionMisFix, reconInvokeMonitorMisFix, "Phase 46 R5 / 46-02: 7d 误修复率超阈告警(D-C5 1h 节流防风暴 + WS + SysNotice + operlog 三通道,W-2026-07-05:Stats() 加 MinSampleSize=5 避免小样本假阳性)"},
	}

	for _, j := range reconJobs {
		fullInvokeTarget := "reconciliation:" + j.invokeTarget

		var existing models.Job
		err := db.Unscoped().Where("job_name = ?", j.jobName).First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 状态 1: 不存在 → INSERT
			nextMinute := time.Now().Add(5 * time.Minute)
			newJob := &models.Job{
				JobName:        j.jobName,
				JobGroup:       reconciliationJobGroup,
				InvokeTarget:   fullInvokeTarget,
				CronExpression: j.cronExpression,
				MisfirePolicy:  1, // MisfirePolicyImmediately
				Status:         0, // 0=启用
				NextRunTime:    &nextMinute,
				Remark:         &j.remark,
			}
			if cErr := db.Create(newJob).Error; cErr != nil {
				applogger.Warnf("创建对账任务 %s 失败: %v", j.jobName, cErr)
			} else {
				applogger.Infof("对账任务 %s 已创建 (cron=%s)", j.jobName, j.cronExpression)
			}
			continue
		}
		if err != nil {
			applogger.Warnf("查询对账任务 %s 失败,跳过自愈: %v", j.jobName, err)
			continue
		}

		// 状态 2/3: 存在 → 检查是否需要自愈
		updates := map[string]interface{}{}

		// 自愈 a: cron 命中历史脏值黑名单(运维手动改的自定义 cron 不在 map 内,不动)
		if newCron, ok := legacyCronOverrides[existing.CronExpression]; ok {
			updates["cron_expression"] = newCron
			applogger.Warnf("[reconciliation] 自愈: job=%s cron '%s' → '%s' (历史脏值)",
				j.jobName, existing.CronExpression, newCron)
		}
		// 自愈 b: JobGroup 大小写不一致(此前兜底 INSERT 用过 "RECONCILIATION" 大写)
		if existing.JobGroup != reconciliationJobGroup {
			updates["job_group"] = reconciliationJobGroup
			applogger.Warnf("[reconciliation] 自愈: job=%s job_group '%s' → '%s'",
				j.jobName, existing.JobGroup, reconciliationJobGroup)
		}
		// 自愈 c: InvokeTarget 漂移(兜底防备:target 错了 taskType 路由就废了)
		if existing.InvokeTarget != fullInvokeTarget {
			updates["invoke_target"] = fullInvokeTarget
			applogger.Warnf("[reconciliation] 自愈: job=%s invoke_target '%s' → '%s'",
				j.jobName, existing.InvokeTarget, fullInvokeTarget)
		}

		if len(updates) > 0 {
			if uErr := db.Model(&models.Job{}).Where("id = ?", existing.ID).Updates(updates).Error; uErr != nil {
				applogger.Warnf("自愈对账任务 %s 失败: %v", j.jobName, uErr)
			}
		} else {
			applogger.Debugf("对账任务 %s 已存在且无需自愈", j.jobName)
		}
	}
}

// createWorkorderBySeverity 扫描指定 severity 的未转单异常并调转单 service
//
// 设计考量:
//   - 单函数支持 critical / high 两种 severity,避免重复 switch 代码
//   - SELECT 含 deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL 4 条件,
//     确保只处理真正"开放 + 未转单"的异常(防重复转单)
//   - **BLOCKER-4 (Phase 44 R3 / D-R3-A1-02)**:WHERE 强制含
//     `AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))`。
//     理由:`internal/models/reconciliation.go:50` 的 `AppliedActions pq.StringArray` 无 default tag,
//     PG INSERT 未指定值时默认 NULL(非空数组)。PG 三值逻辑下
//     `'no_workorder' != ANY(NULL)` 返回 NULL,整个 WHERE 子句该行被过滤——会导致所有
//     "无例外命中"(applied_actions=NULL)的 critical/high 异常被漏转,反向破坏 R2 WORKORDER-01/02。
//     IS NULL 兜底保证 NULL 数组的异常仍被转单。
//     参考: https://www.postgresql.org/docs/current/functions-comparisons.html#id-1.5.8.30.16
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
	// BLOCKER-4:WHERE 强制含 applied_actions IS NULL 兜底(见函数 doc)。
	// 方言注意:'no_workorder' != ANY(applied_actions) 是 PG 数组操作符;
	// sqlite 下 applied_actions 由 sanitizeSQLiteModelDefaults 降为 text 列
	// ("{a,b}" 字面量),sqlite 无 ANY 函数,直接执行报语法错误。
	// sqlite 等价:数组文本去花括号后以逗号包裹,LIKE '%,no_workorder,%' 判定包含,
	// 空数组 '{}' → ',,' 不命中(正确:无动作应转单),NULL 由 IS NULL 兜底。
	whereClause := "severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL " +
		"AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))"
	if db.Dialector.Name() == "sqlite" {
		whereClause = "severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL " +
			"AND (applied_actions IS NULL OR (',' || TRIM(applied_actions, '{}') || ',') NOT LIKE '%,no_workorder,%')"
	}
	if err := db.WithContext(ctx).
		Where(whereClause, severity).
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

		// 🆕 Phase 45 R4 / D-A4-04: 缓存主动失效(避免用户重看页面仍命中旧缓存)
		// 反查 exception → workstation_id,然后调 woSvc.InvalidateWorkstationHealth 失效缓存。
		// 失效失败仅 logrus.Warnf,不阻断 cron 流程(缓存最终会在 5min TTL 后自动失效)。
		wsID, wsErr := woSvc.WorkstationIDForException(ctx, rec.ID)
		if wsErr != nil {
			logrus.Warnf("[reconciliation:createWorkorder%s] WorkstationIDForException 失败 exceptionID=%s: %v", severity, rec.ID, wsErr)
		} else if wsID != "" {
			if invErr := woSvc.InvalidateWorkstationHealth(ctx, wsID); invErr != nil {
				logrus.Warnf("[reconciliation:createWorkorder%s] R4 invalidate cache failed exceptionID=%s wsID=%s: %v", severity, rec.ID, wsID, invErr)
			}
		}
	}

	applogger.Infof("[reconciliation:createWorkorder%s] 完成: success=%d failure=%d total=%d",
		severity, successCount, failureCount, len(exceptions))
	return nil
}

// cleanupExpiredExceptionsDirect 软停用过期例外规则(Phase 44 R3 / Plan 44-02 Task 1)
//
// 行为(D-R3-A4-03 + T-44-07 审计链缓解):
//   - WHERE expires_at IS NOT NULL AND expires_at < ?(now) AND is_active = 0 AND deleted_at IS NULL
//   - UPDATE is_active = 1 (Status Convention: 0=启用 → 1=停用)
//   - **软停用不软删除**:deleted_at 保持 NULL,历史 sys_data_reconciliation.exception_rule_id
//     仍指向有效(虽停用)记录,审计链不断(D-R3-A4-03 / Pitfall 4 防外键断链)
//   - 幂等:WHERE 含 is_active=0(仅启用规则),二次 cron 调用 rowsAffected=0
//
// now 参数(cron 调用方传 time.Now())让函数 deterministic 可测(SQLite 上 NOW() 不可移植)。
// 生产 cron 调度时,now 等价于 PG 的 NOW()(同一 cron 周期内的时刻)。
//
// 抽出为独立函数便于单测(避免依赖完整 cron 引擎 + Scheduler 实例)。
func cleanupExpiredExceptionsDirect(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	result := db.WithContext(ctx).
		Model(&models.SysReconciliationException{}).
		Where("expires_at IS NOT NULL AND expires_at < ? AND is_active = ? AND deleted_at IS NULL", now, 0).
		Update("is_active", 1) // Status Convention: 0=启用 → 1=停用
	if result.Error != nil {
		return 0, fmt.Errorf("软停用过期例外规则失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// jobNameToInvokeTarget 已于 2026-07-05 修订移除。
//
// 此前 cron/InvokeTarget 用 name → target 的 switch 映射,导致权威源分散(常量 slice
// + 此 switch + migration_169 seed 三份)。InvokeTarget 现作为 reconJobs slice 字段
// 直接引用 reconciliation_crons.go 的常量,单一权威源,无需映射函数。

// checkPortStatusDrift 每日检测 sys_device_port_status.device_id 与
// ops_info_points.device_id 漂移(Phase 45 R5 数据治理 2026-06-30, 2026-07-01 修订)
//
// 背景:
//   - 1247/1483 (84.14%) 信息点存在历史漂移(2026-06-30 发现)
//   - 用户决策 (2026-07-01): 不做数据修复 migration, 改用 UUID PK 锚定让 query 不依赖 device_id 一致性
//     (workstation_device_service.go strict JOIN 已砍, MAC JOIN 锚定 ip.device_id — collector 按 info_point 写 MAC, port.device_id 是历史脏数据)
//   - migration_183 FK + Excel DependsOn + REST API 校验 + 本任务告警 四层防御
//   - 漂移行不影响物理链路查询(2026-07-01 验证), 仅供运维核对趋势
//
// 检测条件(safe-fix 等价, 用于判断"是否新漂移"):
//   - port.id::text = ip.port_id(同一端口)
//   - port.device_id::text != ip.device_id::text(确实漂移)
//   - ip.deleted_at IS NULL AND ip.status = 0
//   - ip.device_id 存在于 sys_network_device
//   - ip.device_id 在 sys_device_mac_address 有 MAC(物理证据)
//
// 行为 (自适应基线, sys_config 存历史最低漂移数, 不硬编码):
//   - 首次观测: 记录当前 drift 为基线 (info, 不告警)
//   - drift 下降: 更新基线, info "数据治理见效" (历史漂移被修复后基线自动下降)
//   - drift 上涨超阈值: WARN 新增 (基线不更新, 持续告警直到人工处理)
//   - drift == 0: info 健康, 基线归 0
//   - 后续 PR 可在此处加企业微信/钉钉 webhook 推送告警
func checkPortStatusDrift(ctx context.Context, db *gorm.DB) error {
	// 方言感知(2026-08-17):`::text` 是 PG 专有 cast(SQLite 报 unrecognized token ":")。
	// SQLite 下各 id/device_id 列均为文本,直接等值/不等比较即可;PG SQL 保持原文不变。
	query := `
SELECT COUNT(*)
  FROM sys_device_port_status port
  JOIN ops_info_points ip ON port.id::text = ip.port_id
 WHERE port.device_id::text != ip.device_id::text
   AND ip.deleted_at IS NULL
   AND ip.status = 0
   AND ip.device_id IS NOT NULL
   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
   AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id::text = ip.device_id)
`
	if db.Dialector.Name() != "postgres" {
		query = `
SELECT COUNT(*)
  FROM sys_device_port_status port
  JOIN ops_info_points ip ON port.id = ip.port_id
 WHERE port.device_id != ip.device_id
   AND ip.deleted_at IS NULL
   AND ip.status = 0
   AND ip.device_id IS NOT NULL
   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id = ip.device_id)
   AND EXISTS (SELECT 1 FROM sys_device_mac_address WHERE device_id = ip.device_id)
`
	}

	var driftedCount int64
	err := db.WithContext(ctx).Raw(query).Scan(&driftedCount).Error

	if err != nil {
		return fmt.Errorf("查询 port_status 漂移失败: %w", err)
	}

	// 自适应基线告警 (2026-07-01):
	//   drift 是历史脏数据, 不影响物理链路查询 (MAC JOIN 已锚定 ip.device_id)。
	//   基线 = sys_config 记录的"历史最低漂移数", 自适应追踪:
	//     - 历史漂移被修复 → 基线自动下降, 不锁死在初始高值
	//     - 新增漂移 → 持续 WARN 直到人工处理 (基线不随上涨更新)
	//   阈值容差避免采集抖动 ±几行触发告警。
	const driftBaselineKey = "reconciliation.port_status.drift_baseline"
	const driftWarnThreshold int64 = 5

	baseline, baselineExists := readDriftBaseline(ctx, db, driftBaselineKey)
	switch {
	case driftedCount == 0:
		applogger.Infof("[reconciliation:checkPortStatusDrift] 0 行漂移(健康)")
		if baselineExists && baseline != 0 {
			upsertDriftBaseline(ctx, db, driftBaselineKey, 0)
		}
	case !baselineExists:
		applogger.Infof("[reconciliation:checkPortStatusDrift] 首次观测 %d 行漂移, 记录为基线", driftedCount)
		upsertDriftBaseline(ctx, db, driftBaselineKey, driftedCount)
	case driftedCount > baseline+driftWarnThreshold:
		applogger.Warnf("[reconciliation:checkPortStatusDrift] %d 行漂移(基线 %d, 新增 %d 行) — 需排查采集器/IP 录入流程",
			driftedCount, baseline, driftedCount-baseline)
	case driftedCount < baseline:
		applogger.Infof("[reconciliation:checkPortStatusDrift] %d 行漂移(基线 %d → %d 下降, 数据治理见效)",
			driftedCount, baseline, driftedCount)
		upsertDriftBaseline(ctx, db, driftBaselineKey, driftedCount)
	default:
		applogger.Infof("[reconciliation:checkPortStatusDrift] %d 行漂移(基线 %d, 持平, 不影响 query)", driftedCount, baseline)
	}
	return nil
}

// readDriftBaseline 读取 sys_config 中记录的历史最低漂移数基线。
// 未初始化或查询失败都按"首次观测"处理 (返回 exists=false)。
func readDriftBaseline(ctx context.Context, db *gorm.DB, key string) (int64, bool) {
	var config models.Config
	if err := db.WithContext(ctx).Where("config_key = ?", key).First(&config).Error; err != nil {
		return 0, false
	}
	val, err := strconv.ParseInt(config.ConfigValue, 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// upsertDriftBaseline 更新或创建 sys_config 中的漂移基线 (历史最低值)。
// 用 map 形式 Assign 避免覆盖 config_key 与其他字段。
func upsertDriftBaseline(ctx context.Context, db *gorm.DB, key string, value int64) {
	var config models.Config
	db.WithContext(ctx).
		Where("config_key = ?", key).
		Assign(map[string]interface{}{
			"config_value": strconv.FormatInt(value, 10),
			"config_name":  "端口状态漂移基线(自适应)",
		}).
		FirstOrCreate(&config, models.Config{ConfigKey: key, ConfigName: "端口状态漂移基线(自适应)"})
}
