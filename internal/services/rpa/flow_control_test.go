package rpa

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// Phase 74-05: flow_control.go + utils.go 测试（纯逻辑层, 无 DB 依赖）。
// =====================================================================

func TestExpressionEvaluator_EvaluateBool_Operators(t *testing.T) {
	ev := NewExpressionEvaluator(nil)
	ctx := context.Background()
	vars := map[string]interface{}{"name": "alice", "age": 20, "nothing": nil}

	cases := []struct {
		expr   string
		want   bool
		wantEr bool
	}{
		{"${name} equals alice", true, false},
		{"${name} equals bob", false, false},
		{"${name} notEquals bob", true, false},
		{"${name} contains lic", true, false},
		{"${name} notContains lic", false, false},
		{"${age} greaterThan 10", true, false},
		{"${age} lessThan 10", false, false},
		{"${age} greaterOrEqual 20", true, false},
		{"${age} lessOrEqual 20", true, false},
		{"${name} matches al.*", true, false},
		{"${name} exists", true, false},
		{"${nothing} exists", false, false},
		{"${nothing} empty", true, false},
		{"${name} empty", false, false},
		{"${name} bogusOp x", false, true},
		{"${missing} equals x", false, true},
		{"one-token", false, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.expr, func(t *testing.T) {
			got, err := ev.EvaluateBool(ctx, tc.expr, vars)
			if tc.wantEr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExpressionEvaluator_EvaluateBool_NumberErrors(t *testing.T) {
	ev := NewExpressionEvaluator(nil)
	ctx := context.Background()

	// 左侧值不是数字
	_, err := ev.EvaluateBool(ctx, "${name} greaterThan 5", map[string]interface{}{"name": "abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "左侧值不是数字")

	// 右侧值不是数字
	_, err = ev.EvaluateBool(ctx, "${age} greaterThan xyz", map[string]interface{}{"age": 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "右侧值不是数字")
}

func TestExpressionEvaluator_EvaluateBool_NestedAccess(t *testing.T) {
	ev := NewExpressionEvaluator(nil)
	ctx := context.Background()
	vars := map[string]interface{}{
		"data": map[string]interface{}{"user": map[string]interface{}{"name": "bob"}},
	}

	got, err := ev.EvaluateBool(ctx, "${data.user.name} equals bob", vars)
	require.NoError(t, err)
	assert.True(t, got)

	// 嵌套字段不存在
	_, err = ev.EvaluateBool(ctx, "${data.user.missing} equals x", vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "字段不存在")

	// 不支持嵌套访问的类型
	_, err = ev.EvaluateBool(ctx, "${data.user.name.x} equals x", vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持嵌套访问类型")
}

func TestExpressionEvaluator_EvaluateString_Number(t *testing.T) {
	ev := NewExpressionEvaluator(nil)
	ctx := context.Background()
	vars := map[string]interface{}{"user": map[string]interface{}{"name": "carl"}}

	got, err := ev.EvaluateString(ctx, "hello ${user.name}!", vars)
	require.NoError(t, err)
	assert.Equal(t, "hello carl!", got)

	// 无占位符原样返回
	got, err = ev.EvaluateString(ctx, "plain", vars)
	require.NoError(t, err)
	assert.Equal(t, "plain", got)

	// 占位符变量不存在
	_, err = ev.EvaluateString(ctx, "${missing}", vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "变量不存在")

	num, err := ev.EvaluateNumber(ctx, "42", vars)
	require.NoError(t, err)
	assert.Equal(t, 42.0, num)

	_, err = ev.EvaluateNumber(ctx, "not-a-number", vars)
	require.Error(t, err)
}

func TestFlowControlService_ExecuteCondition(t *testing.T) {
	svc := NewFlowControlService(nil, nil)
	ctx := context.Background()

	got, err := svc.ExecuteCondition(ctx, &ConditionAction{
		Type:       ActionTypeCondition,
		Expression: "${v} equals 1",
	}, map[string]interface{}{"v": 1})
	require.NoError(t, err)
	assert.True(t, got)

	// 表达式非法 → 包装错误
	_, err = svc.ExecuteCondition(ctx, &ConditionAction{Expression: "bad"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "条件评估失败")

	ok, err := svc.EvaluateCondition(ctx, "${v} equals 1", map[string]interface{}{"v": 1})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestFlowControlService_ExecuteLoop_Count(t *testing.T) {
	svc := NewFlowControlService(nil, nil)
	ctx := context.Background()
	vars := map[string]interface{}{}

	results, err := svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeCount, Count: 3}, vars)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, 2, vars["loopIndex"])

	// 超过最大迭代次数
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeCount, Count: 5, MaxIter: 2}, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "循环超过最大迭代次数")
}

func TestFlowControlService_ExecuteLoop_WhileUntil(t *testing.T) {
	svc := NewFlowControlService(nil, nil)
	ctx := context.Background()

	// while: 条件恒真 → 迭代至 MaxIter 上限
	vars := map[string]interface{}{"flag": 1}
	results, err := svc.ExecuteLoop(ctx, &LoopAction{
		Type:       LoopTypeWhile,
		Expression: "${flag} equals 1",
		MaxIter:    5,
	}, vars)
	require.NoError(t, err)
	require.Len(t, results, 5)
	// 条件为假 → 0 次迭代直接 break
	vars2 := map[string]interface{}{"flag": 0}
	results, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeWhile, Expression: "${flag} equals 1"}, vars2)
	require.NoError(t, err)
	assert.Empty(t, results)

	// while 条件评估失败
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeWhile, Expression: "bad"}, vars2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "while 条件评估失败")

	// until: 条件为真时停止
	vars3 := map[string]interface{}{"stop": 1}
	results, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeUntil, Expression: "${stop} equals 1"}, vars3)
	require.NoError(t, err)
	assert.Empty(t, results)

	// until 迭代至 maxIter 上限
	vars4 := map[string]interface{}{"stop": 0}
	results, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeUntil, Expression: "${stop} equals 1", MaxIter: 3}, vars4)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// until 条件评估失败
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeUntil, Expression: "bad"}, vars4)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "until 条件评估失败")
}

