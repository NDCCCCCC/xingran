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
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// Compile-time assertion: mock services satisfy rpa service interfaces.
var (
	_ rpa.FlowControlService    = (*mockFlowControlService)(nil)
	_ rpa.ErrorHandlingService  = (*mockErrorHandlingService)(nil)
	_ rpa.DataMapperService     = (*mockDataMapperService)(nil)
)

// mockFlowControlService implements rpa.FlowControlService via function fields.
type mockFlowControlService struct {
	rpa.FlowControlService

	ExecuteConditionFunc func(ctx context.Context, condition *rpa.ConditionAction, variables map[string]interface{}) (bool, error)
	ExecuteLoopFunc      func(ctx context.Context, loop *rpa.LoopAction, variables map[string]interface{}) ([]map[string]interface{}, error)
	EvaluateConditionFunc func(ctx context.Context, expr string, variables map[string]interface{}) (bool, error)
}

func (m *mockFlowControlService) ExecuteCondition(ctx context.Context, condition *rpa.ConditionAction, variables map[string]interface{}) (bool, error) {
	if m.ExecuteConditionFunc != nil {
		return m.ExecuteConditionFunc(ctx, condition, variables)
	}
	return false, nil
}

func (m *mockFlowControlService) ExecuteLoop(ctx context.Context, loop *rpa.LoopAction, variables map[string]interface{}) ([]map[string]interface{}, error) {
	if m.ExecuteLoopFunc != nil {
		return m.ExecuteLoopFunc(ctx, loop, variables)
	}
	return nil, nil
}

func (m *mockFlowControlService) EvaluateCondition(ctx context.Context, expr string, variables map[string]interface{}) (bool, error) {
	if m.EvaluateConditionFunc != nil {
		return m.EvaluateConditionFunc(ctx, expr, variables)
	}
	return false, nil
}

// mockErrorHandlingService implements rpa.ErrorHandlingService via function fields.
type mockErrorHandlingService struct {
	rpa.ErrorHandlingService

	HandleErrorFunc    func(ctx context.Context, req *rpa.ErrorHandleRequest) (*rpa.ErrorHandleResult, error)
	ExecuteRetryFunc   func(ctx context.Context, req *rpa.RetryRequest) (*rpa.RetryResult, error)
	ExecuteRollbackFunc func(ctx context.Context, req *rpa.RollbackRequest) error
	ExecuteFallbackFunc func(ctx context.Context, req *rpa.FallbackRequest) (*rpa.FallbackResult, error)
}

func (m *mockErrorHandlingService) HandleError(ctx context.Context, req *rpa.ErrorHandleRequest) (*rpa.ErrorHandleResult, error) {
	if m.HandleErrorFunc != nil {
		return m.HandleErrorFunc(ctx, req)
	}
	return &rpa.ErrorHandleResult{}, nil
}

func (m *mockErrorHandlingService) ExecuteRetry(ctx context.Context, req *rpa.RetryRequest) (*rpa.RetryResult, error) {
	if m.ExecuteRetryFunc != nil {
		return m.ExecuteRetryFunc(ctx, req)
	}
	return &rpa.RetryResult{}, nil
}

func (m *mockErrorHandlingService) ExecuteRollback(ctx context.Context, req *rpa.RollbackRequest) error {
	if m.ExecuteRollbackFunc != nil {
		return m.ExecuteRollbackFunc(ctx, req)
	}
	return nil
}

func (m *mockErrorHandlingService) ExecuteFallback(ctx context.Context, req *rpa.FallbackRequest) (*rpa.FallbackResult, error) {
	if m.ExecuteFallbackFunc != nil {
		return m.ExecuteFallbackFunc(ctx, req)
	}
	return &rpa.FallbackResult{}, nil
}

// mockDataMapperService implements rpa.DataMapperService via function fields.
type mockDataMapperService struct {
	rpa.DataMapperService

	MapDataFunc         func(ctx context.Context, config *rpa.DataMappingConfig, source interface{}) (map[string]interface{}, error)
	MapFieldsFunc       func(ctx context.Context, rule *rpa.DataMappingRule, source interface{}) (interface{}, error)
	TransformValueFunc  func(ctx context.Context, value interface{}, transform rpa.TransformFunction, params map[string]interface{}) (interface{}, error)
	ExtractJSONPathFunc func(ctx context.Context, data interface{}, path string) (interface{}, error)
	AggregateDataFunc   func(ctx context.Context, data []interface{}, aggregateType string) (interface{}, error)
}

