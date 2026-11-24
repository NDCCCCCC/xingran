package executor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 154 / ENGINE-02：动作全集实现
//
// 本文件承载 executeAction switch 里「差集 A（常量已声明、switch 无 case）」
// 与「差集 B（常量与 case 双缺）」的动作实现体。switch 分支本身仍在
// engine.go（覆盖率静态断言直接扫那个 switch），实现细节集中在这里，
// 避免 engine.go 无限膨胀。
// ============================================================================

// actionFileList 归一化 upload 的 files 参数（string / []interface{} / []string）。
func actionFileList(v interface{}) ([]string, error) {
	switch files := v.(type) {
	case nil:
		return nil, fmt.Errorf("upload 缺少 files 参数")
	case string:
		if strings.TrimSpace(files) == "" {
			return nil, fmt.Errorf("upload 缺少 files 参数")
		}
		parts := strings.Split(files, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	case []interface{}:
		out := make([]string, 0, len(files))
		for _, item := range files {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("upload files 元素非字符串: %T", item)
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return files, nil
	default:
		return nil, fmt.Errorf("upload files 类型不支持: %T", v)
	}
}

// executeUpload ENGINE-02：upload → Element.SetFiles。
func (e *Engine) executeUpload(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return fmt.Errorf("upload 缺少 selector")
	}
	files, err := actionFileList(action.Params["files"])
	if err != nil {
		return err
	}
	return pageMgr.Upload(action.Selector, files)
}

// executeDownload ENGINE-02：download → Browser.WaitDownload(dir) + 产物归档。
//
// 参数：action.Selector 非空时点击该元素触发下载；否则 params["url"] 导航；
// 都缺省时假定下载已由前序动作发起（直接等待产物）。dir 缺省取
// worker.downloads_dir，产物路径回写 targetVar（默认 downloadPath）。
func (e *Engine) executeDownload(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	dir := ""
	if d, ok := action.Params["dir"].(string); ok {
		dir = d
	}
	if dir == "" && e.workerConfig != nil {
		dir = e.workerConfig.DownloadsDir
	}
	if dir == "" {
		dir = "./downloads"
	}

	var trigger func() error
	switch {
	case action.Selector != "":
		trigger = func() error { return pageMgr.Click(action.Selector) }
	case action.Params["url"] != nil:
		url, ok := action.Params["url"].(string)
		if !ok {
			return fmt.Errorf("download url 参数非字符串")
		}
		trigger = func() error { return pageMgr.Goto(url) }
	default:
		trigger = nil
	}

	path, err := pageMgr.Download(ctx, dir, trigger)
	if err != nil {
		return fmt.Errorf("download 失败: %w", err)
	}

	targetVar := "downloadPath"
	if tv, ok := action.Params["targetVar"].(string); ok && tv != "" {
		targetVar = tv
	}
	if err := scope.Set(ScopeTask, targetVar, path); err != nil {
		return fmt.Errorf("download 产物路径回写变量失败: %w", err)
	}
	e.logger.Info("下载完成", logger.String("path", path))
	return nil
}

// executeEvaluate ENGINE-02：evaluate → PageManager.Evaluate 返回值入变量。
func (e *Engine) executeEvaluate(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	script := ""
	if s, ok := action.Params["script"].(string); ok && s != "" {
		script = s
	} else if action.Value != "" {
		script = action.Value
	}
	if script == "" {
		return fmt.Errorf("evaluate 缺少 script 参数")
	}

	val, err := pageMgr.Evaluate(script)
	if err != nil {
		return fmt.Errorf("evaluate 执行失败: %w", err)
	}

	targetVar := "result"
	if tv, ok := action.Params["targetVar"].(string); ok && tv != "" {
		targetVar = tv
	}
	if err := scope.Set(ScopeTask, targetVar, val); err != nil {
		return fmt.Errorf("evaluate 返回值回写变量失败: %w", err)
	}

	e.logger.Debug("evaluate 结果",
		logger.String("target_var", targetVar),
		logger.String("value", fmt.Sprintf("%v", MaskSensitive(targetVar, val))))
	return nil
}

// executeClose ENGINE-02：close → 关闭当前 page（最后一个页签时新建空白页兜底）。
func (e *Engine) executeClose(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	target := "page"
	if t, ok := action.Params["target"].(string); ok && t != "" {
		target = t
	}
	switch target {
	case "page", "tab", "current":
		return pageMgr.CloseCurrentTab()
	default:
		return fmt.Errorf("close 不支持的 target: %s（允许 page|tab|current）", target)
	}
}

// executeHover ENGINE-02：hover → Element.Hover。
func (e *Engine) executeHover(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return fmt.Errorf("hover 缺少 selector")
	}
	return pageMgr.Hover(action.Selector)
}

