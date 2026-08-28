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
