package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// Phase 74-05: metrics.go + scaling_service.go + docker_client.go 测试。
// =====================================================================

func newMetricsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_executions (
			id TEXT PRIMARY KEY, created_at DATETIME, deleted_at DATETIME,
			task_id TEXT, status TEXT, duration INTEGER, end_time DATETIME
		);
		CREATE TABLE sys_rpa_workers (
			id TEXT PRIMARY KEY, deleted_at DATETIME, updated_at DATETIME,
			worker_name TEXT, worker_id TEXT,
			status TEXT, max_concurrency INTEGER DEFAULT 3, current_tasks INTEGER DEFAULT 0,
			docker_container_id TEXT, last_heartbeat INTEGER
		);
		CREATE TABLE sys_rpa_scaling_events (
			id TEXT PRIMARY KEY, created_at DATETIME,
			event_type TEXT, from_count INTEGER, to_count INTEGER,
			trigger_reason TEXT, queue_length INTEGER, active_workers INTEGER,
			worker_capacity INTEGER, average_exec_time INTEGER,
			container_ids TEXT, status TEXT, error_message TEXT
		)
	`).Error)
	return db
}

func TestMetricsService_Queries(t *testing.T) {
	db := newMetricsTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_executions (id, task_id, status, created_at, duration, end_time) VALUES
		('e1','t','pending',?,5000,?),
		('e2','t','running',?,NULL,NULL),
		('e3','t','success',?,8000,?)`, now, now, now, now, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_workers (id, status, max_concurrency, current_tasks) VALUES
		('w1','online',4,1),('w2','busy',2,2),('w3','offline',9,0)`).Error)

	svc := NewMetricsService(db)
	ctx := context.Background()

	ql, err := svc.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, ql)

	aw, err := svc.GetActiveWorkers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, aw)

	cap, err := svc.GetWorkerCapacity(ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, cap)

	pe, err := svc.GetPendingExecutions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, pe)

	re, err := svc.GetRunningExecutions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, re)

	avg, err := svc.GetAverageExecutionTime(ctx)
	require.NoError(t, err)
	assert.Equal(t, 8*time.Second, avg, "仅 success 记录参与 AVG (8000ms)")

	// 无成功执行 → 默认 30s
	require.NoError(t, db.Exec(`UPDATE sys_rpa_executions SET status='failed' WHERE id='e3'`).Error)
	avg, err = svc.GetAverageExecutionTime(ctx)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, avg)

	// 事件记录 + 历史
	require.NoError(t, svc.RecordScalingEvent(ctx, &ScalingEvent{
		ID: "ev1", EventType: "scale_up", FromCount: 1, ToCount: 2, Status: "success",
	}))
	history, err := svc.GetScalingHistory(ctx, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "scale_up", history[0].EventType)
}

// fakeMetricsService MetricsService mock。
type fakeMetricsService struct {
	mu          sync.Mutex
	pending     int
	running     int
	active      int
	capacity    int
	avg         time.Duration
	pendingErr  error
	events      []ScalingEvent
	historyErr  error
}

func (f *fakeMetricsService) GetQueueLength(ctx context.Context) (int, error) { return f.pending, nil }
func (f *fakeMetricsService) GetActiveWorkers(ctx context.Context) (int, error) {
	return f.active, nil
}
func (f *fakeMetricsService) GetWorkerCapacity(ctx context.Context) (int, error) { return f.capacity, nil }
func (f *fakeMetricsService) GetPendingExecutions(ctx context.Context) (int, error) {
	return f.pending, f.pendingErr
}
func (f *fakeMetricsService) GetRunningExecutions(ctx context.Context) (int, error) {
	return f.running, nil
}
func (f *fakeMetricsService) GetAverageExecutionTime(ctx context.Context) (time.Duration, error) {
	return f.avg, nil
}
func (f *fakeMetricsService) RecordScalingEvent(ctx context.Context, event *ScalingEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, *event)
	return nil
}
func (f *fakeMetricsService) GetScalingHistory(ctx context.Context, limit int) ([]ScalingEvent, error) {
	return nil, f.historyErr
}

func TestMetricsCollector(t *testing.T) {
	fake := &fakeMetricsService{pending: 2, running: 1, active: 3, capacity: 9, avg: time.Second}
	mc := NewMetricsCollector(fake, 10*time.Millisecond)

	// 指标未采集 → nil
	assert.Nil(t, mc.GetLatestMetrics())

	ctx, cancel := context.WithCancel(context.Background())
	go mc.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for mc.GetLatestMetrics() == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	got := mc.GetLatestMetrics()
	require.NotNil(t, got, "采集器应立即采集一次")
	assert.Equal(t, 2, got.QueueLength)
	assert.Equal(t, 3, got.ActiveWorkers)

	cancel()
	mc.Stop()
}

func TestScalingMetrics_Decisions(t *testing.T) {
	m := &ScalingMetrics{QueueLength: 10, RunningTasks: 5, ActiveWorkers: 3, WorkerCapacity: 10}

	// 扩容: 队列压力 10/10=1.0 > 0.7
	assert.True(t, m.ShouldScaleUp(2, 10, 0.7))
	// 已达上限
	assert.False(t, m.ShouldScaleUp(2, 3, 0.7))
	// 低负载
	idle := &ScalingMetrics{QueueLength: 0, RunningTasks: 1, ActiveWorkers: 5, WorkerCapacity: 10}
	assert.False(t, idle.ShouldScaleUp(2, 10, 0.7))

	// 缩容
	assert.True(t, idle.ShouldScaleDown(2, 0.3))
	assert.False(t, idle.ShouldScaleDown(5, 0.3), "低于最小值不缩容")
	busy := &ScalingMetrics{QueueLength: 3, RunningTasks: 9, ActiveWorkers: 5, WorkerCapacity: 10}
	assert.False(t, busy.ShouldScaleDown(2, 0.3), "有积压不缩容")

	// 目标数计算
	target := m.CalculateTargetWorkers(2, 10, 0.7)
	assert.GreaterOrEqual(t, target, 2)
	assert.LessOrEqual(t, target, 10)
	empty := &ScalingMetrics{}
	assert.Equal(t, 2, empty.CalculateTargetWorkers(2, 10, 0.7), "容量为 0 返回最小值")

	// 决策描述
	assert.Contains(t, m.DescribeDecision("scale_up"), "扩容")
	assert.Contains(t, m.DescribeDecision("scale_down"), "缩容")
	assert.Contains(t, m.DescribeDecision("other"), "无需调整")
}

func TestValidateScalingConfig_And_ParseContainerIDs(t *testing.T) {
	ok := DefaultScalingConfig()
	require.NoError(t, ValidateScalingConfig(ok))
	assert.False(t, ok.Enabled)
	assert.True(t, ok.EnableMockDocker)

	bad := DefaultScalingConfig()
	bad.MinWorkers = -1
	require.Error(t, ValidateScalingConfig(bad))

	bad = DefaultScalingConfig()
	bad.MaxWorkers = 1
	require.Error(t, ValidateScalingConfig(bad))

	bad = DefaultScalingConfig()
	bad.ScaleUpThreshold = 1.5
	require.Error(t, ValidateScalingConfig(bad))

	bad = DefaultScalingConfig()
	bad.ScaleUpLimit = 0
	require.Error(t, ValidateScalingConfig(bad))

	bad = DefaultScalingConfig()
	bad.CheckInterval = time.Second
	require.Error(t, ValidateScalingConfig(bad))

	ids, err := ParseContainerIDs(`["a","b"]`)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, ids)

	ids, err = ParseContainerIDs("a, b ,c")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, ids)

	_, err = ParseContainerIDs("  ")
	require.Error(t, err)

	_, err = ParseContainerIDs("[bad json")
	require.Error(t, err)
}

func TestScalingService_ManualScaleWithMockDocker(t *testing.T) {
	db := newMetricsTestDB(t)
	cfg := DefaultScalingConfig()
	cfg.EnableMockDocker = true
	cfg.ScaleUpLimit = 3
	svc := NewScalingService(db, cfg, nil).(*scalingServiceImpl)
	ctx := context.Background()

	// 5 个在线 worker, 缩容 2 个仍 >= MinWorkers(2)
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_workers (id, status, max_concurrency) VALUES
		('mw1','online',3),('mw2','online',3),('mw3','online',3),('mw4','online',3),('mw5','online',3)`).Error)

	// 手动扩容
	ids, err := svc.ScaleUp(ctx, 2)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	// 非法数量
	_, err = svc.ScaleUp(ctx, 0)
	require.Error(t, err)
	_, err = svc.ScaleUp(ctx, 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "单次扩容数量")

	// 手动缩容
	require.NoError(t, svc.ScaleDown(ctx, ids))

	require.Error(t, svc.ScaleDown(ctx, nil))

	// 缩容低于最小值
	err = svc.ScaleDown(ctx, []string{"mw1", "mw2", "mw3", "mw4"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "低于最小值")

	// 状态 + 历史
	st := svc.GetStatus()
	require.NotNil(t, st)
	assert.False(t, st.IsRunning)

	history, err := svc.GetHistory(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, history, 2, "一次 scale_up + 一次 scale_down 成功事件（参数校验失败不落事件）")

	// 指标未就绪
	_, err = svc.GetWorkerStats(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "指标数据未就绪")

	// MonitorAndScale 无指标
	require.Error(t, svc.MonitorAndScale(ctx))

	// FindIdleContainers: mock stats CPU=10.5 不满足 <10 → 空
	idle, err := svc.FindIdleContainers(ctx, 5)
	require.NoError(t, err)
	assert.Empty(t, idle)
}

