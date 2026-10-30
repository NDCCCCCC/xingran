package system

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// =====================================================================
// Phase 74-04: dashboard_handler tests
//
// Scope (dashboard_handler.go — 27 funcs):
//   - NewDashboardHandler / WithCore
//   - getUserContext / requireUserContext / handleServiceError / requireID
//   - GetDefault, List, GetByID, Create, Update, Delete, Duplicate, SetDefault
//   - GetTemplates, CreateFromTemplate
//   - GetVersions, CreateVersion, RestoreVersion
//   - Export, Import
//   - GetWidgetData, GetBatchWidgetData
//   - GetAvailableEndpoints, ValidateEndpoint, GetUserEndpointsWithFilter, InvalidateEndpointCache
//
// Mock pattern: per-method *Func fields on mockDashboardService satisfying
// the full system.DashboardService interface.
// =====================================================================

// mockDashboardService satisfies system.DashboardService via per-method *Func fields.
type mockDashboardService struct {
	GetDashboardsFunc                 func(ctx context.Context, params system.DashboardListParams) (*system.DashboardListResponse, error)
	GetAccessibleDashboardsFunc       func(ctx context.Context, params system.DashboardListParams, userID, userDeptID, dataScope string) (*system.DashboardListResponse, error)
	GetDashboardFunc                  func(ctx context.Context, id string) (*models.Dashboard, error)
	GetAccessibleDefaultDashboardFunc func(ctx context.Context, userID, userDeptID, dataScope string) (*models.Dashboard, error)
	CreateDashboardFunc               func(ctx context.Context, userID string, req system.CreateDashboardRequest) (*models.Dashboard, error)
	CreateDashboardWithPermissionsFunc func(ctx context.Context, userID, userDeptID, dataScope string, isAdmin bool, req system.CreateDashboardRequest) (*models.Dashboard, error)
	UpdateDashboardFunc               func(ctx context.Context, userID, id string, req system.UpdateDashboardRequest) error
	DeleteDashboardFunc               func(ctx context.Context, userID, id string) error
	DuplicateDashboardFunc            func(ctx context.Context, userID, id string) (*models.Dashboard, error)
	SetDefaultDashboardFunc           func(ctx context.Context, userID, id string) error
	GetTemplatesFunc                  func(ctx context.Context, scope *string) ([]models.Dashboard, error)
	CreateFromTemplateFunc            func(ctx context.Context, userID, templateID, name string) (*models.Dashboard, error)
	GetVersionsFunc                   func(ctx context.Context, dashboardID string) ([]models.DashboardVersion, error)
	CreateVersionFunc                 func(ctx context.Context, userID, dashboardID, comment string) (*models.DashboardVersion, error)
	RestoreVersionFunc                func(ctx context.Context, userID, dashboardID, versionID string) error
	ExportDashboardFunc               func(ctx context.Context, id string) (string, error)
	ImportDashboardFunc               func(ctx context.Context, userID, config string) (*models.Dashboard, error)
	GetWidgetDataFunc                 func(ctx context.Context, widgetID, apiEndpoint string, params map[string]interface{}) (interface{}, error)
	GetBatchWidgetDataFunc            func(ctx context.Context, widgetIDs []string, bypassCache bool) (map[string]system.WidgetDataResult, error)
	GetUserAccessibleEndpointsFunc    func(ctx context.Context, userID string) ([]systemServices.CategoryEndpoints, error)
	ValidateEndpointFunc              func(route, method string) (*systemServices.EndpointDetail, error)
	InvalidateUserCacheFunc           func(ctx context.Context, userID string)
	FilterEndpointsByWidgetTypeFunc   func(categories []systemServices.CategoryEndpoints, widgetType string) []systemServices.CategoryEndpoints
}

