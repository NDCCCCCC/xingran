package base

// =====================================================================
// Phase 91-01 Task 3: GORMRepository 泛型契约锁值（CRUD-REUSE-07）。
//
// 把 91-RESEARCH spike 验证的 GORM 链语义固化为可重跑契约：
//   契约 1 (P4): 空 scopes List → 全表分页，PageResult.List 为值切片 []T
//   契约 2:      Joins+Select scope 下 Count 正确（join 不增殖行）且 Find 保留 Select 列
//   契约 3:      顺序两个 Order 的 scope → 复合排序（第二排序键稳定有序）
//   契约 4 (P1): BatchDelete(nil)/[]string{} → nil；非空软删后 Total 减少、软删行不出现
//   契约 5 (A4): GetByID 空 scope → WHERE id = ?；有 scope → <TableName()>.id = ?
//   契约 6:      SortScope 三态 + SortScopeWithTail 尾随语义（Query 回调捕获 SQL 断言）
//
// 纪律（v1.27 规约）：t.TempDir 文件库 + AutoMigrate + t.Cleanup 关闭；
// 零 sleep、零 t.Parallel；行模型 basRepoRow8005 复用 base_80_05_test.go 脚手架，
// join 契约另用独立双表模型 + 独立 sqlite 库。
// =====================================================================

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// base91Child join 契约用子表模型（与 basRepoRow8005 同含 id 列，
// 供契约 2/5 构造跨表 join 场景）。
type base91Child struct {
	models.BaseModel
	Name     string `gorm:"size:100;not null"`
	ParentID string `gorm:"size:64;index"`
}

// TableName 表名。
func (base91Child) TableName() string { return "base91_children" }

// newBase91JoinDB 双表 sqlite 脚手架（basRepoRow8005 + base91Child），
// 附 Query 回调 SQL 捕获器；返回 repo、原生 db 句柄与 getSQL 读取器。
func newBase91JoinDB(t *testing.T) (*GORMRepository[basRepoRow8005], *gorm.DB, func() string) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "base91.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(&basRepoRow8005{}, &base91Child{}))

	var lastSQL string
	require.NoError(t, gormDB.Callback().Query().After("gorm:query").Register("base91_sql_capture", func(tx *gorm.DB) {
		lastSQL = tx.Statement.SQL.String()
	}))

	return NewGORMRepository[basRepoRow8005](gormDB), gormDB, func() string { return lastSQL }
}

func seedBase91Child(t *testing.T, db *gorm.DB, name, parentID string) base91Child {
	t.Helper()
	child := base91Child{Name: name, ParentID: parentID}
	require.NoError(t, db.Create(&child).Error)
	return child
}

// stripSQLQuotes 去掉 SQL 标识符引号（" ` [ ]），便于断言限定/裸列名形态。
func stripSQLQuotes(sql string) string {
	return strings.Map(func(r rune) rune {
		if r == '"' || r == '`' || r == '[' || r == ']' {
			return -1
		}
		return r
	}, sql)
}

// TestBase91_List_ValueSlice 契约 1（P4）: 空 scopes List 返回全表分页，
// page.List 经类型断言为 []basRepoRow8005 值切片，Total/Current/PageSize 回填正确。
func TestBase91_List_ValueSlice(t *testing.T) {
	repo, _ := newBasRepo8005(t)
	ctx := context.Background()

	seedBasRow8005(t, repo, "alice", 20)
	seedBasRow8005(t, repo, "bob", 21)
	seedBasRow8005(t, repo, "carol", 22)

	page, err := repo.List(ctx, PageParams{Current: 2, PageSize: 2})
	require.NoError(t, err)

	assert.Equal(t, int64(3), page.Total)
	assert.Equal(t, 2, page.Current)
	assert.Equal(t, 2, page.PageSize)

	rows, ok := page.List.([]basRepoRow8005)
	require.True(t, ok, "PageResult.List 必须是值切片 []T（crud_services_test.go:179 类型断言依赖）")
	require.Len(t, rows, 1, "3 行 size 2 page 2 → 第 3 行")
}

