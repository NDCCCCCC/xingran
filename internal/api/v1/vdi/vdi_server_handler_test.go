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
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
)

// Compile-time assertion: mockVDIServerService implements vdiServices.VDIServerService.
var _ vdiServices.VDIServerService = (*mockVDIServerService)(nil)

// mockVDIServerService implements vdiServices.VDIServerService via function fields.
type mockVDIServerService struct {
	vdiServices.VDIServerService

	CreateServerFunc  func(ctx context.Context, req *vdiServices.CreateVDIServerRequest) (*vdiServices.VDIServerDTO, error)
	GetServerFunc     func(ctx context.Context, id string) (*vdiServices.VDIServerDTO, error)
	ListServersFunc   func(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*vdiServices.VDIServerPageResult, error)
	UpdateServerFunc  func(ctx context.Context, id string, req *vdiServices.UpdateVDIServerRequest) error
	DeleteServerFunc  func(ctx context.Context, id string) error
	TestConnectionFunc func(ctx context.Context, id string) error
}

func (m *mockVDIServerService) CreateServer(ctx context.Context, req *vdiServices.CreateVDIServerRequest) (*vdiServices.VDIServerDTO, error) {
	if m.CreateServerFunc != nil {
		return m.CreateServerFunc(ctx, req)
	}
	return &vdiServices.VDIServerDTO{}, nil
}

func (m *mockVDIServerService) GetServer(ctx context.Context, id string) (*vdiServices.VDIServerDTO, error) {
	if m.GetServerFunc != nil {
		return m.GetServerFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockVDIServerService) ListServers(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*vdiServices.VDIServerPageResult, error) {
	if m.ListServersFunc != nil {
		return m.ListServersFunc(ctx, page, pageSize, orderByColumn, isAsc)
	}
	return &vdiServices.VDIServerPageResult{}, nil
}

func (m *mockVDIServerService) UpdateServer(ctx context.Context, id string, req *vdiServices.UpdateVDIServerRequest) error {
	if m.UpdateServerFunc != nil {
		return m.UpdateServerFunc(ctx, id, req)
	}
	return nil
}

func (m *mockVDIServerService) DeleteServer(ctx context.Context, id string) error {
	if m.DeleteServerFunc != nil {
		return m.DeleteServerFunc(ctx, id)
	}
	return nil
}

func (m *mockVDIServerService) TestConnection(ctx context.Context, id string) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx, id)
	}
	return nil
}

// ==================== Test helpers ====================

func newTestCtxVDIServer(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupVDIServerHandler(mock *mockVDIServerService) *VDIServerHandler {
	return NewVDIServerHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestVDIServerHandler_CompileOnly(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	assert.NotNil(t, h)
}

// ==================== Create ====================

func TestVDIServerHandler_Create_Success(t *testing.T) {
	mock := &mockVDIServerService{
		CreateServerFunc: func(ctx context.Context, req *vdiServices.CreateVDIServerRequest) (*vdiServices.VDIServerDTO, error) {
			return &vdiServices.VDIServerDTO{Name: req.Name}, nil
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", vdiServices.CreateVDIServerRequest{
		Name:     "vdi-1",
		Endpoint: "https://10.0.0.1",
		Username: "admin",
		Password: "pass",
	})
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_Create_BindError(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", "not-json")
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_Create_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		CreateServerFunc: func(ctx context.Context, req *vdiServices.CreateVDIServerRequest) (*vdiServices.VDIServerDTO, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", vdiServices.CreateVDIServerRequest{
		Name: "x", Endpoint: "https://x", Username: "u", Password: "p",
	})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== List ====================

func TestVDIServerHandler_List_Success(t *testing.T) {
	mock := &mockVDIServerService{
		ListServersFunc: func(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*vdiServices.VDIServerPageResult, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 10, pageSize)
			return &vdiServices.VDIServerPageResult{}, nil
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_List_BindError(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_List_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		ListServersFunc: func(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*vdiServices.VDIServerPageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_List_WithFilters(t *testing.T) {
	// Verify orderByColumn / isAsc pass-through
	mock := &mockVDIServerService{
		ListServersFunc: func(ctx context.Context, page, pageSize int, orderByColumn string, isAsc *bool) (*vdiServices.VDIServerPageResult, error) {
			assert.Equal(t, "name", orderByColumn)
			assert.NotNil(t, isAsc)
			assert.True(t, *isAsc)
			return &vdiServices.VDIServerPageResult{}, nil
		},
	}
	h := setupVDIServerHandler(mock)
	isAsc := true
	c, w := newTestCtxVDIServer("POST", "/list", map[string]interface{}{
		"orderByColumn": "name",
		"isAsc":         isAsc,
	})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== GetByID ====================

func TestVDIServerHandler_GetByID_Success(t *testing.T) {
	mock := &mockVDIServerService{
		GetServerFunc: func(ctx context.Context, id string) (*vdiServices.VDIServerDTO, error) {
			return &vdiServices.VDIServerDTO{Name: "vdi-1"}, nil
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_GetByID_EmptyID(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_GetByID_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		GetServerFunc: func(ctx context.Context, id string) (*vdiServices.VDIServerDTO, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Update ====================

func TestVDIServerHandler_Update_Success(t *testing.T) {
	mock := &mockVDIServerService{
		UpdateServerFunc: func(ctx context.Context, id string, req *vdiServices.UpdateVDIServerRequest) error {
			assert.Equal(t, "v-1", id)
			return nil
		},
	}
	h := setupVDIServerHandler(mock)
	name := "updated"
	c, w := newTestCtxVDIServer("POST", "/v-1/update", vdiServices.UpdateVDIServerRequest{
		Name: &name,
	})
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_Update_EmptyID(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_Update_BindError(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1/update", "not-json")
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_Update_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		UpdateServerFunc: func(ctx context.Context, id string, req *vdiServices.UpdateVDIServerRequest) error {
			return errors.New("update fail")
		},
	}
	h := setupVDIServerHandler(mock)
	name := "x"
	c, w := newTestCtxVDIServer("POST", "/v-1/update", vdiServices.UpdateVDIServerRequest{
		Name: &name,
	})
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Delete ====================

func TestVDIServerHandler_Delete_Success(t *testing.T) {
	mock := &mockVDIServerService{
		DeleteServerFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_Delete_EmptyID(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		DeleteServerFunc: func(ctx context.Context, id string) error {
			return errors.New("del fail")
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== TestConnection ====================

func TestVDIServerHandler_TestConnection_Success(t *testing.T) {
	mock := &mockVDIServerService{
		TestConnectionFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1/test", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.TestConnection(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVDIServerHandler_TestConnection_EmptyID(t *testing.T) {
	mock := &mockVDIServerService{}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.TestConnection(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVDIServerHandler_TestConnection_ServiceError(t *testing.T) {
	mock := &mockVDIServerService{
		TestConnectionFunc: func(ctx context.Context, id string) error {
			return errors.New("conn fail")
		},
	}
	h := setupVDIServerHandler(mock)
	c, w := newTestCtxVDIServer("POST", "/v-1/test", nil)
	c.Params = gin.Params{{Key: "id", Value: "v-1"}}
	h.TestConnection(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