func (m *mockDashboardService) GetDashboards(ctx context.Context, params system.DashboardListParams) (*system.DashboardListResponse, error) {
	if m.GetDashboardsFunc != nil {
		return m.GetDashboardsFunc(ctx, params)
	}
	return &system.DashboardListResponse{}, nil
}
func (m *mockDashboardService) GetAccessibleDashboards(ctx context.Context, params system.DashboardListParams, userID, userDeptID, dataScope string) (*system.DashboardListResponse, error) {
	if m.GetAccessibleDashboardsFunc != nil {
		return m.GetAccessibleDashboardsFunc(ctx, params, userID, userDeptID, dataScope)
	}
	return &system.DashboardListResponse{}, nil
}
func (m *mockDashboardService) GetDashboard(ctx context.Context, id string) (*models.Dashboard, error) {
	if m.GetDashboardFunc != nil {
		return m.GetDashboardFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockDashboardService) GetAccessibleDefaultDashboard(ctx context.Context, userID, userDeptID, dataScope string) (*models.Dashboard, error) {
	if m.GetAccessibleDefaultDashboardFunc != nil {
		return m.GetAccessibleDefaultDashboardFunc(ctx, userID, userDeptID, dataScope)
	}
	return nil, nil
}
func (m *mockDashboardService) CreateDashboard(ctx context.Context, userID string, req system.CreateDashboardRequest) (*models.Dashboard, error) {
	if m.CreateDashboardFunc != nil {
		return m.CreateDashboardFunc(ctx, userID, req)
	}
	return nil, nil
}
func (m *mockDashboardService) CreateDashboardWithPermissions(ctx context.Context, userID, userDeptID, dataScope string, isAdmin bool, req system.CreateDashboardRequest) (*models.Dashboard, error) {
	if m.CreateDashboardWithPermissionsFunc != nil {
		return m.CreateDashboardWithPermissionsFunc(ctx, userID, userDeptID, dataScope, isAdmin, req)
	}
	return nil, nil
}
func (m *mockDashboardService) UpdateDashboard(ctx context.Context, userID, id string, req system.UpdateDashboardRequest) error {
	if m.UpdateDashboardFunc != nil {
		return m.UpdateDashboardFunc(ctx, userID, id, req)
	}
	return nil
}
func (m *mockDashboardService) DeleteDashboard(ctx context.Context, userID, id string) error {
	if m.DeleteDashboardFunc != nil {
		return m.DeleteDashboardFunc(ctx, userID, id)
	}
	return nil
}
func (m *mockDashboardService) DuplicateDashboard(ctx context.Context, userID, id string) (*models.Dashboard, error) {
	if m.DuplicateDashboardFunc != nil {
		return m.DuplicateDashboardFunc(ctx, userID, id)
	}
	return nil, nil
}
func (m *mockDashboardService) SetDefaultDashboard(ctx context.Context, userID, id string) error {
	if m.SetDefaultDashboardFunc != nil {
		return m.SetDefaultDashboardFunc(ctx, userID, id)
	}
	return nil
}
func (m *mockDashboardService) GetTemplates(ctx context.Context, scope *string) ([]models.Dashboard, error) {
	if m.GetTemplatesFunc != nil {
		return m.GetTemplatesFunc(ctx, scope)
	}
	return nil, nil
}
func (m *mockDashboardService) CreateFromTemplate(ctx context.Context, userID, templateID, name string) (*models.Dashboard, error) {
	if m.CreateFromTemplateFunc != nil {
		return m.CreateFromTemplateFunc(ctx, userID, templateID, name)
	}
	return nil, nil
}
func (m *mockDashboardService) GetVersions(ctx context.Context, dashboardID string) ([]models.DashboardVersion, error) {
	if m.GetVersionsFunc != nil {
		return m.GetVersionsFunc(ctx, dashboardID)
	}
	return nil, nil
}
func (m *mockDashboardService) CreateVersion(ctx context.Context, userID, dashboardID, comment string) (*models.DashboardVersion, error) {
	if m.CreateVersionFunc != nil {
		return m.CreateVersionFunc(ctx, userID, dashboardID, comment)
	}
	return nil, nil
}
func (m *mockDashboardService) RestoreVersion(ctx context.Context, userID, dashboardID, versionID string) error {
	if m.RestoreVersionFunc != nil {
		return m.RestoreVersionFunc(ctx, userID, dashboardID, versionID)
	}
	return nil
}
func (m *mockDashboardService) ExportDashboard(ctx context.Context, id string) (string, error) {
	if m.ExportDashboardFunc != nil {
		return m.ExportDashboardFunc(ctx, id)
	}
	return "", nil
}
func (m *mockDashboardService) ImportDashboard(ctx context.Context, userID, config string) (*models.Dashboard, error) {
	if m.ImportDashboardFunc != nil {
		return m.ImportDashboardFunc(ctx, userID, config)
	}
	return nil, nil
}
func (m *mockDashboardService) GetWidgetData(ctx context.Context, widgetID, apiEndpoint string, params map[string]interface{}) (interface{}, error) {
	if m.GetWidgetDataFunc != nil {
		return m.GetWidgetDataFunc(ctx, widgetID, apiEndpoint, params)
	}
	return nil, nil
}
func (m *mockDashboardService) GetBatchWidgetData(ctx context.Context, widgetIDs []string, bypassCache bool) (map[string]system.WidgetDataResult, error) {
	if m.GetBatchWidgetDataFunc != nil {
		return m.GetBatchWidgetDataFunc(ctx, widgetIDs, bypassCache)
	}
	return nil, nil
}
func (m *mockDashboardService) GetUserAccessibleEndpoints(ctx context.Context, userID string) ([]systemServices.CategoryEndpoints, error) {
	if m.GetUserAccessibleEndpointsFunc != nil {
		return m.GetUserAccessibleEndpointsFunc(ctx, userID)
	}
	return nil, nil
}
func (m *mockDashboardService) ValidateEndpoint(route, method string) (*systemServices.EndpointDetail, error) {
	if m.ValidateEndpointFunc != nil {
		return m.ValidateEndpointFunc(route, method)
	}
	return nil, nil
}
func (m *mockDashboardService) InvalidateUserCache(ctx context.Context, userID string) {
	if m.InvalidateUserCacheFunc != nil {
		m.InvalidateUserCacheFunc(ctx, userID)
	}
}
func (m *mockDashboardService) FilterEndpointsByWidgetType(categories []systemServices.CategoryEndpoints, widgetType string) []systemServices.CategoryEndpoints {
	if m.FilterEndpointsByWidgetTypeFunc != nil {
		return m.FilterEndpointsByWidgetTypeFunc(categories, widgetType)
	}
	return categories
}

// dashboardTestUserID is the canonical test user for all dashboard handler tests.
const dashboardTestUserID = "user-dash-1"

// newDashboardHandler builds a DashboardHandler with mock service and empty core.
func newDashboardHandler(t *testing.T, svc system.DashboardService) *DashboardHandler {
	t.Helper()
	h := NewDashboardHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: nil},
	}
	return h
}

