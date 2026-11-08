package portcollection

// Phase 143 (PORT-02)：型号级物理能力矩阵框架单测（全部不依赖真机）。
//
// 手法沿用 device_capability_matrix_test.go:138-175 的「备份 vendorCapabilities[vendor]
// → 替换成自定义 capabilityEntry → defer/t.Cleanup 恢复」，避免污染其它测试。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// withPhysicalEntry 临时把某厂商的能力条目替换成自定义值，t.Cleanup 恢复。
func withPhysicalEntry(t *testing.T, vendor models.DeviceVendor, entry capabilityEntry) {
	t.Helper()
	original, existed := vendorCapabilities[vendor]
	t.Cleanup(func() {
		if existed {
			vendorCapabilities[vendor] = original
		} else {
			delete(vendorCapabilities, vendor)
		}
	})
	vendorCapabilities[vendor] = entry
}

// customPhysicalProfile 构造一个「型号级」画像用于测试分叉（与全局默认不同）。
func customPhysicalProfile() physicalProfile {
	return physicalProfile{
		AllowedSpeeds:   []int{100, 1000},
		AllowedDuplexes: []string{DuplexFull, DuplexAuto},
		MTUMin:          1500,
		MTUMax:          9000,
		SpeedCommand:    "port speed %s",
		DuplexCommand:   "port duplex %s",
		MTUCommand:      "jumboframe enable %d",
		RequiresView:    true,
		Evidence:        "测试型号真机依据（fixture，非真实设备）",
	}
}

// TestPhysical143_DefaultProfileEquivalentFIX04 全局默认画像 == FIX-04 现状（三家交集），
// 保证单条物理写零回归；Evidence="" 明确表示未取证，level=unverified。
func TestPhysical143_DefaultProfileEquivalentFIX04(t *testing.T) {
	p := defaultPhysicalProfile()
	assert.Equal(t, []int{10, 100, 1000, 10000}, p.AllowedSpeeds)
	assert.ElementsMatch(t, []string{DuplexHalf, DuplexFull, DuplexAuto}, p.AllowedDuplexes)
	assert.Equal(t, MTUMin, p.MTUMin)
	assert.Equal(t, MTUMax, p.MTUMax)
	assert.Equal(t, "speed %s", p.SpeedCommand)
	assert.Equal(t, "duplex %s", p.DuplexCommand)
	assert.Equal(t, "mtu %d", p.MTUCommand)
	assert.True(t, p.RequiresView)
	assert.Empty(t, p.Evidence, "Evidence 空串表示未取证，严禁写成已验证")
	assert.Equal(t, "unverified", PhysicalEvidenceLevel(p), "默认放行 ≠ 有证据")
}

// TestPhysical143_ResolveProfileFallbackChain 回退链：型号命中 > 厂商基线 > 全局默认。
// 覆盖型号命中/未命中/空 Model/大小写/多规则优先级。
func TestPhysical143_ResolveProfileFallbackChain(t *testing.T) {
	modelProfile := customPhysicalProfile()
	vendorBaseline := defaultPhysicalProfile()
	vendorBaseline.SpeedCommand = "vendor-baseline speed %s"
	vendorBaseline.Evidence = "厂商基线，未逐型号取证"

	withPhysicalEntry(t, models.VendorHuawei, capabilityEntry{
		physical: &vendorBaseline,
		modelPhysical: []modelPhysicalRule{
			{matchModelSubstr: []string{"s8700"}, profile: modelProfile},
			{matchModelSubstr: []string{"ce12800"}, profile: modelProfile},
		},
	})

	cases := []struct {
		name      string
		vendor    models.DeviceVendor
		model     string
		wantLevel string
		wantCmd   string
	}{
		{"型号命中(小写子串)", models.VendorHuawei, "S8700-10", "model", "port speed %s"},
		{"型号命中(大小写不敏感)", models.VendorHuawei, "s8700-32c", "model", "port speed %s"},
		{"型号命中(第二条规则)", models.VendorHuawei, "CE12800", "model", "port speed %s"},
		{"型号未命中 → 厂商基线", models.VendorHuawei, "S5735", "vendor", "vendor-baseline speed %s"},
		{"空 Model → 厂商基线", models.VendorHuawei, "", "vendor", "vendor-baseline speed %s"},
		{"厂商不在矩阵 → 全局默认", models.VendorH3C, "S5560", "unverified", "speed %s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := ResolvePhysicalProfile(tc.vendor, tc.model)
			assert.Equal(t, tc.wantLevel, PhysicalEvidenceLevel(p))
			assert.Equal(t, tc.wantCmd, p.SpeedCommand)
		})
	}

	// 多规则优先级：第一条命中即返回（不继续叠加第二条）
	first := customPhysicalProfile()
	first.SpeedCommand = "first-rule speed %s"
	second := customPhysicalProfile()
	second.SpeedCommand = "second-rule speed %s"
	withPhysicalEntry(t, models.VendorRuijie, capabilityEntry{
		modelPhysical: []modelPhysicalRule{
			{matchModelSubstr: []string{"rs8607"}, profile: first},
			{matchModelSubstr: []string{"rs8607"}, profile: second},
		},
	})
	assert.Equal(t, "first-rule speed %s", ResolvePhysicalProfile(models.VendorRuijie, "RS8607E").SpeedCommand,
		"多条型号规则命中时按声明顺序，第一条优先")
}

