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

// mockLocationAliasService implements opsServices.LocationAliasService.
type mockLocationAliasService struct {
	ListFunc   func(ctx context.Context, pageNum, pageSize int) (*opsServices.PageResult, error)
	GetByIDFunc func(ctx context.Context, id string) (*models.SysDeptLocationAlias, error)
	CreateFunc func(ctx context.Context, req *opsServices.LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error)
	UpdateFunc func(ctx context.Context, id string, req *opsServices.LocationAliasUpdateRequest) error
	DeleteFunc func(ctx context.Context, id string) error
}

func (m *mockLocationAliasService) List(ctx context.Context, pageNum, pageSize int) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, pageNum, pageSize)
	}
	return nil, errNotImplemented
}
func (m *mockLocationAliasService) GetByID(ctx context.Context, id string) (*models.SysDeptLocationAlias, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockLocationAliasService) Create(ctx context.Context, req *opsServices.LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockLocationAliasService) Update(ctx context.Context, id string, req *opsServices.LocationAliasUpdateRequest) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return errNotImplemented
}
func (m *mockLocationAliasService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}

func newLocationAliasRouter(h *LocationAliasHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/location-alias/list", h.List},
		{http.MethodPost, "/location-alias", h.Create},
		{http.MethodPost, "/location-alias/:id/update", h.Update},
		{http.MethodPost, "/location-alias/:id/delete", h.Delete},
	})
}

func TestLocationAliasHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockLocationAliasService{ListFunc: func(_ context.Context, pn, ps int) (*opsServices.PageResult, error) {
		called = true
		assert.Equal(t, 2, pn)
		assert.Equal(t, 20, ps)
		return &opsServices.PageResult{Total: 3}, nil
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/list", `{"pageNum":2,"pageSize":20}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestLocationAliasHandler_List_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockLocationAliasService{ListFunc: func(_ context.Context, pn, ps int) (*opsServices.PageResult, error) {
		called = true
		assert.Equal(t, 1, pn)
		assert.Equal(t, 10, ps)
		return &opsServices.PageResult{}, nil
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/list", ``)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestLocationAliasHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockLocationAliasService{ListFunc: func(_ context.Context, _, _ int) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // apperrors.Wrap CodeServerError → 500
}
func TestLocationAliasHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockLocationAliasService{CreateFunc: func(_ context.Context, _ *opsServices.LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error) {
		called = true
		return &models.SysDeptLocationAlias{}, nil
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias", `{"deptId":"d1","locationId":"l1","scope":"workstation"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestLocationAliasHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocationAliasHandler(&mockLocationAliasService{}).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestLocationAliasHandler_Create_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockLocationAliasService{CreateFunc: func(_ context.Context, _ *opsServices.LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error) {
		return nil, errors.New("自映射不允许")
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestLocationAliasHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockLocationAliasService{UpdateFunc: func(_ context.Context, id string, _ *opsServices.LocationAliasUpdateRequest) error {
		called = true
		assert.NotEmpty(t, id)
		return nil
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/abc/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestLocationAliasHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocationAliasHandler(&mockLocationAliasService{}).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/abc/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestLocationAliasHandler_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockLocationAliasService{UpdateFunc: func(_ context.Context, _ string, _ *opsServices.LocationAliasUpdateRequest) error {
		return errors.New("u")
	}}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/abc/update", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestLocationAliasHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockLocationAliasService{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/abc/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestLocationAliasHandler_Delete_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockLocationAliasService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewLocationAliasHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newLocationAliasRouter(h), http.MethodPost, "/location-alias/abc/delete", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg quirk
}
func TestLocationAliasHandler_WithCore_NilSafe(t *testing.T) {
	var h *LocationAliasHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}

// mockRoomDeviceService implements opsServices.RoomDeviceService.
type mockRoomDeviceService struct {
	CreateFunc    func(ctx context.Context, d *opsModels.OpsRoomDevice) error
	UpdateFunc    func(ctx context.Context, d *opsModels.OpsRoomDevice) error
	DeleteFunc    func(ctx context.Context, id string) error
	GetByIDFunc   func(ctx context.Context, id string) (*opsModels.OpsRoomDevice, error)
	ListFunc      func(ctx context.Context, req requests.RoomDeviceListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc func(ctx context.Context, ids []string) error
	StatisticsFunc func(ctx context.Context) (*opsServices.RoomDeviceStatisticsResult, error)
	SearchFunc    func(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error)
}

func (m *mockRoomDeviceService) Create(ctx context.Context, d *opsModels.OpsRoomDevice) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, d)
	}
	return errNotImplemented
}
func (m *mockRoomDeviceService) Update(ctx context.Context, d *opsModels.OpsRoomDevice) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, d)
	}
	return errNotImplemented
}
func (m *mockRoomDeviceService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockRoomDeviceService) GetByID(ctx context.Context, id string) (*opsModels.OpsRoomDevice, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockRoomDeviceService) List(ctx context.Context, req requests.RoomDeviceListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockRoomDeviceService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}
func (m *mockRoomDeviceService) Statistics(ctx context.Context) (*opsServices.RoomDeviceStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNotImplemented
}
func (m *mockRoomDeviceService) SearchRoomDeviceOptions(ctx context.Context, p map[string]interface{}) ([]opsServices.DropdownOption, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, p)
	}
	return nil, errNotImplemented
}

