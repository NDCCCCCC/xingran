package operations

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockDedicatedLineService implements opsServices.DedicatedLineService.
type mockDedicatedLineService struct {
	CreateFunc                       func(ctx context.Context, l *opsModels.OpsDedicatedLine) error
	UpdateFunc                       func(ctx context.Context, l *opsModels.OpsDedicatedLine) error
	DeleteFunc                       func(ctx context.Context, id string) error
	GetByIDFunc                      func(ctx context.Context, id string) (*opsModels.OpsDedicatedLine, error)
	ListFunc                         func(ctx context.Context, req requests.DedicatedLineListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc                  func(ctx context.Context, ids []string) error
	StatisticsFunc                   func(ctx context.Context) (*opsServices.DedicatedLineStatisticsResult, error)
	SearchDedicatedLineOptionsFunc   func(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockDedicatedLineService) Create(ctx context.Context, l *opsModels.OpsDedicatedLine) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, l)
	}
	return errNotImplemented
}
func (m *mockDedicatedLineService) Update(ctx context.Context, l *opsModels.OpsDedicatedLine) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, l)
	}
	return errNotImplemented
}
func (m *mockDedicatedLineService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockDedicatedLineService) GetByID(ctx context.Context, id string) (*opsModels.OpsDedicatedLine, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockDedicatedLineService) List(ctx context.Context, req requests.DedicatedLineListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockDedicatedLineService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockDedicatedLineService) Statistics(ctx context.Context) (*opsServices.DedicatedLineStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockDedicatedLineService) SearchDedicatedLineOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchDedicatedLineOptionsFunc != nil {
		return m.SearchDedicatedLineOptionsFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newDedicatedLineRouter(h *DedicatedLineHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/lines", h.Create},
		{http.MethodPost, "/lines/list", h.List},
		{http.MethodPost, "/lines/:id", h.GetByID},
		{http.MethodPost, "/lines/:id/update", h.Update},
		{http.MethodPost, "/lines/:id/delete", h.Delete},
		{http.MethodPost, "/lines/batch", h.BatchOperation},
		{http.MethodPost, "/lines/statistics", h.Statistics},
		{http.MethodPost, "/lines/search-options", h.SearchDedicatedLineOptions},
	})
}

func TestDedicatedLineHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDedicatedLineService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsDedicatedLine) error { called = true; return nil },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines", `{"name":"L1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDedicatedLineHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDedicatedLineHandler(&mockDedicatedLineService{}).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsDedicatedLine) error { return errors.New("c err") },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDedicatedLineHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		ListFunc: func(_ context.Context, _ requests.DedicatedLineListRequest) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 6}, nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDedicatedLineHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDedicatedLineHandler(&mockDedicatedLineService{}).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		ListFunc: func(_ context.Context, _ requests.DedicatedLineListRequest) (*opsServices.PageResult, error) {
			return nil, errors.New("l err")
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDedicatedLineHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsDedicatedLine, error) {
			return &opsModels.OpsDedicatedLine{BaseModel: models.BaseModel{ID: id}}, nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDedicatedLineHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsDedicatedLine, error) {
			return nil, errors.New("nf")
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsDedicatedLine) error { return nil },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDedicatedLineHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDedicatedLineHandler(&mockDedicatedLineService{}).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsDedicatedLine) error { return errors.New("u err") },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDedicatedLineHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDedicatedLineService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDedicatedLineHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/l1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDedicatedLineHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDedicatedLineService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDedicatedLineHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDedicatedLineHandler(&mockDedicatedLineService{}).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDedicatedLineHandler(&mockDedicatedLineService{}).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd err") },
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDedicatedLineHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		StatisticsFunc: func(_ context.Context) (*opsServices.DedicatedLineStatisticsResult, error) {
			return &opsServices.DedicatedLineStatisticsResult{Total: 8}, nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDedicatedLineHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		StatisticsFunc: func(_ context.Context) (*opsServices.DedicatedLineStatisticsResult, error) {
			return nil, errors.New("s err")
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_SearchDedicatedLineOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		SearchDedicatedLineOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "LL"}}, nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDedicatedLineHandler_SearchDedicatedLineOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDedicatedLineService{
		SearchDedicatedLineOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			return nil, nil
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDedicatedLineHandler_SearchDedicatedLineOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDedicatedLineService{
		SearchDedicatedLineOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("o err")
		},
	}
	h := NewDedicatedLineHandler(svc).WithCore(newTestCore(t))
	r := newDedicatedLineRouter(h)
	w := httpDo(r, http.MethodPost, "/lines/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDedicatedLineHandler_WithCore_NilSafe(t *testing.T) {
	var h *DedicatedLineHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}

// mockDoorService implements opsServices.DoorService.
type mockDoorService struct {
	CreateFunc      func(ctx context.Context, d *opsModels.Door) error
	UpdateFunc      func(ctx context.Context, d *opsModels.Door) error
	DeleteFunc      func(ctx context.Context, id string) error
	GetByIDFunc     func(ctx context.Context, id string) (*opsModels.Door, error)
	ListFunc        func(ctx context.Context, req requests.DoorListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc func(ctx context.Context, ids []string) error
}

func (m *mockDoorService) Create(ctx context.Context, d *opsModels.Door) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, d)
	}
	return errNotImplemented
}
func (m *mockDoorService) Update(ctx context.Context, d *opsModels.Door) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, d)
	}
	return errNotImplemented
}
func (m *mockDoorService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockDoorService) GetByID(ctx context.Context, id string) (*opsModels.Door, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockDoorService) List(ctx context.Context, req requests.DoorListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockDoorService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}

func newDoorRouter(h *DoorHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/doors", h.Create},
		{http.MethodPost, "/doors/list", h.List},
		{http.MethodPost, "/doors/:id", h.GetByID},
		{http.MethodPost, "/doors/:id/update", h.Update},
		{http.MethodPost, "/doors/:id/delete", h.Delete},
		{http.MethodPost, "/doors/batch", h.BatchOperation},
	})
}

func TestDoorHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDoorService{
		CreateFunc: func(_ context.Context, _ *opsModels.Door) error { called = true; return nil },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors", `{"name":"D1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDoorHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDoorHandler(&mockDoorService{}).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		CreateFunc: func(_ context.Context, _ *opsModels.Door) error { return errors.New("c err") },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDoorHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		ListFunc: func(_ context.Context, _ requests.DoorListRequest) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 3}, nil
		},
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDoorHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDoorHandler(&mockDoorService{}).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		ListFunc: func(_ context.Context, _ requests.DoorListRequest) (*opsServices.PageResult, error) {
			return nil, errors.New("l err")
		},
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDoorHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		GetByIDFunc: func(_ context.Context, id string) (*opsModels.Door, error) {
			return &opsModels.Door{BaseModel: models.BaseModel{ID: id}}, nil
		},
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDoorHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.Door, error) {
			return nil, errors.New("nf")
		},
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		UpdateFunc: func(_ context.Context, _ *opsModels.Door) error { return nil },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestDoorHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDoorHandler(&mockDoorService{}).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		UpdateFunc: func(_ context.Context, _ *opsModels.Door) error { return errors.New("u err") },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDoorHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDoorService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDoorHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/d1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDoorHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDoorService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestDoorHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDoorHandler(&mockDoorService{}).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDoorHandler(&mockDoorService{}).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestDoorHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDoorService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd err") },
	}
	h := NewDoorHandler(svc).WithCore(newTestCore(t))
	r := newDoorRouter(h)
	w := httpDo(r, http.MethodPost, "/doors/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestDoorHandler_WithCore_NilSafe(t *testing.T) {
	var h *DoorHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}
