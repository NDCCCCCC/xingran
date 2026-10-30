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

// mockInfoPointService implements opsServices.InfoPointService.
type mockInfoPointService struct {
	CreateFunc                   func(ctx context.Context, p *opsModels.OpsInfoPoint) error
	UpdateFunc                   func(ctx context.Context, p *opsModels.OpsInfoPoint) error
	DeleteFunc                   func(ctx context.Context, id string) error
	GetByIDFunc                  func(ctx context.Context, id string) (*opsModels.OpsInfoPoint, error)
	ListFunc                     func(ctx context.Context, req requests.InfoPointListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc              func(ctx context.Context, ids []string) error
	StatisticsFunc               func(ctx context.Context) (*opsServices.InfoPointStatisticsResult, error)
	SearchInfoPointOptionsFunc   func(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockInfoPointService) Create(ctx context.Context, p *opsModels.OpsInfoPoint) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, p)
	}
	return errNotImplemented
}
func (m *mockInfoPointService) Update(ctx context.Context, p *opsModels.OpsInfoPoint) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, p)
	}
	return errNotImplemented
}
func (m *mockInfoPointService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockInfoPointService) GetByID(ctx context.Context, id string) (*opsModels.OpsInfoPoint, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockInfoPointService) List(ctx context.Context, req requests.InfoPointListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockInfoPointService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockInfoPointService) Statistics(ctx context.Context) (*opsServices.InfoPointStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockInfoPointService) SearchInfoPointOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchInfoPointOptionsFunc != nil {
		return m.SearchInfoPointOptionsFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newInfoPointRouter(h *InfoPointHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/infoPoints", h.Create},
		{http.MethodPost, "/infoPoints/list", h.List},
		{http.MethodPost, "/infoPoints/:id", h.GetByID},
		{http.MethodPost, "/infoPoints/:id/update", h.Update},
		{http.MethodPost, "/infoPoints/:id/delete", h.Delete},
		{http.MethodPost, "/infoPoints/batch", h.BatchOperation},
		{http.MethodPost, "/infoPoints/statistics", h.Statistics},
		{http.MethodPost, "/infoPoints/search-options", h.SearchInfoPointOptions},
	})
}

// TestInfoPointHandler_Create_Success
func TestInfoPointHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockInfoPointService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsInfoPoint) error { called = true; return nil },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints", `{"name":"IP1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestInfoPointHandler_Create_BindError
func TestInfoPointHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInfoPointHandler(&mockInfoPointService{}).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_Create_ServiceError
func TestInfoPointHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsInfoPoint) error { return errors.New("c err") },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestInfoPointHandler_List_Success
func TestInfoPointHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		ListFunc: func(_ context.Context, _ requests.InfoPointListRequest) (*opsServices.PageResult, error) {
			return &opsServices.PageResult{Total: 11}, nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInfoPointHandler_List_BindError
func TestInfoPointHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInfoPointHandler(&mockInfoPointService{}).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_List_ServiceError
func TestInfoPointHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		ListFunc: func(_ context.Context, _ requests.InfoPointListRequest) (*opsServices.PageResult, error) {
			return nil, errors.New("l err")
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestInfoPointHandler_GetByID_Success
func TestInfoPointHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsInfoPoint, error) {
			return &opsModels.OpsInfoPoint{BaseModel: models.BaseModel{ID: id}}, nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInfoPointHandler_GetByID_NotFound
func TestInfoPointHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsInfoPoint, error) {
			return nil, errors.New("nf")
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_Update_Success
func TestInfoPointHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsInfoPoint) error { return nil },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInfoPointHandler_Update_BindError
func TestInfoPointHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInfoPointHandler(&mockInfoPointService{}).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_Update_ServiceError
func TestInfoPointHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		UpdateFunc: func(_ context.Context, _ *opsModels.OpsInfoPoint) error { return errors.New("u err") },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1/update", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestInfoPointHandler_Delete_Success
func TestInfoPointHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockInfoPointService{
		DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestInfoPointHandler_Delete_Error
func TestInfoPointHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d err") },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/ip1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestInfoPointHandler_BatchOperation_Delete
func TestInfoPointHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockInfoPointService{
		BatchDeleteFunc: func(_ context.Context, ids []string) error {
			called = true
			assert.Len(t, ids, 2)
			return nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/batch", `{"ids":["a","b"],"action":"delete"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestInfoPointHandler_BatchOperation_UnsupportedAction
func TestInfoPointHandler_BatchOperation_UnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInfoPointHandler(&mockInfoPointService{}).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/batch", `{"ids":["a"],"action":"unknown"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_BatchOperation_BindError
func TestInfoPointHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInfoPointHandler(&mockInfoPointService{}).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_BatchOperation_ServiceError
func TestInfoPointHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd err") },
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/batch", `{"ids":["a"],"action":"delete"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestInfoPointHandler_Statistics_Success
func TestInfoPointHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		StatisticsFunc: func(_ context.Context) (*opsServices.InfoPointStatisticsResult, error) {
			return &opsServices.InfoPointStatisticsResult{Total: 5}, nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInfoPointHandler_Statistics_Error
func TestInfoPointHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		StatisticsFunc: func(_ context.Context) (*opsServices.InfoPointStatisticsResult, error) {
			return nil, errors.New("s err")
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_SearchInfoPointOptions_Success
func TestInfoPointHandler_SearchInfoPointOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		SearchInfoPointOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return []opsServices.DropdownOption{{Value: "1", Label: "IP"}}, nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInfoPointHandler_SearchInfoPointOptions_InvalidJSON
func TestInfoPointHandler_SearchInfoPointOptions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockInfoPointService{
		SearchInfoPointOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			called = true
			return nil, nil
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// TestInfoPointHandler_SearchInfoPointOptions_Error
func TestInfoPointHandler_SearchInfoPointOptions_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		SearchInfoPointOptionsFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
			return nil, errors.New("o err")
		},
	}
	h := NewInfoPointHandler(svc).WithCore(newTestCore(t))
	r := newInfoPointRouter(h)
	w := httpDo(r, http.MethodPost, "/infoPoints/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestInfoPointHandler_ValidateDevicePortConsistency_NoCore — short-circuit when core==nil
func TestInfoPointHandler_ValidateDevicePortConsistency_NoCore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockInfoPointService{
		CreateFunc: func(_ context.Context, _ *opsModels.OpsInfoPoint) error { return nil },
	}
	h := NewInfoPointHandler(svc) // no WithCore
	deviceID := "d1"
	portID := "p1"
	ok, msg := h.validateDevicePortConsistency(context.Background(), &deviceID, &portID)
	assert.True(t, ok)
	assert.Empty(t, msg)
}

// TestInfoPointHandler_WithCore_NilSafe
func TestInfoPointHandler_WithCore_NilSafe(t *testing.T) {
	var h *InfoPointHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}