// TestBase91_List_JoinsSelectCount 契约 2: Joins+Select scope 下 Count 正确
// （1:1 join 不增殖行，自定义 Select 不使 Count 失真）且 Find 保留 Select 列。
func TestBase91_List_JoinsSelectCount(t *testing.T) {
	repo, db, _ := newBase91JoinDB(t)
	ctx := context.Background()

	p1 := seedBasRow8005(t, repo, "alpha", 20)
	p2 := seedBasRow8005(t, repo, "beta", 21)
	seedBase91Child(t, db, "kid-a", p1.ID)
	seedBase91Child(t, db, "kid-b", p2.ID)

	joinSelectScope := func(dbc *gorm.DB) *gorm.DB {
		return dbc.Select("bas_repo_rows_8005.id AS id, bc.name AS name").
			Joins("LEFT JOIN base91_children bc ON bc.parent_id = bas_repo_rows_8005.id")
	}

	page, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10}, joinSelectScope)
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total, "Count 必须正确（自定义 Select 被临时替换为 count(*) 后恢复）")

	rows, ok := page.List.([]basRepoRow8005)
	require.True(t, ok)
	require.Len(t, rows, 2)
	byID := make(map[string]string, len(rows))
	for _, r := range rows {
		byID[r.ID] = r.Name
	}
	assert.Equal(t, "kid-a", byID[p1.ID], "Find 必须保留 Select 列（name 来自子表而非父表）")
	assert.Equal(t, "kid-b", byID[p2.ID], "Find 必须保留 Select 列（name 来自子表而非父表）")
}

// TestBase91_CompositeSort 契约 3: 顺序两个 Order 的 scope 产生复合排序，
// 同列值两行按第二排序键稳定有序（door/wall 复合尾随语义的 GORM 底座）。
func TestBase91_CompositeSort(t *testing.T) {
	repo, _, _ := newBase91JoinDB(t)
	ctx := context.Background()

	seedBasRow8005(t, repo, "twin", 20)
	seedBasRow8005(t, repo, "twin", 40)
	seedBasRow8005(t, repo, "ana", 30)

	compositeSortScope := func(dbc *gorm.DB) *gorm.DB {
		return dbc.Order("name ASC").Order("age DESC")
	}

	page, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10}, compositeSortScope)
	require.NoError(t, err)
	rows, ok := page.List.([]basRepoRow8005)
	require.True(t, ok)
	require.Len(t, rows, 3)
	assert.Equal(t, "ana", rows[0].Name)
	assert.Equal(t, "twin", rows[1].Name)
	assert.Equal(t, 40, rows[1].Age, "同列值必须按第二排序键（age DESC）稳定有序")
	assert.Equal(t, 20, rows[2].Age)
}

