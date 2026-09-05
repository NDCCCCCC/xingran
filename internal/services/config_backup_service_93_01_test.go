// Phase 93-01 — compress/decompress regression tests (D-23..D-26).
//
// Covers: gzip helper roundtrip + corrupt-input error paths (BACKUP-CLOSED-02),
// the CreateBackup large-file .conf.gz branch (D-23/D-24), the small-config
// DB-boundary semantics, the auto-path unification (D-25) and the
// GetBackupContent double-check decompression.
//
// Helpers newCbk7906 / cbk7906Chdir / newDriver7906FromFixture /
// writeFixture7906 / newDB7906 / cbk7906SeedDevice / cbk7906SeedBackup are
// reused from the 79_06 file (same package). Working-directory discipline
// follows 79_06's cbk7906Chdir (Pitfall-8: process temp dir, NOT t.TempDir —
// the applogger keeps ./logs/app.log open, which breaks t.TempDir RemoveAll on
// Windows).
package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
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

// cbk93LargeOutput builds a >1KB device-config output so backups cross a
// threshold=1 (KB) sys_config and land in the filesystem branch.
func cbk93LargeOutput() string {
	return strings.Repeat("vlan 93 description cbk93-padding-line-for-threshold-crossing\n", 20)
}

// newExecutor93 assembles a *device.DeviceExecutor whose seeded FileTransport
// replies with the given output string (93-01 needs >1KB outputs to cross the
// compress threshold; newExecutor7906's fixture output is fixed and small).
// Shutdown discipline mirrors newExecutor7906 (pool.Close watchdog vs the
// FileTransport close-time blocking pitfall).
func newExecutor93(t *testing.T, db *gorm.DB, deviceID, output string) *device.DeviceExecutor {
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

	drv := newDriver7906FromFixture(t, writeFixture7906(t, 2, "show running-config", output))
	conn := device.NewPooledConnectionForTesting(drv)
	device.SeedConnectionForTesting(pool, deviceID, conn)
	conn.ReleaseRef() // hand the factory ref back so pool.Close sees an idle conn

	return executor
}

// -----------------------------------------------------------------------------
// gzip helper（BACKUP-CLOSED-01/02 内核）
// -----------------------------------------------------------------------------

// TestCbk93GzipRoundtrip — compress → decompress restores the original text;
// the compressed artifact is NOT the plaintext and actually shrinks; the gzip
// magic header (0x1f 0x8b) is present.
func TestCbk93GzipRoundtrip(t *testing.T) {
	original := strings.Repeat("hostname cbk93-roundtrip\n", 100)

	compressed, err := gzipCompress(original)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	assert.False(t, bytes.Equal(compressed, []byte(original)), "压缩产物不得是明文")
	assert.Greater(t, len(original), len(compressed), "重复行配置必须真实收缩")
	assert.GreaterOrEqual(t, len(compressed), 2)

	// gzip magic header: 0x1f 0x8b (RFC 1952)
	assert.Equal(t, byte(0x1f), compressed[0], "gzip magic first byte")
	assert.Equal(t, byte(0x8b), compressed[1], "gzip magic second byte")

	roundtrip, err := gzipDecompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, original, roundtrip, "roundtrip 必须还原原文")
}

// TestCbk93DecompressCorruptHeader — random bytes without the 0x1f magic fail
// inside gzip.NewReader with the 备份文件损坏 wrapper; no panic escapes (D-28②).
func TestCbk93DecompressCorruptHeader(t *testing.T) {
	garbage := make([]byte, 32)
	_, err := rand.Read(garbage)
	require.NoError(t, err)
	garbage[0] = 'X' // force a non-gzip magic deterministically

	got, err := gzipDecompress(garbage)
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "备份文件损坏")
}

// TestCbk93DecompressTruncated — a valid stream cut in half passes NewReader
// (header intact) but fails ReadAll with the 备份文件解压失败 wrapper.
func TestCbk93DecompressTruncated(t *testing.T) {
	original := strings.Repeat("interface GE0/0/1\n shutdown\n", 50)
	compressed, err := gzipCompress(original)
	require.NoError(t, err)
	require.Greater(t, len(compressed), 8, "footer (CRC+ISIZE) must exist before truncation")

	truncated := compressed[:len(compressed)/2]

	got, err := gzipDecompress(truncated)
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "备份文件解压失败")
}

