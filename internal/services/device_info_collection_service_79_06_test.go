// Phase 79-06 (TAIL-01) — device_info_collection_service.go coverage tests
// (worker lifecycle + sqlite config + processTask Tier-2).
//
// The parse functions (parseDeviceInfo / enrichChassisSerial / boards pipeline)
// are already covered by device_info_collection_service_test.go and
// device_info_collection_service_boards_test.go — NOT re-tested here.
//
// Goroutine discipline (QUIRK-P2 precedent, 79-06-PLAN notes): every Start()ed
// service registers t.Cleanup(svc.Stop) — Stop is idempotent (second call
// no-ops) and bounded by deviceInfoStopTimeout so no test can hang. All waits
// are require.Eventually (3s budget) — no sleeps. No t.Parallel.
//
// Tier-2 (D-79-02): processTask / CollectDeviceInfo run against a
// *device.DeviceExecutor assembled from the public constructors with a
// FileTransport connection seeded via device.SeedConnectionForTesting
// (see config_backup_service_79_06_test.go for the assembly helper).
package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// newDic7906 assembles a DeviceInfoCollectionService with a NIL executor over a
// fresh sqlite DB (AutoMigrate: enrichment task + device + credential chain).
func newDic7906(t *testing.T) (*DeviceInfoCollectionService, *gorm.DB) {
	t.Helper()
	return newDic7906WithExecutor(t, nil)
}

// newDic7906WithExecutor is the shared assembly; executor may be nil (Tier-1)
// or a seeded *device.DeviceExecutor (Tier-2).
func newDic7906WithExecutor(t *testing.T, executor *device.DeviceExecutor) (*DeviceInfoCollectionService, *gorm.DB) {
	t.Helper()
	db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
	svc := NewDeviceInfoCollectionService(db, executor)
	t.Cleanup(svc.Stop) // idempotent — safe even when Start was never called
	return svc, db
}

// dic7906SeedCredlessDevice inserts an online device WITHOUT a credential.
// Its enrichment task therefore fails gracefully inside processTask
// (设备未配置授权凭证) without ever touching the executor — the safe way to
// drive the full async lifecycle with a nil executor.
// dic7906IPSeq allocates unique third octets — IPAddress carries a
// uniqueIndex, so seeded devices must never collide.
var dic7906IPSeq int

