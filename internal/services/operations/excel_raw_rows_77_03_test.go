package operations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// Phase 77-03 Task 1 — ReadRawRowsByName 全分支 + normalizeHeaderTrim 测试
//
// 覆盖矩阵:
//   - 正常 map[name]row 返回
//   - 同名后者覆盖前者
//   - 空 name 跳过
//   - 配置 sheet 名匹配优先
//   - 配置 sheet 找不到 → 回退首个 sheet (D-06: 内存字节流)
//   - 缺 name 列报错
//   - 仅表头(无数据)报错
//   - 畸形输入([]byte("not a zip"))报错
//   - normalizeHeaderTrim 三态:首尾空格 / 不转小写 / 内部空格保留
//
// Q-77-C 行为锁定 (D-03 注释侧修复): normalizeHeaderTrim 不调用 ToLower,
// 仅按 TrimSpace 后原大小写匹配表头。本测试断言**不转小写**的行为并固化。
// =====================================================================

func TestImp77_ReadRawRowsByName_Normal(t *testing.T) {
	rows := [][]string{
		{"name", "ip_range", "severity_override"},
		{"rule-A", "10.0.0.0/24", "high"},
		{"rule-B", "192.168.0.0/16", "low"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "rule-A", got["rule-A"]["name"])
	assert.Equal(t, "10.0.0.0/24", got["rule-A"]["ip_range"])
	assert.Equal(t, "high", got["rule-A"]["severity_override"])
	assert.Equal(t, "rule-B", got["rule-B"]["name"])
}

func TestImp77_ReadRawRowsByName_SameNameOverride(t *testing.T) {
	rows := [][]string{
		{"name", "ip_range"},
		{"dup", "10.0.0.0/24"},
		{"dup", "192.168.0.0/16"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "192.168.0.0/16", got["dup"]["ip_range"],
		"同名后者覆盖前者")
}

func TestImp77_ReadRawRowsByName_EmptyNameSkipped(t *testing.T) {
	rows := [][]string{
		{"name", "ip_range"},
		{"keep", "10.0.0.0/24"},
		{"", "192.168.0.0/16"},
		{"   ", "10.1.0.0/16"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Contains(t, got, "keep")
	assert.NotContains(t, got, "")
	assert.NotContains(t, got, "   ")
}

func TestImp77_ReadRawRowsByName_FallbackFirstSheet(t *testing.T) {
	rows := [][]string{
		{"name", "ip_range"},
		{"fb-rule", "10.0.0.0/24"},
	}
	// 用其它名字的 sheet,请求"对账例外规则"应回退到此 sheet
	data := buildTestXLSX(t, "随便什么名字", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err, "找不到配置 sheet 时应回退首个 sheet")
	assert.Len(t, got, 1)
	assert.Equal(t, "10.0.0.0/24", got["fb-rule"]["ip_range"])
}

func TestImp77_ReadRawRowsByName_MissingNameColumn(t *testing.T) {
	rows := [][]string{
		{"ip_range", "severity_override"},
		{"10.0.0.0/24", "high"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	_, err := ReadRawRowsByName(file, "对账例外规则")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缺少 name 列")
}

func TestImp77_ReadRawRowsByName_HeaderOnlyNoData(t *testing.T) {
	rows := [][]string{
		{"name", "ip_range"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	_, err := ReadRawRowsByName(file, "对账例外规则")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Excel 数据为空")
}

func TestImp77_ReadRawRowsByName_MalformedBytes(t *testing.T) {
	// D-06: 畸形输入手工字节构造,非 zip 魔数必败
	file := xlsxFileHeader(t, []byte("not a zip"), "bad.xlsx")

	_, err := ReadRawRowsByName(file, "对账例外规则")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析 Excel 失败")
}

func TestImp77_ReadRawRowsByName_NilFile(t *testing.T) {
	_, err := ReadRawRowsByName(nil, "对账例外规则")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "文件不能为空")
}

func TestImp77_ReadRawRowsByName_AltNameColumn(t *testing.T) {
	// nameIdx 接受 "规则名称" 中文别名
	rows := [][]string{
		{"规则名称", "ip_range"},
		{"中文规则-1", "10.0.0.0/24"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "10.0.0.0/24", got["中文规则-1"]["ip_range"])
}

func TestImp77_NormalizeHeaderTrim_BehaviorLockdown(t *testing.T) {
	// Q-77-C (D-03): normalizeHeaderTrim 不转小写,只 TrimSpace
	// 测试断言:任意大小写在 helper 处理后维持原大小写。
	// 如果未来有人误加 ToLower,本测试 FAIL(行为变更需要额外评审)。
	cases := []struct {
		in, want string
	}{
		{"name", "name"},
		{"Name", "Name"},
		{"NAME", "NAME"},
		{"  name  ", "name"},
		{"\t规则名称\n", "规则名称"},
		{"Section_Office_Code", "Section_Office_Code"},
		{"  Mixed_CASE  ", "Mixed_CASE"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := normalizeHeaderTrim(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestImp77_ReadRawRowsByName_HeaderTrailingWhitespace(t *testing.T) {
	// D-06: 表头含首尾空格 → normalizeHeaderTrim 切除后匹配 name
	rows := [][]string{
		{"  name  ", "ip_range"},
		{"whitespace-rule", "10.0.0.0/24"},
	}
	data := buildTestXLSX(t, "对账例外规则", rows)
	file := xlsxFileHeader(t, data, "rules.xlsx")

	got, err := ReadRawRowsByName(file, "对账例外规则")
	require.NoError(t, err)
	require.Len(t, got, 1)
	ipVal, _ := got["whitespace-rule"]["ip_range"].(string)
	assert.True(t, strings.Contains(ipVal, "10.0.0.0"))
}
