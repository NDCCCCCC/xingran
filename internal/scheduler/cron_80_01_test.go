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
//   - 全文件禁 t.Parallel(全局 var seams + 包级注册表)
//   - status 断言只引用 models.* 常量,禁裸 0/1 字面量(CLAUDE.md Status Value Convention)
//   - helper 一律 8001 后缀(D-80-07,防同包既有 helper 撞名)

import (
	"context"
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

func (l *schedStubLogger8001) Infof(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infof = append(l.infof, fmt.Sprintf(format, args...))
}

func (l *schedStubLogger8001) Warnf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnf = append(l.warnf, fmt.Sprintf(format, args...))
}

func (l *schedStubLogger8001) Errorf(format string, args ...interface{}) {
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
	s.RegisterTask(taskType, func(ctx context.Context, params map[string]interface{}) error {
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

// quiet8001 抑制未使用告警的占位(time 包在后续 task 使用)。
var _ = time.Now
