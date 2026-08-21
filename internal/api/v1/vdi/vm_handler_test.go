package vdi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
)

// Compile-time assertion: mockVMService implements vdiServices.VMService.
var _ vdiServices.VMService = (*mockVMService)(nil)

// mockVMService implements vdiServices.VMService via function fields.
type mockVMService struct {
	vdiServices.VMService

	ListResourceGroupsFunc      func(ctx context.Context, vdiServerID string) ([]vdiServices.VDIResourceGroupDTO, error)
	ListResourcesFunc           func(ctx context.Context, vdiServerID string, groupID string) ([]vdiServices.VDIResourceDTO, error)
	CreateVMFunc                func(ctx context.Context, req *vdiServices.CreateVMServiceRequest) (*vdiServices.VDIVMDTO, error)
	GetVMFunc                   func(ctx context.Context, id string) (*vdiServices.VDIVMDTO, error)
	ListVMsFunc                 func(ctx context.Context, req *vdiServices.ListVMRequest, userID string, dataScope models.DataScope) (*vdiServices.PageResult, error)
	UpdateVMFunc                func(ctx context.Context, id string, req *vdiServices.UpdateVMRequest) error
	DeleteVMFunc                func(ctx context.Context, ids []string) error
	OperateVMFunc               func(ctx context.Context, req *vdiServices.VMOperateRequest) error
	BindUserFunc                func(ctx context.Context, vmID string, req *vdiServices.BindUserServiceRequest) error
	UnbindUserFunc              func(ctx context.Context, vmID string) error
	SyncVMFromVDIFunc           func(ctx context.Context, vmID string) error
	SyncAllVMsFunc              func(ctx context.Context, serverID string) error
	SyncVMsFromVDIByServerFunc  func(ctx context.Context, server *models.VDIServer) error
}

func (m *mockVMService) ListResourceGroups(ctx context.Context, vdiServerID string) ([]vdiServices.VDIResourceGroupDTO, error) {
	if m.ListResourceGroupsFunc != nil {
		return m.ListResourceGroupsFunc(ctx, vdiServerID)
	}
	return nil, nil
}

func (m *mockVMService) ListResources(ctx context.Context, vdiServerID string, groupID string) ([]vdiServices.VDIResourceDTO, error) {
	if m.ListResourcesFunc != nil {
		return m.ListResourcesFunc(ctx, vdiServerID, groupID)
	}
	return nil, nil
}

func (m *mockVMService) CreateVM(ctx context.Context, req *vdiServices.CreateVMServiceRequest) (*vdiServices.VDIVMDTO, error) {
	if m.CreateVMFunc != nil {
		return m.CreateVMFunc(ctx, req)
	}
	return &vdiServices.VDIVMDTO{}, nil
}

