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

// Compile-time assertion: mockWorkerService implements rpa.WorkerService.
var _ rpa.WorkerService = (*mockWorkerService)(nil)

// mockWorkerService implements rpa.WorkerService via function fields.
// WorkerService composes WorkerRepository + WorkerRuntime + WorkerProgressReporter.
type mockWorkerService struct {
	rpa.WorkerService

	GetByIDFunc            func(ctx context.Context, id string) (*rpamodels.Worker, error)
	ListFunc               func(ctx context.Context, params *rpa.WorkerListParams) (*rpa.PageResult, error)
	StatisticsFunc         func(ctx context.Context) (*rpa.WorkerStatisticsResult, error)
	RegisterFunc           func(ctx context.Context, req *rpa.WorkerRegisterRequest) (*rpamodels.Worker, error)
	HeartbeatFunc          func(ctx context.Context, req *rpa.WorkerHeartbeatRequest) error
	OfflineFunc            func(ctx context.Context, id string) error
	GetAvailableFunc       func(ctx context.Context) ([]rpamodels.Worker, error)
	CheckOfflineWorkersFunc func(ctx context.Context, timeoutSeconds int64) error
	ProgressFunc           func(ctx context.Context, req *rpa.WorkerProgressRequest) error
}

func (m *mockWorkerService) GetByID(ctx context.Context, id string) (*rpamodels.Worker, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockWorkerService) List(ctx context.Context, params *rpa.WorkerListParams) (*rpa.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return &rpa.PageResult{}, nil
}

func (m *mockWorkerService) Statistics(ctx context.Context) (*rpa.WorkerStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return &rpa.WorkerStatisticsResult{}, nil
}

func (m *mockWorkerService) Register(ctx context.Context, req *rpa.WorkerRegisterRequest) (*rpamodels.Worker, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return &rpamodels.Worker{}, nil
}

func (m *mockWorkerService) Heartbeat(ctx context.Context, req *rpa.WorkerHeartbeatRequest) error {
	if m.HeartbeatFunc != nil {
		return m.HeartbeatFunc(ctx, req)
	}
	return nil
}

func (m *mockWorkerService) Offline(ctx context.Context, id string) error {
	if m.OfflineFunc != nil {
		return m.OfflineFunc(ctx, id)
	}
	return nil
}

func (m *mockWorkerService) GetAvailable(ctx context.Context) ([]rpamodels.Worker, error) {
	if m.GetAvailableFunc != nil {
		return m.GetAvailableFunc(ctx)
	}
	return nil, nil
}

func (m *mockWorkerService) CheckOfflineWorkers(ctx context.Context, timeoutSeconds int64) error {
	if m.CheckOfflineWorkersFunc != nil {
		return m.CheckOfflineWorkersFunc(ctx, timeoutSeconds)
	}
	return nil
}

func (m *mockWorkerService) Progress(ctx context.Context, req *rpa.WorkerProgressRequest) error {
	if m.ProgressFunc != nil {
		return m.ProgressFunc(ctx, req)
	}
	return nil
}

// ==================== Test helpers ====================

// newTestCtxWorker creates a gin.Context with optional JSON body.
func newTestCtxWorker(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

// setupWorkerHandler creates a handler wired to a mock service and minimal core.
// core.OperLogService is nil and core.DB is empty so operlog.Record early-returns.
// redis client is intentionally nil — the handler reaches Redis only via Scale*
// paths which our tests don't exercise.
func setupWorkerHandler(mock *mockWorkerService) *WorkerHandler {
	h := NewWorkerHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	}).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
	return h
}

// ==================== Compile-only smoke ====================

func TestWorkerHandler_CompileOnly(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	assert.NotNil(t, h)
}

// ==================== List ====================

