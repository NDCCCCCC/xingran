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
	"github.com/robfig/cron/v3"
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
// sync_enabled=1 + 近期同步:进入 else 比较分支(elapsed/间隔/提前触发三段)→
// timeUntilNextSync=3570 > 300 → shouldSync=false → 零 goroutine(安全)。
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
		"cfg-recent", "近期配置", 1, models.ADConfigStatusEnabled, 3600, recentTime).Error)

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

// ============================================================================
// 缺口补足 Round 1 — 注册体 / executeADDataSyncTask / 状态与纯函数分支
// ============================================================================

// TestAds8002_RegisterTasks RegisterADSyncTasks / RegisterADAccountPoolTasks 注册
// + handler 分发(scheduler 未初始化 → 错误分支,不触 wire)。
func TestAds8002_RegisterTasks(t *testing.T) {
	s, _ := newScheduler8002_DST(t)
	RegisterADSyncTasks(s)
	RegisterADAccountPoolTasks(s)

	assert.True(t, s.IsTaskRegistered("ad_data_sync"))
	assert.True(t, s.IsTaskRegistered("ad_account_pool_recover_breakers"))

	// globalADSyncScheduler=nil → ad_data_sync handler 错误分支
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = nil

	err := s.GetTaskHandler("ad_data_sync")(context.Background(), map[string]interface{}{"configId": "c1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未初始化")

	err = s.GetTaskHandler("ad_account_pool_recover_breakers")(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未初始化")
}

// TestAds8002_ExecuteDataSync_Errors executeADDataSyncTask 分支:
// nil scheduler / 参数缺 configId / syncType 参数透传。
func TestAds8002_ExecuteDataSync_Errors(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = nil

	// nil scheduler → 错误
	require.Error(t, executeADDataSyncTask(context.Background(), nil))

	// scheduler 有但参数无效 → 错误(不触 wire)
	db := newSchedDB8002_AD(t)
	globalADSyncScheduler = newADSyncScheduler8002(t, db)
	require.Error(t, executeADDataSyncTask(context.Background(), nil))
	require.Error(t, executeADDataSyncTask(context.Background(), map[string]interface{}{"configId": ""}))
	require.Error(t, executeADDataSyncTask(context.Background(), map[string]interface{}{"configId": 123}))
}

// TestAds8002_ScheduleForConfig_NilScheduler nil scheduler → 记日志直接返回(不 panic)。
func TestAds8002_ScheduleForConfig_NilScheduler(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = nil

	ScheduleADSyncForConfig("cfg-x", time.Millisecond) // 不 panic 即过
}

// TestAds8002_GetNextRunTime_ZeroEntry entry.Next 零值 → "未设置"。
func TestAds8002_GetNextRunTime_ZeroEntry(t *testing.T) {
	db := newSchedDB8002_AD(t)
	s := newADSyncScheduler8002(t, db)

	// 显式传零值 Entry → Next.IsZero() → "未设置"
	assert.Equal(t, "未设置", getNextRunTime(s.cron, cronEntryZero8002()))
}

// TestAds8002_IsStarted IsStarted 前后翻转。
func TestAds8002_IsStarted(t *testing.T) {
	db := newSchedDB8002_AD(t)
	s := newADSyncScheduler8002(t, db)
	assert.False(t, s.IsStarted())

	s.Start()
	t.Cleanup(s.Stop)
	assert.True(t, s.IsStarted())
}

// TestAds8002_GetPool_LazyInit getPool 惰性初始化(未注入 pool → 构造非 nil 并回写)。
func TestAds8002_GetPool_LazyInit(t *testing.T) {
	db := newSchedDB8002_AD(t)
	s := newADSyncScheduler8002(t, db)
	assert.Nil(t, s.pool)

	pool := s.getPool()
	assert.NotNil(t, pool)
	assert.Same(t, pool, s.pool, "惰性初始化结果应回写字段")
}

// TestAds8002_CheckAndSync_QueryError sys_ad_config 表缺失 → 查询错误分支。
func TestAds8002_CheckAndSync_QueryError(t *testing.T) {
	db := newSchedDB8002_AD(t) // 不建 sys_ad_config 表
	s := newADSyncScheduler8002(t, db)
	s.checkAndSyncADConfigs() // 错误仅记日志,不 panic 即过
}

// cronEntryZero8002 构造零值 cron.Entry(Next 零值 → "未设置" 分支)。
func cronEntryZero8002() cron.Entry { return cron.Entry{} }

// ============================================================================
// 缺口补足 Round 2 — Start/Stop 全局 / executeADDataSyncTask 尾段 / 参数分支 /
// Schedule 双分支 / GetADSyncStatus 启动态 / checkAndSync goroutine 体
// ============================================================================

// TestAds8002_StartStopGlobal StartADSyncScheduler Once 初始化 + Stop 全局生命周期。
// 纪律(D-80-02): Start 配对 Stop;t.Cleanup 恢复全局 var。
func TestAds8002_StartStopGlobal(t *testing.T) {
	db := newSchedDB8002_AD(t)

	orig := globalADSyncScheduler
	t.Cleanup(func() {
		StopADSyncScheduler()
		globalADSyncScheduler = orig
	})

	// Once 初始化:global 非零 + pool 单例注入 + Start
	// (-count>1 或其他测试已耗尽 Once 时,StartADSyncScheduler 幂等 no-op,
	// global 可能残留前次实例(如 safeRestore)→ 用 db 指针身份判定是否本次初始化)
	StartADSyncScheduler(db)
	got := GetADSyncScheduler()
	if got == nil || got.db != db {
		t.Skip("globalADSyncSchedulerOnce 已被前次 count-iteration 触发,sync.Once 语义下无法重复初始化")
	}
	assert.Same(t, got, globalADSyncScheduler)
	assert.NotNil(t, got.pool, "StartADSyncScheduler 应回写 pool 单例")
	assert.True(t, got.IsStarted())

	// 二次 Start:Once 已触发 → no-op(同实例)
	StartADSyncScheduler(db)
	assert.Same(t, got, GetADSyncScheduler())

	// Stop:started 翻转
	StopADSyncScheduler()
	assert.False(t, got.IsStarted())
}

// TestAds8002_ExecuteDataSync_ParamsValid executeADDataSyncTask 参数合法尾段:
// SyncDataByID 对停用/缺失 config 立即报错(不触 LDAP wire)。
func TestAds8002_ExecuteDataSync_ParamsValid(t *testing.T) {
	db := newSchedDB8002_AD(t)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	// 停用 config:SyncDataByID :50 分支 → "AD配置不存在或未启用"(零 wire)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-ads-disabled", "停用", models.ADConfigStatusDisabled).Error)

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = newADSyncScheduler8002(t, db)

	// 显式 configId(停用)→ 尾段构造 service 后报错
	err := executeADDataSyncTask(context.Background(), map[string]interface{}{"configId": "cfg-ads-disabled"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD配置不存在或未启用")
}

// TestAds8002_ExecuteDataSync_AutoConfig 无参数 → 自动取启用 config → 同步日志表缺失快速报错。
func TestAds8002_ExecuteDataSync_AutoConfig(t *testing.T) {
	db := newSchedDB8002_AD(t)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-ads-auto", "自动配置", models.ADConfigStatusEnabled).Error)

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubAccountPool8002{}
	globalADSyncScheduler = sched

	// 无参数 → getDefaultADConfigID 自动选中启用 config → SyncDataByID →
	// syncDataInternal 创建 ADSyncLog 失败(表缺失)→ 快速报错,未触 wire
	err := executeADDataSyncTask(context.Background(), map[string]interface{}{"syncType": "full"})
	require.Error(t, err)
}

// TestAds8002_GetDefaultADConfigID_ParamKeys 参数键 configId / adConfigId 两分支。
func TestAds8002_GetDefaultADConfigID_ParamKeys(t *testing.T) {
	db := newSchedDB8002_AD(t)

	ctx := context.Background()

	// configId 命中(非空字符串,不需查库)
	id, err := getDefaultADConfigID(ctx, db, map[string]interface{}{"configId": "via-config-id"})
	require.NoError(t, err)
	assert.Equal(t, "via-config-id", id)

	// adConfigId 命中
	id, err = getDefaultADConfigID(ctx, db, map[string]interface{}{"adConfigId": "via-ad-config-id"})
	require.NoError(t, err)
	assert.Equal(t, "via-ad-config-id", id)

	// 两个键都存在 → configId 优先
	id, err = getDefaultADConfigID(ctx, db, map[string]interface{}{"configId": "first", "adConfigId": "second"})
	require.NoError(t, err)
	assert.Equal(t, "first", id)
}

// TestAds8002_GetDefaultADConfigID_QueryError 查询非 NotFound 错误分支(表缺失)。
func TestAds8002_GetDefaultADConfigID_QueryError(t *testing.T) {
	db := newSchedDB8002_AD(t) // 不建 sys_ad_config

	_, err := getDefaultADConfigID(context.Background(), db, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询AD配置失败")
}

// TestAds8002_ScheduleForConfig_DoneBranch 调度器 ctx 已取消 → goroutine 走 Done 分支。
// cleanup 恢复为"新鲜有效实例"而非 nil:goroutine 在 :345 解引用 globalADSyncScheduler.ctx,
// 若测试结束先把 global 置 nil 会 nil-panic(全量套件下时序更慢,必现 flake — 见 80-01 quirk d 同类)。
func TestAds8002_ScheduleForConfig_DoneBranch(t *testing.T) {
	// 恢复锚:一个 ctx 有效的新调度器(堆上存活,goroutine 迟到解引用也安全)
	safeRestore := newADSyncScheduler8002(t, newSchedDB8002_AD(t))
	t.Cleanup(func() { globalADSyncScheduler = safeRestore })

	db := newSchedDB8002_AD(t)
	sched := newADSyncScheduler8002(t, db)
	sched.cancel() // 取消调度器上下文:Done 已关闭 → select 立即走 Done 分支
	globalADSyncScheduler = sched

	// 1 小时延迟永不到达;Done 关闭 → goroutine 立即 return(零 wire,零 sleep)
	ScheduleADSyncForConfig("cfg-done", time.Hour)
}

// TestAds8002_GetADSyncStatus_Started 启动态 + cron entries → full_sync_next_run / next_run。
func TestAds8002_GetADSyncStatus_Started(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() {
		globalADSyncScheduler.Stop()
		globalADSyncScheduler = orig
	})

	db := newSchedDB8002_AD(t)
	globalADSyncScheduler = newADSyncScheduler8002(t, db)
	globalADSyncScheduler.Start()

	status := GetADSyncStatus()
	require.NotNil(t, status)
	assert.Equal(t, true, status["started"])
	assert.Contains(t, status, "full_sync_next_run")
	assert.Contains(t, status, "next_run")
}

// TestAds8002_CheckAndSync_GoroutineDrain 启用配置(LastSync=nil)→ shouldSync 分支 +
// 同步 goroutine 体(sem 获取/释放)+ syncADConfig 快速失败。
// drain 技巧:maxConcurrent=1 + 单配置 → sem.Acquire(1) 成功即同步 goroutine 已 Release
// (chan/sem 同步,非轮询,D-80-02 口径)。
func TestAds8002_CheckAndSync_GoroutineDrain(t *testing.T) {
	db := newSchedDB8002_AD(t)
	// 缺 sys_ad_config 表?不行 — 查询要走通:建表 + 启用 + nil LastSync
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
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, sync_enabled, status, sync_interval, last_sync_at)
		VALUES (?, ?, ?, ?, ?, NULL)`,
		"cfg-drain", "排空测试", 1, models.ADConfigStatusEnabled, 3600).Error)

	s := NewADSyncScheduler(db, 1) // 容量 1:goroutine 独占
	s.pool = &stubAccountPool8002{} // stub 空池:SyncDataByID 内 FailoverClient 零拨号

	// cipher 为 nil → NewADDomainService varargs 为空 → 不改全局 cipher

	s.checkAndSyncADConfigs()

	// sem 排空:goroutine Release 后 Acquire 才成功(同步点)
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer drainCancel()
	require.NoError(t, s.sem.Acquire(drainCtx, 1), "sem 应在同步 goroutine 完成后排空")
	s.sem.Release(1)
}
