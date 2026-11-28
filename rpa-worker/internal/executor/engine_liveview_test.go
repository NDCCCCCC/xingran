package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/liveview"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-03 端到端：三模式各一条用例（真 Chromium + 真闸门）
//
// 这是 LIVEVIEW-03 的「三模式各一条端到端用例」验收落点：不 mock 闸门，
// 用真实 liveview.Hub + 真实 Engine.Execute，动作序列在真浏览器上跑。
//
//   TestLiveView03_E2E_ThreeModes/auto     auto：全程不暂停，动作一口气跑完
//   TestLiveView03_E2E_ThreeModes/manual   manual：动作边界阻塞，需人 continue 才推进
//   TestLiveView03_E2E_ThreeModes/copilot  copilot：普通动作直通，带 confirm 标记处阻塞
//
// 三个子用例**共用同一 Engine + Hub + fixture**（串行执行）：每子用例各起一个
// Chrome 会让同机并发压满，导航偶发 net::ERR_ABORTED 造成机器相关 flake。
// 共用后既满足「三模式各一条端到端用例」，也保持用例可重复。
//
// 共同断言：阻塞是真的（用「Execute 未返回」证明），Continue 后从断点继续
// （Step 精确一致，不是重跑）。无 Chromium 或 -short 时 skip。
// ============================================================================

// liveViewFixtureHTML 本用例专用页面：只保留可断言副作用的交互点。
const liveViewFixtureHTML = `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>liveview fixture</title></head>
<body>
 <div id="log">init</div>
 <input id="field" type="text" value="">
 <button id="btn" type="button">点我</button>
 <div id="field-marker"></div>
 <div style="height:1200px"></div>
<script>
 document.getElementById('btn').addEventListener('click', function(){
   document.getElementById('log').textContent = 'clicked';
 });
 document.getElementById('field').addEventListener('input', function(){
   document.getElementById('field-marker').textContent = 'filled:' + this.value;
 });
</script>
</body></html>`

// startLiveViewFixture 起本用例的 fixture 服务，返回 URL。
//
// 注意：不复用 fixture_test.go 的 startFixture —— 它的第二个返回值是**下载目录**
// 而不是 URL，误用会让 TargetURL 变成磁盘路径（端到端用例首版即踩到此坑）。
func startLiveViewFixture(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, liveViewFixtureHTML)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// liveViewEnv 三个子用例共用的运行环境。
type liveViewEnv struct {
	engine *Engine
	hub    *liveview.Hub
	url    string
}

// newLiveViewEnv 起一套真 Engine + 真 Hub（不启 screencast：本用例只验闸门与模式）。
func newLiveViewEnv(t *testing.T) *liveViewEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	url := startLiveViewFixture(t)

	hub := liveview.NewHub(&config.LiveViewConfig{
		Enabled:         true,
		MaxStreams:      2,
		IdleTimeout:     10 * time.Second,
		TakeoverTimeout: 30 * time.Second,
		DefaultMode:     string(types.LiveModeCopilot),
	}, nil, nil, nil)
	t.Cleanup(hub.Close)

	engine, mgr := newIntegrationEngine(t, t.TempDir())
	engine.SetLiveViewHub(hub)
	t.Cleanup(func() { _ = mgr.Close() })

	return &liveViewEnv{engine: engine, hub: hub, url: url}
}

// run 起 goroutine 跑 Execute，返回结果 channel。mode 通过 hub 的 SetMode 设定。
func (e *liveViewEnv) run(t *testing.T, execID string, mode types.LiveMode, actions []types.Action) <-chan *types.ExecutionResult {
	t.Helper()
	ch := make(chan *types.ExecutionResult, 1)
	task := &types.TaskMessage{
		ExecutionID: execID,
		TaskID:      "liveview-p155-task",
		TaskName:    "liveview-e2e",
		TargetURL:   e.url,
		Script:      actions,
		Variables:   map[string]interface{}{},
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, _ := e.engine.Execute(ctx, task, "worker-e2e")
		ch <- res
	}()

	// Execute 内部在 Acquire 后 Attach，此时会话已登记；随后立刻把模式设为目标档位
	// （Attach 用 hub 的默认模式来自 cfg.DefaultMode，这里再用 SetMode 精准覆盖）。
	if !e.waitSession(execID, 30*time.Second) {
		t.Logf("等待会话登记超时（exec=%s）", execID)
		return ch
	}
	if err := e.hub.SetMode(execID, mode, "e2e"); err != nil {
		t.Logf("SetMode(%s)=%v", mode, err)
	}
	return ch
}

