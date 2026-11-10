package portcollection

// B-3 / v1.56：SysObjectID 匹配键机制回归测试。
//
// 生产规则表不预置任何具体设备的 OID→型号结论（真机逐型号取证属 Known Gap，
// 预置会伪造「已识别型号」）；本测试用 withPhysicalEntry 临时注入自定义规则，
// 证明「OID 精确 > OID 前缀（最长前缀） > Model 子串 > 厂商基线 / 默认」的
// 优先级真实生效，且 OID 命中绝不污染 Evidence / level 的取证诚实性口径。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// b3TestProfiles 三条可区分的型号画像（命令形态不同，便于断言命中了哪一条）。
func b3Profile(tag string) physicalProfile {
	p := defaultPhysicalProfile()
	p.SpeedCommand = tag + " speed %s"
	p.Evidence = "" // 保持「未取证」口径
	return p
}

func b3Entry() capabilityEntry {
	return capabilityEntry{
		physical: nil, // 厂商基线留空，逼测试走「全局默认」回退
		modelPhysical: []modelPhysicalRule{
			{
				matchModelSubstr:  []string{"s8700"},
				matchSysObjectIDs: []string{"1.3.6.1.4.1.2011.2.23"}, // 精确
				profile:           b3Profile("exact"),
			},
			{
				matchModelSubstr:  []string{"s5700"},
				matchSysObjectIDs: []string{"1.3.6.1.4.1.2011."}, // 前缀（以点结尾）
				profile:           b3Profile("prefix"),
			},
			{
				matchModelSubstr: []string{"ce6800"}, // 纯 Model 子串
				profile:          b3Profile("model"),
			},
		},
	}
}

func TestB3_OIDMatchPriority(t *testing.T) {
	vendor := models.VendorHuawei
	withPhysicalEntry(t, vendor, b3Entry())

	cases := []struct {
		name       string
		model      string
		oid        string
		wantCmd    string
		wantMatched MatchedBy
	}{
		{"OID 精确优先于 Model 子串", "S8700-other", "1.3.6.1.4.1.2011.2.23", "exact", MatchedByOIDExact},
		{"OID 前缀命中（最长前缀）", "some-other", "1.3.6.1.4.1.2011.2.99", "prefix", MatchedByOIDPrefix},
		{"OID 不命中 → Model 子串兜底", "CE6800-48S", "9.9.9.9", "model", MatchedByModelSubstr},
		{"OID 空 + Model 命中 → Model 子串", "S8700-X", "", "exact", MatchedByModelSubstr},
		{"OID 精确优先于前缀", "x", "1.3.6.1.4.1.2011.2.23", "exact", MatchedByOIDExact},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 1) 逐规则调用 modelKeyMatches（与 resolveModelPhysicalProfileWithOID
			//    的真实语义一致：每条规则各自判断自己的 OID/Model 键集合）
			var by MatchedBy
			for _, rule := range b3Entry().modelPhysical {
				if ok, got := modelKeyMatches(rule.matchModelSubstr, rule.matchSysObjectIDs, tc.model, tc.oid); ok {
					by = got
					break
				}
			}
			assert.Equal(t, tc.wantMatched, by, "命中方式应与优先级一致")

			// 2) 画像解析确实按该优先级选到对应规则
			prof, matched := resolveModelPhysicalProfileWithOID(vendor, tc.model, tc.oid)
			require.True(t, matched)
			assert.Equal(t, tc.wantCmd+" speed %s", prof.SpeedCommand,
				"应命中 %q 对应的型号画像", tc.wantCmd)
			// 3) OID 命中绝不构成真机取证：Evidence 仍为空、level 仍为 model
			assert.Empty(t, prof.Evidence,
				"OID 是设备识别依据，绝不写进 physicalProfile.Evidence（破坏口径）")
			assert.Equal(t, "model", PhysicalEvidenceLevel(prof))
		})
	}
}

func TestB3_OIDMissFallsBackToVendorBaselineOrDefault(t *testing.T) {
	vendor := models.VendorHuawei
	entry := b3Entry()
	entry.physical = &physicalProfile{SpeedCommand: "baseline speed %s"}
	withPhysicalEntry(t, vendor, entry)

	// OID 未命中且 Model 未命中 → 厂商基线（不是默认）
	prof := ResolvePhysicalProfileWithOID(vendor, "unknown-box", "8.8.8.8")
	assert.Equal(t, "baseline speed %s", prof.SpeedCommand)
	assert.Equal(t, "vendor", PhysicalEvidenceLevel(prof))

	// 旧签名（OID 为空）零回归：Model 仍按子串命中
	profOld := ResolvePhysicalProfile(vendor, "S8700-X")
	assert.Equal(t, "exact speed %s", profOld.SpeedCommand)
}

func TestB3_CheckPhysicalSupportWithOID_KeepsEvidenceDiscipline(t *testing.T) {
	vendor := models.VendorHuawei
	withPhysicalEntry(t, vendor, b3Entry())

	params := PortTemplateParams{Speed: 1000, SysObjectID: "1.3.6.1.4.1.2011.2.23"}
	// 校验侧传 OID
	profCheck, err := CheckPhysicalSupportWithOID(vendor, "anything", ActionSetSpeed, params, "1.3.6.1.4.1.2011.2.23")
	require.NoError(t, err)
	assert.Empty(t, profCheck.Evidence, "OID 命中不改变 Evidence 空串口径")
	// 渲染侧走 PortTemplateParams.SysObjectID（校验/渲染同源）
	cmds, err := renderPhysical(params, vendor, ActionSetSpeed)
	require.NoError(t, err)
	require.NotEmpty(t, cmds)
	// 命令体应来自 exact 画像（wrapInterface 会加 interface 前缀与 quit，故逐条找）
	joined := ""
	for _, c := range cmds {
		joined += c + "\n"
	}
	assert.Contains(t, joined, "exact speed 1000",
		"渲染必须按同一 OID 命中 exact 画像，保证校验放行 ⇔ 渲染接受")
}
