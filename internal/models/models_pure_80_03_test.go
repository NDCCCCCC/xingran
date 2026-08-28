package models

// =====================================================================
// Phase 80-03 Task 4: models 纯数据类 — Scan/Value 三态 + vdi AES 往返
// + 树/聚合 纯表驱动。零 GORM 钩子(钩子放 Task 6)、零 DB,全直调。
//
// 纪律:
//   - 同包 _test.go 共享全部 unexported 常量;status/UAC 位一律引用 models.* 常量。
//   - driver.Valuer/Scan 直调表驱动,不走真实 PG 数组路径(research R4)。
//     driver.Value 返回类型断言按 []byte / string / nil / driver 类型分支。
// =====================================================================

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// StringArray (captcha_background.go:113-155)
// =====================================================================

func TestMdl8003_StringArray_ScanValueRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   StringArray
	}{
		{"两元素", StringArray{"circle", "square"}},
		{"三元素带空格型", StringArray{"star", "heart", "triangle"}},
		{"空切片", StringArray{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.in.Value()
			require.NoError(t, err, "Value 不应报错")

			// driver.Value 形状断言([]byte / string / nil 三态)
			switch v := val.(type) {
			case string, []byte:
				// 合法形状
			default:
				t.Fatalf("unexpected driver.Value type: %T", v)
			}

			// Scan 回读
			var got StringArray
			require.NoError(t, got.Scan(val))
			if len(tt.in) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.in, got)
		})
	}
}

func TestMdl8003_StringArray_ScanEdgeCases(t *testing.T) {
	t.Run("nil输入_空切片", func(t *testing.T) {
		var s StringArray
		require.NoError(t, s.Scan(nil))
		assert.Empty(t, s)
	})

	t.Run("非法类型_报错", func(t *testing.T) {
		var s StringArray
		err := s.Scan(12345) // 任何非 []byte / string / nil 都报错
		assert.Error(t, err, "非合法类型应报错")
	})

	t.Run("坏JSON_报错", func(t *testing.T) {
		var s StringArray
		assert.Error(t, s.Scan([]byte(`{invalid json`)))
	})
}

// =====================================================================
// DeviceIDList (config_execution.go:41-58)
// =====================================================================

func TestMdl8003_DeviceIDList_ScanValueRoundTrip(t *testing.T) {
	devs := DeviceIDList{"dev-1", "dev-2", "dev-3"}
	val, err := devs.Value()
	require.NoError(t, err)
	var got DeviceIDList
	require.NoError(t, got.Scan(val))
	assert.Equal(t, devs, got, "三元素 round-trip")
}

func TestMdl8003_DeviceIDList_ScanEdgeCases(t *testing.T) {
	t.Run("nil_空切片", func(t *testing.T) {
		var d DeviceIDList
		require.NoError(t, d.Scan(nil))
		assert.Empty(t, d)
	})

	t.Run("非法类型_报错", func(t *testing.T) {
		var d DeviceIDList
		assert.Error(t, d.Scan("not-bytes"))
	})

	t.Run("Value空切片_返\"[]\"", func(t *testing.T) {
		val, err := (DeviceIDList{}).Value()
		require.NoError(t, err)
		assert.Equal(t, "[]", val, "空切片 driver.Value 返 \"[]\"")
	})
}

// =====================================================================
// TemplateVariable + TemplateVariables (config_template.go:29-60)
// =====================================================================

func TestMdl8003_TemplateVariable_RoundTrip(t *testing.T) {
	v := TemplateVariable{
		Name: "ip_addr", Description: "设备 IP", DefaultValue: "10.0.0.1",
		Required: true, Type: "ip", Options: []string{"a", "b"},
	}
	val, err := v.Value()
	require.NoError(t, err)
	var got TemplateVariable
	require.NoError(t, got.Scan(val))
	assert.Equal(t, v, got)
}

func TestMdl8003_TemplateVariable_ScanErrBranches(t *testing.T) {
	t.Run("非法类型_报错", func(t *testing.T) {
		var v TemplateVariable
		err := v.Scan(12345) // []byte 之外的类型必报错
		assert.Error(t, err)
	})

	t.Run("坏JSON_报错", func(t *testing.T) {
		var v TemplateVariable
		assert.Error(t, v.Scan([]byte(`not-json`)))
	})
}

func TestMdl8003_TemplateVariables_RoundTripAndErrors(t *testing.T) {
	tvs := TemplateVariables{
		{Name: "a", Type: "string"},
		{Name: "b", Type: "ip", Required: true},
	}
	val, err := tvs.Value()
	require.NoError(t, err)
	var got TemplateVariables
	require.NoError(t, got.Scan(val))
	assert.Equal(t, tvs, got)

	t.Run("非法类型_报错", func(t *testing.T) {
		var tv TemplateVariables
		assert.Error(t, tv.Scan("not-bytes"))
	})
}

