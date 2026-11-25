package rpa

// Phase 158 / CLEANUP-01~02 服务层真值用例。
//
// 覆盖三条断言：
//  1. CLEANUP-01：批量执行报告「无数据」返回空统计、「有 3 条数据」返回真值
//     （此前恒返回硬编码 100/95/5，任何数据都返回同一组常量）。
//  2. CLEANUP-02：调度 CRUD + 到期计算（cron 表达式 → 下次执行时间）+ 启停摘 job。
//
// 真库：sqlite 内存库 + 真实 GORM AutoMigrate（模型 tag 与生产一致，
// 禁止手写 CREATE TABLE 掩盖 schema 漂移）。

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// cleanup158DB 建真 sqlite 内存库（真实 AutoMigrate）。
func cleanup158DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&rpamodels.BatchItemReport{},
		&rpamodels.Schedule{},
	), "真实 AutoMigrate 必须能建出批量报告表与调度表")
	return db
}

// TestCleanup01_BatchReport_NoData_ReturnsEmptyStats 无数据时不得返回任何常量假值。
func TestCleanup01_BatchReport_NoData_ReturnsEmptyStats(t *testing.T) {
	db := cleanup158DB(t)
	svc := NewRPAExcelService(db)
	ctx := context.Background()

	report, err := svc.GetBatchExecutionReport(ctx, "exec-empty")
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "exec-empty", report.ExecutionID)
	assert.Equal(t, 0, report.TotalItems, "无数据时总数必须是 0，不能是硬编码的 100")
	assert.Equal(t, 0, report.SuccessCount, "无数据时成功数必须是 0，不能是硬编码的 95")
	assert.Equal(t, 0, report.FailedCount, "无数据时失败数必须是 0，不能是硬编码的 5")
	assert.Equal(t, 0, report.SkippedCount)
	assert.Len(t, report.Items, 0, "无数据时明细必须为空")
	assert.Equal(t, 0, report.Summary["total"])
}

// TestCleanup01_BatchReport_ThreeItems_ReturnsRealCounts 有 3 条数据时返回真统计。
func TestCleanup01_BatchReport_ThreeItems_ReturnsRealCounts(t *testing.T) {
	db := cleanup158DB(t)
	svc := NewRPAExcelService(db)
	ctx := context.Background()

	task := &rpamodels.Task{TaskName: "批量任务"}
	task.ID = "task-batch-1"

	items := []map[string]interface{}{
		{"row": 1},
		{"row": 2},
		{"row": 3},
	}
	require.NoError(t, svc.CreateBatchExecutionReport(ctx, "exec-3", task, items))

	// 逐项落终态：2 成功 + 1 失败
	require.NoError(t, svc.UpdateBatchItemReport(ctx, "exec-3", 0, rpamodels.BatchItemStatusSuccess,
		map[string]interface{}{"out": "ok-1"}, "", ""))
	require.NoError(t, svc.UpdateBatchItemReport(ctx, "exec-3", 1, rpamodels.BatchItemStatusSuccess,
		map[string]interface{}{"out": "ok-2"}, "", ""))
	require.NoError(t, svc.UpdateBatchItemReport(ctx, "exec-3", 2, rpamodels.BatchItemStatusFailed,
		nil, "元素不存在", "shot.png"))

	report, err := svc.GetBatchExecutionReport(ctx, "exec-3")
	require.NoError(t, err)

	assert.Equal(t, 3, report.TotalItems)
	assert.Equal(t, 2, report.SuccessCount)
	assert.Equal(t, 1, report.FailedCount)
	assert.Equal(t, 0, report.SkippedCount)
	assert.Len(t, report.Items, 3)

	assert.Equal(t, "success", report.Items[0].Status)
	assert.Equal(t, "ok-1", report.Items[0].OutputData["out"])
	assert.Equal(t, float64(1), report.Items[0].InputData["row"])

	assert.Equal(t, "failed", report.Items[2].Status)
	assert.Equal(t, "元素不存在", report.Items[2].Error)
	assert.Equal(t, "shot.png", report.Items[2].Screenshot)

	// 汇总口径与明细一致
	assert.Equal(t, 3, report.Summary["total"])
	assert.Equal(t, 2, report.Summary["success"])
	assert.Equal(t, 1, report.Summary["failed"])
	assert.Equal(t, 0, report.Summary["pending"])
}

// TestCleanup01_BatchReport_UpdateUnknownItem_ReturnsError 未落库的项不得被静默忽略。
func TestCleanup01_BatchReport_UpdateUnknownItem_ReturnsError(t *testing.T) {
	db := cleanup158DB(t)
	svc := NewRPAExcelService(db)

	err := svc.UpdateBatchItemReport(context.Background(), "exec-none", 0, rpamodels.BatchItemStatusSuccess, nil, "", "")
	require.Error(t, err, "更新不存在的批量项必须报错（不能静默吞掉）")
}

// fakeScheduleJobSyncer 记录 job 同步调用（ScheduleJobSyncer 接口替身）。
type fakeScheduleJobSyncer struct {
	ensured  []string
	updated  []string
	removed  []string
	cronExpr string
}

func (f *fakeScheduleJobSyncer) EnsureScheduleJob(ctx context.Context, scheduleID, jobName, cronExpr string) error {
	f.ensured = append(f.ensured, scheduleID)
	f.cronExpr = cronExpr
	return nil
}

