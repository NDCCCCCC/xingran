package scheduler

// W7.2 / 任务 A-4：孤儿 job 对账（prune）的真库集成用例。
//
// 覆盖 REQUIREMENTS 的验收句：
//   「构造 3 种孤儿（物理删 schedule / 软删 schedule / 停用 schedule）+ 1 条正常，
//     断言只保留正常的 1 条，其余 3 条 sys_job 被摘除」。
//
// 真库：sqlite 内存库 + 真实 GORM AutoMigrate（models.Job + rpamodels.Schedule，
// 禁止手写 CREATE TABLE 掩盖 schema 漂移）。
//
// 真同步器：RPAScheduleJobSyncer（真实 findJob / RemoveScheduleJob / prune 路径），
// 真服务：rpav1.ScheduleService.SyncEnabledJobs（与启动时同一条调用链）。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	rpav1 "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// pruneTestDB 建真 sqlite 内存库（真实 AutoMigrate）。
func pruneTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Job{},
		&rpamodels.Schedule{},
	), "真实 AutoMigrate 必须能建出定时任务表与调度表")
	return db
}

// pruneSeedSchedule 落一条调度行。
func pruneSeedSchedule(t *testing.T, db *gorm.DB, name string, status int) *rpamodels.Schedule {
	t.Helper()
	sched := &rpamodels.Schedule{
		Name:     name,
		TaskID:   "11111111-1111-1111-1111-111111111111",
		CronExpr: "0 * * * * *",
		Status:   status,
	}
	require.NoError(t, db.Create(sched).Error)
	return sched
}

// pruneSeedJob 造一条 RPA 分组的调度派生 job（invoke_target + remark 标记齐全）。
func pruneSeedJob(t *testing.T, db *gorm.DB, scheduleID, jobName string) *models.Job {
	t.Helper()
	remark := rpaScheduleMarker(scheduleID) + "（W7.2 测试播种）"
	job := &models.Job{
		JobName:        jobName,
		JobGroup:       rpaScheduleJobGroup,
		InvokeTarget:   rpaScheduleInvokeTarget + ":" + scheduleID,
		CronExpression: "0 * * * * *",
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         models.JobStatusNormal,
		Remark:         &remark,
	}
	require.NoError(t, db.Create(job).Error)
	return job
}

// countLiveRPAJobs 统计 RPA 分组下**未软删**的 job 数（= 调度器启动时会加载的量）。
func countLiveRPAJobs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.Job{}).
		Where("job_group = ?", rpaScheduleJobGroup).
		Where("deleted_at IS NULL").
		Count(&n).Error)
	return n
}

