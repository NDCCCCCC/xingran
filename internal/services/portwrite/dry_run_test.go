package portwrite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ====================== Phase 121 (v1.51 PORTW-01) dry-run 预览测试 ======================

// countingExecutor 记录 ExecuteCustom 调用次数 —— dry-run 零网络调用证明的真相源。
type countingExecutor struct {
	calls int
}

func (c *countingExecutor) ExecuteCustom(
	ctx context.Context, deviceID string,
	fn func(context.Context, *device.PooledConnection) error, timeout time.Duration,
) error {
	c.calls++
	return nil
}

// seedPreviewDevice 按厂商 seed 设备 + 端口。
func seedPreviewDevice(t *testing.T, db *gorm.DB, vendor models.DeviceVendor) {
	t.Helper()
	dev := &models.NetworkDevice{
		DeviceName: "dev-" + string(vendor),
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     vendor,
		IPAddress:  "10.0." + vendorSuffix(vendor) + ".1",
		Status:     models.DeviceStatusOnline,
		Model:      "model-" + string(vendor),
	}
	dev.ID = "device-" + string(vendor)
	require.NoError(t, db.Create(dev).Error)

	port := &models.DevicePortStatus{
		ID:            "port-" + string(vendor),
		DeviceID:      dev.ID,
		InterfaceName: "GE1/0/1",
		AdminStatus:   "up",
		Dot1xEnabled:  false,
		CollectedAt:   time.Now(),
	}
	require.NoError(t, db.Create(port).Error)
}

func vendorSuffix(v models.DeviceVendor) string {
	switch v {
	case models.VendorHuawei:
		return "1"
	case models.VendorH3C:
		return "2"
	case models.VendorRuijie:
		return "3"
	case models.VendorCisco:
		return "4"
	default:
		return "9"
	}
}

// TestPreview_ZeroNetworkCalls 核心验收：dry-run 不产生任何设备连接。
func TestPreview_ZeroNetworkCalls(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	req := PreviewRequest{
		DeviceID: "device-huawei",
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"port-huawei"},
	}
	result, err := svc.PreviewPortWrite(context.Background(), req, "tester")
	require.NoError(t, err)
	assert.Equal(t, 0, exec.calls, "dry-run 不得触发任何 ExecuteCustom 调用")
	assert.True(t, result.DryRun)
	assert.Equal(t, "critical", result.RiskLevel)
	require.Len(t, result.Ports, 1)
	assert.Equal(t, "preview", result.Ports[0].Status)
	assert.NotEmpty(t, result.Ports[0].Commands)
	// shutdown 命令必须含 interface 进入 + shutdown 本体
	cmds := result.Ports[0].Commands
	assert.Contains(t, cmds[0].Command, "interface")
	assert.Equal(t, "shutdown", cmds[1].Command)
	assert.True(t, cmds[1].Critical)
}

// TestPreview_AllVendors_AllActions 3 厂商 × 7 action 渲染全覆盖且各厂商命令有差异。
func TestPreview_AllVendors_AllActions(t *testing.T) {
	vendors := []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie}
	actions := []Action{
		portcollection.ActionShutdown,
		portcollection.ActionUndoShutdown,
		portcollection.ActionDescription,
		portcollection.ActionDot1xEnable,
		portcollection.ActionDot1xDisable,
		portcollection.ActionSetAccessVLAN,
		portcollection.ActionPortBinding,
	}
	for _, vendor := range vendors {
		for _, action := range actions {
			t.Run(string(vendor)+"/"+string(action), func(t *testing.T) {
				db := newTestDB(t)
				seedPreviewDevice(t, db, vendor)
				exec := &countingExecutor{}
				svc := newTestService(exec, nil, db)

				req := PreviewRequest{
					DeviceID:    "device-" + string(vendor),
					Action:      action,
					PortIDs:     []string{"port-" + string(vendor)},
					Description: "test-desc",
					VLANID:      100,
					BindOp:      "add",
					IPAddress:   "192.168.1.10",
				}
				result, err := svc.PreviewPortWrite(context.Background(), req, "tester")
				require.NoError(t, err)
				require.Len(t, result.Ports, 1)
				pr := result.Ports[0]
				// pre-state NoOp 语义与真实写路径同源：up 端口 undo_shutdown / 未启用端口
				// dot1x_disable 会 skipped；其余 action 必须渲染出命令。
				noOpExpected := action == portcollection.ActionUndoShutdown ||
					action == portcollection.ActionDot1xDisable
				if noOpExpected {
					assert.True(t, pr.NoOp, "%s/%s 应 NoOp skipped", vendor, action)
					assert.Empty(t, pr.Commands)
				} else {
					assert.NotEmpty(t, pr.Commands, "%s/%s 必须渲染出命令", vendor, action)
				}
				assert.Equal(t, 0, exec.calls)
			})
		}
	}
}