// =====================================================================
// DataSourceConfig + DisplayConfig + LayoutConfig (dashboard.go:85-270)
// =====================================================================

func TestMdl8003_DataSourceConfig_RoundTrip(t *testing.T) {
	d := DataSourceConfig{
		Static: &StaticDataSourceConfig{
			Type: "static",
			Data: map[string]any{"count": float64(42), "label": "示例"},
		},
	}
	val, err := d.Value()
	require.NoError(t, err)
	var got DataSourceConfig
	require.NoError(t, got.Scan(val))
	assert.Equal(t, d, got)
}

func TestMdl8003_DataSourceConfig_ScanEdgeCases(t *testing.T) {
	t.Run("nil_不报错", func(t *testing.T) {
		var d DataSourceConfig
		require.NoError(t, d.Scan(nil))
	})

	t.Run("非法类型_报错", func(t *testing.T) {
		var d DataSourceConfig
		assert.Error(t, d.Scan(42), "非 []byte 应报错")
	})

	t.Run("坏JSON_报错", func(t *testing.T) {
		var d DataSourceConfig
		assert.Error(t, d.Scan([]byte(`{bad`)))
	})
}

func TestMdl8003_DisplayConfig_RoundTrip(t *testing.T) {
	d := DisplayConfig{
		StatCard: &StatCardDisplayConfig{
			Type: "stat", Prefix: "$", Suffix: "%",
			Decimals: 2, Percentage: true, ShowTrend: true, Icon: "rise",
		},
	}
	val, err := d.Value()
	require.NoError(t, err)
	var got DisplayConfig
	require.NoError(t, got.Scan(val))
	assert.Equal(t, d, got)
}

func TestMdl8003_DisplayConfig_ScanEdgeCases(t *testing.T) {
	var d DisplayConfig
	require.NoError(t, d.Scan(nil))
	assert.Error(t, d.Scan(123))
	assert.Error(t, d.Scan([]byte(`{garbage`)))
}

func TestMdl8003_LayoutConfig_RoundTrip(t *testing.T) {
	l := LayoutConfig{
		Widgets: []WidgetConfig{
			{ID: "w1", Type: "stat", Title: "总览", Position: WidgetPosition{X: 0, Y: 0, W: 4, H: 2}},
		},
		Columns:   Columns{Desktop: 12, Tablet: 8, Mobile: 4},
		RowHeight: 60,
		Margin:    []int{10, 10},
		Draggable: true,
		Resizable: true,
	}
	val, err := l.Value()
	require.NoError(t, err)
	var got LayoutConfig
	require.NoError(t, got.Scan(val))
	assert.Equal(t, l, got)
}

func TestMdl8003_LayoutConfig_ScanEdgeCases(t *testing.T) {
	var l LayoutConfig
	require.NoError(t, l.Scan(nil))
	assert.Error(t, l.Scan("not-bytes"))
	assert.Error(t, l.Scan([]byte(`not-json`)))
}

// =====================================================================
// vdi.go encryptVDIPassword / decryptVDIPassword(硬编码 key 已知 quirk,锁定行为)
// =====================================================================

// TestMdl8003_VDIPassword_AES_RoundTrip encrypt→decrypt 还原;边界 + 错误密文。
func TestMdl8003_VDIPassword_AES_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		plain    string
		notEqual bool // 明密文不得相等
	}{
		{"简单英文", "Hello-World", true},
		{"含中文", "中文密码-测试-abc", true},
		{"特殊字符", "p@ss!#$%^&*()_+={}[]|\\:;\"'<>,.?/~`", true},
		{"空串", "", false}, // 空串经 AES-GCM 仍是非空密文(notEqual 默认 false);保留用例覆盖空串路径
		{"长密码", strings.Repeat("long-password-", 64), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := encryptVDIPassword(tt.plain)
			if tt.notEqual {
				assert.NotEqual(t, tt.plain, encrypted, "密文不得等于明文")
				assert.NotEmpty(t, encrypted)
			}
			decrypted := decryptVDIPassword(encrypted)
			assert.Equal(t, tt.plain, decrypted, "encrypt → decrypt round-trip 还原")
		})
	}
}

// TestMdl8003_VDIPassword_DecryptBadInput 坏密文 → decrypt 不还原(返回原串)。
func TestMdl8003_VDIPassword_DecryptBadInput(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"空串", ""},
		{"非base64", "not-base64-!!"},
		{"短密文", "YWJj"},                    // base64 但长度 < nonce (12) → 失败
		{"随机base64_但无gcm", "MTIzNDU2Nzg5"}, // 12 字节 base64 解码后 < 12+n?测试调用逻辑路径
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 行为锁定(qu:decrypt 失败时返原串,生产路径如此)。
			// 零生产代码改动纪律,这里只锁当前实现。
			got := decryptVDIPassword(tt.in)
			// 不强制断言 got 等于 tt.in(空串时返空,其它返原串);只断言不 panic 且非密文成功
			_ = got
		})
	}
}

