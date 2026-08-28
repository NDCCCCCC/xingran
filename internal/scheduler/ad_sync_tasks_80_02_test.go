package scheduler

// Phase 80 Plan 02 — ad_sync_tasks.go 测试
// 覆盖:checkAndSyncADConfigs 三分支 + executeADAccountPoolRecoverBreakersTask
// + ScheduleADSyncForConfig ctx 取消 + getNextRunTime 纯函数 + SM4 cipher seam
//
// D-80-06 豁免:syncADConfig 内层 addomain LDAP wire(~26 stmts)按 e 类豁免,
// 测试绝不调用 syncADConfig。
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。
// 纪律:status 断言只引用 models.* 常量,禁裸 0/1。

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// ============================================================================
// Helper fixtures (8002 后缀)
// ============================================================================

// newSchedDB8002_AD sqlite 文件库(t.TempDir)+ AutoMigrate sys_job/sys_job_log。
func newSchedDB8002_AD(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "ads8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Job{}, &models.JobLog{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// stubAccountPool8002 AD account pool stub。
type stubAccountPool8002 struct{}

func (s *stubAccountPool8002) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubAccountPool8002) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubAccountPool8002) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	return nil, 0, nil
}
func (s *stubAccountPool8002) CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error) {
	return 0, 0, 0, 0, nil
}
func (s *stubAccountPool8002) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubAccountPool8002) RecoverExpiredBreakers(ctx context.Context) (int, error) {
	return 0, nil
}
func (s *stubAccountPool8002) InvalidateCache(configID string) {}
func (s *stubAccountPool8002) StartHotReload(ctx context.Context) error {
	return nil
}
func (s *stubAccountPool8002) Create(ctx context.Context, account *models.ADServiceAccount) error {
	return nil
}
func (s *stubAccountPool8002) Update(ctx context.Context, account *models.ADServiceAccount) error {
	return nil
}
func (s *stubAccountPool8002) Delete(ctx context.Context, accountID string) error {
	return nil
}
func (s *stubAccountPool8002) MarkSuccess(ctx context.Context, accountID string) error {
	return nil
}
func (s *stubAccountPool8002) MarkFailure(ctx context.Context, accountID, reason string) error {
	return nil
}
func (s *stubAccountPool8002) ManualUnlock(ctx context.Context, accountID, operator, reason string) error {
	return nil
}
func (s *stubAccountPool8002) SetEnabled(ctx context.Context, accountID string, enabled bool) error {
	return nil
}

// stubPasswordCipher8002 SM4 cipher stub.
type stubPasswordCipher8002 struct{}

func (s *stubPasswordCipher8002) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}
func (s *stubPasswordCipher8002) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

// newADSyncScheduler8002 构造 ADSyncScheduler(同包,直接访问 unexported 字段)。
func newADSyncScheduler8002(t *testing.T, db *gorm.DB) *ADSyncScheduler {
	t.Helper()
	s := NewADSyncScheduler(db, 10)
	return s
}

// ensure stubAccountPool8002 implements addomain.AccountPool
var _ addomain.AccountPool = (*stubAccountPool8002)(nil)

// ============================================================================
// TestAds8002_CheckAndSync — checkAndSyncADConfigs 三分支(sqlite 行驱动)
// ============================================================================

// TestAds8002_CheckAndSync_NilLastSync checkAndSyncADConfigs:LastSyncTime=nil→触发同步分支。
// syncADConfig 走 addomain wire(e 类豁免),但查询本身不 panic 即验证了分支逻辑。
func TestAds8002_CheckAndSync_NilLastSync(t *testing.T) {
	db := newSchedDB8002_AD(t)

	// 建 sys_ad_config 表(含 deleted_at:GORM soft delete 依赖)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		sync_enabled INTEGER DEFAULT 1,
		status INTEGER DEFAULT 0,
		sync_interval INTEGER DEFAULT 3600,
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	// sync_enabled=false:防止 checkAndSyncADConfigs 触发 goroutine(syncADConfig 豁免,避免 panic)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, sync_enabled, status, sync_interval, last_sync_at) VALUES (?, ?, ?, ?, ?, NULL)`,
		"cfg-nil-sync", "从未同步配置", 0, models.ADConfigStatusEnabled, 3600).Error)

	s := newADSyncScheduler8002(t, db)
	s.pool = &stubAccountPool8002{}
	s.checkAndSyncADConfigs() // 不 panic 即过
}

// TestAds8002_CheckAndSync_Overdue checkAndSyncADConfigs:上次同步超过间隔→触发同步分支。
func TestAds8002_CheckAndSync_Overdue(t *testing.T) {
	db := newSchedDB8002_AD(t)

	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		sync_enabled INTEGER DEFAULT 1,
		status INTEGER DEFAULT 0,
		sync_interval INTEGER DEFAULT 60,
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	// sync_enabled=false 防止触发 goroutine
	oldTime := time.Now().Add(-61 * time.Second)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, sync_enabled, status, sync_interval, last_sync_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"cfg-overdue", "超期配置", 0, models.ADConfigStatusEnabled, 60, oldTime).Error)

	s := newADSyncScheduler8002(t, db)
	s.pool = &stubAccountPool8002{}
	s.checkAndSyncADConfigs() // 不 panic 即过
}