// TestPreview_VendorDifferences 同一 action 在 3 厂商下命令串存在差异（验收 #2）。
func TestPreview_VendorDifferences(t *testing.T) {
	type rendered map[models.DeviceVendor][]string
	collect := func(t *testing.T, action Action, extra func(*PreviewRequest)) rendered {
		out := rendered{}
		for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
			db := newTestDB(t)
			seedPreviewDevice(t, db, vendor)
			svc := newTestService(&countingExecutor{}, nil, db)
			req := PreviewRequest{
				DeviceID:    "device-" + string(vendor),
				Action:      action,
				PortIDs:     []string{"port-" + string(vendor)},
				Description: "desc-x",
				VLANID:      100,
				BindOp:      "add",
				IPAddress:   "192.168.1.10",
			}
			if extra != nil {
				extra(&req)
			}
			result, err := svc.PreviewPortWrite(context.Background(), req, "tester")
			require.NoError(t, err)
			out[vendor] = previewCommandsOf(result)
		}
		return out
	}

	t.Run("set_access_vlan", func(t *testing.T) {
		r := collect(t, portcollection.ActionSetAccessVLAN, nil)
		// 锐捷用 switchport access vlan，华为/H3C 用 port access vlan / port default vlan —— 至少两家不同
		assert.NotEqual(t, r[models.VendorHuawei], r[models.VendorRuijie])
	})
	t.Run("dot1x_enable", func(t *testing.T) {
		r := collect(t, portcollection.ActionDot1xEnable, nil)
		assert.NotEqual(t, r[models.VendorHuawei], r[models.VendorH3C])
	})
	t.Run("shutdown", func(t *testing.T) {
		// up 端口 shutdown 非 NoOp；末尾 exit cleanup（华为/H3C quit vs 锐捷 exit）构成差异
		r := collect(t, portcollection.ActionShutdown, nil)
		assert.NotEqual(t, r[models.VendorHuawei], r[models.VendorRuijie])
	})
}

func previewCommandsOf(r *PreviewResult) []string {
	var out []string
	for _, p := range r.Ports {
		for _, c := range p.Commands {
			out = append(out, c.Command)
		}
	}
	return out
}

// TestPreview_NoOp_PreState pre-state 匹配 → skipped + 无命令（与真实写路径同语义）。
func TestPreview_NoOp_PreState(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	// 端口 admin_status=up 时 shutdown 才会真正下发；undo_shutdown 对 up 端口是 NoOp
	svc := newTestService(&countingExecutor{}, nil, db)

	result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID: "device-huawei",
		Action:   portcollection.ActionUndoShutdown,
		PortIDs:  []string{"port-huawei"},
	}, "tester")
	require.NoError(t, err)
	require.Len(t, result.Ports, 1)
	assert.True(t, result.Ports[0].NoOp)
	assert.Equal(t, "skipped", result.Ports[0].Status)
	assert.Empty(t, result.Ports[0].Commands)
}

// TestPreview_SentinelErrors validator / 资源不存在 sentinel 语义与真实写路径同源。
func TestPreview_SentinelErrors(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	svc := newTestService(&countingExecutor{}, nil, db)
	ctx := context.Background()

	cases := []struct {
		name    string
		req     PreviewRequest
		wantErr error
	}{
		{"vlan 越界", PreviewRequest{DeviceID: "device-huawei", Action: portcollection.ActionSetAccessVLAN, PortIDs: []string{"port-huawei"}, VLANID: 5000}, ErrVlanIdOutOfRange},
		{"bindOp 非法", PreviewRequest{DeviceID: "device-huawei", Action: portcollection.ActionPortBinding, PortIDs: []string{"port-huawei"}, BindOp: "delete", IPAddress: "192.168.1.1"}, ErrBindOpInvalid},
		{"IP 非法", PreviewRequest{DeviceID: "device-huawei", Action: portcollection.ActionPortBinding, PortIDs: []string{"port-huawei"}, BindOp: "add", IPAddress: "not-an-ip"}, ErrIPAddressInvalid},
		{"设备不存在", PreviewRequest{DeviceID: "no-dev", Action: portcollection.ActionShutdown, PortIDs: []string{"port-huawei"}}, ErrDeviceNotFound},
		{"端口不存在", PreviewRequest{DeviceID: "device-huawei", Action: portcollection.ActionShutdown, PortIDs: []string{"no-port"}}, ErrPortNotFound},
		{"描述为空", PreviewRequest{DeviceID: "device-huawei", Action: portcollection.ActionDescription, PortIDs: []string{"port-huawei"}}, portcollection.ErrDescriptionEmpty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.PreviewPortWrite(ctx, tc.req, "tester")
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestRiskLevelForAction 风险等级标注表锁定。
func TestRiskLevelForAction(t *testing.T) {
	assert.Equal(t, RiskLevelCritical, riskLevelForAction(portcollection.ActionShutdown))
	assert.Equal(t, RiskLevelCritical, riskLevelForAction(portcollection.ActionSetAccessVLAN))
	assert.Equal(t, RiskLevelMedium, riskLevelForAction(portcollection.ActionUndoShutdown))
	assert.Equal(t, RiskLevelMedium, riskLevelForAction(portcollection.ActionDot1xEnable))
	assert.Equal(t, RiskLevelMedium, riskLevelForAction(portcollection.ActionPortBinding))
	assert.Equal(t, RiskLevelLow, riskLevelForAction(portcollection.ActionDescription))
}
