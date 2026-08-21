package rpa

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"sync"
	"testing"
	"time"

	gsqlite "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// =====================================================================
// Phase 74-05: worker_service.go 测试。
//
// 含 f0d0a1f PK NULL 回归测试: Register 走原生 SQL INSERT, 不经过 GORM
// BeforeCreate 钩子链, id 必须由 SQL 内 gen_random_uuid() 显式生成。
// sqlite 没有该函数, 通过 glebarez/go-sqlite 的全局标量函数注册注入
// 等价实现 (依赖已在 go.mod indirect 列表中, 无需改 go.mod)。
// 表定义 id NOT NULL — 若有人把 id 从 INSERT 列清单移除, sqlite 将收到
// NULL 主键 → 约束违规 → 测试失败, 锁定 23502 修复。
// =====================================================================

var registerSQLiteFuncsOnce sync.Once

// registerPGCompatFunctions 向 glebarez sqlite 注册 PG 兼容函数。
func registerPGCompatFunctions(t *testing.T) {
	t.Helper()
	registerSQLiteFuncsOnce.Do(func() {
		require.NoError(t, gsqlite.RegisterScalarFunction("gen_random_uuid", 0,
			func(ctx *gsqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				return uuid.NewString(), nil
			}))
		require.NoError(t, gsqlite.RegisterScalarFunction("NOW", 0,
			func(ctx *gsqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				return time.Now().Format("2006-01-02 15:04:05"), nil
			}))
	})
}

func newWorkerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	registerPGCompatFunctions(t)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_workers (
			id TEXT PRIMARY KEY NOT NULL,
			worker_name TEXT,
			worker_id TEXT NOT NULL UNIQUE,
			ip_address TEXT,
			port INTEGER DEFAULT 0,
			status TEXT DEFAULT 'offline',
			capabilities TEXT,
			max_concurrency INTEGER DEFAULT 3,
			current_tasks INTEGER DEFAULT 0,
			last_heartbeat INTEGER,
			docker_container_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

func newExecutionsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_executions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			task_id TEXT,
			task_name TEXT,
			worker_id TEXT,
			worker_name TEXT,
			status TEXT DEFAULT 'pending',
			start_time DATETIME,
			end_time DATETIME,
			duration INTEGER,
			progress_current INTEGER DEFAULT 0,
			progress_total INTEGER DEFAULT 0,
			screenshots TEXT,
			logs TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			triggered_by TEXT,
			trigger_type TEXT
		)
	`).Error)
	return db
}

// mockRPAExecutionService ExecutionService 接口 mock（*Func 注入）。
type mockRPAExecutionService struct {
	CreateFunc      func(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error)
	UpdateFunc      func(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateProgressF func(ctx context.Context, id string, current, total int, message string) error
	AddLogFunc      func(ctx context.Context, id string, log string) error
	CancelFunc      func(ctx context.Context, id string) error
	ListFunc        func(ctx context.Context, params *ExecutionListParams) (*PageResult, error)
	GetByIDFunc     func(ctx context.Context, id string) (*rpamodels.Execution, error)
	PublishProgressFunc func(ctx context.Context, update *ProgressUpdate) error
	StatisticsFunc  func(ctx context.Context) (*ExecutionStatisticsResult, error)
}

func (m *mockRPAExecutionService) Create(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, taskID, taskName, triggeredBy)
	}
	return nil, nil
}
func (m *mockRPAExecutionService) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, updates)
	}
	return nil
}
func (m *mockRPAExecutionService) UpdateProgress(ctx context.Context, id string, current, total int, message string) error {
	if m.UpdateProgressF != nil {
		return m.UpdateProgressF(ctx, id, current, total, message)
	}
	return nil
}
func (m *mockRPAExecutionService) AddLog(ctx context.Context, id string, log string) error {
	if m.AddLogFunc != nil {
		return m.AddLogFunc(ctx, id, log)
	}
	return nil
}
func (m *mockRPAExecutionService) Cancel(ctx context.Context, id string) error {
	if m.CancelFunc != nil {
		return m.CancelFunc(ctx, id)
	}
	return nil
}
func (m *mockRPAExecutionService) List(ctx context.Context, params *ExecutionListParams) (*PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return &PageResult{}, nil
}
func (m *mockRPAExecutionService) GetByID(ctx context.Context, id string) (*rpamodels.Execution, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockRPAExecutionService) PublishProgress(ctx context.Context, update *ProgressUpdate) error {
	if m.PublishProgressFunc != nil {
		return m.PublishProgressFunc(ctx, update)
	}
	return nil
}
func (m *mockRPAExecutionService) Statistics(ctx context.Context) (*ExecutionStatisticsResult, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx)
	}
	return &ExecutionStatisticsResult{}, nil
}

// TestWorkerService_Register_PKNotNullRegression f0d0a1f 回归:
// Register 的原生 INSERT 必须显式写 id 列(gen_random_uuid()),
// 表 NOT NULL 约束确保移除 id 列时本测试失败。
//
// 注: RETURNING * 在 sqlite 下扫描 capabilities(TEXT→json.RawMessage) 会报
// Scan 错误（PG jsonb 正常），属测试环境差异；INSERT 本身已成功落库，
// 回归断言以 DB 行的 id 为准（若 id 从 INSERT 列清单移除 → NOT NULL 违规
// → 无行落库 → 本测试失败）。
func TestWorkerService_Register_PKNotNullRegression(t *testing.T) {
	db := newWorkerTestDB(t)
	svc := NewWorkerService(db, nil, t.TempDir())

	_, _ = svc.Register(context.Background(), &WorkerRegisterRequest{
		WorkerID:       "wk-001",
		Name:           "node-1",
		Host:           "127.0.0.1",
		Port:           9001,
		MaxConcurrency: 5,
		Capabilities:   []string{"chromium"},
	})

	var id string
	require.NoError(t, db.Raw(`SELECT id FROM sys_rpa_workers WHERE worker_id = 'wk-001'`).Scan(&id).Error)
	require.NotEmpty(t, id, "id 必须由 SQL 显式生成, 不得为空 (23502 回归)")
	_, parseErr := uuid.Parse(id)
	require.NoError(t, parseErr, "生成的 id 应为 UUID")

	// 同 worker_id 再注册 → ON CONFLICT DO UPDATE 走更新路径
	_, _ = svc.Register(context.Background(), &WorkerRegisterRequest{
		WorkerID: "wk-001",
		Name:     "node-1-renamed",
		Host:     "127.0.0.2",
		Port:     9002,
	})

	var count int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_rpa_workers WHERE worker_id = 'wk-001'`).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "ON CONFLICT 不应产生重复行")

	var name string
	require.NoError(t, db.Raw(`SELECT worker_name FROM sys_rpa_workers WHERE worker_id = 'wk-001'`).Scan(&name).Error)
	assert.Equal(t, "node-1-renamed", name)
}

