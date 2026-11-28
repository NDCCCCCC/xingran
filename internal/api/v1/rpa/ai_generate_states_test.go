package rpa

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// ============================================================================
// Phase 156 / NOCODE-03：/rpa/ai/generate 三态错误映射
//
// 验收口径（任务书）：AI 可用 / 不可用 / 超时 三态各有明确且可操作的用户引导。
//
// 前端要能**区分**三态并给出不同引导，因此后端必须在状态码 + message 上区分：
//
//	可用   → 200 + actions
//	不可用 → 400 + 「请改用录制」   （这是关键体验路径，不能是报错死路）
//	超时   → 500 + 「稍后重试或改录制」
//
// 本文件锁定这三条映射，防止有人把 400 改回 500 导致前端只能笼统提示重试。
// ============================================================================

// decodeAPIResp 解出统一响应体的 code/message。
func decodeAPIResp(t *testing.T, body []byte) (int, string) {
	t.Helper()
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("响应体解析失败: %v，原文: %s", err, string(body))
	}
	return resp.Code, resp.Message
}

// TestGenerateScript_AIDisabled_Returns400WithRecordHint AI 不可用 → 400 + 引导录制。
func TestGenerateScript_AIDisabled_Returns400WithRecordHint(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return nil, rpa.ErrAIDisabled
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "打开报销系统并提交"})
	h.GenerateScript(c)

	// 关键：必须是 400（用户可操作的问题），不是 500（系统故障）
	assert.Equal(t, http.StatusBadRequest, w.Code)

	_, msg := decodeAPIResp(t, w.Body.Bytes())
	// 必须明确指向「录制」这条替代路径 —— 这是 NOCODE-03 的核心体验要求
	assert.Contains(t, msg, "录制", "AI 不可用时必须引导用户改用录制，而不是只报错")
	assert.Contains(t, msg, "不可用")
	// 不得泄漏技术细节
	assert.NotContains(t, msg, "Enabled")
	assert.NotContains(t, msg, "config")
}

// TestGenerateScript_InvalidActions_Returns400WithStepDetail 非法动作 → 400 + 逐条原因。
func TestGenerateScript_InvalidActions_Returns400WithStepDetail(t *testing.T) {
	invalidErr := &rpa.ActionValidationError{
		Total: 2,
		Invalid: []*rpa.InvalidActionError{
			{Index: 2, Type: "teleport", Reason: `动作类型「teleport」系统不支持`},
		},
	}
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return nil, invalidErr
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "x"})
	h.GenerateScript(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, msg := decodeAPIResp(t, w.Body.Bytes())
	// 用户必须能定位到「哪一步」「什么类型」
	assert.Contains(t, msg, "第 2 步")
	assert.Contains(t, msg, "teleport")
	// 且给出下一步动作
	assert.Contains(t, msg, "录制")
}

// TestGenerateScript_Timeout_Returns500Retryable 超时 → 500（系统侧）+ 可重试语义。
func TestGenerateScript_Timeout_Returns500Retryable(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return nil, errors.New("API 请求失败: context deadline exceeded")
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "x"})
	h.GenerateScript(c)

	// 超时是系统侧问题 → 5xx，前端据此提示「稍后重试」而不是「改描述」
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, msg := decodeAPIResp(t, w.Body.Bytes())
	assert.Contains(t, msg, "脚本生成失败")
}

// TestGenerateScript_Success 可用态 → 200 + 动作。
func TestGenerateScript_Success_Returns200(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return &rpa.AIScriptGenerateResponse{
				Actions:     []json.RawMessage{json.RawMessage(`{"type":"click","selector":"#a"}`)},
				Explanation: "打开页面后点击按钮",
				Confidence:  0.92,
			}, nil
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "点击按钮"})
	h.GenerateScript(c)

	assert.Equal(t, http.StatusOK, w.Code)
	code, _ := decodeAPIResp(t, w.Body.Bytes())
	assert.Equal(t, 0, code, "成功响应业务码应为 0")
}

// TestGenerateScript_ThreeStatesAreDistinguishable 三态在「状态码」上可区分。
//
// 这是前端三态引导的契约：如果三态都返回同一个码，前端只能靠字符串猜，
// 引导就会退化成一句笼统的「生成失败」。
func TestGenerateScript_ThreeStatesAreDistinguishable(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"AI 不可用", rpa.ErrAIDisabled, http.StatusBadRequest},
		{"非法动作", &rpa.ActionValidationError{Total: 1, Invalid: []*rpa.InvalidActionError{{Index: 1, Reason: "x"}}}, http.StatusBadRequest},
		{"超时/网络", errors.New("context deadline exceeded"), http.StatusInternalServerError},
	}
	seen := map[int]string{}
	for _, tc := range cases {
		err := tc.err
		mock := &mockAIService{
			GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
				return nil, err
			},
		}
		h := setupAIHandler(mock)
		c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "x"})
		h.GenerateScript(c)
		if w.Code != tc.wantCode {
			t.Errorf("%s: 期望状态码 %d，实际 %d", tc.name, tc.wantCode, w.Code)
		}
		seen[w.Code] = tc.name
	}
	// 「不可用」与「超时」必须落在不同状态码上（前端据此分流引导）
	if seen[http.StatusBadRequest] == seen[http.StatusInternalServerError] {
		t.Fatal("AI 不可用与超时必须能通过状态码区分")
	}
}

// TestGenerateScript_AIDisabledMessageHasNoTechnicalJargon 不可用文案零技术术语。
func TestGenerateScript_AIDisabledMessageHasNoTechnicalJargon(t *testing.T) {
	mock := &mockAIService{
		GenerateScriptFunc: func(ctx context.Context, req *rpa.AIScriptGenerateRequest) (*rpa.AIScriptGenerateResponse, error) {
			return nil, rpa.ErrAIDisabled
		},
	}
	h := setupAIHandler(mock)
	c, w := newTestCtxAI("POST", "/generate", rpa.AIScriptGenerateRequest{Description: "x"})
	h.GenerateScript(c)

	_, msg := decodeAPIResp(t, w.Body.Bytes())
	banned := []string{"enabled", "config", "API Key", "provider", "baseURL", "nil", "err"}
	for _, b := range banned {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(b)) {
			t.Errorf("面向员工的文案泄漏技术术语 %q: %s", b, msg)
		}
	}
}
