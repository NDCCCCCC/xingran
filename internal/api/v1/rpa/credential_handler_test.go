package rpa

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
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// Compile-time assertion: mockCredentialService implements rpa.CredentialService.
var _ rpa.CredentialService = (*mockCredentialService)(nil)

// mockCredentialService implements rpa.CredentialService via function fields.
type mockCredentialService struct {
	rpa.CredentialService

	CreateCredentialFunc        func(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error)
	UpdateCredentialFunc        func(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error
	DeleteCredentialFunc        func(ctx context.Context, id string, userID string) error
	GetCredentialFunc           func(ctx context.Context, id string, userID string) (*rpamodels.RPACredential, error)
	ListCredentialsFunc         func(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error)
	GetCredentialForExecutionFunc func(ctx context.Context, targetSystem string, userID string, deptID string) (*rpamodels.RPACredential, error)
	DecryptCredentialFunc       func(ctx context.Context, cred *rpamodels.RPACredential) (*rpa.CredentialData, error)
	CreateSessionFunc           func(ctx context.Context, req *rpamodels.SessionCreateRequest) (*rpamodels.RPASession, error)
	GetValidSessionFunc         func(ctx context.Context, credentialID string, targetSystem string) (*rpamodels.RPASession, error)
	InvalidateSessionFunc       func(ctx context.Context, sessionID string, reason string) error
	CleanupExpiredSessionsFunc  func(ctx context.Context) error
	RecordLoginSuccessFunc      func(ctx context.Context, credentialID string) error
	RecordLoginFailureFunc      func(ctx context.Context, credentialID string) error
	UpdateLastUsedFunc          func(ctx context.Context, credentialID string) error
}

func (m *mockCredentialService) CreateCredential(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error) {
	if m.CreateCredentialFunc != nil {
		return m.CreateCredentialFunc(ctx, req, userID)
	}
	return &rpamodels.RPACredential{}, nil
}

func (m *mockCredentialService) UpdateCredential(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error {
	if m.UpdateCredentialFunc != nil {
		return m.UpdateCredentialFunc(ctx, id, req, userID)
	}
	return nil
}

func (m *mockCredentialService) DeleteCredential(ctx context.Context, id string, userID string) error {
	if m.DeleteCredentialFunc != nil {
		return m.DeleteCredentialFunc(ctx, id, userID)
	}
	return nil
}

func (m *mockCredentialService) GetCredential(ctx context.Context, id string, userID string) (*rpamodels.RPACredential, error) {
	if m.GetCredentialFunc != nil {
		return m.GetCredentialFunc(ctx, id, userID)
	}
	return nil, nil
}

func (m *mockCredentialService) ListCredentials(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error) {
	if m.ListCredentialsFunc != nil {
		return m.ListCredentialsFunc(ctx, params, userID, deptID)
	}
	return nil, 0, nil
}

func (m *mockCredentialService) GetCredentialForExecution(ctx context.Context, targetSystem string, userID string, deptID string) (*rpamodels.RPACredential, error) {
	if m.GetCredentialForExecutionFunc != nil {
		return m.GetCredentialForExecutionFunc(ctx, targetSystem, userID, deptID)
	}
	return nil, nil
}

func (m *mockCredentialService) DecryptCredential(ctx context.Context, cred *rpamodels.RPACredential) (*rpa.CredentialData, error) {
	if m.DecryptCredentialFunc != nil {
		return m.DecryptCredentialFunc(ctx, cred)
	}
	return nil, nil
}

func (m *mockCredentialService) CreateSession(ctx context.Context, req *rpamodels.SessionCreateRequest) (*rpamodels.RPASession, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockCredentialService) GetValidSession(ctx context.Context, credentialID string, targetSystem string) (*rpamodels.RPASession, error) {
	if m.GetValidSessionFunc != nil {
		return m.GetValidSessionFunc(ctx, credentialID, targetSystem)
	}
	return nil, nil
}

func (m *mockCredentialService) InvalidateSession(ctx context.Context, sessionID string, reason string) error {
	if m.InvalidateSessionFunc != nil {
		return m.InvalidateSessionFunc(ctx, sessionID, reason)
	}
	return nil
}

func (m *mockCredentialService) CleanupExpiredSessions(ctx context.Context) error {
	if m.CleanupExpiredSessionsFunc != nil {
		return m.CleanupExpiredSessionsFunc(ctx)
	}
	return nil
}

func (m *mockCredentialService) RecordLoginSuccess(ctx context.Context, credentialID string) error {
	if m.RecordLoginSuccessFunc != nil {
		return m.RecordLoginSuccessFunc(ctx, credentialID)
	}
	return nil
}

func (m *mockCredentialService) RecordLoginFailure(ctx context.Context, credentialID string) error {
	if m.RecordLoginFailureFunc != nil {
		return m.RecordLoginFailureFunc(ctx, credentialID)
	}
	return nil
}

func (m *mockCredentialService) UpdateLastUsed(ctx context.Context, credentialID string) error {
	if m.UpdateLastUsedFunc != nil {
		return m.UpdateLastUsedFunc(ctx, credentialID)
	}
	return nil
}

// ==================== Test helpers ====================

func newTestCtxCred(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupCredentialHandler(mock *mockCredentialService) *CredentialHandler {
	return NewCredentialHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestCredentialHandler_CompileOnly(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	assert.NotNil(t, h)
}

// ==================== List ====================

func TestCredentialHandler_List_Success(t *testing.T) {
	mock := &mockCredentialService{
		ListCredentialsFunc: func(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error) {
			assert.Equal(t, "user-1", userID)
			return []rpamodels.RPACredential{{}}, 1, nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/list", map[string]interface{}{})
	c.Set("userId", "user-1")
	c.Set("deptId", "dept-1")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_List_BindError(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_List_ServiceError(t *testing.T) {
	mock := &mockCredentialService{
		ListCredentialsFunc: func(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error) {
			return nil, 0, errors.New("list fail")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Create ====================

func TestCredentialHandler_Create_Success(t *testing.T) {
	mock := &mockCredentialService{
		CreateCredentialFunc: func(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error) {
			return &rpamodels.RPACredential{Name: req.Name}, nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", rpamodels.CredentialCreateRequest{
		Name:         "test-cred",
		TargetSystem: "system-1",
		Username:     "u",
		Password:     "p",
	})
	c.Set("userId", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_Create_BindError(t *testing.T) {
	// Missing required field (e.g. Name)
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", map[string]interface{}{})
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_Create_ServiceError(t *testing.T) {
	mock := &mockCredentialService{
		CreateCredentialFunc: func(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", rpamodels.CredentialCreateRequest{
		Name:         "x",
		TargetSystem: "sys",
		Username:     "u",
		Password:     "p",
	})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetByID ====================

func TestCredentialHandler_GetByID_Success(t *testing.T) {
	mock := &mockCredentialService{
		GetCredentialFunc: func(ctx context.Context, id, userID string) (*rpamodels.RPACredential, error) {
			return &rpamodels.RPACredential{Name: "cred-1"}, nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	c.Set("userId", "user-1")
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_GetByID_EmptyID(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_GetByID_NotFound(t *testing.T) {
	mock := &mockCredentialService{
		GetCredentialFunc: func(ctx context.Context, id, userID string) (*rpamodels.RPACredential, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Update ====================

func TestCredentialHandler_Update_Success(t *testing.T) {
	mock := &mockCredentialService{
		UpdateCredentialFunc: func(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error {
			assert.Equal(t, "c-1", id)
			return nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1/update", rpamodels.CredentialUpdateRequest{
		Name: "updated",
	})
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	c.Set("userId", "user-1")
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_Update_EmptyID(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_Update_ServiceError(t *testing.T) {
	mock := &mockCredentialService{
		UpdateCredentialFunc: func(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error {
			return errors.New("update fail")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1/update", rpamodels.CredentialUpdateRequest{
		Name: "x",
	})
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Delete ====================

func TestCredentialHandler_Delete_Success(t *testing.T) {
	mock := &mockCredentialService{
		DeleteCredentialFunc: func(ctx context.Context, id, userID string) error {
			return nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	c.Set("userId", "user-1")
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_Delete_EmptyID(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockCredentialService{
		DeleteCredentialFunc: func(ctx context.Context, id, userID string) error {
			return errors.New("del fail")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/c-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "c-1"}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ListSessions ====================

func TestCredentialHandler_ListSessions_MissingCredID(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/sessions/list", map[string]interface{}{})
	h.ListSessions(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_ListSessions_Success(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/sessions/list?credentialId=c-1", map[string]interface{}{})
	c.Request.URL.RawQuery = "credentialId=c-1"
	h.ListSessions(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== InvalidateSession ====================

func TestCredentialHandler_InvalidateSession_Success(t *testing.T) {
	mock := &mockCredentialService{
		InvalidateSessionFunc: func(ctx context.Context, sessionID string, reason string) error {
			assert.Equal(t, "s-1", sessionID)
			return nil
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/sessions/s-1/invalidate", nil)
	c.Params = gin.Params{{Key: "id", Value: "s-1"}}
	h.InvalidateSession(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCredentialHandler_InvalidateSession_EmptyID(t *testing.T) {
	mock := &mockCredentialService{}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/sessions//invalidate", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.InvalidateSession(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCredentialHandler_InvalidateSession_ServiceError(t *testing.T) {
	mock := &mockCredentialService{
		InvalidateSessionFunc: func(ctx context.Context, sessionID string, reason string) error {
			return errors.New("inv fail")
		},
	}
	h := setupCredentialHandler(mock)
	c, w := newTestCtxCred("POST", "/sessions/s-1/invalidate", nil)
	c.Params = gin.Params{{Key: "id", Value: "s-1"}}
	h.InvalidateSession(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
