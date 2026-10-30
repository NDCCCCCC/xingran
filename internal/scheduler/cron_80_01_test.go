package scheduler

// Phase 80 Plan 01 — internal/scheduler 引擎主体 cron.go 全量测试
// (任务执行器 + Scheduler 公开方法 + 注册表 + defaultLogger + 三组全局 var seams)
//
// 断言口径(D-80-02): 零 cron 触发时序断言 —— 直调 JobExecutor.Execute /
// Scheduler.ExecuteJob / 公开方法;Start/Stop 生命周期用例不 sleep、不等实时时钟。
// "引擎并发"由 GetJobCount/GetJobStatus 并发读覆盖(引用既有 cron_test.go 并发范式)。
//
// 纪律:
//   - 凡 Start 必须 t.Cleanup(s.Stop);不测 Start 的用例一律不起 cron
//   - 任务 handler 体内绝不调 Scheduler 方法(handler 只做纯计数,规避 Stop 持锁互斥)
//   - 全文件禁并行子测试(全局 var seams + 包级注册表,串行执行)
//   - status 断言只引用 models.* 常量,禁裸 0/1 字面量(CLAUDE.md Status Value Convention)
//   - helper 一律 8001 后缀(D-80-07,防同包既有 helper 撞名)

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// 引擎 fixture(80-02 六个 *_tasks.go 测试直接复用)
// ============================================================================

// newSchedDB8001 构造 sqlite 文件库(t.TempDir,防跨用例串数据)+
// sys_job / sys_job_log 两表 AutoMigrate;t.Cleanup 关连接。
func newSchedDB8001(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "sched8001.db")
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

// schedStubLogger8001 可观察 Logger stub —— 收集三法调用供断言;mutex 保护(-race)。
type schedStubLogger8001 struct {
	mu     sync.Mutex
	infof  []string
	warnf  []string
	errorf []string
}

func (l *schedStubLogger8001) Infof(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infof = append(l.infof, fmt.Sprintf(format, args...))
}

func (l *schedStubLogger8001) Warnf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnf = append(l.warnf, fmt.Sprintf(format, args...))
}

func (l *schedStubLogger8001) Errorf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorf = append(l.errorf, fmt.Sprintf(format, args...))
}

// counts 返回三法各自的调用次数(snapshot)。
func (l *schedStubLogger8001) counts() (info, warn, err int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.infof), len(l.warnf), len(l.errorf)
}

// newScheduler8001 newSchedDB8001 + NewScheduler 装配,SetLogger 注入可观察 stub。
// 取回 stub 用 stubLoggerOf8001(s)。
func newScheduler8001(t *testing.T) (*Scheduler, *gorm.DB) {
	t.Helper()
	db := newSchedDB8001(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	return s, db
}

// stubLoggerOf8001 从 scheduler 取回 fixture 注入的 stub Logger(同包读 unexported 字段)。
func stubLoggerOf8001(t *testing.T, s *Scheduler) *schedStubLogger8001 {
	t.Helper()
	stub, ok := s.logger.(*schedStubLogger8001)
	require.True(t, ok, "scheduler 未注入 schedStubLogger8001")
	return stub
}

// newJob8001 构造一条合法 job(默认正常状态 + 每日 08:00 cron)。
func newJob8001(name, invokeTarget string) *models.Job {
	return &models.Job{
		JobName:        name,
		JobGroup:       "DEFAULT",
		InvokeTarget:   invokeTarget,
		CronExpression: "0 0 8 * * *",
		Status:         models.JobStatusNormal,
	}
}

// registerCountingTask8001 注册一个纯计数 handler(绝不调 Scheduler 方法,R2 纪律),
// 返回计数器指针供断言。
func registerCountingTask8001(t *testing.T, s *Scheduler, taskType string) *int32 {
	t.Helper()
	var counter int32
	s.RegisterTask(taskType, func(ctx context.Context, params map[string]any) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})
	return &counter
}

// ============================================================================
// Task 1 — 注册表(RegisterTask / GetTaskHandler / IsTaskRegistered)+ Register* 函数群
// ============================================================================

