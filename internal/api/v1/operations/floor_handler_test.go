package operations

import (
	"context"
	"encoding/json"
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

// mockFloorService implements opsServices.FloorService for handler tests.
type mockFloorService struct {
	CreateFunc              func(ctx context.Context, floor *opsModels.OpsFloor) error
	UpdateFunc              func(ctx context.Context, floor *opsModels.OpsFloor) error
	DeleteFunc              func(ctx context.Context, id string) error
	GetByIDFunc             func(ctx context.Context, id string) (*opsModels.OpsFloor, error)
	ListFunc                func(ctx context.Context, params map[string]interface{}) (*opsServices.PageResult, error)
	GetTreeFunc             func(ctx context.Context) ([]opsServices.FloorTreeNode, error)
	BatchDeleteFunc         func(ctx context.Context, ids []string) error
	StatisticsFunc          func(ctx context.Context) (*opsServices.FloorStatisticsResult, error)
	SearchFloorOptionsFunc  func(ctx context.Context, params map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockFloorService) Create(ctx context.Context, f *opsModels.OpsFloor) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, f)
	}
	return errNotImplemented
}
func (m *mockFloorService) Update(ctx context.Context, f *opsModels.OpsFloor) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, f)
	}
	return errNotImplemented
}
func (m *mockFloorService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockFloorService) GetByID(ctx context.Context, id string) (*opsModels.OpsFloor, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockFloorService) List(ctx context.Context, p map[string]interface{}) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, p)
	}
	return nil, errNotImplemented
}
func (m *mockFloorService) GetTree(ctx context.Context) ([]opsServices.FloorTreeNode, error) {
	if m.GetTreeFunc != nil {
		return m.GetTreeFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockFloorService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockFloorService) Statistics(ctx context.Context) (*opsServices.FloorStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockFloorService) SearchFloorOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchFloorOptionsFunc != nil {
		return m.SearchFloorOptionsFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newFloorRouter(h *FloorHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/floors", h.Create},
		{http.MethodPost, "/floors/list", h.List},
		{http.MethodPost, "/floors/tree", h.GetTree},
		{http.MethodPost, "/floors/:id", h.GetByID},
		{http.MethodPost, "/floors/:id/update", h.Update},
		{http.MethodPost, "/floors/:id/delete", h.Delete},
		{http.MethodPost, "/floors/batch", h.BatchOperation},
		{http.MethodPost, "/floors/statistics", h.Statistics},
		{http.MethodPost, "/floors/search-options", h.SearchFloorOptions},
	})
}

func newFloorHandler(svc opsServices.FloorService) *FloorHandler {
	return NewFloorHandler(svc).WithCore(newTestCore(&testing.T{}))
}

// TestFloorHandler_Create_Success
func TestFloorHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorService{
		CreateFunc: func(_ context.Context, f *opsModels.OpsFloor) error {
			called = true
			f.ID = "f-1"
			return nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors", `{"name":"L1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestFloorHandler_Create_BindError
func TestFloorHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newFloorHandler(&mockFloorService{}).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_Create_ServiceError
func TestFloorHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsFloor) error { return errors.New("db err") },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_List_Success
func TestFloorHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 5}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":5`)
}

// TestFloorHandler_List_BindErrorFallback — floor List uses fallback to empty params on bind error.
func TestFloorHandler_List_BindErrorFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorService{
		ListFunc: func(_ context.Context, p map[string]interface{}) (*opsServices.PageResult, error) {
			called = true
			assert.NotNil(t, p)
			return &opsServices.PageResult{}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/list", `{not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestFloorHandler_List_ServiceError
func TestFloorHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
			return nil, errors.New("query fail")
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_GetTree_Success
func TestFloorHandler_GetTree_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		GetTreeFunc: func(_ context.Context) ([]opsServices.FloorTreeNode, error) {
			return []opsServices.FloorTreeNode{{ID: "b1", Name: "B1"}}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/tree", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFloorHandler_GetTree_Error
func TestFloorHandler_GetTree_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		GetTreeFunc: func(_ context.Context) ([]opsServices.FloorTreeNode, error) {
			return nil, errors.New("tree fail")
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/tree", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_GetByID_Success
func TestFloorHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsFloor, error) {
			return &opsModels.OpsFloor{BaseModel: models.BaseModel{ID: id}, Name: "L1"}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFloorHandler_GetByID_NotFound — apperrors.FloorNotFound quirk (code 3040 → 400).
func TestFloorHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsFloor, error) {
			return nil, errors.New("not found")
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_Update_Success
func TestFloorHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		UpdateFunc: func(_ context.Context, f *opsModels.OpsFloor) error { return nil },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1/update", `{"name":"L2"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFloorHandler_Update_BindError
func TestFloorHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newFloorHandler(&mockFloorService{}).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_Update_ServiceError
func TestFloorHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsFloor) error { return errors.New("u err") },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_Delete_Success
func TestFloorHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestFloorHandler_Delete_Error
func TestFloorHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/f1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_BatchOperation_Delete
func TestFloorHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestFloorHandler_BatchOperation_UnsupportedAction
func TestFloorHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newFloorHandler(&mockFloorService{}).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_BatchOperation_BindError
func TestFloorHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newFloorHandler(&mockFloorService{}).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_BatchOperation_ServiceError
func TestFloorHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd err") },
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFloorHandler_Statistics_Success
func TestFloorHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		StatisticsFunc: func(_ context.Context) (*opsServices.FloorStatisticsResult, error) {
			return &opsServices.FloorStatisticsResult{Total: 3, Active: 2, Inactive: 1}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":3`)
}

// TestFloorHandler_Statistics_Error — same int-as-first-arg quirk as building handler.
func TestFloorHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		StatisticsFunc: func(_ context.Context) (*opsServices.FloorStatisticsResult, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_SearchFloorOptions_Success
func TestFloorHandler_SearchFloorOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		SearchFloorOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "F1"}}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFloorHandler_SearchFloorOptions_InvalidJSON
func TestFloorHandler_SearchFloorOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorService{
		SearchFloorOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			return nil, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestFloorHandler_SearchFloorOptions_Error — same int-as-first-arg quirk.
func TestFloorHandler_SearchFloorOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		SearchFloorOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("opt fail")
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFloorHandler_WithCore_NilSafe
func TestFloorHandler_WithCore_NilSafe(t *testing.T) {
	var h *FloorHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}

// TestFloorHandler_SearchFloorOptions_ResponseShape
func TestFloorHandler_SearchFloorOptions_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorService{
		SearchFloorOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "x", Label: "y"}}, nil
		},
	}
	h := newFloorHandler(svc).WithCore(newTestCore(t))
	r := newFloorRouter(h)
	w := httpDo(r, http.MethodPost, "/floors/search-options", `{}`)
	var resp struct {
		Code int                        `json:"code"`
		Data []opsServices.DropdownOption `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 1)
}

// ensure strings is referenced (used in router comment indirectly)
var _ = strings.HasSuffix