// waitSession 等 hub 登记该执行（Execute 内部 Acquire 后 Attach 的时机点）。
func (e *liveViewEnv) waitSession(execID string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if e.hub.Context(execID) != nil {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// continueUntilDone 反复 Continue 直到执行结束（容忍「引擎尚未进入闸门」的时序）。
//
// 真实 UX 里用户也会再点一次「继续」，故允许最多 n 次、每次间隔 d。
func (e *liveViewEnv) continueUntilDone(
	t *testing.T, execID string, done <-chan *types.ExecutionResult, n int, d time.Duration,
) (*types.ExecutionResult, bool) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case res := <-done:
			return res, true
		case <-time.After(d):
			if err := e.hub.Continue(execID, "", "人工确认继续"); err != nil {
				t.Logf("Continue #%d: %v", i+1, err)
			}
		}
	}
	return nil, false
}

// TestLiveView03_E2E_ThreeModes 三模式各一条端到端用例（共用一套真浏览器环境）。
func TestLiveView03_E2E_ThreeModes(t *testing.T) {
	env := newLiveViewEnv(t)

	t.Run("auto", func(t *testing.T) {
		execID := "p155-auto-exec"
		done := env.run(t, execID, types.LiveModeAuto, []types.Action{
			{Type: types.ActionClick, Selector: "#btn", Description: "点一下"},
			{Type: types.ActionFill, Selector: "#field",
				Params: map[string]interface{}{"value": "auto-mode"}, Description: "填一个"},
		})

		select {
		case res := <-done:
			if res == nil {
				t.Fatalf("Execute 未返回结果")
			}
			if string(res.Status) != string(types.StatusSuccess) {
				t.Fatalf("auto 模式应跑完为 success，实际 %s（err=%s）", res.Status, res.ErrorMessage)
			}
			t.Logf("auto 模式 E2E：状态=%s 步骤=%d/%d 耗时=%v", res.Status, res.Step, res.Total, res.Duration)
		case <-time.After(90 * time.Second):
			t.Fatalf("auto 模式 90s 未完成（不应阻塞）")
		}
	})

	t.Run("manual", func(t *testing.T) {
		execID := "p155-manual-exec"
		done := env.run(t, execID, types.LiveModeManual, []types.Action{
			{Type: types.ActionClick, Selector: "#btn", Description: "点一下（manual 需确认）"},
			{Type: types.ActionFill, Selector: "#field",
				Params: map[string]interface{}{"value": "manual-mode"}},
		})

		// 首个动作必须被阻塞：700ms 内 Execute 不应返回。
		select {
		case res := <-done:
			t.Fatalf("manual 模式首个动作未被闸门阻塞，Execute 直接返回 status=%s（err=%s）",
				res.Status, res.ErrorMessage)
		case <-time.After(700 * time.Millisecond):
		}

		res, ok := env.continueUntilDone(t, execID, done, 10, 300*time.Millisecond)
		if !ok {
			t.Fatalf("manual 模式 10 次 continue 后仍未完成")
		}
		if string(res.Status) != string(types.StatusSuccess) {
			t.Fatalf("manual 模式最终应 success，实际 %s（err=%s）", res.Status, res.ErrorMessage)
		}
		if res.Step != 2 {
			t.Fatalf("manual 模式 Step=%d，期望 2（断点续跑不得重跑或跳步）", res.Step)
		}
		t.Logf("manual 模式 E2E：状态=%s 步骤=%d/%d（断点续跑）", res.Status, res.Step, res.Total)
	})

	t.Run("copilot", func(t *testing.T) {
		execID := "p155-copilot-exec"
		done := env.run(t, execID, types.LiveModeCopilot, []types.Action{
			// 普通动作：copilot 不暂停，直通执行。
			{Type: types.ActionClick, Selector: "#btn", Description: "普通动作（不暂停）"},
			// 关键节点（动作级标记 confirm=true）：必须停下来等人确认。
			{Type: types.ActionFill, Selector: "#field",
				Params:      map[string]interface{}{"value": "copilot-mode", "confirm": true},
				Description: "关键节点：等人工确认"},
		})

		// 普通动作直通、关键节点处停住：700ms 内不应返回。
		select {
		case res := <-done:
			t.Fatalf("copilot 的关键节点（confirm=true）未暂停，Execute 直接返回 status=%s（err=%s）",
				res.Status, res.ErrorMessage)
		case <-time.After(700 * time.Millisecond):
		}

		res, ok := env.continueUntilDone(t, execID, done, 10, 300*time.Millisecond)
		if !ok {
			t.Fatalf("copilot 确认后 10 次 Continue 仍未推进")
		}
		if string(res.Status) != string(types.StatusSuccess) {
			t.Fatalf("Continue 后应跑完为 success，实际 %s（err=%s）", res.Status, res.ErrorMessage)
		}
		if res.Step != 2 {
			t.Fatalf("Continue 后 Step=%d，期望 2（断点续跑不得重跑或跳步）", res.Step)
		}
		t.Logf("copilot 模式 E2E：状态=%s 步骤=%d/%d（闸门已放行、断点续跑）", res.Status, res.Step, res.Total)
	})
}
