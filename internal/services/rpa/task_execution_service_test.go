package rpa

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-05: task_service.go + execution_service.go 测试。
// =====================================================================

// fakeXAddCache 实现 cache.Cache（嵌入接口nil兜底）+ DirectRedisXAdd，
// 走 publishTaskToRedis 的 MultiLevelCache 分支。
type fakeXAddCache struct {
	cache.Cache
	mu       sync.Mutex
	streams  []string
	payloads []string
	fail     bool
}

func (f *fakeXAddCache) DirectRedisXAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return assert.AnError
	}
	f.streams = append(f.streams, stream)
	if d, ok := values["data"].(string); ok {
		f.payloads = append(f.payloads, d)
	}
	return nil
}

// fakeNilRedisCache 带 getClient() 的假缓存（同包可满足未导出接口），
// getClient 返回 nil → publishTaskToRedis 报 "Redis客户端未初始化"。
type fakeNilRedisCache struct {
	cache.Cache
}

func (f *fakeNilRedisCache) getClient() *redis.Client { return nil }

// fakePlainCache 两个 Redis 分支都不满足的缓存（仅实现 Cache 接口）。
type fakePlainCache struct {
	cache.Cache
}

func newTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_tasks (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			name TEXT,
			description TEXT,
			script TEXT,
			timeout_seconds INTEGER DEFAULT 300,
			retry_count INTEGER DEFAULT 0,
			priority INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			tags TEXT
		)
	`).Error)
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
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_sessions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			credential_id TEXT,
			execution_id TEXT,
			target_system TEXT,
			target_url TEXT,
			access_token_encrypted TEXT,
			refresh_token_encrypted TEXT,
			cookies_encrypted TEXT,
			session_data_encrypted TEXT,
			expires_at DATETIME,
			is_valid BOOLEAN DEFAULT 1,
			invalid_reason TEXT
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_credentials (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			name TEXT,
			target_system TEXT,
			target_url TEXT,
			username_encrypted TEXT,
			password_encrypted TEXT,
			extra_data_encrypted TEXT,
			user_id TEXT,
			dept_id TEXT,
			is_shared BOOLEAN DEFAULT 0,
			status INTEGER DEFAULT 0,
			last_used_at DATETIME,
			last_login_at DATETIME,
			login_success_count INTEGER DEFAULT 0,
			login_fail_count INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

func TestTaskService_CRUD(t *testing.T) {
	db := newTaskTestDB(t)
	svc := NewTaskService(db, nil, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTaskRequest{
		Name:     "t1",
		Script:   []interface{}{map[string]interface{}{"type": "click", "selector": "#btn"}},
		Timeout:  60,
		MaxRetry: 2,
	}, "u1")
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "t1", got.TaskName)
	assert.True(t, got.IsEnabled())

	// GetByID 不存在
	_, err = svc.GetByID(ctx, "missing")
	require.Error(t, err)

	// Update
	require.NoError(t, svc.Update(ctx, &UpdateTaskRequest{
		ID: created.ID, Name: "t1v2", Script: []interface{}{map[string]interface{}{"type": "wait"}},
	}, "u1"))
	got, err = svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "t1v2", got.TaskName)

	// List
	page, err := svc.List(ctx, &TaskListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// List + Name 过滤走 task_name LIKE, 但 Task 模型列名是 name →
	// sqlite 报 no such column (D-12 记录的 quirk, 不修业务码)
	_, err = svc.List(ctx, &TaskListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, Name: "t1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task_name")

	// List + status/priority/tags 过滤
	st := 0
	pr := 0
	_, err = svc.List(ctx, &TaskListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, Status: &st, Priority: &pr, Tags: "x"})
	require.NoError(t, err)

	// Delete（软删除）
	require.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.GetByID(ctx, created.ID)
	require.Error(t, err)
}

func TestTaskService_Execute_Success(t *testing.T) {
	db := newTaskTestDB(t)
	c := &fakeXAddCache{}
	svc := NewTaskService(db, c, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTaskRequest{
		Name:   "nav-task",
		Script: []interface{}{
			map[string]interface{}{"type": "navigate", "value": "http://erp.local/home"},
			map[string]interface{}{"type": "fill", "selector": "#user", "value": "bob",
				"attributes": map[string]interface{}{"description": "填用户名", "x": 1}},
			map[string]interface{}{"type": "wait", "attributes": map[string]interface{}{"duration": float64(2)}},
		},
	}, "u1")
	require.NoError(t, err)

	exec, err := svc.Execute(ctx, &ExecuteTaskRequest{TaskID: created.ID, InputParams: map[string]interface{}{"a": 1}}, "u1")
	require.NoError(t, err)
	assert.Equal(t, string(rpamodels.RPAExecutionStatusPending), exec.Status)
	assert.Equal(t, "u1", exec.TriggeredBy)

	c.mu.Lock()
	require.Len(t, c.streams, 1)
	require.Len(t, c.payloads, 1)
	payload := c.payloads[0]
	c.mu.Unlock()

	assert.Contains(t, payload, `"targetUrl":"http://erp.local/home"`)
	assert.Contains(t, payload, `"url":"http://erp.local/home"`, "navigate 的 value 应转 params.url")
	assert.Contains(t, payload, `"value":"bob"`, "fill 的 value 应转 params.value")
	assert.Contains(t, payload, `"duration":2`, "wait 的 duration 应转 int")
	assert.Contains(t, payload, `"description":"填用户名"`)
	assert.Contains(t, payload, `"inputParams":{"a":1}`)
}

func TestTaskService_Execute_Failures(t *testing.T) {
	db := newTaskTestDB(t)
	ctx := context.Background()

	// 任务不存在
	svc := NewTaskService(db, &fakeXAddCache{}, nil)
	_, err := svc.Execute(ctx, &ExecuteTaskRequest{TaskID: "missing"}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务不存在")

	// 任务禁用
	created, err := svc.Create(ctx, &CreateTaskRequest{Name: "t", Script: []interface{}{map[string]interface{}{"type": "click"}}}, "u1")
	require.NoError(t, err)
	require.NoError(t, svc.Update(ctx, &UpdateTaskRequest{ID: created.ID, Status: 1}, "u1"))
	_, err = svc.Execute(ctx, &ExecuteTaskRequest{TaskID: created.ID}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "任务已禁用")

	// XADD 失败 → 删除执行记录并报错
	bad := &fakeXAddCache{fail: true}
	svc2 := NewTaskService(db, bad, nil)
	created2, err := svc2.Create(ctx, &CreateTaskRequest{Name: "t2", Script: []interface{}{map[string]interface{}{"type": "click"}}}, "u1")
	require.NoError(t, err)
	_, err = svc2.Execute(ctx, &ExecuteTaskRequest{TaskID: created2.ID}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "发布任务到Redis失败")
	var cnt int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_rpa_executions WHERE task_id = ?`, created2.ID).Scan(&cnt).Error)
	assert.Equal(t, int64(0), cnt, "发布失败应回滚执行记录")

	// getClient 返回 nil 的 RedisCache 分支 → "Redis客户端未初始化"
	svc3 := NewTaskService(db, &fakeNilRedisCache{}, nil)
	created3, err := svc3.Create(ctx, &CreateTaskRequest{Name: "t3", Script: []interface{}{map[string]interface{}{"type": "click"}}}, "u1")
	require.NoError(t, err)
	_, err = svc3.Execute(ctx, &ExecuteTaskRequest{TaskID: created3.ID}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Redis客户端未初始化")

	// 两个分支都不满足的缓存类型 → "缓存类型不支持Redis操作"
	svc4 := NewTaskService(db, &fakePlainCache{}, nil)
	created4, err := svc4.Create(ctx, &CreateTaskRequest{Name: "t4", Script: []interface{}{map[string]interface{}{"type": "click"}}}, "u1")
	require.NoError(t, err)
	_, err = svc4.Execute(ctx, &ExecuteTaskRequest{TaskID: created4.ID}, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缓存类型不支持Redis操作")
}

func TestTaskService_Execute_WithCredentialSession(t *testing.T) {
	db := newTaskTestDB(t)
	c := &fakeXAddCache{}
	credSvc := NewCredentialService(db, &fakeRPACipher{}, nil)
	svc := NewTaskService(db, c, credSvc)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTaskRequest{
		Name:   "login-task",
		Script: []interface{}{map[string]interface{}{"type": "navigate", "value": "http://erp.local"}},
	}, "u1")
	require.NoError(t, err)

	// 有会话
	_, err = credSvc.CreateSession(ctx, &rpamodels.SessionCreateRequest{
		CredentialID: "cred1", TargetSystem: "default", AccessToken: "at",
	})
	require.NoError(t, err)
	// 注: InputParams 必须非 nil —— publishTaskToRedis 对 Variables[__credentials] 赋值
	// 在凭证命中时发生, nil map 会 panic (D-12 记录的 quirk)
	_, err = svc.Execute(ctx, &ExecuteTaskRequest{TaskID: created.ID, CredentialID: "cred1", InputParams: map[string]interface{}{}}, "u1")
	require.NoError(t, err)
	c.mu.Lock()
	last := c.payloads[len(c.payloads)-1]
	c.mu.Unlock()
	assert.Contains(t, last, `"accessToken":"at"`, "会话命中时传递 token")

	// 无会话 → 凭证兜底
	_, err = credSvc.CreateCredential(ctx, &rpamodels.CredentialCreateRequest{
		Name: "c", TargetSystem: "default", Username: "u", Password: "p",
	}, "u1")
	require.NoError(t, err)
	_, err = svc.Execute(ctx, &ExecuteTaskRequest{TaskID: created.ID, CredentialID: "cred-none", InputParams: map[string]interface{}{}}, "u1")
	require.NoError(t, err)
	c.mu.Lock()
	last = c.payloads[len(c.payloads)-1]
	c.mu.Unlock()
	assert.Contains(t, last, "__credentials")
	assert.Contains(t, last, `"username":"u"`)
}

