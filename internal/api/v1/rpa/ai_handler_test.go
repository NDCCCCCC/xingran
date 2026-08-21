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

// Compile-time assertion: mockAIService implements rpa.AIService.
var _ rpa.AIService = (*mockAIService)(nil)

// mockAIService implements rpa.AIService via function fields.
type mockAIService struct {
	rpa.AIService

	GenerateScriptFunc          func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error)
	OptimizeScriptFunc          func(ctx context.Context, req *rpa.AIScriptOptimizeRequest) (*rpa.AIScriptOptimizeResponse, error)
	DecideNextActionFunc        func(ctx context.Context, req *rpa.AIAgentDecisionRequest) (*rpa.AIAgentAction, error)
	AnalyzeFailureFunc          func(ctx context.Context, req *rpa.AnalyzeFailureRequest) (*rpa.FailureAnalysisResult, error)
	SuggestFixFunc              func(ctx context.Context, req *rpa.SuggestFixRequest) (*rpa.FixAction, error)
	ClassifyErrorFunc           func(ctx context.Context, errorMessage string) (*rpa.ErrorClassification, error)
	RecordSelectorSuccessFunc   func(ctx context.Context, record *rpa.SelectorSuccessRecord) error
	RecordSelectorFailureFunc   func(ctx context.Context, record *rpa.SelectorFailureRecord) error
	GetBestSelectorFunc         func(ctx context.Context, pageURL, elementID string) (*rpa.SelectorRecommendation, error)
	ScoreSelectorFunc           func(ctx context.Context, selector, pageURL string) (float64, error)
	GetSelectorAlternativesFunc func(ctx context.Context, selector, pageURL string) ([]string, error)
}

func (m *mockAIService) GenerateScript(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
	if m.GenerateScriptFunc != nil {
		return m.GenerateScriptFunc(ctx, req)
	}
	return &rpa.AIScriptGenerateResponse{}, nil
}

func (m *mockAIService) OptimizeScript(ctx context.Context, req *rpa.AIScriptOptimizeRequest) (*rpa.AIScriptOptimizeResponse, error) {
	if m.OptimizeScriptFunc != nil {
		return m.OptimizeScriptFunc(ctx, req)
	}
	return &rpa.AIScriptOptimizeResponse{}, nil
}

func (m *mockAIService) DecideNextAction(ctx context.Context, req *rpa.AIAgentDecisionRequest) (*rpa.AIAgentAction, error) {
	if m.DecideNextActionFunc != nil {
		return m.DecideNextActionFunc(ctx, req)
	}
	return &rpa.AIAgentAction{}, nil
}

func (m *mockAIService) AnalyzeFailure(ctx context.Context, req *rpa.AnalyzeFailureRequest) (*rpa.FailureAnalysisResult, error) {
	if m.AnalyzeFailureFunc != nil {
		return m.AnalyzeFailureFunc(ctx, req)
	}
	return &rpa.FailureAnalysisResult{}, nil
}

func (m *mockAIService) SuggestFix(ctx context.Context, req *rpa.SuggestFixRequest) (*rpa.FixAction, error) {
	if m.SuggestFixFunc != nil {
		return m.SuggestFixFunc(ctx, req)
	}
	return &rpa.FixAction{}, nil
}

func (m *mockAIService) ClassifyError(ctx context.Context, errorMessage string) (*rpa.ErrorClassification, error) {
	if m.ClassifyErrorFunc != nil {
		return m.ClassifyErrorFunc(ctx, errorMessage)
	}
	return &rpa.ErrorClassification{}, nil
}

func (m *mockAIService) RecordSelectorSuccess(ctx context.Context, record *rpa.SelectorSuccessRecord) error {
	if m.RecordSelectorSuccessFunc != nil {
		return m.RecordSelectorSuccessFunc(ctx, record)
	}
	return nil
}

func (m *mockAIService) RecordSelectorFailure(ctx context.Context, record *rpa.SelectorFailureRecord) error {
	if m.RecordSelectorFailureFunc != nil {
		return m.RecordSelectorFailureFunc(ctx, record)
	}
	return nil
}

func (m *mockAIService) GetBestSelector(ctx context.Context, pageURL, elementID string) (*rpa.SelectorRecommendation, error) {
	if m.GetBestSelectorFunc != nil {
		return m.GetBestSelectorFunc(ctx, pageURL, elementID)
	}
	return &rpa.SelectorRecommendation{}, nil
}

func (m *mockAIService) ScoreSelector(ctx context.Context, selector, pageURL string) (float64, error) {
	if m.ScoreSelectorFunc != nil {
		return m.ScoreSelectorFunc(ctx, selector, pageURL)
	}
	return 0, nil
}

func (m *mockAIService) GetSelectorAlternatives(ctx context.Context, selector, pageURL string) ([]string, error) {
	if m.GetSelectorAlternativesFunc != nil {
		return m.GetSelectorAlternativesFunc(ctx, selector, pageURL)
	}
	return nil, nil
}

