package utils

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

)

// =====================================================================
// 74-11 escalation gap-closure: internal/utils — template_engine 全 API
// + pagination + response_builder + db_helper(sqlite) + string_helper
// 缺口。D-12: 仅 *_test.go。
// =====================================================================

// ---------------- template_engine.go ----------------

func TestTemplateEngine_RenderFuncs(t *testing.T) {
	e := NewTemplateEngine()

	// 字符串函数
	out, err := e.Render(`{{toUpper .v}}|{{toLower .V}}|{{trim .s}}|{{replace .r "a" "b"}}`,
		map[string]interface{}{"v": "abc", "V": "DEF", "s": "  x  ", "r": "banana"})
	require.NoError(t, err)
	assert.Equal(t, "ABC|def|x|bbnbnb", out)

	// contains/hasPrefix/hasSuffix/split/join/repeat
	out, err = e.Render(`{{contains .c "ell"}}|{{hasPrefix .c "he"}}|{{hasSuffix .c "lo"}}|{{join (split .c "l") "-"}}|{{repeat "x" .n}}`,
		map[string]interface{}{"c": "hello", "n": 2})
	require.NoError(t, err)
	assert.Equal(t, "true|true|true|he--o|xx", out)

	// 条件判断
	out, err = e.Render(`{{eq .a .b}}|{{ne .a .b}}|{{gt .n1 .n2}}|{{lt .n1 .n2}}|{{gte .n1 .n1}}|{{lte .n1 .n1}}|{{and true false}}|{{or false true}}|{{not true}}`,
		map[string]interface{}{"a": 1, "b": 1, "n1": 5, "n2": 3})
	require.NoError(t, err)
	assert.Equal(t, "true|false|true|false|true|true|false|true|false", out)

	// 类型转换
	out, err = e.Render(`{{toString .v}}|{{toInt .i}}|{{toBool .b}}`,
		map[string]interface{}{"v": 42, "i": "7", "b": "yes"})
	require.NoError(t, err)
	assert.Equal(t, "42|7|true", out)

	// 网络函数
	out, err = e.Render(`{{ipToHex .ip}}|{{macToHex .mac}}|{{formatVLAN .vlan}}|{{formatInterface .if}}|{{formatIP .ipx}}`,
		map[string]interface{}{"ip": "192.168.1.10", "mac": "aa:bb:cc:dd:ee:ff", "vlan": 100, "if": "gi0/0/1", "ipx": " 1.2.3.4 "})
	require.NoError(t, err)
	assert.Equal(t, "C0A8010A|AABBCCDDEEFF|100|GigabitEthernet0/0/1|1.2.3.4", out)

	// 列表函数
	out, err = e.Render(`{{first .l}}|{{last .l}}|{{len .l}}|{{len .s}}|{{first .empty}}`,
		map[string]interface{}{"l": []interface{}{"a", "b", "c"}, "s": "abcd", "empty": []interface{}{}})
	require.NoError(t, err)
	assert.Equal(t, "a|c|3|4|<no value>", out)

	// title
	out, err = e.Render(`{{title .w}}`, map[string]interface{}{"w": "hello world"})
	require.NoError(t, err)
	assert.Equal(t, "Hello World", out)
}

func TestTemplateEngine_RenderErrors(t *testing.T) {
	e := NewTemplateEngine()

	// 非法模板语法 → 解析失败
	_, err := e.Render("{{.broken", nil)
	assert.ErrorContains(t, err, "解析模板失败")

	// 执行失败(函数不存在 → 解析期;字段类型错误 → 执行期)
	_, err = e.Render(`{{toInt .map}}`, map[string]interface{}{"map": map[string]int{}})
	_ = err // toInt 对 map 返回 0,不报错 — 换成真正执行失败:
	_, err = e.Render(`{{.x.y}}`, map[string]interface{}{"x": 5})
	assert.ErrorContains(t, err, "执行模板失败", "int 无字段 y")
}

