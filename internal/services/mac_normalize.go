// Package services - mac_normalize.go
// 2026-07-03 Phase 47 R5 (D-04): MAC 格式归一化与 canonical 校验工具
// 在 parseRuijiePortSecurityLine 与 (未来) parseMACLine 调用。
// 复用项目记忆 `mac-address-normalize-returns-colon-format` 的归一规则。
package services

import (
	"regexp"
	"strings"
)

// canonicalMACPattern 匹配标准大写冒号格式 MAC 地址: AA:BB:CC:DD:EE:FF
var canonicalMACPattern = regexp.MustCompile(`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)

// isHexOnlyMACPattern 用于 NormalizeMACAddress 中间态校验: 12 位 hex 字符
var isHexOnlyMACPattern = regexp.MustCompile(`^[0-9A-F]{12}$`)

// NormalizeMACAddress 将任意 MAC 格式归一为大写冒号格式 `AA:BB:CC:DD:EE:FF`。
//
// 支持的输入格式:
//   - `aa:bb:cc:dd:ee:ff` (冒号)
//   - `aa-bb-cc-dd-ee-ff` (连字符)
//   - `aabb.ccdd.eeff`    (点分 cisco 风格)
//   - `AABBCCDDEEFF`      (无分隔符)
//
// 不符合 12 hex 字符的输入返回 "" (丢弃语义与 parseRuijiePortSecurityLine 失败路径对齐)。
//
// 调用方应配合 isCanonicalMAC 一起使用:
//
//	mac := NormalizeMACAddress(raw)
//	if mac == "" || !isCanonicalMAC(mac) { return /* skip */ }
func NormalizeMACAddress(input string) string {
	if input == "" {
		return ""
	}
	// 去除常见分隔符
	stripped := strings.NewReplacer(
		":", "",
		"-", "",
		".", "",
		" ", "",
	).Replace(input)
	stripped = strings.ToUpper(stripped)
	if !isHexOnlyMACPattern.MatchString(stripped) {
		return ""
	}
	// 重新插入冒号
	var b strings.Builder
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(stripped[i : i+2])
	}
	return b.String()
}

// isCanonicalMAC 判定输入是否符合 `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$` canonical 格式。
// 用于 parseRuijiePortSecurityLine 末尾守卫 + 数据清理 migration 脏行判定。
func isCanonicalMAC(mac string) bool {
	if mac == "" {
		return false
	}
	return canonicalMACPattern.MatchString(mac)
}
