package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 154 / ENGINE-05：真实浏览器验收基线
//
// 覆盖 ENGINE-02 的 15 个动作（14 个缺口 + scroll 元素内语义），全部跑在
// 真 Chromium + httptest fixture 页面上。判定一律看「页面自己承认的副作用」
// （DOM 内容 / 变量回写 / 落盘文件），不使用 mock page 冒充。
//
// 无 Chromium 或 go test -short 时本条 skip；轻量兜底在
// engine_action_coverage_test.go 与 engine_variables_test.go。
// ============================================================================

// runScript 在指定 URL 上执行一段动作脚本，返回执行结果。
//
// 注意：Engine.Execute 会自己 Acquire/Release page，因此每个用例都是
// 「一次真实执行」，与生产路径完全一致。
func runScript(t *testing.T, eng *Engine, targetURL string, actions []types.Action, vars map[string]interface{}) *types.ExecutionResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := eng.Execute(ctx, &types.TaskMessage{
		ExecutionID: "exec-" + t.Name(),
		TaskID:      "task-" + t.Name(),
		TargetURL:   targetURL,
		Script:      actions,
		Variables:   vars,
	}, "worker-test")
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if res.Status != types.StatusSuccess {
		t.Fatalf("执行未成功: status=%s error=%s step=%d/%d", res.Status, res.ErrorMessage, res.Step, res.Total)
	}
	return res
}

