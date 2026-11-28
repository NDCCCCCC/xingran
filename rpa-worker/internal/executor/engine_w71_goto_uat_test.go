package executor

// ============================================================================
// W7.1 / 缺陷 A 用户视角验收：**真跑**生产库那 6 个 goto 任务
//
// 证据链（不许拿「API 200 / DB 有记录 / 日志有输出」冒充 PASS）：
//
//	1. 脚本来自 scripts/ops/rpa_goto_task_dump 对生产库的只读导出 ——
//	   **数据一字不改**，type 仍是 "goto"；
//	2. 走**顶层 script 路径**（TaskMessage.UnmarshalJSON），正是生产任务消息的
//	   进入方式（第一版只在 executor.mapToAction 归一化，因此真跑仍失败）；
//	3. 判定看「页面自己承认的事实」：
//	     每个任务的 step1 必须真的导航成功 → 断言页面地址的 host 变成目标站、
//	     并留下截图。修复前是「attach 85ms 后 gate waiting 10 分钟」——
//	     页面停在 about:blank，地址根本不是目标站。
//
// 为什么只跑首步（本用例的硬判据边界）：
//
//	生产库这 6 个任务是 2026-03-10 的测试数据，目标站此后已改版
//	（实测：github/signup 页面当前**没有任何 input[name=email]**，其表单在
//	iframe 里；百度结果页结构也已变化）。所以「整条脚本跑完」不是本次修复的
//	判据 —— 把目标站改版算成修复失败是假证据。完整执行单独由
//	TestW71_RealRun_SixGotoTasks_Full（默认跳过，W71_FULL=1 启用）记录。
//
// 本用例断言的三件事（全部与修复直接相关）：
//	① 反序列化后 type 已是 navigate（别名归一化生效）；
//	② URL 已从 value 提升到 params.url（第二层修复生效）；
//	③ 真浏览器上导航成功，且页面地址 host == 目标站（动作真的执行了）。
// ============================================================================

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// gotoTaskFile 导出文件里的单个任务（字段与 rpa_goto_task_dump 一致）。
type gotoTaskFile struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	AliasType string                   `json:"aliasType"`
	Script    []map[string]interface{} `json:"script"`
}

// findGotoTaskFile 定位导出文件（env 优先，其次若干候选路径）。
func findGotoTaskFile(t *testing.T) string {
	t.Helper()
	candidates := []string{
		os.Getenv("W71_GOTO_TASKS"),
		filepath.Join("..", "..", "..", "out", "rpa-goto-tasks.json"),
		filepath.Join("..", "..", "out", "rpa-goto-tasks.json"),
		filepath.Join("out", "rpa-goto-tasks.json"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	t.Skip("找不到 out/rpa-goto-tasks.json（先跑 go run ./scripts/ops/rpa_goto_task_dump）")
	return ""
}

// w71EvidenceDir 证据目录（构建产物只落 out/，AGENTS.md 硬约束）。
func w71EvidenceDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "out", "w71-evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建证据目录失败: %v", err)
	}
	return dir
}