func (f *fakeScheduleJobSyncer) UpdateScheduleJobCron(ctx context.Context, scheduleID, cronExpr string) error {
	f.updated = append(f.updated, scheduleID)
	f.cronExpr = cronExpr
	return nil
}

func (f *fakeScheduleJobSyncer) RemoveScheduleJob(ctx context.Context, scheduleID string) error {
	f.removed = append(f.removed, scheduleID)
	return nil
}

// TestCleanup02_Schedule_CRUD_Lifecycle 调度 CRUD + 启停摘 job 全链路。
func TestCleanup02_Schedule_CRUD_Lifecycle(t *testing.T) {
	db := cleanup158DB(t)
	syncer := &fakeScheduleJobSyncer{}
	svc := NewScheduleService(db, nil, syncer)
	ctx := context.Background()

	// 创建：cron 表达式非法必须报错（不静默接受）
	_, err := svc.Create(ctx, &CreateScheduleRequest{
		Name: "坏 cron", TaskID: "task-1", CronExpr: "not-a-cron",
	})
	require.Error(t, err, "非法 cron 表达式必须被拒")

	// 创建：正常 cron
	sched, err := svc.Create(ctx, &CreateScheduleRequest{
		Name: "每分钟调度", TaskID: "task-1", CronExpr: "*/1 * * * *",
		Description: "CLEANUP-02 用例",
	})
	require.NoError(t, err)
	require.NotNil(t, sched)
	assert.NotEmpty(t, sched.ID)
	assert.Equal(t, ScheduleStatusEnabled, sched.Status)
	assert.NotNil(t, sched.NextRunTime, "创建后必须算出下次执行时间")
	require.Len(t, syncer.ensured, 1, "启用态创建必须注册调度 job")
	assert.Equal(t, sched.ID, syncer.ensured[0])

	// 查询
	got, err := svc.GetByID(ctx, sched.ID)
	require.NoError(t, err)
	assert.Equal(t, "每分钟调度", got.Name)
	assert.Equal(t, "*/1 * * * *", got.CronExpr)

	// 列表
	listParams := &ScheduleListParams{}
	listParams.Current = 1
	listParams.PageSize = 10
	list, err := svc.List(ctx, listParams)
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)

	// 改 cron → job 同步更新
	require.NoError(t, svc.Update(ctx, &UpdateScheduleRequest{ID: sched.ID, CronExpr: "*/2 * * * *"}))
	require.Len(t, syncer.updated, 1)
	assert.Equal(t, sched.ID, syncer.updated[0])

	// 停用 → 摘除 job
	require.NoError(t, svc.SetStatus(ctx, sched.ID, ScheduleStatusDisabled))
	require.Len(t, syncer.removed, 1, "停用必须摘除调度 job")
	assert.Equal(t, sched.ID, syncer.removed[0])

	got, err = svc.GetByID(ctx, sched.ID)
	require.NoError(t, err)
	assert.Equal(t, ScheduleStatusDisabled, got.Status)

	// 删除 → 再摘一次（幂等）
	require.NoError(t, svc.Delete(ctx, sched.ID))
	assert.Len(t, syncer.removed, 2)
}

// TestCleanup02_Schedule_ComputeNextRunTime 到期计算（5 位与 6 位表达式）。
func TestCleanup02_Schedule_ComputeNextRunTime(t *testing.T) {
	db := cleanup158DB(t)
	svc := NewScheduleService(db, nil, nil)
	from := time.Date(2026, 11, 25, 10, 30, 0, 0, time.UTC)

	// 5 位表达式（分 时 日 月 周）→ 下一分钟
	next5, err := svc.ComputeNextRunTime("*/1 * * * *", "Asia/Shanghai", from)
	require.NoError(t, err)
	require.NotNil(t, next5)
	assert.True(t, next5.After(from), "下次执行时间必须晚于基准时刻")

	// 6 位表达式（秒 分 时 日 月 周）→ 秒级精度
	next6, err := svc.ComputeNextRunTime("0 * * * * *", "Asia/Shanghai", from)
	require.NoError(t, err)
	require.NotNil(t, next6)
	assert.True(t, next6.After(from))

	// 非法表达式
	_, err = svc.ComputeNextRunTime("@@@", "Asia/Shanghai", from)
	require.Error(t, err)

	// 非法时区
	_, err = svc.ComputeNextRunTime("*/1 * * * *", "Not/AZone", from)
	require.Error(t, err)
}

// TestCleanup02_Schedule_TriggerNow_RejectsDisabled 停用态不得触发。
func TestCleanup02_Schedule_TriggerNow_RejectsDisabled(t *testing.T) {
	db := cleanup158DB(t)
	svc := NewScheduleService(db, nil, &fakeScheduleJobSyncer{})
	ctx := context.Background()

	sched, err := svc.Create(ctx, &CreateScheduleRequest{
		Name: "停用调度", TaskID: "task-x", CronExpr: "*/1 * * * *",
	})
	require.NoError(t, err)
	require.NoError(t, svc.SetStatus(ctx, sched.ID, ScheduleStatusDisabled))

	_, err = svc.TriggerNow(ctx, sched.ID)
	require.Error(t, err, "停用态调度必须拒绝触发")
}