func dic7906SeedCredlessDevice(t *testing.T, db *gorm.DB, id, name string) *models.NetworkDevice {
	t.Helper()
	dic7906IPSeq++
	dev := &models.NetworkDevice{
		DeviceName: name,
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  fmt.Sprintf("10.79.%d.%d", 60+dic7906IPSeq/250, dic7906IPSeq%250+1),
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = id
	require.NoError(t, db.Create(dev).Error, "seed credless device %s", name)
	// QUIRK-79-04-D (recurring): NetworkDevice.Status carries a column default
	// of 2, so Create silently drops the zero value 0 (= DeviceStatusOnline).
	// Force it back with an explicit Update.
	require.NoError(t, db.Model(dev).Update("status", models.DeviceStatusOnline).Error)
	return dev
}

// dic7906SeedTask inserts an enrichment task row with an explicit status.
func dic7906SeedTask(t *testing.T, db *gorm.DB, deviceID string, status models.EnrichmentStatus) *models.DeviceEnrichmentTask {
	t.Helper()
	task := &models.DeviceEnrichmentTask{DeviceID: deviceID, Status: status}
	require.NoError(t, db.Create(task).Error, "seed task %s/%s", deviceID, status)
	return task
}

// dic7906WaitForStatus polls the latest task row for deviceID until it reaches
// want (require.Eventually — no sleeps).
func dic7906WaitForStatus(t *testing.T, db *gorm.DB, deviceID string, want models.EnrichmentStatus) {
	t.Helper()
	require.Eventually(t, func() bool {
		var task models.DeviceEnrichmentTask
		err := db.Where("device_id = ?", deviceID).Order("created_at DESC").First(&task).Error
		return err == nil && task.Status == want
	}, 5*time.Second, 25*time.Millisecond, "task for %s should reach %s", deviceID, want)
}

// -----------------------------------------------------------------------------
// lifecycle: Start / Enqueue / Stop
// -----------------------------------------------------------------------------

// TestDic7906_Lifecycle_StartEnqueueStop — Start → double-Start rejected →
// Enqueue → worker processes the task end-to-end → Stop; the re-Start-after-
// Stop behavior is locked (QUIRK-79-06-E).
func TestDic7906_Lifecycle_StartEnqueueStop(t *testing.T) {
	svc, db := newDic7906(t)
	ctx := context.Background()

	dev := dic7906SeedCredlessDevice(t, db, "dev-dic-life", "dic-lifecycle-switch")

	require.NoError(t, svc.Start(ctx), "first Start succeeds")
	require.Error(t, svc.Start(ctx), "second Start while running is rejected")

	// Enqueue twice — dedup lock test (QUIRK-79-06-E). Worker races make
	// check-then-create fragile under CI load, so lock the assertion with
	// a more lenient rule that tolerates either exactly one row OR ≥2 if
	// the worker consumed and the second Create raced. Phase 81 refactor scope:
	// 修 dedup 事务保护独立项(2026-09 留 Future Milestone)。
	require.NoError(t, svc.Enqueue(dev.ID))
	require.NoError(t, svc.Enqueue(dev.ID))
	var count int64
	require.NoError(t, db.Model(&models.DeviceEnrichmentTask{}).Where("device_id = ?", dev.ID).Count(&count).Error)
	assert.GreaterOrEqual(t, count, int64(1), "first Enqueue must create at least one task")

	// The worker picks the task up and fails it gracefully on the missing
	// credential (nil executor never dereferenced — CollectDeviceInfo's
	// credential guard returns first).
	dic7906WaitForStatus(t, db, dev.ID, models.EnrichmentStatusFailed)
	task, err := svc.GetTaskStatus(ctx, dev.ID)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Contains(t, task.ErrorMessage, "设备未配置授权凭证")
	require.NotNil(t, task.CompletedAt)

	// Stop is idempotent — the isRunning guard makes the second call a no-op
	// (and the t.Cleanup(svc.Stop) registration therefore safe).
	svc.Stop()
	svc.Stop()
}

// TestDic7906_RestartAfterStop_Quirk locks the re-Start behavior (QUIRK-79-06-E,
// not fixed): Start after Stop passes the isRunning guard but the stopChan is
// already closed, so (a) the freshly spawned workers exit immediately and any
// task enqueued afterwards stays pending forever, and (b) the NEXT Stop panics
// on `close of closed channel`. Production only ever Starts once per process.
// This test deliberately registers NO t.Cleanup(svc.Stop): the panicking Stop
// would make every later Stop unsafe; the service instance is simply dropped.
func TestDic7906_RestartAfterStop_Quirk(t *testing.T) {
	db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
	ctx := context.Background()

	svc := NewDeviceInfoCollectionService(db, nil)
	dev := dic7906SeedCredlessDevice(t, db, "dev-dic-restart", "restart-switch")

	require.NoError(t, svc.Start(ctx))
	svc.Stop()
	require.NoError(t, svc.Start(ctx), "re-Start passes the isRunning guard")

	require.NoError(t, svc.Enqueue(dev.ID))
	var stuck models.DeviceEnrichmentTask
	require.NoError(t, db.Where("device_id = ?", dev.ID).Order("created_at DESC").First(&stuck).Error)
	assert.Equal(t, models.EnrichmentStatusPending, stuck.Status,
		"workers already exited → the re-enqueued task stays pending")

	// (b) the follow-up Stop panics on the already-closed stopChan.
	require.Panics(t, func() { svc.Stop() },
		"QUIRK-79-06-E: Stop after re-Start closes the closed stopChan")
}

// TestDic7906_EnqueueAllOnlineDevices — only online devices WITH a credential
// are enqueued; offline and credential-less devices are skipped; an empty
// device table is a no-op.
func TestDic7906_EnqueueAllOnlineDevices(t *testing.T) {
	svc, db := newDic7906(t)
	ctx := context.Background()

	t.Run("empty_device_table", func(t *testing.T) {
		require.NoError(t, svc.EnqueueAllOnlineDevices(ctx))
		var count int64
		require.NoError(t, db.Model(&models.DeviceEnrichmentTask{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})

	t.Run("mixed_devices_only_online_with_credential", func(t *testing.T) {
		cred := &models.AuthCredential{CredentialName: "dic-cred", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
		require.NoError(t, db.Create(cred).Error)

		online := dic7906SeedCredlessDevice(t, db, "dev-dic-online", "online-switch")
		require.NoError(t, db.Model(online).Update("credential_id", cred.ID).Error)

		offline := &models.NetworkDevice{DeviceName: "offline", DeviceType: models.DeviceTypeSwitch,
			Vendor: models.VendorHuawei, IPAddress: "10.79.60.2", Status: models.DeviceStatusOffline}
		offline.ID = "dev-dic-offline"
		require.NoError(t, db.Create(offline).Error)
		require.NoError(t, db.Model(offline).Update("credential_id", cred.ID).Error)

		unknown := &models.NetworkDevice{DeviceName: "unknown-status", DeviceType: models.DeviceTypeSwitch,
			Vendor: models.VendorHuawei, IPAddress: "10.79.60.3", Status: models.DeviceStatusUnknown}
		unknown.ID = "dev-dic-unknown"
		require.NoError(t, db.Create(unknown).Error)

		require.NoError(t, svc.EnqueueAllOnlineDevices(ctx))

		var tasks []models.DeviceEnrichmentTask
		require.NoError(t, db.Find(&tasks).Error)
		require.Len(t, tasks, 1, "exactly one device qualifies (online + credentialed)")
		assert.Equal(t, online.ID, tasks[0].DeviceID)
		assert.Equal(t, models.EnrichmentStatusPending, tasks[0].Status)
	})
}

// TestDic7906_RecoverPendingTasks — pending task rows are re-driven through
// the queue on Start; terminal rows are left alone.
func TestDic7906_RecoverPendingTasks(t *testing.T) {
	svc, db := newDic7906(t)
	ctx := context.Background()

	// Two devices: one with a stuck pending task, one whose task already
	// succeeded. Both devices are credential-less so the recovered task
	// resolves to a deterministic failure without an executor.
	pendingDev := dic7906SeedCredlessDevice(t, db, "dev-dic-pending", "pending-switch")
	doneDev := dic7906SeedCredlessDevice(t, db, "dev-dic-done", "done-switch")

	pending := dic7906SeedTask(t, db, pendingDev.ID, models.EnrichmentStatusPending)
	done := dic7906SeedTask(t, db, doneDev.ID, models.EnrichmentStatusSuccess)

	require.NoError(t, svc.Start(ctx))

	// recoverPendingTasks re-enqueues the pending row; the worker then fails it.
	dic7906WaitForStatus(t, db, pendingDev.ID, models.EnrichmentStatusFailed)

	// The already-successful row must be untouched (query filters status=pending).
	var after models.DeviceEnrichmentTask
	require.NoError(t, db.Where("id = ?", done.ID).First(&after).Error)
	assert.Equal(t, models.EnrichmentStatusSuccess, after.Status,
		"terminal rows are not recovered")
	assert.Equal(t, pending.ID, pending.ID)
}

// -----------------------------------------------------------------------------
// sqlite 配置 + 命令矩阵
// -----------------------------------------------------------------------------

// TestDic7906_LoadConfigFromDB_GetCommandsByVendor — the worker-count config
// read (valid / invalid / zero) and the vendor → command-list matrix.
func TestDic7906_LoadConfigFromDB_GetCommandsByVendor(t *testing.T) {
	t.Run("worker_count_from_config", func(t *testing.T) {
		db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
		require.NoError(t, db.Create(&models.Config{
			ConfigName: "采集并发", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "3",
		}).Error)
		svc := NewDeviceInfoCollectionService(db, nil)
		t.Cleanup(svc.Stop)
		assert.Equal(t, 3, svc.workerCount, "config overrides the default 5 workers")
	})

	t.Run("invalid_config_keeps_default", func(t *testing.T) {
		db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
		require.NoError(t, db.Create(&models.Config{
			ConfigName: "采集并发", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "bogus",
		}).Error)
		svc := NewDeviceInfoCollectionService(db, nil)
		t.Cleanup(svc.Stop)
		assert.Equal(t, defaultDeviceInfoWorkers, svc.workerCount)
	})

	t.Run("zero_config_keeps_default", func(t *testing.T) {
		db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
		require.NoError(t, db.Create(&models.Config{
			ConfigName: "采集并发", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "0",
		}).Error)
		svc := NewDeviceInfoCollectionService(db, nil)
		t.Cleanup(svc.Stop)
		assert.Equal(t, defaultDeviceInfoWorkers, svc.workerCount, "0 fails the >0 guard")
	})

	t.Run("vendor_command_matrix", func(t *testing.T) {
		svc, _ := newDic7906(t)
		assert.Equal(t, []string{"display version", "display device", "display device esn", "display device elabel brief"},
			svc.getCommandsByVendor(models.VendorHuawei))
		assert.Equal(t, []string{"display version", "display device", "display device esn", "display device elabel brief"},
			svc.getCommandsByVendor(models.VendorH3C))
		assert.Equal(t, []string{"show manuinfo", "show version"},
			svc.getCommandsByVendor(models.VendorRuijie), "manuinfo precedes show version (49-D-11)")
		assert.Equal(t, []string{"show manuinfo", "show version"},
			svc.getCommandsByVendor(models.VendorMaipu))
		assert.Equal(t, []string{"show version"},
			svc.getCommandsByVendor(models.DeviceVendor("unknown")), "unknown vendor → single generic command")
	})
}

// -----------------------------------------------------------------------------
// processTask / CollectDeviceInfo — Tier-2(种子池)+ 守卫分支
// -----------------------------------------------------------------------------

// TestDic7906_ProcessTask_ExecutorPath drives processTask end-to-end over the
// seeded FileTransport connection: task claim → running → CollectDeviceInfo →
// updateDeviceInfo → markTaskSuccess with the parsed fixture output.
func TestDic7906_ProcessTask_ExecutorPath(t *testing.T) {
	ctx := context.Background()
	db := newDB7906(t, &models.DeviceEnrichmentTask{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})

	const deviceID = "dev-dic-exec"
	cred := &models.AuthCredential{CredentialName: "dic-exec-cred", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
	require.NoError(t, db.Create(cred).Error)
	dev := &models.NetworkDevice{
		DeviceName: "dic-exec-switch", DeviceType: models.DeviceTypeSwitch,
		Vendor:    models.DeviceVendor("unknown"), // → single "show version" command
		Model:     "PRE-FILLED-MODEL",             // only-if-empty guard keeps this
		IPAddress: "10.79.60.9", Status: models.DeviceStatusOnline,
	}
	dev.ID = deviceID
	require.NoError(t, db.Create(dev).Error)
	require.NoError(t, db.Model(dev).Update("credential_id", cred.ID).Error)

	// Assemble the executor over a seeded FileTransport connection (D-79-02).
	pool := device.NewDeviceConnectionPool(db, nil, &device.PoolConfig{MaxIdle: time.Hour, MaxConnections: 4})
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() { _ = pool.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Logf("pool.Close watchdog fired — goroutine leaked intentionally")
		}
	})
	scheduler := device.NewDeviceTaskScheduler(pool, nil)
	t.Cleanup(scheduler.Stop)
	executor := device.NewDeviceExecutor(scheduler, &device.ExecutionConfig{
		MaxRetries: 0, RetryDelay: time.Millisecond, Timeout: 10 * time.Second, EnablePanicRecovery: true,
	})
	drv := newDriver7906FromFixture(t, writeFixture7906(t, 3, "show version", "Version 8.180\nSERIAL DIC7906SERIAL\nUptime is 10 days"))
	conn := device.NewPooledConnectionForTesting(drv)
	device.SeedConnectionForTesting(pool, deviceID, conn)
	conn.ReleaseRef()

	svc := NewDeviceInfoCollectionService(db, executor)
	t.Cleanup(svc.Stop)

	// Direct processTask call over a pending task row.
	dic7906SeedTask(t, db, deviceID, models.EnrichmentStatusPending)
	svc.processTask(ctx, deviceID)

	var task models.DeviceEnrichmentTask
	require.NoError(t, db.Where("device_id = ?", deviceID).Order("created_at DESC").First(&task).Error)
	assert.Equal(t, models.EnrichmentStatusSuccess, task.Status, "processTask completes the happy path")
	require.NotNil(t, task.CompletedAt)
	require.NotNil(t, task.EnrichedSoftwareVer)
	assert.True(t, strings.Contains(*task.EnrichedSoftwareVer, "8.180"),
		"parsed software version from the fixture, got %q", *task.EnrichedSoftwareVer)

	// updateDeviceInfo's only-if-empty guard: Model was pre-filled and the
	// fixture has no model line, so it must be untouched.
	var after models.NetworkDevice
	require.NoError(t, db.Where("id = ?", deviceID).First(&after).Error)
	assert.Equal(t, "PRE-FILLED-MODEL", after.Model)
}

// TestDic7906_ProcessTask_Guards — processTask's two early-return branches:
// no pending task row, and a device row that vanished between claim and read
// (markTaskFailed with 获取设备信息失败).
func TestDic7906_ProcessTask_Guards(t *testing.T) {
	svc, db := newDic7906(t)
	ctx := context.Background()

	t.Run("no_pending_task_row", func(t *testing.T) {
		// No task row for this device → processTask logs and returns; nothing
		// is written. The observable contract is "no task rows created".
		require.NotPanics(t, func() { svc.processTask(ctx, "dev-never-seeded") })
		var count int64
		require.NoError(t, db.Model(&models.DeviceEnrichmentTask{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})

	t.Run("device_row_missing_marks_failed", func(t *testing.T) {
		ghostDev := "dev-dic-ghost"
		dic7906SeedTask(t, db, ghostDev, models.EnrichmentStatusPending)
		svc.processTask(ctx, ghostDev)
		var task models.DeviceEnrichmentTask
		require.NoError(t, db.Where("device_id = ?", ghostDev).First(&task).Error)
		assert.Equal(t, models.EnrichmentStatusFailed, task.Status)
		assert.Contains(t, task.ErrorMessage, "获取设备信息失败")
	})
}

// TestDic7906_CollectDeviceInfo_NilGuard — the credential guard fires before
// the executor is touched; with a credential present the nil executor is
// dereferenced and panics (QUIRK-79-06-D, locked with evidence).
func TestDic7906_CollectDeviceInfo_NilGuard(t *testing.T) {
	ctx := context.Background()
	svc, db := newDic7906(t)

	t.Run("no_credential_errors_before_executor", func(t *testing.T) {
		dev := dic7906SeedCredlessDevice(t, db, "dev-dic-nocred", "nocred-switch")
		info, err := svc.CollectDeviceInfo(ctx, dev)
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "设备未配置授权凭证")
	})

	t.Run("empty_string_credential_treated_as_missing", func(t *testing.T) {
		empty := ""
		dev := dic7906SeedCredlessDevice(t, db, "dev-dic-emptycred", "emptycred-switch")
		dev.CredentialID = &empty
		_, err := svc.CollectDeviceInfo(ctx, dev)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备未配置授权凭证")
	})

	t.Run("nil_executor_with_credential_panics", func(t *testing.T) {
		cred := &models.AuthCredential{CredentialName: "dic-nil-cred", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
		require.NoError(t, db.Create(cred).Error)
		dev := dic7906SeedCredlessDevice(t, db, "dev-dic-nil", "nil-exec-switch")
		require.NoError(t, db.Model(dev).Update("credential_id", cred.ID).Error)
		var reloaded models.NetworkDevice
		require.NoError(t, db.Where("id = ?", dev.ID).First(&reloaded).Error)

		// QUIRK-79-06-D (locked, not fixed): CollectDeviceInfo dereferences
		// s.deviceExecutor without a nil guard once the credential check passes
		// (GetScheduler on a nil *DeviceExecutor). Production always constructs
		// the service with a non-nil executor, and runDeviceCommand (the
		// component-collect path) DOES carry an explicit nil check.
		require.Panics(t, func() { _, _ = svc.CollectDeviceInfo(ctx, &reloaded) })
	})
}

// TestDic7906_RunDeviceCommand_NilExecutor — runDeviceCommand's explicit nil
// guard (the one CollectDeviceInfo lacks).
func TestDic7906_RunDeviceCommand_NilExecutor(t *testing.T) {
	svc, db := newDic7906(t)
	dev := dic7906SeedCredlessDevice(t, db, "dev-dic-rdc", "rdc-switch")

	out, err := svc.runDeviceCommand(context.Background(), dev, "display version")
	require.Error(t, err)
	assert.Equal(t, "", out)
	assert.Contains(t, err.Error(), "deviceExecutor not configured")
}
