package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/liveview"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// W7.1 / 缺陷 A + 缺陷 B 的验收落点
//
// 缺陷 A（动作类型别名归一化）：
//   TestMapToAction_AliasNormalization    goto/goTo/GoTo/GOTO/url → ActionNavigate
//   TestMapToAction_UnknownTypeUntouched  未知类型不被静默吞掉，仍走 unsupported
//
// 缺陷 B（force 语义区分配置错误与运行时失败）：
//   TestEngine_NonRetryableError_FailFast    retry=5 的配置错误：不 gate、不重试、
//                                            立刻 failed（生产事故的回归锁）
//   TestEngine_RetryableError_StillGates    可重试失败 + copilot：仍按原设计 gate
//
// 后两条走真 Chromium（与 engine_liveview_test.go 同款 fixture / skip 约定）：
// 闸门是「阻塞」这种时序行为，只有真实 Execute 才能证明 —— mock 一个 page
// 只能证明「没进闸门函数」，证明不了「不会卡 10 分钟」。
// ============================================================================

// recordingLogger 记录全部日志行（用于断言「闸门有没有被进入」）。
//
// 与 fixture_test.go 的 testLogger 不同：这里要保留 WARN/INFO 文本供匹配，
// 因为「没打 gate waiting」本身就是缺陷 B 的判定依据之一。
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) append(level, msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var parts []string
	for _, f := range fields {
		parts = append(parts, fmt.Sprint(f))
	}
	l.lines = append(l.lines, level+" "+msg+" "+strings.Join(parts, " "))
}

func (l *recordingLogger) Debug(msg string, f ...logger.Field) { l.append("DEBUG", msg, f...) }
func (l *recordingLogger) Info(msg string, f ...logger.Field)  { l.append("INFO", msg, f...) }
func (l *recordingLogger) Warn(msg string, f ...logger.Field)  { l.append("WARN", msg, f...) }
func (l *recordingLogger) Error(msg string, f ...logger.Field) { l.append("ERROR", msg, f...) }
func (l *recordingLogger) With(...logger.Field) logger.Logger  { return l }
func (l *recordingLogger) Sync() error                         { return nil }

// has 是否出现过包含 substr 的日志行。
func (l *recordingLogger) has(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

// all 返回全部日志（失败时打到测试输出便于定位）。
func (l *recordingLogger) all() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

// reset 清空已记录日志（用于把「暖池/前置」的日志与被测执行隔离）。
func (l *recordingLogger) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = nil
}

// ---------------------------------------------------------------------------
// 缺陷 A：别名归一化
// ---------------------------------------------------------------------------

// TestMapToAction_AliasNormalization W7.1 / 缺陷 A 主断言。
//
// 大小写不敏感是刻意的：历史数据/人工编辑会出现 GoTo / GOTO，按每种大小写
// 各写一行会让别名表随数据增长失控。这里用四种写法覆盖任务书要求的全部形态。
func TestMapToAction_AliasNormalization(t *testing.T) {
	cases := []struct {
		raw  string
		want types.ActionType
	}{
		{"goto", types.ActionNavigate}, // 生产库实测 6 步
		{"goTo", types.ActionNavigate},
		{"GoTo", types.ActionNavigate},
		{"GOTO", types.ActionNavigate},
		{"url", types.ActionNavigate},      // 历史脚本把 URL 当类型名
		{"navigate", types.ActionNavigate}, // 规范写法不得被改写
	}

	// 用 nil logger 的 Engine：归一化必须与日志解耦（nestedActionsFrom 里
	// `var tmp Engine` 就是这么用的，logger 为 nil 不能 panic）。
	var eng Engine
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			action := eng.mapToAction(map[string]interface{}{
				"id":    "step1",
				"type":  c.raw,
				"value": "https://www.baidu.com",
				"params": map[string]interface{}{
					"waitUntil": "networkidle",
				},
			})
			if action.Type != c.want {
				t.Fatalf("type=%q 归一化后 = %q，期望 %q", c.raw, action.Type, c.want)
			}
			// 第二层：URL 键位提升（navigate case 读 params["url"]，
			// 而存量数据把 URL 放在 value）。不补这一层，别名修好了也照样
			// 报 "missing url parameter" —— 只是换了个错误。
			if action.Type == types.ActionNavigate {
				if got, _ := action.Params["url"].(string); got != "https://www.baidu.com" {
					t.Fatalf("type=%q 的 URL 未从 value 提升到 params.url，实际 %q", c.raw, got)
				}
			}
		})
	}
}

