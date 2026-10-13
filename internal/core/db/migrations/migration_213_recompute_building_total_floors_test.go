package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// 261013-lqb Task 3: Migrate213 存量重算迁移单测。
// 业务表用裸 SQL 建最小表（excel_import_rest_77_03_test.go 先例，避免
// AutoMigrate 耦合）；锁定语义：软删楼层不计入 + 脏值归正 + 二次执行幂等。
// =====================================================================

// freshSQLiteDBForMigrate213 内存 SQLite + 裸 SQL 最小表。
func freshSQLiteDBForMigrate213(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_buildings (
			id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME, total_floors INTEGER DEFAULT 0
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_floors (
			id TEXT PRIMARY KEY, building_id TEXT, deleted_at DATETIME
		)`).Error)
	return db
}

// migrate213BuildingTotalFloors 读取楼栋当前 total_floors。
func migrate213BuildingTotalFloors(t *testing.T, db *gorm.DB, id string) int {
	t.Helper()
	var got int
	require.NoError(t, db.Raw(`SELECT total_floors FROM ops_buildings WHERE id = ?`, id).Scan(&got).Error)
	return got
}

// TestMigrate213RecomputeBuildingTotalFloors 脏值归正 + 幂等。
func TestMigrate213RecomputeBuildingTotalFloors(t *testing.T) {
	db := freshSQLiteDBForMigrate213(t)

	// B1 挂 2 行楼层（1 软删）→ 活行数 1；B2 挂 1 层 → 活行数 1；脏值 99/0
	for _, s := range []string{
		`INSERT INTO ops_buildings (id, name, total_floors) VALUES ('b1', 'LQB-B1', 99)`,
		`INSERT INTO ops_buildings (id, name, total_floors) VALUES ('b2', 'LQB-B2', 0)`,
		`INSERT INTO ops_floors (id, building_id) VALUES ('f1', 'b1')`,
		`INSERT INTO ops_floors (id, building_id, deleted_at) VALUES ('f2', 'b1', '2026-01-01 00:00:00')`,
		`INSERT INTO ops_floors (id, building_id) VALUES ('f3', 'b2')`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}

	// 第一次执行
	require.NoError(t, Migrate213RecomputeBuildingTotalFloors(db))
	require.Equal(t, 1, migrate213BuildingTotalFloors(t, db, "b1"), "B1 软删 1 层后应归正为 1")
	require.Equal(t, 1, migrate213BuildingTotalFloors(t, db, "b2"), "B2 应归正为 1")

	// 第二次执行（幂等：结果不变）
	require.NoError(t, Migrate213RecomputeBuildingTotalFloors(db))
	require.Equal(t, 1, migrate213BuildingTotalFloors(t, db, "b1"), "二次执行幂等")
	require.Equal(t, 1, migrate213BuildingTotalFloors(t, db, "b2"), "二次执行幂等")
}
