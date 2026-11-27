package types

import (
	"encoding/json"
	"testing"
)

// ============================================================================
// W7.1 / 缺陷 A：别名归一化的「两条路径」都必须覆盖
//
// 这是本轮最深的一层坑，必须锁死：
//
//	路径 1（顶层 script）：TaskMessage.Script 是强类型 []Action，
//	  json.Unmarshal 直接按字段赋值，"type":"goto" 原样落进 Action.Type，
//	  **完全不经过** executor.mapToAction。生产库 6 个 goto 任务走的正是这条。
//	  第一版修复只在 mapToAction 里加了别名表，因此真跑仍然 failed。
//
//	路径 2（嵌套动作）：loop/condition 的 params.actions 是
//	  []map[string]interface{}，经 executor.mapToAction 转换。
//
//   TestTaskMessageUnmarshal_NormalizesTopLevel 路径 1：goto/goTo/GOTO/url 全部
//                                                → ActionNavigate + params.url
//   TestTaskMessageUnmarshal_UnknownTypeUntouched 未知类型原样保留（不静默）
//   TestNormalizeAction_ExplicitURLWins          显式 params.url 不被覆盖
//   TestNormalizeAction_Idempotent               重复归一化无副作用
// ============================================================================

// TestTaskMessageUnmarshal_NormalizesTopLevel 路径 1 主断言。
//
// 消息形状取自生产库真实任务（scripts/ops/rpa_goto_task_dump 导出的 6 个任务的
// 公共形态：URL 在 value、params 只有 waitUntil、retry=2）。
func TestTaskMessageUnmarshal_NormalizesTopLevel(t *testing.T) {
	payload := `{
		"executionId": "exec-1",
		"taskId": "task-1",
		"taskName": "百度搜索结果提取测试",
		"targetUrl": "",
		"inputParams": {},
		"script": [
			{"id":"step1","type":"goto","value":"https://www.baidu.com","selector":"","timeout":15000,"retry":2,"params":{"waitUntil":"networkidle"}},
			{"id":"step2","type":"fill","selector":"#kw","value":"RPA"}
		]
	}`

	var task TaskMessage
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		t.Fatalf("反序列化任务消息失败: %v", err)
	}
	if len(task.Script) != 2 {
		t.Fatalf("动作数=%d，期望 2", len(task.Script))
	}

	first := task.Script[0]
	if first.Type != ActionNavigate {
		t.Fatalf(`顶层 script 的 "goto" 未被归一化：%q（顶层路径走 json.Unmarshal,
		不经过 executor.mapToAction —— 这是第一版修复漏掉的那一层）`, first.Type)
	}
	// URL 键位必须被提升（否则 navigate case 会报 missing url parameter）。
	if got, _ := first.Params["url"].(string); got != "https://www.baidu.com" {
		t.Fatalf("value 未提升到 params.url，实际 %q（params=%v）", got, first.Params)
	}
	// waitUntil 等既有 params 必须保留。
	if first.Params["waitUntil"] != "networkidle" {
		t.Fatalf("既有 params 被破坏：%v", first.Params)
	}
	// 其它字段原样保留。
	if first.Retry != 2 || first.Timeout != 15000 {
		t.Fatalf("retry/timeout 被改动：retry=%d timeout=%d", first.Retry, first.Timeout)
	}
	// 第二个（规范类型）不得被碰。
	if task.Script[1].Type != ActionFill || task.Script[1].Selector != "#kw" {
		t.Fatalf("规范动作被误改：%+v", task.Script[1])
	}
}

// TestTaskMessageUnmarshal_AliasVariants 覆盖任务书要求的四种写法。
func TestTaskMessageUnmarshal_AliasVariants(t *testing.T) {
	cases := []struct {
		raw  string
		want ActionType
	}{
		{"goto", ActionNavigate},
		{"goTo", ActionNavigate},
		{"GoTo", ActionNavigate},
		{"GOTO", ActionNavigate},
		{"url", ActionNavigate},
		{"navigate", ActionNavigate},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			payload := `{"executionId":"e","script":[{"id":"s1","type":"` + c.raw + `","value":"https://example.com"}]}`
			var task TaskMessage
			if err := json.Unmarshal([]byte(payload), &task); err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}
			if task.Script[0].Type != c.want {
				t.Fatalf("type=%q 归一化后=%q，期望 %q", c.raw, task.Script[0].Type, c.want)
			}
			if got, _ := task.Script[0].Params["url"].(string); got != "https://example.com" {
				t.Fatalf("type=%q 的 URL 未提升：%q", c.raw, got)
			}
		})
	}
}

// TestTaskMessageUnmarshal_UnknownTypeUntouched 反向用例：未知类型不被静默吞掉。
func TestTaskMessageUnmarshal_UnknownTypeUntouched(t *testing.T) {
	payload := `{"executionId":"e","script":[{"id":"s1","type":"frobnicate","retry":5}]}`
	var task TaskMessage
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if task.Script[0].Type != ActionType("frobnicate") {
		t.Fatalf("未知类型被改写为 %q（别名表不得做近似猜测）", task.Script[0].Type)
	}
	if task.Script[0].Retry != 5 {
		t.Fatalf("retry 被改动：%d", task.Script[0].Retry)
	}
}

// TestNormalizeAction_ExplicitURLWins 显式 params.url 优先于 value。
func TestNormalizeAction_ExplicitURLWins(t *testing.T) {
	a := &Action{
		Type:  "goto",
		Value: "https://placeholder.example",
		Params: map[string]interface{}{
			"url": "https://real.example",
		},
	}
	NormalizeAction(a)
	if a.Type != ActionNavigate {
		t.Fatalf("Type=%q，期望 navigate", a.Type)
	}
	if got, _ := a.Params["url"].(string); got != "https://real.example" {
		t.Fatalf("params.url 被 value 覆盖：%q", got)
	}
}

// TestNormalizeAction_Idempotent 重复归一化不产生副作用（数据迁移后仍要能跑）。
func TestNormalizeAction_Idempotent(t *testing.T) {
	a := &Action{Type: ActionNavigate, Value: "https://example.com",
		Params: map[string]interface{}{"url": "https://example.com"}}

	NormalizeAction(a)
	if a.Type != ActionNavigate {
		t.Fatalf("第二次归一化改变了 type: %q", a.Type)
	}
	if got, _ := a.Params["url"].(string); got != "https://example.com" {
		t.Fatalf("第二次归一化改变了 url: %q", got)
	}
	// 再跑一次：字段数与内容都不该变。
	before := len(a.Params)
	NormalizeAction(a)
	if len(a.Params) != before {
		t.Fatalf("重复归一化新增了 params 键：%d → %d", before, len(a.Params))
	}
}

// TestNormalizeAction_UnknownKeepsNoURLInjection 未知类型不得被注入 params.url。
//
// 只在类型确实是 navigate 时才提升 URL；否则会把「一个坏动作」伪装成「一个
// 缺 url 的 navigate」，反而更难诊断。
func TestNormalizeAction_UnknownKeepsNoURLInjection(t *testing.T) {
	a := &Action{Type: ActionType("frobnicate"), Value: "https://example.com"}
	NormalizeAction(a)
	if a.Params != nil {
		if _, exists := a.Params["url"]; exists {
			t.Fatalf("未知类型被注入了 params.url（会把配置错误伪装成缺参数）: %v", a.Params)
		}
	}
	if a.Type != ActionType("frobnicate") {
		t.Fatalf("未知类型被改写：%q", a.Type)
	}
}