func TestWorkerHandler_List_Success(t *testing.T) {
	mock := &mockWorkerService{
		ListFunc: func(ctx context.Context, params *rpa.WorkerListParams) (*rpa.PageResult, error) {
			assert.Equal(t, 1, params.Current)
			assert.Equal(t, 10, params.PageSize)
			return &rpa.PageResult{List: []rpamodels.Worker{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_List_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_List_ServiceError(t *testing.T) {
	mock := &mockWorkerService{
		ListFunc: func(ctx context.Context, params *rpa.WorkerListParams) (*rpa.PageResult, error) {
			return nil, errors.New("db down")
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Statistics ====================

func TestWorkerHandler_Statistics_Success(t *testing.T) {
	mock := &mockWorkerService{
		StatisticsFunc: func(ctx context.Context) (*rpa.WorkerStatisticsResult, error) {
			return &rpa.WorkerStatisticsResult{Total: 3, Online: 2, Offline: 1}, nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/statistics", nil)
	h.Statistics(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_Statistics_Error(t *testing.T) {
	mock := &mockWorkerService{
		StatisticsFunc: func(ctx context.Context) (*rpa.WorkerStatisticsResult, error) {
			return nil, errors.New("stat fail")
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/statistics", nil)
	h.Statistics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Register ====================

func TestWorkerHandler_Register_Success(t *testing.T) {
	mock := &mockWorkerService{
		RegisterFunc: func(ctx context.Context, req *rpa.WorkerRegisterRequest) (*rpamodels.Worker, error) {
			assert.Equal(t, "w-1", req.WorkerID)
			return &rpamodels.Worker{WorkerID: "w-1", WorkerName: "node-1"}, nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/register", rpa.WorkerRegisterRequest{
		WorkerID: "w-1", Name: "node-1",
	})
	h.Register(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_Register_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/register", map[string]interface{}{})
	h.Register(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_Register_ServiceError(t *testing.T) {
	mock := &mockWorkerService{
		RegisterFunc: func(ctx context.Context, req *rpa.WorkerRegisterRequest) (*rpamodels.Worker, error) {
			return nil, errors.New("dup")
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/register", rpa.WorkerRegisterRequest{
		WorkerID: "w-1", Name: "node-1",
	})
	h.Register(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Heartbeat ====================

func TestWorkerHandler_Heartbeat_Success(t *testing.T) {
	mock := &mockWorkerService{
		HeartbeatFunc: func(ctx context.Context, req *rpa.WorkerHeartbeatRequest) error {
			assert.Equal(t, "w-1", req.WorkerID)
			return nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/workers/w-1/heartbeat", rpa.WorkerHeartbeatRequest{
		WorkerID:     "w-1",
		CurrentTasks: 2,
		Status:       "online",
	})
	c.Params = gin.Params{{Key: "id", Value: "w-1"}}
	h.Heartbeat(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_Heartbeat_EmptyID(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/workers//heartbeat", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Heartbeat(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_Heartbeat_ServiceError(t *testing.T) {
	mock := &mockWorkerService{
		HeartbeatFunc: func(ctx context.Context, req *rpa.WorkerHeartbeatRequest) error {
			return errors.New("hb fail")
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/workers/w-1/heartbeat", nil)
	c.Params = gin.Params{{Key: "id", Value: "w-1"}}
	h.Heartbeat(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_Heartbeat_NoBody(t *testing.T) {
	// When JSON bind fails, handler falls back to constructing request from path param
	mock := &mockWorkerService{
		HeartbeatFunc: func(ctx context.Context, req *rpa.WorkerHeartbeatRequest) error {
			assert.Equal(t, "w-1", req.WorkerID)
			return nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/workers/w-1/heartbeat", "not-json-body")
	c.Params = gin.Params{{Key: "id", Value: "w-1"}}
	h.Heartbeat(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== Progress ====================

func TestWorkerHandler_Progress_Success(t *testing.T) {
	mock := &mockWorkerService{
		ProgressFunc: func(ctx context.Context, req *rpa.WorkerProgressRequest) error {
			assert.Equal(t, "exec-1", req.ExecutionID)
			return nil
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/progress", rpa.WorkerProgressRequest{
		ExecutionID:     "exec-1",
		ProgressCurrent: 5,
		ProgressTotal:   10,
	})
	h.Progress(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_Progress_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/progress", "not-json")
	h.Progress(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_Progress_ServiceError(t *testing.T) {
	mock := &mockWorkerService{
		ProgressFunc: func(ctx context.Context, req *rpa.WorkerProgressRequest) error {
			return errors.New("progress fail")
		},
	}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/progress", rpa.WorkerProgressRequest{
		ExecutionID: "exec-1",
	})
	h.Progress(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ScaleUp ====================

func TestWorkerHandler_ScaleUp_RequiresRedisClient(t *testing.T) {
	// h.publishScaleCommand calls redisClient.Publish — with nil client this
	// will panic. The handler does NOT recover, so we skip the live assertion.
	// Just verify the bind-error path which short-circuits before publish.
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-up", map[string]interface{}{})
	h.ScaleUp(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_ScaleUp_EmptyID(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-up", map[string]interface{}{
		"concurrency": 5,
	})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.ScaleUp(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_ScaleUp_BindError(t *testing.T) {
	// Missing required field `concurrency` returns 400
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-up", "not-json")
	c.Params = gin.Params{{Key: "id", Value: "w-1"}}
	h.ScaleUp(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== ScaleDown ====================

func TestWorkerHandler_ScaleDown_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-down", map[string]interface{}{})
	h.ScaleDown(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_ScaleDown_EmptyID(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-down", map[string]interface{}{
		"concurrency": 5,
	})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.ScaleDown(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== ScaleAll ====================

func TestWorkerHandler_ScaleAll_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-all", map[string]interface{}{})
	h.ScaleAll(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkerHandler_ScaleAll_InvalidDirection(t *testing.T) {
	// Direction is oneof=up|down — invalid value triggers bind error
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/scale-all", map[string]interface{}{
		"direction":   "sideways",
		"concurrency": 5,
	})
	h.ScaleAll(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== AutoScaleConfig ====================

func TestWorkerHandler_GetAutoScaleConfig_Success(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("GET", "/autoscale/config", nil)
	h.GetAutoScaleConfig(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int                    `json:"code"`
		Data    map[string]interface{} `json:"data"`
		Message string                 `json:"message"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, resp.Data, "enabled")
	assert.Contains(t, resp.Data, "scale_up_threshold")
}

func TestWorkerHandler_UpdateAutoScaleConfig_Success(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/autoscale/config", AutoScaleConfig{
		Enabled:            true,
		ScaleUpThreshold:   20,
		ScaleDownThreshold: 5,
		MinConcurrency:     1,
		MaxConcurrency:     50,
		CheckInterval:      60,
	})
	h.UpdateAutoScaleConfig(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkerHandler_UpdateAutoScaleConfig_BindError(t *testing.T) {
	mock := &mockWorkerService{}
	h := setupWorkerHandler(mock)
	c, w := newTestCtxWorker("POST", "/autoscale/config", "not-json")
	h.UpdateAutoScaleConfig(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== handler_helpers coverage ====================

func TestBindAndValidate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := map[string]string{"key": "value"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	type payload struct {
		Key string `json:"key"`
	}
	var p payload
	ok := bindAndValidate(c, &p)
	assert.True(t, ok)
	assert.Equal(t, "value", p.Key)
}

func TestBindAndValidate_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	var p map[string]string
	ok := bindAndValidate(c, &p)
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetIDParam_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	id := getIDParam(c)
	assert.Equal(t, "", id)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetIDParam_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc-123"}}
	id := getIDParam(c)
	assert.Equal(t, "abc-123", id)
}

func TestHandleError_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	assert.False(t, handleError(c, nil, http.StatusInternalServerError, "fail"))
}

func TestHandleError_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	assert.True(t, handleError(c, errors.New("boom"), http.StatusInternalServerError, "fail"))
	// handleError delegates to response.Error which writes the response.
	// We verify the response was written (code != 0 indicates an error code).
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, int(resp["code"].(float64)))
}

func TestSetPaginationDefaults(t *testing.T) {
	current := 0
	pageSize := 0
	setPaginationDefaults(&current, &pageSize)
	assert.Equal(t, 1, current)
	assert.Equal(t, 10, pageSize)

	current = 3
	pageSize = 25
	setPaginationDefaults(&current, &pageSize)
	assert.Equal(t, 3, current)
	assert.Equal(t, 25, pageSize)
}