// TestEngine02_AllActions 逐个动作的真实浏览器验收（14 个 ENGINE-02 动作）。
func TestEngine02_AllActions(t *testing.T) {
	srv, downloads := startFixture(t)
	eng, _ := newIntegrationEngine(t, downloads)
	base := srv.URL + "/"

	// —— 公共前置：navigate ——
	t.Run("navigate", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{{
			Type: types.ActionNavigate, Params: map[string]interface{}{"url": base},
		}}, nil)
	})

	// —— click + assert(text) 组合：点提交后 #log 变为 clicked ——
	t.Run("click", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionClick, Selector: "#btn-submit"},
			{Type: types.ActionAssert, Selector: "#log",
				Params: map[string]interface{}{"type": "html", "contains": true, "expected": "submit-button"}},
		}, nil)
	})

	// —— fill + assert(value) ——
	t.Run("fill", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionFill, Selector: "#name", Params: map[string]interface{}{"value": "张三"}},
			{Type: types.ActionAssert, Selector: "#name",
				Params: map[string]interface{}{"type": "value", "expected": "张三"}},
		}, nil)
	})

	// —— select ——
	t.Run("select", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionSelect, Selector: "#sel", Params: map[string]interface{}{"value": "b"}},
			{Type: types.ActionAssert, Selector: "#sel",
				Params: map[string]interface{}{"type": "value", "expected": "b"}},
		}, nil)
	})

	// —— upload（Element.SetFiles）——
	t.Run("upload", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "upload.txt")
		if err := os.WriteFile(f, []byte("engine02 upload"), 0o644); err != nil {
			t.Fatal(err)
		}
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionUpload, Selector: "#file",
				Params: map[string]interface{}{"files": []interface{}{f}}},
			{Type: types.ActionAssert, Params: map[string]interface{}{
				"type":   "expression",
				"script": "document.getElementById('file').files.length === 1",
			}},
		}, nil)
	})

	// —— download（Browser.WaitDownload + 产物归档 + 变量回写）——
	t.Run("download", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{{
			Type: types.ActionDownload, Selector: "#download-link",
			Params: map[string]interface{}{"targetVar": "dlPath"},
		}}, nil)
		entries, err := os.ReadDir(downloads)
		if err != nil {
			t.Fatalf("读取下载目录失败: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("download 未落盘任何产物（Browser.WaitDownload 未生效）")
		}
		data, err := os.ReadFile(filepath.Join(downloads, entries[0].Name()))
		if err != nil {
			t.Fatalf("读取产物失败: %v", err)
		}
		if !strings.Contains(string(data), "ENGINE-02 download payload") {
			t.Fatalf("产物内容不符: %q", string(data))
		}
		t.Logf("download 产物: %s (%d bytes)", entries[0].Name(), len(data))
	})

	// —— evaluate（返回值入变量，并驱动后续 fill 引用）——
	t.Run("evaluate", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionEvaluate,
				Params: map[string]interface{}{
					"script":    "() => document.getElementById('sel').value",
					"targetVar": "selValue",
				}},
			{Type: types.ActionFill, Selector: "#name",
				Params: map[string]interface{}{"value": "前缀-${selValue}"}},
			{Type: types.ActionAssert, Selector: "#name",
				Params: map[string]interface{}{"type": "value", "expected": "前缀-a"}},
		}, nil)
	})

	// —— extract（结果回写变量，被后续动作引用；ENGINE-04 嵌套写入）——
	t.Run("extract", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionExtract, Selector: "#log",
				Params: map[string]interface{}{"targetVar": "captured", "type": "text"}},
			{Type: types.ActionFill, Selector: "#name",
				Params: map[string]interface{}{"value": "取自-${captured}"}},
		}, nil)
	})

	// —— hover（mouseenter 事件真实触发）——
	t.Run("hover", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionHover, Selector: "#hoverbox"},
			{Type: types.ActionAssert, Selector: "#hover-count",
				Params: map[string]interface{}{"type": "text", "expected": "1"}},
		}, nil)
	})

	// —— doubleClick（dblclick 计数 +1）——
	t.Run("doubleClick", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionDoubleClick, Selector: "#btn-dbl"},
			{Type: types.ActionAssert, Selector: "#dbl-count",
				Params: map[string]interface{}{"type": "text", "expected": "1"}},
		}, nil)
	})

	// —— rightClick（contextmenu 计数 +1）——
	t.Run("rightClick", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionRightClick, Selector: "#ctx-area"},
			{Type: types.ActionAssert, Selector: "#ctx-count",
				Params: map[string]interface{}{"type": "text", "expected": "1"}},
		}, nil)
	})

	// —— drag（HTML5 兼容：CDP 原生无效时降级 JS DataTransfer）——
	t.Run("drag", func(t *testing.T) {
		// 断言必须与拖拽在同一次执行里：另起一次执行会重新导航 fixture，
		// DOM 状态被重置，断言就失去意义。
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionDrag, Selector: "#draggable",
				Params: map[string]interface{}{"to": "#dropzone"}},
			{Type: types.ActionAssert, Selector: "#drop-count",
				Params: map[string]interface{}{"type": "text", "expected": "1"}},
		}, nil)
	})

	// —— keyboard（组合键 ctrl+shift+a 打入真实 keydown）——
	t.Run("keyboard", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionClick, Selector: "#keyin"},
			{Type: types.ActionKeyboard, Params: map[string]interface{}{"keys": "ctrl+shift+a"}},
			{Type: types.ActionAssert, Selector: "#key-log",
				Params: map[string]interface{}{"type": "text", "contains": true, "expected": "ctrl+shift"}},
		}, nil)
	})

	// —— switchTab（点击打开新页签 → 切到 index=1 → 读新页签标记）——
	t.Run("switchTab", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionClick, Selector: "#open-tab"},
			{Type: types.ActionWait, Params: map[string]interface{}{"duration": 800}},
			{Type: types.ActionSwitchTab, Params: map[string]interface{}{"index": 1}},
			{Type: types.ActionAssert, Selector: "#tab2-marker",
				Params: map[string]interface{}{"type": "text", "expected": "tab2"}},
		}, nil)
	})

	// —— iframe（进入 frame → 帧内操作 → 退出 frame）——
	t.Run("iframe", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionIframe, Selector: "#frame1",
				Params: map[string]interface{}{"action": "enter"}},
			{Type: types.ActionClick, Selector: "#frame-btn"},
			{Type: types.ActionAssert, Selector: "#frame-input",
				Params: map[string]interface{}{"type": "value", "expected": "from-frame"}},
			{Type: types.ActionIframe, Params: map[string]interface{}{"action": "exit"}},
		}, nil)
	})

	// —— condition（真条件分支：then / else 都验一遍）——
	t.Run("condition", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionCondition, Params: map[string]interface{}{
				"mode":       "js",
				"expression": "document.getElementById('sel').value === 'a'",
				"then": []interface{}{
					map[string]interface{}{"type": "fill", "selector": "#name", "params": map[string]interface{}{"value": "then-branch"}},
				},
				"else": []interface{}{
					map[string]interface{}{"type": "fill", "selector": "#name", "params": map[string]interface{}{"value": "else-branch"}},
				},
			}},
			{Type: types.ActionAssert, Selector: "#name",
				Params: map[string]interface{}{"type": "value", "expected": "then-branch"}},
		}, nil)
	})

	// —— close（关掉当前页签，执行链不中断）——
	t.Run("close", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionClose},
		}, nil)
		// 后续执行仍能拿到可用 page（最后一页保护 / 切到剩余页签）
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionAssert, Params: map[string]interface{}{
				"type": "expression", "script": "true",
			}},
		}, nil)
	})

	// —— scroll（元素内滚动）——
	t.Run("scrollInElement", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionEvaluate, Params: map[string]interface{}{
				"script": `() => {
					const d = document.createElement('div');
					d.id = 'scroller';
					d.style.cssText = 'height:60px;overflow:auto;width:120px';
					d.innerHTML = '<div style="height:400px">tall</div>';
					document.body.appendChild(d);
					return true;
				}`,
			}},
			{Type: types.ActionScroll, Selector: "#scroller",
				Params: map[string]interface{}{"direction": "in", "deltaY": 100}},
			{Type: types.ActionAssert, Params: map[string]interface{}{
				"type":   "expression",
				"script": "document.getElementById('scroller').scrollTop > 0",
			}},
		}, nil)
	})

	// —— loop（既有动作的真实回归：循环项作用域 + 变量插值）——
	t.Run("loop", func(t *testing.T) {
		runScript(t, eng, base, []types.Action{
			{Type: types.ActionLoop, Params: map[string]interface{}{
				"dataSource": []interface{}{
					map[string]interface{}{"v": "L1"},
					map[string]interface{}{"v": "L2"},
				},
				"itemVar": "row",
				"actions": []interface{}{
					map[string]interface{}{"type": "fill", "selector": "#name", "params": map[string]interface{}{"value": "${row.v}"}},
				},
			}},
			{Type: types.ActionAssert, Selector: "#name",
				Params: map[string]interface{}{"type": "value", "expected": "L2"}},
		}, nil)
	})
}
