package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestV130R02_GracePeriodRunningNotConverged V130R-02: 5 分钟前的 running 任务（grace period 内）
// 不被 RecoverStaleRunningTasks 收敛。
func TestV130R02_GracePeriodRunningNotConverged(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "grace-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.200.1",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-grace-5min"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:             "bk-grace-5min",
		DeviceID:      "dev-grace-5min",
		DeviceName:    "grace-device",
		ConfigContent:  "sysname Grace\n",
		ConfigHash:     "gracehash",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	task := &models.ConfigRestoreTask{
		ID:       "task-grace-5min",
		DeviceID: "dev-grace-5min",
		BackupID: "bk-grace-5min",
		Status:   string(models.RestoreTaskStatusRunning),
	}
	require.NoError(t, db.Create(task).Error)

	// Backdate updated_at to 5 minutes ago (within 10-min grace period)
	old := time.Now().Add(-5 * time.Minute)
	require.NoError(t, db.Model(&models.ConfigRestoreTask{}).Where("id = ?", task.ID).
		Update("updated_at", old).Error)

	// RecoverStaleRunningTasks should NOT converge this running task
	taskSvc.RecoverStaleRunningTasks(context.Background())

	var got models.ConfigRestoreTask
	require.NoError(t, db.Where("id = ?", task.ID).First(&got).Error)
	assert.Equal(t, string(models.RestoreTaskStatusRunning), got.Status,
		"running task within 10-min grace period must NOT be converged")
}

// TestV130R02_GracePeriodExpiredRunningConverged V130R-02: 15 分钟前的 running 任务（grace period 外）
// 被 RecoverStaleRunningTasks 收敛为 failed。
func TestV130R02_GracePeriodExpiredRunningConverged(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "expired-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.200.2",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-expired"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:             "bk-expired",
		DeviceID:      "dev-expired",
		DeviceName:    "expired-device",
		ConfigContent:  "sysname Expired\n",
		ConfigHash:     "expiredhash",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	task := &models.ConfigRestoreTask{
		ID:       "task-expired",
		DeviceID: "dev-expired",
		BackupID: "bk-expired",
		Status:   string(models.RestoreTaskStatusRunning),
	}
	require.NoError(t, db.Create(task).Error)

	// Backdate updated_at to 15 minutes ago (outside 10-min grace period)
	old := time.Now().Add(-15 * time.Minute)
	require.NoError(t, db.Model(&models.ConfigRestoreTask{}).Where("id = ?", task.ID).
		Update("updated_at", old).Error)

	// RecoverStaleRunningTasks SHOULD converge this running task
	taskSvc.RecoverStaleRunningTasks(context.Background())

	var got models.ConfigRestoreTask
	require.NoError(t, db.Where("id = ?", task.ID).First(&got).Error)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status,
		"running task outside 10-min grace period must be converged to failed")
	assert.Contains(t, got.ErrorMessage, "服务重启")
}

// TestV130R02_PendingAlwaysConverged V130R-02: 无论 grace period 内外，pending 任务都收敛
// （pending 无 goroutine 认领，是真孤儿）。
func TestV130R02_PendingAlwaysConverged(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "orphan-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.200.3",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-orphan"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:             "bk-orphan",
		DeviceID:      "dev-orphan",
		DeviceName:    "orphan-device",
		ConfigContent:  "sysname Orphan\n",
		ConfigHash:     "orphanhash",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	orphan := &models.ConfigRestoreTask{
		ID:       "task-orphan",
		DeviceID: "dev-orphan",
		BackupID: "bk-orphan",
		Status:   string(models.RestoreTaskStatusPending),
	}
	require.NoError(t, db.Create(orphan).Error)

	// pending 无 grace period，即使刚刚创建也应该被收敛
	taskSvc.RecoverStaleRunningTasks(context.Background())

	var got models.ConfigRestoreTask
	require.NoError(t, db.Where("id = ?", orphan.ID).First(&got).Error)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status,
		"orphaned pending must always converge regardless of age")
	assert.Contains(t, got.ErrorMessage, "服务重启")
}
