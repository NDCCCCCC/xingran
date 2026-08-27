// Phase 79-06 (TAIL-01) — config_backup_service.go coverage tests.
//
// Tier-1 scope (Task 3): pure helpers (calculateHash / generateDiff /
// getDefaultThreshold / getBackupDir), the sqlite+file CRUD chain
// (GetBackupList / GetBackupByID / GetBackupContent / DiffBackups /
// RestoreBackup / GetBackupStatistics / DeleteBackup) and the
// loadConfigFromDB / ReloadConfig concurrency-config path.
//
// Tier-2 scope (D-79-02, Task 1 helper): CreateBackup / BatchBackupDevices /
// AutoBackupSingleDevice / AutoBackupAllDevices run for real against a
// *device.DeviceExecutor assembled from the PUBLIC constructors
// (NewDeviceConnectionPool → NewDeviceTaskScheduler → NewDeviceExecutor) with
// a FileTransport connection seeded via device.SeedConnectionForTesting —
// zero SSH, zero production change.
//
// Working-directory discipline: getBackupDir hard-codes the RELATIVE
// "data/config-backups/..." root (config_backup_service.go:87-91) with no
// injection point, so tests t.Chdir into t.TempDir() — the same isolation the
// 78-01 StoragePath discipline and 79-05 ImportOUIData used. No file ever
// lands in the repository working tree.
package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// newCbk7906 assembles a ConfigBackupService over a fresh sqlite DB and a
// process working directory moved into t.TempDir() (returns that dir so tests
// can assert on written backup files). executor is nil — Tier-1 only.
func newCbk7906(t *testing.T) (*ConfigBackupService, *gorm.DB, string) {
	t.Helper()
	tmp := cbk7906Chdir(t)
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})
	return NewConfigBackupService(db, nil), db, tmp
}

// cbk7906Chdir moves the process into a process-temp working directory (NOT
// t.TempDir) and returns it. Two reasons:
//   - getBackupDir's relative "data/config-backups" root must not land in the
//     repository working tree;
//   - the process-wide applogger lazily opens ./logs/app.log under the CWD and
//     keeps it open, which would make t.TempDir's RemoveAll fail the test on
//     Windows ("file in use"). Best-effort cleanup accepts that leak instead.
func cbk7906Chdir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cbk7906-cwd")
	require.NoError(t, err, "MkdirTemp")
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Chdir(dir)
	return dir
}

// newDriver7906FromFixture opens a scrapligo FileTransport driver over an
// existing fixture path (see writeFixture7906). The driver is never Closed —
// close-time reads on an exhausted fixture block forever (78-03 S-2).
func newDriver7906FromFixture(t *testing.T, fixturePath string) *network.Driver {
	t.Helper()
	p, err := platform.NewPlatform(
		"huawei_vrp",
		"dummy-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(fixturePath),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
	)
	require.NoError(t, err, "NewPlatform")
	d, err := p.GetNetworkDriver()
	require.NoError(t, err, "GetNetworkDriver")
	require.NoError(t, d.Open(), "driver.Open")
	return d
}

// cbk7906FixtureCycles writes a FileTransport fixture with n identical
// "show running-config" reply cycles (same output each time → identical hash
// across AutoBackup calls) plus spare prompts for IsReady/close reads.
func cbk7906Fixture(t *testing.T, cycles int) string {
	t.Helper()
	return writeFixture7906(t, cycles, "show running-config", "hostname cbk7906-fixture")
}