// invokeDashboardHandler builds a gin context with auth claims and invokes the handler.
func invokeDashboardHandler(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", dashboardTestUserID)
	c.Set("dept_id", "dept-1")
	c.Set("data_scope", "all")
	c.Set("is_admin", false)
	var buf *bytes.Buffer
	if body != nil {
		switch v := body.(type) {
		case string:
			buf = bytes.NewBufferString(v)
		default:
			b, _ := json.Marshal(body)
			buf = bytes.NewBuffer(b)
		}
	} else {
		buf = bytes.NewBuffer(nil)
	}
	c.Request = httptest.NewRequest(method, path, buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if params != nil {
		c.Params = params
	}
	if handler != nil {
		handler(c)
	}
	return w
}

// ----------------------------------------------------------------------------
// Constructor + DI + helpers
// ----------------------------------------------------------------------------

func TestDashboardHandler_New_WithCore(t *testing.T) {
	svc := &mockDashboardService{}
	h := NewDashboardHandler(svc)
	require.NotNil(t, h)
	out := h.WithCore(&core.Core{})
	assert.Same(t, h, out)
	assert.NotNil(t, h.core)
}

func TestDashboardHandler_getUserContext_Exists(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "u1")
	c.Set("dept_id", "d1")
	c.Set("data_scope", "all")
	c.Set("is_admin", true)

	ctx, err := h.getUserContext(c)
	require.NoError(t, err)
	assert.Equal(t, "u1", ctx.UserID)
	assert.Equal(t, "d1", ctx.UserDeptID)
	assert.Equal(t, "all", ctx.DataScope)
	assert.True(t, ctx.IsAdmin)
}