func newRoomDeviceRouter(h *RoomDeviceHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/roomDevice", h.Create},
		{http.MethodPost, "/roomDevice/list", h.List},
		{http.MethodPost, "/roomDevice/:id", h.GetByID},
		{http.MethodPost, "/roomDevice/:id/update", h.Update},
		{http.MethodPost, "/roomDevice/:id/delete", h.Delete},
		{http.MethodPost, "/roomDevice/batch", h.BatchOperation},
		{http.MethodPost, "/roomDevice/statistics", h.Statistics},
		{http.MethodPost, "/roomDevice/search-options", h.SearchRoomDeviceOptions},
	})
}

func TestRoomDeviceHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockRoomDeviceService{CreateFunc: func(_ context.Context, _ *opsModels.OpsRoomDevice) error { called = true; return nil }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice", `{"name":"D1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestRoomDeviceHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRoomDeviceHandler(&mockRoomDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestRoomDeviceHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{CreateFunc: func(_ context.Context, _ *opsModels.OpsRoomDevice) error { return errors.New("c") }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // handleServiceError CodeServerError → 500
}
func TestRoomDeviceHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{ListFunc: func(_ context.Context, _ requests.RoomDeviceListRequest) (*opsServices.PageResult, error) {
		return &opsServices.PageResult{Total: 4}, nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestRoomDeviceHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRoomDeviceHandler(&mockRoomDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestRoomDeviceHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{ListFunc: func(_ context.Context, _ requests.RoomDeviceListRequest) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // apperrors.InternalServerErrorWithMsg → 500
}
func TestRoomDeviceHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{GetByIDFunc: func(_ context.Context, id string) (*opsModels.OpsRoomDevice, error) {
		return &opsModels.OpsRoomDevice{BaseModel: models.BaseModel{ID: id}}, nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/d1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestRoomDeviceHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{GetByIDFunc: func(_ context.Context, _ string) (*opsModels.OpsRoomDevice, error) {
		return nil, errors.New("nf")
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/missing", "")
	assert.Equal(t, http.StatusNotFound, w.Code) // apperrors.RoomDeviceNotFound → 404
}
func TestRoomDeviceHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockRoomDeviceService{UpdateFunc: func(_ context.Context, _ *opsModels.OpsRoomDevice) error { called = true; return nil }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/d1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestRoomDeviceHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRoomDeviceHandler(&mockRoomDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/d1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestRoomDeviceHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockRoomDeviceService{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/d1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestRoomDeviceHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/d1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code) // handleServiceError → 500
}
func TestRoomDeviceHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockRoomDeviceService{BatchDeleteFunc: func(_ context.Context, ids []string) error {
		called = true
		assert.Len(t, ids, 2)
		return nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/batch", `{"action":"delete","ids":["a","b"]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestRoomDeviceHandler_BatchOperation_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRoomDeviceHandler(&mockRoomDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/batch", `{"action":"x","ids":["a"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestRoomDeviceHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRoomDeviceHandler(&mockRoomDeviceService{}).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestRoomDeviceHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd") }}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/batch", `{"action":"delete","ids":["a"]}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code) // handleServiceError → 500
}
func TestRoomDeviceHandler_Statistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{StatisticsFunc: func(_ context.Context) (*opsServices.RoomDeviceStatisticsResult, error) {
		return &opsServices.RoomDeviceStatisticsResult{Total: 12, Normal: 8, Fault: 3, Scrapped: 1}, nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/statistics", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestRoomDeviceHandler_Statistics_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{StatisticsFunc: func(_ context.Context) (*opsServices.RoomDeviceStatisticsResult, error) {
		return nil, errors.New("s")
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/statistics", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // int-first-arg
}
func TestRoomDeviceHandler_Search_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{SearchFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
		return []opsServices.DropdownOption{{Value: "d1", Label: "DD"}}, nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/search-options", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestRoomDeviceHandler_Search_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockRoomDeviceService{SearchFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
		called = true
		return nil, nil
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/search-options", `not-json`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestRoomDeviceHandler_Search_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockRoomDeviceService{SearchFunc: func(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
		return nil, errors.New("o")
	}}
	h := NewRoomDeviceHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newRoomDeviceRouter(h), http.MethodPost, "/roomDevice/search-options", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestRoomDeviceHandler_WithCore_NilSafe(t *testing.T) {
	var h *RoomDeviceHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}