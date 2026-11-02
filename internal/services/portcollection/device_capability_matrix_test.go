package portcollection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== Phase 129 能力矩阵：厂商差异化 + 参数级组合 ======================
//
// 真机依据（2026-11-01）：
//   - 华为 S8700（10.62.25.253，GE1/0/0）：port-security ? 无 max-mac-num；
//     port-security mac-address ? 仅 H-H-H/sticky 无 static；sticky <MAC> 报
//     Wrong parameter；user-bind static ip-address ? 只接受 X.X.X.X
//   - 锐捷 RS8607E（产品 scrapli 链路复测）：switchport port-security maximum 5 /
//     mac-address sticky / mac-address <mac> vlan <vlan> 均实测无错误；
//     switchport port-security binding <IP> <MAC>（IP 在前）
//
// ⇒ 矩阵必须按厂商区分：同一动作在两家结论相反，不能一刀切。

// TestCapabilityMatrix_VendorDifferentiation 同一动作在华为 S8700 与锐捷 RS8607E
// 上结论相反 —— 这是矩阵按厂商区分的核心断言。
func TestCapabilityMatrix_VendorDifferentiation(t *testing.T) {
	const s8700 = "S8700-10"
	const rs8607e = "RS8607E"

	// 华为 S8700：static_bind / max_mac 不支持（真机 Wrong parameter / 无子命令）
	for _, action := range []PortAction{PortSecStaticBind, PortSecMaxMac} {
		err := CheckActionSupport(models.VendorHuawei, s8700, action)
		require.Error(t, err, "华为 S8700 的 %s 必须被拒绝", action)
		assert.ErrorIs(t, err, ErrActionNotSupported)
		assert.Contains(t, err.Error(), "2026-11-01 真机", "拒绝原因必须携带真机依据")
	}

	// 锐捷 RS8607E：同样两个动作实测无错误 → 必须放行
	for _, action := range []PortAction{PortSecStaticBind, PortSecMaxMac, PortSecSticky} {
		assert.NoError(t, CheckActionSupport(models.VendorRuijie, rs8607e, action),
			"锐捷 RS8607E 的 %s 真机实测可用，必须放行", action)
	}

	// 锐捷 IP+MAC 组合绑定支持（华为同组合不支持）→ 矩阵必须给出相反结论
	assert.NoError(t, CheckActionSupportParams(models.VendorRuijie, rs8607e, ActionPortBinding, "AA-BB-CC-DD-EE-FF"),
		"锐捷 RS8607E 支持 binding <IP> <MAC> 组合")
	assert.ErrorIs(t, CheckActionSupportParams(models.VendorHuawei, s8700, ActionPortBinding, "AA-BB-CC-DD-EE-FF"),
		ErrActionNotSupported, "华为 S8700 不支持 IP+MAC 组合绑定")
}

// TestCapabilityMatrix_RuijieEvidence 锐捷的「支持」结论必须带真机依据，
// 且不得被后续误改成 Unsupported（supported 规则优先于基线降级）。
func TestCapabilityMatrix_RuijieEvidence(t *testing.T) {
	const rs8607e = "RS8607E"

	ev := SupportEvidence(models.VendorRuijie, rs8607e, PortSecStaticBind)
	assert.Contains(t, ev, "锐捷 RS8607E 真机")
	assert.Contains(t, ev, "mac-address")
	assert.NotContains(t, ev, "不支持静态 MAC 绑定", "锐捷是支持结论，不得出现拒绝措辞")

	evMax := SupportEvidence(models.VendorRuijie, rs8607e, PortSecMaxMac)
	assert.Contains(t, evMax, "maximum 5")
	assert.Contains(t, evMax, "<1-128>", "真机实测的取值范围必须留在依据里")

	evSticky := SupportEvidence(models.VendorRuijie, rs8607e, PortSecSticky)
	assert.Contains(t, evSticky, "sticky")

	evBind := SupportEvidence(models.VendorRuijie, rs8607e, ActionPortBinding)
	assert.Contains(t, evBind, "binding <IP> <MAC>")
	assert.Contains(t, evBind, "IP 在前", "参数顺序是与华为相反的关键差异")

	// 华为侧的「支持」结论同样要有依据（sticky 开关式 + IP-only 绑定）
	assert.Contains(t, SupportEvidence(models.VendorHuawei, "S8700-10", PortSecSticky), "sticky")
	assert.Contains(t, SupportEvidence(models.VendorHuawei, "S8700-10", ActionPortBinding), "IP-only")

	// 华为侧的「不支持」依据（Unsupported 优先于 supported）
	assert.Contains(t, SupportEvidence(models.VendorHuawei, "S8700-10", PortSecStaticBind), "不支持静态 MAC 绑定")

	// 未命中型号级规则 → 无依据文本（按厂商基线处理）
	assert.Equal(t, "", SupportEvidence(models.VendorHuawei, "CE12800", PortSecStaticBind))
	assert.Equal(t, "", SupportEvidence(models.VendorRuijie, "", PortSecStaticBind))
	assert.Equal(t, "", SupportEvidence(models.VendorH3C, "whatever", PortSecStaticBind))
}