func (m *mockDataMapperService) MapData(ctx context.Context, config *rpa.DataMappingConfig, source interface{}) (map[string]interface{}, error) {
	if m.MapDataFunc != nil {
		return m.MapDataFunc(ctx, config, source)
	}
	return nil, nil
}

func (m *mockDataMapperService) MapFields(ctx context.Context, rule *rpa.DataMappingRule, source interface{}) (interface{}, error) {
	if m.MapFieldsFunc != nil {
		return m.MapFieldsFunc(ctx, rule, source)
	}
	return nil, nil
}

func (m *mockDataMapperService) TransformValue(ctx context.Context, value interface{}, transform rpa.TransformFunction, params map[string]interface{}) (interface{}, error) {
	if m.TransformValueFunc != nil {
		return m.TransformValueFunc(ctx, value, transform, params)
	}
	return nil, nil
}

func (m *mockDataMapperService) ExtractJSONPath(ctx context.Context, data interface{}, path string) (interface{}, error) {
	if m.ExtractJSONPathFunc != nil {
		return m.ExtractJSONPathFunc(ctx, data, path)
	}
	return nil, nil
}

func (m *mockDataMapperService) AggregateData(ctx context.Context, data []interface{}, aggregateType string) (interface{}, error) {
	if m.AggregateDataFunc != nil {
		return m.AggregateDataFunc(ctx, data, aggregateType)
	}
	return nil, nil
}

// ==================== Test helpers ====================

