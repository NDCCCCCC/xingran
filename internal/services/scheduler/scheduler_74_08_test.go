package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/common"
)

// =====================================================================
// 74-08 Batch B: internal/services/scheduler — JobService CRUD/状态/
// 执行 + JobLogService Create/List/CleanOldLogs(Statistics 已有
// job_log_statistics_test.go 覆盖)。sqlite 内存库 + mock SchedulerClient。
// =====================================================================

// mockSchedulerClient 记录调用并可注入错误。
type mockSchedulerClient struct {
	registered map[string]bool

	addErr    error
	updateErr error
	removeErr error
	startErr  error
	stopErr   error
	execErr   error

	added    []string
	updated  []string
	removed  []string
	started  []string
	stopped  []string
	executed []string
}

func (m *mockSchedulerClient) AddJob(job *models.Job) error {
	m.added = append(m.added, job.ID)
	return m.addErr
}
func (m *mockSchedulerClient) UpdateJob(job *models.Job) error {
	m.updated = append(m.updated, job.ID)
	return m.updateErr
}
func (m *mockSchedulerClient) RemoveJob(id string) error {
	m.removed = append(m.removed, id)
	return m.removeErr
}
func (m *mockSchedulerClient) StartJob(id string) error {
	m.started = append(m.started, id)
	return m.startErr
}
func (m *mockSchedulerClient) StopJob(id string) error {
	m.stopped = append(m.stopped, id)
	return m.stopErr
}
func (m *mockSchedulerClient) ExecuteJob(id string) error {
	m.executed = append(m.executed, id)
	return m.execErr
}
func (m *mockSchedulerClient) IsTaskRegistered(taskType string) bool {
	return m.registered[taskType]
}