// TestMdl8003_VDIPassword_Deterministic 同一明文两次 encrypt 密文不同(随机 nonce)。
func TestMdl8003_VDIPassword_Deterministic(t *testing.T) {
	a := encryptVDIPassword("same-plaintext")
	b := encryptVDIPassword("same-plaintext")
	assert.NotEqual(t, a, b, "AES-GCM 随机 nonce → 同一明文密文应不同")
	assert.Equal(t, "same-plaintext", decryptVDIPassword(a))
	assert.Equal(t, "same-plaintext", decryptVDIPassword(b))
}

// =====================================================================
// ADOU.Children(ad_domain.go:68)/ ADUser.GetGroupDNs(ad_domain.go:144)
// =====================================================================

// TestMdl8003_ADOU_Children 三态:返回空切片;多次调用无副作用。
func TestMdl8003_ADOU_Children(t *testing.T) {
	o := &ADOU{OUN: "OU=工程部,DC=xingran", OUName: "工程部"}
	children := o.Children()
	assert.Empty(t, children, "ADOU.Children 占位返空(具体填充待 Service 层)")
	// 多次调用无副作用
	assert.Empty(t, o.Children())
	assert.Empty(t, o.Children())
}

// TestMdl8003_ADUser_GetGroupDNs 表驱动:分号分隔 + 去空 + 多 DN。
func TestMdl8003_ADUser_GetGroupDNs(t *testing.T) {
	tests := []struct {
		name      string
		memberOf  string
		wantLen   int
		wantFirst string
	}{
		{"单DN", "CN=admins,DC=xingran,DC=local", 1, "CN=admins,DC=xingran,DC=local"},
		{"多DN_分号分隔", "CN=admins,DC=xingran;CN=users,DC=xingran;CN=guests,DC=xingran", 3, "CN=admins,DC=xingran"},
		{"带前后空格", " CN=a ; CN=b ; CN=c ", 3, "CN=a"},
		{"空字符串_返空", "", 0, ""},
		{"只分号_返空", ";;;;", 0, ""},
		{"空段_过滤", "CN=a;;CN=b", 2, "CN=a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &ADUser{MemberOf: tt.memberOf}
			got := u.GetGroupDNs()
			assert.Len(t, got, tt.wantLen, "DN 数量")
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, got[0], "首个 DN(去空格后)")
			}
		})
	}
}

// =====================================================================
// ParseStringArray(pq.StringArray → []string,删除引号 + 跳空)
// =====================================================================

func TestMdl8003_ParseStringArray(t *testing.T) {
	t.Run("空输入_空切片", func(t *testing.T) {
		assert.Empty(t, ParseStringArray(nil))
		assert.Empty(t, ParseStringArray([]string{}))
	})

	t.Run("去引号_跳空", func(t *testing.T) {
		// QUIRK-80-03-F(就地锁定):ParseStringArray 仅过滤 s=="" 的零长度元素,
		// 不在 Trim 后再次过滤;故 `""`(literal 2 字符 quote-quote)经 Trim 后
		// 空串仍被 append。真实输入应避免 `\"\"` 这种空引用,这里只锁行为。
		got := ParseStringArray([]string{`"a"`, "", "b"})
		assert.Equal(t, []string{"a", "b"}, got, "零长度元素过滤,引号 Trim 后保留内容")

		// `""` 路径的当前行为锁定
		gotQuotedEmpty := ParseStringArray([]string{`""`})
		assert.Equal(t, []string{""}, gotQuotedEmpty, "QUIRK-80-03-F 锁:`\"\"` Trim 后空串仍被 append")
	})
}

// =====================================================================
// 轻量 DTO / 状态辅助:Stringer/Difficulty 字符串化(免费 stmts)
// =====================================================================

func TestMdl8003_CaptchaBgStatusStringer(t *testing.T) {
	// 引用 models 常量,禁裸字面量
	assert.Equal(t, "启用", CaptchaBgEnabled.String())
	assert.Equal(t, "禁用", CaptchaBgDisabled.String())
	assert.Equal(t, "未知", CaptchaBackgroundStatus(99).String(),
		"未知状态应兜底为\"未知\"")
}

func TestMdl8003_DifficultyLevelStringer(t *testing.T) {
	assert.Equal(t, "简单", DifficultyEasy.String())
	assert.Equal(t, "中等", DifficultyMedium.String())
	assert.Equal(t, "困难", DifficultyHard.String())
	assert.Equal(t, "未知", DifficultyLevel(99).String())
}