// executeDoubleClick ENGINE-02：doubleClick → Element.Click(left, 2)。
func (e *Engine) executeDoubleClick(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return fmt.Errorf("doubleClick 缺少 selector")
	}
	return pageMgr.DoubleClick(action.Selector)
}

// executeRightClick ENGINE-02：rightClick → Element.Click(right, 1)。
func (e *Engine) executeRightClick(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return fmt.Errorf("rightClick 缺少 selector")
	}
	return pageMgr.RightClick(action.Selector)
}

// executeDrag ENGINE-02：drag → HTML5 兼容（CDP 原生优先 + JS DataTransfer 兜底）。
func (e *Engine) executeDrag(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	from := action.Selector
	if from == "" {
		if s, ok := action.Params["from"].(string); ok {
			from = s
		}
	}
	to, _ := action.Params["to"].(string)
	if to == "" {
		if s, ok := action.Params["targetSelector"].(string); ok {
			to = s
		}
	}
	if from == "" || to == "" {
		return fmt.Errorf("drag 需要 from 与 to 两个选择器")
	}
	return pageMgr.Drag(from, to)
}

// executeKeyboard ENGINE-02：keyboard → 组合键（page.Keyboard.Press）。
func (e *Engine) executeKeyboard(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	keys := ""
	if s, ok := action.Params["keys"].(string); ok && s != "" {
		keys = s
	} else if action.Value != "" {
		keys = action.Value
	}
	if keys == "" {
		return fmt.Errorf("keyboard 缺少 keys 参数")
	}
	return pageMgr.PressKeys(keys)
}

// executeSwitchTab ENGINE-02：switchTab → browser.Pages() 切换。
func (e *Engine) executeSwitchTab(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	index := -1
	switch v := action.Params["index"].(type) {
	case int:
		index = v
	case float64:
		index = int(v)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("switchTab index 非整数: %q", v)
		}
		index = i
	}
	if index < 0 {
		return fmt.Errorf("switchTab 缺少 index 参数")
	}
	return pageMgr.SwitchTab(index)
}

// executeIframe ENGINE-02：iframe → 进入 / 退出 frame（params.action=enter|exit）。
func (e *Engine) executeIframe(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	mode := "enter"
	if m, ok := action.Params["action"].(string); ok && m != "" {
		mode = m
	}
	switch mode {
	case "enter":
		if action.Selector == "" {
			return fmt.Errorf("iframe(enter) 缺少 selector")
		}
		return pageMgr.EnterFrame(action.Selector)
	case "exit":
		return pageMgr.ExitFrame()
	default:
		return fmt.Errorf("iframe 不支持 action: %s（允许 enter|exit）", mode)
	}
}