// TestCapabilityMatrix_SupportedRuleKeepsSystemView 型号级「支持」不得抹掉
// 厂商基线的系统视图要求（华为 user-bind 是系统视图命令，接口视图 Unrecognized）。
func TestCapabilityMatrix_SupportedRuleKeepsSystemView(t *testing.T) {
	assert.False(t, RequiresInterfaceView(models.VendorHuawei, ActionPortBinding),
		"华为 port_binding 必须仍是系统视图命令")
	assert.Equal(t, ActionSystemView, lookupSupport(models.VendorHuawei, "S8700-10", ActionPortBinding))
	// 其他型号的华为设备同样保持系统视图
	assert.Equal(t, ActionSystemView, lookupSupport(models.VendorHuawei, "CE12800", ActionPortBinding))
	// 锐捷 port_binding 是接口视图命令
	assert.Equal(t, ActionSupported, lookupSupport(models.VendorRuijie, "RS8607E", ActionPortBinding))
}

// TestCheckActionSupportParams_UnsupportedActionShortCircuit 动作级不支持时，
// 参数感知校验必须直接返回该错误（不再叠加参数组合判定）。
func TestCheckActionSupportParams_UnsupportedActionShortCircuit(t *testing.T) {
	err := CheckActionSupportParams(models.VendorHuawei, "S8700-10", PortSecStaticBind, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrActionNotSupported)
	assert.Contains(t, err.Error(), "不支持静态 MAC 绑定")
}

// TestCheckActionSupportParams_NonMatchingModel 模型未命中规则时，参数组合规则
// 不生效（非 S87 型号的华为设备仍支持 IP+MAC 组合绑定）。
func TestCheckActionSupportParams_NonMatchingModel(t *testing.T) {
	assert.NoError(t, CheckActionSupportParams(models.VendorHuawei, "CE12800", ActionPortBinding, "AA-BB-CC-DD-EE-FF"),
		"非 S87 型号不命中组合拒绝规则")
	assert.NoError(t, CheckActionSupportParams(models.VendorHuawei, "", ActionPortBinding, "AA-BB-CC-DD-EE-FF"),
		"型号为空不命中组合拒绝规则")
	// 同厂商同动作但 MAC 为空 → 组合规则不触发（IP-only 放行）
	assert.NoError(t, CheckActionSupportParams(models.VendorHuawei, "S8700-10", ActionPortBinding, ""),
		"IP-only 变体必须放行")
}

// TestUnsupportedReason_Direct unsupportedReason 的空值分支（型号为空 / 型号未命中 /
// 动作不在拒绝集合内）。
func TestUnsupportedReason_Direct(t *testing.T) {
	assert.Equal(t, "", unsupportedReason(models.VendorHuawei, "", PortSecMaxMac),
		"型号为空 → 无型号级原因")
	assert.Equal(t, "", unsupportedReason(models.VendorHuawei, "CE12800", PortSecMaxMac),
		"型号未命中规则 → 无型号级原因")
	assert.Equal(t, "", unsupportedReason(models.VendorHuawei, "S8700-10", ActionShutdown),
		"动作不在拒绝集合内 → 无型号级原因")
	assert.Contains(t, unsupportedReason(models.VendorHuawei, "S8700-10", PortSecMaxMac), "max-mac-num")
	assert.Equal(t, "", unsupportedReason(models.VendorH3C, "S5560", PortSecMaxMac),
		"厂商不在矩阵内 → 无型号级原因")
}

