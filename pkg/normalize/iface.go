// Package normalize 提供网络接口名与 MAC 地址的格式归一化,作为全系统单一真实源。
//
// 本包无任何内部依赖(仅标准库),供 models/services/collectors 各层共用,
// 避免"models import services 导致 import cycle"。
//
// 归一化目标(2026-07-01 port-mac-format-unify 对称化后):
//   - 接口名统一为"大写短名 + 数字",如 GE0/0/1 / XGE1/0/1 / FE0/1 / ET3/46
//   - MAC 统一为"大写 + 冒号",如 AA:BB:CC:DD:EE:FF
//
// 历史背景见 internal/services/portcollection/utils.go(本包由该处下沉而来)。
package normalize

import (
	"regexp"
	"strings"
)

// garbledShortFullPattern 匹配"短名 + 全称残 + 数字"拼接的乱码形式。
// 实测样本(2026-07-01 verify-format-unify):
//   - "GEGigabitEthernet5/29" → 短 GE + 全 GigabitEthernet + 数字
//   - "XGigabitEthernet0/0/1" → 短 XGE + 全 GigabitEthernet + 数字
//   - "GEgabitetherngigabitethernet4/48" → 短 GE + 多次重复 gigabitethernet
//
// 提取: capture[1] = 短名 (GE/XGE/HGE/FOE/FE), capture[3] = 数字+斜杠部分
// 排除 TWE(短名 twentyfivegige 内嵌 gige 子串会冲突);XG 单独走 garbledXGPrefix。
var garbledShortFullPattern = regexp.MustCompile(`(?i)^(GE|XGE|HGE|FOE|FE)(.*?(?:igabit|gigabit|gige).*?)(\d.*)$`)

// garbledXGPrefix 显式处理 XGigabitEthernet0/0/1 → XGE0/0/1
// H3C 设备特定乱码(短名 XG 截断了 XGE 末尾 E,全名截断了 Gigabit 开头 G)
var garbledXGPrefix = regexp.MustCompile(`(?i)^XG(.*?(?:igabit|gigabit|gige).*?)(\d.*)$`)

// standardShortPrefix 守卫: 已是标准大写速率短名 + 数字的接口名直接返回。
//
// 背景(2026-07-01 反向展开 bug): prefixList 的"短名→全称"映射会把 ToLower 后的
// GE0/0/1 命中 'ge' 前缀反向展开成 GigabitEthernet0/0/1,与"短名大写"目标冲突,
// 导致采集入库后 M186 折叠→下次采集又展开→无限拉锯。华为 VRP display interface
// brief 在窄屏/多端口时会自动输出短名 GE0/0/1 适应列宽,必须保护。
//
// 匹配: 短名(GE/XGE/TWE/HGE/FOE/FE/ET) 后紧跟数字。
//   - GE0/0/1, XGE1/0/1, ET3/46 → 命中,返回 ToUpper(原值)
//   - GEet5/34 (乱码) → GE 后 'e' 非数字,不命中,走原逻辑
//   - GigabitEthernet0/0/1 → 不命中,走 fullToShort 折叠
var standardShortPrefix = regexp.MustCompile(`^(GE|XGE|TWE|HGE|FOE|FE|ET)[0-9]`)

// shortIfaceBodyPattern 在 standardShortPrefix 守卫命中后,提取"速率短名 + 数字 + 数字/斜杠/点/冒号"
// 的合法接口名主体,丢弃尾部粘连的字母垃圾。
//
// 背景(2026-07-02 mac-iface-security-suffix): 华为 display mac-address 输出 security 类型
// MAC 时,Learned-From 列接口名会粘连 security 标记(无空格),如 GE1/0/4security。守卫命中
// GE1/... 后若直接 ToUpper 返回会得到 GE1/0/4SECURITY 入库。本正则提取合法主体 GE1/0/4,
// 丢弃尾部 SECURITY。
//
// 字符集 [0-9/.:]: 接口名数字段常见字符。不含字母(物理口数字段无大段字母);不含横杠
// (Eth-Trunk 等逻辑接口前缀不匹配本守卫,走另一路径)。
//   - GE1/0/4SECURITY → GE1/0/4
//   - GE0/0/1 → GE0/0/1(不变)
//   - GE0/0/1.5(子接口) → GE0/0/1.5(点保留)
var shortIfaceBodyPattern = regexp.MustCompile(`^(GE|XGE|TWE|HGE|FOE|FE|ET)[0-9][0-9/.:]*`)

