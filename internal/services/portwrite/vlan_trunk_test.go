package portwrite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ====================== Phase 124 (PWX-01) trunk/hybrid 测试 ======================

// TestTrunkRender_AllVendors 3 厂商 × 2 新动作渲染（hybrid 仅华为/H3C）+ cleanup 命令。
func TestTrunkRender_AllVendors(t *testing.T) {
	cases := []struct {
		vendor  models.DeviceVendor
		action  Action
		params  TrunkParams
		wantSub []string // 命令序列必须包含的片段
	}{
		{models.VendorHuawei, portcollection.TrunkSetNative, TrunkParams{NativeVLAN: 10}, []string{"port trunk pvid vlan 10"}},
		{models.VendorH3C, portcollection.TrunkSetNative, TrunkParams{NativeVLAN: 10}, []string{"port trunk pvid vlan 10"}},
		{models.VendorRuijie, portcollection.TrunkSetNative, TrunkParams{NativeVLAN: 10}, []string{"switchport trunk native vlan 10"}},
		{models.VendorHuawei, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeAdd}, []string{"port trunk allow-pass vlan 10 20"}},
		{models.VendorHuawei, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{30}, AllowedMode: portcollection.AllowedModeRemove}, []string{"undo port trunk allow-pass vlan 30"}},
		{models.VendorH3C, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10}, AllowedMode: portcollection.AllowedModeAdd}, []string{"port trunk permit vlan 10"}},
		{models.VendorH3C, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10}, AllowedMode: portcollection.AllowedModeRemove}, []string{"undo port trunk permit vlan 10"}},
		{models.VendorRuijie, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeAdd}, []string{"switchport trunk allowed vlan add 10,20"}},
		{models.VendorRuijie, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeRemove}, []string{"switchport trunk allowed vlan remove 10,20"}},
		{models.VendorRuijie, portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeReplace}, []string{"switchport trunk allowed vlan 10,20"}},
		{models.VendorHuawei, portcollection.TrunkSetHybrid, TrunkParams{HybridTagged: []int{30}, HybridUntagged: []int{10, 20}},
			[]string{"port link-type hybrid", "port hybrid tagged vlan 30", "port hybrid untagged vlan 10 20"}},
		{models.VendorH3C, portcollection.TrunkSetHybrid, TrunkParams{HybridTagged: []int{30}, HybridUntagged: []int{10}},
			[]string{"port link-type hybrid", "port hybrid vlan 30", "port hybrid vlan untagged 10"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.vendor)+"/"+string(tc.action), func(t *testing.T) {
			db := newTestDB(t)
			seedPreviewDevice(t, db, tc.vendor)
			svc := newTestService(&countingExecutor{}, nil, db)
			result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
				DeviceID:       "device-" + string(tc.vendor),
				Action:         tc.action,
				PortIDs:        []string{"port-" + string(tc.vendor)},
				NativeVLAN:     tc.params.NativeVLAN,
				AllowedVLANs:   tc.params.AllowedVLANs,
				AllowedMode:    tc.params.AllowedMode,
				HybridTagged:   tc.params.HybridTagged,
				HybridUntagged: tc.params.HybridUntagged,
			}, "tester")
			require.NoError(t, err)
			cmds := previewCommandsOf(result)
			joined := ""
			for _, c := range cmds {
				joined += c + "\n"
			}
			for _, want := range tc.wantSub {
				assert.Contains(t, joined, want)
			}
			// cleanup 命令（exit/quit）出现在预览末尾
			assert.NotEmpty(t, cmds)
			last := cmds[len(cmds)-1]
			assert.True(t, last == "exit" || last == "quit", "末尾必须是 view 退出命令，got %q", last)
			assert.Equal(t, "critical", result.RiskLevel, "trunk 动作必须标注 critical")
		})
	}
}

// TestTrunkRender_RuijieHybridSupported 锐捷 hybrid 真机支持（2026-11-02 `?` 帮助取证）：
// switchport hybrid ? → allowed/native；
// switchport hybrid allowed vlan ? → add/only/remove/tagged/untagged ⇒ 列表必须带 tagged/untagged 关键字；
// switchport hybrid native vlan ? → <1-4094> Vid ⇒ native 是单值；
// switchport mode ? 含 hybrid ⇒ 渲染须前置 switchport mode hybrid。
//
// 2026-11-02 之前本用例断言"锐捷 hybrid 必须被渲染层拒绝"（当时设备未取证、
// 渲染未注册）—— 真机事实推翻该认知后改为"支持 + 命令形态断言"。
func TestTrunkRender_RuijieHybridSupported(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)
	svc := newTestService(&countingExecutor{}, nil, db)
	res, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-ruijie", Action: portcollection.TrunkSetHybrid,
		PortIDs: []string{"port-ruijie"}, HybridTagged: []int{30}, HybridUntagged: []int{10, 20},
	}, "tester")
	require.NoError(t, err, "锐捷 hybrid 真机支持，预览不应报错")

	require.Len(t, res.Ports, 1)
	var cmds []string
	for _, c := range res.Ports[0].Commands {
		cmds = append(cmds, c.Command)
	}
	joined := strings.Join(cmds, " | ")
	assert.Contains(t, joined, "switchport mode hybrid", "必须前置切 hybrid 模式")
	// ★ 2026-11-02 第三次真机取证：tagged **必须带 only**（缺 only = 命令成功但配置不落地）；
	//   untagged **不能带 only**（only untagged 被设备拒绝）⇒ 两条形态不对称。
	assert.Contains(t, joined, "switchport hybrid allowed vlan only tagged 30", "tagged 列表必须带 only + tagged 关键字")
	assert.Contains(t, joined, "switchport hybrid allowed vlan untagged 10,20", "untagged 列表必须带 untagged 关键字且不得带 only")
	assert.NotContains(t, joined, "allowed vlan only untagged", "untagged 不得带 only（真机 % Invalid input）")
	assert.NotContains(t, joined, "switchport hybrid allowed vlan 30", "不得出现缺关键字的 allowed 形态")

	// dry-run 预览必须显式标注"端口模式变更"（不能藏在命令列表里）
	var modeNote string
	for _, c := range res.Ports[0].Commands {
		if c.Command == "switchport mode hybrid" {
			modeNote = c.Note
		}
	}
	assert.Contains(t, modeNote, "端口模式变更为 hybrid", "模式变更必须在预览里显式标注")
}

