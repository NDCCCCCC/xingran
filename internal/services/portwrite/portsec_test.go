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

// ====================== Phase 125 (PWX-02) 端口安全测试 ======================

// TestPortSecRender_AllVendors 3 厂商 × 5 动作渲染 + cleanup。
func TestPortSecRender_AllVendors(t *testing.T) {
	cases := []struct {
		vendor  models.DeviceVendor
		action  Action
		params  PortSecParams
		wantSub []string
	}{
		{models.VendorHuawei, portcollection.PortSecEnable, PortSecParams{}, []string{"port-security enable"}},
		{models.VendorH3C, portcollection.PortSecEnable, PortSecParams{}, []string{"port-security enable"}},
		{models.VendorRuijie, portcollection.PortSecEnable, PortSecParams{}, []string{"switchport port-security"}},
		{models.VendorHuawei, portcollection.PortSecDisable, PortSecParams{}, []string{"undo port-security enable"}},
		{models.VendorRuijie, portcollection.PortSecDisable, PortSecParams{}, []string{"no switchport port-security"}},
		{models.VendorH3C, portcollection.PortSecDisable, PortSecParams{}, []string{"undo port-security enable"}},
		{models.VendorHuawei, portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 5}, []string{"port-security max-mac-num 5"}},
		{models.VendorH3C, portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 5}, []string{"port-security max-mac-count 5"}},
		{models.VendorRuijie, portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 5}, []string{"switchport port-security maximum 5"}},
		{models.VendorHuawei, portcollection.PortSecSticky, PortSecParams{}, []string{"port-security mac-address sticky"}},
		{models.VendorRuijie, portcollection.PortSecSticky, PortSecParams{}, []string{"switchport port-security mac-address sticky"}},
		{models.VendorHuawei, portcollection.PortSecStaticBind, PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}, []string{"port-security mac-address sticky AA-BB-CC-DD-EE-FF vlan 100"}},
		{models.VendorH3C, portcollection.PortSecStaticBind, PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}, []string{"port-security mac-address security AA-BB-CC-DD-EE-FF vlan 100"}},
		{models.VendorRuijie, portcollection.PortSecStaticBind, PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}, []string{"switchport port-security mac-address aabb.ccdd.eeff vlan 100"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.vendor)+"/"+string(tc.action), func(t *testing.T) {
			db := newTestDB(t)
			seedPreviewDevice(t, db, tc.vendor)
			// portsec_disable 需要"已启用"端口才不会 NoOp 短路
			if tc.action == portcollection.PortSecDisable {
				port := models.DevicePortStatus{}
				require.NoError(t, db.Where("id = ?", "port-"+string(tc.vendor)).First(&port).Error)
				port.PortSecurityEnabled = true
				require.NoError(t, db.Save(&port).Error)
			}
			svc := newTestService(&countingExecutor{}, nil, db)
			result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
				DeviceID:    "device-" + string(tc.vendor),
				Action:      tc.action,
				PortIDs:     []string{"port-" + string(tc.vendor)},
				MaxMacCount: tc.params.MaxMACCount,
				MACAddress:  tc.params.MACAddress,
				VLANID:      tc.params.VLANID,
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
			last := cmds[len(cmds)-1]
			assert.True(t, last == "exit" || last == "quit", "末尾必须是 view 退出命令，got %q", last)
			// 风险标注：static_bind critical，其余 portsec medium
			if tc.action == portcollection.PortSecStaticBind {
				assert.Equal(t, "critical", result.RiskLevel)
			} else {
				assert.Equal(t, "medium", result.RiskLevel)
			}
		})
	}
}

