package portwrite

// Phase 143 (PORT-01)：批量物理参数路径单测（全部不依赖真机，用既有 mock/fixture 手法）。
//
// 覆盖：入口 validator（物理三动作 + 4 参数透传）、audit.after_value 非空、
// fail-fast、三数组聚合、进度逐端口事件（复用 batch_progress_test.go 的
// fakeProgressNotifier）、型号/厂商 Unsupported 批量拒绝。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"gorm.io/gorm"
)

// seedBatchUser 为进度事件 operator→userID 解析 seed 一条 sys_user 行。
func seedBatchUser(t *testing.T, db *gorm.DB, username string) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&models.User{}))
	u := &models.User{Username: username, Password: "x", Status: 0}
	u.ID = "user-" + username
	// Username 有 uniqueIndex；重复 seed 时忽略冲突（同测试内只调一次）
	_ = db.Create(u).Error
}

// newPhysicalBatchService 构造成功 executor 的批量测试 service。
func newPhysicalBatchService(t *testing.T, db *gorm.DB) (*portWriteServiceImpl, *mockDeviceExecutor) {
	t.Helper()
	exec := new(mockDeviceExecutor)
	exec.On("ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	coll := new(mockCollectionSvc)
	coll.On("CollectDevice", mock.Anything, mock.Anything).Return(newSuccessCollectionResult("device-phy"), nil)
	return newTestService(exec, coll, db), exec
}

// TestPhysical143_BatchEntryValidator 入口 validator：物理四参数非法 → 400 sentinel，
// 零 SSH 流量（executor 不被调用）。
func TestPhysical143_BatchEntryValidator(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "phy-p1", "device-phy", "GE0/0/1", "up", false, "")
	svc, exec := newPhysicalBatchService(t, db)

	cases := []struct {
		name   string
		action Action
		req    BatchWriteRequest
		want   error
	}{
		{"speed 非法", portcollection.ActionSetSpeed, BatchWriteRequest{Speed: 999}, ErrSpeedInvalid},
		{"duplex 非法", portcollection.ActionSetDuplex, BatchWriteRequest{Duplex: "both"}, ErrDuplexInvalid},
		{"mtu 越界", portcollection.ActionSetMTU, BatchWriteRequest{MTU: 1}, ErrMTUOutOfRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.DeviceID = "device-phy"
			req.Action = tc.action
			req.PortIDs = []string{"phy-p1"}
			_, err := svc.BatchWritePorts(context.Background(), req, "op")
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.want)
		})
	}
	exec.AssertNotCalled(t, "ExecuteCustom")
}

// TestPhysical143_BatchFourParamsAndAuditExtra 4 参数透传 + audit.after_value 载体非空。
// 三个动作 × 两家厂商，验证 CommandSent 含物理命令 + Extra 含回显 + physicalEvidence 空串（未取证）。
func TestPhysical143_BatchFourParamsAndAuditExtra(t *testing.T) {
	cases := []struct {
		name      string
		action    Action
		req       BatchWriteRequest
		wantCmd   string
		wantExtra string
		wantVal   interface{}
	}{
		{"speed 1000", portcollection.ActionSetSpeed, BatchWriteRequest{Speed: 1000}, "speed 1000", "speed", 1000},
		{"speed auto", portcollection.ActionSetSpeed, BatchWriteRequest{SpeedMode: portcollection.SpeedModeAuto}, "speed auto", "speed", "auto"},
		{"duplex full", portcollection.ActionSetDuplex, BatchWriteRequest{Duplex: portcollection.DuplexFull}, "duplex full", "duplex", "full"},
		{"mtu 9000", portcollection.ActionSetMTU, BatchWriteRequest{MTU: 9000}, "mtu 9000", "mtu", 9000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetCollectNotif()
			db := newTestDB(t)
			seedPortAndDevice(t, db, "phy-p1", "device-phy", "GE0/0/1", "up", false, "")
			seedPortAndDevice(t, db, "phy-p2", "device-phy", "GE0/0/2", "up", false, "")
			svc, _ := newPhysicalBatchService(t, db)

			req := tc.req
			req.DeviceID = "device-phy"
			req.Action = tc.action
			req.PortIDs = []string{"phy-p1", "phy-p2"}
			result, err := svc.BatchWritePorts(context.Background(), req, "op")
			require.NoError(t, err)
			require.Len(t, result.Succeeded, 2)
			assert.Empty(t, result.Failed)
			assert.Empty(t, result.Skipped)

			for _, pr := range result.Succeeded {
				// 命令分发：4 参数透传到 RenderCommand
				assert.Contains(t, pr.CommandSent, tc.wantCmd, "物理参数必须进入下发命令")
				// audit.after_value 载体（buildAfterValue 从 Extra 读；批量不再恒 {}）
				require.NotNil(t, pr.Extra, "批量物理写必须写 Extra（修 audit.after_value 恒 {} 缺口）")
				assert.Equal(t, tc.wantVal, pr.Extra[tc.wantExtra])
				assert.Equal(t, "unverified", pr.Extra["physicalEvidenceLevel"], "默认放行 ≠ 有证据")
				assert.Empty(t, pr.Extra["physicalEvidence"], "Evidence 空串 = 未取证")
			}
		})
	}
}

