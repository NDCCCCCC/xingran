package scheduler

// Phase 80 Plan 02 — mac_history + mac_matview 测试
// 覆盖:PartitionService/MACHistoryPurgeService 双 seam stub + upsertMACHistoryJob 三态
// + executeCleanup/Purge 分发 + RegisterMACHistoryMatViewTasks
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

	"github.com/xingran-next/xingran-go-backend/internal/services"
)

// newSchedDB8002_MAC sqlite 文件库。
func newSchedDB8002_MAC(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "mac8002.db")
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

// stubPartitionService8002 PartitionService stub。
type stubPartitionService8002 struct{}

func (s *stubPartitionService8002) CreateMonthlyPartition(ctx context.Context, year int, month int) error {
	return nil
}
func (s *stubPartitionService8002) EnsurePartitionsExist(ctx context.Context, monthsAhead int) error {
	return nil
}
func (s *stubPartitionService8002) DropExpiredPartitions(ctx context.Context) error {
	return nil
}
func (s *stubPartitionService8002) GetRetentionDays(ctx context.Context) int {
	return 90
}

var _ PartitionService = (*stubPartitionService8002)(nil)

// stubPurgeService8002 MACHistoryPurgeService stub。
type stubPurgeService8002 struct {
	err error
}

func (s *stubPurgeService8002) PurgeMeaninglessRecords(ctx context.Context, dryRun bool) (int64, string, error) {
	if s.err != nil {
		return 0, "", s.err
	}
	return 42, "backup_20260101", nil
}

var _ MACHistoryPurgeService = (*stubPurgeService8002)(nil)

// stubMatViewService8002 MACHistoryMatViewService stub。
type stubMatViewService8002 struct{}

func (s *stubMatViewService8002) RefreshAllMaterializedViews(ctx context.Context) error {
	return nil
}
func (s *stubMatViewService8002) RefreshSingleMatView(ctx context.Context, name string) error {
	return nil
}

var _ services.MACHistoryMatViewService = (*stubMatViewService8002)(nil)

// ============================================================================
// TestMht8002_UpsertJob — upsertMACHistoryJob 三态
// ============================================================================

// TestMht8002_UpsertJob upsertMACHistoryJob:不存在→create。
func TestMht8002_UpsertJob(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	err := upsertMACHistoryJob(db, s, "MAC测试任务", "mac_history_cleanup", "0 0 2 * * ?", "MAC清理", 2)
	require.NoError(t, err)

	var job models.Job
	require.NoError(t, db.Where("job_name = ?", "MAC测试任务").First(&job).Error)
	assert.Equal(t, models.JobStatusNormal, job.Status)
}

// ============================================================================
// TestMht8002_Execute — executeMACHistoryCleanup / executeMACHistoryPurge 分发
// ============================================================================

// TestMht8002_ExecuteCleanup 分发到 globalPartitionService(需先触发 Once 设置,故跳过 nil-guard 分支)。
func TestMht8002_ExecuteCleanup(t *testing.T) {
	// 注意:sync.Once 导致同包后续测试无法 reset,本测试只验证成功路径。
	// nil-guard 分支被 sync.Once 污染,无法在包级测试套件中可靠运行。
	stub := &stubPartitionService8002{}
	SetPartitionService(stub)

	err := executeMACHistoryCleanup(context.Background(), nil)
	require.NoError(t, err)
}

// TestMht8002_ExecutePurge 分发到 globalPurgeService。
func TestMht8002_ExecutePurge(t *testing.T) {
	stub := &stubPurgeService8002{}
	SetMACHistoryPurgeService(stub)

	err := executeMACHistoryPurge(context.Background(), nil)
	require.NoError(t, err)
}

// ============================================================================
// TestMmv8002 — RegisterMACHistoryMatViewTasks
// ============================================================================

// TestMmv8002_Register Register → IsTaskRegistered true。
func TestMmv8002_Register(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	svc := &stubMatViewService8002{}
	RegisterMACHistoryMatViewTasks(s, db, svc)

	assert.True(t, s.IsTaskRegistered("mac_history_matview_refresh"))
}

// ============================================================================
// 缺口补足 Round 1 — RegisterMACHistoryTasks / upsert 三态 / 错误分发 / matview 分支
// ============================================================================

// errPartitionService8002 DropExpiredPartitions 返回错误的 PartitionService stub。
type errPartitionService8002 struct{ stubPartitionService8002 }

func (e *errPartitionService8002) DropExpiredPartitions(ctx context.Context) error {
	return assert.AnError
}

// errPurgeService8002 PurgeMeaninglessRecords 返回错误的 MACHistoryPurgeService stub。
type errPurgeService8002 struct{}

