package network

// DeviceHandler tests (Phase 74-03).
//
// DeviceHandler depends on networkServices.CacheService (an interface), so it uses
// the D-08 mock pattern: mockDeviceCacheService carries one *Func field per method
// the handler actually calls; the remaining interface methods are inert stubs.
// operlog assertions use mockOperLogService from port_write_handler_test.go (D-03).

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
)

// errNetSvc is the default error returned by unset mock funcs so missing stubs are loud.
var errNetSvc = errors.New("not implemented")

// mockDeviceCacheService implements networkServices.CacheService for handler tests.
type mockDeviceCacheService struct {
	ListFunc        func(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error)
	GetByIDFunc     func(ctx context.Context, id string) (*models.NetworkDevice, error)
	CreateFunc      func(ctx context.Context, req *services.CreateDeviceRequest) (*models.NetworkDevice, error)
	QuickCreateFunc func(ctx context.Context, req *services.QuickCreateRequest) (*models.NetworkDevice, error)
	UpdateFunc      func(ctx context.Context, req *services.UpdateDeviceRequest) error
	DeleteFunc      func(ctx context.Context, id string) error
	BatchDeleteFunc func(ctx context.Context, ids []string) error
	StatisticsFunc  func(ctx context.Context) (map[string]interface{}, error)

	// captured inputs for assertion
	lastListReq     *services.ListDeviceRequest
	lastCreateReq   *services.CreateDeviceRequest
	lastUpdateReq   *services.UpdateDeviceRequest
	lastQuickCreate *services.QuickCreateRequest
	lastDeleteID    string
	lastBatchIDs    []string
	lastGetID       string
}

func (m *mockDeviceCacheService) List(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error) {
	m.lastListReq = req
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, 0, errNetSvc
}

func (m *mockDeviceCacheService) GetByID(ctx context.Context, id string) (*models.NetworkDevice, error) {
	m.lastGetID = id
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNetSvc
}

func (m *mockDeviceCacheService) Create(ctx context.Context, req *services.CreateDeviceRequest) (*models.NetworkDevice, error) {
	m.lastCreateReq = req
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockDeviceCacheService) QuickCreateDevice(ctx context.Context, req *services.QuickCreateRequest) (*models.NetworkDevice, error) {
	m.lastQuickCreate = req
	if m.QuickCreateFunc != nil {
		return m.QuickCreateFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockDeviceCacheService) Update(ctx context.Context, req *services.UpdateDeviceRequest) error {
	m.lastUpdateReq = req
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req)
	}
	return errNetSvc
}

func (m *mockDeviceCacheService) Delete(ctx context.Context, id string) error {
	m.lastDeleteID = id
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNetSvc
}

func (m *mockDeviceCacheService) BatchDelete(ctx context.Context, ids []string) error {
	m.lastBatchIDs = ids
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return errNetSvc
}

func (m *mockDeviceCacheService) UpdateStatus(ctx context.Context, id string, status models.DeviceStatus) error {
	return nil
}

func (m *mockDeviceCacheService) UpdateStatusBatch(ctx context.Context, ids []string, status models.DeviceStatus) error {
	return nil
}

func (m *mockDeviceCacheService) GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return nil, errNetSvc
}

func (m *mockDeviceCacheService) GetDevicesByDept(ctx context.Context, deptID string) ([]models.NetworkDevice, error) {
	return nil, nil
}

func (m *mockDeviceCacheService) GetDevicesByCredential(ctx context.Context, credentialID string) ([]models.NetworkDevice, error) {
	return nil, nil
}

func (m *mockDeviceCacheService) InvalidateDeviceCache(ctx context.Context, deviceID string) error { return nil }
func (m *mockDeviceCacheService) InvalidateStatisticsCache(ctx context.Context) error             { return nil }
func (m *mockDeviceCacheService) InvalidateDeptCache(ctx context.Context, deptID string) error     { return nil }
func (m *mockDeviceCacheService) InvalidateCredentialCache(ctx context.Context, credentialID string) error {
	return nil
}
func (m *mockDeviceCacheService) InvalidateAllDeviceCache(ctx context.Context) error { return nil }

// Compile-time interface satisfaction check (guards against silent drift).
var _ networkServices.CacheService = (*mockDeviceCacheService)(nil)

// newDeviceHandler wires the handler with the mock service + operlog-backed core.
func newDeviceHandler(mockSvc *mockDeviceCacheService, env *netTestEnv) *DeviceHandler {
	return NewDeviceHandler(mockSvc).WithCore(env.core)
}

func TestDeviceHandler_Statistics_Success(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{StatisticsFunc: func(ctx context.Context) (map[string]interface{}, error) {
		return map[string]interface{}{"total": int64(5), "online": int64(3)}, nil
	}}
	h := newDeviceHandler(svc, env)

	w := netPost(t, "/statistics", h.Statistics, `{}`)
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, string(resp.Data), `"total":5`)
}

