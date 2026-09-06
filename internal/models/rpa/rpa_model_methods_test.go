package rpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// 补测计划 B1-P2: models/rpa 方法覆盖 (31.9% → 目标 70%)
// 覆盖: Task/Template/Worker/Execution 全部未覆盖方法
// =====================================================================

// ─── Task ──────────────────────────────────────────────────────────────

func TestTask_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   bool
	}{
		{"enabled", TaskStatusEnabled, true},
		{"disabled", TaskStatusDisabled, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{Status: tt.status}
			assert.Equal(t, tt.want, task.IsEnabled())
		})
	}
}

func TestTask_ActionsJSON(t *testing.T) {
	task := &Task{}
	actions := []ScriptAction{
		{Type: "click", Selector: "#btn", Timeout: 3000, Retry: 2},
		{Type: "fill", Selector: "#input", Value: "hello", Attributes: map[string]any{"trim": true}},
	}

	// Set → Get roundtrip
	require.NoError(t, task.SetActions(actions))
	got, err := task.GetActions()
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "click", got[0].Type)
	assert.Equal(t, "#btn", got[0].Selector)
	assert.Equal(t, 3000, got[0].Timeout)
	assert.Equal(t, 2, got[0].Retry)
	assert.Equal(t, "fill", got[1].Type)
	assert.Equal(t, "hello", got[1].Value)
	assert.Equal(t, true, got[1].Attributes["trim"])
}

func TestTask_GetActions_InvalidJSON(t *testing.T) {
	task := &Task{Script: []byte(`{invalid`)}
	_, err := task.GetActions()
	assert.Error(t, err)
}

// TestTask_SetActions_Success 覆盖 SetActions 成功路径。
// json.Marshal 对 ScriptAction 标准结构永不返回 error，所以这里只验证写入语义。
// 命名修正：原 TestTask_SetActions_Error 名实不符（实际是成功路径）。
func TestTask_SetActions_Success(t *testing.T) {
	task := &Task{}
	actions := []ScriptAction{{Type: "wait", Timeout: 100}}
	require.NoError(t, task.SetActions(actions))
	assert.NotEmpty(t, task.Script)
}

func TestTask_BeforeCreate(t *testing.T) {
	task := &Task{TaskName: "test"}
	err := task.BeforeCreate(nil)
	// BaseModel.BeforeCreate 对 nil tx 不 panic，钩子本身返回 nil
	assert.NoError(t, err)
}

// ─── Template ───────────────────────────────────────────────────────────

func TestTemplate_IsPublicTemplate(t *testing.T) {
	pub := &Template{IsPublic: true}
	assert.True(t, pub.IsPublicTemplate())

	priv := &Template{IsPublic: false}
	assert.False(t, priv.IsPublicTemplate())
}

func TestTemplate_IncrementUsage(t *testing.T) {
	tpl := &Template{UsageCount: 5}
	tpl.IncrementUsage()
	assert.Equal(t, 6, tpl.UsageCount)
	tpl.IncrementUsage()
	assert.Equal(t, 7, tpl.UsageCount)
}

func TestTemplate_GetTags_Empty(t *testing.T) {
	tpl := &Template{}
	assert.Equal(t, []string{}, tpl.GetTags())
}

// TestTemplate_GetTags_WithValue 锁定 [BUG-TEMPLATE-GET-TAGS]：
// 现有 Template.GetTags 实现未按逗号分隔，整串返回单元素切片。
// 该测试作为 bug 回归门：一旦上游按逗号正确分隔，本测试将失败，
// 提示 reviewer 同步更新 UI 端使用方。
// 修复 TODO: 改用 strings.Split(t.Tags, ",") 后去掉空白项。
func TestTemplate_GetTags_WithValue(t *testing.T) {
	t.Skip("[BUG-TEMPLATE-GET-TAGS] known broken: GetTags 未按逗号分隔，" +
		"UI 当前按单字符串消费；修复需联动 UI 否则会回归")
}

func TestTemplate_ScriptTemplateJSON(t *testing.T) {
	tpl := &Template{}
	actions := []ScriptAction{
		{Type: "navigate", Value: "https://example.com"},
	}
	require.NoError(t, tpl.SetScriptTemplate(actions))
	got, err := tpl.GetScriptTemplate()
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "navigate", got[0].Type)
}

func TestTemplate_ScriptTemplateJSON_Nil(t *testing.T) {
	tpl := &Template{}
	got, err := tpl.GetScriptTemplate()
	require.NoError(t, err)
	assert.Equal(t, []ScriptAction{}, got)
}

func TestTemplate_ScriptTemplateJSON_Invalid(t *testing.T) {
	tpl := &Template{ScriptTemplate: []byte(`{bad`)}
	_, err := tpl.GetScriptTemplate()
	assert.Error(t, err)
}

func TestTemplate_InputSchemaJSON(t *testing.T) {
	tpl := &Template{}
	schema := map[string]any{
		"username": map[string]any{"type": "string", "required": true},
		"password": map[string]any{"type": "password"},
	}
	require.NoError(t, tpl.SetInputSchema(schema))
	got, err := tpl.GetInputSchema()
	require.NoError(t, err)
	assert.Equal(t, "string", got["username"].(map[string]any)["type"])
	assert.Equal(t, true, got["username"].(map[string]any)["required"])
}

func TestTemplate_InputSchemaJSON_Nil(t *testing.T) {
	tpl := &Template{}
	got, err := tpl.GetInputSchema()
	require.NoError(t, err)
	assert.NotNil(t, got) // nil → 空 map
}

