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
		ActionDescription, ActionDot1xEnable, ActionDot1xDisable,
		PortSecEnable, PortSecDisable, PortSecSticky} {
		assert.NoError(t, CheckActionSupport(models.VendorHuawei, s8700, action), "%s 应支持", action)
	}
	// 二轮真机：static_bind 整动作拒绝 + port_binding IP+MAC 组合拒绝（参数级）、IP-only 放行
	assert.ErrorIs(t, CheckActionSupport(models.VendorHuawei, s8700, PortSecStaticBind), ErrActionNotSupported)
	assert.ErrorIs(t, CheckActionSupportParams(models.VendorHuawei, s8700, ActionPortBinding, "AA-BB-CC-DD-EE-FF"), ErrActionNotSupported)
	assert.NoError(t, CheckActionSupportParams(models.VendorHuawei, s8700, ActionPortBinding, ""))
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

// TestCheckActionSupport_RuijieHybrid 锐捷 hybrid 能力矩阵放行。
//
// 2026-11-02 起渲染层也已按真机形态注册（switchport mode hybrid +
// switchport hybrid allowed vlan tagged/untagged + native vlan），矩阵与渲染不再割裂；
// 锐捷 RS8607E 的 supported 条目见 device_capability.go（真机 `?` 帮助依据）。
func TestCheckActionSupport_RuijieHybrid(t *testing.T) {
	assert.NoError(t, CheckActionSupport(models.VendorRuijie, "RS8607E", TrunkSetHybrid),
		"锐捷 RS8607E hybrid 真机支持（`?` 帮助确认），能力矩阵必须放行")
	ev := SupportEvidence(models.VendorRuijie, "RS8607E", TrunkSetHybrid)
	assert.Contains(t, ev, "switchport hybrid", "supported 依据必须写明真机命令形态")
	// 二次取证修正：依据必须写清 tagged/untagged 关键字与 native 单值，
	// 防止依据回退到"allowed vlan <list>"这种设备会拒绝的形态。
	// 三次取证修正：tagged 必带 only、untagged 不能带 only、还原走模式回退。
	assert.Contains(t, ev, "allowed vlan only tagged", "依据必须写明 tagged 带 only（缺 only = 不落地）")
	assert.Contains(t, ev, "allowed vlan untagged", "依据必须写明 untagged 关键字")
	assert.Contains(t, ev, "native vlan", "依据必须写明 native 单值形态")
	assert.Contains(t, ev, "switchport mode access", "依据必须写明还原走模式回退")
	assert.Contains(t, ev, "802.1X", "依据必须写明 802.1X 互斥（由预检层拦截）")

	// ★ 2026-11-02 补漏配：库里真实存在 RG8607E / RG-S8607E-iSeries 两种命名，
	//   且真机确认其 hybrid 形态与 RS8607E 一致 ⇒ 矩阵必须一并放行。
	for _, model := range []string{"RS8607E", "RG8607E", "RG-S8607E-iSeries"} {
		assert.NoError(t, CheckActionSupport(models.VendorRuijie, model, TrunkSetHybrid),
			"锐捷 "+model+" hybrid 真机支持（形态与 RS8607E 一致），能力矩阵必须放行")
	}
}

// TestCheckActionSupport_Cisco 2026-11-02 裁定：Cisco 的 trunk/hybrid **显式标 Unsupported**。
//
// ★ 为什么必须显式（lead 裁定理由三条）：
//  1. 矩阵「无条目即默认放行」= **把推测当默认**（我们要消灭的正是这个）；
//  2. 渲染层本来就没有 Cisco 的 trunk/hybrid 实现 ⇒ 事实层面就是不支持；
//  3. 依据要诚实：写「渲染层未实现 + 无真机取证依据」，**不伪造真机结论**。
func TestCheckActionSupport_Cisco(t *testing.T) {
	for _, action := range []PortAction{TrunkSetNative, TrunkSetAllowed, TrunkSetHybrid} {
		// 型号已知与否都要拒绝（厂商级规则与型号无关）
		for _, model := range []string{"Catalyst 9300", "C9500-48Y4C", ""} {
			err := CheckActionSupport(models.VendorCisco, model, action)
			assert.ErrorIs(t, err, ErrActionNotSupported,
				"Cisco 的 %s 必须被显式拒绝（model=%q）", action, model)
			assert.Contains(t, err.Error(), "无真机取证依据",
				"拒绝依据必须诚实标注「无真机取证依据」")
			assert.Contains(t, err.Error(), "渲染层未实现",
				"拒绝依据必须说明「渲染层未实现」")
		}
	}

	// Cisco 的其它动作（产品层有渲染）**不受影响** —— 不能一刀切
	for _, action := range []PortAction{
		ActionShutdown, ActionDescription, ActionSetAccessVLAN, ActionPortBinding,
		PortSecEnable, PortSecMaxMac, PortSecStaticBind,
	} {
		assert.NoError(t, CheckActionSupport(models.VendorCisco, "Catalyst 9300", action),
			"Cisco 的 %s 不应被这次裁定波及（产品层有渲染）", action)
	}
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

// TestSupportEvidence_CiscoVendorUnsupported 厂商级 unsupported 通道的依据返回路径。
//
// ★ 为何要单独测：CheckActionSupport 只走 lookupSupport（覆盖了“支持/不支持判定”），
//
//	而 **SupportEvidence 的厂商级命中分支**（“为何拒绝”的文案来源）是另一条路径 ⇒
//	若不单独测，它会因为「写路径先被矩阵拢截」而永远不被执行 ⇒ 那段代码会无条件潮化。
func TestSupportEvidence_CiscoVendorUnsupported(t *testing.T) {
	for _, action := range []PortAction{TrunkSetNative, TrunkSetAllowed, TrunkSetHybrid} {
		ev := SupportEvidence(models.VendorCisco, "", action)
		assert.NotEmpty(t, ev, "Cisco 的 %s 厂商级拒绝必须有依据文本", action)
		assert.Contains(t, ev, "渲染层未实现",
			"依据必须说明「渲染层未实现」")
		assert.Contains(t, ev, "无真机取证依据",
			"依据必须诚实标注「无真机取证依据」")
	}

	// 型号级依据仍然受影响（厂商级通道不得覆盖型号级 supported）
	ev := SupportEvidence(models.VendorRuijie, "RS8607E", TrunkSetHybrid)
	assert.Contains(t, ev, "真机",
		"锐捷 RS8607E 的 hybrid 依据应仍为型号级真机依据")
}
