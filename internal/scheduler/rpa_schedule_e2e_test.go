package scheduler

// Phase 158 / CLEANUP-02 真跑集成用例：RPA 定时调度「到点真触发」。
//
// 覆盖 REQUIREMENTS CLEANUP-02 的验收句：
//   「有『到点真触发』集成用例（可用秒级 cron）」+「秒级 cron 到点真触发 RPA 执行，
//     trigger_type=schedule」。
//
// 本用例是真链路，不是模拟：
//   - 真 sqlite 库（db.NewDatabase + 真实 AutoMigrate）
//   - 真 Redis（REDIS_ADDR，默认 127.0.0.1:6379；不可达则 skip ——
//     零新增依赖约束下不引入 miniredis，与 Phase 153 的条件化范式一致）
//   - 真 cron 引擎（robfig/cron，秒级表达式「0 * * * * *」= 每分钟第 0 秒）
//   - 真 TaskService.Execute（调度触发与手动触发共用同一执行入口）
//
// 断言：
//   1. 到点后 sys_rpa_executions 出现 trigger_type='schedule' 的执行记录
//   2. 调度行的 run_count / last_run_time 被回写（调度真跑了，不是空转）

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	rpav1 "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// scheduleE2ERedisAddr 取 Redis 地址（环境变量优先）。
func scheduleE2ERedisAddr() string {
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:6379"
}

// scheduleE2EHostPort 拆 Redis 地址。
func scheduleE2EHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err, "REDIS_ADDR 格式非法: %s", addr)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "REDIS_ADDR 端口非法: %s", addr)
	return host, port
}

// TestRPASchedule_E2E_TriggersRealExecution 真跑：建每分钟调度 → 到点真触发执行记录。
func TestRPASchedule_E2E_TriggersRealExecution(t *testing.T) {
	addr := scheduleE2ERedisAddr()
	host, port := scheduleE2EHostPort(t, addr)

	probe := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second})
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := probe.Ping(pingCtx).Err(); err != nil {
		_ = probe.Close()
		t.Skipf("Redis 不可达（%s），跳过真 Redis 调度集成用例: %v", addr, err)
	}
	require.NoError(t, probe.Close())

	ctx := context.Background()

	// 真 sqlite 库（真实 AutoMigrate：缺表/缺注册会在此直接炸）
	database, err := db.NewDatabase(&config.DatabaseConfig{
		Type: "sqlite",
		Path: filepath.Join(t.TempDir(), "rpa-schedule-e2e.db"),
	})
	require.NoError(t, err, "创建 sqlite 测试库失败")
	require.NoError(t, database.AutoMigrate(), "真实 AutoMigrate 必须成功")
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	// 真 Redis 缓存（MultiLevel：L1 内存 + L2 Redis；L2 提供 DirectRedisXAdd）
	redisCache, err := cache.NewRedisCache(&cache.CacheConfig{
		Host:     host,
		Port:     port,
		DB:       15,
		PoolSize: 2,
	}, "xingran")
	require.NoError(t, err, "Redis 缓存构造失败")
	t.Cleanup(func() { _ = redisCache.Close() })
	mlCache := cache.NewMultiLevelCache(cache.NewMemoryCache(256, time.Minute), redisCache)

	// 任务执行服务（调度触发与手动触发共用同一入口）
	credentialSvc := rpav1.NewCredentialService(database.GetDB(), nil, mlCache)
	taskSvc := rpav1.NewTaskService(database.GetDB(), mlCache, credentialSvc)

	// 真调度器
	sched := NewScheduler(database.GetDB())
	require.NoError(t, sched.Start(ctx), "调度器启动失败")
	t.Cleanup(func() { sched.Stop() })

	// 调度服务 + job 同步器 + 处理器注册
	syncer := NewRPAScheduleJobSyncer(sched, database.GetDB())
	scheduleSvc := rpav1.NewScheduleService(database.GetDB(), taskSvc, syncer)
	RegisterRPAScheduleTasks(sched, database.GetDB(), scheduleSvc)

	// 播种一条启用的 RPA 任务（被调度触发）
	task := &rpamodels.Task{
		TaskName: "Phase158 定时调度验收任务",
		Script:   []byte(`[{"type":"navigate","value":"https://example.com"}]`),
		Status:   rpamodels.TaskStatusEnabled,
		Timeout:  300,
	}
	require.NoError(t, database.GetDB().Create(task).Error)

	// 建一条「每分钟第 0 秒」的调度（秒级 cron，REQUIREMENTS 允许秒级表达式）
	created, err := scheduleSvc.Create(ctx, &rpav1.CreateScheduleRequest{
		Name:     "Phase158 每分钟验收调度",
		TaskID:   task.ID,
		CronExpr: "0 * * * * *",
	})
	require.NoError(t, err, "创建调度失败")
	require.NotNil(t, created.NextRunTime, "创建后必须算出下次执行时间")

	// 调度 job 真落到 sys_job（启动即注册，不是只在内存里）
	var jobs []struct {
		ID           string
		JobName      string
		InvokeTarget string
		CronExpr     string
	}
	require.NoError(t, database.GetDB().Table("sys_job").
		Where("invoke_target LIKE ?", "rpa_schedule:%").
		Find(&jobs).Error)
	require.Len(t, jobs, 1, "启用的调度必须注册一条 cron job")
	assert.Equal(t, "rpa_schedule:"+created.ID, jobs[0].InvokeTarget)

	// 到点等待（每分钟第 0 秒 → 最长 75 秒）
	deadline := time.Now().Add(75 * time.Second)
	var exec rpamodels.Execution
	var execErr error
	for time.Now().Before(deadline) {
		execErr = database.GetDB().
			Where("task_id = ? AND trigger_type = ?", task.ID, "schedule").
			Order("created_at DESC").
			First(&exec).Error
		if execErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	require.NoError(t, execErr, "调度到点未触发执行记录（真跑失败）")
	assert.Equal(t, "schedule", exec.TriggerType, "调度触发的执行记录 trigger_type 必须是 schedule")
	assert.Equal(t, task.ID, exec.TaskID)

	// 调度行回写（上次/下次执行时间 + 执行次数）
	var schedRow rpamodels.Schedule
	require.NoError(t, database.GetDB().First(&schedRow, "id = ?", created.ID).Error)
	assert.GreaterOrEqual(t, schedRow.RunCount, 1, "调度执行次数必须回写")
	assert.NotNil(t, schedRow.LastRunTime, "上次执行时间必须回写")
	assert.NotNil(t, schedRow.NextRunTime, "下次执行时间必须回写")

	// 停用即摘除 job：停用后 sys_job 不再有该调度的**未删**行
	// （models.Job 是软删模型，RemoveJob 走的是软删，故显式加 deleted_at IS NULL）。
	require.NoError(t, scheduleSvc.SetStatus(ctx, created.ID, rpav1.ScheduleStatusDisabled))
	var afterDisable int64
	require.NoError(t, database.GetDB().Table("sys_job").
		Where("invoke_target = ?", "rpa_schedule:"+created.ID).
		Where("deleted_at IS NULL").
		Count(&afterDisable).Error)
	assert.Equal(t, int64(0), afterDisable, "停用后调度 job 必须被摘除")
}
