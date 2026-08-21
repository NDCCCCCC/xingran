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

// mockWorkstationDeviceService implements opsServices.WorkstationDeviceService.
type mockWorkstationDeviceService struct {
	GetDevicesByWorkstationFunc func(ctx context.Context, wsID string, source ...string) ([]*models.WorkstationDevice, error)
	AddDeviceManualFunc         func(ctx context.Context, req *opsServices.AddDeviceRequest) (*models.WorkstationDevice, error)
	GetADDevicesFunc            func(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error)
	GetAssetDevicesFunc         func(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error)
	GetPhysicalDevicesFunc      func(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error)
	SetPrimaryAndSaveFunc       func(ctx context.Context, id string, req *opsServices.SetPrimaryAndSaveRequest) error
	SyncFromADFunc              func(ctx context.Context, wsID string) error
	SyncFromAssetFunc           func(ctx context.Context, wsID string) error
	UpdateDeviceFunc            func(ctx context.Context, id string, req *opsServices.UpdateDeviceRequest) error
	DeleteDeviceFunc            func(ctx context.Context, id string) error
	SetPrimaryDeviceFunc        func(ctx context.Context, id string) error
}

func (m *mockWorkstationDeviceService) GetDevicesByWorkstation(ctx context.Context, wsID string, source ...string) ([]*models.WorkstationDevice, error) {
	if m.GetDevicesByWorkstationFunc != nil {
		return m.GetDevicesByWorkstationFunc(ctx, wsID, source...)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationDeviceService) AddDeviceManual(ctx context.Context, req *opsServices.AddDeviceRequest) (*models.WorkstationDevice, error) {
	if m.AddDeviceManualFunc != nil {
		return m.AddDeviceManualFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationDeviceService) GetADDevices(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error) {
	if m.GetADDevicesFunc != nil {
		return m.GetADDevicesFunc(ctx, wsID)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationDeviceService) GetAssetDevices(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error) {
	if m.GetAssetDevicesFunc != nil {
		return m.GetAssetDevicesFunc(ctx, wsID)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationDeviceService) GetPhysicalDevices(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error) {
	if m.GetPhysicalDevicesFunc != nil {
		return m.GetPhysicalDevicesFunc(ctx, wsID)
	}
	return nil, errNotImplemented
}
func (m *mockWorkstationDeviceService) SetPrimaryAndSave(ctx context.Context, id string, req *opsServices.SetPrimaryAndSaveRequest) error {
	if m.SetPrimaryAndSaveFunc != nil {
		return m.SetPrimaryAndSaveFunc(ctx, id, req)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) SyncFromAD(ctx context.Context, wsID string) error {
	if m.SyncFromADFunc != nil {
		return m.SyncFromADFunc(ctx, wsID)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) SyncFromAsset(ctx context.Context, wsID string) error {
	if m.SyncFromAssetFunc != nil {
		return m.SyncFromAssetFunc(ctx, wsID)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) UpdateDevice(ctx context.Context, id string, req *opsServices.UpdateDeviceRequest) error {
	if m.UpdateDeviceFunc != nil {
		return m.UpdateDeviceFunc(ctx, id, req)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) DeleteDevice(ctx context.Context, id string) error {
	if m.DeleteDeviceFunc != nil {
		return m.DeleteDeviceFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) SetPrimaryDevice(ctx context.Context, id string) error {
	if m.SetPrimaryDeviceFunc != nil {
		return m.SetPrimaryDeviceFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockWorkstationDeviceService) GetADDevicesByWorkstations(ctx context.Context, ids []string) (map[string][]*models.WorkstationDevice, error) {
	return map[string][]*models.WorkstationDevice{}, nil
}
func (m *mockWorkstationDeviceService) GetAssetDevicesByWorkstations(ctx context.Context, ids []string) (map[string][]*models.WorkstationDevice, error) {
	return map[string][]*models.WorkstationDevice{}, nil
}
func (m *mockWorkstationDeviceService) GetPhysicalDevicesByWorkstations(ctx context.Context, ids []string) (map[string][]*models.WorkstationDevice, error) {
	return map[string][]*models.WorkstationDevice{}, nil
}
func (m *mockWorkstationDeviceService) GetADDevicesByUser(ctx context.Context, userID string) ([]*opsServices.ADDeviceMatch, error) {
	return nil, nil
}
func (m *mockWorkstationDeviceService) GetAssetDevicesByUser(ctx context.Context, userID, username, nickname string) ([]*opsServices.AssetDeviceMatch, error) {
	return nil, nil
}
func (m *mockWorkstationDeviceService) SetPrimaryAndSaveBySerial(ctx context.Context, workstationID, serial string, req *opsServices.SetPrimaryAndSaveRequest) error {
	return nil
}

func newWorkstationDeviceRouter(h *WorkstationDeviceHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/workstation-device/:id", h.GetByWorkstation},
		{http.MethodPost, "/workstation-device/manual", h.AddManual},
		{http.MethodPost, "/workstation-device/:id/ad", h.GetADDevices},
		{http.MethodPost, "/workstation-device/:id/asset", h.GetAssetDevices},
		{http.MethodPost, "/workstation-device/:id/physical", h.GetPhysicalDevices},
		{http.MethodPost, "/workstation-device/:id/set-primary-and-save", h.SetPrimaryAndSave},
		{http.MethodPost, "/workstation-device/sync-ad", h.SyncAD},
		{http.MethodPost, "/workstation-device/sync-asset", h.SyncAsset},
		{http.MethodPost, "/workstation-device/:id/update", h.Update},
		{http.MethodPost, "/workstation-device/:id/delete", h.Delete},
		{http.MethodPost, "/workstation-device/:id/set-primary", h.SetPrimary},
	})
}

func TestWorkstationDeviceHandler_GetByWorkstation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{GetDevicesByWorkstationFunc: func(_ context.Context, wsID string, _ ...string) ([]*models.WorkstationDevice, error) {
		assert.NotEmpty(t, wsID)
		return []*models.WorkstationDevice{{BaseModel: models.BaseModel{ID: "d1"}}}, nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/ws1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWorkstationDeviceHandler_GetByWorkstation_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{GetDevicesByWorkstationFunc: func(_ context.Context, _ string, _ ...string) ([]*models.WorkstationDevice, error) {
		return nil, errors.New("g")
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/ws1", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code) // response.Error(c, err) with err.Error → 500
}
func TestWorkstationDeviceHandler_AddManual_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{AddDeviceManualFunc: func(_ context.Context, _ *opsServices.AddDeviceRequest) (*models.WorkstationDevice, error) {
		called = true
		return &models.WorkstationDevice{}, nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/manual", `{"workstationId":"ws1","deviceSerial":"sn1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_AddManual_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkstationDeviceHandler(&mockWorkstationDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/manual", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWorkstationDeviceHandler_AddManual_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{AddDeviceManualFunc: func(_ context.Context, _ *opsServices.AddDeviceRequest) (*models.WorkstationDevice, error) {
		return nil, errors.New("a")
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/manual", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // AppError passthrough
}
func TestWorkstationDeviceHandler_GetADDevices_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{GetADDevicesFunc: func(_ context.Context, _ string) ([]*models.WorkstationDevice, error) {
		return []*models.WorkstationDevice{}, nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/ws1/ad", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWorkstationDeviceHandler_GetAssetDevices_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{GetAssetDevicesFunc: func(_ context.Context, _ string) ([]*models.WorkstationDevice, error) {
		return []*models.WorkstationDevice{}, nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/ws1/asset", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWorkstationDeviceHandler_GetPhysicalDevices_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{GetPhysicalDevicesFunc: func(_ context.Context, _ string) ([]*models.WorkstationDevice, error) {
		return []*models.WorkstationDevice{}, nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/ws1/physical", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWorkstationDeviceHandler_SetPrimaryAndSave_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{SetPrimaryAndSaveFunc: func(_ context.Context, id string, _ *opsServices.SetPrimaryAndSaveRequest) error {
		called = true
		assert.NotEmpty(t, id)
		return nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/x1/set-primary-and-save", `{"workstationId":"ws1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_SetPrimaryAndSave_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkstationDeviceHandler(&mockWorkstationDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/x1/set-primary-and-save", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWorkstationDeviceHandler_SyncAD_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{SyncFromADFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/sync-ad", `{"workstation_id":"ws1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_SyncAD_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkstationDeviceHandler(&mockWorkstationDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/sync-ad", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWorkstationDeviceHandler_SyncAsset_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{SyncFromAssetFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/sync-asset", `{"workstation_id":"ws1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_SyncAsset_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkstationDeviceHandler(&mockWorkstationDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/sync-asset", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWorkstationDeviceHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{UpdateDeviceFunc: func(_ context.Context, _ string, _ *opsServices.UpdateDeviceRequest) error {
		called = true
		return nil
	}}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkstationDeviceHandler(&mockWorkstationDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWorkstationDeviceHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{DeleteDeviceFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{DeleteDeviceFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code) // AppError passthrough
}
func TestWorkstationDeviceHandler_SetPrimary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWorkstationDeviceService{SetPrimaryDeviceFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/set-primary", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWorkstationDeviceHandler_SetPrimary_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWorkstationDeviceService{SetPrimaryDeviceFunc: func(_ context.Context, _ string) error { return errors.New("p") }}
	h := NewWorkstationDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWorkstationDeviceRouter(h), http.MethodPost, "/workstation-device/d1/set-primary", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code) // AppError passthrough
}
func TestWorkstationDeviceHandler_WithCore_NilSafe(t *testing.T) {
	var h *WorkstationDeviceHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}