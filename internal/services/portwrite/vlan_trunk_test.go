package portwrite

import (
	"context"
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

// TestTrunkRender_RuijieHybridUnsupported 锐捷无 hybrid 模式 → 渲染层拒绝。
func TestTrunkRender_RuijieHybridUnsupported(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)
	svc := newTestService(&countingExecutor{}, nil, db)
	_, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-ruijie", Action: portcollection.TrunkSetHybrid,
		PortIDs: []string{"port-ruijie"}, HybridTagged: []int{10},
	}, "tester")
	require.Error(t, err)
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
