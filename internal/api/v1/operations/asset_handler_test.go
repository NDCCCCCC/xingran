package operations

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockAssetService implements opsServices.AssetService.
type mockAssetService struct {
	CreateFunc           func(ctx context.Context, a *models.Asset) error
	UpdateFunc           func(ctx context.Context, a *models.Asset) error
	DeleteFunc           func(ctx context.Context, id string) error
	GetByIDFunc          func(ctx context.Context, id string) (*models.Asset, error)
	GetByDeviceSNFunc    func(ctx context.Context, sn string) (*models.Asset, error)
	ListFunc             func(ctx context.Context, p map[string]interface{}) (*opsServices.PageResult, error)
	BatchDeleteFunc      func(ctx context.Context, ids []string) error
	GetDeviceTypesFunc   func(ctx context.Context) ([]opsServices.DeviceTypeCount, error)
	GetDeviceCatsFunc    func(ctx context.Context) ([]opsServices.DeviceTypeCount, error)
	GetStatusValuesFunc  func(ctx context.Context) ([]opsServices.DeviceTypeCount, error)
	StatisticsFunc       func(ctx context.Context) (*opsServices.AssetStatisticsResult, error)
}

func (m *mockAssetService) Create(ctx context.Context, a *models.Asset) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, a)
	}
	return errNotImplemented
}
func (m *mockAssetService) Update(ctx context.Context, a *models.Asset) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, a)
	}
	return errNotImplemented
}
func (m *mockAssetService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockAssetService) GetByID(ctx context.Context, id string) (*models.Asset, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) GetByDeviceSN(ctx context.Context, sn string) (*models.Asset, error) {
	if m.GetByDeviceSNFunc != nil {
		return m.GetByDeviceSNFunc(ctx, sn)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) List(ctx context.Context, p map[string]interface{}) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, p)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockAssetService) GetDeviceTypes(ctx context.Context) ([]opsServices.DeviceTypeCount, error) {
	if m.GetDeviceTypesFunc != nil {
		return m.GetDeviceTypesFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) GetDeviceCategories(ctx context.Context) ([]opsServices.DeviceTypeCount, error) {
	if m.GetDeviceCatsFunc != nil {
		return m.GetDeviceCatsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) GetStatusValues(ctx context.Context) ([]opsServices.DeviceTypeCount, error) {
	if m.GetStatusValuesFunc != nil {
		return m.GetStatusValuesFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockAssetService) Statistics(ctx context.Context) (*opsServices.AssetStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}

func newAssetRouter(h *AssetHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/assets", h.Create},
		{http.MethodPost, "/assets/list", h.List},
		{http.MethodPost, "/assets/:id", h.GetByID},
		{http.MethodPost, "/assets/:id/update", h.Update},
		{http.MethodPost, "/assets/:id/delete", h.Delete},
		{http.MethodPost, "/assets/batch", h.BatchOperation},
		{http.MethodPost, "/assets/device-types", h.GetDeviceTypes},
		{http.MethodPost, "/assets/device-categories", h.GetDeviceCategories},
		{http.MethodPost, "/assets/status-values", h.GetStatusValues},
		{http.MethodPost, "/assets/statistics", h.Statistics},
		{http.MethodPost, "/assets/serial/:serial", h.SearchBySerial},
	})
}

func TestAssetHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockAssetService{CreateFunc: func(_ context.Context, _ *models.Asset) error { called = true; return nil }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets", `{"name":"A1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestAssetHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{CreateFunc: func(_ context.Context, _ *models.Asset) error { return errors.New("c") }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockAssetService{UpdateFunc: func(_ context.Context, _ *models.Asset) error { called = true; return nil }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestAssetHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{UpdateFunc: func(_ context.Context, _ *models.Asset) error { return errors.New("u") }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1/update", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockAssetService{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestAssetHandler_Delete_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1/delete", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetByIDFunc: func(_ context.Context, id string) (*models.Asset, error) {
		return &models.Asset{ID: id}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_GetByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetByIDFunc: func(_ context.Context, _ string) (*models.Asset, error) {
		return nil, errors.New("g")
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/a1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
		return &opsServices.PageResult{Total: 9}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{ListFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/list", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockAssetService{BatchDeleteFunc: func(_ context.Context, ids []string) error {
		called = true
		assert.Len(t, ids, 2)
		return nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/batch", `{"action":"delete","ids":["x","y"]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestAssetHandler_BatchOperation_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/batch", `{"action":"x","ids":["a"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_BatchOperation_DeleteError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd") }}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/batch", `{"action":"delete","ids":["x"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_GetDeviceTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetDeviceTypesFunc: func(_ context.Context) ([]opsServices.DeviceTypeCount, error) {
		return []opsServices.DeviceTypeCount{{Value: "switch", Count: 5}}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/device-types", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_GetDeviceTypes_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAssetHandler(&mockAssetService{}).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/device-types", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestAssetHandler_GetDeviceTypes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetDeviceTypesFunc: func(_ context.Context) ([]opsServices.DeviceTypeCount, error) {
		return nil, errors.New("dt")
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/device-types", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_GetDeviceCategories_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetDeviceCatsFunc: func(_ context.Context) ([]opsServices.DeviceTypeCount, error) {
		return []opsServices.DeviceTypeCount{{Value: "network", Count: 2}}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/device-categories", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_GetStatusValues_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetStatusValuesFunc: func(_ context.Context) ([]opsServices.DeviceTypeCount, error) {
		return []opsServices.DeviceTypeCount{{Value: "in_use", Count: 7}}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/status-values", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{StatisticsFunc: func(_ context.Context) (*opsServices.AssetStatisticsResult, error) {
		return &opsServices.AssetStatisticsResult{Total: 100, Normal: 80, Stopped: 15, NBF: 5}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{StatisticsFunc: func(_ context.Context) (*opsServices.AssetStatisticsResult, error) {
		return nil, errors.New("s")
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_SearchBySerial_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetByDeviceSNFunc: func(_ context.Context, sn string) (*models.Asset, error) {
		return &models.Asset{DeviceSN: sn}, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/serial/SN-1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAssetHandler_SearchBySerial_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetByDeviceSNFunc: func(_ context.Context, _ string) (*models.Asset, error) {
		return nil, nil
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/serial/MISSING", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_SearchBySerial_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAssetService{GetByDeviceSNFunc: func(_ context.Context, _ string) (*models.Asset, error) {
		return nil, errors.New("s")
	}}
	h := NewAssetHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newAssetRouter(h), http.MethodPost, "/assets/serial/SN", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestAssetHandler_WithCore_NilSafe(t *testing.T) {
	var h *AssetHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}