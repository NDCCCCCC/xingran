package addomain

// quick 261019-mmc —— extractRDNFromDN 的 RFC4514 转义语义测试。
//
// 背景（Phase 58 MISSED-04，数据破坏级）：旧实现用 strings.Index(dn, ",") 无条件
// 切第一个逗号，遇到 CN 值内含转义逗号（CN=Zhang\,San）时把 RDN 截断成 "CN=Zhang\"，
// 而 MoveUser 把它交给 ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)
// （deleteOldRDN=true）会让 AD 把对象改名成截断串并删掉旧 RDN。
//
// 本文件不连 DB、不引用 fixture，故不带 `!skip_db_tests` build tag（与 dn_util_test.go 同款）。
// 命名一律 `TestPhase58_*`，便于 `go test ./internal/services/addomain/ -run TestPhase58 -count=1`
// 一把捞起（修复面来自 Phase 58 遗留）。

import (
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rfc4514Case 是 extractRDNFromDN 的用例表。
//
// dn / want 一律用 Go raw string（反引号）书写，因此 `\\` 就是**两个**反斜杠字节，
// `\,` 就是**一个**反斜杠字节 + 一个逗号字节。
//
// crossCheck=false 的两条不是"实现有问题"，而是**输入本身不是合法 RFC4514 DN**，
// 无法用 ldap.ParseDN 对整串交叉校验；它们只由 want 的硬编码期望守住。
// 详见 crossCheck 字段处注释。
type rfc4514Case struct {
	name string
	dn   string
	want string

	// crossCheck 控制本条是否参与 TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP 的
	// ldap.ParseDN 交叉校验（默认应为 true，只有下面两条显式 false）。
	crossCheck bool
}

// rfc4514Cases 是 A 表（硬编码期望）的全部用例，B 表遍历同一份数据。
//
// ⚠ 每一条 crossCheck=false 都必须在下方注释里写明"为什么 ldap.ParseDN 不能校验它"，
//   不允许为了让它变绿而静默扩大豁免范围。
var rfc4514Cases = []rfc4514Case{
	// ---- 转义逗号（本次修复的核心，旧实现全部截断）----
	{
		name:       "转义逗号-中文姓名",
		dn:         `CN=Zhang\,San,OU=IT,DC=corp,DC=com`,
		want:       `CN=Zhang\,San`,
		crossCheck: true,
	},
	{
		name:       "转义逗号-英文姓",
		dn:         `CN=John\,Doe,OU=Users,DC=example,DC=com`,
		want:       `CN=John\,Doe`,
		crossCheck: true,
	},
	{
		name:       "转义逗号后仍有多段",
		dn:         `CN=a\,b\,c,OU=IT,DC=corp,DC=com`,
		want:       `CN=a\,b\,c`,
		crossCheck: true,
	},
	{
		name:       "转义逗号出现在末段",
		dn:         `CN=x\,y,DC=corp,DC=com`,
		want:       `CN=x\,y`,
		crossCheck: true,
	},
	{
		name:       "十六进制转义值",
		dn:         `CN=\41\,B,OU=IT,DC=corp`,
		want:       `CN=\41\,B`,
		crossCheck: true,
	},
	{
		name:       "多值 RDN 带转义逗号",
		dn:         `CN=Zhang\,San+SN=Z,OU=IT,DC=corp,DC=com`,
		want:       `CN=Zhang\,San+SN=Z`,
		crossCheck: true,
	},

	// ---- 多值 RDN（无转义，回归不变）----
	{
		name:       "多值 RDN（+ 连接）",
		dn:         `CN=x+SN=y,OU=IT,DC=corp,DC=com`,
		want:       `CN=x+SN=y`,
		crossCheck: true,
	},

	// ---- 字面反斜杠奇偶（T-Q261019-02）----
	{
		// 两个连续反斜杠 = 一个转义的反斜杠（值域里的字面 `\`），
		// 其后的逗号是**未转义**分隔符 ⇒ 必须切开。
		name:       "字面反斜杠 + 未转义逗号",
		dn:         `CN=a\\,OU=IT,DC=corp`,
		want:       `CN=a\\`,
		crossCheck: true,
	},
	{
		// 值以「转义的反斜杠」结尾、后接分隔逗号 —— 与上一条同形。
		name:       "值以转义反斜杠结尾且后接分隔逗号",
		dn:         `CN=trailing\\,OU=IT,DC=corp`,
		want:       `CN=trailing\\`,
		crossCheck: true,
	},
	{
		// 无逗号 ⇒ 原样返回（等价于旧实现 Index==-1 分支）。
		name:       "值以反斜杠结尾（无逗号）",
		dn:         `CN=ends-with-backslash\\`,
		want:       `CN=ends-with-backslash\\`,
		crossCheck: true,
	},
	{
		// ⚠ 豁免理由：按 RFC4514，这里的逗号**确实是**未转义分隔符，于是后半段 "b"
		// 成了没有 '=' 的残段 —— ldap.ParseDN 对整串报
		// "DN ended with incomplete type, value pair"。这是**输入本身非法**，
		// 不是实现的问题，所以无法对整串交叉校验；期望由 want 硬编码守住。
		name:       "字面反斜杠且后半段无 =",
		dn:         `CN=a\\,b`,
		want:       `CN=a\\`,
		crossCheck: false,
	},

	// ---- 回归不变（不含转义逗号）----
	{
		name:       "普通多段",
		dn:         `CN=John,OU=Users,DC=example,DC=com`,
		want:       `CN=John`,
		crossCheck: true,
	},
	{
		name:       "多层 OU",
		dn:         `CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com`,
		want:       `CN=lisi`,
		crossCheck: true,
	},
	{
		name:       "单 RDN 无逗号",
		dn:         `CN=John`,
		want:       `CN=John`,
		crossCheck: true,
	},
	{
		// ⚠ 豁免理由：ldap.ParseDN("") 返回 len(RDNs)==0（不报错），
		// 交叉校验里 parsed.RDNs[0] 无从比较 ⇒ 无法验证"首 RDN 语义相等"。
		// 期望由 want 的 "" 硬编码守住。
		name:       "空串",
		dn:         ``,
		want:       ``,
		crossCheck: false,
	},
	{
		name:       "含空格",
		dn:         `CN=zhang san,OU=Test Dept,DC=company,DC=com`,
		want:       `CN=zhang san`,
		crossCheck: true,
	},
	{
		name:       "中文 CN",
		dn:         `CN=张三,OU=研发部,DC=corp,DC=com`,
		want:       `CN=张三`,
		crossCheck: true,
	},
}

// TestPhase58_ExtractRDNFromDN_RFC4514Table A 表：硬编码期望。
func TestPhase58_ExtractRDNFromDN_RFC4514Table(t *testing.T) {
	for _, tt := range rfc4514Cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractRDNFromDN(tt.dn))
		})
	}
}

