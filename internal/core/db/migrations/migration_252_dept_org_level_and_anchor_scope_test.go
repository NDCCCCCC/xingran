package migrations

// migration_252_dept_org_level_and_anchor_scope_test.go —— 迁移 252 的幂等性与列定义回归。
//
// 为什么必须测：迁移是**生产上线时唯一自动执行的环节**。若列没建出来，
// pkg/datascope/anchor.go 的锚点解析会全部失败 → 中间件按「无锚点」降级 →
// 机构维度静默失效（用户看不到任何报错，只是"配置了但没效果"）。这是最难排查的
// 失败模式，故用测试把列的存在性锁死。
//
// 覆盖：
//   - 空库（无 sys_dept/sys_role/sys_post）→ 不报错（非阻断降级）
//   - 正常库 → 五个列全部创建，且 default 值正确（0 / ''）
//   - 幂等 → 连续执行两次不报错、不重复加列
//   - 已有列（模拟已迁移过）→ 跳过而不报错

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMigration252TestDB(t *testing.T, withTables bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m252.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	if withTables {
		// 模拟迁移前的存量库：三张表都在，但都没有本次新增的列。
		require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT,
			ancestors TEXT DEFAULT '', order_num INTEGER DEFAULT 0, leader TEXT,
			phone TEXT, email TEXT, is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0, remark TEXT, deleted_at DATETIME)`).Error)
		require.NoError(t, db.Exec(`CREATE TABLE sys_role (
			id TEXT PRIMARY KEY, role_name TEXT, role_key TEXT, role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1, menu_check_strictly INTEGER DEFAULT 1,
			dept_check_strictly INTEGER DEFAULT 1, status INTEGER DEFAULT 0,
			remark TEXT, deleted_at DATETIME)`).Error)
		require.NoError(t, db.Exec(`CREATE TABLE sys_post (
			id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT, post_sort INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0, remark TEXT, deleted_at DATETIME)`).Error)
	}
	return db
}

// TestMigrate252_CreatesAllColumns 迁移后五个列全部存在，且默认值正确。
func TestMigrate252_CreatesAllColumns(t *testing.T) {
	db := newMigration252TestDB(t, true)

	require.NoError(t, Migrate252DeptOrgLevelAndAnchorScope(db))

	assert.True(t, db.Migrator().HasColumn("sys_dept", "org_level"), "sys_dept.org_level 必须存在")
	assert.True(t, db.Migrator().HasColumn("sys_role", "anchor_mode"), "sys_role.anchor_mode 必须存在")
	assert.True(t, db.Migrator().HasColumn("sys_role", "anchor_param"), "sys_role.anchor_param 必须存在")
	assert.True(t, db.Migrator().HasColumn("sys_post", "anchor_mode"), "sys_post.anchor_mode 必须存在")
	assert.True(t, db.Migrator().HasColumn("sys_post", "anchor_param"), "sys_post.anchor_param 必须存在")

	// 存量行（迁移前插入）必须拿到零值默认，不能是 NULL：anchor.go 用
	// COALESCE(anchor_mode, 0) 兜底了 NULL，但 dept 的 org_level 直接比较，
	// NULL 会让 level 判定永远为 false。
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d0','存量部门','D0')`).Error)
	var lv int
	require.NoError(t, db.Raw(`SELECT org_level FROM sys_dept WHERE id='d0'`).Scan(&lv).Error)
	assert.Equal(t, 0, lv, "存量部门的 org_level 必须默认 0（非机构），不得为 NULL")

	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key) VALUES ('r0','存量角色','r0')`).Error)
	var mode int
	var param string
	require.NoError(t, db.Raw(`SELECT anchor_mode, anchor_param FROM sys_role WHERE id='r0'`).Scan(&struct {
		AnchorMode  *int
		AnchorParam *string
	}{&mode, &param}).Error)
	assert.Equal(t, 0, mode)
	assert.Equal(t, "", param)
}

// TestMigrate252_Idempotent 连续执行两次不报错、列不重复。
func TestMigrate252_Idempotent(t *testing.T) {
	db := newMigration252TestDB(t, true)

	require.NoError(t, Migrate252DeptOrgLevelAndAnchorScope(db))
	require.NoError(t, Migrate252DeptOrgLevelAndAnchorScope(db), "第二次执行必须幂等通过")

	assert.True(t, db.Migrator().HasColumn("sys_dept", "org_level"))
	assert.True(t, db.Migrator().HasColumn("sys_role", "anchor_param"))
}

// TestMigrate252_MissingTablesDegrades 空库（无三张表）→ 不报错，非阻断降级。
func TestMigrate252_MissingTablesDegrades(t *testing.T) {
	db := newMigration252TestDB(t, false)

	err := Migrate252DeptOrgLevelAndAnchorScope(db)
	assert.NoError(t, err, "缺表时必须非阻断降级，不能让启动失败")
}

// TestMigrate252_OrgLevelIsQueryableByAnchorSQL 迁移后 anchor.go 的 org_level SQL 能跑通。
//
// 这是跨包契约：迁移建的列必须能被 pkg/datascope 的查询直接消费（列名 org_level）。
func TestMigrate252_OrgLevelIsQueryableByAnchorSQL(t *testing.T) {
	db := newMigration252TestDB(t, true)
	require.NoError(t, Migrate252DeptOrgLevelAndAnchorScope(db))

	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, org_level) VALUES
		('D-root','湖北分公司','HB','',1),
		('D-wh','武汉中支','WH','D-root',2),
		('D-qk','硚口支公司','QK','D-root,D-wh',3)`).Error)

	for _, target := range []int{1, 2, 3} {
		var cnt int64
		require.NoError(t, db.Raw(
			`SELECT COUNT(*) FROM sys_dept WHERE org_level = ? AND deleted_at IS NULL`, target,
		).Scan(&cnt).Error)
		assert.Equal(t, int64(1), cnt, "层级 %d 应恰好有一个节点（供 ValidateAnchor 计数）", target)
	}
}