func TestTaskService_Helpers(t *testing.T) {
	actions := []rpamodels.ScriptAction{
		{Type: "click", Selector: "#a"},
		{Type: "navigate", Value: "http://x"},
	}
	assert.Equal(t, "http://x", extractURLFromScript(actions))
	assert.Equal(t, "", extractURLFromScript(nil))

	assert.Equal(t, "unknown", extractTargetSystem(""))
	assert.Equal(t, "default", extractTargetSystem("http://erp.local"))

	wa := convertToWorkerAction(rpamodels.ScriptAction{
		Type: "fill", Value: "v", Attributes: map[string]interface{}{
			"description": "d", "extra": 1,
		},
	}, 3)
	assert.Equal(t, "action_3", wa.ID)
	assert.Equal(t, "d", wa.Description)
	assert.Equal(t, "v", wa.Params["value"])
	assert.Equal(t, 1, wa.Params["extra"])
}

func TestExecutionService_CRUD(t *testing.T) {
	db := newTaskTestDB(t)
	svc := NewExecutionService(db, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, "t1", "task1", "u1")
	require.NoError(t, err)
	assert.Equal(t, string(rpamodels.RPAExecutionStatusPending), created.Status)
	assert.Equal(t, "manual", created.TriggerType)

	require.NoError(t, svc.Update(ctx, created.ID, map[string]interface{}{"worker_name": "w"}))
	require.NoError(t, svc.UpdateProgress(ctx, created.ID, 3, 9, "going"))
	require.NoError(t, svc.AddLog(ctx, created.ID, "line"))

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "w", got.WorkerName)
	assert.Equal(t, 3, got.Step)
	assert.Equal(t, 9, got.TotalSteps)
	assert.Contains(t, got.Logs, "line")
	assert.Contains(t, got.Logs, "going")

	// AddLog 无 message 的 UpdateProgress
	require.NoError(t, svc.UpdateProgress(ctx, created.ID, 4, 9, ""))

	// Cancel: pending → cancelled
	require.NoError(t, svc.Cancel(ctx, created.ID))
	got, err = svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, string(rpamodels.RPAExecutionStatusCancelled), got.Status)
	assert.NotNil(t, got.EndTime)

	// Cancel 已结束 → 报错
	require.NoError(t, svc.Update(ctx, created.ID, map[string]interface{}{"status": "success"}))
	err = svc.Cancel(ctx, created.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "执行已结束")

	// Cancel 不存在
	require.Error(t, svc.Cancel(ctx, "missing"))

	// List + 过滤
	_, err = svc.Create(ctx, "t1", "task1", "u1")
	require.NoError(t, err)
	st := 0
	page, err := svc.List(ctx, &ExecutionListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, TaskID: "t1", Status: &st})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total, "状态过滤: e1 已 cancelled, 新建为 pending")

	wid := "w-9"
	page, err = svc.List(ctx, &ExecutionListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, WorkerID: wid})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total)

	// PublishProgress: noticeHub 为 nil → 静默成功
	require.NoError(t, svc.PublishProgress(ctx, &ProgressUpdate{ExecutionID: created.ID, Status: "success"}))
}