// extractShortIfaceBody 从已通过 standardShortPrefix 守卫的接口名中提取合法主体,ToUpper 返回。
// 守卫已保证开头是合法短名+数字,故 shortIfaceBodyPattern 必能匹配;理论失配时回退 ToUpper 原值。
func extractShortIfaceBody(name string) string {
	upper := strings.ToUpper(name)
	if matched := shortIfaceBodyPattern.FindString(upper); matched != "" {
		return matched
	}
	return upper
}

// InterfaceName 把任意厂商接口名归一化为"大写短名 + 数字"。
//
// 对称化策略(2026-07-01):
//   - 全称(GigabitEthernet/TenGigE/TenGigabitEthernet/FortyGigE/HundredGigE/TwentyFiveGigE/FastEthernet)
//     → 大写短名(GE/XGE/FOE/HGE/TWE/FE)
//   - 已是大写短名(GE0/0/1) → 原样返回(守卫防反向展开)
//   - cisco 小写短名(gi/te/xe/fo/fa/fge/hge/twe/tw) → 大写短名
//   - 全小写 full name(gigabitethernet0/0/1) → 大写短名
//   - 乱码(GEGigabitEthernet5/29) → 大写短名(GE5/29)
//
// 逻辑接口(Vlan/Vlanif/Loop/NULL 等)保持其规范形式。
// 未识别前缀原样返回(由调用方或迁移处理)。
func InterfaceName(name string) string {
	name = strings.ReplaceAll(name, " ", "")
	if name == "" {
		return name
	}

	// 阶段 -1: 已是标准大写短名 + 数字,提取合法主体并剥离尾部字母垃圾后返回(防反向展开)
	// 2026-07-02 fix: 原 ToUpper(name) 直接返回会放行尾部粘连(如 GE1/0/4security → GE1/0/4SECURITY),
	// 改用 extractShortIfaceBody 剥离尾部字母垃圾。
	if standardShortPrefix.MatchString(strings.ToUpper(name)) {
		return extractShortIfaceBody(name)
	}

	// 阶段 0: 清理乱码形式(短名+全称残+数字)
	if matched := garbledXGPrefix.FindStringSubmatch(name); matched != nil {
		return "XGE" + matched[2]
	}
	if matched := garbledShortFullPattern.FindStringSubmatch(name); matched != nil {
		return strings.ToUpper(matched[1]) + matched[3]
	}

	// 阶段 1: 全称 → 短名
	fullToShort := map[string]string{
		"FastEthernet":       "FE",
		"GigabitEthernet":    "GE",
		"TenGigE":            "XGE",
		"TenGigabitEthernet": "XGE",
		"FortyGigE":          "FOE",
		"HundredGigE":        "HGE",
		"TwentyFiveGigE":     "TWE",
		"Vlanif":             "Vlanif",
		"Loopback":           "Loop",
	}
	for full, short := range fullToShort {
		if strings.HasPrefix(name, full) {
			return short + name[len(full):]
		}
	}

	// 已是合法全称/逻辑接口原样返回
	validPrefixes := []string{"FastEthernet", "GigabitEthernet", "TenGigE", "FortyGigE", "HundredGigE", "TwentyFiveGigE", "Vlan", "Vlanif", "Loopback", "NULL"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return name
		}
	}

	// 阶段 2: 小写短名 / 全小写 full name → 大写短名
	lowerName := strings.ToLower(name)
	type prefixMapping struct{ short, canonical string }
	// 按前缀长度倒序,避免短前缀误匹配长前缀(twe 早于 tw,et 早于 te)
	prefixList := []prefixMapping{
		{"gigabitethernet", "GE"},
		{"gigabitether", "GE"},
		{"hundredgige", "HGE"},
		{"twentyfivegige", "TWE"},
		{"fortygige", "FOE"},
		{"fastethernet", "FE"},
		{"tengige", "XGE"},
		{"loopback", "Loop"},
		{"twe", "TWE"},
		{"tw", "TWE"},
		{"et", "ET"},
		{"hge", "HGE"},
		{"fge", "FOE"},
		{"xe", "XGE"},
		{"ge", "GE"},
		{"gi", "GE"},
		{"te", "XGE"},
		{"fo", "FOE"},
		{"fa", "FE"},
		{"vlanif", "Vlanif"},
		{"vlan", "Vlan"},
		{"vl", "Vlan"},
		{"null", "NULL"},
	}
	for _, m := range prefixList {
		if strings.HasPrefix(lowerName, m.short) {
			suffix := lowerName[len(m.short):]
			// suffix 必须以数字开头,避免把乱码(短名+字母碎片+数字)误识别
			if len(suffix) > 0 && suffix[0] >= '0' && suffix[0] <= '9' {
				return m.canonical + suffix
			}
		}
	}

	return name
}
