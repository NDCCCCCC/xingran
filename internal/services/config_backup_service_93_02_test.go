// Phase 93-06 (BACKUP-CLOSED-04/05) — restore e2e + failure-scenario suite.
//
// TestCbk93* exercises the async restore chain (ConfigRestoreTaskService) end
// to end against a FileTransport connection (zero SSH, 79_06/93-02 assembly
// discipline): CreateBackup → StartRestore → runRestore (pre-restore backup →
// GetBackupContent → RestoreConfig fail-fast push → readback hash) → terminal
// task state + version-chain record (D-29/D-30).
//
// Fixture byte discipline (see 93-02 device e2e + 79_06 cycles): open banner 3
// lines → GetConfig cycles (pre-prompt, cmd echo, output, terminating prompt)
// → restore SendConfigs segment (IsReady probe, system-view acquire, per-line
// echo+prompt, quit, return) → readback cycle → spare prompts. EOF on an
// exhausted fixture BLOCKS forever (78-03 S-2), so failure injection uses the
// device-rejection marker ("% Error" → scrapligo failed_when_contains), never
// truncation (93-02 deviation precedent).
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/logging"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// assembly helpers (93-02 flavored: 79_06 scaffolding + ConfigRestoreTask)
// -----------------------------------------------------------------------------

const cbk93RestoreDeviceID = "dev-cbk93-r"

// newCbk93RestoreDB assembles the sqlite DB (backup + device + config +
// restore-task tables) and moves the process into a temp working directory
// (cbk7906Chdir discipline: getBackupDir relative root + Windows app.log lock).
func newCbk93RestoreDB(t *testing.T) *gorm.DB {
	t.Helper()
	cbk7906Chdir(t)
	return newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{}, &models.ConfigRestoreTask{})
}