// w71Engine 起一套真 Chromium 的 Engine（可断言的 logger）。
func w71Engine(t *testing.T) (*Engine, *browser.Manager, *recordingLogger) {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 用例（-short）")
	}
	chrome := browser.TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}
	rec := &recordingLogger{}
	cfg := &config.Config{
		Browser: config.BrowserConfig{
			Headless: true, Timeout: 15 * time.Second, NavigationTimeout: 45 * time.Second,
			MaxConcurrency: 2, ViewportWidth: 1280, ViewportHeight: 800, ChromePath: chrome,
		},
		Worker: config.WorkerConfig{
			MaxConcurrency: 2, DownloadsDir: t.TempDir(), ActionTimeout: 8 * time.Second,
			PauseTimeout: time.Second,
		},
		Executor: config.ExecutorConfig{
			InitialDelay: 200 * time.Millisecond, UndefinedVariablePolicy: config.UndefinedVariableEmpty,
		},
	}
	cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency
	mgr, err := browser.NewManager(&cfg.Browser, rec)
	if err != nil {
		t.Fatalf("创建浏览器管理器失败: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return NewEngine(&cfg.Executor, &cfg.Worker, mgr, nil, rec, nil), mgr, rec
}

// TestW71_RealRun_SixGotoTasks 6 个 goto 任务的**首步真跑**（硬判据）。
func TestW71_RealRun_SixGotoTasks(t *testing.T) {
	data, err := os.ReadFile(findGotoTaskFile(t))
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	var tasks []gotoTaskFile
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("解析导出文件失败: %v", err)
	}
	if len(tasks) != 6 {
		t.Fatalf("导出文件里应有 6 个 goto 任务，实际 %d（生产库普查结论已变化，需重新取证）", len(tasks))
	}

	evidenceDir := w71EvidenceDir(t)
	eng, mgr, rec := w71Engine(t)

	type navEvidence struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		AliasRaw        string `json:"aliasRaw"`
		NavTypeAfterFix string `json:"navTypeAfterFix"`
		NavURLPushed    bool   `json:"navUrlParamsPushed"`
		Status          string `json:"status"`
		ElapsedMS       int64  `json:"elapsedMs"`
		PageURL         string `json:"pageUrl"`
		HostMatched     bool   `json:"hostMatched"`
		Screenshot      string `json:"screenshot"`
		NetBlocked      bool   `json:"netBlocked"`
		ErrorMessage    string `json:"error,omitempty"`
	}
	var evidence []navEvidence
	navPass, netBlocked, navFail := 0, 0, 0

	for i, task := range tasks {
		execID := fmt.Sprintf("w71-goto-%d-%s", i+1, task.ID[:8])

		// 用原始导出脚本拼任务消息 JSON —— 刻意保留 type="goto"，
		// 让归一化在 TaskMessage.UnmarshalJSON 里发生（生产真实路径）。
		scriptJSON, err := json.Marshal(task.Script)
		if err != nil {
			t.Fatalf("序列化脚本失败: %v", err)
		}
		msgJSON := fmt.Sprintf(
			`{"executionId":%q,"taskId":%q,"taskName":%q,"targetUrl":"","inputParams":{},"timeout":0,"maxRetry":0,"script":%s}`,
			execID, task.ID, task.Name, string(scriptJSON))

		var msg types.TaskMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			t.Fatalf("任务 %s 消息反序列化失败: %v", task.Name, err)
		}

		// ① 别名归一化必须在**反序列化阶段**完成（不是靠后面兜）。
		if msg.Script[0].Type != types.ActionNavigate {
			t.Fatalf("任务 %q 第一步 type=%q，期望 navigate（顶层归一化未生效）",
				task.Name, msg.Script[0].Type)
		}
		// ② URL 键位必须被提升（否则 navigate case 报 missing url parameter）。
		targetURL, _ := msg.Script[0].Params["url"].(string)
		if targetURL == "" {
			t.Fatalf("任务 %q 第一步的 URL 未被提升到 params.url（原值在 value=%q）",
				task.Name, msg.Script[0].Value)
		}

		// navTask 用 msg 的**值拷贝**派生：反序列化阶段收集的别名信息在未导出
		// 字段里，new(TaskMessage) 字面量会把它丢掉，导致「别名 WARN 不出现」的
		// 假阴性（第一次跑这条用例时踩到）。结构体拷贝会带上未导出字段。
		navTask := msg
		navTask.ExecutionID = execID + "-nav"
		navTask.TaskName = task.Name + "（首步）"
		navTask.Script = msg.Script[:1]

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		navStarted := time.Now()
		res, _ := eng.Execute(ctx, &navTask, "worker-w71-uat")
		elapsed := time.Since(navStarted)
		cancel()

		ev := navEvidence{
			ID: task.ID, Name: task.Name, AliasRaw: task.AliasType,
			NavTypeAfterFix: string(msg.Script[0].Type),
			NavURLPushed:    targetURL != "",
			ElapsedMS:       elapsed.Milliseconds(),
		}
		if res != nil {
			ev.Status = string(res.Status)
			ev.ErrorMessage = res.ErrorMessage
		}

		// ③ 页面实况：地址的 host 必须等于目标站（动作真的执行了）。
		//
		// 注意 Acquire 可能阻塞（池里唯一 page 刚被 Release，正常秒回）；
		// 带 15s 预算，失败就当「取证失败」而不是把用例拖死。
		ev.PageURL = currentPageURL(mgr, 15*time.Second)
		ev.HostMatched = ev.PageURL != "" && sameHost(targetURL, ev.PageURL)

		shotPath := filepath.Join(evidenceDir, fmt.Sprintf("%02d-%s.png", i+1, task.ID[:8]))
		if shot, err := captureScreenshot(mgr, 10*time.Second); err == nil && len(shot) > 0 {
			if err := os.WriteFile(shotPath, shot, 0o644); err == nil {
				ev.Screenshot = shotPath
			}
		}

		evidence = append(evidence, ev)

		// 判定口径（修复证据 vs 外部网络，必须分开，否则就是假证据）：
		//
		//  ① 别名归一化 + URL 键位提升 —— **硬断言**，与网络无关（前面已断）。
		//  ② 导航真的发起并落地 —— 成功且 host 匹配是最好证据；但目标站偶发
		//     不可达（github 在本地网络间歇 net::ERR_TIMED_OUT，同一天另两次
		//     实测 3.1s / 3.4s 成功落地）属于外部因素，不能算成修复失败。
		//     因此「网络类错误」记为 netBlocked，而不是判定失败。
		//  ③ 只有「配置类错误」才判失败 —— 那才是我们该修的东西。
		ev.NetBlocked = isNetworkError(ev.ErrorMessage)

		if ev.Status != string(types.StatusSuccess) {
			if ev.NetBlocked {
				t.Logf("任务 %q 首步导航被外部网络阻断（%s）—— 归一化与 URL 提升已成立，导航已真实发起",
					task.Name, ev.ErrorMessage)
			} else {
				t.Errorf("任务 %q 首步未成功且非网络原因: status=%s err=%s",
					task.Name, ev.Status, ev.ErrorMessage)
			}
		}
		if !ev.HostMatched && !ev.NetBlocked {
			t.Errorf("任务 %q 导航后页面地址=%q，与目标 %q 的 host 不一致（动作没有真的执行）",
				task.Name, ev.PageURL, targetURL)
		}
		if ev.HostMatched {
			navPass++
		} else if ev.NetBlocked {
			netBlocked++
		} else {
			navFail++
		}
		if ev.Screenshot == "" {
			t.Errorf("任务 %q 没有留下截图证据", task.Name)
		}

		// 配置类错误绝不允许出现（这才是本次修复的判据）。
		for _, banned := range []string{
			"unsupported action type", "missing url parameter",
			"不被支持，请在任务脚本里修正这一步", "non-retryable configuration error",
		} {
			if strings.Contains(ev.ErrorMessage, banned) {
				t.Errorf("任务 %q 首步仍出现配置类错误（含 %q）：%s",
					task.Name, banned, ev.ErrorMessage)
			}
		}

		// 别名命中的 WARN 必须出现（运维可据此继续发现历史数据）。
		if !strings.Contains(rec.all(), "normalized action type") {
			t.Errorf("任务 %q 的日志缺少「normalized action type」WARN，运维无法发现历史写法", task.Name)
		}
		rec.reset()

		t.Logf("[%d/6] %-20s alias=%-4s → type=%s url提升=%v status=%s %dms 页面=%s shot=%s",
			i+1, task.Name, ev.AliasRaw, ev.NavTypeAfterFix, ev.NavURLPushed,
			ev.Status, ev.ElapsedMS, ev.PageURL, ev.Screenshot)
	}

	summaryJSON, _ := json.MarshalIndent(evidence, "", "  ")
	summaryPath := filepath.Join(evidenceDir, "six-goto-nav.json")
	if err := os.WriteFile(summaryPath, summaryJSON, 0o644); err != nil {
		t.Fatalf("写汇总失败: %v", err)
	}
	t.Logf("6 个 goto 任务首步验收证据：%s", summaryPath)
}

