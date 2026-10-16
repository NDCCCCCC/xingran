package gormutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newGUDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE depts (id TEXT PRIMARY KEY, name TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, dept_id TEXT, name TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_roles (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE roles (id TEXT PRIMARY KEY, name TEXT)`).Error)
	return db
}

// =====================================================================
// batch_loader.go: LoadBelongsTo / LoadManyToMany / LoadAssociationsBatch
// =====================================================================

type deptModel struct{ ID, Name string }

func (deptModel) TableName() string { return "depts" }

func TestBatchLoader_LoadBelongsTo(t *testing.T) {
	db := newGUDB(t)
	require.NoError(t, db.Exec(`INSERT INTO depts VALUES ('d1', '研发')`).Error)

	loader := NewBatchLoader(db)

	// 空 dests → nil err
	require.NoError(t, loader.LoadBelongsTo(nil,
		func(d interface{}) string { return "" },
		&deptModel{},
		func(d, m interface{}) {}))

	// 全空外键 → 不查表直接返回
	dests := []interface{}{&deptModel{ID: ""}}
	require.NoError(t, loader.LoadBelongsTo(dests,
		func(d interface{}) string { return "" },
		&deptModel{},
		func(d, m interface{}) {}))
}

func TestBatchLoader_LoadManyToMany(t *testing.T) {
	db := newGUDB(t)
	loader := NewBatchLoader(db)

	// 空 IDs → 直接返回空 map,不查表
	m, err := loader.LoadManyToMany(db, nil, "user_roles", "user_id", "role_id", &struct{}{})
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestBatchLoader_LoadAssociationsBatch(t *testing.T) {
	db := newGUDB(t)
	require.NoError(t, db.Exec(`INSERT INTO user_roles VALUES ('u1','r1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO roles VALUES ('r1', 'admin')`).Error)

	loader := NewBatchLoader(db)
	m, err := loader.LoadAssociationsBatch(db, []string{"u1"},
		"user_roles", "user_id", "role_id",
		"roles", "id")
	require.NoError(t, err)
	assert.Len(t, m["u1"], 1)

	// 空 IDs
	m, err = loader.LoadAssociationsBatch(db, nil, "user_roles", "user_id", "role_id", "roles", "id")
	require.NoError(t, err)
	assert.Empty(t, m)
}

// =====================================================================
// join_builder.go: JoinType / NewJoinBuilder / *Join / Build
// =====================================================================

func TestJoinType_Constants(t *testing.T) {
	assert.Equal(t, JoinType("INNER JOIN"), InnerJoin)
	assert.Equal(t, JoinType("LEFT JOIN"), LeftJoin)
	assert.Equal(t, JoinType("RIGHT JOIN"), RightJoin)
}

func TestJoinBuilder_AllJoinMethods(t *testing.T) {
	db := newGUDB(t)

	b := NewJoinBuilder(db).
		InnerJoin("depts", "depts.id = users.dept_id").
		LeftJoin("roles", "roles.id = ?", "r1").
		RightJoin("user_roles", "user_roles.user_id = users.id").
		LeftJoinWithAlias("depts", "d", "d.id = users.dept_id")

	// 4 个 join
	require.Len(t, b.configs, 4)
}

func TestJoinBuilder_Build(t *testing.T) {
	db := newGUDB(t)
	b := NewJoinBuilder(db).LeftJoin("depts", "depts.id = users.dept_id")
	q := b.Build()
	assert.NotNil(t, q)

	// NewJoinBuilderWithModel
	b2 := NewJoinBuilderWithModel(db, &struct{}{})
	assert.NotNil(t, b2)
}

// =====================================================================
// preload_helper.go / result_mapper.go: 简化冒烟测试
// =====================================================================

func TestPreloadBuilder(t *testing.T) {
	b := NewPreloadBuilder().Add("Roles").Add("Posts")
	assert.Len(t, b.GetConfigs(), 2)
	q := b.Apply(newGUDB(t))
	assert.NotNil(t, q)
	assert.Equal(t, "Posts.Comments", BuildPreloadPath("Posts", "Comments"))
}

func TestMapBuilderAndGenerics(t *testing.T) {
	mb := NewMapBuilder().
		Add(map[string]interface{}{"id": "1"}).
		Add(map[string]interface{}{"id": "2"})
	assert.Len(t, mb.Build(), 2)

	items := []string{"a", "b", "c"}
	m := ToIDMap(items, func(s string) string { return s })
	assert.Equal(t, "a", m["a"])

	g := GroupBy(items, func(s string) string { return s })
	assert.Len(t, g["a"], 1)

	ids := ExtractIDs(items, func(s string) string { return s })
	assert.Equal(t, []string{"a", "b", "c"}, ids)

	idx := IndexBy(items, func(s string) string { return s })
	assert.Equal(t, "a", idx["a"])

	// BatchMap
	pairs := BatchMap([]string{"a", "b"}, []string{"a", "b"},
		func(s string) string { return s },
		func(s string) int { return len(s) })
	assert.Equal(t, 2, len(pairs))

	// MergeMaps
	merged := MergeMaps[string, int](map[string]int{"a": 1}, map[string]int{"b": 2})
	assert.Equal(t, 2, len(merged))
	assert.Equal(t, 1, merged["a"])

	mapped := MapSlice(items, func(s string) string { return s + "!" })
	assert.Equal(t, "a!", mapped[0])

	filtered := FilterSlice(items, func(s string) bool { return s != "b" })
	assert.Equal(t, []string{"a", "c"}, filtered)

	reduced := ReduceSlice(items, "", func(r string, s string) string { return r + s })
	assert.Equal(t, "abc", reduced)
}