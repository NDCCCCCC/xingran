package portwrite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
)

// ====================== Phase 129 真机修复：写路径能力矩阵拒绝 ======================
//
// 真机依据（2026-11-01）：
//   - 华为 S8700：port-security mac-address sticky <MAC> 报 Wrong parameter，
//     port-security mac-address ? 无 static 子命令 ⇒ 不支持静态 MAC 绑定；
//     user-bind static ip-address ? 只接受 X.X.X.X ⇒ IP+MAC 组合不支持
//   - 锐捷 RS8607E：static bind / max_mac / sticky / IP+MAC 组合全部实测无错误
//
// 写路径必须在下发前拒绝（不连设备、不下发必失败命令），dry-run 与实跑同源。

// TestWriteRejectsStaticBindOnS8700 华为 S8700 实跑静态 MAC 绑定 → 能力矩阵拒绝。
func TestWriteRejectsStaticBindOnS8700(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-s87", "device-s87", "GE0/0/1", "up", false, "")
	dev := models.NetworkDevice{}
	require.NoError(t, db.Where("id = ?", "device-s87").First(&dev).Error)
	dev.Model = "S8700-10"
	require.NoError(t, db.Save(&dev).Error)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	result, err := svc.PortSecWrite(context.Background(), "port-s87",
		portcollection.PortSecStaticBind,
		PortSecParams{MACAddress: "AA-BB-CC-DD-EE-FF", VLANID: 100}, "tester")
	require.Error(t, err, "S8700 不支持静态 MAC 绑定，必须拒绝")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, portcollection.ErrActionNotSupported)
	assert.Contains(t, err.Error(), "2026-11-01 真机", "拒绝原因必须携带真机依据")
	assert.Equal(t, 0, exec.calls, "能力矩阵拒绝不得触发任何设备连接")
}

// TestWriteRejectsPortBindingComboOnS8700 华为 S8700 的 IP+MAC 组合绑定 → 拒绝；
// IP-only 变体放行（真机 user-bind static ip-address X 合法）。
func TestWriteRejectsPortBindingComboOnS8700(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-s87b", "device-s87b", "GE0/0/1", "up", false, "")
	dev := models.NetworkDevice{}
	require.NoError(t, db.Where("id = ?", "device-s87b").First(&dev).Error)
	dev.Model = "S8700-10"
	require.NoError(t, db.Save(&dev).Error)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)
	ctx := context.Background()

	// IP+MAC 组合 → 拒绝
	_, err := svc.PortBinding(ctx, "port-s87b", "add", "10.62.25.5", "AA-BB-CC-DD-EE-FF", "tester")
	require.Error(t, err)
	assert.ErrorIs(t, err, portcollection.ErrActionNotSupported)
	assert.Contains(t, err.Error(), "IP+MAC")
	assert.Equal(t, 0, exec.calls, "拒绝路径不得连接设备")

	// IP-only → 放行（真机语法合法）
	result, err := svc.PortBinding(ctx, "port-s87b", "add", "10.62.25.5", "", "tester")
	require.NoError(t, err, "IP-only 变体必须放行")
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, 1, exec.calls, "放行路径应下发一次")
}

// TestWriteAllowsPortSecOnRuijieRS8607E 锐捷 RS8607E 与华为结论相反：
// 静态 MAC 绑定 / MAC 上限必须放行（真机实测无错误）。
func TestWriteAllowsPortSecOnRuijieRS8607E(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-rj", "device-rj", "GE0/0/1", "up", false, "")
	dev := models.NetworkDevice{}
	require.NoError(t, db.Where("id = ?", "device-rj").First(&dev).Error)
	dev.Vendor = models.VendorRuijie
	dev.Model = "RS8607E"
	require.NoError(t, db.Save(&dev).Error)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	// static_bind：锐捷支持（华为 S8700 同动作被拒）
	result, err := svc.PortSecWrite(context.Background(), "port-rj",
		portcollection.PortSecStaticBind,
		PortSecParams{MACAddress: "AA-BB-CC-DD-EE-FF", VLANID: 100}, "tester")
	require.NoError(t, err, "锐捷 RS8607E 真机实测静态 MAC 绑定可用")
	assert.Equal(t, "succeeded", result.Status)

	// max_mac：锐捷支持（华为 S8700 无 max-mac-num 子命令）
	result, err = svc.PortSecWrite(context.Background(), "port-rj",
		portcollection.PortSecMaxMac, PortSecParams{MaxMACCount: 5}, "tester")
	require.NoError(t, err, "锐捷 RS8607E 真机实测 maximum 5 可用")
	assert.Equal(t, "succeeded", result.Status)
}