// ==================== Test helpers ====================

func newTestCtxAI(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupAIHandler(mock *mockAIService) *AIHandler {
	return NewAIHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestAIHandler_CompileOnly(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	assert.NotNil(t, h)
}

// ==================== GenerateScript ====================

func TestAIHandler_GenerateScript_Success(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return &rpa.AIScriptGenerateResponse{Explanation: "ok", Confidence: 0.9}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{
		Description: "click login button",
	})
	h.GenerateScript(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_GenerateScript_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", map[string]interface{}{}) // missing Description
	h.GenerateScript(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_GenerateScript_ServiceError(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return nil, errors.New("ai fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "x"})
	h.GenerateScript(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== OptimizeScript ====================

func TestAIHandler_OptimizeScript_Success(t *testing.T) {
	mock := &mockAIService{
		OptimizeScriptFunc: func(ctx context.Context, req *rpa.AIScriptOptimizeRequest) (*rpa.AIScriptOptimizeResponse, error) {
			return &rpa.AIScriptOptimizeResponse{Explanation: "improved"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/optimize", rpa.AIScriptOptimizeRequest{
		Script: []interface{}{},
	})
	h.OptimizeScript(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_OptimizeScript_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/optimize", map[string]interface{}{})
	h.OptimizeScript(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_OptimizeScript_ServiceError(t *testing.T) {
	mock := &mockAIService{
		OptimizeScriptFunc: func(ctx context.Context, req *rpa.AIScriptOptimizeRequest) (*rpa.AIScriptOptimizeResponse, error) {
			return nil, errors.New("opt fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/optimize", rpa.AIScriptOptimizeRequest{Script: []interface{}{}})
	h.OptimizeScript(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Decide ====================

func TestAIHandler_Decide_Success(t *testing.T) {
	mock := &mockAIService{
		DecideNextActionFunc: func(ctx context.Context, req *rpa.AIAgentDecisionRequest) (*rpa.AIAgentAction, error) {
			return &rpa.AIAgentAction{Type: "click"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/decide", rpa.AIAgentDecisionRequest{
		TaskDescription: "login",
	})
	h.Decide(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_Decide_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/decide", map[string]interface{}{})
	h.Decide(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_Decide_ServiceError(t *testing.T) {
	mock := &mockAIService{
		DecideNextActionFunc: func(ctx context.Context, req *rpa.AIAgentDecisionRequest) (*rpa.AIAgentAction, error) {
			return nil, errors.New("decide fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/decide", rpa.AIAgentDecisionRequest{TaskDescription: "x"})
	h.Decide(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== AnalyzeFailure ====================

func TestAIHandler_AnalyzeFailure_Success(t *testing.T) {
	mock := &mockAIService{
		AnalyzeFailureFunc: func(ctx context.Context, req *rpa.AnalyzeFailureRequest) (*rpa.FailureAnalysisResult, error) {
			return &rpa.FailureAnalysisResult{}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/analyze-failure", rpa.AnalyzeFailureRequest{
		TaskDescription: "login",
		ErrorMessage:     "timeout",
	})
	h.AnalyzeFailure(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_AnalyzeFailure_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/analyze-failure", map[string]interface{}{})
	h.AnalyzeFailure(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_AnalyzeFailure_ServiceError(t *testing.T) {
	mock := &mockAIService{
		AnalyzeFailureFunc: func(ctx context.Context, req *rpa.AnalyzeFailureRequest) (*rpa.FailureAnalysisResult, error) {
			return nil, errors.New("ana fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/analyze-failure", rpa.AnalyzeFailureRequest{
		TaskDescription: "x",
		ErrorMessage:     "e",
	})
	h.AnalyzeFailure(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== SuggestFix ====================

func TestAIHandler_SuggestFix_Success(t *testing.T) {
	mock := &mockAIService{
		SuggestFixFunc: func(ctx context.Context, req *rpa.SuggestFixRequest) (*rpa.FixAction, error) {
			return &rpa.FixAction{Reason: "retry"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/suggest-fix", rpa.SuggestFixRequest{
		TaskDescription: "login",
		ErrorMessage:     "timeout",
	})
	h.SuggestFix(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_SuggestFix_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/suggest-fix", map[string]interface{}{})
	h.SuggestFix(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_SuggestFix_ServiceError(t *testing.T) {
	mock := &mockAIService{
		SuggestFixFunc: func(ctx context.Context, req *rpa.SuggestFixRequest) (*rpa.FixAction, error) {
			return nil, errors.New("sug fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/suggest-fix", rpa.SuggestFixRequest{
		TaskDescription: "x",
		ErrorMessage:     "e",
	})
	h.SuggestFix(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ClassifyError ====================

func TestAIHandler_ClassifyError_Success(t *testing.T) {
	mock := &mockAIService{
		ClassifyErrorFunc: func(ctx context.Context, msg string) (*rpa.ErrorClassification, error) {
			return &rpa.ErrorClassification{Category: "network"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/classify-error", map[string]string{
		"errorMessage": "connection refused",
	})
	h.ClassifyError(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_ClassifyError_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/classify-error", map[string]string{})
	h.ClassifyError(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_ClassifyError_ServiceError(t *testing.T) {
	mock := &mockAIService{
		ClassifyErrorFunc: func(ctx context.Context, msg string) (*rpa.ErrorClassification, error) {
			return nil, errors.New("class fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/classify-error", map[string]string{"errorMessage": "x"})
	h.ClassifyError(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== RecordSelectorSuccess ====================

func TestAIHandler_RecordSelectorSuccess_Success(t *testing.T) {
	mock := &mockAIService{
		RecordSelectorSuccessFunc: func(ctx context.Context, rec *rpa.SelectorSuccessRecord) error {
			return nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/record-success", rpa.SelectorSuccessRecord{
		Selector: "#btn",
		PageURL:  "http://x",
	})
	h.RecordSelectorSuccess(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_RecordSelectorSuccess_ServiceError(t *testing.T) {
	mock := &mockAIService{
		RecordSelectorSuccessFunc: func(ctx context.Context, rec *rpa.SelectorSuccessRecord) error {
			return errors.New("rec fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/record-success", rpa.SelectorSuccessRecord{
		Selector: "#btn",
		PageURL:  "http://x",
	})
	h.RecordSelectorSuccess(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== RecordSelectorFailure ====================

func TestAIHandler_RecordSelectorFailure_Success(t *testing.T) {
	mock := &mockAIService{
		RecordSelectorFailureFunc: func(ctx context.Context, rec *rpa.SelectorFailureRecord) error {
			return nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/record-failure", rpa.SelectorFailureRecord{
		Selector: "#btn",
		PageURL:  "http://x",
	})
	h.RecordSelectorFailure(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_RecordSelectorFailure_ServiceError(t *testing.T) {
	mock := &mockAIService{
		RecordSelectorFailureFunc: func(ctx context.Context, rec *rpa.SelectorFailureRecord) error {
			return errors.New("rec fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/record-failure", rpa.SelectorFailureRecord{
		Selector: "#btn",
		PageURL:  "http://x",
	})
	h.RecordSelectorFailure(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetBestSelector ====================

func TestAIHandler_GetBestSelector_Success(t *testing.T) {
	mock := &mockAIService{
		GetBestSelectorFunc: func(ctx context.Context, pageURL, elementID string) (*rpa.SelectorRecommendation, error) {
			return &rpa.SelectorRecommendation{Selector: "#btn"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/best", map[string]string{
		"pageUrl":   "http://x",
		"elementId": "login-btn",
	})
	h.GetBestSelector(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_GetBestSelector_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/best", map[string]string{})
	h.GetBestSelector(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_GetBestSelector_ServiceError(t *testing.T) {
	mock := &mockAIService{
		GetBestSelectorFunc: func(ctx context.Context, pageURL, elementID string) (*rpa.SelectorRecommendation, error) {
			return nil, errors.New("get fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/best", map[string]string{
		"pageUrl":   "http://x",
		"elementId": "y",
	})
	h.GetBestSelector(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== ScoreSelector ====================

func TestAIHandler_ScoreSelector_Success(t *testing.T) {
	mock := &mockAIService{
		ScoreSelectorFunc: func(ctx context.Context, selector, pageURL string) (float64, error) {
			return 0.95, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/score", map[string]string{
		"selector": "#btn",
		"pageUrl":  "http://x",
	})
	h.ScoreSelector(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_ScoreSelector_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/score", map[string]string{})
	h.ScoreSelector(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_ScoreSelector_ServiceError(t *testing.T) {
	mock := &mockAIService{
		ScoreSelectorFunc: func(ctx context.Context, selector, pageURL string) (float64, error) {
			return 0, errors.New("score fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/score", map[string]string{
		"selector": "#btn",
		"pageUrl":  "http://x",
	})
	h.ScoreSelector(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetSelectorAlternatives ====================

func TestAIHandler_GetSelectorAlternatives_Success(t *testing.T) {
	mock := &mockAIService{
		GetSelectorAlternativesFunc: func(ctx context.Context, selector, pageURL string) ([]string, error) {
			return []string{"#btn", ".btn", "button"}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/alternatives", map[string]string{
		"selector": "#btn",
		"pageUrl":  "http://x",
	})
	h.GetSelectorAlternatives(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAIHandler_GetSelectorAlternatives_BindError(t *testing.T) {
	mock := &mockAIService{}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/alternatives", map[string]string{})
	h.GetSelectorAlternatives(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIHandler_GetSelectorAlternatives_ServiceError(t *testing.T) {
	mock := &mockAIService{
		GetSelectorAlternativesFunc: func(ctx context.Context, selector, pageURL string) ([]string, error) {
			return nil, errors.New("alt fail")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/selector/alternatives", map[string]string{
		"selector": "#btn",
		"pageUrl":  "http://x",
	})
	h.GetSelectorAlternatives(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
