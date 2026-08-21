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

// mockFloorPlanTextService implements opsServices.FloorPlanTextService.
type mockFloorPlanTextService struct {
	CreateFunc    func(ctx context.Context, t *opsModels.FloorPlanText) error
	UpdateFunc    func(ctx context.Context, t *opsModels.FloorPlanText) error
	DeleteFunc    func(ctx context.Context, id string) error
	GetByIDFunc   func(ctx context.Context, id string) (*opsModels.FloorPlanText, error)
	ListFunc    func(ctx context.Context, req requests.FloorPlanTextListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc func(ctx context.Context, ids []string) error
}

func (m *mockFloorPlanTextService) Create(ctx context.Context, t *opsModels.FloorPlanText) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	return errNotImplemented
}
func (m *mockFloorPlanTextService) Update(ctx context.Context, t *opsModels.FloorPlanText) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	return errNotImplemented
}
func (m *mockFloorPlanTextService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockFloorPlanTextService) GetByID(ctx context.Context, id string) (*opsModels.FloorPlanText, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockFloorPlanTextService) List(ctx context.Context, req requests.FloorPlanTextListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockFloorPlanTextService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}

func newFloorPlanTextRouter(h *FloorPlanTextHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/floor-plan-texts", h.Create},
		{http.MethodPost, "/floor-plan-texts/list", h.List},
		{http.MethodPost, "/floor-plan-texts/:id", h.GetByID},
		{http.MethodPost, "/floor-plan-texts/:id/update", h.Update},
		{http.MethodPost, "/floor-plan-texts/:id/delete", h.Delete},
		{http.MethodPost, "/floor-plan-texts/batch", h.BatchOperation},
	})
}

func TestFloorPlanTextHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorPlanTextService{CreateFunc: func(_ context.Context, _ *opsModels.FloorPlanText) error { called = true; return nil }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts", `{"text":"hello"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestFloorPlanTextHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFloorPlanTextHandler(&mockFloorPlanTextService{}).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestFloorPlanTextHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{CreateFunc: func(_ context.Context, _ *opsModels.FloorPlanText) error { return errors.New("c") }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestFloorPlanTextHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{ListFunc: func(_ context.Context, _ requests.FloorPlanTextListRequest) (*opsServices.PageResult, error) {
		return &opsServices.PageResult{Total: 3}, nil
	}}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestFloorPlanTextHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFloorPlanTextHandler(&mockFloorPlanTextService{}).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestFloorPlanTextHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{ListFunc: func(_ context.Context, _ requests.FloorPlanTextListRequest) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestFloorPlanTextHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{GetByIDFunc: func(_ context.Context, id string) (*opsModels.FloorPlanText, error) {
		return &opsModels.FloorPlanText{BaseModel: models.BaseModel{ID: id}}, nil
	}}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/t1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestFloorPlanTextHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{GetByIDFunc: func(_ context.Context, _ string) (*opsModels.FloorPlanText, error) {
		return nil, errors.New("nf")
	}}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestFloorPlanTextHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorPlanTextService{UpdateFunc: func(_ context.Context, _ *opsModels.FloorPlanText) error { called = true; return nil }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/t1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestFloorPlanTextHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFloorPlanTextHandler(&mockFloorPlanTextService{}).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/t1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestFloorPlanTextHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorPlanTextService{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/t1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestFloorPlanTextHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/t1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestFloorPlanTextHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockFloorPlanTextService{BatchDeleteFunc: func(_ context.Context, ids []string) error {
		called = true
		assert.Len(t, ids, 2)
		return nil
	}}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/batch", `{"action":"delete","ids":["a","b"]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestFloorPlanTextHandler_BatchOperation_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFloorPlanTextHandler(&mockFloorPlanTextService{}).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/batch", `{"action":"x","ids":["a"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestFloorPlanTextHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFloorPlanTextHandler(&mockFloorPlanTextService{}).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestFloorPlanTextHandler_BatchOperation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockFloorPlanTextService{BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("bd") }}
	h := NewFloorPlanTextHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newFloorPlanTextRouter(h), http.MethodPost, "/floor-plan-texts/batch", `{"action":"delete","ids":["a"]}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestFloorPlanTextHandler_WithCore_NilSafe(t *testing.T) {
	var h *FloorPlanTextHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}

// mockWallService implements opsServices.WallService.
type mockWallService struct {
	CreateFunc    func(ctx context.Context, w *opsModels.Wall) error
	UpdateFunc    func(ctx context.Context, w *opsModels.Wall) error
	DeleteFunc    func(ctx context.Context, id string) error
	GetByIDFunc   func(ctx context.Context, id string) (*opsModels.Wall, error)
	ListFunc    func(ctx context.Context, req requests.WallListRequest) (*opsServices.PageResult, error)
	BatchDeleteFunc func(ctx context.Context, ids []string) error
}

func (m *mockWallService) Create(ctx context.Context, w *opsModels.Wall) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, w)
	}
	return errNotImplemented
}
func (m *mockWallService) Update(ctx context.Context, w *opsModels.Wall) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, w)
	}
	return errNotImplemented
}
func (m *mockWallService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNotImplemented
}
func (m *mockWallService) GetByID(ctx context.Context, id string) (*opsModels.Wall, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}
func (m *mockWallService) List(ctx context.Context, req requests.WallListRequest) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, errNotImplemented
}
func (m *mockWallService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNotImplemented
}