// TestMapToAction_ExplicitURLWins 显式 params.url 永远优先于 value（不覆盖）。
func TestMapToAction_ExplicitURLWins(t *testing.T) {
	var eng Engine
	action := eng.mapToAction(map[string]interface{}{
		"type":  "goto",
		"value": "https://placeholder.example",
		"params": map[string]interface{}{
			"url": "https://real.example",
		},
	})
	if got, _ := action.Params["url"].(string); got != "https://real.example" {
		t.Fatalf("params.url 被 value 覆盖了：%q", got)
	}
	if action.Type != types.ActionNavigate {
		t.Fatalf("Type=%q，期望 navigate", action.Type)
	}
}

// TestMapToAction_UnknownTypeUntouched 反向用例：未知类型不得被静默归一化。
//
// 任务书明确要求「未知类型 frobnicate → 仍返回 unsupported（不静默吞掉）」。
// 两段断言：① mapToAction 不改写它（Type 原样保留，别名表不做近似猜测）；
// ② executeAction 走 default 返回不可重试错误（既给可读原因，也不会被重试）。
func TestMapToAction_UnknownTypeUntouched(t *testing.T) {
	var eng Engine
	action := eng.mapToAction(map[string]interface{}{
		"type":   "frobnicate",
		"retry":  5,
		"params": map[string]interface{}{},
	})
	if action.Type != types.ActionType("frobnicate") {
		t.Fatalf("未知类型被改写成 %q（不得静默归一化）", action.Type)
	}

	// executeAction 的 default 分支在碰 page 之前就返回，但函数入口会读
	// e.logger（nil 会 panic）与 variables（nil 安全），因此用一个带
	// recording logger 的空 Engine + 空作用域来走真实分支。
	eng2 := &Engine{logger: &recordingLogger{}}
	vars := NewVariableScope(map[string]interface{}{})
	err := eng2.executeAction(context.Background(), nil, nil, &action, vars)
	if err == nil {
		t.Fatal("未知类型必须返回错误，不得静默成功")
	}
	if !types.IsNonRetryable(err) {
		t.Fatalf("未知动作类型必须标记为不可重试（否则会空转重试并误触发人工闸门），实际 err=%v", err)
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Fatalf("错误文案必须带上原始类型便于排查，实际: %v", err)
	}
	// 文案面向一般员工：不得出现内部术语。
	for _, banned := range []string{"unsupported", "ActionType", "switch"} {
		if strings.Contains(err.Error(), banned) {
			t.Fatalf("错误文案含内部术语 %q，用户看不懂: %v", banned, err)
		}
	}
	t.Logf("未知类型 fail-fast 文案: %v", err)
}

// ---------------------------------------------------------------------------
// 缺陷 B：fail-fast vs 仍 gate（真 Chromium）
// ---------------------------------------------------------------------------

// newIntegrationEngineWithLogger 与 fixture_test.go 的 newIntegrationEngine 同款，
// 但把 Engine 的 logger 换成可断言的 recordingLogger（闸门行为只能从日志确认）。
func newIntegrationEngineWithLogger(t *testing.T, downloadsDir string, rec logger.Logger) (*Engine, *browser.Manager) {
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
			Headless: true, Timeout: 10 * time.Second, NavigationTimeout: 15 * time.Second,
			MaxConcurrency: 2, ViewportWidth: 1280, ViewportHeight: 800,
			ChromePath: chrome,
		},
		Worker: config.WorkerConfig{
			MaxConcurrency: 2,
			DownloadsDir:   downloadsDir,
			ActionTimeout:  10 * time.Second,
			PauseTimeout:   1 * time.Second,
		},
		Executor: config.ExecutorConfig{
			InitialDelay: 100 * time.Millisecond, UndefinedVariablePolicy: config.UndefinedVariableEmpty,
		},
	}
	cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency

	mgr, err := browser.NewManager(&cfg.Browser, rec)
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	eng := NewEngine(&cfg.Executor, &cfg.Worker, mgr, nil, rec, nil)
	return eng, mgr
}

