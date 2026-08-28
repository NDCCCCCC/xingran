package scheduler

// Phase 80 Plan 02 — dept_sync_tasks.go 测试
// 覆盖:RegisterDeptSyncTasks 注册 + pool stub 同包直注 + executeDeptToADSyncTask 分支
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// newSchedDB8002_DST sqlite 文件库。
func newSchedDB8002_DST(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "dst8002.db")
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

// newScheduler8002_DST scheduler fixture。
func newScheduler8002_DST(t *testing.T) (*Scheduler, *gorm.DB) {
	t.Helper()
	db := newSchedDB8002_DST(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	return s, db
}

// stubADPool8002 DST pool stub。
type stubADPool8002 struct{}

func (s *stubADPool8002) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	return nil, 0, nil
}
func (s *stubADPool8002) CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error) {
	return 0, 0, 0, 0, nil
}
func (s *stubADPool8002) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) RecoverExpiredBreakers(ctx context.Context) (int, error) {
	return 0, nil
}
func (s *stubADPool8002) InvalidateCache(configID string) {}
func (s *stubADPool8002) StartHotReload(ctx context.Context) error { return nil }
func (s *stubADPool8002) Create(ctx context.Context, account *models.ADServiceAccount) error { return nil }
func (s *stubADPool8002) Update(ctx context.Context, account *models.ADServiceAccount) error { return nil }
func (s *stubADPool8002) Delete(ctx context.Context, accountID string) error { return nil }
func (s *stubADPool8002) MarkSuccess(ctx context.Context, accountID string) error { return nil }
func (s *stubADPool8002) MarkFailure(ctx context.Context, accountID, reason string) error { return nil }
func (s *stubADPool8002) ManualUnlock(ctx context.Context, accountID, operator, reason string) error { return nil }
func (s *stubADPool8002) SetEnabled(ctx context.Context, accountID string, enabled bool) error { return nil }

var _ addomain.AccountPool = (*stubADPool8002)(nil)

// TestDst8002_RegisterDeptSyncTasks 注册函数。
func TestDst8002_RegisterDeptSyncTasks(t *testing.T) {
	s, _ := newScheduler8002_DST(t)
	RegisterDeptSyncTasks(s)
	assert.True(t, s.IsTaskRegistered("dept_to_ad_sync"))
	assert.True(t, s.IsTaskRegistered("dept_member_to_ad_group_sync"))
}

// TestDst8002_PoolSeam getGlobalADAccountPool → pool stub 同包直注。
func TestDst8002_PoolSeam(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })

	db := newSchedDB8002_DST(t)
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{}
	globalADSyncScheduler = sched

	pool := getGlobalADAccountPool()
	assert.NotNil(t, pool)
}

// TestDst8002_GetDefaultADConfigIDForDept 空 config / 有 config 两分支。
func TestDst8002_GetDefaultADConfigIDForDept(t *testing.T) {
	db := newSchedDB8002_DST(t)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	// 分支 1:空表
	_, err := getDefaultADConfigIDForDept(context.Background(), db, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到启用的AD配置")

	// 分支 2:有启用配置
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-dst", "测试配置", models.ADConfigStatusEnabled).Error)
	id, err := getDefaultADConfigIDForDept(context.Background(), db, nil)
	require.NoError(t, err)
	assert.Equal(t, "cfg-dst", id)
}