// TestPhysical143_CheckPhysicalSupportFullChain CheckPhysicalSupport 完整回退链：
// 型号 unsupported 拒绝 > 型号画像 > 厂商 unsupported 拒绝 > 厂商基线 > 全局默认。
func TestPhysical143_CheckPhysicalSupportFullChain(t *testing.T) {
	modelProfile := customPhysicalProfile()
	withPhysicalEntry(t, models.VendorCisco, capabilityEntry{
		physical: &modelProfile, // 即便配了厂商基线，vendorUnsupported 仍优先拒绝
		vendorUnsupported: map[PortAction]string{
			ActionSetSpeed: physicalVendorUnsupportedReason,
		},
		modelRules: []modelCapabilityRule{
			{matchModelSubstr: []string{"testbox"}, unsupported: map[PortAction]string{
				ActionSetSpeed: "型号级真机拒绝依据（fixture）",
			}},
		},
		modelPhysical: []modelPhysicalRule{
			{matchModelSubstr: []string{"c9300"}, profile: modelProfile},
		},
	})

	// 1. 型号 unsupported 优先于一切（含型号画像 / 厂商 unsupported）
	_, err := CheckPhysicalSupport(models.VendorCisco, "testbox-1", ActionSetSpeed, PortTemplateParams{Speed: 1000})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrActionNotSupported)
	assert.Contains(t, err.Error(), "型号级真机拒绝依据")

	// 2. 厂商 vendorUnsupported 拒绝（无型号命中）
	_, err = CheckPhysicalSupport(models.VendorCisco, "unknown-model", ActionSetSpeed, PortTemplateParams{Speed: 1000})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrActionNotSupported)
	assert.Contains(t, err.Error(), "渲染层未实现")
	assert.Contains(t, err.Error(), "无真机取证依据")

	// 3. 型号画像命中优先于厂商 unsupported（对齐 lookupSupport 优先级）
	prof, err := CheckPhysicalSupport(models.VendorCisco, "C9300-48", ActionSetSpeed, PortTemplateParams{Speed: 1000})
	require.NoError(t, err)
	assert.Equal(t, "model", PhysicalEvidenceLevel(prof))
	assert.Equal(t, "port speed %s", prof.SpeedCommand)
}

// TestPhysical143_CheckPhysicalSupportVendorModelRealMatrix 真实矩阵：Cisco / 迈普
// 物理三动作显式 Unsupported；三家默认放行且 Evidence 空串（未取证）。
func TestPhysical143_CheckPhysicalSupportVendorModelRealMatrix(t *testing.T) {
	valid := map[PortAction]PortTemplateParams{
		ActionSetSpeed:  {Speed: 1000},
		ActionSetDuplex: {Duplex: DuplexFull},
		ActionSetMTU:    {MTU: 9000},
	}
	for _, vendor := range []models.DeviceVendor{models.VendorCisco, models.VendorMaipu} {
		for action, params := range valid {
			_, err := CheckPhysicalSupport(vendor, "ANY-MODEL", action, params)
			require.Error(t, err, "vendor=%s action=%s 必须显式拒绝", vendor, action)
			assert.ErrorIs(t, err, ErrActionNotSupported)
		}
	}
	for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		for action, params := range valid {
			prof, err := CheckPhysicalSupport(vendor, "UNKNOWN-MODEL", action, params)
			require.NoError(t, err, "vendor=%s action=%s 应默认放行", vendor, action)
			assert.Equal(t, "unverified", PhysicalEvidenceLevel(prof))
			assert.Empty(t, prof.Evidence)
		}
	}
}

