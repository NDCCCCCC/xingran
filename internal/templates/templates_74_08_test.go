package templates

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// 74-08 Batch B: internal/templates — ParseTemplate 各输入格式/错误路径
// + FSM ParseText/状态转换/记录提取/Clone/Reset 并发安全。
// =====================================================================

// simpleTpl 一个最小完整模板: 记录行 + Record 转换。
const simpleTpl = `Value NAME (\S+)
Value AGE (\d+)

Start
  ^name ${NAME} age ${AGE} -> Record
`

func TestParseTemplate_EmbedFormats(t *testing.T) {
	// 真实嵌入模板(生产路径)
	fsm, err := ParseTemplate("templates/ruijie_os_show_interfaces_status.textfsm")
	require.NoError(t, err)
	assert.NotEmpty(t, fsm.Variables)

	// 无 templates/ 前缀
	fsm2, err := ParseTemplate("ruijie_os_show_interfaces_status.textfsm")
	require.NoError(t, err)
	assert.NotEmpty(t, fsm2.Variables)

	// 不存在的模板 → 本地文件系统 fallback 失败
	_, err = ParseTemplate("no_such_template_xyz.textfsm")
	assert.ErrorContains(t, err, "读取模板文件失败")

	// 相对路径逃逸 → 不解析为 embed 路径 → 走文件系统 → 失败
	_, err = ParseTemplate("../no_such.textfsm")
	assert.Error(t, err)

	// 纯 "." 输入
	_, err = ParseTemplate(".")
	assert.Error(t, err)
}

func TestParseTemplate_AbsolutePath(t *testing.T) {
	// 写一个临时模板文件,用绝对路径读取
	tmp := filepath.Join(t.TempDir(), "abs_tpl.textfsm")
	require.NoError(t, os.WriteFile(tmp, []byte(simpleTpl), 0o644))

	fsm, err := ParseTemplate(tmp)
	require.NoError(t, err)
	assert.Contains(t, fsm.Variables, "NAME")

	// 绝对路径不存在 → 错误
	_, err = ParseTemplate(filepath.Join(t.TempDir(), "no_such.textfsm"))
	assert.ErrorContains(t, err, "读取模板文件失败")

	// 裸 "((" 被转义为字面量分组括号而非构造正则分组(escapeRegexLiteral 行为锁定)
	bad := filepath.Join(t.TempDir(), "paren_tpl.textfsm")
	require.NoError(t, os.WriteFile(bad, []byte("Value X (\\S+)\n\nStart\n  ^v ((broken ${X}$$\n"), 0o644))
	parenFSM, err := ParseTemplate(bad)
	require.NoError(t, err)
	require.Len(t, parenFSM.States["Start"].Rules, 1)
	assert.Equal(t, `^v \(\(broken (\S+)$`, parenFSM.States["Start"].Rules[0].RegexPattern)
}

func TestParseTemplateFromFilesystem_Relative(t *testing.T) {
	// 相对路径在测试中从项目根解析(internal/templates 包 cwd 即项目根下)
	fsm, err := ParseTemplate("templates/ruijie_os_show_interfaces_status.textfsm")
	require.NoError(t, err)
	_ = fsm // 已在 EmbedFormats 验证;此处保证 filesystem fallback 不被触发时也稳定
}

func TestFindProjectRoot(t *testing.T) {
	// 从本测试文件所在目录(项目根/internal/templates)向上应找到项目根
	here, err := filepath.Abs(".")
	require.NoError(t, err)
	root := findProjectRoot(here)
	require.NotEmpty(t, root)
	base := filepath.Base(filepath.Clean(root))
	assert.Equal(t, "guoguo", base, "找到 go.mod 所在项目根")
}

func TestResolveEmbedPath(t *testing.T) {
	p, ok := resolveEmbedPath("templates/a.textfsm")
	assert.True(t, ok)
	assert.Equal(t, "embedded/templates/a.textfsm", p)

	p, ok = resolveEmbedPath("lldp/x.textfsm")
	assert.True(t, ok)
	assert.Equal(t, "embedded/templates/lldp/x.textfsm", p)

	_, ok = resolveEmbedPath("..")
	assert.False(t, ok)
	_, ok = resolveEmbedPath("/")
	assert.False(t, ok)
}

