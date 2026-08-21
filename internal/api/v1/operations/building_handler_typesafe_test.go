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

// mockBuildingServiceTypeSafe implements opsServices.BuildingServiceTypeSafe.
type mockBuildingServiceTypeSafe struct {
	CreateFunc      func(ctx context.Context, b *opsModels.OpsBuilding) error
	UpdateFunc      func(ctx context.Context, b *opsModels.OpsBuilding) error
	DeleteFunc      func(ctx context.Context, id string) error
	GetByIDFunc     func(ctx context.Context, id string) (*opsModels.OpsBuilding, error)
	ListFunc        func(ctx context.Context, req requests.BuildingListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc func(ctx context.Context, ids []string) error
}

func (m *mockBuildingServiceTypeSafe) Create(ctx context.Context, b *opsModels.OpsBuilding) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, b)
	}
	return errNotImplemented
}
func (m *mockBuildingServiceTypeSafe) Update(ctx context.Context, b *opsModels.OpsBuilding) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, b)
	}
	return errNotImplemented
}
func (m *mockBuildingServiceTypeSafe) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockBuildingServiceTypeSafe) GetByID(ctx context.Context, id string) (*opsModels.OpsBuilding, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockBuildingServiceTypeSafe) List(ctx context.Context, req requests.BuildingListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockBuildingServiceTypeSafe) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}

func newBuildingTypeSafeRouter(h *BuildingHandlerTypeSafe) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/buildings", h.Create},
		{http.MethodPost, "/buildings/list", h.List},
		{http.MethodPost, "/buildings/:id", h.GetByID},
		{http.MethodPost, "/buildings/:id/update", h.Update},
		{http.MethodPost, "/buildings/:id/delete", h.Delete},
		{http.MethodPost, "/buildings/batch", h.BatchOperation},
		{http.MethodPost, "/buildings/geocode", h.Geocode},
	})
}

func TestBuildingHandlerTypeSafe_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingServiceTypeSafe{CreateFunc: func(_ context.Context, _ *opsModels.OpsBuilding) error { called = true; return nil }}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings", `{"name":"B1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestBuildingHandlerTypeSafe_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{CreateFunc: func(_ context.Context, _ *opsModels.OpsBuilding) error { return errors.New("c") }}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestBuildingHandlerTypeSafe_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{ListFunc: func(_ context.Context, _ requests.BuildingListRequest) (*opsServices.PageResult, error) {
		return &opsServices.PageResult{Total: 5}, nil
	}}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestBuildingHandlerTypeSafe_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{ListFunc: func(_ context.Context, _ requests.BuildingListRequest) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestBuildingHandlerTypeSafe_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsBuilding, error) {
		return &opsModels.OpsBuilding{BaseModel: models.BaseModel{ID: id}}, nil
	}}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/b1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestBuildingHandlerTypeSafe_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsBuilding, error) {
		return nil, errors.New("nf")
	}}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestBuildingHandlerTypeSafe_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingServiceTypeSafe{UpdateFunc: func(_ context.Context, _ *opsModels.OpsBuilding) error { called = true; return nil }}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/b1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestBuildingHandlerTypeSafe_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/b1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingServiceTypeSafe{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/b1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestBuildingHandlerTypeSafe_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/b1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestBuildingHandlerTypeSafe_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockBuildingServiceTypeSafe{BatchDeleteFunc: func(_ context.Context, ids []string) error {
		called = true
		assert.Len(t, ids, 2)
		return nil
	}}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/batch", `{"action":"delete","ids":["a","b"]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestBuildingHandlerTypeSafe_BatchOperation_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/batch", `{"action":"x","ids":["a"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestBuildingHandlerTypeSafe_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_Geocode_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/geocode", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_Geocode_EmptyAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/geocode", `{"address":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestBuildingHandlerTypeSafe_Geocode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, opsServices.NewGeocodingService(""))
	w := httpDo(newBuildingTypeSafeRouter(h), http.MethodPost, "/buildings/geocode", `{"address":"某地址"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // geocode with empty API key fails
}

// silence unused import
var _ = opsServices.NewGeocodingService("").Geocode