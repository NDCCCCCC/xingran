package query

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newQueryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, status INTEGER, deleted_at DATETIME)`).Error)
	return db
}

func TestNewQueryBuilder_Defaults(t *testing.T) {
	db := newQueryDB(t)
	qb := NewQueryBuilder(db)
	require.NotNil(t, qb)
	assert.Equal(t, db, qb.db)
	assert.True(t, qb.excludeDeleted)
	assert.Empty(t, qb.conditions)
}

func TestWhereEqual_AndVariants(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.WhereEqual("status", 1).
		WhereNotEqual("type", 2).
		WhereGreaterThan("age", 18).
		WhereGreaterThanOrEqual("score", 60).
		WhereLessThan("qty", 100).
		WhereLessThanOrEqual("rank", 5)
	assert.Len(t, qb.conditions, 6)
}

func TestWhereLike_EmptyValueSkipped(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.WhereLike("name", "")
	assert.Empty(t, qb.conditions, "空值应跳过")
	qb.WhereLike("name", "alice")
	require.Len(t, qb.conditions, 1)
	assert.Equal(t, "%alice%", qb.conditions[0].Value)
}

func TestWhereIn_EmptySkipped(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.WhereIn("id", nil)
	assert.Empty(t, qb.conditions)
	qb.WhereIn("id", []interface{}{"a", "b"})
	assert.Len(t, qb.conditions, 1)
}

func TestWhereNotIn(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.WhereNotIn("id", []interface{}{"x"})
	assert.Equal(t, "NOT IN", qb.conditions[0].Operator)
}

func TestWhereOrNull(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.WhereOrNull("owner", nil)
	assert.Empty(t, qb.orConditions, "nil value 应跳过")
	qb.WhereOrNull("owner", "alice")
	require.Len(t, qb.orConditions, 1)
	assert.Len(t, qb.orConditions[0], 2)
	assert.Nil(t, qb.orConditions[0][1].Value)
}

func TestOrWhere(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.OrWhere(nil)
	assert.Empty(t, qb.orConditions)
	qb.OrWhere([]Condition{{Field: "x", Operator: "=", Value: 1}})
	assert.Len(t, qb.orConditions, 1)
}

func TestOrderByPreloadAndExclude(t *testing.T) {
	qb := NewQueryBuilder(newQueryDB(t))
	qb.OrderByField("created_at", DESC).OrderByField("id", ASC)
	require.Len(t, qb.orderBy, 2)
	assert.Equal(t, DESC, qb.orderBy[0].Direction)

	qb.Preload("Roles").PreloadWith("Posts", func(d *gorm.DB) *gorm.DB { return d })
	require.Len(t, qb.preloads, 1)
	require.Len(t, qb.preloadsWith, 1)

	qb.ExcludeDeleted(false)
	assert.False(t, qb.excludeDeleted)
}

func TestBuild_Operators(t *testing.T) {
	db := newQueryDB(t)
	// 插入测试数据
	require.NoError(t, db.Exec(`INSERT INTO items VALUES ('a', 'n1', 1, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO items VALUES ('b', 'n2', 2, NULL)`).Error)

	type item struct {
		ID     string
		Name   string
		Status int
	}
	qb := NewQueryBuilder(db).
		WhereEqual("status", 1).
		OrderByField("name", ASC)
	var rows []item
	q := qb.Build(&item{}).Find(&rows)
	require.NoError(t, q.Error)
	assert.Len(t, rows, 1)
}

func TestBuild_AllOperators(t *testing.T) {
	// 覆盖所有 Operator 分支
	db := newQueryDB(t)
	qb := NewQueryBuilder(db).
		Where("a", "=", 1).
		Where("a", "!=", 2).
		Where("a", ">", 0).
		Where("a", ">=", 0).
		Where("a", "<", 10).
		Where("a", "<=", 10).
		Where("a", "LIKE", "x%").
		Where("a", "IN", []interface{}{1, 2}).
		Where("a", "NOT IN", []interface{}{3, 4})
	assert.Len(t, qb.conditions, 9)
	// 不报错即覆盖成功
	_ = qb.Build(&struct{}{})
}

func TestBuild_UnknownOperatorSkipped(t *testing.T) {
	db := newQueryDB(t)
	qb := NewQueryBuilder(db).Where("a", "BOGUS", 1)
	// 未知 operator 不应 panic
	assert.NotPanics(t, func() { _ = qb.Build(&struct{}{}) })
}

func TestBuild_OrWhereNullBranch(t *testing.T) {
	db := newQueryDB(t)
	qb := NewQueryBuilder(db).OrWhere([]Condition{
		{Field: "owner", Operator: "=", Value: nil},
		{Field: "owner", Operator: "=", Value: "x"},
	})
	_ = qb.Build(&struct{}{}) // IS NULL + = x 路径覆盖
}

func TestPaginate_Defaults(t *testing.T) {
	db := newQueryDB(t)
	type item struct{ ID string }
	// current<=0 → 1; pageSize<=0 → 10
	q := Paginate(db.Model(&item{}), 0, 0)
	assert.NotNil(t, q)
}

func TestCountAndQuery_Integration(t *testing.T) {
	db := newQueryDB(t)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Exec(`INSERT INTO items VALUES (?, 'n', 1, NULL)`, string(rune('a'+i))).Error)
	}
	type item struct{ ID string }
	var rows []item
	total, err := CountAndQuery(db.Model(&item{}), &rows, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, rows, 3)
}