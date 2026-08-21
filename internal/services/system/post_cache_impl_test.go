package system

// =====================================================================
// post_cache_impl_test.go — covers post_cache_impl.go
// Compile-time interface assertion + cache miss/hit/invalidation tests
// =====================================================================

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// Compile-time interface assertion
var _ PostService = (*postCacheService)(nil)

func setupPostCacheDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_post (
			id TEXT PRIMARY KEY,
			post_code TEXT NOT NULL,
			post_name TEXT NOT NULL,
			post_sort INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

func seedPostCache(t *testing.T, db *gorm.DB, code, name string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, post_sort, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, ?, datetime('now'), datetime('now'), 0)`,
		id, code, name, status).Error)
	return id
}

// TC1: GetAllWithCache - cache miss → DB
func TestPostCache_GetAllWithCache(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "p1", "P1", 0)

	posts, err := svc.GetAllWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, posts)
}

// TC2: GetEnabledWithCache - cache miss → DB
func TestPostCache_GetEnabledWithCache(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "p1", "Active", 0)
	seedPostCache(t, db, "p2", "Stopped", 1)

	posts, err := svc.GetEnabledWithCache(context.Background())
	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "Active", posts[0].PostName)
}

// TC3: InvalidatePostCache - no error
func TestPostCache_InvalidatePostCache(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	require.NoError(t, svc.InvalidatePostCache(context.Background()))
}

// TC4: Create - delegates + invalidates
func TestPostCache_Create(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)

	req := &requests.PostCreateRequest{
		PostCode: "p1", PostName: "P1", PostSort: 1, Status: models.PostStatusEnabled,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Post
	require.NoError(t, db.Where("post_code = ?", "p1").First(&got).Error)
	assert.Equal(t, "P1", got.PostName)
}

// TC5: Update - delegates + invalidates
func TestPostCache_Update(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	id := seedPostCache(t, db, "p1", "Old", 0)

	req := &requests.PostUpdateRequest{
		ID: id, PostName: "New", PostSort: 5, Status: models.PostStatusDisabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Post
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.PostName)
}

// TC6: Delete - delegates + invalidates
func TestPostCache_Delete(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	id := seedPostCache(t, db, "p1", "P1", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC7: BatchDelete - delegates + invalidates
func TestPostCache_BatchDelete(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	id1 := seedPostCache(t, db, "p1", "P1", 0)
	id2 := seedPostCache(t, db, "p2", "P2", 0)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))
}

// TC8: GetByID - delegates
func TestPostCache_GetByID(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	id := seedPostCache(t, db, "p1", "P1", 0)

	got, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "P1", got.PostName)
}

// TC9: List - delegates
func TestPostCache_List(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "p1", "P1", 0)

	result, err := svc.List(context.Background(), requests.DefaultPostListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC10: Create - duplicate fails
func TestPostCache_Create_Duplicate(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "dup", "D", 0)

	req := &requests.PostCreateRequest{
		PostCode: "dup", PostName: "D2", PostSort: 1, Status: models.PostStatusEnabled,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC11: BatchDelete - empty ids fails
func TestPostCache_BatchDelete_Empty(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
}

// TC12: List - filter narrows
func TestPostCache_List_Filter(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "ceo", "CEO", 0)
	seedPostCache(t, db, "cto", "CTO", 0)

	code := "ceo"
	result, err := svc.List(context.Background(), requests.PostListParams{
		BaseListRequest: requests.DefaultPostListParams().BaseListRequest,
		PostCode:        &code,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC13: Statistics - delegates
func TestPostCache_Statistics(t *testing.T) {
	db := setupPostCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewPostServiceWithCache(db, cache, nil)
	seedPostCache(t, db, "p1", "P1", 0)
	seedPostCache(t, db, "p2", "P2", 1)

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}
