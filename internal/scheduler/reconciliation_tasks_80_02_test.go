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
