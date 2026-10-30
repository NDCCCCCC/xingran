package db

// Phase 84 PMODEL-04: sys_schema_migrations 迁移记录表的单测（sqlite 内存库）。
//
// 方向锁定：
//   - TestRunAndRecord_NonBlocking：记录失败只 WARN 不阻断 —— 迁移错误仍返回、
//     启动路径不被记录错误打断（防止有人把 WARN 改成吞错或改回阻断）；
//   - 不做 skip 控制执行：记录表内容不参与任何迁移是否运行的判断。

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
)

// countMigrationRecords 统计 sys_schema_migrations 行数。
func countMigrationRecords(t *testing.T, d *Database) int64 {
	t.Helper()
	var n int64
	require.NoError(t, d.DB.Raw(`SELECT count(*) FROM sys_schema_migrations`).Scan(&n).Error)
	return n
}

// TestMigrate225 建表 + seed：跑 Migrate225 后表存在、含 225 记录（seed 历史 + 225 自身
// 由 runAndRecord 记录——此处直接补插以模拟调用块行为）。
func TestMigrate225(t *testing.T) {
	d := newBackfillTestDB(t)

	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	assert.True(t, d.DB.Migrator().HasTable("sys_schema_migrations"), "sys_schema_migrations 必须建出")

	// seed 覆盖全部活跃版本（除 225 自身）
	var versions []int
	require.NoError(t, d.DB.Raw(`SELECT version FROM sys_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Contains(t, versions, 175, "历史活跃版本应被 seed")
	assert.Contains(t, versions, 224, "最新历史版本应被 seed")
	assert.NotContains(t, versions, 225, "225 由调用块 runAndRecord 记录，seed 不含自身")

	// 幂等：重跑一次记录数不变（CREATE IF NOT EXISTS + INSERT 冲突忽略）
	before := countMigrationRecords(t, d)
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	assert.Equal(t, before, countMigrationRecords(t, d), "Migrate225 必须幂等可重跑")

	// 模拟调用块：225 自身经 runAndRecord 落记录
	require.NoError(t, d.runAndRecord(225, "schema_migrations_table", func(db *gorm.DB) error { return nil }))
	var v225 int64
	require.NoError(t, d.DB.Raw(`SELECT count(*) FROM sys_schema_migrations WHERE version = 225`).Scan(&v225).Error)
	assert.Equal(t, int64(1), v225, "225 记录应存在且唯一")
}

// TestRunAndRecord_Idempotent 同一迁移跑两遍 → 记录数不变（幂等 upsert）。
func TestRunAndRecord_Idempotent(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))

	fn := func(db *gorm.DB) error { return nil }
	before := countMigrationRecords(t, d) // Migrate225 已 seed 25 条历史版本（seed 不含 225 自身）
	require.NoError(t, d.runAndRecord(225, "schema_migrations_table", fn))
	afterFirst := countMigrationRecords(t, d)
	require.Equal(t, before+1, afterFirst, "首次记录应恰好新增一行")

	require.NoError(t, d.runAndRecord(225, "schema_migrations_table", fn))
	assert.Equal(t, afterFirst, countMigrationRecords(t, d), "重复执行不得产生第二行（幂等 upsert）")
}

// TestRunAndRecord_NonBlocking 方向锁定：注入会失败的 fn → 迁移错误仍返回、
// 记录未写入（WARN 路径可达）、启动路径不被记录错误阻断。
func TestRunAndRecord_NonBlocking(t *testing.T) {
	d := newBackfillTestDB(t)
	// 故意先不跑 Migrate225：记录表不存在 → 记录必然走 WARN 分支。
	// 即使表存在，下面注入的迁移失败也不应留下记录。

	migrateErr := errors.New("迁移本身失败")
	fn := func(db *gorm.DB) error { return migrateErr }

	err := d.runAndRecord(216, "ad_push_drain_sys_config", fn)
	require.Error(t, err, "迁移失败必须原样返回错误（绝不吞错）")
	assert.ErrorIs(t, err, migrateErr)

	var n int64
	// 表不存在时查询会报错 —— 属预期：此时记录自然失败且只 WARN
	if d.DB.Migrator().HasTable("sys_schema_migrations") {
		require.NoError(t, d.DB.Raw(`SELECT count(*) FROM sys_schema_migrations WHERE version = 216`).Scan(&n).Error)
		assert.Equal(t, int64(0), n, "迁移失败的版本不得留下记录")
	}

	// 迁移成功 + 表存在 → 记录成功；迁移成功 + 表缺失 → 仍返回 nil（记录失败只 WARN）
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	okFn := func(db *gorm.DB) error { return nil }
	require.NoError(t, d.runAndRecord(216, "ad_push_drain_sys_config", okFn), "迁移成功时记录成功，返回 nil")
	require.NoError(t, d.DB.Migrator().DropTable("sys_schema_migrations"))
	require.NoError(t, d.runAndRecord(217, "dept_ou_mismatch_view", okFn),
		"记录表缺失时 runAndRecord 必须返回 nil —— 记录失败只 WARN，绝不阻断启动路径")
}

// TestReconcile 只读对账：人工删一条记录后 MissingRecords 报该版本 missing；
// 插入 registry 之外的版本报 extra。
func TestReconcile(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	require.NoError(t, d.runAndRecord(225, "schema_migrations_table", func(db *gorm.DB) error { return nil }))

	// 全量记录时：missing 只应缺 PG-only 版本（sqlite 库上 175/176/202-206/211/214 由 seed
	// 补齐 —— seed 双方言统一插入，故此处 missing/extra 均应为空）
	res, err := migrations.MissingRecords(context.Background(), d.DB)
	require.NoError(t, err)
	assert.Empty(t, res.Missing, "全量 seed + 全量 runAndRecord 后不应有 missing（seed 含 PG-only 版本）")
	assert.Empty(t, res.Extra)

	// 人工删 223 → missing 应报 223
	require.NoError(t, d.DB.Exec(`DELETE FROM sys_schema_migrations WHERE version = 223`).Error)
	res, err = migrations.MissingRecords(context.Background(), d.DB)
	require.NoError(t, err)
	assert.Equal(t, []int{223}, res.Missing, "被删除的版本应报 missing")
	assert.Empty(t, res.Extra)

	// 插入 registry 外版本 999 → extra 应报 999
	require.NoError(t, d.DB.Exec(`INSERT INTO sys_schema_migrations (version, name, applied_at) VALUES (999, 'ghost', CURRENT_TIMESTAMP)`).Error)
	res, err = migrations.MissingRecords(context.Background(), d.DB)
	require.NoError(t, err)
	assert.Equal(t, []int{223}, res.Missing)
	assert.Contains(t, res.Extra, 999, "registry 之外的版本应报 extra")

	// 表不存在 → 明确报错（提示先跑 migration_225）
	require.NoError(t, d.DB.Migrator().DropTable("sys_schema_migrations"))
	_, err = migrations.MissingRecords(context.Background(), d.DB)
	require.Error(t, err, "表缺失时对账必须报错而非静默返回空")
}
