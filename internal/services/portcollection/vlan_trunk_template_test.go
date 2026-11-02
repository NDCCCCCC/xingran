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

// TestRenderRuijieTrunkSetHybrid 锐捷 hybrid 渲染形态锁定。
func TestRenderRuijieTrunkSetHybrid(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{10},
		HybridTagged:   []int{30, 40},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport mode hybrid",               // 前置切模式（设备要求）
		"switchport hybrid native vlan 10",     // native 承载 untagged
		"switchport hybrid allowed vlan 30,40", // allowed 承载 tagged 列表
	}, got)
}

// TestRenderRuijieTrunkSetHybridOnlyTagged 只给 tagged 时不渲染 native 行。
func TestRenderRuijieTrunkSetHybridOnlyTagged(t *testing.T) {
	got, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName: "GigabitEthernet 1/1",
		HybridTagged:  []int{30},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"interface GigabitEthernet 1/1",
		"switchport mode hybrid",
		"switchport hybrid allowed vlan 30",
	}, got)
}

// TestRenderRuijieTrunkSetHybridNativeSingleValue native 单值约束。
//
// 锐捷 native 只能表达一个 untagged VLAN；传多个必须**拒绝**（而不是静默取第一个），
// 否则设备会把端口切到 hybrid 模式却只配上部分 VLAN。
func TestRenderRuijieTrunkSetHybridNativeSingleValue(t *testing.T) {
	_, err := RenderCommand(models.VendorRuijie, TrunkSetHybrid, PortTemplateParams{
		InterfaceName:  "GigabitEthernet 1/1",
		HybridUntagged: []int{10, 20},
		HybridTagged:   []int{30},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrActionNotSupported, "参数组合不受支持 → 必须走 400 语义")
	assert.Contains(t, err.Error(), "native")
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