func TestTemplateEngine_ValidateExtractBuild(t *testing.T) {
	e := NewTemplateEngine()

	// ValidateTemplate
	assert.NoError(t, e.ValidateTemplate("{{.a}} {{toUpper .a}}"))
	assert.Error(t, e.ValidateTemplate("{{.a"))

	// ExtractVariables: 去重/管道/函数调用/嵌套结束符
	vars := e.ExtractVariables(`{{.ip}} {{.ip | toUpper}} {{.mac}} {{.if formatInterface}} text {{.ip}}`)
	assert.ElementsMatch(t, []string{"ip", "mac", "if"}, vars)

	// 无变量 → 空
	assert.Empty(t, e.ExtractVariables("plain text"))

	// 未闭合 → 已提取部分保留
	vars = e.ExtractVariables("{{.a}} {{.b")
	assert.Equal(t, []string{"a"}, vars)

	// BuildVariablesMap
	defs := []TemplateVariable{
		{Name: "ip", Required: true, Type: "ip"},
		{Name: "vlan", DefaultValue: "100", Type: "vlan"},
		{Name: "num", Type: "int"},
		{Name: "flag", Type: "bool"},
		{Name: "mac", Type: "mac"},
		{Name: "unknown-t", Type: "weird"},
	}
	m, err := e.BuildVariablesMap(defs, map[string]string{"ip": " 10.0.0.1 ", "num": "5", "flag": "on", "mac": "aa:bb:cc:dd:ee:ff"})
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", m["ip"])
	assert.Equal(t, "100", m["vlan"], "默认值生效")
	assert.Equal(t, 5, m["num"])
	assert.Equal(t, true, m["flag"])
	assert.Equal(t, "AABBCCDDEEFF", m["mac"])

	// 未知类型 → 原样字符串
	mWeird, err := e.BuildVariablesMap([]TemplateVariable{{Name: "v", Type: "weird"}}, map[string]string{"v": "x"})
	require.NoError(t, err)
	assert.Equal(t, "x", mWeird["v"])

	// 缺必需变量
	_, err = e.BuildVariablesMap([]TemplateVariable{{Name: "req", Required: true}}, map[string]string{})
	assert.ErrorContains(t, err, "缺少必需变量: req")

	// RenderWithValidation
	_, err = e.RenderWithValidation("{{.a}}", map[string]interface{}{}, []string{"a"})
	assert.ErrorContains(t, err, "缺少必需变量: a")
	out, err := e.RenderWithValidation("{{.a}}", map[string]interface{}{"a": "x"}, []string{"a"})
	require.NoError(t, err)
	assert.Equal(t, "x", out)
}

func TestTemplateEngine_GeneratePreview(t *testing.T) {
	e := NewTemplateEngine()

	// 各类型示例值
	defs := []TemplateVariable{
		{Name: "n", Type: "int"},
		{Name: "f", Type: "bool"},
		{Name: "ip", Type: "ip"},
		{Name: "m", Type: "mac"},
		{Name: "v", Type: "vlan"},
		{Name: "sel", Type: "select"},
		{Name: "other", Type: "custom"},
		{Name: "d", DefaultValue: "DEF"},
	}
	out, err := e.GeneratePreview("[{{.n}} {{.f}} {{.ip}} {{.m}} {{.v}} {{.sel}} {{.other}} {{.d}}]", defs)
	require.NoError(t, err)
	assert.Equal(t, "[100 true 192.168.1.1 00:11:22:33:44:55 100 option1 <other> DEF]", out)
}

func TestParseCommandVariables(t *testing.T) {
	// 空
	assert.Empty(t, ParseCommandVariables(""))

	// 混合 = / := / 空段
	m := ParseCommandVariables("a=1, b:=2, ,c=x=y")
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "x=y"}, m)
}