func newTestCtxFlow(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupFlowHandler(flow *mockFlowControlService, errSvc *mockErrorHandlingService, mapper *mockDataMapperService) *FlowHandler {
	return NewFlowHandler(flow, errSvc, mapper).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestFlowHandler_CompileOnly(t *testing.T) {
	flow := &mockFlowControlService{}
	errSvc := &mockErrorHandlingService{}
	mapper := &mockDataMapperService{}
	h := setupFlowHandler(flow, errSvc, mapper)
	assert.NotNil(t, h)
}

// ==================== EvaluateCondition ====================

func TestFlowHandler_EvaluateCondition_Success(t *testing.T) {
	flow := &mockFlowControlService{
		EvaluateConditionFunc: func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error) {
			return true, nil
		},
	}
	h := setupFlowHandler(flow, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/evaluate-condition", EvaluateConditionRequest{
		Expression: "x > 0",
	})
	h.EvaluateCondition(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_EvaluateCondition_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/evaluate-condition", map[string]interface{}{}) // missing Expression
	h.EvaluateCondition(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_EvaluateCondition_ServiceError(t *testing.T) {
	flow := &mockFlowControlService{
		EvaluateConditionFunc: func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error) {
			return false, errors.New("eval fail")
		},
	}
	h := setupFlowHandler(flow, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/evaluate-condition", EvaluateConditionRequest{
		Expression: "x > 0",
	})
	h.EvaluateCondition(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== MapData ====================

func TestFlowHandler_MapData_Success(t *testing.T) {
	mapper := &mockDataMapperService{
		MapDataFunc: func(ctx context.Context, config *rpa.DataMappingConfig, source interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"out": "value"}, nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/map-data", MapDataRequest{
		Config: rpa.DataMappingConfig{},
		Source: map[string]interface{}{"a": "b"},
	})
	h.MapData(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_MapData_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/map-data", map[string]interface{}{}) // missing required fields
	h.MapData(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_MapData_ServiceError(t *testing.T) {
	mapper := &mockDataMapperService{
		MapDataFunc: func(ctx context.Context, config *rpa.DataMappingConfig, source interface{}) (map[string]interface{}, error) {
			return nil, errors.New("map fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/map-data", MapDataRequest{
		Config: rpa.DataMappingConfig{},
		Source: map[string]interface{}{},
	})
	h.MapData(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== TransformValue ====================

func TestFlowHandler_TransformValue_Success(t *testing.T) {
	mapper := &mockDataMapperService{
		TransformValueFunc: func(ctx context.Context, value interface{}, transform rpa.TransformFunction, params map[string]interface{}) (interface{}, error) {
			return "transformed", nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/transform-value", TransformValueRequest{
		Value:     "input",
		Transform: rpa.TransformFunction("uppercase"),
	})
	h.TransformValue(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_TransformValue_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/transform-value", map[string]interface{}{})
	h.TransformValue(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_TransformValue_ServiceError(t *testing.T) {
	mapper := &mockDataMapperService{
		TransformValueFunc: func(ctx context.Context, value interface{}, transform rpa.TransformFunction, params map[string]interface{}) (interface{}, error) {
			return nil, errors.New("trans fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/transform-value", TransformValueRequest{
		Value:     "x",
		Transform: rpa.TransformFunction("upper"),
	})
	h.TransformValue(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ExtractJSONPath ====================

func TestFlowHandler_ExtractJSONPath_Success(t *testing.T) {
	mapper := &mockDataMapperService{
		ExtractJSONPathFunc: func(ctx context.Context, data interface{}, path string) (interface{}, error) {
			return "leaf", nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/extract-jsonpath", ExtractJSONPathRequest{
		Data: map[string]interface{}{"a": map[string]interface{}{"b": "leaf"}},
		Path: "a.b",
	})
	h.ExtractJSONPath(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_ExtractJSONPath_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/extract-jsonpath", map[string]interface{}{})
	h.ExtractJSONPath(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_ExtractJSONPath_ServiceError(t *testing.T) {
	mapper := &mockDataMapperService{
		ExtractJSONPathFunc: func(ctx context.Context, data interface{}, path string) (interface{}, error) {
			return nil, errors.New("path fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/extract-jsonpath", ExtractJSONPathRequest{
		Data: map[string]interface{}{},
		Path: "x",
	})
	h.ExtractJSONPath(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== HandleError ====================

func TestFlowHandler_HandleError_Success(t *testing.T) {
	errSvc := &mockErrorHandlingService{
		HandleErrorFunc: func(ctx context.Context, req *rpa.ErrorHandleRequest) (*rpa.ErrorHandleResult, error) {
			return &rpa.ErrorHandleResult{Action: "retry"}, nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, errSvc, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/handle-error", HandleErrorRequest{
		ExecutionID: "exec-1",
		StepIndex:   1,
		Error:       "boom",
	})
	h.HandleError(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_HandleError_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/handle-error", map[string]interface{}{})
	h.HandleError(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_HandleError_ServiceError(t *testing.T) {
	errSvc := &mockErrorHandlingService{
		HandleErrorFunc: func(ctx context.Context, req *rpa.ErrorHandleRequest) (*rpa.ErrorHandleResult, error) {
			return nil, errors.New("handle fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, errSvc, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/handle-error", HandleErrorRequest{
		ExecutionID: "exec-1",
		StepIndex:   1,
		Error:       "boom",
	})
	h.HandleError(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ExecuteRetry ====================

func TestFlowHandler_ExecuteRetry_Success(t *testing.T) {
	errSvc := &mockErrorHandlingService{
		ExecuteRetryFunc: func(ctx context.Context, req *rpa.RetryRequest) (*rpa.RetryResult, error) {
			return &rpa.RetryResult{ShouldRetry: true}, nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, errSvc, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/execute-retry", ExecuteRetryRequest{
		ExecutionID: "exec-1",
		StepIndex:   1,
	})
	h.ExecuteRetry(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_ExecuteRetry_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/execute-retry", map[string]interface{}{})
	h.ExecuteRetry(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_ExecuteRetry_ServiceError(t *testing.T) {
	errSvc := &mockErrorHandlingService{
		ExecuteRetryFunc: func(ctx context.Context, req *rpa.RetryRequest) (*rpa.RetryResult, error) {
			return nil, errors.New("retry fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, errSvc, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/execute-retry", ExecuteRetryRequest{
		ExecutionID: "exec-1",
		StepIndex:   1,
	})
	h.ExecuteRetry(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== AggregateData ====================

func TestFlowHandler_AggregateData_Success(t *testing.T) {
	mapper := &mockDataMapperService{
		AggregateDataFunc: func(ctx context.Context, data []interface{}, aggType string) (interface{}, error) {
			return 42, nil
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/aggregate-data", AggregateDataRequest{
		Data:          []interface{}{1, 2, 3},
		AggregateType: "sum",
	})
	h.AggregateData(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowHandler_AggregateData_BindError(t *testing.T) {
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, &mockDataMapperService{})
	c, w := newTestCtxFlow("POST", "/aggregate-data", map[string]interface{}{})
	h.AggregateData(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFlowHandler_AggregateData_ServiceError(t *testing.T) {
	mapper := &mockDataMapperService{
		AggregateDataFunc: func(ctx context.Context, data []interface{}, aggType string) (interface{}, error) {
			return nil, errors.New("agg fail")
		},
	}
	h := setupFlowHandler(&mockFlowControlService{}, &mockErrorHandlingService{}, mapper)
	c, w := newTestCtxFlow("POST", "/aggregate-data", AggregateDataRequest{
		Data:          []interface{}{},
		AggregateType: "sum",
	})
	h.AggregateData(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== rpaError type ====================

func TestRpaError_ErrorString(t *testing.T) {
	e := &rpaError{message: "boom"}
	assert.Equal(t, "boom", e.Error())
}
