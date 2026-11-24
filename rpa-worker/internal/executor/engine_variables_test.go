package executor

import (
	"strings"
	"testing"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 154 / ENGINE-04 轻量单测（-short 也跑，无 Chromium 依赖）
//   变量三层作用域 / 嵌套路径 / 未定义策略 / extract 嵌套写回 / 敏感脱敏
// ============================================================================

// TestParsePath 嵌套路径解析。
func TestParsePath(t *testing.T) {
	cases := []struct {
		path      string
		wantKeys  []string
		wantIdx   []int
		expectErr bool
	}{
		{"a", []string{"a"}, nil, false},
		{"a.b", []string{"a", "b"}, nil, false},
		{"a.b[0].c", []string{"a", "b", "c"}, []int{0}, false},
		{"rows[2].name", []string{"rows", "name"}, []int{2}, false},
		{"a[0][1]", []string{"a"}, []int{0, 1}, false},
		{"", nil, nil, true},
		{"a.", nil, nil, true},
		{"a[x]", nil, nil, true},
	}
	for _, tc := range cases {
		segs, err := parsePath(tc.path)
		if tc.expectErr {
			if err == nil {
				t.Errorf("parsePath(%q) 应报错", tc.path)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePath(%q) 意外报错: %v", tc.path, err)
			continue
		}
		var keys []string
		var idxs []int
		for _, s := range segs {
			if s.isIdx {
				idxs = append(idxs, s.index)
			} else {
				keys = append(keys, s.key)
			}
		}
		if strings.Join(keys, ",") != strings.Join(tc.wantKeys, ",") {
			t.Errorf("parsePath(%q) keys=%v，期望 %v", tc.path, keys, tc.wantKeys)
		}
		if len(idxs) != len(tc.wantIdx) {
			t.Errorf("parsePath(%q) 下标数=%d，期望 %d", tc.path, len(idxs), len(tc.wantIdx))
		}
	}
}

// TestResolveNestedPath 嵌套取值 + 数组下标 + 负下标。
func TestResolveNestedPath(t *testing.T) {
	vars := map[string]interface{}{
		"order": map[string]interface{}{
			"id": "A-1",
			"items": []interface{}{
				map[string]interface{}{"name": "first", "qty": 2},
				map[string]interface{}{"name": "second", "qty": 3},
			},
		},
		"flat": "v",
	}
	cases := []struct {
		path string
		want interface{}
	}{
		{"flat", "v"},
		{"order.id", "A-1"},
		{"order.items[0].name", "first"},
		{"order.items[1].qty", 3},
		{"order.items[-1].name", "second"},
	}
	for _, tc := range cases {
		got, ok := ResolvePath(vars, tc.path)
		if !ok {
			t.Errorf("ResolvePath(%q) 未命中", tc.path)
			continue
		}
		if got != tc.want {
			t.Errorf("ResolvePath(%q)=%v，期望 %v", tc.path, got, tc.want)
		}
	}
	if _, ok := ResolvePath(vars, "order.items[9].name"); ok {
		t.Error("越界下标必须返回未命中")
	}
	if _, ok := ResolvePath(vars, "missing.deep"); ok {
		t.Error("缺失路径必须返回未命中")
	}
}

// TestResolveTemplate 扁平 + 嵌套 + 默认值 + 策略。
func TestResolveTemplate(t *testing.T) {
	vars := map[string]interface{}{
		"name": "张三",
		"a":    map[string]interface{}{"b": []interface{}{"x", "y"}},
	}

	got, err := ResolveTemplate("你好 ${name}，第一项 ${a.b[0]}", vars, config.UndefinedVariableEmpty)
	if err != nil {
		t.Fatalf("空串策略不应报错: %v", err)
	}
	if got != "你好 张三，第一项 x" {
		t.Fatalf("模板替换结果=%q", got)
	}

	// 未定义 + empty → 空串
	got, err = ResolveTemplate("前缀[${missing}]", vars, config.UndefinedVariableEmpty)
	if err != nil || got != "前缀[]" {
		t.Fatalf("empty 策略应替换为空串: %q %v", got, err)
	}

	// 未定义 + strict → 错误且错误里带变量名
	_, err = ResolveTemplate("前缀[${missing}]", vars, config.UndefinedVariableStrict)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("strict 策略必须报错且指明变量名: %v", err)
	}

	// 默认值语法 ${a.b[9]:fallback}
	got, err = ResolveTemplate("${a.b[9]:fallback}", vars, config.UndefinedVariableStrict)
	if err != nil || got != "fallback" {
		t.Fatalf("默认值语法失败: %q %v", got, err)
	}
}

// TestResolveValueRecursive 参数树递归求值（嵌套 map/数组）。
func TestResolveValueRecursive(t *testing.T) {
	vars := map[string]interface{}{"host": "example.com", "n": 3}
	in := map[string]interface{}{
		"url":  "https://${host}/api",
		"list": []interface{}{"${host}", "static"},
		"nested": map[string]interface{}{
			"deep": "n=${n}",
		},
		"keep": true,
	}
	out, err := ResolveValue(in, vars, config.UndefinedVariableEmpty)
	if err != nil {
		t.Fatalf("递归求值失败: %v", err)
	}
	m := out.(map[string]interface{})
	if m["url"] != "https://example.com/api" {
		t.Fatalf("url=%v", m["url"])
	}
	if list, ok := m["list"].([]interface{}); !ok || list[0] != "example.com" {
		t.Fatalf("list=%v", m["list"])
	}
	if nested := m["nested"].(map[string]interface{}); nested["deep"] != "n=3" {
		t.Fatalf("nested=%v", nested)
	}
	if m["keep"] != true {
		t.Fatalf("非字符串值应原样保留: %v", m["keep"])
	}
}

// TestVariableScopeLayering 三层作用域：窄层优先、写回只落指定层。
func TestVariableScopeLayering(t *testing.T) {
	vs := NewVariableScope(map[string]interface{}{
		"v":             "task",
		"__credentials": map[string]interface{}{"username": "u", "password": "p"},
	})

	if got, scope, ok := vs.Lookup("v"); !ok || got != "task" || scope != ScopeTask {
		t.Fatalf("任务层查找失败: %v %v %v", got, scope, ok)
	}
	if err := vs.Set(ScopeLoop, "v", "loop"); err != nil {
		t.Fatal(err)
	}
	if got, scope, _ := vs.Lookup("v"); got != "loop" || scope != ScopeLoop {
		t.Fatalf("循环层必须优先: %v %v", got, scope)
	}
	vs.BreakLoopScope()
	if got, scope, _ := vs.Lookup("v"); got != "task" || scope != ScopeTask {
		t.Fatalf("BreakLoopScope 后应回落任务层: %v %v", got, scope)
	}

	// 敏感变量进会话层
	if _, scope, ok := vs.Lookup("__credentials"); !ok || scope != ScopeSession {
		// __credentials 同时存在于 Task 层（兼容读），但必须能在 Session 层取到
		t.Logf("__credentials 查找层级=%v（需同时存在于会话层）", scope)
	}
	if _, ok := vs.Session["__credentials"]; !ok {
		t.Fatal("敏感变量必须落会话层（跨任务复用）")
	}
}

// TestExtractNestedWriteBack extract 结果支持嵌套路径回写（ENGINE-04）。
func TestExtractNestedWriteBack(t *testing.T) {
	vs := NewVariableScope(map[string]interface{}{
		"rows": []interface{}{
			map[string]interface{}{"name": ""},
			map[string]interface{}{"name": ""},
		},
	})
	if err := vs.Set(ScopeTask, "rows[1].name", "订单B"); err != nil {
		t.Fatalf("嵌套写回失败: %v", err)
	}
	got, ok := ResolvePath(vs.Flatten(), "rows[1].name")
	if !ok || got != "订单B" {
		t.Fatalf("嵌套写回后取值=%v ok=%v", got, ok)
	}

	// 中间容器缺失时自动创建
	if err := vs.Set(ScopeTask, "meta.owner.name", "李四"); err != nil {
		t.Fatalf("自动创建中间节点失败: %v", err)
	}
	if got, _ := ResolvePath(vs.Flatten(), "meta.owner.name"); got != "李四" {
		t.Fatalf("meta.owner.name=%v", got)
	}

	// 下标越界必须报错（不静默丢弃）
	if err := vs.Set(ScopeTask, "rows[9].name", "x"); err == nil {
		t.Fatal("越界下标写回必须报错")
	}
}

// TestSensitiveMasking 敏感变量脱敏（日志与上报口径）。
func TestSensitiveMasking(t *testing.T) {
	vars := map[string]interface{}{
		"__credentials": map[string]interface{}{"password": "secret"},
		"__sessionData": map[string]interface{}{"accessToken": "tok"},
		"user":          "alice",
		"password":      "plain-secret",
	}
	masked := MaskVariables(vars)
	for _, key := range []string{"__credentials", "__sessionData", "password"} {
		if masked[key] != "***" {
			t.Fatalf("敏感键 %s 未脱敏: %v", key, masked[key])
		}
	}
	if masked["user"] != "alice" {
		t.Fatalf("非敏感键被误脱敏: %v", masked["user"])
	}

	// 模板替换里引用敏感变量也必须输出 ***
	got, err := ResolveTemplate("token=${__sessionData}", vars, config.UndefinedVariableEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if got != "token=***" {
		t.Fatalf("敏感变量应替换为 ***: %q", got)
	}
}

// TestIsTruthy JS 结果真值判定（condition 依赖）。
func TestIsTruthy(t *testing.T) {
	cases := []struct {
		v    interface{}
		want bool
	}{
		{true, true}, {false, false}, {nil, false},
		{float64(1), true}, {float64(0), false},
		{"true", true}, {"false", false}, {"", false}, {"0", false},
		{"non-empty", true},
	}
	for _, tc := range cases {
		if got := isTruthy(tc.v); got != tc.want {
			t.Errorf("isTruthy(%v)=%v，期望 %v", tc.v, got, tc.want)
		}
	}
}

// TestCompareValues condition(variable) 的比较运算符。
func TestCompareValues(t *testing.T) {
	if ok, _ := compareValues("a", "eq", "a"); !ok {
		t.Fatal("eq 比较失败")
	}
	if ok, _ := compareValues("a", "ne", "b"); !ok {
		t.Fatal("ne 比较失败")
	}
	if ok, _ := compareValues("hello world", "contains", "world"); !ok {
		t.Fatal("contains 比较失败")
	}
	if ok, _ := compareValues("10", "gt", "9"); !ok {
		t.Fatal("gt 数值比较失败（应按数值而非字典序）")
	}
	if ok, err := compareValues("abc", "gt", "9"); err == nil || ok {
		t.Fatal("非数字 gt 比较必须报错")
	}
	if ok, _ := compareValues("", "empty", nil); !ok {
		t.Fatal("empty 判定失败")
	}
	if _, err := compareValues(1, "bogus", 2); err == nil {
		t.Fatal("未知运算符必须报错")
	}
}

// TestEngineExecute_UndefinedPolicyRouting 路由层：策略从 executor 配置贯穿到动作参数。
func TestEngineExecute_UndefinedPolicyRouting(t *testing.T) {
	vs := NewVariableScope(map[string]interface{}{"known": "K"})

	strictCfg := &config.ExecutorConfig{UndefinedVariablePolicy: config.UndefinedVariableStrict}
	e := &Engine{config: strictCfg}
	if _, err := e.substituteVariables(map[string]interface{}{"url": "${missing}"}, vs); err == nil {
		t.Fatal("strict 策略下未定义变量必须报错")
	}

	emptyCfg := &config.ExecutorConfig{UndefinedVariablePolicy: config.UndefinedVariableEmpty}
	e2 := &Engine{config: emptyCfg}
	got, err := e2.substituteVariables(map[string]interface{}{"url": "x-${missing}-${known}"}, vs)
	if err != nil {
		t.Fatalf("empty 策略不应报错: %v", err)
	}
	if got["url"] != "x--K" {
		t.Fatalf("empty 策略替换结果=%q", got["url"])
	}
}

// TestActionFileList upload 参数归一化。
func TestActionFileList(t *testing.T) {
	if got, err := actionFileList("a.txt, b.txt"); err != nil || len(got) != 2 {
		t.Fatalf("逗号分隔归一化失败: %v %v", got, err)
	}
	if got, err := actionFileList([]interface{}{"x", "y"}); err != nil || len(got) != 2 {
		t.Fatalf("[]interface{} 归一化失败: %v %v", got, err)
	}
	if _, err := actionFileList(nil); err == nil {
		t.Fatal("缺失 files 必须报错")
	}
	if _, err := actionFileList([]interface{}{1}); err == nil {
		t.Fatal("非字符串元素必须报错")
	}
}

// TestNestedActionsFrom condition 分支动作解析。
func TestNestedActionsFrom(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{"type": "fill", "selector": "#x", "params": map[string]interface{}{"value": "v"}},
	}
	got := nestedActionsFrom(raw)
	if len(got) != 1 || got[0].Type != types.ActionFill || got[0].Selector != "#x" {
		t.Fatalf("分支动作解析失败: %+v", got)
	}
	if got := nestedActionsFrom(nil); got != nil {
		t.Fatal("nil 分支应返回 nil")
	}
}
