package asset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
)

// baseListRequestDefault 测试辅助:返回 current=1 pageSize=20 的默认分页
func baseListRequestDefault() base.BaseListRequest {
	return base.BaseListRequest{Current: 1, PageSize: 20}
}

// ============================================================================
// ListExceptions silence 默认过滤测试(D-R3-A1-01 / SC 7)
// ============================================================================

func setupReconServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:recsvc_"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 复用 detection_test 的 setupTestDB schema(直接调)
	setupTestDBOnDB(t, db)
	return db
}

// setupTestDBOnDB 在已打开的 gorm.DB 上建表(供多个 test 共用 schema)
func setupTestDBOnDB(t *testing.T, db *gorm.DB) {
	t.Helper()
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
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`DROP VIEW IF EXISTS reconciliation_normalized`).Error)
	require.NoError(t, db.Exec(`
		CREATE VIEW reconciliation_normalized AS
		SELECT
			a.id AS asset_id,
			a.devicesn AS asset_code,
			a.machine_ip AS asset_ip,
			NULL AS physical_user_id,
			NULL AS physical_username,
			a.user_id AS asset_user_id,
			NULL AS asset_username,
			NULL AS ad_id,
			NULL AS ad_username,
			NULL AS ad_is_enabled,
			NULL AS mv_refreshed_at,
			NULL AS mac1,
			NULL AS mac2,
			NULL AS mac_join,
			NULL AS last_resolved_at,
			NULL AS last_resolved_by,
			NULL AS last_conflict_type
		FROM ops_asset a
	`).Error)
}

func TestListExceptionsSilenceFilter(t *testing.T) {
	db := setupReconServiceTestDB(t, "silence_filter")
	svc := &reconciliationServiceImpl{db: db, mvExists: 1} // mvExists=1 跳过探测

	now := time.Now()
	// 2 条异常:1 条 silence,1 条 nil
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, confidence_score, raw_snapshot, asset_ip, applied_actions, detected_at, deleted_at)
		VALUES
		('sil-1', 'asset-sil', 'B', 'high', 0.5, '{}', '10.0.0.1', '{silence}', ?, NULL),
		('nil-1', 'asset-nil', 'B', 'high', 0.5, '{}', '10.0.0.2', NULL, ?, NULL)`, now, now).Error)

	// 默认调用(ShowSilenced=false):仅返回 nil-1
	result, err := svc.ListExceptions(context.Background(), &ExceptionListParams{
		BaseListRequest: baseListRequestDefault(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "默认应过滤 silence 记录,返回 1 条")
}

func TestListExceptionsShowSilenced(t *testing.T) {
	db := setupReconServiceTestDB(t, "show_silenced")
	svc := &reconciliationServiceImpl{db: db, mvExists: 1}

	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation
		(id, asset_id, conflict_type, severity, confidence_score, raw_snapshot, asset_ip, applied_actions, detected_at, deleted_at)
		VALUES
		('sil-2', 'asset-sil2', 'B', 'high', 0.5, '{}', '10.0.0.1', '{silence}', ?, NULL),
		('nil-2', 'asset-nil2', 'B', 'high', 0.5, '{}', '10.0.0.2', NULL, ?, NULL)`, now, now).Error)

	// ShowSilenced=true:返回 2 条
	result, err := svc.ListExceptions(context.Background(), &ExceptionListParams{
		BaseListRequest: baseListRequestDefault(),
		ShowSilenced:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "ShowSilenced=true 时返回全部 2 条")
}