func (e *errPurgeService8002) PurgeMeaninglessRecords(ctx context.Context, dryRun bool) (int64, string, error) {
	return 0, "backup_tbl", assert.AnError
}

var (
	_ PartitionService       = (*errPartitionService8002)(nil)
	_ MACHistoryPurgeService = (*errPurgeService8002)(nil)
)

// TestMht8002_RegisterTasks RegisterMACHistoryTasks:db nil 只注册 handler;有 db 落两条 job。
func TestMht8002_RegisterTasks(t *testing.T) {
	s, db := newScheduler8002_DST(t)

	// db=nil → 只注册 handler,提前返回
	RegisterMACHistoryTasks(s, nil)
	assert.True(t, s.IsTaskRegistered("mac_history_cleanup"))
	assert.True(t, s.IsTaskRegistered("mac_history_purge_monthly"))

	// 有 db → 两条 job 落库
	RegisterMACHistoryTasks(s, db)
	var count int64
	require.NoError(t, db.Model(&models.Job{}).Where("job_name LIKE ?", "MAC历史%").Count(&count).Error)
	assert.Equal(t, int64(2), count)

	// 重复注册 → 已存在跳过(幂等)
	RegisterMACHistoryTasks(s, db)
	require.NoError(t, db.Model(&models.Job{}).Where("job_name LIKE ?", "MAC历史%").Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// TestMht8002_UpsertJob_Exists upsertMACHistoryJob 已存在分支。
func TestMht8002_UpsertJob_Exists(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	require.NoError(t, upsertMACHistoryJob(db, s, "MAC幂等任务", "mac_history_cleanup", "0 0 2 * * ?", "r", 2))
	// 二次 → 已存在跳过
	require.NoError(t, upsertMACHistoryJob(db, s, "MAC幂等任务", "mac_history_cleanup", "0 0 2 * * ?", "r", 2))

	var count int64
	require.NoError(t, db.Model(&models.Job{}).Where("job_name = ?", "MAC幂等任务").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestMht8002_UpsertJob_BadCron upsertMACHistoryJob 非法 cron → AddJob 错误分支。
func TestMht8002_UpsertJob_BadCron(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	s.running = true // 同包直写:running 态 AddJob 才会走 cron.AddFunc 校验

	err := upsertMACHistoryJob(db, s, "MAC坏cron", "mac_history_cleanup", "not-a-cron", "r", 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "创建任务失败")
}

// TestMht8002_ExecuteCleanup_Error 清理错误分支(同包直写 var,绕 sync.Once)。
func TestMht8002_ExecuteCleanup_Error(t *testing.T) {
	orig := globalPartitionService
	t.Cleanup(func() { globalPartitionService = orig })
	globalPartitionService = &errPartitionService8002{}

	err := executeMACHistoryCleanup(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "清理MAC历史分区失败")
}

// TestMht8002_ExecutePurge_Error purge 错误分支(同包直写 var,绕 sync.Once)。
func TestMht8002_ExecutePurge_Error(t *testing.T) {
	orig := globalPurgeService
	t.Cleanup(func() { globalPurgeService = orig })
	globalPurgeService = &errPurgeService8002{}

	err := executeMACHistoryPurge(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAC历史 purge 失败")
}

// ============================================================================
// TestMmv8002 缺口补足 — handler 执行 / db nil / 已存在
// ============================================================================

// TestMmv8002_HandlerExec 注册后的 handler 执行一次分发(matview stub 被调)。
func TestMmv8002_HandlerExec(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	RegisterMACHistoryMatViewTasks(s, db, &stubMatViewService8002{})
	handler := s.GetTaskHandler("mac_history_matview_refresh")
	require.NotNil(t, handler)
	require.NoError(t, handler(context.Background(), nil))
}

// TestMmv8002_Register_DBNil db nil → 只注册 handler,提前返回。
func TestMmv8002_Register_DBNil(t *testing.T) {
	s := NewScheduler(newSchedDB8002_MAC(t))
	s.SetLogger(&schedStubLogger8001{})

	RegisterMACHistoryMatViewTasks(s, nil, &stubMatViewService8002{})
	assert.True(t, s.IsTaskRegistered("mac_history_matview_refresh"))
}

// TestMmv8002_Register_Exists 已存在同名 job → 跳过创建(幂等)。
func TestMmv8002_Register_Exists(t *testing.T) {
	db := newSchedDB8002_MAC(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	existing := &models.Job{JobName: "MAC历史物化视图刷新"}
	require.NoError(t, db.Create(existing).Error)

	RegisterMACHistoryMatViewTasks(s, db, &stubMatViewService8002{})

	var count int64
	require.NoError(t, db.Model(&models.Job{}).Where("job_name = ?", "MAC历史物化视图刷新").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
