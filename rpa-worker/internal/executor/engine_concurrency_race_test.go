package executor

// W7.2 / 任务 B-1：Engine 并发共享字段竞态的**复现**用例（先复现，后修）。
//
// 缺陷（rpa-worker/internal/executor/engine.go）：
//
//	Engine 结构体持有 4 个共享可变字段：
//	  currentExecutionID / currentWorkerID / currentStep / currentTotal
//	Worker 只 NewEngine 一次（worker.go:186），max_concurrency 默认 5~8 →
//	最多 N 个 goroutine 并发跑**同一个** *Engine 实例：
//	  e.currentExecutionID = task.ExecutionID   // :85  多个 goroutine 同时写
//	  e.currentStep = i + 1                     // :177
//	  e.currentTotal = len(task.Script)         // :178
//	  ExecutionID: e.currentExecutionID         // :420（截图进度上报）
//	  ExecutionID: e.currentExecutionID         // :645（pause 进度上报）
//	  go e.subscribeForHumanInput(ctx, e.currentExecutionID, e.currentWorkerID, ...) // :659
//
// 危害：进度上报写到**别的**执行记录上；人工干预订阅挂到别人的 execution_id。
//
// 本用例是黑盒复现（不 mock 内部）：真 Chromium + 真 fixture 页面 +
// 真 HTTP 上报通道（httptest 承接 /rpa/workers/progress），
// 同一 *Engine 实例并发跑 2 个不同 execution_id 的执行，
// 断言每个执行的截图进度上报里 ExecutionID 必须是自己那个。
//
// 复现原理（时序可控）：
//
//	exec-A：动作1 = wait(5s)，动作2 = pause(message=act-A-pause)
//	exec-B：在 exec-A 的 wait 期间启动（t≈0.8s），动作1 = wait(5s)（不带 pause）
//	→ t≈0.8s：B 覆写 e.currentExecutionID = "exec-B"
//	→ t≈5.0s：A 进入 pause 分支，读取 e.currentExecutionID → 拿到 "exec-B"
//	→ A 的人工干预进度上报里 executionId = "exec-B"（串号！）
//
// 为什么用 pause 分支而不是 screenshot 分支：两者都读共享 currentExecutionID，
// 但 rod 的并发 fullPage 截图在同一个 Chrome 下不稳定（实测
// "Unable to capture screenshot" 与 "context deadline exceeded"）——
// 那是浏览器层的并发噪声，与本用例要判定的「上报归属」无关。
// pause 分支的截图是 best-effort（失败不中断动作），判定更稳。
//
// 断言口径：动作描述/暂停消息里带归属标记（act-A- / act-B-），于是**每一条**
// 进度上报的 message 都能反查它属于哪个执行 —— 断言覆盖全部上报，
// 而不只是某一条。
//
// 修复后（共享字段改为每次 Execute 的局部变量）：无论时序如何，
// 每个执行的上报都带自己的 execution_id → 用例恒绿。
//
// 注：rpa-worker module 不引入 testify（零新增依赖约束），断言用标准库。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/communication"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// progressCapture 承接 worker 进度上报的本地服务（真 HTTP 通道）。
type progressCapture struct {
	mu    sync.Mutex
	items []types.ProgressReport
	srv   *httptest.Server
}

// reports 返回已捕获的上报快照。
func (c *progressCapture) reports() []types.ProgressReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.ProgressReport, len(c.items))
	copy(out, c.items)
	return out
}

// screenshotReports 返回截图类上报（message 以「截图: 」开头）。
func (c *progressCapture) screenshotReports() []types.ProgressReport {
	var out []types.ProgressReport
	for _, r := range c.reports() {
		if strings.HasPrefix(r.Message, "截图: ") {
			out = append(out, r)
		}
	}
	return out
}

// pausedReports 返回人工干预（pause）类上报。
func (c *progressCapture) pausedReports() []types.ProgressReport {
	var out []types.ProgressReport
	for _, r := range c.reports() {
		if r.Status == types.StatusPaused {
			out = append(out, r)
		}
	}
	return out
}