// TestReg8001_RegisterLookup 注册 → 查得到 handler;未注册类型 → miss。
func TestReg8001_RegisterLookup(t *testing.T) {
	s, _ := newScheduler8001(t)

	// 未注册:miss 两连
	assert.False(t, s.IsTaskRegistered("reg8001_typeA"))
	assert.Nil(t, s.GetTaskHandler("reg8001_typeA"))

	// 注册后:hit
	counter := registerCountingTask8001(t, s, "reg8001_typeA")
	assert.True(t, s.IsTaskRegistered("reg8001_typeA"))
	handler := s.GetTaskHandler("reg8001_typeA")
	require.NotNil(t, handler)

	// handler 可调且只累加计数器(纯函数,不碰 Scheduler)
	require.NoError(t, handler(context.Background(), nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(counter))
}

// TestReg8001_ReregisterOverwrites 同 taskType 二次注册覆盖旧 handler。
func TestReg8001_ReregisterOverwrites(t *testing.T) {
	s, _ := newScheduler8001(t)

	oldCounter := registerCountingTask8001(t, s, "reg8001_dup")
	newCounter := registerCountingTask8001(t, s, "reg8001_dup")

	handler := s.GetTaskHandler("reg8001_dup")
	require.NotNil(t, handler)
	require.NoError(t, handler(context.Background(), nil))

	// 只累加新计数器 ⇒ 注册表按 taskType 覆盖
	assert.Equal(t, int32(0), atomic.LoadInt32(oldCounter), "旧 handler 应被覆盖")
	assert.Equal(t, int32(1), atomic.LoadInt32(newCounter), "新 handler 生效")
}

// TestReg8001_RegisterNoticeTasks cron.go 内全部 Register* 函数群逐一调用:
// RegisterNoticeTasks(通知/值班 2 项)+ RegisterNetworkDeviceTasks(设备 5 项)。
// 只断言注册成功,不执行 handler 体(notice/duty/device 的 wire 依赖是 80-02 范围)。
func TestReg8001_RegisterNoticeTasks(t *testing.T) {
	s, db := newScheduler8001(t)

	RegisterNoticeTasks(s)
	RegisterNetworkDeviceTasks(s, db)

	// cron.go 注册函数群注册的全部 taskType(逐一列名)
	registered := []string{
		"notice_publish",       // RegisterNoticeTasks
		"duty_reminder",        // RegisterNoticeTasks
		"device_status_check",  // RegisterNetworkDeviceTasks
		"device_info_update",   // RegisterNetworkDeviceTasks
		"port_collection",      // RegisterNetworkDeviceTasks
		"mac_collection",       // RegisterNetworkDeviceTasks
		"config_backup",        // RegisterNetworkDeviceTasks
	}
	for _, taskType := range registered {
		assert.True(t, s.IsTaskRegistered(taskType), "taskType %s 应已注册", taskType)
		assert.NotNil(t, s.GetTaskHandler(taskType), "taskType %s handler 应非 nil", taskType)
	}
}

// TestReg8001_GetJobCount_ConcurrentRead 预置 3 个 job 行(1 正常 2 暂停),
// 10 goroutine 并发 GetJobCount + GetJobStatus(DQ2-a 引擎并发口径,引用 cron_test.go 范式)。
func TestReg8001_GetJobCount_ConcurrentRead(t *testing.T) {
	s, _ := newScheduler8001(t)

	paused := newJob8001("并发读-暂停1", "reg8001_conc_a")
	paused.Status = models.JobStatusPause
	require.NoError(t, s.AddJob(paused))
	paused2 := newJob8001("并发读-暂停2", "reg8001_conc_b")
	paused2.Status = models.JobStatusPause
	require.NoError(t, s.AddJob(paused2))
	normal := newJob8001("并发读-正常", "reg8001_conc_c")
	require.NoError(t, s.AddJob(normal))

	jobIDs := []string{paused.ID, paused2.ID, normal.ID}
	require.Len(t, jobIDs, 3)

	// !running 调度器下:AddJob 只落库+内存 map,executors 为空
	total, running := s.GetJobCount()
	require.Equal(t, 3, total)
	require.Equal(t, 0, running)

	const goroutines = 10
	done := make(chan struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			gotTotal, gotRunning := s.GetJobCount()
			assert.Equal(t, 3, gotTotal)
			assert.Equal(t, 0, gotRunning)
			for _, id := range jobIDs {
				exists, err := s.GetJobStatus(id)
				assert.NoError(t, err)
				assert.False(t, exists, "!running 下 executors 为空")
			}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// 并发读后状态不变
	total, running = s.GetJobCount()
	assert.Equal(t, 3, total)
	assert.Equal(t, 0, running)
}

// ============================================================================
// Task 2 — JobExecutor 全链(Execute/executeTask 分发 + 纯函数 + defaultLogger)
// ============================================================================

// errSentinel8001 handler 失败 sentinel(错误路径断言锚点)。
var errSentinel8001 = errors.New("reg8001 处理器故障")

// newExecutor8001 直调路径用 JobExecutor 装配(scheduler 必须非 nil ——
// Execute 会经 calculateNextRunTime 解引用 scheduler.cron.Location())。
func newExecutor8001(t *testing.T, s *Scheduler, db *gorm.DB, job *models.Job) *JobExecutor {
	return &JobExecutor{job: job, db: db, logger: stubLoggerOf8001(t, s), scheduler: s}
}

// lastJobLog8001 取该 job 名下唯一一条执行日志。
func lastJobLog8001(t *testing.T, db *gorm.DB, jobName string) *models.JobLog {
	t.Helper()
	var log models.JobLog
	require.NoError(t, db.Where("job_name = ?", jobName).Order("start_time DESC").First(&log).Error)
	return &log
}

// TestExec8001_Execute_SuccessPath 直调 Execute:handler 计数 +1、
// 成功日志落库、job 行 prev/next_run_time 回写。
func TestExec8001_Execute_SuccessPath(t *testing.T) {
	s, db := newScheduler8001(t)
	counter := registerCountingTask8001(t, s, "reg8001_ok")

	job := newJob8001("执行器-成功", "reg8001_ok")
	require.NoError(t, db.Create(job).Error)

	err := newExecutor8001(t, s, db, job).Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(counter), "注册 handler 应被分发调用一次")

	// 日志:成功态 + 时间区间非空
	log := lastJobLog8001(t, db, job.JobName)
	assert.Equal(t, int(models.JobLogStatusSuccess), log.Status)
	assert.NotNil(t, log.StartTime)
	assert.NotNil(t, log.EndTime)
	assert.GreaterOrEqual(t, log.Duration, int64(0))
	assert.Equal(t, "任务执行成功", log.JobMessage)
	assert.Nil(t, log.ExceptionInfo)

	// job 行执行时间回写(合法 cron ⇒ next_run_time 非 nil)
	var updated models.Job
	require.NoError(t, db.First(&updated, "id = ?", job.ID).Error)
	assert.NotNil(t, updated.PrevRunTime)
	assert.NotNil(t, updated.NextRunTime)
	if updated.NextRunTime != nil {
		assert.True(t, updated.NextRunTime.After(time.Now().Add(-time.Minute)),
			"next_run_time 应在未来")
	}

	// 成功路径走 Infof,不走 Errorf
	stub := stubLoggerOf8001(t, s)
	info, warn, errCalls := stub.counts()
	assert.GreaterOrEqual(t, info, 1)
	assert.Equal(t, 0, warn)
	assert.Equal(t, 0, errCalls)
}

// TestExec8001_Execute_HandlerError handler 返回 sentinel ⇒ Execute 透传该 error,
// 日志落失败态且 ExceptionInfo 携带 sentinel 文案。
func TestExec8001_Execute_HandlerError(t *testing.T) {
	s, db := newScheduler8001(t)
	s.RegisterTask("reg8001_boom", func(ctx context.Context, params map[string]any) error {
		return errSentinel8001
	})

	job := newJob8001("执行器-失败", "reg8001_boom")
	require.NoError(t, db.Create(job).Error)

	err := newExecutor8001(t, s, db, job).Execute(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, errSentinel8001)

	log := lastJobLog8001(t, db, job.JobName)
	assert.Equal(t, int(models.JobLogStatusFailure), log.Status)
	require.NotNil(t, log.ExceptionInfo)
	assert.Contains(t, *log.ExceptionInfo, errSentinel8001.Error())
	assert.Contains(t, log.JobMessage, errSentinel8001.Error())

	// 失败路径走 Errorf
	stub := stubLoggerOf8001(t, s)
	_, _, errCalls := stub.counts()
	assert.GreaterOrEqual(t, errCalls, 1)
}

// TestExec8001_ExecuteTask_UnregisteredTarget 注册表 miss 分支 + nil-scheduler 分支。
func TestExec8001_ExecuteTask_UnregisteredTarget(t *testing.T) {
	s, db := newScheduler8001(t)

	// 未注册 taskType → miss
	job := newJob8001("执行器-未注册", "reg8001_missing_task")
	require.NoError(t, db.Create(job).Error)
	executor := newExecutor8001(t, s, db, job)
	err := executor.executeTask(context.Background(), &models.JobLog{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到任务处理器")

	// scheduler 引用为 nil → 显式错误(非 panic)
	orphan := &JobExecutor{job: job, db: db, logger: stubLoggerOf8001(t, s), scheduler: nil}
	err = orphan.executeTask(context.Background(), &models.JobLog{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "调度器引用为空")
}

// TestExec8001_ParseInvokeTarget_Table 表驱动覆盖 parseInvokeTarget 全部分支。
//
// quirk 记录(D-78-10 口径,零生产改动): plan <interfaces> 把该函数描述为
// `type:{"k":v}` 的 JSON 解析;实际实现(cron.go parseInvokeTarget)是"首个冒号
// 一次性切分",冒号后的整段原文作为 params["param"] 字符串,不做 JSON 解析、
// 坏 JSON 也不报错 —— 与源码 docstring"统一通过 params[\"param\"] 获取参数值"
// 一致,故按实际行为断言(坏 JSON 的 params 非 nil,含原文)。
func TestExec8001_ParseInvokeTarget_Table(t *testing.T) {
	s, db := newScheduler8001(t)
	executor := newExecutor8001(t, s, db, newJob8001("纯函数-解析", "reg8001_parse"))

	cases := []struct {
		name         string
		in           string
		wantType     string
		wantParams   map[string]any
		wantNilParam bool
	}{
		{name: "无冒号_无参数", in: "reg8001_typeA", wantType: "reg8001_typeA", wantNilParam: true},
		{
			name:       "带冒号_单参数原文",
			in:         `reg8001_typeB:{"k":1}`,
			wantType:   "reg8001_typeB",
			wantParams: map[string]any{"param": `{"k":1}`},
		},
		{
			name:       "坏JSON_原文透传不报错",
			in:         "reg8001_typeC:{bad json",
			wantType:   "reg8001_typeC",
			wantParams: map[string]any{"param": "{bad json"},
		},
		{name: "空串", in: "", wantType: "", wantNilParam: true},
		{
			name:       "多冒号_首个切分",
			in:         "reg8001_notice:abc:def",
			wantType:   "reg8001_notice",
			wantParams: map[string]any{"param": "abc:def"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskType, params := executor.parseInvokeTarget(tc.in)
			assert.Equal(t, tc.wantType, taskType)
			if tc.wantNilParam {
				assert.Nil(t, params)
				return
			}
			require.NotNil(t, params)
			assert.Equal(t, tc.wantParams, params)
		})
	}
}

// TestExec8001_CalculateNextRunTime 空/非法/合法 cron 三分支(调用个位数,
// 临时 cron 有 Stop 不泄漏)。
func TestExec8001_CalculateNextRunTime(t *testing.T) {
	s, db := newScheduler8001(t)

	// 空 cron → nil(短路径)
	empty := newJob8001("纯函数-空cron", "reg8001_cron")
	empty.CronExpression = ""
	require.NoError(t, db.Create(empty).Error)
	assert.Nil(t, newExecutor8001(t, s, db, empty).calculateNextRunTime())

	// 合法 6 位表达式 → 非 nil 且在未来
	valid := newJob8001("纯函数-合法cron", "reg8001_cron")
	valid.CronExpression = "*/1 * * * * *"
	require.NoError(t, db.Create(valid).Error)
	next := newExecutor8001(t, s, db, valid).calculateNextRunTime()
	require.NotNil(t, next)
	assert.True(t, next.After(time.Now().Add(-time.Minute)), "next 应在未来, got %v", *next)

	// 非法表达式 → AddFunc 报错 → nil
	invalid := newJob8001("纯函数-非法cron", "reg8001_cron")
	invalid.CronExpression = "not-a-cron-expression"
	require.NoError(t, db.Create(invalid).Error)
	assert.Nil(t, newExecutor8001(t, s, db, invalid).calculateNextRunTime())
}

// TestExec8001_DefaultLogger_Direct defaultLogger 三法冒烟直调 +
// SetLogger 注入缝验证(ExecuteJob 成功走 Infof / 失败走 Errorf)。
func TestExec8001_DefaultLogger_Direct(t *testing.T) {
	// 三法直调(冒烟:不 panic 即过)
	dl := &defaultLogger{}
	dl.Infof("exec8001 直调 %s", "Infof")
	dl.Warnf("exec8001 直调 %s", "Warnf")
	dl.Errorf("exec8001 直调 %s", "Errorf")

	// Logger 接口满足性(defaultLogger 是 Logger 的实现)
	var _ Logger = dl

	// SetLogger 注入缝:成功路径 → Infof;失败路径 → Errorf
	s, _ := newScheduler8001(t)
	counter := registerCountingTask8001(t, s, "reg8001_logger")
	job := newJob8001("注入缝-成功", "reg8001_logger")
	require.NoError(t, s.AddJob(job))
	require.NoError(t, s.ExecuteJob(job.ID))
	assert.Equal(t, int32(1), atomic.LoadInt32(counter))
	stub := stubLoggerOf8001(t, s)
	info, _, errCalls := stub.counts()
	assert.GreaterOrEqual(t, info, 1, "成功路径应产生 Infof")
	assert.Equal(t, 0, errCalls)

	s.RegisterTask("reg8001_logger_fail", func(ctx context.Context, params map[string]any) error {
		return errSentinel8001
	})
	failJob := newJob8001("注入缝-失败", "reg8001_logger_fail")
	require.NoError(t, s.AddJob(failJob))
	require.Error(t, s.ExecuteJob(failJob.ID))
	_, _, errCalls = stub.counts()
	assert.GreaterOrEqual(t, errCalls, 1, "失败路径应产生 Errorf")
}

// TestExec8001_ExecuteJob_ViaScheduler Scheduler.ExecuteJob 公开入口全链:
// 内存命中(AddJob !running 落入 s.jobs)→ 分发计数;不存在 ID → 报错不 panic。
func TestExec8001_ExecuteJob_ViaScheduler(t *testing.T) {
	s, _ := newScheduler8001(t)
	counter := registerCountingTask8001(t, s, "reg8001_via_sched")

	job := newJob8001("入口-立即执行", "reg8001_via_sched")
	require.NoError(t, s.AddJob(job))

	require.NoError(t, s.ExecuteJob(job.ID))
	assert.Equal(t, int32(1), atomic.LoadInt32(counter))

	// 不存在的 ID:内存 miss → DB miss → 报错(不 panic)
	err := s.ExecuteJob("00000000-0000-0000-0000-000000000001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务不存在")

	// 二次执行仍可分发(执行器无状态副作用)
	require.NoError(t, s.ExecuteJob(job.ID))
	assert.Equal(t, int32(2), atomic.LoadInt32(counter))
}

// ============================================================================
// Task 3 — Scheduler 公开方法 sqlite 驱动(全部 !running 路径,不起 cron)
// ============================================================================

// TestJob8001_AddJob_NotRunning 未 Start 时 AddJob 只落库 + 内存 map(:294-297 分支);
// 同名 job 重复 AddJob 无唯一约束 ⇒ 各自成行(quirk 记录: 无冲突/去重分支)。
func TestJob8001_AddJob_NotRunning(t *testing.T) {
	s, db := newScheduler8001(t)

	job := newJob8001("增-未运行", "reg8001_add")
	require.NoError(t, s.AddJob(job))

	// 落库
	var row models.Job
	require.NoError(t, db.First(&row, "id = ?", job.ID).Error)
	assert.Equal(t, job.JobName, row.JobName)
	assert.Equal(t, models.JobStatusNormal, row.Status, "默认状态为正常")

	// 内存:jobs 计入,executors 不计入(!running)
	total, running := s.GetJobCount()
	assert.Equal(t, 1, total)
	assert.Equal(t, 0, running)

	// 重复 AddJob 同名 job:再落一行,无冲突分支
	dup := newJob8001(job.JobName, "reg8001_add")
	dup.Status = models.JobStatusPause
	require.NoError(t, s.AddJob(dup))
	assert.NotEqual(t, job.ID, dup.ID, "两次 AddJob 各自生成新 ID")
	total, _ = s.GetJobCount()
	assert.Equal(t, 2, total)

	var rows []models.Job
	require.NoError(t, db.Where("job_name = ?", job.JobName).Find(&rows).Error)
	assert.Len(t, rows, 2, "同名 job 无唯一约束,两行并存")
}

// TestJob8001_UpdateJob 更新内存视图生效;持久化行为按实现断言(见 quirk)。
//
// quirk 记录(D-78-10 口径,零生产改动): UpdateJob 先 removeJob 对 sys_job 行软删,
// 随后 db.Save 走"全字段 UPDATE"(GORM Save 语义,含零值字段)且 UPDATE 不带
// deleted_at IS NULL 作用域 ⇒ 软删行被复活(deleted_at 写回 NULL)并落新值。
// 净效果 = 行存在且为新值;UpdateJob 对不存在的 ID 也不报错(无存在性校验,
// Save 0 行命中回退 OnConflict Create ⇒ 幽灵行被插入)。
func TestJob8001_UpdateJob(t *testing.T) {
	s, db := newScheduler8001(t)

	job := newJob8001("改-更新", "reg8001_update")
	require.NoError(t, s.AddJob(job))

	job.CronExpression = "0 */5 * * * *"
	job.Status = models.JobStatusPause
	require.NoError(t, s.UpdateJob(job))

	// 内存视图:仍 1 个 job
	total, _ := s.GetJobCount()
	assert.Equal(t, 1, total)

	// 持久化:行可读(被 Save 复活)且为新值
	var row models.Job
	require.NoError(t, db.First(&row, "id = ?", job.ID).Error, "Save 全字段更新复活软删行")
	assert.Equal(t, "0 */5 * * * *", row.CronExpression, "新表达式已持久化")
	assert.Equal(t, models.JobStatusPause, row.Status, "新状态已持久化")
	assert.False(t, row.DeletedAt.Valid, "软删标记被 Save 写回 NULL")

	// 不存在的 jobID:无存在性校验,不报错
	// quirk 续: Save 的 UPDATE 0 行命中会回退 OnConflict Create ⇒ 幽灵 job 意外落库
	ghost := newJob8001("改-幽灵", "reg8001_update")
	ghost.ID = "00000000-0000-0000-0000-000000000002"
	require.NoError(t, s.UpdateJob(ghost))
	total, _ = s.GetJobCount()
	assert.Equal(t, 2, total, "幽灵 job 进内存 map")
	var ghostRow models.Job
	require.NoError(t, db.First(&ghostRow, "id = ?", ghost.ID).Error,
		"Save 0 行命中回退 Create ⇒ 幽灵 job 意外落库(quirk)")
	assert.Equal(t, ghost.JobName, ghostRow.JobName)
}

// TestJob8001_RemoveJob 移除后内存计数减一、DB 行软删;不存在的 ID 幂等不报错。
func TestJob8001_RemoveJob(t *testing.T) {
	s, db := newScheduler8001(t)

	keep := newJob8001("删-保留", "reg8001_remove_a")
	require.NoError(t, s.AddJob(keep))
	gone := newJob8001("删-移除", "reg8001_remove_b")
	require.NoError(t, s.AddJob(gone))

	require.NoError(t, s.RemoveJob(gone.ID))

	total, _ := s.GetJobCount()
	assert.Equal(t, 1, total, "内存计数减一")

	var row models.Job
	assert.ErrorIs(t, db.First(&row, "id = ?", gone.ID).Error, gorm.ErrRecordNotFound)
	var soft models.Job
	require.NoError(t, db.Unscoped().First(&soft, "id = ?", gone.ID).Error)
	assert.NotNil(t, soft.DeletedAt, "removeJob 为软删(审计链保留)")

	// 保留的那条不受影响
	require.NoError(t, db.First(&row, "id = ?", keep.ID).Error)

	// 移除不存在的 ID:幂等返回 nil(removeJob 无存在性校验)
	require.NoError(t, s.RemoveJob("00000000-0000-0000-0000-000000000003"))
	total, _ = s.GetJobCount()
	assert.Equal(t, 1, total)
}

// TestJob8001_StartStopJob_Status StartJob → models.JobStatusNormal;
// StopJob → models.JobStatusPause;不存在/重复启停各走错误分支。
func TestJob8001_StartStopJob_Status(t *testing.T) {
	s, db := newScheduler8001(t)

	job := newJob8001("启停-状态机", "reg8001_toggle")
	job.Status = models.JobStatusPause
	require.NoError(t, db.Create(job).Error)

	// 暂停态 job → StartJob → 正常
	require.NoError(t, s.StartJob(job.ID))
	var row models.Job
	require.NoError(t, db.First(&row, "id = ?", job.ID).Error)
	assert.Equal(t, models.JobStatusNormal, row.Status, "StartJob 后应为正常态")
	exists, err := s.GetJobStatus(job.ID)
	require.NoError(t, err)
	assert.True(t, exists, "StartJob 后进入 executors")

	// 重复 StartJob:已在运行 → 错误
	err = s.StartJob(job.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务已经在运行")

	// StopJob → 暂停
	require.NoError(t, s.StopJob(job.ID))
	require.NoError(t, db.First(&row, "id = ?", job.ID).Error)
	assert.Equal(t, models.JobStatusPause, row.Status, "StopJob 后应为暂停态")
	exists, err = s.GetJobStatus(job.ID)
	require.NoError(t, err)
	assert.False(t, exists, "StopJob 后移出 executors")

	// 重复 StopJob:未在运行 → 错误
	err = s.StopJob(job.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务未在运行")

	// 不存在的 ID:启/停各自错误分支
	err = s.StartJob("00000000-0000-0000-0000-000000000004")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务不存在")
	err = s.StopJob("00000000-0000-0000-0000-000000000004")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务未在运行")
}

// TestJob8001_GetJobStatus executors 视角的存在性:!running 下 AddJob 不算运行,
// StartJob 后算;不存在 → false + nil error。
func TestJob8001_GetJobStatus(t *testing.T) {
	s, _ := newScheduler8001(t)

	// 不存在:不 panic
	exists, err := s.GetJobStatus("00000000-0000-0000-0000-000000000005")
	require.NoError(t, err)
	assert.False(t, exists)

	job := newJob8001("状态-查询", "reg8001_status")
	require.NoError(t, s.AddJob(job))

	exists, err = s.GetJobStatus(job.ID)
	require.NoError(t, err)
	assert.False(t, exists, "!running 下 AddJob 不进 executors")

	require.NoError(t, s.StartJob(job.ID))
	exists, err = s.GetJobStatus(job.ID)
	require.NoError(t, err)
	assert.True(t, exists, "StartJob 后在 executors 中")
}

// TestJob8001_GetJobCount_TotalRunning 预置 3 job(2 正常 1 暂停):
// !running 下 total==3 / running==0;StartJob 单独拉起一个 → running==1(不起 cron)。
func TestJob8001_GetJobCount_TotalRunning(t *testing.T) {
	s, _ := newScheduler8001(t)

	n1 := newJob8001("计数-正常1", "reg8001_count_a")
	require.NoError(t, s.AddJob(n1))
	n2 := newJob8001("计数-正常2", "reg8001_count_b")
	require.NoError(t, s.AddJob(n2))
	p1 := newJob8001("计数-暂停", "reg8001_count_c")
	p1.Status = models.JobStatusPause
	require.NoError(t, s.AddJob(p1))

	total, running := s.GetJobCount()
	assert.Equal(t, 3, total)
	assert.Equal(t, 0, running, "!running 调度器下无 executors")

	// StartJob 只 AddFunc 到未启动的 cron(不产生 goroutine,D-80-02 口径)
	require.NoError(t, s.StartJob(p1.ID))
	total, running = s.GetJobCount()
	assert.Equal(t, 3, total)
	assert.Equal(t, 1, running, "StartJob 后 executors 计数 +1")

	require.NoError(t, s.RemoveJob(p1.ID))
	total, running = s.GetJobCount()
	assert.Equal(t, 2, total)
	assert.Equal(t, 0, running)
}

// ============================================================================
// Task 4 — Start/Stop 生命周期 + 三组全局 var seams
// ============================================================================

// TestLife8001_StartAddStop Start 加载 DB 正常态 job → 运行中 AddJob 进 cron →
// Stop 正常返回(有 job 在册);二次 Start/二次 Stop 幂等。
// 纪律(R2): Start 配对 t.Cleanup(s.Stop);零 sleep、零时序断言(D-80-02)。
func TestLife8001_StartAddStop(t *testing.T) {
	s, db := newScheduler8001(t)

	job := newJob8001("生命周期-加载", "reg8001_life_idle")
	require.NoError(t, db.Create(job).Error)

	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(s.Stop)

	// 重复 Start:已在运行 → 错误
	err := s.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "调度器已经在运行")

	// Start 加载了 DB 中正常态 job(进入 cron + executors)
	total, running := s.GetJobCount()
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, running)

	// 运行中 AddJob → addJob 内部路径(:289-292)
	extra := newJob8001("生命周期-运行中新增", "reg8001_life_idle")
	require.NoError(t, s.AddJob(extra))
	total, running = s.GetJobCount()
	assert.Equal(t, 2, total)
	assert.Equal(t, 2, running)

	// 运行中 RemoveJob 退出一员(生命周期对称)
	require.NoError(t, s.RemoveJob(extra.ID))
	total, running = s.GetJobCount()
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, running)

	// Stop:有 job 在册下正常返回;二次 Stop 幂等不 panic
	s.Stop()
	s.Stop()

	// Stop 后可重新 Start(证明 running 翻转)→ 收口仍由 t.Cleanup 兜底
	require.NoError(t, s.Start(context.Background()))
}

// TestLife8001_StopWithoutStart 未 Start 直接 Stop → 不 panic(幂等边界)。
func TestLife8001_StopWithoutStart(t *testing.T) {
	s, _ := newScheduler8001(t)

	s.Stop() // !running → 直接 return

	// 状态查询仍可用
	total, running := s.GetJobCount()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, running)
}

// TestLife8001_StopTimeoutBounded Stop 的等待路径受 defaultShutdownTimeout 约束;
// 仅断言返回与状态翻转,不测墙钟时长(不真触发 cron,D-80-02)。
func TestLife8001_StopTimeoutBounded(t *testing.T) {
	s, _ := newScheduler8001(t)

	// 注册"长任务"handler 但绝不执行(cron 表达式远离当前时刻)
	registerCountingTask8001(t, s, "reg8001_long_task")
	job := newJob8001("生命周期-长任务", "reg8001_long_task")
	require.NoError(t, s.AddJob(job))

	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(s.Stop)

	// 等待路径:ctx.Done() 立即(无运行中任务);若任务卡死也会被 5s 上限截断
	s.Stop()

	// Stop 完成 ⇒ running 翻转 ⇒ 再次 Start 成功
	require.NoError(t, s.Start(context.Background()))
}

// stubNoticeHub8001 NoticeHub seam stub —— 记录广播调用。
type stubNoticeHub8001 struct {
	mu      sync.Mutex
	calls   int
	lastIDs []string
	lastMsg any
}

func (h *stubNoticeHub8001) BroadcastToUsers(userIDs []string, message any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls++
	h.lastIDs = append([]string(nil), userIDs...)
	h.lastMsg = message
}

func (h *stubNoticeHub8001) snapshot() (int, []string, any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls, h.lastIDs, h.lastMsg
}

// stubDBGetter8001 DBGetter seam stub —— 返回 fixture 库。
type stubDBGetter8001 struct{ db *gorm.DB }

func (g *stubDBGetter8001) GetDB() *gorm.DB { return g.db }

// TestSeam8001_NoticeHub GlobalNoticeHub 缝:save → Set → Get 同实例 → 可清空 →
// t.Cleanup restore(77-05 var-seam 纪律)。
func TestSeam8001_NoticeHub(t *testing.T) {
	old := GlobalNoticeHub
	t.Cleanup(func() { SetNoticeHub(old) })

	hub := &stubNoticeHub8001{}
	SetNoticeHub(hub)
	assert.Same(t, hub, GetNoticeHub())

	// 缝上广播可用(接口直达,不依赖 websocket)
	hub.BroadcastToUsers([]string{"u1", "u2"}, map[string]string{"t": "reg8001"})
	calls, ids, msg := hub.snapshot()
	assert.Equal(t, 1, calls)
	assert.Equal(t, []string{"u1", "u2"}, ids)
	assert.NotNil(t, msg)

	// 清空后读 nil 不 panic
	SetNoticeHub(nil)
	assert.Nil(t, GetNoticeHub())
}

// TestSeam8001_DBGetter GlobalDB 缝:save → Set → GetDB 取回 fixture 库 → restore。
func TestSeam8001_DBGetter(t *testing.T) {
	old := GlobalDB
	t.Cleanup(func() { SetDB(old) })

	db := newSchedDB8001(t)
	getter := &stubDBGetter8001{db: db}
	SetDB(getter)
	assert.Same(t, getter, GetDB())
	assert.Same(t, db, GetDB().GetDB())

	// 取回的库真实可用(往返一次 sqlite 查询)
	var count int64
	require.NoError(t, GetDB().GetDB().Model(&models.Job{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "fixture 库暂无 job 行")

	SetDB(nil)
	assert.Nil(t, GetDB())
}

// TestSeam8001_GlobalScheduler GlobalScheduler 缝:save → Set → Get 同实例 → restore。
func TestSeam8001_GlobalScheduler(t *testing.T) {
	old := GlobalScheduler
	t.Cleanup(func() { SetGlobalScheduler(old) })

	s, _ := newScheduler8001(t)
	SetGlobalScheduler(s)
	assert.Same(t, s, GetGlobalScheduler())

	// 取回的调度器真实可用(注册表读写走同实例)
	counter := registerCountingTask8001(t, GetGlobalScheduler(), "reg8001_global_sched")
	require.NoError(t, GetGlobalScheduler().GetTaskHandler("reg8001_global_sched")(context.Background(), nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(counter))

	SetGlobalScheduler(nil)
	assert.Nil(t, GetGlobalScheduler())
}

// ============================================================================
// Task 5 收口 — cron.go 缺口补足(纯函数 + var seams + 设备任务分支)
// ============================================================================

// TestGap8001_GetNoticeJobName notice_publish_<id> 拼字符串,纯函数直调。
func TestGap8001_GetNoticeJobName(t *testing.T) {
	assert.Equal(t, "notice_publish_xyz", getNoticeJobName("xyz"))
	assert.Equal(t, "notice_publish_", getNoticeJobName(""))
}

// TestGap8001_JoinWithSeparator 中文顿号连接,表驱动覆盖空/单/多分支。
func TestGap8001_JoinWithSeparator(t *testing.T) {
	assert.Equal(t, "", joinWithSeparator(nil, "、"), "空切片 → 空串")
	assert.Equal(t, "甲", joinWithSeparator([]string{"甲"}, "、"), "单元素不加分隔符")
	assert.Equal(t, "甲、乙、丙", joinWithSeparator([]string{"甲", "乙", "丙"}, "、"), "多元素以分隔符相连")
}

// TestGap8001_FormatMemberList 优先 nickname,缺则 fallback username,空集合 → ""。
func TestGap8001_FormatMemberList(t *testing.T) {
	assert.Equal(t, "", formatMemberList(nil))
	assert.Equal(t, "张三", formatMemberList([]dutyMember{{UserID: "u1", Username: "u1", NickName: "张三"}}))
	assert.Equal(t, "u2", formatMemberList([]dutyMember{{UserID: "u2", Username: "u2"}}),
		"缺 nickname → fallback username")
	assert.Equal(t, "A、B、C", formatMemberList([]dutyMember{
		{NickName: "A"}, {NickName: "B"}, {NickName: "C"},
	}))
}

// stubVDIVMService8001 VDI seam stub,实现 VDIVMService 双方法。
type stubVDIVMService8001 struct{}

func (s *stubVDIVMService8001) SyncAllVMs(ctx context.Context, serverID string) error {
	return nil
}
func (s *stubVDIVMService8001) SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error {
	return nil
}

// TestGap8001_SetGetVDIVMService VDI seam save/set/get/restore 纪律。
func TestGap8001_SetGetVDIVMService(t *testing.T) {
	old := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(old) })

	stub := &stubVDIVMService8001{}
	SetVDIVMService(stub)
	assert.Same(t, stub, GetVDIVMService())

	SetVDIVMService(nil)
	assert.Nil(t, GetVDIVMService())
}

// stubDeviceInfoCollection8001 设备信息采集 seam stub,实现 EnqueueAllOnlineDevices。
type stubDeviceInfoCollection8001 struct {
	errOnEnqueue error
}

func (s *stubDeviceInfoCollection8001) EnqueueAllOnlineDevices(ctx context.Context) error {
	return s.errOnEnqueue
}

// TestGap8001_SetGetDeviceInfoCollectionService 设备信息采集 seam 纪律。
func TestGap8001_SetGetDeviceInfoCollectionService(t *testing.T) {
	old := GlobalDeviceInfoCollectionService
	t.Cleanup(func() { SetDeviceInfoCollectionService(old) })

	stub := &stubDeviceInfoCollection8001{}
	SetDeviceInfoCollectionService(stub)
	assert.Same(t, stub, GetDeviceInfoCollectionService())

	SetDeviceInfoCollectionService(nil)
	assert.Nil(t, GetDeviceInfoCollectionService())
}

// errDeviceMonitorService8001 设备监控 stub,所有方法返回 sentinel error,
// 覆盖设备任务的"stub 返回 error"分支。
type errDeviceMonitorService8001 struct{}

var errDevice8001 = errors.New("reg8001 设备服务故障")

func (e *errDeviceMonitorService8001) CheckAllDevicesStatus(ctx context.Context) (int, int, error) {
	return 0, 0, errDevice8001
}
func (e *errDeviceMonitorService8001) CollectAllPortStatus(ctx context.Context) error {
	return errDevice8001
}
func (e *errDeviceMonitorService8001) CollectAllMACAddresses(ctx context.Context) error {
	return errDevice8001
}
func (e *errDeviceMonitorService8001) BackupAllConfigurations(ctx context.Context) error {
	return errDevice8001
}

// TestGap8001_DeviceTasks_NilSeam 5 个设备任务在 seam 为 nil 时统一报错分支。
func TestGap8001_DeviceTasks_NilSeam(t *testing.T) {
	origDMS := GlobalDeviceMonitorService
	origDIC := GlobalDeviceInfoCollectionService
	t.Cleanup(func() {
		SetDeviceMonitorService(origDMS)
		SetDeviceInfoCollectionService(origDIC)
	})
	SetDeviceMonitorService(nil)
	SetDeviceInfoCollectionService(nil)

	assert.Error(t, executeDeviceStatusCheckTask(context.Background()), "device_status_check")
	assert.Error(t, executePortCollectionTask(context.Background()), "port_collection")
	assert.Error(t, executeMACCollectionTask(context.Background()), "mac_collection")
	assert.Error(t, executeConfigBackupTask(context.Background()), "config_backup")
	assert.Error(t, executeDeviceInfoUpdateTask(context.Background()), "device_info_update")
}

// TestGap8001_DeviceTasks_StubSuccess seam 注入成功 → 5 任务走完 nil 守卫到业务完成。
func TestGap8001_DeviceTasks_StubSuccess(t *testing.T) {
	origDMS := GlobalDeviceMonitorService
	origDIC := GlobalDeviceInfoCollectionService
	t.Cleanup(func() {
		SetDeviceMonitorService(origDMS)
		SetDeviceInfoCollectionService(origDIC)
	})

	// mockDeviceMonitorService 见 cron_test.go:50(全成功返回);同名复用
	SetDeviceMonitorService(&mockDeviceMonitorService{})
	SetDeviceInfoCollectionService(&stubDeviceInfoCollection8001{})

	assert.NoError(t, executeDeviceStatusCheckTask(context.Background()))
	assert.NoError(t, executePortCollectionTask(context.Background()))
	assert.NoError(t, executeMACCollectionTask(context.Background()))
	assert.NoError(t, executeConfigBackupTask(context.Background()))
	assert.NoError(t, executeDeviceInfoUpdateTask(context.Background()))
}

// TestGap8001_DeviceTasks_StubError seam 注入返错 → 5 任务的 error 分支(device_info_update
// 不走 DeviceMonitorService,只走 DeviceInfoCollectionService;此处仅覆盖 4 个 DMS 路径)。
func TestGap8001_DeviceTasks_StubError(t *testing.T) {
	origDMS := GlobalDeviceMonitorService
	origDIC := GlobalDeviceInfoCollectionService
	t.Cleanup(func() {
		SetDeviceMonitorService(origDMS)
		SetDeviceInfoCollectionService(origDIC)
	})

	SetDeviceMonitorService(&errDeviceMonitorService8001{})
	SetDeviceInfoCollectionService(&stubDeviceInfoCollection8001{errOnEnqueue: errDevice8001})

	assert.Error(t, executeDeviceStatusCheckTask(context.Background()))
	assert.Error(t, executePortCollectionTask(context.Background()))
	assert.Error(t, executeMACCollectionTask(context.Background()))
	assert.Error(t, executeConfigBackupTask(context.Background()))
	assert.Error(t, executeDeviceInfoUpdateTask(context.Background()))
}

// ============================================================================
// Task 5 收口续 — Round 2:cron.go 任务族(duty/notice)sqlite 驱动分支
// ============================================================================

// newDutyDB8001 sqlite 文件库,基于 newSchedDB8001 + 手写 DDL(sys_duty_config /
// sys_user / sys_duty_pool / sys_duty_schedule / sys_notice / sys_notice_target);
// AutoMigrate 已落 sys_job/sys_job_log,本 fixture 追加 duty/notice 侧表。
func newDutyDB8001(t *testing.T) *gorm.DB {
	t.Helper()
	db := newSchedDB8001(t)
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS sys_duty_config (
			id TEXT PRIMARY KEY,
			reminder_enabled INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			nickname TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_duty_pool (
			id TEXT PRIMARY KEY,
			pool_name TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_duty_schedule (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			pool_id TEXT,
			schedule_date TEXT,
			status INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS sys_notice (
			id TEXT PRIMARY KEY,
			notice_title TEXT,
			priority INTEGER,
			publish_status INTEGER,
			target_type INTEGER,
			end_date TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_notice_target (
			id TEXT PRIMARY KEY,
			notice_id TEXT,
			target_id TEXT
		)`,
	}
	for _, stmt := range ddl {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

// TestGap8001_IsDutyReminderEnabled 三分支:无配置默认启用 / 显式禁用 / 显式启用。
func TestGap8001_IsDutyReminderEnabled(t *testing.T) {
	db := newDutyDB8001(t)

	// 无配置 → 默认 true(ErrRecordNotFound 分支)
	enabled, err := isDutyReminderEnabled(db)
	require.NoError(t, err)
	assert.True(t, enabled)

	// 显式禁用
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_config (id, reminder_enabled) VALUES (?, ?)`, "cfg1", 0).Error)
	enabled, err = isDutyReminderEnabled(db)
	require.NoError(t, err)
	assert.False(t, enabled)

	// 显式启用
	require.NoError(t, db.Exec(`UPDATE sys_duty_config SET reminder_enabled = ? WHERE id = ?`, 1, "cfg1").Error)
	enabled, err = isDutyReminderEnabled(db)
	require.NoError(t, err)
	assert.True(t, enabled)
}

// TestGap8001_GetTodayDutyMembers 空集 + 1 条带 join 信息。
func TestGap8001_GetTodayDutyMembers(t *testing.T) {
	db := newDutyDB8001(t)
	today := time.Now().Format("2006-01-02")

	// 空
	members, err := getTodayDutyMembers(db, today)
	require.NoError(t, err)
	assert.Empty(t, members)

	// 1 条
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname) VALUES (?, ?, ?)`, "u1", "alice", "爱丽丝").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_pool (id, pool_name) VALUES (?, ?)`, "p1", "A 班").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_schedule (id, user_id, pool_id, schedule_date, status) VALUES (?, ?, ?, ?, ?)`,
		"s1", "u1", "p1", today, 0).Error)
	members, err = getTodayDutyMembers(db, today)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "u1", members[0].UserID)
	assert.Equal(t, "alice", members[0].Username)
	assert.Equal(t, "爱丽丝", members[0].NickName)
	assert.Equal(t, "A 班", members[0].PoolName)
}

// TestGap8001_GetTargetUserIDs 全员(targetType 0, 排除软删)+ 通知目标(targetType 非 0)。
func TestGap8001_GetTargetUserIDs(t *testing.T) {
	db := newDutyDB8001(t)

	// targetType 0:全部未删除用户
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id) VALUES (?)`, "u1").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, deleted_at) VALUES (?, ?)`, "u2", time.Now()).Error)
	ids, err := getTargetUserIDs(db, "n1", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"u1"}, ids, "软删用户被排除")

	// targetType 1:通知目标表
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_target (id, notice_id, target_id) VALUES (?, ?, ?)`, "t1", "n1", "u3").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_target (id, notice_id, target_id) VALUES (?, ?, ?)`, "t2", "n1", "u4").Error)
	ids, err = getTargetUserIDs(db, "n1", 1)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u3", "u4"}, ids)
}

// TestGap8001_SendDutyReminderNotification 多 member 经 stubNoticeHub 验证
// fallback username + 消息结构(空 members 也广播但 ids 为空 — 由 formatMemberList
// 空串分支直接返回 — 仍走完 BroadcastToUsers 调用,本测试聚焦多 member 主路径)。
func TestGap8001_SendDutyReminderNotification(t *testing.T) {
	origHub := GlobalNoticeHub
	t.Cleanup(func() { SetNoticeHub(origHub) })

	hub := &stubNoticeHub8001{}
	SetNoticeHub(hub)

	// 空 members → BroadcastToUsers 仍被调用(函数无 len 守卫),userIDs 为空
	sendDutyReminderNotification(nil)
	calls, ids, _ := hub.snapshot()
	assert.Equal(t, 1, calls)
	assert.Empty(t, ids)

	// 多 member:nickname 优先,缺则 fallback username
	members := []dutyMember{
		{UserID: "u1", Username: "u1", NickName: "张三"},
		{UserID: "u2", Username: "u2"},
	}
	sendDutyReminderNotification(members)
	calls, ids, msg := hub.snapshot()
	assert.Equal(t, 2, calls)
	assert.Equal(t, []string{"u1", "u2"}, ids)
	require.NotNil(t, msg)
	raw := fmt.Sprintf("%+v", msg)
	assert.Contains(t, raw, "张三")
	assert.Contains(t, raw, "u2")
	assert.Contains(t, raw, "duty_reminder")
}

// TestGap8001_HandleTaskLifecycle nil-scheduler 守卫 + 一次性 RemoveJob + 周期保留。
func TestGap8001_HandleTaskLifecycle(t *testing.T) {
	s, _ := newScheduler8001(t)

	// GlobalScheduler nil → no-op(不 panic)
	orig := GlobalScheduler
	t.Cleanup(func() { SetGlobalScheduler(orig) })
	SetGlobalScheduler(nil)
	handleTaskLifecycle(models.Job{MisfirePolicy: models.MisfirePolicyExecuteOnce}, "x", "n1")

	// 一次性 + job 存在 → RemoveJob
	SetGlobalScheduler(s)
	once := newJob8001("handleLifecycle-once-8001", "x")
	once.MisfirePolicy = models.MisfirePolicyExecuteOnce
	require.NoError(t, s.AddJob(once))
	require.Equal(t, 1, sGetTotal(s))
	handleTaskLifecycle(*once, "handleLifecycle-once-8001", "n1")
	assert.Equal(t, 0, sGetTotal(s), "一次性任务 RemoveJob 后内存清零")

	// 周期 + job 存在 → 保留
	periodic := newJob8001("handleLifecycle-periodic-8001", "x")
	periodic.MisfirePolicy = models.MisfirePolicyDefault
	require.NoError(t, s.AddJob(periodic))
	handleTaskLifecycle(*periodic, "handleLifecycle-periodic-8001", "n1")
	assert.Equal(t, 1, sGetTotal(s), "周期任务保留")
}

// sGetTotal 小封装(GetJobCount 总数断言可读性)。
func sGetTotal(s *Scheduler) int {
	total, _ := s.GetJobCount()
	return total
}

// TestGap8001_ShouldStopNotice endDate nil / 格式错 / 未来 / 过去 + GlobalScheduler 4 分支。
func TestGap8001_ShouldStopNotice(t *testing.T) {
	db := newDutyDB8001(t)

	// endDate nil → false
	assert.False(t, shouldStopNotice(nil, "title", "n1", db))

	// endDate 格式错 → false
	bad := "not-rfc3339"
	assert.False(t, shouldStopNotice(&bad, "title", "n1", db))

	// endDate 未来 → false
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	assert.False(t, shouldStopNotice(&future, "title", "n1", db))

	// 过去 + GlobalScheduler nil → true(内部块被守卫)
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	orig := GlobalScheduler
	t.Cleanup(func() { SetGlobalScheduler(orig) })
	SetGlobalScheduler(nil)
	assert.True(t, shouldStopNotice(&past, "title", "n1", db))

	// 过去 + GlobalScheduler 有 + 对应 job → 调 RemoveJob + true
	// 注意:scheduler 与 shouldStopNotice 必须共享同一 db(否则按 job_name 查不到)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	SetGlobalScheduler(s)
	noticeID := "n-past-8001"
	jobName := getNoticeJobName(noticeID) // "notice_publish_n-past-8001"
	job := newJob8001(jobName, "x")
	require.NoError(t, s.AddJob(job))
	assert.Equal(t, 1, sGetTotal(s))
	assert.True(t, shouldStopNotice(&past, "title", noticeID, db))
	assert.Equal(t, 0, sGetTotal(s), "过去 + 有 job + GlobalScheduler → RemoveJob 触发")
}

// TestGap8001_BroadcastNoticeToUsers hub nil 无广播;有 hub + targetType 0 走全用户列表。
func TestGap8001_BroadcastNoticeToUsers(t *testing.T) {
	db := newDutyDB8001(t)
	orig := GlobalNoticeHub
	t.Cleanup(func() { SetNoticeHub(orig) })

	// hub nil → no-op(不 panic)
	SetNoticeHub(nil)
	broadcastNoticeToUsers(db, "n1", "title", 1, 0)

	// 有 hub + 2 个用户(targetType 0) → 广播
	hub := &stubNoticeHub8001{}
	SetNoticeHub(hub)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id) VALUES (?), (?)`, "u1", "u2").Error)
	broadcastNoticeToUsers(db, "n1", "title", 1, 0)
	calls, ids, msg := hub.snapshot()
	assert.Equal(t, 1, calls)
	assert.ElementsMatch(t, []string{"u1", "u2"}, ids)
	require.NotNil(t, msg)
	assert.Contains(t, fmt.Sprintf("%+v", msg), "new_notice")
}