// writeFixture7906 is the shared 79-06 fixture builder: open banner prompt →
// screen-length echo → post-open prompt → n × (pre-prompt, echo, output,
// terminating prompt) → 8 spare prompts. Written under the process temp dir
// (NOT t.TempDir — the FileTransport keeps the file open, which breaks
// t.TempDir's RemoveAll on Windows).
func writeFixture7906(t *testing.T, cycles int, cmd, output string) string {
	t.Helper()
	var b []byte
	add := func(s string) { b = append(b, s...) }
	add("<dummy-host>\n")
	add("screen-length 0 temporary\n")
	add("<dummy-host>\n")
	for i := 0; i < cycles; i++ {
		add("<dummy-host>\n")
		add(cmd + "\n")
		add(output + "\n")
		add("<dummy-host>\n")
	}
	for i := 0; i < 8; i++ {
		add("<dummy-host>\n")
	}
	dir, err := os.MkdirTemp("", "svc7906-fixture")
	require.NoError(t, err, "MkdirTemp")
	t.Cleanup(func() { _ = os.RemoveAll(dir) }) // best effort on Windows file locks
	path := filepath.Join(dir, "svc7906.fixture")
	require.NoError(t, os.WriteFile(path, b, 0o644), "write fixture")
	return path
}

// newExecutor7906 assembles a fully wired *device.DeviceExecutor over db with
// a FileTransport connection seeded for deviceID. Everything goroutine-backed
// (pool cleanup, scheduler workers) is shut down via t.Cleanup, with the pool
// Close bounded by a watchdog because FileTransport close-time reads can block
// on an exhausted fixture (78-03 S-2 pitfall).
func newExecutor7906(t *testing.T, db *gorm.DB, deviceID string, fixtureCycles int) *device.DeviceExecutor {
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

	drv := newDriver7906FromFixture(t, cbk7906Fixture(t, fixtureCycles))
	conn := device.NewPooledConnectionForTesting(drv)
	device.SeedConnectionForTesting(pool, deviceID, conn)
	conn.ReleaseRef() // hand the factory ref back so pool.Close sees an idle conn

	return executor
}

// cbk7906SeedDevice inserts a NetworkDevice row with an explicit ID (the
// executor resolves the vendor-dependent config command from this row).
func cbk7906SeedDevice(t *testing.T, db *gorm.DB, id, name string) *models.NetworkDevice {
	t.Helper()
	dev := &models.NetworkDevice{
		DeviceName: name,
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.DeviceVendor("generic"), // unknown vendor → "show running-config"
		IPAddress:  "10.88.7.1",
		Status:     models.DeviceStatusOnline,
	}
	dev.ID = id
	require.NoError(t, db.Create(dev).Error, "seed device %s", name)
	return dev
}

// cbk7906SeedBackup inserts a ConfigBackup row with explicit timestamps so
// list ordering is deterministic.
func cbk7906SeedBackup(t *testing.T, db *gorm.DB, bk *models.ConfigBackup) *models.ConfigBackup {
	t.Helper()
	require.NoError(t, db.Create(bk).Error, "seed backup %+v", bk)
	return bk
}

// -----------------------------------------------------------------------------
// pure helpers
// -----------------------------------------------------------------------------

// TestCbk7906_CalculateHash — md5 hex digest: deterministic, content-sensitive,
// empty-content stable, 32 hex chars.
func TestCbk7906_CalculateHash(t *testing.T) {
	require.Len(t, calculateHash(""), 32, "md5 hex digest is 32 chars")
	require.Len(t, calculateHash("config line\n"), 32)
	assert.Equal(t, calculateHash("same"), calculateHash("same"), "same content → same hash")
	assert.NotEqual(t, calculateHash("config A"), calculateHash("config B"), "content change must move the hash")
	assert.NotEqual(t, calculateHash("config A"), calculateHash("config A\n"), "trailing newline is content")
}

