package main

// 解析层自测（纯函数，不连设备）。
//
// 覆盖锐捷真机复测最容易失真的两处：
//   - 回显含 ANSI 光标控制序列；
//   - 接口块内混入设备错误行（% Invalid input 等）不能被当成配置行。

import (
	"reflect"
	"strings"
	"testing"
)

// rawRuijieConfig 模拟锐捷 `show running-config interface GigabitEthernet 1/1` 输出。
const rawRuijieConfig = `Building configuration...
Current configuration : 512 bytes
!
interface GigabitEthernet 1/1
 description 3D001
 switchport access vlan 318
 dot1x port-control auto
 dot1x mac-auth-bypass multi-user
 spanning-tree bpduguard enable
 spanning-tree portfast
 rldp port loop-detect shutdown-port
 switchport port-security binding 10.62.51.173
 switchport port-security
!
`

// expectedBaseline lead 实测取得的 9 行原始基线。
var expectedBaseline = baselineRuijieGE11

func TestIfaceConfigBlock(t *testing.T) {
	lines, errs, warns := ifaceConfigBlock(rawRuijieConfig, "GigabitEthernet 1/1")
	if len(errs) != 0 {
		t.Fatalf("不应有错误行: %v", errs)
	}
	if len(warns) != 0 {
		t.Fatalf("不应有告警: %v", warns)
	}
	if !reflect.DeepEqual(lines, expectedBaseline) {
		t.Fatalf("基线解析不符:\n got=%v\nwant=%v", lines, expectedBaseline)
	}
}

func TestIfaceConfigBlockStripsANSI(t *testing.T) {
	// 模拟 RGOS 回显里的光标控制序列
	dirty := "\x1b[2K\rinterface GigabitEthernet 1/1\r\n\x1b[1;32m description 3D001\x1b[0m\r\n!"
	lines, _, warns := ifaceConfigBlock(dirty, "GigabitEthernet 1/1")
	if len(warns) != 0 {
		t.Fatalf("不应有告警: %v", warns)
	}
	if len(lines) != 1 || lines[0] != "description 3D001" {
		t.Fatalf("ANSI 清洗失败: %v", lines)
	}
}

func TestIfaceConfigBlockIgnoresErrorLines(t *testing.T) {
	dirty := "interface GigabitEthernet 1/1\n description 3D001\n% Invalid input detected at '^' marker.\n"
	lines, errs, _ := ifaceConfigBlock(dirty, "GigabitEthernet 1/1")
	if len(lines) != 1 || lines[0] != "description 3D001" {
		t.Fatalf("错误行混入配置行: %v", lines)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "Invalid input") {
		t.Fatalf("错误行未被单独收集: %v", errs)
	}
}

func TestDiffConfig(t *testing.T) {
	after := append([]string{}, expectedBaseline...)
	after = append(after, "switchport port-security maximum 5")

	diff := diffConfig(expectedBaseline, after)
	if len(diff) != 1 || diff[0] != "多出: switchport port-security maximum 5" {
		t.Fatalf("diff 不符: %v", diff)
	}
	if len(diffConfig(expectedBaseline, expectedBaseline)) != 0 {
		t.Fatalf("相同集合 diff 应为空")
	}
	missing := diffConfig(expectedBaseline, expectedBaseline[:len(expectedBaseline)-1])
	if len(missing) != 1 || !strings.HasPrefix(missing[0], "缺失:") {
		t.Fatalf("缺失判定不符: %v", missing)
	}
}

func TestBuildWriteCmds(t *testing.T) {
	got := buildWriteCmds("GigabitEthernet 1/1", []string{
		"interface GigabitEthernet 1/1",
		"switchport port-security",
	})
	want := []string{
		"interface GigabitEthernet 1/1",
		"switchport port-security",
		"exit",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("写命令组装不符:\n got=%v\nwant=%v", got, want)
	}

	// 系统视图动作：不追加 interface 前缀，也不追加 exit
	sys := buildWriteCmds("GigabitEthernet 1/1", []string{"user-bind static ip-address 1.1.1.1"})
	if len(sys) != 1 {
		t.Fatalf("系统视图命令不应被改写: %v", sys)
	}
}