// TestBase91_BatchDelete_NilIDs 契约 4（P1）: BatchDelete(ctx, nil) 与
// BatchDelete(ctx, []string{}) 均返回 NoError；非空 ids 软删后 List Total 减少、
// 软删行不再出现在 Find 结果。
func TestBase91_BatchDelete_NilIDs(t *testing.T) {
	repo, db := newBasRepo8005(t)
	ctx := context.Background()

	r1 := seedBasRow8005(t, repo, "a", 1)
	r2 := seedBasRow8005(t, repo, "b", 2)
	seedBasRow8005(t, repo, "survivor", 3)

	require.NoError(t, repo.BatchDelete(ctx, nil), "P1 语义反转：空 ids 是合法 no-op 返回 nil")
	require.NoError(t, repo.BatchDelete(ctx, []string{}))

	page, err := repo.List(ctx, PageParams{Current: 1, PageSize: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total, "空 ids 批删后行数不变")

	require.NoError(t, repo.BatchDelete(ctx, []string{r1.ID, r2.ID}))

	page2, err := repo.List(ctx, PageParams{Current: 1, PageSize: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page2.Total, "非空软删后 Total 减少")

	rows, ok := page2.List.([]basRepoRow8005)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "survivor", rows[0].Name, "软删行不再出现在 Find 结果")

	var physicalCount int64
	require.NoError(t, db.Unscoped().Model(&basRepoRow8005{}).Count(&physicalCount).Error)
	assert.Equal(t, int64(3), physicalCount, "BatchDelete 是软删除（行仍在库中，仅 deleted_at 置位）")
}

// TestBase91_GetByID_TableQualified 契约 5（A4）: GetByID 无 scope 命中行且
// id 过滤为裸 WHERE id = ?；带 join scope（同含 id 列的两表 join）时 id 过滤
// 为 <TableName()>.id = ? 表限定并命中（裸列名在该场景会 ambiguous）。
func TestBase91_GetByID_TableQualified(t *testing.T) {
	repo, db, getSQL := newBase91JoinDB(t)
	ctx := context.Background()

	parent := seedBasRow8005(t, repo, "alpha", 20)
	seedBase91Child(t, db, "kid-a", parent.ID)

	// 无 scope：命中行；SQL 为 WHERE id = ?（不表限定）。
	got, err := repo.GetByID(ctx, parent.ID)
	require.NoError(t, err)
	assert.Equal(t, "alpha", got.Name)
	plainSQL := stripSQLQuotes(getSQL())
	assert.Contains(t, plainSQL, "WHERE id = ?", "空 scope 的 id 过滤不得表限定")

	// 带 join scope：id 过滤表限定 → 命中（行为证明：两表同含 id 列，
	// 裸 WHERE id 会因列名 ambiguous 失败，表限定成功）。
	joinScope := func(dbc *gorm.DB) *gorm.DB {
		return dbc.Select("bas_repo_rows_8005.*").
			Joins("LEFT JOIN base91_children bc ON bc.parent_id = bas_repo_rows_8005.id")
	}
	got2, err := repo.GetByID(ctx, parent.ID, joinScope)
	require.NoError(t, err, "带 join scope 时 id 过滤必须表限定")
	require.NotNil(t, got2)
	assert.Equal(t, "alpha", got2.Name)
	joinSQL := stripSQLQuotes(getSQL())
	assert.Contains(t, joinSQL, "bas_repo_rows_8005.id = ?", "非空 scope 的 id 过滤必须为 <TableName()>.id = ?（Tabler 断言）")
}

// TestBase91_SortScope_TwoModes 契约 6: SortScope 三态（白名单命中 →
// ORDER BY <col> <dir>；OrderByColumn 空 → defaultOrder；非法 key → 无显式排序）
// 与 SortScopeWithTail 尾随语义（命中 → 复合、未命中 → 仅 tail），
// 经 Query 回调捕获 List 的 Find SQL 断言。
func TestBase91_SortScope_TwoModes(t *testing.T) {
	repo, _, getSQL := newBase91JoinDB(t)
	ctx := context.Background()

	seedBasRow8005(t, repo, "alice", 20)
	seedBasRow8005(t, repo, "bob", 21)

	allowed := map[string]string{"name": "name", "age": "age"}

	// B 型 SortScope —— 白名单命中 → 仅用户排序。
	hit := BaseListRequest{OrderByColumn: "name"}
	_, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10}, SortScope(hit, allowed, "created_at DESC"))
	require.NoError(t, err)
	sqlHit := stripSQLQuotes(getSQL())
	assert.Contains(t, sqlHit, "ORDER BY name ASC")
	assert.NotContains(t, sqlHit, "ORDER BY created_at DESC", "命中时默认序不得出现（排他语义）")

	// B 型 SortScope —— OrderByColumn 空 → defaultOrder。
	empty := BaseListRequest{}
	_, err = repo.List(ctx, PageParams{Current: 1, PageSize: 10}, SortScope(empty, allowed, "created_at DESC"))
	require.NoError(t, err)
	sqlEmpty := stripSQLQuotes(getSQL())
	assert.Contains(t, sqlEmpty, "ORDER BY created_at DESC", "OrderByColumn 为空必须应用 defaultOrder")

	// B 型 SortScope —— 非法 key → warn + 无显式排序（含默认序，与现状一致）。
	illegal := BaseListRequest{OrderByColumn: "evil; --"}
	_, err = repo.List(ctx, PageParams{Current: 1, PageSize: 10}, SortScope(illegal, allowed, "created_at DESC"))
	require.NoError(t, err)
	sqlIllegal := stripSQLQuotes(getSQL())
	assert.NotContains(t, sqlIllegal, "ORDER BY", "非法 key 必须无任何显式排序")

	// A 型 SortScopeWithTail —— 命中 → 复合（用户排序 + 尾随）。
	tailHit := BaseListRequest{OrderByColumn: "name"}
	_, err = repo.List(ctx, PageParams{Current: 1, PageSize: 10}, SortScopeWithTail(tailHit, allowed, "created_at DESC"))
	require.NoError(t, err)
	sqlTailHit := stripSQLQuotes(getSQL())
	assert.Contains(t, sqlTailHit, "ORDER BY name ASC")
	assert.Contains(t, sqlTailHit, "created_at DESC", "命中时必须复合尾随（A 型语义）")

	// A 型 SortScopeWithTail —— 空 → 仅尾随。
	tailEmpty := BaseListRequest{}
	_, err = repo.List(ctx, PageParams{Current: 1, PageSize: 10}, SortScopeWithTail(tailEmpty, allowed, "created_at DESC"))
	require.NoError(t, err)
	sqlTailEmpty := stripSQLQuotes(getSQL())
	assert.Contains(t, sqlTailEmpty, "ORDER BY created_at DESC")
	assert.NotContains(t, sqlTailEmpty, "ORDER BY name", "未命中时仅尾随（无用户排序）")
}
