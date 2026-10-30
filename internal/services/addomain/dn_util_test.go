package addomain

// Phase 58 Wave 1 —— DN 单一真实源（NormalizeDN / BuildOUDN）纯函数测试。
//
// 命名一律 `TestPhase58_*`，便于 `go test ./internal/services/addomain/ -run TestPhase58 -count=1`
// 一把捞起。本文件不连 DB、不引用包内 fixture，故不带 `!skip_db_tests` build tag。

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPhase58_NormalizeDN_CaseSpaceComma 归一化必须同时吃掉三类差异：
// attr 名大小写、**值侧大小写**、分段首尾空格。
//
// ⚠ 这条就是"值侧 ToLower"的验收依据：若实现只归一 attr 名，本用例必红。
func TestPhase58_NormalizeDN_CaseSpaceComma(t *testing.T) {
	upper := NormalizeDN("CN=ZhangSan,OU=IT,DC=corp,DC=com")
	lower := NormalizeDN("cn=zhangsan , ou=it, dc=corp , dc=com")
	assert.Equal(t, upper, lower,
		"值侧大小写差异（ZhangSan/zhangsan、IT/it）+ attr 名 + 分段空格三类差异必须同时归一")
	assert.Equal(t, "cn=zhangsan,ou=it,dc=corp,dc=com", upper,
		"归一后的规范形态：attr 名与值全部小写、无空格")

	assert.Equal(t, "", NormalizeDN(""), "空串返回空串")

	// 无 "=" 的段：整段同样 ToLower
	assert.Equal(t, "weird", NormalizeDN("Weird"), "无 = 的段原样小写化整段")

	// 转义序列不被破坏：反斜杠转义原样保留，仅大小写归一
	assert.Equal(t, `cn=a\,b,ou=it`, NormalizeDN(`CN=a\,b,OU=IT`),
		"RFC4514 转义序列（\\,）必须原样保留，只做大小写归一")
}

// TestPhase58_NormalizeDN_Idempotent 归一化必须幂等。
func TestPhase58_NormalizeDN_Idempotent(t *testing.T) {
	// 含中文 OU 名 + 多层逗号的真实形态 DN
	src := "CN=张三,OU=研发部,OU=科技创新部,OU=分公司本部,DC=corp,DC=com"
	once := NormalizeDN(src)
	twice := NormalizeDN(once)
	assert.Equal(t, once, twice, "NormalizeDN 必须幂等（归一后再次归一结果不变）")
	assert.NotEmpty(t, once)
}

// TestPhase58_NormalizeDN_DistinguishesDifferentDN 反例锁定：
// 归一不能"过头"把不同的 DN 判相等（否则 compare-and-move 永远判"已一致"而漏搬）。
func TestPhase58_NormalizeDN_DistinguishesDifferentDN(t *testing.T) {
	a := NormalizeDN("CN=a,OU=x,DC=corp,DC=com")
	b := NormalizeDN("CN=a,OU=y,DC=corp,DC=com")
	assert.NotEqual(t, a, b, "OU 不同的两个 DN 归一后必须仍不相等")
}

// TestPhase58_BuildOUDN_EscapesSpecialChars 特殊字符部门名 → 合法 DN（HYGIENE-02）。
func TestPhase58_BuildOUDN_EscapesSpecialChars(t *testing.T) {
	// 部门名含逗号 → 转义为 \,
	got := BuildOUDN("研发,一部", "OU=公司,DC=corp,DC=com")
	assert.Equal(t, `OU=研发\,一部,OU=公司,DC=corp,DC=com`, got,
		"部门名内的逗号必须被转义，结果只含一个未转义的顶层逗号分隔符")

	// 圆括号不在 RFC4514 转义集内 → 原样保留（与库行为一致）
	gotParen := BuildOUDN("研发(一部)", "DC=corp,DC=com")
	assert.Equal(t, "OU=研发(一部),DC=corp,DC=com", gotParen,
		"圆括号 RFC4514 不要求转义，断言与 ldap.EscapeDN 行为一致即可")

	// 前导 "=" → EscapeDN 不转义，本 helper 补一个 "\"
	gotEq := BuildOUDN("=异常部门", "DC=corp,DC=com")
	assert.True(t, strings.HasPrefix(gotEq, `OU=\=`),
		"以 = 开头的部门名必须在 EscapeDN 之后补前导 \\ 转义，否则 RDN 非法；实际: %q", gotEq)

	// parentOUDN 为空 → 不产生悬空逗号
	assert.Equal(t, "OU=独部门", BuildOUDN("独部门", ""), "parentOUDN 为空时不得产生悬空逗号")
}

// TestPhase58_BuildOUDN_PlainNameUnchanged 无特殊字符的部门名逐字不变 ——
// Q4「存量口径」基线：新写入对普通部门名与旧行为完全一致。
func TestPhase58_BuildOUDN_PlainNameUnchanged(t *testing.T) {
	got := BuildOUDN("IT部", "OU=公司,DC=corp,DC=com")
	assert.Equal(t, "OU=IT部,OU=公司,DC=corp,DC=com", got,
		"普通部门名不得被额外转义（零行为变更基线）")
}
