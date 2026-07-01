package services

import (
	"regexp"
	"strings"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// NormalizeMACAddress 规范化MAC地址格式（统一大写+冒号）
//
// 将各种厂商输出格式统一转换为 AA:BB:CC:DD:EE:FF 格式。
// 这是整个系统的 MAC 单一真实源（2026-07-01 port-mac-format-unify）。
// 原有两个并行实现（mac_history_service 大写版 + mac_collector 小写版）已合并。
//
// 支持的输入格式：
//   - aabb.ccdd.eeff (Cisco/Huawei) -> AA:BB:CC:DD:EE:FF
//   - aa-bb-cc-dd-ee-ff (标准格式) -> AA:BB:CC:DD:EE:FF
//   - aa:bb:cc:dd:ee:ff (已规范) -> AA:BB:CC:DD:EE:FF
//   - AABBCCDDEEFF (无分隔符) -> AA:BB:CC:DD:EE:FF
//
// 契约：返回大写+冒号格式（参见 memory mac-address-normalize-returns-colon-format.md）
func NormalizeMACAddress(mac string) string {
	if mac == "" {
		return ""
	}

	// 移除所有空格
	mac = strings.TrimSpace(mac)
	if mac == "" {
		return ""
	}

	// 移除所有分隔符（.: -）
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(strings.TrimSpace(normalized))

	// 验证格式：应该是12个十六进制字符
	macPattern := regexp.MustCompile(`^[0-9A-F]{12}$`)
	if !macPattern.MatchString(normalized) {
		// 格式不合法，返回原始值
		applogger.Warnf("[MAC] 规范化失败，格式无效: %s", mac)
		return mac
	}

	// 插入冒号分隔符：AA:BB:CC:DD:EE:FF
	var result strings.Builder
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			result.WriteString(":")
		}
		result.WriteString(normalized[i : i + 2])
	}

	return result.String()
}