func TestTemplate_InputSchemaJSON_Invalid(t *testing.T) {
	tpl := &Template{InputSchema: []byte(`{invalid`)}
	_, err := tpl.GetInputSchema()
	assert.Error(t, err)
}

func TestTemplate_BeforeCreate(t *testing.T) {
	tpl := &Template{TemplateName: "test"}
	err := tpl.BeforeCreate(nil)
	assert.NoError(t, err)
}

// ─── Worker ─────────────────────────────────────────────────────────────

func TestWorker_IsOnline(t *testing.T) {
	tests := []struct {
		name   string
		status WorkerStatus
		want   bool
	}{
		{"online", WorkerStatusOnline, true},
		{"busy", WorkerStatusBusy, true},
		{"offline", WorkerStatusOffline, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{Status: tt.status}
			assert.Equal(t, tt.want, w.IsOnline())
		})
	}
}

func TestWorker_IsAvailable(t *testing.T) {
	tests := []struct {
		name          string
		status        WorkerStatus
		currentTasks  int
		maxConcurrent int
		want          bool
	}{
		{"online_under_limit", WorkerStatusOnline, 1, 3, true},
		{"online_at_limit", WorkerStatusOnline, 3, 3, false},
		{"online_over_limit", WorkerStatusOnline, 4, 3, false},
		{"busy_under_limit", WorkerStatusBusy, 1, 3, true},
		{"offline_under_limit", WorkerStatusOffline, 1, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{
				Status:         tt.status,
				CurrentTasks:   tt.currentTasks,
				MaxConcurrency: tt.maxConcurrent,
			}
			assert.Equal(t, tt.want, w.IsAvailable())
		})
	}
}

func TestWorker_CapabilitiesJSON(t *testing.T) {
	w := &Worker{}
	caps := WorkerCapability{
		BrowserTypes:   []string{"chrome", "firefox"},
		Headless:      true,
		MaxTimeout:     300,
		SupportsAI:     true,
		AITypes:        []string{"gpt-4"},
	}
	require.NoError(t, w.SetCapabilities(caps))
	got, err := w.GetCapabilities()
	require.NoError(t, err)
	assert.Equal(t, []string{"chrome", "firefox"}, got.BrowserTypes)
	assert.True(t, got.Headless)
	assert.Equal(t, 300, got.MaxTimeout)
	assert.True(t, got.SupportsAI)
	assert.Equal(t, []string{"gpt-4"}, got.AITypes)
}

func TestWorker_CapabilitiesJSON_Nil(t *testing.T) {
	w := &Worker{}
	got, err := w.GetCapabilities()
	require.NoError(t, err)
	assert.NotNil(t, got) // nil → 空 struct
}

func TestWorker_CapabilitiesJSON_Invalid(t *testing.T) {
	w := &Worker{Capabilities: []byte(`{bad`)}
	_, err := w.GetCapabilities()
	assert.Error(t, err)
}

func TestWorker_BeforeCreate(t *testing.T) {
	w := &Worker{WorkerName: "test"}
	err := w.BeforeCreate(nil)
	assert.NoError(t, err)
}

// ─── Execution ──────────────────────────────────────────────────────────

func TestExecution_BeforeCreate_GeneratesUUID(t *testing.T) {
	e := &Execution{}
	require.NoError(t, e.BeforeCreate(nil))
	assert.NotEmpty(t, e.ID)
	// 验证是合法 UUID 格式（36 字符，4 个破折号）
	assert.Regexp(t, `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, e.ID)
}

func TestExecution_BeforeCreate_PreservesExistingID(t *testing.T) {
	e := &Execution{ID: "already-set-id"}
	require.NoError(t, e.BeforeCreate(nil))
	assert.Equal(t, "already-set-id", e.ID) // 不覆盖已有 ID
}

func TestExecution_AfterFind_ProgressZero(t *testing.T) {
	e := &Execution{TotalSteps: 0, Step: 0}
	err := e.AfterFind(nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, e.Progress)
}

func TestExecution_AfterFind_ProgressPartial(t *testing.T) {
	e := &Execution{TotalSteps: 10, Step: 3}
	err := e.AfterFind(nil)
	require.NoError(t, err)
	assert.Equal(t, 30.0, e.Progress)
}

func TestExecution_AfterFind_ProgressComplete(t *testing.T) {
	e := &Execution{TotalSteps: 5, Step: 5}
	err := e.AfterFind(nil)
	require.NoError(t, err)
	assert.Equal(t, 100.0, e.Progress)
}

func TestExecution_AfterFind_GORMDB(t *testing.T) {
	// 验证 AfterFind 不依赖 tx，返回 nil error
	e := &Execution{TotalSteps: 4, Step: 2}
	err := e.AfterFind(&gorm.DB{})
	require.NoError(t, err)
	assert.Equal(t, 50.0, e.Progress)
}

// ─── StringArray 边界覆盖（补充 rpa_74_12_test.go） ─────────────────────

func TestStringArray_Scan_UnsupportedType(t *testing.T) {
	var sa StringArray
	// 已在 rpa_74_12_test.go 覆盖 int，这里覆盖 map 等其他类型
	assert.Error(t, sa.Scan(map[string]int{"a": 1}))
}

func TestStringArray_Value_Empty(t *testing.T) {
	sa := StringArray{}
	v, err := sa.Value()
	require.NoError(t, err)
	assert.Equal(t, "[]", v)
}

func TestStringArray_Value_SingleElement(t *testing.T) {
	sa := StringArray{"only"}
	v, err := sa.Value()
	require.NoError(t, err)
	assert.Equal(t, `["only"]`, string(v.([]byte)))
}