func TestScalingService_StartStop(t *testing.T) {
	db := newMetricsTestDB(t)
	cfg := DefaultScalingConfig()
	cfg.EnableMockDocker = true
	svc := NewScalingService(db, cfg, nil)

	// 未启用 → 直接返回
	require.NoError(t, svc.Start(context.Background()))

	cfg2 := DefaultScalingConfig()
	cfg2.Enabled = true
	cfg2.EnableMockDocker = true
	svc2 := NewScalingService(db, cfg2, nil)
	require.NoError(t, svc2.Start(context.Background()))
	assert.True(t, svc2.GetStatus().IsRunning)
	svc2.Stop()
	assert.False(t, svc2.GetStatus().IsRunning)
	svc2.Stop() // 二次 Stop 幂等（IsRunning 已 false 直接返回）

	// Docker 不健康 → 启动失败
	t.Setenv("MOCK_DOCKER_UNHEALTHY", "1")
	cfg3 := DefaultScalingConfig()
	cfg3.Enabled = true
	cfg3.EnableMockDocker = true
	svc3 := NewScalingService(db, cfg3, nil)
	err := svc3.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Docker 服务不可用")
}

func TestScalingService_SyncWorkersWithContainers(t *testing.T) {
	db := newMetricsTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_workers (id, status, docker_container_id) VALUES
		('sw1','online','mock-real'),
		('sw2','online','container-gone')`).Error)

	cfg := DefaultScalingConfig()
	svc := NewScalingService(db, cfg, nil).(*scalingServiceImpl)
	mock := NewMockDockerClient()
	_, err := mock.ScaleUp(context.Background(), 1)
	require.NoError(t, err)
	svc.dockerClient = mock

	require.NoError(t, svc.SyncWorkersWithContainers(context.Background()))
	var st2 string
	require.NoError(t, db.Raw(`SELECT status FROM sys_rpa_workers WHERE id = 'sw2'`).Scan(&st2).Error)
	assert.Equal(t, "offline", st2, "容器不存在的 worker 标记离线")
}

func TestDockerClient_HTTP(t *testing.T) {
	var created []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.40/containers/create":
			var body struct{ ID string `json:"Id"` }
			body.ID = fmt.Sprintf("cid-%d", len(created)+1)
			created = append(created, body.ID)
			_ = json.NewEncoder(w).Encode(body)
		case strings.HasPrefix(r.URL.Path, "/v1.40/containers/") && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case strings.HasPrefix(r.URL.Path, "/v1.40/containers/") && strings.Contains(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/v1.40/containers/json":
			_ = json.NewEncoder(w).Encode([]DockerContainer{
				{ID: "cid-1", Names: []string{"/rpa-worker-1"}, State: "running"},
				{ID: "other", Names: []string{"/unrelated"}, State: "running"},
			})
		case r.URL.Path == "/v1.40/_ping":
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/stats"):
			_, _ = w.Write([]byte(`{
				"cpu_stats": {"cpu_usage": {"total_usage": 300}, "system_usage": 1000, "online_cpus": 2},
				"precpu_stats": {"cpu_usage": {"total_usage": 100}},
				"memory_stats": {"usage": 512, "limit": 1024},
				"networks": {"eth0": {"rx_bytes": 10, "tx_bytes": 20}}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	port := 80
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		port, _ = strconv.Atoi(host[idx+1:])
		host = host[:idx]
	}

	client := NewDockerClient(&DockerConfig{
		DockerHost: host, DockerPort: port,
		ContainerName: "rpa-worker", ImageName: "img", NetworkName: "net",
	})
	ctx := context.Background()

	// 未配置 host → getBaseURL 错误
	noHost := NewDockerClient(&DockerConfig{})
	_, err := noHost.ScaleUp(ctx, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Docker 主机地址未配置")

	// ScaleUp 成功
	ids, err := client.ScaleUp(ctx, 2)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	// ScaleDown
	require.NoError(t, client.ScaleDown(ctx, ids))

	// ListContainers 过滤出 rpa 容器
	containers, err := client.ListContainers(ctx)
	require.NoError(t, err)
	require.Len(t, containers, 1)
	assert.Equal(t, "cid-1", containers[0].ID)

	// GetContainerStats
	stats, err := client.GetContainerStats(ctx, "cid-1")
	require.NoError(t, err)
	assert.Equal(t, 512, int(stats.MemoryUsage))
	assert.InDelta(t, 20.0, stats.CPUPercent, 0.01, "(300-100)/(1000-100)*100")
	assert.Equal(t, int64(10), stats.NetworkRx)

	// IsHealthy
	assert.True(t, client.IsHealthy(ctx))
}

func TestMockDockerClient(t *testing.T) {
	mc := NewMockDockerClient()
	ctx := context.Background()

	ids, err := mc.ScaleUp(ctx, 2)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	list, err := mc.ListContainers(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	stats, err := mc.GetContainerStats(ctx, ids[0])
	require.NoError(t, err)
	assert.Equal(t, ids[0], stats.ContainerID)
	assert.InDelta(t, 10.5, stats.CPUPercent, 0.001)

	assert.True(t, mc.IsHealthy(ctx))

	require.NoError(t, mc.ScaleDown(ctx, ids[:1]))
	list, err = mc.ListContainers(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}