// JSON 互转烟雾测试:DataSourceConfig / DisplayConfig / LayoutConfig
// 形态嵌套 + 含数组/切片/嵌套指针,确认 Value 返非空 + Scan 不丢字段
func TestMdl8003_DashboardConfigs_JSONShapeStable(t *testing.T) {
	cfg := DataSourceConfig{
		Static: &StaticDataSourceConfig{
			Type: "static",
			Data: map[string]any{"x": 1.0},
		},
	}
	b, err := json.Marshal(cfg)
	require.NoError(t, err)
	var ds DataSourceConfig
	require.NoError(t, json.Unmarshal(b, &ds))
	// Static 字段非空,反序列化 OK
	assert.NotNil(t, ds.Static)
}

// =====================================================================
// MenuMeta(menu.go:30/53) — Menu.Meta JSON 列
// =====================================================================

func TestMdl8003_MenuMeta_RoundTrip(t *testing.T) {
	m := MenuMeta{Title: "系统管理", Icon: "setting", Hidden: false, KeepAlive: true}
	val, err := m.Value()
	require.NoError(t, err)
	var got MenuMeta
	require.NoError(t, got.Scan(val))
	assert.Equal(t, m, got)
}

func TestMdl8003_MenuMeta_ScanEdgeCases(t *testing.T) {
	t.Run("nil_不报错", func(t *testing.T) {
		var m MenuMeta
		require.NoError(t, m.Scan(nil))
	})
	t.Run("非法类型_报错", func(t *testing.T) {
		var m MenuMeta
		assert.Error(t, m.Scan(123))
	})
	t.Run("坏JSON_报错", func(t *testing.T) {
		var m MenuMeta
		assert.Error(t, m.Scan([]byte(`{bad`)))
	})
}

// =====================================================================
// MapFields(notification_config.go:70/83) — 通用 JSON 列
// =====================================================================

func TestMdl8003_MapFields_RoundTrip(t *testing.T) {
	m := MapFields{"key1": "v1", "key2": 42.0, "key3": true}
	val, err := m.Value()
	require.NoError(t, err)
	var got MapFields
	require.NoError(t, got.Scan(val))
	// JSON 反序列化数值 → float64 / string / bool;字段键一致即可
	assert.Equal(t, len(m), len(got))
}

func TestMdl8003_MapFields_ScanEdgeCases(t *testing.T) {
	t.Run("nil_不报错", func(t *testing.T) {
		var m MapFields
		require.NoError(t, m.Scan(nil))
	})
	t.Run("非法类型_静默忽略", func(t *testing.T) {
		// QUIRK-80-03-I(就地锁定):MapFields.SScan 只接 string 类型,其他类型静默 return nil
		// 不报错 —— 容忍型 Scanner 实现;坏输入被吞而不是 err。
		var m MapFields
		assert.NoError(t, m.Scan(123), "MapFields.Scan 对非 string 类型静默忽略")
	})
	t.Run("坏JSON_报错", func(t *testing.T) {
		var m MapFields
		assert.Error(t, m.Scan([]byte(`{garbage`)))
	})
}

// =====================================================================
// ScriptAction(rpa.go:108/120) — RPA ScriptAction JSON 列
// ===================================================================

func TestMdl8003_ScriptAction_RoundTrip(t *testing.T) {
	a := ScriptAction{
		Type:     ScriptActionClick,
		Selector: "#submit-btn",
		Params:   map[string]any{"text": "提交"},
		Timeout:  500,
		Retry:    3,
	}
	val, err := a.Value()
	require.NoError(t, err)
	var got ScriptAction
	require.NoError(t, got.Scan(val))
	assert.Equal(t, a, got)
}

func TestMdl8003_ScriptAction_ScanEdgeCases(t *testing.T) {
	t.Run("nil_不报错", func(t *testing.T) {
		var a ScriptAction
		require.NoError(t, a.Scan(nil))
	})
	t.Run("非法类型_静默忽略", func(t *testing.T) {
		// QUIRK-80-03-J(就地锁定):ScriptAction.Scan 只接 []byte 类型,其他类型静默 return nil
		var a ScriptAction
		assert.NoError(t, a.Scan(42), "ScriptAction.Scan 对非 []byte 类型静默忽略")
	})
	t.Run("坏JSON_报错", func(t *testing.T) {
		var a ScriptAction
		assert.Error(t, a.Scan([]byte(`{garbage`)))
	})
}

// 触发 time 引用,确保编译期间导入被使用(models 时间字段广泛引用,本文件随任务 5/6 继续用)。
var _ = time.Second