// -----------------------------------------------------------------------------
// CreateBackup 压缩分支（D-23/D-24/D-26 语义边界）
// -----------------------------------------------------------------------------

// TestCbk93CreateBackupCompressedLargeFile — threshold=1KB + >1KB config +
// CompressLarge=true: a .conf.gz file lands on disk whose bytes decompress back
// to the original config; Compressed=true and BackupSize keeps the ORIGINAL
// byte size (D-24 — not the compressed size).
func TestCbk93CreateBackupCompressedLargeFile(t *testing.T) {
	ctx := context.Background()
	cbk7906Chdir(t) // getBackupDir writes relative data/config-backups
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk93-gz"
	cbk7906SeedDevice(t, db, deviceID, "cbk93-gz-switch")

	// threshold=1 → thresholdBytes=1024；>1KB config → 文件分支
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  "备份阈值",
		ConfigKey:   "network.config.backup.threshold",
		ConfigValue: "1",
	}).Error)

	executor := newExecutor93(t, db, deviceID, cbk93LargeOutput())
	svc := NewConfigBackupService(db, executor)

	// probe the exact config string the service will back up (fixture cycle 1)
	config, err := svc.executor.GetConfig(ctx, deviceID)
	require.NoError(t, err)
	require.Greater(t, len(config), 1024, "fixture output must cross the 1KB threshold")

	result, err := svc.CreateBackup(ctx, &BackupRequest{
		DeviceID:      deviceID,
		DeviceName:    "cbk93-gz-switch",
		BackupType:    models.BackupTypeManual,
		ChangeReason:  "93-01 压缩回归",
		CreatedBy:     "cbk93",
		CompressLarge: true,
	})
	require.NoError(t, err)

	assert.True(t, result.IsCompressed, "D-23: CompressLarge=true → Compressed 标志")
	assert.True(t, strings.HasSuffix(result.FilePath, ".gz"), "D-23: .conf.gz 命名")
	assert.FileExists(t, result.FilePath)
	assert.Equal(t, len(config), result.ConfigSize, "D-24: BackupSize 保持原始字节口径")

	raw, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	assert.False(t, bytes.Equal(raw, []byte(config)), "落盘内容必须是压缩字节而非明文")
	assert.Equal(t, byte(0x1f), raw[0], "落盘文件带 gzip magic")

	roundtrip, err := gzipDecompress(raw)
	require.NoError(t, err)
	assert.Equal(t, config, roundtrip, "gzip roundtrip 还原原始配置")

	row, err := svc.GetBackupByID(ctx, result.BackupID)
	require.NoError(t, err)
	assert.True(t, row.Compressed)
	assert.Equal(t, len(config), row.BackupSize, "D-24: DB 记录保持原始口径")
}

// TestCbk93CreateBackupSmallStaysDatabase — small config + CompressLarge=true
// stays in the database UNcompressed (compression is a file-branch concern
// only — the D-26 semantic boundary).
func TestCbk93CreateBackupSmallStaysDatabase(t *testing.T) {
	ctx := context.Background()
	cbk7906Chdir(t)
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	const deviceID = "dev-cbk93-small"
	cbk7906SeedDevice(t, db, deviceID, "cbk93-small-switch")

	executor := newExecutor93(t, db, deviceID, "hostname cbk93-small")
	svc := NewConfigBackupService(db, executor)

	result, err := svc.CreateBackup(ctx, &BackupRequest{
		DeviceID:      deviceID,
		DeviceName:    "cbk93-small-switch",
		BackupType:    models.BackupTypeManual,
		CreatedBy:     "cbk93",
		CompressLarge: true,
	})
	require.NoError(t, err)

	assert.Equal(t, models.StorageTypeDatabase, result.StorageType, "小配置不入文件分支")
	assert.False(t, result.IsCompressed, "DB 存储不压缩（D-26 语义边界）")
	assert.Empty(t, result.FilePath)
}

