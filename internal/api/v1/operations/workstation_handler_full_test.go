package operations

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockWorkstationService implements opsServices.WorkstationService for handler tests.
// ReconciliationService is satisfied via stubReconciliationService (see
// workstation_handler_test.go existing pattern).
type mockWorkstationService struct {
	CreateFunc                     func(ctx context.Context, w *models.Workstation) error
	UpdateFunc                     func(ctx context.Context, w *models.Workstation) error
	DeleteFunc                     func(ctx context.Context, id string) error
	GetByIDFunc                    func(ctx context.Context, id string) (*models.Workstation, error)
	ListFunc                       func(ctx context.Context, p map[string]interface{}) (*opsServices.PageResult, error)
	BatchDeleteFunc                func(ctx context.Context, ids []string) error
	BatchUpdatePositionsFunc       func(ctx context.Context, items []opsServices.PositionUpdateItem) error
	StatisticsFunc                 func(ctx context.Context, p map[string]interface{}) (*opsServices.WorkstationStatisticsResult, error)
	GetWorkstationDeptOptionsFunc   func(ctx context.Context, orgID string) ([]opsServices.DeptOption, error)
	SearchWorkstationOptionsFunc   func(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockWorkstationService) Create(ctx context.Context, w *models.Workstation) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, w)
	}
	return errNotImplemented
}
func (m *mockWorkstationService) Update(ctx context.Context, w *models.Workstation) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, w)
	}
	return errNotImplemented
}
func (m *mockWorkstationService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockWorkstationService) GetByID(ctx context.Context, id string) (*models.Workstation, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationService) List(ctx context.Context, p map[string]interface{}) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, p)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockWorkstationService) BatchUpdatePositions(ctx context.Context, items []opsServices.PositionUpdateItem) error {
	if m.BatchUpdatePositionsFunc != nil {
		return m.BatchUpdatePositionsFunc(ctx, items)
	}
	return errNotImplemented
}
func (m *mockWorkstationService) Statistics(ctx context.Context, p map[string]interface{}) (*opsServices.WorkstationStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx, p)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationService) GetWorkstationDeptOptions(ctx context.Context, orgID string) ([]opsServices.DeptOption, error) {
	if m.GetWorkstationDeptOptionsFunc != nil {
		return m.GetWorkstationDeptOptionsFunc(ctx, orgID)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationService) SearchWorkstationOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchWorkstationOptionsFunc != nil {
		return m.SearchWorkstationOptionsFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newWorkstationRouter(h *WorkstationHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/workstations", h.Create},
		{http.MethodPost, "/workstations/list", h.List},
		{http.MethodPost, "/workstations/:id", h.GetByID},
		{http.MethodPost, "/workstations/:id/update", h.Update},
		{http.MethodPost, "/workstations/:id/delete", h.Delete},
		{http.MethodPost, "/workstations/batch", h.BatchOperation},
		{http.MethodPost, "/workstations/positions", h.BatchUpdatePositions},
		{http.MethodPost, "/workstations/statistics", h.Statistics},
		{http.MethodPost, "/workstations/dept-options", h.GetWorkstationDeptOptions},
		{http.MethodPost, "/workstations/search-options", h.SearchWorkstationOptions},
	})
}

func newWorkstationHandler(svc opsServices.WorkstationService) *WorkstationHandler {
	return NewWorkstationHandler(svc).WithCore(newTestCore(&testing.T{}))
}

// TestWorkstationHandler_Create_Success
func TestWorkstationHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		CreateFunc: func(_ context.Context, _ *models.Workstation) error { called = true; return nil },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations", `{"name":"WS-1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_Create_BindError
func TestWorkstationHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newWorkstationHandler(&mockWorkstationService{}).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_Create_ServiceError
func TestWorkstationHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		CreateFunc: func(_ context.Context, _ *models.Workstation) error { return errors.New("c err") },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestWorkstationHandler_List_Success
func TestWorkstationHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 7}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":7`)
}

// TestWorkstationHandler_List_BindErrorFallback
func TestWorkstationHandler_List_BindErrorFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			called = true
			return &opsServices.PageResult{}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/list", `{not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_List_ServiceError
func TestWorkstationHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return nil, errors.New("list err")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestWorkstationHandler_GetByID_Success
func TestWorkstationHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		GetByIDFunc: func(_ context.Context, id string) (*models.Workstation, error) {
			return &models.Workstation{BaseModel: models.BaseModel{ID: id}, WorkstationName: "WS1"}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"WS1"`)
}

// TestWorkstationHandler_GetByID_NotFound
func TestWorkstationHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		GetByIDFunc: func(_ context.Context, _ string) (*models.Workstation, error) {
			return nil, errors.New("not found")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_Update_Success
func TestWorkstationHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		UpdateFunc: func(_ context.Context, _ *models.Workstation) error { return nil },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWorkstationHandler_Update_BindError
func TestWorkstationHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newWorkstationHandler(&mockWorkstationService{}).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_Update_ServiceError
func TestWorkstationHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		UpdateFunc: func(_ context.Context, _ *models.Workstation) error { return errors.New("u err") },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestWorkstationHandler_Delete_Success
func TestWorkstationHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_Delete_Error
func TestWorkstationHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/ws1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestWorkstationHandler_BatchOperation_Delete
func TestWorkstationHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_BatchOperation_UnsupportedAction
func TestWorkstationHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newWorkstationHandler(&mockWorkstationService{}).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_BatchOperation_BindError
func TestWorkstationHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newWorkstationHandler(&mockWorkstationService{}).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_BatchUpdatePositions_Success
func TestWorkstationHandler_BatchUpdatePositions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		BatchUpdatePositionsFunc: func(_ context.Context, items []opsServices.PositionUpdateItem) error {
			called = true
			assert.Len(t, items, 1)
			return nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	body := `{"items":[{"id":"a","x":1,"y":2}]}`
	w := httpDo(r, http.MethodPost, "/workstations/positions", body)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_BatchUpdatePositions_BindError
func TestWorkstationHandler_BatchUpdatePositions_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newWorkstationHandler(&mockWorkstationService{}).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/positions", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_BatchUpdatePositions_ServiceError
func TestWorkstationHandler_BatchUpdatePositions_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		BatchUpdatePositionsFunc: func(_ context.Context, _ []opsServices.PositionUpdateItem) error {
			return errors.New("pos err")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	body := `{"items":[{"id":"a","x":1,"y":2}]}`
	w := httpDo(r, http.MethodPost, "/workstations/positions", body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestWorkstationHandler_Statistics_Success
func TestWorkstationHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		StatisticsFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.WorkstationStatisticsResult, error) {
			return &opsServices.WorkstationStatisticsResult{Total: 10, Available: 7, Occupied: 3, Maintain: 0}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/statistics", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWorkstationHandler_Statistics_Error
func TestWorkstationHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		StatisticsFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.WorkstationStatisticsResult, error) {
			return nil, errors.New("stats err")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/statistics", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_GetWorkstationDeptOptions_Success
func TestWorkstationHandler_GetWorkstationDeptOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		GetWorkstationDeptOptionsFunc: func(_ context.Context, orgID string) ([]opsServices.DeptOption, error) {
			assert.Equal(t, "org-1", orgID)
			return []opsServices.DeptOption{{DeptID: "d1", DeptName: "Dept1"}}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/dept-options", `{"orgId":"org-1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWorkstationHandler_GetWorkstationDeptOptions_BindError
func TestWorkstationHandler_GetWorkstationDeptOptions_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		GetWorkstationDeptOptionsFunc: func(_ context.Context, orgID string) ([]opsServices.DeptOption, error) {
			called = true
			assert.Equal(t, "", orgID) // fallback empty
			return nil, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/dept-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_GetWorkstationDeptOptions_Error — int-first-arg quirk.
func TestWorkstationHandler_GetWorkstationDeptOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		GetWorkstationDeptOptionsFunc: func(_ context.Context, _ string) ([]opsServices.DeptOption, error) {
			return nil, errors.New("dept err")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/dept-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_SearchWorkstationOptions_Success
func TestWorkstationHandler_SearchWorkstationOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		SearchWorkstationOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "WS1"}}, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWorkstationHandler_SearchWorkstationOptions_InvalidJSON
func TestWorkstationHandler_SearchWorkstationOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationService{
		SearchWorkstationOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			return nil, nil
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestWorkstationHandler_SearchWorkstationOptions_Error
func TestWorkstationHandler_SearchWorkstationOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationService{
		SearchWorkstationOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("opt err")
		},
	}
	h := newWorkstationHandler(svc).WithCore(newTestCore(t))
	r := newWorkstationRouter(h)
	w := httpDo(r, http.MethodPost, "/workstations/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestWorkstationHandler_WithCore_NilSafe
func TestWorkstationHandler_WithCore_NilSafe(t *testing.T) {
	var h *WorkstationHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}

// silence unused import
var _ = strings.HasPrefix
var _ = opsModels.OpsBuilding{} // opsModels referenced for parity