// TestCbk7906_GenerateDiff — line-wise diff: unchanged lines carry the two
// space prefix, removals get "- ", additions get "+ " (blank side dropped).
func TestCbk7906_GenerateDiff(t *testing.T) {
	t.Run("identical_content_keeps_prefix", func(t *testing.T) {
		// Locked behavior: identical input is NOT an empty diff — every line is
		// echoed with the two-space unchanged marker.
		assert.Equal(t, "  a\n  b\n", generateDiff("a\nb", "a\nb"))
	})

	t.Run("addition_and_removal", func(t *testing.T) {
		got := generateDiff("keep\nold\n", "keep\nnew\n")
		assert.Contains(t, got, "  keep\n")
		assert.Contains(t, got, "- old\n")
		assert.Contains(t, got, "+ new\n")
	})

	t.Run("trailing_lines_added", func(t *testing.T) {
		got := generateDiff("a\n", "a\nb\nc\n")
		assert.Contains(t, got, "+ b\n")
		assert.Contains(t, got, "+ c\n")
		assert.NotContains(t, got, "- \n", "missing side contributes no marker line")
	})

	t.Run("trailing_lines_removed", func(t *testing.T) {
		got := generateDiff("a\nb\nc\n", "a\n")
		assert.Contains(t, got, "- b\n")
		assert.Contains(t, got, "- c\n")
	})

	t.Run("both_empty_input", func(t *testing.T) {
		assert.Equal(t, "  \n", generateDiff("", ""), "single blank line echo (locked)")
	})

	t.Run("special_characters_survive", func(t *testing.T) {
		got := generateDiff("配置 中文 §1\n", "配置 中文 §2\n")
		assert.Contains(t, got, "- 配置 中文 §1\n")
		assert.Contains(t, got, "+ 配置 中文 §2\n")
	})
}

// TestCbk7906_GetDefaultThreshold — sys_config override vs the 100KB fallback,
// including the non-numeric-value branch.
func TestCbk7906_GetDefaultThreshold(t *testing.T) {
	ctx := context.Background()

	t.Run("missing_config_falls_back", func(t *testing.T) {
		svc, _, _ := newCbk7906(t)
		assert.Equal(t, defaultBackupThresholdKB, svc.getDefaultThreshold(ctx))
	})

	t.Run("numeric_override", func(t *testing.T) {
		svc, db, _ := newCbk7906(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "备份阈值",
			ConfigKey:   "network.config.backup.threshold",
			ConfigValue: "250",
		}).Error)
		assert.Equal(t, 250, svc.getDefaultThreshold(ctx))
	})

	t.Run("non_numeric_override_falls_back", func(t *testing.T) {
		svc, db, _ := newCbk7906(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "备份阈值",
			ConfigKey:   "network.config.backup.threshold",
			ConfigValue: "not-a-number",
		}).Error)
		assert.Equal(t, defaultBackupThresholdKB, svc.getDefaultThreshold(ctx))
	})
}

// TestCbk7906_BackupDir — the relative backup root is data/config-backups with
// YYYY/MM sharding; the t.Chdir isolation keeps it inside t.TempDir().
func TestCbk7906_BackupDir(t *testing.T) {
	svc, _, tmp := newCbk7906(t)
	got := svc.getBackupDir("device-1")
	want := filepath.Join("data", "config-backups", "device-1", time.Now().Format("2006"), time.Now().Format("01"))
	assert.Equal(t, want, got)
	assert.True(t, filepath.IsAbs(tmp), "cwd moved into the temp dir")
}

// -----------------------------------------------------------------------------
// sqlite/文件 CRUD 链
// -----------------------------------------------------------------------------

