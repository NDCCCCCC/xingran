package portcollection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== Phase 129 真机 UAT 修复：厂商/型号能力矩阵测试 ======================
// 真机事实锚定：2026-11-01 华为 S8700 10.62.25.253 `?` 探测（task-26 卡面）。

// TestCheckActionSupport_S8700 真机事实 #2/#3：S8700 无 trunk/hybrid/max-mac-num。
func TestCheckActionSupport_S8700(t *testing.T) {
	const s8700 = "S8700-10"
	// 不支持集合（每个都有真机依据 reason）
	for _, action := range []PortAction{TrunkSetNative, TrunkSetAllowed, TrunkSetHybrid, PortSecMaxMac} {
		err := CheckActionSupport(models.VendorHuawei, s8700, action)
		require.Error(t, err, "%s 在 S8700 应被拒绝", action)
		assert.ErrorIs(t, err, ErrActionNotSupported)
		assert.Contains(t, err.Error(), "2026-11-01 真机", "reason 必须携带真机依据")
	}
	// 支持集合不受影响
	for _, action := range []PortAction{ActionShutdown, ActionUndoShutdown, ActionSetAccessVLAN,
		ActionDescription, ActionDot1xEnable, ActionDot1xDisable, ActionPortBinding,
		PortSecEnable, PortSecDisable, PortSecSticky, PortSecStaticBind} {
		assert.NoError(t, CheckActionSupport(models.VendorHuawei, s8700, action), "%s 应支持", action)
	}
}

// TestCheckActionSupport_OtherHuaweiModels 非匹配型号沿用厂商基线（保守兼容）。
func TestCheckActionSupport_OtherHuaweiModels(t *testing.T) {
	for _, model := range []string{"", "CE12800", "S5735", "unknown-box"} {
		for _, action := range []PortAction{TrunkSetNative, TrunkSetAllowed, PortSecMaxMac} {
			assert.NoError(t, CheckActionSupport(models.VendorHuawei, model, action),
				"model=%q 未命中 S87 规则应默认支持", model)
		}
	}
}

// TestCheckActionSupport_RuijieHybrid 锐捷 hybrid 渲染层不注册（既有语义不重复表达）。
func TestCheckActionSupport_RuijieHybrid(t *testing.T) {
	assert.NoError(t, CheckActionSupport(models.VendorRuijie, "RS8607E", TrunkSetHybrid),
		"能力矩阵放行；拒绝由渲染层 ErrUnknownAction 承担")
}

// TestRequiresInterfaceView 真机事实 #1：华为 port_binding 是系统视图命令。
func TestRequiresInterfaceView(t *testing.T) {
	assert.False(t, RequiresInterfaceView(models.VendorHuawei, ActionPortBinding),
		"user-bind 系统视图执行（接口视图 Unrecognized）")
	assert.True(t, RequiresInterfaceView(models.VendorHuawei, ActionShutdown))
	assert.True(t, RequiresInterfaceView(models.VendorRuijie, ActionPortBinding))
}

// TestRenderHuaweiPortBinding_SystemView 渲染无 interface 前缀（真机事实 #1）。
func TestRenderHuaweiPortBinding_SystemView(t *testing.T) {
	cmds, err := RenderCommand(models.VendorHuawei, ActionPortBinding, PortTemplateParams{
		InterfaceName: "GE1/0/0",
		BindOp:        "add",
		IPAddress:     "10.62.25.5",
		MACAddress:    "AA-BB-CC-DD-EE-FF",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF"}, cmds)
}
