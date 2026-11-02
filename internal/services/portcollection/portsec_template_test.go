package portcollection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== Phase 129 端口安全渲染模板（包内直测） ======================
//
// 为什么放在 portcollection 包内而不是 portwrite 包：
// Go 的 coverage 按「被测包」统计 —— portwrite 包的测试虽然会执行
// portcollection.renderPortSec，但那块代码的覆盖只记在 portwrite 的 profile 里，
// portcollection 自己的 profile 仍是 0。diff coverage 门禁读的是各包自己的
// profile，所以端口安全渲染必须在 portcollection 包内有直测。
//
// 真机依据（2026-11-01）：
//   - 华为 S8700：port-security mac-address sticky ? 只接受 [H-H-H | sticky]，
//     无 vlan 参数 ⇒ 静态 MAC 绑定渲染必须去掉 vlan
//   - 锐捷 RS8607E：switchport port-security mac-address <mac> vlan <vlan> 实测无错误
//   - H3C Comware：port-security mac-address security <mac> vlan <vlan>

// TestRenderPortSec_AllVendorsAllActions 3 厂商 × 5 动作的渲染结果逐条锁定。
func TestRenderPortSec_AllVendorsAllActions(t *testing.T) {
	const mac = "AA:BB:CC:DD:EE:FF"

	cases := []struct {
		name   string
		vendor models.DeviceVendor
		action PortAction
		params PortTemplateParams
		want   []string
	}{
		{
			name: "华为 port-security enable", vendor: models.VendorHuawei, action: PortSecEnable,
			want: []string{"interface GE1/0/1", "port-security enable"},
		},
		{
			name: "华为 undo port-security enable", vendor: models.VendorHuawei, action: PortSecDisable,
			want: []string{"interface GE1/0/1", "undo port-security enable"},
		},
		{
			name: "华为 max-mac-num", vendor: models.VendorHuawei, action: PortSecMaxMac,
			params: PortTemplateParams{MaxMACCount: 5},
			want:   []string{"interface GE1/0/1", "port-security max-mac-num 5"},
		},
		{
			name: "华为 sticky", vendor: models.VendorHuawei, action: PortSecSticky,
			want: []string{"interface GE1/0/1", "port-security mac-address sticky"},
		},
		{
			// 真机：S8700 的 sticky ? 只接受 [H-H-H | sticky]，无 vlan 参数
			name: "华为 static bind 去 vlan", vendor: models.VendorHuawei, action: PortSecStaticBind,
			params: PortTemplateParams{MACAddress: mac, VLANID: 100},
			want:   []string{"interface GE1/0/1", "port-security mac-address sticky AA-BB-CC-DD-EE-FF"},
		},
		{
			name: "H3C port-security enable", vendor: models.VendorH3C, action: PortSecEnable,
			want: []string{"interface GE1/0/1", "port-security enable"},
		},
		{
			name: "H3C undo port-security enable", vendor: models.VendorH3C, action: PortSecDisable,
			want: []string{"interface GE1/0/1", "undo port-security enable"},
		},
		{
			name: "H3C max-mac-count", vendor: models.VendorH3C, action: PortSecMaxMac,
			params: PortTemplateParams{MaxMACCount: 5},
			want:   []string{"interface GE1/0/1", "port-security max-mac-count 5"},
		},
		{
			name: "H3C sticky", vendor: models.VendorH3C, action: PortSecSticky,
			want: []string{"interface GE1/0/1", "port-security mac-address sticky"},
		},
		{
			name: "H3C static bind 带 vlan", vendor: models.VendorH3C, action: PortSecStaticBind,
			params: PortTemplateParams{MACAddress: mac, VLANID: 100},
			want:   []string{"interface GE1/0/1", "port-security mac-address security AA-BB-CC-DD-EE-FF vlan 100"},
		},
		{
			name: "锐捷 switchport port-security", vendor: models.VendorRuijie, action: PortSecEnable,
			want: []string{"interface GE1/0/1", "switchport port-security"},
		},
		{
			name: "锐捷 no switchport port-security", vendor: models.VendorRuijie, action: PortSecDisable,
			want: []string{"interface GE1/0/1", "no switchport port-security"},
		},
		{
			name: "锐捷 maximum", vendor: models.VendorRuijie, action: PortSecMaxMac,
			params: PortTemplateParams{MaxMACCount: 5},
			want:   []string{"interface GE1/0/1", "switchport port-security maximum 5"},
		},
		{
			name: "锐捷 sticky", vendor: models.VendorRuijie, action: PortSecSticky,
			want: []string{"interface GE1/0/1", "switchport port-security mac-address sticky"},
		},
		{
			// 真机（2026-11-01 锐捷 RS8607E）：mac-address <mac> vlan <vlan> 实测无错误
			name: "锐捷 static bind 带 vlan 且 MAC 用点分格式", vendor: models.VendorRuijie, action: PortSecStaticBind,
			params: PortTemplateParams{MACAddress: mac, VLANID: 100},
			want:   []string{"interface GE1/0/1", "switchport port-security mac-address aabb.ccdd.eeff vlan 100"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := tc.params
			params.InterfaceName = "GE1/0/1"
			cmds, err := RenderCommand(tc.vendor, tc.action, params)
			require.NoError(t, err, "%s 渲染不应失败", tc.name)
			assert.Equal(t, tc.want, cmds)
		})
	}
}