// -----------------------------------------------------------------------------
// auto 路径统一（D-25）+ GetBackupContent 双检查（D-23）
// -----------------------------------------------------------------------------

// TestCbk93AutoBackupCompressesLargeFile — D-25: the auto path
// (createNewAutoBackup) compresses large configs into .conf.gz exactly like
// CreateBackup; BackupSize keeps the original size; GetBackupContent roundtrips.
func TestCbk93AutoBackupCompressesLargeFile(t *testing.T) {
	ctx := context.Background()
	cbk7906Chdir(t)
	db := newDB7906(t, &models.ConfigBackup{}, &models.NetworkDevice{}, &models.Config{})

	// threshold=1KB；>1KB config → 文件分支
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  "备份阈值",
		ConfigKey:   "network.config.backup.threshold",
		ConfigValue: "1",
	}).Error)

	svc := NewConfigBackupService(db, nil) // createNewAutoBackup 不触达 executor

	dev := cbk7906SeedDevice(t, db, "dev-cbk93-auto", "cbk93-auto-switch")
	config := cbk93LargeOutput()
	require.Greater(t, len(config), 1024)

	skipped, err := svc.createNewAutoBackup(ctx, dev, config, calculateHash(config))
	require.NoError(t, err)
	assert.False(t, skipped, "新备份创建 → skipped=false")

	items, total, err := svc.GetBackupList(ctx, 1, 10, dev.ID, "", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	bk := items[0].ConfigBackup

	assert.Equal(t, models.StorageTypeFile, bk.StorageType)
	assert.True(t, bk.Compressed, "D-25: auto 路径大文件必须压缩")
	assert.True(t, strings.HasSuffix(bk.FilePath, ".gz"), "D-25: auto 路径 .conf.gz 命名")
	assert.Equal(t, len(config), bk.BackupSize, "D-24: 原始口径")
	assert.FileExists(t, bk.FilePath)

	content, err := svc.GetBackupContent(ctx, bk.ID)
	require.NoError(t, err)
	assert.Equal(t, config, content, "GetBackupContent 双检查路径解压还原")
}

// TestCbk93GetBackupContentCompressed — D-23 double-check: Compressed=true +
// .gz file decompresses to plaintext; flag/suffix mismatch errors explicitly
// (备份压缩状态不一致) instead of silently returning raw bytes.
func TestCbk93GetBackupContentCompressed(t *testing.T) {
	ctx := context.Background()
	svc, db, tmp := newCbk7906(t)

	original := strings.Repeat("sysname cbk93-content\n", 40)
	compressed, err := gzipCompress(original)
	require.NoError(t, err)

	gzPath := filepath.Join(tmp, "mixed_v1_20260101_000000.conf.gz")
	require.NoError(t, os.WriteFile(gzPath, compressed, 0o644))

	bkGz := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-c93a", DeviceName: "cbk93-a", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeFile, FilePath: gzPath, Compressed: true, Version: 1,
	})
	content, err := svc.GetBackupContent(ctx, bkGz.ID)
	require.NoError(t, err)
	assert.Equal(t, original, content, "Compressed+.gz 双检查成立 → 解压还原明文")

	// mismatch: Compressed=true 但 FilePath 无 .gz 后缀 → 显式报错
	plainPath := filepath.Join(tmp, "mixed_v2_20260101_000000.conf")
	require.NoError(t, os.WriteFile(plainPath, []byte(original), 0o644))
	bkMismatch := cbk7906SeedBackup(t, db, &models.ConfigBackup{
		DeviceID: "dev-c93b", DeviceName: "cbk93-b", BackupType: models.BackupTypeManual,
		StorageType: models.StorageTypeFile, FilePath: plainPath, Compressed: true, Version: 2,
	})
	_, err = svc.GetBackupContent(ctx, bkMismatch.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "备份压缩状态不一致")
	assert.Contains(t, err.Error(), "compressed=true")
}
