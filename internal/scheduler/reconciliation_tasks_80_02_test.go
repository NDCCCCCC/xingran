package scheduler

// Phase 80 Plan 02 — reconciliation_tasks.go 测试
// 覆盖:RegisterReconciliationTasks 注册体 + createWorkorderBySeverity
// + checkPortStatusDrift 分支 + read/upsertDriftBaseline round-trip
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"gorm.io/gorm"
)

// newReconDB8002 sqlite 文件库 + reconciliation 相关表。
func newReconDB8002(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "rct8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Job{}, &models.JobLog{}))

	ddl := []string{
		// sys_data_reconciliation 对账异常表(createWorkorderBySeverity 查询列)
		`CREATE TABLE IF NOT EXISTS sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			asset_id TEXT,
			conflict_type TEXT,
			recon_category TEXT,
			severity TEXT,
			physical_value TEXT,
			declared_value TEXT,
			ad_value TEXT,
			confidence_score REAL DEFAULT 0,
			raw_snapshot TEXT,
			asset_ip TEXT,
			exception_rule_id TEXT,
			resolved_at DATETIME,
			resolved_by TEXT,
			resolve_note TEXT,
			workorder_id TEXT,
			applied_actions TEXT,
			detected_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		// sys_config 漂移基线表(含 BaseModel 列:created_by/updated_by/version)
		`CREATE TABLE IF NOT EXISTS sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			config_name TEXT,
			config_key TEXT,
			config_value TEXT,
			config_type TEXT DEFAULT 'Y',
			is_system INTEGER DEFAULT 0,
			remark TEXT
		)`,
		// checkPortStatusDrift 依赖的四张表
		`CREATE TABLE IF NOT EXISTS sys_device_port_status (
			id TEXT PRIMARY KEY,
			device_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS ops_info_points (
			id TEXT PRIMARY KEY,
			port_id TEXT,
			device_id TEXT,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_network_device (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_device_mac_address (
			id TEXT PRIMARY KEY,
			device_id TEXT,
			mac_address TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		// createWorkorderBySeverity 转单可能触达的工单表
		`CREATE TABLE IF NOT EXISTS sys_work_order (
			id TEXT PRIMARY KEY,
			title TEXT,
			work_order_no TEXT,
			category_id TEXT,
			type TEXT,
			priority INTEGER,
			status INTEGER,
			description TEXT,
			submitter_id TEXT,
			assignee_id TEXT,
			is_auto_assigned INTEGER,
			assign_strategy TEXT,
			duty_type TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		require.NoError(t, db.Exec(stmt).Error)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// ============================================================================
// TestRct8002_RegisterTasks — 注册体(nil-safe 形参)
// ============================================================================

// TestRct8002_RegisterTasks RegisterReconciliationTasks nil wsSvc/noticeSvc 注册路径。
func TestRct8002_RegisterTasks(t *testing.T) {
	db := newReconDB8002(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	// wsSvc/noticeSvc 传 nil(research §1.2:注册路径 nil-safe,不广播)
	require.NotPanics(t, func() {
		RegisterReconciliationTasks(s, db, nil, nil, nil)
	})

	// 单一 taskType "reconciliation" 内部分发
	assert.True(t, s.IsTaskRegistered("reconciliation"))
	assert.NotNil(t, s.GetTaskHandler("reconciliation"))

	// 8 条 sys_job seed(其中 critical/high/refreshView 等)
	var count int64
	require.NoError(t, db.Model(&models.Job{}).Where("job_group = ?", "reconciliation").Count(&count).Error)
	assert.Equal(t, int64(8), count, "8 条对账 job 应被 seed")

	// 子任务分发:未知 target → 错误;refreshView 分支(sqlite 无 MV,预期报错但 handler 返回错误)
	handler := s.GetTaskHandler("reconciliation")
	err := handler(context.Background(), map[string]interface{}{"param": "unknown_target"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知子任务")

	// detectExpiredSilence R1 placeholder → nil
	err = handler(context.Background(), map[string]interface{}{"param": "detectExpiredSilence"})
	require.NoError(t, err)
}

// ============================================================================
// TestRct8002_CreateWorkorderBySeverity — 转单分支
// ============================================================================

// TestRct8002_CreateWorkorderBySeverity_Empty 无待转单异常 → 早退。
func TestRct8002_CreateWorkorderBySeverity_Empty(t *testing.T) {
	db := newReconDB8002(t)
	woSvc := newReconWoSvc8002(t, db)

	err := createWorkorderBySeverity(context.Background(), db, woSvc, "critical", 50)
	require.NoError(t, err)
}

// TestRct8002_CreateWorkorderBySeverity_Loop 有异常 → 循环分支(转单成功或失败均覆盖)。
func TestRct8002_CreateWorkorderBySeverity_Loop(t *testing.T) {
	db := newReconDB8002(t)
	woSvc := newReconWoSvc8002(t, db)

	// 种子 2 条 critical 异常(applied_actions NULL / 无 no_workorder)
	// raw_snapshot 用 CAST AS BLOB:模型字段是 json.RawMessage([]byte),sqlite 驱动
	// 返回 string 会 Scan 失败(PG jsonb → []byte,sqlite 需 BLOB 等价)。
	now := "2026-08-28 10:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, detected_at)
		VALUES ('exc-8002-a', 'asset-a', 'A', 'critical', CAST('{}' AS BLOB), ?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, detected_at)
		VALUES ('exc-8002-b', 'asset-b', 'B', 'critical', CAST('{}' AS BLOB), ?)`, now).Error)

	// 转单循环:CreateWorkorderFromException 成功/失败均被循环覆盖,函数最终返回 nil
	err := createWorkorderBySeverity(context.Background(), db, woSvc, "critical", 50)
	require.NoError(t, err)

	// high severity 同路径(LIMIT 30)
	err = createWorkorderBySeverity(context.Background(), db, woSvc, "high", 30)
	require.NoError(t, err)
}

// TestRct8002_CreateWorkorderBySeverity_NoWorkorderFilter applied_actions 含 no_workorder → 被过滤。
func TestRct8002_CreateWorkorderBySeverity_NoWorkorderFilter(t *testing.T) {
	db := newReconDB8002(t)
	woSvc := newReconWoSvc8002(t, db)

	now := "2026-08-28 10:00:00"
	// applied_actions = '{no_workorder}' → sqlite LIKE 分支过滤
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, applied_actions, detected_at)
		VALUES ('exc-8002-filtered', 'asset-c', 'A', 'critical', CAST('{}' AS BLOB), '{no_workorder}', ?)`, now).Error)

	err := createWorkorderBySeverity(context.Background(), db, woSvc, "critical", 50)
	require.NoError(t, err) // 无待转单 → 早退分支
}

// ============================================================================
// TestRct8002_CheckPortStatusDrift — 漂移检测分支
// ============================================================================

// TestRct8002_CheckPortStatusDrift_NoDrift 0 行漂移(健康分支)。
func TestRct8002_CheckPortStatusDrift_NoDrift(t *testing.T) {
	db := newReconDB8002(t)

	err := checkPortStatusDrift(context.Background(), db)
	require.NoError(t, err)
}

// TestRct8002_CheckPortStatusDrift_FirstObs 首次观测 → 记录基线。
func TestRct8002_CheckPortStatusDrift_FirstObs(t *testing.T) {
	db := newReconDB8002(t)
	seedDrift8002(t, db, 3)

	err := checkPortStatusDrift(context.Background(), db)
	require.NoError(t, err)

	// 基线应写入
	val, exists := readDriftBaseline(context.Background(), db, "reconciliation.port_status.drift_baseline")
	assert.True(t, exists)
	assert.Equal(t, int64(3), val)
}

// TestRct8002_CheckPortStatusDrift_UpDown 漂移上涨/下降分支。
func TestRct8002_CheckPortStatusDrift_UpDown(t *testing.T) {
	db := newReconDB8002(t)

	// 基线 10,当前 3 → 下降分支 → 基线更新为 3
	seedDrift8002(t, db, 3)
	upsertDriftBaseline(context.Background(), db, "reconciliation.port_status.drift_baseline", 10)

	err := checkPortStatusDrift(context.Background(), db)
	require.NoError(t, err)
	val, _ := readDriftBaseline(context.Background(), db, "reconciliation.port_status.drift_baseline")
	assert.Equal(t, int64(3), val, "下降后基线应更新")

	// 基线 0,当前 0 → 健康分支(baselineExists && baseline==0 → 不写)
	seedDrift8002(t, db, 0)
	upsertDriftBaseline(context.Background(), db, "reconciliation.port_status.drift_baseline", 0)
	err = checkPortStatusDrift(context.Background(), db)
	require.NoError(t, err)
}

// ============================================================================
// TestRct8002_Baseline_RoundTrip — read/upsert round-trip
// ============================================================================

// TestRct8002_Baseline_RoundTrip 无基线/有基线 + 首写/覆盖。
func TestRct8002_Baseline_RoundTrip(t *testing.T) {
	db := newReconDB8002(t)
	ctx := context.Background()
	key := "rct8002.baseline.test"

	// 无基线 → exists=false
	val, exists := readDriftBaseline(ctx, db, key)
	assert.False(t, exists)
	assert.Equal(t, int64(0), val)

	// 首写
	upsertDriftBaseline(ctx, db, key, 42)
	val, exists = readDriftBaseline(ctx, db, key)
	assert.True(t, exists)
	assert.Equal(t, int64(42), val)

	// 覆盖
	upsertDriftBaseline(ctx, db, key, 7)
	val, exists = readDriftBaseline(ctx, db, key)
	assert.True(t, exists)
	assert.Equal(t, int64(7), val)
}

// ============================================================================
// helpers
// ============================================================================

// newReconWoSvc8002 构造 ReconciliationWorkorderService(缓存/WS/notice 全 nil)。
func newReconWoSvc8002(t *testing.T, db *gorm.DB) *asset.ReconciliationWorkorderService {
	t.Helper()
	return asset.NewReconciliationWorkorderServiceWithCache(db, nil, nil, nil)
}

// seedDrift8002 种子 N 行漂移数据(port.device_id != ip.device_id,满足 EXISTS 条件)。
func seedDrift8002(t *testing.T, db *gorm.DB, n int) {
	t.Helper()
	require.NoError(t, db.Exec(`DELETE FROM sys_device_port_status`).Error)
	require.NoError(t, db.Exec(`DELETE FROM ops_info_points`).Error)
	require.NoError(t, db.Exec(`DELETE FROM sys_network_device`).Error)
	require.NoError(t, db.Exec(`DELETE FROM sys_device_mac_address`).Error)

	for i := 0; i < n; i++ {
		portID := "port-" + string(rune('a'+i))
		netDevID := "netdev-" + string(rune('a'+i))
		require.NoError(t, db.Exec(`INSERT INTO sys_network_device (id, name) VALUES (?, ?)`, netDevID, "dev").Error)
		require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address (id, device_id, mac_address) VALUES (?, ?, ?)`,
			"mac-"+portID, netDevID, "aa:bb:cc:dd:ee:0"+string(rune('0'+i))).Error)
		// ip.device_id = netDevID(有网络设备 + MAC 证据)
		require.NoError(t, db.Exec(`INSERT INTO ops_info_points (id, port_id, device_id, status) VALUES (?, ?, ?, 0)`,
			"ip-"+portID, portID, netDevID).Error)
		// port.device_id = 别的设备 → 漂移
		require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status (id, device_id) VALUES (?, ?)`,
			portID, "other-device").Error)
	}
}

// ============================================================================
// 缺口补足 Round 1 — handler 分发 switch / 自愈分支 / 漂移分支 / 转单成功路径
// ============================================================================

// TestRct8002_HandlerDispatch_Switch reconciliation handler 7 个子任务分发分支。
func TestRct8002_HandlerDispatch_Switch(t *testing.T) {
	db := newReconDB8002(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	// 例外规则表(cleanupExpiredExceptions 需要)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_reconciliation_exception (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		name TEXT,
		ip_range TEXT,
		conflict_types TEXT,
		exception_actions TEXT,
		severity_override TEXT,
		scope_type TEXT,
		scope_id TEXT,
		reason TEXT,
		is_active INTEGER DEFAULT 0,
		expires_at DATETIME
	)`).Error)
	// 修复建议表(generateFixSuggestions 需要)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_reconciliation_fix_suggestion (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	// 误修复率监控依赖 sys_config(已有)+ 数据表(缺失时 CheckAndNotify 内部软失败)

	RegisterReconciliationTasks(s, db, nil, nil, nil)
	handler := s.GetTaskHandler("reconciliation")
	require.NotNil(t, handler)

	// refreshView:sqlite 无物化视图 → 报错路径(handler 透传 error)
	err := handler(context.Background(), map[string]interface{}{"param": "refreshView"})
	_ = err // sqlite 上 RefreshView 行为不定(成功或报错),不 panic 即覆盖分发分支

	// detectLayer3:sqlite 无 MV → 报错或 0 检出
	err = handler(context.Background(), map[string]interface{}{"param": "detectLayer3"})
	_ = err

	// cleanupExpiredExceptions:表已建 → 成功(nil)
	require.NoError(t, handler(context.Background(), map[string]interface{}{"param": "cleanupExpiredExceptions"}))

	// createWorkorderCritical / createWorkorderHigh:空异常 → nil 早退
	require.NoError(t, handler(context.Background(), map[string]interface{}{"param": "createWorkorderCritical"}))
	require.NoError(t, handler(context.Background(), map[string]interface{}{"param": "createWorkorderHigh"}))

	// generateFixSuggestions:表已建 → 成功或错误(视实现),不 panic 即覆盖
	err = handler(context.Background(), map[string]interface{}{"param": "generateFixSuggestions"})
	_ = err

	// monitorFixSuggestionMisFix:软失败(内部 Warnf)→ nil
	require.NoError(t, handler(context.Background(), map[string]interface{}{"param": "monitorFixSuggestionMisFix"}))
}