// TestCbk7906_BackupListAndGet — list pagination + device filter + sort
// whitelist + the IPAddress join, and GetBackupByID hit/miss.
func TestCbk7906_BackupListAndGet(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newCbk7906(t)
	dev := cbk7906SeedDevice(t, db, "dev-cbk-list", "cbk-list-switch")

	base := time.Now().Add(-time.Hour)
	for i, bk := range []*models.ConfigBackup{
		{DeviceID: dev.ID, DeviceName: dev.DeviceName, BackupType: models.BackupTypeAuto, StorageType: models.StorageTypeDatabase, ConfigContent: "v1", Version: 1, BackupSize: 2, CreatedBy: "t"},
		{DeviceID: dev.ID, DeviceName: dev.DeviceName, BackupType: models.BackupTypeManual, StorageType: models.StorageTypeFile, FilePath: "data/config-backups/x.conf", Version: 2, BackupSize: 5, CreatedBy: "t"},
		{DeviceID: "dev-other", DeviceName: "other", BackupType: models.BackupTypeAuto, StorageType: models.StorageTypeDatabase, ConfigContent: "o1", Version: 1, BackupSize: 2, CreatedBy: "t"},
	} {
		bk.ID = ""
		bk.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		bk.UpdatedAt = bk.CreatedAt
		cbk7906SeedBackup(t, db, bk)
	}

	t.Run("list_all_with_ip_join", func(t *testing.T) {
		items, total, err := svc.GetBackupList(ctx, 1, 10, "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, items, 3)
		byDevice := map[string]string{}
		for _, it := range items {
			byDevice[it.DeviceID] = it.IPAddress
		}
		assert.Equal(t, dev.IPAddress, byDevice[dev.ID], "existing device joins its IP")
		assert.Equal(t, "", byDevice["dev-other"], "unknown device yields an empty IP (locked)")
	})

	t.Run("filter_by_device", func(t *testing.T) {
		items, total, err := svc.GetBackupList(ctx, 1, 10, dev.ID, "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, items, 2)
		for _, it := range items {
			assert.Equal(t, dev.ID, it.DeviceID)
		}
	})

	t.Run("sort_whitelist_version_desc", func(t *testing.T) {
		items, _, err := svc.GetBackupList(ctx, 1, 10, dev.ID, "version", boolPtr7906(false))
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, 2, items[0].Version, "version descending")
	})

	t.Run("pagination_second_page", func(t *testing.T) {
		_, total, err := svc.GetBackupList(ctx, 2, 2, "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
	})

	t.Run("get_by_id_hit_and_miss", func(t *testing.T) {
		items, _, err := svc.GetBackupList(ctx, 1, 10, dev.ID, "", nil)
		require.NoError(t, err)
		got, err := svc.GetBackupByID(ctx, items[0].ID)
		require.NoError(t, err)
		assert.Equal(t, items[0].ID, got.ID)

		missing, err := svc.GetBackupByID(ctx, "no-such-backup")
		require.Error(t, err)
		assert.Nil(t, missing)
		assert.Contains(t, err.Error(), "备份记录不存在")
	})
}

// TestCbk7906_DiffBackups — DB-stored and file-stored content both feed the
// diff; the labels carry device name + version; missing content errors.
func TestCbk7906_DiffBackups(t *testing.T) {
	ctx := context.Background()
	svc, db, tmp := newCbk7906(t)

	filePath := filepath.Join(tmp, "data", "config-backups", "manual.conf")
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
	require.NoError(t, os.WriteFile(filePath, []byte("hostname new\n"), 0o644))

	bkDB := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-d1", DeviceName: "cbk-diff", BackupType: models.BackupTypeAuto,
		StorageType: models.StorageTypeDatabase, ConfigContent: "hostname old\n", Version: 1,
	})
	bkFile := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-d1", DeviceName: "cbk-diff", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeFile, FilePath: filePath, Version: 2,
	})

	t.Run("db_vs_file_diff", func(t *testing.T) {
		label1, label2, diff, err := svc.DiffBackups(ctx, bkDB.ID, bkFile.ID)
		require.NoError(t, err)
		assert.Equal(t, "cbk-diff (版本1)", label1)
		assert.Equal(t, "cbk-diff (版本2)", label2)
		assert.Contains(t, diff, "- hostname old\n")
		assert.Contains(t, diff, "+ hostname new\n")
	})

	t.Run("missing_source_backup", func(t *testing.T) {
		_, _, _, err := svc.DiffBackups(ctx, "no-such", bkFile.ID)
		require.Error(t, err)
	})

	t.Run("missing_file_on_disk", func(t *testing.T) {
		bkGhost := cbk7906SeedBackup(t, db, &models.ConfigBackup{
			DeviceID: "dev-d2", DeviceName: "cbk-ghost", BackupType: models.BackupTypeManual,
			StorageType: models.StorageTypeFile, FilePath: filepath.Join(tmp, "gone.conf"), Version: 3,
		})
		_, _, _, err := svc.DiffBackups(ctx, bkDB.ID, bkGhost.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "读取备份文件失败")
	})
}