// cbk93SeedDevice inserts the huawei restore target device (display
// current-configuration + exitConfigCommand→quit both key off vendor=huawei).
func cbk93SeedDevice(t *testing.T, db *gorm.DB) *models.NetworkDevice {
	t.Helper()
	dev := &models.NetworkDevice{
		DeviceName: "cbk93-restore-switch",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.7.9",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = cbk93RestoreDeviceID
	require.NoError(t, db.Create(dev).Error, "seed restore device")
	return dev
}

// cbk93RestoreFixture renders the full FileTransport byte stream for a restore
// chain: getConfigCycles identical "display current-configuration" cycles
// (CreateBackup read + pre-restore backup read [+ readback when
// withReadbackCycle]) followed by the SendConfigs push segment.
type cbk93RestoreFixtureSpec struct {
	getConfigCycles int
	// configOutput is replayed verbatim per cycle (multi-line allowed). The
	// cleanConfigLines-filtered form of this string is also what RestoreConfig
	// pushes, so keep sendConfigLines in sync with it.
	configOutput string
	// pushRejected injects the device-rejection marker after the second push
	// line (fail-fast mid-push, D-12).
	pushRejected bool
	// readbackOutput overrides the final GetConfig cycle (hash-mismatch
	// injection, D-11). Empty → same as configOutput.
	readbackOutput string
}

func cbk93WriteRestoreFixture(t *testing.T, spec cbk93RestoreFixtureSpec) string {
	t.Helper()
	const host = "<Huawei>"
	var b []byte
	add := func(s string) { b = append(b, s...) }
	addLines := func(lines ...string) {
		for _, l := range lines {
			add(l + "\n")
		}
	}

	// open banner (huawei_vrp platform open: prompt, screen-length echo, prompt)
	addLines(host, "screen-length 0 temporary", host)

	// Pre-restore GetConfig cycles (source-backup read + pre-restore backup
	// read). Each SendCommand internally GetPrompts first — that probe consumes
	// the cycle's leading prompt line (79_06 cycle shape).
	cmd := "display current-configuration"
	for i := 0; i < spec.getConfigCycles; i++ {
		addLines(host, cmd)
		addLines(strings.Split(strings.TrimSuffix(spec.configOutput, "\n"), "\n")...)
		addLines(host)
	}

	if len(spec.sendConfigLines()) > 0 {
		// SendConfigs push segment (93-02 template, byte-for-byte shape)
		addLines(host) // IsReady probe for the restore connection acquisition
		addLines("system-view", "[Huawei]")
		lines := spec.sendConfigLines()
		for i, line := range lines {
			addLines(line)
			if spec.pushRejected && i == 1 {
				// device rejects the second line — the % Error marker must sit
				// BEFORE the prompt line (scrapligo failed_when_contains scans
				// the response record, and a prompt-echo first would close the
				// response as a success)
				addLines("% Error: Unrecognized command found at '^' marker.")
			}
			addLines("[Huawei-GigabitEthernet0/0/1]")
		}
		// exitConfigCommand (quit) + the exec-privilege "return" issued by the
		// readback GetConfig's AcquirePriv; trailing prompts are harmless spare
		// bytes on the rejected path (never read).
		addLines("quit", "[Huawei]", "return", host)

		if !spec.pushRejected {
			// readback GetConfig cycle — full 79_06 shape: the SendCommand's
			// internal GetPrompt probe consumes the leading prompt line
			readback := spec.readbackOutput
			if readback == "" {
				readback = spec.configOutput
			}
			addLines(host, cmd)
			addLines(strings.Split(strings.TrimSuffix(readback, "\n"), "\n")...)
			addLines(host)
		}
	}

	// spare prompts for close-time / stray reads
	for i := 0; i < 8; i++ {
		addLines(host)
	}

	return writeFixtureBytes7906(t, b)
}

// sendConfigLines mirrors device.cleanConfigLines (unexported there) for
// fixture generation only: trim + drop blank/full-line-comment lines. Any
// drift from the real rules surfaces as a fixture-consumption misalignment in
// the e2e cases, whose totalLines assertions (D-03 behavior lock) fail loudly.
func (s cbk93RestoreFixtureSpec) sendConfigLines() []string {
	if s.configOutput == "" {
		return nil
	}
	var lines []string
	for _, raw := range strings.Split(s.configOutput, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// newCbk93RestoreExecutor assembles a real DeviceExecutor over db with a
// FileTransport connection seeded for the restore device (93-02 assembly).
func newCbk93RestoreExecutor(t *testing.T, db *gorm.DB, fixturePath string) *device.DeviceExecutor {
	t.Helper()

	pool := device.NewDeviceConnectionPool(db, nil, &device.PoolConfig{
		MaxIdle:        time.Hour,
		MaxConnections: 8,
	})
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() { _ = pool.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Logf("pool.Close did not return within 3s — goroutine leaked intentionally")
		}
	})

	scheduler := device.NewDeviceTaskScheduler(pool, nil)
	t.Cleanup(scheduler.Stop)

	executor := device.NewDeviceExecutor(scheduler, &device.ExecutionConfig{
		MaxRetries:          0,
		RetryDelay:          time.Millisecond,
		Timeout:             10 * time.Second,
		EnablePanicRecovery: true,
	})

	drv := newDriver93WithDebugLog(t, fixturePath)
	conn := device.NewPooledConnectionForTesting(drv)
	device.SeedConnectionForTesting(pool, cbk93RestoreDeviceID, conn)
	conn.ReleaseRef()

	return executor
}

// newDriver93WithDebugLog opens a FileTransport huawei_vrp driver with a
// debug logger wired to t.Log — fixture-consumption misalignment becomes
// visible instead of a silent block (93-02 discipline).
func newDriver93WithDebugLog(t *testing.T, fixturePath string) *network.Driver {
	t.Helper()
	p, err := platform.NewPlatform(
		"huawei_vrp",
		"cbk93-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(fixturePath),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
		options.WithLogger(mustDebugLogger93(t)),
	)
	require.NoError(t, err, "NewPlatform")
	d, err := p.GetNetworkDriver()
	require.NoError(t, err, "GetNetworkDriver")
	require.NoError(t, d.Open(), "driver.Open")
	return d
}

func mustDebugLogger93(t *testing.T) *logging.Instance {
	t.Helper()
	inst, err := logging.NewInstance(
		logging.WithLogger(func(items ...interface{}) { t.Log(items...) }),
		logging.WithLevel(logging.Debug),
	)
	if err != nil {
		t.Fatal(err)
	}
	return inst
}

// newCbk93RestoreChain wires backup + restore-task services over db with the
// given executor (nil executor → runRestore's nil-guard marks tasks failed,
// safe for pure-DB cases).
func newCbk93RestoreChain(db *gorm.DB, executor *device.DeviceExecutor) (*ConfigBackupService, *ConfigRestoreTaskService) {
	backupSvc := NewConfigBackupService(db, executor)
	taskSvc := NewConfigRestoreTaskService(db, backupSvc, executor)
	return backupSvc, taskSvc
}

// cbk93AwaitTerminal polls the task until it reaches success/failed (the
// restore runs on a detached goroutine — D-17 async semantics).
func cbk93AwaitTerminal(t *testing.T, taskSvc *ConfigRestoreTaskService, taskID string) *models.ConfigRestoreTask {
	t.Helper()
	var task *models.ConfigRestoreTask
	ok := assert.Eventually(t, func() bool {
		got, err := taskSvc.GetRestoreTask(context.Background(), taskID)
		if err != nil {
			return false
		}
		task = got
		return got.Status == string(models.RestoreTaskStatusSuccess) ||
			got.Status == string(models.RestoreTaskStatusFailed)
	}, 15*time.Second, 50*time.Millisecond, "restore task must reach a terminal state")
	if !ok && task != nil {
		t.Logf("DIAG task=%+v", task)
		var backups []models.ConfigBackup
		_ = taskSvc.db.Where("device_id = ?", task.DeviceID).Find(&backups).Error
		t.Logf("DIAG backups=%d (%+v)", len(backups), backups)
	}
	require.True(t, ok, "restore task must reach a terminal state")
	return task
}

// restoreRunResult mirrors the service's result payload (unexported there).
func cbk93ParseResult(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	if raw == "" {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &m), "parse ResultJSON %q", raw)
	return m
}