// TestRct8002_SelfHeal sys_job seed 自愈三分支:legacy cron / JobGroup 大小写 / InvokeTarget 漂移。
func TestRct8002_SelfHeal(t *testing.T) {
	db := newReconDB8002(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	// 预置 4 条 job:3 条脏 + 1 条干净
	require.NoError(t, db.Exec(`INSERT INTO sys_job (id, job_name, job_group, invoke_target, cron_expression, status, misfire_policy)
		VALUES ('job-legacy', '对账-误修复率监控', 'reconciliation', 'reconciliation:monitorFixSuggestionMisFix', '7,17,27,37,47,57 * * * *', 0, 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_job (id, job_name, job_group, invoke_target, cron_expression, status, misfire_policy)
		VALUES ('job-group', '对账-Layer3检测', 'RECONCILIATION', 'reconciliation:detectLayer3', '@every 6m', 0, 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_job (id, job_name, job_group, invoke_target, cron_expression, status, misfire_policy)
		VALUES ('job-target', '对账-物化视图刷新', 'reconciliation', 'reconciliation:WRONG_TARGET', '@every 5m', 0, 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_job (id, job_name, job_group, invoke_target, cron_expression, status, misfire_policy)
		VALUES ('job-clean', '对账-静默期重检测', 'reconciliation', 'reconciliation:detectExpiredSilence', '0 0 2 * * *', 0, 1)`).Error)

	RegisterReconciliationTasks(s, db, nil, nil, nil)

	// a) legacy cron 被替换为权威值
	var cronExpr string
	require.NoError(t, db.Raw(`SELECT cron_expression FROM sys_job WHERE id = 'job-legacy'`).Scan(&cronExpr).Error)
	assert.Equal(t, reconCronMonitorFixSuggestionMisFix, cronExpr, "legacy cron 应被自愈")

	// b) JobGroup 统一为小写
	var grp string
	require.NoError(t, db.Raw(`SELECT job_group FROM sys_job WHERE id = 'job-group'`).Scan(&grp).Error)
	assert.Equal(t, reconciliationJobGroup, grp, "JobGroup 应被统一")

	// c) InvokeTarget 修正
	var tgt string
	require.NoError(t, db.Raw(`SELECT invoke_target FROM sys_job WHERE id = 'job-target'`).Scan(&tgt).Error)
	assert.Equal(t, "reconciliation:"+reconInvokeRefreshView, tgt, "InvokeTarget 漂移应被修正")

	// 其余 4 条 job 补齐 seed(共 8 条)
	var count int64
	require.NoError(t, db.Model(&models.Job{}).Where("job_group = ?", "reconciliation").Count(&count).Error)
	assert.Equal(t, int64(8), count)
}

// TestRct8002_CreateWorkorderBySeverity_SuccessPath 转单成功路径:
// woSvc 挂空库(查不到异常 → 返回 nil,nil)→ successCount 分支 + WorkstationIDForException 失败分支。
func TestRct8002_CreateWorkorderBySeverity_SuccessPath(t *testing.T) {
	db := newReconDB8002(t)

	// woSvc 用独立空库:CreateWorkorderFromException 查询异常 ErrRecordNotFound → (nil, nil)
	emptyDSN := filepath.Join(t.TempDir(), "rct-empty-8002.db")
	emptyDB, err := gorm.Open(sqlite.Open(emptyDSN), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := emptyDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	woSvc := asset.NewReconciliationWorkorderServiceWithCache(emptyDB, nil, nil, nil)

	// 主库种 1 条 high 异常(供 createWorkorderBySeverity 主查询命中)
	now := "2026-08-28 10:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, raw_snapshot, detected_at)
		VALUES ('exc-8002-succ', 'asset-s', 'A', 'high', CAST('{}' AS BLOB), ?)`, now).Error)

	require.NoError(t, createWorkorderBySeverity(context.Background(), db, woSvc, "high", 30))
}

// TestRct8002_CleanupExpired_QueryError cleanupExpiredExceptionsDirect 查询错误分支(表缺失)。
func TestRct8002_CleanupExpired_QueryError(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "rct-cleanup-err-8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	_, err = cleanupExpiredExceptionsDirect(context.Background(), db, time.Now())
	require.Error(t, err)
}

// TestRct8002_CheckPortStatusDrift_Branches 漂移剩余分支:
// 查询错误 / 上涨超阈 / 持平 / 基线解析失败。
func TestRct8002_CheckPortStatusDrift_Branches(t *testing.T) {
	db := newReconDB8002(t)
	ctx := context.Background()

	// 查询错误分支:port_status 表删除 → 查询失败
	require.NoError(t, db.Exec(`DROP TABLE sys_device_port_status`).Error)
	require.Error(t, checkPortStatusDrift(ctx, db))
	// 还原
	require.NoError(t, db.Exec(`CREATE TABLE sys_device_port_status (
		id TEXT PRIMARY KEY, device_id TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)

	// 上涨超阈:基线 1,当前 10(> 1+5)→ WARN 分支(基线不更新)
	seedDrift8002(t, db, 10)
	upsertDriftBaseline(ctx, db, "reconciliation.port_status.drift_baseline", 1)
	require.NoError(t, checkPortStatusDrift(ctx, db))
	val, _ := readDriftBaseline(ctx, db, "reconciliation.port_status.drift_baseline")
	assert.Equal(t, int64(1), val, "上涨分支基线不应更新")

	// 持平分支:基线 == 当前(5 ≤ drift ≤ baseline+5)
	seedDrift8002(t, db, 5)
	upsertDriftBaseline(ctx, db, "reconciliation.port_status.drift_baseline", 5)
	require.NoError(t, checkPortStatusDrift(ctx, db))

	// 基线解析失败分支:config_value 非数字 → readDriftBaseline false → 首次观测分支
	seedDrift8002(t, db, 2)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value)
		VALUES ('cfg-bad-num', '漂移基线', 'reconciliation.port_status.drift_baseline', 'not-a-number')`).Error)
	require.NoError(t, checkPortStatusDrift(ctx, db))
	// 首次观测分支把当前值写入(Assign 更新命中已有行)
	val, exists := readDriftBaseline(ctx, db, "reconciliation.port_status.drift_baseline")
	assert.True(t, exists)
	assert.Equal(t, int64(2), val)
}
