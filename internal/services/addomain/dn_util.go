package addomain

// Phase 58 DN 语义的单一真实源（QUEUE-04 比较 + HYGIENE-02 转义）。
// addomain 包内所有 DN 归一化/拼接一律走本文件，禁在调用点手写。

import (
	"strings"

	"github.com/go-ldap/ldap/v3"
)

// NormalizeDN DN 归一化，**仅用于比较**，绝不用于回写 AD 或生成 DN。
//
// 语义（纯字符串，永不失败、永不 panic、可单测）：
//  1. 整体 TrimSpace
//  2. 按 "," split，逐段 TrimSpace
//  3. 每段按首个 "=" 切分：attr 名 ToLower；值去首尾空格后**同样 ToLower**
//  4. 用 "," 无空格重新拼接
//
// ⚠ 为什么值也要 ToLower（Phase 58 revision 决策，取代早期"值原样保留"的注释）：
//
//	AD DN 的**属性名**（CN/OU/DC）与**属性值**（zhangsan / IT / corp）对 LDAP 匹配都是
//	大小写不敏感的（matchingRuleInCaseIgnoreMatch）。真实域控回写的 OU DN 大小写与本地
//	sys_dept_ou_mapping.ou_dn 经常不一致（`OU=IT,DC=corp,DC=com` vs `ou=it,dc=corp,dc=com`），
//	只归一 attr 名会让 compare-and-move 的幂等比较判为不等 → 重复 MoveUser → 第二次对已搬
//	移的源 DN 调 ModifyDN 直接 code 32（ldap_client.go:350-361，deleteOldRDN=true）。
//	ToLower 对转义序列是安全的：RFC4514 的转义字符（`\,` `\+` `\=` `\<` `\>` `\;` `\"`
//	`\\` `#` 前导空格）与十六进制转义（`\XX`）都**不含大写字母以外的语义** —— 即 `\C` 与
//	`\c` 在 LDAP 语义下等价，ToLower 不破坏转义含义。
//
// ⚠ 刻意不用 ldap.ParseDN：存量 sys_dept_ou_mapping.ou_dn 是 dept_sync_service.go 写入的
// 未转义值，ParseDN 遇未转义特殊字符会失败，只能退回字符串路径。
// ⚠ 与 utils.go ParseOUDN 的分工：ParseOUDN 返回 DN 的层级切片供解析使用；
// NormalizeDN 返回可比较的规范化字符串。两者都不做大小写归一之外的处理，
// 但 NormalizeDN 额外把 attr 名小写化 —— AD 的 DN 属性名本身大小写不敏感。
func NormalizeDN(dn string) string {
	trimmed := strings.TrimSpace(dn)
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		eq := strings.Index(part, "=")
		if eq == -1 {
			// 无 "=" 的段：整段同样 ToLower（保持"纯大小写不敏感比较"的一致语义）
			parts[i] = strings.ToLower(part)
			continue
		}
		attr := strings.ToLower(strings.TrimSpace(part[:eq]))
		value := strings.ToLower(strings.TrimSpace(part[eq+1:]))
		parts[i] = attr + "=" + value
	}
	return strings.Join(parts, ",")
}

// BuildOUDN 部门名 → OU DN 拼接（HYGIENE-02）。全仓唯一拼接点的唯一出口。
//   - 转义一律走 ldap.EscapeDN（RFC4514），**禁手写转义表**（双重转义/漏字符）
//   - ⚠ ldap.EscapeDN 不转义 "="：以 "=" 开头的部门名是残留边界，
//     在 EscapeDN 之后再对前导 "=" 补一个 "\"（不补会产生非法 RDN）
//   - parentOUDN 为空时只返回 "OU=<escaped>"，不产生悬空逗号
func BuildOUDN(deptName, parentOUDN string) string {
	escaped := ldap.EscapeDN(deptName)
	// ldap.EscapeDN 覆盖 RFC4514 的 `"+,;<>\`、首尾空格、前导 `#`、NUL，
	// 但**不**转义 "="；以 "=" 开头的部门名会产出非法 RDN，此处补一个反斜杠。
	if strings.HasPrefix(escaped, "=") {
		escaped = "\\" + escaped
	}
	if parentOUDN == "" {
		return "OU=" + escaped
	}
	return "OU=" + escaped + "," + parentOUDN
}