// TestSyncEnabledJobs_PrunesOrphans 对账必须摘掉 3 种孤儿，只留正常那 1 条。
func TestSyncEnabledJobs_PrunesOrphans(t *testing.T) {
	db := pruneTestDB(t)
	ctx := context.Background()

	// 正常调度：启用态，且已有对应 job（模拟生产既有状态，ensure 应幂等跳过）
	alive := pruneSeedSchedule(t, db, "正常调度", rpav1.ScheduleStatusEnabled)
	aliveJob := pruneSeedJob(t, db, alive.ID, "RPA调度:正常调度")

	// 孤儿 1：schedule 被**物理删除**（绕过 API 直接 DELETE 的典型形态）
	physical := pruneSeedSchedule(t, db, "物理删除调度", rpav1.ScheduleStatusEnabled)
	pruneSeedJob(t, db, physical.ID, "RPA调度:物理删除调度")
	require.NoError(t, db.Unscoped().Delete(&rpamodels.Schedule{}, "id = ?", physical.ID).Error)

	// 孤儿 2：schedule 被**软删除**（Delete API 旧顺序留下的典型形态）
	soft := pruneSeedSchedule(t, db, "软删除调度", rpav1.ScheduleStatusEnabled)
	pruneSeedJob(t, db, soft.ID, "RPA调度:软删除调度")
	require.NoError(t, db.Delete(&rpamodels.Schedule{}, "id = ?", soft.ID).Error)

	// 孤儿 3：schedule 存在但已停用（status=1）
	disabled := pruneSeedSchedule(t, db, "停用调度", rpav1.ScheduleStatusDisabled)
	pruneSeedJob(t, db, disabled.ID, "RPA调度:停用调度")

	require.Equal(t, int64(4), countLiveRPAJobs(t, db), "前置条件：4 条 RPA job 都在")

	// 真同步器 + 真服务（与启动时同一条链路）
	sched := NewScheduler(db)
	syncer := NewRPAScheduleJobSyncer(sched, db)
	svc := rpav1.NewScheduleService(db, nil, syncer)

	require.NoError(t, svc.SyncEnabledJobs(ctx), "对账不得让同步失败")

	// 断言：只剩正常那 1 条
	assert.Equal(t, int64(1), countLiveRPAJobs(t, db),
		"3 条孤儿 job 必须被摘除，只保留正常那 1 条")

	var remaining []models.Job
	require.NoError(t, db.Model(&models.Job{}).
		Where("job_group = ?", rpaScheduleJobGroup).
		Where("deleted_at IS NULL").
		Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, aliveJob.ID, remaining[0].ID, "保留的必须是正常调度对应的 job")
	assert.Equal(t, "rpa_schedule:"+alive.ID, remaining[0].InvokeTarget)

	// 逐条核对：被摘的 3 条确实处于软删状态（不是物理删，历史可追溯）
	var softDeleted []models.Job
	require.NoError(t, db.Unscoped().Model(&models.Job{}).
		Where("job_group = ?", rpaScheduleJobGroup).
		Where("deleted_at IS NOT NULL").
		Find(&softDeleted).Error)
	assert.Len(t, softDeleted, 3, "3 条孤儿 job 应以软删方式摘除（保留审计痕迹）")

	// 幂等：再跑一次不再摘任何东西
	result, err := syncer.PruneOrphanJobs(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Pruned, "重复对账必须幂等（不再摘任何 job）")
	assert.Equal(t, int64(1), countLiveRPAJobs(t, db))
}

// TestPruneOrphanJobs_KeepsEnabledSchedule 启用态调度的 job 绝不能被摘。
func TestPruneOrphanJobs_KeepsEnabledSchedule(t *testing.T) {
	db := pruneTestDB(t)
	ctx := context.Background()

	alive := pruneSeedSchedule(t, db, "启用调度", rpav1.ScheduleStatusEnabled)
	aliveJob := pruneSeedJob(t, db, alive.ID, "RPA调度:启用调度")

	syncer := NewRPAScheduleJobSyncer(NewScheduler(db), db)
	result, err := syncer.PruneOrphanJobs(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Pruned)
	assert.Equal(t, int64(1), countLiveRPAJobs(t, db))

	var job models.Job
	require.NoError(t, db.Where("id = ?", aliveJob.ID).First(&job).Error,
		"启用态调度的 job 必须仍在（未被摘除）")
}

// TestPruneOrphanJobs_IgnoresNonDerivedJobs RPA 分组下人工建的普通任务不动。
func TestPruneOrphanJobs_IgnoresNonDerivedJobs(t *testing.T) {
	db := pruneTestDB(t)
	ctx := context.Background()

	// 人工建在 RPA 分组、invoke_target 不是 rpa_schedule:* 的任务
	remark := "运维手工建的任务"
	manual := &models.Job{
		JobName:        "人工任务",
		JobGroup:       rpaScheduleJobGroup,
		InvokeTarget:   "device_status_check",
		CronExpression: "0 * * * * *",
		Status:         models.JobStatusNormal,
		Remark:         &remark,
	}
	require.NoError(t, db.Create(manual).Error)

	syncer := NewRPAScheduleJobSyncer(NewScheduler(db), db)
	result, err := syncer.PruneOrphanJobs(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Pruned, "非派生 job 不得被当成孤儿")
	assert.Equal(t, int64(1), countLiveRPAJobs(t, db), "人工建的任务必须保留")
}