// TestPhase58_ExtractRDNFromDN_NotTruncatedAtEscapedComma 单独钉死"不再被截断"。
// 旧实现在 CN=Zhang\,San 上返回 `CN=Zhang\`（6 字节的后缀被砍掉），
// 这个断言是为了让回归一目了然，而不必从表里挑行对照。
func TestPhase58_ExtractRDNFromDN_NotTruncatedAtEscapedComma(t *testing.T) {
	const dn = `CN=Zhang\,San,OU=IT,DC=corp,DC=com`

	got := extractRDNFromDN(dn)

	assert.NotEqual(t, `CN=Zhang\`, got, "不得在转义逗号处截断")
	assert.Equal(t, `CN=Zhang\,San`, got)
	// 切分确实发生了：结果既非原串，也保留了尾部 OU 段之前的所有字节。
	assert.Less(t, len(got), len(dn))
	assert.Equal(t, dn[:len(got)], got, "结果必须是原串的前缀")
}

// TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP B 表：与 go-ldap 自带的 RFC4514 解析器
// 交叉校验 —— 本任务的核心价值，只有硬编码期望的测试证明不了"手写拆分器的语义正确"，
// 只能证明"输出等于我写下的字面量"。
//
// 对每条 crossCheck=true 的用例：
//  1. ldap.ParseDN(整串) 必须成功（用例本身是合法 DN）；
//  2. ldap.ParseDN(截断输出) 必须成功，且**恰好只含 1 个 RDN**（不能多带后面的段）；
//  3. 两个解析结果的第 0 个 RDN 必须 **Equal**（go-ldap 会小写 attr 名并对多值 RDN
//     排序，不能直接比字符串）。
func TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP(t *testing.T) {
	for _, tt := range rfc4514Cases {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.crossCheck {
				// 显式豁免（理由见 rfc4514Cases 里该条的注释），不是静默跳过。
				t.Skipf("该输入不是合法 RFC4514 DN 或无 RDN 可比，交叉校验不适用: %q", tt.dn)
			}

			parsed, err := ldap.ParseDN(tt.dn)
			require.NoError(t, err, "用例本身必须是合法 DN")

			got := extractRDNFromDN(tt.dn)
			gotParsed, err := ldap.ParseDN(got)
			require.NoError(t, err, "截断输出必须是合法 DN")

			require.Len(t, gotParsed.RDNs, 1, "截断输出必须恰好只含 1 个 RDN")
			assert.True(t, parsed.RDNs[0].Equal(gotParsed.RDNs[0]),
				"手写拆分器的第一个 RDN 必须与 go-ldap RFC4514 解析结果语义相等: got=%q", got)
		})
	}
}
