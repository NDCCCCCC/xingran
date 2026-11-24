package communication

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 154 / ENGINE-05：communication 包首次拥有真实用例
//   - ProgressReporter 真发 HTTP（httptest 收包断言字段）
//   - WorkerID / 时间戳由 reporter 填充（调用方无需关心）
//   - 选择器契约信封行随 ReportCompletion 进 log（一行 JSON 不损坏）
//   - 上报失败时返回可读错误（不 panic、不静默）
// ============================================================================

type nopLogger struct{}

func (nopLogger) Debug(string, ...logger.Field)      {}
func (nopLogger) Info(string, ...logger.Field)       {}
func (nopLogger) Warn(string, ...logger.Field)       {}
func (nopLogger) Error(string, ...logger.Field)      {}
func (nopLogger) With(...logger.Field) logger.Logger { return nopLogger{} }
func (nopLogger) Sync() error                        { return nil }

// captureServer 收集 /rpa/workers/progress 请求体。
type captureServer struct {
	mu     sync.Mutex
	bodies []map[string]interface{}
	fail   bool
	srv    *httptest.Server
}

func newCaptureServer(t *testing.T) *captureServer {
	t.Helper()
	cs := &captureServer{}
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpa/workers/progress" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]interface{}
		_ = json.Unmarshal(body, &m)
		cs.mu.Lock()
		cs.bodies = append(cs.bodies, m)
		cs.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if cs.fail {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":500,"message":"boom"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":null}`))
	}))
	t.Cleanup(cs.srv.Close)
	return cs
}

func (c *captureServer) lastBody(t *testing.T) map[string]interface{} {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.bodies) == 0 {
		t.Fatal("未收到任何上报请求")
	}
	return c.bodies[len(c.bodies)-1]
}

func newReporter(t *testing.T, cs *captureServer) *ProgressReporter {
	t.Helper()
	client := NewAPIClient(&config.BackendConfig{
		BaseURL: cs.srv.URL,
		Timeout: 5 * time.Second,
	}, nopLogger{})
	return NewProgressReporter(client, "worker-test-1", nopLogger{})
}

// TestProgressReporter_ReportsToBackend 真实 HTTP 上报 + 字段填充。
func TestProgressReporter_ReportsToBackend(t *testing.T) {
	cs := newCaptureServer(t)
	r := newReporter(t, cs)

	err := r.ReportProgress(context.Background(), &types.ProgressReport{
		ExecutionID:     "exec-1",
		ProgressCurrent: 3,
		ProgressTotal:   10,
		Message:         "executing: click",
		Status:          types.StatusRunning,
	})
	if err != nil {
		t.Fatalf("上报失败: %v", err)
	}

	body := cs.lastBody(t)
	if body["executionId"] != "exec-1" {
		t.Fatalf("executionId=%v", body["executionId"])
	}
	if body["workerId"] != "worker-test-1" {
		t.Fatalf("workerId 必须由 reporter 填充，实际 %v", body["workerId"])
	}
	if body["status"] != "running" {
		t.Fatalf("status=%v", body["status"])
	}
	if _, ok := body["timestamp"]; !ok {
		t.Fatal("timestamp 必须由 reporter 填充")
	}
	if body["progressCurrent"].(float64) != 3 {
		t.Fatalf("progressCurrent=%v", body["progressCurrent"])
	}
}

// TestProgressReporter_CompletionCarriesSelectorEnvelope 信封行随收尾上报。
func TestProgressReporter_CompletionCarriesSelectorEnvelope(t *testing.T) {
	cs := newCaptureServer(t)
	r := newReporter(t, cs)

	envelope := `{"selectorEvents":[{"pageUrl":"http://x","elementId":"btn","selector":"#btn","success":true}]}`
	result := &types.ExecutionResult{
		ExecutionID: "exec-2",
		Status:      types.StatusSuccess,
		Step:        2,
		Total:       2,
		Logs:        []string{envelope},
	}
	if err := r.ReportCompletion(context.Background(), result); err != nil {
		t.Fatalf("收尾上报失败: %v", err)
	}

	body := cs.lastBody(t)
	logText, _ := body["log"].(string)
	if logText == "" {
		t.Fatal("收尾上报必须携带 log（含选择器信封）")
	}
	// 契约：信封行必须独占一行且以 "{" 开头（后端按行 HasPrefix("{") 解析）
	lines := strings.Split(logText, "\n")
	hasEnvelope := false
	for _, line := range lines {
		if strings.HasPrefix(line, "{") && strings.Contains(line, "selectorEvents") {
			hasEnvelope = true
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("信封行不是合法 JSON: %q: %v", line, err)
			}
		}
	}
	if !hasEnvelope {
		t.Fatalf("log 中未找到可解析的信封行: %q", logText)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v", body["status"])
	}
}

// TestProgressReporter_ErrorResponse 后端 500 时返回可读错误（不 panic）。
func TestProgressReporter_ErrorResponse(t *testing.T) {
	cs := newCaptureServer(t)
	cs.fail = true
	r := newReporter(t, cs)

	err := r.ReportProgress(context.Background(), &types.ProgressReport{
		ExecutionID: "exec-3",
		Status:      types.StatusRunning,
	})
	if err == nil {
		t.Fatal("后端 500 时必须返回错误")
	}
	if !strings.Contains(err.Error(), "500") && !strings.Contains(strings.ToLower(err.Error()), "fail") {
		t.Fatalf("错误信息应可读（含状态码或失败原因）: %v", err)
	}
}

// TestProgressReporter_ErrorReport 失败上报走同一通道。
func TestProgressReporter_ErrorReport(t *testing.T) {
	cs := newCaptureServer(t)
	r := newReporter(t, cs)

	if err := r.ReportError(context.Background(), "exec-4", 5, 10, "assert failed", "data:image/png;base64,xx"); err != nil {
		t.Fatalf("失败上报失败: %v", err)
	}
	body := cs.lastBody(t)
	if body["status"] != "failed" {
		t.Fatalf("status=%v", body["status"])
	}
	if body["message"] != "assert failed" {
		t.Fatalf("message=%v", body["message"])
	}
	if body["workerId"] != "worker-test-1" {
		t.Fatalf("workerId=%v", body["workerId"])
	}
}
