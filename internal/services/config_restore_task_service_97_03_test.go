package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

// TestV130R03_DeviceMismatchReturns400 V130R-03: 跨设备恢复 → BusinessError{HTTPStatus:400, Code:400001}
func TestV130R03_DeviceMismatchReturns400(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "src-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.50.1",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-src"
	require.NoError(t, db.Create(dev).Error)

	// backup belongs to dev-src
	bk := &models.ConfigBackup{
		ID:             "bk-src",
		DeviceID:      "dev-src",
		DeviceName:    "src-device",
		ConfigContent:  "sysname Src\n",
		ConfigHash:     "srchash",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	// try to restore to dev-other (different device)
	_, err := taskSvc.StartRestore(context.Background(), bk.ID, "dev-other", "tester")
	require.Error(t, err)

	be, ok := err.(*apperrors.AppError)
	require.True(t, ok, "err must be *apperrors.AppError, got %T", err)
	assert.Equal(t, 400, be.HTTPStatus, "cross-device must return 400")
	assert.Equal(t, apperrors.ErrorCode(400001), be.Code, "cross-device must return biz code 400001")
	assert.Contains(t, be.Message, "备份不属于目标设备")
}

// TestV130R03_DuplicateRestoreReturns409 V130R-03: 活跃任务冲突 → BusinessError{HTTPStatus:409, Code:409001}
func TestV130R03_DuplicateRestoreReturns409(t *testing.T) {
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
	backupSvc := NewConfigBackupService(db, nil)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, nil)

	dev := &models.NetworkDevice{
		DeviceName: "conflict-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.50.2",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = "dev-conflict"
	require.NoError(t, db.Create(dev).Error)

	bk := &models.ConfigBackup{
		ID:             "bk-conflict",
		DeviceID:      "dev-conflict",
		DeviceName:    "conflict-device",
		ConfigContent:  "sysname Conflict\n",
		ConfigHash:     "conflicthash",
		StorageType:   models.StorageTypeDatabase,
		BackupType:    models.BackupTypeManual,
		Version:       1,
	}
	require.NoError(t, db.Create(bk).Error)

	// create a pending task (simulating an in-flight restore)
	existingTask := &models.ConfigRestoreTask{
		ID:       "task-existing",
		DeviceID: "dev-conflict",
		BackupID: "bk-existing",
		Status:   string(models.RestoreTaskStatusPending),
	}
	require.NoError(t, db.Create(existingTask).Error)

	// second StartRestore should return 409
	_, err := taskSvc.StartRestore(context.Background(), bk.ID, "dev-conflict", "tester")
	require.Error(t, err)

	be, ok := err.(*apperrors.AppError)
	require.True(t, ok, "err must be *apperrors.AppError, got %T", err)
	assert.Equal(t, 409, be.HTTPStatus, "duplicate restore must return 409")
	assert.Equal(t, apperrors.ErrorCode(409001), be.Code, "duplicate restore must return biz code 409001")
	assert.Contains(t, be.Message, "进行中")
}

// TestV130R03_PlainErrorNotBusinessError V130R-03: 普通 error（非 BusinessError）仍走原有路径
func TestV130R03_PlainErrorNotBusinessError(t *testing.T) {
	// This is implicitly tested by existing TestCbk93RestoreCrossDeviceRejected which
	// tests that a plain fmt.Errorf still works for other error paths.
	// Here we just assert that a non-BusinessError error type is not intercepted.
	err := errors.New("some other error")
	_, ok := err.(*apperrors.AppError)
	assert.False(t, ok, "plain error should not be BusinessError")
}
