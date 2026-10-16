package asset

// ============================================================================
// 回归测试 — debug session reconciliation-sqlite-cast-400 (2026-08-18)
//
// 根因(两层) — sqlite 运行期模式下 /asset/reconciliation/exception/list 400:
//  1. ListExceptions 的 SELECT/JOIN 片段硬编码 PG 专有 ::cast
//     (COALESCE(a.machine_ip::text,'') / ru.id = a.user_id::uuid / ''::text)
//     → SQLite 词法错误 "SQL logic error: unrecognized token: \":\""。
//  2. probeMaterializedView() 对非 postgres 硬编码 return true(注释假设
//     "SQLite 测试用 view 模拟" — 单测 setupTestDB 建 view 成立,运行期文件库
//     不成立:PG-only 迁移 168/176 在 sqlite 分支不执行,reconciliation_normalized
//     不存在)→ 即使修好 cast,MV 路径 JOIN 仍会报 "no such table"。
//
// 本文件模拟运行期两种形态:
//   - runtime-like: 表齐全但无 reconciliation_normalized 视图 → 必须走
//     设计内 MV-missing fallback 路径成功(fallback SELECT 无 rn.* 列)
//   - view present: 单测形态(setupTestDB 建 view)/运行期 sqlite bootstrap 补建视图
//     (database.go ensureSQLiteReconciliationViews,sqlite-recon-normalized-view)
//     → MV 路径在 sqlite 上也必须成功
//
// 修复前两个 ListExceptions 测试均以 `unrecognized token: ":"` 失败(复现 400)。
// ============================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedRuntimeListRows 造最小数据链:sys_user(u1) ← ops_asset(a1, user_id=u1)
// ← sys_data_reconciliation(r1, asset_id=a1);withAD 时补 ops_asset_ad 供视图取 ad_username。
//
// raw_snapshot 用 CAST(... AS BLOB) 写入:GORM json.RawMessage 走 Valuer 返回
// []byte,sqlite 以 BLOB 存储类落库,读回 []byte 可 Scan 进 RawMessage;若以
// TEXT 字符串字面量写入,driver 返回 string,会触发 database/sql 的
// "unsupported Scan, storing driver.Value type string into type *json.RawMessage"
// (种子形态问题,非业务 bug)。
func seedRuntimeListRows(t *testing.T, db *gorm.DB, withAD bool) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES (?, ?)`, "u1", "zhangsan").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id) VALUES (?, ?, ?, ?)`,
		"a1", "SN001", "10.0.0.1", "u1").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at, created_at)
		 VALUES (?, ?, ?, ?, CAST(? AS BLOB), ?, ?)`,
		"r1", "a1", "B", "high", "{}", now, now).Error)
	if withAD {
		require.NoError(t, db.Exec(
			`INSERT INTO ops_asset_ad (asset_id, ad_id, ad_username, ad_is_enabled) VALUES (?, ?, ?, ?)`,
			"a1", "ad1", "zhangsan@ad", 1).Error)
	}
}

// TestListExceptionsSQLiteRuntimeNoView 运行期同构库(无视图)必须成功(走 fallback)
func TestListExceptionsSQLiteRuntimeNoView(t *testing.T) {
	db := setupTestDB(t, t.Name()+"_noview")
	// 模拟运行期 sqlite 文件库:reconciliation_normalized 视图不存在
	require.NoError(t, db.Exec(`DROP VIEW IF EXISTS reconciliation_normalized`).Error)
	seedRuntimeListRows(t, db, false)

	svc := NewReconciliationService(db, nil, nil)
	result, err := svc.ListExceptions(context.Background(), &ExceptionListParams{})
	require.NoError(t, err,
		"运行期 sqlite(无 MV 视图)ListExceptions 必须走 fallback 路径成功,不得因 ::cast / 缺视图 400")

	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	list, ok := result.List.([]ExceptionListItem)
	require.True(t, ok, "List 应为 []ExceptionListItem,实际 %T", result.List)
	require.Len(t, list, 1)
	assert.Equal(t, "SN001", list[0].AssetCode, "asset_code 来自 a.devicesn JOIN")
	assert.Equal(t, "10.0.0.1", list[0].AssetIPDisplay, "asset_ip 来自 a.machine_ip JOIN")
	require.NotNil(t, list[0].ResponsibleUsername, "responsible_username 来自 sys_user JOIN")
	assert.Equal(t, "zhangsan", *list[0].ResponsibleUsername)
}

// TestProbeMaterializedViewSQLiteHonest probe 在 sqlite 下必须诚实探测 sqlite_master,
// 不得无条件 return true(否则缺视图时误走 MV 路径 → no such table)。
func TestProbeMaterializedViewSQLiteHonest(t *testing.T) {
	dbNoView := setupTestDB(t, t.Name()+"_noview")
	require.NoError(t, dbNoView.Exec(`DROP VIEW IF EXISTS reconciliation_normalized`).Error)
	svcNoView := NewReconciliationService(dbNoView, nil, nil).(*reconciliationServiceImpl)
	assert.False(t, svcNoView.mvAvailable(),
		"sqlite 运行期无视图时 probe 必须返回 false(触发设计内 fallback 降级)")

	dbView := setupTestDB(t, t.Name()+"_withview")
	svcView := NewReconciliationService(dbView, nil, nil).(*reconciliationServiceImpl)
	assert.True(t, svcView.mvAvailable(), "sqlite 有视图(单测形态)probe 必须返回 true")
}

// TestListExceptionsSQLiteWithView 有视图(单测形态)MV 路径在 sqlite 上必须成功
func TestListExceptionsSQLiteWithView(t *testing.T) {
	db := setupTestDB(t, t.Name()+"_withview")
	seedRuntimeListRows(t, db, true)

	svc := NewReconciliationService(db, nil, nil)
	result, err := svc.ListExceptions(context.Background(), &ExceptionListParams{})
	require.NoError(t, err,
		"sqlite 有视图时 MV 路径必须成功,不得因 PG 专有 ::cast 词法错误 400")

	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	list, ok := result.List.([]ExceptionListItem)
	require.True(t, ok)
	require.Len(t, list, 1)
	assert.Equal(t, "SN001", list[0].AssetCode)
	require.NotNil(t, list[0].AdUsername, "ad_username 来自 reconciliation_normalized 视图")
	assert.Equal(t, "zhangsan@ad", *list[0].AdUsername)
}