func TestTemplateEngine_NetHelpers(t *testing.T) {
	// ipToHex 边角
	assert.Equal(t, "C0A8010A", ipToHex("192.168.1.10"))
	assert.Equal(t, "", ipToHex("not-ip"))

	// macToHex 分隔符与长度
	assert.Equal(t, "AABBCCDDEEFF", macToHex("aa-bb.cc dd:ee:ff"))
	assert.Equal(t, "", macToHex("aabbccddeeff00"))
	assert.Equal(t, "AABBCCDDEEFF", macToHex("aabbccddeeff"))

	// formatVLAN int/string/default
	assert.Equal(t, "100", formatVLAN(100))
	assert.Equal(t, "200", formatVLAN("200"))
	assert.Equal(t, "1", formatVLAN(1.5), "default 分支走 toInt 截断")

	// formatInterface 简写映射
	assert.Equal(t, "FastEthernet0/1", formatInterface(" fa0/1 "))
	assert.Equal(t, "TenGigabitEthernet1/1", formatInterface("TE1/1"))
	assert.Equal(t, "FortyGigE1/1", formatInterface("FO1/1"))
	assert.Equal(t, "GigabitEthernet0/0/1", formatInterface("ge0/0/1"), "ge 大写后 GE 映射 GigabitEthernet")
	assert.Equal(t, "XGIG", formatInterface("XGIG"))

	// toString/toInt/toBool 边角
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "x", toString("x"))
	assert.Equal(t, 3, toInt(3))
	assert.Equal(t, 7, toInt(int64(7)))
	assert.Equal(t, 9, toInt(float64(9.7)))
	assert.Equal(t, 5, toInt("5"))
	assert.Equal(t, 0, toInt("bad"))
	assert.Equal(t, 0, toInt(1.5 == 1.5))
	assert.True(t, toBool(true))
	assert.True(t, toBool("TRUE"))
	assert.True(t, toBool("1"))
	assert.True(t, toBool(2))
	assert.False(t, toBool("nope"))
	assert.False(t, toBool(0))
	assert.False(t, toBool(nil))
}

// ---------------- pagination.go ----------------

func TestParsePaginationAndOffset(t *testing.T) {
	// 默认值
	p := ParsePagination(0, 0)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 10, p.PageSize)

	// 上限截断
	p = ParsePagination(-1, 99999)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 100, p.PageSize, "MaxListPageSize 上限")

	// 正常值
	p = ParsePagination(3, 20)
	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 20, p.PageSize)
	assert.Equal(t, 40, p.Offset())
	assert.Equal(t, 20, p.Limit())

	// 第一页 offset=0
	assert.Equal(t, 0, ParsePagination(1, 10).Offset())

	// BuildPaginationResponse
	res := BuildPaginationResponse([]int{1, 2}, 2, PaginationParams{Page: 1, PageSize: 10})
	assert.Equal(t, []int{1, 2}, res["list"])
	assert.Equal(t, int64(2), res["total"])
	assert.Equal(t, 1, res["page"])
	assert.Equal(t, 10, res["pageSize"])
}

// ---------------- response_builder.go ----------------

func TestResponseBuilders(t *testing.T) {
	assert.Equal(t, "x", BuildSuccessResponse("x")["data"])

	lr := BuildListResponse([]int{}, 5, 1, 10)
	assert.Equal(t, int64(5), lr["total"])
	assert.Equal(t, 1, lr["page"])
	assert.Equal(t, 10, lr["pageSize"])

	assert.Equal(t, 7, BuildCountResponse(7)["count"])
	assert.Equal(t, "ok", BuildMessageResponse("ok")["message"])
	assert.Equal(t, "abc", BuildIDResponse("abc")["id"])
}

// ---------------- string_helper.go 缺口 ----------------

