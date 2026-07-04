package asset

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestClassifyType 表驱动测试所有 6 个分类结果
func TestClassifyType(t *testing.T) {
	svc := &reconciliationDetectionImpl{}
	tests := []struct {
		name string
		sig  ConflictSignals
		want string
	}{
		{"physical+declared+AD all match", ConflictSignals{HasPhysical: true, HasDeclared: true, HasAD: true, PhysicalMatchDeclared: true, PhysicalMatchAD: true}, "A"},
		{"physical+declared match but AD diff", ConflictSignals{HasPhysical: true, HasDeclared: true, HasAD: true, PhysicalMatchDeclared: true, PhysicalMatchAD: false}, "F"},
		{"physical+declared no AD", ConflictSignals{HasPhysical: true, HasDeclared: true, HasAD: false, PhysicalMatchDeclared: true}, "A"},
		{"B: physical only", ConflictSignals{HasPhysical: true, HasDeclared: false, HasAD: false}, "B"},
		{"C: physical+declared mismatch", ConflictSignals{HasPhysical: true, HasDeclared: true, HasAD: false, PhysicalMatchDeclared: false}, "C"},
		{"D: declared only", ConflictSignals{HasPhysical: false, HasDeclared: true, HasAD: false}, "D"},
		{"E: nothing", ConflictSignals{HasPhysical: false, HasDeclared: false, HasAD: false}, "E"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, svc.ClassifyType(tt.sig))
		})
	}
}

// TestComputeConfidence 置信度公式
func TestComputeConfidence(t *testing.T) {
	svc := &reconciliationDetectionImpl{}
	tests := []struct {
		name string
		sig  ConflictSignals
		want float64
	}{
		{"all match", ConflictSignals{HasDeclared: true, PhysicalMatchDeclared: true, HasAD: true, PhysicalMatchAD: true}, 1.0},
		{"physical+declared match, no AD", ConflictSignals{HasDeclared: true, PhysicalMatchDeclared: true}, 0.8},
		{"physical only", ConflictSignals{HasPhysical: true}, 0.0},
		{"nothing", ConflictSignals{}, 0.0},
		{"declared only", ConflictSignals{HasDeclared: true}, 0.3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, svc.ComputeConfidence(tt.sig))
		})
	}
}

// TestComputeSeverity severity 映射
func TestComputeSeverity(t *testing.T) {
	svc := &reconciliationDetectionImpl{}
	tests := []struct {
		typ  string
		want string
	}{
		{"B", "high"},
		{"C", "high"},
		{"D", "medium"},
		{"E", "low"},
		{"F", "medium"},
		{"A", "low"}, // 不入主表,但函数应不报错
		{"X", "low"},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			require.Equal(t, tt.want, svc.ComputeSeverity(tt.typ))
		})
	}
}

// TestDetectLayer3_TypeA_NotInserted D-09 健康资产不入主表
func TestDetectLayer3_TypeA_NotInserted(t *testing.T) {
	db := setupTestDB(t, "test_type_a")
	svc := &reconciliationDetectionImpl{db: db}

	uid := "00000000-0000-0000-0000-000000000001"
	// physical = uid, declared = uid, AD = uid → 全匹配,期望 Type A
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, user_id, status, deleted_at) VALUES (?, ?, ?, 0, NULL)`, uid, "HEALTHY-001", uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'alice')`, uid, uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_ad (asset_id, ad_id, ad_username, ad_is_enabled) VALUES (?, ?, 'alice', 1)`, uid, uid).Error)

	inserted, skipped, skippedSilence, skippedThrottle, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, inserted, "Type A 不应插入 sys_data_reconciliation")
	require.Equal(t, 1, skipped, "Type A 行计入 skipped")
	require.Equal(t, 0, skippedSilence, "Type A 不应命中静默期")
	require.Equal(t, 0, skippedThrottle, "Type A 不应命中节流")

	var count int64
	require.NoError(t, db.Table("sys_data_reconciliation").Count(&count).Error)
	require.Equal(t, int64(0), count, "D-09: 健康资产不入主表")
}

// TestDetectLayer3_DuplicateViolation_Skipped D-11 unique violation 静默跳过
//
// R2 更新:DetectLayer3 在 INSERT 前先走 24h 节流 guard(D-A3-02),命中后计入
// skippedThrottle 而不是抛 unique violation。这意味着 D-11 的 unique violation
// catch 路径在 R2 仅作为"没有 24h 内记录但碰巧唯一索引冲突"的兜底(例如并发 cron)。
// 本测试保留"不重复插入"语义,但验证计数走 skippedThrottle。
func TestDetectLayer3_DuplicateViolation_Skipped(t *testing.T) {
	db := setupTestDB(t, "test_duplicate")
	svc := &reconciliationDetectionImpl{db: db}

	uid := "00000000-0000-0000-0000-000000000002"
	now := time.Now()
	// physical 有 (uid), 责任人无 (user_id=NULL) → Type B
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, user_id, status, deleted_at) VALUES (?, ?, NULL, 0, NULL)`, uid, "BSET-001").Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'bob')`, uid, uid).Error)

	// 预插入一条 (uid, "B") 异常(模拟上次检测已写入,detected_at=NOW() 命中 24h 节流窗口)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, confidence_score, raw_snapshot, detected_at) VALUES (?, ?, 'B', 'high', 0.5, '{}', ?)`,
		"11111111-1111-1111-1111-111111111111", uid, now).Error)

	inserted, _, _, skippedThrottle, err := svc.DetectLayer3(context.Background())
	require.NoError(t, err, "duplicate 不应抛错,仅静默跳过")
	require.Equal(t, 0, inserted, "已存在则不重复插入")
	require.GreaterOrEqual(t, skippedThrottle, 1, "R2: 24h 节流命中应计入 skippedThrottle")

	var count int64
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ?", uid).Count(&count).Error)
	require.Equal(t, int64(1), count, "D-11: 只应存在 1 条")
}

