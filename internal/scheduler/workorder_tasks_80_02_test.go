package scheduler

// Phase 80 Plan 02 — workorder_tasks.go 测试
// 覆盖:4 纯函数表驱动 + GlobalNoticeHub nil-guard/stub + assignWorkOrderHandler 4 分支
// + syncWorkOrderJob 三态 + Disable/Enable + getTodayDutyPerson
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。
// 纪律:status 断言只引用 models.* 常量,禁裸 0/1。

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ============================================================================
// Helper fixtures (8002 后缀)
// ============================================================================

// newSchedDB8002 sqlite 文件库(t.TempDir)+ AutoMigrate sys_job/sys_job_log。
func newSchedDB8002(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "wot8002.db")
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

// newDutyDB8002 sqlite 文件库 + duty/workorder 相关表 DDL。
func newDutyDB8002(t *testing.T) *gorm.DB {
	t.Helper()
	db := newSchedDB8002(t)
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
		`CREATE TABLE IF NOT EXISTS sys_work_order (
			id TEXT PRIMARY KEY,
			title TEXT,
			work_order_no TEXT,
			category_id TEXT,
			type TEXT,
			priority INTEGER,
			status INTEGER,
			description TEXT,
			submitter_id TEXT,
			assignee_id TEXT,
			is_auto_assigned INTEGER,
			assign_strategy TEXT,
			duty_type TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		require.NoError(t, db.Exec(stmt).Error)
	}
	// AutoMigrate sys_periodic_workorder_template to ensure correct column names
	require.NoError(t, db.AutoMigrate(&models.PeriodicWorkOrderTemplate{}))
	return db
}

// stubNoticeHub8002 记录 BroadcastToUsers 调用。
type stubNoticeHub8002 struct {
	mu      sync.Mutex
	calls   int
	lastIDs []string
	lastMsg interface{}
}

func (h *stubNoticeHub8002) BroadcastToUsers(userIDs []string, message interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls++
	h.lastIDs = append([]string(nil), userIDs...)
	h.lastMsg = message
}

func (h *stubNoticeHub8002) snapshot() (int, []string, interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls, h.lastIDs, h.lastMsg
}

// newScheduler8002 装配 scheduler + stub logger (复用 newDutyDB8002 以共享同一 DB)。
func newScheduler8002(t *testing.T) (*Scheduler, *gorm.DB) {
	t.Helper()
	db := newDutyDB8002(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	return s, db
}

// ============================================================================
// TestWot8002_PureFuncs_Table — 4 纯函数表驱动
// ============================================================================

// TestWot8002_PureFuncs_GenerateWorkOrderNo 工单号格式前缀+时间戳+UUID后缀。
func TestWot8002_PureFuncs_GenerateWorkOrderNo(t *testing.T) {
	no := generateWorkOrderNo()
	assert.True(t, strings.HasPrefix(no, "WO"), "应前缀 WO")
	assert.Len(t, no, 22, "WO+8位日期+12位hex后缀(实际 22)")
	// 二次调用保证唯一性
	no2 := generateWorkOrderNo()
	assert.NotEqual(t, no, no2, "两次调用应生成不同工单号")
}

// TestWot8002_PureFuncs_ReplaceVariables 变量替换表驱动。
func TestWot8002_PureFuncs_ReplaceVariables(t *testing.T) {
	ts := time.Date(2026, 8, 27, 14, 30, 0, 0, time.Local)

	cases := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "全变量",
			title: "工单-{date}-{year}-{month}-{day}-{weekday}-{hour}-{minute}",
			want:  "工单-2026-08-27-2026-08-27-周四-14-30",
		},
		{
			name:  "无变量",
			title: "普通工单",
			want:  "普通工单",
		},
		{
			name:  "空串",
			title: "",
			want:  "",
		},
		{
			name:  "部分变量",
			title: "工单-{date}生成",
			want:  "工单-2026-08-27生成",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceVariables(tc.title, ts)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestWot8002_PureFuncs_GetWeekdayName 7 天全表驱动。
func TestWot8002_PureFuncs_GetWeekdayName(t *testing.T) {
	cases := []struct {
		wd    time.Weekday
		want  string
	}{
		{time.Sunday, "周日"},
		{time.Monday, "周一"},
		{time.Tuesday, "周二"},
		{time.Wednesday, "周三"},
		{time.Thursday, "周四"},
		{time.Friday, "周五"},
		{time.Saturday, "周六"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := getWeekdayName(tc.wd)
			assert.Equal(t, tc.want, got)
		})
	}
	// 越界:Go time.Weekday 范围 0-6,无越界分支
}

// TestWot8002_PureFuncs_CalculateNextRunTimeCron 合法/非法/空表达式。
func TestWot8002_PureFuncs_CalculateNextRunTimeCron(t *testing.T) {
	// 合法表达式
	next, err := calculateNextRunTimeCron("0 0 8 * * *")
	require.NoError(t, err)
	assert.True(t, next.After(time.Now()), "下次运行应在未来")

	// 非法表达式
	_, err = calculateNextRunTimeCron("not-a-cron")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析Cron表达式失败")

	// 空表达式
	_, err = calculateNextRunTimeCron("")
	require.Error(t, err)
}

// ============================================================================
// TestWot8002_NoticeHub — nil-guard 与 stub 注入两态
// ============================================================================

// TestWot8002_NoticeHub_NilGuard GlobalNoticeHub nil → sendWorkOrderNotification 不 panic。
func TestWot8002_NoticeHub_NilGuard(t *testing.T) {
	orig := GlobalNoticeHub
	t.Cleanup(func() { SetNoticeHub(orig) })
	SetNoticeHub(nil)

	// nil-guard 路径(:242-247)不应 panic
	sendWorkOrderNotification("wo-id", "user-1", "测试工单")
}

// TestWot8002_NoticeHub_Stub GlobalNoticeHub stub → BroadcastToUsers 被调且参数含 workOrderId。
func TestWot8002_NoticeHub_Stub(t *testing.T) {
	orig := GlobalNoticeHub
	t.Cleanup(func() { SetNoticeHub(orig) })

	hub := &stubNoticeHub8002{}
	SetNoticeHub(hub)

	sendWorkOrderNotification("wo-8002-001", "user-8002", "周期工单测试")
	calls, ids, msg := hub.snapshot()
	assert.Equal(t, 1, calls, "应调用一次")
	assert.Equal(t, []string{"user-8002"}, ids)
	require.NotNil(t, msg)
	raw := fmt.Sprintf("%+v", msg)
	assert.Contains(t, raw, "wo-8002-001")
	assert.Contains(t, raw, "周期工单测试")
}

// ============================================================================
// TestWot8002_AssignWorkOrderHandler — 4 分支表驱动
// ============================================================================

// TestWot8002_AssignWorkOrderHandler_Branches assignWorkOrderHandler 派单分支。
func TestWot8002_AssignWorkOrderHandler_Branches(t *testing.T) {
	db := newDutyDB8002(t)
	today := time.Now().Format("2006-01-02")

	// 准备:system 用户 + 模板
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('system-id', 'system')`).Error)

	template := &models.PeriodicWorkOrderTemplate{
		BaseModel:        models.BaseModel{ID: "tpl-8002"},
		TemplateName:     "测试模板",
		WorkOrderTitle:   "测试工单",
		AssignType:       models.PeriodicAssignTypeManual,
		AssignTargetID:   strPtr("assignee-1"),
	}
	workOrder := &models.WorkOrder{}

	// 分支 1:Manual — 有 AssignTargetID
	got, err := assignWorkOrderHandler(db, template, workOrder)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "assignee-1", *got)
	assert.False(t, workOrder.IsAutoAssigned)

	// 分支 1b:Manual — 无 AssignTargetID
	template.AssignTargetID = nil
	workOrder2 := &models.WorkOrder{}
	got2, err := assignWorkOrderHandler(db, template, workOrder2)
	require.NoError(t, err)
	assert.Nil(t, got2)

	// 分支 2:DutyPool — 值班池有当日值班
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_schedule (id, user_id, schedule_date, status) VALUES ('sch-1', 'u-duty', ?, 0)`, today).Error)
	template2 := &models.PeriodicWorkOrderTemplate{
		BaseModel:  models.BaseModel{ID: "tpl-dpool"},
		AssignType: models.PeriodicAssignTypeDutyPool,
	}
	workOrder3 := &models.WorkOrder{}
	got3, err := assignWorkOrderHandler(db, template2, workOrder3)
	require.NoError(t, err)
	assert.NotNil(t, got3)
	assert.Equal(t, "duty_pool", workOrder3.DutyType)
	assert.Equal(t, "assign_one", workOrder3.AssignStrategy)

	// 分支 3:Rotation — 暂未实现
	template3 := &models.PeriodicWorkOrderTemplate{
		BaseModel:  models.BaseModel{ID: "tpl-rot"},
		AssignType: models.PeriodicAssignTypeRotation,
	}
	workOrder4 := &models.WorkOrder{}
	got4, err := assignWorkOrderHandler(db, template3, workOrder4)
	require.NoError(t, err)
	assert.Nil(t, got4, "轮询未实现应返回 nil")

	// 分支 4:default — 未知类型
	template4 := &models.PeriodicWorkOrderTemplate{
		BaseModel:  models.BaseModel{ID: "tpl-unknown"},
		AssignType: "unknown_type",
	}
	workOrder5 := &models.WorkOrder{}
	got5, err := assignWorkOrderHandler(db, template4, workOrder5)
	require.NoError(t, err)
	assert.Nil(t, got5)
}

// ============================================================================
// TestWot8002_SyncCreateUpdateJob — syncWorkOrderJob 三态 + Disable/Enable
// ============================================================================

// TestWot8002_SyncCreateUpdateJob_Create syncWorkOrderJob:job 不存在→create。
func TestWot8002_SyncCreateUpdateJob_Create(t *testing.T) {
	s, db := newScheduler8002(t)

	// 设置 GlobalDB (syncWorkOrderJob 内部间接依赖 GlobalDB)
	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	// 模板有 job_id=nil → create 分支
	template := &models.PeriodicWorkOrderTemplate{
		BaseModel:        models.BaseModel{ID: "tpl-new"},
		TemplateName:     "新建任务模板",
		WorkOrderTitle:   "工单-{date}",
		CronExpression:   "0 0 8 * * *",
		NotifyAssignee:   true,
	}
	require.NoError(t, db.Create(template).Error)

	err := syncWorkOrderJob(db, s, template)
	require.NoError(t, err)

	// 验证 job 行
	var job models.Job
	require.NoError(t, db.Where("job_name = ?", "periodic_workorder_tpl-new").First(&job).Error)
	assert.Equal(t, models.JobStatusNormal, job.Status, "新建 job 应为正常态")
	assert.NotEmpty(t, job.ID)
}

// TestWot8002_SyncCreateUpdateJob_Update syncWorkOrderJob:job 存在且表达式不变→update。
func TestWot8002_SyncCreateUpdateJob_Update(t *testing.T) {
	s, db := newScheduler8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	// 已有 job
	existingJob := &models.Job{
		JobName:        "periodic_workorder_tpl-upd",
		JobGroup:       "PERIODIC_WORKORDER",
		InvokeTarget:   "periodic_workorder_create:tpl-upd",
		CronExpression: "0 0 8 * * *",
		Status:         models.JobStatusNormal,
	}
	require.NoError(t, db.Create(existingJob).Error)

	template := &models.PeriodicWorkOrderTemplate{
		BaseModel:        models.BaseModel{ID: "tpl-upd"},
		TemplateName:     "更新任务模板",
		WorkOrderTitle:   "工单-{date}",
		CronExpression:   "0 0 9 * * *", // 表达式变更
		JobID:            &existingJob.ID,
	}
	require.NoError(t, db.Create(template).Error)

	err := syncWorkOrderJob(db, s, template)
	require.NoError(t, err)

	// Job 行状态不变(仍 Normal)
	var job models.Job
	require.NoError(t, db.First(&job, "id = ?", existingJob.ID).Error)
	assert.Equal(t, models.JobStatusNormal, job.Status)
}

// TestWot8002_DisableEnableJob DisablePeriodicWorkOrderJob→Status=Pause;
// EnablePeriodicWorkOrderJob→Status=Normal(重建 job)。
func TestWot8002_DisableEnableJob(t *testing.T) {
	s, db := newScheduler8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(s.Stop)

	template := &models.PeriodicWorkOrderTemplate{
		BaseModel:      models.BaseModel{ID: "tpl-toggle"},
		TemplateName:   "启停测试",
		CronExpression: "0 0 8 * * *",
	}
	require.NoError(t, db.Create(template).Error)

	// AddJob 在 running=true 时会先 db.Create 再 addJob
	job := &models.Job{
		JobName:        "periodic_workorder_tpl-toggle",
		JobGroup:       "PERIODIC_WORKORDER",
		InvokeTarget:   "periodic_workorder_create:tpl-toggle",
		CronExpression: "0 0 8 * * *",
		Status:         models.JobStatusNormal,
	}
	require.NoError(t, s.AddJob(job))
	template.JobID = &job.ID
	require.NoError(t, db.Model(template).Update("job_id", job.ID).Error)

	// Disable → Pause
	err := DisablePeriodicWorkOrderJob(s, template.ID)
	require.NoError(t, err)
	var row models.Job
	require.NoError(t, db.Unscoped().First(&row, "id = ?", job.ID).Error)
	assert.Equal(t, models.JobStatusPause, row.Status, "Disable 后应为暂停态")

	// Enable → Normal(重建 job)
	err = EnablePeriodicWorkOrderJob(s, template.ID)
	require.NoError(t, err)
	// 刷新 template 获取新的 JobID
	require.NoError(t, db.First(template, "id = ?", template.ID).Error)
	require.NotNil(t, template.JobID, "Enable 后模板应关联新 job")
	var newRow models.Job
	require.NoError(t, db.First(&newRow, "id = ?", *template.JobID).Error)
	assert.Equal(t, models.JobStatusNormal, newRow.Status, "Enable 后应为正常态")
}

// ============================================================================
// TestWot8002_GetTodayDutyPerson — sqlite 值班人查询
// ============================================================================

// TestWot8002_GetTodayDutyPerson 值班人有/无两分支。
func TestWot8002_GetTodayDutyPerson(t *testing.T) {
	db := newDutyDB8002(t)
	today := time.Now().Format("2006-01-02")

	// 分支 1:有值班人
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_schedule (id, user_id, schedule_date, status) VALUES ('sch-today', 'u-today', ?, 0)`, today).Error)
	got, err := getTodayDutyPerson(db)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "u-today", *got)

	// 分支 2:无值班人 — 把今日记录删掉即可
	require.NoError(t, db.Exec(`DELETE FROM sys_duty_schedule WHERE schedule_date = ?`, today).Error)
	got2, err := getTodayDutyPerson(db)
	require.Error(t, err)
	assert.Nil(t, got2)
	assert.Contains(t, err.Error(), "今日暂无值班人员")
}

// strPtr 辅助。
func strPtr(s string) *string { return &s }