// newJobTestDB 建 sys_job / sys_job_log 全列表(Job 带软删除,JobLog 硬删除)。
func newJobTestDB(t *testing.T, suffix ...string) *gorm.DB {
	t.Helper()
	name := t.Name()
	if len(suffix) > 0 {
		name += suffix[0]
	}
	db, err := gorm.Open(sqlite.Open("file:sched_"+name+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_job (
		id TEXT PRIMARY KEY,
		job_name TEXT, job_group TEXT, invoke_target TEXT, cron_expression TEXT,
		misfire_policy INTEGER DEFAULT 0, concurrent BOOLEAN DEFAULT 0, status INTEGER DEFAULT 0,
		next_run_time DATETIME, prev_run_time DATETIME, remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_job_log (
		id TEXT PRIMARY KEY,
		job_name TEXT, job_group TEXT, invoke_target TEXT, job_message TEXT,
		status INTEGER DEFAULT 0, exception_info TEXT,
		start_time DATETIME, end_time DATETIME, duration INTEGER DEFAULT 0,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	return db
}

func boolPtrSched(b bool) *bool { return &b }

// ---------------- JobService.Create ----------------

func TestJobService_Create(t *testing.T) {
	ctx := context.Background()

	// 成功(nil scheduler → 跳过白名单与 AddJob)
	db := newJobTestDB(t)
	svc := NewJobService(db, nil)
	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j1", JobGroup: "g1", InvokeTarget: "noop", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.ID)
	assert.Equal(t, models.JobStatusNormal, job.Status, "默认正常状态")

	// 重名 → 拒绝
	_, err = svc.Create(ctx, &JobCreateRequest{
		JobName: "j1", JobGroup: "g1", InvokeTarget: "noop", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "已存在")

	// 不同组同名 → 允许
	_, err = svc.Create(ctx, &JobCreateRequest{
		JobName: "j1", JobGroup: "g2", InvokeTarget: "noop", CronExpression: "0 0 * * * *",
	})
	assert.NoError(t, err)

	// 非法 cron → 拒绝
	_, err = svc.Create(ctx, &JobCreateRequest{
		JobName: "j2", JobGroup: "g1", InvokeTarget: "noop", CronExpression: "not-a-cron",
	})
	assert.ErrorContains(t, err, "Cron")

	// 白名单: 未注册 taskType → 拒绝
	db2 := newJobTestDB(t, "_2")
	mock := &mockSchedulerClient{registered: map[string]bool{"cleanup": true}}
	svc2 := NewJobService(db2, mock)
	_, err = svc2.Create(ctx, &JobCreateRequest{
		JobName: "j3", JobGroup: "g1", InvokeTarget: "evil:rm -rf", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "未在调度器注册")

	// 注册 taskType(带参数) → 通过且 AddJob 被调
	job2, err := svc2.Create(ctx, &JobCreateRequest{
		JobName: "j4", JobGroup: "g1", InvokeTarget: "cleanup:days=7", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{job2.ID}, mock.added)

	// AddJob 失败 → 返回错误
	mock.addErr = errors.New("scheduler down")
	_, err = svc2.Create(ctx, &JobCreateRequest{
		JobName: "j5", JobGroup: "g1", InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "添加任务到调度器失败")
}

// ---------------- JobService.Update / GetByID ----------------

func TestJobService_Update(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	mock := &mockSchedulerClient{registered: map[string]bool{"cleanup": true, "sync": true}}
	svc := NewJobService(db, mock)

	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j1", JobGroup: "g1", InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)

	// 不存在 → 错误
	err = svc.Update(ctx, &JobUpdateRequest{ID: "no-such-id"})
	assert.ErrorContains(t, err, "不存在")

	// 正常更新(status=Normal → UpdateJob 被调)
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: job.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "sync:full", CronExpression: "0 */5 * * * *",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{job.ID}, mock.updated)

	got, err := svc.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, "j1-new", got.JobName)
	assert.Equal(t, "sync:full", got.InvokeTarget)

	// 重名(与其他任务) → 拒绝
	other, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j2", JobGroup: "g1", InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: other.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "已存在")

	// 非法 cron → 拒绝
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: job.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "cleanup", CronExpression: "bad",
	})
	assert.ErrorContains(t, err, "Cron")

	// 未注册 taskType → 拒绝
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: job.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "evil", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "未在调度器注册")

	// 暂停状态更新 → 不调 UpdateJob
	require.NoError(t, db.Exec(`UPDATE sys_job SET status = 1 WHERE id = ?`, job.ID).Error)
	before := len(mock.updated)
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: job.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	assert.Len(t, mock.updated, before, "暂停任务不同步调度器")

	// UpdateJob 失败 → 返回错误
	require.NoError(t, db.Exec(`UPDATE sys_job SET status = 0 WHERE id = ?`, job.ID).Error)
	mock.updateErr = errors.New("update fail")
	err = svc.Update(ctx, &JobUpdateRequest{
		ID: job.ID, JobName: "j1-new", JobGroup: "g1",
		InvokeTarget: "cleanup", CronExpression: "0 0 * * * *",
	})
	assert.ErrorContains(t, err, "更新调度器")
}

func TestJobService_GetByID(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	svc := NewJobService(db, nil)

	_, err := svc.GetByID(ctx, "missing")
	assert.ErrorContains(t, err, "不存在")

	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j", JobGroup: "g", InvokeTarget: "x", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	got, err := svc.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, "j", got.JobName)
}

// ---------------- JobService.Delete ----------------

func TestJobService_Delete(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	mock := &mockSchedulerClient{registered: map[string]bool{"x": true}}
	svc := NewJobService(db, mock)

	err := svc.Delete(ctx, "missing")
	assert.ErrorContains(t, err, "不存在")

	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j1", JobGroup: "g1", InvokeTarget: "x", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	// 关联日志 2 条 + 其他任务日志 1 条
	for _, name := range []string{"j1", "j1", "j2"} {
		require.NoError(t, NewJobLogService(db).Create(ctx, &models.JobLog{
			JobName: name, JobGroup: "g1", InvokeTarget: "x",
		}))
	}

	require.NoError(t, svc.Delete(ctx, job.ID))
	assert.Equal(t, []string{job.ID}, mock.removed, "调度器移除被调")

	// 软删除: GetByID 查不到
	_, err = svc.GetByID(ctx, job.ID)
	assert.Error(t, err)

	// j1 日志被清,j2 保留
	var cnt int64
	require.NoError(t, db.Model(&models.JobLog{}).Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt, "仅删除被删任务的日志")
}

// ---------------- JobService.List ----------------

func TestJobService_List(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	svc := NewJobService(db, nil)

	seed := []struct{ name, group string; status int }{
		{"alpha-job", "g1", 0},
		{"beta-job", "g1", 1},
		{"gamma", "g2", 0},
	}
	for _, s := range seed {
		_, err := svc.Create(ctx, &JobCreateRequest{
			JobName: s.name, JobGroup: s.group, InvokeTarget: "x", CronExpression: "0 0 * * * *",
		})
		require.NoError(t, err)
		if s.status != 0 {
			require.NoError(t, db.Exec(`UPDATE sys_job SET status = ? WHERE job_name = ?`, s.status, s.name).Error)
		}
	}

	// 全量 + 默认分页(非法分页参数回退 1/10)
	res, err := svc.List(ctx, &JobListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
	assert.Equal(t, 1, res.Current)
	assert.Equal(t, 10, res.PageSize)

	// jobName 模糊
	res, err = svc.List(ctx, &JobListParams{JobName: "job"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// jobGroup 精确
	res, err = svc.List(ctx, &JobListParams{JobGroup: "g2"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// status 过滤
	st := 1
	res, err = svc.List(ctx, &JobListParams{Status: &st})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// 白名单排序: jobName 升序
	res, err = svc.List(ctx, &JobListParams{
		ListParams: common.ListParams{BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "jobName", IsAsc: boolPtrSched(true),
		}},
	})
	require.NoError(t, err)
	jobs := res.List.([]models.Job)
	require.Len(t, jobs, 3)
	assert.Equal(t, "alpha-job", jobs[0].JobName)
	assert.Equal(t, "gamma", jobs[2].JobName)

	// 非白名单列 → 忽略排序,不报错
	_, err = svc.List(ctx, &JobListParams{
		ListParams: common.ListParams{BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "evil;drop",
		}},
	})
	assert.NoError(t, err)
}

// ---------------- JobService.UpdateStatus / Execute ----------------

func TestJobService_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	mock := &mockSchedulerClient{registered: map[string]bool{"x": true}}
	svc := NewJobService(db, mock)

	err := svc.UpdateStatus(ctx, "missing", 0)
	assert.ErrorContains(t, err, "不存在")

	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j", JobGroup: "g", InvokeTarget: "x", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)

	// 启用 → StartJob
	require.NoError(t, svc.UpdateStatus(ctx, job.ID, 0))
	assert.Equal(t, []string{job.ID}, mock.started)

	// 停用 → StopJob
	require.NoError(t, svc.UpdateStatus(ctx, job.ID, 1))
	assert.Equal(t, []string{job.ID}, mock.stopped)

	// StartJob 失败 → 错误
	mock.startErr = errors.New("start fail")
	err = svc.UpdateStatus(ctx, job.ID, 0)
	assert.ErrorContains(t, err, "启动任务失败")

	// StopJob 失败 → 错误
	mock.stopErr = errors.New("stop fail")
	err = svc.UpdateStatus(ctx, job.ID, 1)
	assert.ErrorContains(t, err, "停止任务失败")
}

func TestJobService_Execute(t *testing.T) {
	ctx := context.Background()

	// 不存在
	db := newJobTestDB(t)
	svc := NewJobService(db, nil)
	assert.Error(t, svc.Execute(ctx, "missing"))

	// nil scheduler → 本地写一条成功日志
	job, err := svc.Create(ctx, &JobCreateRequest{
		JobName: "j", JobGroup: "g", InvokeTarget: "x", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	require.NoError(t, svc.Execute(ctx, job.ID))

	var logs []models.JobLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, "任务执行成功(调度器未启动)", logs[0].JobMessage)
	assert.Equal(t, int(models.JobLogStatusSuccess), logs[0].Status)
	assert.NotNil(t, logs[0].StartTime)
	assert.NotNil(t, logs[0].EndTime)

	// 有 scheduler → ExecuteJob 被调
	db2 := newJobTestDB(t, "_2")
	mock := &mockSchedulerClient{registered: map[string]bool{"x": true}}
	svc2 := NewJobService(db2, mock)
	job2, err := svc2.Create(ctx, &JobCreateRequest{
		JobName: "j", JobGroup: "g", InvokeTarget: "x", CronExpression: "0 0 * * * *",
	})
	require.NoError(t, err)
	require.NoError(t, svc2.Execute(ctx, job2.ID))
	assert.Equal(t, []string{job2.ID}, mock.executed)

	// ExecuteJob 失败 → 错误
	mock.execErr = errors.New("exec fail")
	err = svc2.Execute(ctx, job2.ID)
	assert.ErrorContains(t, err, "执行任务失败")
}

// ---------------- JobLogService ----------------

func TestJobLogService_CreateAndList(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	svc := NewJobLogService(db)

	// Create
	st1 := time.Now().Add(-time.Hour)
	st2 := time.Now()
	require.NoError(t, svc.Create(ctx, &models.JobLog{
		JobName: "alpha", JobGroup: "g1", InvokeTarget: "x",
		Status: int(models.JobLogStatusSuccess), StartTime: &st1,
	}))
	require.NoError(t, svc.Create(ctx, &models.JobLog{
		JobName: "beta", JobGroup: "g2", InvokeTarget: "x",
		Status: int(models.JobLogStatusFailure), StartTime: &st2,
	}))

	// 全量
	res, err := svc.List(ctx, &JobLogListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// jobName 模糊 + jobGroup 精确 + status
	fail := int(models.JobLogStatusFailure)
	res, err = svc.List(ctx, &JobLogListParams{JobName: "a", Status: &fail})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total, "beta 命中(a 模糊 + 失败状态)")

	res, err = svc.List(ctx, &JobLogListParams{JobGroup: "g1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// 时间范围(合法格式过滤;非法格式被忽略)
	start := st1.Add(-time.Minute).Format("2006-01-02 15:04:05")
	end := st1.Add(time.Minute).Format("2006-01-02 15:04:05")
	res, err = svc.List(ctx, &JobLogListParams{StartTime: &start, EndTime: &end})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total, "只命中 st1 附近的 alpha")

	bad := "not-a-time"
	res, err = svc.List(ctx, &JobLogListParams{StartTime: &bad, EndTime: &bad})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "非法时间格式被忽略")

	// 排序白名单
	res, err = svc.List(ctx, &JobLogListParams{
		ListParams: common.ListParams{BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "jobName", IsAsc: boolPtrSched(true),
		}},
	})
	require.NoError(t, err)
	logs := res.List.([]models.JobLog)
	require.Len(t, logs, 2)
	assert.Equal(t, "alpha", logs[0].JobName)
}

func TestJobLogService_Statistics_Empty(t *testing.T) {
	db := newJobTestDB(t)
	svc := NewJobLogService(db)
	res, err := svc.Statistics(context.Background(), &JobLogListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total)
}

func TestJobLogService_CleanOldLogs(t *testing.T) {
	ctx := context.Background()
	db := newJobTestDB(t)
	svc := NewJobLogService(db)

	// days<=0 → 拒绝
	assert.ErrorContains(t, svc.CleanOldLogs(ctx, 0), "天数必须大于0")
	assert.Error(t, svc.CleanOldLogs(ctx, -1))

	// 1 条旧日志(40 天前) + 1 条新日志
	require.NoError(t, svc.Create(ctx, &models.JobLog{JobName: "old", JobGroup: "g", InvokeTarget: "x"}))
	require.NoError(t, svc.Create(ctx, &models.JobLog{JobName: "new", JobGroup: "g", InvokeTarget: "x"}))
	old := time.Now().AddDate(0, 0, -40)
	require.NoError(t, db.Exec(`UPDATE sys_job_log SET created_at = ? WHERE job_name = 'old'`, old).Error)

	require.NoError(t, svc.CleanOldLogs(ctx, 30))
	var names []string
	require.NoError(t, db.Model(&models.JobLog{}).Pluck("job_name", &names).Error)
	assert.Equal(t, []string{"new"}, names, "仅保留 30 天内日志")
}