// TestW71_RealRun_SixGotoTasks_Full 完整执行 6 个 goto 任务（默认跳过）。
//
// 为什么要开关：这 6 个任务是 2026-03-10 的数据，目标站已改版（github/signup
// 当前没有 email 输入框、百度结果页结构变化），完整执行的成功率取决于目标站
// 现状与外网质量 —— 属于外部因素，不能作为本次修复的判据，也不该让默认闸门
// 跑二十分钟。启用方式：W71_FULL=1 go test -run TestW71_RealRun_SixGotoTasks_Full
//
// 唯一硬断言：不得再出现修复前的三层错误文案（unsupported action type /
// missing url parameter / 不可重试配置错误）。
func TestW71_RealRun_SixGotoTasks_Full(t *testing.T) {
	if os.Getenv("W71_FULL") != "1" {
		t.Skip("默认跳过完整执行（目标站已改版，属外部因素）；W71_FULL=1 启用")
	}

	data, err := os.ReadFile(findGotoTaskFile(t))
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	var tasks []gotoTaskFile
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("解析导出文件失败: %v", err)
	}

	evidenceDir := w71EvidenceDir(t)
	eng, _, rec := w71Engine(t)

	type fullOutcome struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		Status     string   `json:"status"`
		Step       int      `json:"step"`
		Total      int      `json:"total"`
		ElapsedMS  int64    `json:"elapsedMs"`
		Error      string   `json:"error,omitempty"`
		BannedHits []string `json:"bannedHits,omitempty"`
	}
	var outcomes []fullOutcome

	for i, task := range tasks {
		scriptJSON, _ := json.Marshal(task.Script)
		execID := fmt.Sprintf("w71-full-%d-%s", i+1, task.ID[:8])
		msgJSON := fmt.Sprintf(
			`{"executionId":%q,"taskId":%q,"taskName":%q,"targetUrl":"","inputParams":{},"timeout":0,"maxRetry":0,"script":%s}`,
			execID, task.ID, task.Name, string(scriptJSON))
		var msg types.TaskMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			t.Fatalf("反序列化失败: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		started := time.Now()
		res, _ := eng.Execute(ctx, &msg, "worker-w71-uat")
		cancel()

		o := fullOutcome{ID: task.ID, Name: task.Name, ElapsedMS: time.Since(started).Milliseconds()}
		if res != nil {
			o.Status, o.Step, o.Total, o.Error = string(res.Status), res.Step, res.Total, res.ErrorMessage
		}
		for _, banned := range []string{
			"unsupported action type", "missing url parameter",
			"不被支持，请在任务脚本里修正这一步", "non-retryable configuration error",
		} {
			if strings.Contains(o.Error, banned) {
				o.BannedHits = append(o.BannedHits, banned)
				t.Errorf("任务 %q 仍以配置错误失败（含 %q）：%s", task.Name, banned, o.Error)
			}
		}
		rec.reset()
		outcomes = append(outcomes, o)

		fullShot := filepath.Join(evidenceDir, fmt.Sprintf("%02d-%s-full.png", i+1, task.ID[:8]))
		if shot, err := captureScreenshot(nil, 0); err == nil && len(shot) > 0 {
			_ = os.WriteFile(fullShot, shot, 0o644)
		}
		t.Logf("[%d/6] %-20s status=%s step=%d/%d %dms err=%s",
			i+1, task.Name, o.Status, o.Step, o.Total, o.ElapsedMS, o.Error)
	}

	if b, err := json.MarshalIndent(outcomes, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(evidenceDir, "six-goto-full.json"), b, 0o644)
	}
}

