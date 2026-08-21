// =====================================================================
// knowledge_cache_impl_test.go — covers knowledge_cache_impl.go (85 stmts)
// Pattern: portwrite pure-mock (interface assertion + testify/mock, D-02).
// CacheProvider is fully mocked; base KnowledgeService runs on glebarez
// sqlite in-memory (unavoidable gorm path — plan allows minimal sqlite).
// Per Phase 73 Plan 03 — IMP-05 (services/knowledge)
//
// Fixture notes:
//   - Seeds use raw db.Create (never the service) so seeding cannot
//     trigger cache invalidation expectations; testify m.Called PANICS on
//     unexpected calls (lesson from the duty file's prior partial run).
//   - KnowledgeArticle.Status / KnowledgeCategory.Status default to 1
//     (published); GORM skips zero values on Create, so draft (0) rows
//     are seeded via create-then-Update-column.
// =====================================================================

package knowledge

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// Compile-time interface assertion — locks mockability contract (D-02).
var _ systemServices.CacheProvider = (*mockCacheProvider)(nil)

// mockCacheProvider embeds mock.Mock and implements the CacheProvider
// interface. GetOrSet/Delete/DeleteByPattern go through m.Called; the
// remaining methods are deterministic no-ops (never invoked by the impl).
type mockCacheProvider struct {
	mock.Mock
}

// GetOrSet mock:
//   - error return → propagate WITHOUT invoking query (cache failure path)
//   - nil error → invoke query (cache miss → DB fallback), then populate
//     dest via reflection (same semantics as NoOpCacheProvider.setValue).
func (m *mockCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	args := m.Called(ctx, key, dest, expiration, query)
	if err := args.Error(0); err != nil {
		return err
	}
	result, err := query()
	if err != nil {
		return err
	}
	setMockDest(dest, result)
	return nil
}

func (m *mockCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

// Untouched-by-impl methods: deterministic no-ops (never asserted).
func (m *mockCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *mockCacheProvider) MDelete(ctx context.Context, keys ...string) error { return nil }
func (m *mockCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}
func (m *mockCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (m *mockCacheProvider) GetStats(ctx context.Context) (*systemServices.CacheStats, error) {
	return &systemServices.CacheStats{}, nil
}

// setMockDest reflect-assigns query() result into the dest pointer passed
// by the service (same semantics as system.NoOpCacheProvider.setValue).
// Pointer results (e.g. *KnowledgeArticle) are dereferenced first.
func setMockDest(dest interface{}, value interface{}) {
	if dest == nil || value == nil {
		return
	}
	dv := reflect.ValueOf(dest)
	if dv.Kind() != reflect.Ptr {
		return
	}
	elem := dv.Elem()
	vv := reflect.ValueOf(value)
	if vv.Kind() == reflect.Ptr {
		if vv.IsNil() {
			return
		}
		vv = vv.Elem()
	}
	if elem.IsValid() && vv.IsValid() && vv.Type().AssignableTo(elem.Type()) {
		elem.Set(vv)
	}
}

// newKnowledgeTestDB creates a sqlite in-memory DB with every table the
// base KnowledgeService touches.
//
// DSN uses a UNIQUE NAMED shared-cache memory DB (not bare ":memory:"):
// bare :memory: is PRIVATE PER CONNECTION — CreateKnowledgeArticle's tag
// path queries s.db while a Transaction holds another pooled connection,
// which would otherwise hit an empty second DB ("no such table").
// The unique name isolates each test from siblings.
func newKnowledgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:kbtest_%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.KnowledgeCategory{},
		&models.KnowledgeArticle{},
		&models.KnowledgeTag{},
		&models.KnowledgeArticleTag{},
		&models.WorkOrder{},
	))
	return db
}

// newKnowledgeSvcOver wires a fresh cache impl + fresh mock cache over an
// existing DB.
func newKnowledgeSvcOver(db *gorm.DB) (*knowledgeCacheServiceImpl, *mockCacheProvider) {
	cache := new(mockCacheProvider)
	return &knowledgeCacheServiceImpl{
		base:   services.NewKnowledgeService(db),
		cache:  cache,
		config: nil,
	}, cache
}

