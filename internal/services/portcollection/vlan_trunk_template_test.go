package portcollection

// 锐捷 trunk/hybrid 渲染真机口径锁定（2026-11-02 `?` 帮助取证）。
//
// 取证原文（锐捷 RS8607E GigabitEthernet 1/1，零配置变更）：
//
//	switchport mode ?           → access / dot1q-tunnel / hybrid / private-vlan / trunk / uplink
//	switchport hybrid ?        → allowed / native
//	switchport hybrid native ?  → vlan
//	switchport hybrid allowed ? → vlan
//	switchport trunk ?          → native / allowed
//	switchport trunk native ?   → vlan
//
// ⇒ hybrid 命令形态：switchport mode hybrid（前置）+ switchport hybrid native vlan <id>
//   + switchport hybrid allowed vlan <list>；native 单值。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestRenderRuijieTrunkSetHybrid 锐捷 hybrid 渲染形态锁定（2026-11-02 第二次真机取证）。
//
// 真机证据：
//
//	switchport hybrid allowed vlan ? → add / only / remove / tagged / untagged
//	    ⇒ 列表必须带 tagged / untagged 关键字
//	switchport hybrid native vlan ? → <1-4094> Vid
//	    ⇒ native 是单值 VID，与 tagged/untagged 两个列表解耦
//
// ★ 第三次真机取证（2026-11-02，RG8607E GE1/25）追加：
//
//	switchport hybrid allowed vlan tagged 3990     → 无错误但【不落地】
//	switchport hybrid allowed vlan add tagged 3990 → 无错误但【不落地】
//	switchport hybrid allowed vlan only tagged 3990 → ✅【落地】
//	   ⇒ 列表必须带 **only** 关键字（缺 only = 命令成功但配置不变，最阴险）
func TestRenderRuijieTrunkSetHybrid(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{10, 20},
		HybridTagged:   []int{30, 40},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport mode hybrid",                           // 前置切模式（设备要求）
		"switchport hybrid allowed vlan only tagged 30,40", // tagged 列表（必须带 only + tagged）
		"switchport hybrid allowed vlan untagged 10,20",    // untagged 列表（★ 不能带 only）
	}, got)
}

// TestRenderRuijieTrunkSetHybridNative 锐捷 hybrid 的 native 单值形态：
// `switchport hybrid native vlan <id>` —— 与 tagged/untagged 两个列表解耦的独立项。
func TestRenderRuijieTrunkSetHybridNative(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1",
		HybridTagged:  []int{30},
		NativeVLAN:    10,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport mode hybrid",
		"switchport hybrid allowed vlan only tagged 30",
		"switchport hybrid native vlan 10",
	}, got)
}

// TestRenderRuijieTrunkSetHybridOnlyTagged 只给 tagged 时不渲染 untagged 行。
func TestRenderRuijieTrunkSetHybridOnlyTagged(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1",
		HybridTagged:  []int{30},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport mode hybrid",
		"switchport hybrid allowed vlan only tagged 30",
	}, got)
}

// TestRenderRuijieTrunkSetHybridMultipleUntaggedAllowed 多 untagged 合法。
//
// 第二次真机取证前，untagged 曾被映射到 native（单值）→ 当时按"多 untagged 拒绝"
// 处理。现在 untagged 映射到 `allowed vlan untagged <list>`（设备明确支持列表）⇒
// 多个 untagged 合法；继续拒绝等于把设备支持的能力禁掉。
func TestRenderRuijieTrunkSetHybridMultipleUntaggedAllowed(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{10, 20},
		HybridTagged:   []int{30},
	})
	require.NoError(t, err, "锐捷 allowed vlan untagged 是列表形态，多个 untagged 合法")
	assert.Contains(t, got, "switchport hybrid allowed vlan untagged 10,20")
}

// TestRenderRuijieTrunkSetHybridEmptyList 两个列表都空 → 拒绝。
func TestRenderRuijieTrunkSetHybridEmptyList(t *testing.T) {
	_, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1",
	})
	require.Error(t, err)
}

// TestRenderRuijieTrunkSetHybridVlanOutOfRange 渲染层的 VLAN 范围防御分支。
//
// service 层 validateTrunkRequest 会先拦越界（ErrVlanIdOutOfRange），但渲染层
// 仍需自带防御（与华为/H3C 同款：渲染层是最后一道防线，防止绕过 service 直调）。
func TestRenderRuijieTrunkSetHybridVlanOutOfRange(t *testing.T) {
	// untagged 越界
	_, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{0},
		HybridTagged:   []int{30},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")

	// tagged 越界
	_, err = RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{10},
		HybridTagged:   []int{4095},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")

	// native 单值越界（native 是与两个列表解耦的独立参数，同样要带范围防御）
	_, err = RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1",
		HybridTagged:  []int{30},
		NativeVLAN:    4095,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestRenderRuijieTrunkUnchanged 锐捷 trunk 分支核对：真机 `?` 取证未推翻现有实现，
// 渲染保持不变（native → `switchport trunk native vlan N`；
// allowed → `switchport trunk allowed vlan add|remove|<list>`）。
func TestRenderRuijieTrunkUnchanged(t *testing.T) {
	native, err := RenderCommand(models.VendorRuijie, TrunkSetNative, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1", NativeVLAN: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport trunk native vlan 10",
	}, native)

	allowed, err := RenderCommand(models.VendorRuijie, TrunkSetAllowed, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1", AllowedVLANs: []int{10, 20}, AllowedMode: AllowedModeAdd,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport trunk allowed vlan add 10,20",
	}, allowed)
}

// TestRenderHuaweiH3CHybridUnchanged 华为/H3C hybrid 渲染不受锐捷改动影响（回归）。
func TestRenderHuaweiH3CHybridUnchanged(t *testing.T) {
	huawei, err := RenderCommand(models.VendorHuawei, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GE1/0/0",
		HybridUntagged: []int{10, 20},
		HybridTagged:   []int{30},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GE1/0/0",
		"port link-type hybrid",
		"port hybrid tagged vlan 30",
		"port hybrid untagged vlan 10 20",
	}, huawei)

	h3c, err := RenderCommand(models.VendorH3C, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet1/0/1",
		HybridUntagged: []int{10},
		HybridTagged:   []int{30},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet1/0/1",
		"port link-type hybrid",
		"port hybrid vlan 30",
		"port hybrid vlan untagged 10",
	}, h3c)
}