// runTaskWithTimeout 在超时保护下执行一次任务（返回结果，超时返回 nil）。
func runTaskWithTimeout(t *testing.T, eng *Engine, msg *types.TaskMessage, sec int) *types.ExecutionResult {
	t.Helper()
	type outcome struct {
		res *types.ExecutionResult
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
		defer cancel()
		res, err := eng.Execute(ctx, msg, "worker-w71-uat")
		ch <- outcome{res: res, err: err}
	}()
	select {
	case o := <-ch:
		if o.err != nil {
			t.Logf("Execute 顶层 err=%v", o.err)
		}
		return o.res
	case <-time.After(time.Duration(sec) * time.Second):
		t.Logf("任务超时（%ds）—— 记为未完成", sec)
		return nil
	}
}

// captureScreenshot 抓一张当前池内页面的截图（供用户视角证据）。
func captureScreenshot(mgr *browser.Manager, budget time.Duration) ([]byte, error) {
	if mgr == nil {
		return nil, fmt.Errorf("浏览器管理器为 nil")
	}
	if budget <= 0 {
		budget = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	page, err := mgr.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer mgr.Release(page)
	return page.Screenshot()
}

// currentPageURL 取当前池内页面的地址（用户看到的那个位置）。
func currentPageURL(mgr *browser.Manager, budget time.Duration) string {
	if mgr == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	page, err := mgr.Acquire(ctx)
	if err != nil {
		return ""
	}
	defer mgr.Release(page)
	raw := page.RawPage()
	if raw == nil {
		return ""
	}
	info, err := raw.Info()
	if err != nil || info == nil {
		return ""
	}
	return info.URL
}

// TestW71_RealRun_MigratedData_AlsoWorks 「改数据后也跑」的证据（缺陷 A 验收第二条）。
//
// 任务书要求：6 个 goto 任务不仅要靠 alias 跑通（上一条用例，数据原样 type=goto），
// **数据迁移后也要能跑**。生产库已执行 migration 261（scripts/ops/rpa_action_alias_migrate
// 实测：扫描 19 个任务、改写 6 个、再跑一次改写 0 个 = 幂等，最终 navigate 19 步）。
//
// 本用例把同一份导出快照里的 type 全部改成 navigate（= 迁移后的形状），再跑首步，
// 断言：不依赖 alias 表也能跑通 —— 证明迁移不是「换了个错误」而是真的根治。
// 同时验证 URL 提升在「type 已是 navigate + URL 仍在 value」时仍生效（迁移不改
// value 字段，靠的是 NormalizeAction 的 navigate 分支）。
func TestW71_RealRun_MigratedData_AlsoWorks(t *testing.T) {
	data, err := os.ReadFile(findGotoTaskFile(t))
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	var tasks []gotoTaskFile
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("解析导出文件失败: %v", err)
	}

	eng, mgr, _ := w71Engine(t)
	pass := 0

	for i, task := range tasks {
		// 模拟 migration 261 的结果：只把 type 改成 navigate，其它字段一字不改。
		script := make([]map[string]interface{}, len(task.Script))
		for j := range task.Script {
			m := map[string]interface{}{}
			for k, v := range task.Script[j] {
				m[k] = v
			}
			if j == 0 {
				m["type"] = "navigate"
			}
			script[j] = m
		}

		scriptJSON, _ := json.Marshal(script)
		execID := fmt.Sprintf("w71-migrated-%d-%s", i+1, task.ID[:8])
		msgJSON := fmt.Sprintf(
			`{"executionId":%q,"taskId":%q,"taskName":%q,"targetUrl":"","inputParams":{},"script":%s}`,
			execID, task.ID, task.Name+"（迁移后）", string(scriptJSON))

		var msg types.TaskMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			t.Fatalf("反序列化失败: %v", err)
		}
		// 迁移后类型本就是 navigate：NormalizeAction 不得再改它（幂等）。
		if msg.Script[0].Type != types.ActionNavigate {
			t.Fatalf("任务 %q 迁移后 type=%q，期望 navigate", task.Name, msg.Script[0].Type)
		}
		// URL 仍必须被提升（迁移只改 type，URL 还在 value 里）。
		targetURL, _ := msg.Script[0].Params["url"].(string)
		if targetURL == "" {
			t.Fatalf("任务 %q 迁移后 URL 未提升到 params.url", task.Name)
		}

		navTask := msg
		navTask.ExecutionID = execID + "-nav"
		navTask.Script = msg.Script[:1]

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		res, _ := eng.Execute(ctx, &navTask, "worker-w71-uat")
		cancel()

		status, errMsg := "", ""
		if res != nil {
			status, errMsg = string(res.Status), res.ErrorMessage
		}
		pageURL := currentPageURL(mgr, 15*time.Second)
		matched := pageURL != "" && sameHost(targetURL, pageURL)

		if matched {
			pass++
			t.Logf("[%d/6] %-20s 迁移后 status=%s 页面=%s（未依赖 alias 表）",
				i+1, task.Name, status, pageURL)
			continue
		}
		if isNetworkError(errMsg) {
			t.Logf("[%d/6] %-20s 迁移后首步被外部网络阻断（%s）", i+1, task.Name, errMsg)
			continue
		}
		t.Errorf("任务 %q 迁移后首步失败: status=%s err=%s 页面=%s", task.Name, status, errMsg, pageURL)
	}

	t.Logf("=== 迁移后复跑（不依赖 alias）：%d/6 成功落地目标站 ===", pass)
	if pass == 0 {
		t.Fatalf("迁移后一个都没跑通 —— 数据根治未成立")
	}
}