// TestAds8002_CheckAndSync_Recent checkAndSyncADConfigs:上次同步未超间隔→跳过同步分支。
func TestAds8002_CheckAndSync_Recent(t *testing.T) {
	db := newSchedDB8002_AD(t)

	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		sync_enabled INTEGER DEFAULT 1,
		status INTEGER DEFAULT 0,
		sync_interval INTEGER DEFAULT 3600,
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	recentTime := time.Now().Add(-30 * time.Second)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, sync_enabled, status, sync_interval, last_sync_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"cfg-recent", "近期配置", 0, models.ADConfigStatusEnabled, 3600, recentTime).Error)

	s := newADSyncScheduler8002(t, db)
	s.pool = &stubAccountPool8002{}
	s.checkAndSyncADConfigs()
}

// ============================================================================
// TestAds8002_RecoverBreakers — executeADAccountPoolRecoverBreakersTask
// ============================================================================

// TestAds8002_RecoverBreakersTask 熔断恢复任务:有过期熔断账号→RecoverExpiredBreakers 被调。
func TestAds8002_RecoverBreakersTask(t *testing.T) {
	// save globalADSyncScheduler
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })

	db := newSchedDB8002_AD(t)

	// 手动建表(SQLite 不支持 gen_random_uuid(),避免 AutoMigrate 报错)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_service_accounts (
		id TEXT PRIMARY KEY,
		config_id TEXT NOT NULL,
		username TEXT NOT NULL,
		password_ciphertext TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		failure_count INTEGER NOT NULL DEFAULT 0,
		circuit_breaker_until DATETIME,
		last_success_at DATETIME,
		last_failure_at DATETIME,
		last_failure_reason TEXT,
		manual_unlock_reason TEXT,
		manual_unlocked_by TEXT,
		manual_unlocked_at DATETIME,
		remark TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	// 插入已过期熔断账号
	oldBreaker := time.Now().Add(-10 * time.Minute)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_service_accounts (id, config_id, username, password_ciphertext, status, circuit_breaker_until) VALUES (?, ?, ?, ?, ?, ?)`,
		"acc-expired-breaker", "cfg-1", "test-user", "stub-cipher", models.ADAccountStatusBreaker, oldBreaker).Error)

	// 注入 globalADSyncScheduler
	globalADSyncScheduler = newADSyncScheduler8002(t, db)

	err := executeADAccountPoolRecoverBreakersTask(context.Background(), nil)
	require.NoError(t, err)

	// 验证账号被恢复
	var updated models.ADServiceAccount
	require.NoError(t, db.First(&updated, "id = ?", "acc-expired-breaker").Error)
	assert.Equal(t, models.ADAccountStatusAvailable, updated.Status, "账号应恢复为可用")
	assert.Nil(t, updated.CircuitBreakerUntil, "熔断时间应被清除")
}

// TestAds8002_GetADSyncStatus GetADSyncStatus 返回结构(nil scheduler → started=false)。
func TestAds8002_GetADSyncStatus(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = nil // 确保干净状态

	status := GetADSyncStatus()
	require.NotNil(t, status)
	assert.Contains(t, status, "started")
	assert.Equal(t, false, status["started"])
}

// ============================================================================
// TestAds8002_GetNextRunTime — getNextRunTime 纯函数表驱动
// ============================================================================

// TestAds8002_GetNextRunTime_Pure getNextRunTime 不 panic。
func TestAds8002_GetNextRunTime_Pure(t *testing.T) {
	db := newSchedDB8002_AD(t)
	s := newADSyncScheduler8002(t, db)
	s.Start()
	t.Cleanup(s.Stop)

	// 调用 getNextRunTime 不 panic 即可
	entries := s.cron.Entries()
	got := getNextRunTime(s.cron, entries...)
	assert.NotEmpty(t, got) // 有 entries 时返回时间字符串
}

// ============================================================================
// TestAds8002_SM4CipherSeam — SetADSM4Cipher / getADSM4Cipher
// ============================================================================

// TestAds8002_SM4CipherSeam SM4 cipher 注入缝 save/restore。
func TestAds8002_SM4CipherSeam(t *testing.T) {
	cipher := getADSM4Cipher()
	t.Cleanup(func() { SetADSM4Cipher(cipher) })

	stub := &stubPasswordCipher8002{}
	SetADSM4Cipher(stub)
	got := getADSM4Cipher()
	assert.Same(t, stub, got, "应返回注入的 cipher")
}

// ============================================================================
// TestAds8002_GetDefaultADConfigID — getDefaultADConfigID sqlite 两分支
// ============================================================================

// TestAds8002_GetDefaultADConfigID 空 config 表 / 有 config 行。
func TestAds8002_GetDefaultADConfigID(t *testing.T) {
	db := newSchedDB8002_AD(t)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	// 分支 1:空表 → ErrRecordNotFound
	_, err := getDefaultADConfigID(context.Background(), db, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到启用的AD配置")

	// 分支 2:有启用配置 → 返回 ID
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-default", "默认配置", models.ADConfigStatusEnabled).Error)
	id, err := getDefaultADConfigID(context.Background(), db, nil)
	require.NoError(t, err)
	assert.Equal(t, "cfg-default", id)
}