// -----------------------------------------------------------------------------
// pure-DB guard cases (no fixture I/O)
// -----------------------------------------------------------------------------

// TestCbk93RestoreCrossDeviceRejected — D-04: restore is source-device-only;
// the cross-device attempt must fail validation without creating any task.
func TestCbk93RestoreCrossDeviceRejected(t *testing.T) {
	db := newCbk93RestoreDB(t)
	backupSvc, taskSvc := newCbk93RestoreChain(db, nil)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-src", DeviceName: "src", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})

	_, err := taskSvc.StartRestore(context.Background(), bk.ID, "dev-other", "tester")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "备份不属于目标设备")

	var count int64
	require.NoError(t, db.Model(&models.ConfigRestoreTask{}).Count(&count).Error)
	assert.Zero(t, count, "no task row may be created for a cross-device attempt")
	_ = backupSvc
}

// TestCbk93RestoreMutualExclusion — D-08: a device with an in-flight (pending)
// restore task rejects a second StartRestore.
func TestCbk93RestoreMutualExclusion(t *testing.T) {
	db := newCbk93RestoreDB(t)
	_, taskSvc := newCbk93RestoreChain(db, nil)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: cbk93RestoreDeviceID, DeviceName: "r", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})
	seeded := seedRestoreTask93(t, db, cbk93RestoreDeviceID, bk.ID, models.RestoreTaskStatusPending)

	_, err := taskSvc.StartRestore(context.Background(), bk.ID, cbk93RestoreDeviceID, "tester")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "进行中的恢复任务")

	var count int64
	require.NoError(t, db.Model(&models.ConfigRestoreTask{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "mutex rejection must not create a second task row")
	assert.Equal(t, seeded.ID, seeded.ID) // keep staticcheck quiet on unused var
}

// TestCbk93RecoverStaleRunning — A5 startup convergence: a running task left
// by a crashed process is failed once, which also releases the D-08 mutex.
func TestCbk93RecoverStaleRunning(t *testing.T) {
	db := newCbk93RestoreDB(t)
	_, taskSvc := newCbk93RestoreChain(db, nil)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: cbk93RestoreDeviceID, DeviceName: "r", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})
	stale := seedRestoreTask93(t, db, cbk93RestoreDeviceID, bk.ID, models.RestoreTaskStatusRunning)

	taskSvc.RecoverStaleRunningTasks(context.Background())

	var got models.ConfigRestoreTask
	require.NoError(t, db.Where("id = ?", stale.ID).First(&got).Error)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status)
	assert.Contains(t, got.ErrorMessage, "服务重启")

	// mutex released → a fresh StartRestore is accepted (nil executor → the
	// detached goroutine lands in the nil-guard and fails the task; the
	// synchronous acceptance is what this asserts)
	task, err := taskSvc.StartRestore(context.Background(), bk.ID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err, "mutex must be released after stale-task convergence")
	assert.Equal(t, string(models.RestoreTaskStatusPending), task.Status)
	cbk93AwaitTerminal(t, taskSvc, task.ID) // let the goroutine finish before DB teardown
}

