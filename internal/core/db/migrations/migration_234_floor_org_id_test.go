package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 建一张不含 org_id 的存量 ops_floors，模拟迁移前的表。
func setupMigrate234DB(t *testing.T, withColumn bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	col := ""
	if withColumn {
		col = ", org_id TEXT"
	}
	require.NoError(t, db.Exec(
		`CREATE TABLE ops_floors (id TEXT PRIMARY KEY, name TEXT, building_id TEXT` + col + `)`).Error)
	return db
}

// TestMigrate234FloorOrgID_AddsNullableColumn 存量表补列，且不影响已有行。
func TestMigrate234FloorOrgID_AddsNullableColumn(t *testing.T) {
	db := setupMigrate234DB(t, false)
	require.NoError(t, db.Exec(
		`INSERT INTO ops_floors (id, name, building_id) VALUES ('f1', '1楼', 'b1')`).Error)

	require.NoError(t, Migrate234FloorOrgID(db))

	assert.True(t, db.Migrator().HasColumn("ops_floors", "org_id"))
	var orgID *string
	require.NoError(t, db.Raw(`SELECT org_id FROM ops_floors WHERE id = 'f1'`).Scan(&orgID).Error)
	assert.Nil(t, orgID, "存量行 org_id 必须为 NULL（继承楼宇），不得臆造值")
}

// TestMigrate234FloorOrgID_Idempotent 重复执行不报错、不重复加列。
func TestMigrate234FloorOrgID_Idempotent(t *testing.T) {
	db := setupMigrate234DB(t, false)
	require.NoError(t, Migrate234FloorOrgID(db))
	require.NoError(t, Migrate234FloorOrgID(db))
	assert.True(t, db.Migrator().HasColumn("ops_floors", "org_id"))
}

// TestMigrate234FloorOrgID_ExistingColumnNoop 已由 AutoMigrate 建列时跳过。
func TestMigrate234FloorOrgID_ExistingColumnNoop(t *testing.T) {
	db := setupMigrate234DB(t, true)
	require.NoError(t, db.Exec(
		`INSERT INTO ops_floors (id, name, building_id, org_id) VALUES ('f1', '1楼', 'b1', 'org-wh')`).Error)

	require.NoError(t, Migrate234FloorOrgID(db))
	var orgID string
	require.NoError(t, db.Raw(`SELECT org_id FROM ops_floors WHERE id = 'f1'`).Scan(&orgID).Error)
	assert.Equal(t, "org-wh", orgID, "已有覆盖值不得被迁移改写")
}

// TestMigrate234FloorOrgID_NoTableIsNoop 表不存在时返回 nil。
func TestMigrate234FloorOrgID_NoTableIsNoop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	assert.NoError(t, Migrate234FloorOrgID(db))
}