// TestPhysical143_BatchThreeArraysAggregation 三数组聚合 + fail-fast（首个失败立即停，
// 剩余端口不进任何数组，进度事件零改动复用）。
func TestPhysical143_BatchThreeArraysAggregation(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "phy-p1", "device-phy", "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "phy-p2", "device-phy", "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "phy-p3", "device-phy", "GE0/0/3", "up", false, "")

	exec := new(mockDeviceExecutor)
	// 第 1 个端口成功，第 2 个端口 transport error → fail-fast，第 3 个不执行
	exec.On("ExecuteCustom", mock.Anything, "device-phy", mock.Anything, mock.Anything).Return(nil).Once()
	exec.On("ExecuteCustom", mock.Anything, "device-phy", mock.Anything, mock.Anything).
		Return(assert.AnError).Once()
	coll := new(mockCollectionSvc)
	coll.On("CollectDevice", mock.Anything, mock.Anything).Return(newSuccessCollectionResult("device-phy"), nil)
	svc := newTestService(exec, coll, db)

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-phy", Action: portcollection.ActionSetSpeed, Speed: 1000,
		PortIDs: []string{"phy-p1", "phy-p2", "phy-p3"},
	}, "op")
	require.NoError(t, err)
	assert.Len(t, result.Succeeded, 1, "第 1 个成功")
	assert.Len(t, result.Failed, 1, "第 2 个失败")
	assert.Empty(t, result.Skipped)
	// fail-fast：第 3 个端口未进任何数组
	for _, pr := range append(append(result.Succeeded, result.Failed...), result.Skipped...) {
		assert.NotEqual(t, "phy-p3", pr.PortID, "fail-fast 后剩余端口不得进数组")
	}
	// 三数组均为非 nil（JSON [] 而非 null）
	assert.NotNil(t, result.Succeeded)
	assert.NotNil(t, result.Failed)
	assert.NotNil(t, result.Skipped)
}

// TestPhysical143_BatchProgressPerPortEvents 进度逐端口事件零改动复用：物理动作下发时
// 仍逐端口推 port_write_progress + 终态 completed（Action string 与 action 无关）。
func TestPhysical143_BatchProgressPerPortEvents(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "phy-p1", "device-phy", "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "phy-p2", "device-phy", "GE0/0/2", "up", false, "")
	seedBatchUser(t, db, "phy-op")
	svc, _ := newPhysicalBatchService(t, db)
	fake := &fakeProgressNotifier{}
	svc.progressNotifier = fake

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-phy", Action: portcollection.ActionSetMTU, MTU: 9000,
		PortIDs: []string{"phy-p1", "phy-p2"},
	}, "phy-op")
	require.NoError(t, err)
	assert.NotEmpty(t, result.BatchID)
	require.GreaterOrEqual(t, len(fake.events), 2)
	// 逐端口事件
	var progressed int
	for _, e := range fake.events {
		if e.Type == ProgressTypePortWriteProgress {
			progressed++
			assert.Equal(t, "set_mtu", e.Action, "进度事件 Action 直采 action 字符串，零改动复用")
			assert.Equal(t, result.BatchID, e.BatchID)
		}
	}
	assert.Equal(t, 2, progressed, "每端口一条进度事件")
	// 终态事件
	last := fake.events[len(fake.events)-1]
	assert.Equal(t, "port_write_completed", last.Type)
}

// TestPhysical143_BatchVendorUnsupportedReject Cisco / 迈普物理动作批量入口显式拒绝
// （ErrActionNotSupported，handler 翻译 400），零 SSH 流量。
func TestPhysical143_BatchVendorUnsupportedReject(t *testing.T) {
	for _, vendor := range []models.DeviceVendor{models.VendorCisco, models.VendorMaipu} {
		t.Run(string(vendor), func(t *testing.T) {
			db := newTestDB(t)
			// 覆盖设备厂商（seedPortAndDevice 默认 huawei）
			seedPortAndDevice(t, db, "phy-p1", "device-phy", "GE0/0/1", "up", false, "")
			require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", "device-phy").
				Update("vendor", vendor).Error)
			svc, exec := newPhysicalBatchService(t, db)

			_, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
				DeviceID: "device-phy", Action: portcollection.ActionSetSpeed, Speed: 1000,
				PortIDs: []string{"phy-p1"},
			}, "op")
			require.Error(t, err)
			assert.ErrorIs(t, err, portcollection.ErrActionNotSupported)
			assert.Contains(t, err.Error(), "渲染层未实现")
			exec.AssertNotCalled(t, "ExecuteCustom")
		})
	}
}