func TestRuijieMAC(t *testing.T) {
	cases := map[string]string{
		"00:11:22:33:44:55": "0011.2233.4455",
		"00-11-22-33-44-55": "0011.2233.4455",
		"001122334455":      "0011.2233.4455",
	}
	for in, want := range cases {
		if got := ruijieMAC(in); got != want {
			t.Fatalf("ruijieMAC(%q)=%q want %q", in, got, want)
		}
	}
}

func TestExpandInterfaceName(t *testing.T) {
	if got := expandInterfaceName("GE1/1", "ruijie"); got != "GigabitEthernet 1/1" {
		t.Fatalf("锐捷接口展开错误: %q", got)
	}
	if got := expandInterfaceName("GE1/0/0", "huawei"); got != "GE1/0/0" {
		t.Fatalf("华为不应展开全称: %q", got)
	}
}

// TestUndoSymmetry 锁定还原命令的对称性（还原后 diff 必须为空的前提）。
func TestUndoSymmetry(t *testing.T) {
	p := actionParams{CliIface: "GigabitEthernet 1/1", VLAN: 319, MAC: "00:11:22:33:44:55",
		IP: "10.62.255.254", BaselineVLAN: 318}

	// dot1x_enable：设备会自动清 default-user-limit → undo 必须无条件下发 no dot1x port-control auto
	got := actionRegistry["dot1x_enable"].Undo(expectedBaseline, p)
	if len(got) != 1 || got[0] != "no dot1x port-control auto" {
		t.Fatalf("dot1x_enable undo 不对称: %v", got)
	}

	// dot1x_disable：基线有 port-control auto → 恢复它；基线无 limit 行 → 不补 limit
	got = actionRegistry["dot1x_disable"].Undo(expectedBaseline, p)
	if len(got) != 1 || got[0] != "dot1x port-control auto" {
		t.Fatalf("dot1x_disable undo 不符: %v", got)
	}

	// set_access_vlan：目标 VLAN != 基线 → 先 clear PVID
	got = actionRegistry["set_access_vlan"].Undo(expectedBaseline, p)
	if len(got) != 1 || got[0] != "no switchport access vlan" {
		t.Fatalf("set_access_vlan undo 不符: %v", got)
	}
	// 目标 VLAN == 基线 → 无需 clear
	pSame := p
	pSame.VLAN = 318
	if got = actionRegistry["set_access_vlan"].Undo(expectedBaseline, pSame); len(got) != 0 {
		t.Fatalf("同值时不应 clear: %v", got)
	}

	// port_binding：undo 必须与产品 remove 渲染同形（MAC 在前、IP 在后）
	got = actionRegistry["port_binding"].Undo(expectedBaseline, p)
	if len(got) != 1 || got[0] != "no switchport port-security binding 0011.2233.4455 10.62.255.254" {
		t.Fatalf("port_binding undo 不符: %v", got)
	}
}

func TestParseVlanIDs(t *testing.T) {
	out := `VLAN Name                             Status    Ports
---- -------------------------------- --------- -------------------------------
1    default                          active    Gi0/1, Gi0/2
318  VLAN0318                          active    Gi1/1
399  VLAN0399                          active
`
	vlans := parseVlanIDs(out)
	for _, want := range []int{1, 318, 399} {
		if !vlans[want] {
			t.Fatalf("VLAN %d 未解析到: %v", want, sortedIntKeys(vlans))
		}
	}
	if vlans[4095] {
		t.Fatalf("非法 VLAN 被解析: %v", sortedIntKeys(vlans))
	}
	// 自动选择：应跳过基线 VLAN 318
	if got := pickTargetVLAN(vlans, 318); got != 399 {
		t.Fatalf("自动 VLAN 选择错误: %d", got)
	}
}