func newWallRouter(h *WallHandler) *gin.Engine {
	return mountRouter([]routeMount{
		{http.MethodPost, "/walls", h.Create},
		{http.MethodPost, "/walls/list", h.List},
		{http.MethodPost, "/walls/:id", h.GetByID},
		{http.MethodPost, "/walls/:id/update", h.Update},
		{http.MethodPost, "/walls/:id/delete", h.Delete},
		{http.MethodPost, "/walls/batch", h.BatchOperation},
	})
}

func TestWallHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWallService{CreateFunc: func(_ context.Context, _ *opsModels.Wall) error { called = true; return nil }}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls", `{"name":"W1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWallHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWallHandler(&mockWallService{}).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWallHandler_Create_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{CreateFunc: func(_ context.Context, _ *opsModels.Wall) error { return errors.New("c") }}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestWallHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{ListFunc: func(_ context.Context, _ requests.WallListRequest) (*opsServices.PageResult, error) {
		return &opsServices.PageResult{Total: 5}, nil
	}}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWallHandler_List_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWallHandler(&mockWallService{}).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/list", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWallHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{ListFunc: func(_ context.Context, _ requests.WallListRequest) (*opsServices.PageResult, error) {
		return nil, errors.New("l")
	}}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestWallHandler_GetByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{GetByIDFunc: func(_ context.Context, id string) (*opsModels.Wall, error) {
		return &opsModels.Wall{BaseModel: models.BaseModel{ID: id}}, nil
	}}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/w1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestWallHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{GetByIDFunc: func(_ context.Context, _ string) (*opsModels.Wall, error) {
		return nil, errors.New("nf")
	}}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/missing", "")
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestWallHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWallService{UpdateFunc: func(_ context.Context, _ *opsModels.Wall) error { called = true; return nil }}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/w1/update", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWallHandler_Update_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWallHandler(&mockWallService{}).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/w1/update", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWallHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWallService{DeleteFunc: func(_ context.Context, _ string) error { called = true; return nil }}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/w1/delete", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWallHandler_Delete_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockWallService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("d") }}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/w1/delete", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestWallHandler_BatchOperation_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockWallService{BatchDeleteFunc: func(_ context.Context, ids []string) error {
		called = true
		assert.Len(t, ids, 2)
		return nil
	}}
	h := NewWallHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/batch", `{"action":"delete","ids":["a","b"]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
func TestWallHandler_BatchOperation_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWallHandler(&mockWallService{}).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/batch", `{"action":"x","ids":["a"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code) // apperrors quirk
}
func TestWallHandler_BatchOperation_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWallHandler(&mockWallService{}).WithCore(newTestCore(t))
	w := httpDo(newWallRouter(h), http.MethodPost, "/walls/batch", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
func TestWallHandler_WithCore_NilSafe(t *testing.T) {
	var h *WallHandler
	out := h.WithCore(newTestCore(t))
	assert.Nil(t, out)
}