// TestCbk7906_GetBackupContent — database-stored content is returned as-is;
// file-stored content is read from disk; a missing record errors.
func TestCbk7906_GetBackupContent(t *testing.T) {
	ctx := context.Background()
	svc, db, tmp := newCbk7906(t)

	bkDB := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-c1", DeviceName: "c", BackupType: models.BackupTypeAuto,
		StorageType: models.StorageTypeDatabase, ConfigContent: "from-db\n", Version: 1,
	})
	content, err := svc.GetBackupContent(ctx, bkDB.ID)
	require.NoError(t, err)
	assert.Equal(t, "from-db\n", content)

	filePath := filepath.Join(tmp, "fb.conf")
	require.NoError(t, os.WriteFile(filePath, []byte("from-file\n"), 0o644))
	bkFile := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-c2", DeviceName: "c2", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeFile, FilePath: filePath, Version: 1,
	})
	content, err = svc.GetBackupContent(ctx, bkFile.ID)
	require.NoError(t, err)
	assert.Equal(t, "from-file\n", content)

	_, err = svc.GetBackupContent(ctx, "no-such")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "备份记录不存在")
}

// TestCbk7906_RestoreBackup — locked stub: the restore endpoint validates the
// backup content then always fails with 配置恢复功能待实现 (TODO in source).
func TestCbk7906_RestoreBackup(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newCbk7906(t)

	bk := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-r1", DeviceName: "r", BackupType: models.BackupTypeAuto,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})

	err := svc.RestoreBackup(ctx, bk.ID, "dev-r1")
	require.Error(t, err, "restore is a stub — never succeeds (locked)")
	assert.Contains(t, err.Error(), "配置恢复功能待实现")

	err = svc.RestoreBackup(ctx, "no-such", "dev-r1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "备份记录不存在")
}

// TestCbk7906_BackupStatistics — counts per type/storage plus size sum and the
// distinct device count, all hand-computed.
func TestCbk7906_BackupStatistics(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newCbk7906(t)

	rows := []*models.ConfigBackup{
		{DeviceID: "dev-s1", BackupType: models.BackupTypeAuto, StorageType: models.StorageTypeDatabase, BackupSize: 100},
		{DeviceID: "dev-s1", BackupType: models.BackupTypeManual, StorageType: models.StorageTypeFile, FilePath: "x.conf", BackupSize: 250},
		{DeviceID: "dev-s2", BackupType: models.BackupTypeAuto, StorageType: models.StorageTypeDatabase, BackupSize: 52},
	}
	for _, bk := range rows {
		cbk7906SeedBackup(t, db, bk)
	}

	stats, err := svc.GetBackupStatistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats["totalBackups"])
	assert.Equal(t, int64(2), stats["autoBackups"])
	assert.Equal(t, int64(1), stats["manualBackups"])
	assert.Equal(t, int64(2), stats["dbStorageCount"])
	assert.Equal(t, int64(1), stats["fileStorageCount"])
	assert.Equal(t, int64(402), stats["totalSize"])
	assert.Equal(t, int64(0), stats["totalSizeMB"], "402 bytes → 0 MB")
	assert.Equal(t, int64(2), stats["uniqueDevices"])
}

// TestCbk7906_DeleteBackup — file-stored rows delete their file; soft-delete
// removes the row from queries; batch continues past missing IDs.
func TestCbk7906_DeleteBackup(t *testing.T) {
	ctx := context.Background()
	svc, db, tmp := newCbk7906(t)

	filePath := filepath.Join(tmp, "del.conf")
	require.NoError(t, os.WriteFile(filePath, []byte("cfg\n"), 0o644))
	bkFile := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-d", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeFile, FilePath: filePath, Version: 1,
	})
	bkDB := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-d", BackupType: models.BackupTypeAuto,
		StorageType: models.StorageTypeDatabase, ConfigContent: "cfg\n", Version: 1,
	})

	require.NoError(t, svc.DeleteBackup(ctx, bkFile.ID))
	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err), "file-stored backup must delete its file")

	require.NoError(t, svc.DeleteBackup(ctx, bkDB.ID))
	items, _, err := svc.GetBackupList(ctx, 1, 10, "", "", nil)
	require.NoError(t, err)
	assert.Empty(t, items, "both rows soft-deleted")

	require.NoError(t, svc.BatchDeleteBackups(ctx, []string{"no-such-a", "no-such-b"}),
		"batch delete continues past missing IDs")
	err = svc.DeleteBackup(ctx, "no-such")
	require.Error(t, err)
}

