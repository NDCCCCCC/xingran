// Package ldaputils 提供LDAP相关的工具函数
package ldaputils

import (
	"strings"
)

// ExtractOUDNFromUserDN 从用户DN中提取OU DN
// 例如: CN=zhangsan,OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com
// 提取为: OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com
func ExtractOUDNFromUserDN(userDN string) string {
	if userDN == "" {
		return ""
	}

	parts := strings.Split(userDN, ",")
	if len(parts) <= 1 {
		return ""
	}

	// 跳过第一个部分（通常是CN=xxx）
	// 返回剩余部分（从第一个OU开始到DN结束）
	for i, part := range parts {
		if strings.HasPrefix(strings.ToUpper(part), "OU=") {
			return strings.Join(parts[i:], ",")
		}
	}

	// 如果没有找到OU，返回DN的Base部分（去掉CN）
	if len(parts) > 1 {
		return strings.Join(parts[1:], ",")
	}

	return ""
}

// ParseOUDN 解析 AD OU DN，返回 OU 部分数组
// 输入示例：OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
// 输出示例：[OU=基础运维科, OU=科技创新部, OU=分公司本部, OU=湖北分公司, OU=CX]
func ParseOUDN(ouDN string) []string {
	if ouDN == "" {
		return []string{}
	}

	// 按逗号分割
	parts := strings.Split(ouDN, ",")

	// 过滤掉 DC= 部分，只保留 OU= 部分
	var ous []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "OU=") {
			ous = append(ous, part)
		}
	}

	return ous
}

// ExtractParentDN 提取父DN
func ExtractParentDN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], ",")
}

// BuildOUPath 构建OU路径
func BuildOUPath(ouDN, baseDN string) string {
	if ouDN == baseDN {
		return "/"
	}
	// 简化处理：去掉baseDN后反转DN部分
	parts := strings.Split(ouDN, ",")
	var pathParts []string
	for _, part := range parts {
		if strings.Contains(part, "OU=") {
			name := strings.TrimPrefix(part, "OU=")
			pathParts = append([]string{name}, pathParts...)
		}
	}
	return "/" + strings.Join(pathParts, "/")
}
