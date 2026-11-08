package portwrite

// FIX-04（v1.53 Phase 138）：物理参数 service 层单测。
//
// 覆盖：3 动作 validator（合法/非法）+ 成功路径 Extra 回显（审计 after_value 载体）
// + writeSinglePort 真实下发链（mock executor，零网络）。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// TestPhysical138_ValidatePhysicalRequest validator 表驱动（合法/非法/未知动作）。
func TestPhysical138_ValidatePhysicalRequest(t *testing.T) {
	cases := []struct {
		name    string
		action  Action
		params  PhysicalParams
		wantErr error
	}{
		// 速率
		{"speed 10 合法", portcollection.ActionSetSpeed, PhysicalParams{Speed: 10}, nil},
		{"speed 10000 合法", portcollection.ActionSetSpeed, PhysicalParams{Speed: 10000}, nil},
		{"speed auto 合法", portcollection.ActionSetSpeed, PhysicalParams{SpeedMode: portcollection.SpeedModeAuto}, nil},
		{"speed 0 非法", portcollection.ActionSetSpeed, PhysicalParams{Speed: 0}, ErrSpeedInvalid},
		{"speed 40 非法", portcollection.ActionSetSpeed, PhysicalParams{Speed: 40}, ErrSpeedInvalid},
		{"speedMode 非法", portcollection.ActionSetSpeed, PhysicalParams{SpeedMode: "fast"}, ErrSpeedInvalid},
		// 双工
		{"duplex full 合法", portcollection.ActionSetDuplex, PhysicalParams{Duplex: portcollection.DuplexFull}, nil},
		{"duplex half 合法", portcollection.ActionSetDuplex, PhysicalParams{Duplex: portcollection.DuplexHalf}, nil},
		{"duplex auto 合法", portcollection.ActionSetDuplex, PhysicalParams{Duplex: portcollection.DuplexAuto}, nil},
		{"duplex 空非法", portcollection.ActionSetDuplex, PhysicalParams{}, ErrDuplexInvalid},
		{"duplex both 非法", portcollection.ActionSetDuplex, PhysicalParams{Duplex: "both"}, ErrDuplexInvalid},
		// MTU
		{"mtu 1280 合法", portcollection.ActionSetMTU, PhysicalParams{MTU: 1280}, nil},
		{"mtu 1500 合法", portcollection.ActionSetMTU, PhysicalParams{MTU: 1500}, nil},
		{"mtu 9216 合法", portcollection.ActionSetMTU, PhysicalParams{MTU: 9216}, nil},
		{"mtu 1279 越界", portcollection.ActionSetMTU, PhysicalParams{MTU: 1279}, ErrMTUOutOfRange},
		{"mtu 9217 越界", portcollection.ActionSetMTU, PhysicalParams{MTU: 9217}, ErrMTUOutOfRange},
		{"mtu 0 越界", portcollection.ActionSetMTU, PhysicalParams{MTU: 0}, ErrMTUOutOfRange},
		// 未知动作
		{"未知动作", "not_physical", PhysicalParams{}, ErrPhysicalParamsInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePhysicalRequest(tc.action, tc.params)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// TestPhysical138_PhysicalWrite_SuccessExtra 成功路径：3 动作回显进 PortResult.Extra
// 供 handler buildAfterValue 写 audit.after_value（禁默默吞字段）。
func TestPhysical138_PhysicalWrite_SuccessExtra(t *testing.T) {
	cases := []struct {
		name      string
		action    Action
		params    PhysicalParams
		wantExtra map[string]interface{}
	}{
		{"speed 1000", portcollection.ActionSetSpeed, PhysicalParams{Speed: 1000}, map[string]interface{}{"speed": 1000}},
		{"speed auto", portcollection.ActionSetSpeed, PhysicalParams{SpeedMode: portcollection.SpeedModeAuto}, map[string]interface{}{"speed": "auto"}},
		{"duplex full", portcollection.ActionSetDuplex, PhysicalParams{Duplex: portcollection.DuplexFull}, map[string]interface{}{"duplex": "full"}},
		{"mtu 9000", portcollection.ActionSetMTU, PhysicalParams{MTU: 9000}, map[string]interface{}{"mtu": 9000}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetCollectNotif()
			db := newTestDB(t)
			seedPortAndDevice(t, db, "port-phy-1", "device-1", "GE0/0/1", "up", false, "")

			exec := new(mockDeviceExecutor)
			// success path：executor 返回 nil，不调 fn（service 端有 nil-guard）
			exec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, mock.Anything).Return(nil)
			coll := new(mockCollectionSvc)
			// 注意：PhysicalWrite 与 PortSecWrite/TrunkWrite 同构，直接调 writeSinglePort，
			// 不经 writeAndRefresh ⇒ 不触发 refreshPortStatus/CollectDevice（既有设计）。
			// 故此处不设 CollectDevice 期望、不 waitCollectedCalls。
			svc := newTestService(exec, coll, db)

			result, err := svc.PhysicalWrite(context.Background(), "port-phy-1", tc.action, tc.params, "op-138")
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "succeeded", result.Status)
			assert.Equal(t, tc.action, result.Action)
			// CommandSent 含渲染出的物理参数命令
			assert.NotEmpty(t, result.CommandSent)
			// Extra 回显完整
			for k, want := range tc.wantExtra {
				assert.Equal(t, want, result.Extra[k], "Extra[%s]", k)
			}
			exec.AssertExpectations(t)
		})
	}
}

// TestPhysical138_PhysicalWrite_RejectsInvalid 非法参数在 DB/SSH 之前被拒（零副作用）。
func TestPhysical138_PhysicalWrite_RejectsInvalid(t *testing.T) {
	db := newTestDB(t)
	exec := new(mockDeviceExecutor)
	coll := new(mockCollectionSvc)
	svc := newTestService(exec, coll, db)

	_, err := svc.PhysicalWrite(context.Background(), "p", portcollection.ActionSetSpeed, PhysicalParams{Speed: 999}, "op")
	assert.ErrorIs(t, err, ErrSpeedInvalid)
	_, err = svc.PhysicalWrite(context.Background(), "p", portcollection.ActionSetDuplex, PhysicalParams{Duplex: "x"}, "op")
	assert.ErrorIs(t, err, ErrDuplexInvalid)
	_, err = svc.PhysicalWrite(context.Background(), "p", portcollection.ActionSetMTU, PhysicalParams{MTU: 1}, "op")
	assert.ErrorIs(t, err, ErrMTUOutOfRange)
	_, err = svc.PhysicalWrite(context.Background(), "p", "shutdown", PhysicalParams{}, "op")
	assert.ErrorIs(t, err, ErrPhysicalParamsInvalid)

	// 零网络调用
	exec.AssertNotCalled(t, "ExecuteCustom")
}
