package executor

import (
	"context"
	"encoding/json"
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
		return nil, types.NewNonRetryableError("上传步骤没有设置要上传的文件")
	case string:
		if strings.TrimSpace(files) == "" {
			return nil, types.NewNonRetryableError("上传步骤没有设置要上传的文件")
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
				return nil, types.NewNonRetryableError("上传步骤的文件列表格式不正确")
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return files, nil
	default:
		return nil, types.NewNonRetryableError("上传步骤的文件参数类型不正确")
	}
}

// executeUpload ENGINE-02：upload → Element.SetFiles。
func (e *Engine) executeUpload(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return types.NewNonRetryableError("上传步骤没有设置目标（缺少选择器）")
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
			return types.NewNonRetryableError("下载步骤的 url 参数格式不正确")
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
//
// W7.1 / 缺陷 A 同族（第三层）：历史脚本的 script 动作常用「多语句 + 分号
// 结尾」的写法（生产库实证：综合端到端业务流程测试 的 params.code 里有
// `console.log('✓ 搜索框存在');` 这类语句，且形参被 {{var}} 替换成值）。
// ChromePage.Evaluate 的回退路径会把整段包成 `() => { return ( <script> ); }`，
// 多语句 + 分号落进括号里就变成语法错误（TypeError: Cannot read properties
// of undefined (reading 'apply') / eval js error）。
//
// 处置：识别「语句体」（含 console.log / return / 分号或换行分隔的赋值）时，
// 用 `() => { <script> }` 包成函数体执行，不用 return 包裹。两条路径都以
// 「返回值」收尾（语句体最后一行的表达式值不保证可返回，此时就是 undefined，
// 与历史脚本的意图一致：script 动作多为日志/副作用）。
func (e *Engine) executeEvaluate(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	script := ""
	if s, ok := action.Params["script"].(string); ok && s != "" {
		script = s
	} else if action.Value != "" {
		script = action.Value
	}
	if script == "" {
		return types.NewNonRetryableError("执行脚本步骤没有写脚本内容（缺少 script 参数）")
	}

	val, err := evaluateScriptBody(pageMgr, script)
	if err != nil {
		return fmt.Errorf("evaluate 执行失败: %w", err)
	}

	targetVar := "result"
	if tv, ok := action.Params["targetVar"].(string); ok && tv != "" {
		targetVar = tv
	}
	// 语句体执行没有可回写的值时不污染变量（历史脚本本就没打算取返回值）。
	if val != nil {
		if err := scope.Set(ScopeTask, targetVar, val); err != nil {
			return fmt.Errorf("evaluate 返回值回写变量失败: %w", err)
		}
	}

	e.logger.Debug("evaluate 结果",
		logger.String("target_var", targetVar),
		logger.String("value", fmt.Sprintf("%v", MaskSensitive(targetVar, val))))
	return nil
}

// evaluateScriptBody 按脚本形态选择求值方式（单表达式 vs 语句体）。
//
// 判定「语句体」的保守规则（任一即视为语句体）：
//   - 含分号（多语句结束符；单表达式的 `a;b` 极少见且多半是历史写法）
//   - 含换行且不是单行三元/箭头函数
//   - 以 console./return/let/var/const/if/for/while 开头
//
// 单表达式仍走既有路径（保持原语义：返回值可直接入变量）。
func evaluateScriptBody(pageMgr *browser.PageManager, script string) (interface{}, error) {
	if !isStatementBody(script) {
		return pageMgr.Evaluate(script)
	}
	body := fmt.Sprintf("() => { %s }", script)
	return pageMgr.Evaluate(body)
}

// isStatementBody 判定脚本是否为「语句体」（见 evaluateScriptBody）。
func isStatementBody(script string) bool {
	t := strings.TrimSpace(script)
	if t == "" {
		return false
	}
	// ① 已经是完整表达式/函数形体 → 原样求值（**必须在语句关键字判断之前**）。
	//
	// 反例（真跑回归踩到）：ENGINE-02 的 scrollInElement 用例用多行箭头函数
	// 建 DOM，`() => { const d = ...; document.body.appendChild(d); return true; }`。
	// 它含换行与分号，若被判成语句体再包一层 `() => { … }`，就变成「创建了一个
	// 函数却没调用」——副作用全不发生，后续步骤找不到元素。HEAD 基线能过，
	// 就是因为当时没有这层包裹。
	for _, prefix := range []string{"() =>", "()=>", "async ", "function", "(function", "(() =>"} {
		if strings.HasPrefix(t, prefix) {
			return false
		}
	}
	// ② 以 `(` 开头（IIFE 或括号表达式）也视为表达式：如 (function(){...})()。
	if strings.HasPrefix(t, "(") {
		return false
	}
	// ③ 单表达式特征：三元/箭头单行、字面量、明确的 `return ...` 之外的运算符表达式。
	//    只在这些**明显是语句**的形态下返回 true，其余保持原语义。
	for _, kw := range []string{"console.", "let ", "var ", "const ", "if ", "for ", "while ", "try "} {
		if strings.HasPrefix(t, kw) {
			return true
		}
	}
	// ④ 含分号（多语句结束符）→ 语句体。
	if strings.Contains(t, ";") {
		return true
	}
	return false
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
		return types.NewNonRetryableErrorf("关闭步骤的目标「%s」不被支持（可选 page/tab/current）", target)
	}
}