func newKnowledgeTestService(t *testing.T) (*gorm.DB, *knowledgeCacheServiceImpl, *mockCacheProvider) {
	t.Helper()
	db := newKnowledgeTestDB(t)
	svc, cache := newKnowledgeSvcOver(db)
	return db, svc, cache
}

// ---- raw seed helpers (bypass the cache impl entirely) ----

func seedCategory(t *testing.T, db *gorm.DB, name string, parentID *string) *models.KnowledgeCategory {
	t.Helper()
	category := &models.KnowledgeCategory{
		CategoryName: name,
		ParentID:     parentID,
		SortOrder:    0,
		Status:       models.KnowledgeArticleStatusPublished,
	}
	require.NoError(t, db.Create(category).Error)
	return category
}

// seedArticle creates a published article (default status survives zero
// value skip); pass draft to additionally flip the status column.
func seedArticle(t *testing.T, db *gorm.DB, title, content, categoryID string, draft bool) *models.KnowledgeArticle {
	t.Helper()
	article := &models.KnowledgeArticle{
		Title:      title,
		Content:    content,
		Summary:    "summary of " + title,
		CategoryID: categoryID,
		Status:     models.KnowledgeArticleStatusPublished,
	}
	require.NoError(t, db.Create(article).Error)
	if draft {
		require.NoError(t, db.Model(&models.KnowledgeArticle{}).Where("id = ?", article.ID).
			Update("status", int(models.KnowledgeArticleStatusDraft)).Error)
	}
	return article
}

func seedTag(t *testing.T, db *gorm.DB, name string, useCount int) *models.KnowledgeTag {
	t.Helper()
	tag := &models.KnowledgeTag{TagName: name, UseCount: useCount}
	require.NoError(t, db.Create(tag).Error)
	return tag
}

func seedWorkOrder(t *testing.T, db *gorm.DB, status models.WorkOrderStatus) *models.WorkOrder {
	t.Helper()
	order := &models.WorkOrder{
		Title:       "wo-" + uuid.NewString()[:8],
		WorkOrderNo: "WO-" + uuid.NewString()[:8],
		CategoryID:  uuid.NewString(),
		Type:        "fault",
		Status:      status,
		SubmitterID: uuid.NewString(),
	}
	require.NoError(t, db.Create(order).Error)
	return order
}

// assertNoCacheInteraction guards that uncached paths never touch the
// cache provider.
func assertNoCacheInteraction(t *testing.T, cache *mockCacheProvider) {
	t.Helper()
	cache.AssertNotCalled(t, "GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "DeleteByPattern", mock.Anything, mock.Anything)
}

// ==================== Smoke / constructor ====================

// TestKnowledgeService_CompileOnly — smoke test ensures file compiles.
func TestKnowledgeService_CompileOnly(t *testing.T) {
	svc, cache := newKnowledgeSvcOver(newKnowledgeTestDB(t))
	assert.NotNil(t, svc)
	assert.NotNil(t, cache)
}

// TestKnowledgeService_NewKnowledgeServiceWithCache — constructor returns
// a KnowledgeCacheService implementation.
func TestKnowledgeService_NewKnowledgeServiceWithCache(t *testing.T) {
	db := newKnowledgeTestDB(t)
	var svc KnowledgeCacheService = NewKnowledgeServiceWithCache(db, new(mockCacheProvider), nil)
	assert.NotNil(t, svc)
}

// ==================== Article list / statistics (uncached) ====================

func TestKnowledgeService_GetKnowledgeArticleList_Empty(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	list, total, err := svc.GetKnowledgeArticleList(context.Background(), &services.KnowledgeArticleListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_GetKnowledgeArticleList_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "netops", nil)
	seedArticle(t, db, "BGP basics", "bgp content", category.ID, false)
	seedArticle(t, db, "OSPF deep dive", "ospf content", category.ID, false)

	list, total, err := svc.GetKnowledgeArticleList(context.Background(), &services.KnowledgeArticleListRequest{
		Title: "BGP",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "BGP basics", list[0].Title)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_GetArticleStatistics_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "stats-cat", nil)
	seedArticle(t, db, "published-1", "c1", category.ID, false)
	seedArticle(t, db, "draft-1", "c2", category.ID, true)

	stats, err := svc.GetArticleStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.Total)
	assert.Equal(t, int64(1), stats.Draft)
	assert.Equal(t, int64(1), stats.Published)
	assertNoCacheInteraction(t, cache)
}