// TestCheckActionSupport_EmptyReasonFallback 拒绝规则缺少 reason 时，
// 必须回落到通用文案而不是返回空原因。
//
// 用临时注入的规则覆盖兜底分支：真机矩阵每条规则都带 reason，这里验证
// 「配置写漏了也不至于给出空白依据」。
func TestCheckActionSupport_EmptyReasonFallback(t *testing.T) {
	original, existed := vendorCapabilities[models.VendorCisco]
	t.Cleanup(func() {
		if existed {
			vendorCapabilities[models.VendorCisco] = original
		} else {
			delete(vendorCapabilities, models.VendorCisco)
		}
	})

	entry := capabilityEntry{
		modelRules: []modelCapabilityRule{
			{
				matchModelSubstr: []string{"testbox"},
				unsupported:       map[PortAction]string{PortSecMaxMac: ""},
			},
		},
	}
	vendorCapabilities[models.VendorCisco] = entry

	err := CheckActionSupport(models.VendorCisco, "testbox-1", PortSecMaxMac)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrActionNotSupported)
	assert.Contains(t, err.Error(), "该厂商不支持此动作", "缺少 reason 时必须回落到通用文案")
}

// TestCheckActionSupport_UnsupportedWinsOverSupported 同一条型号规则里同时写了
// supported 与 unsupported 时，「不支持」优先（拒绝结论不可被支持结论覆盖）。
func TestCheckActionSupport_UnsupportedWinsOverSupported(t *testing.T) {
	original, existed := vendorCapabilities[models.VendorCisco]
	t.Cleanup(func() {
		if existed {
			vendorCapabilities[models.VendorCisco] = original
		} else {
			delete(vendorCapabilities, models.VendorCisco)
		}
	})

	vendorCapabilities[models.VendorCisco] = capabilityEntry{
		modelRules: []modelCapabilityRule{
			{
				matchModelSubstr: []string{"testbox"},
				unsupported:       map[PortAction]string{PortSecMaxMac: "真机不支持（优先级更高）"},
				supported:         map[PortAction]string{PortSecMaxMac: "真机支持（应被忽略）"},
			},
		},
	}

	err := CheckActionSupport(models.VendorCisco, "testbox-1", PortSecMaxMac)
	require.Error(t, err, "同一动作同时声明支持与不支持时必须拒绝")
	assert.Contains(t, err.Error(), "真机不支持（优先级更高）")
	assert.NotContains(t, err.Error(), "真机支持（应被忽略）")
}

// TestMatchModelRules 型号匹配的公共语义（厂商不在矩阵 / 型号为空 / 命中 / 未命中）。
func TestMatchModelRules(t *testing.T) {
	assert.Empty(t, matchModelRules(models.VendorH3C, "S5560"), "厂商不在矩阵内 → 无型号级规则")
	assert.Empty(t, matchModelRules(models.VendorHuawei, ""), "型号为空 → 无型号级规则")

	hits := matchModelRules(models.VendorHuawei, "S8700-10")
	require.Len(t, hits, 1, "S8700-10 应命中 S87 规则")
	assert.Contains(t, hits[0].unsupported, PortSecMaxMac)

	assert.Empty(t, matchModelRules(models.VendorHuawei, "CE12800"), "CE12800 不命中 S87 规则")
	assert.Len(t, matchModelRules(models.VendorRuijie, "RS8607E-24T"), 1, "RS8607E-24T 应命中锐捷规则")
}

// TestLookupSupport_VendorBaselineFallback 厂商基线与默认值（未登记厂商 → 支持）。
func TestLookupSupport_VendorBaselineFallback(t *testing.T) {
	assert.Equal(t, ActionSupported, lookupSupport(models.VendorCisco, "C9300", ActionShutdown),
		"未登记厂商 → 默认支持")
	assert.Equal(t, ActionSupported, lookupSupport(models.VendorHuawei, "CE12800", ActionShutdown))
	assert.Equal(t, ActionSystemView, lookupSupport(models.VendorHuawei, "", ActionPortBinding),
		"型号为空 → 只看厂商基线（华为 port_binding 是系统视图）")
}