// executeScript ENGINE-02：script → 等价 evaluate（前端 ACTION_TYPE_OPTIONS
// 早已暴露 "script"，此处补执行体，避免同一语义两个名字）。
func (e *Engine) executeScript(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {
	return e.executeEvaluate(ctx, pageMgr, action, scope)
}

// executeCondition ENGINE-02：condition → 真条件分支（嵌套动作执行）。
//
// 参数：mode=js（默认）时用 expression/condition 的 JS 表达式；
// mode=variable 时用 left/op/right（op: eq|ne|gt|ge|lt|le|contains|empty）。
// then / else 是嵌套动作数组，按分支执行。
func (e *Engine) executeCondition(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	mode := "js"
	if m, ok := action.Params["mode"].(string); ok && m != "" {
		mode = m
	}

	var truthy bool
	switch mode {
	case "js":
		expr := ""
		if s, ok := action.Params["expression"].(string); ok && s != "" {
			expr = s
		} else if s, ok := action.Params["condition"].(string); ok && s != "" {
			expr = s
		}
		if expr == "" {
			return fmt.Errorf("condition(js) 缺少 expression 参数")
		}
		script := fmt.Sprintf("(function(){ return Boolean(%s); })()", expr)
		val, err := pageMgr.Evaluate(script)
		if err != nil {
			return fmt.Errorf("condition 表达式求值失败: %w", err)
		}
		truthy = isTruthy(val)

	case "variable":
		left := action.Params["left"]
		op, _ := action.Params["op"].(string)
		if op == "" {
			op = "eq"
		}
		if s, ok := left.(string); ok {
			if v, _, found := scope.Lookup(s); found {
				left = v
			}
		}
		right := action.Params["right"]
		if s, ok := right.(string); ok {
			if v, _, found := scope.Lookup(s); found {
				right = v
			}
		}
		res, err := compareValues(left, op, right)
		if err != nil {
			return fmt.Errorf("condition(variable) 比较失败: %w", err)
		}
		truthy = res

	default:
		return fmt.Errorf("condition 不支持的 mode: %s（允许 js|variable）", mode)
	}

	branchKey := "else"
	if truthy {
		branchKey = "then"
	}
	nested := nestedActionsFrom(action.Params[branchKey])
	e.logger.Debug("condition 分支",
		logger.Any("truthy", truthy),
		logger.Int("actions", len(nested)))

	for i := range nested {
		if err := e.executeAction(ctx, pageMgr, &nested[i], scope); err != nil {
			return fmt.Errorf("condition %s 分支第 %d 个动作失败: %w", branchKey, i+1, err)
		}
	}
	return nil
}

// executeAssert ENGINE-02：assert → 断言，失败即终止（返回 error 由上层短路）。
//
// 参数：type=text（默认）|value|html|attribute|expression|exists；
// expected 期望值；contains=true 用「包含」而非「相等」；
// attribute 供 type=attribute；script 供 type=expression。
func (e *Engine) executeAssert(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	assertType := "text"
	if t, ok := action.Params["type"].(string); ok && t != "" {
		assertType = t
	}
	expected, _ := action.Params["expected"].(string)
	contains := false
	if c, ok := action.Params["contains"].(bool); ok {
		contains = c
	}

	var actual string
	switch assertType {
	case "text":
		if action.Selector == "" {
			return fmt.Errorf("assert(text) 缺少 selector")
		}
		v, err := pageMgr.GetText(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取文本失败: %w", err)
		}
		actual = v
	case "value":
		if action.Selector == "" {
			return fmt.Errorf("assert(value) 缺少 selector")
		}
		v, err := pageMgr.GetValue(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取值失败: %w", err)
		}
		actual = v
	case "html":
		if action.Selector == "" {
			return fmt.Errorf("assert(html) 缺少 selector")
		}
		v, err := pageMgr.GetHTML(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取 HTML 失败: %w", err)
		}
		actual = v
	case "attribute":
		if action.Selector == "" {
			return fmt.Errorf("assert(attribute) 缺少 selector")
		}
		attr, _ := action.Params["attribute"].(string)
		if attr == "" {
			return fmt.Errorf("assert(attribute) 缺少 attribute 参数")
		}
		v, err := pageMgr.GetAttribute(action.Selector, attr)
		if err != nil {
			return fmt.Errorf("assert 取属性失败: %w", err)
		}
		actual = v
	case "expression":
		script, _ := action.Params["script"].(string)
		if script == "" {
			return fmt.Errorf("assert(expression) 缺少 script 参数")
		}
		val, err := pageMgr.Evaluate(fmt.Sprintf("(function(){ return Boolean(%s); })()", script))
		if err != nil {
			return fmt.Errorf("assert 表达式求值失败: %w", err)
		}
		if !isTruthy(val) {
			return fmt.Errorf("断言失败: 表达式 %q 为假", script)
		}
		return nil
	case "exists":
		sel := action.Selector
		if sel == "" {
			sel, _ = action.Params["selector"].(string)
		}
		if sel == "" {
			return fmt.Errorf("assert(exists) 缺少 selector")
		}
		if err := pageMgr.WaitFor(sel, actionTimeoutOr(e, action)); err != nil {
			return fmt.Errorf("断言失败: 元素不存在 %s: %w", sel, err)
		}
		return nil
	default:
		return fmt.Errorf("assert 不支持的 type: %s", assertType)
	}

	ok := actual == expected
	if contains {
		ok = strings.Contains(actual, expected)
	}
	if !ok {
		if contains {
			return fmt.Errorf("断言失败(%s): 实际 %q 不包含 %q", assertType, actual, expected)
		}
		return fmt.Errorf("断言失败(%s): 实际 %q != 期望 %q", assertType, actual, expected)
	}
	e.logger.Debug("断言通过", logger.String("type", assertType))
	return nil
}

// actionTimeoutOr 取动作超时（0 时回落到 worker.action_timeout，再 30s）。
func actionTimeoutOr(e *Engine, action *types.Action) time.Duration {
	if action.Timeout > 0 {
		return time.Duration(action.Timeout) * time.Millisecond
	}
	if e.workerConfig != nil && e.workerConfig.ActionTimeout > 0 {
		return e.workerConfig.ActionTimeout
	}
	return 30 * time.Second
}

// compareValues 比较两个值（condition(variable) 用）。
func compareValues(left interface{}, op string, right interface{}) (bool, error) {
	if op == "empty" {
		return isEmptyValue(left), nil
	}
	ls := fmt.Sprintf("%v", left)
	rs := fmt.Sprintf("%v", right)

	switch op {
	case "eq":
		return ls == rs, nil
	case "ne":
		return ls != rs, nil
	case "contains":
		return strings.Contains(ls, rs), nil
	case "gt", "ge", "lt", "le":
		lf, err1 := strconv.ParseFloat(strings.TrimSpace(ls), 64)
		rf, err2 := strconv.ParseFloat(strings.TrimSpace(rs), 64)
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("数值比较需要数字: %q %s %q", ls, op, rs)
		}
		switch op {
		case "gt":
			return lf > rf, nil
		case "ge":
			return lf >= rf, nil
		case "lt":
			return lf < rf, nil
		default:
			return lf <= rf, nil
		}
	default:
		return false, fmt.Errorf("不支持的比较运算符: %s", op)
	}
}

// isEmptyValue 空值判定（nil / "" / 空切片 / 空 map / false / 0）。
func isEmptyValue(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return val == ""
	case bool:
		return !val
	case int:
		return val == 0
	case int64:
		return val == 0
	case float64:
		return val == 0
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	default:
		return false
	}
}

// isTruthy JS 求值结果的 Go 侧真值判定。
func isTruthy(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return false
	case bool:
		return val
	case string:
		return val != "" && val != "0" && strings.ToLower(val) != "false"
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	default:
		return v != nil
	}
}

// nestedActionsFrom 把 params 里的动作数组归一化为 []types.Action。
func nestedActionsFrom(raw interface{}) []types.Action {
	switch v := raw.(type) {
	case nil:
		return nil
	case []types.Action:
		return v
	case []interface{}:
		out := make([]types.Action, 0, len(v))
		for _, item := range v {
			switch a := item.(type) {
			case types.Action:
				out = append(out, a)
			case map[string]interface{}:
				var tmp Engine
				out = append(out, tmp.mapToAction(a))
			}
		}
		return out
	default:
		return nil
	}
}
