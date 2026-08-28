package base

// =====================================================================
// Phase 80-05 Task 4a: GORMRepository 泛型 CRUD/List/BatchDelete +
// WrapError/IsNotFound/IsDuplicate(sqlite)。
//
// 纪律:
//   - glebarez sqlite(t.TempDir 文件库);内嵌 models.BaseModel 作为测试行
//     模型;BaseModel 自带 BeforeCreate 自动填 uuid。
//   - 零 sleep、零 t.Parallel(共享 sqlite fixture)。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// basRepoRow8005 测试行模型:嵌 BaseModel + Name + Age。
type basRepoRow8005 struct {
	models.BaseModel
	Name string `gorm:"size:100;not null"`
	Age  int    `gorm:"default:0"`
}

// TableName 表名。
func (basRepoRow8005) TableName() string { return "bas_repo_rows_8005" }

func newBasRepo8005(t *testing.T) (*GORMRepository[basRepoRow8005], *gorm.DB) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "base8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(&basRepoRow8005{}))
	return NewGORMRepository[basRepoRow8005](gormDB), gormDB
}

func seedBasRow8005(t *testing.T, repo *GORMRepository[basRepoRow8005], name string, age int) *basRepoRow8005 {
	t.Helper()
	row := &basRepoRow8005{Name: name, Age: age}
	require.NoError(t, repo.Create(context.Background(), row))
	return row
}

// TestBas8005_CRUD_RoundTrip:Create → GetByID → Update → Delete 全链;
// GetByID 不存在 → ErrRecordNotFound。
func TestBas8005_CRUD_RoundTrip(t *testing.T) {
	repo, _ := newBasRepo8005(t)
	ctx := context.Background()

	row := &basRepoRow8005{Name: "alice", Age: 30}
	require.NoError(t, repo.Create(ctx, row))
	assert.NotEmpty(t, row.ID, "BaseModel.BeforeCreate 应自动填 uuid")

	got, err := repo.GetByID(ctx, row.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Name)
	assert.Equal(t, 30, got.Age)

	got.Name = "alice-2"
	got.Age = 31
	require.NoError(t, repo.Update(ctx, got))

	got2, err := repo.GetByID(ctx, row.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice-2", got2.Name)
	assert.Equal(t, 31, got2.Age)

	require.NoError(t, repo.Delete(ctx, row.ID))
	_, err = repo.GetByID(ctx, row.ID)
	assert.True(t, IsNotFound(err), "删除后 GetByID 应返回 ErrRecordNotFound")
}

// TestBas8005_List_Paged:5 行种子 → List with Where + Order + Limit + Offset +
// 操作符覆盖(= != > < >= <= LIKE IN)+ 空 Where 全部行。
func TestBas8005_List_Paged(t *testing.T) {
	repo, _ := newBasRepo8005(t)
	ctx := context.Background()

	for i, n := range []string{"alice", "bob", "carol", "dave", "eve"} {
		_ = i
		seedBasRow8005(t, repo, n, 20+i)
	}

	// 基本分页:page 1 size 2 + 排序 name asc → alice, bob。
	page1, err := repo.List(ctx, &Query{OrderBy: "name ASC", Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), page1.Total)
	assert.Equal(t, 1, page1.Current)
	assert.Equal(t, 2, page1.PageSize)
	rows1 := page1.List.([]basRepoRow8005)
	require.Len(t, rows1, 2)
	assert.Equal(t, "alice", rows1[0].Name)
	assert.Equal(t, "bob", rows1[1].Name)

	// page 2 size 2 → carol, dave。
	page2, err := repo.List(ctx, &Query{OrderBy: "name ASC", Offset: 2, Limit: 2})
	require.NoError(t, err)
	rows2 := page2.List.([]basRepoRow8005)
	require.Len(t, rows2, 2)
	assert.Equal(t, "carol", rows2[0].Name)
	assert.Equal(t, "dave", rows2[1].Name)
	assert.Equal(t, 2, page2.Current)

	// 操作符覆盖(各分支配对 sqlite 类型断言)。
	ops := []struct {
		op   string
		val  interface{}
	}{
		{"=", "bob"},
		{"!=", "bob"},
		{">", 20},
		{"<", 23},
		{">=", 23},
		{"<=", 21},
		{"LIKE", "%a%"},
	}
	for _, tc := range ops {
		_, err := repo.List(ctx, &Query{
			Where:   []WhereCondition{{Field: "name", Operator: tc.op, Value: tc.val}},
			OrderBy: "name ASC",
			Limit:   100,
		})
		assert.NoError(t, err, "操作符 %s 应可执行", tc.op)
	}

	// IN:[]int 多值命中断言(20,22 → alice, carol)。
	res, err := repo.List(ctx, &Query{
		Where:   []WhereCondition{{Field: "age", Operator: "IN", Value: []int{20, 22}}},
		OrderBy: "name ASC",
		Limit:   100,
	})
	require.NoError(t, err)
	rows := res.List.([]basRepoRow8005)
	require.Len(t, rows, 2)
	assert.Equal(t, "alice", rows[0].Name)
	assert.Equal(t, "carol", rows[1].Name)

	// 空 Where 列表 → 全部行。
	all, err := repo.List(ctx, &Query{OrderBy: "age ASC", Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(5), all.Total)
}

// TestBas8005_BatchDelete:3 ID 批删 → 剩余计数;空切片 → BadRequest 分支。
func TestBas8005_BatchDelete(t *testing.T) {
	repo, db := newBasRepo8005(t)
	ctx := context.Background()

	r1 := seedBasRow8005(t, repo, "a", 1)
	r2 := seedBasRow8005(t, repo, "b", 2)
	r3 := seedBasRow8005(t, repo, "c", 3)
	seedBasRow8005(t, repo, "survivor", 4)

	require.NoError(t, repo.BatchDelete(ctx, []string{r1.ID, r2.ID, r3.ID}))

	var count int64
	require.NoError(t, db.Model(&basRepoRow8005{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// 空切片 → BadRequest 错误。
	err := repo.BatchDelete(ctx, nil)
	require.Error(t, err)
	assert.True(t, apperrors.IsAppError(err), "空切片应包装为 AppError")
	assert.Equal(t, apperrors.CodeParamError, apperrors.GetAppError(err).GetCode())
}

// TestBas8005_ErrorHelpers:WrapError + IsNotFound + IsDuplicate 三连。
func TestBas8005_ErrorHelpers(t *testing.T) {
	// WrapError nil → nil。
	assert.Nil(t, WrapError(nil, "ignored"))

	// WrapError err → message: %w 包装。
	wrapped := WrapError(assert.AnError, "context-prefix")
	require.Error(t, wrapped)
	assert.Contains(t, wrapped.Error(), "context-prefix")
	assert.ErrorIs(t, wrapped, assert.AnError)

	// IsNotFound:gorm.ErrRecordNotFound → true。
	assert.True(t, IsNotFound(gorm.ErrRecordNotFound))

	// IsNotFound:app error NotFound → true。
	notFound := apperrors.RecordNotFoundWithMsg("missing")
	assert.True(t, IsNotFound(notFound))

	// IsNotFound:其他错误 → false。
	assert.False(t, IsNotFound(assert.AnError))

	// IsDuplicate:app error Exists → true。
	dup := apperrors.RecordExists()
	assert.True(t, IsDuplicate(dup))

	// IsDuplicate:其他错误 → false。
	assert.False(t, IsDuplicate(assert.AnError))
	assert.False(t, IsDuplicate(gorm.ErrRecordNotFound))
}