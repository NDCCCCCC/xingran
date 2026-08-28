package gormutil

// =====================================================================
// Phase 80-05 Task 5a: join_builder.go 链式 SQL + 纯函数段。
// (基线 63.4% → ≥70%;链式 Select/Where/Or/Order/Limit/Offset/Count/
// Scan/Find/First/Pluck/GetDB/Reset + BuildOnClause/BuildJoinClause/
// ParseSelectFields。)
//
// 纪律:零 sleep、零 t.Parallel。
// =====================================================================

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// jbtUser8005 / jbtOrg8005 双表 JOIN 测试模型。
type jbtUser8005 struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"size:50"`
	OrgID uint
}

func (jbtUser8005) TableName() string { return "jbt_users_8005" }

type jbtOrg8005 struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:50"`
}

func (jbtOrg8005) TableName() string { return "jbt_orgs_8005" }

func newJbtDB8005(t *testing.T) *gorm.DB {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "jbt8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(&jbtUser8005{}, &jbtOrg8005{}))
	for _, o := range []jbtOrg8005{{ID: 1, Name: "运维"}, {ID: 2, Name: "研发"}} {
		require.NoError(t, gormDB.Create(&o).Error)
	}
	for _, u := range []jbtUser8005{
		{ID: 1, Name: "alice", OrgID: 1},
		{ID: 2, Name: "bob", OrgID: 1},
		{ID: 3, Name: "carol", OrgID: 2},
	} {
		require.NoError(t, gormDB.Create(&u).Error)
	}
	return gormDB
}

// TestJbt8005_ChainMethods_SQL:链式 + ToSQL 验证片段。
func TestJbt8005_ChainMethods_SQL(t *testing.T) {
	db := newJbtDB8005(t)
	b := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.name LIKE ?", "%a%").
		Or("jbt_users_8005.org_id = ?", 2).
		Order("jbt_users_8005.id ASC").
		Limit(10).Offset(5)

	sql := b.Build().ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]jbtUser8005{}) })
	assert.Contains(t, sql, "INNER JOIN", "InnerJoin 应生成 INNER JOIN")
	assert.Contains(t, sql, "jbt_orgs_8005")
	assert.Contains(t, sql, "ON")
	assert.Contains(t, sql, "LIKE", "Where 应透传 LIKE")
	assert.Contains(t, sql, " OR ", "Or 应产生 OR")
	assert.Contains(t, sql, "ORDER BY", "Order 应生成 ORDER BY")
	assert.Contains(t, sql, "LIMIT", "Limit 应生成 LIMIT")
	assert.Contains(t, sql, "OFFSET", "Offset 应生成 OFFSET")

	// Count:不返回结果,但 count 被绑定。Count(&n) 走 Count 语句。
	var n int64
	b2 := NewJoinBuilder(db.Model(&jbtUser8005{})).InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id")
	b2.Count(&n)
	rows := b2.Build().Find(&[]jbtUser8005{})
	require.NoError(t, rows.Error)
	assert.Equal(t, int64(3), n)

	// Reset:清空 configs,后续 Build 无 JOIN。
	b3 := NewJoinBuilder(db.Model(&jbtUser8005{})).InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").Reset()
	sql3 := b3.Build().ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]jbtUser8005{}) })
	assert.NotContains(t, sql3, "INNER JOIN", "Reset 后不应含 JOIN")
	assert.Len(t, b3.GetConfigs(), 0, "Reset 应清 configs")
}

// TestJbt8005_BuildJoinClause_BuildOnClause:纯函数段断言格式。
func TestJbt8005_BuildJoinClause_BuildOnClause(t *testing.T) {
	// BuildOnClause:左表.字段 = 右表.字段。
	assert.Equal(t, "users.id = roles.user_id", BuildOnClause("users", "id", "roles", "user_id"))

	// BuildJoinClause:JOIN 类型 + 表 + ON 子句。
	assert.Equal(t, "INNER JOIN orders ON orders.user_id = users.id",
		BuildJoinClause(InnerJoin, "orders", BuildOnClause("orders", "user_id", "users", "id")))
	assert.Equal(t, "LEFT JOIN profiles ON profiles.user_id = users.id",
		BuildJoinClause(LeftJoin, "profiles", "profiles.user_id = users.id"))
	assert.Equal(t, "RIGHT JOIN logs ON logs.entity_id = entities.id",
		BuildJoinClause(RightJoin, "logs", "logs.entity_id = entities.id"))
}