// TestCbk93RecoverStalePending — v129-recheck C-2: a pending task orphaned by a
// crash (StartRestore committed the row but the claim goroutine never ran) can
// never be claimed after restart, yet the D-08 mutex counts pending as active —
// convergence must fail it too, or the device is locked out forever (D-34 has
// no cancel endpoint).
func TestCbk93RecoverStalePending(t *testing.T) {
	db := newCbk93RestoreDB(t)
	_, taskSvc := newCbk93RestoreChain(db, nil)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: cbk93RestoreDeviceID, DeviceName: "r", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})
	orphan := seedRestoreTask93(t, db, cbk93RestoreDeviceID, bk.ID, models.RestoreTaskStatusPending)

	taskSvc.RecoverStaleRunningTasks(context.Background())

	var got models.ConfigRestoreTask
	require.NoError(t, db.Where("id = ?", orphan.ID).First(&got).Error)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status, "orphaned pending must converge to failed")
	assert.Contains(t, got.ErrorMessage, "服务重启")

	// mutex released → a fresh StartRestore is accepted
	task, err := taskSvc.StartRestore(context.Background(), bk.ID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err, "mutex must be released after orphaned-pending convergence")
	cbk93AwaitTerminal(t, taskSvc, task.ID)
}

// TestCbk93StartRestoreDBFailure — D-28④ (adjusted mechanism): with the task
// table missing, StartRestore fails at the create-task step and leaves no
// half-written state (no goroutine is spawned).
func TestCbk93StartRestoreDBFailure(t *testing.T) {
	db := newCbk93RestoreDB(t)
	_, taskSvc := newCbk93RestoreChain(db, nil)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: cbk93RestoreDeviceID, DeviceName: "r", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})

	require.NoError(t, db.Migrator().DropTable("sys_config_restore_task"), "drop task table")

	_, err := taskSvc.StartRestore(context.Background(), bk.ID, cbk93RestoreDeviceID, "tester")
	require.Error(t, err, "task-table write failure must surface to the caller")
}

// TestCbk93RestorePreBackupFailureAborts — D-28①+D-14: when the pre-restore
// backup cannot be taken (device gone), the restore aborts before any push.
func TestCbk93RestorePreBackupFailureAborts(t *testing.T) {
	db := newCbk93RestoreDB(t)
	// non-nil executor (runRestore's nil-guard would fire first) backed by a
	// spares-only fixture: the device lookup fails before any connection I/O.
	executor := newCbk93RestoreExecutor(t, db, cbk93WriteRestoreFixture(t, cbk93RestoreFixtureSpec{}))
	backupSvc, taskSvc := newCbk93RestoreChain(db, executor)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: cbk93RestoreDeviceID, DeviceName: "r", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})
	// NOTE: no NetworkDevice row → runRestore's pre-restore CreateBackup fails
	// at the device lookup, before any device I/O or push.

	task, err := taskSvc.StartRestore(context.Background(), bk.ID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err)

	got := cbk93AwaitTerminal(t, taskSvc, task.ID)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status)
	assert.Contains(t, got.ErrorMessage, "恢复前自动备份失败")

	// no version-chain record beyond the seeded source backup (nothing pushed)
	var total int64
	require.NoError(t, db.Model(&models.ConfigBackup{}).Count(&total).Error)
	assert.Equal(t, int64(1), total, "only the seeded source backup may exist")
	_ = backupSvc
}

// -----------------------------------------------------------------------------
// FileTransport e2e (full chain)
// -----------------------------------------------------------------------------

