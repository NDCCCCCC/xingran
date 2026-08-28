package query

// =====================================================================
// Phase 80-05 Task 4b: pagination.go 全 0% 收口 — NewPaginatedResult /
// Normalize / GetOffset / ApplyOrder / ApplyPagination /
// NewDefaultQueryExecutor + Execute(sqlite 落库断言)。
//
// 纪律:零 sleep、零 t.Parallel(共享 sqlite fixture)。
// =====================================================================

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// pagRow8005 测试行模型。
type pagRow8005 struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"size:50"`
	Order int
}

func newPagDB8005(t *testing.T) *gorm.DB {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pag8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(&pagRow8005{}))
	for i, n := range []string{"a", "c", "b", "e", "d"} {
		require.NoError(t, gormDB.Create(&pagRow8005{Name: n, Order: i}).Error)
	}
	return gormDB
}

// TestPag8005_Normalize_Offset:PaginationRequest.Normalize(0/负/超大 pageSize
// 默认化)+ GetOffset 数学(current=3,pageSize=10 → 20,手算注释)。
func TestPag8005_Normalize_Offset(t *testing.T) {
	// Current <= 0 → 1。
	p := &PaginationRequest{Current: 0, PageSize: 10}
	p.Normalize()
	assert.Equal(t, 1, p.Current)
	assert.Equal(t, 10, p.PageSize)

	// 负数 Current / PageSize → 默认值。
	p = &PaginationRequest{Current: -5, PageSize: -3}
	p.Normalize()
	assert.Equal(t, 1, p.Current)
	assert.Equal(t, 10, p.PageSize)

	// PageSize > 100 → 100。
	p = &PaginationRequest{Current: 2, PageSize: 500}
	p.Normalize()
	assert.Equal(t, 2, p.Current)
	assert.Equal(t, 100, p.PageSize)

	// 合法值保持不变。
	p = &PaginationRequest{Current: 3, PageSize: 10}
	p.Normalize()
	assert.Equal(t, 3, p.Current)
	assert.Equal(t, 10, p.PageSize)

	// GetOffset:current=3, pageSize=10 → (3-1)*10 = 20。
	assert.Equal(t, 20, p.GetOffset())

	// GetOffset 隐式触 Normalize;Current=0 自动升 1 → offset=0。
	p2 := &PaginationRequest{Current: 0, PageSize: 10}
	assert.Equal(t, 0, p2.GetOffset())
	assert.Equal(t, 1, p2.Current, "GetOffset 内 Normalize 副作用")
}

// TestPag8005_NewPaginatedResult:零 total / 空 data / 正常三态字段映射 +
// TotalPages 取整(round-up)。
func TestPag8005_NewPaginatedResult(t *testing.T) {
	// 零 total → TotalPages=0;正常数据。
	pr := NewPaginatedResult(0, 1, 10, []int{})
	assert.Equal(t, int64(0), pr.Total)
	assert.Equal(t, 1, pr.Current)
	assert.Equal(t, 10, pr.PageSize)
	assert.Equal(t, 0, pr.TotalPages)
	assert.Empty(t, pr.Data)

	// 整除:25 / 10 = 2 pages;余 5 → +1 = 3 pages。
	pr = NewPaginatedResult(25, 1, 10, []string{"a"})
	assert.Equal(t, int64(25), pr.Total)
	assert.Equal(t, 3, pr.TotalPages, "25 行 / 10 size → 3 页(向上取整)")

	// 整除:20 / 10 = 2 pages;余 0 → 2 pages。
	pr = NewPaginatedResult(20, 1, 10, []string{"a"})
	assert.Equal(t, 2, pr.TotalPages, "20 行 / 10 size → 2 页(整除)")
}

// TestPag8005_ListParams_Order_Pagination:Normalize 默认值(OrderBy=created_at /
// OrderDir=DESC)+ ApplyOrder + ApplyPagination 落库断言 LIMIT/OFFSET 生效。
func TestPag8005_ListParams_Order_Pagination(t *testing.T) {
	db := newPagDB8005(t)

	// Normalize 默认值。
	lp := &ListParams{PaginationRequest: PaginationRequest{Current: 0, PageSize: 0}, OrderBy: "", OrderDir: ""}
	lp.Normalize()
	assert.Equal(t, 1, lp.Current)
	assert.Equal(t, 10, lp.PageSize)
	assert.Equal(t, "created_at", lp.OrderBy)
	assert.Equal(t, "DESC", lp.OrderDir)

	// OrderDir 大小写不敏感("asc" 非 "ASC" → 默认 DESC)。
	lp2 := &ListParams{PaginationRequest: PaginationRequest{Current: 1, PageSize: 5}, OrderBy: "name", OrderDir: "asc"}
	lp2.Normalize()
	assert.Equal(t, "DESC", lp2.OrderDir, "非 'ASC'/'DESC' → 默认 DESC")

	// ApplyOrder + ApplyPagination:sqlite 断言排序 + 分页。
	lp3 := &ListParams{PaginationRequest: PaginationRequest{Current: 2, PageSize: 2}, OrderBy: "name", OrderDir: "ASC"}
	var names []string
	q := lp3.ApplyOrder(db.Model(&pagRow8005{}))
	q = lp3.ApplyPagination(q)
	require.NoError(t, q.Pluck("name", &names).Error)
	// 种子:5 行 name = a, c, b, e, d → asc by name: a, b, c, d, e。
	// page 2 size 2 → c, d。
	assert.Equal(t, []string{"c", "d"}, names)
}

// TestPag8005_Executor_Execute:NewDefaultQueryExecutor + sqlite 种子 → Execute
// 返回 PaginatedResult(total/list/current/pageSize 四字段,分页约定断言)。
func TestPag8005_Executor_Execute(t *testing.T) {
	db := newPagDB8005(t)

	exe := NewDefaultQueryExecutor(1, 3)
	assert.Equal(t, 1, exe.Current)
	assert.Equal(t, 3, exe.PageSize)

	var rows []pagRow8005
	res, err := exe.Execute(db.Model(&pagRow8005{}).Order("name ASC"), &rows)
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.Total)
	assert.Equal(t, 1, res.Current)
	assert.Equal(t, 3, res.PageSize)
	require.Len(t, rows, 3)
	assert.Equal(t, "a", rows[0].Name)
	assert.Equal(t, "b", rows[1].Name)
	assert.Equal(t, "c", rows[2].Name)

	// page 2 → 剩余 d, e(独立 slice:GORM Find 不重置 dest,故用新变量)。
	rows2 := []pagRow8005{}
	res2, err := NewDefaultQueryExecutor(2, 3).Execute(db.Model(&pagRow8005{}).Order("name ASC"), &rows2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), res2.Total)
	assert.Equal(t, 2, res2.Current)
	require.Len(t, rows2, 2)
	assert.Equal(t, "d", rows2[0].Name)
	assert.Equal(t, "e", rows2[1].Name)
}