// TestCbk7906_LoadAndReloadConfig — the concurrency config is read from
// sys_config at construction and re-read by ReloadConfig; invalid values keep
// the default.
func TestCbk7906_LoadAndReloadConfig(t *testing.T) {
	t.Run("valid_override", func(t *testing.T) {
		svc, db, _ := newCbk7906(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "备份并发",
			ConfigKey:   backupConcurrentConfigKey,
			ConfigValue: "3",
		}).Error)
		svc.ReloadConfig()
		assert.Equal(t, 3, svc.maxConcurrent)
	})

	t.Run("invalid_values_keep_default", func(t *testing.T) {
		svc, db, _ := newCbk7906(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "备份并发",
			ConfigKey:   backupConcurrentConfigKey,
			ConfigValue: "not-a-number",
		}).Error)
		svc.ReloadConfig()
		assert.Equal(t, 5, svc.maxConcurrent, "parse failure keeps the default 5")
	})

	t.Run("zero_and_negative_rejected", func(t *testing.T) {
		svc, db, _ := newCbk7906(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "备份并发",
			ConfigKey:   backupConcurrentConfigKey,
			ConfigValue: "0",
		}).Error)
		svc.ReloadConfig()
		assert.Equal(t, 5, svc.maxConcurrent, "0 fails the >0 guard")
	})
}

// -----------------------------------------------------------------------------
// Tier-2 (D-79-02 种子池) — CreateBackup / AutoBackup / BatchBackup
// -----------------------------------------------------------------------------