const cbk93Config = "interface GE0/0/1\nshutdown\n"

// TestCbk93RestoreE2E_Happy — D-29/D-30 main assertion chain: backup → restore
// task → clean → push (fail-fast wrapper) → readback hash match → version-chain
// restore record, all over FileTransport with zero SSH.
func TestCbk93RestoreE2E_Happy(t *testing.T) {
	db := newCbk93RestoreDB(t)
	cbk93SeedDevice(t, db)
	fixture := cbk93WriteRestoreFixture(t, cbk93RestoreFixtureSpec{
		getConfigCycles: 2, // source-backup read + pre-restore backup read (readback is its own segment)
		configOutput:    cbk93Config,
	})
	executor := newCbk93RestoreExecutor(t, db, fixture)
	backupSvc, taskSvc := newCbk93RestoreChain(db, executor)

	ctx := context.Background()

	// ① source backup through the real device-read path
	created, err := backupSvc.CreateBackup(ctx, &BackupRequest{
		DeviceID:     cbk93RestoreDeviceID,
		DeviceName:   "cbk93-restore-switch",
		BackupType:   models.BackupTypeManual,
		ChangeReason: "e2e source",
		CreatedBy:    "tester",
	})
	require.NoError(t, err, "source backup")
	require.Equal(t, models.StorageTypeDatabase, created.StorageType)

	// ② async restore
	task, err := taskSvc.StartRestore(ctx, created.BackupID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err)

	// ③ terminal state + progress payload (D-07)
	got := cbk93AwaitTerminal(t, taskSvc, task.ID)
	assert.Equal(t, string(models.RestoreTaskStatusSuccess), got.Status)
	require.Empty(t, got.ErrorMessage)

	result := cbk93ParseResult(t, got.ResultJSON)
	assert.Equal(t, true, result["hashMatched"], "readback hash must match the backup hash (D-30)")
	assert.Equal(t, float64(3), result["totalLines"], "2 config lines + quit")
	assert.Equal(t, float64(3), result["sentLines"], "all lines pushed")

	// ④ version-chain restore record (D-09)
	var restoreRecords []models.ConfigBackup
	require.NoError(t, db.Where("device_id = ? AND change_reason LIKE ?", cbk93RestoreDeviceID, "恢复自版本%").
		Find(&restoreRecords).Error)
	require.Len(t, restoreRecords, 1, "exactly one version-chain restore record")
	assert.Equal(t, 3, restoreRecords[0].Version, "source v1 + pre-restore v2 + restore record v3")
	// D-30: the restored-device readback must hash to the SOURCE BACKUP's hash
	// (GetConfig result bytes — not the fixture constant, whose exact tail
	// newlines are owned by the scrapligo Response layer)
	var sourceBackup models.ConfigBackup
	require.NoError(t, db.Where("id = ?", created.BackupID).First(&sourceBackup).Error)
	assert.Equal(t, sourceBackup.ConfigHash, restoreRecords[0].ConfigHash)
	assert.Equal(t, sourceBackup.ConfigContent, restoreRecords[0].ConfigContent)

	// ⑤ statistics path stays healthy
	_, statErr := backupSvc.GetBackupStatistics(ctx)
	require.NoError(t, statErr)
}

// TestCbk93RestoreE2E_HashMismatchWarns — D-11: a readback that differs from
// the backup content must NOT fail the task (warn-only) and must record
// hashMatched=false in the result payload.
func TestCbk93RestoreE2E_HashMismatchWarns(t *testing.T) {
	db := newCbk93RestoreDB(t)
	cbk93SeedDevice(t, db)
	fixture := cbk93WriteRestoreFixture(t, cbk93RestoreFixtureSpec{
		getConfigCycles: 2,
		configOutput:    cbk93Config,
		readbackOutput:  cbk93Config + "vlan 999 description drifted\n",
	})
	executor := newCbk93RestoreExecutor(t, db, fixture)
	backupSvc, taskSvc := newCbk93RestoreChain(db, executor)

	created, err := backupSvc.CreateBackup(context.Background(), &BackupRequest{
		DeviceID:   cbk93RestoreDeviceID,
		DeviceName: "cbk93-restore-switch",
		BackupType: models.BackupTypeManual,
		CreatedBy:  "tester",
	})
	require.NoError(t, err)

	task, err := taskSvc.StartRestore(context.Background(), created.BackupID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err)

	got := cbk93AwaitTerminal(t, taskSvc, task.ID)
	assert.Equal(t, string(models.RestoreTaskStatusSuccess), got.Status, "mismatch warns, never fails (D-11)")

	result := cbk93ParseResult(t, got.ResultJSON)
	assert.Equal(t, false, result["hashMatched"])
	assert.NotEmpty(t, result["restoredHash"])
}