// TestJbt8005_ParseSelectFields:含点的字段原样;纯字段加表前缀;空字段。
func TestJbt8005_ParseSelectFields(t *testing.T) {
	// 纯字段 → 自动添加表前缀。
	out := ParseSelectFields("users", []string{"id", "name"})
	assert.Equal(t, []string{"users.id", "users.name"}, out)

	// 含点的字段原样返回。
	out = ParseSelectFields("users", []string{"id", "profiles.bio", "name"})
	assert.Equal(t, []string{"users.id", "profiles.bio", "users.name"}, out)

	// 空切片 → 空结果。
	assert.Empty(t, ParseSelectFields("users", nil))
}

// TestJbt8005_Find_Scan_First_Pluck_LeftJoin:sqlite 实跑,断言行数/列值。
func TestJbt8005_Find_Scan_First_Pluck_LeftJoin(t *testing.T) {
	db := newJbtDB8005(t)

	// Find:InnerJoin + 限定 ID = 1。
	var got []jbtUser8005
	b := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	b.Find(&got)
	require.NoError(t, b.Build().Error)
	assert.Len(t, got, 1, "InnerJoin 命中 1 行")
	assert.Equal(t, "alice", got[0].Name)

	// Scan:InnerJoin + 限定 ID = 1(独立 builder)。
	var scanned []jbtUser8005
	b2 := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	b2.Scan(&scanned)
	assert.NoError(t, b2.Build().Error, "Scan 委托应可执行")
	assert.Len(t, scanned, 1)

	// First:独立 builder。
	var first jbtUser8005
	b3 := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	b3.First(&first)
	assert.NoError(t, b3.Build().Error, "First 委托应可执行")
	assert.Equal(t, "alice", first.Name)

	// Pluck:独立 builder;用限定列名避免两表都有 name 触发 ambiguous。
	var names []string
	b4 := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	b4.Pluck("jbt_users_8005.name", &names)
	assert.NoError(t, b4.Build().Error, "Pluck 委托应可执行")
	assert.Equal(t, []string{"alice"}, names)

	// InnerJoin 行数(2 个运维成员)。
	var allOrg1 []jbtUser8005
	b5 := NewJoinBuilder(db.Model(&jbtUser8005{})).
		InnerJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_orgs_8005.name = ?", "运维")
	b5.Find(&allOrg1)
	assert.NoError(t, b5.Build().Error, "InnerJoin 委托应可执行")
	assert.Len(t, allOrg1, 2, "InnerJoin + WHERE name=运维 应命中 alice/bob")
	assert.ElementsMatch(t, []string{"alice", "bob"}, []string{allOrg1[0].Name, allOrg1[1].Name})

	// LeftJoin 单独覆盖分支(无 Select 限定 → GORM 默认 SELECT user.*;
	//  InnerJoin 路径已知 OK,此处仅验 LeftJoin 字符串生成路径)。
	var leftGot []jbtUser8005
	b6 := NewJoinBuilder(db.Model(&jbtUser8005{})).
		LeftJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	b6.Find(&leftGot)
	assert.NoError(t, b6.Build().Error, "LeftJoin 委托应可执行")
	assert.Len(t, leftGot, 1)

	// LeftJoinWithAlias 单独覆盖分支(别名 `o` → ON 中用 `o.id`)。
	bAlias := NewJoinBuilder(db.Model(&jbtUser8005{})).
		LeftJoinWithAlias("jbt_orgs_8005", "o", "o.id = jbt_users_8005.org_id").
		Where("jbt_users_8005.id = ?", 1)
	var aliased []jbtUser8005
	bAlias.Find(&aliased)
	assert.NoError(t, bAlias.Build().Error, "LeftJoinWithAlias 应可执行")
	assert.Len(t, aliased, 1)

	// GetDB 不应用 JOIN(无 configs 副本)。
	plainDB := NewJoinBuilder(db.Model(&jbtUser8005{})).GetDB()
	assert.NotNil(t, plainDB)
}

// TestJbt8005_Select_OnJoin:Select 在已有 configs 时走 last-config 分支。
func TestJbt8005_Select_OnJoin(t *testing.T) {
	db := newJbtDB8005(t)
	b := NewJoinBuilder(db.Model(&jbtUser8005{})).
		LeftJoin("jbt_orgs_8005", "jbt_orgs_8005.id = jbt_users_8005.org_id").
		Select("jbt_users_8005.name", "jbt_orgs_8005.name as org_name")

	cfg := b.GetConfigs()
	require.Len(t, cfg, 1)
	assert.ElementsMatch(t, []string{"jbt_users_8005.name", "jbt_orgs_8005.name as org_name"}, cfg[0].Selects)

	// 无 configs 时 Select 直接作用 db。
	b2 := NewJoinBuilder(db.Model(&jbtUser8005{})).Select("id", "name")
	sql := b2.Build().ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]jbtUser8005{}) })
	assert.Contains(t, sql, "SELECT")
	assert.Contains(t, sql, "id")
	assert.Contains(t, sql, "name")
}