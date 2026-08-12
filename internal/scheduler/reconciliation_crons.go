package scheduler

// ============================================================================
// 对账定时任务 cron / InvokeTarget / JobGroup 权威常量
//
// 历史背景(2026-07-05 修订):
//   - 此前 cron 表达式硬编码内联在 reconciliation_tasks.go 的 reconJobs slice,
//     且与 sys_job DB 实际值 / migration_169 seed 各一份,出现 5-field vs 6-field
//     不一致(incident 260705-fix-suggestion-flood)。
//   - 260704-ne5 已删除启动期 migrations.MigrateNNN(d.DB) 调用块,故
//     migration_169/200 的 sys_job seed 函数不再启动期执行 —— reconciliation_tasks.go
//     的兜底 INSERT 是唯一启动期 seed 途径,本文件成为 cron 表达式唯一权威源。
//
// 格式约定:
//   - scheduler 用 cron.WithSeconds() (见 cron.go:186),非 descriptor 表达式
//     必须是 6-field (秒 分 时 日 月 周)。5-field 会被 6-field parser 误解析。
//   - descriptor 表达式 (@every / @daily / ...) 不受 6-field 约束。
//   - 单测 TestReconCronsAreSixField 锁定此约定,任何回退立即失败。
// ============================================================================

// reconciliationJobGroup 对账任务 JobGroup
//
// 统一小写,与 migration_169 历史保持一致。此前代码兜底 INSERT 用过
// "RECONCILIATION"(大写)导致 sys_job.job_group 大小写分裂,
// 启动自愈会统一修正为小写。
const reconciliationJobGroup = "reconciliation"

// 对账 cron 表达式权威源(8 条)
const (
	// 物化视图刷新:D-01 5min 周期。CONCURRENTLY 要求 ≥ MV 刷新最慢耗时(实测 ~30s)。
	reconCronRefreshView = "@every 5m"
	// Layer3 检测:错峰 MV 刷新 1min,避免 MV 刷新未完成时扫到旧数据。
	reconCronDetectLayer3 = "@every 6m"
	// 静默期重检测:每天 02:00(业务低峰),R2 真实实现。
	// 6-field:秒=0 分=0 时=2 日月周=*。
	reconCronDetectExpiredSilence = "0 0 2 * * *"
	// 例外规则清理:每天 03:00,与静默期错峰。
	// 6-field:秒=0 分=0 时=3 日月周=*。
	reconCronCleanupExpiredExceptions = "0 0 3 * * *"
	// critical 异常转工单:D-A1-01 ≤2min SLA。
	reconCronCreateWorkorderCritical = "@every 2m"
	// high 异常转工单:D-A1-02 ≤5min SLA。
	reconCronCreateWorkorderHigh = "@every 5m"
	// 修复建议生成:W-7 错峰 monitor 4min(monitor 在 :07/:17/...,
	// generator 由 @every 5m 启动后约 :03/:08/...),避免同时争用
	// sys_reconciliation_fix_suggestion 写锁。
	reconCronGenerateFixSuggestions = "@every 5m"
	// 误修复率监控:W-7 错峰 generator 4min,每 10min 第 7 分钟触发。
	// 6-field:秒=0 分=7,17,27,37,47,57 时日月周=*。
	// 历史脏值 "7,17,27,37,47,57 * * * *"(5-field)在 legacyCronOverrides 自愈。
	reconCronMonitorFixSuggestionMisFix = "0 7,17,27,37,47,57 * * * *"
)

// 对账 InvokeTarget 子任务名(8 个,作为 params["param"] 路由键)
//
// 完整 InvokeTarget = "reconciliation:" + 子任务名。taskType="reconciliation"
// 在 reconciliation_tasks.go 的 switch 内通过 target 分发。
const (
	reconInvokeRefreshView              = "refreshView"
	reconInvokeDetectLayer3             = "detectLayer3"
	reconInvokeDetectExpiredSilence     = "detectExpiredSilence"
	reconInvokeCleanupExpiredExceptions = "cleanupExpiredExceptions"
	reconInvokeCreateWorkorderCritical  = "createWorkorderCritical"
	reconInvokeCreateWorkorderHigh      = "createWorkorderHigh"
	reconInvokeGenerateFixSuggestions   = "generateFixSuggestions"
	reconInvokeMonitorMisFix            = "monitorFixSuggestionMisFix"
)

// legacyCronOverrides 历史脏 cron → 权威 cron 自愈映射
//
// 启动时若 sys_job.cron_expression 命中 key,UPDATE 为 value。
// 仅列"代码历史硬编码过的错误值"——运维手动改的自定义 cron 不在 map 内,
// 不会被覆盖(尊重运维自治)。新增脏值在此追加即可。
//
// 自愈逻辑见 reconciliation_tasks.go RegisterReconciliationTasks 末尾循环。
var legacyCronOverrides = map[string]string{
	// incident 260705 修订前的 5-field 脏值(缺秒位,scheduler 6-field parser 解析乱套)
	"7,17,27,37,47,57 * * * *": reconCronMonitorFixSuggestionMisFix,
}