// TestCbk93RestoreE2E_SendInterrupted — D-28③ (device-rejection mechanism,
// 93-02 precedent: EOF truncation blocks forever instead of erroring): a
// mid-push rejection fails the task while preserving partial progress (D-12).
func TestCbk93RestoreE2E_SendInterrupted(t *testing.T) {
	db := newCbk93RestoreDB(t)
	cbk93SeedDevice(t, db)
	fixture := cbk93WriteRestoreFixture(t, cbk93RestoreFixtureSpec{
		getConfigCycles: 2, // source read + pre-restore backup read (no readback — push fails)
		configOutput:    cbk93Config,
		pushRejected:    true,
	})
	executor := newCbk93RestoreExecutor(t, db, fixture)
	backupSvc, taskSvc := newCbk93RestoreChain(db, executor)

	created, err := backupSvc.CreateBackup(context.Background(), &BackupRequest{
		DeviceID:   cbk93RestoreDeviceID,
		DeviceName: "cbk93-restore-switch",
		BackupType: models.BackupTypeManual,
		CreatedBy:  "tester",
	})
	require.NoError(t, err)

	task, err := taskSvc.StartRestore(context.Background(), created.BackupID, cbk93RestoreDeviceID, "tester")
	require.NoError(t, err)

	got := cbk93AwaitTerminal(t, taskSvc, task.ID)
	assert.Equal(t, string(models.RestoreTaskStatusFailed), got.Status)

	result := cbk93ParseResult(t, got.ResultJSON)
	assert.Equal(t, "shutdown", result["failedLine"], "failed line identified (D-12)")
	assert.Less(t, result["sentLines"], result["totalLines"], "partial progress recorded (D-12)")

	// no version-chain record on failure
	var restoreRecords int64
	require.NoError(t, db.Model(&models.ConfigBackup{}).
		Where("device_id = ? AND change_reason LIKE ?", cbk93RestoreDeviceID, "恢复自版本%").
		Count(&restoreRecords).Error)
	assert.Zero(t, restoreRecords)
}

// TestCbk93SortWhitelistNoStatus — D-33②: sys_config_backup has no status
// column, so orderByColumn="status" must be ignored by the whitelist (falling
// back to the default ordering) instead of producing ORDER BY status → SQL 500.
func TestCbk93SortWhitelistNoStatus(t *testing.T) {
	db := newCbk93RestoreDB(t)
	backupSvc, _ := newCbk93RestoreChain(db, nil)

	cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-s1", DeviceName: "s1", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "a", Version: 1,
	})
	cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-s2", DeviceName: "s2", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeDatabase, ConfigContent: "b", Version: 1,
	})

	assert.NotPanics(t, func() {
		_, _, err := backupSvc.GetBackupList(context.Background(), 1, 10, "", "status", nil)
		require.NoError(t, err, "status must be whitelist-ignored, not an ORDER BY SQL error")
	})
}

// seedRestoreTask93 inserts a restore-task row with an explicit status (local
// 93_02 helper — the network-package seedRestoreTask lives behind a different
// package boundary).
func seedRestoreTask93(t *testing.T, db *gorm.DB, deviceID, backupID string, status models.RestoreTaskStatus) *models.ConfigRestoreTask {
	t.Helper()
	task := &models.ConfigRestoreTask{
		DeviceID: deviceID,
		BackupID: backupID,
		Status:   string(status),
	}
	task.ID = fmt.Sprintf("task-cbk93-%s-%d", status, time.Now().UnixNano())
	require.NoError(t, db.Create(task).Error)
	return task
}
