package system

// =====================================================================
// post_service_test.go — covers post_service.go (246 lines)
// Extends existing post_statistics_test.go (PRESERVED)
// Per Plan 72-09 Task 3
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

func setupPostServiceDB(t *testing.T) *gorm.DB {
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

func seedPostRaw(t *testing.T, db *gorm.DB, code, name string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, post_sort, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, ?, datetime('now'), datetime('now'), 0)`,
		id, code, name, status).Error)
	return id
}

// TC1: Create - success
func TestPostService_Create_Success(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	req := &requests.PostCreateRequest{
		PostCode: "p1", PostName: "P1", PostSort: 1,
		Status: models.PostStatusEnabled,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Post
	require.NoError(t, db.Where("post_code = ?", "p1").First(&got).Error)
	assert.Equal(t, "P1", got.PostName)
}

// TC2: Create - duplicate post_code fails
func TestPostService_Create_Duplicate(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "dup", "D", 0)

	req := &requests.PostCreateRequest{
		PostCode: "dup", PostName: "D2", PostSort: 1,
		Status: models.PostStatusEnabled,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC3: Update - success
func TestPostService_Update_Success(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	id := seedPostRaw(t, db, "p1", "Old", 0)

	req := &requests.PostUpdateRequest{
		ID: id, PostName: "New", PostSort: 5,
		Status: models.PostStatusDisabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Post
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.PostName)
}

// TC4: Update - not found
func TestPostService_Update_NotFound(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	req := &requests.PostUpdateRequest{
		ID: uuid.NewString(), PostName: "X", PostSort: 1, Status: models.PostStatusEnabled,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC5: Delete - success
func TestPostService_Delete_Success(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	id := seedPostRaw(t, db, "p1", "P1", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC6: Delete - not found
func TestPostService_Delete_NotFound(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC7: GetByID - success
func TestPostService_GetByID_Success(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	id := seedPostRaw(t, db, "p1", "P1", 0)

	got, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "P1", got.PostName)
}

// TC8: GetByID - not found
func TestPostService_GetByID_NotFound(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC9: List - empty
func TestPostService_List_Empty(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	result, err := svc.List(context.Background(), requests.DefaultPostListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC10: List - filter by postCode
func TestPostService_List_FilterByCode(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "ceo", "CEO", 0)
	seedPostRaw(t, db, "cto", "CTO", 0)

	code := "ceo"
	result, err := svc.List(context.Background(), requests.PostListParams{
		BaseListRequest: requests.DefaultPostListParams().BaseListRequest,
		PostCode:        &code,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC11: List - filter by postName
func TestPostService_List_FilterByName(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "c1", "董事长", 0)
	seedPostRaw(t, db, "c2", "总经理", 0)

	name := "董事"
	result, err := svc.List(context.Background(), requests.PostListParams{
		BaseListRequest: requests.DefaultPostListParams().BaseListRequest,
		PostName:        &name,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC12: List - filter by status
func TestPostService_List_FilterByStatus(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "p1", "P1", 0)
	seedPostRaw(t, db, "p2", "P2", 1)

	status := 0
	result, err := svc.List(context.Background(), requests.PostListParams{
		BaseListRequest: requests.DefaultPostListParams().BaseListRequest,
		Status:          &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC13: BatchDelete - success
func TestPostService_BatchDelete_Success(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	id1 := seedPostRaw(t, db, "p1", "P1", 0)
	id2 := seedPostRaw(t, db, "p2", "P2", 0)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))
}

// TC14: BatchDelete - empty ids fails
func TestPostService_BatchDelete_Empty(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
}

// TC15: GetAllWithCache - returns all posts
func TestPostService_GetAllWithCache(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "p1", "Active", 0)
	seedPostRaw(t, db, "p2", "Stopped", 1)

	posts, err := svc.GetAllWithCache(context.Background())
	require.NoError(t, err)
	// GetAllWithCache returns ALL posts (regardless of status)
	assert.Len(t, posts, 2)
}

// TC16: GetEnabledWithCache - returns only enabled
func TestPostService_GetEnabledWithCache(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	seedPostRaw(t, db, "p1", "Active", 0)
	seedPostRaw(t, db, "p2", "Stopped", 1)

	posts, err := svc.GetEnabledWithCache(context.Background())
	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "Active", posts[0].PostName)
}

// TC17: InvalidatePostCache - no-op
func TestPostService_InvalidatePostCache(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	require.NoError(t, svc.InvalidatePostCache(context.Background()))
}

// TC18: Update - success with full assertion
func TestPostService_Update_VerifyDB(t *testing.T) {
	db := setupPostServiceDB(t)
	svc := NewPostService(db).(*postService)
	id := seedPostRaw(t, db, "p1", "Old", 0)

	req := &requests.PostUpdateRequest{
		ID: id, PostName: "New", PostSort: 5,
		Status: models.PostStatusDisabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Post
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.PostName)
	assert.Equal(t, 5, got.PostSort)
	assert.Equal(t, models.PostStatusDisabled, got.Status)
}
