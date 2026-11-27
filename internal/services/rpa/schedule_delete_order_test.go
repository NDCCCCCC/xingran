package rpa

// W7.2 / 任务 A-4：Delete 的「先摘 job、后软删 schedule」顺序用例。
//
// 覆盖 REQUIREMENTS 的验收句：
//   「断言顺序正确且失败可重试」。
//
// 为什么要锁这个顺序（生产实证的根因链）：
//   旧实现 = 先 db.Delete(软删 schedule) → 再 RemoveScheduleJob。
//   若第二步失败：schedule 已软删（GORM 软删自动过滤 → 正向同步再也看不到它）
//   而 job 还挂在 sys_job 上且 deleted_at IS NULL → 永久孤儿 job，
//   调度器启动会把它加载进 cron，到点报错刷日志，且**没有任何机制会回收它**。
//
// 新顺序 = 先 RemoveScheduleJob（失败则不删 schedule，可重试）→ 再软删 schedule。
// 两种失败路径都不产生孤儿；对账（prune）是最后兜底。

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// orderedSyncer 记录 job 同步调用顺序，并在「摘 job 的那一刻」回查调度是否仍存活
// （用于断言摘 job 发生在软删 schedule 之前）。
type orderedSyncer struct {
	db *gorm.DB
	// removeErr 摘 job 错误（注入失败路径）。
	removeErr error
	// removeCalls 摘 job 的调用序列。
	removeCalls []string
	// aliveAtRemove 每次摘 job 时调度是否仍未软删（true = 尚未软删）。
	aliveAtRemove []bool
	// pruneCalls 对账调用次数。
	pruneCalls int
}

func (f *orderedSyncer) EnsureScheduleJob(ctx context.Context, scheduleID, jobName, cronExpr string) error {
	return nil
}

func (f *orderedSyncer) UpdateScheduleJobCron(ctx context.Context, scheduleID, cronExpr string) error {
	return nil
}

func (f *orderedSyncer) RemoveScheduleJob(ctx context.Context, scheduleID string) error {
	f.removeCalls = append(f.removeCalls, scheduleID)

	var alive int64
	if err := f.db.Unscoped().Model(&rpamodels.Schedule{}).
		Where("id = ?", scheduleID).
		Where("deleted_at IS NULL").
		Count(&alive).Error; err == nil {
		f.aliveAtRemove = append(f.aliveAtRemove, alive > 0)
	}

	if f.removeErr != nil {
		return f.removeErr
	}
	return nil
}

func (f *orderedSyncer) PruneOrphanJobs(ctx context.Context) (ScheduleJobPruneResult, error) {
	f.pruneCalls++
	return ScheduleJobPruneResult{}, nil
}

// TestDelete_JobRemovedBeforeScheduleSoftDelete 删除顺序 + 失败可重试。
func TestDelete_JobRemovedBeforeScheduleSoftDelete(t *testing.T) {
	db := cleanup158DB(t)
	ctx := context.Background()

	syncer := &orderedSyncer{db: db, removeErr: errors.New("模拟摘 job 中途失败")}
	svc := NewScheduleService(db, nil, syncer)

	sched, err := svc.Create(ctx, &CreateScheduleRequest{
		Name: "待删调度", TaskID: "task-delete-order", CronExpr: "*/1 * * * *",
	})
	require.NoError(t, err)

	// 第一次删除：摘 job 失败 → 必须报错，且**不得**软删 schedule
	err = svc.Delete(ctx, sched.ID)
	require.Error(t, err, "摘 job 失败必须报错（否则留下永久孤儿 job）")
	assert.Contains(t, err.Error(), "可重试", "错误信息要提示可重试")

	var row rpamodels.Schedule
	require.NoError(t, db.Unscoped().Where("id = ?", sched.ID).First(&row).Error)
	assert.False(t, row.DeletedAt.Valid,
		"摘 job 失败时不得软删 schedule —— 那正是产生永久孤儿 job 的根因")

	// 重试：摘 job 成功 → 软删 schedule
	syncer.removeErr = nil
	require.NoError(t, svc.Delete(ctx, sched.ID), "重试必须成功（幂等）")

	require.NoError(t, db.Unscoped().Where("id = ?", sched.ID).First(&row).Error)
	assert.True(t, row.DeletedAt.Valid, "重试成功后 schedule 必须被软删")

	// 顺序断言：两次摘 job 时 schedule 都还活着 → 摘 job 一定发生在软删之前
	assert.Equal(t, []string{sched.ID, sched.ID}, syncer.removeCalls)
	assert.Equal(t, []bool{true, true}, syncer.aliveAtRemove,
		"摘 job 必须发生在 schedule 软删之前（先摘后删）")
}

// TestDelete_NoSyncer_StillSoftDeletes 未接调度器时（syncer=nil）删除照常生效。
func TestDelete_NoSyncer_StillSoftDeletes(t *testing.T) {
	db := cleanup158DB(t)
	ctx := context.Background()
	svc := NewScheduleService(db, nil, nil)

	sched, err := svc.Create(ctx, &CreateScheduleRequest{
		Name: "无调度器删除", TaskID: "task-nosyncer", CronExpr: "*/1 * * * *",
	})
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, sched.ID))

	var row rpamodels.Schedule
	require.NoError(t, db.Unscoped().Where("id = ?", sched.ID).First(&row).Error)
	assert.True(t, row.DeletedAt.Valid)
}
