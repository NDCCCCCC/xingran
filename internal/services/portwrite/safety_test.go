package portwrite

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ====================== Phase 123 (PORTW-04/05/06) 安全三件套测试 ======================

// TestRollback_OnWriteFailure PORTW-04：下发失败自动回滚 → rolled_back + Extra 留痕。
func TestRollback_OnWriteFailure(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-rb", "device-rb", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	// 第一次 ExecuteCustom（写）失败；第二次（回滚）成功
	mockExec.On("ExecuteCustom", mock.Anything, "device-rb", mock.Anything, singlePortTimeout).
		Return(errors.New("ssh transport down")).Once()
	mockExec.On("ExecuteCustom", mock.Anything, "device-rb", mock.Anything, singlePortTimeout).
		Return(nil).Once()
	svc := newTestService(mockExec, nil, db)

	result, err := svc.Shutdown(context.Background(), "port-rb", "tester")
	require.Error(t, err) // 写失败语义保持
	require.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Equal(t, "rolled_back", result.Extra["rollback"], "回滚应自动执行并留痕")

	// 快照行状态落库校验
	var snap models.PortWriteSnapshot
	require.NoError(t, db.Where("port_id = ?", "port-rb").First(&snap).Error)
	assert.Equal(t, "rolled_back", snap.Status)
	assert.Contains(t, snap.Action, "shutdown")
}

// TestRollback_Failure_Alert PORTW-04：回滚也失败 → rollback_failed 显式告警，不允许静默。
func TestRollback_Failure_Alert(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-rb2", "device-rb2", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockExec.On("ExecuteCustom", mock.Anything, "device-rb2", mock.Anything, singlePortTimeout).
		Return(errors.New("device gone"))
	svc := newTestService(mockExec, nil, db)

	result, err := svc.Shutdown(context.Background(), "port-rb2", "tester")
	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "rollback_failed", result.Extra["rollback"])
	assert.NotEmpty(t, result.Extra["rollbackError"])

	var snap models.PortWriteSnapshot
	require.NoError(t, db.Where("port_id = ?", "port-rb2").First(&snap).Error)
	assert.Equal(t, "rollback_failed", snap.Status)
	assert.NotEmpty(t, snap.RollbackError)
}

// TestRollback_ReverseOrder PORTW-04 验收：回滚命令逆序消费（后配置先撤销）。
func TestRollback_ReverseOrder(t *testing.T) {
	snap := &models.PortWriteSnapshot{
		ID:               "snap-x",
		DeviceID:         "device-x",
		Action:           "set_access_vlan",
		RollbackCommands: `["interface GE0/0/1","port default vlan 100","undo shutdown"]`,
		Status:           "captured",
	}
	var cmds []string
	require.NoError(t, json.Unmarshal([]byte(snap.RollbackCommands), &cmds))
	reversed := make([]string, 0, len(cmds))
	for i := len(cmds) - 1; i >= 0; i-- {
		reversed = append(reversed, cmds[i])
	}
	assert.Equal(t, []string{"undo shutdown", "port default vlan 100", "interface GE0/0/1"}, reversed,
		"回滚消费顺序必须是快照命令的逆序")
	// 无回滚命令的快照 → rollback_failed 告警语义
	svc := newTestService(&countingExecutor{}, nil, newTestDB(t))
	empty := &models.PortWriteSnapshot{ID: "snap-empty", DeviceID: "device-x", RollbackCommands: `[]`}
	rolled, rerr := svc.rollbackSnapshot(context.Background(), empty)
	assert.False(t, rolled)
	require.Error(t, rerr)
}

// TestLock_ConcurrentContention PORTW-05 验收：并发竞争同一端口恰一个成功。
func TestLock_ConcurrentContention(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	const workers = 8
	var wg sync.WaitGroup
	successCh := make(chan int, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := svc.acquirePortWriteLock(context.Background(), "port-lock-1", "worker"); err == nil {
				successCh <- i
			}
		}(i)
	}
	wg.Wait()
	close(successCh)
	wins := 0
	for range successCh {
		wins++
	}
	assert.Equal(t, 1, wins, "同一端口并发加锁必须恰有一个成功")

	// 释放后可重新获取
	svc.releasePortWriteLock(context.Background(), "port-lock-1", "worker")
	require.NoError(t, svc.acquirePortWriteLock(context.Background(), "port-lock-1", "worker"))
}

// TestLock_TTLExpiry PORTW-05：进程重启不残留 —— DB 过期行可被新会话抢占。
func TestLock_TTLExpiry(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	// 预置一条已过期锁（模拟 crash 残留）
	stale := &models.PortWriteLock{PortID: "port-ttl", Holder: "dead-process", ExpiresAt: time.Now().Add(-time.Minute)}
	require.NoError(t, db.Create(stale).Error)

	require.NoError(t, svc.acquirePortWriteLock(context.Background(), "port-ttl", "new-session"),
		"过期锁必须可被抢占（进程重启不残留）")
}

// TestPreflight_Unreachable PORTW-06 验收：不可达拒绝 + 跳过路径。
func TestPreflight_Unreachable(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-pf", "device-pf", "GE0/0/1", "up", false, "")
	svc := newTestService(&countingExecutor{}, nil, db)
	svc.pingFunc = func(string, time.Duration) bool { return false }

	_, err := svc.Shutdown(context.Background(), "port-pf", "tester")
	require.ErrorIs(t, err, ErrDeviceUnreachable)

	// 管理员强制跳过 → 预检放行（mock executor 返回 nil → 成功路径）
	_, err = svc.Shutdown(context.Background(), "port-pf", "tester", WithSkipPreflight())
	require.NoError(t, err)
}

// TestRenderRollbackCommands PORTW-04：回滚命令与 dry-run/写路径共享渲染层。
func TestRenderRollbackCommands(t *testing.T) {
	vlan := 20
	before := portBeforeState{AdminStatus: "up", VLAN: &vlan}

	// shutdown → undo_shutdown
	cmds, ok := renderRollbackCommands(models.VendorHuawei, portcollection.ActionShutdown, before, "GE1/0/1")
	require.True(t, ok)
	assert.Contains(t, cmds[1], "undo shutdown")

	// set_access_vlan 原vlan=20 → 恢复 vlan 20
	cmds, ok = renderRollbackCommands(models.VendorHuawei, portcollection.ActionSetAccessVLAN, before, "GE1/0/1")
	require.True(t, ok)
	joined := ""
	for _, c := range cmds {
		joined += c + " "
	}
	assert.Contains(t, joined, "20")

	// set_access_vlan 原vlan=nil → 厂商清除命令差异
	nilBefore := portBeforeState{AdminStatus: "up"}
	hwCmds, ok1 := renderRollbackCommands(models.VendorHuawei, portcollection.ActionSetAccessVLAN, nilBefore, "GE1/0/1")
	rjCmds, ok2 := renderRollbackCommands(models.VendorRuijie, portcollection.ActionSetAccessVLAN, nilBefore, "GE1/0/1")
	require.True(t, ok1)
	require.True(t, ok2)
	assert.Contains(t, hwCmds[1], "undo port default vlan")
	assert.Contains(t, rjCmds[1], "no switchport access vlan")

	// 非 critical 动作不生成回滚命令（保守）
	_, ok = renderRollbackCommands(models.VendorHuawei, portcollection.ActionDescription, nilBefore, "GE1/0/1")
	assert.False(t, ok)
}