// TestRenderPortSec_HuaweiStaticBindIgnoresVLAN 真机语义：华为静态 MAC 绑定
// 渲染不带 vlan（VLANID 字段被显式忽略），不能因为传了 VLANID 就拼上 vlan 参数。
func TestRenderPortSec_HuaweiStaticBindIgnoresVLAN(t *testing.T) {
	for _, vlan := range []int{0, 100, 4094} {
		cmds, err := RenderCommand(models.VendorHuawei, PortSecStaticBind, PortTemplateParams{
			InterfaceName: "GE1/0/0",
			MACAddress:    "AA-BB-CC-DD-EE-FF",
			VLANID:        vlan,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"interface GE1/0/0", "port-security mac-address sticky AA-BB-CC-DD-EE-FF"}, cmds,
			"华为静态 MAC 绑定不得带 vlan 参数（vlan=%d）", vlan)
	}
}

// TestRenderPortSec_MaxMacOutOfRange maxMacCount 必须落在 1-4096。
func TestRenderPortSec_MaxMacOutOfRange(t *testing.T) {
	for _, count := range []int{0, -1, 4097, 99999} {
		for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
			cmds, err := RenderCommand(vendor, PortSecMaxMac, PortTemplateParams{
				InterfaceName: "GE1/0/1",
				MaxMACCount:   count,
			})
			require.Error(t, err, "maxMacCount=%d 必须被拒绝", count)
			assert.Nil(t, cmds)
			assert.Contains(t, err.Error(), "out of range 1-4096")
		}
	}
}

// TestRenderPortSec_StaticBindRequiresMAC 静态 MAC 绑定必须带 MAC。
func TestRenderPortSec_StaticBindRequiresMAC(t *testing.T) {
	for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		cmds, err := RenderCommand(vendor, PortSecStaticBind, PortTemplateParams{
			InterfaceName: "GE1/0/1",
			VLANID:        100,
		})
		require.Error(t, err, "缺少 MAC 必须被拒绝")
		assert.Nil(t, cmds)
		assert.Contains(t, err.Error(), "macAddress is required for static bind")
	}
}

// TestRenderPortSec_StaticBindVlanOutOfRange 锐捷 / H3C 的静态绑定要求 vlan 1-4094。
//
// 注意：华为不走这条路径（华为静态绑定不带 vlan，S8700 真机语义）。
func TestRenderPortSec_StaticBindVlanOutOfRange(t *testing.T) {
	for _, vendor := range []models.DeviceVendor{models.VendorH3C, models.VendorRuijie} {
		for _, vlan := range []int{0, -1, 4095, 5000} {
			cmds, err := RenderCommand(vendor, PortSecStaticBind, PortTemplateParams{
				InterfaceName: "GE1/0/1",
				MACAddress:    "AA:BB:CC:DD:EE:FF",
				VLANID:        vlan,
			})
			require.Error(t, err, "vendor=%s vlan=%d 必须被拒绝", vendor, vlan)
			assert.Nil(t, cmds)
			assert.Contains(t, err.Error(), "out of range 1-4094")
		}
	}
}

// TestRenderPortSec_UnknownAction 未注册的 port-security 动作必须显式报错
// （而不是静默返回空命令序列）。
func TestRenderPortSec_UnknownAction(t *testing.T) {
	cmds, err := renderPortSec(PortTemplateParams{InterfaceName: "GE1/0/1"}, models.VendorHuawei, PortAction("portsec_bogus"))
	require.Error(t, err)
	assert.Nil(t, cmds)
	assert.Contains(t, err.Error(), "unknown port-security action")
}

// TestToCiscoMACFormat MAC 归一为锐捷/Cisco 点分格式；非法输入返回空串。
func TestToCiscoMACFormat(t *testing.T) {
	assert.Equal(t, "aabb.ccdd.eeff", toCiscoMACFormat("AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, "aabb.ccdd.eeff", toCiscoMACFormat("aa-bb-cc-dd-ee-ff"))
	assert.Equal(t, "aabb.ccdd.eeff", toCiscoMACFormat("aabb.ccdd.eeff"))
	assert.Equal(t, "aabb.ccdd.eeff", toCiscoMACFormat("AA BB CC DD EE FF"))
	assert.Equal(t, "", toCiscoMACFormat(""), "空输入返回空串")
	assert.Equal(t, "", toCiscoMACFormat("AA:BB:CC:DD:EE"), "长度不足 12 位返回空串")
	assert.Equal(t, "", toCiscoMACFormat("ZZ:BB:CC:DD:EE:FF"), "非 hex 返回空串")
}
