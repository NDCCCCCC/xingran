package executor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 153 / RUNTIME-02：会话注入真实验收
// httptest fixture 页面自证登录态：cookie / localStorage 由页面 JS 读出并
// 写回 DOM，断言只看页面自己承认的副作用（无 Chromium 或 -short 时 skip）。
// ============================================================================

const sessionFixtureHTML = `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>session fixture</title></head>
<body>
  <div id="auth-state">anonymous</div>
  <div id="cookie-user">-</div>
  <div id="storage-user">-</div>
<script>
  function readCookie(name) {
    const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
    return m ? decodeURIComponent(m[1]) : '';
  }
  const user = readCookie('session_user');
  const token = localStorage.getItem('token') || localStorage.getItem('accessToken') || '';
  document.getElementById('cookie-user').textContent = user || '(empty)';
  document.getElementById('storage-user').textContent = token || '(empty)';
  if (user && token) {
    document.getElementById('auth-state').textContent = 'logged-in';
  }
</script>
</body></html>`

func TestRuntime02_ApplySessionData_LoggedInState(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, sessionFixtureHTML)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	eng, mgr := newIntegrationEngine(t, t.TempDir())
	page, err := mgr.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer mgr.Release(page)

	sessionData := map[string]interface{}{
		"accessToken": "test-token-123",
		"cookies": []map[string]interface{}{
			{"name": "session_user", "value": "alice", "path": "/"},
		},
		"sessionData": map[string]interface{}{
			"tenant": "acme",
		},
	}

	if err := eng.applySessionData(context.Background(), page, sessionData, srv.URL); err != nil {
		t.Fatalf("applySessionData: %v", err)
	}

	// 导航后断言登录态（EvalOnNewDocument 脚本在文档创建时执行）
	pm := browser.NewPageManager(page, testLogger{t})
	if err := pm.Goto(srv.URL); err != nil {
		t.Fatalf("Goto: %v", err)
	}

	state, err := page.Evaluate("document.getElementById('auth-state').textContent")
	if err != nil {
		t.Fatalf("Evaluate auth-state: %v", err)
	}
	if s, _ := state.(string); s != "logged-in" {
		t.Fatalf("期望 logged-in，实际 %q（cookie/localStorage 注入未生效）", s)
	}

	cookieUser, _ := page.Evaluate("document.getElementById('cookie-user').textContent")
	if s, _ := cookieUser.(string); s != "alice" {
		t.Fatalf("cookie 注入未生效: %q", s)
	}

	storageUser, _ := page.Evaluate("document.getElementById('storage-user').textContent")
	if s, _ := storageUser.(string); s != "test-token-123" {
		t.Fatalf("localStorage/token 注入未生效: %q", s)
	}
}

// ============================================================================
// RUNTIME-03 轻量单测（-short 也跑）：ctx 检查点 + cancelled 结果
// ============================================================================

// TestRuntime03_ExecuteAction_ContextCancelledCtx 动作边界的 ctx 检查点：
// ctx 已取消时 executeAction 立即返回 ctx.Err()，不触碰浏览器。
func TestRuntime03_ExecuteAction_ContextCancelled(t *testing.T) {
	cfg := &struct{}{}
	_ = cfg
	eng := &Engine{logger: testLogger{t}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := eng.executeAction(ctx, nil, nil, &types.Action{Type: types.ActionNavigate,
		Params: map[string]interface{}{"url": "http://example.invalid"}}, NewVariableScope(nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx 已取消时 executeAction 应返回 context.Canceled，实际 %v", err)
	}
}

// TestRuntime03_CreateCancelledResult 取消结果状态恒为 cancelled。
func TestRuntime03_CreateCancelledResult(t *testing.T) {
	eng := &Engine{logger: testLogger{t}}
	result := &types.ExecutionResult{Status: types.StatusRunning, StartedAt: time.Now()}
	got := eng.createCancelledResult(result, context.Canceled, 3, 10)
	if got.Status != types.StatusCancelled {
		t.Fatalf("取消结果状态应为 cancelled，实际 %s", got.Status)
	}
	if !strings.Contains(got.ErrorMessage, "cancelled") {
		t.Fatalf("取消结果应保留取消原因: %q", got.ErrorMessage)
	}
}

// TestRuntime04_AbortSentinel abort 哨兵错误 errors.Is 判定。
func TestRuntime04_AbortSentinel(t *testing.T) {
	var target *types.AbortError
	if !errors.Is(types.ErrExecutionAborted, target) {
		t.Fatal("ErrExecutionAborted 应被 errors.Is 判定为 *AbortError")
	}
	wrapped := fmt.Errorf("wrap: %w", types.ErrExecutionAborted)
	if !errors.Is(wrapped, target) {
		t.Fatal("包装后的 abort 错误应可被 errors.Is 识别")
	}
}

// TestRuntime03_Execute_CancelReleasesPage RUNTIME-03 真 Chromium 验收：
// 执行中途取消 ctx → 引擎在动作边界停止、结果 cancelled、page 被 Release
//（后续 Acquire 立即可用，无泄漏）。无 Chromium 或 -short 时 skip。
func TestRuntime03_Execute_CancelReleasesPage(t *testing.T) {
	eng, mgr := newIntegrationEngine(t, t.TempDir())

	srv, _ := startFixture(t)
	task := &types.TaskMessage{
		ExecutionID: "rt03-cancel-1",
		TaskID:      "t1",
		TargetURL:   srv.URL,
		Script: []types.Action{
			{Type: types.ActionWait, Params: map[string]interface{}{"duration": 5000}},
			{Type: types.ActionClick, Selector: "#btn-submit"},
		},
		Variables: map[string]interface{}{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan *types.ExecutionResult, 1)
	go func() {
		result, _ := eng.Execute(ctx, task, "wk-test")
		done <- result
	}()

	time.Sleep(300 * time.Millisecond) // 等 Goto 完成、进入 wait 动作
	cancel()

	select {
	case result := <-done:
		if result.Status != types.StatusCancelled {
			t.Fatalf("取消后结果状态应为 cancelled，实际 %s", result.Status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("取消后 10s 内未返回（ctx 检查点未生效）")
	}

	// page 已释放：再次 Acquire 立即可用（池未泄漏）
	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer acquireCancel()
	page, err := mgr.Acquire(acquireCtx)
	if err != nil {
		t.Fatalf("取消后 page 应已释放且可再次 Acquire: %v", err)
	}
	mgr.Release(page)
}