// TestGap8001_ExecuteDutyReminderTask GlobalDB nil → 错误;禁用配置 → 跳过;无成员 → 跳过。
func TestGap8001_ExecuteDutyReminderTask(t *testing.T) {
	db := newDutyDB8001(t)
	orig := GlobalDB
	t.Cleanup(func() { SetDB(orig) })

	// nil GlobalDB → error
	SetDB(nil)
	require.Error(t, executeDutyReminderTask(context.Background(), nil))

	// 禁用配置 → 跳过(nil 返回)
	SetDB(&stubDBGetter8001{db: db})
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_config (id, reminder_enabled) VALUES (?, ?)`, "cfg1", 0).Error)
	require.NoError(t, executeDutyReminderTask(context.Background(), nil))

	// 启用(默认)但无今日值班 → 跳过
	require.NoError(t, db.Exec(`DELETE FROM sys_duty_config WHERE id = ?`, "cfg1").Error)
	require.NoError(t, executeDutyReminderTask(context.Background(), nil))
}

// TestGap8001_ExecuteNoticePublishTask nil GlobalDB → 错误;不存在的 noticeID →
// 内部 ErrRecordNotFound → 静默返回 nil(not "已发布或不存在"日志路径)。
func TestGap8001_ExecuteNoticePublishTask(t *testing.T) {
	db := newDutyDB8001(t)
	orig := GlobalDB
	t.Cleanup(func() { SetDB(orig) })

	// nil GlobalDB → error
	SetDB(nil)
	require.Error(t, executeNoticePublishTask(context.Background(), map[string]any{"param": "x"}))

	// 不存在的 noticeID → ErrRecordNotFound 分支 → 返回 nil(不调用 senderService)
	SetDB(&stubDBGetter8001{db: db})
	require.NoError(t, executeNoticePublishTask(context.Background(), map[string]any{"param": "no-such-notice-8001"}))
}



// TestGap8001_AddJob_Running_InvalidCron addJob 错误分支:running 状态下 AddJob
// 传入非法 cron 表达式 → cron.AddFunc 报错 → AddJob 透传错误。
// 直接写 s.running = true(同包)绕开 Start 以避免起 cron goroutine。
func TestGap8001_AddJob_Running_InvalidCron(t *testing.T) {
	s, _ := newScheduler8001(t)
	s.running = true

	bad := newJob8001("运行中-坏cron", "reg8001_bad_cron")
	bad.CronExpression = "this-is-not-a-cron"
	err := s.AddJob(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "添加任务到调度器失败")

	// running 标记未被错误回滚(本次 AddJob 错误后 s.running 仍为 true)
	assert.True(t, s.running)
}