// setupTestDB 构造测试用 SQLite in-memory DB,建 reconciliation_normalized view + sys_data_reconciliation 表
//
// 用 unique cache 名(testName)避免 shared memory 跨测试串扰。
func setupTestDB(t *testing.T, testName string) *gorm.DB {
	t.Helper()
	dsn := "file:" + testName + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// 建 sys_data_reconciliation 表(SQLite 子集;含 BaseModel 字段 created_by/updated_by/version)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			asset_id TEXT NOT NULL,
			conflict_type TEXT NOT NULL,
			recon_category TEXT,
			severity TEXT NOT NULL,
			physical_value TEXT,
			declared_value TEXT,
			ad_value TEXT,
			confidence_score REAL,
			raw_snapshot TEXT NOT NULL,
			asset_ip TEXT,
			exception_rule_id TEXT,
			applied_actions TEXT,
			detected_at DATETIME NOT NULL,
			resolved_at DATETIME,
			resolved_by TEXT,
			resolution_note TEXT,
			workorder_id TEXT
		)
	`).Error)

	// 三列 partial unique index (模拟 migration_201 的 uniq_recon_asset_type_cat_open;
	// b8fd2f45 切换: 旧两列 uniq_recon_asset_type_open 已 DROP)。SQLite 3.8+ 支持 partial index。
	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS uniq_recon_asset_type_cat_open
			ON sys_data_reconciliation (asset_id, conflict_type, recon_category)
			WHERE resolved_at IS NULL AND deleted_at IS NULL
	`).Error)

	// 建 ops_asset 依赖(SQLite)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			mac1 TEXT,
			mac2 TEXT,
			machine_ip TEXT,
			user_id TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME
		)
	`).Error)

	// 建 sys_user 依赖
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			deleted_at DATETIME
		)
	`).Error)

	// 建 sys_ad_user 依赖
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_ad_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			is_enabled INTEGER DEFAULT 0,
			deleted_at DATETIME
		)
	`).Error)

	// 建 ops_asset_physical 测试用(物理链路表)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS ops_asset_physical (
			asset_id TEXT PRIMARY KEY,
			physical_user_id TEXT,
			physical_username TEXT
		)
	`).Error)

	// 建 ops_asset_ad 测试用(AD 关联表)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS ops_asset_ad (
			asset_id TEXT PRIMARY KEY,
			ad_id TEXT,
			ad_username TEXT,
			ad_is_enabled INTEGER
		)
	`).Error)

	// R3: 建 sys_reconciliation_exception 表(DetectLayer3 Layer 3.5 预加载 active 规则)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_reconciliation_exception (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
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
		)
	`).Error)

	// 建 reconciliation_normalized view(SQLite 不支持 MV,用 view 模拟)
	// DROP VIEW IF EXISTS 兼容 SQLite,允许重跑测试
	// R2 扩展:last_resolved_at / last_resolved_by / last_conflict_type 三字段(NULL 占位,
	// 因 SQLite 测试无真实 sys_data_reconciliation 数据,这两条 guard 永远不命中)
	require.NoError(t, db.Exec(`DROP VIEW IF EXISTS reconciliation_normalized`).Error)
	require.NoError(t, db.Exec(`
		CREATE VIEW reconciliation_normalized AS
		SELECT
			a.id AS asset_id,
			a.devicesn AS asset_code,
			a.machine_ip AS asset_ip,
			p.physical_user_id AS physical_user_id,
			p.physical_username AS physical_username,
			a.user_id AS asset_user_id,
			NULL AS asset_username,
			ad.ad_id AS ad_id,
			ad.ad_username AS ad_username,
			ad.ad_is_enabled AS ad_is_enabled,
			NULL AS mv_refreshed_at,
			a.mac1 AS mac1,
			a.mac2 AS mac2,
			NULL AS mac_join,
			NULL AS last_resolved_at,
			NULL AS last_resolved_by,
			NULL AS last_conflict_type
		FROM ops_asset a
		LEFT JOIN ops_asset_physical p ON p.asset_id = a.id
		LEFT JOIN ops_asset_ad ad ON ad.asset_id = a.id
	`).Error)

	// 旧两列非 partial unique index (uniq_recon_asset_type_open) 已废弃:
	// migration_201 (b8fd2f45) DROP 它换三列 partial (uniq_recon_asset_type_cat_open, 见上方建表后)。
	// 测试基础设施同步, 此处不再建两列 index。
	// TestDetectLayer3_DuplicateViolation_Skipped 靠 24h 节流 guard (非 unique index) 防重复。

	// 抑制 unused import 警告
	_ = sql.ErrNoRows

	return db
}