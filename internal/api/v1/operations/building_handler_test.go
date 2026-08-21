package operations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// uuidNew returns a simple test ID (numeric, valid in URL paths).
func uuidNew() string {
	const hex = "0123456789abcdef"
	b := make([]byte, 36)
	b[8], b[13], b[18], b[23] = '-', '-', '-', '-'
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '-':
			continue
		default:
			b[i] = hex[(i*7+3)%16]
		}
	}
	return string(b)
}

// mockBuildingService implements opsServices.BuildingService for handler tests (D-08 mock pattern).
type mockBuildingService struct {
	CreateFunc                func(ctx context.Context, building *opsModels.OpsBuilding) error
	UpdateFunc                func(ctx context.Context, building *opsModels.OpsBuilding) error
	DeleteFunc                func(ctx context.Context, id string) error
	GetByIDFunc               func(ctx context.Context, id string) (*opsModels.OpsBuilding, error)
	ListFunc                  func(ctx context.Context, params map[string]interface{}) (*opsServices.PageResult, error)
	BatchDeleteFunc           func(ctx context.Context, ids []string) error
	StatisticsFunc            func(ctx context.Context, params map[string]interface{}) (*opsServices.BuildingStatisticsResult, error)
	SearchBuildingOptionsFunc func(ctx context.Context, params map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockBuildingService) Create(ctx context.Context, b *opsModels.OpsBuilding) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, b)
	}
	return errors.New("Create not implemented")
}

func (m *mockBuildingService) Update(ctx context.Context, b *opsModels.OpsBuilding) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, b)
	}
	return errors.New("Update not implemented")
}

func (m *mockBuildingService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errors.New("Delete not implemented")
}

func (m *mockBuildingService) GetByID(ctx context.Context, id string) (*opsModels.OpsBuilding, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("GetByID not implemented")
}

func (m *mockBuildingService) List(ctx context.Context, params map[string]interface{}) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, errors.New("List not implemented")
}

func (m *mockBuildingService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errors.New("BatchDelete not implemented")
}

func (m *mockBuildingService) Statistics(ctx context.Context, params map[string]interface{}) (*opsServices.BuildingStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx, params)
	}
	return nil, errors.New("Statistics not implemented")
}

func (m *mockBuildingService) SearchBuildingOptions(ctx context.Context, params map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchBuildingOptionsFunc != nil {
		return m.SearchBuildingOptionsFunc(ctx, params)
	}
	return nil, errors.New("SearchBuildingOptions not implemented")
}

// Note: GeocodingService is a concrete struct (not an interface), so we test only
// the Geocode handler paths that don't require a live Baidu API:
//   - bind error
//   - empty address
//   - nil geocodingService (NewGeocodingService("") will succeed but geocode will fail; handler errors)
//   - happy-path is covered by service-level tests.

// stubRecorder and newBuildingHandlerCore moved to handlers_test_helpers_test.go.

func newBuildingHandlerCore(t *testing.T) *core.Core { return newTestCore(t) }

func newTestBuildingHandler(svc opsServices.BuildingService, geo *opsServices.GeocodingService, c *core.Core) *BuildingHandler {
	h := NewBuildingHandler(svc, geo)
	if c != nil {
		h.WithCore(c)
	}
	return h
}