// TestTrunkRender_RuijieHybridNativeSingleValue 锐捷 hybrid 的 native 是单值 VID。
//
// native 与 tagged/untagged 两个列表解耦，由 NativeVLAN 单独承载（单值字段）。
func TestTrunkRender_RuijieHybridNativeSingleValue(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)
	svc := newTestService(&countingExecutor{}, nil, db)
	res, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-ruijie", Action: portcollection.TrunkSetHybrid,
		PortIDs: []string{"port-ruijie"}, HybridTagged: []int{30}, NativeVLAN: 10,
	}, "tester")
	require.NoError(t, err)
	var cmds []string
	for _, c := range res.Ports[0].Commands {
		cmds = append(cmds, c.Command)
	}
	assert.Contains(t, strings.Join(cmds, " | "), "switchport hybrid native vlan 10",
		"native 单值走独立命令形态")
}

// TestValidateTrunkRequest sentinel 错误路径（验收 #3 参数校验）。
func TestValidateTrunkRequest(t *testing.T) {
	// native VLAN 越界
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetNative, TrunkParams{NativeVLAN: 4095}), ErrVlanIdOutOfRange)
	// allowed 空列表
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetAllowed, TrunkParams{}), ErrTrunkParamsInvalid)
	// allowed 非法 mode
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{10}, AllowedMode: "all"}), ErrAllowedModeInvalid)
	// allowed 列表越界
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetAllowed, TrunkParams{AllowedVLANs: []int{0}}), ErrVlanIdOutOfRange)
	// hybrid 双空列表
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetHybrid, TrunkParams{}), ErrTrunkParamsInvalid)
	// hybrid tagged/untagged 重叠（华为设备侧兜底；H3C 设备侧不报错 → 系统侧是唯一防线）
	assert.ErrorIs(t, validateTrunkRequest(portcollection.TrunkSetHybrid, TrunkParams{HybridTagged: []int{10, 30}, HybridUntagged: []int{10}}), ErrHybridOverlap)
	// 非重叠合法
	assert.NoError(t, validateTrunkRequest(portcollection.TrunkSetHybrid, TrunkParams{HybridTagged: []int{30}, HybridUntagged: []int{10, 20}}))
}

// TestTrunkWrite_ExtraAuditing 实跑成功后 Extra 携带 trunk 参数（审计 after_value 完整性）。
func TestTrunkWrite_ExtraAuditing(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-trunk", "device-trunk", "GE0/0/1", "up", false, "")
	svc := newTestService(&countingExecutor{}, nil, db)

	result, err := svc.TrunkWrite(context.Background(), "port-trunk", portcollection.TrunkSetAllowed,
		TrunkParams{AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeAdd}, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	trunkExtra, ok := result.Extra["trunk"].(TrunkParams)
	require.True(t, ok, "Extra.trunk 必须携带 TrunkParams 供审计 after_value")
	assert.Equal(t, []int{10, 20}, trunkExtra.AllowedVLANs)
	assert.Equal(t, portcollection.AllowedModeAdd, trunkExtra.AllowedMode)
}

// TestTrunkWrite_SentinelPropagates 错误沿管线传播（端口不存在 / 非法参数）。
func TestTrunkWrite_SentinelPropagates(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	svc := newTestService(&countingExecutor{}, nil, db)
	ctx := context.Background()

	// 端口行不存在走 D-13 fallback → executeWrite deviceID 空 → 设备不存在语义
	_, err := svc.TrunkWrite(ctx, "no-port", portcollection.TrunkSetNative, TrunkParams{NativeVLAN: 10}, "tester")
	assert.ErrorIs(t, err, ErrDeviceNotFound)

	_, err = svc.TrunkWrite(ctx, "port-huawei", portcollection.TrunkSetHybrid,
		TrunkParams{HybridTagged: []int{10}, HybridUntagged: []int{10}}, "tester")
	assert.ErrorIs(t, err, ErrHybridOverlap)

	// 非 trunk action 拒绝
	_, err = svc.TrunkWrite(ctx, "port-huawei", portcollection.ActionShutdown, TrunkParams{}, "tester")
	assert.ErrorIs(t, err, ErrTrunkParamsInvalid)
}

// TestTrunkZeroNetworkInPreview trunk dry-run 仍零网络调用（管线回归）。
func TestTrunkZeroNetworkInPreview(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)
	_, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-huawei", Action: portcollection.TrunkSetNative,
		PortIDs: []string{"port-huawei"}, NativeVLAN: 10,
	}, "tester")
	require.NoError(t, err)
	assert.Equal(t, 0, exec.calls)
}

// 编译期守护：writeSinglePort 签名扩展未破坏 helper（time import 保值）。
var _ = time.Now