// TestWriteDeviceRowMissingReturnsError 端口行存在但设备行缺失 → 查询设备失败。
func TestWriteDeviceRowMissingReturnsError(t *testing.T) {
	db := newTestDB(t)
	port := &models.DevicePortStatus{
		ID:            "port-orphan",
		DeviceID:      "device-missing",
		InterfaceName: "GE0/0/1",
		AdminStatus:   "up",
		CollectedAt:   time.Now(),
	}
	require.NoError(t, db.Create(port).Error)

	svc := newTestService(&countingExecutor{}, nil, db)
	_, err := svc.Shutdown(context.Background(), "port-orphan", "tester")
	require.Error(t, err, "设备行缺失必须报错")
	assert.Contains(t, err.Error(), "query device")
}

// TestWritePortLockedReturnsConflict 端口已被其他写操作持锁 → 写互斥冲突。
func TestWritePortLockedReturnsConflict(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-locked", "device-locked", "GE0/0/1", "up", false, "")

	// 预占进程内锁（另一个 holder）→ service 侧加锁失败
	require.True(t, globalPortLocks.tryLockMemory("port-locked", "other@tester", portWriteLockTTL))
	t.Cleanup(func() { globalPortLocks.unlockMemory("port-locked", "other@tester") })

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)
	_, err := svc.Shutdown(context.Background(), "port-locked", "tester")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPortWriteLocked)
	assert.Contains(t, err.Error(), "other@tester", "冲突错误应带持锁者信息")
	assert.Equal(t, 0, exec.calls, "持锁冲突不得下发命令")
}

// TestSnapshotFailureDoesNotBlockWrite 快照抓取失败（DB 写不进去）不得阻断写路径。
func TestSnapshotFailureDoesNotBlockWrite(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-nosnap", "device-nosnap", "GE0/0/1", "up", false, "")
	// 删掉快照表 → captureSnapshot 的 INSERT 必然失败
	require.NoError(t, db.Exec("DROP TABLE sys_port_write_snapshot").Error)

	mockExec := new(mockDeviceExecutor)
	mockExec.On("ExecuteCustom", mock.Anything, "device-nosnap", mock.Anything, singlePortTimeout).Return(nil)
	svc := newTestService(mockExec, nil, db)

	result, err := svc.Shutdown(context.Background(), "port-nosnap", "tester")
	require.NoError(t, err, "快照失败不得阻断写路径（可用性优先）")
	require.NotNil(t, result)
	assert.Equal(t, "succeeded", result.Status)
}

// TestPortSecStaticBindFailureLeavesNoRollbackSnapshot 静态 MAC 绑定失败时，
// 快照无可回滚命令（DB 无绑定原值）→ no_rollback 语义，不是回滚失败。
func TestPortSecStaticBindFailureLeavesNoRollbackSnapshot(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-norb", "device-norb", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockExec.On("ExecuteCustom", mock.Anything, "device-norb", mock.Anything, singlePortTimeout).
		Return(errors.New("ssh transport down"))
	svc := newTestService(mockExec, nil, db)

	_, err := svc.PortSecWrite(context.Background(), "port-norb",
		portcollection.PortSecStaticBind,
		PortSecParams{MACAddress: "AA-BB-CC-DD-EE-FF", VLANID: 100}, "tester")
	require.Error(t, err, "下发失败必须返回错误")

	var snap models.PortWriteSnapshot
	require.NoError(t, db.Where("port_id = ?", "port-norb").First(&snap).Error)
	assert.Equal(t, "no_rollback", snap.Status,
		"静态 MAC 绑定无 DB 原值可回滚 → no_rollback（≠ rollback_failed）")
}

// TestWithProgressNotifierInjectsNotifier WithProgressNotifier 必须把 notifier
// 挂到 service 上并返回自身（链式注入）。
func TestWithProgressNotifierInjectsNotifier(t *testing.T) {
	svc := newTestService(&countingExecutor{}, nil, newTestDB(t))
	notifier := &recordingNotifier{}

	returned := svc.WithProgressNotifier(notifier)
	assert.Same(t, svc, returned, "必须返回接口自身以支持链式调用")
	assert.Same(t, notifier, svc.progressNotifier, "notifier 必须被挂到 service 上")
}

// TestNewPortWriteServiceWiresPingFunc 构造函数必须装上默认 pingFunc（预检依赖）。
func TestNewPortWriteServiceWiresPingFunc(t *testing.T) {
	db := newTestDB(t)
	svc := NewPortWriteService(db, nil, nil)
	require.NotNil(t, svc)
	impl, ok := svc.(*portWriteServiceImpl)
	require.True(t, ok)
	require.NotNil(t, impl.pingFunc, "默认 pingFunc 必须装配（预检不可达探测依赖）")
	assert.Same(t, db, impl.db, "构造函数必须原样持有传入的 db")
}

// recordingNotifier 记录批量进度推送（WithProgressNotifier 注入点）。
type recordingNotifier struct {
	calls int
}

func (n *recordingNotifier) BroadcastPortWriteProgressToUsers(_ []string, _ websocket.PortWriteProgressMessage) {
	n.calls++
}