func (m *mockVMService) GetVM(ctx context.Context, id string) (*vdiServices.VDIVMDTO, error) {
	if m.GetVMFunc != nil {
		return m.GetVMFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockVMService) ListVMs(ctx context.Context, req *vdiServices.ListVMRequest, userID string, dataScope models.DataScope) (*vdiServices.PageResult, error) {
	if m.ListVMsFunc != nil {
		return m.ListVMsFunc(ctx, req, userID, dataScope)
	}
	return &vdiServices.PageResult{}, nil
}

func (m *mockVMService) UpdateVM(ctx context.Context, id string, req *vdiServices.UpdateVMRequest) error {
	if m.UpdateVMFunc != nil {
		return m.UpdateVMFunc(ctx, id, req)
	}
	return nil
}

func (m *mockVMService) DeleteVM(ctx context.Context, ids []string) error {
	if m.DeleteVMFunc != nil {
		return m.DeleteVMFunc(ctx, ids)
	}
	return nil
}

func (m *mockVMService) OperateVM(ctx context.Context, req *vdiServices.VMOperateRequest) error {
	if m.OperateVMFunc != nil {
		return m.OperateVMFunc(ctx, req)
	}
	return nil
}

func (m *mockVMService) BindUser(ctx context.Context, vmID string, req *vdiServices.BindUserServiceRequest) error {
	if m.BindUserFunc != nil {
		return m.BindUserFunc(ctx, vmID, req)
	}
	return nil
}

func (m *mockVMService) UnbindUser(ctx context.Context, vmID string) error {
	if m.UnbindUserFunc != nil {
		return m.UnbindUserFunc(ctx, vmID)
	}
	return nil
}

func (m *mockVMService) SyncVMFromVDI(ctx context.Context, vmID string) error {
	if m.SyncVMFromVDIFunc != nil {
		return m.SyncVMFromVDIFunc(ctx, vmID)
	}
	return nil
}

func (m *mockVMService) SyncAllVMs(ctx context.Context, serverID string) error {
	if m.SyncAllVMsFunc != nil {
		return m.SyncAllVMsFunc(ctx, serverID)
	}
	return nil
}

func (m *mockVMService) SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error {
	if m.SyncVMsFromVDIByServerFunc != nil {
		return m.SyncVMsFromVDIByServerFunc(ctx, server)
	}
	return nil
}

// ==================== Test helpers ====================

func newTestCtxVM(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func setupVMHandler(mock *mockVMService) *VMHandler {
	return NewVMHandler(mock, nil).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// setupVMHandlerWithDB returns a handler wired to a real in-memory SQLite DB
// (with sys_vdi_servers table created). Required for handlers that call
// ensureVDIServer/verifyVDIServerExists.
func setupVMHandlerWithDB(t *testing.T, mock *mockVMService) *VMHandler {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
		CREATE TABLE IF NOT EXISTS sys_vdi_server (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			deleted_at DATETIME
		)
	`).Error)
	return NewVMHandler(mock, gdb).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestVMHandler_CompileOnly(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	assert.NotNil(t, h)
}

// ==================== Create ====================

func TestVMHandler_Create_Success(t *testing.T) {
	mock := &mockVMService{
		CreateVMFunc: func(ctx context.Context, req *vdiServices.CreateVMServiceRequest) (*vdiServices.VDIVMDTO, error) {
			return &vdiServices.VDIVMDTO{Name: req.Name}, nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", vdiServices.CreateVMServiceRequest{
		Name: "vm-1",
	})
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_Create_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", "not-json")
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_Create_ServiceError(t *testing.T) {
	mock := &mockVMService{
		CreateVMFunc: func(ctx context.Context, req *vdiServices.CreateVMServiceRequest) (*vdiServices.VDIVMDTO, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", vdiServices.CreateVMServiceRequest{Name: "x"})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== List ====================

func TestVMHandler_List_Success(t *testing.T) {
	mock := &mockVMService{
		ListVMsFunc: func(ctx context.Context, req *vdiServices.ListVMRequest, userID string, dataScope models.DataScope) (*vdiServices.PageResult, error) {
			return &vdiServices.PageResult{}, nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_List_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_List_ServiceError(t *testing.T) {
	mock := &mockVMService{
		ListVMsFunc: func(ctx context.Context, req *vdiServices.ListVMRequest, userID string, dataScope models.DataScope) (*vdiServices.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetByID ====================

func TestVMHandler_GetByID_Success(t *testing.T) {
	mock := &mockVMService{
		GetVMFunc: func(ctx context.Context, id string) (*vdiServices.VDIVMDTO, error) {
			return &vdiServices.VDIVMDTO{Name: "vm-1"}, nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_GetByID_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_GetByID_ServiceError(t *testing.T) {
	mock := &mockVMService{
		GetVMFunc: func(ctx context.Context, id string) (*vdiServices.VDIVMDTO, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Update ====================

func TestVMHandler_Update_Success(t *testing.T) {
	mock := &mockVMService{
		UpdateVMFunc: func(ctx context.Context, id string, req *vdiServices.UpdateVMRequest) error {
			assert.Equal(t, "vm-1", id)
			return nil
		},
	}
	h := setupVMHandler(mock)
	name := "x"
	c, w := newTestCtxVM("POST", "/vm-1/update", vdiServices.UpdateVMRequest{Name: &name})
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_Update_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_Update_ServiceError(t *testing.T) {
	mock := &mockVMService{
		UpdateVMFunc: func(ctx context.Context, id string, req *vdiServices.UpdateVMRequest) error {
			return errors.New("update fail")
		},
	}
	h := setupVMHandler(mock)
	name := "x"
	c, w := newTestCtxVM("POST", "/vm-1/update", vdiServices.UpdateVMRequest{Name: &name})
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Delete ====================

func TestVMHandler_Delete_Success(t *testing.T) {
	mock := &mockVMService{
		DeleteVMFunc: func(ctx context.Context, ids []string) error {
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_Delete_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockVMService{
		DeleteVMFunc: func(ctx context.Context, ids []string) error {
			return errors.New("del fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Operate ====================

func TestVMHandler_Operate_Success(t *testing.T) {
	mock := &mockVMService{
		OperateVMFunc: func(ctx context.Context, req *vdiServices.VMOperateRequest) error {
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/operate", vdiServices.VMOperateRequest{
		VMIDs:  []string{"vm-1"},
		Action: "start",
	})
	h.Operate(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_Operate_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/operate", "not-json")
	h.Operate(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_Operate_ServiceError(t *testing.T) {
	mock := &mockVMService{
		OperateVMFunc: func(ctx context.Context, req *vdiServices.VMOperateRequest) error {
			return errors.New("operate fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/operate", vdiServices.VMOperateRequest{VMIDs: []string{"v"}, Action: "stop"})
	h.Operate(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== StartVM ====================

func TestVMHandler_StartVM_Success(t *testing.T) {
	mock := &mockVMService{
		OperateVMFunc: func(ctx context.Context, req *vdiServices.VMOperateRequest) error {
			assert.Equal(t, vdiServices.VMPowerOn, req.Action)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/start", vdiServices.VMOperateRequest{VMIDs: []string{"vm-1"}})
	h.StartVM(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_StartVM_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/start", "not-json")
	h.StartVM(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== StopVM ====================

func TestVMHandler_StopVM_Success(t *testing.T) {
	mock := &mockVMService{
		OperateVMFunc: func(ctx context.Context, req *vdiServices.VMOperateRequest) error {
			assert.Equal(t, vdiServices.VMPowerOff, req.Action)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/stop", vdiServices.VMOperateRequest{VMIDs: []string{"vm-1"}})
	h.StopVM(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_StopVM_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/stop", "not-json")
	h.StopVM(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== RestartVM ====================

func TestVMHandler_RestartVM_Success(t *testing.T) {
	mock := &mockVMService{
		OperateVMFunc: func(ctx context.Context, req *vdiServices.VMOperateRequest) error {
			assert.Equal(t, vdiServices.VMPowerRestart, req.Action)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/restart", vdiServices.VMOperateRequest{VMIDs: []string{"vm-1"}})
	h.RestartVM(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_RestartVM_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/restart", "not-json")
	h.RestartVM(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== BindUser ====================

func TestVMHandler_BindUser_Success(t *testing.T) {
	mock := &mockVMService{
		BindUserFunc: func(ctx context.Context, vmID string, req *vdiServices.BindUserServiceRequest) error {
			assert.Equal(t, "vm-1", vmID)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/bind_user", vdiServices.BindUserServiceRequest{
		Username: "u-1",
	})
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.BindUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_BindUser_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.BindUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_BindUser_BindError(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/bind_user", "not-json")
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.BindUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_BindUser_ServiceError(t *testing.T) {
	mock := &mockVMService{
		BindUserFunc: func(ctx context.Context, vmID string, req *vdiServices.BindUserServiceRequest) error {
			return errors.New("bind fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/bind_user", vdiServices.BindUserServiceRequest{Username: "u"})
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.BindUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== UnbindUser ====================

func TestVMHandler_UnbindUser_Success(t *testing.T) {
	mock := &mockVMService{
		UnbindUserFunc: func(ctx context.Context, vmID string) error {
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/unbind_user", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.UnbindUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_UnbindUser_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.UnbindUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_UnbindUser_ServiceError(t *testing.T) {
	mock := &mockVMService{
		UnbindUserFunc: func(ctx context.Context, vmID string) error {
			return errors.New("unbind fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/unbind_user", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.UnbindUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== SyncFromVDI ====================

func TestVMHandler_SyncFromVDI_Success(t *testing.T) {
	mock := &mockVMService{
		SyncVMFromVDIFunc: func(ctx context.Context, vmID string) error {
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/sync", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.SyncFromVDI(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_SyncFromVDI_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.SyncFromVDI(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_SyncFromVDI_ServiceError(t *testing.T) {
	mock := &mockVMService{
		SyncVMFromVDIFunc: func(ctx context.Context, vmID string) error {
			return errors.New("sync fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/vm-1/sync", nil)
	c.Params = gin.Params{{Key: "id", Value: "vm-1"}}
	h.SyncFromVDI(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ListResourceGroups ====================

func TestVMHandler_ListResourceGroups_Success(t *testing.T) {
	mock := &mockVMService{
		ListResourceGroupsFunc: func(ctx context.Context, vdiServerID string) ([]vdiServices.VDIResourceGroupDTO, error) {
			return []vdiServices.VDIResourceGroupDTO{{Name: "group-1"}}, nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/resource-groups", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListResourceGroups(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_ListResourceGroups_ServiceError(t *testing.T) {
	mock := &mockVMService{
		ListResourceGroupsFunc: func(ctx context.Context, vdiServerID string) ([]vdiServices.VDIResourceGroupDTO, error) {
			return nil, errors.New("group fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/resource-groups", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListResourceGroups(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ListResources ====================

func TestVMHandler_ListResources_Success(t *testing.T) {
	mock := &mockVMService{
		ListResourcesFunc: func(ctx context.Context, vdiServerID string, groupID string) ([]vdiServices.VDIResourceDTO, error) {
			return []vdiServices.VDIResourceDTO{{Name: "res-1"}}, nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/resources", map[string]interface{}{
		"vdi_server_id": "vdi-1",
		"group_id":      "g-1",
	})
	h.ListResources(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_ListResources_MissingGroupID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/resources", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListResources(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_ListResources_ServiceError(t *testing.T) {
	mock := &mockVMService{
		ListResourcesFunc: func(ctx context.Context, vdiServerID string, groupID string) ([]vdiServices.VDIResourceDTO, error) {
			return nil, errors.New("resource fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/resources", map[string]interface{}{
		"group_id": "g-1",
	})
	h.ListResources(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== SyncAll ====================

func TestVMHandler_SyncAll_WithServerID(t *testing.T) {
	mock := &mockVMService{
		SyncAllVMsFunc: func(ctx context.Context, serverID string) error {
			assert.Equal(t, "vdi-1", serverID)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/sync-all", map[string]interface{}{
		"server_id": "vdi-1",
	})
	h.SyncAll(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_SyncAll_AutoServerID(t *testing.T) {
	// When server_id is missing, default to "auto"
	mock := &mockVMService{
		SyncAllVMsFunc: func(ctx context.Context, serverID string) error {
			assert.Equal(t, "auto", serverID)
			return nil
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/sync-all", map[string]interface{}{})
	h.SyncAll(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVMHandler_SyncAll_ServiceError(t *testing.T) {
	mock := &mockVMService{
		SyncAllVMsFunc: func(ctx context.Context, serverID string) error {
			return errors.New("sync fail")
		},
	}
	h := setupVMHandler(mock)
	c, w := newTestCtxVM("POST", "/sync-all", map[string]interface{}{})
	h.SyncAll(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== EnsureVDIServer (base_handler.go) ====================
// These tests use a real in-memory SQLite DB with sys_vdi_server table.
// Note: ensureVDIServer uses response.Error(c, int(httpStatus), msg) which
// (per pkg/response toAppError) defaults int to HTTPStatus=400. So all
// ensureVDIServer failures return 400 — the actual production code has a
// latent bug where the HTTPStatus arg is ignored for int args. We document
// this in the test assertions (D-12: do NOT fix business code).

func TestVMHandler_ListVTPPlatforms_NoServer(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/vtp-platforms", map[string]interface{}{
		"vdi_server_id": "missing",
	})
	// ensureVDIServer fails because no such server → returns response code != 0
	h.ListVTPPlatforms(c)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code, "should return error code when VDI server not found")
}

func TestVMHandler_ListVTPPlatforms_EmptyID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/vtp-platforms", map[string]interface{}{})
	h.ListVTPPlatforms(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_ListRunPositions_NoServer(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/run-positions", map[string]interface{}{
		"vdi_server_id": "missing",
		"vtp_id":        1,
	})
	h.ListRunPositions(c)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestVMHandler_ListRunPositions_NoVTPID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/run-positions", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListRunPositions(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_ListStorages_NoServer(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/storages", map[string]interface{}{
		"vdi_server_id": "missing",
		"vtp_id":        1,
	})
	h.ListStorages(c)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestVMHandler_ListStorages_NoVTPID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/storages", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListStorages(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVMHandler_ListNetworks_NoServer(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/networks", map[string]interface{}{
		"vdi_server_id": "missing",
		"vtp_id":        1,
	})
	h.ListNetworks(c)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestVMHandler_ListNetworks_NoVTPID(t *testing.T) {
	mock := &mockVMService{}
	h := setupVMHandlerWithDB(t, mock)
	c, w := newTestCtxVM("POST", "/networks", map[string]interface{}{
		"vdi_server_id": "vdi-1",
	})
	h.ListNetworks(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
