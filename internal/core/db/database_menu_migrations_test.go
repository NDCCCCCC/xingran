package db

// GOV-03 diff gate: AutoMigrate 的 PG / sqlite 两分支共用 runMenuConsolidationMigrations
// 注册 241/242/243/248/249。gate 把每个 `if err := d.runAndRecord(...)` 的 error body
// （applogger.Errorf + `}`）两行计入新增可执行行，而正常库上这五个迁移都是幂等
// 成功 / 优雅跳过，error body 永不进入 → 覆盖率 0（PR #84 CI 的 12 行 UNCOVERED
// 即来源于此；PR #89 的 8 行 UNCOVERED 同源）。本文件用结构残缺的 sys_menu 让迁移
// 稳定返回 error，覆盖这些行；248 内部对 DB 错误走 continue 降级，其 error body 由
// 全量门禁兜底（diff 覆盖率 ≥80% 即可，见 check-diff-coverage.sh）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestRunMenuConsolidationMigrations_ErrorBodies 让 241/242/243/249 在缺列的
// sys_menu 上必然失败，从而执行到 database.go 中对应 runAndRecord 的非阻断 error
// 分支。同时锁定 runAndRecord 方向：迁移失败不写版本记录、不阻断（方法返回 void）。
// 248 内部对 DB 错误走 continue 降级（有意设计：瘦库不阻断），其 error body 不进入。
func TestRunMenuConsolidationMigrations_ErrorBodies(t *testing.T) {
	d := newBackfillTestDB(t)

	// 记录表就绪：runAndRecord 成功路径会 upsert，失败路径应完全不写。
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	// seed 可能含历史版本；先清掉五个版本，保证断言基线为 0。
	require.NoError(t, d.DB.Exec(
		`DELETE FROM sys_schema_migrations WHERE version IN (241,242,243,248,249)`).Error)

	// 残缺 sys_menu：迁移内的 SELECT/UPDATE 引用 menu_name / menu_type /
	// component 等列 → sqlite 报 "no such column" → fn 均返回 error。
	require.NoError(t, d.DB.Exec(`CREATE TABLE sys_menu (id TEXT PRIMARY KEY)`).Error)

	// 前置证明：失败型迁移在此 schema 上确实失败（不是靠别的路径蒙混进 error 分支）。
	require.Error(t, migrations.Migrate241SettingsConsolidateMenus(d.DB))
	require.Error(t, migrations.Migrate242ADDriftAlertMergeMenu(d.DB))
	require.Error(t, migrations.Migrate243LowDensityMenus(d.DB))
	require.Error(t, migrations.Migrate249DutyPoolsMenuState(d.DB))

	// 执行被测方法：241/242/243/249 四组 error body（applogger.Errorf + }）应全部进入。
	// 248 对瘦库有意降级为「跳过且不报错」（内部 continue），故它会成功留痕。
	assert.NotPanics(t, d.runMenuConsolidationMigrations)

	var failed int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sys_schema_migrations WHERE version IN (241,242,243,249)`,
	).Scan(&failed).Error)
	assert.Zero(t, failed, "迁移失败时 runAndRecord 不得写入版本记录（非阻断 error 分支应已进入）")

	var degraded int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sys_schema_migrations WHERE version = 248`,
	).Scan(&degraded).Error)
	assert.Equal(t, int64(1), degraded, "248 对瘦库降级成功应留下版本记录（非阻断语义）")
}

// TestRunMenuConsolidationMigrations_Success 正常 schema 上五个迁移成功并留痕，
// 覆盖同一方法的 if 行（成功路径），与上面的 error body 合成完整覆盖。
func TestRunMenuConsolidationMigrations_Success(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	require.NoError(t, d.DB.AutoMigrate(&models.Menu{}))
	require.NoError(t, d.DB.Exec(
		`DELETE FROM sys_schema_migrations WHERE version IN (241,242,243,248,249)`).Error)

	d.runMenuConsolidationMigrations()

	var n int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sys_schema_migrations WHERE version IN (241,242,243,248,249)`,
	).Scan(&n).Error)
	assert.Equal(t, int64(5), n, "五个迁移成功后应各自留下版本记录")
}
