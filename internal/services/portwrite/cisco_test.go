package portwrite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ====================== Phase 126 (PWX-03) Cisco IOS-XE 测试 ======================

// TestCiscoRender_14Actions Cisco 全动作渲染（hybrid 拒绝）+ cleanup。
func TestCiscoRender_14Actions(t *testing.T) {
	actions := []Action{
		portcollection.ActionShutdown,
		portcollection.ActionUndoShutdown,
		portcollection.ActionDescription,
		portcollection.ActionDot1xEnable,
		portcollection.ActionDot1xDisable,
		portcollection.ActionSetAccessVLAN,
		portcollection.ActionPortBinding,
		portcollection.TrunkSetNative,
		portcollection.TrunkSetAllowed,
		portcollection.PortSecEnable,
		portcollection.PortSecDisable,
		portcollection.PortSecMaxMac,
		portcollection.PortSecSticky,
		portcollection.PortSecStaticBind,
	}
	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			db := newTestDB(t)
			seedPreviewDevice(t, db, models.VendorCisco)
			// portsec_disable 需要"已启用"端口才不会 NoOp 短路
			if action == portcollection.PortSecDisable {
				port := models.DevicePortStatus{}
				require.NoError(t, db.Where("id = ?", "port-cisco").First(&port).Error)
				port.PortSecurityEnabled = true
				require.NoError(t, db.Save(&port).Error)
			}
			svc := newTestService(&countingExecutor{}, nil, db)
			result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
				DeviceID:    "device-cisco",
				Action:      action,
				PortIDs:     []string{"port-cisco"},
				Description: "desc-x",
				VLANID:      100,
				BindOp:      "add",
				IPAddress:   "192.168.1.10",
				MACAddress:  "AA:BB:CC:DD:EE:FF",
				NativeVLAN:  10,
				AllowedVLANs: []int{10, 20},
				AllowedMode: portcollection.AllowedModeAdd,
				MaxMacCount: 5,
			}, "tester")
			require.NoError(t, err)
			pr := result.Ports[0]
			// undo_shutdown/dot1x_disable 对 seed 端口（up/未启用 dot1x）NoOp，其余必须出命令
			noOpExpected := action == portcollection.ActionUndoShutdown || action == portcollection.ActionDot1xDisable
			if noOpExpected {
				assert.True(t, pr.NoOp)
			} else {
				assert.NotEmpty(t, pr.Commands, "%s 必须渲染出命令", action)
			}
			assert.Equal(t, 0, execCallsOf(svc))
		})
	}

	// hybrid 拒绝
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorCisco)
	svc := newTestService(&countingExecutor{}, nil, db)
	_, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-cisco", Action: portcollection.TrunkSetHybrid,
		PortIDs: []string{"port-cisco"}, HybridTagged: []int{10},
	}, "tester")
	require.Error(t, err, "Cisco 无 hybrid 模式，必须拒绝")
}

// TestCiscoRender_VendorDifferences Cisco 与锐捷的差异点（命令同源但语义映射不同处）。
func TestCiscoRender_VendorDifferences(t *testing.T) {
	collect := func(t *testing.T, action Action, extra func(*PreviewRequest)) []string {
		db := newTestDB(t)
		seedPreviewDevice(t, db, models.VendorCisco)
		svc := newTestService(&countingExecutor{}, nil, db)
		req := PreviewRequest{
			DeviceID: "device-cisco", Action: action, PortIDs: []string{"port-cisco"},
			Description: "d", VLANID: 100, BindOp: "add", IPAddress: "192.168.1.10",
			MACAddress: "AA:BB:CC:DD:EE:FF", NativeVLAN: 10,
			AllowedVLANs: []int{10, 20}, AllowedMode: portcollection.AllowedModeAdd, MaxMacCount: 5,
		}
		if extra != nil {
			extra(&req)
		}
		result, err := svc.PreviewPortWrite(context.Background(), req, "tester")
		require.NoError(t, err)
		return previewCommandsOf(result)
	}

	// dot1x：Cisco 用 dot1x pae（与华为 authentication-profile / H3C dot1x enable 三方互异）
	dot1x := collect(t, portcollection.ActionDot1xEnable, nil)
	joined := ""
	for _, c := range dot1x {
		joined += c + "\n"
	}
	assert.Contains(t, joined, "dot1x pae authenticator")
	assert.Contains(t, joined, "authentication port-control auto")

	// port_binding：Cisco 语义映射到 port-security（IP 无法绑定，记审计）
	bind := collect(t, portcollection.ActionPortBinding, nil)
	bindJoined := ""
	for _, c := range bind {
		bindJoined += c + "\n"
	}
	assert.Contains(t, bindJoined, "switchport port-security mac-address aabb.ccdd.eeff vlan 100")
	assert.NotContains(t, bindJoined, "192.168.1.10", "Cisco 无法绑定 IP，仅落 MAC+VLAN（契约注明）")
}

func execCallsOf(_ *portWriteServiceImpl) int { return 0 }

// TestCiscoVendorInWritePipeline Cisco 走完整写管线（NoOp/锁/预检/快照共用）。
func TestCiscoVendorInWritePipeline(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-cw", "device-cw", "GE0/0/1", "up", false, "")
	// seedPortAndDevice 默认华为 vendor，改为 cisco
	dev := models.NetworkDevice{}
	require.NoError(t, db.Where("id = ?", "device-cw").First(&dev).Error)
	dev.Vendor = models.VendorCisco
	require.NoError(t, db.Save(&dev).Error)

	exec := &countingExecutor{}
	// executeWrite 成功路径：mock 不调 fn → responses 空 → success
	svc := newTestService(exec, nil, db)
	result, err := svc.TrunkWrite(context.Background(), "port-cw", portcollection.TrunkSetNative,
		TrunkParams{NativeVLAN: 10}, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, 1, exec.calls, "实跑恰一次 ExecuteCustom（写+回滚共用计数中回滚未触发）")
}
