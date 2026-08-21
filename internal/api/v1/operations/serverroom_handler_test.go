package operations

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	apiOpsRequests "github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockServerRoomService implements opsServices.ServerRoomService.
type mockServerRoomService struct {
	CreateFunc                    func(ctx context.Context, r *opsModels.OpsServerRoom) error
	UpdateFunc                    func(ctx context.Context, r *opsModels.OpsServerRoom) error
	DeleteFunc                    func(ctx context.Context, id string) error
	GetByIDFunc                   func(ctx context.Context, id string) (*opsModels.OpsServerRoom, error)
	ListFunc                      func(ctx context.Context, req apiOpsRequests.ServerRoomListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc               func(ctx context.Context, ids []string) error
	StatisticsFunc                func(ctx context.Context) (*opsServices.ServerRoomStatisticsResult, error)
	SearchServerRoomOptionsFunc   func(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockServerRoomService) Create(ctx context.Context, r *opsModels.OpsServerRoom) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, r)
	}
	return errNotImplemented
}
func (m *mockServerRoomService) Update(ctx context.Context, r *opsModels.OpsServerRoom) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, r)
	}
	return errNotImplemented
}
func (m *mockServerRoomService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockServerRoomService) GetByID(ctx context.Context, id string) (*opsModels.OpsServerRoom, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockServerRoomService) List(ctx context.Context, req apiOpsRequests.ServerRoomListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockServerRoomService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockServerRoomService) Statistics(ctx context.Context) (*opsServices.ServerRoomStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockServerRoomService) SearchServerRoomOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchServerRoomOptionsFunc != nil {
		return m.SearchServerRoomOptionsFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newServerRoomRouter(h *ServerRoomHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/serverRooms", h.Create},
		{http.MethodPost, "/serverRooms/list", h.List},
		{http.MethodPost, "/serverRooms/:id", h.GetByID},
		{http.MethodPost, "/serverRooms/:id/update", h.Update},
		{http.MethodPost, "/serverRooms/:id/delete", h.Delete},
		{http.MethodPost, "/serverRooms/batch", h.BatchOperation},
		{http.MethodPost, "/serverRooms/statistics", h.Statistics},
		{http.MethodPost, "/serverRooms/search-options", h.SearchServerRoomOptions},
	})
}

func TestServerRoomHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockServerRoomService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsServerRoom) error { called = true; return nil },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms", `{"name":"R1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestServerRoomHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewServerRoomHandler(&mockServerRoomService{}).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsServerRoom) error { return errors.New("c err") },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestServerRoomHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		ListFunc: func(_ context.Context, _ apiOpsRequests.ServerRoomListRequest) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 4}, nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestServerRoomHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewServerRoomHandler(&mockServerRoomService{}).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		ListFunc: func(_ context.Context, _ apiOpsRequests.ServerRoomListRequest) (*opsServices.PageResult, error) {
			return nil, errors.New("l err")
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestServerRoomHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsServerRoom, error) {
			return &opsModels.OpsServerRoom{BaseModel: models.BaseModel{ID: id}}, nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestServerRoomHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsServerRoom, error) {
			return nil, errors.New("nf")
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsServerRoom) error { return nil },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestServerRoomHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewServerRoomHandler(&mockServerRoomService{}).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsServerRoom) error { return errors.New("u err") },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestServerRoomHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockServerRoomService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestServerRoomHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/sr1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestServerRoomHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockServerRoomService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestServerRoomHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewServerRoomHandler(&mockServerRoomService{}).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewServerRoomHandler(&mockServerRoomService{}).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd err") },
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestServerRoomHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		StatisticsFunc: func(_ context.Context) (*opsServices.ServerRoomStatisticsResult, error) {
			return &opsServices.ServerRoomStatisticsResult{Total: 2, Active: 1, Inactive: 1}, nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestServerRoomHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		StatisticsFunc: func(_ context.Context) (*opsServices.ServerRoomStatisticsResult, error) {
			return nil, errors.New("s err")
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_SearchServerRoomOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		SearchServerRoomOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "SR"}}, nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestServerRoomHandler_SearchServerRoomOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockServerRoomService{
		SearchServerRoomOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			return nil, nil
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestServerRoomHandler_SearchServerRoomOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockServerRoomService{
		SearchServerRoomOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("o err")
		},
	}
	h := NewServerRoomHandler(svc).WithCore(newTestCore(t))
	r := newServerRoomRouter(h)
	w := httpDo(r, http.MethodPost, "/serverRooms/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestServerRoomHandler_WithCore_NilSafe(t *testing.T) {
	var h *ServerRoomHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}