func TestWorkerService_Heartbeat(t *testing.T) {
	db := newWorkerTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_workers (id, worker_name, worker_id, status, created_at) VALUES ('w1', 'n', 'wk-1', 'online', ?)`, now).Error)

	svc := NewWorkerService(db, nil, "")
	require.NoError(t, svc.Heartbeat(context.Background(), &WorkerHeartbeatRequest{
		WorkerID:     "wk-1",
		CurrentTasks: 2,
		Status:       "busy",
	}))

	var status string
	var tasks int
	require.NoError(t, db.Raw(`SELECT status, current_tasks FROM sys_rpa_workers WHERE worker_id = 'wk-1'`).Row().Scan(&status, &tasks))
	assert.Equal(t, "busy", status)
	assert.Equal(t, 2, tasks)
}

func TestWorkerService_Progress(t *testing.T) {
	db := newExecutionsTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_executions (id, task_id, task_name, status, created_at) VALUES ('e1', 't1', 'task', 'running', ?)`, now).Error)

	published := make(chan *ProgressUpdate, 1)
	execMock := &mockRPAExecutionService{
		PublishProgressFunc: func(ctx context.Context, u *ProgressUpdate) error {
			published <- u
			return nil
		},
	}
	svc := NewWorkerService(db, execMock, t.TempDir())

	// log + 状态 running + 截图(base64 png)
	png := base64.StdEncoding.EncodeToString([]byte("fakepng"))
	require.NoError(t, svc.Progress(context.Background(), &WorkerProgressRequest{
		ExecutionID:     "e1",
		ProgressCurrent: 2,
		ProgressTotal:   10,
		Message:         "step done",
		Status:          "running",
		Log:             "clicking",
		Screenshot:      png,
	}))

	var status string
	var logs string
	var shots string
	require.NoError(t, db.Raw(`SELECT status, logs, screenshots FROM sys_rpa_executions WHERE id = 'e1'`).Row().Scan(&status, &logs, &shots))
	assert.Equal(t, "running", status)
	assert.Contains(t, logs, "clicking", "log 字段应追加 FormatLog 条目")
	assert.Contains(t, shots, "rpa/screenshots/", "截图应保存为文件路径")

	// 异步 WebSocket 发布
	select {
	case u := <-published:
		assert.Equal(t, "e1", u.ExecutionID)
		assert.Equal(t, 2, u.Step)
		assert.Equal(t, 10, u.Total)
		assert.NotEmpty(t, u.ScreenshotURL)
	case <-time.After(2 * time.Second):
		t.Fatal("PublishProgress 未被调用")
	}

	// failed 状态 + message → error_message 更新
	require.NoError(t, svc.Progress(context.Background(), &WorkerProgressRequest{
		ExecutionID: "e1",
		Status:      "failed",
		Message:     "boom",
	}))
	var errMsg string
	require.NoError(t, db.Raw(`SELECT error_message FROM sys_rpa_executions WHERE id = 'e1'`).Scan(&errMsg).Error)
	assert.Equal(t, "boom", errMsg)

	// 执行记录不存在
	err := svc.Progress(context.Background(), &WorkerProgressRequest{ExecutionID: "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "执行记录不存在")

	// 非法 base64 截图 → 记录日志但不中断
	require.NoError(t, svc.Progress(context.Background(), &WorkerProgressRequest{
		ExecutionID: "e1",
		Screenshot:  "data:image/png;base64,!!!not-base64!!!",
	}))
}

func TestWorkerService_ListAndGetters(t *testing.T) {
	db := newWorkerTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	for i := 1; i <= 3; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_rpa_workers (id, worker_name, worker_id, status, current_tasks, max_concurrency, last_heartbeat, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("w%d", i), fmt.Sprintf("node-%d", i), fmt.Sprintf("wk-%d", i), "online", 0, 3, time.Now().Unix(), now,
		).Error)
	}
	svc := NewWorkerService(db, nil, "")

	result, err := svc.List(context.Background(), &WorkerListParams{ListParams: ListParams{}, Name: "node"})
	// Name 过滤走 worker_name LIKE; 注意 WorkerListParams 无 Current/PageSize 默认值, 传 0 会 Limit(0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)

	// GetByID
	w, err := svc.GetByID(context.Background(), "w1")
	require.NoError(t, err)
	assert.Equal(t, "wk-1", w.WorkerID)

	_, err = svc.GetByID(context.Background(), "nope")
	require.Error(t, err)

	// GetAvailable: current_tasks < max_concurrency 且 online
	require.NoError(t, db.Exec(`UPDATE sys_rpa_workers SET current_tasks = 3 WHERE id = 'w1'`).Error)
	avail, err := svc.GetAvailable(context.Background())
	require.NoError(t, err)
	assert.Len(t, avail, 2)

	// Offline
	require.NoError(t, svc.Offline(context.Background(), "w2"))
	var st string
	require.NoError(t, db.Raw(`SELECT status FROM sys_rpa_workers WHERE id = 'w2'`).Scan(&st).Error)
	assert.Equal(t, "offline", st)

	// CheckOfflineWorkers: 心跳超时的 online worker 被标记离线
	stale := time.Now().Unix() - 300
	require.NoError(t, db.Exec(`UPDATE sys_rpa_workers SET status = 'online', last_heartbeat = ? WHERE id = 'w3'`, stale).Error)
	require.NoError(t, svc.CheckOfflineWorkers(context.Background(), 120))
	require.NoError(t, db.Raw(`SELECT status FROM sys_rpa_workers WHERE id = 'w3'`).Scan(&st).Error)
	assert.Equal(t, "offline", st)
}
