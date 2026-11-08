package portcollection

// FIX-04（v1.53 Phase 138）：物理参数命令模板 3 厂商 × 3 动作单测。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestPhysical138_ThreeVendorsThreeActions 锁定 3 厂商 × 3 动作命令序列。
//
// 三家接口视图下 speed/duplex/mtu 关键字一致（wrapInterface 统一进 interface），
// 本测试同时验证 wrapInterface 前缀存在 + 参数值正确渲染。
func TestPhysical138_ThreeVendorsThreeActions(t *testing.T) {
	cases := []struct {
		name   string
		vendor models.DeviceVendor
		action PortAction
		params PortTemplateParams
		want   []string
	}{
		// ---- 速率 ----
		{"huawei speed 1000", models.VendorHuawei, ActionSetSpeed,
			PortTemplateParams{InterfaceName: "GE0/0/1", Speed: 1000},
			[]string{"interface GE0/0/1", "speed 1000"}},
		{"h3c speed auto", models.VendorH3C, ActionSetSpeed,
			PortTemplateParams{InterfaceName: "GE0/0/1", SpeedMode: SpeedModeAuto},
			[]string{"interface GE0/0/1", "speed auto"}},
		{"ruijie speed 100", models.VendorRuijie, ActionSetSpeed,
			PortTemplateParams{InterfaceName: "GigabitEthernet 1/0/1", Speed: 100},
			[]string{"interface GigabitEthernet 1/0/1", "speed 100"}},
		// ---- 双工 ----
		{"huawei duplex full", models.VendorHuawei, ActionSetDuplex,
			PortTemplateParams{InterfaceName: "GE0/0/1", Duplex: DuplexFull},
			[]string{"interface GE0/0/1", "duplex full"}},
		{"h3c duplex half", models.VendorH3C, ActionSetDuplex,
			PortTemplateParams{InterfaceName: "GE0/0/1", Duplex: DuplexHalf},
			[]string{"interface GE0/0/1", "duplex half"}},
		{"ruijie duplex auto", models.VendorRuijie, ActionSetDuplex,
			PortTemplateParams{InterfaceName: "GE0/0/1", Duplex: DuplexAuto},
			[]string{"interface GE0/0/1", "duplex auto"}},
		// ---- MTU ----
		{"huawei mtu 9000", models.VendorHuawei, ActionSetMTU,
			PortTemplateParams{InterfaceName: "GE0/0/1", MTU: 9000},
			[]string{"interface GE0/0/1", "mtu 9000"}},
		{"h3c mtu 1500", models.VendorH3C, ActionSetMTU,
			PortTemplateParams{InterfaceName: "GE0/0/1", MTU: 1500},
			[]string{"interface GE0/0/1", "mtu 1500"}},
		{"ruijie mtu 9216", models.VendorRuijie, ActionSetMTU,
			PortTemplateParams{InterfaceName: "GE0/0/1", MTU: 9216},
			[]string{"interface GE0/0/1", "mtu 9216"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmds, err := RenderCommand(tc.vendor, tc.action, tc.params)
			require.NoError(t, err)
			assert.Equal(t, tc.want, cmds)
		})
	}
}

// TestPhysical138_RendererRejectsInvalid 渲染层对非法枚举/范围兜底拒绝
// （service 层 validator 是 400 真相源，renderer 是第二道防线）。
// Phase 143：renderer 改走 CheckPhysicalSupport（与 service 同源），错误文案改为
// 画像值域表述；断言同步为「拒绝 + 值域信息」，语义（拒非法值）不变。
func TestPhysical138_RendererRejectsInvalid(t *testing.T) {
	_, err := RenderCommand(models.VendorHuawei, ActionSetSpeed, PortTemplateParams{InterfaceName: "GE0/0/1", Speed: 999})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)
	assert.Contains(t, err.Error(), "speed 999")

	_, err = RenderCommand(models.VendorH3C, ActionSetDuplex, PortTemplateParams{InterfaceName: "GE0/0/1", Duplex: "both"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)
	assert.Contains(t, err.Error(), `duplex "both"`)

	_, err = RenderCommand(models.VendorRuijie, ActionSetMTU, PortTemplateParams{InterfaceName: "GE0/0/1", MTU: 100})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)
	assert.Contains(t, err.Error(), "out of range")
}

// TestPhysical138_AllowedHelpers 导出 helper 单一真相源。
func TestPhysical138_AllowedHelpers(t *testing.T) {
	assert.True(t, SpeedAllowed(10))
	assert.True(t, SpeedAllowed(10000))
	assert.False(t, SpeedAllowed(0))
	assert.False(t, SpeedAllowed(40))
	assert.True(t, DuplexAllowed(DuplexFull))
	assert.True(t, DuplexAllowed(DuplexAuto))
	assert.False(t, DuplexAllowed(""))
	assert.Equal(t, 1280, MTUMin)
	assert.Equal(t, 9216, MTUMax)
}

// TestPhysical138_RequiresInterfaceView 物理参数动作需 interface 视图（能力矩阵默认支持）。
func TestPhysical138_RequiresInterfaceView(t *testing.T) {
	for _, v := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		for _, a := range []PortAction{ActionSetSpeed, ActionSetDuplex, ActionSetMTU} {
			assert.True(t, RequiresInterfaceView(v, a), "vendor=%s action=%s", v, a)
			require.NoError(t, CheckActionSupport(v, "", a), "vendor=%s action=%s 应默认支持", v, a)
		}
	}
}

// ==================== Phase 143 (PORT-02) 型号维度模板测试 ====================

// TestPhysical143_DefaultProfileUnverified 未取证默认画像：三家命令体零回归 +
// Evidence="" / level=unverified（默认放行 ≠ 有证据）。
func TestPhysical143_DefaultProfileUnverified(t *testing.T) {
	for _, v := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		prof := ResolvePhysicalProfile(v, "UNKNOWN-MODEL")
		assert.Equal(t, "unverified", PhysicalEvidenceLevel(prof), "vendor=%s", v)
		assert.Empty(t, prof.Evidence, "Evidence 空串 = 未取证，严禁写成已验证")
		assert.Equal(t, []int{10, 100, 1000, 10000}, prof.AllowedSpeeds)
		assert.Equal(t, MTUMin, prof.MTUMin)
		assert.Equal(t, MTUMax, prof.MTUMax)
	}
	// 三家默认命令体与 FIX-04 现状完全一致（单条物理写零回归）
	cmds, err := RenderCommand(models.VendorHuawei, ActionSetMTU, PortTemplateParams{
		InterfaceName: "GE0/0/1", MTU: 9000,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"interface GE0/0/1", "mtu 9000"}, cmds)
}

// TestPhysical143_CiscoExplicitUnsupported Cisco 物理三动作显式 Unsupported
// （渲染层未实现 + 无真机取证依据，不靠 ErrUnsupportedVendor 含糊兜底）。
func TestPhysical143_CiscoExplicitUnsupported(t *testing.T) {
	for _, action := range []PortAction{ActionSetSpeed, ActionSetDuplex, ActionSetMTU} {
		_, err := RenderCommand(models.VendorCisco, action, PortTemplateParams{
			InterfaceName: "GE0/0/1", Speed: 1000, Duplex: DuplexFull, MTU: 9000,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrActionNotSupported, "Cisco %s 必须显式拒绝", action)
		assert.Contains(t, err.Error(), "渲染层未实现")
		assert.Contains(t, err.Error(), "无真机取证依据")
	}
}
