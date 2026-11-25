package rpa

import (
	"context"
	"sync"
	"testing"

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

	// List + Name 过滤（CONTRACT-03 / D3）：真实列是 name，
	// 旧实现写 task_name 会报 no such column，现修复后应命中。
	page, err = svc.List(ctx, &TaskListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, Name: "t1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total, "任务名过滤应命中真实 name 列")

	// List + status/priority/tags 过滤（status 已是任务枚举类型）
	st := rpamodels.TaskStatusEnabled
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
		Name: "nav-task",
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
	disabled := int(rpamodels.TaskStatusDisabled)
	require.NoError(t, svc.Update(ctx, &UpdateTaskRequest{ID: created.ID, Status: &disabled}, "u1"))
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
		CredentialID: "cred1", TargetSystem: "erp", AccessToken: "at",
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
	assert.Contains(t, last, "__sessionData", "RUNTIME-02: 会话注入 __sessionData")

	// 无会话 → 凭证兜底
	_, err = credSvc.CreateCredential(ctx, &rpamodels.CredentialCreateRequest{
		Name: "c", TargetSystem: "erp", Username: "u", Password: "p",
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
	assert.Equal(t, "erp", extractTargetSystem("http://erp.local"))
	assert.Equal(t, "default", extractTargetSystem("http://192.168.1.5:8080/"))
	assert.Equal(t, "erp", extractTargetSystem("http://ERP.Company.com/path"))

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
	require.NoError(t, svc.Cancel(ctx, created.ID, "tester", ""))
	got, err = svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, string(rpamodels.RPAExecutionStatusCancelled), got.Status)
	assert.NotNil(t, got.EndTime)

	// Cancel 已结束 → 报错
	require.NoError(t, svc.Update(ctx, created.ID, map[string]interface{}{"status": "success"}))
	err = svc.Cancel(ctx, created.ID, "tester", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无法取消")
	assert.Contains(t, err.Error(), "终态不可覆盖")

	// Cancel 不存在
	require.Error(t, svc.Cancel(ctx, "missing", "tester", ""))

	// List + 过滤
	_, err = svc.Create(ctx, "t1", "task1", "u1")
	require.NoError(t, err)
	// 状态过滤按字符串（CONTRACT-02）：t1 新建记录为 pending，筛 timeout 应 0 命中
	st := string(rpamodels.RPAExecutionStatusTimeout)
	page, err := svc.List(ctx, &ExecutionListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, TaskID: "t1", Status: &st})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total, "状态过滤: t1 新建记录为 pending，按 timeout 筛应无命中")

	wid := "w-9"
	page, err = svc.List(ctx, &ExecutionListParams{ListParams: ListParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}}, WorkerID: wid})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total)

	// PublishProgress: noticeHub 为 nil → 静默成功
	require.NoError(t, svc.PublishProgress(ctx, &ProgressUpdate{ExecutionID: created.ID, Status: "success"}))
}