func TestDeviceHandler_Statistics_Error(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{} // StatisticsFunc unset → errNetSvc
	h := newDeviceHandler(svc, env)

	w := netPost(t, "/statistics", h.Statistics, `{}`)
	resp := decodeNetResp(t, w)
	// response.Error(c, err) with plain error → HTTP 500, body code 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 500, resp.Code)
}

func TestDeviceHandler_List_ParsesFiltersAndSorting(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{ListFunc: func(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error) {
		return []models.NetworkDevice{{BaseModel: models.BaseModel{ID: "d1"}, DeviceName: "dev-1"}}, 1, nil
	}}
	h := newDeviceHandler(svc, env)

	body := `{"current":3,"pageSize":25,"orderByColumn":"deviceName","isAsc":false,` +
		`"deviceName":"core","deviceType":"switch","vendor":"huawei","ip":"10.0.0.","status":1,"deptId":"dept-9"}`
	w := netPost(t, "/list", h.List, body)
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, string(resp.Data), `"deviceName":"dev-1"`)

	req := svc.lastListReq
	require.NotNil(t, req)
	assert.Equal(t, 3, req.Current)
	assert.Equal(t, 25, req.PageSize)
	assert.Equal(t, "deviceName", req.OrderByColumn)
	require.NotNil(t, req.IsAsc)
	assert.False(t, *req.IsAsc)
	require.NotNil(t, req.DeviceName)
	assert.Equal(t, "core", *req.DeviceName)
	require.NotNil(t, req.DeviceType)
	assert.Equal(t, models.DeviceType("switch"), *req.DeviceType)
	require.NotNil(t, req.Vendor)
	assert.Equal(t, models.DeviceVendor("huawei"), *req.Vendor)
	require.NotNil(t, req.IP)
	assert.Equal(t, "10.0.0.", *req.IP)
	require.NotNil(t, req.Status)
	assert.Equal(t, models.DeviceStatus(1), *req.Status)
	require.NotNil(t, req.DeptID)
	assert.Equal(t, "dept-9", *req.DeptID)
}

func TestDeviceHandler_List_InvalidBodyFallsBackToDefaults(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{ListFunc: func(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error) {
		return nil, 0, nil
	}}
	h := newDeviceHandler(svc, env)

	// Malformed JSON → ShouldBindJSON fails → rawReq reset to empty map → defaults 1/10,
	// and every optional filter stays nil (the != "" guards skip empty strings).
	w := netPost(t, "/list", h.List, `{not-json`)
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	req := svc.lastListReq
	require.NotNil(t, req)
	assert.Equal(t, 1, req.Current)
	assert.Equal(t, 10, req.PageSize)
	assert.Nil(t, req.DeviceName)
	assert.Nil(t, req.Status)
	assert.Nil(t, req.IsAsc)
}

func TestDeviceHandler_List_ServiceError(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{} // ListFunc unset → error
	h := newDeviceHandler(svc, env)

	w := netPost(t, "/list", h.List, `{}`)
	resp := decodeNetResp(t, w)
	// apperrors.InternalServerError → code 1500, HTTP 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1500, resp.Code)
}