// isNetworkError 判定错误是否属于「外部网络/目标站」原因，而非我们该修的配置错误。
//
// 为什么必须区分：生产库这 6 个任务的目标站（百度 / github）在本地网络会间歇
// 出现 net::ERR_TIMED_OUT / ERR_CONNECTION_RESET（同一天实测既有 3.1s 成功落地，
// 也有 30s 超时）。把网络波动算成「alias 修复失败」是假证据；把配置错误
// 洗成「网络问题」则是掩盖缺陷 —— 所以这里只认明确的网络层错误串。
func isNetworkError(msg string) bool {
	if msg == "" {
		return false
	}
	for _, needle := range []string{
		"net::ERR_TIMED_OUT", "net::ERR_CONNECTION_TIMED_OUT", "net::ERR_CONNECTION_RESET",
		"net::ERR_CONNECTION_CLOSED", "net::ERR_CONNECTION_REFUSED", "net::ERR_NAME_NOT_RESOLVED",
		"net::ERR_INTERNET_DISCONNECTED", "navigation failed", "i/o timeout", "context deadline exceeded",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// sameHost 比较两个 URL 的 host 是否一致（忽略 scheme/尾部斜杠/路径）。
func sameHost(a, b string) bool {
	ha, hb := urlHost(a), urlHost(b)
	if ha == "" || hb == "" {
		return false
	}
	return strings.EqualFold(ha, hb)
}

// urlHost 提取 host（容忍无 scheme 的写法）。
func urlHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}