// newGateTestEnv 起一套真 Engine + 真 Hub（copilot 模式、无 screencast）。
//
// 模式来源就是 hub 的 DefaultMode=copilot（Attach 时写入会话），因此不必先
// SetMode —— 那个调用在会话尚未登记时还会失败，反而制造噪声。
func newGateTestEnv(t *testing.T) (*Engine, *liveview.Hub, *recordingLogger, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	url := startLiveViewFixture(t)

	rec := &recordingLogger{}
	hub := liveview.NewHub(&config.LiveViewConfig{
		Enabled:         true,
		MaxStreams:      2,
		IdleTimeout:     10 * time.Second,
		TakeoverTimeout: 30 * time.Second,
		DefaultMode:     string(types.LiveModeCopilot),
	}, nil, nil, rec)
	t.Cleanup(hub.Close)

	engine, mgr := newIntegrationEngineWithLogger(t, t.TempDir(), rec)
	engine.SetLiveViewHub(hub)
	t.Cleanup(func() { _ = mgr.Close() })

	return engine, hub, rec, url
}

// TestEngine_NonRetryableError_FailFast 缺陷 B 的回归锁。
//
// 复现生产事故形状：动作 type 未知（等价于 goto 未被识别）+ retry=5 +
// copilot 模式。修复前：进 gate(force=true) → needWait 恒真 → 卡到 ctx 超时
// （生产里是 10 分钟 deadline exceeded）。修复后：立刻 failed。
//
// 断言三件事：
//  1. 不 gate：日志里没有 "liveview gate waiting"；
//  2. 不空转重试：日志里没有 "action failed, retrying"；
//  3. 立刻 failed：耗时远小于 ctx 预算，状态 failed 且文案可读。
func TestEngine_NonRetryableError_FailFast(t *testing.T) {
	engine, _, rec, url := newGateTestEnv(t)
	execID := "w71-nonretryable-exec"

	// copilot 是 hub 的 DefaultMode（Attach 会写入会话），这里不再 SetMode
	// —— 会话尚未登记时该调用会返回「执行不在本 Worker 运行」，只是噪声。

	// 暖池：先跑一次正常导航，让 Chrome/page 就绪。这样下面计时的
	// 「fail-fast 耗时」只剩执行阶段本身（否则 4~5s 里绝大多数是冷启动 +
	// 导航，无法证明「立刻」）。
	warmCtx, warmCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer warmCancel()
	if _, warmErr := engine.Execute(warmCtx, &types.TaskMessage{
		ExecutionID: execID + "-warmup",
		TaskID:      "w71-warmup",
		TaskName:    "暖池",
		TargetURL:   url,
		Script:      []types.Action{{Type: types.ActionClick, Selector: "#btn"}},
		Variables:   map[string]interface{}{},
	}, "worker-w71"); warmErr != nil {
		t.Logf("暖池执行 err=%v（继续，冷启动会体现在耗时上）", warmErr)
	}
	rec.reset() // 暖池日志不算数，只保留被测执行的日志

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	started := time.Now()
	res, err := engine.Execute(ctx, &types.TaskMessage{
		ExecutionID: execID,
		TaskID:      "w71-nonretryable",
		TaskName:    "配置错误 fail-fast",
		TargetURL:   url,
		Script: []types.Action{
			{Type: types.ActionType("frobnicate"), Retry: 5, Description: "未知类型（等价生产 goto）"},
		},
		Variables: map[string]interface{}{},
	}, "worker-w71")
	elapsed := time.Since(started)

	if err != nil {
		// Execute 的错误分支返回 nil error 走 result；真报错才怀疑链路。
		t.Logf("Execute 顶层 err=%v（非致命，看 result）", err)
	}
	if res == nil {
		t.Fatalf("Execute 未返回结果")
	}
	if string(res.Status) != string(types.StatusFailed) {
		t.Fatalf("配置错误必须立刻 failed，实际 status=%s（err=%s）", res.Status, res.ErrorMessage)
	}
	// 判定「立刻」的口径：修复前这里会一直卡到 ctx 预算（20s）耗尽（生产
	// 是 10 分钟 task_timeout）。3s 阈值远小于 ctx 预算，且余量覆盖 CI 抖动。
	if elapsed > 3*time.Second {
		t.Fatalf("配置错误耗时 %v —— 疑似仍进入闸门/空转重试（修复前会卡到 ctx 超时）", elapsed)
	}
	// 反向自证：ctx 预算远大于实际耗时，说明不是「恰好卡到超时」的假象。
	if ctx.Err() != nil {
		t.Fatalf("ctx 已被耗尽（elapsed=%v），说明仍在阻塞而非 fail-fast", elapsed)
	}
	if !strings.Contains(res.ErrorMessage, "frobnicate") {
		t.Fatalf("失败原因必须指出是哪一步的类型有问题，实际: %s", res.ErrorMessage)
	}

	// 1) 没有进闸门
	if rec.has("liveview gate waiting") {
		t.Fatalf("配置错误不应触发人工闸门（日志出现 gate waiting）:\n%s", rec.all())
	}
	// 2) 没有空转重试
	if rec.has("action failed, retrying") {
		t.Fatalf("配置错误不应重试（日志出现 action failed, retrying）:\n%s", rec.all())
	}
	// 3) 有明确的「跳过重试」WARN，便于运维在日志里识别这类事故
	if !rec.has("non-retryable configuration error") {
		t.Fatalf("缺少「配置错误跳过重试」WARN，运维无法在日志里识别:\n%s", rec.all())
	}
	t.Logf("fail-fast 证据: 耗时=%v status=%s step=%d/%d 原因=%s",
		elapsed, res.Status, res.Step, res.Total, res.ErrorMessage)
}