func TestErrorHandlingService_Strategies(t *testing.T) {
	db := newTaskTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_executions (id, task_id, status, logs, created_at) VALUES ('e1', 't1', 'running', '', ?)`, now).Error)

	svc := NewErrorHandlingService(db, nil, nil)
	ctx := context.Background()
	boom := assert.AnError

	// ignore / skip / abort / rollback / fallback
	for _, strategy := range []ErrorHandlingStrategy{ErrorStrategyIgnore, ErrorStrategySkip, ErrorStrategyAbort, ErrorStrategyRollback} {
		res, err := svc.HandleError(ctx, &ErrorHandleRequest{
			ExecutionID: "e1", StepIndex: 2, Error: boom,
			Config: &ErrorHandlingConfig{Strategy: strategy},
		})
		require.NoError(t, err)
		assert.True(t, res.Handled)
	}

	// abort/rollback 后执行记录标记失败
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM sys_rpa_executions WHERE id = 'e1'`).Scan(&status).Error)
	assert.Equal(t, "failed", status)

	// ignore → continue + NextStepIndex
	res, err := svc.HandleError(ctx, &ErrorHandleRequest{
		ExecutionID: "e1", StepIndex: 2, Error: boom,
		Config: &ErrorHandlingConfig{Strategy: ErrorStrategyIgnore},
	})
	require.NoError(t, err)
	assert.Equal(t, "continue", res.Action)
	require.NotNil(t, res.NextStepIndex)
	assert.Equal(t, 3, *res.NextStepIndex)

	// rollback: 执行记录缺失 → 报错
	_, err = svc.HandleError(ctx, &ErrorHandleRequest{
		ExecutionID: "missing", Error: boom,
		Config: &ErrorHandlingConfig{Strategy: ErrorStrategyRollback},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取执行记录失败")

	// 未知策略 → handled=false
	res, err = svc.HandleError(ctx, &ErrorHandleRequest{
		Config: &ErrorHandlingConfig{Strategy: ErrorHandlingStrategy("weird")},
	})
	require.NoError(t, err)
	assert.False(t, res.Handled)
}

func TestErrorHandlingService_Retry(t *testing.T) {
	db := newTaskTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_executions (id, task_id, status, logs, created_at) VALUES ('e1', 't1', 'running', '', ?)`, now).Error)

	svc := NewErrorHandlingService(db, nil, nil)
	ctx := context.Background()

	// 未配置策略
	_, err := svc.ExecuteRetry(ctx, &RetryRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "重试策略未配置")

	// 超过最大次数
	_, err = svc.ExecuteRetry(ctx, &RetryRequest{Attempt: 3, Policy: &RetryPolicy{MaxAttempts: 2}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "超过最大重试次数")

	// 错误类型不匹配 RetryOn
	res, err := svc.ExecuteRetry(ctx, &RetryRequest{
		Policy:    &RetryPolicy{MaxAttempts: 3, InitialDelay: time.Millisecond, RetryOn: []string{"TimeoutError"}},
		LastError: notTimeoutErr{}, // %T 不含 TimeoutError
	})
	require.NoError(t, err)
	assert.False(t, res.ShouldRetry)

	// 匹配 RetryOn (%T 含 *errors.errorString)
	res, err = svc.ExecuteRetry(ctx, &RetryRequest{
		Policy:    &RetryPolicy{MaxAttempts: 3, InitialDelay: time.Millisecond, RetryOn: []string{"errorString"}},
		LastError: assert.AnError,
	})
	require.NoError(t, err)
	assert.True(t, res.ShouldRetry)
	assert.Equal(t, 1, res.Attempt)

	// ctx 取消中断延迟等待
	// 注: RetryPolicy.MaxDelay 零值会把任意 InitialDelay 钳为 0 (delay > 0 即触发),
	// 导致 time.After(0) 与已取消的 ctx.Done() 在 select 中随机竞态 —
	// 测试必须显式给 MaxDelay (D-12 记录的 quirk)。
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = svc.ExecuteRetry(canceled, &RetryRequest{
		Policy:    &RetryPolicy{MaxAttempts: 3, InitialDelay: 5 * time.Second, MaxDelay: 10 * time.Second},
		LastError: assert.AnError,
	})
	require.ErrorIs(t, err, context.Canceled)

	// HandleError retry 策略集成
	res2, err := svc.HandleError(ctx, &ErrorHandleRequest{
		ExecutionID: "e1", Error: assert.AnError,
		Config: &ErrorHandlingConfig{Strategy: ErrorStrategyRetry, RetryPolicy: &RetryPolicy{
			MaxAttempts: 1, InitialDelay: time.Millisecond,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "retry", res2.Action)
}

// notTimeoutErr 类型名不含 TimeoutError, 用于 RetryOn 不匹配路径。
type notTimeoutErr struct{}

func (notTimeoutErr) Error() string { return "plain failure" }

func TestErrorHandlingService_CalculateDelay(t *testing.T) {
	svc := &errorHandlingServiceImpl{}
	p := &RetryPolicy{InitialDelay: 100 * time.Millisecond, MaxDelay: time.Second}

	assert.Equal(t, 100*time.Millisecond, svc.calculateDelay(1, p))

	linear := &RetryPolicy{InitialDelay: 100 * time.Millisecond, BackoffType: BackoffTypeLinear, MaxDelay: time.Second}
	assert.Equal(t, 300*time.Millisecond, svc.calculateDelay(3, linear))

	exp := &RetryPolicy{InitialDelay: 100 * time.Millisecond, BackoffType: BackoffTypeExponential, MaxDelay: 350 * time.Millisecond}
	assert.Equal(t, 200*time.Millisecond, svc.calculateDelay(2, exp))
	assert.Equal(t, 350*time.Millisecond, svc.calculateDelay(5, exp), "不超过 MaxDelay")

	capped := &RetryPolicy{InitialDelay: 100 * time.Millisecond, MaxDelay: 50 * time.Millisecond}
	assert.Equal(t, 50*time.Millisecond, svc.calculateDelay(1, capped))
}

func TestErrorHandlingService_Fallback(t *testing.T) {
	db := newTaskTestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_executions (id, task_id, status, logs, created_at) VALUES ('e1', 't1', 'running', '', ?)`, now).Error)

	svc := NewErrorHandlingService(db, nil, nil)
	ctx := context.Background()

	// 未配置降级动作
	_, err := svc.ExecuteFallback(ctx, &FallbackRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "降级动作未配置")

	// 动作非 JSON
	bad := json.RawMessage(`{bad`)
	_, err = svc.ExecuteFallback(ctx, &FallbackRequest{FallbackAction: &bad, Variables: map[string]interface{}{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析降级动作失败")

	// 正常降级
	action := json.RawMessage(`{"type":"click"}`)
	res, err := svc.ExecuteFallback(ctx, &FallbackRequest{
		ExecutionID:    "e1",
		StepIndex:      1,
		FallbackAction: &action,
		Variables:      map[string]interface{}{},
		Error:          assert.AnError,
	})
	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, true, res.Variables["fallback"])
	assert.Equal(t, assert.AnError.Error(), res.Variables["originalError"])

	// HandleError fallback 策略集成
	res2, err := svc.HandleError(ctx, &ErrorHandleRequest{
		ExecutionID: "e1", Error: assert.AnError,
		Config:      &ErrorHandlingConfig{Strategy: ErrorStrategyFallback, FallbackAction: &action},
		Variables:   map[string]interface{}{},
	})
	require.NoError(t, err)
	assert.True(t, res2.Handled)
	assert.Equal(t, "fallback", res2.Action)
}

func TestContainsHelper(t *testing.T) {
	assert.True(t, contains("hello", "ell"))
	assert.True(t, contains("abc", "abc"))
	assert.True(t, contains("abc", ""))
	assert.False(t, contains("ab", "abcdef"))
	assert.False(t, contains("", "x"))
}
