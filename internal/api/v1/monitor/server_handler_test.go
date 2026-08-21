package monitor

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
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// mockServerService implements monitorServices.ServerService via function fields
type mockServerService struct {
	monitorServices.ServerService

	GetServerInfoFunc          func(ctx context.Context, params monitorServices.ServerInfoParams) ([]*models.ServerInfo, int64, error)
	GetCurrentServerMetricsFunc func(ctx context.Context) (*monitorServices.SystemMetricsData, error)
	SaveSystemMetricsFunc      func(ctx context.Context, metrics *models.SystemMetrics) error
	GetSystemMetricsHistoryFunc func(ctx context.Context, params monitorServices.MetricsHistoryParams) ([]*models.SystemMetrics, int64, error)
}

func (m *mockServerService) GetServerInfo(ctx context.Context, params monitorServices.ServerInfoParams) ([]*models.ServerInfo, int64, error) {
	if m.GetServerInfoFunc != nil {
		return m.GetServerInfoFunc(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockServerService) GetCurrentServerMetrics(ctx context.Context) (*monitorServices.SystemMetricsData, error) {
	if m.GetCurrentServerMetricsFunc != nil {
		return m.GetCurrentServerMetricsFunc(ctx)
	}
	return nil, nil
}
func (m *mockServerService) SaveSystemMetrics(ctx context.Context, metrics *models.SystemMetrics) error {
	if m.SaveSystemMetricsFunc != nil {
		return m.SaveSystemMetricsFunc(ctx, metrics)
	}
	return nil
}
func (m *mockServerService) GetSystemMetricsHistory(ctx context.Context, params monitorServices.MetricsHistoryParams) ([]*models.SystemMetrics, int64, error) {
	if m.GetSystemMetricsHistoryFunc != nil {
		return m.GetSystemMetricsHistoryFunc(ctx, params)
	}
	return nil, 0, nil
}

func newTestCtxSV(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func parseRespSV(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func setupServerHandler(mock *mockServerService) *ServerHandler {
	return NewServerHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// TC1: GetServerInfo - success
func TestServer_GetServerInfo_Success(t *testing.T) {
	mock := &mockServerService{
		GetServerInfoFunc: func(ctx context.Context, params monitorServices.ServerInfoParams) ([]*models.ServerInfo, int64, error) {
			return []*models.ServerInfo{{}}, 1, nil
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/list", map[string]interface{}{})
	h.GetServerInfo(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC2: GetServerInfo - error
func TestServer_GetServerInfo_Error(t *testing.T) {
	mock := &mockServerService{
		GetServerInfoFunc: func(ctx context.Context, params monitorServices.ServerInfoParams) ([]*models.ServerInfo, int64, error) {
			return nil, 0, errors.New("info fail")
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/list", map[string]interface{}{})
	h.GetServerInfo(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC3: GetCurrentServerMetrics - success
func TestServer_GetCurrentServerMetrics_Success(t *testing.T) {
	mock := &mockServerService{
		GetCurrentServerMetricsFunc: func(ctx context.Context) (*monitorServices.SystemMetricsData, error) {
			return &monitorServices.SystemMetricsData{
				CPUUsage: 50.0, MemoryUsage: 60.0, DiskUsage: 70.0,
				ProcessNum: 100, TotalMemory: 16000, UsedMemory: 8000,
			}, nil
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/current", nil)
	h.GetCurrentServerMetrics(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseRespSV(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	assert.EqualValues(t, 50.0, data["cpuUsage"])
	assert.EqualValues(t, 100, data["processCount"])
}

// TC4: GetCurrentServerMetrics - error
func TestServer_GetCurrentServerMetrics_Error(t *testing.T) {
	mock := &mockServerService{
		GetCurrentServerMetricsFunc: func(ctx context.Context) (*monitorServices.SystemMetricsData, error) {
			return nil, errors.New("metrics fail")
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/current", nil)
	h.GetCurrentServerMetrics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC5: SaveSystemMetrics - invalid JSON
func TestServer_SaveSystemMetrics_InvalidJSON(t *testing.T) {
	mock := &mockServerService{}
	h := setupServerHandler(mock)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/save", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.SaveSystemMetrics(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC6: SaveSystemMetrics - success
func TestServer_SaveSystemMetrics_Success(t *testing.T) {
	mock := &mockServerService{}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/save", map[string]interface{}{
		"serverId":     "srv-1",
		"cpuUsage":     50.0,
		"memoryUsage":  60.0,
		"diskUsage":    70.0,
		"processCount": 100,
	})
	h.SaveSystemMetrics(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC7: SaveSystemMetrics - error
func TestServer_SaveSystemMetrics_Error(t *testing.T) {
	mock := &mockServerService{
		SaveSystemMetricsFunc: func(ctx context.Context, metrics *models.SystemMetrics) error {
			return errors.New("save fail")
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/save", map[string]interface{}{
		"serverId": "srv-1",
	})
	h.SaveSystemMetrics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC8: GetSystemMetricsHistory - success
func TestServer_GetSystemMetricsHistory_Success(t *testing.T) {
	mock := &mockServerService{
		GetSystemMetricsHistoryFunc: func(ctx context.Context, params monitorServices.MetricsHistoryParams) ([]*models.SystemMetrics, int64, error) {
			return []*models.SystemMetrics{{}}, 1, nil
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/history", map[string]interface{}{})
	h.GetSystemMetricsHistory(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC9: GetSystemMetricsHistory - error
func TestServer_GetSystemMetricsHistory_Error(t *testing.T) {
	mock := &mockServerService{
		GetSystemMetricsHistoryFunc: func(ctx context.Context, params monitorServices.MetricsHistoryParams) ([]*models.SystemMetrics, int64, error) {
			return nil, 0, errors.New("history fail")
		},
	}
	h := setupServerHandler(mock)
	c, w := newTestCtxSV("POST", "/history", map[string]interface{}{})
	h.GetSystemMetricsHistory(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC10: WithCore chain
func TestServer_WithCore(t *testing.T) {
	h := &ServerHandler{}
	result := h.WithCore(&core.Core{CoreInfra: &core.CoreInfra{}})
	assert.Same(t, h, result)
	assert.NotNil(t, result.core)
}
