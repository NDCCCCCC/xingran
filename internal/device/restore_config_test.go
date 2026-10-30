package device

// restore_config_test.go — Phase 93-02 RestoreConfig tests.
//
// Pure-function cases cover the D-03 clean rules and the D-06 vendor exit map
// (quit/exit, never "end" — V7 lesson). The e2e cases replay a real Huawei VRP
// byte stream (fixture copied from portwrite's huawei_shutdown_success,
// command-for-command identical: interface GE0/0/1 + shutdown + trailing quit)
// against the actual RestoreConfig → ExecuteCustom → wrapper.SendConfigs
// pipeline via the pool/scheduler/executor assembly (newPool7906 pattern).
//
// Pitfall discipline: FileTransport drivers are never Closed (close-time reads
// block on an exhausted fixture — 78-03 S-2); pool.Close is bounded by a
// watchdog in t.Cleanup; the process-wide applogger keeps ./logs/app.log open,
// which is harmless here because these tests never chdir (fixtures are testdata
// files, no relative-path writes).

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/logging"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// pure functions (D-03 / D-06)
// -----------------------------------------------------------------------------

func TestCleanConfigLines(t *testing.T) {
	in := "sysname R1\n\n  \n# full-line comment\n! full-line comment\n  # indented comment\nvlan 93 # inline hash stays\ninterface GE0/0/1\n description keeps ! here\n"
	got := cleanConfigLines(in)

	assert.Equal(t, []string{
		"sysname R1",
		"vlan 93 # inline hash stays", // mid-line # must survive (Huawei section separator)
		"interface GE0/0/1",
		"description keeps ! here", // mid-line ! must survive
	}, got)
}

func TestExitConfigCommand(t *testing.T) {
	assert.Equal(t, "quit", exitConfigCommand("huawei"))
	assert.Equal(t, "quit", exitConfigCommand("h3c"))
	assert.Equal(t, "exit", exitConfigCommand("ruijie"))
	assert.Equal(t, "exit", exitConfigCommand("maipu"))
	assert.Equal(t, "exit", exitConfigCommand(""))
	assert.Equal(t, "exit", exitConfigCommand("cisco"))
	for _, vendor := range []string{"huawei", "h3c", "ruijie", "maipu", ""} {
		assert.NotEqual(t, "end", exitConfigCommand(vendor), "V7: exit must only drop one level, never 'end'")
	}
}

// -----------------------------------------------------------------------------
// e2e harness
// -----------------------------------------------------------------------------

// newRestoreE2EExecutor assembles pool → scheduler → executor over a temp-file
// sqlite DB (AutoMigrate NetworkDevice + seed one huawei device with the given
// ID) and seeds a FileTransport connection replaying the given fixture file.
// Cleanup discipline mirrors newExecutor7906 (79_06:147): bounded pool.Close
// watchdog, scheduler.Stop, and NO driver.Close (78-03 S-2 close-time block).
func newRestoreE2EExecutor(t *testing.T, deviceID string, fixturePath string) (*DeviceExecutor, *gorm.DB) {
	t.Helper()

	pool, db := newPool7906(t)
	require.NoError(t, db.AutoMigrate(&models.NetworkDevice{}), "AutoMigrate NetworkDevice")

	dev := &models.NetworkDevice{
		DeviceName: "cbk93-restore-switch",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  "10.88.7.9",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = deviceID
	require.NoError(t, db.Create(dev).Error, "seed restore device")

	scheduler := NewDeviceTaskScheduler(pool, nil)
	t.Cleanup(scheduler.Stop)

	executor := NewDeviceExecutor(scheduler, &ExecutionConfig{
		MaxRetries:          0,
		RetryDelay:          time.Millisecond,
		Timeout:             10 * time.Second,
		EnablePanicRecovery: true,
	})

	p, err := platform.NewPlatform(
		"huawei_vrp",
		"cbk93-restore-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(fixturePath),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
		options.WithLogger(mustDebugLogger(t)),
	)
	require.NoError(t, err, "NewPlatform")
	drv, err := p.GetNetworkDriver()
	require.NoError(t, err, "GetNetworkDriver")
	require.NoError(t, drv.Open(), "driver.Open")

	conn := NewPooledConnectionForTesting(drv)
	SeedConnectionForTesting(pool, deviceID, conn)
	conn.ReleaseRef() // hand the factory ref back so pool.Close sees an idle conn

	return executor, db
}

func restoreFixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

// mustDebugLogger builds a scrapligo debug logger so fixture-consumption
// misalignment surfaces in -v output instead of a silent hang.
func mustDebugLogger(t *testing.T) *logging.Instance {
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

// -----------------------------------------------------------------------------
// e2e (FileTransport — real SendConfigs pipeline)
// -----------------------------------------------------------------------------

// TestRestoreConfig_E2E_Success replays a real Huawei VRP stream whose config
// commands match the fixture byte-for-byte (interface GE0/0/1 + shutdown; the
// leading system-view and trailing quit/return are scrapligo priv handling and
// the D-06 vendor exit command).
func TestRestoreConfig_E2E_Success(t *testing.T) {
	executor, _ := newRestoreE2EExecutor(t, "dev-cbk93-ok", restoreFixturePath(t, "restore_huawei_success.fixture"))

	result, err := executor.RestoreConfig(context.Background(), "dev-cbk93-ok", "interface GE0/0/1\nshutdown\n")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalLines, "2 config lines + trailing quit")
	assert.Equal(t, result.TotalLines, result.SentLines, "all lines sent on success")
	assert.Empty(t, result.FailedLine)
	assert.Greater(t, result.Duration, time.Duration(0))
}

// TestRestoreConfig_E2E_FailFast replays a stream where the second config
// command (shutdown) is rejected by the device ("% Error" marker → scrapligo
// failed_when_contains). StopOnFailed must discontinue the remaining lines
// (quit never sent), and RestoreConfig must report the partial progress with
// the failed line identified (D-12).
func TestRestoreConfig_E2E_FailFast(t *testing.T) {
	executor, _ := newRestoreE2EExecutor(t, "dev-cbk93-ff", restoreFixturePath(t, "restore_huawei_rejected.fixture"))

	result, err := executor.RestoreConfig(context.Background(), "dev-cbk93-ff", "interface GE0/0/1\nshutdown\n")
	require.Error(t, err, "device-rejected line must fail the restore")
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalLines, "2 config lines + trailing quit")
	assert.Less(t, result.SentLines, result.TotalLines, "partial progress recorded (D-12)")
	assert.Equal(t, "shutdown", result.FailedLine, "failed line identified (D-12)")
	assert.NotEqual(t, "quit", result.FailedLine, "StopOnFailed must not send trailing quit")
}