func doBuildingRequest(h *BuildingHandler, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	switch {
	case strings.HasSuffix(path, "/list"):
		r.POST(path, h.List)
	case strings.HasSuffix(path, "/batch"):
		r.POST(path, h.BatchOperation)
	case strings.HasSuffix(path, "/update"):
		// Convert literal id to :id placeholder for c.Param("id") to work.
		r.POST(strings.Replace(path, extractID(path), ":id", 1), h.Update)
	case strings.HasSuffix(path, "/delete"):
		r.POST(strings.Replace(path, extractID(path), ":id", 1), h.Delete)
	case strings.Contains(path, "/geocode"):
		r.POST(path, h.Geocode)
	case strings.HasSuffix(path, "/statistics"):
		r.POST(path, h.Statistics)
	case strings.HasSuffix(path, "/search-options"):
		r.POST(path, h.SearchBuildingOptions)
	default:
		// GetByID route: path has 3 segments (e.g. /buildings/<uuid>).
		// strings.Count returns the number of "/" runes; "/buildings/<uuid>" has 2.
		// Anything >= 2 here is treated as GetByID.
		if strings.Count(path, "/") >= 2 {
			r.POST(strings.Replace(path, extractID(path), ":id", 1), h.GetByID)
		} else {
			r.POST(path, h.Create)
		}
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// extractID pulls the UUID-style segment from a path like "/buildings/<id>[/...]".
// For both "/buildings/<id>" (len==3) and "/buildings/<id>/delete" (len==4),
// the id is at index 2.
func extractID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// TestBuildingHandler_Create_Success
func TestBuildingHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	coreInst := newBuildingHandlerCore(t)
	captured := &opsModels.OpsBuilding{}
	svc := &mockBuildingService{
		CreateFunc: func(_ context.Context, b *opsModels.OpsBuilding) error {
			*captured = *b
			b.ID = "b-1"
			return nil
		},
	}
	h := newTestBuildingHandler(svc, nil, coreInst)

	body := `{"name":"Main","orgId":"` + uuidNew() + `"}`
	w := doBuildingRequest(h, http.MethodPost, "/buildings", body)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":0`)
	assert.Equal(t, "Main", captured.Name)
}

// TestBuildingHandler_Create_BindError covers JSON parse failure path.
func TestBuildingHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings", `{invalid}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_Create_ServiceError
func TestBuildingHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsBuilding) error {
			return errors.New("db error")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	body := `{"name":"X"}`
	w := doBuildingRequest(h, http.MethodPost, "/buildings", body)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_List_Success
func TestBuildingHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{
				List:     []opsModels.OpsBuilding{{Name: "A"}, {Name: "B"}},
				Total:    2,
				Current:  1,
				PageSize: 10,
			}, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/list", `{}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":2`)
}

// TestBuildingHandler_List_BindError — building List uses handleJSONBinding (returns 400
// on invalid JSON). Floor/workstation List handlers use fallback params instead — this
// divergence is preserved per D-12.
func TestBuildingHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{}, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/list", `{not-json`)
	// handleJSONBinding returns 400 for invalid JSON.
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_List_ServiceError
func TestBuildingHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return nil, errors.New("query fail")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/list", `{}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_GetByID_Success
func TestBuildingHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := "11111111-2222-3333-4444-555555555555"
	svc := &mockBuildingService{
		GetByIDFunc: func(_ context.Context, gotID string) (*opsModels.OpsBuilding, error) {
			assert.Equal(t, id, gotID)
			return &opsModels.OpsBuilding{BaseModel: models.BaseModel{ID: gotID}, Name: "B"}, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/"+id, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"B"`)
}

// TestBuildingHandler_GetByID_NotFound — apperrors.BuildingNotFound (code 3010) maps
// to HTTP 400 (per apperrors.DefaultHTTPStatus code range 2000-8999 → BadRequest).
// This is a quirk in apperrors package — not fixed per D-12.
func TestBuildingHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsBuilding, error) {
			return nil, errors.New("not found")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/missing", "")
	// Quirky: apperrors.BuildingNotFound → 400 not 404.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "楼宇")
}

// TestBuildingHandler_Update_Success
func TestBuildingHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuidNew()
	updated := &opsModels.OpsBuilding{}
	svc := &mockBuildingService{
		UpdateFunc: func(_ context.Context, b *opsModels.OpsBuilding) error {
			assert.Equal(t, id, b.ID)
			*updated = *b
			return nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	body := `{"name":"Updated","orgId":"` + uuidNew() + `"}`
	w := doBuildingRequest(h, http.MethodPost, "/buildings/"+id+"/update", body)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, id, updated.ID)
}

// TestBuildingHandler_Update_BindError
func TestBuildingHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/x/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_Update_ServiceError
func TestBuildingHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsBuilding) error {
			return errors.New("update fail")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/x/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_Delete_Success
func TestBuildingHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuidNew()
	called := false
	svc := &mockBuildingService{
		DeleteFunc: func(_ context.Context, gotID string) error {
			assert.Equal(t, id, gotID)
			called = true
			return nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/"+id+"/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestBuildingHandler_Delete_Error
func TestBuildingHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("del fail") },
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/x/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_BatchOperation_Delete
func TestBuildingHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			assert.Len(t, ids, 2)
			called = true
			return nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	body := `{"ids":["a","b"],"action":"delete"}`
	w := doBuildingRequest(h, http.MethodPost, "/buildings/batch", body)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestBuildingHandler_BatchOperation_UnsupportedAction
func TestBuildingHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	body := `{"ids":["a"],"action":"freezelol"}`
	w := doBuildingRequest(h, http.MethodPost, "/buildings/batch", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_BatchOperation_BindError
func TestBuildingHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_BatchOperation_ServiceError
func TestBuildingHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd fail") },
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_Statistics_Success
func TestBuildingHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		StatisticsFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.BuildingStatisticsResult, error) {
			return &opsServices.BuildingStatisticsResult{Total: 10, Active: 7, Inactive: 3}, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/statistics", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":10`)
}

// TestBuildingHandler_Statistics_Error documents a quirk in Statistics handler:
// it calls response.Error(c, http.StatusInternalServerError, err.Error()) with an
// int first arg; response.toAppError's `case int` branch hard-codes HTTPStatus to
// 400, so service errors surface as 400 instead of 500. Per D-12 we do NOT fix
// this — only document.
func TestBuildingHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		StatisticsFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.BuildingStatisticsResult, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/statistics", `{}`)
	// Quirky: int first arg in response.Error → 400 (per toAppError case int).
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_SearchBuildingOptions_Success
func TestBuildingHandler_SearchBuildingOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		SearchBuildingOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "Main"}}, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/search-options", `{"name":"Main"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"value":"1"`)
}

// TestBuildingHandler_SearchBuildingOptions_InvalidJSON
func TestBuildingHandler_SearchBuildingOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingService{
		SearchBuildingOptionsFunc: func(_ context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			assert.NotNil(t, p)
			return nil, nil
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestBuildingHandler_SearchBuildingOptions_Error documents the same int-as-first-arg
// quirk as Statistics — service errors surface as 400 (see toAppError case int).
// Per D-12: not fixed.
func TestBuildingHandler_SearchBuildingOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		SearchBuildingOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("opt fail")
		},
	}
	h := newTestBuildingHandler(svc, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_Geocode_BindError
func TestBuildingHandler_Geocode_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/geocode", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_Geocode_EmptyAddress — documents a dead-code branch:
// GeocodeRequest.Address has binding:"required", so any JSON reaching the empty-
// address branch would fail binding first. The "if req.Address == """ block is
// unreachable in production; we assert it is unreachable in tests too.
func TestBuildingHandler_Geocode_EmptyAddressBranchUnreachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	// Body with address="" still fails binding:"required".
	w := doBuildingRequest(h, http.MethodPost, "/buildings/geocode", `{"address":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Message is from binding validator, not from "地址" missing branch.
	assert.Contains(t, w.Body.String(), "参数错误")
}

// TestBuildingHandler_Geocode_BindErrorBypassesBinding documents that when JSON
// is invalid, binding fails first (400 + "参数错误").
func TestBuildingHandler_Geocode_BindErrorBypassesBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestBuildingHandler(&mockBuildingService{}, nil, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/geocode", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_Geocode_ServiceError — GeocodingService with empty API key
// will fail network call; handler returns 500.
func TestBuildingHandler_Geocode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	geo := opsServices.NewGeocodingService("")
	h := newTestBuildingHandler(&mockBuildingService{}, geo, newBuildingHandlerCore(t))

	w := doBuildingRequest(h, http.MethodPost, "/buildings/geocode", `{"address":"北京市朝阳区"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestBuildingHandler_Geocode_NilService — nil GeocodingService causes nil deref panic? No: handler
// always builds via NewBuildingHandler; this test ensures constructor tolerates nil.
func TestBuildingHandler_Geocode_NilServiceConstructor(t *testing.T) {
	h := NewBuildingHandler(&mockBuildingService{}, nil)
	assert.NotNil(t, h)
	// geocodingService may be nil; binding will reach handler, but address is required, so empty body → 400
	w := doBuildingRequest(h, http.MethodPost, "/buildings/geocode", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildingHandler_WithCore_NilSafe verifies nil-safe injection.
func TestBuildingHandler_WithCore_NilSafe(t *testing.T) {
	var h *BuildingHandler
	out := h.WithCore(newBuildingHandlerCore(t))
	assert.Nil(t, out)
}

// TestBuildingHandler_NewBuildingHandlerWithCore ensures the legacy ctor builds OK
// (requires non-nil core.Config — we skip if Config is nil to avoid touching config).
func TestBuildingHandler_NewBuildingHandlerWithCore(t *testing.T) {
	c := newBuildingHandlerCore(t)
	if c.Config == nil {
		t.Skip("skipping NewBuildingHandlerWithCore: requires non-nil core.Config")
	}
	svc := &mockBuildingService{}
	h := NewBuildingHandlerWithCore(svc, c)
	assert.NotNil(t, h)
}
