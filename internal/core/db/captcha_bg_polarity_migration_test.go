package db

// GOV-03 diff gate: 251（验证码背景图 status 极性归一）的注册行
// （`if err := d.runAndRecord(251, ...)` 的 error body 两行）在正常库上永不进入
// ——迁移成功，error body 是死路径 → 覆盖率 0（PR #96 的 coverage-diff 失败即来源于此）。
//
// 本文件用「结构残缺的 sys_captcha_background」让 251 稳定返回 error，覆盖这些行；
// 再用正常 schema 覆盖同一方法的成功路径，合成完整覆盖。
//
// 参照 database_menu_migrations_test.go 的同型做法（241~250 五组迁移）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestRunCaptchaBgPolarityMigration_ErrorBody 让 251 在缺 status 列的
// sys_captcha_background 上必然失败（UPDATE 引用不存在的列 → sqlite 报错），
// 从而执行到 database.go 中 runCaptchaBgPolarityMigration 的非阻断 error 分支。
// 同时锁定 runAndRecord 方向：迁移失败不写版本记录、不阻断。
func TestRunCaptchaBgPolarityMigration_ErrorBody(t *testing.T) {
	d := newBackfillTestDB(t)

	// 记录表就绪：成功路径会 upsert，失败路径应完全不写。
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	require.NoError(t, d.DB.Exec(
		`DELETE FROM sys_schema_migrations WHERE version = 251`).Error)

	// 残缺表：只有 id 列，没有 status 列 → 迁移内的 UPDATE 报 "no such column"。
	require.NoError(t, d.DB.Exec(
		`CREATE TABLE sys_captcha_background (id TEXT PRIMARY KEY)`).Error)

	// 前置证明：迁移在此 schema 上确实失败（不是靠别的路径蒙混进 error 分支）。
	require.Error(t, migrations.Migrate251CaptchaBgStatusPolarity(d.DB))

	// 执行被测方法：error body（applogger.Errorf + }）应进入，且不 panic。
	assert.NotPanics(t, d.runCaptchaBgPolarityMigration)

	var n int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sys_schema_migrations WHERE version = 251`,
	).Scan(&n).Error)
	assert.Zero(t, n, "迁移失败时 runAndRecord 不得写入版本记录（非阻断 error 分支应已进入）")
}

// TestRunCaptchaBgPolarityMigration_Success 正常 schema 上 251 成功并留痕，
// 覆盖同一方法的 if 行（成功路径），与上面的 error body 合成完整覆盖。
func TestRunCaptchaBgPolarityMigration_Success(t *testing.T) {
	d := newBackfillTestDB(t)
	require.NoError(t, migrations.Migrate225SchemaMigrationsTable(d.DB))
	require.NoError(t, d.DB.AutoMigrate(&models.CaptchaBackground{}))
	require.NoError(t, d.DB.Exec(
		`DELETE FROM sys_schema_migrations WHERE version = 251`).Error)

	d.runCaptchaBgPolarityMigration()

	var n int64
	require.NoError(t, d.DB.Raw(
		`SELECT count(*) FROM sys_schema_migrations WHERE version = 251`,
	).Scan(&n).Error)
	assert.Equal(t, int64(1), n, "迁移成功后应留下版本记录")
}
