package base

// Phase 151 / T3：GORMRepository 读链改造的就近契约测试。
//
// 纪律（沿用 service_test.go）：t.TempDir 文件库 + AutoMigrate + t.Cleanup
// 关闭；零 sleep、零 t.Parallel。注册表由本测试就地注册一个测试用 factory
// （base 包不得 import ops，避免与 internal/services/operations 循环依赖）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type scopedRepoRow struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:64"`
	DeptID string `gorm:"size:64"`
}

func (scopedRepoRow) TableName() string { return "scoped_repo_rows" }

func newScopedRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "scoped_repo.db")),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(&scopedRepoRow{}))
	return db
}

func scopedRepoCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// TestScopedRepo_CountAndFindHomologous：Dept 档注入 → 注册表的可见集谓词
// 经 ScopedDB 的 GORM Scopes 进入 List 起链；Count 在 scopes 之后应用 →
// PageResult.Total == len(List) == M（SIDECHANNEL-04 同源结构性保证）。
func TestScopedRepo_CountAndFindHomologous(t *testing.T) {
	query.ResetRegistryForTest()
	t.Cleanup(query.ResetRegistryForTest)
	require.NoError(t, query.Register("scoped_repo_rows", query.Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			return tx.Where("dept_id IN ?", r.DeptIDs)
		},
		CacheKey: func(context.Context) string { return "d-test" },
	}))

	db := newScopedRepoDB(t)
	require.NoError(t, db.Create(&[]scopedRepoRow{
		{Name: "可见1", DeptID: "deptA"}, {Name: "可见2", DeptID: "deptA"},
		{Name: "越界1", DeptID: "deptB"}, {Name: "越界2", DeptID: "deptB"},
		{Name: "越界3", DeptID: "deptB"},
	}).Error)

	repo := NewGORMRepository[scopedRepoRow](db)
	ctx := scopedRepoCtx(datascope.ScopeResult{DeptIDs: []string{"deptA"}}) // M=2
	res, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	list, ok := res.List.([]scopedRepoRow)
	require.True(t, ok, "List 必须是值切片 []T")
	assert.Equal(t, int64(2), res.Total, "COUNT 必须只数可见集 M 行")
	assert.Equal(t, 2, len(list), "明细必须只返回可见集 M 行")
	assert.Equal(t, int64(len(list)), res.Total, "PageResult.Total == len(List) 同源恒等")
}

// TestScopedRepo_AllFallback：context.Background()（未注入）与 All 档 →
// ScopedDB 原样返回，既有 base 契约结果逐字不变（零回归硬约束）。
func TestScopedRepo_AllFallback(t *testing.T) {
	query.ResetRegistryForTest()
	t.Cleanup(query.ResetRegistryForTest)
	require.NoError(t, query.Register("scoped_repo_rows", query.Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			return tx.Where("dept_id IN ?", r.DeptIDs)
		},
	}))

	db := newScopedRepoDB(t)
	require.NoError(t, db.Create(&[]scopedRepoRow{
		{Name: "A", DeptID: "deptA"}, {Name: "B", DeptID: "deptB"},
	}).Error)
	repo := NewGORMRepository[scopedRepoRow](db)

	for _, ctx := range []context.Context{
		context.Background(),
		scopedRepoCtx(datascope.AllScope("u1")),
	} {
		res, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(2), res.Total, "未注入 / All 档必须按全库 N 行零回归")
		assert.Equal(t, 2, len(res.List.([]scopedRepoRow)))
	}
}

// TestScopedRepo_ScopesStillApplied：变参 scopes（排序 / 业务过滤）与
// ScopedDB 叠加后语义不变（F-11：scopes 在 Count 之前应用）。
func TestScopedRepo_ScopesStillApplied(t *testing.T) {
	query.ResetRegistryForTest()
	t.Cleanup(query.ResetRegistryForTest)
	require.NoError(t, query.Register("scoped_repo_rows", query.Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			return tx.Where("dept_id IN ?", r.DeptIDs)
		},
	}))

	db := newScopedRepoDB(t)
	require.NoError(t, db.Create(&[]scopedRepoRow{
		{Name: "b", DeptID: "deptA"}, {Name: "a", DeptID: "deptA"},
		{Name: "z", DeptID: "deptB"},
	}).Error)
	repo := NewGORMRepository[scopedRepoRow](db)
	ctx := scopedRepoCtx(datascope.ScopeResult{DeptIDs: []string{"deptA"}})

	res, err := repo.List(ctx, PageParams{Current: 1, PageSize: 10}, func(q *gorm.DB) *gorm.DB {
		return q.Order("name ASC")
	})
	require.NoError(t, err)
	list := res.List.([]scopedRepoRow)
	require.Len(t, list, 2)
	assert.Equal(t, "a", list[0].Name, "变参 scope 排序必须与 ScopedDB 叠加生效")
	assert.Equal(t, int64(2), res.Total, "Count 与 Find 共享同一条带 scope 的链")
}

// TestScopedRepo_WriteChainUntouched：Create/Update/Delete/BatchDelete 的
// DryRun SQL 中不含任何 scope 谓词（GORM Scopes 会污染 UPDATE/DELETE 的
// WHERE，写链必须保持裸起链）。注入 Dept 档 + 已注册表，验证写链不被裁剪。
func TestScopedRepo_WriteChainUntouched(t *testing.T) {
	query.ResetRegistryForTest()
	t.Cleanup(query.ResetRegistryForTest)
	require.NoError(t, query.Register("scoped_repo_rows", query.Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			return tx.Where("dept_id IN ?", r.DeptIDs)
		},
	}))

	db := newScopedRepoDB(t)
	ctx := scopedRepoCtx(datascope.ScopeResult{DeptIDs: []string{"deptA"}})
	repo := NewGORMRepository[scopedRepoRow](db)

	// Create：真实写库后确认行存在（未被 scope 谓词破坏）
	require.NoError(t, repo.Create(ctx, &scopedRepoRow{Name: "w", DeptID: "deptB"}))
	var n int64
	require.NoError(t, db.Model(&scopedRepoRow{}).Count(&n).Error)
	assert.Equal(t, int64(1), n, "Create 写链不得挂读 scope 谓词")

	// 源码静态断言：4 个写链方法体不含 ScopedDB（与 ops 侧
	// TestScopedLayer_BaseReadChainUsesScopedDB 双保险）
	src := readBaseServiceGo(t)
	for _, m := range []string{"Create", "Update", "Delete", "BatchDelete"} {
		idx := strings.Index(src, "func (r *GORMRepository[T]) "+m+"(")
		require.GreaterOrEqual(t, idx, 0, "缺少方法 %s", m)
		end := strings.Index(src[idx:], "\nfunc (r *GORMRepository[T])")
		if end < 0 {
			end = len(src) - idx
		}
		body := src[idx : idx+end]
		assert.NotContains(t, body, "ScopedDB", "%s 写链禁止挂 ScopedDB", m)
		assert.Contains(t, body, "r.db.WithContext(ctx)", "%s 写链必须保持裸起链", m)
	}
}

func readBaseServiceGo(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("service.go")
	require.NoError(t, err)
	return string(b)
}
