package services

import (
	"regexp"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/normalize"
)

// NormalizeMACAddress 规范化MAC地址格式(转发到 pkg/normalize 单一真实源)。
//
// 契约不变: 返回大写+冒号 AA:BB:CC:DD:EE:FF。
// 实已于 2026-07-01 port-mac-format-unify 下沉到 pkg/normalize.MACAddress,
// 供 models/services/collectors 各层共用(解 import cycle)。本符号保留向后兼容,
// 历史调用点(collectors/mac_collector, interface_collector, arp_collector,
// mac_history_service, mac_collection_service)无需改动。
//
// 原有两个并行实现(mac_history_service 大写版 + mac_collector 小写版)早已合并,现进一步下沉。
// 详细策略与非法输入处理见 pkg/normalize/mac.go + mac_test.go。
func NormalizeMACAddress(mac string) string {
	return normalize.MACAddress(mac)
}

// canonicalMACPattern 标准大写冒号 MAC 格式(AA:BB:CC:DD:EE:FF),即 NormalizeMACAddress 归一化后的形态。
var canonicalMACPattern = regexp.MustCompile(`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)

// isCanonicalMAC 判断 MAC 是否为归一化后的标准大写冒号形态。
//
// 解析层(parseMACLine)用它过滤设备 display mac-address 输出里的表头/汇总行/注释行
// 被误当 MAC 条目(实测垃圾: 'Flags:'/'Total'/'#'/'Invalid')。NormalizeMACAddress 对这些
// 非 hex 输入会原样返回(无法归一),若不校验就会以 'Flags:' 等入库,污染 mac_address 与
// 连锁的 mac_history 轨迹表。加此校验后,垃圾行在解析层即被丢弃。
func isCanonicalMAC(mac string) bool {
	return canonicalMACPattern.MatchString(mac)
}