func TestStringHelperGaps(t *testing.T) {
	// Contains 忽略大小写
	assert.True(t, Contains("Hello World", "WORLD"))
	assert.False(t, Contains("abc", "z"))

	// IsEmpty / IsNotEmpty(nil 与空串)
	var nilStr *string
	assert.True(t, IsEmpty(nilStr))
	assert.True(t, IsEmpty(strPtrUtil("")))
	assert.False(t, IsEmpty(strPtrUtil("x")))
	assert.False(t, IsNotEmpty(nilStr))
	assert.True(t, IsNotEmpty(strPtrUtil("x")))

	// TrimSpace: nil→nil / 全空白→nil / 正常裁剪
	assert.Nil(t, TrimSpace(nil))
	assert.Nil(t, TrimSpace(strPtrUtil("   ")))
	got := TrimSpace(strPtrUtil("  v  "))
	require.NotNil(t, got)
	assert.Equal(t, "v", *got)

	// JoinStringSlice / SplitString
	assert.Equal(t, "", JoinStringSlice(nil, ","))
	assert.Equal(t, "a,b", JoinStringSlice([]string{"a", "b"}, ","))
	assert.Empty(t, SplitString("", ","))
	assert.Equal(t, []string{"a", "b"}, SplitString(" a , b ", ","))

	// SanitizeAndTruncate 组合(NUL 剔除 + 长度截断;不 TrimSpace 首尾空格)
	got2 := SanitizeAndTruncate("he\x00llo", 20)
	assert.Equal(t, "hello", got2)
	assert.Equal(t, "abcd…", SanitizeAndTruncate("abcdefghij", 5), "maxLen-1 字符 + 省略号")
}

func strPtrUtil(s string) *string { return &s }

// ---------------- db_helper.go (sqlite) ----------------

type dbHelperRow struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func (dbHelperRow) TableName() string { return "db_helper_rows" }

func newHelperDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "helper.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.Migrator().CreateTable(&dbHelperRow{}))
	require.NoError(t, db.Create(&dbHelperRow{ID: "r1", Name: "alpha"}).Error)
	require.NoError(t, db.Create(&dbHelperRow{ID: "r2", Name: "beta"}).Error)
	return db
}

func TestDBHelpers(t *testing.T) {
	db := newHelperDB(t)

	// CheckExists
	ok, err := CheckExists(db, &dbHelperRow{}, "name = ?", "alpha")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, _ = CheckExists(db, &dbHelperRow{}, "name = ?", "ghost")
	assert.False(t, ok)
	_, err = CheckExists(db, &dbHelperRow{}, "bad_col = ?", 1)
	assert.ErrorContains(t, err, "检查记录是否存在失败")

	// CheckExistsExclude: 排除自身 → 不算重名
	ok, err = CheckExistsExclude(db, &dbHelperRow{}, "r1", "name", "alpha")
	require.NoError(t, err)
	assert.False(t, ok, "排除自身")
	ok, _ = CheckExistsExclude(db, &dbHelperRow{}, "", "name", "beta")
	assert.True(t, ok, "不排除 → 存在")

	// GetByID: 命中/不存在/错误
	var row dbHelperRow
	require.NoError(t, GetByID(db, &row, "r1"))
	assert.Equal(t, "alpha", row.Name)
	err = GetByID(db, &row, "missing")
	assert.ErrorContains(t, err, "记录不存在")
	var badModel struct{}
	err = GetByID(db, &badModel, "x")
	assert.Error(t, err)

	// DeleteByID
	require.NoError(t, DeleteByID(db, &dbHelperRow{}, "r2"))
	var cnt int64
	require.NoError(t, db.Model(&dbHelperRow{}).Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt)
	err = DeleteByID(db, &struct{}{}, "x")
	assert.Error(t, err)

	// SoftDeleteByID(自定义 deleted 字段)
	require.NoError(t, db.Exec(`ALTER TABLE db_helper_rows ADD COLUMN deleted INTEGER DEFAULT 0`).Error)
	require.NoError(t, SoftDeleteByID(db, &dbHelperRow{}, "r1", "deleted"))
	var deleted int
	require.NoError(t, db.Raw(`SELECT deleted FROM db_helper_rows WHERE id='r1'`).Scan(&deleted).Error)
	assert.Equal(t, 1, deleted)
}