func TestParseTemplateString_VariableForms(t *testing.T) {
	tpl := `# 注释行被跳过

Value SIMPLE (\S+)
Value Required REQ (\d+)
Value List ITEMS (\w+)
Value Required List REQLIST (\w+)

Start
  ^line ${SIMPLE}
`
	fsm, err := ParseTemplateString(tpl)
	require.NoError(t, err)
	assert.Len(t, fsm.Variables, 4)
	assert.True(t, fsm.Variables["REQ"].Required)
	assert.False(t, fsm.Variables["SIMPLE"].Required)
	assert.True(t, fsm.Variables["ITEMS"].List)
	assert.True(t, fsm.Variables["REQLIST"].Required && fsm.Variables["REQLIST"].List)
	assert.Equal(t, "Start", fsm.CurrentState)

	// 状态定义
	tpl2 := tpl + "\nSomeState\n  ^extra -> Continue\n"
	fsm2, err := ParseTemplateString(tpl2)
	require.NoError(t, err)
	assert.Contains(t, fsm2.States, "SomeState")
	assert.Len(t, fsm2.States["SomeState"].Rules, 1)

	// 未知变量引用 → 通用捕获组 (.+)
	fsm3, err := ParseTemplateString("Value A (\\S+)\n\nStart\n  ^x ${UNKNOWN_VAR} -> Record\n")
	require.NoError(t, err)
	assert.Contains(t, fsm3.States["Start"].Rules[0].Regex.String(), "(.+)")

	// 行尾 $$ 锚点规则(dot1x 同款)被正确解析;可选组形态不锁死
	fsm4, err := ParseTemplateString("Value A (\\S+)\n\nStart\n  ^a ${A}$$ -> Record\n  ^. -> Continue\n")
	require.NoError(t, err)
	require.Len(t, fsm4.States["Start"].Rules, 2)
	assert.Equal(t, "^a (\\S+)$", fsm4.States["Start"].Rules[0].RegexPattern)
	assert.Equal(t, "^.", fsm4.States["Start"].Rules[1].RegexPattern)
}

// ---------------- ParseText / 状态机 ----------------

func TestFSM_ParseText_RecordExtraction(t *testing.T) {
	fsm, err := ParseTemplateString(simpleTpl)
	require.NoError(t, err)

	records, err := fsm.ParseText("name alice age 30\nname bob age 25\n")
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "alice", records[0]["NAME"])
	assert.Equal(t, "30", records[0]["AGE"])
	assert.Equal(t, "bob", records[1]["NAME"])

	// GetRecords / GetFirstRecord / Reset(操作 clone 不影响原模板)
	assert.Len(t, fsm.GetRecords(), 0, "ParseText 内部克隆,原模板记录不变")
	assert.Nil(t, fsm.GetFirstRecord())

	fsm.Reset()
	assert.Empty(t, fsm.Records)
	assert.Equal(t, "Start", fsm.CurrentState)
}

func TestFSM_ParseText_StateTransitions(t *testing.T) {
	tpl := `Value V (\S+)

Start
  ^go -> NextState
  ^err -> Error

NextState
  ^val ${V} -> Record

NoSuchStateIgnored
  ^x
`
	fsm, err := ParseTemplateString(tpl)
	require.NoError(t, err)

	// go → NextState → val 记录
	records, err := fsm.ParseText("go\nval hello\ngo\nval world\n")
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "hello", records[0]["V"])

	// Error → 回 Start;空记录尾不追加
	records, err = fsm.ParseText("err\ngo\nval x\n")
	require.NoError(t, err)
	require.Len(t, records, 1)

	// 不匹配任何规则且无默认转换 → 停留当前状态
	records, err = fsm.ParseText("nothing matches\nval y\n")
	require.NoError(t, err)
	require.Len(t, records, 0, "Start 状态不匹配 → 不进 NextState,val 不匹配 Start 规则")
}

func TestFSM_ParseText_CloneIsolation(t *testing.T) {
	fsm, err := ParseTemplateString(simpleTpl)
	require.NoError(t, err)

	// 并发解析互不污染(Clone 共享只读结构)
	var wg sync.WaitGroup
	results := make([][]map[string]string, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recs, err := fsm.ParseText(strings.Repeat("name user age 20\n", 3))
			require.NoError(t, err)
			results[i] = recs
		}(i)
	}
	wg.Wait()
	for i, recs := range results {
		require.Len(t, recs, 3, "goroutine %d 结果独立", i)
		assert.Equal(t, "user", recs[0]["NAME"])
	}
}

func TestFSM_HandleStateTransition_Unknown(t *testing.T) {
	fsm := &FSM{
		States:        map[string]*State{"Start": {Name: "Start"}},
		CurrentState:  "Start",
		CurrentRecord: map[string]string{"A": "1"},
		Records:       []map[string]string{},
	}

	// 未知状态 → 不变
	fsm.handleStateTransition("NoSuchState")
	assert.Equal(t, "Start", fsm.CurrentState)

	// Record → 保存当前记录并新开
	fsm.handleStateTransition("Record")
	assert.Len(t, fsm.Records, 1)
	assert.Empty(t, fsm.CurrentRecord)

	// Error → 回 Start
	fsm.handleStateTransition("Error")
	assert.Equal(t, "Start", fsm.CurrentState)
}

func TestEscapeRegexLiteral_UnknowLetterEscape(t *testing.T) {
	// 未知字母转义 → 反斜杠被转义
	got := escapeRegexLiteral(`a\zb`)
	assert.Equal(t, `a\\zb`, got)

	// 已知字符类转义保留
	got = escapeRegexLiteral(`a\sb`)
	assert.Equal(t, `a\sb`, got)

	// 尾部孤立反斜杠保留
	got = escapeRegexLiteral(`a\`)
	assert.Equal(t, `a\`, got)
}