func TestDashboardHandler_getUserContext_Missing(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	ctx, err := h.getUserContext(c)
	assert.Error(t, err)
	assert.Nil(t, ctx)
}

func TestDashboardHandler_requireUserContext_Missing(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/x", nil, nil, nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)

	ctx, ok := h.requireUserContext(c)
	assert.False(t, ok)
	assert.Nil(t, ctx)
	assert.NotEqual(t, 0, w.Code)
}

func TestDashboardHandler_handleServiceError(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)

	assert.False(t, h.handleServiceError(c, nil))
	assert.True(t, h.handleServiceError(c, errors.New("boom")))
}

func TestDashboardHandler_requireID_Empty(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	id, ok := h.requireID(c)
	assert.False(t, ok)
	assert.Empty(t, id)
}

// ----------------------------------------------------------------------------
// GetDefault
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetDefault_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetAccessibleDefaultDashboardFunc: func(_ context.Context, _, _, _ string) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: "dash-1"}, Name: "Default"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/default", nil, nil, h.GetDefault)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetDefault_Nil(t *testing.T) {
	svc := &mockDashboardService{
		GetAccessibleDefaultDashboardFunc: func(_ context.Context, _, _, _ string) (*models.Dashboard, error) {
			return nil, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/default", nil, nil, h.GetDefault)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetDefault_ServiceError(t *testing.T) {
	svc := &mockDashboardService{
		GetAccessibleDefaultDashboardFunc: func(_ context.Context, _, _, _ string) (*models.Dashboard, error) {
			return nil, errors.New("boom")
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/default", nil, nil, h.GetDefault)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

func TestDashboardHandler_List_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetAccessibleDashboardsFunc: func(_ context.Context, _ system.DashboardListParams, _, _, _ string) (*system.DashboardListResponse, error) {
			return &system.DashboardListResponse{List: []models.Dashboard{{Name: "D1"}}, Total: 1, Current: 1, PageSize: 10}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/list", map[string]interface{}{"current": 1, "pageSize": 10}, nil, h.List)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_List_BindError(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/list", "not-json", nil, h.List)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_List_ServiceError(t *testing.T) {
	svc := &mockDashboardService{
		GetAccessibleDashboardsFunc: func(_ context.Context, _ system.DashboardListParams, _, _, _ string) (*system.DashboardListResponse, error) {
			return nil, errors.New("boom")
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/list", map[string]interface{}{"current": 1, "pageSize": 10}, nil, h.List)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetByID
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetByID_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetDashboardFunc: func(_ context.Context, id string) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: id}, Name: "Dash"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/d1", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.GetByID)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetByID_EmptyID(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/", nil,
		gin.Params{{Key: "id", Value: ""}}, h.GetByID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockDashboardService{
		GetDashboardFunc: func(_ context.Context, _ string) (*models.Dashboard, error) {
			return nil, errors.New("not found")
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/d1", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.GetByID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

func TestDashboardHandler_Create_Success(t *testing.T) {
	svc := &mockDashboardService{
		CreateDashboardWithPermissionsFunc: func(_ context.Context, _, _, _ string, _ bool, _ system.CreateDashboardRequest) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: "new"}, Name: "New"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{
		"name":            "New",
		"layout":          map[string]interface{}{},
		"refreshInterval": 60,
	}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards", body, nil, h.Create)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_Create_BindError(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards", "not-json", nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_Create_ServiceError(t *testing.T) {
	svc := &mockDashboardService{
		CreateDashboardWithPermissionsFunc: func(_ context.Context, _, _, _ string, _ bool, _ system.CreateDashboardRequest) (*models.Dashboard, error) {
			return nil, errors.New("boom")
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{
		"name":   "New",
		"layout": map[string]interface{}{},
	}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Update
// ----------------------------------------------------------------------------

func TestDashboardHandler_Update_Success(t *testing.T) {
	svc := &mockDashboardService{
		UpdateDashboardFunc: func(_ context.Context, _, _ string, _ system.UpdateDashboardRequest) error { return nil },
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"name": "Updated"}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/update", body,
		gin.Params{{Key: "id", Value: "d1"}}, h.Update)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_Update_ServiceError(t *testing.T) {
	svc := &mockDashboardService{
		UpdateDashboardFunc: func(_ context.Context, _, _ string, _ system.UpdateDashboardRequest) error { return errors.New("boom") },
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"name": "Updated"}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/update", body,
		gin.Params{{Key: "id", Value: "d1"}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Delete
// ----------------------------------------------------------------------------

func TestDashboardHandler_Delete_Success(t *testing.T) {
	svc := &mockDashboardService{
		DeleteDashboardFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "DELETE", "/system/dashboards/d1", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.Delete)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockDashboardService{
		DeleteDashboardFunc: func(_ context.Context, _, _ string) error { return errors.New("boom") },
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "DELETE", "/system/dashboards/d1", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Duplicate / SetDefault
// ----------------------------------------------------------------------------

func TestDashboardHandler_Duplicate_Success(t *testing.T) {
	svc := &mockDashboardService{
		DuplicateDashboardFunc: func(_ context.Context, _, _ string) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: "copy"}, Name: "Copy"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/duplicate", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.Duplicate)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_SetDefault_Success(t *testing.T) {
	svc := &mockDashboardService{
		SetDefaultDashboardFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/set-default", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.SetDefault)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Templates
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetTemplates_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetTemplatesFunc: func(_ context.Context, _ *string) ([]models.Dashboard, error) {
			return []models.Dashboard{{Name: "T1"}}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/templates", nil, nil, h.GetTemplates)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_CreateFromTemplate_Success(t *testing.T) {
	svc := &mockDashboardService{
		CreateFromTemplateFunc: func(_ context.Context, _, _, _ string) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: "new"}, Name: "FromT"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"name": "Clone"}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/t1/from-template", body,
		gin.Params{{Key: "id", Value: "t1"}}, h.CreateFromTemplate)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Versions
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetVersions_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetVersionsFunc: func(_ context.Context, _ string) ([]models.DashboardVersion, error) {
			return []models.DashboardVersion{{ID: "v1", DashboardID: "d1"}}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/d1/versions", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.GetVersions)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_CreateVersion_Success(t *testing.T) {
	svc := &mockDashboardService{
		CreateVersionFunc: func(_ context.Context, _, _, _ string) (*models.DashboardVersion, error) {
			return &models.DashboardVersion{ID: "v1", DashboardID: "d1", Comment: "c1"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"comment": "c1"}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/versions", body,
		gin.Params{{Key: "id", Value: "d1"}}, h.CreateVersion)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_RestoreVersion_Success(t *testing.T) {
	svc := &mockDashboardService{
		RestoreVersionFunc: func(_ context.Context, _, _, _ string) error { return nil },
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/d1/versions/v1/restore", nil,
		gin.Params{{Key: "id", Value: "d1"}, {Key: "versionId", Value: "v1"}}, h.RestoreVersion)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Export / Import
// ----------------------------------------------------------------------------

func TestDashboardHandler_Export_Success(t *testing.T) {
	svc := &mockDashboardService{
		ExportDashboardFunc: func(_ context.Context, _ string) (string, error) {
			return "base64-data", nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/d1/export", nil,
		gin.Params{{Key: "id", Value: "d1"}}, h.Export)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_Import_Success(t *testing.T) {
	svc := &mockDashboardService{
		ImportDashboardFunc: func(_ context.Context, _, _ string) (*models.Dashboard, error) {
			return &models.Dashboard{BaseModel: models.BaseModel{ID: "imp"}, Name: "Imported"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"config": "base64-config"}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/import", body, nil, h.Import)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Widget data
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetWidgetData_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetWidgetDataFunc: func(_ context.Context, _, _ string, _ map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"value": 42}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"endpoint": "/api/foo", "params": map[string]interface{}{"x": 1}}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/widgets/w1/data", body,
		gin.Params{{Key: "id", Value: "w1"}}, h.GetWidgetData)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetWidgetData_EmptyID(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/widgets//data", nil,
		gin.Params{{Key: "id", Value: ""}}, h.GetWidgetData)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetBatchWidgetData_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetBatchWidgetDataFunc: func(_ context.Context, _ []string, _ bool) (map[string]system.WidgetDataResult, error) {
			return map[string]system.WidgetDataResult{"w1": {Data: 1}}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	body := map[string]interface{}{"widgetIds": []string{"w1"}, "bypassCache": true}
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/widgets/batch-data", body, nil, h.GetBatchWidgetData)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetBatchWidgetData_BindError(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/widgets/batch-data", "not-json", nil, h.GetBatchWidgetData)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Endpoints
// ----------------------------------------------------------------------------

func TestDashboardHandler_GetAvailableEndpoints_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetUserAccessibleEndpointsFunc: func(_ context.Context, _ string) ([]systemServices.CategoryEndpoints, error) {
			return []systemServices.CategoryEndpoints{{Module: "System"}}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/endpoints", nil, nil, h.GetAvailableEndpoints)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_ValidateEndpoint_Success(t *testing.T) {
	svc := &mockDashboardService{
		ValidateEndpointFunc: func(_, _ string) (*systemServices.EndpointDetail, error) {
			return &systemServices.EndpointDetail{Route: "/foo"}, nil
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/endpoints/validate?route=/foo&method=GET", nil, nil, h.ValidateEndpoint)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_ValidateEndpoint_MissingParams(t *testing.T) {
	svc := &mockDashboardService{}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/endpoints/validate", nil, nil, h.ValidateEndpoint)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_GetUserEndpointsWithFilter_Success(t *testing.T) {
	svc := &mockDashboardService{
		GetUserAccessibleEndpointsFunc: func(_ context.Context, _ string) ([]systemServices.CategoryEndpoints, error) {
			return []systemServices.CategoryEndpoints{{Module: "Sys"}}, nil
		},
		FilterEndpointsByWidgetTypeFunc: func(_ []systemServices.CategoryEndpoints, _ string) []systemServices.CategoryEndpoints {
			return []systemServices.CategoryEndpoints{{Module: "Filtered"}}
		},
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "GET", "/system/dashboards/endpoints/filter?widgetType=chart", nil, nil, h.GetUserEndpointsWithFilter)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardHandler_InvalidateEndpointCache_Success(t *testing.T) {
	invalidated := false
	svc := &mockDashboardService{
		InvalidateUserCacheFunc: func(_ context.Context, _ string) { invalidated = true },
	}
	h := newDashboardHandler(t, svc)
	w := invokeDashboardHandler(t, "POST", "/system/dashboards/endpoints/cache/invalidate", nil, nil, h.InvalidateEndpointCache)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, invalidated)
}