func TestDeviceHandler_GetByID(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{GetByIDFunc: func(ctx context.Context, id string) (*models.NetworkDevice, error) {
		return &models.NetworkDevice{BaseModel: models.BaseModel{ID: id}, DeviceName: "found"}, nil
	}}
	h := newDeviceHandler(svc, env)

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id", h.GetByID}}, http.MethodPost, "/devices/dev-42", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "dev-42", svc.lastGetID)
		assert.Contains(t, string(resp.Data), `"deviceName":"found"`)
	})

	t.Run("service_error_500", func(t *testing.T) {
		errSvc := &mockDeviceCacheService{} // GetByIDFunc unset
		eh := newDeviceHandler(errSvc, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id", eh.GetByID}}, http.MethodPost, "/devices/x", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestDeviceHandler_Create(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{CreateFunc: func(ctx context.Context, req *services.CreateDeviceRequest) (*models.NetworkDevice, error) {
		return &models.NetworkDevice{BaseModel: models.BaseModel{ID: "new-1"}, DeviceName: "created"}, nil
	}}
	h := newDeviceHandler(svc, env)

	t.Run("success_records_operlog_create", func(t *testing.T) {
		w := netPost(t, "/devices", h.Create, `{"deviceName":"created","deviceType":"switch","vendor":"huawei","ipAddress":"10.1.1.1"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"id":"new-1"`)
		// CreatedBy is injected from the auth middleware's user_id
		require.NotNil(t, svc.lastCreateReq)
		assert.Equal(t, "user-0001", svc.lastCreateReq.CreatedBy)
		assert.Equal(t, 1, env.oper.recordAsyncCalls)
		assert.Equal(t, "网络设备", env.oper.lastTitle)
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/devices", h.Create, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error_wrapped", func(t *testing.T) {
		before := env.oper.recordAsyncCalls
		failSvc := &mockDeviceCacheService{}
		fh := newDeviceHandler(failSvc, env)
		w := netPost(t, "/devices", fh.Create, `{"deviceName":"x","deviceType":"switch","vendor":"huawei","ipAddress":"10.1.1.2"}`)
		resp := decodeNetResp(t, w)
		// HandleServiceError → int 500 → HTTP 400, body code 500 (documented project quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "创建设备失败")
		assert.Equal(t, before, env.oper.recordAsyncCalls, "failed create must not record operlog")
	})
}

func TestDeviceHandler_Update(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{UpdateFunc: func(ctx context.Context, req *services.UpdateDeviceRequest) error {
		return nil
	}}
	h := newDeviceHandler(svc, env)

	t.Run("success", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id/update", h.Update}},
			http.MethodPost, "/devices/dev-9/update", `{"deviceName":"renamed"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "更新成功")
		assert.Equal(t, 2, env.oper.lastBusinessType) // OperTypeUpdate
		// req.ID is forced from the path param; UpdatedBy from auth context
		require.NotNil(t, svc.lastUpdateReq)
		assert.Equal(t, "dev-9", svc.lastUpdateReq.ID)
		assert.Equal(t, "user-0001", svc.lastUpdateReq.UpdatedBy)
	})

	t.Run("service_error", func(t *testing.T) {
		failSvc := &mockDeviceCacheService{}
		fh := newDeviceHandler(failSvc, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id/update", fh.Update}},
			http.MethodPost, "/devices/dev-9/update", `{"deviceName":"renamed"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})

	t.Run("malformed_json", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id/update", h.Update}},
			http.MethodPost, "/devices/dev-9/update", `{oops`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestDeviceHandler_Delete(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{DeleteFunc: func(ctx context.Context, id string) error { return nil }}
	h := newDeviceHandler(svc, env)

	t.Run("success_operType_delete", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id/delete", h.Delete}},
			http.MethodPost, "/devices/dev-3/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "dev-3", svc.lastDeleteID)
		assert.Equal(t, 3, env.oper.lastBusinessType) // OperTypeDelete
	})

	t.Run("service_error", func(t *testing.T) {
		failSvc := &mockDeviceCacheService{}
		fh := newDeviceHandler(failSvc, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/devices/:id/delete", fh.Delete}},
			http.MethodPost, "/devices/dev-3/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestDeviceHandler_BatchDelete(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{BatchDeleteFunc: func(ctx context.Context, ids []string) error { return nil }}
	h := newDeviceHandler(svc, env)

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/devices/batch-delete", h.BatchDelete, `{"ids":["a","b","c"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, []string{"a", "b", "c"}, svc.lastBatchIDs)
		assert.Contains(t, string(resp.Data), `"count":3`)
		assert.Equal(t, 16, env.oper.lastBusinessType) // OperTypeBatch
	})

	t.Run("empty_ids_rejected_by_binding", func(t *testing.T) {
		w := netPost(t, "/devices/batch-delete", h.BatchDelete, `{"ids":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		failSvc := &mockDeviceCacheService{}
		fh := newDeviceHandler(failSvc, env)
		w := netPost(t, "/devices/batch-delete", fh.BatchDelete, `{"ids":["a"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestDeviceHandler_QuickCreate(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockDeviceCacheService{QuickCreateFunc: func(ctx context.Context, req *services.QuickCreateRequest) (*models.NetworkDevice, error) {
		return &models.NetworkDevice{BaseModel: models.BaseModel{ID: "qc-1"}}, nil
	}}
	h := newDeviceHandler(svc, env)

	t.Run("success_returns_id_only", func(t *testing.T) {
		w := netPost(t, "/devices/quick-create", h.QuickCreate,
			`{"ipAddress":"10.2.3.4","credentialId":"11111111-2222-3333-4444-555555555555"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"id":"qc-1"}`, string(resp.Data))
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
		require.NotNil(t, svc.lastQuickCreate)
		assert.Equal(t, "user-0001", svc.lastQuickCreate.CreatedBy)
	})

	t.Run("malformed_json", func(t *testing.T) {
		w := netPost(t, "/devices/quick-create", h.QuickCreate, `nope`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("binding_requires_ip_and_credential", func(t *testing.T) {
		// ipAddress must be a valid IP, credentialId a valid uuid (binding tags).
		w := netPost(t, "/devices/quick-create", h.QuickCreate, `{"ipAddress":"not-an-ip","credentialId":"nope"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		failSvc := &mockDeviceCacheService{}
		fh := newDeviceHandler(failSvc, env)
		w := netPost(t, "/devices/quick-create", fh.QuickCreate,
			`{"ipAddress":"10.2.3.5","credentialId":"11111111-2222-3333-4444-555555555555"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "快速创建设备失败")
	})
}

func TestDeviceHandler_WithCore_NilReceiver(t *testing.T) {
	// WithCore guards against a nil receiver — calling it on nil must not panic
	// and must return the nil handler unchanged.
	var h *DeviceHandler
	assert.NotPanics(t, func() { out := h.WithCore(nil); assert.Nil(t, out) })
}