// TestPhysical143_ParamRangeAndEnum 枚举范围越界按画像拒绝（型号画像收窄值域后生效）。
func TestPhysical143_ParamRangeAndEnum(t *testing.T) {
	prof := customPhysicalProfile() // speeds {100,1000}, duplex {full,auto}, mtu [1500,9000]
	withPhysicalEntry(t, models.VendorHuawei, capabilityEntry{
		modelPhysical: []modelPhysicalRule{{matchModelSubstr: []string{"s8700"}, profile: prof}},
	})
	p := func(v PortTemplateParams) PortTemplateParams { v.Model = "S8700-10"; return v }

	// 型号画像放行
	_, err := CheckPhysicalSupport(models.VendorHuawei, "S8700-10", ActionSetSpeed, p(PortTemplateParams{Speed: 100}))
	require.NoError(t, err)
	// 型号画像拒绝（10000 不在型号画像内）
	_, err = CheckPhysicalSupport(models.VendorHuawei, "S8700-10", ActionSetSpeed, p(PortTemplateParams{Speed: 10000}))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)

	_, err = CheckPhysicalSupport(models.VendorHuawei, "S8700-10", ActionSetDuplex, p(PortTemplateParams{Duplex: DuplexHalf}))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)

	_, err = CheckPhysicalSupport(models.VendorHuawei, "S8700-10", ActionSetMTU, p(PortTemplateParams{MTU: 1280}))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)

	// 未知动作
	_, err = CheckPhysicalSupport(models.VendorHuawei, "S8700-10", "not_physical", p(PortTemplateParams{}))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPhysicalParamInvalid)
}

// TestPhysical143_CommandShapeFork 命令形态分叉：型号画像改变命令体；
// 全局默认三家命令体与 FIX-04 一致（零回归）。
func TestPhysical143_CommandShapeFork(t *testing.T) {
	prof := customPhysicalProfile()
	withPhysicalEntry(t, models.VendorHuawei, capabilityEntry{
		modelPhysical: []modelPhysicalRule{{matchModelSubstr: []string{"s8700"}, profile: prof}},
	})

	// 型号画像：命令体分叉 + interface 前缀
	cmds, err := RenderCommand(models.VendorHuawei, ActionSetSpeed, PortTemplateParams{
		InterfaceName: "GE0/0/1", Model: "S8700-10", Speed: 1000,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"interface GE0/0/1", "port speed 1000"}, cmds)

	cmds, err = RenderCommand(models.VendorHuawei, ActionSetMTU, PortTemplateParams{
		InterfaceName: "GE0/0/1", Model: "S8700-10", MTU: 9000,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"interface GE0/0/1", "jumboframe enable 9000"}, cmds)

	// 全局默认（型号未命中）：命令体与 FIX-04 完全一致
	for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		cmds, err = RenderCommand(vendor, ActionSetSpeed, PortTemplateParams{
			InterfaceName: "GE0/0/1", Model: "UNKNOWN", Speed: 1000,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"interface GE0/0/1", "speed 1000"}, cmds, "vendor=%s 默认命令体零回归", vendor)
	}
}

// TestPhysical143_CiscoMaipuRenderExplicitUnsupported 渲染层直调 Cisco / 迈普物理动作
// 返回 ErrActionNotSupported（不是 ErrUnsupportedVendor 含糊兜底）。
func TestPhysical143_CiscoMaipuRenderExplicitUnsupported(t *testing.T) {
	for _, vendor := range []models.DeviceVendor{models.VendorCisco, models.VendorMaipu} {
		for _, action := range []PortAction{ActionSetSpeed, ActionSetDuplex, ActionSetMTU} {
			_, err := RenderCommand(vendor, action, PortTemplateParams{InterfaceName: "GE0/0/1", Speed: 1000, Duplex: DuplexFull, MTU: 9000})
			require.Error(t, err, "vendor=%s action=%s", vendor, action)
			assert.ErrorIs(t, err, ErrActionNotSupported)
			assert.Contains(t, err.Error(), "渲染层未实现")
			assert.Contains(t, err.Error(), "无真机取证依据")
		}
	}
}