// ==================== GetKnowledgeArticle (cached) ====================

func TestKnowledgeService_GetKnowledgeArticle_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	wantErr := errors.New("redis down")
	cache.On("GetOrSet", mock.Anything, "kb:article:art-1", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	article, err := svc.GetKnowledgeArticle(context.Background(), "art-1")
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, article)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_GetKnowledgeArticle_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedArticle(t, db, "cached article", "body", uuid.NewString(), false)

	cache.On("GetOrSet", mock.Anything, "kb:article:"+seeded.ID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	article, err := svc.GetKnowledgeArticle(context.Background(), seeded.ID)
	require.NoError(t, err)
	require.NotNil(t, article)
	assert.Equal(t, seeded.ID, article.ID)
	assert.Equal(t, "cached article", article.Title)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_GetKnowledgeArticle_NotFound_Error(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("GetOrSet", mock.Anything, "kb:article:missing", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	article, err := svc.GetKnowledgeArticle(context.Background(), "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "知识库文章不存在")
	assert.Nil(t, article)
	cache.AssertExpectations(t)
}

// ==================== Create / Update / Delete article ====================

// CreateKnowledgeArticle deliberately does NOT invalidate any cache (new
// article has no cached entry yet) — lock that contract.
func TestKnowledgeService_CreateKnowledgeArticle_Success_NoInvalidation(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "create-cat", nil)

	article, err := svc.CreateKnowledgeArticle(context.Background(), &services.KnowledgeArticleCreateRequest{
		Title:      "new article",
		Content:    "content body",
		CategoryID: category.ID,
		Status:     1,
	}, "creator-1")
	require.NoError(t, err)
	require.NotNil(t, article)
	assert.Equal(t, "new article", article.Title)
	assert.NotEmpty(t, article.ID)
	assertNoCacheInteraction(t, cache)
}

// TestKnowledgeService_CreateKnowledgeArticle_WithTagNames — the base
// Create resolves non-UUID tag inputs via GetOrCreateTag. The tag is
// PRE-SEEDED so the in-transaction lookup is read-only: creating a tag on
// s.db while the tx holds the shared-cache write lock would self-deadlock
// (tx waits for the tag query; tag INSERT waits for the tx).
func TestKnowledgeService_CreateKnowledgeArticle_WithTagNames(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "tag-cat", nil)
	seedTag(t, db, "networking", 0)

	article, err := svc.CreateKnowledgeArticle(context.Background(), &services.KnowledgeArticleCreateRequest{
		Title:      "tagged article",
		Content:    "content",
		CategoryID: category.ID,
		Status:     1,
		TagIDs:     []string{"networking"}, // non-UUID → GetOrCreateTag lookup path
	}, "creator-1")
	require.NoError(t, err)
	require.NotNil(t, article)

	var gotTag models.KnowledgeTag
	require.NoError(t, db.Where("tag_name = ?", "networking").First(&gotTag).Error)
	assert.Equal(t, 1, gotTag.UseCount) // incremented by association
	var assocCount int64
	require.NoError(t, db.Model(&models.KnowledgeArticleTag{}).Where("article_id = ?", article.ID).Count(&assocCount).Error)
	assert.Equal(t, int64(1), assocCount)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_UpdateKnowledgeArticle_Success_InvalidatesArticleCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedArticle(t, db, "before update", "content", uuid.NewString(), false)

	cache.On("Delete", mock.Anything, "kb:article:"+seeded.ID).Return(nil).Once()

	newTitle := "after update"
	err := svc.UpdateKnowledgeArticle(context.Background(), seeded.ID, &services.KnowledgeArticleUpdateRequest{
		Title: &newTitle,
	}, "operator-1")
	require.NoError(t, err)

	var got models.KnowledgeArticle
	require.NoError(t, db.Where("id = ?", seeded.ID).First(&got).Error)
	assert.Equal(t, "after update", got.Title)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_UpdateKnowledgeArticle_NotFound_Error_NoCacheCall(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	newTitle := "ghost"
	err := svc.UpdateKnowledgeArticle(context.Background(), "missing", &services.KnowledgeArticleUpdateRequest{
		Title: &newTitle,
	}, "operator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "知识库文章不存在")
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_DeleteKnowledgeArticle_Success_InvalidatesArticleCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedArticle(t, db, "to delete", "content", uuid.NewString(), false)

	cache.On("Delete", mock.Anything, "kb:article:"+seeded.ID).Return(nil).Once()

	err := svc.DeleteKnowledgeArticle(context.Background(), seeded.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.KnowledgeArticle{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_DeleteKnowledgeArticle_NotFound_Error_NoCacheCall(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	err := svc.DeleteKnowledgeArticle(context.Background(), "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "知识库文章不存在")
	assertNoCacheInteraction(t, cache)
}

// ==================== Counters (uncached delegation) ====================

func TestKnowledgeService_IncrementViewCount_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedArticle(t, db, "views", "content", uuid.NewString(), false)

	require.NoError(t, svc.IncrementViewCount(context.Background(), seeded.ID))
	require.NoError(t, svc.IncrementViewCount(context.Background(), seeded.ID))

	var got models.KnowledgeArticle
	require.NoError(t, db.Where("id = ?", seeded.ID).First(&got).Error)
	assert.Equal(t, 2, got.ViewCount)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_IncrementLikeCount_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedArticle(t, db, "likes", "content", uuid.NewString(), false)

	require.NoError(t, svc.IncrementLikeCount(context.Background(), seeded.ID))

	var got models.KnowledgeArticle
	require.NoError(t, db.Where("id = ?", seeded.ID).First(&got).Error)
	assert.Equal(t, 1, got.LikeCount)
	assertNoCacheInteraction(t, cache)
}

// ==================== Search (uncached delegation) ====================

func TestKnowledgeService_SearchKnowledgeArticles_Keyword(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "search-cat", nil)
	seedArticle(t, db, "BGP troubleshooting", "border gateway protocol", category.ID, false)
	seedArticle(t, db, "DNS basics", "domain name system", category.ID, false)
	seedArticle(t, db, "BGP draft (hidden)", "border gateway draft", category.ID, true) // draft excluded

	list, total, err := svc.SearchKnowledgeArticles(context.Background(), &services.SearchKnowledgeRequest{
		Keyword: "BGP",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "BGP troubleshooting", list[0].Title)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_SearchKnowledgeArticles_Empty(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	list, total, err := svc.SearchKnowledgeArticles(context.Background(), &services.SearchKnowledgeRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	assertNoCacheInteraction(t, cache)
}

// ==================== ConvertWorkOrderToArticle (uncached delegation) ====================

func TestKnowledgeService_ConvertWorkOrderToArticle_WorkOrderMissing_Error(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	article, err := svc.ConvertWorkOrderToArticle(context.Background(), uuid.NewString(), &services.ConvertWorkOrderToArticleRequest{
		Title:      "t",
		Content:    "c",
		CategoryID: uuid.NewString(),
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "工单不存在")
	assert.Nil(t, article)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_ConvertWorkOrderToArticle_NotCompleted_Error(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	order := seedWorkOrder(t, db, models.WorkOrderStatusPending)

	article, err := svc.ConvertWorkOrderToArticle(context.Background(), order.ID, &services.ConvertWorkOrderToArticleRequest{
		Title:      "t",
		Content:    "c",
		CategoryID: uuid.NewString(),
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "只有已完成或已关闭的工单")
	assert.Nil(t, article)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_ConvertWorkOrderToArticle_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	order := seedWorkOrder(t, db, models.WorkOrderStatusCompleted)
	category := seedCategory(t, db, "convert-cat", nil)

	article, err := svc.ConvertWorkOrderToArticle(context.Background(), order.ID, &services.ConvertWorkOrderToArticleRequest{
		Title:      "solved: how to fix X",
		Content:    "the fix is Y",
		CategoryID: category.ID,
		Status:     1,
	}, "creator-1")
	require.NoError(t, err)
	require.NotNil(t, article)
	assert.Equal(t, order.ID, *article.SourceWorkOrderID)

	// Second conversion of the same work order must be rejected.
	_, err = svc.ConvertWorkOrderToArticle(context.Background(), order.ID, &services.ConvertWorkOrderToArticleRequest{
		Title:      "dup",
		Content:    "c",
		CategoryID: category.ID,
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已经转换")
	assertNoCacheInteraction(t, cache)
}

// ==================== Category list (cached, key construction) ====================

// TestKnowledgeService_GetKnowledgeCategoryList_CacheKeyVariants — locks
// the cache-key construction branches:
//   - no parent, no status  → "kb:category:tree"
//   - parent set            → "kb:category:parent:<id>"
//   - parent + status set   → "kb:category:parent:<id>:status:<n>"
func TestKnowledgeService_GetKnowledgeCategoryList_CacheKeyVariants(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	parent := seedCategory(t, db, "parent-cat", nil)
	seedCategory(t, db, "child-cat", &parent.ID)

	tests := []struct {
		name    string
		req     services.KnowledgeCategoryListRequest
		wantKey string
	}{
		{
			name:    "default_tree_key",
			req:     services.KnowledgeCategoryListRequest{},
			wantKey: "kb:category:tree",
		},
		{
			name:    "parent_key",
			req:     services.KnowledgeCategoryListRequest{ParentID: &parent.ID},
			wantKey: "kb:category:parent:" + parent.ID,
		},
		{
			name:    "parent_plus_status_key",
			req:     services.KnowledgeCategoryListRequest{ParentID: &parent.ID, Status: intPtr(1)},
			wantKey: "kb:category:parent:" + parent.ID + ":status:1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.On("GetOrSet", mock.Anything, tt.wantKey, mock.Anything, mock.Anything, mock.Anything).
				Return(nil).Once()
			list, err := svc.GetKnowledgeCategoryList(context.Background(), &tt.req)
			require.NoError(t, err)
			assert.NotNil(t, list)
			cache.AssertExpectations(t)
		})
	}
}

func TestKnowledgeService_GetKnowledgeCategoryList_TreeStructure(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	parent := seedCategory(t, db, "root-cat", nil)
	seedCategory(t, db, "leaf-cat", &parent.ID)

	cache.On("GetOrSet", mock.Anything, "kb:category:tree", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	list, err := svc.GetKnowledgeCategoryList(context.Background(), &services.KnowledgeCategoryListRequest{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "root-cat", list[0].CategoryName)
	require.Len(t, list[0].Children, 1)
	assert.Equal(t, "leaf-cat", list[0].Children[0].CategoryName)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_GetKnowledgeCategoryList_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	wantErr := errors.New("redis timeout")
	cache.On("GetOrSet", mock.Anything, "kb:category:tree", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	list, err := svc.GetKnowledgeCategoryList(context.Background(), &services.KnowledgeCategoryListRequest{})
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, list)
	cache.AssertExpectations(t)
}

// ==================== Category detail / CRUD ====================

func TestKnowledgeService_GetKnowledgeCategory_Found(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedCategory(t, db, "detail-cat", nil)

	category, err := svc.GetKnowledgeCategory(context.Background(), seeded.ID)
	require.NoError(t, err)
	require.NotNil(t, category)
	assert.Equal(t, "detail-cat", category.CategoryName)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_GetKnowledgeCategory_NotFound_Error(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	category, err := svc.GetKnowledgeCategory(context.Background(), "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "知识库分类不存在")
	assert.Nil(t, category)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_CreateKnowledgeCategory_Success_InvalidatesCategoryCache(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "kb:category:*").Return(nil).Once()

	category, err := svc.CreateKnowledgeCategory(context.Background(), &services.KnowledgeCategoryCreateRequest{
		CategoryName: "fresh-cat",
		Status:       1,
	}, "creator-1")
	require.NoError(t, err)
	require.NotNil(t, category)
	assert.Equal(t, "fresh-cat", category.CategoryName)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_CreateKnowledgeCategory_Duplicate_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seedCategory(t, db, "dup-cat", nil)

	category, err := svc.CreateKnowledgeCategory(context.Background(), &services.KnowledgeCategoryCreateRequest{
		CategoryName: "dup-cat",
		Status:       1,
	}, "creator-1")
	assert.Error(t, err) // unique index on category_name
	assert.Nil(t, category)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_UpdateKnowledgeCategory_Success_InvalidatesCategoryCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedCategory(t, db, "rename-cat", nil)

	cache.On("DeleteByPattern", mock.Anything, "kb:category:*").Return(nil).Once()

	newName := "renamed-cat"
	err := svc.UpdateKnowledgeCategory(context.Background(), seeded.ID, &services.KnowledgeCategoryUpdateRequest{
		CategoryName: &newName,
	}, "operator-1")
	require.NoError(t, err)

	var got models.KnowledgeCategory
	require.NoError(t, db.Where("id = ?", seeded.ID).First(&got).Error)
	assert.Equal(t, "renamed-cat", got.CategoryName)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_UpdateKnowledgeCategory_NotFound_Error_NoCacheCall(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	newName := "ghost"
	err := svc.UpdateKnowledgeCategory(context.Background(), "missing", &services.KnowledgeCategoryUpdateRequest{
		CategoryName: &newName,
	}, "operator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "知识库分类不存在")
	assertNoCacheInteraction(t, cache)
}

// TestKnowledgeService_DeleteKnowledgeCategory_Success_InvalidatesCategoryCache
// uses a digit-string PK: base service passes the id as a bare inline
// condition to db.Delete, and GORM only quotes all-digit strings as PK
// values (UUID strings would be interpolated as raw SQL → syntax error;
// latent base-service quirk, not fixed per D-12).
func TestKnowledgeService_DeleteKnowledgeCategory_Success_InvalidatesCategoryCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := &models.KnowledgeCategory{
		CategoryName: "del-cat",
		Status:       models.KnowledgeArticleStatusPublished,
	}
	seeded.ID = "123456"
	require.NoError(t, db.Create(seeded).Error)

	cache.On("DeleteByPattern", mock.Anything, "kb:category:*").Return(nil).Once()

	err := svc.DeleteKnowledgeCategory(context.Background(), seeded.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.KnowledgeCategory{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_DeleteKnowledgeCategory_HasChildren_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	parent := seedCategory(t, db, "parent-del", nil)
	seedCategory(t, db, "child-del", &parent.ID)

	err := svc.DeleteKnowledgeCategory(context.Background(), parent.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "子分类")
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_DeleteKnowledgeCategory_HasArticles_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	category := seedCategory(t, db, "busy-cat", nil)
	seedArticle(t, db, "attached", "content", category.ID, false)

	err := svc.DeleteKnowledgeCategory(context.Background(), category.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "关联文章")
	assertNoCacheInteraction(t, cache)
}

// ==================== Tags (cached list + invalidation) ====================

func TestKnowledgeService_GetAllTags_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seedTag(t, db, "popular", 10)
	seedTag(t, db, "rare", 1)

	cache.On("GetOrSet", mock.Anything, "kb:tags:all", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	tags, err := svc.GetAllTags(context.Background())
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "popular", tags[0].TagName) // ordered by use_count DESC
	assert.Equal(t, "rare", tags[1].TagName)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_GetAllTags_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	wantErr := errors.New("redis gone")
	cache.On("GetOrSet", mock.Anything, "kb:tags:all", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	tags, err := svc.GetAllTags(context.Background())
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, tags)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_GetTagByName_Found(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedTag(t, db, "named-tag", 3)

	tag, err := svc.GetTagByName(context.Background(), "named-tag")
	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Equal(t, seeded.ID, tag.ID)
	assertNoCacheInteraction(t, cache)
}

// GetTagByName returns (nil, nil) when not found — a documented base
// service contract relied on by GetOrCreateTag.
func TestKnowledgeService_GetTagByName_NotFound_NilNil(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	tag, err := svc.GetTagByName(context.Background(), "no-such-tag")
	assert.NoError(t, err)
	assert.Nil(t, tag)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_CreateTag_Success_InvalidatesTagCache(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("Delete", mock.Anything, "kb:tags:all").Return(nil).Once()

	tag, err := svc.CreateTag(context.Background(), "brand-new")
	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Equal(t, "brand-new", tag.TagName)
	cache.AssertExpectations(t)
}

func TestKnowledgeService_CreateTag_Duplicate_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seedTag(t, db, "existing-tag", 0)

	tag, err := svc.CreateTag(context.Background(), "existing-tag")
	assert.Error(t, err) // unique index on tag_name
	assert.Nil(t, tag)
	assertNoCacheInteraction(t, cache)
}

func TestKnowledgeService_UpdateTag_Success_InvalidatesTagCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := seedTag(t, db, "old-name", 0)

	cache.On("Delete", mock.Anything, "kb:tags:all").Return(nil).Once()

	err := svc.UpdateTag(context.Background(), seeded.ID, "new-name")
	require.NoError(t, err)

	var got models.KnowledgeTag
	require.NoError(t, db.Where("id = ?", seeded.ID).First(&got).Error)
	assert.Equal(t, "new-name", got.TagName)
	cache.AssertExpectations(t)
}

// TestKnowledgeService_DeleteTag_Success_InvalidatesTagCache — same
// digit-string PK technique as DeleteKnowledgeCategory (GORM inline
// condition quirk with non-digit string PKs).
func TestKnowledgeService_DeleteTag_Success_InvalidatesTagCache(t *testing.T) {
	db, svc, cache := newKnowledgeTestService(t)
	seeded := &models.KnowledgeTag{TagName: "doomed-tag", UseCount: 0}
	seeded.ID = "654321"
	require.NoError(t, db.Create(seeded).Error)

	cache.On("Delete", mock.Anything, "kb:tags:all").Return(nil).Once()

	err := svc.DeleteTag(context.Background(), seeded.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.KnowledgeTag{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

// ==================== Cache invalidation methods ====================

func TestKnowledgeService_InvalidateCategoryCache_DeletesPattern(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "kb:category:*").Return(nil).Once()
	assert.NoError(t, svc.InvalidateCategoryCache(context.Background()))
	cache.AssertExpectations(t)
}

func TestKnowledgeService_InvalidateTagCache_DeletesKey(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("Delete", mock.Anything, "kb:tags:all").Return(nil).Once()
	assert.NoError(t, svc.InvalidateTagCache(context.Background()))
	cache.AssertExpectations(t)
}

func TestKnowledgeService_InvalidateArticleCache_DeletesKey(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("Delete", mock.Anything, "kb:article:art-9").Return(nil).Once()
	assert.NoError(t, svc.InvalidateArticleCache(context.Background(), "art-9"))
	cache.AssertExpectations(t)
}

func TestKnowledgeService_InvalidateAllArticleCache_DeletesPattern(t *testing.T) {
	_, svc, cache := newKnowledgeTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "kb:article:*").Return(nil).Once()
	assert.NoError(t, svc.InvalidateAllArticleCache(context.Background()))
	cache.AssertExpectations(t)
}

// ==================== getExpiration helper ====================

func TestKnowledgeService_GetExpiration_NilConfig_ReturnsDefault(t *testing.T) {
	_, svc, _ := newKnowledgeTestService(t)
	got := svc.getExpiration("cache.kb.article", 10*time.Minute)
	assert.Equal(t, 10*time.Minute, got)
}

// intPtr is a small helper for optional int filters.
func intPtr(i int) *int { return &i }
