package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	knowledgeServices "github.com/xingran-next/xingran-go-backend/internal/services/knowledge"
)

// Compile-time assertion: mockKnowledgeService implements knowledgeServices.KnowledgeCacheService
var _ knowledgeServices.KnowledgeCacheService = (*mockKnowledgeService)(nil)

// mockKnowledgeService implements knowledgeServices.KnowledgeCacheService via
// function fields. Embedding the interface as a nil field allows every method
// below to delegate to the corresponding *Func field; unimplemented methods
// return zero values.
type mockKnowledgeService struct {
	knowledgeServices.KnowledgeCacheService

	// Article methods
	GetKnowledgeArticleListFunc func(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error)
	GetArticleStatisticsFunc     func(ctx context.Context) (*services.KnowledgeArticleStatistics, error)
	GetKnowledgeArticleFunc     func(ctx context.Context, id string) (*models.KnowledgeArticle, error)
	CreateKnowledgeArticleFunc  func(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error)
	UpdateKnowledgeArticleFunc  func(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error
	DeleteKnowledgeArticleFunc  func(ctx context.Context, id string) error
	IncrementViewCountFunc       func(ctx context.Context, id string) error
	IncrementLikeCountFunc       func(ctx context.Context, id string) error
	SearchKnowledgeArticlesFunc func(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error)
	ConvertWorkOrderToArticleFunc func(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error)

	// Category methods
	GetKnowledgeCategoryListFunc  func(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error)
	GetKnowledgeCategoryFunc      func(ctx context.Context, id string) (*models.KnowledgeCategory, error)
	CreateKnowledgeCategoryFunc   func(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error)
	UpdateKnowledgeCategoryFunc   func(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error
	DeleteKnowledgeCategoryFunc   func(ctx context.Context, id string) error

	// Tag methods
	GetAllTagsFunc  func(ctx context.Context) ([]models.KnowledgeTag, error)
	GetTagByNameFunc func(ctx context.Context, name string) (*models.KnowledgeTag, error)
	CreateTagFunc   func(ctx context.Context, name string) (*models.KnowledgeTag, error)
	UpdateTagFunc   func(ctx context.Context, id string, name string) error
	DeleteTagFunc   func(ctx context.Context, id string) error

	// Cache invalidation methods
	InvalidateCategoryCacheFunc   func(ctx context.Context) error
	InvalidateTagCacheFunc        func(ctx context.Context) error
	InvalidateArticleCacheFunc    func(ctx context.Context, articleID string) error
	InvalidateAllArticleCacheFunc func(ctx context.Context) error
}

// ==================== Article method overrides ====================

func (m *mockKnowledgeService) GetKnowledgeArticleList(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
	if m.GetKnowledgeArticleListFunc != nil {
		return m.GetKnowledgeArticleListFunc(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockKnowledgeService) GetArticleStatistics(ctx context.Context) (*services.KnowledgeArticleStatistics, error) {
	if m.GetArticleStatisticsFunc != nil {
		return m.GetArticleStatisticsFunc(ctx)
	}
	return &services.KnowledgeArticleStatistics{}, nil
}

func (m *mockKnowledgeService) GetKnowledgeArticle(ctx context.Context, id string) (*models.KnowledgeArticle, error) {
	if m.GetKnowledgeArticleFunc != nil {
		return m.GetKnowledgeArticleFunc(ctx, id)
	}
	return &models.KnowledgeArticle{}, nil
}

func (m *mockKnowledgeService) CreateKnowledgeArticle(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error) {
	if m.CreateKnowledgeArticleFunc != nil {
		return m.CreateKnowledgeArticleFunc(ctx, req, creatorID)
	}
	return &models.KnowledgeArticle{}, nil
}

func (m *mockKnowledgeService) UpdateKnowledgeArticle(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error {
	if m.UpdateKnowledgeArticleFunc != nil {
		return m.UpdateKnowledgeArticleFunc(ctx, id, req, operatorID)
	}
	return nil
}

func (m *mockKnowledgeService) DeleteKnowledgeArticle(ctx context.Context, id string) error {
	if m.DeleteKnowledgeArticleFunc != nil {
		return m.DeleteKnowledgeArticleFunc(ctx, id)
	}
	return nil
}

func (m *mockKnowledgeService) IncrementViewCount(ctx context.Context, id string) error {
	if m.IncrementViewCountFunc != nil {
		return m.IncrementViewCountFunc(ctx, id)
	}
	return nil
}

func (m *mockKnowledgeService) IncrementLikeCount(ctx context.Context, id string) error {
	if m.IncrementLikeCountFunc != nil {
		return m.IncrementLikeCountFunc(ctx, id)
	}
	return nil
}

func (m *mockKnowledgeService) SearchKnowledgeArticles(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
	if m.SearchKnowledgeArticlesFunc != nil {
		return m.SearchKnowledgeArticlesFunc(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockKnowledgeService) ConvertWorkOrderToArticle(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error) {
	if m.ConvertWorkOrderToArticleFunc != nil {
		return m.ConvertWorkOrderToArticleFunc(ctx, workOrderID, req, creatorID)
	}
	return &models.KnowledgeArticle{}, nil
}

// ==================== Category method overrides ====================

func (m *mockKnowledgeService) GetKnowledgeCategoryList(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
	if m.GetKnowledgeCategoryListFunc != nil {
		return m.GetKnowledgeCategoryListFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockKnowledgeService) GetKnowledgeCategory(ctx context.Context, id string) (*models.KnowledgeCategory, error) {
	if m.GetKnowledgeCategoryFunc != nil {
		return m.GetKnowledgeCategoryFunc(ctx, id)
	}
	return &models.KnowledgeCategory{}, nil
}

func (m *mockKnowledgeService) CreateKnowledgeCategory(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error) {
	if m.CreateKnowledgeCategoryFunc != nil {
		return m.CreateKnowledgeCategoryFunc(ctx, req, creatorID)
	}
	return &models.KnowledgeCategory{}, nil
}

func (m *mockKnowledgeService) UpdateKnowledgeCategory(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error {
	if m.UpdateKnowledgeCategoryFunc != nil {
		return m.UpdateKnowledgeCategoryFunc(ctx, id, req, operatorID)
	}
	return nil
}

func (m *mockKnowledgeService) DeleteKnowledgeCategory(ctx context.Context, id string) error {
	if m.DeleteKnowledgeCategoryFunc != nil {
		return m.DeleteKnowledgeCategoryFunc(ctx, id)
	}
	return nil
}

// ==================== Tag method overrides ====================

func (m *mockKnowledgeService) GetAllTags(ctx context.Context) ([]models.KnowledgeTag, error) {
	if m.GetAllTagsFunc != nil {
		return m.GetAllTagsFunc(ctx)
	}
	return nil, nil
}

func (m *mockKnowledgeService) GetTagByName(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	if m.GetTagByNameFunc != nil {
		return m.GetTagByNameFunc(ctx, name)
	}
	return &models.KnowledgeTag{}, nil
}

func (m *mockKnowledgeService) CreateTag(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	if m.CreateTagFunc != nil {
		return m.CreateTagFunc(ctx, name)
	}
	return &models.KnowledgeTag{}, nil
}

func (m *mockKnowledgeService) UpdateTag(ctx context.Context, id string, name string) error {
	if m.UpdateTagFunc != nil {
		return m.UpdateTagFunc(ctx, id, name)
	}
	return nil
}

func (m *mockKnowledgeService) DeleteTag(ctx context.Context, id string) error {
	if m.DeleteTagFunc != nil {
		return m.DeleteTagFunc(ctx, id)
	}
	return nil
}

// ==================== Cache invalidation overrides ====================

func (m *mockKnowledgeService) InvalidateCategoryCache(ctx context.Context) error {
	if m.InvalidateCategoryCacheFunc != nil {
		return m.InvalidateCategoryCacheFunc(ctx)
	}
	return nil
}

func (m *mockKnowledgeService) InvalidateTagCache(ctx context.Context) error {
	if m.InvalidateTagCacheFunc != nil {
		return m.InvalidateTagCacheFunc(ctx)
	}
	return nil
}

func (m *mockKnowledgeService) InvalidateArticleCache(ctx context.Context, articleID string) error {
	if m.InvalidateArticleCacheFunc != nil {
		return m.InvalidateArticleCacheFunc(ctx, articleID)
	}
	return nil
}

func (m *mockKnowledgeService) InvalidateAllArticleCache(ctx context.Context) error {
	if m.InvalidateAllArticleCacheFunc != nil {
		return m.InvalidateAllArticleCacheFunc(ctx)
	}
	return nil
}

// ==================== Test helpers ====================

func newTestCtxKnowledge(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func setupArticleHandler(mock *mockKnowledgeService) *ArticleHandler {
	return NewArticleHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

func setupCategoryHandler(mock *mockKnowledgeService) *CategoryHandler {
	return NewCategoryHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

func setupTagHandler(mock *mockKnowledgeService) *TagHandler {
	return NewTagHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestKnowledgeHandlers_CompileOnly(t *testing.T) {
	mock := &mockKnowledgeService{}
	a := setupArticleHandler(mock)
	c := setupCategoryHandler(mock)
	tg := setupTagHandler(mock)
	assert.NotNil(t, a)
	assert.NotNil(t, c)
	assert.NotNil(t, tg)
}

// ==================== ArticleHandler tests ====================

func TestArticleHandler_List_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeArticleListFunc: func(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
			return []models.KnowledgeArticle{{Title: "A1"}}, 1, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_List_WithFilters(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeArticleListFunc: func(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
			return []models.KnowledgeArticle{{Title: req.Title}}, 1, nil
		},
	}
	h := setupArticleHandler(mock)
	status := 1
	c, w := newTestCtxKnowledge("POST", "/list", map[string]interface{}{
		"current":       2.0,
		"pageSize":      25.0,
		"orderByColumn": "title",
		"isAsc":         true,
		"title":         "T",
		"categoryId":    uuid.NewString(),
		"tagId":         uuid.NewString(),
		"status":        float64(status),
		"createdBy":     "u1",
	})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_List_InvalidJSON(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeArticleListFunc: func(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
			return []models.KnowledgeArticle{}, 0, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", "not-valid-json")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_List_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeArticleListFunc: func(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
			return nil, 0, errors.New("list fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Statistics_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		GetArticleStatisticsFunc: func(ctx context.Context) (*services.KnowledgeArticleStatistics, error) {
			return &services.KnowledgeArticleStatistics{Total: 10, Draft: 3, Published: 7}, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/statistics", nil)
	h.Statistics(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Statistics_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetArticleStatisticsFunc: func(ctx context.Context) (*services.KnowledgeArticleStatistics, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/statistics", nil)
	h.Statistics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_GetByID_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_GetByID_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		IncrementViewCountFunc: func(ctx context.Context, id string) error { return nil },
		GetKnowledgeArticleFunc: func(ctx context.Context, id string) (*models.KnowledgeArticle, error) {
			return &models.KnowledgeArticle{BaseModel: models.BaseModel{ID: id}, Title: "Hello"}, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_GetByID_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeArticleFunc: func(ctx context.Context, id string) (*models.KnowledgeArticle, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Create_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{}) // missing required title/content/categoryId
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_Create_NoUserID(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeArticleCreateRequest{
		Title:      "T",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	h.Create(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestArticleHandler_Create_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateKnowledgeArticleFunc: func(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error) {
			return &models.KnowledgeArticle{BaseModel: models.BaseModel{ID: uuid.NewString()}, Title: req.Title}, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeArticleCreateRequest{
		Title:      "T",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Create_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateKnowledgeArticleFunc: func(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeArticleCreateRequest{
		Title:      "T",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Update_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_Update_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", "not-json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_Update_NoUserID(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeArticleUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestArticleHandler_Update_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateKnowledgeArticleFunc: func(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error {
			return nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeArticleUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Update_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateKnowledgeArticleFunc: func(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error {
			return errors.New("update fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeArticleUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Delete_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_Delete_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteKnowledgeArticleFunc: func(ctx context.Context, id string) error { return nil },
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteKnowledgeArticleFunc: func(ctx context.Context, id string) error {
			return errors.New("del fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_ConvertFromWorkOrder_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.ConvertFromWorkOrder(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_ConvertFromWorkOrder_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", "not-json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.ConvertFromWorkOrder(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_ConvertFromWorkOrder_NoUserID(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.ConvertWorkOrderToArticleRequest{
		Title:      "T",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.ConvertFromWorkOrder(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestArticleHandler_ConvertFromWorkOrder_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		ConvertWorkOrderToArticleFunc: func(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error) {
			return &models.KnowledgeArticle{BaseModel: models.BaseModel{ID: uuid.NewString()}, Title: req.Title}, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.ConvertWorkOrderToArticleRequest{
		Title:      "FromWO",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.ConvertFromWorkOrder(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_ConvertFromWorkOrder_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		ConvertWorkOrderToArticleFunc: func(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error) {
			return nil, errors.New("convert fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.ConvertWorkOrderToArticleRequest{
		Title:      "FromWO",
		Content:    "C",
		CategoryID: uuid.NewString(),
	})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.ConvertFromWorkOrder(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Search_EmptyBody(t *testing.T) {
	mock := &mockKnowledgeService{
		SearchKnowledgeArticlesFunc: func(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
			return []models.KnowledgeArticle{}, 0, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/search", nil)
	h.Search(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Search_InvalidJSON(t *testing.T) {
	mock := &mockKnowledgeService{
		SearchKnowledgeArticlesFunc: func(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
			// After invalid JSON, default PageSize=100 and PageNum=0 should apply.
			assert.Equal(t, 100, req.PageSize)
			assert.Equal(t, 0, req.PageNum)
			return []models.KnowledgeArticle{}, 0, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/search", "not-json")
	h.Search(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Search_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		SearchKnowledgeArticlesFunc: func(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
			return []models.KnowledgeArticle{{Title: "Match"}}, 1, nil
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/search", services.SearchKnowledgeRequest{Keyword: "test", PageSize: 10, PageNum: 0})
	h.Search(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Search_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		SearchKnowledgeArticlesFunc: func(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
			return nil, 0, errors.New("search fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/search", services.SearchKnowledgeRequest{})
	h.Search(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Like_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/like", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Like(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticleHandler_Like_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		IncrementLikeCountFunc: func(ctx context.Context, id string) error { return nil },
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/like", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Like(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticleHandler_Like_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		IncrementLikeCountFunc: func(ctx context.Context, id string) error {
			return errors.New("like fail")
		},
	}
	h := setupArticleHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/like", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Like(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== CategoryHandler tests ====================

func TestCategoryHandler_List_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeCategoryListFunc: func(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
			return []models.KnowledgeCategory{{CategoryName: "C1"}}, nil
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_List_InvalidJSON(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeCategoryListFunc: func(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
			return []models.KnowledgeCategory{}, nil
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_List_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeCategoryListFunc: func(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
			return nil, errors.New("cat list fail")
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_GetByID_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_GetByID_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeCategoryFunc: func(ctx context.Context, id string) (*models.KnowledgeCategory, error) {
			return &models.KnowledgeCategory{BaseModel: models.BaseModel{ID: id}, CategoryName: "C1"}, nil
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_GetByID_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetKnowledgeCategoryFunc: func(ctx context.Context, id string) (*models.KnowledgeCategory, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Create_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{}) // missing required categoryName
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Create_NoUserID(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeCategoryCreateRequest{CategoryName: "C1"})
	h.Create(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCategoryHandler_Create_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateKnowledgeCategoryFunc: func(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error) {
			return &models.KnowledgeCategory{BaseModel: models.BaseModel{ID: uuid.NewString()}, CategoryName: req.CategoryName}, nil
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeCategoryCreateRequest{CategoryName: "C1"})
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Create_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateKnowledgeCategoryFunc: func(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error) {
			return nil, errors.New("cat create fail")
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", services.KnowledgeCategoryCreateRequest{CategoryName: "C1"})
	c.Set("user_id", "user-1")
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Update_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Update_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", "not-json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Update_NoUserID(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeCategoryUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCategoryHandler_Update_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateKnowledgeCategoryFunc: func(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error {
			return nil
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeCategoryUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Update_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateKnowledgeCategoryFunc: func(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error {
			return errors.New("cat update fail")
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", services.KnowledgeCategoryUpdateRequest{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Delete_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryHandler_Delete_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteKnowledgeCategoryFunc: func(ctx context.Context, id string) error { return nil },
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteKnowledgeCategoryFunc: func(ctx context.Context, id string) error {
			return errors.New("cat del fail")
		},
	}
	h := setupCategoryHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== TagHandler tests ====================

func TestTagHandler_GetAll_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		GetAllTagsFunc: func(ctx context.Context) ([]models.KnowledgeTag, error) {
			return []models.KnowledgeTag{{TagName: "T1"}}, nil
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/all", nil)
	h.GetAll(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_GetAll_Error(t *testing.T) {
	mock := &mockKnowledgeService{
		GetAllTagsFunc: func(ctx context.Context) ([]models.KnowledgeTag, error) {
			return nil, errors.New("tags fail")
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/all", nil)
	h.GetAll(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTagHandler_Create_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{}) // missing required tagName
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTagHandler_Create_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateTagFunc: func(ctx context.Context, name string) (*models.KnowledgeTag, error) {
			return &models.KnowledgeTag{ID: uuid.NewString(), TagName: name}, nil
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{"tagName": "Important"})
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_Create_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		CreateTagFunc: func(ctx context.Context, name string) (*models.KnowledgeTag, error) {
			return nil, errors.New("tag create fail")
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/", map[string]interface{}{"tagName": "Important"})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTagHandler_Update_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTagHandler_Update_BindError(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", "not-json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTagHandler_Update_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateTagFunc: func(ctx context.Context, id string, name string) error { return nil },
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", map[string]interface{}{"tagName": "Renamed"})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_Update_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		UpdateTagFunc: func(ctx context.Context, id string, name string) error {
			return errors.New("tag update fail")
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/update", map[string]interface{}{"tagName": "Renamed"})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTagHandler_Delete_Empty(t *testing.T) {
	mock := &mockKnowledgeService{}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTagHandler_Delete_Success(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteTagFunc: func(ctx context.Context, id string) error { return nil },
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTagHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockKnowledgeService{
		DeleteTagFunc: func(ctx context.Context, id string) error {
			return errors.New("tag del fail")
		},
	}
	h := setupTagHandler(mock)
	c, w := newTestCtxKnowledge("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== WithCore nil-safety ====================

func TestArticleHandler_WithCore_NilSafety(t *testing.T) {
	h := &ArticleHandler{}
	result := h.WithCore(nil)
	assert.Same(t, h, result)

	realCore := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	}
	result2 := h.WithCore(realCore)
	assert.Same(t, h, result2)
	assert.Equal(t, realCore, h.core)
}

func TestCategoryHandler_WithCore_NilSafety(t *testing.T) {
	h := &CategoryHandler{}
	result := h.WithCore(nil)
	assertSame(t, h, result)
}

func TestTagHandler_WithCore_NilSafety(t *testing.T) {
	h := &TagHandler{}
	result := h.WithCore(nil)
	assertSame(t, h, result)
}

// assertSame is a tiny alias so the package only depends on assert/require's
// machinery through the helper above.
func assertSame(t *testing.T, expected, actual interface{}) {
	t.Helper()
	assert.Same(t, expected, actual)
}