func TestFlowControlService_ExecuteLoop_ForEach(t *testing.T) {
	svc := NewFlowControlService(nil, nil)
	ctx := context.Background()

	// 显式 items
	vars := map[string]interface{}{}
	results, err := svc.ExecuteLoop(ctx, &LoopAction{
		Type:  LoopTypeForEach,
		Items: []interface{}{"a", "b"},
	}, vars)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "b", vars["loopItem"])

	// 从变量列表提取 (string slice → toSlice 反射路径)
	vars2 := map[string]interface{}{"list": []string{"x", "y", "z"}}
	results, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeForEach, Variable: "list"}, vars2)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// 遍历变量不存在
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeForEach, Variable: "missing"}, vars2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "遍历变量不存在")

	// 无法转换为列表
	vars3 := map[string]interface{}{"scalar": 42}
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopTypeForEach, Variable: "scalar"}, vars3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无法转换为列表")

	// 超过 maxIter
	_, err = svc.ExecuteLoop(ctx, &LoopAction{
		Type: LoopTypeForEach, Variable: "list", MaxIter: 2,
	}, vars2)
	require.Error(t, err)

	// 不支持的循环类型
	_, err = svc.ExecuteLoop(ctx, &LoopAction{Type: LoopType("weird")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的循环类型")
}

func TestSimpleConditionEvaluator(t *testing.T) {
	success := executionForEval("success")
	failed := executionForEval("failed")
	running := executionForEval("running")

	ev := NewSimpleConditionEvaluator("status == 'success'")
	got, err := ev.Evaluate(context.Background(), &success)
	require.NoError(t, err)
	assert.True(t, got)

	got, err = ev.Evaluate(context.Background(), &failed)
	require.NoError(t, err)
	assert.False(t, got)

	evFail := NewSimpleConditionEvaluator("status == 'failed'")
	got, err = evFail.Evaluate(context.Background(), &failed)
	require.NoError(t, err)
	assert.True(t, got)

	got, err = ev.Evaluate(context.Background(), &running)
	require.NoError(t, err)
	assert.False(t, got)

	// 不支持的表达式
	weird := NewSimpleConditionEvaluator("bogus expr")
	_, err = weird.Evaluate(context.Background(), &running)
	require.Error(t, err)
}

func TestUtils_FormatFunctions(t *testing.T) {
	ts := time.Date(2026, 8, 21, 10, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-08-21 10:30:00", FormatTimestamp(ts))

	logLine := FormatLog("hello")
	assert.Contains(t, logLine, "hello")
	assert.Regexp(t, `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] hello$`, logLine)

	appended := AppendLog("base", "more")
	assert.Equal(t, "base", appended[:4])
	assert.Contains(t, appended, "more")

	// SanitizeLogMessage: patterns 以 strings.ReplaceAll 字面替换（非正则），
	// 正则风格 pattern 本身不会命中真实消息 — 断言现状（D-12 quirk, 不改业务码）。
	msg := "login with password=secret123 ok"
	out := SanitizeLogMessage(msg)
	assert.Equal(t, msg, out, "正则风格 pattern 是字面量, 真实消息不会被替换")

	assert.Equal(t, 0.0, CalculateProgress(3, 0))
	assert.InDelta(t, 75.0, CalculateProgress(3, 4), 0.001)
	assert.Equal(t, "msg (步骤 3)", FormatProgress(3, 0, "msg"))
	assert.Regexp(t, `msg \(3/4 - 75\.0%\)`, FormatProgress(3, 4, "msg"))
}
