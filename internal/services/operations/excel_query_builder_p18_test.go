package operations

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// v1.52 P1-8 回归守护：WorkstationQueryBuilder.BuildQuery 的 7 处 PG 专有
// ::uuid/::text 已改为标准 SQL CAST(... AS TEXT)，必须与生产 List 的
// workstationJoinClause(workstation_service.go:18-22) 同款双方言写法。
//
// 用 DryRun 拿 SQL 字符串做静态断言：不执行查询，因此不依赖
// ops_floors/ops_buildings 等表是否存在（既有 setupWSD77DB 夹具缺这两表，
// 直接执行 BuildQuery 只会得到「no such table」而测不到语法层）。
func TestP18_WorkstationQueryBuilder_NoPostgresOnlyCast(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p18.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		DryRun: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	cfg, ok := GetExportConfig("workstation")
	require.True(t, ok, "workstation 导出配置必须存在")

	q := NewWorkstationQueryBuilder().BuildQuery(
		context.Background(), db, cfg, map[string]interface{}{"name": "3f130"},
	)
	stmt := q.Find(&[]map[string]interface{}{}).Statement
	sql := stmt.SQL.String()

	// 1) 不得再出现 PG 专有 :: 强转（SQLite 会报 unrecognized token ':'）。
	assert.NotContains(t, sql, "::uuid", "不得残留 PG 专有 ::uuid")
	assert.NotContains(t, sql, "::text", "不得残留 PG 专有 ::text")
	assert.NotContains(t, sql, "::", "不得残留任何 :: 强转")

	// 2) 四个 JOIN 与三个设备子查询都必须用双方言 CAST(... AS TEXT)。
	//    统计 CAST 出现次数：4 JOIN ×2 侧 + 3 子查询 ×2 侧 = 14 次，
	//    另加 Phase 131 可见集条件里的 CAST（数量不固定，故只断言下限）。
	castCount := strings.Count(sql, "CAST(")
	assert.GreaterOrEqual(t, castCount, 14,
		"JOIN 与相关子查询必须双侧 CAST(... AS TEXT)，实际 SQL: %s", sql)
	assert.Contains(t, sql, "CAST(ops_floors.id AS TEXT)", "楼层 JOIN 主键侧须 CAST")
	assert.Contains(t, sql, "CAST(ops_buildings.id AS TEXT)", "楼宇 JOIN 主键侧须 CAST")
	assert.Contains(t, sql, "CAST(sys_dept.id AS TEXT)", "部门 JOIN 主键侧须 CAST")
	assert.Contains(t, sql, "CAST(sys_user.id AS TEXT)", "用户 JOIN 主键侧须 CAST")
	assert.Contains(t, sql, "CAST(workstation_id AS TEXT)", "设备子查询外键侧须 CAST")

	// 3) Phase 131 注入的可见集过滤仍在（不得因改 JOIN 而丢失）。
	assert.Contains(t, sql, "deleted_at IS NULL", "软删过滤必须保留")
}

// TestP18_WorkstationQueryBuilder_SQLiteSyntaxValid 验证改完后的 SQL 在 SQLite
// 上至少能过语法解析（用最小表夹具让 4 个 JOIN 与相关子查询的表都存在，
// 只断言「错误里不含 unrecognized token ':'」）。
func TestP18_WorkstationQueryBuilder_SQLiteSyntaxValid(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p18_exec.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	// 最小 schema：所有被 JOIN / 子查询引用的表都要存在（列可少，SELECT 里
	// 未选的列不参与解析；这里只验证语法层，不验证业务结果）。
	for _, ddl := range []string{
		`CREATE TABLE sys_workstation (id TEXT, workstation_name TEXT, workstation_type TEXT,
			status INTEGER, location TEXT, description TEXT, dept_id TEXT, user_id TEXT,
			floor_id TEXT, building_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_floors (id TEXT, name TEXT, org_id TEXT, building_id TEXT)`,
		`CREATE TABLE ops_buildings (id TEXT, name TEXT, org_id TEXT)`,
		`CREATE TABLE sys_dept (id TEXT, dept_name TEXT)`,
		`CREATE TABLE sys_user (id TEXT, nickname TEXT)`,
		`CREATE TABLE ops_workstation_device (id TEXT, workstation_id TEXT, device_serial TEXT,
			device_name TEXT, device_model TEXT, is_primary INTEGER, priority INTEGER,
			created_at DATETIME, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}

	cfg, ok := GetExportConfig("workstation")
	require.True(t, ok)

	q := NewWorkstationQueryBuilder().BuildQuery(
		context.Background(), db, cfg, map[string]interface{}{},
	)
	var rows []map[string]interface{}
	err = q.Find(&rows).Error
	// 允许「没有 scope 注入时的空集」或其它业务错误，但绝不能是解析错误。
	if err != nil {
		assert.NotContains(t, err.Error(), "unrecognized token",
			"SQL 不得含 SQLite 无法解析的 PG 专有语法")
		assert.NotContains(t, err.Error(), "syntax error",
			"SQL 不得有语法错误")
	}
}
