package rpa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// Compile-time assertion: mockExecutionService implements rpa.ExecutionService.
var _ rpa.ExecutionService = (*mockExecutionService)(nil)

// mockExecutionService implements rpa.ExecutionService via function fields.
type mockExecutionService struct {
	rpa.ExecutionService

	CreateFunc         func(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error)
	UpdateFunc         func(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateProgressFunc func(ctx context.Context, id string, current, total int, message string) error
	AddLogFunc         func(ctx context.Context, id string, log string) error
	CancelFunc         func(ctx context.Context, id string) error
	ListFunc           func(ctx context.Context, params *rpa.ExecutionListParams) (*rpa.PageResult, error)
	GetByIDFunc        func(ctx context.Context, id string) (*rpamodels.Execution, error)
	PublishProgressFunc func(ctx context.Context, update *rpa.ProgressUpdate) error
	StatisticsFunc     func(ctx context.Context) (*rpa.ExecutionStatisticsResult, error)
}

func (m *mockExecutionService) Create(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, taskID, taskName, triggeredBy)
	}
	return nil, nil
}

func (m *mockExecutionService) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, updates)
	}
	return nil
}

func (m *mockExecutionService) UpdateProgress(ctx context.Context, id string, current, total int, message string) error {
	if m.UpdateProgressFunc != nil {
		return m.UpdateProgressFunc(ctx, id, current, total, message)
	}
	return nil
}

func (m *mockExecutionService) AddLog(ctx context.Context, id string, log string) error {
	if m.AddLogFunc != nil {
		return m.AddLogFunc(ctx, id, log)
	}
	return nil
}

func (m *mockExecutionService) Cancel(ctx context.Context, id string) error {
	if m.CancelFunc != nil {
		return m.CancelFunc(ctx, id)
	}
	return nil
}

func (m *mockExecutionService) List(ctx context.Context, params *rpa.ExecutionListParams) (*rpa.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return &rpa.PageResult{}, nil
}

func (m *mockExecutionService) GetByID(ctx context.Context, id string) (*rpamodels.Execution, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockExecutionService) PublishProgress(ctx context.Context, update *rpa.ProgressUpdate) error {
	if m.PublishProgressFunc != nil {
		return m.PublishProgressFunc(ctx, update)
	}
	return nil
}

func (m *mockExecutionService) Statistics(ctx context.Context) (*rpa.ExecutionStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return &rpa.ExecutionStatisticsResult{}, nil
}

// ==================== Test helpers ====================

// newTestCtxExec creates a gin.Context with optional JSON body.
func newTestCtxExec(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

// setupExecutionHandler wires ExecutionHandler with mock service and minimal core.
func setupExecutionHandler(mock *mockExecutionService) *ExecutionHandler {
	return NewExecutionHandler(mock, nil).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestExecutionHandler_CompileOnly(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	assert.NotNil(t, h)
}

// ==================== List ====================

func TestExecutionHandler_List_Success(t *testing.T) {
	mock := &mockExecutionService{
		ListFunc: func(ctx context.Context, params *rpa.ExecutionListParams) (*rpa.PageResult, error) {
			assert.Equal(t, 1, params.Current)
			assert.Equal(t, 10, params.PageSize)
			return &rpa.PageResult{List: []rpamodels.Execution{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_List_BindError(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_List_ServiceError(t *testing.T) {
	mock := &mockExecutionService{
		ListFunc: func(ctx context.Context, params *rpa.ExecutionListParams) (*rpa.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Statistics ====================

func TestExecutionHandler_Statistics_Success(t *testing.T) {
	mock := &mockExecutionService{
		StatisticsFunc: func(ctx context.Context) (*rpa.ExecutionStatisticsResult, error) {
			return &rpa.ExecutionStatisticsResult{Total: 5, Running: 2, Success: 3}, nil
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/statistics", nil)
	h.Statistics(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_Statistics_Error(t *testing.T) {
	mock := &mockExecutionService{
		StatisticsFunc: func(ctx context.Context) (*rpa.ExecutionStatisticsResult, error) {
			return nil, errors.New("stat fail")
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/statistics", nil)
	h.Statistics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetByID ====================

func TestExecutionHandler_GetByID_Success(t *testing.T) {
	mock := &mockExecutionService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Execution, error) {
			return &rpamodels.Execution{Logs: "log-1"}, nil
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_GetByID_EmptyID(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_GetByID_NotFound(t *testing.T) {
	mock := &mockExecutionService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Execution, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Cancel ====================

func TestExecutionHandler_Cancel_Success(t *testing.T) {
	mock := &mockExecutionService{
		CancelFunc: func(ctx context.Context, id string) error {
			assert.Equal(t, "exec-1", id)
			return nil
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.Cancel(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_Cancel_EmptyID(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Cancel(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_Cancel_ServiceError(t *testing.T) {
	mock := &mockExecutionService{
		CancelFunc: func(ctx context.Context, id string) error {
			return errors.New("cancel fail")
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.Cancel(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetLogs ====================

func TestExecutionHandler_GetLogs_Success(t *testing.T) {
	mock := &mockExecutionService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Execution, error) {
			return &rpamodels.Execution{Logs: "log content"}, nil
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/logs", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.GetLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Logs string `json:"logs"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "log content", resp.Data.Logs)
}

func TestExecutionHandler_GetLogs_EmptyID(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetLogs(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_GetLogs_NotFound(t *testing.T) {
	mock := &mockExecutionService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Execution, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/logs", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.GetLogs(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetBatchReport (excelService nil branch) ====================

func TestExecutionHandler_GetBatchReport_NoService(t *testing.T) {
	// excelService is nil — handler returns 500
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("GET", "/exec-1/batch-report", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.GetBatchReport(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_GetBatchReport_EmptyID(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("GET", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetBatchReport(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== RequestHumanIntervention (excelService nil branch) ====================

func TestExecutionHandler_RequestHumanIntervention_NoService(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("GET", "/exec-1/human-intervention", nil)
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.RequestHumanIntervention(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestExecutionHandler_RequestHumanIntervention_EmptyID(t *testing.T) {
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("GET", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.RequestHumanIntervention(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== SubmitHumanIntervention ====================

func TestExecutionHandler_SubmitHumanIntervention_BindError(t *testing.T) {
	// HumanInterventionRequest is missing required fields
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/human-intervention", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.SubmitHumanIntervention(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecutionHandler_SubmitHumanIntervention_NoService(t *testing.T) {
	// excelService is nil
	mock := &mockExecutionService{}
	h := setupExecutionHandler(mock)
	c, w := newTestCtxExec("POST", "/exec-1/human-intervention", rpa.HumanInterventionRequest{
		ExecutionID: "exec-1",
		Action:      "resume",
		Reason:      "please confirm",
	})
	c.Params = gin.Params{{Key: "id", Value: "exec-1"}}
	h.SubmitHumanIntervention(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== formatTime helper ====================

func TestFormatTime_NilTime(t *testing.T) {
	got := formatTime(nil)
	assert.Equal(t, "-", got)
}

func TestFormatTime_ValidTime(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 30, 0, 0, time.UTC)
	got := formatTime(&now)
	assert.Equal(t, "2026-08-21 10:30:00", got)
}