// TestCbk7906_CreateBackup_SeededExecutor drives the full executor path:
// GetConfig over the seeded FileTransport connection → hash → threshold split
// → version numbering → row persisted. Small config (< threshold) is stored in
// the database.
func TestCbk7906_CreateBackup_SeededExecutor(t *testing.T) {
	ctx := context.Background()
	t.Chdir(t.TempDir()) // getBackupDir writes relative data/config-backups
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk-exec"
	cbk7906SeedDevice(t, db, deviceID, "cbk-exec-switch")

	executor := newExecutor7906(t, db, deviceID, 3)
	svc := NewConfigBackupService(db, executor)

	result, err := svc.CreateBackup(ctx, &BackupRequest{
		DeviceID:     deviceID,
		DeviceName:   "cbk-exec-switch",
		BackupType:   models.BackupTypeManual,
		ChangeReason: "79-06 测试",
		CreatedBy:    "cbk7906",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, deviceID, result.DeviceID)
	assert.Equal(t, models.StorageTypeDatabase, result.StorageType, "fixture output < 100KB → database storage")
	assert.False(t, result.IsCompressed)
	assert.Equal(t, 1, result.Version, "first backup is version 1")
	assert.Equal(t, len("hostname cbk7906-fixture"), result.ConfigSize)

	row, err := svc.GetBackupByID(ctx, result.BackupID)
	require.NoError(t, err)
	assert.Equal(t, calculateHash("hostname cbk7906-fixture"), row.ConfigHash)
	assert.Equal(t, "hostname cbk7906-fixture", row.ConfigContent)

	// Second backup of the same content bumps the version.
	second, err := svc.CreateBackup(ctx, &BackupRequest{
		DeviceID: deviceID, DeviceName: "cbk-exec-switch",
		BackupType: models.BackupTypeManual, CreatedBy: "cbk7906",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, second.Version, "version = latest + 1")
}

// TestCbk7906_CreateBackup_LargeConfigGoesToFile crosses the threshold so the
// filesystem branch (mkdir + write + FilePath) executes inside the temp cwd.
func TestCbk7906_CreateBackup_LargeConfigGoesToFile(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	t.Chdir(tmp)
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk-big"
	cbk7906SeedDevice(t, db, deviceID, "cbk-big-switch")

	// Threshold 0 → thresholdBytes 0; the strict `configSize < thresholdBytes`
	// comparison then routes every non-empty config into the filesystem branch.
	require.NoError(t, db.Create(&models.Config{
		ConfigName: "备份阈值", ConfigKey: "network.config.backup.threshold", ConfigValue: "0",
	}).Error)

	executor := newExecutor7906(t, db, deviceID, 3)
	svc := NewConfigBackupService(db, executor)

	result, err := svc.CreateBackup(ctx, &BackupRequest{
		DeviceID: deviceID, DeviceName: "cbk-big-switch",
		BackupType: models.BackupTypeAuto, CreatedBy: "cbk7906",
	})
	require.NoError(t, err)
	assert.Equal(t, models.StorageTypeFile, result.StorageType, "threshold 0 → file storage branch")
	assert.FileExists(t, result.FilePath, "backup file written under the temp cwd")

	content, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	assert.Equal(t, "hostname cbk7906-fixture", string(content))
}

// TestCbk7906_AutoBackupSmartSkip — AutoBackupSingleDevice creates the first
// backup, then reports skipped=true for identical config (hash match), and
// AutoBackupAllDevices walks the device table.
func TestCbk7906_AutoBackupSmartSkip(t *testing.T) {
	ctx := context.Background()
	t.Chdir(t.TempDir())
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk-auto"
	dev := cbk7906SeedDevice(t, db, deviceID, "cbk-auto-switch")

	executor := newExecutor7906(t, db, deviceID, 4) // 4 identical reply cycles
	svc := NewConfigBackupService(db, executor)

	skipped, err := svc.AutoBackupSingleDevice(ctx, dev)
	require.NoError(t, err)
	assert.False(t, skipped, "no previous backup → creates one")

	skipped, err = svc.AutoBackupSingleDevice(ctx, dev)
	require.NoError(t, err)
	assert.True(t, skipped, "identical fixture output → hash match → skipped")

	items, total, err := svc.GetBackupList(ctx, 1, 10, deviceID, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "skipped backup must not create a second row")
	require.Len(t, items, 1)
	assert.Equal(t, models.BackupTypeAuto, items[0].BackupType)

	// AutoBackupAllDevices: only the seeded device exists → 1 processed, no error.
	require.NoError(t, svc.AutoBackupAllDevices(ctx))
}

// TestCbk7906_BatchBackupDevices — concurrent batch with one live device (over
// the seeded connection) and one nonexistent device (skipped inside the
// worker goroutine).
func TestCbk7906_BatchBackupDevices(t *testing.T) {
	ctx := context.Background()
	t.Chdir(t.TempDir())
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk-batch"
	cbk7906SeedDevice(t, db, deviceID, "cbk-batch-switch")

	executor := newExecutor7906(t, db, deviceID, 3)
	svc := NewConfigBackupService(db, executor)

	results, err := svc.BatchBackupDevices(ctx, []string{deviceID, "dev-no-such"}, models.BackupTypeManual, "cbk7906")
	require.NoError(t, err)
	require.Len(t, results, 1, "only the existing device yields a result")
	assert.Equal(t, deviceID, results[0].DeviceID)
	assert.Equal(t, 1, results[0].Version)
}

// TestCbk7906_CreateBackup_MissingDevice — the device lookup fails before the
// executor is touched, so no seeded connection is needed for this branch.
func TestCbk7906_CreateBackup_MissingDevice(t *testing.T) {
	ctx := context.Background()
	t.Chdir(t.TempDir())
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})
	executor := newExecutor7906(t, db, "unused-dev", 1)
	svc := NewConfigBackupService(db, executor)

	result, err := svc.CreateBackup(ctx, &BackupRequest{DeviceID: "dev-no-such"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "设备不存在")
}
