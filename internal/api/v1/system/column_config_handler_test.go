package system

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
	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemRequests "github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// =====================================================================
// Phase 74-04: column_config_handler tests
//
// Scope (column_config_handler.go — 5 funcs but only 3 handlers):
//   - NewColumnConfigHandler / WithCore
//   - GetByPageKey
//   - Save
//   - Reset
// =====================================================================

type mockColumnConfigService struct {
	GetByPageKeyFunc    func(ctx context.Context, userID, pageKey string) ([]*models.UserColumnConfig, error)
	SaveFunc            func(ctx context.Context, userID string, req *systemRequests.ColumnConfigSaveRequest) error
	ResetFunc           func(ctx context.Context, userID, pageKey string) error
	GetDefaultConfigFunc func(ctx context.Context, pageKey string) ([]*models.UserColumnConfig, error)
}

func (m *mockColumnConfigService) GetByPageKey(ctx context.Context, userID, pageKey string) ([]*models.UserColumnConfig, error) {
	if m.GetByPageKeyFunc != nil {
		return m.GetByPageKeyFunc(ctx, userID, pageKey)
	}
	return nil, errors.New("GetByPageKeyFunc not set")
}
func (m *mockColumnConfigService) Save(ctx context.Context, userID string, req *systemRequests.ColumnConfigSaveRequest) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, userID, req)
	}
	return errors.New("SaveFunc not set")
}
func (m *mockColumnConfigService) Reset(ctx context.Context, userID, pageKey string) error {
	if m.ResetFunc != nil {
		return m.ResetFunc(ctx, userID, pageKey)
	}
	return errors.New("ResetFunc not set")
}
func (m *mockColumnConfigService) GetDefaultConfig(ctx context.Context, pageKey string) ([]*models.UserColumnConfig, error) {
	if m.GetDefaultConfigFunc != nil {
		return m.GetDefaultConfigFunc(ctx, pageKey)
	}
	return nil, nil
}

func newColumnConfigHandler(svc systemServices.ColumnConfigService) *ColumnConfigHandler {
	h := NewColumnConfigHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: nil},
	}
	return h
}

func invokeColumnConfigHandler(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user-col")
	var buf *bytes.Buffer
	if body != nil {
		switch v := body.(type) {
		case string:
			buf = bytes.NewBufferString(v)
		default:
			b, _ := json.Marshal(body)
			buf = bytes.NewBuffer(b)
		}
	} else {
		buf = bytes.NewBuffer(nil)
	}
	c.Request = httptest.NewRequest(method, path, buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if params != nil {
		c.Params = params
	}
	if handler != nil {
		handler(c)
	}
	return w
}

func TestColumnConfigHandler_New_WithCore(t *testing.T) {
	svc := &mockColumnConfigService{}
	h := NewColumnConfigHandler(svc)
	out := h.WithCore(&core.Core{})
	assert.Same(t, h, out)
	assert.NotNil(t, h.core)
}

func TestColumnConfigHandler_GetByPageKey_Success(t *testing.T) {
	svc := &mockColumnConfigService{
		GetByPageKeyFunc: func(_ context.Context, _, _ string) ([]*models.UserColumnConfig, error) {
			return []*models.UserColumnConfig{{BaseModel: models.BaseModel{ID: "c1"}}}, nil
		},
	}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "GET", "/system/column-configs/page-key", nil,
		gin.Params{{Key: "page_key", Value: "user.list"}}, h.GetByPageKey)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_GetByPageKey_EmptyPageKey(t *testing.T) {
	svc := &mockColumnConfigService{}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "GET", "/system/column-configs/", nil,
		gin.Params{{Key: "page_key", Value: ""}}, h.GetByPageKey)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_GetByPageKey_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/column-configs/page-key", nil)
	c.Params = gin.Params{{Key: "page_key", Value: "user.list"}}

	svc := &mockColumnConfigService{}
	h := newColumnConfigHandler(svc)
	h.GetByPageKey(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_GetByPageKey_ServiceError(t *testing.T) {
	svc := &mockColumnConfigService{
		GetByPageKeyFunc: func(_ context.Context, _, _ string) ([]*models.UserColumnConfig, error) {
			return nil, errors.New("boom")
		},
	}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "GET", "/system/column-configs/page-key", nil,
		gin.Params{{Key: "page_key", Value: "user.list"}}, h.GetByPageKey)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Save_Success(t *testing.T) {
	svc := &mockColumnConfigService{
		SaveFunc: func(_ context.Context, _ string, _ *systemRequests.ColumnConfigSaveRequest) error { return nil },
	}
	h := newColumnConfigHandler(svc)
	body := map[string]interface{}{"pageKey": "user.list", "columnConfigs": []map[string]interface{}{{"columnKey": "name", "visible": true}}}
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs", body, nil, h.Save)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Save_BindError(t *testing.T) {
	svc := &mockColumnConfigService{}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs", "not-json", nil, h.Save)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Save_ServiceError(t *testing.T) {
	svc := &mockColumnConfigService{
		SaveFunc: func(_ context.Context, _ string, _ *systemRequests.ColumnConfigSaveRequest) error { return errors.New("boom") },
	}
	h := newColumnConfigHandler(svc)
	body := map[string]interface{}{"pageKey": "user.list", "columnConfigs": []map[string]interface{}{{"columnKey": "name", "visible": true}}}
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs", body, nil, h.Save)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Reset_Success(t *testing.T) {
	svc := &mockColumnConfigService{
		ResetFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs/user.list/reset", nil,
		gin.Params{{Key: "page_key", Value: "user.list"}}, h.Reset)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Reset_EmptyPageKey(t *testing.T) {
	svc := &mockColumnConfigService{}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs//reset", nil,
		gin.Params{{Key: "page_key", Value: ""}}, h.Reset)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_Reset_ServiceError(t *testing.T) {
	svc := &mockColumnConfigService{
		ResetFunc: func(_ context.Context, _, _ string) error { return errors.New("boom") },
	}
	h := newColumnConfigHandler(svc)
	w := invokeColumnConfigHandler(t, "POST", "/system/column-configs/user.list/reset", nil,
		gin.Params{{Key: "page_key", Value: "user.list"}}, h.Reset)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestColumnConfigHandler_WithCore_NilReceiver(t *testing.T) {
	var h *ColumnConfigHandler
	out := h.WithCore(&core.Core{})
	assert.Nil(t, out)
}
