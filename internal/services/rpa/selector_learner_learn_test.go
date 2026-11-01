package rpa

// TIDY-03（v1.51）:256 LearnFromExecution 行为测试（方案 A 契约锁定）。
//
// 覆盖：
//   - Logs 内嵌 selectorEvents 信封解析（success → RecordSuccess upsert、
//     failed → RecordFailure；失败 ≥3 即 :389 广播条件）
//   - 普通日志行共存安全（非 JSON 行静默跳过）
//   - 不完整事件跳过、执行记录缺失/无明细的明确错误
//
// 契约：信封必须是单行合法 JSON（后端按行切分解析），详见
// docs/RPA-选择器上报明细契约.md。

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

func setupLearnTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE sys_rpa_executions (
		id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		task_id TEXT NOT NULL, task_name TEXT, worker_id TEXT, worker_name TEXT,
		status TEXT NOT NULL DEFAULT 'pending', start_time DATETIME, end_time DATETIME,
		duration INTEGER, progress_current INTEGER DEFAULT 0, progress_total INTEGER DEFAULT 0,
		screenshots TEXT DEFAULT '[]', logs TEXT, error_message TEXT,
		retry_count INTEGER DEFAULT 0, triggered_by TEXT, trigger_type TEXT)`).Error)
	// selector_*_records 模型 ID 带 gen_random_uuid() default（sqlite 不接受）→ 手写等价表
	require.NoError(t, db.Exec(`CREATE TABLE selector_success_records (
		id TEXT PRIMARY KEY, page_url TEXT, element_id TEXT, selector TEXT,
		selector_type TEXT, success_count INTEGER DEFAULT 1, avg_duration INTEGER DEFAULT 0,
		last_used_at DATETIME, created_at DATETIME, metadata TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE selector_failure_records (
		id TEXT PRIMARY KEY, page_url TEXT, element_id TEXT, selector TEXT,
		error_type TEXT, error_message TEXT, failure_count INTEGER DEFAULT 1,
		created_at DATETIME, resolved_at DATETIME, resolved_with TEXT)`).Error)
	return db
}

// newLearnerWithHub hub=nil：广播分支由 :389 实现保证；本文件锁定解析与学习闭环。
func newLearnerWithHub(t *testing.T, db *gorm.DB, hub *websocket.NoticeHub) SelectorLearner {
	t.Helper()
	return NewSelectorLearner(db, &fakeSelectorCache{}, newAICfg("", "", false, false), hub)
}

// singleLineEnvelope 组装单行信封（契约形态）。
func singleLineEnvelope(events ...string) string {
	return `{"selectorEvents":[` + strings.Join(events, ",") + `]}`
}

const evSubmitOK = `{"pageUrl":"http://p/login","elementId":"btn-submit","selector":"#submit","selectorType":"css","success":true,"durationMs":120}`
const evUserNotFound = `{"pageUrl":"http://p/login","elementId":"input-user","selector":"xpath://input[1]","success":false,"errorType":"not_found","errorMessage":"element not found"}`
const evGoneTimeout = `{"pageUrl":"http://p/a","elementId":"e1","selector":".gone","success":false,"errorType":"timeout"}`

func seedExecution(t *testing.T, db *gorm.DB, id, logs string) {
	t.Helper()
	require.NoError(t, db.Create(&rpamodels.Execution{
		ID: id, TaskID: "t-1", TaskName: "t", Status: "success", Logs: logs,
	}).Error)
}

func TestLearnFromExecution_ConsumesSelectorEvents(t *testing.T) {
	db := setupLearnTestDB(t)
	l := newLearnerWithHub(t, db, nil)
	ctx := context.Background()

	logs := "2026-11-01 10:00:00 INFO task started\n" +
		singleLineEnvelope(evSubmitOK, evUserNotFound) + "\n" +
		"2026-11-01 10:00:05 INFO task finished"
	seedExecution(t, db, "exec-1", logs)

	consumed, err := l.LearnFromExecution(ctx, "exec-1")
	require.NoError(t, err)
	assert.Equal(t, 2, consumed, "应消费 2 个事件（普通日志行跳过）")

	var success SelectorSuccessRecord
	require.NoError(t, db.Where("page_url = ? AND element_id = ?", "http://p/login", "btn-submit").First(&success).Error)
	assert.Equal(t, "#submit", success.Selector)
	assert.Equal(t, int64(120), success.AvgDuration)

	var failure SelectorFailureRecord
	require.NoError(t, db.Where("page_url = ? AND element_id = ?", "http://p/login", "input-user").First(&failure).Error)
	assert.Equal(t, "not_found", failure.ErrorType)
	assert.Equal(t, 1, failure.FailureCount)
}

func TestLearnFromExecution_FailureThreshold(t *testing.T) {
	db := setupLearnTestDB(t)
	l := newLearnerWithHub(t, db, nil)
	ctx := context.Background()
	seedExecution(t, db, "exec-i", singleLineEnvelope(evGoneTimeout))

	// 三次执行同一失败事件 → FailureCount 累计到 3（:389 广播条件的输入）
	for i := 1; i <= 3; i++ {
		consumed, err := l.LearnFromExecution(ctx, "exec-i")
		require.NoError(t, err)
		assert.Equal(t, 1, consumed)
	}
	var failure SelectorFailureRecord
	require.NoError(t, db.Where("selector = ?", ".gone").First(&failure).Error)
	assert.Equal(t, 3, failure.FailureCount, "失败应累计到 :389 广播阈值")
}

func TestLearnFromExecution_Errors(t *testing.T) {
	db := setupLearnTestDB(t)
	l := newLearnerWithHub(t, db, nil)
	ctx := context.Background()

	// 空 executionID
	_, err := l.LearnFromExecution(ctx, " ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能为空")

	// 执行记录不存在
	_, err = l.LearnFromExecution(ctx, "no-such-exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询执行记录失败")

	// logs 为空
	seedExecution(t, db, "exec-empty", "")
	_, err = l.LearnFromExecution(ctx, "exec-empty")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logs 为空")

	// logs 无 selectorEvents（纯普通日志）→ 明确错误
	seedExecution(t, db, "exec-plain", "plain log line")
	_, err = l.LearnFromExecution(ctx, "exec-plain")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未解析出任何选择器明细")

	// 不完整事件跳过 + 有效事件照常消费
	seedExecution(t, db, "exec-mixed", singleLineEnvelope(
		`{"pageUrl":"","elementId":"","selector":""}`,
		`{"pageUrl":"http://p","elementId":"ok","selector":"#ok","success":true,"durationMs":5}`))
	consumed, err := l.LearnFromExecution(ctx, "exec-mixed")
	require.NoError(t, err)
	assert.Equal(t, 1, consumed, "不完整事件应跳过，有效事件消费")
}