// executeHover ENGINE-02：hover → Element.Hover。
func (e *Engine) executeHover(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return types.NewNonRetryableError("悬停步骤没有设置目标（缺少选择器）")
	}
	return pageMgr.Hover(action.Selector)
}

// executeDoubleClick ENGINE-02：doubleClick → Element.Click(left, 2)。
func (e *Engine) executeDoubleClick(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return types.NewNonRetryableError("双击步骤没有设置目标（缺少选择器）")
	}
	return pageMgr.DoubleClick(action.Selector)
}

// executeRightClick ENGINE-02：rightClick → Element.Click(right, 1)。
func (e *Engine) executeRightClick(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, scope *VariableScope) error {

	if action.Selector == "" {
		return types.NewNonRetryableError("右键步骤没有设置目标（缺少选择器）")
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
		return types.NewNonRetryableError("拖拽步骤需要同时设置起点与终点两个选择器")
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
		return types.NewNonRetryableError("按键步骤没有设置要按的键（缺少 keys 参数）")
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
			return types.NewNonRetryableErrorf("切换页签的序号「%v」不是整数", v)
		}
		index = i
	}
	if index < 0 {
		return types.NewNonRetryableError("切换页签步骤没有设置页签序号（缺少 index 参数）")
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
			return types.NewNonRetryableError("进入内嵌页面步骤没有设置目标（缺少选择器）")
		}
		return pageMgr.EnterFrame(action.Selector)
	case "exit":
		return pageMgr.ExitFrame()
	default:
		return types.NewNonRetryableErrorf("内嵌页面的操作「%s」不被支持（可选 enter/exit）", mode)
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
func (e *Engine) executeCondition(ctx context.Context, page browser.Page, pageMgr *browser.PageManager,
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
			return types.NewNonRetryableError("条件判断步骤没有写判断表达式（缺少 expression 参数）")
		}
		// ============================================================
		// W7.1 / 缺陷 A 同族：历史脚本的 condition 取值不是 JS 表达式，而是
		// 「元素状态断言」的枚举（生产库实证：条件判断与分支测试 / 综合端到端
		// 业务流程测试 两个 goto 任务都写 condition="exists" + selector）。
		//
		// 修复前：这里把 "exists" 原样拼进 Boolean(exists) → ReferenceError:
		// exists is not defined，任务在 step 3 就失败。
		//
		// 处置：先按枚举识别（元素状态），识别不了才当 JS 表达式求值。
		// 两条都是「条件判断」，语义分明，不会把写错的 JS 静默当成 exists。
		// ============================================================
		if state, isElementState := elementStateCondition(expr); isElementState {
			sel := action.Selector
			if sel == "" {
				sel = action.Params["selector"].(string) // 兼容 params 里给选择器
			}
			if sel == "" {
				return types.NewNonRetryableErrorf(
					"条件判断的「%s」需要设置要检查的元素（缺少选择器）", expr)
			}
			truthy = e.elementStateMatches(pageMgr, sel, state, action)
		} else {
			script := fmt.Sprintf("(function(){ return Boolean(%s); })()", expr)
			val, err := pageMgr.Evaluate(script)
			if err != nil {
				return fmt.Errorf("condition 表达式求值失败: %w", err)
			}
			truthy = isTruthy(val)
		}

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
		return types.NewNonRetryableErrorf("条件判断的方式「%s」不被支持（可选 js/variable）", mode)
	}

	branchKey := "else"
	if truthy {
		branchKey = "then"
	}
	// W7.1 / 缺陷 A 同族：历史脚本的分支键名是 trueActions/falseActions
	// （生产库实证：条件判断与分支测试 / 综合端到端 两个 goto 任务都这么写）。
	// 修复前只认 then/else，两段分支都读成空 → 条件判断形同虚设（静默跑分支 0 步）。
	nested := nestedActionsFrom(action.Params[branchKey])
	if len(nested) == 0 {
		if truthy {
			nested = nestedActionsFrom(action.Params["trueActions"])
		} else {
			nested = nestedActionsFrom(action.Params["falseActions"])
		}
	}
	e.logger.Debug("condition 分支",
		logger.Any("truthy", truthy),
		logger.Int("actions", len(nested)))

	for i := range nested {
		if err := e.executeAction(ctx, page, pageMgr, &nested[i], scope); err != nil {
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
			return types.NewNonRetryableError("断言步骤没有设置目标（缺少选择器）")
		}
		v, err := pageMgr.GetText(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取文本失败: %w", err)
		}
		actual = v
	case "value":
		if action.Selector == "" {
			return types.NewNonRetryableError("断言步骤没有设置目标（缺少选择器）")
		}
		v, err := pageMgr.GetValue(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取值失败: %w", err)
		}
		actual = v
	case "html":
		if action.Selector == "" {
			return types.NewNonRetryableError("断言步骤没有设置目标（缺少选择器）")
		}
		v, err := pageMgr.GetHTML(action.Selector)
		if err != nil {
			return fmt.Errorf("assert 取 HTML 失败: %w", err)
		}
		actual = v
	case "attribute":
		if action.Selector == "" {
			return types.NewNonRetryableError("断言步骤没有设置目标（缺少选择器）")
		}
		attr, _ := action.Params["attribute"].(string)
		if attr == "" {
			return types.NewNonRetryableError("断言属性时没有指定属性名（缺少 attribute 参数）")
		}
		v, err := pageMgr.GetAttribute(action.Selector, attr)
		if err != nil {
			return fmt.Errorf("assert 取属性失败: %w", err)
		}
		actual = v
	case "expression":
		script, _ := action.Params["script"].(string)
		if script == "" {
			return types.NewNonRetryableError("断言步骤没有写断言表达式（缺少 script 参数）")
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
			return types.NewNonRetryableError("断言元素存在时没有设置目标（缺少选择器）")
		}
		if err := pageMgr.WaitFor(sel, actionTimeoutOr(e, action)); err != nil {
			return fmt.Errorf("断言失败: 元素不存在 %s: %w", sel, err)
		}
		return nil
	default:
		return types.NewNonRetryableErrorf("断言的类型「%s」不被支持", assertType)
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

// ============================================================================
// W7.1 / 缺陷 A 同族：condition 的「元素状态」写法（历史脚本惯用）
//
// 生产库实证（scripts/ops/rpa_goto_task_dump 导出的 6 个任务里）：
//   · 条件判断与分支测试：{"type":"condition","selector":"#kw","params":{"condition":"exists", ...}}
//   · 综合端到端业务流程测试：同上
// 修复前把 "exists" 当成 JS 表达式拼进 Boolean(exists) → ReferenceError。
//
// 白名单必须**保守**：只有明确是「元素状态枚举」的字符串才走这条路，
// 其余一律当 JS 表达式求值（不能把写错的 JS 静默当成 exists —— 那会把
// 一个真实的配置错误变成「永远走 else 分支」的静默错误）。
// ============================================================================

// elementStateConditions condition 允许的元素状态枚举（小写）。
var elementStateConditions = map[string]bool{
	"exists":      true,
	"notexists":   true,
	"not-exists":  true,
	"absent":      true,
	"visible":     true,
	"hidden":      true,
	"notvisible":  true,
	"not-visible": true,
	"invisible":   true,
	"enabled":     true,
	"disabled":    true,
	"checked":     true,
	"unchecked":   true,
	"empty":       true,
	"notempty":    true,
	"not-empty":   true,
}

// elementStateCondition 判定字符串是否是元素状态枚举（不是 JS 表达式）。
func elementStateCondition(expr string) (state string, ok bool) {
	normalized := strings.ToLower(strings.TrimSpace(expr))
	if normalized == "" {
		return "", false
	}
	// 带空格/运算符号的一律是 JS 表达式（避免把 "exists && foo" 误判成枚举）。
	if strings.ContainsAny(normalized, " ()+-*/&|=<>%!") {
		return "", false
	}
	if elementStateConditions[normalized] {
		return normalized, true
	}
	return "", false
}

// elementStateMatches 在页面上真实检查元素状态（返回是否命中）。
//
// 实现走一次 JS 求值：先 querySelector，失败再按 XPath 兜底（存量脚本两种
// 选择器都有）。求值失败视为「不命中」并留 Debug 日志 —— 条件判断不该因为
// 一次求值异常把整条任务判成失败，但也不能悄悄当成 true。
func (e *Engine) elementStateMatches(pageMgr *browser.PageManager, selector, state string, action *types.Action) bool {
	// selector 用 JSON 编码嵌入脚本，避免引号/反斜杠破坏脚本。
	selJSON, err := json.Marshal(selector)
	if err != nil {
		return false
	}
	stateJSON, _ := json.Marshal(state)

	script := fmt.Sprintf(`(function(){
		var sel = %s, state = %s;
		var el = null;
		try { el = document.querySelector(sel); } catch (e) { el = null; }
		if (!el) {
			try {
				var r = document.evaluate(sel, document, null,
					XPathResult.FIRST_ORDERED_NODE_TYPE, null);
				el = r ? r.singleNodeValue : null;
			} catch (e) { el = null; }
		}
		var exists = !!el;
		switch (state) {
		case "exists": return exists;
		case "notexists": case "not-exists": case "absent": return !exists;
		case "visible":
			if (!exists) return false;
			var r0 = el.getBoundingClientRect();
			var st = window.getComputedStyle(el);
			return r0.width > 0 && r0.height > 0 && st.visibility !== "hidden" && st.display !== "none";
		case "hidden": case "notvisible": case "not-visible": case "invisible":
			if (!exists) return true;
			var r1 = el.getBoundingClientRect();
			var st1 = window.getComputedStyle(el);
			return r1.width === 0 || r1.height === 0 || st1.visibility === "hidden" || st1.display === "none";
		case "enabled": return exists && !el.disabled;
		case "disabled": return exists && !!el.disabled;
		case "checked": return exists && !!el.checked;
		case "unchecked": return exists && !el.checked;
		case "empty": return exists && ((el.value || "").length === 0 && (el.textContent || "").trim() === "");
		case "notempty": case "not-empty":
			return exists && ((el.value || "").length > 0 || (el.textContent || "").trim().length > 0);
		}
		return false;
	})()`, string(selJSON), string(stateJSON))

	val, err := pageMgr.Evaluate(script)
	if err != nil {
		e.logger.Debug("condition 元素状态求值失败（按未命中处理）",
			logger.String("selector", selector),
			logger.String("state", state),
			logger.Err(err))
		return false
	}
	matched := isTruthy(val)
	e.logger.Debug("condition 元素状态",
		logger.String("selector", selector),
		logger.String("state", state),
		logger.Any("matched", matched))
	return matched
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
			return false, types.NewNonRetryableErrorf("数值比较的左右操作数不是数字：%q %s %q", ls, op, rs)
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
		return false, types.NewNonRetryableErrorf("比较运算符「%s」不被支持", op)
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