// TestEngine_RetryableError_StillGates 缺陷 B 的「反向保护」：不能修过头。
//
// 元素不存在（网络/页面时序问题，重试有意义）→ 必须仍然按 LIVEVIEW-03 的设计
// 在 copilot 下停下来等人看一眼。断言：
//  1. Continue 之前 Execute 不返回（真阻塞）；
//  2. 日志确实出现 gate waiting（进入闸门）；
//  3. Continue 之后重试真的发生了（action failed, retrying），最终 failed
//     且**不是**不可重试错误（保留原语义）。
func TestEngine_RetryableError_StillGates(t *testing.T) {
	engine, hub, rec, url := newGateTestEnv(t)
	execID := "w71-retryable-exec"
	// copilot 来自 hub 的 DefaultMode（Attach 写入会话）；执行前 SetMode 会
	// 因会话未登记而失败，故不调用。

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	done := make(chan *types.ExecutionResult, 1)
	go func() {
		res, _ := engine.Execute(ctx, &types.TaskMessage{
			ExecutionID: execID,
			TaskID:      "w71-retryable",
			TaskName:    "可重试失败仍要 gate",
			TargetURL:   url,
			Script: []types.Action{
				// 元素不存在：页面时序问题，重试有意义 → 可重试。
				// 选择器阶梯会先按 ActionTimeout 等元素，失败后进入重试分支。
				{Type: types.ActionClick, Selector: "#no-such-element", Retry: 2,
					Description: "点一个不存在的元素"},
			},
			Variables: map[string]interface{}{},
		}, "worker-w71")
		done <- res
	}()

	// 1) 真阻塞 + 进闸门：轮询等 gate waiting 出现，期间若有结果返回即为失败。
	//    注意「阻塞」的证明是「done 上没有结果」而不是固定睡多久 —— 元素定位
	//    本身要等超时，固定 700ms 会误判成「没进闸门」。
	deadline := time.Now().Add(25 * time.Second)
	gated := false
	for time.Now().Before(deadline) {
		select {
		case res := <-done:
			t.Fatalf("可重试失败未被闸门阻塞，Execute 直接返回 status=%s（err=%s）\n%s",
				res.Status, res.ErrorMessage, rec.all())
		case <-time.After(100 * time.Millisecond):
		}
		if rec.has("liveview gate waiting") {
			gated = true
			break
		}
	}
	if !gated {
		t.Fatalf("可重试失败必须进闸门，25s 内日志无 gate waiting:\n%s", rec.all())
	}

	// 2) 放行：人工点「继续」。
	if err := hub.Continue(execID, "test", "人工确认后重试"); err != nil {
		t.Logf("Continue: %v", err)
	}

	var res *types.ExecutionResult
	select {
	case res = <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("Continue 之后 30s 仍未结束（闸门放行失败）:\n%s", rec.all())
	}

	// 3) 重试确实发生（否则说明我们把可重试分支也改成了 fail-fast，属于修过头）。
	if !rec.has("action failed, retrying") {
		t.Fatalf("放行后应进入重试（日志无 action failed, retrying）:\n%s", rec.all())
	}
	if string(res.Status) != string(types.StatusFailed) {
		t.Fatalf("重试用尽后应 failed，实际 %s", res.Status)
	}
	// 保留原语义：元素不存在是运行时失败，不得被误标成配置错误。
	if rec.has("non-retryable configuration error") {
		t.Fatalf("元素不存在被误判为配置错误（修过头了）:\n%s", rec.all())
	}
	t.Logf("仍 gate 证据: status=%s step=%d/%d 原因=%s", res.Status, res.Step, res.Total, res.ErrorMessage)
}