// TestPortSecPreState_NoOp 复用采集缓存的 NoOp 短路（enable/disable/max_mac）。
func TestPortSecPreState_NoOp(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	// 端口已启用 port-security → enable 预览 NoOp
	port := models.DevicePortStatus{}
	require.NoError(t, db.Where("id = ?", "port-huawei").First(&port).Error)
	port.PortSecurityEnabled = true
	maxMac := 5
	port.MaxMACCount = &maxMac
	require.NoError(t, db.Save(&port).Error)

	svc := newTestService(&countingExecutor{}, nil, db)
	ctx := context.Background()

	res, err := svc.PreviewPortWrite(ctx, PreviewRequest{DeviceID: "device-huawei", Action: portcollection.PortSecEnable, PortIDs: []string{"port-huawei"}}, "t")
	require.NoError(t, err)
	assert.True(t, res.Ports[0].NoOp)
	assert.Equal(t, "portsec_enabled", res.Ports[0].CurrentState)

	res, err = svc.PreviewPortWrite(ctx, PreviewRequest{DeviceID: "device-huawei", Action: portcollection.PortSecMaxMac, PortIDs: []string{"port-huawei"}, MaxMacCount: 5}, "t")
	require.NoError(t, err)
	assert.True(t, res.Ports[0].NoOp)
	assert.Equal(t, "max_mac_match", res.Ports[0].CurrentState)

	// 未启用端口的 disable → NoOp
	// （seed 端口 PortSecurityEnabled=false 已覆盖，改回再测）
	port.PortSecurityEnabled = false
	require.NoError(t, db.Save(&port).Error)
	res, err = svc.PreviewPortWrite(ctx, PreviewRequest{DeviceID: "device-huawei", Action: portcollection.PortSecDisable, PortIDs: []string{"port-huawei"}}, "t")
	require.NoError(t, err)
	assert.True(t, res.Ports[0].NoOp)

	// sticky / static_bind 无缓存语义 → 不短路
	res, err = svc.PreviewPortWrite(ctx, PreviewRequest{DeviceID: "device-huawei", Action: portcollection.PortSecStaticBind, PortIDs: []string{"port-huawei"}, MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}, "t")
	require.NoError(t, err)
	assert.False(t, res.Ports[0].NoOp)
}

// TestValidatePortSecRequest sentinel 错误路径。
func TestValidatePortSecRequest(t *testing.T) {
	assert.ErrorIs(t, validatePortSecRequest(portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 0}), ErrMaxMacOutOfRange)
	assert.ErrorIs(t, validatePortSecRequest(portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 4097}), ErrMaxMacOutOfRange)
	assert.ErrorIs(t, validatePortSecRequest(portcollection.PortSecStaticBind, PortSecParams{VLANID: 100}), ErrPortSecParamsInvalid, "缺 MAC")
	assert.ErrorIs(t, validatePortSecRequest(portcollection.PortSecStaticBind, PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 0}), ErrVlanIdOutOfRange)
	assert.ErrorIs(t, validatePortSecRequest(portcollection.PortSecStaticBind, PortSecParams{MACAddress: "not-a-mac", VLANID: 100}), ErrMACAddressInvalid)
	assert.ErrorIs(t, validatePortSecRequest(portcollection.ActionShutdown, PortSecParams{}), ErrPortSecParamsInvalid)
	// 合法路径
	assert.NoError(t, validatePortSecRequest(portcollection.PortSecEnable, PortSecParams{}))
	assert.NoError(t, validatePortSecRequest(portcollection.PortSecStaticBind, PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}))
}

// TestPortSecWrite_ExtraAndPipeline 实跑管线 + Extra 审计回显。
func TestPortSecWrite_ExtraAndPipeline(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-ps", "device-ps", "GE0/0/1", "up", false, "")
	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	result, err := svc.PortSecWrite(context.Background(), "port-ps", portcollection.PortSecStaticBind,
		PortSecParams{MACAddress: "AA:BB:CC:DD:EE:FF", VLANID: 100}, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	psExtra, ok := result.Extra["portsec"].(PortSecParams)
	require.True(t, ok, "Extra.portsec 必须携带 PortSecParams 供审计 after_value")
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", psExtra.MACAddress)

	// 锁/预检/快照管线同源：trunk 测试已验证此处仅冒烟
	_ = exec
}

// TestPortSecZeroNetworkInPreview dry-run 零网络回归。
func TestPortSecZeroNetworkInPreview(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)
	_, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-huawei", Action: portcollection.PortSecSticky, PortIDs: []string{"port-huawei"},
	}, "tester")
	require.NoError(t, err)
	assert.Equal(t, 0, exec.calls)
}

// 编译期保值（time 引用）。
var _ = time.Now