// newProgressCapture 起一个承接进度上报的本地服务，并返回对应的 reporter。
func newProgressCapture(t *testing.T) (*progressCapture, *communication.ProgressReporter) {
	t.Helper()
	cap := &progressCapture{}
	mux := http.NewServeMux()
	mux.HandleFunc("/rpa/workers/progress", func(w http.ResponseWriter, r *http.Request) {
		var req types.ProgressReport
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			cap.mu.Lock()
			cap.items = append(cap.items, req)
			cap.mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{}}`))
	})
	cap.srv = httptest.NewServer(mux)
	t.Cleanup(cap.srv.Close)

	api := communication.NewAPIClient(&config.BackendConfig{
		BaseURL: cap.srv.URL,
		Timeout: 10 * time.Second,
	}, testLogger{t})
	return cap, communication.NewProgressReporter(api, "worker-race-test", testLogger{t})
}

// newRaceEngine 起真 Chromium 的 Engine（带进度上报器）。
func newRaceEngine(t *testing.T, reporter *communication.ProgressReporter) *Engine {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 集成用例（-short）")
	}
	chrome := browser.TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}

	cfg := &config.Config{
		Browser: config.BrowserConfig{
			// Timeout 压到 10s：pause 分支的 best-effort 截图最多拖这么久，
			// 不让浏览器噪声把用例拖成长跑。
			Headless: true, Timeout: 10 * time.Second, NavigationTimeout: 15 * time.Second,
			MaxConcurrency: 2, ViewportWidth: 1280, ViewportHeight: 800,
			ChromePath: chrome,
		},
		Worker: config.WorkerConfig{
			MaxConcurrency: 2,
			ActionTimeout: 10 * time.Second,
			PauseTimeout:  1 * time.Second,
		},
		Executor: config.ExecutorConfig{
			InitialDelay: 100 * time.Millisecond, UndefinedVariablePolicy: config.UndefinedVariableEmpty,
		},
	}
	cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency

	mgr, err := browser.NewManager(&cfg.Browser, testLogger{t})
	if err != nil {
		t.Fatalf("创建浏览器 Manager 失败: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	return NewEngine(&cfg.Executor, &cfg.Worker, mgr, reporter, testLogger{t}, nil)
}

// raceTask 造一个「先 wait 再 pause」的执行任务。
//
// wait 用于把两个执行的时间窗错开（让后启动者有机会覆写共享字段）；
// pause 分支同样读共享 currentExecutionID（executePause 里的进度上报 +
// 人工干预订阅），且它的截图是 best-effort（失败不中断动作）。
//
// 为什么用 pause 而不是 screenshot 分支：rod 的并发 fullPage 截图在同一个
// Chrome 下不稳定（实测 "Unable to capture screenshot" 与
// "context deadline exceeded"），那是浏览器层的并发噪声，与本用例要判定的
// 「上报归属」无关；pause 分支同样暴露共享字段读取，判定更稳。
//
// 动作描述/暂停消息里带 "act-A-"/"act-B-" 归属标记 —— 于是**每一条**进度上报
// 的 message 都能反查它属于哪个执行，断言可以覆盖全部上报而不是只看某一条。
func raceTask(executionID, marker string, waitMS int, targetURL string, withPause bool) *types.TaskMessage {
	script := []types.Action{
		{Type: types.ActionWait, Description: marker + "-1",
			Params: map[string]interface{}{"duration": waitMS}},
	}
	if withPause {
		script = append(script, types.Action{
			Type: types.ActionPause, Description: marker + "-2",
			Params: map[string]interface{}{
				"message": marker + "-pause",
				"timeout": 300,
			},
		})
	}
	return &types.TaskMessage{
		ExecutionID: executionID,
		TaskID:      "task-" + executionID,
		TaskName:    "并发竞态复现任务 " + executionID,
		TargetURL:   targetURL,
		Script:      script,
	}
}

// ownerOf 从上报 message 反查它属于哪个执行（按归属标记）。
func ownerOf(message, markerA, markerB, execA, execB string) string {
	switch {
	case strings.Contains(message, markerA):
		return execA
	case strings.Contains(message, markerB):
		return execB
	default:
		return ""
	}
}

// TestEngine_ConcurrentExecute_ProgressExecutionIDRace 复现：并发执行时进度上报串号。
//
// 修复前：A 的截图上报（message 含 act-A-）会挂到 exec-B 的 execution_id 上
//         → 本用例失败（复现成功）。
// 修复后：每条上报的 executionId 都与它的归属一致 → 恒绿。
func TestEngine_ConcurrentExecute_ProgressExecutionIDRace(t *testing.T) {
	fixtureSrv, _ := startFixture(t)
	cap, reporter := newProgressCapture(t)
	eng := newRaceEngine(t, reporter)

	const (
		waitMS     = 5000
		delayForB  = 800
		execA      = "exec-A-0001"
		execB      = "exec-B-0002"
		markerA    = "act-A-"
		markerB    = "act-B-"
		workerName = "worker-race-test"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// A：带截图（截图分支读共享字段）
	go func() {
		defer wg.Done()
		_, _ = eng.Execute(ctx, raceTask(execA, markerA, waitMS, fixtureSrv.URL, true), workerName)
	}()
	// B：在 A 的 wait 窗口内启动 → 覆写共享 currentExecutionID（不截图，
	// 避免 rod 并发截图的已知噪声干扰本用例的判定）
	time.Sleep(delayForB * time.Millisecond)
	go func() {
		defer wg.Done()
		_, _ = eng.Execute(ctx, raceTask(execB, markerB, waitMS, fixtureSrv.URL, false), workerName)
	}()
	wg.Wait()

	all := cap.reports()
	if len(all) == 0 {
		t.Fatal("应捕获到进度上报")
	}

	// 核心断言：每一条进度上报的 executionId 必须与它的归属一致。
	checked := 0
	for _, r := range all {
		owner := ownerOf(r.Message, markerA, markerB, execA, execB)
		if owner == "" {
			continue // 非归属类上报（如收尾完成上报）
		}
		checked++
		if r.ExecutionID != owner {
			t.Errorf("进度上报串号了：message=%q 属于 %s，却挂到 execution_id=%s 上",
				r.Message, owner, r.ExecutionID)
		}
	}
	if checked < 2 {
		t.Fatalf("归属可判定的上报过少（%d 条），用例失去意义: %+v", checked, all)
	}
}

// TestEngine_ConcurrentExecute_ScreenshotReportBelongsToOwner 专项：人工干预上报的归属。
func TestEngine_ConcurrentExecute_ScreenshotReportBelongsToOwner(t *testing.T) {
	fixtureSrv, _ := startFixture(t)
	cap, reporter := newProgressCapture(t)
	eng := newRaceEngine(t, reporter)

	const (
		waitMS    = 5000
		delayForB = 800
		execA     = "exec-X-0001"
		execB     = "exec-Y-0002"
		markerA   = "act-X-"
		markerB   = "act-Y-"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = eng.Execute(ctx, raceTask(execA, markerA, waitMS, fixtureSrv.URL, true), "worker-race-test")
	}()
	time.Sleep(delayForB * time.Millisecond)
	go func() {
		defer wg.Done()
		_, _ = eng.Execute(ctx, raceTask(execB, markerB, waitMS, fixtureSrv.URL, false), "worker-race-test")
	}()
	wg.Wait()

	paused := cap.pausedReports()
	if len(paused) == 0 {
		t.Fatal("应捕获到人工干预（pause）进度上报")
	}
	for _, r := range paused {
		want := ownerOf(r.Message, markerA, markerB, execA, execB)
		if want == "" {
			t.Fatalf("无法识别归属的人工干预上报: %+v", r)
		}
		if r.ExecutionID != want {
			t.Errorf("人工干预进度上报串号：message=%q 属于 %s，却挂到 execution_id=%s 上",
				r.Message, want, r.ExecutionID)
		}
	}
}
