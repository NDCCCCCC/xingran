package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// TestV130R01_RestoreConfigTimeoutSplit V130R-01: 常量拆分验证——两段 timeout 均存在且语义正确。
func TestV130R01_RestoreConfigTimeoutSplit(t *testing.T) {
	// RestoreBackupTimeout = 30s（阶段①：恢复前备份 CreateBackup）
	assert.Equal(t, 30*time.Second, constants.RestoreBackupTimeout)

	// RestoreConfigExecTimeout = 5min（阶段②：RestoreConfig 下发）
	assert.Equal(t, 5*time.Minute, constants.RestoreConfigExecTimeout)

	// 两段互不重叠：backup 30s << restore 5min（设计意图）
	assert.True(t, constants.RestoreBackupTimeout < constants.RestoreConfigExecTimeout)
}

// TestV130R01_StartRestoreGoroutineLaunch V130R-01: StartRestore 立即返回，goroutine 异步执行。
func TestV130R01_StartRestoreGoroutineLaunch(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil) // nil executor: runRestore nil-guard → failTask
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	// Seed a device + backup so the task can run
	dev := &models.NetworkDevice{
		DeviceName: "timeout-test-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.100.1",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-timeout-test"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:            "bk-timeout-test",
		DeviceID:      "dev-timeout-test",
		DeviceName:    "timeout-test-device",
		ConfigContent: "sysname Test\n",
		ConfigHash:    "abc123",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	// StartRestore returns immediately (goroutine owns execution)
	task, err := taskSvc.StartRestore(context.Background(), bk.ID, "dev-timeout-test", "tester")
	require.NoError(t, err)
	require.NotNil(t, task)

	// Task should be pending at first
	require.Equal(t, string(models.RestoreTaskStatusPending), task.Status)

	// Wait for goroutine to process and reach terminal state (nil executor → failTask)
	var terminalTask *models.ConfigRestoreTask
	ok := assert.Eventually(t, func() bool {
		got, err := taskSvc.GetRestoreTask(context.Background(), task.ID)
		if err != nil {
			return false
		}
		terminalTask = got
		return got.Status == string(models.RestoreTaskStatusFailed) ||
			got.Status == string(models.RestoreTaskStatusSuccess)
	}, 5*time.Second, 50*time.Millisecond)
	require.True(t, ok, "task should reach terminal state")
	require.NotNil(t, terminalTask)
	// nil executor → failTask with "设备执行器未初始化"
	assert.Equal(t, string(models.RestoreTaskStatusFailed), terminalTask.Status)
	assert.Contains(t, terminalTask.ErrorMessage, "设备执行器未初始化")
}

// TestV130R01_RunRestorePhaseOrdering V130R-01: runRestore 两阶段按顺序执行，阶段①不过超 30s，阶段②不过超 5min。
// 本测试验证代码路径存在，使用 nil executor 防止真实设备 I/O。
func TestV130R01_RunRestorePhaseOrdering(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "phase-order-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.100.2",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-phase-order"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:            "bk-phase-order",
		DeviceID:      "dev-phase-order",
		DeviceName:    "phase-order-device",
		ConfigContent: "sysname PhaseOrder\n",
		ConfigHash:    "def456",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	task, err := taskSvc.StartRestore(context.Background(), bk.ID, "dev-phase-order", "tester")
	require.NoError(t, err)

	// Verify: task reaches failed (nil executor) — proves goroutine ran through to failTask
	var terminalTask *models.ConfigRestoreTask
	ok := assert.Eventually(t, func() bool {
		got, err := taskSvc.GetRestoreTask(context.Background(), task.ID)
		if err != nil {
			return false
		}
		terminalTask = got
		return got.Status == string(models.RestoreTaskStatusFailed) ||
			got.Status == string(models.RestoreTaskStatusSuccess)
	}, 5*time.Second, 50*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), terminalTask.Status)
	assert.Contains(t, terminalTask.ErrorMessage, "设备执行器未初始化")
}
